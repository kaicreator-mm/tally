package taskexecution

import (
	"context"
	"errors"
	"testing"

	"tally/internal/agentruntime"
	"tally/internal/runner"
)

type v152CloseFailRuntime struct{}

func (*v152CloseFailRuntime) Descriptor() agentruntime.RuntimeDescriptor {
	return agentruntime.RuntimeDescriptor{Name: "close-fail", Capabilities: agentruntime.NewCapabilitySet()}
}
func (*v152CloseFailRuntime) Start(context.Context, agentruntime.StartRequest) (agentruntime.Session, error) {
	return &v152CloseFailSession{}, nil
}
func (*v152CloseFailRuntime) StartInEnvironment(context.Context, agentruntime.StartRequest, runner.Runner) (agentruntime.Session, error) {
	return &v152CloseFailSession{}, nil
}

type v152CloseFailSession struct{}

func (*v152CloseFailSession) Execute(context.Context, agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	return agentruntime.AgentRunResult{RunID: "r", SessionID: "s", Status: agentruntime.RunCompleted}, nil
}
func (*v152CloseFailSession) Cancel(context.Context, agentruntime.RunID) error { return nil }
func (*v152CloseFailSession) Events(context.Context) (agentruntime.EventStream, error) {
	return nil, errors.New("unused")
}
func (*v152CloseFailSession) Result(context.Context, agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	return agentruntime.AgentRunResult{}, nil
}
func (*v152CloseFailSession) Close(context.Context) error { return errors.New("remote cleanup failed") }

func TestV152NonLocalSessionCleanupFailureIsTyped(t *testing.T) {
	f := &v152ScriptedRunner{results: []runner.Result{{ExitCode: 0}}}
	env := NewSSHEnvironment("build", f, EnvironmentProfile{}, nil)
	ex, err := NewRuntimeExecutor(&v152CloseFailRuntime{}, env)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ex.Execute(context.Background(), v152RemoteReq())
	var typed *EnvironmentError
	if !errors.As(err, &typed) || typed.Kind != FailureCleanup {
		t.Fatalf("err=%T %v", err, err)
	}
	if typed.Operation != "runtime.session.close" {
		t.Fatalf("operation=%q", typed.Operation)
	}
}

func TestV152LocalSessionCloseCompatibilityStillIgnoresCloseError(t *testing.T) {
	req := req()
	ex, err := NewRuntimeExecutor(&v152CloseFailRuntime{}, LocalEnvironment{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := ex.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Local compatibility path changed: %v", err)
	}
	if got.Status != ExecutionCompleted {
		t.Fatalf("status=%s", got.Status)
	}
}
