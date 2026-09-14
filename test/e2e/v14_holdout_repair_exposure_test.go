//go:build integration

// P2-06 端到端:held-out 修复安全边界与暴露核算(INV-V14-07/08;ADR-0001
// §4/§5;决策锁 10/23)跨真实边界组装:闭环 Engine(closedloop)由真实
// 组成面驱动 —— 执行适配器交付经真实单 Task 工作流(FakeRuntime + 一次性
// Git worktree + RunStoreSink)固化进 PostgreSQL 的候选快照;修复预算用
// P2-07 的 DefaultRepairPolicy;失败路由用 P2-03 的 NewFailureRouter;取代
// 事实经 P2-05 的 NewStoreLineageSink 落真库;蓄意诊断暴露经 P2-06 的
// ExposureSinkFunc → runstore.RecordExposure 落真实 evidence_exposure 表。
//
// 证明目标(任务验收):
//
//	AC1 修复尝试的编码上下文里,失败材料只有安全 RepairFeedback 投影
//	    (P2-04 类型,P2-03/29 的真实出口组合产出):held-out 内容标记、
//	    store 路径、不透明 ref、内容/输出摘要、binding 身份在投影值面
//	    (JSON、%+v)均不存在 —— 严格路径无原始 held-out 内容;
//	AC2 修复在途的一次蓄意诊断暴露经共享账面记账,成为真实库里的
//	    append-only evidence_exposure 事实(取得数据库身份;UPDATE/DELETE
//	    被语句级触发器拒绝);没有暴露的对照路径不产生暴露行;
//	AC3 compromised 暴露(锚定祖先、命中当前候选谱系)下,候选即使
//	    verified 也不得接受:终态 exposure_compromise 升级、basis 恒空、
//	    取代谱系完整保留;无暴露对照路径正常接受;真库事实经
//	    evidence.CompromiseScope 投影出相同的谱系封顶;
//	AC4 反馈构造失败(材料不足/不安全)落 feedback_unavailable 升级:
//	    修复 Run 不启动、取代事实与暴露行均为零 —— 宁可升级绝不泄漏。
//
// 数据库不可达时 SKIP(不产出假失败)。
package e2e

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"tally/internal/changeset"
	"tally/internal/closedloop"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/holdout"
	"tally/internal/runstore"
	"tally/internal/taskcontract"
	"tally/internal/verification"
	"tally/internal/workflow"
	"tally/internal/workspace"
	"tally/runtimes/fake"
)

const (
	hrProject    = core.ProjectID("proj-p2-06")
	hrTask       = "TASK-P2-06"
	hrAcceptance = "AC-EXP"
	hrBinding    = "holdout-core"
	hrFactID     = evidence.FactID("fact-p2-06-holdout-core")

	// hrSecretMarker 是 held-out 内容中的唯一标记:修复编码上下文出现它即泄漏。
	hrSecretMarker = "EXPECT-QUIET-HARBOR-0606"
	hrHoldoutRef   = "p2-06/case-1"
)

var (
	// hrAt / hrAt2 固定时钟:候选固化、暴露观测与反馈事实全部确定。
	hrAt     = time.Date(2026, 9, 8, 9, 0, 0, 0, time.UTC)
	hrAt2    = hrAt.Add(time.Hour)
	hrLeakAt = hrAt2.Add(30 * time.Minute)
)

