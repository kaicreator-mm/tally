package arch

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"tally/internal/assurance"
	"tally/internal/engineeringpolicy"
	"tally/internal/evidence"
	"tally/internal/risk"
	"tally/internal/taskstate"
	"tally/internal/verification"
)

// P1-09 / PRD §13 / DECISION_LOCKS 11/21 / INV-V14-09:L2 要求不变量成为测试。
// 本文件把「不存在低保证配未匹配风险」钉成机器检查(AC5):
//
//	v14-AC5a 档位选择器穷举边界:对一切输入组合,required 永不低于默认地板
//	         L1,永不等于实验档 L2(锁 21);unknown/无法匹配的风险 fail-safe
//	         上调 L3 并携带 ReasonUnmatchedRiskFailsafe,绝不静默选择最弱保证;
//	         minimal 风险保持 L1 地板(held-out 不是 universal 默认,锁 11/21);
//	         PolicyFloor=L2 一律 ErrL2NotSelectable。
//	v14-AC5b 经编译器的可见性(P1-07):未匹配风险信号在 Coverage() 原样可见
//	         (含 unknown 保真),并把全部条款必需档上调 L3 —— 「看起来最小」
//	         的任务配了未匹配信号也不得拿到低保证;全覆盖对照保持 L1 地板。
//	v14-AC5c 经 taskstate 的可见性(P1-08):verified 状态配未匹配风险 ⇒
//	         verdict 必须是 VERIFIED_LIMITED(不得显示普通 VERIFIED),信号
//	         与 unknown 集合逐字保真,必需档可见为 L3。

// v14-AC5a:assurance.Select 穷举边界 —— 任何输入组合都不产生 L0/L2 的必需档。
func TestV14AssuranceSelectTierBoundaryExhaustive(t *testing.T) {
	bands := []assurance.RiskBand{
		assurance.RiskMinimal, assurance.RiskElevated, assurance.RiskHigh,
		"", "unknown", "some_unheard_signal", "UNMATCHED",
	}
	levels := []assurance.Level{
		assurance.LevelL0, assurance.LevelL1, assurance.LevelL2,
		assurance.LevelL3, assurance.LevelL4, assurance.Level(9), assurance.Level(-1),
	}

	for _, band := range bands {
		for _, floor := range levels {
			for _, human := range []bool{false, true} {
				in := assurance.ClauseInput{ClauseID: "AC-01", Risk: band, PolicyFloor: floor, HumanAuthority: human}

				sel, err := assurance.Select(in)
				switch floor {
				case assurance.LevelL2:
					// 锁 21:L2 是实验档,不得作为必需档/地板被选择。
					if !errors.Is(err, assurance.ErrL2NotSelectable) {
						t.Fatalf("PolicyFloor=L2: err = %v, want ErrL2NotSelectable(AC5/锁 21)", err)
					}
					continue
				case assurance.Level(9), assurance.Level(-1):
					if !errors.Is(err, assurance.ErrInvalidLevel) {
						t.Fatalf("PolicyFloor=%d: err = %v, want ErrInvalidLevel", int(floor), err)
					}
					continue
				}
				if err != nil {
					t.Fatalf("Select(%v,floor=%d,human=%v): %v", band, int(floor), human, err)
				}

				// 永不低于默认地板 L1(不存在低保证逃逸)。
				if sel.Required < assurance.LevelL1 {
					t.Fatalf("band=%q floor=%d human=%v: required=%s 低于默认地板 L1(AC5)", band, int(floor), human, sel.Required)
				}
				// 永不输出实验档 L2 作为必需档(锁 21)。
				if sel.Required == assurance.LevelL2 {
					t.Fatalf("band=%q floor=%d human=%v: required=L2,L2 不得成为必需档(AC5/锁 21)", band, int(floor), human)
				}

				// unknown/无法匹配风险 fail-safe:上调 L3 + 理由可见,绝不落到 L0/L1。
				if _, known := assuranceBandKnown(band); !known {
					if !sel.Basis.RiskUnknown {
						t.Fatalf("band=%q 未识别为 unknown(AC5:fail-safe 输入面)", band)
					}
					if sel.Required < assurance.LevelL3 {
						t.Fatalf("band=%q unknown 风险 required=%s:未匹配风险不得配低保证(AC5/INV-V14-09)", band, sel.Required)
					}
					if !containsReason(sel.Basis.Reasons, assurance.ReasonUnmatchedRiskFailsafe) {
						t.Fatalf("band=%q 缺少 ReasonUnmatchedRiskFailsafe(AC5:fail-safe 必须可见于决策链)", band)
					}
				}

				// minimal 风险 + 无 policy/human 上调:保持 L1 地板
				//(held-out 不是 universal 默认,锁 11/21 的另一面)。
				if band == assurance.RiskMinimal && floor == assurance.LevelL0 && !human {
					if sel.Required != assurance.LevelL1 {
						t.Fatalf("minimal 风险 required=%s, want L1(默认地板,锁 21)", sel.Required)
					}
					if !containsReason(sel.Basis.Reasons, assurance.ReasonDefaultFloor) {
						t.Fatal("minimal 决策链缺少 ReasonDefaultFloor(AC5:依据必须可见)")
					}
				}
			}
		}
	}
}

