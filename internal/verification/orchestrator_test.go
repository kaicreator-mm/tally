package verification

// 本文件覆盖 P1-38(一个 contract revision + 一个 immutable ChangeSet 的最终
// 验证编排,PRD §14.2/§14.3、§8.7、§8.8、§8.13、§36.3):
//   AC1 恰好落一行 VerificationRun,精确绑定 contract/change/environment 身份;
//   AC2 最终功能覆盖与 effective independence 覆盖可推导(报告随结论返回);
//   AC3 held-out FAIL 返回非 VERIFIED/needs_review 信号,且实现侧可见投影
//       不含任何 held-out 诊断(ref/内容/输出摘要/可识别 verdict);
//   AC4 既往 valid+applicable FAIL 不被新的 PASS binding 取代(无裁定端口)。

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/policy"
	"tally/internal/runner"
)

// ---- 夹具 ------------------------------------------------------------------

const (
	orchTask    = "TASK-P1-38"
	orchSeed    = "orch-2026-fixed-seed"
	orchRef     = "ho-orch-secret-ref-9f2a1b"
	orchStore   = "holdout-store"
	orchSecret  = "TALLY-ORCH-HELDOUT-CONTENT-4c1e"
	orchProject = core.ProjectID("proj-orch")
)

// orchFixedPast 是候选固化时刻:早于任何 verdict 记录时刻,selection
// exposure 因此不成立(§8.6.2)。
var orchFixedPast = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// orchEnv 返回合法环境身份。
func orchEnv() EnvironmentIdentity {
	return EnvironmentIdentity{
		OS:        "linux",
		Arch:      "amd64",
		Toolchain: "go1.23.4",
		Image:     "tally/orch-verify:1",
		Tools:     []ToolVersion{{Name: "go", Version: "1.23.4"}},
		Env:       []string{"GOFLAGS=-count=1"},
	}
}

// orchContract 返回编排测试主夹具契约:
//   - AC-1/v1:visible test,pre_commit,basis=verification_authority,
//     independence_required=source,冻结协议 1 次 repetition;
//   - AC-2/h1:held_out test,basis=verification_authority+held_out,
//     independence_required=information,冻结协议 2 次 repetition(含 seed);
//   - gate 非空时附加一个 required gate。
func orchContract(t *testing.T, gate *Gate) *Frozen {
	t.Helper()
	c := &Contract{
		TaskID:   orchTask,
		Revision: 7,
		Acceptances: []Acceptance{
			{
				ID:                   "AC-1",
				Statement:            "visible suite passes on the candidate",
				IndependenceRequired: IndependenceSource,
				EvidenceBindings: []Binding{{
					ID:               "v1",
					AuthoringMode:    AuthoringPreCommit,
					VisibilityPolicy: VisibilityVisible,
					Basis:            &IndependenceBasis{Source: SourceVerificationAuthority},
					Evidence:         EvidenceRef{Type: EvidenceTest, Path: "pkg", Selector: "TestPkg"},
					ExecutionProtocol: &ExecutionProtocol{
						Repetitions: 1,
						Aggregation: AggregationAllPass,
						Command: &CommandSpec{
							Argv:    []string{"go", "test", "./pkg/..."},
							Dir:     "pkg",
							Env:     []string{"GOFLAGS=-count=1"},
							Timeout: Duration(time.Minute),
						},
					},
				}},
			},
			{
				ID:                   "AC-2",
				Statement:            "held-out scenario holds at final verification",
				IndependenceRequired: IndependenceInformation,
				EvidenceBindings: []Binding{{
					ID:               "h1",
					AuthoringMode:    AuthoringPreCommit,
					VisibilityPolicy: VisibilityHeldOut,
					Basis:            &IndependenceBasis{Source: SourceVerificationAuthority, Information: InfoHeldOut},
					Evidence:         EvidenceRef{Type: EvidenceTest, Store: orchStore, Ref: orchRef},
					ExecutionProtocol: &ExecutionProtocol{
						Repetitions: 2,
						Aggregation: AggregationAllPass,
						Seed:        orchSeed,
						Command:     &CommandSpec{Argv: []string{"tally-orch-holdout"}, Timeout: Duration(time.Minute)},
					},
				}},
			},
		},
		RuntimeRequirements: RuntimeRequirements{
			SessionIsolation:   true,
			HeldOutUnreachable: true,
			ConformanceClass:   "verification-isolation-v1",
		},
	}
	if gate != nil {
		c.RequiredGates = map[string]Gate{"build": *gate}
	}
	f, err := FreezeContract(c, FreezePolicy{})
	if err != nil {
		t.Fatalf("FreezeContract: %v", err)
	}
	return f
}

