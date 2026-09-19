package main

import (
	"context"
	"testing"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

type recordingExecutor struct {
	name    string
	execute func(context.Context, taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error)
	reqs    []taskexecution.TaskExecutionRequest
}

func (r *recordingExecutor) Execute(ctx context.Context, req taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error) {
	r.reqs = append(r.reqs, req)
	if r.execute != nil {
		return r.execute(ctx, req)
	}
	return taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionCompleted}, nil
}

func (r *recordingExecutor) Cancel(id engineering.ID) error { return nil }

func TestV15MultiAgentVerificationRoutingReviewRolesUseVerifiedExecutor(t *testing.T) {
	base := &recordingExecutor{name: "base"}
	verified := &recordingExecutor{name: "verified"}
	router := newVerificationRoutingExecutor(base, verified)

	for _, instruction := range []string{
		"Tally engineering role: REVIEWER.",
		"Tally engineering role: VERIFIER.",
		"Tally engineering role: SPECIALIST (security).",
	} {
		req := taskexecution.TaskExecutionRequest{Instruction: instruction}
		if _, err := router.Execute(context.Background(), req); err != nil {
			t.Fatalf("execute %q: %v", instruction, err)
		}
	}

	if len(verified.reqs) != 3 {
		t.Fatalf("verified executor received %d requests, want 3", len(verified.reqs))
	}
	if len(base.reqs) != 0 {
		t.Fatalf("base executor unexpectedly received %d review-role requests", len(base.reqs))
	}
}

func TestV15MultiAgentVerificationRoutingProducerRolesUnchanged(t *testing.T) {
	want := taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionCompleted}
	base := &recordingExecutor{name: "base", execute: func(context.Context, taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error) {
		return want, nil
	}}
	verified := &recordingExecutor{name: "verified"}
	router := newVerificationRoutingExecutor(base, verified)

	for _, instruction := range []string{
		"Tally engineering role: IMPLEMENTER.",
		"Tally engineering role: REPAIRER.",
	} {
		got, err := router.Execute(context.Background(), taskexecution.TaskExecutionRequest{Instruction: instruction})
		if err != nil {
			t.Fatalf("execute %q: %v", instruction, err)
		}
		if got.Status != want.Status {
			t.Fatalf("producer role %q result %+v, want unchanged %+v", instruction, got, want)
		}
	}

	if len(base.reqs) != 2 {
		t.Fatalf("base executor received %d requests, want 2", len(base.reqs))
	}
	if len(verified.reqs) != 0 {
		t.Fatalf("verified executor unexpectedly received %d producer-role requests", len(verified.reqs))
	}
}

func TestV15MultiAgentVerificationRoutingNonRoleInstructionUsesBaseExecutor(t *testing.T) {
	base := &recordingExecutor{name: "base"}
	verified := &recordingExecutor{name: "verified"}
	router := newVerificationRoutingExecutor(base, verified)

	if _, err := router.Execute(context.Background(), taskexecution.TaskExecutionRequest{Instruction: "plain task prompt"}); err != nil {
		t.Fatalf("execute non-role instruction: %v", err)
	}
	if len(base.reqs) != 1 || len(verified.reqs) != 0 {
		t.Fatalf("non-role instruction routed to verified executor: base=%d verified=%d", len(base.reqs), len(verified.reqs))
	}
}
