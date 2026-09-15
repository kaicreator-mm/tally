package taskexecution

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tally/internal/runner"
)

type EnvironmentFailureKind string

const (
	FailureEnvironmentUnavailable EnvironmentFailureKind = "environment_unavailable"
	FailureWorkspaceUnavailable   EnvironmentFailureKind = "workspace_unavailable"
	FailureProcess                EnvironmentFailureKind = "process_failed"
	FailureCleanup                EnvironmentFailureKind = "cleanup_failed"
)

// EnvironmentError is a typed failure at the Environment boundary.
type EnvironmentError struct {
	Kind        EnvironmentFailureKind
	Environment EnvironmentKind
	Target      string
	Operation   string
	Err         error
}

func (e *EnvironmentError) Error() string {
	return fmt.Sprintf("taskexecution: %s %s target=%s: %v", e.Environment, e.Operation, e.Target, e.Err)
}
func (e *EnvironmentError) Unwrap() error { return e.Err }

// EnvironmentProfile is discovery output, not a second source of execution state.
type EnvironmentProfile struct {
	OS            string            `json:"os,omitempty"`
	Tools         map[string]string `json:"tools,omitempty"`
	Docker        bool              `json:"docker"`
	Network       bool              `json:"network"`
	SecretsPolicy string            `json:"secrets_policy,omitempty"`
	Resources     map[string]string `json:"resources,omitempty"`
}

type WorkspaceInspection struct {
	Revision string `json:"revision,omitempty"`
	Dirty    bool   `json:"dirty"`
}

type WorkspaceRecreator func(context.Context, WorkspaceRef) error

// ProcessExecutionEnvironment extends (rather than changes) the frozen v1.5.0
// ExecutionEnvironment contract. Runtime adapters can consume ProcessRunner
// without acquiring SSH/Docker-specific classes.
type ProcessExecutionEnvironment interface {
	ExecutionEnvironment
	ProcessRunner() runner.Runner
	RunProcess(context.Context, runner.Command) (runner.Result, error)
	Discover(context.Context) (EnvironmentProfile, error)
	InspectWorkspace(context.Context, WorkspaceRef) (WorkspaceInspection, error)
}

type environmentAdapter struct {
	kind       EnvironmentKind
	target     string
	exec       runner.Runner
	declared   EnvironmentProfile
	probeTools []string
	recreate   WorkspaceRecreator
}

func (a *environmentAdapter) Kind() EnvironmentKind { return a.kind }
func (a *environmentAdapter) Capabilities() []EnvironmentCapability {
	return []EnvironmentCapability{CapabilityCancel, CapabilityProcess, CapabilityWorkspaceRead, CapabilityWorkspaceWrite}
}
func (a *environmentAdapter) ProcessRunner() runner.Runner { return a.exec }

func (a *environmentAdapter) RunProcess(ctx context.Context, cmd runner.Command) (runner.Result, error) {
	if a.exec == nil {
		return runner.Result{}, &EnvironmentError{Kind: FailureEnvironmentUnavailable, Environment: a.kind, Target: a.target, Operation: "process.run", Err: errors.New("process runner is not configured")}
	}
	res, err := a.exec.Run(ctx, cmd)
	if err != nil {
		return res, &EnvironmentError{Kind: FailureProcess, Environment: a.kind, Target: a.target, Operation: "process.run", Err: err}
	}
	return res, nil
}

func (a *environmentAdapter) Resolve(ctx context.Context, w WorkspaceRef) (ResolvedWorkspace, error) {
	if err := w.Validate(); err != nil {
		return ResolvedWorkspace{}, err
	}
	if w.Environment != a.kind {
		return ResolvedWorkspace{}, ErrUnsupportedEnvironment
	}
	if a.target != "" && w.Target != a.target {
		return ResolvedWorkspace{}, &EnvironmentError{Kind: FailureEnvironmentUnavailable, Environment: a.kind, Target: w.Target, Operation: "resolve", Err: errors.New("target does not match configured environment")}
	}
	if strings.TrimSpace(w.Path) == "" {
		return ResolvedWorkspace{}, &EnvironmentError{Kind: FailureWorkspaceUnavailable, Environment: a.kind, Target: w.Target, Operation: "resolve", Err: errors.New("environment-local workspace path is required")}
	}
	if err := a.verifyWorkspace(ctx, w); err != nil {
		if a.recreate == nil {
			return ResolvedWorkspace{}, err
		}
		if recErr := a.recreate(ctx, w); recErr != nil {
			return ResolvedWorkspace{}, &EnvironmentError{Kind: FailureWorkspaceUnavailable, Environment: a.kind, Target: w.Target, Operation: "recreate", Err: recErr}
		}
		if err := a.verifyWorkspace(ctx, w); err != nil {
			return ResolvedWorkspace{}, err
		}
	}
	return ResolvedWorkspace{Location: w.Path}, nil
}

