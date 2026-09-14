// v141_contract_assembler 的测试(P1-03):经真实的生产路径(冻结准入 +
// FreezeCommit + LoadBundle)驱动装配缝,钉住任务 Verification 的四个场景
// 与 fail-safe 裁决 ——
//   - policy match:db_migration 的 task_type 确定性事实推导出 schema_migration,
//     命中 gate 策略,身份/出处全部暴露,编译确定性可从来源提交复现(AC2);
//   - no-match:已知信号无策略声明覆盖 → 编译成功,unmatched 可见,L3
//     fail-safe 上调(绝不静默最弱档);零策略零信号 → L1 默认地板;
//   - invalid policy:被拒收的策略阻断执行(ErrPolicyRejections,fail-safe);
//   - unknown risk:枚举外确认提示 → 保真可见 + 同一 fail-safe 上调;
//   - invalid risk basis:零值意图类型化拒收(ErrInvalidEvidence 穿透);
//   - 同源/身份防御:混合来源与混合身份的手工 bundle 拒绝编译。

package artifacts

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"tally/internal/assurance"
	"tally/internal/contract"
	"tally/internal/engineeringpolicy"
	"tally/internal/intake"
	"tally/internal/intakefreeze"
	"tally/internal/intentevidence"
	"tally/internal/risk"
	"tally/internal/verification"
)

// freezeWithInputs 经真实的冻结准入(intakefreeze.Freeze)构造指定 task_type
// 与确认风险提示的三件套夹具:与 handler 将提交的形态同源。hints 传 nil 即
// 无确认提示(task_type 确定性事实仍然成立)。
func freezeWithInputs(t *testing.T, taskID string, taskType intake.TaskType, hints ...string) intakefreeze.FrozenArtifacts {
	t.Helper()
	fa, err := intakefreeze.Freeze(intakefreeze.ConfirmedDraft{
		TaskID:    taskID,
		Profile:   "standard",
		SourceRef: "GitHub Issue #142",
		Draft: intake.Draft{
			TaskType:  taskType,
			Summary:   "装配缝夹具",
			Objective: taskID + " 的装配缝夹具目标",
			Scope: intake.Scope{
				AllowedPaths: []string{"internal/session/**"},
			},
			Acceptance: []intake.AcceptanceClause{
				{ID: "AC-01", Statement: "夹具验收"},
			},
		},
		RiskHints: hints,
	}, intakefreeze.VerificationPlan{
		Evidence: []intakefreeze.EvidenceDeclaration{{
			AcceptanceID: "AC-01",
			Kind:         verification.EvidenceTest,
			Path:         "internal/session/logout_test.go",
		}},
	})
	if err != nil {
		t.Fatalf("冻结夹具 %s: %v", taskID, err)
	}
	return fa
}

// freezeIntoRepo 在 rf 里落一个 base(可选文件)再经生产事务冻结 taskID,
// 返回冻结提交 sha。
func freezeIntoRepo(t *testing.T, rf *repoFixture, baseFiles map[string]string, taskID string, taskType intake.TaskType, hints ...string) string {
	t.Helper()
	base := rf.commit(t, baseFiles, "base")
	res, err := FreezeCommit(context.Background(), rf.dir, FreezeCommitRequest{
		ExpectedBase: base,
		Artifacts:    freezeWithInputs(t, taskID, taskType, hints...),
		Message:      "freeze: 冻结 " + taskID,
		Signature:    &freezeSig,
	})
	if err != nil {
		t.Fatalf("FreezeCommit(%s): %v", taskID, err)
	}
	return res.Commit
}

