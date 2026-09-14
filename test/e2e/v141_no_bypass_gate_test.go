//go:build integration

// P1-08 — Compiled-authority no-bypass gate(E2E 证据面)。
//
// 被测纵切 = 测试等价组合 clRunTask(v14_closedloop_harness_test.go;P1-08 起
// 与 cmd/worker runV14Task 逐件对齐:同源 bundle 装载 → CompileAuthority →
// Compiled.Requires() 进 ContextPack/指令 → LoopConfig.Authority 必达)。
// 本文件把 P1 权威链的四个 gate 断言钉在真实 Git 仓库 + 真实 PostgreSQL 的
// 闭环执行上(不是 isolated unit-only 检查):
//
//	gate-1 同源不变量:编译权威与 bundle 身份从唯一 pinned 提交确定性可复算
//	       (DL-04);低风险对照在其守卫下合法接受(门不放走、也不误伤)。
//	gate-2 L3-required/L1-achieved 被阻断:命中策略上调必需档后,仅 visible
//	       确定性证据(achieved L1)的 FinalVerified 候选在 accept 转移被
//	       验收守卫裁决 assurance_insufficient(DL-07/A-03)—— 绝不 Accepted、
//	       绝不伪装候选失败、绝不耗修复预算;Prevention 小节(执行前 requires)
//	       已随迭代指令到达实现权威(P1-05 对齐)。
//	gate-3 unknown/unmatched 风险 fail-safe:未被任何策略声明的信号逐字保真
//	       (INV-V14-09),必需档 fail-safe 上调,verified 候选照样不普通接受。
//	gate-4 held-out 内容不泄漏:编译权威接线(bundle/intent/requires/指令)
//	       不给实现可见通道引入任何 held-out 材料(INV-V14-07)。
//	gate-5 蓄意 bypass 夹具在接受之前失败:拒收策略 / 缺 FrozenIntent 的仓库
//	       在执行前置被类型化拒绝(零 runtime 会话、零候选事实);混合来源的
//	       手工 bundle 被 CompileAuthority 的来源复核拒绝(ErrSourceMismatch,
//	       DL-04 防御第二道)。
//
// 数据库不可达时 SKIP(不产出假失败)。
package e2e

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tally/internal/artifacts"
	"tally/internal/assurance"
	"tally/internal/closedloop"
	"tally/internal/evidence"
	"tally/internal/risk"
	"tally/internal/runstore"
)

// gateMigrationPolicyYAML 是 gate-2 场景的命中策略(与 arch 守卫夹具同形):
// 命中 schema_migration 信号的 gate 策略,requires = fresh_database(执行前
// Prevention 通告 + 执行后门禁,PRD §8.3 双重用途)。
const gateMigrationPolicyYAML = `
id: policy-migration-gate
enforcement: gate
rationale: migration-class changes must prove upgrade path on a fresh database
trigger:
  risk_signals:
    - schema_migration
requires:
  - fresh_database
gate_condition:
  all_of:
    - fresh_database
`

// gateUnknownSignal 是 gate-3 场景的 owner 确认风险提示原文:不在冻结信号
// 枚举内(risk.Derive 原样保真进 Unknown),且无任何策略声明它 ⇒ unmatched。
const gateUnknownSignal = "unknown_governance_signal"

// ---- gate-1:同源不变量 + 低风险对照合法接受 ---------------------------------

