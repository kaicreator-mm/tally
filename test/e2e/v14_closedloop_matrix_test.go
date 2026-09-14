//go:build integration

// P2-10 — FakeRuntime 闭环 E2E 矩阵。被测纵切 = 真实 worker v1.4 路径的测试
// 等价组合(harness 见 v14_closedloop_harness_test.go):生产闭环组合根
// (closedloop.NewLoop:WorkspaceExecutor + ServiceVerifier + FailureRouter +
// FeedbackSource + RepairPolicy)驱动 FakeRuntime 的确定性脚本,事实全部落进
// 真实 PostgreSQL(每测试独立 schema、自跑 goose 迁移)。
//
// 矩阵到任务验收的映射:
//
//	AC1 首过 PASS → TestV14ClosedLoopE2EFirstPassAccepts
//	AC2 fail → repair → PASS → TestV14ClosedLoopE2EFailRepairsThenAccepts
//	AC3 修复耗尽 → TestV14ClosedLoopE2ERepairExhaustionEscalates
//	AC4 provider/environment 失败路由 →
//	    TestV14ClosedLoopE2EProviderFailureEscalates(provider 腿:会话错误
//	    绝不伪装成候选失败,直接 infrastructure 升级)
//	    TestV14ClosedLoopE2EEnvironmentErrorRoutedToReverify(environment 腿:
//	    验证 ERROR 走有界复验,FAIL≠ERROR —— 不算候选失败、不耗修复预算)
//	AC5 held-out 失败不泄漏 → TestV14ClosedLoopE2EHeldOutFailureEscalatesWithoutLeak
//	    (held-out FAIL → needs_review → requires_human_judgment 升级;实现
//	    可见通道零 held-out 材料)
//	AC6 supersession + exposure 祖先谱系 →
//	    TestV14ClosedLoopE2EExposureInRepairCompromisesLineage(修复在途蓄意
//	    暴露锚定前候选 → 复验 verified 也不得接受;取代事实与祖先谱系投影
//	    在真库逐项核对)
//	AC7 owner worktree 恒清洁 → 每个用例末尾 clAssertOwnerClean
//	(取消路径)→ TestV14ClosedLoopE2ECancellationDuringVerification
package e2e

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"tally/internal/candidateverify"
	"tally/internal/changeset"
	"tally/internal/closedloop"
	"tally/internal/evidence"
	"tally/internal/runstore"
	"tally/internal/store"
	"tally/internal/verification"
)

// ---- 验证注入缝(只在对应矩阵腿替换生产服务的单次行为)----------------------

// clFlakyEnvVerify 首次调用返回类型化 environment 失败(verification.ErrNoWorkspace
// 判因,无字符串嗅探),其后委托生产验证服务 —— 模拟验证环境瞬断。
type clFlakyEnvVerify struct {
	inner     closedloop.CandidateVerification
	failFirst int

	mu    sync.Mutex
	calls int
}

func (v *clFlakyEnvVerify) Verify(ctx context.Context, req candidateverify.Request) (*verification.FinalOutcome, error) {
	v.mu.Lock()
	v.calls++
	n := v.calls
	v.mu.Unlock()
	if n <= v.failFirst {
		return nil, fmt.Errorf("cl: 验证 workspace 丢失: %w", verification.ErrNoWorkspace)
	}
	return v.inner.Verify(ctx, req)
}

func (v *clFlakyEnvVerify) callCount() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.calls
}

// clCancelVerify 在生产验证服务被触达前取消执行 ctx:验证产出不构成 verdict。
type clCancelVerify struct {
	cancel context.CancelFunc

	mu    sync.Mutex
	calls int
}

func (v *clCancelVerify) Verify(ctx context.Context, _ candidateverify.Request) (*verification.FinalOutcome, error) {
	v.mu.Lock()
	v.calls++
	v.mu.Unlock()
	v.cancel()
	return nil, ctx.Err()
}

func (v *clCancelVerify) callCount() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.calls
}