func (a *environmentAdapter) verifyWorkspace(ctx context.Context, w WorkspaceRef) error {
	if a.exec == nil {
		return &EnvironmentError{Kind: FailureEnvironmentUnavailable, Environment: a.kind, Target: w.Target, Operation: "workspace.verify", Err: errors.New("process runner is not configured")}
	}
	res, err := a.exec.Run(ctx, runner.Command{Argv: []string{"git", "-C", w.Path, "cat-file", "-e", w.BaseRevision + "^{commit}"}})
	if err != nil {
		return &EnvironmentError{Kind: FailureEnvironmentUnavailable, Environment: a.kind, Target: w.Target, Operation: "workspace.verify", Err: err}
	}
	if res.ExitCode != 0 {
		return &EnvironmentError{Kind: FailureWorkspaceUnavailable, Environment: a.kind, Target: w.Target, Operation: "workspace.verify", Err: fmt.Errorf("git base revision check exited %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr))}
	}
	return nil
}

func (a *environmentAdapter) InspectWorkspace(ctx context.Context, w WorkspaceRef) (WorkspaceInspection, error) {
	if err := w.Validate(); err != nil {
		return WorkspaceInspection{}, err
	}
	if w.Environment != a.kind {
		return WorkspaceInspection{}, ErrUnsupportedEnvironment
	}
	if a.exec == nil {
		return WorkspaceInspection{}, &EnvironmentError{Kind: FailureEnvironmentUnavailable, Environment: a.kind, Target: w.Target, Operation: "workspace.inspect", Err: errors.New("process runner is not configured")}
	}
	rev, err := a.exec.Run(ctx, runner.Command{Argv: []string{"git", "-C", w.Path, "rev-parse", "HEAD"}})
	if err != nil {
		return WorkspaceInspection{}, &EnvironmentError{Kind: FailureEnvironmentUnavailable, Environment: a.kind, Target: w.Target, Operation: "workspace.inspect", Err: err}
	}
	if rev.ExitCode != 0 {
		return WorkspaceInspection{}, &EnvironmentError{Kind: FailureWorkspaceUnavailable, Environment: a.kind, Target: w.Target, Operation: "workspace.inspect", Err: fmt.Errorf("git rev-parse exited %d", rev.ExitCode)}
	}
	status, err := a.exec.Run(ctx, runner.Command{Argv: []string{"git", "-C", w.Path, "status", "--porcelain=v1"}})
	if err != nil {
		return WorkspaceInspection{}, &EnvironmentError{Kind: FailureEnvironmentUnavailable, Environment: a.kind, Target: w.Target, Operation: "workspace.inspect", Err: err}
	}
	if status.ExitCode != 0 {
		return WorkspaceInspection{}, &EnvironmentError{Kind: FailureWorkspaceUnavailable, Environment: a.kind, Target: w.Target, Operation: "workspace.inspect", Err: fmt.Errorf("git status exited %d", status.ExitCode)}
	}
	return WorkspaceInspection{Revision: strings.TrimSpace(rev.Stdout), Dirty: strings.TrimSpace(status.Stdout) != ""}, nil
}

func (a *environmentAdapter) Discover(ctx context.Context) (EnvironmentProfile, error) {
	if a.exec == nil {
		return EnvironmentProfile{}, &EnvironmentError{Kind: FailureEnvironmentUnavailable, Environment: a.kind, Target: a.target, Operation: "capability.discover", Err: errors.New("process runner is not configured")}
	}
	p := a.declared
	copyTools := make(map[string]string, len(p.Tools))
	for k, v := range p.Tools {
		copyTools[k] = v
	}
	p.Tools = copyTools
	if res, err := a.exec.Run(ctx, runner.Command{Argv: []string{"uname", "-s"}}); err == nil && res.ExitCode == 0 {
		p.OS = strings.TrimSpace(res.Stdout)
	}
	tools := append([]string{"git"}, a.probeTools...)
	seen := map[string]bool{}
	for _, tool := range tools {
		if tool == "" || seen[tool] {
			continue
		}
		seen[tool] = true
		res, err := a.exec.Run(ctx, runner.Command{Argv: []string{tool, "--version"}})
		if err == nil && res.ExitCode == 0 {
			value := strings.TrimSpace(res.Stdout)
			if value == "" {
				value = strings.TrimSpace(res.Stderr)
			}
			p.Tools[tool] = value
		}
	}
	return p, nil
}

type SSHEnvironment struct{ adapter environmentAdapter }

func NewSSHEnvironment(target string, exec runner.Runner, declared EnvironmentProfile, recreate WorkspaceRecreator) *SSHEnvironment {
	return &SSHEnvironment{adapter: environmentAdapter{kind: EnvironmentSSH, target: target, exec: exec, declared: declared, probeTools: []string{"docker"}, recreate: recreate}}
}
func (e *SSHEnvironment) Kind() EnvironmentKind { return e.adapter.Kind() }
func (e *SSHEnvironment) Capabilities() []EnvironmentCapability { return e.adapter.Capabilities() }
func (e *SSHEnvironment) Resolve(ctx context.Context, w WorkspaceRef) (ResolvedWorkspace, error) { return e.adapter.Resolve(ctx, w) }
func (e *SSHEnvironment) ProcessRunner() runner.Runner { return e.adapter.ProcessRunner() }
func (e *SSHEnvironment) RunProcess(ctx context.Context, cmd runner.Command) (runner.Result, error) { return e.adapter.RunProcess(ctx, cmd) }
func (e *SSHEnvironment) Discover(ctx context.Context) (EnvironmentProfile, error) { return e.adapter.Discover(ctx) }
func (e *SSHEnvironment) InspectWorkspace(ctx context.Context, w WorkspaceRef) (WorkspaceInspection, error) { return e.adapter.InspectWorkspace(ctx, w) }

type DockerEnvironment struct{ adapter environmentAdapter }

func NewDockerEnvironment(target string, exec runner.Runner, declared EnvironmentProfile, recreate WorkspaceRecreator) *DockerEnvironment {
	declared.Docker = true
	return &DockerEnvironment{adapter: environmentAdapter{kind: EnvironmentDocker, target: target, exec: exec, declared: declared, recreate: recreate}}
}
func (e *DockerEnvironment) Kind() EnvironmentKind { return e.adapter.Kind() }
func (e *DockerEnvironment) Capabilities() []EnvironmentCapability { return e.adapter.Capabilities() }
func (e *DockerEnvironment) Resolve(ctx context.Context, w WorkspaceRef) (ResolvedWorkspace, error) { return e.adapter.Resolve(ctx, w) }
func (e *DockerEnvironment) ProcessRunner() runner.Runner { return e.adapter.ProcessRunner() }
func (e *DockerEnvironment) RunProcess(ctx context.Context, cmd runner.Command) (runner.Result, error) { return e.adapter.RunProcess(ctx, cmd) }
func (e *DockerEnvironment) Discover(ctx context.Context) (EnvironmentProfile, error) { return e.adapter.Discover(ctx) }
func (e *DockerEnvironment) InspectWorkspace(ctx context.Context, w WorkspaceRef) (WorkspaceInspection, error) { return e.adapter.InspectWorkspace(ctx, w) }

// Local gains the same optional v1.5.2 extension without changing the frozen
// ExecutionEnvironment interface or its zero-value compatibility behavior.
func (LocalEnvironment) ProcessRunner() runner.Runner { return runner.LocalRunner{} }
func (LocalEnvironment) RunProcess(ctx context.Context, cmd runner.Command) (runner.Result, error) {
	res, err := (runner.LocalRunner{}).Run(ctx, cmd)
	if err != nil {
		return res, &EnvironmentError{Kind: FailureProcess, Environment: EnvironmentLocal, Target: "local", Operation: "process.run", Err: err}
	}
	return res, nil
}
func (LocalEnvironment) Discover(ctx context.Context) (EnvironmentProfile, error) {
	a := environmentAdapter{kind: EnvironmentLocal, target: "local", exec: runner.LocalRunner{}}
	return a.Discover(ctx)
}
func (LocalEnvironment) InspectWorkspace(ctx context.Context, w WorkspaceRef) (WorkspaceInspection, error) {
	a := environmentAdapter{kind: EnvironmentLocal, target: w.Target, exec: runner.LocalRunner{}}
	return a.InspectWorkspace(ctx, w)
}
