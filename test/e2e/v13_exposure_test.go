//go:build integration

// P1-49 门禁端到端:exposure 事实翻转 effective information independence
// (P1 Trust Gate,PRD §36.3「至少一次 exposure fact 让 effective information
// independence 变为 compromised 并在 Review 可见」/ INV-VER-05)。
//
// 被测纵向切片跨真实边界组装:P1-39 单 Task 工作流内核驱动 FakeRuntime 与
// 一次性 Git worktree(FixCandidate 在 worktree 内提交,候选身份是真实
// commit),经 RunStoreSink(P1-33 适配)把 CodegenRun / 不可变 ChangeSet
// 事实落进真实 PostgreSQL(每测试独立 schema、自跑 goose 迁移);rework
// 候选 B 经 append-only 取代事实(A→B)构成 change lineage;held-out PASS
// 证据经 verification_run / verification_evidence 精确绑定候选 B;暴露事实
// 经 P1-12 的 evidence_exposure 落库(runstore.RecordExposure),锚定在
// lineage 根候选 A 上。
//
// 证明目标(任务验收):
//
//	AC1 Exposure 事实 append-only:落库获得身份、可按 task / binding /
//	    anchor 投影读回,UPDATE/DELETE 被语句级触发器拒绝;
//	AC2 冻结 VerificationContract 的 visibility policy 不因暴露改写:契约
//	    内容哈希与 binding 的 held_out 声明不变,evidence 行原样只读;
//	AC3 派生 effective information independence 翻转为 compromised:暴露前
//	    binding information-independent、acceptance 覆盖完整;暴露后(唯一
//	    新增事实是暴露)binding 翻转为 false、报告 Complete=false —— 锚定 A
//	    的暴露经 lineage[A→B] 命中当前候选 B;
//	AC4 Review 显示缺口:review.BuildAccounting 的验收行 IndependenceCovered
//	    =false、strict_unmet,出现 independence_gap 阻断项,Approve 被阻断。
//
// 数据库不可达时 SKIP(不产出假失败)。
package e2e

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tally/internal/approval"
	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/review"
	"tally/internal/runstore"
	"tally/internal/taskcontract"
	"tally/internal/verification"
	"tally/internal/workflow"
	"tally/internal/workspace"
	"tally/runtimes/fake"
)

const (
	expProject    = core.ProjectID("proj-p1-49")
	expTask       = "TASK-P1-49"
	expAcceptance = "AC-EXP"
	expBinding    = "holdout-core"
	expFactID     = evidence.FactID("fact-p1-49-holdout-core")
	// expLeakReason 只是追溯 provenance:暴露事实不携带 held-out 内容或可
	// 识别 verdict(INV-VER-05)。
	expLeakReason = "held-out diagnostic surfaced in implementation-visible debug output during rework"
)

var (
	// expAt / expAt2 固定时钟:候选 A、B 固化,verdict 落账与暴露观测的时刻
	// 全部确定,推导与断言不依赖真实时间。
	expAt     = time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)
	expAt2    = expAt.Add(time.Hour)
	expAtVer  = expAt2.Add(2 * time.Minute) // held-out PASS verdict 落账时刻
	expLeakAt = expAt2.Add(5 * time.Minute) // 暴露被观测到的时刻
)

// expFixture 是暴露翻转场景的全部既成事实:lineage A→B(rework)、绑定 B 的
// held-out PASS 证据、冻结契约与 B 的 lineage 投影。
type expFixture struct {
	runs   *runstore.Store
	pool   *pgxpool.Pool
	repo   string
	base   string
	frozen *verification.Frozen
	csA    *changeset.ChangeSet
	csB    *changeset.ChangeSet
	vrB    int64
	// ref 是 holdout 消费键(BindingRef 刻意不含 task,重挂任务不重置暴露账)。
	ref evidence.BindingRef
	// lineage 是 B 的 change lineage:候选在前,被取代的 A 在后。
	lineage []evidence.ChangeSetID
}

