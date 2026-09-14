//go:build integration

// P3-03 — Bug Fix 闭环 E2E(PRD §24.1 / §32 Scenario A)。被测纵切 = 真实
// worker v1.4 路径的测试等价组合(场景 harness 见 v14_bugfix_harness_test.go):
//
//	AC1 基准失败复现 → TestV14BugFixE2EReproFailsAutoRepairsAccepts
//	    (冻结回归协议在修复前基线上确定性 exit 1;缺陷边界在基线源码中)
//	AC2 候选回归失败 → 自动修复 → PASS → 同上
//	    (错修复候选 FAIL → 可修复路由 → 新 Run + 新一次性 worktree + 修复基
//	    物化 + 安全反馈 + reason=rework 取代 → 真修复候选复验 PASS → accepted)
//	AC3 最终接受候选满足必需证据 → 同上
//	    (accepted 带 basis;束门禁经真实 EvaluateGate 对真实执行投影求值:
//	    修复前不满足且 UnmetKeys 确定性、修复后同一门禁满足;P3-01 可修复
//	    缺陷夹具与真实执行逐键对账)
//	AC4 hands-off/介入事件可度量 →
//	    TestV14BugFixE2EReproFailsAutoRepairsAccepts(hands-off 收口:0 介入)
//	    TestV14BugFixE2ERepairExhaustionInterventionEscalates(修复耗尽 →
//	    恰好 1 次类型化人工介入事件;内层迭代与外层修复可区分地落库)
package e2e

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"tally/internal/changeset"
	"tally/internal/closedloop"
	"tally/internal/engineeringpolicy"
	"tally/internal/evidence"
	"tally/internal/runstore"
)

// ---- AC1/AC2/AC3/AC4:复现 → 回归 FAIL → 自动修复 → accepted(basis)--------

