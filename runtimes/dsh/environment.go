package dsh

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"

	"tally/internal/agentruntime"
	"tally/internal/logpipe"
	"tally/internal/runner"
)

// StartInEnvironment is the v1.5.2 orthogonal Runtime × Environment seam.
// It keeps DSH protocol semantics in the DSH runtime while delegating process
// placement/cancellation to a generic runner.Runner (SSH, Docker, or future
// environment adapters). No environment-specific DSH runtime type is created.
func (r *Runtime) StartInEnvironment(ctx context.Context, req agentruntime.StartRequest, exec runner.Runner) (agentruntime.Session, error) {
	if exec == nil {
		return nil, errors.New("dsh: environment runner is required")
	}
	if err := agentruntime.CheckStart(r.Descriptor(), req); err != nil {
		return nil, err
	}
	permission, err := r.permissionMode()
	if err != nil {
		return nil, err
	}
	patch, err := renderPatchOverlay(r.cfg.Provider, r.cfg.Model)
	if err != nil {
		return nil, err
	}

	bin := strings.TrimSpace(r.cfg.BinPath)
	if bin == "" {
		bin = "dsh"
	}
	version, err := exec.Run(ctx, runner.Command{Argv: []string{bin, "--version"}, Env: r.envList()})
	if err != nil {
		return nil, fmt.Errorf("dsh: environment --version transport failed: %w", err)
	}
	if version.ExitCode != 0 {
		return nil, fmt.Errorf("dsh: environment --version exited %d: %s", version.ExitCode, truncate(strings.TrimSpace(version.Stderr), 200))
	}
	cli := parseCLIIdentity(version.Stdout)
	if !cli.observed() {
		return nil, fmt.Errorf("dsh: environment --version output is not parseable: %q", truncate(strings.TrimSpace(version.Stdout), 200))
	}
	if err := checkVersionPin(r.cfg.PinnedVersion, cli); err != nil {
		return nil, err
	}

	r.mu.Lock()
	r.seq++
	n := r.seq
	r.mu.Unlock()

	// Keep runtime session state outside the candidate Git workspace so DSH_HOME
	// cannot dirty the ChangeSet. It remains under the same environment-visible
	// workspace root, which is also shared by the Docker workspace mount.
	home := path.Join(path.Dir(req.Workspace), ".tally-runtime", "dsh", fmt.Sprintf("session-%06d", n))
	if err := environmentMkdir(ctx, exec, home); err != nil {
		return nil, err
	}
	patchPath := ""
	if patch != "" {
		patchPath = path.Join(home, patchFileName)
		if err := environmentWritePrivateFile(ctx, exec, patchPath, patch); err != nil {
			_ = environmentRemove(ctx, exec, home)
			return nil, err
		}
	}

	return &environmentSession{
		runtime:    r,
		exec:       exec,
		bin:        bin,
		permission: permission,
		session:    agentruntime.SessionID(fmt.Sprintf("dsh-session-%06d", n)),
		workspace:  req.Workspace,
		home:       home,
		patchPath:  patchPath,
		runs:       make(map[agentruntime.RunID]*environmentRun),
	}, nil
}

type environmentSession struct {
	runtime    *Runtime
	exec       runner.Runner
	bin        string
	permission string
	session    agentruntime.SessionID
	workspace  string
	home       string
	patchPath  string

	execMu sync.Mutex
	mu     sync.Mutex
	runSeq int
	runs   map[agentruntime.RunID]*environmentRun
	closed bool
}

type environmentRun struct {
	id              agentruntime.RunID
	status          agentruntime.RunStatus
	result          agentruntime.AgentRunResult
	hasResult       bool
	cancelRequested bool
	cancel          context.CancelFunc
}

func (s *environmentSession) Execute(ctx context.Context, req agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	if err := req.Validate(); err != nil {
		return agentruntime.AgentRunResult{}, err
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return agentruntime.AgentRunResult{}, agentruntime.ErrSessionClosed
	}
	s.runSeq++
	id := agentruntime.RunID(fmt.Sprintf("run-%06d", s.runSeq))
	runCtx, cancel := context.WithCancel(ctx)
	r0 := &environmentRun{id: id, status: agentruntime.RunRunning, cancel: cancel}
	s.runs[id] = r0
	s.mu.Unlock()

	res := s.execTurn(runCtx, r0, req.Instruction)
	cancel()

	s.mu.Lock()
	r0.status = res.Status
	r0.result = res
	r0.hasResult = true
	r0.cancel = nil
	s.mu.Unlock()
	return res, nil
}

