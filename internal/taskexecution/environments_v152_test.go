package taskexecution

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"tally/internal/agentruntime"
	"tally/internal/engineering"
	"tally/internal/runner"
)

type v152ScriptedRunner struct {
	results []runner.Result
	errs    []error
	calls   []runner.Command
}

func (s *v152ScriptedRunner) Run(_ context.Context, c runner.Command) (runner.Result, error) {
	s.calls = append(s.calls, c)
	i := len(s.calls) - 1
	var r runner.Result
	var err error
	if i < len(s.results) {
		r = s.results[i]
	}
	if i < len(s.errs) {
		err = s.errs[i]
	}
	return r, err
}

type v152Runtime struct {
	starts []agentruntime.StartRequest
	bound  runner.Runner
}

func (r *v152Runtime) Descriptor() agentruntime.RuntimeDescriptor {
	return agentruntime.RuntimeDescriptor{Name: "v152-fake", Capabilities: agentruntime.NewCapabilitySet()}
}
func (r *v152Runtime) Start(_ context.Context, s agentruntime.StartRequest) (agentruntime.Session, error) {
	r.starts = append(r.starts, s)
	return &v152Session{}, nil
}
func (r *v152Runtime) StartInEnvironment(_ context.Context, s agentruntime.StartRequest, exec runner.Runner) (agentruntime.Session, error) {
	r.starts = append(r.starts, s)
	r.bound = exec
	return &v152Session{}, nil
}

type v152LocalOnlyRuntime struct{ starts int }

func (r *v152LocalOnlyRuntime) Descriptor() agentruntime.RuntimeDescriptor {
	return agentruntime.RuntimeDescriptor{Name: "local-only", Capabilities: agentruntime.NewCapabilitySet()}
}
func (r *v152LocalOnlyRuntime) Start(context.Context, agentruntime.StartRequest) (agentruntime.Session, error) {
	r.starts++
	return &v152Session{}, nil
}

type v152Session struct{}

func (*v152Session) Execute(context.Context, agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	return agentruntime.AgentRunResult{RunID: "r", SessionID: "s", Status: agentruntime.RunCompleted}, nil
}
func (*v152Session) Cancel(context.Context, agentruntime.RunID) error { return nil }
func (*v152Session) Events(context.Context) (agentruntime.EventStream, error) {
	return nil, errors.New("unused")
}
func (*v152Session) Result(context.Context, agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	return agentruntime.AgentRunResult{}, nil
}
func (*v152Session) Close(context.Context) error { return nil }

func v152RemoteReq() TaskExecutionRequest {
	return TaskExecutionRequest{
		TaskID:             "T-v152",
		FrozenTaskContract: engineering.ArtifactRef{ID: "contract", Kind: engineering.ArtifactTaskContract, URI: "git://task"},
		Runtime:            RuntimeRef{ID: "fake"},
		Environment:        EnvironmentRef{Kind: EnvironmentSSH, Target: "build"},
		Workspace:          WorkspaceRef{Environment: EnvironmentSSH, Target: "build", Identity: "w", Path: "/remote/w", BaseRevision: "abc"},
		Instruction:        "go",
	}
}

func TestV152SSHResolveUsesEnvironmentPathWithoutLocalPath(t *testing.T) {
	f := &v152ScriptedRunner{results: []runner.Result{{ExitCode: 0}}}
	env := NewSSHEnvironment("build", f, EnvironmentProfile{}, nil)
	got, err := env.Resolve(context.Background(), v152RemoteReq().Workspace)
	if err != nil {
		t.Fatal(err)
	}
	if got.Location != "/remote/w" || got.LocalPath != "" {
		t.Fatalf("got=%+v", got)
	}
	want := []string{"git", "-C", "/remote/w", "cat-file", "-e", "abc^{commit}"}
	if !reflect.DeepEqual(f.calls[0].Argv, want) {
		t.Fatalf("call=%v", f.calls[0].Argv)
	}
}

