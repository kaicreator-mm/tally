package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"tally/internal/agentruntime"
	"tally/internal/engineering"
	"tally/internal/multiagent"
	"tally/internal/store"
	"tally/internal/taskexecution"
)

func TestV15MultiAgentMaxConcurrencyFromEnv(t *testing.T) {
	t.Setenv("TALLY_V15_MULTI_AGENT_MAX_CONCURRENCY", "")
	if got, err := v15MultiAgentMaxConcurrencyFromEnv(); err != nil || got != 2 {
		t.Fatalf("default max concurrency=(%d,%v), want (2,nil)", got, err)
	}

	t.Setenv("TALLY_V15_MULTI_AGENT_MAX_CONCURRENCY", "4")
	if got, err := v15MultiAgentMaxConcurrencyFromEnv(); err != nil || got != 4 {
		t.Fatalf("configured max concurrency=(%d,%v), want (4,nil)", got, err)
	}

	for _, bad := range []string{"0", "-1", "not-an-int"} {
		t.Run(bad, func(t *testing.T) {
			t.Setenv("TALLY_V15_MULTI_AGENT_MAX_CONCURRENCY", bad)
			if _, err := v15MultiAgentMaxConcurrencyFromEnv(); err == nil {
				t.Fatalf("value %q unexpectedly accepted", bad)
			}
		})
	}
}