// TestV141GateAuthorityReproducibleFromOneCommit 从一次 clRunTask 的执行权威
// 出发,在同一 pinned HEAD 上独立重载 + 重编译,断言 bundle 身份与编译哈希
// 确定性复现、来源提交唯一(DL-04);低风险 bug_fix 对照(零策略、零信号)
// 的必需档保持 L1 地板并在守卫下合法接受 —— 门是真实的,但不误伤。
func TestV141GateAuthorityReproducibleFromOneCommit(t *testing.T) {
	_, st := clStore(t)
	repo := clNewRepo(t)
	cfg := clConfigFor(t, repo)
	rt := newClScriptedRuntime(clStep{write: clContentGood()})

	obs := clRunTask(t, context.Background(), clDeps{cfg: cfg, queries: st.Queries(), runtime: rt}, clTaskID)
	res := obs.Result

	// 对照终态:required L1 全达成 → 普通 Accepted(守卫放行合法验收)。
	if res.Phase != closedloop.PhaseAccepted || res.Basis == nil {
		t.Fatalf("终态 = %s basis=%+v, 期望低风险对照 accepted(guard-allowed)", res.Phase, res.Basis)
	}
	if obs.Authority == nil || obs.Authority.Compiled == nil {
		t.Fatal("clRunTask 必须暴露编译权威(P1-03 生产缝,无旁路)")
	}
	for _, cl := range obs.Authority.Compiled.Clauses() {
		if cl.Assurance.Required != assurance.LevelL1 {
			t.Fatalf("条款 %s required = %s, 期望零信号零策略保持 L1 地板(锁 21)", cl.AcceptanceID, cl.Assurance.Required)
		}
	}

	// 同源复算:同一 pinned HEAD 独立 LoadBundle + CompileAuthority 产出同一
	// bundle 身份与编译哈希;来源提交 = 观测到的 HEAD(一个提交,DL-04)。
	head := clGit(t, repo, "rev-parse", "HEAD")
	if obs.Authority.SourceCommit != head || obs.Bundle.SourceCommit != head {
		t.Fatalf("权威来源 = %s / bundle 来源 = %s, 期望观测 HEAD %s", obs.Authority.SourceCommit, obs.Bundle.SourceCommit, head)
	}
	reloaded, err := artifacts.LoadBundle(repo, "HEAD", clTaskID)
	if err != nil {
		t.Fatalf("重载 bundle: %v", err)
	}
	recompiled, err := reloaded.CompileAuthority()
	if err != nil {
		t.Fatalf("重编译权威: %v", err)
	}
	if reloaded.Identity() != obs.Bundle.Identity() || recompiled.Compiled.Hash() != obs.Authority.Compiled.Hash() {
		t.Fatalf("同源复算不一致: bundle %s vs %s, compiled %s vs %s(DL-04:权威是 pinned 提交的确定性纯函数)",
			reloaded.Identity(), obs.Bundle.Identity(), recompiled.Compiled.Hash(), obs.Authority.Compiled.Hash())
	}

	clAssertOwnerClean(t, repo)
	clAssertRootEmpty(t, cfg.WorktreeRoot)
}

// ---- gate-2:L3-required / L1-achieved 被阻断 + Prevention 到达运行时 --------

