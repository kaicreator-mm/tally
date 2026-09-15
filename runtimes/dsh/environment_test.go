package dsh

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/runner"
)

type environmentTestRunner struct {
	mu        sync.Mutex
	calls     []runner.Command
	responses []runner.Result
	errs      []error
	blockRun  bool
	started   chan struct{}
}

func (r *environmentTestRunner) Run(ctx context.Context, cmd runner.Command) (runner.Result, error) {
	r.mu.Lock()
	r.calls = append(r.calls, cmd)
	i := len(r.calls) - 1
	block := r.blockRun && len(cmd.Argv) > 0 && cmd.Argv[0] == "dsh" && len(cmd.Argv) > 1 && cmd.Argv[1] != "--version"
	started := r.started
	var res runner.Result
	var err error
	if i < len(r.responses) {
		res = r.responses[i]
	}
	if i < len(r.errs) {
		err = r.errs[i]
	}
	r.mu.Unlock()
	if block {
		if started != nil {
			select {
			case <-started:
			default:
				close(started)
			}
		}
		<-ctx.Done()
		return runner.Result{}, ctx.Err()
	}
	return res, err
}

func TestStartInEnvironmentExecutesDSHThroughGenericRunner(t *testing.T) {
	r := &environmentTestRunner{responses: []runner.Result{
		{Stdout: PinnedDSHVersion + "\n"}, // version
		{ExitCode: 0},                     // mkdir
		{Stdout: "remote answer\n"},       // execute
		{ExitCode: 0},                     // close cleanup
	}}
	rt := New(Config{Env: map[string]string{"DEEPSEEK_API_KEY": "test-secret"}})
	session, err := rt.StartInEnvironment(context.Background(), agentruntime.StartRequest{Workspace: "/workspace/task"}, r)
	if err != nil {
		t.Fatal(err)
	}
	result, err := session.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "implement task"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != agentruntime.RunCompleted || result.Summary != "remote answer" {
		t.Fatalf("result=%+v", result)
	}
	if err := session.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.calls) != 4 {
		t.Fatalf("calls=%d", len(r.calls))
	}
	execCall := r.calls[2]
	if execCall.Dir != "/workspace/task" {
		t.Fatalf("cwd=%q", execCall.Dir)
	}
	if got := strings.Join(execCall.Argv, " "); got != "dsh --profile headless implement task" {
		t.Fatalf("argv=%q", got)
	}
	joinedEnv := strings.Join(execCall.Env, "\n")
	if !strings.Contains(joinedEnv, "DSH_HOME=/workspace/task/.tally/runtime/dsh/session-000001") ||
		!strings.Contains(joinedEnv, "DSH_TELEMETRY_MODE=DISABLED") ||
		!strings.Contains(joinedEnv, "DSH_PERMISSION_MODE=workspace-write") {
		t.Fatalf("environment hardening missing: %s", joinedEnv)
	}
}

func TestEnvironmentSessionCancelPropagatesToRunnerContext(t *testing.T) {
	started := make(chan struct{})
	r := &environmentTestRunner{
		responses: []runner.Result{{Stdout: PinnedDSHVersion + "\n"}, {ExitCode: 0}},
		blockRun:  true,
		started:   started,
	}
	rt := New(Config{})
	sessionRaw, err := rt.StartInEnvironment(context.Background(), agentruntime.StartRequest{Workspace: "/workspace/task"}, r)
	if err != nil {
		t.Fatal(err)
	}
	session := sessionRaw.(*environmentSession)
	resultCh := make(chan agentruntime.AgentRunResult, 1)
	errCh := make(chan error, 1)
	go func() {
		result, execErr := session.Execute(context.Background(), agentruntime.ExecuteRequest{Instruction: "long task"})
		resultCh <- result
		errCh <- execErr
	}()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("environment runner did not start")
	}
	if err := session.Cancel(context.Background(), agentruntime.RunID("run-000001")); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-resultCh:
		if result.Status != agentruntime.RunCancelled {
			t.Fatalf("status=%s", result.Status)
		}
		if execErr := <-errCh; execErr != nil {
			t.Fatal(execErr)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("cancel did not propagate")
	}
}

func TestStartInEnvironmentRejectsVersionMismatch(t *testing.T) {
	r := &environmentTestRunner{responses: []runner.Result{{Stdout: "9.9.9\n"}}}
	_, err := New(Config{}).StartInEnvironment(context.Background(), agentruntime.StartRequest{Workspace: "/workspace/task"}, r)
	if err == nil || !strings.Contains(err.Error(), "版本") {
		t.Fatalf("err=%v", err)
	}
}

func TestStartInEnvironmentTransportFailureIsExplicit(t *testing.T) {
	r := &environmentTestRunner{errs: []error{errors.New("transport down")}}
	_, err := New(Config{}).StartInEnvironment(context.Background(), agentruntime.StartRequest{Workspace: "/workspace/task"}, r)
	if err == nil || !strings.Contains(err.Error(), "transport") {
		t.Fatalf("err=%v", err)
	}
}