func TestV14BugFixE2EReproFailsAutoRepairsAccepts(t *testing.T) {
	runs, st := clStore(t)
	repo := bfNewRepo(t)
	cfg := bfConfigFor(t, repo)
	head := clGit(t, repo, "rev-parse", "HEAD")

	// AC1:基准失败复现 —— 冻结的回归/行为协议在修复前基线上确定性 exit 1,
	// 且基线源码携带夹具所述的缺陷边界(令牌桶 off-by-one)。
	if got := bfExitCode(t, repo, bfRegressionArgv()); got != 1 {
		t.Fatalf("修复前基线回归检查 exit = %d, 期望 1(缺陷必须在基线复现)", got)
	}
	if got := bfExitCode(t, repo, bfBehaviorArgv()); got != 1 {
		t.Fatalf("修复前基线行为验收 exit = %d, 期望 1", got)
	}
	baseSrc, err := os.ReadFile(filepath.Join(repo, filepath.FromSlash(bfLimiterRel)))
	if err != nil {
		t.Fatalf("读基线源码: %v", err)
	}
	if !strings.Contains(string(baseSrc), "capacity+1") {
		t.Fatalf("基线源码缺少缺陷边界表达式 capacity+1:\n%s", baseSrc)
	}

	// 执行前装配:束 Git-first 装载 + 真实匹配器命中 + 门禁键编译。
	_, matched := bfMatchBundle(t, repo, head)
	if got := matched.GateKeys(); !reflect.DeepEqual(got, []string{"bug_repro_failed_before", "regression_check_passes_after"}) {
		t.Fatalf("命中策略 GateKeys = %v, 期望两条回归证据键", got)
	}

	// 脚本:尝试 1 = 形似修复但仍放行第 4 个请求的候选(回归检查确定性
	// FAIL);尝试 2 = 真修复(复验 PASS)。修复尝试开始前断言修复基(冻结
	// 前候选)已在一次性 worktree 内逐字物化。
	var sawMaterialized string
	rt := newBfScriptedRuntime(
		bfStep{write: bfWrongFix1()},
		bfStep{write: bfFixedContent(), inspect: func(ws, instruction string) {
			b, rErr := os.ReadFile(filepath.Join(ws, filepath.FromSlash(bfLimiterRel)))
			if rErr != nil {
				t.Errorf("修复基未物化: %v", rErr)
				return
			}
			sawMaterialized = string(b)
			if !strings.Contains(instruction, "Repair context") {
				t.Errorf("修复迭代指令缺安全反馈段: %q", instruction)
			}
		}},
	)

	obs := bfRunTask(t, context.Background(), bfDeps{cfg: cfg, queries: st.Queries(), runtime: rt}, bfTaskID)
	res := obs.Result

	// 终态:accepted@attempt2 且必带 basis(每个可接受结果携带依据)。
	if res.Phase != closedloop.PhaseAccepted || res.Basis == nil || res.Basis.Attempt != 2 {
		t.Fatalf("终态 = %s basis=%+v, 期望 accepted@attempt2", res.Phase, res.Basis)
	}
	a1, a2 := res.Attempts[0], res.Attempts[1]
	if a2.ParentRun != a1.RunID || a2.Supersedes != a1.ChangeSet {
		t.Fatalf("修复尝试 = %+v, 期望父链接首次尝试并取代其候选", a2)
	}
	if a1.ChangeSet == "" || a1.ChangeSet == a2.ChangeSet {
		t.Fatalf("候选身份 = %s/%s, 期望两个全新不同候选", a1.ChangeSet, a2.ChangeSet)
	}
	if a1.RunID != closedloop.RunIDFor(bfTaskID, 1) || a2.RunID != closedloop.RunIDFor(bfTaskID, 2) {
		t.Fatalf("尝试身份 = %s/%s, 期望 (task, attempt) 幂等推导", a1.RunID, a2.RunID)
	}
	if sawMaterialized != bfWrongFix1() {
		t.Fatalf("修复基内容 = %q, 期望前候选内容逐字一致", sawMaterialized)
	}

	// 安全反馈边界(P2-06/ADR-0001 §4):首次尝试无失败材料通道;修复反馈
	// 只含 allowlist 投影(失败验收语句/绑定 id/失败类别/诊断摘要)。
	instrs := obs.Instrs
	if len(instrs) != 2 {
		t.Fatalf("迭代指令 = %d 条, 期望 2", len(instrs))
	}
	if strings.Contains(instrs[0], "Repair context") {
		t.Errorf("首次尝试不应携带修复反馈: %q", instrs[0])
	}
	for _, want := range []string{
		"AC-02",
		"a regression check fails on the pre-fix limiter and passes after the fix",
		"repairable",
		bfRegressionBinding,
		"diagnostic summary",
	} {
		if !strings.Contains(instrs[1], want) {
			t.Errorf("修复反馈缺 allowlist 材料 %q: %q", want, instrs[1])
		}
	}

	// 真库事实:取代事实 a1→a2 rework;Run 链父链接;两次独立验证 fail→pass
	// 各自精确绑定候选与同一冻结契约(复验绝不放宽)。
	ctx := context.Background()
	supers, err := runs.ChangeSetSupersessions(ctx, a2.ChangeSet)
	if err != nil || len(supers) != 1 || supers[0].Old != a1.ChangeSet ||
		supers[0].New != a2.ChangeSet || supers[0].Reason != changeset.SupersedeRework {
		t.Fatalf("取代事实 = %+v err=%v, 期望 a1→a2 rework", supers, err)
	}
	timeline, err := runs.CodegenRunsForTask(ctx, bfProject, bfTaskID)
	if err != nil || len(timeline) != 2 {
		t.Fatalf("codegen_run = %d 行 err=%v, 期望 2", len(timeline), err)
	}
	first, second := timeline[0], timeline[1]
	if second.EscalatedFrom != first.ID || first.EscalatedFrom != 0 {
		t.Fatalf("Run 链父链接 = %d←%d, 期望 attempt2 链接 attempt1", second.EscalatedFrom, first.ID)
	}
	if first.Status != runstore.RunCompleted || second.Status != runstore.RunCompleted {
		t.Fatalf("Run 终态 = %s/%s, 期望 completed/completed", first.Status, second.Status)
	}
	if anc, err := runs.ChangeSetAncestors(ctx, a2.ChangeSet); err != nil || len(anc) != 1 || anc[0] != a1.ChangeSet {
		t.Fatalf("ancestors(B) = %v err=%v, 期望 [A]", anc, err)
	}
	vruns, err := runs.VerificationRunsForTask(ctx, bfProject, bfTaskID)
	if err != nil || len(vruns) != 2 {
		t.Fatalf("verification_run = %d 行 err=%v, 期望 2", len(vruns), err)
	}
	byCS := map[changeset.ChangeSetID]runstore.VerificationStatus{
		vruns[0].ChangeSetID: vruns[0].Status,
		vruns[1].ChangeSetID: vruns[1].Status,
	}
	if byCS[a1.ChangeSet] != runstore.VerificationFail || byCS[a2.ChangeSet] != runstore.VerificationPass {
		t.Fatalf("验证结论 = %+v, 期望 A fail / B pass", byCS)
	}
	var failRunID, passRunID int64
	for _, vr := range vruns {
		if vr.ContractHash != obs.Frozen.Hash() {
			t.Fatalf("验证运行 %d 契约 = %s, 期望冻结契约逐字不变", vr.ID, vr.ContractHash)
		}
		switch vr.ChangeSetID {
		case a1.ChangeSet:
			failRunID = vr.ID
		case a2.ChangeSet:
			passRunID = vr.ID
		}
	}
	if failRunID == 0 || passRunID == 0 {
		t.Fatalf("验证运行绑定 = %+v, 期望各自精确绑定 A/B 候选", vruns)
	}
	if res.Basis.ChangeSet != a2.ChangeSet || res.Basis.VerifyRun != strconv.FormatInt(passRunID, 10) {
		t.Fatalf("basis = %+v, 期望绑定修复候选与真库验证运行 %d", res.Basis, passRunID)
	}

	// 回归证据面:同一冻结回归检查(REG-1)修复前确定性失败 = bug 复现已
	// 确立(bug_repro_failed_before),修复后通过(regression_check_passes_
	// after);行为验收(AC-01)随之交付。
	if got := bfBindingVerdict(t, runs, bfProject, failRunID, bfRegressionBinding); got != evidence.VerdictFail {
		t.Fatalf("修复前 %s verdict = %s, 期望 fail(bug_repro_failed_before)", bfRegressionBinding, got)
	}
	if got := bfBindingVerdict(t, runs, bfProject, failRunID, bfBehaviorBinding); got != evidence.VerdictFail {
		t.Fatalf("修复前 %s verdict = %s, 期望 fail(行为验收未交付)", bfBehaviorBinding, got)
	}
	if got := bfBindingVerdict(t, runs, bfProject, passRunID, bfRegressionBinding); got != evidence.VerdictPass {
		t.Fatalf("修复后 %s verdict = %s, 期望 pass(regression_check_passes_after)", bfRegressionBinding, got)
	}
	if got := bfBindingVerdict(t, runs, bfProject, passRunID, bfBehaviorBinding); got != evidence.VerdictPass {
		t.Fatalf("修复后 %s verdict = %s, 期望 pass", bfBehaviorBinding, got)
	}

	// AC3 的门禁语义:用真实执行产出的满足投影经真实门禁求值 —— 修复前
	// 不满足且未满足键确定性,修复后同一门禁满足;P3-01 可修复缺陷夹具的
	// pre/post 投影与真实执行逐键对账。
	pre := []engineeringpolicy.RequirementFulfillment{
		{Key: "bug_repro_failed_before", Satisfied: true},
		{Key: "regression_check_passes_after", Satisfied: false},
	}
	post := []engineeringpolicy.RequirementFulfillment{
		{Key: "bug_repro_failed_before", Satisfied: true},
		{Key: "regression_check_passes_after", Satisfied: true},
	}
	bfAssertFixtureGateParity(t, bfLoadRepairFixture(t), pre, post)
	preGate, err := matched.EvaluateGate(pre)
	if err != nil || preGate.Satisfied || !reflect.DeepEqual(preGate.UnmetKeys, []string{"regression_check_passes_after"}) {
		t.Fatalf("修复前门禁 = %+v err=%v, 期望不满足且 UnmetKeys=[regression_check_passes_after]", preGate, err)
	}
	postGate, err := matched.EvaluateGate(post)
	if err != nil || !postGate.Satisfied || len(postGate.UnmetKeys) != 0 {
		t.Fatalf("修复后门禁 = %+v err=%v, 期望同一门禁满足(不放宽)", postGate, err)
	}

	// AC4:hands-off 事件可度量 —— 回归失败被闭环自动修复收口,零人工介入。
	if led := bfLedger(res); !led.HandsOff || led.AutomaticRepairs != 1 || len(led.InterventionEvents) != 0 {
		t.Fatalf("事件账目 = %+v, 期望 hands-off + 1 次自动修复 + 0 介入事件", led)
	}

	clAssertOwnerClean(t, repo)
	clAssertRootEmpty(t, cfg.WorktreeRoot)
}