// clCountRows 是 countRows 的 store 形态便捷封装(每测试独立 schema)。
func clCountRows(t *testing.T, st *store.Store, query string, args ...any) int64 {
	t.Helper()
	return countRows(t, st.Pool(), query, args...)
}

// ---- AC1:首过 PASS ----------------------------------------------------------

func TestV14ClosedLoopE2EFirstPassAccepts(t *testing.T) {
	runs, st := clStore(t)
	repo := clNewRepo(t)
	cfg := clConfigFor(t, repo)
	rt := newClScriptedRuntime(clStep{write: clContentGood()})

	obs := clRunTask(t, context.Background(), clDeps{cfg: cfg, queries: st.Queries(), runtime: rt}, clTaskID)
	res := obs.Result

	// 终态:Accepted 且必带 basis(每个 VERIFIED-like 结果携带依据)。
	if res.Phase != closedloop.PhaseAccepted || res.Basis == nil {
		t.Fatalf("终态 = %s basis=%v, 期望 accepted+basis", res.Phase, res.Basis)
	}
	if res.Basis.Attempt != 1 || res.Escalation != nil {
		t.Fatalf("Basis/Escalation = %+v/%+v, 期望尝试 1、无升级", res.Basis, res.Escalation)
	}
	if len(res.Attempts) != 1 || res.Attempts[0].RunID != closedloop.RunIDFor(clTaskID, 1) {
		t.Fatalf("尝试谱系 = %+v, 期望恰好一次首过尝试", res.Attempts)
	}

	// 真库事实:1 个 completed codegen_run(fake 归因)→ 1 个不可变候选 →
	// 1 次 pass 验证精确绑定候选 → 1 条 visible pass 证据。
	ctx := context.Background()
	timeline, err := runs.CodegenRunsForTask(ctx, clProject, clTaskID)
	if err != nil || len(timeline) != 1 {
		t.Fatalf("codegen_run = %d 行 err=%v, 期望 1", len(timeline), err)
	}
	run := timeline[0]
	if run.Status != runstore.RunCompleted || run.ResultPatchHash == "" ||
		run.Runtime != "fake" || run.WorkspaceID == "" {
		t.Fatalf("codegen_run = %+v, 期望 completed + 候选哈希 + fake 归因 + 沙箱坐标", run)
	}
	if run.ResultPatchHash != string(res.Basis.ChangeSet) {
		t.Fatalf("Run 结果 = %s, 期望候选 %s", run.ResultPatchHash, res.Basis.ChangeSet)
	}
	css, err := runs.ChangeSetsForTask(ctx, clProject, clTaskID)
	if err != nil || len(css) != 1 || css[0].ID() != res.Basis.ChangeSet {
		t.Fatalf("落库候选 = %+v err=%v, 期望恰好 basis 候选", css, err)
	}
	vruns, err := runs.VerificationRunsForTask(ctx, clProject, clTaskID)
	if err != nil || len(vruns) != 1 || vruns[0].Status != runstore.VerificationPass {
		t.Fatalf("verification_run = %+v err=%v, 期望恰好 1 次 pass", vruns, err)
	}
	if vruns[0].ChangeSetID != res.Basis.ChangeSet ||
		vruns[0].ContractHash != obs.Frozen.Hash() {
		t.Fatalf("verification_run 绑定 = %+v, 期望精确绑定候选与冻结契约", vruns[0])
	}
	if res.Basis.VerifyRun != strconv.FormatInt(vruns[0].ID, 10) {
		t.Fatalf("Basis.VerifyRun = %s, 期望真库验证运行 %d", res.Basis.VerifyRun, vruns[0].ID)
	}
	evs, err := runs.EvidenceForRun(ctx, clProject, vruns[0].ID)
	if err != nil || len(evs) != 1 || evs[0].Verdict != evidence.VerdictPass || evs[0].BindingID != clVisID {
		t.Fatalf("evidence = %+v err=%v, 期望 VIS-1 pass", evs, err)
	}

	// 零取代、零暴露、零祖先:首过不产生谱系事实。
	if n := clCountRows(t, st, "SELECT count(*) FROM changeset_superseded_fact"); n != 0 {
		t.Fatalf("取代事实 = %d 行, 期望 0(AC1)", n)
	}
	if n := clCountRows(t, st, "SELECT count(*) FROM evidence_exposure"); n != 0 {
		t.Fatalf("暴露事实 = %d 行, 期望 0(AC1)", n)
	}
	if anc, err := runs.ChangeSetAncestors(ctx, res.Basis.ChangeSet); err != nil || len(anc) != 0 {
		t.Fatalf("ancestors = %v err=%v, 期望空(首过无谱系)", anc, err)
	}

	// AC7:owner 主 worktree 零污染;一次性沙箱全部回收。
	clAssertOwnerClean(t, repo)
	clAssertRootEmpty(t, cfg.WorktreeRoot)
}

