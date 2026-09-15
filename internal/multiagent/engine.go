package multiagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

var (
	ErrAlreadyActive            = errors.New("multiagent: task collaboration is already active")
	ErrIndependenceViolation    = errors.New("multiagent: independent role constraint is not satisfied")
	ErrRepairBudgetExhausted    = errors.New("multiagent: repair budget exhausted")
	ErrRoleExecutionUnavailable = errors.New("multiagent: role execution unavailable")
)

type Event struct {
	Kind   string `json:"kind"`
	Role   Role   `json:"role,omitempty"`
	RunID  string `json:"run_id,omitempty"`
	Reason string `json:"reason,omitempty"`
}

type Checkpoint func(context.Context, WorkflowState, Event) error

type Config struct {
	MaxConcurrency int
	Now            func() time.Time
}

type Engine struct {
	executor    taskexecution.AgentExecutor
	interpreter Interpreter
	now         func() time.Time
	maxParallel int
	sem         chan struct{}

	mu     sync.Mutex
	active map[engineering.ID]context.CancelFunc
}

func NewEngine(executor taskexecution.AgentExecutor, interpreter Interpreter, cfg Config) (*Engine, error) {
	if executor == nil {
		return nil, errors.New("multiagent: AgentExecutor is required")
	}
	if interpreter == nil {
		interpreter = EvidenceInterpreter{}
	}
	if cfg.MaxConcurrency < 1 {
		cfg.MaxConcurrency = 2
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Engine{
		executor:    executor,
		interpreter: interpreter,
		now:         cfg.Now,
		maxParallel: cfg.MaxConcurrency,
		sem:         make(chan struct{}, cfg.MaxConcurrency),
		active:      map[engineering.ID]context.CancelFunc{},
	}, nil
}

func (e *Engine) MaxConcurrency() int { return e.maxParallel }

func (e *Engine) IsActive(id engineering.ID) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ok := e.active[id]
	return ok
}

func (e *Engine) Cancel(id engineering.ID) error {
	e.mu.Lock()
	cancel, ok := e.active[id]
	e.mu.Unlock()
	if !ok {
		return taskexecution.ErrExecutionNotRunning
	}
	cancel()
	_ = e.executor.Cancel(id)
	return nil
}

type Result struct {
	State              WorkflowState                     `json:"state"`
	Verified           bool                              `json:"verified"`
	Reason             string                            `json:"reason,omitempty"`
	Facts              engineering.CompletionFacts       `json:"facts"`
	CandidateExecution taskexecution.TaskExecutionResult `json:"candidate_execution"`
}