// TestV141GateRequiredL3AchievedL1BlockedE2E 命中策略(schema_migration 触发
// 的 gate 束)把必需档上调 L3;脚本的候选通过 visible 验收(achieved L1)——
// FinalVerified 不再单独授权普通 Accept:终态 assurance_insufficient 升级,
// 明细逐项可见;执行前 requires(Prevention)已随首次迭代指令到达实现权威;
// 验证事实如实落库(pass,绝不伪装候选失败),修复预算零消耗。
func TestV141GateRequiredL3AchievedL1BlockedE2E(t *testing.T) {
	runs, st := clStore(t)
	repo := clNewFixtureRepo(t, "bug_fix", []string{"schema_migration"},
		map[string]string{".tally/policies/policy-migration-gate.yaml": gateMigrationPolicyYAML})
	cfg := clConfigFor(t, repo)
	rt := newClScriptedRuntime(clStep{write: clContentGood()})

	obs := clRunTask(t, context.Background(), clDeps{cfg: cfg, queries: st.Queries(), runtime: rt}, clTaskID)
	res := obs.Result

	// 终态:验收守卫裁决 —— 普通接受被拒,typed 升级,无 basis,单次尝试
	// (守卫在 Router 之前,不伪装候选失败、不耗修复预算)。
	if res.Phase != closedloop.PhaseEscalated || res.Escalation == nil ||
		res.Escalation.Reason != closedloop.EscalationAssuranceInsufficient {
		t.Fatalf("终态 = %s / %+v, 期望 assurance_insufficient 升级(L3-required/L1-achieved 被阻断)", res.Phase, res.Escalation)
	}
	if res.Basis != nil {
		t.Fatalf("assurance_insufficient 升级不得携带 basis: %+v", res.Basis)
	}
	if want := "acceptance guard: assurance gap [AC-01 required L3 achieved L1]"; res.Escalation.Detail != want {
		t.Fatalf("守卫裁决明细 = %q, 期望 %q", res.Escalation.Detail, want)
	}
	if len(res.Attempts) != 1 {
		t.Fatalf("尝试 = %d, 期望 1(守卫拒绝不是候选失败,不进修复)", len(res.Attempts))
	}

	// Prevention 对齐(P1-05 test/** 面):编译权威的执行前 requires 通知行
	// 随首次迭代指令到达实现权威(策略公开声明,无 held-out 材料)。
	if len(obs.Instrs) != 1 {
		t.Fatalf("迭代指令 = %d 条, 期望 1", len(obs.Instrs))
	}
	if !strings.Contains(obs.Instrs[0], "Prevention requirements (compiled contract)") ||
		!strings.Contains(obs.Instrs[0], "fresh_database") {
		t.Fatalf("首次指令缺编译权威 Prevention 小节(requires 键 fresh_database): %q", obs.Instrs[0])
	}

	// 验证事实面:候选确实 FinalVerified(可信 pass 落库)—— 阻断来自验收
	// 守卫对编译权威的裁决,不是验证失败,也不是 fabricated verdict。
	ctx := context.Background()
	vruns, err := runs.VerificationRunsForTask(ctx, clProject, clTaskID)
	if err != nil || len(vruns) != 1 || vruns[0].Status != runstore.VerificationPass {
		t.Fatalf("verification_run = %+v err=%v, 期望恰好 1 次 pass(阻断来自守卫,不是验证)", vruns, err)
	}
	if evs, err := runs.EvidenceForRun(ctx, clProject, vruns[0].ID); err != nil || len(evs) != 1 ||
		evs[0].Verdict != evidence.VerdictPass || evs[0].BindingID != clVisID {
		t.Fatalf("evidence = %+v err=%v, 期望 VIS-1 pass", evs, err)
	}
	// 权威面:必需档 L3 来自命中策略的风险上调(不是 unmatched —— 信号已被
	// 策略声明),unmatched 集为空。
	cov := obs.Authority.Compiled.Coverage()
	if cov.HasUnmatchedRisk() {
		t.Fatalf("unmatched = %v, 期望空(schema_migration 由命中策略声明)", cov.UnmatchedRiskSignals)
	}
	for _, cl := range obs.Authority.Compiled.Clauses() {
		if cl.Assurance.Required != assurance.LevelL3 {
			t.Fatalf("条款 %s required = %s, 期望命中 gate 策略后风险上调 L3", cl.AcceptanceID, cl.Assurance.Required)
		}
	}

	clAssertOwnerClean(t, repo)
	clAssertRootEmpty(t, cfg.WorktreeRoot)
}

// ---- gate-3:unknown/unmatched 风险 fail-safe --------------------------------

// TestV141GateUnknownUnmatchedRiskFailsafeE2E owner 确认的 unknown 信号没有任何
// 策略声明:编译覆盖逐字保真(unknown/unmatched 双集合),必需档 fail-safe
// 上调 L3,verified 候选照样不得普通接受 —— 绝不静默选择最弱保证。
func TestV141GateUnknownUnmatchedRiskFailsafeE2E(t *testing.T) {
	runs, st := clStore(t)
	repo := clNewFixtureRepo(t, "bug_fix", []string{gateUnknownSignal}, nil)
	cfg := clConfigFor(t, repo)
	rt := newClScriptedRuntime(clStep{write: clContentGood()})

	obs := clRunTask(t, context.Background(), clDeps{cfg: cfg, queries: st.Queries(), runtime: rt}, clTaskID)
	res := obs.Result

	if res.Phase != closedloop.PhaseEscalated || res.Escalation == nil ||
		res.Escalation.Reason != closedloop.EscalationAssuranceInsufficient {
		t.Fatalf("终态 = %s / %+v, 期望 assurance_insufficient 升级(unknown/unmatched fail-safe)", res.Phase, res.Escalation)
	}
	if res.Basis != nil {
		t.Fatalf("升级不得携带 basis: %+v", res.Basis)
	}
	// 信号词面逐字进入裁决明细(INV-V14-09);unknown 永不丢弃、永不降档。
	if !strings.Contains(res.Escalation.Detail, "unmatched risk signals ["+gateUnknownSignal+"]") {
		t.Fatalf("守卫明细缺 unknown 信号逐字 %q: %q", gateUnknownSignal, res.Escalation.Detail)
	}
	if !strings.Contains(res.Escalation.Detail, "assurance gap [AC-01 required L3 achieved L1]") {
		t.Fatalf("守卫明细缺 fail-safe 上调缺口: %q", res.Escalation.Detail)
	}
	cov := obs.Authority.Compiled.Coverage()
	if !cov.HasUnmatchedRisk() ||
		len(cov.UnmatchedRiskSignals) != 1 || cov.UnmatchedRiskSignals[0] != risk.SignalID(gateUnknownSignal) ||
		len(cov.UnknownRiskSignals) != 1 || cov.UnknownRiskSignals[0] != risk.SignalID(gateUnknownSignal) {
		t.Fatalf("覆盖 = unmatched %v unknown %v, 期望 unknown 信号逐字双保真", cov.UnmatchedRiskSignals, cov.UnknownRiskSignals)
	}
	if len(res.Attempts) != 1 {
		t.Fatalf("尝试 = %d, 期望 1(fail-safe 升级不耗修复预算)", len(res.Attempts))
	}
	// 验证事实仍是可信 pass:阻断只来自守卫(DL-07)。
	if vruns, err := runs.VerificationRunsForTask(context.Background(), clProject, clTaskID); err != nil ||
		len(vruns) != 1 || vruns[0].Status != runstore.VerificationPass {
		t.Fatalf("verification_run = %+v err=%v, 期望 1 次 pass", vruns, err)
	}

	clAssertOwnerClean(t, repo)
	clAssertRootEmpty(t, cfg.WorktreeRoot)
}