// orchTwoVisibleContract 返回仅含两条 visible binding 的契约(预算测试用)。
func orchTwoVisibleContract(t *testing.T) *Frozen {
	t.Helper()
	binding := func(id string) Binding {
		return Binding{
			ID:               id,
			AuthoringMode:    AuthoringPreCommit,
			VisibilityPolicy: VisibilityVisible,
			Evidence:         EvidenceRef{Type: EvidenceTest, Path: "pkg", Selector: "Test" + strings.ToUpper(id)},
			ExecutionProtocol: &ExecutionProtocol{
				Repetitions: 1,
				Aggregation: AggregationAllPass,
				Command:     &CommandSpec{Argv: []string{"go", "test", "./" + id + "/..."}, Timeout: Duration(time.Minute)},
			},
		}
	}
	c := &Contract{
		TaskID:   orchTask,
		Revision: 1,
		Acceptances: []Acceptance{
			{ID: "AC-1", Statement: "v1 passes", EvidenceBindings: []Binding{binding("v1")}},
			{ID: "AC-3", Statement: "v2 passes", EvidenceBindings: []Binding{binding("v2")}},
		},
	}
	f, err := FreezeContract(c, FreezePolicy{})
	if err != nil {
		t.Fatalf("FreezeContract: %v", err)
	}
	return f
}

// orchChangeSet 构造以 content 为可执行文件内容的 immutable ChangeSet。
func orchChangeSet(t *testing.T, content string) *changeset.ChangeSet {
	t.Helper()
	cs, err := changeset.New(changeset.Spec{
		Project:      orchProject,
		BaseRevision: "base-orch-1",
		ProducerRun:  "run-orch-1",
		Files:        []changeset.ChangedFile{{Path: "app/main.go", Content: content, Executable: true}},
	})
	if err != nil {
		t.Fatalf("changeset.New: %v", err)
	}
	return cs
}

// orchFactCall 记录一次 RecordFact 调用。
type orchFactCall struct {
	project core.ProjectID
	fact    FactSpec
}

// orchSink 是 RunSink 的确定性假实现:记录全部调用,按 acceptance 返回既往
// valid 事实。端口只有 RecordRun/RecordFact/ValidApplicableFacts 三个方法:
// 裁定/改写路径在接口上不存在(AC4 的结构层证明)。
type orchSink struct {
	nextID int64
	runs   []RunSpec
	facts  []orchFactCall
	scopes []evidence.Scope
	priors map[string][]PriorFact
	runErr error
}

var _ RunSink = (*orchSink)(nil)

func (s *orchSink) RecordRun(ctx context.Context, spec RunSpec) (int64, error) {
	if s.runErr != nil {
		return 0, s.runErr
	}
	s.nextID++
	s.runs = append(s.runs, spec)
	return s.nextID, nil
}

func (s *orchSink) RecordFact(ctx context.Context, project core.ProjectID, fact FactSpec) error {
	s.facts = append(s.facts, orchFactCall{project: project, fact: fact})
	return nil
}

func (s *orchSink) ValidApplicableFacts(ctx context.Context, scope evidence.Scope) ([]PriorFact, error) {
	s.scopes = append(s.scopes, scope)
	return s.priors[string(scope.Acceptance)], nil
}

// orchOut 返回一个摘要形态合法的 holdout 执行观察。
func orchOut(status EvidenceStatus) HoldoutRunOutput {
	return HoldoutRunOutput{Status: status, OutputDigest: "sha256:" + strings.Repeat("ab", 32)}
}