// ---- AC2:fail → repairable → 修复 → PASS ------------------------------------

func TestV14ClosedLoopE2EFailRepairsThenAccepts(t *testing.T) {
	runs, st := clStore(t)
	repo := clNewRepo(t)
	cfg := clConfigFor(t, repo)

	// 修复尝试的 worktree 里,冻结前候选已被确定性物化(写新内容前观察)。
	var sawMaterialized string
	rt := newClScriptedRuntime(
		clStep{write: clContentBad()},
		clStep{write: clContentGood(), inspect: func(ws, instruction string) {
			b, err := os.ReadFile(filepath.Join(ws, filepath.FromSlash(clGreetingRel)))
			if err != nil {
				t.Errorf("修复基未物化: %v", err)
				return
			}
			sawMaterialized = string(b)
			if !strings.Contains(instruction, "Repair context") {
				t.Errorf("修复迭代指令缺安全反馈段: %q", instruction)
			}
		}},
	)

	obs := clRunTask(t, context.Background(), clDeps{cfg: cfg, queries: st.Queries(), runtime: rt}, clTaskID)
	res := obs.Result

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
	// 修复基物化 = 冻结前候选内容逐字。
	if sawMaterialized != clContentBad() {
		t.Fatalf("修复基内容 = %q, 期望前候选内容逐字一致", sawMaterialized)
	}

	// 安全反馈边界:首次尝试无失败材料通道;修复反馈只含 allowlist 投影。
	instrs := obs.Instrs
	if len(instrs) != 2 {
		t.Fatalf("迭代指令 = %d 条, 期望 2", len(instrs))
	}
	if strings.Contains(instrs[0], "Repair context") {
		t.Errorf("首次尝试不应携带修复反馈: %q", instrs[0])
	}
	for _, want := range []string{"AC-01", "widget greeting delivers the frozen behavior.", "repairable", "VIS-1", "diagnostic summary"} {
		if !strings.Contains(instrs[1], want) {
			t.Errorf("修复反馈缺 allowlist 材料 %q: %q", want, instrs[1])
		}
	}

	// 真库事实:取代事实 reason=rework;Run 链父链接;两次独立验证(fail→pass)
	// 各自精确绑定候选。
	ctx := context.Background()
	facts, err := runs.ChangeSetSupersessions(ctx, a2.ChangeSet)
	if err != nil || len(facts) != 1 || facts[0].Old != a1.ChangeSet ||
		facts[0].New != a2.ChangeSet || facts[0].Reason != changeset.SupersedeRework {
		t.Fatalf("取代事实 = %+v err=%v, 期望 a1→a2 rework", facts, err)
	}
	if n := clCountRows(t, st, "SELECT count(*) FROM changeset_superseded_fact"); n != 1 {
		t.Fatalf("取代事实行数 = %d, 期望 1(AC2/AC6)", n)
	}
	timeline, err := runs.CodegenRunsForTask(ctx, clProject, clTaskID)
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
	// 取代祖先谱系:② 的祖先恰是 ①。
	if anc, err := runs.ChangeSetAncestors(ctx, a2.ChangeSet); err != nil || len(anc) != 1 || anc[0] != a1.ChangeSet {
		t.Fatalf("ancestors(B) = %v err=%v, 期望 [A](AC6)", anc, err)
	}
	vruns, err := runs.VerificationRunsForTask(ctx, clProject, clTaskID)
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
	if res.Basis.VerifyRun == "" {
		t.Fatal("accepted 必须携带验证运行引用(basis)")
	}
	// 复验绝不放宽:两次验证同一冻结契约。
	for _, vr := range vruns {
		if vr.ContractHash != obs.Frozen.Hash() {
			t.Fatalf("验证运行 %d 契约 = %s, 期望冻结契约逐字不变", vr.ID, vr.ContractHash)
		}
	}

	clAssertOwnerClean(t, repo)
	clAssertRootEmpty(t, cfg.WorktreeRoot)
}