func (e *Engine) Run(ctx context.Context, req taskexecution.TaskExecutionRequest, policy TaskPolicy, state WorkflowState, checkpoint Checkpoint) (Result, error) {
	if err := req.Validate(); err != nil {
		return Result{}, err
	}
	policy = policy.Normalize()
	if err := policy.Validate(); err != nil {
		return Result{}, err
	}
	if state.TaskID == "" {
		var err error
		state, err = NewWorkflowState(req.TaskID, e.now())
		if err != nil {
			return Result{}, err
		}
	} else {
		if state.TaskID != req.TaskID {
			return Result{}, errors.New("multiagent: workflow task does not match execution request")
		}
		if err := state.Validate(); err != nil {
			return Result{}, err
		}
	}
	if state.Status == WorkflowAccepted || state.Status == WorkflowVerified {
		return resultFromState(state), nil
	}

	runCtx, cancel := context.WithCancel(ctx)
	if err := e.register(req.TaskID, cancel); err != nil {
		cancel()
		return Result{}, err
	}
	defer func() {
		cancel()
		e.unregister(req.TaskID)
	}()

	// Register before waiting for a concurrency slot so Cancel/Restart can
	// reliably stop collaborations that are queued on the bounded semaphore.
	select {
	case e.sem <- struct{}{}:
		defer func() { <-e.sem }()
	case <-runCtx.Done():
		return Result{State: state, Reason: runCtx.Err().Error()}, runCtx.Err()
	}

	if recovered, ok := recoverRunningRole(&state, e.now()); ok {
		if err := emit(runCtx, checkpoint, state, Event{Kind: "role.recovered_interrupted", Role: recovered.Role, RunID: recovered.ID, Reason: recovered.Reason}); err != nil {
			return Result{State: state, Reason: err.Error()}, err
		}
	}
	if state.Status == WorkflowBlocked || state.Status == WorkflowInterrupted {
		state.Status = WorkflowActive
		state.Reason = ""
		bump(&state, e.now())
		if err := emit(runCtx, checkpoint, state, Event{Kind: "workflow.resumed", Role: state.NextRole}); err != nil {
			return Result{State: state, Reason: err.Error()}, err
		}
	}

	for state.Status == WorkflowActive {
		role := state.NextRole
		specialist := ""
		if role == RoleSpecialist {
			if state.SpecialistIndex >= len(policy.Specialists) {
				state.NextRole = RoleReviewer
				bump(&state, e.now())
				continue
			}
			specialist = policy.Specialists[state.SpecialistIndex]
		}
		if role == RoleRepairer && state.RepairAttempts >= policy.MaxRepairAttempts {
			state.Status = WorkflowBlocked
			state.Reason = ErrRepairBudgetExhausted.Error()
			bump(&state, e.now())
			_ = emit(runCtx, checkpoint, state, Event{Kind: "workflow.blocked", Role: role, Reason: state.Reason})
			return resultFromState(state), ErrRepairBudgetExhausted
		}
		if err := enforcePreRunIndependence(req, policy, state, role, specialist); err != nil {
			state.Status = WorkflowBlocked
			state.Reason = err.Error()
			bump(&state, e.now())
			_ = emit(runCtx, checkpoint, state, Event{Kind: "role.independence_blocked", Role: role, Reason: state.Reason})
			return resultFromState(state), err
		}

		if role == RoleRepairer {
			state.RepairAttempts++
		}
		runID := fmt.Sprintf("%s:%s:%03d", req.TaskID, role, len(state.Runs)+1)
		if specialist != "" {
			runID = fmt.Sprintf("%s:specialist:%s:%03d", req.TaskID, sanitizeID(specialist), len(state.Runs)+1)
		}
		run := RoleRun{
			ID:         runID,
			Sequence:   len(state.Runs) + 1,
			Role:       role,
			Specialist: specialist,
			Status:     RunRunning,
			StartedAt:  e.now().UTC(),
		}
		state.Runs = append(state.Runs, run)
		bump(&state, e.now())
		if err := emit(runCtx, checkpoint, state, Event{Kind: "role.started", Role: role, RunID: runID}); err != nil {
			return Result{State: state, Reason: err.Error()}, err
		}

		roleReq, err := buildRoleRequest(req, policy, state, run)
		if err != nil {
			return e.failRun(runCtx, state, len(state.Runs)-1, checkpoint, err, "role.context_failed")
		}
		execution, execErr := e.executor.Execute(runCtx, roleReq)
		idx := len(state.Runs) - 1
		state.Runs[idx].Execution = execution
		state.Runs[idx].FinishedAt = e.now().UTC()
		if execErr != nil {
			refundRepairAttempt(&state, role)
			state.Runs[idx].Status = RunFailed
			state.Runs[idx].Reason = execErr.Error()
			state.Status = WorkflowBlocked
			state.Reason = fmt.Sprintf("%s: %v", ErrRoleExecutionUnavailable, execErr)
			bump(&state, e.now())
			_ = emit(context.WithoutCancel(runCtx), checkpoint, state, Event{Kind: "role.error", Role: role, RunID: runID, Reason: state.Reason})
			return resultFromState(state), execErr
		}
		if err := enforcePostRunIndependence(policy, state, idx); err != nil {
			state.Runs[idx].Status = RunBlocked
			state.Runs[idx].Reason = err.Error()
			state.Status = WorkflowBlocked
			state.Reason = err.Error()
			bump(&state, e.now())
			_ = emit(context.WithoutCancel(runCtx), checkpoint, state, Event{Kind: "role.independence_blocked", Role: role, RunID: runID, Reason: state.Reason})
			return resultFromState(state), err
		}

		decision, err := e.interpreter.Interpret(runCtx, RoleContext{TaskID: req.TaskID, Role: role, Specialist: specialist, RunID: runID}, execution)
		if err != nil {
			return e.failRun(runCtx, state, idx, checkpoint, err, "role.interpretation_failed")
		}
		for i := range decision.Findings {
			decision.Findings[i].ProducerRole = role
			decision.Findings[i].ProducerRunID = runID
		}
		state.Findings = append(state.Findings, decision.Findings...)
		state.Runs[idx].Reason = strings.TrimSpace(decision.Reason)
		switch decision.Kind {
		case DecisionInterrupted:
			refundRepairAttempt(&state, role)
			state.Runs[idx].Status = RunInterrupted
			state.Status = WorkflowInterrupted
			state.Reason = nonEmpty(decision.Reason, "role execution interrupted")
		case DecisionBlock:
			state.Runs[idx].Status = RunBlocked
			state.Status = WorkflowBlocked
			state.Reason = nonEmpty(decision.Reason, "role blocked workflow")
		case DecisionRepair:
			state.Runs[idx].Status = RunRepair
			if state.RepairAttempts >= policy.MaxRepairAttempts {
				state.Status = WorkflowBlocked
				state.Reason = ErrRepairBudgetExhausted.Error()
			} else {
				state.NextRole = RoleRepairer
				state.Reason = nonEmpty(decision.Reason, "repair requested")
			}
		case DecisionPass:
			state.Runs[idx].Status = RunPassed
			advanceAfterPass(&state, policy, role)
		default:
			return e.failRun(runCtx, state, idx, checkpoint, fmt.Errorf("multiagent: invalid decision %q", decision.Kind), "role.interpretation_failed")
		}
		bump(&state, e.now())
		kind := "role.completed"
		if state.Status == WorkflowBlocked {
			kind = "workflow.blocked"
		} else if state.Status == WorkflowInterrupted {
			kind = "workflow.interrupted"
		} else if state.Status == WorkflowVerified {
			kind = "workflow.verified"
		}
		if err := emit(context.WithoutCancel(runCtx), checkpoint, state, Event{Kind: kind, Role: role, RunID: runID, Reason: state.Reason}); err != nil {
			return Result{State: state, Reason: err.Error()}, err
		}
		if state.Status != WorkflowActive {
			break
		}
	}
	result := resultFromState(state)
	if state.Status == WorkflowInterrupted && runCtx.Err() != nil {
		return result, runCtx.Err()
	}
	return result, nil
}