func TestV15ProductionCompositionEnablesMultiAgentOnlyWithExecutionRuntime(t *testing.T) {
	queries := &store.Queries{}
	contentRepo := t.TempDir()

	callPolicy := func(t *testing.T, runtime string) int {
		t.Helper()
		t.Setenv("TALLY_V15_RUNTIME", runtime)
		t.Setenv("TALLY_V15_STATE_ROOT", t.TempDir())
		t.Setenv("TALLY_V15_MULTI_AGENT_MAX_CONCURRENCY", "2")
		handler, _, err := v15ControlPlaneFromEnv(contentRepo, queries)
		if err != nil {
			t.Fatalf("compose v1.5 control plane: %v", err)
		}
		req := httptest.NewRequest(http.MethodPost, "/api/v15/projects/missing/tasks/T1/multi-agent/policy", bytes.NewBufferString(`{}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		return rec.Code
	}

	if status := callPolicy(t, ""); status != http.StatusServiceUnavailable {
		t.Fatalf("without execution runtime multi-agent policy status=%d, want 503", status)
	}
	if status := callPolicy(t, "fake"); status != http.StatusNotFound {
		t.Fatalf("with fake runtime multi-agent policy status=%d, want 404 from repository after capability gate", status)
	}
}

type recordingExecutor struct {
	requests []taskexecution.TaskExecutionRequest
	result   taskexecution.TaskExecutionResult
	canceled []engineering.ID
}

func (e *recordingExecutor) Execute(_ context.Context, req taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error) {
	e.requests = append(e.requests, req)
	return e.result, nil
}

func (e *recordingExecutor) Cancel(id engineering.ID) error {
	e.canceled = append(e.canceled, id)
	return nil
}

func v15RoleInstruction(roleDirective string) taskexecution.TaskExecutionRequest {
	return taskexecution.TaskExecutionRequest{TaskID: "T1", Instruction: "do the work\n\n" + roleDirective}
}

// The guarded executor is built against an empty content repo, so any request
// that reaches it fails while loading the frozen task bundle. A raw execution
// of the same request would succeed; the outcomes below therefore prove which
// executor handled each role.
func TestV15MultiAgentExecutorGuardsReviewRoles(t *testing.T) {
	raw := &recordingExecutor{result: taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionCompleted}}
	executor, err := composeV15MultiAgentExecutor(raw, multiagent.VerifiedRoleExecutorConfig{
		Repo:         t.TempDir(),
		WorktreeRoot: t.TempDir(),
		EnvName:      "test",
		Runtime:      agentruntime.RuntimeDescriptor{Name: "fake"},
		Queries:      &store.Queries{},
	})
	if err != nil {
		t.Fatalf("compose v1.5 multi-agent executor: %v", err)
	}

	for _, directive := range []string{
		"Tally engineering role: IMPLEMENTER.",
		"Tally engineering role: REPAIRER.",
	} {
		if _, execErr := executor.Execute(context.Background(), v15RoleInstruction(directive)); execErr != nil {
			t.Fatalf("%q execution errored: %v", directive, execErr)
		}
	}
	if len(raw.requests) != 2 {
		t.Fatalf("implementation/repair execution reached the raw executor %d times, want 2", len(raw.requests))
	}

	for _, directive := range []string{
		"Tally engineering role: REVIEWER.",
		"Tally engineering role: VERIFIER.",
		"Tally engineering role: SPECIALIST (security).",
	} {
		_, execErr := executor.Execute(context.Background(), v15RoleInstruction(directive))
		if execErr == nil || !strings.Contains(execErr.Error(), "load frozen task bundle") {
			t.Fatalf("%q execution did not enter the verification guard: %v", directive, execErr)
		}
	}
	if len(raw.requests) != 2 {
		t.Fatalf("review/verification execution bypassed the guard and reached the raw executor: %d raw calls", len(raw.requests))
	}
}

func TestV15MultiAgentExecutorPreservesRawResultAndCancel(t *testing.T) {
	raw := &recordingExecutor{
		result: taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionCompleted, AgentClaim: agentruntime.AgentRunResult{SessionID: "s1", RunID: "r1"}},
	}
	executor, err := composeV15MultiAgentExecutor(raw, multiagent.VerifiedRoleExecutorConfig{
		Repo:         t.TempDir(),
		WorktreeRoot: t.TempDir(),
		EnvName:      "test",
		Runtime:      agentruntime.RuntimeDescriptor{Name: "fake"},
		Queries:      &store.Queries{},
	})
	if err != nil {
		t.Fatalf("compose v1.5 multi-agent executor: %v", err)
	}

	result, execErr := executor.Execute(context.Background(), v15RoleInstruction("Tally engineering role: IMPLEMENTER."))
	if execErr != nil {
		t.Fatalf("implementer execution errored: %v", execErr)
	}
	if result.Status != taskexecution.ExecutionCompleted || result.AgentClaim.SessionID != "s1" || result.AgentClaim.RunID != "r1" {
		t.Fatalf("implementer result was altered by the guarded executor: %+v", result)
	}
	if result.CandidateRef != "" {
		t.Fatalf("implementer result gained a candidate ref through the guarded executor: %q", result.CandidateRef)
	}

	if cancelErr := executor.Cancel("T1"); cancelErr != nil {
		t.Fatalf("cancel errored: %v", cancelErr)
	}
	if len(raw.canceled) != 1 || raw.canceled[0] != engineering.ID("T1") {
		t.Fatalf("cancel was not routed to the raw executor: %v", raw.canceled)
	}
}

func TestV15MultiAgentExecutorRejectsInvalidGuardConfig(t *testing.T) {
	if _, err := composeV15MultiAgentExecutor(&recordingExecutor{}, multiagent.VerifiedRoleExecutorConfig{
		Repo:         t.TempDir(),
		WorktreeRoot: t.TempDir(),
		EnvName:      "test",
		Runtime:      agentruntime.RuntimeDescriptor{Name: "fake"},
	}); err == nil {
		t.Fatal("composition without fact-store queries unexpectedly accepted")
	}
	if _, err := composeV15MultiAgentExecutor(nil, multiagent.VerifiedRoleExecutorConfig{
		Repo:         t.TempDir(),
		WorktreeRoot: t.TempDir(),
		EnvName:      "test",
		Runtime:      agentruntime.RuntimeDescriptor{Name: "fake"},
		Queries:      &store.Queries{},
	}); err == nil {
		t.Fatal("composition without a delegate must fail")
	}
}
