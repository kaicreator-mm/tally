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
	if i < len(s.results) { r = s.results[i] }
	if i < len(s.errs) { err = s.errs[i] }
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
func (r *v152LocalOnlyRuntime) Descriptor() agentruntime.RuntimeDescriptor { return agentruntime.RuntimeDescriptor{Name: "local-only", Capabilities: agentruntime.NewCapabilitySet()} }
func (r *v152LocalOnlyRuntime) Start(context.Context, agentruntime.StartRequest) (agentruntime.Session, error) { r.starts++; return &v152Session{}, nil }

type v152Session struct{}
func (*v152Session) Execute(context.Context, agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) { return agentruntime.AgentRunResult{RunID: "r", SessionID: "s", Status: agentruntime.RunCompleted}, nil }
func (*v152Session) Cancel(context.Context, agentruntime.RunID) error { return nil }
func (*v152Session) Events(context.Context) (agentruntime.EventStream, error) { return nil, errors.New("unused") }
func (*v152Session) Result(context.Context, agentruntime.RunID) (agentruntime.AgentRunResult, error) { return agentruntime.AgentRunResult{}, nil }
func (*v152Session) Close(context.Context) error { return nil }

func v152RemoteReq() TaskExecutionRequest {
	return TaskExecutionRequest{
		TaskID: "T-v152",
		FrozenTaskContract: engineering.ArtifactRef{ID: "contract", Kind: engineering.ArtifactTaskContract, URI: "git://task"},
		Runtime: RuntimeRef{ID: "fake"},
		Environment: EnvironmentRef{Kind: EnvironmentSSH, Target: "build"},
		Workspace: WorkspaceRef{Environment: EnvironmentSSH, Target: "build", Identity: "w", Path: "/remote/w", BaseRevision: "abc"},
		Instruction: "go",
	}
}

func TestV152SSHResolveUsesEnvironmentPathWithoutLocalPath(t *testing.T) {
	f := &v152ScriptedRunner{results: []runner.Result{{ExitCode: 0}}}
	env := NewSSHEnvironment("build", f, EnvironmentProfile{}, nil)
	got, err := env.Resolve(context.Background(), v152RemoteReq().Workspace)
	if err != nil { t.Fatal(err) }
	if got.Location != "/remote/w" || got.LocalPath != "" { t.Fatalf("got=%+v", got) }
	want := []string{"git", "-C", "/remote/w", "cat-file", "-e", "abc^{commit}"}
	if !reflect.DeepEqual(f.calls[0].Argv, want) { t.Fatalf("call=%v", f.calls[0].Argv) }
}

func TestV152ExecutorBindsRemoteRuntimeToEnvironmentRunner(t *testing.T) {
	f := &v152ScriptedRunner{results: []runner.Result{{ExitCode: 0}}}
	env := NewSSHEnvironment("build", f, EnvironmentProfile{}, nil)
	runtime := &v152Runtime{}
	ex, _ := NewRuntimeExecutor(runtime, env)
	if _, err := ex.Execute(context.Background(), v152RemoteReq()); err != nil { t.Fatal(err) }
	if len(runtime.starts) != 1 || runtime.starts[0].Workspace != "/remote/w" { t.Fatalf("starts=%+v", runtime.starts) }
	if runtime.bound != f { t.Fatal("remote runtime was not bound to selected environment runner") }
}

func TestV152NonLocalRuntimeCannotSilentlyFallBackToLocal(t *testing.T) {
	f := &v152ScriptedRunner{results: []runner.Result{{ExitCode: 0}}}
	env := NewSSHEnvironment("build", f, EnvironmentProfile{}, nil)
	runtime := &v152LocalOnlyRuntime{}
	ex, _ := NewRuntimeExecutor(runtime, env)
	_, err := ex.Execute(context.Background(), v152RemoteReq())
	var typed *EnvironmentError
	if !errors.As(err, &typed) || typed.Kind != FailureProcess { t.Fatalf("err=%T %v", err, err) }
	if runtime.starts != 0 { t.Fatal("local Start was called for a non-local environment") }
}

func TestV152LostWorkspaceCanRecreate(t *testing.T) {
	f := &v152ScriptedRunner{results: []runner.Result{{ExitCode: 1, Stderr: "missing"}, {ExitCode: 0}}}
	count := 0
	env := NewSSHEnvironment("build", f, EnvironmentProfile{}, func(context.Context, WorkspaceRef) error { count++; return nil })
	if _, err := env.Resolve(context.Background(), v152RemoteReq().Workspace); err != nil { t.Fatal(err) }
	if count != 1 || len(f.calls) != 2 { t.Fatalf("recreate=%d calls=%d", count, len(f.calls)) }
}

func TestV152EnvironmentCapabilityErrorIsTyped(t *testing.T) {
	env := &v152LimitedEnvironment{}
	err := checkEnvironmentCapabilities(env, []EnvironmentCapability{CapabilityCancel})
	var typed *EnvironmentCapabilityError
	if !errors.As(err, &typed) { t.Fatalf("err=%T %v", err, err) }
}

type v152LimitedEnvironment struct{}
func (*v152LimitedEnvironment) Kind() EnvironmentKind { return EnvironmentSSH }
func (*v152LimitedEnvironment) Capabilities() []EnvironmentCapability { return []EnvironmentCapability{CapabilityProcess} }
func (*v152LimitedEnvironment) Resolve(context.Context, WorkspaceRef) (ResolvedWorkspace, error) { return ResolvedWorkspace{}, nil }