// orchBoundary 返回按脚本回放的 holdout 执行边界。
func orchBoundary(t *testing.T, script ...HoldoutRunOutput) HoldoutRunFunc {
	t.Helper()
	return (&recordingBoundary{script: script}).run
}

// orchNew 构造接好全部通道的编排器(holdout 通道可整体省略)。
func orchNew(t *testing.T, fr *fakeRunner, sink *orchSink, boundary HoldoutRunFunc) *Orchestrator {
	t.Helper()
	o, err := NewOrchestrator(OrchestratorConfig{
		Runner:        fr,
		Policy:        policy.DefaultExecutionPolicy(),
		HoldoutReader: &fakeHoldoutReader{content: []byte(orchSecret)},
		HoldoutRun:    boundary,
		Sink:          sink,
		Recorder:      evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally/orchestrator"},
	})
	if err != nil {
		t.Fatalf("NewOrchestrator: %v", err)
	}
	return o
}

// orchRequest 返回与夹具契约/候选一致的合法请求(基线:无 lineage 接触)。
func orchRequest(t *testing.T, f *Frozen, cs *changeset.ChangeSet) FinalRunRequest {
	t.Helper()
	return FinalRunRequest{
		Project:          orchProject,
		Contract:         f,
		ChangeSet:        cs,
		EnvironmentName:  "tally/orch-verify:1",
		Environment:      orchEnv(),
		VersionCombo:     core.VersionCombo{"go": "1.23.4"},
		Runtime:          "FakeRuntime",
		Workspace:        t.TempDir(),
		CandidateFixedAt: orchFixedPast,
		Lineage:          []evidence.ChangeSetID{evidence.ChangeSetID(cs.ID())},
		Implementation: evidence.ImplementationProvenance{
			Contributors: []evidence.Contributor{{Name: "owner", Role: "human_owner"}},
		},
		Attestations: map[string]BindingAttestation{
			"v1": {},
			"h1": {},
		},
	}
}

// ---- AC1+AC2:全绿 → verified,恰好一行 Run,覆盖可推导 ---------------------

