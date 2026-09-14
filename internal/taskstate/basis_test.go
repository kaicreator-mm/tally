// P1-08 的 VerifiedBasis 投影测试:复用 state_test.go 的同包夹具
// (at/validator/humanR/cs1/hasGap/mustDerive)。
//
// 覆盖面:
//   - AC1:不接入 Compiled 时 Result.VerifiedBasis 恒为 nil(v1.3 完全兼容);
//   - AC2:VERIFIED 不能隐藏未匹配风险、waiver 与保证缺口(VERIFIED_LIMITED);
//   - AC3:同一事实集任意输入顺序 ⇒ 完全相同的 Result(含投影);
//   - AC4:投影是纯派生,没有任何 setter;Compiled 与当前冻结契约跨 revision
//     时显式报错(ErrCompiledMismatch),绝不静默降级。
package taskstate

import (
	"errors"
	"reflect"
	"testing"

	"tally/internal/assurance"
	"tally/internal/contract"
	"tally/internal/engineeringpolicy"
	"tally/internal/risk"
	"tally/internal/taskcontract"
	"tally/internal/verification"
)

// ---- 夹具 -------------------------------------------------------------------

// basisTaskYAML 与 basisFrozen 同 task id 的任务合同(可过 taskcontract 校验)。
const basisTaskYAML = `
id: TASK-023
profile: standard
title: verified basis projection
objective: project the v1.4 VerifiedBasis from derived task state
requirement_refs:
  - .tally/requirements/REQ-007.md
scope:
  allowed_paths:
    - internal/taskstate/**
acceptance:
  - id: AC-01
    statement: visible deterministic check passes
  - id: AC-02
    statement: held-out verifier passes
`