// expFrozenContract 冻结一份含 held_out binding、information 独立要求的
// VerificationContract(freeze 的 planned coverage 校验要求非 implementation
// 的 source basis;INV-VER-05 要求声明 runtime 能力)。
func expFrozenContract(t *testing.T) *verification.Frozen {
	t.Helper()
	c := verification.Contract{
		TaskID:   expTask,
		Revision: 1,
		Acceptances: []verification.Acceptance{{
			ID:                   expAcceptance,
			Statement:            "held-out case passes for the candidate without implementation-visible leakage",
			IndependenceRequired: verification.IndependenceInformation,
			EvidenceBindings: []verification.Binding{{
				ID:               expBinding,
				AuthoringMode:    verification.AuthoringPreCommit,
				VisibilityPolicy: verification.VisibilityHeldOut,
				Basis: &verification.IndependenceBasis{
					Source:      verification.SourceVerificationAuthority,
					Information: verification.InfoHeldOut,
				},
				Evidence:          verification.EvidenceRef{Type: verification.EvidenceTest, Store: "holdout", Ref: "p1-49/case-1"},
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

func expTaskContract() taskcontract.TaskContract {
	return taskcontract.TaskContract{
		ID:        expTask,
		Profile:   taskcontract.ProfileStandard,
		Title:     "P1-49 exposure flip gate",
		Objective: "Prove exposure facts compromise effective information independence and Review shows the gap",
		Scope:     taskcontract.Scope{AllowedPaths: []string{"internal/widget"}},
		Acceptance: []taskcontract.AcceptanceClause{
			{ID: expAcceptance, Statement: "held-out case passes for the candidate", IndependenceRequired: taskcontract.IndependenceInformation},
		},
		Hash: "sha256:p1-49-task-revision",
	}
}

// expFixCandidateHook 在 run 的一次性 worktree 里落改动并提交,候选 revision
// 取真实 worktree HEAD —— ChangeSet 同时携带内容身份与可比对的 Git revision,
// review 的确定性 hunk 清单才可从 base..candidate 重建。
func expFixCandidateHook(t *testing.T, mark, content string) workflow.Hooks {
	return workflow.Hooks{FixCandidate: func(_ context.Context, v workflow.IterationView) (workflow.CandidateSnapshot, bool, error) {
		p := filepath.Join(v.WorkspacePath, "internal", "widget", "widget.go")
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return workflow.CandidateSnapshot{}, false, err
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			return workflow.CandidateSnapshot{}, false, err
		}
		e2eGit(t, v.WorkspacePath, "add", ".")
		e2eGit(t, v.WorkspacePath, "commit", "-m", "candidate "+mark)
		rev := e2eGit(t, v.WorkspacePath, "rev-parse", "HEAD")
		return workflow.CandidateSnapshot{
			CandidateRevision: rev,
			Files:             []changeset.ChangedFile{{Path: "internal/widget/widget.go", Content: content}},
			Entities:          []string{"widget"},
		}, true, nil
	}}
}

// expRunCandidate 驱动一次完整的单 Task 工作流执行(真实 worktree + FakeRuntime
// + RunStoreSink 落库);at 是本次执行的固定时钟。
func expRunCandidate(t *testing.T, runs *runstore.Store, frozen *verification.Frozen, repo, base, mark string, at time.Time) workflow.Outcome {
	t.Helper()
	mgr, err := workspace.New(repo, filepath.Join(t.TempDir(), "wt"))
	if err != nil {
		t.Fatalf("workspace.New: %v", err)
	}
	exec, err := workflow.NewTaskExecution(workflow.TaskExecutionSpec{
		Project:              expProject,
		Task:                 expTaskContract(),
		Role:                 "coding",
		Policy:               e2ePolicy(),
		BaseRevision:         base,
		Profile:              "default-coding",
		Runtime:              fake.New(fake.Config{}),
		Model:                "test-model",
		BehaviorRevision:     "behavior-1",
		VerificationRevision: frozen.Hash(),
		ContextPackHash:      "sha256:p1-49-context-pack",
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
	return out
}

// runExposureFixture 组装既成事实:候选 A(v1)→ rework 候选 B(v2)与取代
// 事实,对 B 落一条 held-out PASS 验证证据(精确绑定 B 的快照身份)。
func runExposureFixture(t *testing.T) expFixture {
	t.Helper()
	runs, pool := e2eStore(t)
	frozen := expFrozenContract(t)
	repo := e2eNewRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, "internal", "widget"), 0o755); err != nil {
		t.Fatalf("建 scope 目录: %v", err)
	}
	base := e2eCommit(t, repo, "internal/widget/widget.go", "package widget\n", "base")

	outA := expRunCandidate(t, runs, frozen, repo, base, "v1", expAt)
	outB := expRunCandidate(t, runs, frozen, repo, base, "v2", expAt2)
	if outA.Status != workflow.TaskCompleted || outA.ChangeSet == nil ||
		outB.Status != workflow.TaskCompleted || outB.ChangeSet == nil {
		t.Fatalf("纵向路径未固定候选:A=%s B=%s", outA.Status, outB.Status)
	}
	csA, csB := outA.ChangeSet, outB.ChangeSet
	if csA.ID() == csB.ID() {
		t.Fatalf("候选 A 与 B 身份相同 %s", csA.ID())
	}

	// rework lineage:A 被 B 取代(append-only 事实;快照本体互不改动)。
	fact, err := changeset.NewSupersededFact(csA.ID(), csB.ID(), changeset.SupersedeRework, expAt2)
	if err != nil {
		t.Fatalf("NewSupersededFact: %v", err)
	}
	if err := runs.AppendSupersededFact(context.Background(), fact); err != nil {
		t.Fatalf("AppendSupersededFact: %v", err)
	}

	// held-out 最终验证:verification_run 精确绑定 B 的快照身份,evidence 记
	// held_out 的 PASS(冻结契约哈希是 scope 的 revision 维度)。
	vrB, err := runs.RecordVerificationRun(context.Background(), runstore.VerificationRunSpec{
		Project:           expProject,
		Task:              expTask,
		ContractRevision:  int32(frozen.Revision()),
		ContractHash:      frozen.Hash(),
		ChangeSetID:       csB.ID(),
		EnvName:           "ci",
		EnvDefinitionHash: "sha256:p1-49-envdef",
		VersionCombo:      core.VersionCombo{"go": "1.27", "os": "windows"},
		Runtime:           "fake",
		StartedAt:         expAt2,
		FinishedAt:        expAt2.Add(time.Minute),
		Status:            runstore.VerificationPass,
	})
	if err != nil {
		t.Fatalf("RecordVerificationRun(B): %v", err)
	}
	if _, err := runs.RecordEvidence(context.Background(), expProject, runstore.EvidenceRecord{
		FactID:           expFactID,
		RunID:            vrB,
		Task:             expTask,
		Acceptance:       expAcceptance,
		BindingID:        expBinding,
		VisibilityPolicy: verification.VisibilityHeldOut,
		Verdict:          evidence.VerdictPass,
		Recorder:         evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-e2e/verification"},
		RecordedAt:       expAtVer,
	}); err != nil {
		t.Fatalf("RecordEvidence: %v", err)
	}

	return expFixture{
		runs:   runs,
		pool:   pool,
		repo:   repo,
		base:   base,
		frozen: frozen,
		csA:    csA,
		csB:    csB,
		vrB:    vrB,
		ref: evidence.BindingRef{
			Project:      expProject,
			Verification: evidence.VerificationRevision(frozen.Hash()),
			BindingID:    expBinding,
		},
		lineage: []evidence.ChangeSetID{evidence.ChangeSetID(csB.ID()), evidence.ChangeSetID(csA.ID())},
	}
}

// expRecordExposure 观测并落库一条暴露事实:held-out 诊断进入实现侧可见的
// debug 通道,lineage 局部(channel 分类),锚定在 rework lineage 根候选 A 上。
func expRecordExposure(t *testing.T, f expFixture) evidence.Exposure {
	t.Helper()
	e, err := f.runs.RecordExposure(context.Background(), runstore.ExposureRecord{
		Task:             expTask,
		ContractRevision: int32(f.frozen.Revision()),
		Binding:          f.ref,
		Acceptance:       expAcceptance,
		ChangeSet:        evidence.ChangeSetID(f.csA.ID()),
		Channel:          evidence.ChannelDebugDisclosure,
		Reason:           expLeakReason,
		ExposedAt:        expLeakAt,
	})
	if err != nil {
		t.Fatalf("RecordExposure: %v", err)
	}
	return e
}

// expEvidenceVisibility 直读 evidence 行的 visibility_policy 列(裸 SQL,
// 供暴露前后逐列对比)。
func expEvidenceVisibility(t *testing.T, pool *pgxpool.Pool, factID string) string {
	t.Helper()
	var vis string
	if err := pool.QueryRow(context.Background(),
		"SELECT visibility_policy FROM verification_evidence WHERE fact_id=$1", factID).Scan(&vis); err != nil {
		t.Fatalf("读 evidence %s visibility_policy: %v", factID, err)
	}
	return vis
}

// expIndependenceInput 把库里的 append-only 事实投影成 P1-20 的求值输入:
// 判定记录来自 verification_evidence(validity 经裁定投影),exposure facts
// 取该 binding 的全部记录(推导按 BindingRef 选择;lineage 展开是调用方
// provenance 责任)。
func expIndependenceInput(t *testing.T, f expFixture) []verification.IndependenceEvidence {
	t.Helper()
	ctx := context.Background()
	run, err := f.runs.VerificationRun(ctx, expProject, f.vrB)
	if err != nil {
		t.Fatalf("VerificationRun: %v", err)
	}
	evs, err := f.runs.EvidenceForRun(ctx, expProject, f.vrB)
	if err != nil {
		t.Fatalf("EvidenceForRun: %v", err)
	}
	var verdicts []evidence.VerdictRecord
	for _, ev := range evs {
		if ev.BindingID != expBinding {
			t.Fatalf("意外 binding %q", ev.BindingID)
		}
		rulings, err := f.runs.EvidenceRulings(ctx, expProject, ev.ID)
		if err != nil {
			t.Fatalf("EvidenceRulings: %v", err)
		}
		valid := true
		for _, r := range rulings {
			if r.Kind.Invalidates() {
				valid = false
			}
		}
		verdicts = append(verdicts, evidence.VerdictRecord{
			Binding: evidence.BindingRef{
				Project:      expProject,
				Verification: evidence.VerificationRevision(run.ContractHash),
				BindingID:    ev.BindingID,
			},
			Task:       evidence.TaskID(expTask),
			ChangeSet:  evidence.ChangeSetID(run.ChangeSetID),
			Verdict:    ev.Verdict,
			Valid:      valid,
			RecordedAt: ev.RecordedAt,
		})
	}
	exposures, err := f.runs.ExposuresForBinding(ctx, expProject, f.frozen.Hash(), expBinding)
	if err != nil {
		t.Fatalf("ExposuresForBinding: %v", err)
	}
	return []verification.IndependenceEvidence{{
		BindingID: expBinding,
		Revision:  f.frozen.Revision(),
		Verdicts:  verdicts,
		Exposures: exposures,
	}}
}

// expInfoInput 用与覆盖推导相同的输入直接调用 P1-18 的 per-binding 推导。
func expInfoInput(f expFixture, ie verification.IndependenceEvidence) evidence.InformationIndependenceInput {
	return evidence.InformationIndependenceInput{
		Binding:          f.ref,
		Visibility:       evidence.VisibilityPolicy(verification.VisibilityHeldOut),
		Candidate:        evidence.ChangeSetID(f.csB.ID()),
		CandidateFixedAt: f.csB.CreatedAt(),
		Lineage:          f.lineage,
		Exposures:        ie.Exposures,
		Verdicts:         ie.Verdicts,
	}
}

// expDeriveIndependence 对当前候选 B 推导 effective independence 覆盖。
// Implementation provenance 如实提供(本切片候选由实现侧 FakeRuntime 产出):
// 生产界面尚无该 provenance 的事实来源时保守取零值;这里提供它,使暴露前的
// 独立成立、翻转可归因于暴露事实本身,而不是缺省 provenance。
func expDeriveIndependence(t *testing.T, f expFixture, inputs []verification.IndependenceEvidence) verification.IndependenceCoverageReport {
	t.Helper()
	rep, err := verification.DeriveIndependenceCoverage(f.frozen, verification.IndependenceContext{
		Project:          expProject,
		Verification:     evidence.VerificationRevision(f.frozen.Hash()),
		ChangeSet:        evidence.ChangeSetID(f.csB.ID()),
		CandidateFixedAt: f.csB.CreatedAt(),
		Lineage:          f.lineage,
		Implementation: evidence.ImplementationProvenance{
			Contributors: []evidence.Contributor{{Name: "fake", Role: "coding_agent"}},
		},
	}, inputs)
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage: %v", err)
	}
	return rep
}

// expFunctionalCoverage 从同一批 evidence 事实推导功能覆盖(与 independence
// 覆盖平行,§8.8.3:暴露不动功能门的事实)。
func expFunctionalCoverage(t *testing.T, f expFixture) verification.FunctionalCoverageReport {
	t.Helper()
	ctx := context.Background()
	evs, err := f.runs.EvidenceForRun(ctx, expProject, f.vrB)
	if err != nil {
		t.Fatalf("EvidenceForRun: %v", err)
	}
	var records []verification.EvidenceRecord
	for _, ev := range evs {
		var status verification.EvidenceStatus
		switch ev.Verdict {
		case evidence.VerdictPass:
			status = verification.EvidenceStatusPass
		case evidence.VerdictFail:
			status = verification.EvidenceStatusFail
		default:
			continue // error 无可用判定
		}
		rulings, err := f.runs.EvidenceRulings(ctx, expProject, ev.ID)
		if err != nil {
			t.Fatalf("EvidenceRulings: %v", err)
		}
		validity := verification.EvidenceValidityValid
		for _, r := range rulings {
			if r.Kind.Invalidates() {
				validity = verification.EvidenceValidityInvalidated
			}
		}
		records = append(records, verification.EvidenceRecord{
			BindingID: ev.BindingID,
			Revision:  f.frozen.Revision(),
			Status:    status,
			Validity:  validity,
		})
	}
	rep, err := verification.DeriveFunctionalCoverage(f.frozen, records)
	if err != nil {
		t.Fatalf("DeriveFunctionalCoverage: %v", err)
	}
	return rep
}

// expBuildReview 对候选 B 做确定性 review 归账(与评审页同一装配链):
// base..candidate diff → 确定性 hunk 清单 → 归账规则 → 显式归属后产出报告。
func expBuildReview(t *testing.T, f expFixture, functional verification.FunctionalCoverageReport, independence verification.IndependenceCoverageReport) review.AccountingReport {
	t.Helper()
	diff := e2eGit(t, f.repo, "diff", f.base, f.csB.CandidateRevision())
	inv, err := review.InventoryFromDiff(diff)
	if err != nil {
		t.Fatalf("InventoryFromDiff: %v", err)
	}
	if len(inv.Files) != 1 || len(inv.Files[0].Hunks) == 0 {
		t.Fatalf("清单形态不符(files=%d),归账场景假设单文件单 hunk", len(inv.Files))
	}
	rules, err := review.RulesFor(expTaskContract().Scope, f.frozen)
	if err != nil {
		t.Fatalf("RulesFor: %v", err)
	}
	rep, err := review.BuildAccounting(review.AccountingInput{
		ChangeSet:    f.csB,
		Inventory:    inv,
		Rules:        rules,
		Frozen:       f.frozen,
		Functional:   functional,
		Independence: independence,
		Attributions: []review.Attribution{{
			HunkID:        inv.Files[0].Hunks[0].ID,
			AcceptanceIDs: []string{expAcceptance},
		}},
	})
	if err != nil {
		t.Fatalf("BuildAccounting: %v", err)
	}
	return rep
}

// ---- AC1:Exposure 事实 append-only;AC2:冻结契约 visibility 不变 -----------

func TestV13ExposureFactAppendOnly(t *testing.T) {
	f := runExposureFixture(t)
	ctx := context.Background()

	// 观测前:没有任何暴露事实;契约侧事实(evidence 行 held_out)已就位。
	if n := countRows(t, f.pool, "SELECT count(*) FROM evidence_exposure"); n != 0 {
		t.Fatalf("观测前已有 %d 行暴露事实", n)
	}
	if vis := expEvidenceVisibility(t, f.pool, string(expFactID)); vis != string(verification.VisibilityHeldOut) {
		t.Fatalf("观测前 evidence visibility_policy = %q, want held_out", vis)
	}

	// 观测:暴露落库并获得身份(领域身份 = 数据库 serial 十进制串)。
	exp := expRecordExposure(t, f)
	if exp.ID() == "" {
		t.Fatal("暴露事实没有获得身份")
	}
	serialID, err := strconv.ParseInt(string(exp.ID()), 10, 64)
	if err != nil || serialID <= 0 {
		t.Fatalf("暴露身份 %q 不是数据库 serial 的十进制串", exp.ID())
	}
	if exp.Binding() != f.ref {
		t.Fatalf("binding ref = %+v, want %+v", exp.Binding(), f.ref)
	}
	if ch := exp.Channel(); ch != evidence.ChannelDebugDisclosure {
		t.Fatalf("channel = %q, want debug_disclosure", ch)
	}
	if anchor, ok := exp.ChangeSet(); !ok || anchor != evidence.ChangeSetID(f.csA.ID()) {
		t.Fatalf("暴露锚点 = %v(ok=%v), want 候选 A %s", anchor, ok, f.csA.ID())
	}
	if !exp.ExposedAt().Equal(expLeakAt) {
		t.Fatalf("exposed_at = %v, want %v(UTC 归一)", exp.ExposedAt(), expLeakAt)
	}
	if exp.Reason() != expLeakReason {
		t.Fatalf("reason = %q, want 追溯 provenance 原样", exp.Reason())
	}
	if exp.BindingWide() {
		t.Fatal("debug_disclosure 被分类为 binding-wide;lineage 局部通道不得按持久/全局通道处置")
	}

	// 读回:task / binding / anchor 三条投影路径都命中同一条事实且逐字段保真。
	forTask, err := f.runs.ExposuresForTask(ctx, expProject, expTask)
	if err != nil || len(forTask) != 1 || forTask[0].ID() != exp.ID() {
		t.Fatalf("ExposuresForTask: %v %d", err, len(forTask))
	}
	forBinding, err := f.runs.ExposuresForBinding(ctx, expProject, f.frozen.Hash(), expBinding)
	if err != nil || len(forBinding) != 1 || forBinding[0].ID() != exp.ID() {
		t.Fatalf("ExposuresForBinding: %v %d", err, len(forBinding))
	}
	if got := forBinding[0]; got.Channel() != exp.Channel() || got.Reason() != exp.Reason() ||
		!got.ExposedAt().Equal(exp.ExposedAt()) || got.Binding() != f.ref {
		t.Fatalf("暴露事实读回不一致: %+v", got)
	}
	forA, err := f.runs.ExposuresForChangeSet(ctx, expProject, f.csA.ID())
	if err != nil || len(forA) != 1 || forA[0].ID() != exp.ID() {
		t.Fatalf("anchor=A 的暴露投影: %v %d", err, len(forA))
	}
	// 锚定 A 的事实不直接出现在 B 的 anchor 查询下:B 的污染由推导经
	// lineage[A→B] 得出(展开是调用方 provenance 责任,不是存储层改写)。
	if forB, err := f.runs.ExposuresForChangeSet(ctx, expProject, f.csB.ID()); err != nil || len(forB) != 0 {
		t.Fatalf("anchor=B 的暴露投影应永空: %v %d", err, len(forB))
	}

	// append-only:任何改写/删除都被语句级触发器拒绝,事实原样保留。
	mustModifyRejected(t, f.pool, "UPDATE evidence_exposure SET channel='knowledge_index' WHERE id=$1::bigint", string(exp.ID()))
	mustModifyRejected(t, f.pool, "UPDATE evidence_exposure SET detail='{\"reason\":\"tampered\"}'::jsonb WHERE id=$1::bigint", string(exp.ID()))
	mustModifyRejected(t, f.pool, "DELETE FROM evidence_exposure WHERE id=$1::bigint", string(exp.ID()))
	mustModifyRejected(t, f.pool, "DELETE FROM evidence_exposure")
	if n := countRows(t, f.pool, "SELECT count(*) FROM evidence_exposure"); n != 1 {
		t.Fatalf("拒绝改写后暴露事实行数 = %d, want 1", n)
	}
	if still, err := f.runs.ExposuresForBinding(ctx, expProject, f.frozen.Hash(), expBinding); err != nil ||
		len(still) != 1 || still[0].Reason() != expLeakReason || !still[0].ExposedAt().Equal(expLeakAt) {
		t.Fatalf("拒绝路径没有原样保留暴露事实: %v %+v", err, still)
	}

	// AC2:暴露不改写冻结契约的 visibility policy —— evidence 行原样只读,
	// 冻结契约的 held_out 声明与内容哈希都保持不变。
	if vis := expEvidenceVisibility(t, f.pool, string(expFactID)); vis != string(verification.VisibilityHeldOut) {
		t.Fatalf("暴露后 evidence visibility_policy = %q, want held_out(AC2)", vis)
	}
	mustModifyRejected(t, f.pool, "UPDATE verification_evidence SET visibility_policy='visible' WHERE fact_id=$1", string(expFactID))
	mustModifyRejected(t, f.pool, "DELETE FROM verification_evidence WHERE fact_id=$1", string(expFactID))
	ev, err := f.runs.Evidence(ctx, expProject, expFactID)
	if err != nil || ev.VisibilityPolicy != verification.VisibilityHeldOut || ev.Verdict != evidence.VerdictPass {
		t.Fatalf("暴露后 evidence 读回: %v %+v", err, ev)
	}
	if b := f.frozen.Contract().Acceptances[0].EvidenceBindings[0]; b.VisibilityPolicy != verification.VisibilityHeldOut {
		t.Fatalf("冻结契约 binding visibility = %q, want held_out(AC2)", b.VisibilityPolicy)
	}
	runRow, err := f.runs.VerificationRun(ctx, expProject, f.vrB)
	if err != nil || runRow.ContractHash != f.frozen.Hash() {
		t.Fatalf("暴露后 verification_run 契约哈希不一致: %v %q != %q", err, runRow.ContractHash, f.frozen.Hash())
	}
}

// ---- AC3:派生 effective information independence 翻转为 compromised -------

func TestV13ExposureFlipsInformationIndependence(t *testing.T) {
	f := runExposureFixture(t)

	// 暴露前:held-out PASS 证据使 binding 的有效信息独立,acceptance 覆盖
	// 完整(planned ✓ = effective ✓)。
	inBefore := expIndependenceInput(t, f)
	if len(inBefore) != 1 || len(inBefore[0].Verdicts) != 1 || len(inBefore[0].Exposures) != 0 {
		t.Fatalf("暴露前求值输入不一致(应有 1 条判定、0 条暴露): %+v", inBefore)
	}
	if !evidence.EffectiveInformationIndependence(expInfoInput(f, inBefore[0])) {
		t.Fatal("暴露前 effective information independence = false, want true")
	}
	repBefore := expDeriveIndependence(t, f, inBefore)
	if !repBefore.Complete {
		t.Fatal("暴露前 independence 覆盖应完整")
	}
	accBefore, bBefore := repBefore.Acceptances[0], repBefore.Acceptances[0].Bindings[0]
	if !accBefore.Covered || len(accBefore.SatisfiedBy) != 1 || accBefore.SatisfiedBy[0] != expBinding {
		t.Fatalf("暴露前 acceptance 覆盖不一致: covered=%v satisfiedBy=%v", accBefore.Covered, accBefore.SatisfiedBy)
	}
	if !bBefore.Applicable || !bBefore.SourceIndependent || !bBefore.InformationIndependent || !bBefore.Qualifies {
		t.Fatalf("暴露前 binding 推导不一致: %+v", bBefore)
	}

	// 观测暴露(append-only 与不可改写性已由 TestV13ExposureFactAppendOnly 证明)。
	expRecordExposure(t, f)

	// 暴露后:同一投影路径,唯一新增的事实是这条暴露 —— 派生翻转为 compromised。
	inAfter := expIndependenceInput(t, f)
	if len(inAfter) != 1 || len(inAfter[0].Exposures) != 1 || len(inAfter[0].Verdicts) != 1 {
		t.Fatalf("暴露后求值输入不一致(判定不变,新增恰一条暴露): %+v", inAfter)
	}
	if evidence.EffectiveInformationIndependence(expInfoInput(f, inAfter[0])) {
		t.Fatal("暴露后 effective information independence 仍为 true:暴露事实没有生效(AC3)")
	}
	repAfter := expDeriveIndependence(t, f, inAfter)
	accAfter, bAfter := repAfter.Acceptances[0], repAfter.Acceptances[0].Bindings[0]
	if bAfter.Applicable != bBefore.Applicable {
		t.Fatalf("暴露只允许改变独立性维度: applicability %v -> %v", bBefore.Applicable, bAfter.Applicable)
	}
	// 锚定 A 的暴露经 lineage[B,A] 命中当前候选 B:内容暴露污染整条 change
	// lineage,不只是被锚定的快照(§8.6.1)。
	if bAfter.InformationIndependent {
		t.Fatal("暴露后 binding 仍 information-independent;lineage 展开未命中(AC3)")
	}
	if bAfter.Qualifies || accAfter.Covered || len(accAfter.SatisfiedBy) != 0 {
		t.Fatalf("暴露后 acceptance 覆盖未收口: qualifies=%v covered=%v satisfiedBy=%v",
			bAfter.Qualifies, accAfter.Covered, accAfter.SatisfiedBy)
	}
	if repAfter.Complete {
		t.Fatal("暴露后 independence 覆盖报告仍 Complete(AC3)")
	}

	// AC2:翻转来自 observed fact,不是契约被改写 —— 冻结契约的 held_out
	// 声明与 evidence 行在翻转前后原样不变。
	if b := f.frozen.Contract().Acceptances[0].EvidenceBindings[0]; b.VisibilityPolicy != verification.VisibilityHeldOut {
		t.Fatalf("翻转后冻结契约 binding visibility = %q, want held_out(AC2)", b.VisibilityPolicy)
	}
	if vis := expEvidenceVisibility(t, f.pool, string(expFactID)); vis != string(verification.VisibilityHeldOut) {
		t.Fatalf("翻转后 evidence visibility_policy = %q, want held_out(AC2)", vis)
	}
}

// ---- AC4:Review 显示缺口 ----------------------------------------------------

func TestV13ReviewShowsExposureGap(t *testing.T) {
	f := runExposureFixture(t)

	// 暴露前:功能覆盖与 independence 覆盖都成立,归账无阻断、可 Approve
	// (唯一改动 hunk 已显式归属到验收条款)。
	functional := expFunctionalCoverage(t, f)
	if acc := functional.Acceptances[0]; !acc.Covered {
		t.Fatalf("暴露前功能覆盖应成立: %+v", acc)
	}
	inBefore := expIndependenceInput(t, f)
	repBefore := expDeriveIndependence(t, f, inBefore)
	revBefore := expBuildReview(t, f, functional, repBefore)
	accBefore := revBefore.Acceptances[0]
	if !accBefore.FunctionalCovered || !accBefore.IndependenceCovered {
		t.Fatalf("暴露前 Review 覆盖行不一致: functional=%v independence=%v",
			accBefore.FunctionalCovered, accBefore.IndependenceCovered)
	}
	if accBefore.Strict != review.StrictSatisfied {
		t.Fatalf("暴露前 strict 结论 = %q, want strict_satisfied", accBefore.Strict)
	}
	if len(revBefore.Blockers) != 0 || revBefore.ApproveBlocked() {
		t.Fatalf("暴露前不应有阻断项: %+v", revBefore.Blockers)
	}
	if revBefore.StrictOutcome != approval.StrictSatisfied {
		t.Fatalf("暴露前候选级 strict 结论 = %q, want strict_satisfied", revBefore.StrictOutcome)
	}

	// 观测暴露:此后 Review 必须显示缺口。
	expRecordExposure(t, f)

	repAfter := expDeriveIndependence(t, f, expIndependenceInput(t, f))
	revAfter := expBuildReview(t, f, functional, repAfter)
	accAfter := revAfter.Acceptances[0]
	if !accAfter.FunctionalCovered {
		t.Fatalf("功能覆盖是独立维度,不受暴露影响(§8.8.3): %+v", accAfter)
	}
	if accAfter.IndependenceCovered {
		t.Fatal("暴露后 Review 仍显示 independence 覆盖成立(AC4)")
	}
	if accAfter.Strict != review.StrictUnmet {
		t.Fatalf("暴露后 strict 结论 = %q, want strict_unmet(AC4)", accAfter.Strict)
	}
	if revAfter.StrictOutcome == approval.StrictSatisfied || revAfter.StrictOutcome == approval.StrictWaived {
		t.Fatalf("暴露后候选级 strict 结论 = %q:缺口不得呈现为满足/豁免(§8.10)", revAfter.StrictOutcome)
	}
	if revAfter.StrictResolution == nil || !errors.Is(revAfter.StrictResolution, approval.ErrWaiverRequired) {
		t.Fatalf("暴露后 strict 判定应停在「未满足且无豁免」: %v", revAfter.StrictResolution)
	}
	// 恰好一个 independence_gap 阻断项指向该验收条款;Approve 被阻断。
	var gaps []review.Blocker
	for _, b := range revAfter.Blockers {
		if b.Kind == review.BlockerIndependenceGap {
			gaps = append(gaps, b)
		}
	}
	if len(gaps) != 1 || gaps[0].AcceptanceID != expAcceptance {
		t.Fatalf("暴露后 independence_gap 阻断项不一致: %+v", revAfter.Blockers)
	}
	if !revAfter.ApproveBlocked() {
		t.Fatal("存在未豁免 independence 缺口但 Approve 未被阻断(AC4)")
	}
}
