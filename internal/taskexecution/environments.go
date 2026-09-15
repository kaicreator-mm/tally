package taskexecution

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
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

// EnvironmentProfile is discovered/declared capability output. Secret values are
// never stored here; Secrets only records availability by logical name.
type EnvironmentProfile struct {
	OS            string            `json:"os,omitempty"`
	Architecture  string            `json:"architecture,omitempty"`
	Tools         map[string]string `json:"tools,omitempty"`
	Git           bool              `json:"git"`
	Docker        bool              `json:"docker"`
	Network       bool              `json:"network"`
	SecretsPolicy string            `json:"secrets_policy,omitempty"`
	Secrets       map[string]bool   `json:"secrets,omitempty"`
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
	EnvironmentCapabilityValidator
	ProcessRunner() runner.Runner
	RunProcess(context.Context, runner.Command) (runner.Result, error)
	Discover(context.Context) (EnvironmentProfile, error)
	InspectWorkspace(context.Context, WorkspaceRef) (WorkspaceInspection, error)
	CleanupWorkspace(context.Context, WorkspaceRef) error
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
	if err := a.validateWorkspaceTarget(w); err != nil {
		return ResolvedWorkspace{}, err
	}
	if strings.TrimSpace(w.Path) == "" {
		return ResolvedWorkspace{}, &EnvironmentError{Kind: FailureWorkspaceUnavailable, Environment: a.kind, Target: w.Target, Operation: "resolve", Err: errors.New("environment-local workspace path is required")}
	}
	if err := a.verifyWorkspace(ctx, w); err != nil {
		var typed *EnvironmentError
		if a.recreate == nil || !errors.As(err, &typed) || typed.Kind != FailureWorkspaceUnavailable {
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

func (a *environmentAdapter) validateWorkspaceTarget(w WorkspaceRef) error {
	if err := w.Validate(); err != nil {
		return err
	}
	if w.Environment != a.kind {
		return ErrUnsupportedEnvironment
	}
	if a.target != "" && w.Target != a.target {
		return &EnvironmentError{Kind: FailureEnvironmentUnavailable, Environment: a.kind, Target: w.Target, Operation: "target.validate", Err: errors.New("target does not match configured environment")}
	}
	return nil
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
	if err := a.validateWorkspaceTarget(w); err != nil {
		return WorkspaceInspection{}, err
	}
	if strings.TrimSpace(w.Path) == "" {
		return WorkspaceInspection{}, &EnvironmentError{Kind: FailureWorkspaceUnavailable, Environment: a.kind, Target: w.Target, Operation: "workspace.inspect", Err: errors.New("workspace path is required")}
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

func (a *environmentAdapter) CleanupWorkspace(ctx context.Context, w WorkspaceRef) error {
	if err := a.validateWorkspaceTarget(w); err != nil {
		return err
	}
	if unsafeCleanupPath(a.kind, w.Path) {
		return &EnvironmentError{Kind: FailureCleanup, Environment: a.kind, Target: w.Target, Operation: "workspace.cleanup", Err: fmt.Errorf("refusing unsafe workspace cleanup path %q", w.Path)}
	}
	if a.exec == nil {
		return &EnvironmentError{Kind: FailureEnvironmentUnavailable, Environment: a.kind, Target: w.Target, Operation: "workspace.cleanup", Err: errors.New("process runner is not configured")}
	}
	res, err := a.exec.Run(ctx, runner.Command{Argv: []string{"rm", "-rf", "--", w.Path}})
	if err != nil {
		return &EnvironmentError{Kind: FailureCleanup, Environment: a.kind, Target: w.Target, Operation: "workspace.cleanup", Err: err}
	}
	if res.ExitCode != 0 {
		return &EnvironmentError{Kind: FailureCleanup, Environment: a.kind, Target: w.Target, Operation: "workspace.cleanup", Err: fmt.Errorf("cleanup exited %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr))}
	}
	return nil
}

func (a *environmentAdapter) Discover(ctx context.Context) (EnvironmentProfile, error) {
	if a.exec == nil {
		return EnvironmentProfile{}, &EnvironmentError{Kind: FailureEnvironmentUnavailable, Environment: a.kind, Target: a.target, Operation: "capability.discover", Err: errors.New("process runner is not configured")}
	}
	p := cloneEnvironmentProfile(a.declared)
	if res, err := a.exec.Run(ctx, runner.Command{Argv: []string{"uname", "-s"}}); err == nil && res.ExitCode == 0 {
		p.OS = strings.TrimSpace(res.Stdout)
	}
	if res, err := a.exec.Run(ctx, runner.Command{Argv: []string{"uname", "-m"}}); err == nil && res.ExitCode == 0 {
		p.Architecture = normalizeArchitecture(strings.TrimSpace(res.Stdout))
	}
	tools := append([]string{"git", "docker"}, a.probeTools...)
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
			if tool == "git" {
				p.Git = true
			}
			if tool == "docker" {
				p.Docker = true
			}
		}
	}
	return p, nil
}

func cloneEnvironmentProfile(in EnvironmentProfile) EnvironmentProfile {
	out := in
	out.Tools = make(map[string]string, len(in.Tools))
	for k, v := range in.Tools {
		out.Tools[k] = v
	}
	out.Secrets = make(map[string]bool, len(in.Secrets))
	for k, v := range in.Secrets {
		out.Secrets[k] = v
	}
	out.Resources = make(map[string]string, len(in.Resources))
	for k, v := range in.Resources {
		out.Resources[k] = v
	}
	out.Architecture = normalizeArchitecture(out.Architecture)
	return out
}

func (a *environmentAdapter) ValidateCapabilities(ctx context.Context, required []EnvironmentCapability) error {
	static := map[EnvironmentCapability]bool{}
	for _, c := range a.Capabilities() {
		static[c] = true
	}
	needsProfile := false
	for _, c := range required {
		if !static[c] {
			needsProfile = true
			break
		}
	}
	var profile EnvironmentProfile
	var err error
	if needsProfile {
		profile, err = a.Discover(ctx)
		if err != nil {
			return err
		}
		for _, c := range required {
			tool, ok := strings.CutPrefix(strings.TrimSpace(string(c)), "tool:")
			tool = strings.TrimSpace(tool)
			if !ok || tool == "" {
				continue
			}
			if _, exists := profile.Tools[tool]; exists {
				continue
			}
			res, runErr := a.exec.Run(ctx, runner.Command{Argv: []string{tool, "--version"}})
			if runErr == nil && res.ExitCode == 0 {
				value := strings.TrimSpace(res.Stdout)
				if value == "" {
					value = strings.TrimSpace(res.Stderr)
				}
				profile.Tools[tool] = value
			}
		}
	}
	var missing []EnvironmentCapability
	for _, c := range required {
		if static[c] || profileHasCapability(profile, c) {
			continue
		}
		missing = append(missing, c)
	}
	if len(missing) > 0 {
		return &EnvironmentCapabilityError{Missing: missing}
	}
	return nil
}

func profileHasCapability(p EnvironmentProfile, capability EnvironmentCapability) bool {
	raw := strings.TrimSpace(string(capability))
	switch raw {
	case "git":
		return p.Git
	case "docker":
		return p.Docker
	case "network":
		return p.Network
	}
	if value, ok := strings.CutPrefix(raw, "os:"); ok {
		return strings.EqualFold(strings.TrimSpace(p.OS), strings.TrimSpace(value))
	}
	if value, ok := strings.CutPrefix(raw, "arch:"); ok {
		return normalizeArchitecture(p.Architecture) == normalizeArchitecture(value)
	}
	if value, ok := strings.CutPrefix(raw, "tool:"); ok {
		_, exists := p.Tools[strings.TrimSpace(value)]
		return exists
	}
	if value, ok := strings.CutPrefix(raw, "secret:"); ok {
		return p.Secrets[strings.TrimSpace(value)]
	}
	if value, ok := strings.CutPrefix(raw, "resource:"); ok {
		name, expected, hasExpected := strings.Cut(value, "=")
		actual, exists := p.Resources[strings.TrimSpace(name)]
		if !exists {
			return false
		}
		return !hasExpected || strings.TrimSpace(actual) == strings.TrimSpace(expected)
	}
	return false
}

func normalizeArchitecture(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "x86_64", "x64", "amd64":
		return "amd64"
	case "aarch64", "arm64":
		return "arm64"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

type SSHEnvironment struct{ adapter environmentAdapter }

func NewSSHEnvironment(target string, exec runner.Runner, declared EnvironmentProfile, recreate WorkspaceRecreator) *SSHEnvironment {
	return &SSHEnvironment{adapter: environmentAdapter{kind: EnvironmentSSH, target: target, exec: exec, declared: declared, recreate: recreate}}
}
func (e *SSHEnvironment) Kind() EnvironmentKind                  { return e.adapter.Kind() }
func (e *SSHEnvironment) Capabilities() []EnvironmentCapability { return e.adapter.Capabilities() }
func (e *SSHEnvironment) Resolve(ctx context.Context, w WorkspaceRef) (ResolvedWorkspace, error) {
	return e.adapter.Resolve(ctx, w)
}
func (e *SSHEnvironment) ProcessRunner() runner.Runner { return e.adapter.ProcessRunner() }
func (e *SSHEnvironment) RunProcess(ctx context.Context, cmd runner.Command) (runner.Result, error) {
	return e.adapter.RunProcess(ctx, cmd)
}
func (e *SSHEnvironment) Discover(ctx context.Context) (EnvironmentProfile, error) {
	return e.adapter.Discover(ctx)
}
func (e *SSHEnvironment) ValidateCapabilities(ctx context.Context, c []EnvironmentCapability) error {
	return e.adapter.ValidateCapabilities(ctx, c)
}
func (e *SSHEnvironment) InspectWorkspace(ctx context.Context, w WorkspaceRef) (WorkspaceInspection, error) {
	return e.adapter.InspectWorkspace(ctx, w)
}
func (e *SSHEnvironment) CleanupWorkspace(ctx context.Context, w WorkspaceRef) error {
	return e.adapter.CleanupWorkspace(ctx, w)
}

type DockerEnvironment struct{ adapter environmentAdapter }

func NewDockerEnvironment(target string, exec runner.Runner, declared EnvironmentProfile, recreate WorkspaceRecreator) *DockerEnvironment {
	declared.Docker = true
	return &DockerEnvironment{adapter: environmentAdapter{kind: EnvironmentDocker, target: target, exec: exec, declared: declared, recreate: recreate}}
}
func (e *DockerEnvironment) Kind() EnvironmentKind                  { return e.adapter.Kind() }
func (e *DockerEnvironment) Capabilities() []EnvironmentCapability { return e.adapter.Capabilities() }
func (e *DockerEnvironment) Resolve(ctx context.Context, w WorkspaceRef) (ResolvedWorkspace, error) {
	return e.adapter.Resolve(ctx, w)
}
func (e *DockerEnvironment) ProcessRunner() runner.Runner { return e.adapter.ProcessRunner() }
func (e *DockerEnvironment) RunProcess(ctx context.Context, cmd runner.Command) (runner.Result, error) {
	return e.adapter.RunProcess(ctx, cmd)
}
func (e *DockerEnvironment) Discover(ctx context.Context) (EnvironmentProfile, error) {
	return e.adapter.Discover(ctx)
}
func (e *DockerEnvironment) ValidateCapabilities(ctx context.Context, c []EnvironmentCapability) error {
	return e.adapter.ValidateCapabilities(ctx, c)
}
func (e *DockerEnvironment) InspectWorkspace(ctx context.Context, w WorkspaceRef) (WorkspaceInspection, error) {
	return e.adapter.InspectWorkspace(ctx, w)
}
func (e *DockerEnvironment) CleanupWorkspace(ctx context.Context, w WorkspaceRef) error {
	return e.adapter.CleanupWorkspace(ctx, w)
}

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
func localEnvironmentAdapter() environmentAdapter {
	return environmentAdapter{
		kind:   EnvironmentLocal,
		target: "local",
		exec:   runner.LocalRunner{},
		declared: EnvironmentProfile{
			OS:           runtime.GOOS,
			Architecture: runtime.GOARCH,
		},
	}
}
func (LocalEnvironment) Discover(ctx context.Context) (EnvironmentProfile, error) {
	a := localEnvironmentAdapter()
	return a.Discover(ctx)
}
func (LocalEnvironment) ValidateCapabilities(ctx context.Context, c []EnvironmentCapability) error {
	a := localEnvironmentAdapter()
	return a.ValidateCapabilities(ctx, c)
}
func (LocalEnvironment) InspectWorkspace(ctx context.Context, w WorkspaceRef) (WorkspaceInspection, error) {
	a := localEnvironmentAdapter()
	a.target = w.Target
	return a.InspectWorkspace(ctx, w)
}
func (LocalEnvironment) CleanupWorkspace(_ context.Context, w WorkspaceRef) error {
	if err := w.Validate(); err != nil {
		return err
	}
	if w.Environment != EnvironmentLocal {
		return ErrUnsupportedEnvironment
	}
	if unsafeCleanupPath(EnvironmentLocal, w.Path) {
		return &EnvironmentError{Kind: FailureCleanup, Environment: EnvironmentLocal, Target: w.Target, Operation: "workspace.cleanup", Err: fmt.Errorf("refusing unsafe workspace cleanup path %q", w.Path)}
	}
	if err := os.RemoveAll(w.Path); err != nil {
		return &EnvironmentError{Kind: FailureCleanup, Environment: EnvironmentLocal, Target: w.Target, Operation: "workspace.cleanup", Err: err}
	}
	return nil
}