// assuranceBandKnown 报告 band 是否是三个已知风险档之一。
func assuranceBandKnown(band assurance.RiskBand) (assurance.RiskBand, bool) {
	switch band {
	case assurance.RiskMinimal, assurance.RiskElevated, assurance.RiskHigh:
		return band, true
	}
	return band, false
}

func containsReason(reasons []assurance.Reason, want assurance.Reason) bool {
	for _, r := range reasons {
		if r == want {
			return true
		}
	}
	return false
}

// v14-AC5b:未匹配风险经 P1-07 编译器原样可见,并把全部条款必需档 fail-safe
// 上调 L3;全覆盖对照保持 L1 地板。
func TestV14CompilerSurfacesUnmatchedRiskAndUpliftsToL3(t *testing.T) {
	frozen := v14FrozenTwoClauses(t, 1)

	// 对照组:零策略、零风险 ⇒ 全条款 L1 地板(锁 21:held-out 不是默认)。
	plain := v14Compile(t, frozen, nil, v14MustRisk(t))
	for _, cl := range plain.Clauses() {
		if cl.Assurance.Required != assurance.LevelL1 {
			t.Fatalf("无风险条款 %s required=%s, want L1(AC5 对照:默认地板)", cl.AcceptanceID, cl.Assurance.Required)
		}
		if cl.Assurance.Basis.RiskUnknown {
			t.Fatalf("无风险条款 %s 被标成 unknown(AC5 对照)", cl.AcceptanceID)
		}
	}

	// 实验组:命中策略覆盖了 schema_migration,但 unknown 信号无任何策略声明
	// ⇒ 未匹配。Coverage 原样可见(unknown 逐字保真),全部条款上调 L3。
	const unheard = risk.SignalID("unheard_governance_signal")
	withUnmatched := v14Compile(t, frozen,
		[]engineeringpolicy.Policy{v14MustPolicy(t, v14GatePolicyYAML)},
		v14MustRisk(t, risk.SignalSchemaMigration, unheard))
	cov := withUnmatched.Coverage()
	if !cov.HasUnmatchedRisk() {
		t.Fatal("Coverage 未暴露未匹配风险(AC5:未覆盖风险必须可见)")
	}
	if len(cov.UnmatchedRiskSignals) != 1 || cov.UnmatchedRiskSignals[0] != unheard {
		t.Fatalf("UnmatchedRiskSignals = %v, want [%s](逐字保真,AC5/INV-V14-09)", cov.UnmatchedRiskSignals, unheard)
	}
	if len(cov.UnknownRiskSignals) != 1 || cov.UnknownRiskSignals[0] != unheard {
		t.Fatalf("UnknownRiskSignals = %v, want [%s](unknown 永不丢弃,AC5)", cov.UnknownRiskSignals, unheard)
	}
	for _, cl := range withUnmatched.Clauses() {
		if cl.Assurance.Required != assurance.LevelL3 {
			t.Fatalf("未匹配风险下条款 %s required=%s, want L3(AC5:fail-safe 上调,绝不最弱保证)", cl.AcceptanceID, cl.Assurance.Required)
		}
		if !cl.Assurance.Basis.RiskUnknown || !containsReason(cl.Assurance.Basis.Reasons, assurance.ReasonUnmatchedRiskFailsafe) {
			t.Fatalf("条款 %s 的保证决策链缺少 unknown/failsafe 依据(AC5)", cl.AcceptanceID)
		}
	}
}

// ---- v14-AC5c 夹具:完整 v1.4 管线(taskstate 投影)---------------------------