func TestOrchestratorVerifiesAndPersistsOneRun(t *testing.T) {
	fr := &fakeRunner{results: []runner.Result{{Stdout: "ok", ExitCode: 0, Duration: 5 * time.Millisecond}}}
	sink := &orchSink{}
	o := orchNew(t, fr, sink, orchBoundary(t, orchOut(EvidenceStatusPass), orchOut(EvidenceStatusPass)))
	f := orchContract(t, nil)
	cs := orchChangeSet(t, "candidate-v1")
	req := orchRequest(t, f, cs)

	out, err := o.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// AC1:恰好一行 VerificationRun,精确绑定 contract/change/environment 身份。
	if out.Status != FinalVerified {
		t.Fatalf("Status = %q, 期望 verified(outcome=%+v)", out.Status, out)
	}
	if len(sink.runs) != 1 {
		t.Fatalf("RecordRun 调用 %d 次,期望恰好 1 次", len(sink.runs))
	}
	run := sink.runs[0]
	envHash, err := orchEnv().Hash()
	if err != nil {
		t.Fatalf("env hash: %v", err)
	}
	switch {
	case run.Project != orchProject:
		t.Errorf("run.Project = %q", run.Project)
	case run.Task != orchTask:
		t.Errorf("run.Task = %q", run.Task)
	case run.ContractRevision != 7:
		t.Errorf("run.ContractRevision = %d", run.ContractRevision)
	case run.ContractHash != f.Hash():
		t.Errorf("run.ContractHash = %q, 期望冻结哈希 %s", run.ContractHash, f.Hash())
	case run.ChangeSetID != cs.ID():
		t.Errorf("run.ChangeSetID = %q, 期望 %q", run.ChangeSetID, cs.ID())
	case run.EnvironmentHash != envHash:
		t.Errorf("run.EnvironmentHash = %q, 期望 %s", run.EnvironmentHash, envHash)
	case run.EnvironmentName != "tally/orch-verify:1":
		t.Errorf("run.EnvironmentName = %q", run.EnvironmentName)
	case run.Runtime != "FakeRuntime":
		t.Errorf("run.Runtime = %q", run.Runtime)
	case len(run.VersionCombo) == 0:
		t.Error("run.VersionCombo 为空")
	case run.Status != RunStatusPass:
		t.Errorf("run.Status = %q, 期望 pass", run.Status)
	case run.StartedAt.IsZero() || run.FinishedAt.IsZero():
		t.Error("run 起止时刻为零值")
	}
	if out.RunID != 1 {
		t.Errorf("out.RunID = %d, 期望 sink 返回的行 id 1", out.RunID)
	}

	// AC2:最终功能覆盖与 effective independence 覆盖可推导。
	if !out.Functional.Complete || !out.Independence.Complete {
		t.Fatalf("覆盖不完整:functional=%+v independence=%+v", out.Functional, out.Independence)
	}
	if got := out.Functional.Acceptances[0]; !got.Covered || len(got.SatisfiedBy) != 1 || got.SatisfiedBy[0] != "v1" {
		t.Errorf("AC-1 功能覆盖 = %+v", got)
	}
	if got := out.Independence.Acceptances[1]; !got.Covered || len(got.SatisfiedBy) != 1 || got.SatisfiedBy[0] != "h1" {
		t.Errorf("AC-2 independence 覆盖 = %+v", got)
	}

	// evidence 事实:visible pass + held-out pass,带 recorder provenance。
	if len(sink.facts) != 2 {
		t.Fatalf("evidence 事实 %d 条,期望 2 条(%+v)", len(sink.facts), sink.facts)
	}
	vfact := sink.facts[0].fact
	if sink.facts[0].project != orchProject || vfact.BindingID != "v1" || vfact.Verdict != EvidenceStatusPass ||
		vfact.VisibilityPolicy != VisibilityVisible || vfact.FactID != "vrun-1-v1" ||
		vfact.Recorder.Kind != evidence.RecorderVerificationRun || vfact.Recorder.Actor == "" {
		t.Errorf("v1 fact = %+v", vfact)
	}
	if vfact.Command == nil || len(vfact.Command.Argv) == 0 {
		t.Error("v1 fact 缺少冻结结构化命令")
	}
	hfact := sink.facts[1].fact
	if hfact.BindingID != "h1" || hfact.VisibilityPolicy != VisibilityHeldOut || hfact.Verdict != EvidenceStatusPass ||
		hfact.FactID != "vrun-1-h1" || hfact.StdoutDigest == "" {
		t.Errorf("h1 fact = %+v", hfact)
	}
	if len(out.FactIDs) != 2 {
		t.Errorf("FactIDs = %v", out.FactIDs)
	}

	// prior 读回按精确 scope 逐 acceptance 发生(§8.8.2)。
	if len(sink.scopes) != 2 {
		t.Fatalf("scope 读回 %d 次,期望 2 次", len(sink.scopes))
	}
	for i, want := range []string{"AC-1", "AC-2"} {
		sc := sink.scopes[i]
		if string(sc.Acceptance) != want || sc.Task != evidence.TaskID(orchTask) ||
			string(sc.Verification) != f.Hash() || string(sc.ChangeSet) != string(cs.ID()) || sc.Project != orchProject {
			t.Errorf("scope[%d] = %+v", i, sc)
		}
	}

	// 实现侧投影:verified 信号 + held-out 完成事实(allowlist)。
	sig := out.RedactedSignal()
	if sig.Status != FinalVerified || len(sig.Holdouts) != 1 || sig.Holdouts[0].AcceptanceID != "AC-2" {
		t.Errorf("RedactedSignal = %+v", sig)
	}
}

// ---- AC3:held-out FAIL → needs_review 且实现侧零诊断 -----------------------