// TestCompileAuthorityPolicyMatch:db_migration 任务无任何确认提示,task_type
// 确定性事实推导出 schema_migration(P1-02 接线),命中 gate 策略;装配产物
// 暴露全部出处(任务/来源提交/bundle 身份),编译 pin 原样来自 bundle 的合同
// (编译器不改写权威),同一提交复装载再装配逐字节可复现(任务 AC2);冻结
// 之后策略换版,编译身份必变。
func TestCompileAuthorityPolicyMatch(t *testing.T) {
	rf := initRepo(t)
	freezeCommit := freezeIntoRepo(t, rf, map[string]string{
		ManifestPath:                   manifestYAML,
		".tally/policies/db_gate.yaml": policyYAML("db_gate", "迁移类任务必须证明全新库升级路径与数据完整性"),
	}, "TASK-0042", intake.TaskTypeMigration)

	b, err := LoadBundle(rf.dir, freezeCommit, "TASK-0042")
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}
	ca, err := b.CompileAuthority()
	if err != nil {
		t.Fatalf("CompileAuthority: %v", err)
	}

	// 出处(实现要求 3):任务、观测来源提交、bundle 身份全部暴露。
	if ca.TaskID != "TASK-0042" || ca.SourceCommit != freezeCommit {
		t.Errorf("出处 = (%s, %s),want (TASK-0042, %s)", ca.TaskID, ca.SourceCommit, freezeCommit)
	}
	if ca.BundleIdentity != b.Identity() {
		t.Errorf("BundleIdentity = %s,want bundle 身份 %s", ca.BundleIdentity, b.Identity())
	}
	if ca.Compiled == nil || ca.Compiled.Hash() == "" {
		t.Fatalf("Compiled 未产出或无身份哈希")
	}

	// 权威风险输入:task_type 确定性事实(basis tasktype:<task-id>),非确认提示。
	if n := len(ca.Risk.Detected); n != 1 || ca.Risk.Detected[0].ID != risk.SignalSchemaMigration {
		t.Fatalf("detected = %+v,want [schema_migration](db_migration 确定性事实)", ca.Risk.Detected)
	}
	if want := []string{"tasktype:TASK-0042"}; !reflect.DeepEqual(ca.Risk.Detected[0].Basis, want) {
		t.Errorf("basis = %v,want %v", ca.Risk.Detected[0].Basis, want)
	}

	// 编译产物:策略命中(match 场景)、双重用途键、pin 原样来自 bundle。
	if ca.Compiled.TaskID() != "TASK-0042" {
		t.Errorf("Compiled.TaskID = %s", ca.Compiled.TaskID())
	}
	cov := ca.Compiled.Coverage()
	if cov.Confidence != engineeringpolicy.ConfidenceFull || len(cov.MatchedPolicies) != 1 || cov.MatchedPolicies[0].Policy.ID != "db_gate" {
		t.Fatalf("coverage = %+v,want full 且命中 db_gate", cov)
	}
	if want := []string{"data_integrity", "fresh_database"}; !reflect.DeepEqual(ca.Compiled.GateKeys(), want) {
		t.Errorf("GateKeys = %v,want %v", ca.Compiled.GateKeys(), want)
	}
	pins := ca.Compiled.Pins()
	if pins.Task.RevisionHash != b.Task.Contract.RevisionHash() || pins.Verification.Hash != b.Verification.Frozen.Hash() {
		t.Fatalf("Compiled pin %+v 与 bundle 合同不符(编译器不得改写权威)", pins)
	}
	// 保证档:高于最小风险 → 常规 L3 上调(决策链不含 fail-safe)。
	clauses := ca.Compiled.Clauses()
	if len(clauses) != 1 || clauses[0].AcceptanceID != "AC-01" {
		t.Fatalf("clauses = %+v", clauses)
	}
	if clauses[0].Assurance.Required != assurance.LevelL3 {
		t.Errorf("required = %s,want L3(风险上调)", clauses[0].Assurance.Required)
	}
	if !reflect.DeepEqual(clauses[0].Assurance.Basis.Reasons, []assurance.Reason{assurance.ReasonDefaultFloor, assurance.ReasonRiskUplift}) {
		t.Errorf("reasons = %v,want [default_floor, risk_uplift]", clauses[0].Assurance.Basis.Reasons)
	}

	// 确定性(任务 AC2):同 bundle 重复装配同哈希;同一提交复装载再装配
	// 同哈希、同 bundle 身份 —— 输入可从来源提交复现。
	again, err := b.CompileAuthority()
	if err != nil {
		t.Fatalf("CompileAuthority(重放): %v", err)
	}
	if again.Compiled.Hash() != ca.Compiled.Hash() {
		t.Errorf("同一 bundle 两次装配哈希不同:%s vs %s", again.Compiled.Hash(), ca.Compiled.Hash())
	}
	b2, err := LoadBundle(rf.dir, freezeCommit, "TASK-0042")
	if err != nil {
		t.Fatalf("复装载: %v", err)
	}
	ca2, err := b2.CompileAuthority()
	if err != nil {
		t.Fatalf("CompileAuthority(复装载): %v", err)
	}
	if ca2.Compiled.Hash() != ca.Compiled.Hash() || ca2.BundleIdentity != ca.BundleIdentity {
		t.Errorf("同一提交复装配不可复现:hash %s vs %s, identity %s vs %s",
			ca2.Compiled.Hash(), ca.Compiled.Hash(), ca2.BundleIdentity, ca.BundleIdentity)
	}

	// 冻结后策略换版(内容不同 rationale → 不同修订):同一任务合同重新编译,
	// 编译身份必变 —— 编译身份是策略输入的敏感函数。
	v2 := rf.commit(t, map[string]string{
		".tally/policies/db_gate.yaml": policyYAML("db_gate", "换版后的迁移门禁理由"),
	}, "策略 v2")
	b3, err := LoadBundle(rf.dir, v2, "TASK-0042")
	if err != nil {
		t.Fatalf("LoadBundle(v2): %v", err)
	}
	ca3, err := b3.CompileAuthority()
	if err != nil {
		t.Fatalf("CompileAuthority(v2): %v", err)
	}
	if ca3.Compiled.Hash() == ca.Compiled.Hash() {
		t.Error("策略换版后编译身份未变")
	}
}