// ---- AC4(升级腿):修复耗尽 → 恰好一次类型化人工介入事件 --------------------

func TestV14BugFixE2ERepairExhaustionInterventionEscalates(t *testing.T) {
	runs, st := clStore(t)
	repo := bfNewRepo(t)
	cfg := bfConfigFor(t, repo)

	// 缺省修复预算 2(决策锁 23):首试 + 2 次修复全部仍是含缺陷候选 →
	// 第 3 次修复被拒 → 显式升级(绝不无限修复、绝不静默接受)。
	rt := newBfScriptedRuntime(
		bfStep{write: bfWrongFix1()},
		bfStep{write: bfWrongFix2()},
		bfStep{write: bfWrongFix3()},
	)

	res := bfRunTask(t, context.Background(), bfDeps{cfg: cfg, queries: st.Queries(), runtime: rt}, bfTaskID).Result

	if res.Phase != closedloop.PhaseEscalated || res.Escalation == nil ||
		res.Escalation.Reason != closedloop.EscalationRepairExhausted {
		t.Fatalf("终态 = %s / %+v, 期望 repair_exhausted 升级", res.Phase, res.Escalation)
	}
	if res.Basis != nil {
		t.Fatalf("耗尽升级不得携带 basis: %+v", res.Basis)
	}
	if len(res.Attempts) != 3 {
		t.Fatalf("尝试 = %d, 期望 3(首试 + 2 次修复)", len(res.Attempts))
	}
	a1, a2, a3 := res.Attempts[0], res.Attempts[1], res.Attempts[2]
	if a2.Supersedes != a1.ChangeSet || a3.Supersedes != a2.ChangeSet {
		t.Fatalf("取代链 = %+v, 期望 A→B→C", res.Attempts)
	}

	ctx := context.Background()
	if n := clCountRows(t, st, "SELECT count(*) FROM changeset_superseded_fact"); n != 2 {
		t.Fatalf("取代事实 = %d 行, 期望 2", n)
	}
	if anc, err := runs.ChangeSetAncestors(ctx, a3.ChangeSet); err != nil ||
		len(anc) != 2 || anc[0] != a2.ChangeSet || anc[1] != a1.ChangeSet {
		t.Fatalf("ancestors(C) = %v err=%v, 期望 [B, A](多代谱系)", anc, err)
	}
	timeline, err := runs.CodegenRunsForTask(ctx, bfProject, bfTaskID)
	if err != nil || len(timeline) != 3 {
		t.Fatalf("codegen_run = %d 行 err=%v, 期望 3", len(timeline), err)
	}
	// 内层迭代与外层修复可区分地可度量:每个 Run 的内层迭代恰好 1,外层
	// 尝试链长度 3,父链接成链,两个预算互不混淆(P2-11 AC2 纪律)。
	for _, run := range timeline {
		if run.Status != runstore.RunCompleted || run.Iterations != 1 {
			t.Fatalf("codegen_run %d = %s/迭代 %d, 期望 completed + 内层恰好 1 次迭代", run.ID, run.Status, run.Iterations)
		}
	}
	if timeline[1].EscalatedFrom != timeline[0].ID || timeline[2].EscalatedFrom != timeline[1].ID {
		t.Fatalf("Run 链 = %d←%d←%d, 期望父链接链", timeline[0].ID, timeline[1].ID, timeline[2].ID)
	}
	vruns, err := runs.VerificationRunsForTask(ctx, bfProject, bfTaskID)
	if err != nil || len(vruns) != 3 {
		t.Fatalf("verification_run = %d 行 err=%v, 期望 3", len(vruns), err)
	}
	for _, vr := range vruns {
		if vr.Status != runstore.VerificationFail {
			t.Fatalf("验证运行 %d = %s, 期望 fail(三个缺陷候选各自独立验证)", vr.ID, vr.Status)
		}
	}

	// AC4:恰好 1 次类型化人工介入事件(repair_exhausted = 人必须裁决/
	// 手工修复),非 hands-off;自动修复次数 = 2 可对账取代事实行数。
	led := bfLedger(res)
	if led.HandsOff || led.AutomaticRepairs != 2 ||
		!reflect.DeepEqual(led.InterventionEvents, []closedloop.EscalationReason{closedloop.EscalationRepairExhausted}) {
		t.Fatalf("事件账目 = %+v, 期望非 hands-off + 2 次自动修复 + 1 次 repair_exhausted 介入事件", led)
	}

	clAssertOwnerClean(t, repo)
	clAssertRootEmpty(t, cfg.WorktreeRoot)
}