func (e *Engine) failRun(ctx context.Context, state WorkflowState, idx int, checkpoint Checkpoint, cause error, kind string) (Result, error) {
	if idx >= 0 && idx < len(state.Runs) {
		refundRepairAttempt(&state, state.Runs[idx].Role)
		state.Runs[idx].Status = RunFailed
		state.Runs[idx].FinishedAt = e.now().UTC()
		state.Runs[idx].Reason = cause.Error()
	}
	state.Status = WorkflowBlocked
	state.Reason = cause.Error()
	bump(&state, e.now())
	role := Role("")
	runID := ""
	if idx >= 0 && idx < len(state.Runs) {
		role = state.Runs[idx].Role
		runID = state.Runs[idx].ID
	}
	_ = emit(context.WithoutCancel(ctx), checkpoint, state, Event{Kind: kind, Role: role, RunID: runID, Reason: cause.Error()})
	return resultFromState(state), cause
}

func (e *Engine) register(id engineering.ID, cancel context.CancelFunc) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.active[id]; ok {
		return ErrAlreadyActive
	}
	e.active[id] = cancel
	return nil
}

func (e *Engine) unregister(id engineering.ID) {
	e.mu.Lock()
	delete(e.active, id)
	e.mu.Unlock()
}

func resultFromState(state WorkflowState) Result {
	return Result{
		State:              state,
		Verified:           state.Status == WorkflowVerified || state.Status == WorkflowAccepted,
		Reason:             state.Reason,
		Facts:              state.Facts(),
		CandidateExecution: state.CandidateExecution(),
	}
}

func recoverRunningRole(state *WorkflowState, now time.Time) (RoleRun, bool) {
	if len(state.Runs) == 0 {
		return RoleRun{}, false
	}
	idx := len(state.Runs) - 1
	if state.Runs[idx].Status != RunRunning {
		return RoleRun{}, false
	}
	state.Runs[idx].Status = RunInterrupted
	if state.Runs[idx].Role == RoleRepairer && state.RepairAttempts > 0 {
		state.RepairAttempts--
	}
	state.Runs[idx].FinishedAt = now.UTC()
	state.Runs[idx].Reason = "process restart recovered an in-flight role; role will restart fresh"
	state.Status = WorkflowActive
	state.NextRole = state.Runs[idx].Role
	state.Reason = ""
	bump(state, now)
	return state.Runs[idx], true
}