// TestCompileAuthorityNoMatchKeepsFailSafe:no-match 场景 —— 策略在但未命中
// (已知信号无任何策略声明覆盖):编译成功(无策略命中不是拒收),unmatched
// 经 Coverage 保持可见,全部条款 fail-safe 上调 L3,绝不静默最弱档;零策略
// 零信号的 bundle 同样可编译,维持 L1 默认地板(P1-01 留给本装配器的零策略
// 裁决:空策略集合法,阻断的不是它,而是拒收策略)。
func TestCompileAuthorityNoMatchKeepsFailSafe(t *testing.T) {
	rf := initRepo(t)
	freezeCommit := freezeIntoRepo(t, rf, map[string]string{
		".tally/policies/db_gate.yaml": policyYAML("db_gate", "只覆盖迁移类信号"),
	}, "TASK-0042", intake.TaskTypeBugFix, "auth_symbol_change")

	b, err := LoadBundle(rf.dir, freezeCommit, "TASK-0042")
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}
	ca, err := b.CompileAuthority()
	if err != nil {
		t.Fatalf("CompileAuthority(no-match): %v", err)
	}
	cov := ca.Compiled.Coverage()
	if cov.Confidence != engineeringpolicy.ConfidenceNone || len(cov.MatchedPolicies) != 0 {
		t.Fatalf("confidence/matched = (%s, %d),want none/0", cov.Confidence, len(cov.MatchedPolicies))
	}
	if want := []risk.SignalID{"auth_symbol_change"}; !reflect.DeepEqual(cov.DetectedRiskSignals, want) || !reflect.DeepEqual(cov.UnmatchedRiskSignals, want) {
		t.Fatalf("detected/unmatched = %v/%v,want %v", cov.DetectedRiskSignals, cov.UnmatchedRiskSignals, want)
	}
	if n := len(ca.Compiled.GateKeys()); n != 0 {
		t.Errorf("GateKeys = %v,want 空(没有命中 gate 策略)", ca.Compiled.GateKeys())
	}
	for _, cl := range ca.Compiled.Clauses() {
		if cl.Assurance.Required != assurance.LevelL3 {
			t.Fatalf("clause %s required = %s,want L3(unmatched fail-safe)", cl.AcceptanceID, cl.Assurance.Required)
		}
		if last := cl.Assurance.Basis.Reasons[len(cl.Assurance.Basis.Reasons)-1]; last != assurance.ReasonUnmatchedRiskFailsafe {
			t.Fatalf("clause %s reasons = %v,want 末条 fail-safe", cl.AcceptanceID, cl.Assurance.Basis.Reasons)
		}
	}

	// 零策略 + 零信号:合法的 ConfidenceNone 投影,L1 默认地板。
	rf2 := initRepo(t)
	freezeCommit2 := freezeIntoRepo(t, rf2, map[string]string{ManifestPath: manifestYAML}, "TASK-0043", intake.TaskTypeBugFix)
	b2, err := LoadBundle(rf2.dir, freezeCommit2, "TASK-0043")
	if err != nil {
		t.Fatalf("LoadBundle(零策略): %v", err)
	}
	ca2, err := b2.CompileAuthority()
	if err != nil {
		t.Fatalf("CompileAuthority(零策略): %v", err)
	}
	if cov2 := ca2.Compiled.Coverage(); cov2.Confidence != engineeringpolicy.ConfidenceNone || len(cov2.DetectedRiskSignals) != 0 {
		t.Fatalf("零策略 coverage = %+v,want none 且无信号", cov2)
	}
	for _, cl := range ca2.Compiled.Clauses() {
		if cl.Assurance.Required != assurance.LevelL1 || !reflect.DeepEqual(cl.Assurance.Basis.Reasons, []assurance.Reason{assurance.ReasonDefaultFloor}) {
			t.Fatalf("零策略 clause %s = %s %v,want L1 [default_floor]", cl.AcceptanceID, cl.Assurance.Required, cl.Assurance.Basis.Reasons)
		}
	}
}