// hrFrozenContract 冻结一份含 held_out binding、information 独立要求的契约
// (严格路径:该验收的 strict 主张依赖 held-out 信息的独立)。
func hrFrozenContract(t *testing.T) *verification.Frozen {
	t.Helper()
	c := verification.Contract{
		TaskID:   hrTask,
		Revision: 1,
		Acceptances: []verification.Acceptance{{
			ID:                   hrAcceptance,
			Statement:            "held-out case passes for the candidate without implementation-visible leakage",
			IndependenceRequired: verification.IndependenceInformation,
			EvidenceBindings: []verification.Binding{{
				ID:               hrBinding,
				AuthoringMode:    verification.AuthoringPreCommit,
				VisibilityPolicy: verification.VisibilityHeldOut,
				Basis: &verification.IndependenceBasis{
					Source:      verification.SourceVerificationAuthority,
					Information: verification.InfoHeldOut,
				},
				Evidence:          verification.EvidenceRef{Type: verification.EvidenceTest, Store: "holdout", Ref: hrHoldoutRef},
				ExecutionProtocol: &verification.ExecutionProtocol{Repetitions: 1, Aggregation: verification.AggregationAllPass},
			}},
		}},
		RuntimeRequirements: verification.RuntimeRequirements{
			SessionIsolation:   true,
			HeldOutUnreachable: true,
			ConformanceClass:   "verification-isolation-v1",
		},
	}
	f, err := c.Freeze()
	if err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	return f
}

func hrTaskContract() taskcontract.TaskContract {
	return taskcontract.TaskContract{
		ID:        hrTask,
		Profile:   taskcontract.ProfileStandard,
		Title:     "P2-06 held-out repair safety gate",
		Objective: "Prove safe repair feedback and lineage-monotonic exposure accounting in the closed loop",
		Scope:     taskcontract.Scope{AllowedPaths: []string{"internal/widget"}},
		Acceptance: []taskcontract.AcceptanceClause{
			{ID: hrAcceptance, Statement: "held-out case passes for the candidate", IndependenceRequired: taskcontract.IndependenceInformation},
		},
		Hash: "sha256:p2-06-task-revision",
	}
}

// hrRunCandidate 经真实单 Task 工作流(一次性 worktree + FakeRuntime +
// RunStoreSink)固化一个候选,候选身份是真实 commit 派生的 ChangeSet。
func hrRunCandidate(t *testing.T, runs *runstore.Store, frozen *verification.Frozen, repo, base, mark string, at time.Time) *changeset.ChangeSet {
	t.Helper()
	mgr, err := workspace.New(repo, filepath.Join(t.TempDir(), "wt"))
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	exec, err := workflow.NewTaskExecution(workflow.TaskExecutionSpec{
		Project:              hrProject,
		Task:                 hrTaskContract(),
		Role:                 "coding",
		Policy:               e2ePolicy(),
		BaseRevision:         base,
		Profile:              "default-coding",
		Runtime:              fake.New(fake.Config{}),
		Model:                "test-model",
		BehaviorRevision:     "behavior-1",
		VerificationRevision: frozen.Hash(),
		ContextPackHash:      "sha256:p2-06-context-pack",
		Worktrees:            workspace.NewRunTable(mgr),
		Sink:                 workflow.NewRunStoreSink(runs),
		Hooks:                expFixCandidateHook(t, mark, "package widget // candidate "+mark+"\n"),
		Now:                  func() time.Time { return at },
	})
	if err != nil {
		t.Fatalf("NewTaskExecution(%s): %v", mark, err)
	}
	out, err := exec.Execute(context.Background())
	if err != nil {
		t.Fatalf("Execute(%s): %v", mark, err)
	}
	if out.Status != workflow.TaskCompleted || out.ChangeSet == nil {
		t.Fatalf("候选 %s 未固化: %s", mark, out.Status)
	}
	return out.ChangeSet
}

// hrHoldoutLeakMaterial 构造真实 held-out 泄漏面:FilesystemStore 落盘密标
// 内容,取得 store 根、不透明 ref 与内容摘要 —— 这些串出现在修复上下文即为
// 泄漏;同时给出 Control/Review 侧的判定事实(真实投影出口的唯一输入)。
type hrHoldoutLeakMaterial struct {
	root          string
	ref           string
	contentDigest string
	outputDigest  string
	record        verification.HoldoutVerdictRecord
}