func advanceAfterPass(state *WorkflowState, policy TaskPolicy, role Role) {
	switch role {
	case RoleImplementer:
		state.SpecialistIndex = 0
		if len(policy.Specialists) > 0 {
			state.NextRole = RoleSpecialist
		} else {
			state.NextRole = RoleReviewer
		}
	case RoleSpecialist:
		state.SpecialistIndex++
		if state.SpecialistIndex < len(policy.Specialists) {
			state.NextRole = RoleSpecialist
		} else {
			state.NextRole = RoleReviewer
		}
	case RoleReviewer:
		state.NextRole = RoleVerifier
	case RoleRepairer:
		for i := range state.Findings {
			if !state.Findings[i].Resolved {
				state.Findings[i].Resolved = true
			}
		}
		state.SpecialistIndex = 0
		if len(policy.Specialists) > 0 {
			state.NextRole = RoleSpecialist
		} else {
			state.NextRole = RoleReviewer
		}
	case RoleVerifier:
		state.Status = WorkflowVerified
		state.NextRole = ""
		state.Reason = ""
	}
}

func enforcePreRunIndependence(base taskexecution.TaskExecutionRequest, policy TaskPolicy, state WorkflowState, role Role, specialist string) error {
	independence := policy.IndependenceFor(role)
	if independence != IndependenceRuntime {
		return nil
	}
	candidate, ok := state.CandidateRun()
	if !ok {
		return fmt.Errorf("%w: %s has no candidate-producing run to compare", ErrIndependenceViolation, role)
	}
	assignment := policy.Assignment(role, specialist)
	runtimeID := strings.TrimSpace(assignment.RuntimeID)
	if runtimeID == "" {
		runtimeID = strings.TrimSpace(base.Runtime.ID)
	}
	if runtimeID == "" || candidate.RuntimeID() == "" || runtimeID == candidate.RuntimeID() {
		return fmt.Errorf("%w: %s must use a different runtime/provider from candidate producer", ErrIndependenceViolation, role)
	}
	return nil
}