// ---- gate-4:held-out 内容不泄漏 ---------------------------------------------

// TestV141GateHeldOutNeverLeaksThroughCompiledChannels held-out PASS 场景
// (visible + held-out 双 binding,内容含验收 token):编译权威接线后的全部
// 实现可见通道(编码指令,含 requires/Prevention 渲染路径)零 held-out 材料
// —— 密标、store 根、不透明 ref、内容摘要零出现(INV-V14-07);候选在守卫
// 下合法接受(L3 held-out 事实 ≥ L1 必需档)。
func TestV141GateHeldOutNeverLeaksThroughCompiledChannels(t *testing.T) {
	_, st := clStore(t)
	repo, holdoutRoot := clNewHeldRepo(t, true)
	cfg := clConfigFor(t, repo)
	cfg.HoldoutRoot = holdoutRoot
	rt := newClScriptedRuntime(clStep{write: clContentGood()})

	obs := clRunTask(t, context.Background(), clDeps{cfg: cfg, queries: st.Queries(), runtime: rt}, clTaskID)
	res := obs.Result

	if res.Phase != closedloop.PhaseAccepted || res.Basis == nil {
		t.Fatalf("终态 = %s basis=%+v, 期望 held-out PASS 在守卫下 accepted(L3 事实 ≥ L1 必需档)", res.Phase, res.Basis)
	}
	content := []byte("P2-10 controlled held-out expectation; marker " + clHoldoutSecret + "\n")
	for _, s := range clHoldoutLeakSurfaces(t, holdoutRoot, content) {
		for i, in := range obs.Instrs {
			if strings.Contains(in, s) {
				t.Fatalf("实现可见通道泄漏 %q(指令 %d): %q", s, i, in)
			}
		}
	}

	clAssertOwnerClean(t, repo)
	clAssertRootEmpty(t, cfg.WorktreeRoot)
}

// ---- gate-5:蓄意 bypass 夹具在接受之前失败 ----------------------------------

