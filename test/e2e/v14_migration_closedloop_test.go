//go:build integration

// P3-04 — Database Migration closed-loop E2E(PRD §24.2)。
//
// 被测纵切 = 真实 worker v1.4 路径的测试等价组合(harness 见
// v14_migration_harness_test.go;P1-08 起组合与 worker 逐件对齐:同源
// bundle 装载 → CompileAuthority → LoopConfig.Authority 必达):迁移探针
// 作为冻结 binding 的 argv 命令真实执行 P3-02 夹具迁移链(fresh DB /
// N-1 seed 升级 / 数据完整性),生产闭环组合根(closedloop.NewLoop,含
// P1-06 验收守卫)驱动 FakeRuntime 脚本,事实全部落进真实 PostgreSQL
// (每测试独立 schema + 每执行独立一次性 database)。
//
// 场景:TASK-304 交付迁移修订 0003(FrozenIntent task_type=db_migration ⇒
// P1-02 权威风险输入确定性派生 schema_migration ⇒ 命中迁移束 ⇒ 必需保证档
// 上调 L3)。首个候选是缺陷内容(NOT NULL 无 DEFAULT):fresh binding 假置信
// 绿,N-1/完整性 binding 被 PostgreSQL 确定性拒绝 → 可信 FAIL → 修复
// (repairable);修复候选 = repair/0003(新候选 ChangeSet 内容,缺陷文件
// 不改写,决策锁 8/23)→ 同一冻结契约复验 → 三类证据齐备 → FinalVerified。
//
// 终态(P1-08 门面):修复候选虽然 FinalVerified,但其全部证据是 visible
// 确定性检查(achieved L1 < required L3)—— 编译权威验收守卫(DL-07/A-03)
// 在 accept 转移裁决:普通 Accept 被拒,终态 = assurance_insufficient 升级
// (fail-safe 到人/策略裁决;不是候选失败、不耗修复预算、不伪造 verdict)。
//
// 到任务验收的映射:
//
//	AC1 fresh 通过 ≠ 验收:缺陷候选只拿到 fresh_database 绿,验证 FAIL,
//	    bundle 门禁在真实证据上不放行(unmet = data_integrity +
//	    n_minus_1_upgrade);只有正确迁移后 fresh 路径才有意义。
//	AC2 N-1 升级被真实执行:缺陷候选确定性失败(23502),修复候选通过。
//	AC3 数据完整性被真实断言:seed 行身份/链路/时间戳逐字保留 +
//	    total_cents 全量回填 0。
//	AC4 修复不触碰验证权威:同一冻结契约逐字复验(两行 verification_run 的
//	    contract_hash 相同);修复候选内容 = repair 夹具字节;失败的证据
//	    append-only 保留;隐藏夹具(证据集)只读消费。
//	AC5 保证缺口同时被运行时与读模型如实暴露:运行时侧 = 验收守卫
//	    assurance_insufficient 升级(FinalVerified 不再单独授权 Accept);
//	    读模型侧 = taskstate.Derive(真实 FinalOutcome 覆盖报告 + 已编译
//	    合同)产出 VERIFIED_LIMITED —— required=L3(迁移风险上调,P1
//	    selector)、achieved=L1(确定性 visible 证据事实),两个面裁决一致,
//	    缺口不隐藏,matched_policies/satisfied 与 unmet 逐项可追溯。
package e2e

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tally/internal/assurance"
	"tally/internal/changeset"
	"tally/internal/closedloop"
	"tally/internal/engineeringpolicy"
	"tally/internal/evidence"
	"tally/internal/runstore"
	"tally/internal/taskstate"
	"tally/internal/verification"
)

