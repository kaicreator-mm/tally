package taskexecution

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/engineering"
)

type fakeRuntime struct {
	caps       agentruntime.CapabilitySet
	mu         sync.Mutex
	starts     []agentruntime.StartRequest
	failResume bool
	block      bool
}

func (f *fakeRuntime) Descriptor() agentruntime.RuntimeDescriptor {
	return agentruntime.RuntimeDescriptor{Name: "fake", Capabilities: f.caps}
}
func (f *fakeRuntime) Start(_ context.Context, r agentruntime.StartRequest) (agentruntime.Session, error) {
	f.mu.Lock()
	f.starts = append(f.starts, r)
	f.mu.Unlock()
	if r.ResumeOf != "" && f.failResume {
		return nil, errors.New("resume lost")
	}
	return &fakeSession{block: f.block}, nil
}

type fakeSession struct{ block bool }

func (s *fakeSession) Execute(ctx context.Context, _ agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	if s.block {
		<-ctx.Done()
		return agentruntime.AgentRunResult{RunID: "r", SessionID: "s", Status: agentruntime.RunCancelled}, nil
	}
	return agentruntime.AgentRunResult{RunID: "r", SessionID: "s", Status: agentruntime.RunCompleted, Summary: "claim only"}, nil
}
func (*fakeSession) Cancel(context.Context, agentruntime.RunID) error { return nil }
func (*fakeSession) Events(context.Context) (agentruntime.EventStream, error) {
	return nil, errors.New("unused")
}
func (*fakeSession) Result(context.Context, agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	return agentruntime.AgentRunResult{}, nil
}
func (*fakeSession) Close(context.Context) error { return nil }

func req() TaskExecutionRequest {
	return TaskExecutionRequest{
		TaskID:             "T1",
		FrozenTaskContract: engineering.ArtifactRef{ID: "contract", Kind: engineering.ArtifactTaskContract, URI: "git://task"},
		Runtime:            RuntimeRef{ID: "fake"}, Environment: EnvironmentRef{Kind: EnvironmentLocal, Target: "host"},
		Workspace:            WorkspaceRef{Environment: EnvironmentLocal, Target: "host", Identity: "ws-1", Path: "/tmp/ws", BaseRevision: "abc"},
		RequiredCapabilities: CapabilityRequirement{Environment: []EnvironmentCapability{CapabilityProcess, CapabilityWorkspaceRead}},
		Instruction:          "implement task",
	}
}
func TestLocalExecutionReturnsAgentClaimNotEvidence(t *testing.T) {
	rt := &fakeRuntime{caps: agentruntime.NewCapabilitySet()}
	ex, _ := NewRuntimeExecutor(rt, LocalEnvironment{})
	got, err := ex.Execute(context.Background(), req())
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != ExecutionCompleted || got.AgentClaim.Summary != "claim only" {
		t.Fatalf("got=%+v", got)
	}
	if len(got.EvidenceRefs) != 0 {
		t.Fatal("runtime claim must not fabricate verification evidence")
	}
}
func TestCapabilityMismatchFailsBeforeRuntimeStart(t *testing.T) {
	rt := &fakeRuntime{caps: agentruntime.NewCapabilitySet()}
	ex, _ := NewRuntimeExecutor(rt, LocalEnvironment{})
	r := req()
	r.RequiredCapabilities.Runtime.Required = []agentruntime.Capability{agentruntime.CapSessionResume}
	if _, err := ex.Execute(context.Background(), r); err == nil {
		t.Fatal("expected capability failure")
	}
	if len(rt.starts) != 0 {
		t.Fatal("runtime started despite capability failure")
	}
}
func TestResumeFailureFallsBackFreshOnce(t *testing.T) {
	rt := &fakeRuntime{caps: agentruntime.NewCapabilitySet(agentruntime.CapSessionResume), failResume: true}
	ex, _ := NewRuntimeExecutor(rt, LocalEnvironment{})
	r := req()
	r.Policy.ResumeSession = "old"
	r.Policy.FallbackToFresh = true
	if _, err := ex.Execute(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if len(rt.starts) != 2 || rt.starts[0].ResumeOf != "old" || rt.starts[1].ResumeOf != "" {
		t.Fatalf("starts=%+v", rt.starts)
	}
}
func TestResumeUnsupportedCanRestartFreshWhenNotRequired(t *testing.T) {
	rt := &fakeRuntime{caps: agentruntime.NewCapabilitySet()}
	ex, _ := NewRuntimeExecutor(rt, LocalEnvironment{})
	r := req()
	r.Policy.ResumeSession = "old"
	r.Policy.FallbackToFresh = true
	if _, err := ex.Execute(context.Background(), r); err != nil {
		t.Fatal(err)
	}
	if len(rt.starts) != 1 || rt.starts[0].ResumeOf != "" {
		t.Fatalf("starts=%+v", rt.starts)
	}
}
func TestCancelInterruptsTask(t *testing.T) {
	rt := &fakeRuntime{caps: agentruntime.NewCapabilitySet(), block: true}
	ex, _ := NewRuntimeExecutor(rt, LocalEnvironment{})
	ch := make(chan TaskExecutionResult, 1)
	errch := make(chan error, 1)
	go func() { got, err := ex.Execute(context.Background(), req()); ch <- got; errch <- err }()
	deadline := time.After(time.Second)
	for {
		ex.mu.Lock()
		active := len(ex.active) > 0
		ex.mu.Unlock()
		if active {
			break
		}
		select {
		case <-deadline:
			t.Fatal("task never active")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	if err := ex.Cancel("T1"); err != nil {
		t.Fatal(err)
	}
	got := <-ch
	if err := <-errch; err != nil {
		t.Fatal(err)
	}
	if got.Status != ExecutionInterrupted {
		t.Fatalf("status=%s", got.Status)
	}
}
func TestLocalEnvironmentRejectsRemoteWorkspace(t *testing.T) {
	_, err := (LocalEnvironment{}).Resolve(context.Background(), WorkspaceRef{Environment: EnvironmentSSH, Target: "h", Identity: "w", BaseRevision: "a", Path: "/x"})
	if !errors.Is(err, ErrUnsupportedEnvironment) {
		t.Fatalf("err=%v", err)
	}
}