// ---- AC3:修复耗尽 → 显式升级 -------------------------------------------------

func TestV14ClosedLoopE2ERepairExhaustionEscalates(t *testing.T) {
	runs, st := clStore(t)
	repo := clNewRepo(t)
	cfg := clConfigFor(t, repo)
	// 缺省修复预算 2(决策锁 23):首试 + 2 次修复全部 FAIL → 第 3 次修复被拒。
	rt := newClScriptedRuntime(
		clStep{write: clContentBad()},
		clStep{write: clContentBad()},
		clStep{write: clContentBad()},
	)

	obs := clRunTask(t, context.Background(), clDeps{cfg: cfg, queries: st.Queries(), runtime: rt}, clTaskID)
	res := obs.Result

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
	// 谱系链:②取代①,③取代②。
	a1, a2, a3 := res.Attempts[0], res.Attempts[1], res.Attempts[2]
	if a2.Supersedes != a1.ChangeSet || a3.Supersedes != a2.ChangeSet {
		t.Fatalf("取代链 = %+v, 期望 A→B→C", res.Attempts)
	}

	ctx := context.Background()
	if n := clCountRows(t, st, "SELECT count(*) FROM changeset_superseded_fact"); n != 2 {
		t.Fatalf("取代事实 = %d 行, 期望 2(AC3)", n)
	}
	if anc, err := runs.ChangeSetAncestors(ctx, a3.ChangeSet); err != nil ||
		len(anc) != 2 || anc[0] != a2.ChangeSet || anc[1] != a1.ChangeSet {
		t.Fatalf("ancestors(C) = %v err=%v, 期望 [B, A](AC6 的多代谱系)", anc, err)
	}
	if n := clCountRows(t, st, "SELECT count(*) FROM codegen_run WHERE project_id=$1", string(clProject)); n != 3 {
		t.Fatalf("codegen_run = %d 行, 期望 3", n)
	}
	// 三次候选各自得到独立 FAIL 验证;绝无静默接受。
	vruns, err := runs.VerificationRunsForTask(ctx, clProject, clTaskID)
	if err != nil || len(vruns) != 3 {
		t.Fatalf("verification_run = %d 行 err=%v, 期望 3", len(vruns), err)
	}
	for _, vr := range vruns {
		if vr.Status != runstore.VerificationFail {
			t.Fatalf("验证运行 %d = %s, 期望 fail", vr.ID, vr.Status)
		}
	}

	clAssertOwnerClean(t, repo)
	clAssertRootEmpty(t, cfg.WorktreeRoot)
}

// ---- AC4(provider 腿):runtime 会话失败 → infrastructure 升级 ---------------

