package controlplane

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tally/internal/engineering"
	"tally/internal/taskexecution"
)

type fakeExecutor struct {
	result taskexecution.TaskExecutionResult
	err    error
	calls  int
}

func (f *fakeExecutor) Execute(context.Context, taskexecution.TaskExecutionRequest) (taskexecution.TaskExecutionResult, error) {
	f.calls++
	return f.result, f.err
}
func (f *fakeExecutor) Cancel(engineering.ID) error { return nil }

type fakeWorkflow struct {
	result TaskWorkflowResult
	err    error
	calls  int
}

func (f *fakeWorkflow) VerifyAndAccept(context.Context, engineering.ID, taskexecution.TaskExecutionResult) (TaskWorkflowResult, error) {
	f.calls++
	return f.result, f.err
}

func artifact(id engineering.ID, kind engineering.ArtifactKind) engineering.ArtifactRef {
	return engineering.ArtifactRef{ID: id, Kind: kind, URI: "git://repo/" + string(id)}
}
func baseline(t *testing.T) engineering.RequirementsBaseline {
	t.Helper()
	b, err := engineering.NewRequirementsBaseline("req-v1", "r1", []engineering.Requirement{{ID: "REQ-1", Statement: "x", AcceptanceCriteria: []string{"done"}}}, artifact("req-art", engineering.ArtifactRequirements), engineering.Provenance{Actor: "owner", Source: "test", At: time.Unix(1, 0)})
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func plan(t *testing.T) engineering.ProjectPlan {
	t.Helper()
	r, err := engineering.NewPlanningRevision("plan-r1", 1, "", []string{"build"}, artifact("plan-art", engineering.ArtifactProjectPlan), engineering.Provenance{Actor: "owner", Source: "test", At: time.Unix(2, 0)})
	if err != nil {
		t.Fatal(err)
	}
	return engineering.ProjectPlan{ID: "plan", AcceptedRevision: r}
}
func arch(t *testing.T) engineering.ArchitectureRevision {
	t.Helper()
	a, err := engineering.NewArchitectureRevision("arch-r1", engineering.ArchitectureDesign, "design", nil, artifact("arch-art", engineering.ArtifactArchitecture), engineering.Provenance{Actor: "owner", Source: "test", At: time.Unix(3, 0)})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func newHarness(t *testing.T, exec *fakeExecutor, wf TaskWorkflow) (*Service, *FileRepository) {
	t.Helper()
	repo, err := NewFileRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewService(repo, exec, wf)
	if err != nil {
		t.Fatal(err)
	}
	n := int64(10)
	s.Now = func() time.Time { n++; return time.Unix(n, 0) }
	return s, repo
}

func bootstrap(t *testing.T, s *Service) {
	t.Helper()
	ctx := context.Background()
	if _, err := s.AcceptRequirements(ctx, "project-1", baseline(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AcceptPlan(ctx, "project-1", plan(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AcceptArchitecture(ctx, "project-1", arch(t)); err != nil {
		t.Fatal(err)
	}
	_, err := s.ActivateDAG(ctx, "project-1", engineering.TaskDAGProposal{RevisionID: "dag-r1", Nodes: []engineering.TaskNode{{ID: "A"}, {ID: "B", Dependencies: []engineering.ID{"A"}}}}, map[engineering.ID]engineering.CompletionContract{"A": {RequiredState: engineering.NodeAccepted, RequiredEvidence: []engineering.ID{"verify-A"}}, "B": {RequiredState: engineering.NodeAccepted}})
	if err != nil {
		t.Fatal(err)
	}
}

func request(id engineering.ID) taskexecution.TaskExecutionRequest {
	return taskexecution.TaskExecutionRequest{TaskID: id, FrozenTaskContract: artifact("task-"+id, engineering.ArtifactTaskContract), Runtime: taskexecution.RuntimeRef{ID: "fake"}, Environment: taskexecution.EnvironmentRef{Kind: taskexecution.EnvironmentLocal, Target: "local"}, Workspace: taskexecution.WorkspaceRef{Environment: taskexecution.EnvironmentLocal, Target: "local", Identity: "wt-1", Path: "/tmp/wt", BaseRevision: "abc"}, Instruction: "implement"}
}

func TestLifecyclePersistsAndReloadsReadiness(t *testing.T) {
	s, repo := newHarness(t, &fakeExecutor{}, &fakeWorkflow{})
	bootstrap(t, s)
	got, err := repo.Load(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ActiveDAG == nil || got.ActiveDAG.Nodes[0].State != engineering.TaskReady || got.ActiveDAG.Nodes[1].State != engineering.TaskBlocked {
		t.Fatalf("bad readiness: %+v", got.ActiveDAG)
	}
	if len(got.Audit.Records) != 4 {
		t.Fatalf("audit=%d", len(got.Audit.Records))
	}
	s2, err := NewService(repo, &fakeExecutor{}, &fakeWorkflow{})
	if err != nil {
		t.Fatal(err)
	}
	again, err := s2.Snapshot(context.Background(), "project-1")
	if err != nil || again.ActiveDAG.RevisionID != "dag-r1" {
		t.Fatalf("restart snapshot=%+v err=%v", again.ActiveDAG, err)
	}
}

func TestStageOrderRejectsPrematureDAG(t *testing.T) {
	s, _ := newHarness(t, &fakeExecutor{}, &fakeWorkflow{})
	_, err := s.ActivateDAG(context.Background(), "missing", engineering.TaskDAGProposal{}, nil)
	if !errors.Is(err, ErrProjectNotFound) {
		t.Fatalf("err=%v", err)
	}
	if _, err := s.AcceptRequirements(context.Background(), "p", baseline(t)); err != nil {
		t.Fatal(err)
	}
	_, err = s.AcceptArchitecture(context.Background(), "p", arch(t))
	if !errors.Is(err, ErrStageOrder) {
		t.Fatalf("err=%v", err)
	}
}

func TestAgentClaimAloneCannotAdvance(t *testing.T) {
	exec := &fakeExecutor{result: taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionCompleted}}
	s, _ := newHarness(t, exec, nil)
	bootstrap(t, s)
	res, err := s.Dispatch(context.Background(), "project-1", request("A"))
	if !errors.Is(err, ErrWorkflowUnavailable) {
		t.Fatalf("err=%v", err)
	}
	n, _ := findNode(*res.Snapshot.ActiveDAG, "A")
	if n.State != engineering.TaskBlocked || exec.calls != 1 {
		t.Fatalf("state=%s calls=%d", n.State, exec.calls)
	}
}

func TestAcceptedWorkflowStillBlockedByCompletionContract(t *testing.T) {
	exec := &fakeExecutor{result: taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionCompleted}}
	wf := &fakeWorkflow{result: TaskWorkflowResult{Accepted: true, Facts: engineering.CompletionFacts{State: engineering.NodeAccepted}}}
	s, _ := newHarness(t, exec, wf)
	bootstrap(t, s)
	res, err := s.Dispatch(context.Background(), "project-1", request("A"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Completion == nil || res.Completion.Satisfied {
		t.Fatalf("completion=%+v", res.Completion)
	}
	n, _ := findNode(*res.Snapshot.ActiveDAG, "A")
	if n.State != engineering.TaskBlocked {
		t.Fatalf("state=%s", n.State)
	}
}

func TestAcceptedWorkflowAndContractAdvanceDAG(t *testing.T) {
	exec := &fakeExecutor{result: taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionCompleted}}
	wf := &fakeWorkflow{result: TaskWorkflowResult{Accepted: true, Facts: engineering.CompletionFacts{State: engineering.NodeAccepted, Evidence: []engineering.EvidenceRef{{ID: "verify-A", Kind: "verification", URI: "evidence://A", Status: engineering.EvidencePass}}}}}
	s, repo := newHarness(t, exec, wf)
	bootstrap(t, s)
	res, err := s.Dispatch(context.Background(), "project-1", request("A"))
	if err != nil {
		t.Fatal(err)
	}
	a, _ := findNode(*res.Snapshot.ActiveDAG, "A")
	b, _ := findNode(*res.Snapshot.ActiveDAG, "B")
	if a.State != engineering.TaskDone || b.State != engineering.TaskReady {
		t.Fatalf("A=%s B=%s", a.State, b.State)
	}
	persisted, err := repo.Load(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	a, _ = findNode(*persisted.ActiveDAG, "A")
	b, _ = findNode(*persisted.ActiveDAG, "B")
	if a.State != engineering.TaskDone || b.State != engineering.TaskReady {
		t.Fatalf("persisted A=%s B=%s", a.State, b.State)
	}
}

func TestExecutionErrorAndInterruptionPersist(t *testing.T) {
	exec := &fakeExecutor{err: errors.New("runtime down")}
	s, _ := newHarness(t, exec, &fakeWorkflow{})
	bootstrap(t, s)
	res, err := s.Dispatch(context.Background(), "project-1", request("A"))
	if err == nil {
		t.Fatal("expected error")
	}
	n, _ := findNode(*res.Snapshot.ActiveDAG, "A")
	if n.State != engineering.TaskBlocked {
		t.Fatalf("state=%s", n.State)
	}

	exec2 := &fakeExecutor{result: taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionInterrupted}}
	s2, _ := newHarness(t, exec2, &fakeWorkflow{})
	bootstrap(t, s2)
	res, err = s2.Dispatch(context.Background(), "project-1", request("A"))
	if err != nil {
		t.Fatal(err)
	}
	n, _ = findNode(*res.Snapshot.ActiveDAG, "A")
	if n.State != engineering.TaskInterrupted {
		t.Fatalf("state=%s", n.State)
	}
}

func TestFileRepositoryRejectsCorruptSnapshotAndUnsafeID(t *testing.T) {
	repo, err := NewFileRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Load(context.Background(), "../x"); err == nil {
		t.Fatal("expected unsafe id")
	}
	path := filepath.Join(repo.Root, "project-1.json")
	if err := os.WriteFile(path, []byte(`{"schema_version":1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Load(context.Background(), "project-1"); err == nil {
		t.Fatal("expected invalid snapshot")
	}
}

func TestAuthorityDecisionParticipatesInCompletion(t *testing.T) {
	exec := &fakeExecutor{result: taskexecution.TaskExecutionResult{Status: taskexecution.ExecutionCompleted}}
	wf := &fakeWorkflow{result: TaskWorkflowResult{Accepted: true, Facts: engineering.CompletionFacts{State: engineering.NodeAccepted, Evidence: []engineering.EvidenceRef{{ID: "verify-A", Kind: "verification", URI: "evidence://A", Status: engineering.EvidencePass}}}}}
	s, _ := newHarness(t, exec, wf)
	ctx := context.Background()
	if _, err := s.AcceptRequirements(ctx, "project-1", baseline(t)); err != nil { t.Fatal(err) }
	if _, err := s.AcceptPlan(ctx, "project-1", plan(t)); err != nil { t.Fatal(err) }
	if _, err := s.AcceptArchitecture(ctx, "project-1", arch(t)); err != nil { t.Fatal(err) }
	_, err := s.ActivateDAG(ctx, "project-1", engineering.TaskDAGProposal{RevisionID: "dag-r1", Nodes: []engineering.TaskNode{{ID: "A"}}}, map[engineering.ID]engineering.CompletionContract{"A": {RequiredState: engineering.NodeAccepted, RequiredEvidence: []engineering.ID{"verify-A"}, RequiredAuthority: []engineering.ID{"AUTH-1"}}})
	if err != nil { t.Fatal(err) }
	res, err := s.Dispatch(ctx, "project-1", request("A"))
	if err != nil { t.Fatal(err) }
	if res.Completion == nil || res.Completion.Satisfied { t.Fatalf("authority should block: %+v", res.Completion) }
	if _, err := s.RestartTask(ctx, "project-1", "A"); err != nil { t.Fatal(err) }
	_, err = s.RecordAuthority(ctx, "project-1", "A", engineering.AuthorityDecision{RequirementID: "AUTH-1", Status: engineering.AuthorityAuthorized, Actor: "owner", At: time.Unix(100, 0), Provenance: engineering.Provenance{Actor: "owner", Source: "test", At: time.Unix(100, 0)}})
	if err != nil { t.Fatal(err) }
	res, err = s.Dispatch(ctx, "project-1", request("A"))
	if err != nil { t.Fatal(err) }
	n, _ := findNode(*res.Snapshot.ActiveDAG, "A")
	if n.State != engineering.TaskDone { t.Fatalf("state=%s completion=%+v", n.State, res.Completion) }
}

func TestRestartTaskRecoversPersistedRunning(t *testing.T) {
	s, repo := newHarness(t, &fakeExecutor{}, &fakeWorkflow{})
	bootstrap(t, s)
	snap, err := repo.Load(context.Background(), "project-1")
	if err != nil { t.Fatal(err) }
	running, err := snap.ActiveDAG.Transition("A", engineering.TaskRunning, "")
	if err != nil { t.Fatal(err) }
	snap.ActiveDAG = &running
	if err := repo.Save(context.Background(), snap); err != nil { t.Fatal(err) }

	restarted, err := s.RestartTask(context.Background(), "project-1", "A")
	if err != nil { t.Fatal(err) }
	n, _ := findNode(*restarted.ActiveDAG, "A")
	if n.State != engineering.TaskReady { t.Fatalf("state=%s", n.State) }
	if len(restarted.Audit.Records) < 2 || restarted.Audit.Records[len(restarted.Audit.Records)-2].Kind != "task.recovered_interrupted" || restarted.Audit.Records[len(restarted.Audit.Records)-1].Kind != "task.restart.ready" {
		t.Fatalf("audit=%+v", restarted.Audit.Records)
	}
}