// TestV14MigrationClosedLoopE2EDefectFailsThenRepairGuardEscalates 驱动完整
// 闭环:缺陷候选(假置信 fresh 绿 + 确定性 N-1 失败)→ repairable 修复 →
// 修复候选(repair 夹具内容)三类证据齐备 → FinalVerified,但 visible-only
// 证据达不成编译权威上调的 L3 必需档 → 验收守卫 assurance_insufficient 升级
// (绝不普通 Accepted);basis + 达成保证逐项暴露,运行时裁决与读模型投影
// 一致。被驱动的夹具:迁移链/seed/证据集/策略束全部来自
// test/fixtures/v14/migration/** 与 catalog/v14/policies(只读);
// FrozenIntent(db_migration)由 harness 夹具仓库冻结(P1-02 权威风险输入)。
func TestV14MigrationClosedLoopE2EDefectFailsThenRepairGuardEscalates(t *testing.T) {
	runs, st := clStore(t)
	base := migBaseDSN()
	if !e2eDBReachable(base) {
		t.Skipf("P3-04 迁移闭环 E2E 需要 PostgreSQL(%s 不可达),跳过", base)
	}
	probe := migBuildProbe(t)
	repo := migNewRepo(t, probe,
		migCreateDB(t, base, "cl_fresh"),
		migCreateDB(t, base, "cl_n1"),
		migCreateDB(t, base, "cl_integrity"),
		migFixturePath(t, "seed", "n_minus_1.sql"))
	cfg := migConfigFor(t, repo)

	defect := migReadFixture(t, "migrations", "0003_orders_total_not_null.sql")
	repair := migReadFixture(t, "repair", "0003_orders_total_not_null.sql")

	// 修复尝试的 worktree 里,冻结前候选(缺陷 0003)已被确定性物化;指令面
	// 断言在当下发生(worktree 随尝试结束销毁)。
	var sawMaterialized, repairInstruction string
	rt := newMigScriptedRuntime(
		migStep{files: map[string]string{migTaskPath: defect}},
		migStep{files: map[string]string{migTaskPath: repair}, inspect: func(ws, instruction string) {
			b, err := os.ReadFile(filepath.Join(ws, filepath.FromSlash(migTaskPath)))
			if err != nil {
				t.Errorf("修复基未物化: %v", err)
				return
			}
			sawMaterialized = string(b)
			repairInstruction = instruction
		}},
	)

	obs := migRunTask(t, context.Background(), st.Queries(), rt, cfg)
	res := obs.Result

	// ---- 终态:修复候选 FinalVerified 但被验收守卫裁决,assurance_insufficient
	// 升级(DL-07/A-03):绝不普通 Accepted,升级不携带 basis,修复预算只被
	// 真实验证 FAIL 消耗一次(守卫裁决在 Router 之前,不伪装候选失败)。------
	if res.Phase != closedloop.PhaseEscalated || res.Escalation == nil ||
		res.Escalation.Reason != closedloop.EscalationAssuranceInsufficient {
		t.Fatalf("终态 = %s / %+v, 期望 assurance_insufficient 升级(required L3 未被 visible-only 证据达成)", res.Phase, res.Escalation)
	}
	if res.Basis != nil {
		t.Fatalf("assurance_insufficient 升级不得携带 basis: %+v", res.Basis)
	}
	if want := "acceptance guard: assurance gap [AC-304 required L3 achieved L1]"; res.Escalation.Detail != want {
		t.Fatalf("守卫裁决明细 = %q, 期望 %q(缺口逐项可见,无 held-out 材料)", res.Escalation.Detail, want)
	}
	a1, a2 := res.Attempts[0], res.Attempts[1]
	if len(res.Attempts) != 2 || a2.ParentRun != a1.RunID || a2.Supersedes != a1.ChangeSet {
		t.Fatalf("尝试谱系 = %+v,期望父链接 + 取代链 A→B(守卫不耗修复预算:恰一次真实验证 FAIL 驱动的修复)", res.Attempts)
	}
	if a1.ChangeSet == "" || a1.ChangeSet == a2.ChangeSet {
		t.Fatalf("候选身份 = %s/%s,期望两个全新不同候选", a1.ChangeSet, a2.ChangeSet)
	}

	// ---- Contract-as-prevention:首次执行前实现权威已知最终验收条件 ---------
	// P1-08:编译权威的 requires 通知行(Prevention 小节)随每轮迭代进入
	// 实现可见指令(与 worker v14Instruction 同渲染,纳入 context_hash)。
	if len(obs.Instrs) != 2 {
		t.Fatalf("迭代指令 = %d 条,期望 2", len(obs.Instrs))
	}
	if !strings.Contains(obs.Instrs[0], "Prevention requirements (compiled contract)") {
		t.Errorf("首次指令缺编译权威 Prevention 小节: %q", obs.Instrs[0])
	}
	for _, want := range []string{migTaskID, "database_migration", "fresh_database", "n_minus_1_upgrade", "data_integrity"} {
		if !strings.Contains(obs.Instrs[0], want) {
			t.Errorf("首次指令缺 Prevention 材料 %q(policy requires 的执行前告知): %q", want, obs.Instrs[0])
		}
	}
	if strings.Contains(obs.Instrs[0], "Repair context") {
		t.Errorf("首次尝试不应携带修复反馈: %q", obs.Instrs[0])
	}
	// 修复反馈只含 allowlist 投影:失败类 + 可见证据 binding(无 held-out 面)。
	for _, want := range []string{"Repair context", "repairable", migAccID, "n_minus_1_upgrade", "data_integrity", "diagnostic summary"} {
		if !strings.Contains(repairInstruction, want) {
			t.Errorf("修复反馈缺 allowlist 材料 %q: %q", want, repairInstruction)
		}
	}
	// 修复继续基 = 冻结前候选内容逐字(ADR-0001 §2)。
	if sawMaterialized != defect {
		t.Fatalf("修复基内容与前一候选不一致(期望缺陷 0003 逐字物化)")
	}

	// ---- AC4:修复是新候选内容;验证权威逐字不变 ----------------------------
	css, err := runs.ChangeSetsForTask(context.Background(), migProject, migTaskID)
	if err != nil || len(css) != 2 {
		t.Fatalf("落库候选 = %d err=%v,期望 2", len(css), err)
	}
	for _, cs := range css {
		files := cs.Files()
		if len(files) != 1 || files[0].Path != migTaskPath {
			t.Fatalf("候选 %s 改动 = %+v,期望恰好 %s(修复不得触碰验证权威/其他文件)", cs.ID(), files, migTaskPath)
		}
		want := defect
		if cs.ID() == a2.ChangeSet {
			want = repair
		}
		if files[0].Content != want {
			t.Fatalf("候选 %s 内容与%s夹具字节不一致(修复候选必须是 repair 夹具内容)", cs.ID(), map[bool]string{true: "repair", false: "缺陷"}[cs.ID() == a2.ChangeSet])
		}
	}

	// ---- 真库事实:Run 链 + 取代事实 + 祖先谱系 -----------------------------
	ctx := context.Background()
	timeline, err := runs.CodegenRunsForTask(ctx, migProject, migTaskID)
	if err != nil || len(timeline) != 2 {
		t.Fatalf("codegen_run = %d 行 err=%v,期望 2", len(timeline), err)
	}
	first, second := timeline[0], timeline[1]
	if second.EscalatedFrom != first.ID || first.EscalatedFrom != 0 {
		t.Fatalf("Run 链父链接 = %d←%d,期望 attempt2 链接 attempt1", second.EscalatedFrom, first.ID)
	}
	if first.Status != runstore.RunCompleted || second.Status != runstore.RunCompleted {
		t.Fatalf("Run 终态 = %s/%s,期望 completed/completed", first.Status, second.Status)
	}
	facts, err := runs.ChangeSetSupersessions(ctx, a2.ChangeSet)
	if err != nil || len(facts) != 1 || facts[0].Old != a1.ChangeSet ||
		facts[0].New != a2.ChangeSet || facts[0].Reason != changeset.SupersedeRework {
		t.Fatalf("取代事实 = %+v err=%v,期望 a1→a2 rework", facts, err)
	}
	if anc, err := runs.ChangeSetAncestors(ctx, a2.ChangeSet); err != nil || len(anc) != 1 || anc[0] != a1.ChangeSet {
		t.Fatalf("ancestors(B) = %v err=%v,期望 [A]", anc, err)
	}

	// ---- 验证事实:两次独立验证,同一冻结契约(AC4 核心断言)----------------
	vruns, err := runs.VerificationRunsForTask(ctx, migProject, migTaskID)
	if err != nil || len(vruns) != 2 {
		t.Fatalf("verification_run = %d 行 err=%v,期望 2", len(vruns), err)
	}
	byCS := map[string]runstore.VerificationRun{}
	for _, vr := range vruns {
		byCS[string(vr.ChangeSetID)] = vr
	}
	if byCS[string(a1.ChangeSet)].Status != runstore.VerificationFail ||
		byCS[string(a2.ChangeSet)].Status != runstore.VerificationPass {
		t.Fatalf("验证结论 = %+v,期望 A fail / B pass", byCS)
	}
	for _, vr := range vruns {
		if vr.ContractHash != obs.Frozen.Hash() {
			t.Fatalf("验证运行 %d 契约 = %s,期望冻结契约逐字不变(修复不得改验证权威)", vr.ID, vr.ContractHash)
		}
	}
	runForCand := func(cs changeset.ChangeSetID) runstore.VerificationRun {
		t.Helper()
		vr, ok := byCS[string(cs)]
		if !ok {
			t.Fatalf("候选 %s 没有验证运行", cs)
		}
		return vr
	}
	defectRun, repairRun := runForCand(a1.ChangeSet), runForCand(a2.ChangeSet)

	// ---- 证据事实:fresh 假置信绿 + N-1/完整性红 → 修复后三绿 ----------------
	want1 := map[string]evidence.Verdict{
		"fresh_database":    evidence.VerdictPass,
		"n_minus_1_upgrade": evidence.VerdictFail,
		"data_integrity":    evidence.VerdictFail,
	}
	want2 := map[string]evidence.Verdict{
		"fresh_database":    evidence.VerdictPass,
		"n_minus_1_upgrade": evidence.VerdictPass,
		"data_integrity":    evidence.VerdictPass,
	}
	migAssertRunEvidence(t, runs, ctx, defectRun.ID, want1)
	migAssertRunEvidence(t, runs, ctx, repairRun.ID, want2)

	// AC1 的门禁面:缺陷候选的真实证据(fresh 单独绿)在 bundle 门禁上不放行;
	// 修复候选三类证据齐备才放行 —— gate_condition.all_of 与 evidence 键一一对应。
	preFul := migFulfillmentsFromEvidence(t, runs, ctx, defectRun.ID)
	pre, err := obs.Bundle.EvaluateGate(preFul)
	if err != nil {
		t.Fatalf("EvaluateGate(缺陷候选真实证据): %v", err)
	}
	if pre.Satisfied {
		t.Fatal("缺陷候选(fresh 单独绿)门禁不得放行(AC1)")
	}
	if want := []string{"data_integrity", "n_minus_1_upgrade"}; strings.Join(pre.UnmetKeys, ",") != strings.Join(want, ",") {
		t.Fatalf("缺陷候选未满足键 = %v,期望 %v", pre.UnmetKeys, want)
	}
	postFul := migFulfillmentsFromEvidence(t, runs, ctx, repairRun.ID)
	post, err := obs.Bundle.EvaluateGate(postFul)
	if err != nil {
		t.Fatalf("EvaluateGate(修复候选真实证据): %v", err)
	}
	if !post.Satisfied || len(post.UnmetKeys) != 0 {
		t.Fatalf("修复候选三类证据齐备必须放行: satisfied=%v unmet=%v", post.Satisfied, post.UnmetKeys)
	}

	// ---- 失败证据 append-only:修复不清除缺陷候选的失败事实 ----------------
	if n := clCountRows(t, st, "SELECT count(*) FROM verification_evidence WHERE verdict='fail'"); n != 2 {
		t.Fatalf("失败证据 = %d 行,期望 2(append-only;修复不抹历史)", n)
	}

	// ---- AC5:最终 basis 暴露达成保证(taskstate 纯派生投影)----------------
	migAssertFinalBasis(t, obs, a1, a2)

	// owner 主 worktree 零污染;一次性沙箱全部回收(INV-EXEC-01)。
	clAssertOwnerClean(t, repo)
	clAssertRootEmpty(t, cfg.WorktreeRoot)
	clAssertRootEmpty(t, cfg.VerifyWorktreeRoot)
}