func TestV14ClosedLoopE2EProviderFailureEscalates(t *testing.T) {
	runs, st := clStore(t)
	repo := clNewRepo(t)
	cfg := clConfigFor(t, repo)
	rt := newClScriptedRuntime(clStep{err: errors.New("cl: provider 进程崩溃")})

	obs := clRunTask(t, context.Background(), clDeps{cfg: cfg, queries: st.Queries(), runtime: rt}, clTaskID)
	res := obs.Result

	// provider 失败绝不伪装成候选失败:infrastructure 升级,不入谱系、不触发
	// 验证、不产生候选事实(ADR-0001 §3)。
	if res.Phase != closedloop.PhaseEscalated || res.Escalation == nil ||
		res.Escalation.Reason != closedloop.EscalationInfrastructure {
		t.Fatalf("终态 = %s / %+v, 期望 infrastructure_failure 升级", res.Phase, res.Escalation)
	}
	if len(res.Attempts) != 0 || res.Basis != nil {
		t.Fatalf("尝试/basis = %+v/%+v, 期望失败尝试不入谱系", res.Attempts, res.Basis)
	}
	if n := clCountRows(t, st, "SELECT count(*) FROM changeset WHERE project_id=$1", string(clProject)); n != 0 {
		t.Fatalf("候选 = %d 行, 期望 0(AC4)", n)
	}
	if n := clCountRows(t, st, "SELECT count(*) FROM verification_run WHERE project_id=$1", string(clProject)); n != 0 {
		t.Fatalf("验证运行 = %d 行, 期望 0(AC4)", n)
	}
	if n := clCountRows(t, st, "SELECT count(*) FROM changeset_superseded_fact"); n != 0 {
		t.Fatalf("取代事实 = %d 行, 期望 0(AC4)", n)
	}
	timeline, err := runs.CodegenRunsForTask(context.Background(), clProject, clTaskID)
	if err != nil || len(timeline) != 1 || timeline[0].Status != runstore.RunFailed ||
		timeline[0].FailureReason != "runtime_error" {
		t.Fatalf("codegen_run = %+v err=%v, 期望 failed/runtime_error", timeline, err)
	}

	clAssertOwnerClean(t, repo)
	clAssertRootEmpty(t, cfg.WorktreeRoot)
}

// ---- AC4(environment 腿):验证 ERROR → 有界复验,绝不成为候选 FAIL ----------

func TestV14ClosedLoopE2EEnvironmentErrorRoutedToReverify(t *testing.T) {
	runs, st := clStore(t)
	repo := clNewRepo(t)
	cfg := clConfigFor(t, repo)
	rt := newClScriptedRuntime(clStep{write: clContentGood()})
	flaky := &clFlakyEnvVerify{failFirst: 1}

	obs := clRunTask(t, context.Background(), clDeps{
		cfg: cfg, queries: st.Queries(), runtime: rt,
		verifyWrap: func(inner closedloop.CandidateVerification) closedloop.CandidateVerification {
			flaky.inner = inner
			return flaky
		},
	}, clTaskID)
	res := obs.Result

	// FAIL≠ERROR 端到端(ADR-0001 §3):environment ERROR 不算候选失败 ——
	// 同一候选获得有界复验(默认 2 次内)并正常接受;取代事实为零、修复预算
	// 未被消耗(谱系恰好 1 次尝试)。
	if res.Phase != closedloop.PhaseAccepted || res.Basis == nil || res.Basis.Attempt != 1 {
		t.Fatalf("终态 = %s basis=%+v, 期望 accepted@attempt1(环境错误后复验)", res.Phase, res.Basis)
	}
	if got := flaky.callCount(); got != 2 {
		t.Fatalf("验证服务调用 = %d, 期望 2(ERROR 复验恰好一次)", got)
	}
	if len(res.Attempts) != 1 {
		t.Fatalf("尝试 = %d, 期望 1(环境错误绝不消耗修复预算)", len(res.Attempts))
	}
	ctx := context.Background()
	if n := clCountRows(t, st, "SELECT count(*) FROM changeset_superseded_fact"); n != 0 {
		t.Fatalf("取代事实 = %d 行, 期望 0(环境错误不进修复)", n)
	}
	// ERROR 未产生任何失败事实:唯一验证运行是复验的 pass,绑定原候选。
	vruns, err := runs.VerificationRunsForTask(ctx, clProject, clTaskID)
	if err != nil || len(vruns) != 1 || vruns[0].Status != runstore.VerificationPass {
		t.Fatalf("verification_run = %+v err=%v, 期望恰好 1 次 pass(复验)", vruns, err)
	}
	if vruns[0].ChangeSetID != res.Basis.ChangeSet {
		t.Fatalf("复验绑定 = %s, 期望原候选 %s", vruns[0].ChangeSetID, res.Basis.ChangeSet)
	}
	if evs, err := runs.EvidenceForRun(ctx, clProject, vruns[0].ID); err != nil || len(evs) != 1 ||
		evs[0].Verdict != evidence.VerdictPass {
		t.Fatalf("evidence = %+v err=%v, 期望恰 1 条 pass(无 ERROR 伪装的失败事实)", evs, err)
	}

	clAssertOwnerClean(t, repo)
	clAssertRootEmpty(t, cfg.WorktreeRoot)
}