func hrLeakMaterial(t *testing.T, cs changeset.ChangeSetID) hrHoldoutLeakMaterial {
	t.Helper()
	root := filepath.Join(t.TempDir(), "holdout")
	s, err := holdout.NewFilesystemStore(root, t.TempDir())
	if err != nil {
		t.Fatalf("NewFilesystemStore: %v", err)
	}
	content := []byte("P2-06 controlled held-out evidence, marker " + hrSecretMarker + "\n")
	ref, err := s.Put(context.Background(), content)
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	outDigest := "sha256:" + strings.Repeat("77", 32)
	m := hrHoldoutLeakMaterial{
		root:          root,
		ref:           ref.String(),
		contentDigest: holdout.ContentDigest(content),
		outputDigest:  outDigest,
	}
	m.record = verification.HoldoutVerdictRecord{
		TaskID:           hrTask,
		ContractRevision: 1,
		ContractHash:     "sha256:p2-06-frozen-contract",
		AcceptanceID:     hrAcceptance,
		BindingID:        hrBinding,
		CandidateID:      string(cs),
		ContextHash:      "ctx-hash-control-side-only",
		EvidenceType:     verification.EvidenceTest,
		ContentDigest:    m.contentDigest,
		Status:           verification.EvidenceStatusFail,
		Repetitions:      1,
		Failures:         1,
		OutputDigests:    []string{outDigest},
		CompletedAt:      hrAt2,
	}
	return m
}

// hrExecutor 是闭环的执行适配器:按尝试交付预固化的真实候选;修复尝试在途
// 时(可配置)经共享账面蓄意披露一份诊断 —— 这正是「修复期间暴露」的物理
// 位置。它同时是 AC1 的观测面:收到的 ExecuteRequest 就是交给编码上下文的
// 全部材料。
type hrExecutor struct {
	candidates []*changeset.ChangeSet
	calls      []closedloop.ExecuteRequest

	// 蓄意披露配置(测试 1 用;nil = 无暴露)。
	ledger  *closedloop.ExposureLedger
	anchor  changeset.ChangeSetID
	binding evidence.BindingRef
	expose  bool

	t *testing.T
}

func (x *hrExecutor) ExecuteCandidate(ctx context.Context, req closedloop.ExecuteRequest) (closedloop.ExecuteOutcome, error) {
	x.calls = append(x.calls, req)
	if x.expose && req.Attempt.IsRepair() {
		if _, err := x.ledger.RecordExposure(ctx, closedloop.RepairExposureSpec{
			TaskID:    req.TaskID,
			Binding:   x.binding,
			Anchor:    x.anchor,
			Channel:   evidence.ChannelDebugDisclosure,
			Reason:    "repair diagnostic surfaced held-out assertion internals",
			ExposedAt: hrLeakAt,
		}); err != nil {
			return closedloop.ExecuteOutcome{}, fmt.Errorf("蓄意披露记账: %w", err)
		}
	}
	if len(x.candidates) == 0 {
		return closedloop.ExecuteOutcome{}, errors.New("hrExecutor: 候选脚本耗尽")
	}
	cand := x.candidates[0]
	x.candidates = x.candidates[1:]
	return closedloop.ExecuteOutcome{Candidate: cand}, nil
}

// hrVerifier 是闭环的验证适配器:按脚本给出 verdict(验证权威在 P2-08 接线;
// 本测试的被测面是循环内的边界纪律,不是验证服务本身)。
type hrVerifier struct {
	calls []closedloop.VerifyRequest
	steps []closedloop.VerifyOutcome
}

func (v *hrVerifier) VerifyCandidate(_ context.Context, req closedloop.VerifyRequest) (closedloop.VerifyOutcome, error) {
	v.calls = append(v.calls, req)
	if len(v.steps) == 0 {
		return closedloop.VerifyOutcome{}, errors.New("hrVerifier: 脚本耗尽")
	}
	s := v.steps[0]
	v.steps = v.steps[1:]
	return s, nil
}