// migAssertRunEvidence 断言一次验证运行的证据键 → verdict 精确映射。
func migAssertRunEvidence(t *testing.T, runs *runstore.Store, ctx context.Context, runID int64, want map[string]evidence.Verdict) {
	t.Helper()
	evs, err := runs.EvidenceForRun(ctx, migProject, runID)
	if err != nil {
		t.Fatalf("EvidenceForRun(%d): %v", runID, err)
	}
	got := map[string]evidence.Verdict{}
	for _, ev := range evs {
		if ev.VisibilityPolicy != verification.VisibilityVisible {
			t.Fatalf("证据 %s 可见性 = %s,期望 visible(迁移验证权威全部可见)", ev.BindingID, ev.VisibilityPolicy)
		}
		got[string(ev.BindingID)] = ev.Verdict
	}
	if len(got) != len(want) {
		t.Fatalf("验证运行 %d 证据 = %v,期望 %v", runID, got, want)
	}
	for key, verdict := range want {
		if got[key] != verdict {
			t.Fatalf("验证运行 %d 证据 %s = %s,期望 %s", runID, key, got[key], verdict)
		}
	}
}

// migFulfillmentsFromEvidence 从真库证据行编译门禁 fulfillments(binding id =
// bundle requires 键,verdict = 是否满足)。
func migFulfillmentsFromEvidence(t *testing.T, runs *runstore.Store, ctx context.Context, runID int64) []engineeringpolicy.RequirementFulfillment {
	t.Helper()
	evs, err := runs.EvidenceForRun(ctx, migProject, runID)
	if err != nil {
		t.Fatalf("EvidenceForRun(%d): %v", runID, err)
	}
	var out []engineeringpolicy.RequirementFulfillment
	for _, ev := range evs {
		out = append(out, engineeringpolicy.RequirementFulfillment{
			Key:       string(ev.BindingID),
			Satisfied: ev.Verdict == evidence.VerdictPass,
		})
	}
	return out
}