func TestOrchestratorHeldOutFailNeedsReviewWithoutDisclosure(t *testing.T) {
	fr := &fakeRunner{results: []runner.Result{{Stdout: "ok", ExitCode: 0, Duration: 5 * time.Millisecond}}}
	sink := &orchSink{}
	o := orchNew(t, fr, sink, orchBoundary(t, orchOut(EvidenceStatusFail), orchOut(EvidenceStatusFail)))
	f := orchContract(t, nil)
	cs := orchChangeSet(t, "candidate-holdout-fail")

	out, err := o.Run(context.Background(), orchRequest(t, f, cs))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// 非 VERIFIED/needs_review 信号(AC3)。
	if out.Status != FinalNeedsReview {
		t.Fatalf("Status = %q, 期望 needs_review", out.Status)
	}
	if sink.runs[0].Status != RunStatusError {
		t.Errorf("run 行状态 = %q, 期望 error(needs_review 无干净 pass/fail)", sink.runs[0].Status)
	}
	// 完整判定只在 Control/Review 侧。
	if len(out.Holdout) != 1 || out.Holdout[0].Status != EvidenceStatusFail {
		t.Fatalf("Holdout 判定 = %+v", out.Holdout)
	}
	// 功能覆盖:h1 的 valid FAIL 阻断 AC-2。
	ac2 := out.Functional.Acceptances[1]
	if ac2.Gap != GapBlockingFailure || len(ac2.Blocking) != 1 || ac2.Blocking[0] != "h1" {
		t.Errorf("AC-2 功能覆盖 = %+v", ac2)
	}

	// 实现侧可见投影:只含生命周期信号与 allowlist 完成事实。
	sig := out.RedactedSignal()
	if sig.Status != FinalNeedsReview {
		t.Errorf("信号状态 = %q, 期望 needs_review", sig.Status)
	}
	if len(sig.Holdouts) != 1 || sig.Holdouts[0].AcceptanceID != "AC-2" || sig.Holdouts[0].EvidenceType != EvidenceTest {
		t.Errorf("信号 holdouts = %+v", sig.Holdouts)
	}
	data, err := json.Marshal(sig)
	if err != nil {
		t.Fatalf("marshal 信号: %v", err)
	}
	s := string(data)
	for _, leak := range []string{orchRef, orchStore, orchSecret, "h1", "fail", "sha256:"} {
		if strings.Contains(s, leak) {
			t.Fatalf("实现侧信号泄露 held-out 诊断 %q: %s", leak, s)
		}
	}
}

// ---- AC3(补):visible FAIL → failed,失败事实照常落库 ---------------------

func TestOrchestratorVisibleFailRecordsFactAndFails(t *testing.T) {
	fr := &fakeRunner{results: []runner.Result{{Stdout: "FAIL a_test", ExitCode: 1, Duration: 5 * time.Millisecond}}}
	sink := &orchSink{}
	o := orchNew(t, fr, sink, orchBoundary(t, orchOut(EvidenceStatusPass), orchOut(EvidenceStatusPass)))
	f := orchContract(t, nil)
	cs := orchChangeSet(t, "candidate-visible-fail")

	out, err := o.Run(context.Background(), orchRequest(t, f, cs))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Status != FinalFailed {
		t.Fatalf("Status = %q, 期望 failed", out.Status)
	}
	if sink.runs[0].Status != RunStatusFail {
		t.Errorf("run 行状态 = %q, 期望 fail", sink.runs[0].Status)
	}
	if out.Bindings[0].Status != EvidenceStatusFail || out.Bindings[0].Failures != 1 {
		t.Errorf("v1 观察 = %+v", out.Bindings[0])
	}
	if sink.facts[0].fact.Verdict != EvidenceStatusFail {
		t.Errorf("v1 fact verdict = %q, 期望 fail(失败事实必须落库,AC4 素材)", sink.facts[0].fact.Verdict)
	}
	// visible 失败对实现侧可见:failed 信号不是 held-out 诊断。
	if sig := out.RedactedSignal(); sig.Status != FinalFailed {
		t.Errorf("信号状态 = %q, 期望 failed", sig.Status)
	}
	if !out.Functional.Complete && out.Functional.Acceptances[0].Gap != GapBlockingFailure {
		t.Errorf("AC-1 功能覆盖 = %+v", out.Functional.Acceptances[0])
	}
}