// ---- AC5:held-out 失败 → 人裁决升级,实现可见通道零泄漏 ----------------------

func TestV14ClosedLoopE2EHeldOutFailureEscalatesWithoutLeak(t *testing.T) {
	runs, st := clStore(t)
	// held-out 内容不含验收 token:visible(go version)过、held-out(grep
	// token)确定性 FAIL —— 阻断验收并推向 needs_review(§8.8.1)。
	repo, holdoutRoot := clNewHeldRepo(t, false)
	cfg := clConfigFor(t, repo)
	cfg.HoldoutRoot = holdoutRoot
	rt := newClScriptedRuntime(clStep{write: clContentGood()})

	obs := clRunTask(t, context.Background(), clDeps{cfg: cfg, queries: st.Queries(), runtime: rt}, clTaskID)
	res := obs.Result

	// held-out FAIL 不是候选缺陷而是独立性/人裁决材料:requires_human_judgment
	// 升级(绝不进修复 —— 修复通道结构上不可能为它构造安全反馈主体)。
	if res.Phase != closedloop.PhaseEscalated || res.Escalation == nil ||
		res.Escalation.Reason != closedloop.EscalationReason(closedloop.FailureRequiresHuman) {
		t.Fatalf("终态 = %s / %+v, 期望 requires_human_judgment 升级", res.Phase, res.Escalation)
	}
	if res.Basis != nil {
		t.Fatalf("升级不得携带 basis: %+v", res.Basis)
	}
	if len(res.Attempts) != 1 {
		t.Fatalf("尝试 = %d, 期望 1(无修复)", len(res.Attempts))
	}
	if n := clCountRows(t, st, "SELECT count(*) FROM changeset_superseded_fact"); n != 0 {
		t.Fatalf("取代事实 = %d 行, 期望 0(AC5)", n)
	}

	// 无隐藏泄漏:实现可见通道(编码指令)恰好 1 条,密标/store 根/不透明
	// ref/内容摘要零出现(AC5;INV-V14-07)。
	instrs := obs.Instrs
	if len(instrs) != 1 {
		t.Fatalf("编码指令 = %d 条, 期望 1(held-out 失败不得触发修复 Run)", len(instrs))
	}
	content := []byte("P2-10 controlled held-out expectation; marker " + clHoldoutSecret + "\n")
	for _, s := range clHoldoutLeakSurfaces(t, holdoutRoot, content) {
		for i, in := range instrs {
			if strings.Contains(in, s) {
				t.Fatalf("实现可见通道泄漏 %q(指令 %d): %q", s, i, in)
			}
		}
	}

	// 验证权威侧:held-out FAIL 如实落库(held_out 可见性),结论 needs_review
	// 映射为 error 状态的验证运行 —— 绝不 pass。
	ctx := context.Background()
	if n := clCountRows(t, st, "SELECT count(*) FROM verification_evidence WHERE visibility_policy='held_out' AND verdict='fail'"); n != 1 {
		t.Fatalf("held_out fail 证据 = %d 行, 期望 1(AC5)", n)
	}
	vruns, err := runs.VerificationRunsForTask(ctx, clProject, clTaskID)
	if err != nil || len(vruns) != 1 || vruns[0].Status != runstore.VerificationError {
		t.Fatalf("verification_run = %+v err=%v, 期望 1 次 error(needs_review 映射)", vruns, err)
	}

	clAssertOwnerClean(t, repo)
	clAssertRootEmpty(t, cfg.WorktreeRoot)
}