// TestV141GateBypassFailsBeforeAcceptance 三个蓄意 bypass 夹具,全部在产生任何
// runtime 会话或候选事实之前被类型化拒绝:
//
//	a. 拒收策略制品:策略快照不完整 → CompileAuthority ErrPolicyRejections
//	   (绝不带规则缺口执行,DL-09);
//	b. 缺 FrozenIntent:同源 bundle 不完整 → LoadBundle ErrIntentNotFound
//	   (缺一角也能跑的 bundle 不存在,DL-04);
//	c. 混合来源手工 bundle:策略快照来自另一提交 → ErrSourceMismatch
//	   (绕过装载器的手工构造在装配入口复核被拒,DL-04 第二道)。
func TestV141GateBypassFailsBeforeAcceptance(t *testing.T) {
	_, st := clStore(t)

	// -- (a) 拒收策略制品 -----------------------------------------------------
	repoA := clNewFixtureRepo(t, "bug_fix", nil,
		map[string]string{".tally/policies/broken.yaml": "id: [broken\n"})
	cfgA := clConfigFor(t, repoA)
	rtA := newClScriptedRuntime(clStep{write: clContentGood()})
	if _, err := clComposeTask(t, context.Background(), clDeps{cfg: cfgA, queries: st.Queries(), runtime: rtA}, clTaskID); !errors.Is(err, artifacts.ErrPolicyRejections) {
		t.Fatalf("拒收策略夹具: err = %v, want ErrPolicyRejections(编译权威拒绝带规则缺口执行)", err)
	}
	if got := rtA.startsCount(); got != 0 {
		t.Fatalf("拒收策略夹具 runtime 会话 = %d, 期望 0(编译先于任何会话)", got)
	}
	clAssertRootEmpty(t, cfgA.WorktreeRoot)

	// -- (b) 缺 FrozenIntent ---------------------------------------------------
	repoB := clNewRepo(t)
	if err := os.Remove(filepath.Join(repoB, ".tally", "intents", clTaskID+".yaml")); err != nil {
		t.Fatalf("删 intent 制品: %v", err)
	}
	clGit(t, repoB, "add", "-A")
	clGit(t, repoB, "commit", "-m", "gate fixture: drop FrozenIntent")
	cfgB := clConfigFor(t, repoB)
	rtB := newClScriptedRuntime(clStep{write: clContentGood()})
	if _, err := clComposeTask(t, context.Background(), clDeps{cfg: cfgB, queries: st.Queries(), runtime: rtB}, clTaskID); !errors.Is(err, artifacts.ErrIntentNotFound) {
		t.Fatalf("缺 intent 夹具: err = %v, want ErrIntentNotFound(同源 bundle 缺一角即拒)", err)
	}
	if got := rtB.startsCount(); got != 0 {
		t.Fatalf("缺 intent 夹具 runtime 会话 = %d, 期望 0", got)
	}
	clAssertRootEmpty(t, cfgB.WorktreeRoot)

	// -- (c) 混合来源手工 bundle ------------------------------------------------
	repoC := clNewFixtureRepo(t, "bug_fix", []string{"schema_migration"},
		map[string]string{".tally/policies/policy-migration-gate.yaml": gateMigrationPolicyYAML})
	b1, err := artifacts.LoadBundle(repoC, "HEAD", clTaskID)
	if err != nil {
		t.Fatalf("装载基线 bundle: %v", err)
	}
	// 策略换版提交:只动策略内容(内容修订必变),HEAD 前进。
	clWriteFile(t, repoC, ".tally/policies/policy-migration-gate.yaml", gateMigrationPolicyYAML+"\n# v2\n")
	clGit(t, repoC, "add", ".")
	clGit(t, repoC, "commit", "-m", "gate fixture: policy v2")
	ps2, err := artifacts.LoadPolicies(repoC, "HEAD")
	if err != nil {
		t.Fatalf("装载策略快照(HEAD): %v", err)
	}
	// 手工拼装:旧来源提交的 task/intent/verification + 新提交的策略快照 ——
	// 正是 DL-04 禁止的混合来源;CompileAuthority 装配入口复核必须拒绝。
	forged := &artifacts.ExecutionBundle{
		TaskID:       b1.TaskID,
		SourceCommit: b1.SourceCommit,
		Intent:       b1.Intent,
		Task:         b1.Task,
		Verification: b1.Verification,
		Policies:     *ps2,
		ProjectID:    b1.ProjectID,
	}
	if _, err := forged.CompileAuthority(); !errors.Is(err, artifacts.ErrSourceMismatch) {
		t.Fatalf("混合来源手工 bundle: err = %v, want ErrSourceMismatch(DL-04 装配入口复核)", err)
	}

	// 全部 bypass 夹具零候选事实:没有任务在任何 bypass 状态下产生 ChangeSet。
	if n := clCountRows(t, st, "SELECT count(*) FROM changeset WHERE project_id=$1", string(clProject)); n != 0 {
		t.Fatalf("候选事实 = %d 行, 期望 0(bypass 在接受之前失败)", n)
	}
}