// ---- AC4:既往 valid FAIL 不被新 PASS 取代 ----------------------------------

func TestOrchestratorPriorValidFailureCannotBeReplacedByPass(t *testing.T) {
	fr := &fakeRunner{results: []runner.Result{{Stdout: "ok", ExitCode: 0, Duration: 5 * time.Millisecond}}}
	sink := &orchSink{priors: map[string][]PriorFact{
		"AC-1": {{BindingID: "v1", Status: EvidenceStatusFail, RecordedAt: orchFixedPast}},
	}}
	o := orchNew(t, fr, sink, orchBoundary(t, orchOut(EvidenceStatusPass), orchOut(EvidenceStatusPass)))
	f := orchContract(t, nil)
	cs := orchChangeSet(t, "candidate-rerun-after-fail")

	// 同一 (contract revision, ChangeSet) 的再次最终验证:本轮 v1 PASS,
	// 但既往 valid+applicable FAIL 仍是 append-only 事实(§8.8.1)。
	out, err := o.Run(context.Background(), orchRequest(t, f, cs))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Status != FinalFailed {
		t.Fatalf("Status = %q, 期望 failed(新 PASS 不能取代既往 valid FAIL)", out.Status)
	}
	ac1 := out.Functional.Acceptances[0]
	if ac1.Gap != GapBlockingFailure || len(ac1.Blocking) != 1 || ac1.Blocking[0] != "v1" {
		t.Errorf("AC-1 功能覆盖 = %+v", ac1)
	}
	if len(ac1.SatisfiedBy) != 1 || ac1.SatisfiedBy[0] != "v1" {
		t.Errorf("AC-1 SatisfiedBy = %+v(PASS 并存但 FAIL 优先,§8.7.2)", ac1.SatisfiedBy)
	}
}

// ---- effective independence loss(planned ✓ → effective ✗)→ needs_review --

func TestOrchestratorIndependenceLossNeedsReview(t *testing.T) {
	fr := &fakeRunner{results: []runner.Result{{Stdout: "ok", ExitCode: 0, Duration: 5 * time.Millisecond}}}
	sink := &orchSink{}
	o := orchNew(t, fr, sink, orchBoundary(t, orchOut(EvidenceStatusPass), orchOut(EvidenceStatusPass)))
	f := orchContract(t, nil)
	cs := orchChangeSet(t, "candidate-lineage-contact")

	// v1 的证据内容在当前实现决策 lineage 内被修改:effective source
	// independence 丧失(§8.2.2 活动接触),planned ✓ → effective ✗。
	req := orchRequest(t, f, cs)
	req.Attestations["v1"] = BindingAttestation{ChangedInLineage: true}

	out, err := o.Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Status != FinalNeedsReview {
		t.Fatalf("Status = %q, 期望 needs_review", out.Status)
	}
	if !out.Functional.Complete {
		t.Errorf("功能覆盖不应受影响: %+v", out.Functional)
	}
	if out.Independence.Complete {
		t.Fatalf("independence 覆盖应不完整: %+v", out.Independence)
	}
	bi := out.Independence.Acceptances[0].Bindings[0]
	if bi.SourceIndependent || bi.Qualifies {
		t.Errorf("v1 独立性推导 = %+v, 期望 source-dependent", bi)
	}
	if sink.runs[0].Status != RunStatusError {
		t.Errorf("run 行状态 = %q, 期望 error", sink.runs[0].Status)
	}
}

// ---- required gate(§14.2)--------------------------------------------------