// v14GatePolicyYAML 命中 schema_migration 的 gate 策略(内容修订由 Parse 填充)。
const v14GatePolicyYAML = `
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

// v14FrozenTwoClauses 两条款契约:AC-01(visible deterministic,required
// independence 声明齐全)+ AC-02(held_out binding);held_out 要求声明
// 会话隔离与 held-out 不可达(INV-VER-05)。
func v14FrozenTwoClauses(t *testing.T, rev int) *verification.Frozen {
	t.Helper()
	c := &verification.Contract{
		TaskID:   "TASK-P109",
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

// v14-AC5c:verified 状态配未匹配风险 ⇒ VERIFIED_LIMITED,缺口与信号不得
// 藏在普通 VERIFIED 后面(PRD §14 / INV-11)。
func TestV14TaskstateVerifiedLimitedOnUnmatchedRisk(t *testing.T) {
	frozen := v14FrozenTwoClauses(t, 1)
	const unheard = risk.SignalID("unheard_governance_signal")
	compiled := v14Compile(t, frozen,
		[]engineeringpolicy.Policy{v14MustPolicy(t, v14GatePolicyYAML)},
		v14MustRisk(t, risk.SignalSchemaMigration, unheard))

	// 全绿事实链 + 绑定当前冻结契约的两份覆盖报告(validator 通道记录)。
	rec := evidence.Recorder{Kind: evidence.RecorderValidator, Actor: "tally/arch"}
	base := time.Date(2026, 9, 9, 8, 0, 0, 0, time.UTC)
	facts := []taskstate.Fact{
		{ID: "f-plan", Kind: taskstate.FactTaskPlanned, At: base, Recorder: rec, RevisionRef: "sha256:task-contract"},
		{ID: "f-frz", Kind: taskstate.FactVerificationFrozen, At: base.Add(time.Second), Recorder: rec, RevisionRef: frozen.Hash()},
		{ID: "f-run", Kind: taskstate.FactCodegenRunStarted, At: base.Add(2 * time.Second), Recorder: rec, RunID: "run-1"},
		{ID: "f-cand", Kind: taskstate.FactChangeSetProduced, At: base.Add(3 * time.Second), Recorder: rec, RunID: "run-1", ChangeSet: "sha256:cs-p109"},
		{ID: "f-vrun", Kind: taskstate.FactVerificationRunStarted, At: base.Add(4 * time.Second), Recorder: rec, RunID: "vrun-1", ChangeSet: "sha256:cs-p109"},
	}
	functional := &verification.FunctionalCoverageReport{
		TaskID: frozen.TaskID(), ContractRevision: frozen.Revision(), ContractHash: frozen.Hash(),
		Acceptances: []verification.AcceptanceCoverage{
			{AcceptanceID: "AC-01", Covered: true, SatisfiedBy: []string{"EV-1"}},
			{AcceptanceID: "AC-02", Covered: true, SatisfiedBy: []string{"EV-2"}},
		},
		Complete: true,
	}
	independence := &verification.IndependenceCoverageReport{
		TaskID: frozen.TaskID(), ContractRevision: frozen.Revision(), ContractHash: frozen.Hash(),
		Acceptances: []verification.AcceptanceIndependence{
			{AcceptanceID: "AC-01", Required: verification.IndependenceSource, Covered: true},
			{AcceptanceID: "AC-02", Required: verification.IndependenceNone, Covered: true},
		},
		Complete: true,
	}

	res, err := taskstate.Derive(taskstate.Snapshot{
		Contract:     frozen,
		Facts:        facts,
		Functional:   functional,
		Independence: independence,
		Compiled:     compiled,
	})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if res.State != taskstate.StateVerified {
		t.Fatalf("state = %s, want VERIFIED(夹具失真)", res.State)
	}
	vb := res.VerifiedBasis
	if vb == nil {
		t.Fatal("接入 Compiled 后 VerifiedBasis 不应为 nil(AC5)")
	}
	// 未匹配风险 ⇒ 不得是普通 VERIFIED。
	if vb.Verdict != taskstate.VerdictVerifiedLimited {
		t.Fatalf("verdict = %q, want VERIFIED_LIMITED(AC5:未匹配风险不得藏在普通 VERIFIED 后)", vb.Verdict)
	}
	if len(vb.UnmatchedRiskSignals) != 1 || vb.UnmatchedRiskSignals[0] != unheard {
		t.Fatalf("UnmatchedRiskSignals = %v, want [%s](AC5:逐字保真)", vb.UnmatchedRiskSignals, unheard)
	}
	if len(vb.PolicyCoverage.UnknownRiskSignals) != 1 || vb.PolicyCoverage.UnknownRiskSignals[0] != unheard {
		t.Fatalf("PolicyCoverage.UnknownRiskSignals = %v(AC5:unknown 集合保真)", vb.PolicyCoverage.UnknownRiskSignals)
	}
	if vb.AssuranceRequired != assurance.LevelL3 {
		t.Fatalf("AssuranceRequired = %s, want L3(AC5:未匹配风险的必需档可见为 L3)", vb.AssuranceRequired)
	}
	// satisfied 与 waived 之外,缺口条款进入 residual,不被消化。
	if !reflect.DeepEqual(vb.ResidualRisk.UnmetClauses, []string{"AC-01"}) {
		t.Fatalf("ResidualRisk.UnmetClauses = %v, want [AC-01](AC5:保证缺口不得隐藏)", vb.ResidualRisk.UnmetClauses)
	}
}
