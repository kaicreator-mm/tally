package main

import (
	"context"

	"tally/internal/engineering"
	"tally/internal/multiagent"
	"tally/internal/taskexecution"
)

// verificationRoutingExecutor composes the frozen v1.5.4 VerifiedRoleExecutor
// into the production multi-agent assembly for review-oriented read-only roles
// (REVIEWER, VERIFIER, SPECIALIST). Producer roles (IMPLEMENTER, REPAIRER) are
// passed straight to the base executor so ordinary implementation/repair
// execution semantics remain unchanged. No second verification executor is
// introduced: this is a routing composition over the existing capability.
type verificationRoutingExecutor struct {
	review   taskexecution.AgentExecutor
	producer taskexecution.AgentExecutor
}

func newVerificationRoutingExecutor(base taskexecution.AgentExecutor, verified taskexecution.AgentExecutor) verificationRoutingExecutor {
	return verificationRoutingExecutor{review: verified, producer: base}
}

func (e verificationRoutingExecutor) Cancel(id engineering.ID) error {
	// The verified executor delegates cancellation to the same base executor,
	// so either route owns the identical lifecycle.
	return e.producer.Cancel(id)
}

func (e verificationRoutingExecutor) Execute(ctx context.Context, req taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error) {
	if role, ok := multiagent.RoleFromInstruction(req.Instruction); ok && multiagent.IsReadOnlyRole(role) {
		return e.review.Execute(ctx, req)
	}
	return e.producer.Execute(ctx, req)
}