func TestOrchestratorRequiredGateFailBlocksVerified(t *testing.T) {
	fr := &fakeRunner{results: []runner.Result{
		{Stdout: "ok", ExitCode: 0, Duration: 5 * time.Millisecond},           // v1
		{Stderr: "build failed", ExitCode: 1, Duration: 5 * time.Millisecond}, // gate build
	}}
	sink := &orchSink{}
	o := orchNew(t, fr, sink, orchBoundary(t, orchOut(EvidenceStatusPass), orchOut(EvidenceStatusPass)))
	f := orchContract(t, &Gate{Required: true, Command: &CommandSpec{Argv: []string{"go", "build", "./..."}, Timeout: Duration(time.Minute)}})
	cs := orchChangeSet(t, "candidate-gate-fail")

	out, err := o.Run(context.Background(), orchRequest(t, f, cs))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Status != FinalFailed {
		t.Fatalf("Status = %q, 期望 failed(required gate FAIL 阻断 VERIFIED,§14.2)", out.Status)
	}
	if len(out.Gates) != 1 || out.Gates[0].Name != "build" || out.Gates[0].Status != VisibleStatusFail || !out.Gates[0].Required {
		t.Fatalf("gate 结果 = %+v", out.Gates)
	}
	if sink.runs[0].Status != RunStatusFail {
		t.Errorf("run 行状态 = %q, 期望 fail", sink.runs[0].Status)
	}
}

func TestOrchestratorGateErrorNeedsReview(t *testing.T) {
	fr := &fakeRunner{
		results: []runner.Result{{Stdout: "ok", ExitCode: 0, Duration: 5 * time.Millisecond}},
		errs:    []error{nil, errors.New("spawn failed")},
	}
	sink := &orchSink{}
	o := orchNew(t, fr, sink, orchBoundary(t, orchOut(EvidenceStatusPass), orchOut(EvidenceStatusPass)))
	f := orchContract(t, &Gate{Required: true, Command: &CommandSpec{Argv: []string{"go", "build", "./..."}, Timeout: Duration(time.Minute)}})
	cs := orchChangeSet(t, "candidate-gate-error")

	out, err := o.Run(context.Background(), orchRequest(t, f, cs))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Status != FinalNeedsReview {
		t.Fatalf("Status = %q, 期望 needs_review(执行出错无 verdict)", out.Status)
	}
	if out.Gates[0].Status != VisibleStatusError {
		t.Errorf("gate 状态 = %q, 期望 error", out.Gates[0].Status)
	}
}

// ---- 墙钟预算:整轮硬上限(§11.2)--------------------------------------------

func TestOrchestratorBudgetExhaustionNeedsReview(t *testing.T) {
	fr := &fakeRunner{results: []runner.Result{{Stdout: "ok", ExitCode: 0, Duration: 100 * time.Millisecond}}}
	sink := &orchSink{}
	pol := policy.DefaultExecutionPolicy()
	pol.Budgets.MaxWallTime = 50 * time.Millisecond
	o, err := NewOrchestrator(OrchestratorConfig{
		Runner:   fr,
		Policy:   pol,
		Sink:     sink,
		Recorder: evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally/orchestrator"},
	})
	if err != nil {
		t.Fatalf("NewOrchestrator: %v", err)
	}
	f := orchTwoVisibleContract(t)
	cs := orchChangeSet(t, "candidate-budget")

	out, err := o.Run(context.Background(), orchRequest(t, f, cs))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.Status != FinalNeedsReview {
		t.Fatalf("Status = %q, 期望 needs_review(预算耗尽无完整 verdict)", out.Status)
	}
	if out.Bindings[0].Err != "" || out.Bindings[1].Err != ErrBudgetExhausted.Error() {
		t.Errorf("binding 观察 = [%+v, %+v]", out.Bindings[0], out.Bindings[1])
	}
	// 只有产生 verdict 的 binding 落 evidence 事实。
	if len(sink.facts) != 1 || sink.facts[0].fact.BindingID != "v1" {
		t.Errorf("facts = %+v", sink.facts)
	}
}

// ---- 请求校验:执行发生前拒绝,不落任何行 -----------------------------------