func (s *environmentSession) execTurn(ctx context.Context, r0 *environmentRun, instruction string) agentruntime.AgentRunResult {
	s.execMu.Lock()
	defer s.execMu.Unlock()

	if ctx.Err() != nil || s.cancelRequested(r0) {
		return s.finish(r0, agentruntime.RunCancelled, "dsh headless: run cancelled (context done before start)")
	}
	argv := []string{s.bin, "--profile", "headless"}
	if s.patchPath != "" {
		argv = append(argv, patchFlag, s.patchPath)
	}
	argv = append(argv, instruction)
	res, err := s.exec.Run(ctx, runner.Command{
		Argv: argv,
		Dir:  s.workspace,
		Env:  envPairs(sessionSpawnEnv(s.runtime.cfg.Env, s.home, s.permission)),
	})
	if ctx.Err() != nil || s.cancelRequested(r0) {
		return s.finish(r0, agentruntime.RunCancelled, "dsh headless: run cancelled")
	}
	if err != nil {
		return s.finish(r0, agentruntime.RunFailed, "dsh headless: environment transport failed: "+err.Error())
	}
	if res.ExitCode != 0 {
		msg := fmt.Sprintf("dsh headless: process exited %d", res.ExitCode)
		if tail := strings.TrimSpace(res.Stderr); tail != "" {
			msg += "; stderr: " + truncate(tail, 500)
		}
		return s.finish(r0, agentruntime.RunFailed, msg)
	}
	answer := strings.TrimSpace(res.Stdout)
	if answer == "" {
		return s.finish(r0, agentruntime.RunFailed, "dsh headless: successful exit without final stdout answer")
	}
	return s.finish(r0, agentruntime.RunCompleted, answer)
}

func (s *environmentSession) finish(r0 *environmentRun, status agentruntime.RunStatus, summary string) agentruntime.AgentRunResult {
	if status != agentruntime.RunCompleted {
		summary = logpipe.RedactFine(summary)
	}
	return agentruntime.AgentRunResult{RunID: r0.id, SessionID: s.session, Status: status, Summary: summary}
}

func (s *environmentSession) cancelRequested(r0 *environmentRun) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return r0.cancelRequested
}

func (s *environmentSession) Cancel(_ context.Context, id agentruntime.RunID) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return agentruntime.ErrSessionClosed
	}
	r0, ok := s.runs[id]
	if !ok {
		s.mu.Unlock()
		return agentruntime.ErrRunNotFound
	}
	if r0.status != agentruntime.RunRunning {
		s.mu.Unlock()
		return nil
	}
	r0.cancelRequested = true
	cancel := r0.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func (s *environmentSession) Result(_ context.Context, id agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return agentruntime.AgentRunResult{}, agentruntime.ErrSessionClosed
	}
	r0, ok := s.runs[id]
	if !ok {
		return agentruntime.AgentRunResult{}, agentruntime.ErrRunNotFound
	}
	if !r0.hasResult {
		return agentruntime.AgentRunResult{RunID: id, SessionID: s.session, Status: agentruntime.RunRunning}, nil
	}
	return r0.result, nil
}

func (s *environmentSession) Events(_ context.Context) (agentruntime.EventStream, error) {
	if err := agentruntime.CheckEvents(s.runtime.Descriptor()); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, agentruntime.ErrSessionClosed
	}
	return nil, agentruntime.ErrStreamingNotDeclared
}

func (s *environmentSession) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	var cancels []context.CancelFunc
	for _, r0 := range s.runs {
		if r0.status == agentruntime.RunRunning {
			r0.cancelRequested = true
			if r0.cancel != nil {
				cancels = append(cancels, r0.cancel)
			}
		}
	}
	s.mu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
	if s.home != "" {
		if err := environmentRemove(ctx, s.exec, s.home); err != nil && ctx.Err() == nil {
			return err
		}
	}
	return nil
}

func environmentMkdir(ctx context.Context, exec runner.Runner, dir string) error {
	res, err := exec.Run(ctx, runner.Command{Argv: []string{"mkdir", "-p", "--", dir}})
	if err != nil {
		return fmt.Errorf("dsh: create environment session home: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("dsh: create environment session home exited %d: %s", res.ExitCode, truncate(strings.TrimSpace(res.Stderr), 200))
	}
	return nil
}

func environmentWritePrivateFile(ctx context.Context, exec runner.Runner, filePath, content string) error {
	// The patch carries requested model/provider configuration, not credentials.
	// Passing it as an argv value avoids inventing a file-transfer protocol at the
	// Environment boundary while keeping the file mode private.
	res, err := exec.Run(ctx, runner.Command{Argv: []string{
		"sh", "-c", `umask 077; printf '%s' "$1" > "$2"`, "tally-dsh-patch", content, filePath,
	}})
	if err != nil {
		return fmt.Errorf("dsh: write environment patch: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("dsh: write environment patch exited %d: %s", res.ExitCode, truncate(strings.TrimSpace(res.Stderr), 200))
	}
	return nil
}

func environmentRemove(ctx context.Context, exec runner.Runner, target string) error {
	res, err := exec.Run(ctx, runner.Command{Argv: []string{"rm", "-rf", "--", target}})
	if err != nil {
		return fmt.Errorf("dsh: remove environment session state: %w", err)
	}
	if res.ExitCode != 0 {
		return fmt.Errorf("dsh: remove environment session state exited %d: %s", res.ExitCode, truncate(strings.TrimSpace(res.Stderr), 200))
	}
	return nil
}