// ---- AC6:修复在途暴露 → 谱系封顶;supersession + 祖先谱系落真库 --------------

func TestV14ClosedLoopE2EExposureInRepairCompromisesLineage(t *testing.T) {
	runs, st := clStore(t)
	repo := clNewRepo(t)
	cfg := clConfigFor(t, repo)
	rt := newClScriptedRuntime(
		clStep{write: clContentBad()},
		clStep{write: clContentGood()},
	)

	obs := clRunTask(t, context.Background(), clDeps{
		cfg: cfg, queries: st.Queries(), runtime: rt,
		// 修复在途的蓄意诊断暴露(P2-06 的物理位置):锚定冻结前候选,
		// 经共享账面 → runstore append-only 落 evidence_exposure。
		ExposeRepairReason: "repair diagnostic surfaced held-out assertion internals",
	}, clTaskID)
	res := obs.Result
	a1, a2 := res.Attempts[0], res.Attempts[1]

	// 修复候选即使 verified 也不得接受:谱系封顶 → exposure_compromise 升级,
	// basis 恒空(INV-V14-08:assurance 只能保持或下降,后续「干净」不解封)。
	if res.Phase != closedloop.PhaseEscalated || res.Escalation == nil ||
		res.Escalation.Reason != closedloop.EscalationExposureCompromise {
		t.Fatalf("终态 = %s / %+v, 期望 exposure_compromise 升级", res.Phase, res.Escalation)
	}
	if res.Basis != nil {
		t.Fatalf("compromised 谱系不得产生 basis: %+v", res.Basis)
	}
	if len(res.Attempts) != 2 || a2.Supersedes != a1.ChangeSet || a2.ParentRun != a1.RunID {
		t.Fatalf("尝试谱系 = %+v, 期望完整取代链 A→B", res.Attempts)
	}

	ctx := context.Background()
	// 暴露事实成为真库 append-only 行:channel/锚点各就各位。
	if n := clCountRows(t, st, "SELECT count(*) FROM evidence_exposure"); n != 1 {
		t.Fatalf("evidence_exposure = %d 行, 期望 1(AC6)", n)
	}
	exps, err := runs.ExposuresForTask(ctx, clProject, clTaskID)
	if err != nil || len(exps) != 1 {
		t.Fatalf("ExposuresForTask = %+v err=%v, 期望 1 条", exps, err)
	}
	if exps[0].Channel() != evidence.ChannelDebugDisclosure {
		t.Fatalf("channel = %q, 期望 debug_disclosure", exps[0].Channel())
	}
	if anchor, ok := exps[0].ChangeSet(); !ok || anchor != evidence.ChangeSetID(a1.ChangeSet) {
		t.Fatalf("暴露锚点 = %v(ok=%v), 期望前候选 A", anchor, ok)
	}
	// supersession 落真库;祖先谱系可确定性展开(取代不能掩盖历史)。
	facts, err := runs.ChangeSetSupersessions(ctx, a2.ChangeSet)
	if err != nil || len(facts) != 1 || facts[0].Old != a1.ChangeSet || facts[0].Reason != changeset.SupersedeRework {
		t.Fatalf("取代事实 = %+v err=%v, 期望 a1→a2 rework", facts, err)
	}
	if anc, err := runs.ChangeSetAncestors(ctx, a2.ChangeSet); err != nil || len(anc) != 1 || anc[0] != a1.ChangeSet {
		t.Fatalf("ancestors(B) = %v err=%v, 期望 [A](AC6)", anc, err)
	}
	// 暴露沿取代谱系投影:B 自身干净(局部空),祖先 A 的暴露 depth 1 仍命中。
	if local, err := runs.ExposuresForChangeSet(ctx, clProject, a2.ChangeSet); err != nil || len(local) != 0 {
		t.Fatalf("B 局部暴露 = %+v err=%v, 期望空(自身干净)", local, err)
	}
	lineage, err := runs.ExposuresForChangeSetLineage(ctx, clProject, a2.ChangeSet)
	if err != nil || len(lineage) != 1 || lineage[0].ChangeSet != changeset.ChangeSetID(a1.ChangeSet) ||
		lineage[0].AncestorDepth != 1 {
		t.Fatalf("lineage(B) = %+v err=%v, 期望祖先 A 暴露 depth 1(AC6)", lineage, err)
	}
	// 修复候选的复验确实产出 verified(封顶来自暴露,不是验证失败)。
	vruns, err := runs.VerificationRunsForChangeSet(ctx, clProject, a2.ChangeSet)
	if err != nil || len(vruns) != 1 || vruns[0].Status != runstore.VerificationPass {
		t.Fatalf("B 的验证 = %+v err=%v, 期望 pass(封顶来自暴露账面)", vruns, err)
	}

	clAssertOwnerClean(t, repo)
	clAssertRootEmpty(t, cfg.WorktreeRoot)
}