// hrFeedbackSource 是反馈缝适配器(P2-08 组合面的形态预演):从验证权威
// 已有的类型化事实(P2-29 判定记录 → 真实 RedactHoldoutRecord 出口)经
// P2-04 的 FeedbackBuilder 构造安全投影。
type hrFeedbackSource struct {
	leak    hrHoldoutLeakMaterial
	calls   []closedloop.VerifyFailure
	called  bool
	failOdd bool
}

func (s *hrFeedbackSource) RepairFeedback(_ context.Context, f closedloop.VerifyFailure) (closedloop.RepairFeedback, error) {
	s.calls = append(s.calls, f)
	s.called = true
	if s.failOdd {
		// 材料不足的真实形态:builder 拒绝非 repairable 失败类。
		return closedloop.RepairFeedback{}, closedloop.ErrFailureNotRepairable
	}
	fb, err := closedloop.NewFeedbackBuilder(f.TaskID, f.Attempt, f.ChangeSet, f.RunRef).
		ForAcceptance(hrAcceptance, "held-out case passes for the candidate without implementation-visible leakage").
		WithFailure(closedloop.FailureRepairable).
		WithAffectedBehavior("the hidden case exercises the batch-limit path").
		WithDiagnosticSummary("the held-out case failed; the visible suite stays green").
		AddHoldoutOutcome(verification.RedactHoldoutRecord(s.leak.record), verification.EvidenceStatusFail).
		Build()
	if err != nil {
		return closedloop.RepairFeedback{}, err
	}
	return fb, nil
}

// hrSink 把闭环暴露出口接到真实 evidence_exposure 表(生产适配形态,
// P2-08 组合根同型)。
func hrSink(runs *runstore.Store, frozen *verification.Frozen) closedloop.ExposureSink {
	return closedloop.ExposureSinkFunc(func(ctx context.Context, spec closedloop.RepairExposureSpec) (evidence.Exposure, error) {
		return runs.RecordExposure(ctx, runstore.ExposureRecord{
			Task:             spec.TaskID,
			ContractRevision: int32(frozen.Revision()),
			Binding:          spec.Binding,
			Acceptance:       spec.Acceptance,
			ChangeSet:        evidence.ChangeSetID(spec.Anchor),
			Channel:          spec.Channel,
			Reason:           spec.Reason,
			ExposedAt:        spec.ExposedAt,
		})
	})
}