func enforcePostRunIndependence(policy TaskPolicy, state WorkflowState, current int) error {
	if current < 0 || current >= len(state.Runs) {
		return nil
	}
	run := state.Runs[current]
	independence := policy.IndependenceFor(run.Role)
	if independence == IndependenceNone {
		return nil
	}
	var candidate RoleRun
	found := false
	for i := current - 1; i >= 0; i-- {
		if (state.Runs[i].Role == RoleImplementer || state.Runs[i].Role == RoleRepairer) && state.Runs[i].Status == RunPassed {
			candidate = state.Runs[i]
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("%w: %s cannot establish candidate producer", ErrIndependenceViolation, run.Role)
	}
	if independence == IndependenceRuntime {
		if run.RuntimeID() == "" || candidate.RuntimeID() == "" || run.RuntimeID() == candidate.RuntimeID() {
			return fmt.Errorf("%w: %s runtime/provider equals candidate producer", ErrIndependenceViolation, run.Role)
		}
	}
	if run.SessionID() == "" || run.AgentRunID() == "" || candidate.SessionID() == "" || candidate.AgentRunID() == "" {
		return fmt.Errorf("%w: %s run/session identity is missing", ErrIndependenceViolation, run.Role)
	}
	if run.SessionID() == candidate.SessionID() || run.AgentRunID() == candidate.AgentRunID() {
		return fmt.Errorf("%w: %s reused candidate producer run/session", ErrIndependenceViolation, run.Role)
	}
	return nil
}

func buildRoleRequest(base taskexecution.TaskExecutionRequest, policy TaskPolicy, state WorkflowState, run RoleRun) (taskexecution.TaskExecutionRequest, error) {
	req := base
	req.EngineeringContext = append([]engineering.ArtifactRef(nil), base.EngineeringContext...)
	req.Policy.ResumeSession = ""
	req.Policy.FallbackToFresh = true
	assignment := policy.Assignment(run.Role, run.Specialist)
	if strings.TrimSpace(assignment.RuntimeID) != "" {
		req.Runtime.ID = strings.TrimSpace(assignment.RuntimeID)
	}
	if strings.TrimSpace(assignment.ModelProfile) != "" {
		req.Policy.ModelProfile = strings.TrimSpace(assignment.ModelProfile)
	}
	if candidate, ok := state.CandidateRun(); ok {
		for _, artifact := range candidate.Execution.ArtifactRefs {
			req.EngineeringContext = appendUniqueArtifact(req.EngineeringContext, artifact)
		}
	}
	view := struct {
		Role         Role                      `json:"role"`
		Specialist   string                    `json:"specialist,omitempty"`
		TaskID       engineering.ID            `json:"task_id"`
		RoleRunID    string                    `json:"role_run_id"`
		CandidateRef string                    `json:"candidate_ref,omitempty"`
		Artifacts    []engineering.ArtifactRef `json:"artifacts,omitempty"`
		Evidence     []engineering.EvidenceRef `json:"evidence,omitempty"`
		Findings     []Finding                 `json:"findings,omitempty"`
	}{Role: run.Role, Specialist: run.Specialist, TaskID: base.TaskID, RoleRunID: run.ID, Findings: unresolvedFindings(state.Findings)}
	if candidate, ok := state.CandidateRun(); ok {
		view.CandidateRef = candidate.Execution.CandidateRef
		view.Artifacts = append([]engineering.ArtifactRef(nil), candidate.Execution.ArtifactRefs...)
		view.Evidence = append([]engineering.EvidenceRef(nil), candidate.Execution.EvidenceRefs...)
	}
	raw, err := json.Marshal(view)
	if err != nil {
		return taskexecution.TaskExecutionRequest{}, err
	}
	req.Instruction = strings.TrimSpace(base.Instruction) + "\n\n" + roleDirective(run.Role, run.Specialist) + "\nTally role context (durable facts/references only; no prior chain-of-thought):\n" + string(raw)
	return req, req.Validate()
}

func roleDirective(role Role, specialist string) string {
	switch role {
	case RoleImplementer:
		return "Tally engineering role: IMPLEMENTER. Produce the candidate required by the frozen TaskContract. Do not claim acceptance."
	case RoleReviewer:
		return "Tally engineering role: REVIEWER. Independently inspect the candidate. Do not modify implementation. Route factual checks through standard evidence outputs; a summary alone is not evidence."
	case RoleRepairer:
		return "Tally engineering role: REPAIRER. Repair only the durable unresolved findings, preserve the frozen TaskContract, and do not claim acceptance."
	case RoleVerifier:
		return "Tally engineering role: VERIFIER. Independently verify the candidate against the frozen TaskContract/gates. Do not modify implementation. PASS requires factual evidence."
	case RoleSpecialist:
		return "Tally engineering role: SPECIALIST (" + specialist + "). Inspect only your assigned engineering concern. Do not mutate implementation; report through factual evidence."
	default:
		return "Tally engineering role: " + string(role)
	}
}

func appendUniqueArtifact(in []engineering.ArtifactRef, artifact engineering.ArtifactRef) []engineering.ArtifactRef {
	for _, current := range in {
		if current.ID == artifact.ID {
			return in
		}
	}
	return append(in, artifact)
}

func unresolvedFindings(in []Finding) []Finding {
	out := make([]Finding, 0, len(in))
	for _, finding := range in {
		if !finding.Resolved {
			out = append(out, finding)
		}
	}
	return out
}

func emit(ctx context.Context, checkpoint Checkpoint, state WorkflowState, event Event) error {
	if checkpoint == nil {
		return nil
	}
	return checkpoint(ctx, state, event)
}

func bump(state *WorkflowState, now time.Time) {
	state.Revision++
	state.UpdatedAt = now.UTC()
}

func refundRepairAttempt(state *WorkflowState, role Role) {
	if role == RoleRepairer && state.RepairAttempts > 0 {
		state.RepairAttempts--
	}
}

func sanitizeID(in string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(in)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('-')
		}
	}
	if b.Len() == 0 {
		return "specialist"
	}
	return b.String()
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}