func TestV152ExecutorBindsRemoteRuntimeToEnvironmentRunner(t *testing.T) {
	f := &v152ScriptedRunner{results: []runner.Result{{ExitCode: 0}, {ExitCode: 0}}}
	env := NewSSHEnvironment("build", f, EnvironmentProfile{}, nil)
	runtime := &v152Runtime{}
	ex, _ := NewRuntimeExecutor(runtime, env)
	bindReq := v152RemoteReq()
	bindReq.Runtime.ID = "v152-fake"
	if _, err := ex.Execute(context.Background(), bindReq); err != nil {
		t.Fatal(err)
	}
	if len(runtime.starts) != 1 || runtime.starts[0].Workspace != "/remote/w" {
		t.Fatalf("starts=%+v", runtime.starts)
	}
	if runtime.bound != f {
		t.Fatal("remote runtime was not bound to selected environment runner")
	}
	if !reflect.DeepEqual(f.calls[0].Argv, []string{"true"}) {
		t.Fatalf("preflight=%v", f.calls[0].Argv)
	}
}

func TestV152NonLocalRuntimeCannotSilentlyFallBackToLocal(t *testing.T) {
	f := &v152ScriptedRunner{results: []runner.Result{{ExitCode: 0}, {ExitCode: 0}}}
	env := NewSSHEnvironment("build", f, EnvironmentProfile{}, nil)
	runtime := &v152LocalOnlyRuntime{}
	ex, _ := NewRuntimeExecutor(runtime, env)
	localReq := v152RemoteReq()
	localReq.Runtime.ID = "local-only"
	_, err := ex.Execute(context.Background(), localReq)
	var typed *EnvironmentError
	if !errors.As(err, &typed) || typed.Kind != FailureProcess {
		t.Fatalf("err=%T %v", err, err)
	}
	if runtime.starts != 0 {
		t.Fatal("local Start was called for a non-local environment")
	}
}

func TestV152NonLocalPreflightClassifiesTransportFailure(t *testing.T) {
	f := &v152ScriptedRunner{errs: []error{errors.New("ssh unavailable")}}
	env := NewSSHEnvironment("build", f, EnvironmentProfile{}, nil)
	runtime := &v152Runtime{}
	ex, _ := NewRuntimeExecutor(runtime, env)
	preflightReq := v152RemoteReq()
	preflightReq.Runtime.ID = "v152-fake"
	_, err := ex.Execute(context.Background(), preflightReq)
	var typed *EnvironmentError
	if !errors.As(err, &typed) || typed.Kind != FailureEnvironmentUnavailable || typed.Operation != "environment.preflight" {
		t.Fatalf("err=%T %v", err, err)
	}
	if len(runtime.starts) != 0 {
		t.Fatal("runtime started despite unavailable environment")
	}
	if len(f.calls) != 1 || !reflect.DeepEqual(f.calls[0].Argv, []string{"true"}) {
		t.Fatalf("calls=%v", f.calls)
	}
}

func TestV152LostWorkspaceCanRecreate(t *testing.T) {
	f := &v152ScriptedRunner{results: []runner.Result{{ExitCode: 1, Stderr: "missing"}, {ExitCode: 0}}}
	count := 0
	env := NewSSHEnvironment("build", f, EnvironmentProfile{}, func(context.Context, WorkspaceRef) error { count++; return nil })
	if _, err := env.Resolve(context.Background(), v152RemoteReq().Workspace); err != nil {
		t.Fatal(err)
	}
	if count != 1 || len(f.calls) != 2 {
		t.Fatalf("recreate=%d calls=%d", count, len(f.calls))
	}
}