// TestCompileAuthorityUnknownRiskVisible:unknown risk 场景 —— 枚举外的确认
// 提示保真进入 unknown(绝不丢弃/改名/低风险解读),经装配进入编译产物的
// Coverage,并以同一 fail-safe 上调 L3(INV-V14-09)。
func TestCompileAuthorityUnknownRiskVisible(t *testing.T) {
	rf := initRepo(t)
	freezeCommit := freezeIntoRepo(t, rf, nil, "TASK-0042", intake.TaskTypeBugFix, "auth_session")

	b, err := LoadBundle(rf.dir, freezeCommit, "TASK-0042")
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}
	ca, err := b.CompileAuthority()
	if err != nil {
		t.Fatalf("CompileAuthority(unknown): %v", err)
	}
	if !ca.Risk.HasUnknown() || len(ca.Risk.Unknown) != 1 || ca.Risk.Unknown[0].ID != "auth_session" {
		t.Fatalf("Risk.Unknown = %+v,want [auth_session] 保真", ca.Risk.Unknown)
	}
	cov := ca.Compiled.Coverage()
	if want := []risk.SignalID{"auth_session"}; !reflect.DeepEqual(cov.UnknownRiskSignals, want) || !reflect.DeepEqual(cov.UnmatchedRiskSignals, want) {
		t.Fatalf("unknown/unmatched = %v/%v,want %v", cov.UnknownRiskSignals, cov.UnmatchedRiskSignals, want)
	}
	for _, cl := range ca.Compiled.Clauses() {
		if cl.Assurance.Required != assurance.LevelL3 || !cl.Assurance.Basis.RiskUnknown {
			t.Fatalf("clause %s = %s RiskUnknown=%v,want L3 + fail-safe", cl.AcceptanceID, cl.Assurance.Required, cl.Assurance.Basis.RiskUnknown)
		}
	}
}

// TestCompileAuthorityRejectsInvalidPolicy:invalid policy 场景 —— 拒收策略
// fail-safe 阻断(P1-01 显式留给本装配器的裁决):bundle 照常可装载(P1-01),
// 但装配拒绝产出 Compiled,绝不带着规则缺口执行。
func TestCompileAuthorityRejectsInvalidPolicy(t *testing.T) {
	rf := initRepo(t)
	freezeCommit := freezeIntoRepo(t, rf, map[string]string{
		".tally/policies/db_gate.yaml":     policyYAML("db_gate", "有效策略"),
		".tally/policies/broken_gate.yaml": gateWithoutConditionYAML,
	}, "TASK-0042", intake.TaskTypeBugFix)

	b, err := LoadBundle(rf.dir, freezeCommit, "TASK-0042")
	if err != nil {
		t.Fatalf("LoadBundle(装载应成功,拒收不属装载裁决): %v", err)
	}
	ca, err := b.CompileAuthority()
	if !errors.Is(err, ErrPolicyRejections) {
		t.Fatalf("CompileAuthority = %v,want ErrPolicyRejections", err)
	}
	if ca != nil {
		t.Fatalf("拒收不得产出 CompiledAuthority:%+v", ca)
	}
	if !strings.Contains(err.Error(), ".tally/policies/broken_gate.yaml") {
		t.Errorf("错误应指认被拒文件, got %v", err)
	}
	if errors.Is(err, ErrSourceMismatch) {
		t.Errorf("拒收策略不应误判为来源不一致")
	}
}