// ---- 取消路径:验证中取消 → Cancelled 终态,不伪造 verdict --------------------

func TestV14ClosedLoopE2ECancellationDuringVerification(t *testing.T) {
	runs, st := clStore(t)
	repo := clNewRepo(t)
	cfg := clConfigFor(t, repo)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	rt := newClScriptedRuntime(clStep{write: clContentGood()})
	cv := &clCancelVerify{cancel: cancel}

	res := clRunTask(t, ctx, clDeps{
		cfg: cfg, queries: st.Queries(), runtime: rt,
		verifyWrap: func(closedloop.CandidateVerification) closedloop.CandidateVerification { return cv },
	}, clTaskID).Result

	if res.Phase != closedloop.PhaseCancelled {
		t.Fatalf("终态 = %s, 期望 cancelled", res.Phase)
	}
	if res.Basis != nil || res.Escalation != nil {
		t.Fatalf("取消不得产生 basis/升级: %+v / %+v", res.Basis, res.Escalation)
	}
	// 已完成尝试保留为事实;取消的验证产出不构成 verdict,也不复验。
	if len(res.Attempts) != 1 {
		t.Fatalf("已完成尝试 = %d, 期望保留 1", len(res.Attempts))
	}
	if got := cv.callCount(); got != 1 {
		t.Fatalf("验证调用 = %d, 期望 1(取消后不得复验)", got)
	}
	if n := clCountRows(t, st, "SELECT count(*) FROM verification_run WHERE project_id=$1", string(clProject)); n != 0 {
		t.Fatalf("验证运行 = %d 行, 期望 0(取消不产生 verdict)", n)
	}
	if n := clCountRows(t, st, "SELECT count(*) FROM changeset WHERE project_id=$1", string(clProject)); n != 1 {
		t.Fatalf("候选 = %d 行, 期望 1(已完成尝试的候选是事实)", n)
	}
	if timeline, err := runs.CodegenRunsForTask(context.Background(), clProject, clTaskID); err != nil ||
		len(timeline) != 1 || timeline[0].Status != runstore.RunCompleted {
		t.Fatalf("codegen_run = %+v err=%v, 期望 1 次 completed(取消前已完成)", timeline, err)
	}

	clAssertOwnerClean(t, repo)
	clAssertRootEmpty(t, cfg.WorktreeRoot)
}