// migAssertFinalBasis 以真实 FinalOutcome(功能/独立性覆盖报告)+ 已编译合同 +
// append-only 生命周期事实驱动 taskstate.Derive,断言最终 basis 暴露:
//
//	required = L3(迁移风险经 P1 selector 上调);
//	achieved = L1(唯一执行事实 = 确定性 visible 检查;没有 held-out 事实,
//	绝不冒充更强保证 —— Agent/runtime 主张不是事实);
//	verdict = VERIFIED_LIMITED(缺口不隐藏:达成 < 必需的未豁免条款存在)。
//
// 读模型投影与运行时裁决同源一致:守卫在 accept 转移按同一编译权威拒绝普通
// Accept(assurance_insufficient,由测试主体终态断言钉住)—— 两个面不各说
// 各话,缺口不隐藏。
func migAssertFinalBasis(t *testing.T, obs migObservation, a1, a2 closedloop.Attempt) {
	t.Helper()
	if len(obs.Outcomes) != 2 {
		t.Fatalf("FinalOutcome = %d 次,期望 2(缺陷 fail + 修复 pass)", len(obs.Outcomes))
	}
	out1, out2 := obs.Outcomes[0], obs.Outcomes[1]
	if out1.Status != verification.FinalFailed || out2.Status != verification.FinalVerified {
		t.Fatalf("验证结论 = %s/%s,期望 failed/verified", out1.Status, out2.Status)
	}
	if out2.ChangeSetID != a2.ChangeSet || !out2.Functional.Complete ||
		out2.Functional.ContractHash != obs.Frozen.Hash() {
		t.Fatalf("修复候选覆盖报告 = %+v,期望绑定冻结契约且完整", out2.Functional)
	}

	rec := evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-e2e/p3-04"}
	at := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	run1, run2 := string(a1.RunID), string(a2.RunID)
	facts := []taskstate.Fact{
		{ID: "mig-f-planned", Kind: taskstate.FactTaskPlanned, At: at, Recorder: rec, RevisionRef: obs.Task.RevisionHash()},
		{ID: "mig-f-frozen", Kind: taskstate.FactVerificationFrozen, At: at, Recorder: rec, RevisionRef: obs.Frozen.Hash()},
		{ID: "mig-f-run1", Kind: taskstate.FactCodegenRunStarted, At: at, Recorder: rec, RunID: run1},
		{ID: "mig-f-cs1", Kind: taskstate.FactChangeSetProduced, At: at, Recorder: rec, RunID: run1, ChangeSet: evidence.ChangeSetID(a1.ChangeSet)},
		{ID: "mig-f-vr1", Kind: taskstate.FactVerificationRunStarted, At: at, Recorder: rec, RunID: run1, ChangeSet: evidence.ChangeSetID(a1.ChangeSet)},
		{ID: "mig-f-run2", Kind: taskstate.FactCodegenRunStarted, At: at, Recorder: rec, RunID: run2},
		{ID: "mig-f-cs2", Kind: taskstate.FactChangeSetProduced, At: at, Recorder: rec, RunID: run2, ChangeSet: evidence.ChangeSetID(a2.ChangeSet)},
		{ID: "mig-f-vr2", Kind: taskstate.FactVerificationRunStarted, At: at, Recorder: rec, RunID: run2, ChangeSet: evidence.ChangeSetID(a2.ChangeSet)},
	}
	derived, err := taskstate.Derive(taskstate.Snapshot{
		Contract:     obs.Frozen,
		Facts:        facts,
		Functional:   &out2.Functional,
		Independence: &out2.Independence,
		Compiled:     obs.Compiled,
	})
	if err != nil {
		t.Fatalf("taskstate.Derive: %v", err)
	}
	if derived.State != taskstate.StateVerified {
		t.Fatalf("taskstate = %s gaps=%+v,期望 VERIFIED(当前候选验证门全过)", derived.State, derived.Gaps)
	}
	basis := derived.VerifiedBasis
	if basis == nil {
		t.Fatal("Compiled 在场时必须产出 VerifiedBasis(AC5)")
	}
	if basis.AssuranceRequired != assurance.LevelL3 {
		t.Fatalf("必需保证 = %s,期望 L3(schema_migration 风险经 P1 selector 上调)", basis.AssuranceRequired)
	}
	if basis.AssuranceAchieved != assurance.LevelL1 {
		t.Fatalf("达成保证 = %s,期望 L1(唯一事实 = 确定性 visible 检查)", basis.AssuranceAchieved)
	}
	if basis.Verdict != taskstate.VerdictVerifiedLimited {
		t.Fatalf("verdict = %s,期望 VERIFIED_LIMITED(达成 < 必需的缺口不得隐藏)", basis.Verdict)
	}
	if len(basis.SatisfiedRequirements) != 0 || len(basis.WaivedRequirements) != 0 ||
		len(basis.ResidualRisk.UnmetClauses) != 1 || basis.ResidualRisk.UnmetClauses[0] != migAccID {
		t.Fatalf("satisfied/waived/unmet = %v/%v/%v,期望未被豁免的 %s 缺口如实可见(satisfied 与 waived 永不合并)",
			basis.SatisfiedRequirements, basis.WaivedRequirements, basis.ResidualRisk.UnmetClauses, migAccID)
	}
	if basis.EvidenceCoverage.Covered != 1 || basis.EvidenceCoverage.Total != 1 {
		t.Fatalf("证据覆盖 = %+v,期望 1/1", basis.EvidenceCoverage)
	}
	if len(basis.MatchedPolicies) != 1 || basis.MatchedPolicies[0].ID != "database_migration" {
		t.Fatalf("matched_policies = %+v,期望恰为 database_migration", basis.MatchedPolicies)
	}
	if len(basis.UnmatchedRiskSignals) != 0 {
		t.Fatalf("未匹配风险 = %v,期望空(schema_migration 由命中策略声明)", basis.UnmatchedRiskSignals)
	}
	if basis.CompiledHash != obs.Compiled.Hash() {
		t.Fatalf("basis 编译哈希 = %s,期望 %s(投影可单独复算)", basis.CompiledHash, obs.Compiled.Hash())
	}
}