func TestOrchestratorRejectsInvalidRequestsBeforeExecution(t *testing.T) {
	f := orchContract(t, nil)
	cs := orchChangeSet(t, "candidate-invalid")
	good := orchRequest(t, f, cs)

	noChannel := &orchSink{}
	oNoChannel := orchNewNoHoldout(t, noChannel)

	cases := []struct {
		name    string
		o       *Orchestrator
		req     FinalRunRequest
		wantErr error
	}{
		{"nil 契约", mustOrch(t), FinalRunRequest{ChangeSet: cs, Project: orchProject, Workspace: t.TempDir(), Environment: orchEnv(), VersionCombo: core.VersionCombo{"go": "1"}, Runtime: "FakeRuntime", EnvironmentName: "env"}, ErrInvalidRequest},
		{"nil 候选", mustOrch(t), func() FinalRunRequest { r := good; r.ChangeSet = nil; return r }(), ErrInvalidRequest},
		{"项目不匹配", mustOrch(t), func() FinalRunRequest { r := good; r.Project = "other"; return r }(), ErrInvalidRequest},
		{"缺环境名", mustOrch(t), func() FinalRunRequest { r := good; r.EnvironmentName = " "; return r }(), ErrInvalidRequest},
		{"缺 runtime", mustOrch(t), func() FinalRunRequest { r := good; r.Runtime = ""; return r }(), ErrInvalidRequest},
		{"缺 version combo", mustOrch(t), func() FinalRunRequest { r := good; r.VersionCombo = nil; return r }(), ErrInvalidRequest},
		{"缺 workspace", mustOrch(t), func() FinalRunRequest { r := good; r.Workspace = ""; return r }(), ErrNoWorkspace},
		{"相对 workspace", mustOrch(t), func() FinalRunRequest { r := good; r.Workspace = "rel/worktree"; return r }(), ErrInvalidRequest},
		{"held-out 通道缺失", oNoChannel, good, ErrHeldOutChannelMissing},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := tc.o.Run(context.Background(), tc.req)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, 期望 %v", err, tc.wantErr)
			}
			if out != nil {
				t.Errorf("出错时不应返回 outcome: %+v", out)
			}
		})
	}
	// 请求被拒绝的路径不得落任何行/事实(无 verdict 的半次执行不是事实)。
	if len(noChannel.runs) != 0 || len(noChannel.facts) != 0 || len(noChannel.scopes) != 0 {
		t.Errorf("被拒绝的请求产生了持久化调用: runs=%d facts=%d scopes=%d", len(noChannel.runs), len(noChannel.facts), len(noChannel.scopes))
	}
}

// TestOrchestratorRejectsNonFactCapableRecorder 覆盖 INV-AUTH-01/02:构造期
// 拒绝非 fact-capable 的 recorder 通道。
func TestOrchestratorRejectsNonFactCapableRecorder(t *testing.T) {
	_, err := NewOrchestrator(OrchestratorConfig{
		Runner:   &fakeRunner{},
		Policy:   policy.DefaultExecutionPolicy(),
		Sink:     &orchSink{},
		Recorder: evidence.Recorder{Kind: evidence.RecorderAgent, Actor: "agent"},
	})
	if !errors.Is(err, ErrNilDependency) {
		t.Fatalf("err = %v, 期望 ErrNilDependency", err)
	}
}

// orchNewNoHoldout 构造不接 holdout 通道的编排器(契约含 held_out binding 时
// Run 前拒绝)。
func orchNewNoHoldout(t *testing.T, sink *orchSink) *Orchestrator {
	t.Helper()
	o, err := NewOrchestrator(OrchestratorConfig{
		Runner:   &fakeRunner{},
		Policy:   policy.DefaultExecutionPolicy(),
		Sink:     sink,
		Recorder: evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally/orchestrator"},
	})
	if err != nil {
		t.Fatalf("NewOrchestrator: %v", err)
	}
	return o
}

// mustOrch 返回接全通道的编排器与独立 sink(请求校验用,忽略 sink 断言)。
func mustOrch(t *testing.T) *Orchestrator {
	t.Helper()
	return orchNew(t, &fakeRunner{}, &orchSink{}, orchBoundary(t, orchOut(EvidenceStatusPass), orchOut(EvidenceStatusPass)))
}
