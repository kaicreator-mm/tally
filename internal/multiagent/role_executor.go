package multiagent

import (
	"context"
	"errors"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

// RoleExecutorRouter keeps ordinary implementation/repair execution on the
// production AgentExecutor while routing reviewer/verifier roles through the
// repository's existing verification-enforcement executor. It deliberately
// contains no verification or worktree logic of its own.
type RoleExecutorRouter struct {
	defaultExecutor      taskexecution.AgentExecutor
	verificationExecutor taskexecution.AgentExecutor
}

func NewRoleExecutorRouter(defaultExecutor, verificationExecutor taskexecution.AgentExecutor) (*RoleExecutorRouter, error) {
	if defaultExecutor == nil {
		return nil, errors.New("multiagent: default AgentExecutor is required")
	}
	if verificationExecutor == nil {
		return nil, errors.New("multiagent: verification AgentExecutor is required")
	}
	return &RoleExecutorRouter{defaultExecutor: defaultExecutor, verificationExecutor: verificationExecutor}, nil
}

func (e *RoleExecutorRouter) Execute(ctx context.Context, req taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error) {
	role, ok := roleFromInstruction(req.Instruction)
	if ok && (role == RoleReviewer || role == RoleVerifier) {
		return e.verificationExecutor.Execute(ctx, req)
	}
	return e.defaultExecutor.Execute(ctx, req)
}

func (e *RoleExecutorRouter) Cancel(id engineering.ID) error {
	// VerifiedRoleExecutor.Cancel delegates to the same production executor, so
	// cancelling the default authority is sufficient and avoids double cancel.
	return e.defaultExecutor.Cancel(id)
}