// hrRun 驱动一次闭环:候选 A 失败(修复裁决)→ 候选 B 复验;返回终态与观测面。
func hrRun(t *testing.T, runs *runstore.Store, frozen *verification.Frozen, csA, csB *changeset.ChangeSet, leak hrHoldoutLeakMaterial, expose, failFeedback bool) (closedloop.Result, *hrExecutor, *hrVerifier, *closedloop.ExposureLedger) {
	t.Helper()
	src := &hrFeedbackSource{leak: leak, failOdd: failFeedback}
	exec := &hrExecutor{candidates: []*changeset.ChangeSet{csA, csB}, t: t, expose: expose, anchor: csA.ID(), binding: evidence.BindingRef{
		Project:      hrProject,
		Verification: evidence.VerificationRevision(frozen.Hash()),
		BindingID:    hrBinding,
	}}
	verify := &hrVerifier{steps: []closedloop.VerifyOutcome{
		{Verdict: closedloop.VerdictFailed, RunRef: "vrun-a"},
		{Verdict: closedloop.VerdictVerified, RunRef: "vrun-b"},
	}}
	sink := hrSink(runs, frozen)
	lineage, err := closedloop.NewStoreLineageSink(runs)
	if err != nil {
		t.Fatalf("NewStoreLineageSink: %v", err)
	}
	budget, err := closedloop.NewRepairPolicy(closedloop.DefaultMaxRepairAttempts)
	if err != nil {
		t.Fatalf("NewRepairPolicy: %v", err)
	}
	router := closedloop.NewFailureRouter(func(closedloop.VerifyFailure) closedloop.FailureClass {
		return closedloop.FailureRepairable
	})
	e, err := closedloop.NewEngine(closedloop.Config{
		Executor:       exec,
		Verifier:       verify,
		Router:         router,
		RepairBudget:   budget,
		LineageSink:    lineage,
		FeedbackSource: src,
		ExposureSink:   sink,
		Now:            func() time.Time { return hrAt2.Add(time.Minute) },
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	exec.ledger = e.ExposureLedger()
	res, err := e.Run(context.Background(), hrTask)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	return res, exec, verify, e.ExposureLedger()
}

// hrNewRepoAndCandidates 组装真实仓库与两个候选(A 失败、B 为修复候选)。
func hrNewRepoAndCandidates(t *testing.T, runs *runstore.Store, frozen *verification.Frozen) (base string, csA, csB *changeset.ChangeSet) {
	t.Helper()
	repo := e2eNewRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, "internal", "widget"), 0o755); err != nil {
		t.Fatalf("建 scope 目录: %v", err)
	}
	base = e2eCommit(t, repo, "internal/widget/widget.go", "package widget\n", "base")
	csA = hrRunCandidate(t, runs, frozen, repo, base, "v1", hrAt)
	csB = hrRunCandidate(t, runs, frozen, repo, base, "v2", hrAt2)
	if csA.ID() == csB.ID() {
		t.Fatalf("候选 A 与 B 身份相同 %s", csA.ID())
	}
	return base, csA, csB
}