func TestV152EnvironmentCapabilityErrorIsTyped(t *testing.T) {
	env := &v152LimitedEnvironment{}
	err := checkEnvironmentCapabilities(context.Background(), env, []EnvironmentCapability{CapabilityCancel})
	var typed *EnvironmentCapabilityError
	if !errors.As(err, &typed) {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestV152ProfileCapabilityDiscoveryCoversArchitectureToolingAndSecrets(t *testing.T) {
	f := &v152ScriptedRunner{results: []runner.Result{
		{Stdout: "Linux\n"},
		{Stdout: "x86_64\n"},
		{Stdout: "git version 2.45.0\n"},
		{Stdout: "Docker version 29.0.0\n"},
	}}
	env := NewSSHEnvironment("build", f, EnvironmentProfile{
		Network:   true,
		Secrets:   map[string]bool{"GITHUB_TOKEN": true},
		Resources: map[string]string{"cpu": "4"},
	}, nil)
	required := []EnvironmentCapability{
		"os:linux", "arch:x86_64", "tool:git", "git", "docker", "network", "secret:GITHUB_TOKEN", "resource:cpu=4",
	}
	if err := env.ValidateCapabilities(context.Background(), required); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 4 {
		t.Fatalf("discovery calls=%d", len(f.calls))
	}
}

func TestV152ProfileMismatchBlocksBeforeRuntimeStart(t *testing.T) {
	f := &v152ScriptedRunner{results: []runner.Result{
		{ExitCode: 0},
		{Stdout: "Linux\n"},
		{Stdout: "x86_64\n"},
		{Stdout: "git version 2.45.0\n"},
		{ExitCode: 127},
	}}
	env := NewSSHEnvironment("build", f, EnvironmentProfile{}, nil)
	runtime := &v152Runtime{}
	ex, _ := NewRuntimeExecutor(runtime, env)
	req := v152RemoteReq()
	req.Runtime.ID = "v152-fake"
	req.RequiredCapabilities.Environment = []EnvironmentCapability{"arch:arm64"}
	_, err := ex.Execute(context.Background(), req)
	var typed *EnvironmentCapabilityError
	if !errors.As(err, &typed) {
		t.Fatalf("err=%T %v", err, err)
	}
	if len(runtime.starts) != 0 {
		t.Fatal("runtime started despite capability mismatch")
	}
	if len(f.calls) != 5 {
		t.Fatalf("unexpected calls after capability rejection: %d", len(f.calls))
	}
}

func TestV152RemoteWorkspaceCleanupUsesEnvironmentRunner(t *testing.T) {
	f := &v152ScriptedRunner{results: []runner.Result{{ExitCode: 0}}}
	env := NewSSHEnvironment("build", f, EnvironmentProfile{}, nil)
	if err := env.CleanupWorkspace(context.Background(), v152RemoteReq().Workspace); err != nil {
		t.Fatal(err)
	}
	want := []string{"rm", "-rf", "--", "/remote/w"}
	if !reflect.DeepEqual(f.calls[0].Argv, want) {
		t.Fatalf("cleanup=%v", f.calls[0].Argv)
	}
}

func TestV152WorkspaceCleanupRejectsUnsafeRoot(t *testing.T) {
	f := &v152ScriptedRunner{}
	env := NewSSHEnvironment("build", f, EnvironmentProfile{}, nil)
	w := v152RemoteReq().Workspace
	w.Path = "/"
	err := env.CleanupWorkspace(context.Background(), w)
	var typed *EnvironmentError
	if !errors.As(err, &typed) || typed.Kind != FailureCleanup {
		t.Fatalf("err=%T %v", err, err)
	}
	if len(f.calls) != 0 {
		t.Fatal("unsafe cleanup reached runner")
	}
}

type v152LimitedEnvironment struct{}

func (*v152LimitedEnvironment) Kind() EnvironmentKind { return EnvironmentSSH }
func (*v152LimitedEnvironment) Capabilities() []EnvironmentCapability {
	return []EnvironmentCapability{CapabilityProcess}
}
func (*v152LimitedEnvironment) Resolve(context.Context, WorkspaceRef) (ResolvedWorkspace, error) {
	return ResolvedWorkspace{}, nil
}
