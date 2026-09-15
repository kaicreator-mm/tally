package taskexecution

import (
	"context"
	"errors"
	"sync"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/engineering"
)

var ErrExecutionNotRunning = errors.New("taskexecution: task is not running")

type RuntimeExecutor struct {
	Runtime     agentruntime.Runtime
	Environment ExecutionEnvironment
	mu          sync.Mutex
	active      map[engineering.ID]context.CancelFunc
}

func NewRuntimeExecutor(runtime agentruntime.Runtime, env ExecutionEnvironment) (*RuntimeExecutor, error) {
	if runtime == nil || env == nil {
		return nil, errors.New("taskexecution: runtime and environment are required")
	}
	return &RuntimeExecutor{Runtime: runtime, Environment: env, active: map[engineering.ID]context.CancelFunc{}}, nil
}

func (e *RuntimeExecutor) Execute(ctx context.Context, req TaskExecutionRequest) (TaskExecutionResult, error) {
	if err := req.Validate(); err != nil {
		return TaskExecutionResult{}, err
	}
	if e.Environment.Kind() != req.Environment.Kind {
		return TaskExecutionResult{}, ErrUnsupportedEnvironment
	}
	if err := checkEnvironmentCapabilities(e.Environment, req.RequiredCapabilities.Environment); err != nil {
		return TaskExecutionResult{}, err
	}
	if _, err := agentruntime.Negotiate(e.Runtime.Descriptor().Capabilities, req.RequiredCapabilities.Runtime); err != nil {
		return TaskExecutionResult{}, err
	}
	resolved, err := e.Environment.Resolve(ctx, req.Workspace)
	if err != nil {
		return TaskExecutionResult{}, err
	}
	workspace, err := resolved.RuntimeLocation(req.Environment.Kind)
	if err != nil {
		return TaskExecutionResult{}, err
	}

	runCtx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	if _, exists := e.active[req.TaskID]; exists {
		e.mu.Unlock()
		cancel()
		return TaskExecutionResult{}, errors.New("taskexecution: task already running")
	}
	e.active[req.TaskID] = cancel
	e.mu.Unlock()
	defer func() { cancel(); e.mu.Lock(); delete(e.active, req.TaskID); e.mu.Unlock() }()

	start := agentruntime.StartRequest{Workspace: workspace, ModelProfile: req.Policy.ModelProfile, ResumeOf: req.Policy.ResumeSession}
	if start.ResumeOf != "" && !e.Runtime.Descriptor().Capabilities.Has(agentruntime.CapSessionResume) {
		if !req.Policy.FallbackToFresh {
			return TaskExecutionResult{}, &agentruntime.CapabilityError{Missing: []agentruntime.Capability{agentruntime.CapSessionResume}}
		}
		start.ResumeOf = ""
	}
	session, err := e.Runtime.Start(runCtx, start)
	if err != nil && req.Policy.ResumeSession != "" && req.Policy.FallbackToFresh {
		start.ResumeOf = ""
		session, err = e.Runtime.Start(runCtx, start)
	}
	if err != nil {
		if runCtx.Err() != nil {
			return interruptedResult(req), nil
		}
		return TaskExecutionResult{}, err
	}
	defer closeSession(session)

	claim, err := session.Execute(runCtx, agentruntime.ExecuteRequest{Instruction: req.Instruction})
	if err != nil {
		if runCtx.Err() != nil {
			return interruptedResult(req), nil
		}
		return TaskExecutionResult{}, err
	}
	status := ExecutionCompleted
	if claim.Status == agentruntime.RunFailed {
		status = ExecutionFailed
	}
	if claim.Status == agentruntime.RunCancelled || runCtx.Err() != nil {
		status = ExecutionInterrupted
	}
	return TaskExecutionResult{AgentClaim: claim, Workspace: req.Workspace, Runtime: req.Runtime, Environment: req.Environment, Status: status}, nil
}

func (e *RuntimeExecutor) Cancel(id engineering.ID) error {
	e.mu.Lock()
	cancel, ok := e.active[id]
	e.mu.Unlock()
	if !ok {
		return ErrExecutionNotRunning
	}
	cancel()
	return nil
}

const sessionCloseTimeout = 5 * time.Second

func closeSession(session agentruntime.Session) {
	ctx, cancel := context.WithTimeout(context.Background(), sessionCloseTimeout)
	defer cancel()
	_ = session.Close(ctx)
}

func interruptedResult(req TaskExecutionRequest) TaskExecutionResult {
	return TaskExecutionResult{Workspace: req.Workspace, Runtime: req.Runtime, Environment: req.Environment, Status: ExecutionInterrupted}
}