// hrAssertNoLeak 扫描反馈投影的值面(JSON 与 %+v):密标、store 根、不透明
// ref、内容/输出摘要、Control 侧上下文哈希、binding 身份一律不得出现(AC1)。
func hrAssertNoLeak(t *testing.T, leak hrHoldoutLeakMaterial, fb closedloop.RepairFeedback) {
	t.Helper()
	rendered := fmt.Sprintf("%+v", fb)
	jsonRaw, err := json.Marshal(fb)
	if err != nil {
		t.Fatalf("json.Marshal(RepairFeedback): %v", err)
	}
	jsonRendered := string(jsonRaw)
	leaks := []string{
		hrSecretMarker,
		leak.root,
		leak.ref,
		leak.contentDigest,
		leak.outputDigest,
		leak.record.ContextHash,
		hrBinding,
	}
	if runtime.GOOS == "windows" {
		leaks = append(leaks, strings.ReplaceAll(leak.root, `\`, `\\`))
	}
	for _, s := range leaks {
		if strings.Contains(rendered, s) {
			t.Fatalf("修复反馈 %%+v 泄漏 %q: %s", s, rendered)
		}
		if strings.Contains(jsonRendered, s) {
			t.Fatalf("修复反馈 JSON 泄漏 %q: %s", s, jsonRendered)
		}
	}
}

// ---- AC1+AC2+AC3:安全反馈 + 蓄意暴露记账 + compromised 暴露阻断接受 ------

func TestV14HoldoutRepairFeedbackSafeAndExposureEscalates(t *testing.T) {
	runs, pool := e2eStore(t)
	frozen := hrFrozenContract(t)
	_, csA, csB := hrNewRepoAndCandidates(t, runs, frozen)
	leak := hrLeakMaterial(t, csA.ID())

	res, exec, _, ledger := hrRun(t, runs, frozen, csA, csB, leak, true, false)

	// AC1:修复尝试收到的失败材料只有安全投影;密标/ref/摘要/身份零出现,
	// 且投影确实可诊断(held-out 类别 fail + acceptance 语义在场)。
	if len(exec.calls) != 2 {
		t.Fatalf("执行调用 = %d, 期望 2", len(exec.calls))
	}
	fb := exec.calls[1].Feedback
	if fb == nil {
		t.Fatal("修复 Run 必须携带安全反馈投影")
	}
	hrAssertNoLeak(t, leak, *fb)
	if fb.Subject.AcceptanceID != hrAcceptance || fb.Failure != closedloop.FailureRepairable {
		t.Fatalf("反馈主体/失败类不符: %+v", fb.Subject)
	}
	if len(fb.Holdout) != 1 || fb.Holdout[0].Status != verification.EvidenceStatusFail {
		t.Fatalf("held-out 投影应恰一条 fail 类别: %+v", fb.Holdout)
	}
	if fb.Holdout[0].Outcome.AcceptanceID != hrAcceptance {
		t.Fatalf("投影 acceptance 不符: %+v", fb.Holdout[0].Outcome)
	}
	// 首次尝试没有失败材料通道。
	if exec.calls[0].Feedback != nil {
		t.Fatalf("首次尝试不应携带反馈: %+v", exec.calls[0].Feedback)
	}

	// AC2:蓄意暴露成为真实 evidence_exposure 行(append-only,数据库身份)。
	facts := ledger.Facts()
	if len(facts) != 1 {
		t.Fatalf("在账暴露 = %d 条, 期望 1", len(facts))
	}
	if n := countRows(t, pool, "SELECT count(*) FROM evidence_exposure"); n != 1 {
		t.Fatalf("evidence_exposure 行数 = %d, 期望 1(AC2)", n)
	}
	stored, err := runs.ExposuresForTask(context.Background(), hrProject, hrTask)
	if err != nil || len(stored) != 1 {
		t.Fatalf("ExposuresForTask: %v %d", err, len(stored))
	}
	exp := stored[0]
	if ch := exp.Channel(); ch != evidence.ChannelDebugDisclosure {
		t.Fatalf("channel = %q, want debug_disclosure", ch)
	}
	if anchor, ok := exp.ChangeSet(); !ok || anchor != evidence.ChangeSetID(csA.ID()) {
		t.Fatalf("暴露锚点 = %v(ok=%v), want 候选 A", anchor, ok)
	}
	if _, err := pool.Exec(context.Background(),
		"UPDATE evidence_exposure SET channel='knowledge_index' WHERE id=$1::bigint", string(exp.ID())); err == nil {
		t.Fatal("append-only 触发器未拦截暴露事实改写(AC2)")
	}
	if _, err := pool.Exec(context.Background(),
		"DELETE FROM evidence_exposure WHERE id=$1::bigint", string(exp.ID())); err == nil {
		t.Fatal("append-only 触发器未拦截暴露事实删除(AC2)")
	}

	// AC3:compromised 暴露封顶接受 —— 候选 verified 在先也不得 accept。
	if res.Phase != closedloop.PhaseEscalated {
		t.Fatalf("Phase = %s, 期望 escalated(AC3)", res.Phase)
	}
	if res.Escalation == nil || res.Escalation.Reason != closedloop.EscalationExposureCompromise {
		t.Fatalf("期望 exposure_compromise 升级,得到 %+v", res.Escalation)
	}
	if res.Basis != nil {
		t.Fatal("compromised 暴露下不得产生 basis(AC3)")
	}
	if len(res.Attempts) != 2 || res.Attempts[1].Supersedes != csA.ID() || res.Attempts[1].ChangeSet != csB.ID() {
		t.Fatalf("尝试谱系不完整: %+v", res.Attempts)
	}
	// 取代谱系在真库里(修复候选就位时经 LineageSink 落库)。
	sess, err := runs.ChangeSetSupersessions(context.Background(), csB.ID())
	if err != nil || len(sess) != 1 || sess[0].Old != csA.ID() {
		t.Fatalf("真库取代事实: %v %+v", err, sess)
	}
	// 真库事实经领域推导给出同一封顶:候选 B 的谱系被 lineage 封顶。
	storedExps, err := runs.ExposuresForBinding(context.Background(), hrProject, frozen.Hash(), hrBinding)
	if err != nil || len(storedExps) != 1 {
		t.Fatalf("ExposuresForBinding: %v %d", err, len(storedExps))
	}
	scope, hit := evidence.CompromiseScope(storedExps, []evidence.ChangeSetID{evidence.ChangeSetID(csB.ID()), evidence.ChangeSetID(csA.ID())}, csB.CreatedAt())
	if !hit || scope != evidence.ChannelScopeLineage {
		t.Fatalf("CompromiseScope = %s/%v, want lineage/true(真库事实与循环裁决一致)", scope, hit)
	}
}

// ---- AC3 对照:无暴露的修复候选正常接受 --------------------------------------

func TestV14HoldoutRepairWithoutExposureAccepts(t *testing.T) {
	runs, pool := e2eStore(t)
	frozen := hrFrozenContract(t)
	_, csA, csB := hrNewRepoAndCandidates(t, runs, frozen)
	leak := hrLeakMaterial(t, csA.ID())

	res, exec, _, ledger := hrRun(t, runs, frozen, csA, csB, leak, false, false)

	if res.Phase != closedloop.PhaseAccepted || res.Basis == nil {
		t.Fatalf("无暴露的修复候选应正常接受: %s / %+v", res.Phase, res.Basis)
	}
	if res.Basis.ChangeSet != csB.ID() || res.Basis.VerifyRun != "vrun-b" || res.Basis.Attempt != 2 {
		t.Fatalf("basis = %+v, 期望绑定修复候选与修复尝试", res.Basis)
	}
	if res.Escalation != nil {
		t.Fatalf("接受终态不得携带升级载荷: %+v", res.Escalation)
	}
	if len(ledger.Facts()) != 0 {
		t.Fatalf("无暴露路径不得产生暴露事实: %d", len(ledger.Facts()))
	}
	if n := countRows(t, pool, "SELECT count(*) FROM evidence_exposure"); n != 0 {
		t.Fatalf("无暴露路径 evidence_exposure 行数 = %d, 期望 0", n)
	}
	_ = exec
}

// ---- AC4:反馈材料不足 → 升级,修复 Run 不启动、零副作用 --------------------

func TestV14HoldoutRepairInsufficientFeedbackEscalates(t *testing.T) {
	runs, pool := e2eStore(t)
	frozen := hrFrozenContract(t)
	_, csA, csB := hrNewRepoAndCandidates(t, runs, frozen)
	leak := hrLeakMaterial(t, csA.ID())

	res, exec, _, ledger := hrRun(t, runs, frozen, csA, csB, leak, true, true)

	if res.Phase != closedloop.PhaseEscalated || res.Escalation == nil ||
		res.Escalation.Reason != closedloop.EscalationFeedbackUnavailable {
		t.Fatalf("期望 feedback_unavailable 升级,得到 %s / %+v", res.Phase, res.Escalation)
	}
	if len(exec.calls) != 1 {
		t.Fatalf("执行调用 = %d, 期望 1(没有安全反馈就不开始修复 Run,AC4)", len(exec.calls))
	}
	if len(ledger.Facts()) != 0 {
		t.Fatalf("修复未启动却产生了暴露事实: %d", len(ledger.Facts()))
	}
	if n := countRows(t, pool, "SELECT count(*) FROM evidence_exposure"); n != 0 {
		t.Fatalf("evidence_exposure 行数 = %d, 期望 0(AC4)", n)
	}
	if sess, err := runs.ChangeSetSupersessions(context.Background(), csB.ID()); err != nil || len(sess) != 0 {
		t.Fatalf("修复未启动却产生了取代事实: %v %+v", err, sess)
	}
	if len(res.Attempts) != 1 {
		t.Fatalf("尝试谱系 = %d, 期望 1", len(res.Attempts))
	}
}