// basisGatePolicyYAML 命中 schema_migration 的 gate 策略(内容修订 pin 由
// engineeringpolicy.Parse 填充)。
const basisGatePolicyYAML = `
id: policy-migration
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

// basisFrozen 两条款契约:AC-01(visible deterministic binding,planned
// source independence 声明齐全)+ AC-02(held_out binding)。含 held_out
// 时 RuntimeRequirements 必须声明会话隔离与 held-out 不可达(INV-VER-05)。
func basisFrozen(t *testing.T, rev int) *verification.Frozen {
	t.Helper()
	c := &verification.Contract{
		TaskID:   "TASK-023",
		Revision: rev,
		Acceptances: []verification.Acceptance{
			{
				ID:                   "AC-01",
				Statement:            "visible deterministic check passes",
				IndependenceRequired: verification.IndependenceSource,
				EvidenceBindings: []verification.Binding{{
					ID:               "EV-1",
					AuthoringMode:    verification.AuthoringPreCommit,
					VisibilityPolicy: verification.VisibilityVisible,
					Basis:            &verification.IndependenceBasis{Source: verification.SourceVerificationAuthority},
					Evidence:         verification.EvidenceRef{Type: verification.EvidenceBuild, Path: "Makefile"},
				}},
			},
			{
				ID:        "AC-02",
				Statement: "held-out verifier passes",
				EvidenceBindings: []verification.Binding{{
					ID:               "EV-2",
					AuthoringMode:    verification.AuthoringPreCommit,
					VisibilityPolicy: verification.VisibilityHeldOut,
					Evidence:         verification.EvidenceRef{Type: verification.EvidenceTest, Store: "holdout", Ref: "corpus/secret-12.json"},
				}},
			},
		},
		RuntimeRequirements: verification.RuntimeRequirements{
			SessionIsolation:   true,
			HeldOutUnreachable: true,
			ConformanceClass:   "verification-isolation-v1",
		},
	}
	f, err := verification.FreezeContract(c, verification.FreezePolicy{})
	if err != nil {
		t.Fatalf("FreezeContract: %v", err)
	}
	return f
}

// basisCompiled 用 P1-07 编译器把任务合同、冻结契约与(可选)策略/风险事实
// 编译成已编译合同(seam 原样使用,不手构 contract.Compiled)。
func basisCompiled(t *testing.T, f *verification.Frozen, withPolicy bool, riskIDs ...risk.SignalID) *contract.Compiled {
	t.Helper()
	tc, err := taskcontract.Parse([]byte(basisTaskYAML))
	if err != nil {
		t.Fatalf("taskcontract.Parse: %v", err)
	}
	snap, err := contract.Compile(tc, f)
	if err != nil {
		t.Fatalf("contract.Compile: %v", err)
	}
	var policies []engineeringpolicy.Policy
	if withPolicy {
		p, err := engineeringpolicy.Parse([]byte(basisGatePolicyYAML))
		if err != nil {
			t.Fatalf("engineeringpolicy.Parse: %v", err)
		}
		policies = append(policies, p)
	}
	sigs := make([]risk.Signal, 0, len(riskIDs))
	for _, id := range riskIDs {
		sigs = append(sigs, risk.Signal{ID: id, Basis: []string{"intent:TASK-023"}})
	}
	r, err := risk.Derive(sigs)
	if err != nil {
		t.Fatalf("risk.Derive: %v", err)
	}
	c, err := contract.Bind(snap, policies, r)
	if err != nil {
		t.Fatalf("contract.Bind: %v", err)
	}
	return c
}

// basisLifecycle PLANNED→READY→RUNNING→CANDIDATE(→VALIDATING)事实链。
func basisLifecycle(f *verification.Frozen, withVerifRun bool) []Fact {
	facts := []Fact{
		{ID: "f-plan", Kind: FactTaskPlanned, At: at(0), Recorder: validator, RevisionRef: "sha256:task-contract"},
		{ID: "f-frz", Kind: FactVerificationFrozen, At: at(1), Recorder: validator, RevisionRef: f.Hash()},
		{ID: "f-run", Kind: FactCodegenRunStarted, At: at(2), Recorder: validator, RunID: "run-1"},
		{ID: "f-cand", Kind: FactChangeSetProduced, At: at(3), Recorder: validator, RunID: "run-1", ChangeSet: cs1},
	}
	if withVerifRun {
		facts = append(facts, Fact{ID: "f-vrun", Kind: FactVerificationRunStarted, At: at(4), Recorder: validator, RunID: "vrun-1", ChangeSet: cs1})
	}
	return facts
}

// basisFuncReport 两条款功能覆盖报告(默认全部由各自 binding 满足)。
func basisFuncReport(f *verification.Frozen) *verification.FunctionalCoverageReport {
	return &verification.FunctionalCoverageReport{
		TaskID:           f.TaskID(),
		ContractRevision: f.Revision(),
		ContractHash:     f.Hash(),
		Acceptances: []verification.AcceptanceCoverage{
			{AcceptanceID: "AC-01", Covered: true, SatisfiedBy: []string{"EV-1"}},
			{AcceptanceID: "AC-02", Covered: true, SatisfiedBy: []string{"EV-2"}},
		},
		Complete: true,
	}
}

// basisIndepReport independence 覆盖报告;ac01Covered=false 时 AC-01
// (required=source)存在 planned ✓ → effective ✗ 缺口(§8.10 waiver 面)。
func basisIndepReport(f *verification.Frozen, ac01Covered bool) *verification.IndependenceCoverageReport {
	return &verification.IndependenceCoverageReport{
		TaskID:           f.TaskID(),
		ContractRevision: f.Revision(),
		ContractHash:     f.Hash(),
		Acceptances: []verification.AcceptanceIndependence{
			{AcceptanceID: "AC-01", Required: verification.IndependenceSource, Covered: ac01Covered},
			{AcceptanceID: "AC-02", Required: verification.IndependenceNone, Covered: true},
		},
		Complete: ac01Covered,
	}
}

// basisVerifiedSnapshot 组装「两条款全绿」的推导快照(可按需覆盖字段)。
func basisVerifiedSnapshot(t *testing.T, f *verification.Frozen, c *contract.Compiled, indep *verification.IndependenceCoverageReport) Snapshot {
	t.Helper()
	return Snapshot{
		Contract:     f,
		Facts:        basisLifecycle(f, true),
		Functional:   basisFuncReport(f),
		Independence: indep,
		Compiled:     c,
	}
}

// ---- AC1:向后兼容 -----------------------------------------------------------

// 不接入 Compiled 时,v1.3 推导完全不变:VerifiedBasis 恒为 nil,结论照旧。
func TestBasisNilWithoutCompiled(t *testing.T) {
	res := mustDerive(t, verifiedSnapshot(t))
	if res.VerifiedBasis != nil {
		t.Fatalf("未接入 Compiled 时 VerifiedBasis 应为 nil,得到 %+v", res.VerifiedBasis)
	}
	if res.State != StateVerified || res.VerificationClass != ClassStrictSatisfied {
		t.Fatalf("v1.3 结论改变: state=%s class=%s", res.State, res.VerificationClass)
	}
}

// ---- 普通 VERIFIED ----------------------------------------------------------

// 零策略 + 零风险:两条款 required=L1;AC-01 由 visible binding 满足(achieved
// L1),AC-02 由 held_out binding 满足(achieved L3);无 waiver/无 unmatched ⇒
// 普通 VERIFIED,覆盖率 2/2。
func TestBasisVerifiedPlain(t *testing.T) {
	f := basisFrozen(t, 1)
	c := basisCompiled(t, f, false)
	res := mustDerive(t, basisVerifiedSnapshot(t, f, c, basisIndepReport(f, true)))

	if res.State != StateVerified || res.VerificationClass != ClassStrictSatisfied {
		t.Fatalf("state=%s class=%s", res.State, res.VerificationClass)
	}
	vb := res.VerifiedBasis
	if vb == nil {
		t.Fatal("接入 Compiled 后 VerifiedBasis 不应为 nil")
	}
	if vb.Verdict != VerdictVerified {
		t.Fatalf("verdict = %q, want VERIFIED", vb.Verdict)
	}
	if vb.AssuranceRequired != assurance.LevelL1 {
		t.Fatalf("AssuranceRequired = %s, want L1", vb.AssuranceRequired)
	}
	// 聚合 achieved 按最弱条款诚实上报(min(L1, L3) = L1)。
	if vb.AssuranceAchieved != assurance.LevelL1 {
		t.Fatalf("AssuranceAchieved = %s, want L1", vb.AssuranceAchieved)
	}
	ec := vb.EvidenceCoverage
	if !ec.FunctionalReportMatched || ec.Covered != 2 || ec.Total != 2 || len(ec.Uncovered) != 0 {
		t.Fatalf("EvidenceCoverage = %+v", ec)
	}
	if ratio, ok := ec.Ratio(); !ok || ratio != 1.0 {
		t.Fatalf("Ratio = %v,%v", ratio, ok)
	}
	if vb.PolicyCoverage.MatchedCount != 0 || vb.PolicyCoverage.Confidence != engineeringpolicy.ConfidenceNone {
		t.Fatalf("PolicyCoverage = %+v", vb.PolicyCoverage)
	}
	if len(vb.MatchedPolicies) != 0 || len(vb.UnmatchedRiskSignals) != 0 {
		t.Fatalf("matched=%+v unmatched=%+v", vb.MatchedPolicies, vb.UnmatchedRiskSignals)
	}
	if reflect.DeepEqual(vb.SatisfiedRequirements, []string{"AC-01", "AC-02"}) == false {
		t.Fatalf("SatisfiedRequirements = %v", vb.SatisfiedRequirements)
	}
	if len(vb.WaivedRequirements) != 0 || !residualEmpty(vb.ResidualRisk) {
		t.Fatalf("waived=%v residual=%+v", vb.WaivedRequirements, vb.ResidualRisk)
	}
	if vb.CompiledHash != c.Hash() {
		t.Fatalf("CompiledHash = %q, want %q", vb.CompiledHash, c.Hash())
	}
	if !hasRef(res.Basis, "compiled:"+c.Hash()) {
		t.Fatalf("Basis 缺少 compiled 引用: %v", res.Basis)
	}
}

func hasRef(refs []string, want string) bool {
	for _, r := range refs {
		if r == want {
			return true
		}
	}
	return false
}

// residualEmpty 报告残余风险面是否为空(ResidualRisk 含切片,不可 == 比较)。
func residualEmpty(r ResidualRisk) bool {
	return len(r.UnmetClauses) == 0 && len(r.UnresolvedViolations) == 0 && len(r.BlockingFailures) == 0
}

// ---- AC2:VERIFIED 不能隐藏未匹配风险与保证缺口 ------------------------------

// 未匹配风险(auth_symbol_change 未被任何命中策略声明)→ 条款 required 被
// fail-safe 上调 L3;AC-01 仅 achieved L1 → 保证缺口。状态是 VERIFIED,但
// verdict 必须是 VERIFIED_LIMITED,且缺口与信号原样可见。
func TestBasisVerifiedLimitedUnmatchedRisk(t *testing.T) {
	f := basisFrozen(t, 1)
	c := basisCompiled(t, f, true, risk.SignalSchemaMigration, risk.SignalAuthSymbolChange)
	res := mustDerive(t, basisVerifiedSnapshot(t, f, c, basisIndepReport(f, true)))

	if res.State != StateVerified {
		t.Fatalf("state = %s", res.State)
	}
	vb := res.VerifiedBasis
	if vb == nil {
		t.Fatal("VerifiedBasis 为 nil")
	}
	if vb.Verdict != VerdictVerifiedLimited {
		t.Fatalf("verdict = %q, want VERIFIED_LIMITED(不得显示成普通 VERIFIED)", vb.Verdict)
	}
	if len(vb.UnmatchedRiskSignals) != 1 || vb.UnmatchedRiskSignals[0] != risk.SignalAuthSymbolChange {
		t.Fatalf("UnmatchedRiskSignals = %v", vb.UnmatchedRiskSignals)
	}
	if vb.AssuranceRequired != assurance.LevelL3 || vb.AssuranceAchieved != assurance.LevelL1 {
		t.Fatalf("required=%s achieved=%s, want L3/L1", vb.AssuranceRequired, vb.AssuranceAchieved)
	}
	if reflect.DeepEqual(vb.ResidualRisk.UnmetClauses, []string{"AC-01"}) == false {
		t.Fatalf("UnmetClauses = %v", vb.ResidualRisk.UnmetClauses)
	}
	if reflect.DeepEqual(vb.SatisfiedRequirements, []string{"AC-02"}) == false {
		t.Fatalf("SatisfiedRequirements = %v", vb.SatisfiedRequirements)
	}
	if len(vb.WaivedRequirements) != 0 {
		t.Fatalf("WaivedRequirements = %v", vb.WaivedRequirements)
	}
	// P1-05 覆盖面:命中策略与 confidence 原样投影。
	if len(vb.MatchedPolicies) != 1 || vb.MatchedPolicies[0].ID != "policy-migration" {
		t.Fatalf("MatchedPolicies = %+v", vb.MatchedPolicies)
	}
	if vb.MatchedPolicies[0].Revision == "" {
		t.Fatal("命中策略缺少内容修订 pin")
	}
	if vb.PolicyCoverage.MatchedCount != 1 || vb.PolicyCoverage.Confidence != engineeringpolicy.ConfidencePartial {
		t.Fatalf("PolicyCoverage = %+v", vb.PolicyCoverage)
	}
}

// ---- AC2 / INV-07:waiver 与 satisfied 永不合并 ------------------------------

// AC-01 independence 缺口经有效 Waiver 放行:整体 VERIFIED,但 class 必须是
// strict_waived,verdict 必须是 VERIFIED_LIMITED;AC-01 只出现在 waived 列表,
// 即使其 achieved 档位已满足 required 也不得出现在 satisfied 列表。
func TestBasisVerifiedLimitedWaiver(t *testing.T) {
	f := basisFrozen(t, 1)
	c := basisCompiled(t, f, false)
	snap := basisVerifiedSnapshot(t, f, c, basisIndepReport(f, false))
	snap.WaiverPermitted = true
	snap.Waivers = []WaiverEvidence{waiverFor(f, cs1, "AC-01")}
	res := mustDerive(t, snap)

	if res.State != StateVerified || res.VerificationClass != ClassStrictWaived {
		t.Fatalf("state=%s class=%s", res.State, res.VerificationClass)
	}
	vb := res.VerifiedBasis
	if vb == nil {
		t.Fatal("VerifiedBasis 为 nil")
	}
	if vb.Verdict != VerdictVerifiedLimited {
		t.Fatalf("verdict = %q, want VERIFIED_LIMITED(waiver 不得隐藏在普通 VERIFIED 后)", vb.Verdict)
	}
	if reflect.DeepEqual(vb.WaivedRequirements, []string{"AC-01"}) == false {
		t.Fatalf("WaivedRequirements = %v", vb.WaivedRequirements)
	}
	if reflect.DeepEqual(vb.SatisfiedRequirements, []string{"AC-02"}) == false {
		t.Fatalf("SatisfiedRequirements = %v", vb.SatisfiedRequirements)
	}
	if !residualEmpty(vb.ResidualRisk) {
		t.Fatalf("waived 条款不得计入 residual: %+v", vb.ResidualRisk)
	}
	for _, s := range vb.SatisfiedRequirements {
		for _, w := range vb.WaivedRequirements {
			if s == w {
				t.Fatalf("satisfied 与 waived 合并: %s", s)
			}
		}
	}
}

// ---- NOT_VERIFIED 与残余风险 -------------------------------------------------

// 主状态未达 VERIFIED(CANDIDATE):verdict NOT_VERIFIED,但条款满足面仍按
// 事实投影;Gaps 照旧解释原因。
func TestBasisNotVerifiedBeforeVerificationRun(t *testing.T) {
	f := basisFrozen(t, 1)
	c := basisCompiled(t, f, false)
	snap := basisVerifiedSnapshot(t, f, c, basisIndepReport(f, true))
	snap.Facts = basisLifecycle(f, false)
	res := mustDerive(t, snap)

	if res.State != StateCandidate {
		t.Fatalf("state = %s", res.State)
	}
	vb := res.VerifiedBasis
	if vb == nil || vb.Verdict != VerdictNotVerified {
		t.Fatalf("verdict = %v, want NOT_VERIFIED", vb)
	}
	if reflect.DeepEqual(vb.SatisfiedRequirements, []string{"AC-01", "AC-02"}) == false {
		t.Fatalf("SatisfiedRequirements = %v", vb.SatisfiedRequirements)
	}
	if !hasGap(res, ReqVerificationRun) {
		t.Fatalf("Gaps 缺少 verification_run: %+v", res.Gaps)
	}
}

// unresolved integrity 违规与当前候选的 applicable FAIL 进入 residual_risk。
func TestBasisResidualRisk(t *testing.T) {
	f := basisFrozen(t, 1)
	c := basisCompiled(t, f, false)
	snap := basisVerifiedSnapshot(t, f, c, basisIndepReport(f, true))
	snap.Violations = []IntegrityViolation{{
		ID: "v-1", Kind: IntegrityHoldoutExposure, Ref: "holdout/corpus/secret-12.json", Recorder: validator, At: at(5),
	}}
	snap.Failures = []ApplicableFailure{{
		FactID: "e-fail-1", BindingID: "EV-1", ChangeSet: cs1, Recorder: validator, At: at(6),
	}}
	res := mustDerive(t, snap)

	if res.State != StateValidating {
		t.Fatalf("state = %s", res.State)
	}
	vb := res.VerifiedBasis
	if vb == nil || vb.Verdict != VerdictNotVerified {
		t.Fatalf("verdict = %v, want NOT_VERIFIED", vb)
	}
	if reflect.DeepEqual(vb.ResidualRisk.UnresolvedViolations, []string{"v-1"}) == false {
		t.Fatalf("UnresolvedViolations = %v", vb.ResidualRisk.UnresolvedViolations)
	}
	if reflect.DeepEqual(vb.ResidualRisk.BlockingFailures, []string{"e-fail-1"}) == false {
		t.Fatalf("BlockingFailures = %v", vb.ResidualRisk.BlockingFailures)
	}
}

// ---- AC3:确定性 -------------------------------------------------------------

// 同一事实集任意输入顺序 ⇒ 完全相同的 Result(含 VerifiedBasis 与 Basis)。
func TestBasisDeterministicUnderInputOrder(t *testing.T) {
	f := basisFrozen(t, 1)
	c := basisCompiled(t, f, true, risk.SignalSchemaMigration, risk.SignalAuthSymbolChange)

	snap1 := basisVerifiedSnapshot(t, f, c, basisIndepReport(f, true))
	snap2 := basisVerifiedSnapshot(t, f, c, basisIndepReport(f, true))
	// 逆序重排输入切片(事实、报告条款顺序)。
	reverseFacts(snap2.Facts)
	reverseCoverage(snap2.Functional)
	res1, err := Derive(snap1)
	if err != nil {
		t.Fatalf("Derive(1): %v", err)
	}
	res2, err := Derive(snap2)
	if err != nil {
		t.Fatalf("Derive(2): %v", err)
	}
	if !reflect.DeepEqual(res1, res2) {
		t.Fatalf("输入顺序改变了推导结果:\n%+v\nvs\n%+v", res1, res2)
	}
	if res1.VerifiedBasis == nil || res1.VerifiedBasis.Verdict != VerdictVerifiedLimited {
		t.Fatalf("verdict = %v", res1.VerifiedBasis)
	}
}

func reverseFacts(fs []Fact) {
	for i, j := 0, len(fs)-1; i < j; i, j = i+1, j-1 {
		fs[i], fs[j] = fs[j], fs[i]
	}
}

func reverseCoverage(r *verification.FunctionalCoverageReport) {
	for i, j := 0, len(r.Acceptances)-1; i < j; i, j = i+1, j-1 {
		r.Acceptances[i], r.Acceptances[j] = r.Acceptances[j], r.Acceptances[i]
	}
}

// ---- AC4:无旁路;跨 revision 拒绝投影 ---------------------------------------

// Compiled 与当前冻结契约非同一内容修订(或缺失当前冻结契约)时,Derive 显式
// 报 ErrCompiledMismatch,绝不跨 revision 投影,也不静默降级为 v1.3 结果。
func TestBasisCompiledMismatchBlocked(t *testing.T) {
	f1 := basisFrozen(t, 1)
	f2 := basisFrozen(t, 2)
	c2 := basisCompiled(t, f2, false)

	snap := basisVerifiedSnapshot(t, f1, c2, basisIndepReport(f1, true))
	res, err := Derive(snap)
	if !errors.Is(err, ErrCompiledMismatch) {
		t.Fatalf("err = %v, want ErrCompiledMismatch", err)
	}
	if res.VerifiedBasis != nil || res.State != "" {
		t.Fatalf("错误路径不得输出结论: %+v", res)
	}

	// 缺失当前冻结契约身份:同样拒绝(无法建立修订绑定)。
	snapNoContract := Snapshot{
		Facts:    basisLifecycle(f1, true),
		Compiled: basisCompiled(t, f1, false),
	}
	if _, err := Derive(snapNoContract); !errors.Is(err, ErrCompiledMismatch) {
		t.Fatalf("err = %v, want ErrCompiledMismatch", err)
	}
}