// TestCompileAuthorityRejectsInvalidRiskBasis:invalid risk basis 停止执行
// (P1-02 的委托)—— 零值意图不是证据,类型化 ErrInvalidEvidence 穿透,
// 绝不从非证据推导「无风险」;nil intent 是手工构造错误,类型化报错不 panic。
func TestCompileAuthorityRejectsInvalidRiskBasis(t *testing.T) {
	zero := &ExecutionBundle{
		TaskID:       "TASK-0042",
		SourceCommit: "0000000000000000000000000000000000000000",
		Intent:       &IntentArtifact{},
		Policies:     PolicySnapshot{SourceCommit: "0000000000000000000000000000000000000000"},
	}
	if _, err := zero.CompileAuthority(); !errors.Is(err, intentevidence.ErrInvalidEvidence) {
		t.Fatalf("零值意图 = %v,want ErrInvalidEvidence 穿透", err)
	}
	if _, err := (&ExecutionBundle{Policies: PolicySnapshot{SourceCommit: "c"}, SourceCommit: "c"}).CompileAuthority(); err == nil {
		t.Fatal("nil intent 应报错而非 panic")
	}
}

// TestCompileAuthorityRejectsSourceMismatch:手工拼装的混合来源 bundle
// (策略快照来自另一提交)在装配入口 fail-closed(DL-04)。
func TestCompileAuthorityRejectsSourceMismatch(t *testing.T) {
	rf := initRepo(t)
	freezeCommit := freezeIntoRepo(t, rf, map[string]string{
		".tally/policies/db_gate.yaml": policyYAML("db_gate", "有效策略"),
	}, "TASK-0042", intake.TaskTypeBugFix)
	b, err := LoadBundle(rf.dir, freezeCommit, "TASK-0042")
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}
	b.Policies.SourceCommit = "ffffffffffffffffffffffffffffffffffffffff"
	if _, err := b.CompileAuthority(); !errors.Is(err, ErrSourceMismatch) {
		t.Fatalf("CompileAuthority = %v,want ErrSourceMismatch", err)
	}
}

// TestCompileAuthorityRejectsMixedIdentity:同一仓库冻结两个任务后手工拼装
// 混合身份 bundle —— contract 权威的身份门(ErrTaskMismatch)经装配原样
// 生效,装配层不绕过也不重复实现它。
func TestCompileAuthorityRejectsMixedIdentity(t *testing.T) {
	rf := initRepo(t)
	base := rf.commit(t, map[string]string{ManifestPath: manifestYAML}, "base")
	res1, err := FreezeCommit(context.Background(), rf.dir, FreezeCommitRequest{
		ExpectedBase: base,
		Artifacts:    freezeWithInputs(t, "TASK-0042", intake.TaskTypeBugFix),
		Message:      "freeze: 冻结 TASK-0042",
		Signature:    &freezeSig,
	})
	if err != nil {
		t.Fatalf("FreezeCommit(0042): %v", err)
	}
	res2, err := FreezeCommit(context.Background(), rf.dir, FreezeCommitRequest{
		ExpectedBase: res1.Commit,
		Artifacts:    freezeWithInputs(t, "TASK-0043", intake.TaskTypeBugFix),
		Message:      "freeze: 冻结 TASK-0043",
		Signature:    &freezeSig,
	})
	if err != nil {
		t.Fatalf("FreezeCommit(0043): %v", err)
	}
	b42, err := LoadBundle(rf.dir, res2.Commit, "TASK-0042")
	if err != nil {
		t.Fatalf("LoadBundle(0042): %v", err)
	}
	b43, err := LoadBundle(rf.dir, res2.Commit, "TASK-0043")
	if err != nil {
		t.Fatalf("LoadBundle(0043): %v", err)
	}
	mixed := &ExecutionBundle{
		TaskID:       "TASK-0042",
		SourceCommit: res2.Commit,
		Intent:       b42.Intent,
		Task:         b43.Task,
		Verification: b42.Verification,
		Policies:     b42.Policies,
	}
	if _, err := mixed.CompileAuthority(); !errors.Is(err, contract.ErrTaskMismatch) {
		t.Fatalf("混合身份 = %v,want contract.ErrTaskMismatch 穿透", err)
	}
}
