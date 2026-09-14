package verification

import (
	"strings"
	"testing"
	"time"

	"tally/internal/evidence"
)

// P1-20 测试:Acceptance 级 effective independence coverage(PRD §7.3/§8.5/§8.5.2)。
//
// 共享场景时钟:
//
//	11:00  可选 precommit 冻结
//	11:30  候选 cs-1 固化(根候选)
//	11:45  binding 首个 held-out verdict 记录
//	12:00  典型 exposure 观察时间
var (
	indNow          = time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)
	indFixedAt      = indNow.Add(-30 * time.Minute)
	indFirstVerdict = indNow.Add(-15 * time.Minute)
)

const indProj = "proj-1"

// indMultiYAML:AC-02(information)带三条 binding —— visible 的 EV-AC02-V1
// (只能满足 source 维度)、会被危害的 EV-AC02-H1 与 fresh 的 EV-AC02-H2
// (§8.8.3 的 HOLDOUT-1/HOLDOUT-2 场景)。
const indMultiYAML = `
task_id: TASK-024
revision: 6

acceptances:
  - id: AC-01
    statement: stale leases can be reclaimed
    independence_required: source
    evidence_bindings:
      - id: EV-AC01-01
        authoring_mode: pre_commit
        visibility_policy: visible
        independence_basis:
          source: verification_authority
          information: visible
        evidence:
          type: test
          path: internal/jobs/lease_owner_test.go
          selector: TestReclaimStaleLease

  - id: AC-02
    statement: active leases must never be reclaimed
    independence_required: information
    evidence_bindings:
      - id: EV-AC02-V1
        authoring_mode: pre_commit
        visibility_policy: visible
        independence_basis:
          source: verification_authority
          information: visible
        evidence:
          type: test
          path: internal/jobs/lease_safety_test.go
          selector: TestNeverReclaimLiveLease
      - id: EV-AC02-H1
        authoring_mode: pre_commit
        visibility_policy: held_out
        independence_basis:
          source: verification_authority
          information: held_out
        evidence:
          type: test
          store: holdout
          ref: jobs/lease_safety#TestNeverReclaimLiveLease
      - id: EV-AC02-H2
        authoring_mode: pre_commit
        visibility_policy: held_out
        independence_basis:
          source: verification_authority
          information: held_out
        evidence:
          type: test
          store: holdout
          ref: jobs/lease_recovery#TestRecoveryReclaim

  - id: AC-03
    statement: retries must respect max_attempts
    independence_required: none
    evidence_bindings:
      - id: EV-AC03-01
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: validator
          validator: retry-budget-v1

implementation_runtime_requirements:
  session_isolation: true
  held_out_unreachable: true
  conformance_class: verification-isolation-v1
`

// indMultiYAMLRev7 与 indMultiYAML 同构,但 revision 7 且 AC-02 statement 修订
// → 内容哈希不同:fresh binding 必须形成新 contract revision(§8.6.3)。
const indMultiYAMLRev7 = `
task_id: TASK-024
revision: 7

acceptances:
  - id: AC-01
    statement: stale leases can be reclaimed
    independence_required: source
    evidence_bindings:
      - id: EV-AC01-01
        authoring_mode: pre_commit
        visibility_policy: visible
        independence_basis:
          source: verification_authority
          information: visible
        evidence:
          type: test
          path: internal/jobs/lease_owner_test.go
          selector: TestReclaimStaleLease

  - id: AC-02
    statement: active leases must never be reclaimed during recovery
    independence_required: information
    evidence_bindings:
      - id: EV-AC02-V1
        authoring_mode: pre_commit
        visibility_policy: visible
        independence_basis:
          source: verification_authority
          information: visible
        evidence:
          type: test
          path: internal/jobs/lease_safety_test.go
          selector: TestNeverReclaimLiveLease
      - id: EV-AC02-H1
        authoring_mode: pre_commit
        visibility_policy: held_out
        independence_basis:
          source: verification_authority
          information: held_out
        evidence:
          type: test
          store: holdout
          ref: jobs/lease_safety#TestNeverReclaimLiveLease
      - id: EV-AC02-H2
        authoring_mode: pre_commit
        visibility_policy: held_out
        independence_basis:
          source: verification_authority
          information: held_out
        evidence:
          type: test
          store: holdout
          ref: jobs/lease_recovery#TestRecoveryReclaim

  - id: AC-03
    statement: retries must respect max_attempts
    independence_required: none
    evidence_bindings:
      - id: EV-AC03-01
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: validator
          validator: retry-budget-v1

implementation_runtime_requirements:
  session_isolation: true
  held_out_unreachable: true
  conformance_class: verification-isolation-v1
`

func indMultiFrozen(t *testing.T) *Frozen {
	t.Helper()
	frozen, err := FreezeContract(mustParse(t, indMultiYAML), FreezePolicy{})
	if err != nil {
		t.Fatalf("FreezeContract(indMulti): %v", err)
	}
	return frozen
}

func indMultiFrozenRev7(t *testing.T) *Frozen {
	t.Helper()
	frozen, err := FreezeContract(mustParse(t, indMultiYAMLRev7), FreezePolicy{})
	if err != nil {
		t.Fatalf("FreezeContract(indMulti rev7): %v", err)
	}
	return frozen
}

// indCtx 返回相对 frozen 契约的一致求值上下文:候选 cs-1 在 11:30 固化,
// 实现权威有一个已 provenance 的贡献者。
func indCtx(f *Frozen) IndependenceContext {
	return IndependenceContext{
		Project:          indProj,
		Verification:     evidence.VerificationRevision(f.Hash()),
		ChangeSet:        "cs-1",
		CandidateFixedAt: indFixedAt,
		Lineage:          []evidence.ChangeSetID{"cs-1"},
		Implementation: evidence.ImplementationProvenance{Contributors: []evidence.Contributor{
			{Name: "agent-1", Role: "coding_agent"},
		}},
	}
}

func indRef(f *Frozen, binding string) evidence.BindingRef {
	return evidence.BindingRef{Project: indProj, Verification: evidence.VerificationRevision(f.Hash()), BindingID: binding}
}

func indVerdictRec(f *Frozen, task, binding, cs string, v evidence.Verdict, valid bool, at time.Time) evidence.VerdictRecord {
	return evidence.VerdictRecord{
		Binding:    indRef(f, binding),
		Task:       evidence.TaskID(task),
		ChangeSet:  evidence.ChangeSetID(cs),
		Verdict:    v,
		Valid:      valid,
		RecordedAt: at,
	}
}

func indIn(binding string, revision int) IndependenceEvidence {
	return IndependenceEvidence{BindingID: binding, Revision: revision}
}

// indSourceOK:visible binding 的干净 source-independent applicable 输入
// (pre_commit + verification_authority 由冻结契约声明,无 lineage 接触)。
func indSourceOK(f *Frozen, task, binding string) IndependenceEvidence {
	in := indIn(binding, 6)
	in.Verdicts = []evidence.VerdictRecord{
		indVerdictRec(f, task, binding, "cs-1", evidence.VerdictPass, true, indFirstVerdict),
	}
	return in
}

// indHeldOutOK:held-out binding 的干净输入:valid applicable verdict,候选在
// 首个 verdict 之前固化,无 exposure → source 与 information 两维独立。
func indHeldOutOK(f *Frozen, task, binding string) IndependenceEvidence {
	in := indSourceOK(f, task, binding)
	return in
}

func indExposure(t *testing.T, f *Frozen, binding string, channel evidence.ExposureChannel) evidence.Exposure {
	t.Helper()
	e, err := evidence.NewExposure(evidence.ExposureSpec{
		ID:        evidence.ExposureID("exp-" + binding),
		Binding:   indRef(f, binding),
		Channel:   channel,
		ExposedAt: indNow,
	})
	if err != nil {
		t.Fatalf("NewExposure(%s): %v", binding, err)
	}
	return e
}

// AC1:none 不要求任何独立证据 —— 零输入恒覆盖,SatisfiedBy 恒空;其余
// acceptance 不受影响(覆盖按 acceptance 判定)。
func TestDeriveIndependenceCoverageNoneRequiresNoEvidence(t *testing.T) {
	frozen := indMultiFrozen(t)

	rep, err := DeriveIndependenceCoverage(frozen, indCtx(frozen), nil)
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage: %v", err)
	}
	if rep.TaskID != "TASK-024" || rep.ContractRevision != 6 || rep.ContractHash != frozen.Hash() {
		t.Fatalf("report identity = %s/%d/%s, want TASK-024/6/%s", rep.TaskID, rep.ContractRevision, rep.ContractHash, frozen.Hash())
	}
	if len(rep.Acceptances) != 3 {
		t.Fatalf("acceptances = %d, want 3", len(rep.Acceptances))
	}
	ac01, ac02, ac03 := rep.Acceptances[0], rep.Acceptances[1], rep.Acceptances[2]
	if ac01.AcceptanceID != "AC-01" || ac02.AcceptanceID != "AC-02" || ac03.AcceptanceID != "AC-03" {
		t.Fatalf("acceptance order changed: %s/%s/%s", ac01.AcceptanceID, ac02.AcceptanceID, ac03.AcceptanceID)
	}
	if ac03.Required != IndependenceNone || !ac03.Covered || len(ac03.SatisfiedBy) != 0 {
		t.Fatalf("AC-03 required=%s covered=%v satisfiedBy=%v, want none/true/empty", ac03.Required, ac03.Covered, ac03.SatisfiedBy)
	}
	for _, ac := range []AcceptanceIndependence{ac01, ac02} {
		if ac.Covered {
			t.Fatalf("%s covered with no evidence at all", ac.AcceptanceID)
		}
	}
	if rep.Complete {
		t.Fatalf("source/information requirements unmet: Complete must be false")
	}

	// 有 applicable 输入时 none 仍恒覆盖,且 SatisfiedBy 保持空:没有独立性
	// 要求,也就没有任何 binding「提供独立性」。
	in := indSourceOK(frozen, "TASK-024", "EV-AC03-01")
	rep2, err := DeriveIndependenceCoverage(frozen, indCtx(frozen), []IndependenceEvidence{in})
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage(input): %v", err)
	}
	ac03 = rep2.Acceptances[2]
	if !ac03.Covered || len(ac03.SatisfiedBy) != 0 {
		t.Fatalf("AC-03 with input: covered=%v satisfiedBy=%v, want true/empty", ac03.Covered, ac03.SatisfiedBy)
	}
	if !ac03.Bindings[0].Applicable || !ac03.Bindings[0].Qualifies {
		t.Fatalf("AC-03 binding applicable=%v qualifies=%v, want true/true", ac03.Bindings[0].Applicable, ac03.Bindings[0].Qualifies)
	}
}

// AC2:source 要求至少一条 applicable 且 effective source-independent 的
// evidence;authoring 活动接触、未 provenance 的实现侧、或无当前 candidate 的
// usable verdict 都不能满足。
func TestDeriveIndependenceCoverageSourceRequirement(t *testing.T) {
	frozen := frozenStdContract(t)

	// 干净路径:verification_authority + pre_commit + 无 lineage 接触 + 当前
	// candidate 上的 valid PASS → AC-01(source)覆盖。
	rep, err := DeriveIndependenceCoverage(frozen, indCtx(frozen), []IndependenceEvidence{
		indSourceOK(frozen, "TASK-023", "EV-AC01-01"),
	})
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage: %v", err)
	}
	ac01 := rep.Acceptances[0]
	if ac01.Required != IndependenceSource || !ac01.Covered || len(ac01.SatisfiedBy) != 1 || ac01.SatisfiedBy[0] != "EV-AC01-01" {
		t.Fatalf("AC-01 required=%s covered=%v satisfiedBy=%v, want source/true/[EV-AC01-01]", ac01.Required, ac01.Covered, ac01.SatisfiedBy)
	}
	bi := ac01.Bindings[0]
	if !bi.Applicable || !bi.SourceIndependent || !bi.Qualifies {
		t.Fatalf("binding detail = %+v, want applicable/source-independent/qualifying", bi)
	}
	if rep.Acceptances[1].Covered || !rep.Acceptances[2].Covered {
		t.Fatalf("AC-02=%v AC-03=%v, want false/true", rep.Acceptances[1].Covered, rep.Acceptances[2].Covered)
	}

	// 失败路径:每条都只破坏 source 独立性或 applicability 之一,AC-01 不覆盖。
	dirty := func(mutate func(*IndependenceEvidence)) IndependenceEvidence {
		in := indSourceOK(frozen, "TASK-023", "EV-AC01-01")
		mutate(&in)
		return in
	}
	cases := map[string]IndependenceEvidence{
		"authored in implementation lineage": dirty(func(in *IndependenceEvidence) { in.AuthoredInLineage = true }),
		"changed in implementation lineage":  dirty(func(in *IndependenceEvidence) { in.ChangedInLineage = true }),
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			rep, err := DeriveIndependenceCoverage(frozen, indCtx(frozen), []IndependenceEvidence{in})
			if err != nil {
				t.Fatalf("DeriveIndependenceCoverage: %v", err)
			}
			ac01 := rep.Acceptances[0]
			if ac01.Covered || ac01.Bindings[0].SourceIndependent {
				t.Fatalf("AC-01 covered=%v sourceIndependent=%v, want false/false", ac01.Covered, ac01.Bindings[0].SourceIndependent)
			}
			if rep.Complete {
				t.Fatalf("Complete must be false")
			}
		})
	}

	// §8.2.1:post_hoc authoring 只有 candidate-blind 路径保持 source 独立。
	// post_hoc 契约需 FreezePolicy 显式放行(P1 standard path 默认拒绝)。
	postHoc := &Contract{
		TaskID:   "TASK-025",
		Revision: 1,
		Acceptances: []Acceptance{{
			ID:                   "AC-01",
			Statement:            "post hoc governance upgrade path",
			IndependenceRequired: IndependenceSource,
			EvidenceBindings: []Binding{{
				ID:               "EV-PH",
				AuthoringMode:    AuthoringPostHoc,
				VisibilityPolicy: VisibilityVisible,
				Basis:            &IndependenceBasis{Source: SourceVerificationAuthority},
				Evidence:         EvidenceRef{Type: EvidenceTest, Path: "internal/jobs/ph_test.go", Selector: "TestPH"},
			}},
		}},
	}
	pf, err := FreezeContract(postHoc, FreezePolicy{AllowPostHoc: true})
	if err != nil {
		t.Fatalf("FreezeContract(post_hoc): %v", err)
	}
	phIn := IndependenceEvidence{BindingID: "EV-PH", Revision: 1}
	phIn.Verdicts = []evidence.VerdictRecord{
		indVerdictRec(pf, "TASK-025", "EV-PH", "cs-1", evidence.VerdictPass, true, indFirstVerdict),
	}
	rep, err = DeriveIndependenceCoverage(pf, indCtx(pf), []IndependenceEvidence{phIn})
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage(post_hoc): %v", err)
	}
	if rep.Acceptances[0].Covered || rep.Acceptances[0].Bindings[0].SourceIndependent {
		t.Fatalf("post_hoc without candidate-blind attestation must be source-dependent")
	}
	phIn.CandidateBlind = true
	rep, err = DeriveIndependenceCoverage(pf, indCtx(pf), []IndependenceEvidence{phIn})
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage(post_hoc blind): %v", err)
	}
	if !rep.Acceptances[0].Covered || !rep.Acceptances[0].Bindings[0].SourceIndependent {
		t.Fatalf("candidate-blind post_hoc authoring must preserve source independence")
	}

	// 未 provenance 的实现侧:effective source independence 相对实现权威推导,
	// 空贡献者记录保守取 source-dependent(§8.2.2)。
	ctx := indCtx(frozen)
	ctx.Implementation = evidence.ImplementationProvenance{}
	rep, err = DeriveIndependenceCoverage(frozen, ctx, []IndependenceEvidence{indSourceOK(frozen, "TASK-023", "EV-AC01-01")})
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage(unprovenanced): %v", err)
	}
	if rep.Acceptances[0].Covered {
		t.Fatalf("unprovenanced implementation must keep AC-01 uncovered")
	}

	// 无当前 candidate 的 usable verdict → 不属于 applicable evidence set:
	// 其他 candidate / invalid / error / 历史 revision 的记录都不算。
	notApplicable := map[string][]IndependenceEvidence{
		"verdict on other candidate": {func() IndependenceEvidence {
			in := indIn("EV-AC01-01", 6)
			in.Verdicts = []evidence.VerdictRecord{indVerdictRec(frozen, "TASK-023", "EV-AC01-01", "cs-0", evidence.VerdictPass, true, indFirstVerdict)}
			return in
		}()},
		"invalidated verdict": {func() IndependenceEvidence {
			in := indIn("EV-AC01-01", 6)
			in.Verdicts = []evidence.VerdictRecord{indVerdictRec(frozen, "TASK-023", "EV-AC01-01", "cs-1", evidence.VerdictPass, false, indFirstVerdict)}
			return in
		}()},
		"error verdict": {func() IndependenceEvidence {
			in := indIn("EV-AC01-01", 6)
			in.Verdicts = []evidence.VerdictRecord{indVerdictRec(frozen, "TASK-023", "EV-AC01-01", "cs-1", evidence.VerdictError, true, indFirstVerdict)}
			return in
		}()},
		"record for older revision": {func() IndependenceEvidence {
			in := indSourceOK(frozen, "TASK-023", "EV-AC01-01")
			in.Revision = 5
			return in
		}()},
	}
	for name, records := range notApplicable {
		t.Run(name, func(t *testing.T) {
			rep, err := DeriveIndependenceCoverage(frozen, indCtx(frozen), records)
			if err != nil {
				t.Fatalf("DeriveIndependenceCoverage: %v", err)
			}
			ac01 := rep.Acceptances[0]
			if ac01.Covered || ac01.Bindings[0].Applicable {
				t.Fatalf("AC-01 covered=%v applicable=%v, want false/false", ac01.Covered, ac01.Bindings[0].Applicable)
			}
		})
	}
}

// AC3:information 要求同一条 applicable evidence 同时满足 source 与
// information 独立 —— visible binding 永远不是 information-independent;
// information 独立也永远不能补救 source dependence(§8.5)。
func TestDeriveIndependenceCoverageInformationRequirement(t *testing.T) {
	frozen := indMultiFrozen(t)

	// 干净路径:fresh held-out binding → AC-02 覆盖。
	rep, err := DeriveIndependenceCoverage(frozen, indCtx(frozen), []IndependenceEvidence{
		indHeldOutOK(frozen, "TASK-024", "EV-AC02-H2"),
	})
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage: %v", err)
	}
	ac02 := rep.Acceptances[1]
	if ac02.Required != IndependenceInformation || !ac02.Covered || len(ac02.SatisfiedBy) != 1 || ac02.SatisfiedBy[0] != "EV-AC02-H2" {
		t.Fatalf("AC-02 required=%s covered=%v satisfiedBy=%v, want information/true/[EV-AC02-H2]", ac02.Required, ac02.Covered, ac02.SatisfiedBy)
	}

	// visible + source-independent ≠ information coverage。
	rep, err = DeriveIndependenceCoverage(frozen, indCtx(frozen), []IndependenceEvidence{
		indSourceOK(frozen, "TASK-024", "EV-AC02-V1"),
	})
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage(visible only): %v", err)
	}
	ac02 = rep.Acceptances[1]
	bi := ac02.Bindings[0]
	if !bi.Applicable || !bi.SourceIndependent || bi.InformationIndependent || bi.Qualifies {
		t.Fatalf("visible binding detail = %+v, want applicable+sourceIndependent but not information-independent", bi)
	}
	if ac02.Covered || len(ac02.SatisfiedBy) != 0 {
		t.Fatalf("visible evidence must never satisfy information requirement: covered=%v satisfiedBy=%v", ac02.Covered, ac02.SatisfiedBy)
	}

	// information 独立不能补救 source dependence:held-out binding 无任何
	// contamination,但在实现决策 lineage 内被创作 → source-dependent →
	// 不满足 information 要求。
	in := indHeldOutOK(frozen, "TASK-024", "EV-AC02-H1")
	in.AuthoredInLineage = true
	rep, err = DeriveIndependenceCoverage(frozen, indCtx(frozen), []IndependenceEvidence{in})
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage(source-dependent holdout): %v", err)
	}
	ac02 = rep.Acceptances[1]
	bi = ac02.Bindings[1]
	if !bi.Applicable || bi.SourceIndependent || !bi.InformationIndependent {
		t.Fatalf("holdout detail = %+v, want applicable+informationIndependent but source-dependent", bi)
	}
	if ac02.Covered || len(ac02.SatisfiedBy) != 0 {
		t.Fatalf("information independence must not remedy source dependence: covered=%v satisfiedBy=%v", ac02.Covered, ac02.SatisfiedBy)
	}

	// content exposure(§8.6.1):persistent 通道 exposure 使 binding 的
	// information independence 永久失效。
	in = indHeldOutOK(frozen, "TASK-024", "EV-AC02-H1")
	in.Exposures = []evidence.Exposure{indExposure(t, frozen, "EV-AC02-H1", evidence.ChannelKnowledgeIndex)}
	rep, err = DeriveIndependenceCoverage(frozen, indCtx(frozen), []IndependenceEvidence{in})
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage(exposure): %v", err)
	}
	ac02 = rep.Acceptances[1]
	if ac02.Bindings[1].InformationIndependent || ac02.Covered {
		t.Fatalf("persistent exposure must compromise information independence: %+v covered=%v", ac02.Bindings[1], ac02.Covered)
	}

	// selection exposure(§8.6.2):候选未在首个 held-out verdict 之前固化且无
	// 机械 proven 的 precommit → selection-exposed。
	in = indHeldOutOK(frozen, "TASK-024", "EV-AC02-H1")
	ctx := indCtx(frozen)
	ctx.CandidateFixedAt = indNow.Add(-5 * time.Minute) // 晚于 11:45 的首个 verdict
	rep, err = DeriveIndependenceCoverage(frozen, ctx, []IndependenceEvidence{in})
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage(selection exposure): %v", err)
	}
	ac02 = rep.Acceptances[1]
	if ac02.Bindings[1].InformationIndependent || ac02.Covered {
		t.Fatalf("late-fixed candidate must be selection-exposed: %+v covered=%v", ac02.Bindings[1], ac02.Covered)
	}

	// 唯一例外:首个 verdict 之前冻结且严格遵守的 precommit(§8.6.2)。
	ctx.Precommit = evidence.SelectionPrecommit{
		FixedAt:                  indNow.Add(-60 * time.Minute),
		GenerationSchedule:       true,
		CandidateCount:           true,
		VisibleSelectionCriteria: true,
		RetryConditions:          true,
		Adhered:                  true,
	}
	rep, err = DeriveIndependenceCoverage(frozen, ctx, []IndependenceEvidence{in})
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage(precommit): %v", err)
	}
	ac02 = rep.Acceptances[1]
	if !ac02.Bindings[1].InformationIndependent || !ac02.Covered {
		t.Fatalf("proven precommit must restore information independence: %+v covered=%v", ac02.Bindings[1], ac02.Covered)
	}
}

// AC4:被危害的 binding 保持功能 regression gate 的事实,不被抹掉也不阻断
// independence;fresh binding 提供独立性(§8.8.3 HOLDOUT-1/HOLDOUT-2)。
func TestDeriveIndependenceCoverageFreshBindingSuppliesIndependence(t *testing.T) {
	frozen := indMultiFrozen(t)

	compromised := indHeldOutOK(frozen, "TASK-024", "EV-AC02-H1")
	compromised.Exposures = []evidence.Exposure{indExposure(t, frozen, "EV-AC02-H1", evidence.ChannelKnowledgeIndex)}
	fresh := indHeldOutOK(frozen, "TASK-024", "EV-AC02-H2")

	rep, err := DeriveIndependenceCoverage(frozen, indCtx(frozen), []IndependenceEvidence{compromised, fresh})
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage: %v", err)
	}
	ac02 := rep.Acceptances[1]
	if !ac02.Covered || len(ac02.SatisfiedBy) != 1 || ac02.SatisfiedBy[0] != "EV-AC02-H2" {
		t.Fatalf("AC-02 covered=%v satisfiedBy=%v, want covered by fresh EV-AC02-H2 only", ac02.Covered, ac02.SatisfiedBy)
	}
	h1, h2 := ac02.Bindings[1], ac02.Bindings[2]
	if !h1.Applicable || h1.InformationIndependent || h1.Qualifies {
		t.Fatalf("compromised binding detail = %+v, want applicable but not information-independent/qualifying", h1)
	}
	if !h2.Qualifies {
		t.Fatalf("fresh binding detail = %+v, want qualifying", h2)
	}

	// 被危害 binding 的功能 regression gate 保持原样(§8.8.3):同一场景下
	// HOLDOUT-1 的合法 FAIL 仍阻断功能覆盖,HOLDOUT-2 满足 independence ——
	// 两个维度互不抹掉(Decision Lock 10)。
	frep, err := DeriveFunctionalCoverage(frozen, []EvidenceRecord{
		failRec("EV-AC02-H1", 6),
		passRec("EV-AC02-H2", 6),
		passRec("EV-AC01-01", 6),
		passRec("EV-AC03-01", 6),
	})
	if err != nil {
		t.Fatalf("DeriveFunctionalCoverage: %v", err)
	}
	if frep.Acceptances[1].Covered || len(frep.Acceptances[1].Blocking) != 1 || frep.Acceptances[1].Blocking[0] != "EV-AC02-H1" {
		t.Fatalf("functional AC-02 covered=%v blocking=%v, want blocked by EV-AC02-H1", frep.Acceptances[1].Covered, frep.Acceptances[1].Blocking)
	}
	if !rep.Acceptances[1].Covered || rep.Acceptances[1].SatisfiedBy[0] != "EV-AC02-H2" {
		t.Fatalf("independence AC-02 must stay covered by the fresh binding")
	}

	// §8.6.3:fresh binding 必须形成新 contract revision,是不同 BindingRef ——
	// 旧 revision 的 exposure fact 不会泄漏到新 revision 的同 id binding 上;
	// 但 fresh ≠ independent:新 revision 仍要满足 valid verdict 等全部规则。
	frozen7 := indMultiFrozenRev7(t)
	oldExposure := indExposure(t, frozen, "EV-AC02-H1", evidence.ChannelKnowledgeIndex) // keyed to rev-6 hash
	in := indIn("EV-AC02-H1", 7)
	in.Exposures = []evidence.Exposure{oldExposure}
	in.Verdicts = []evidence.VerdictRecord{indVerdictRec(frozen7, "TASK-024", "EV-AC02-H1", "cs-1", evidence.VerdictPass, true, indFirstVerdict)}
	rep7, err := DeriveIndependenceCoverage(frozen7, indCtx(frozen7), []IndependenceEvidence{in})
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage(rev7): %v", err)
	}
	if !rep7.Acceptances[1].Bindings[1].InformationIndependent {
		t.Fatalf("exposure keyed to old revision BindingRef must not compromise the fresh revision binding")
	}
	if !rep7.Acceptances[1].Covered {
		t.Fatalf("AC-02 (rev 7) covered=%v, want covered", rep7.Acceptances[1].Covered)
	}
}

// AC5:计数永远不是 gate 输入 —— 同一 binding 的多条 usable verdict、跨
// acceptance 的多条 evidence、多条不满足要求的输入,都推不出覆盖;一条满足即可。
func TestDeriveIndependenceCoverageCountNeverGateInput(t *testing.T) {
	frozen := indMultiFrozen(t)

	// AC-01 的 binding 有三条 usable verdict(一条输入内的多条投影):只覆盖
	// AC-01,对 AC-02(information)毫无贡献 —— 覆盖按 acceptance 判定。
	many := indSourceOK(frozen, "TASK-024", "EV-AC01-01")
	many.Verdicts = append(many.Verdicts,
		indVerdictRec(frozen, "TASK-024", "EV-AC01-01", "cs-1", evidence.VerdictPass, true, indFirstVerdict),
		indVerdictRec(frozen, "TASK-024", "EV-AC01-01", "cs-1", evidence.VerdictPass, true, indFirstVerdict),
	)
	rep, err := DeriveIndependenceCoverage(frozen, indCtx(frozen), []IndependenceEvidence{many})
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage: %v", err)
	}
	if !rep.Acceptances[0].Covered || len(rep.Acceptances[0].SatisfiedBy) != 1 {
		t.Fatalf("AC-01 covered=%v satisfiedBy=%v, want covered by one binding", rep.Acceptances[0].Covered, rep.Acceptances[0].SatisfiedBy)
	}
	if rep.Acceptances[1].Covered || len(rep.Acceptances[1].SatisfiedBy) != 0 {
		t.Fatalf("AC-02 covered=%v satisfiedBy=%v, want uncovered by another acceptance's evidence count", rep.Acceptances[1].Covered, rep.Acceptances[1].SatisfiedBy)
	}

	// 多条不满足要求的输入(被危害的 held-out + source-dependent 的 held-out)
	// 无论多少条都推不出 information coverage。
	compromised := indHeldOutOK(frozen, "TASK-024", "EV-AC02-H1")
	compromised.Exposures = []evidence.Exposure{indExposure(t, frozen, "EV-AC02-H1", evidence.ChannelKnowledgeIndex)}
	dependent := indHeldOutOK(frozen, "TASK-024", "EV-AC02-H2")
	dependent.AuthoredInLineage = true
	rep, err = DeriveIndependenceCoverage(frozen, indCtx(frozen), []IndependenceEvidence{compromised, dependent})
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage(non-qualifying): %v", err)
	}
	ac02 := rep.Acceptances[1]
	for _, bi := range ac02.Bindings[1:] {
		if bi.Qualifies {
			t.Fatalf("binding %s must not qualify", bi.BindingID)
		}
	}
	if ac02.Covered || len(ac02.SatisfiedBy) != 0 || rep.Complete {
		t.Fatalf("non-qualifying evidence count must never gate: covered=%v satisfiedBy=%v complete=%v", ac02.Covered, ac02.SatisfiedBy, rep.Complete)
	}
}

// 输入形态与上下文校验:显式报错,不做静默修正;契约外 binding 列入
// Unmatched;历史 revision 输入跳过;重复当前 revision 输入报错。
func TestDeriveIndependenceCoverageInputValidation(t *testing.T) {
	frozen := indMultiFrozen(t)

	if _, err := DeriveIndependenceCoverage(nil, indCtx(frozen), nil); err == nil {
		t.Fatalf("nil contract must error")
	}

	ctx := indCtx(frozen)
	ctx.Verification = evidence.VerificationRevision("sha256:deadbeef")
	if _, err := DeriveIndependenceCoverage(frozen, ctx, nil); err == nil || !strings.Contains(err.Error(), "不一致") {
		t.Fatalf("mismatched verification identity must error explicitly, got %v", err)
	}

	for name, mutate := range map[string]func(*IndependenceContext){
		"empty project":      func(c *IndependenceContext) { c.Project = "" },
		"empty changeset":    func(c *IndependenceContext) { c.ChangeSet = "" },
		"empty verification": func(c *IndependenceContext) { c.Verification = "" },
	} {
		t.Run(name, func(t *testing.T) {
			c := indCtx(frozen)
			mutate(&c)
			if _, err := DeriveIndependenceCoverage(frozen, c, nil); err == nil {
				t.Fatalf("incomplete context must error")
			}
		})
	}

	badRecords := map[string][]IndependenceEvidence{
		"missing binding id": {{Revision: 6}},
		"zero revision":      {{BindingID: "EV-AC01-01", Revision: 0}},
	}
	for name, records := range badRecords {
		t.Run(name, func(t *testing.T) {
			if _, err := DeriveIndependenceCoverage(frozen, indCtx(frozen), records); err == nil {
				t.Fatalf("malformed record must error")
			}
		})
	}

	// 契约外 binding:不参与推导,去重列入 Unmatched(INV-STATE-02)。
	rep, err := DeriveIndependenceCoverage(frozen, indCtx(frozen), []IndependenceEvidence{
		indSourceOK(frozen, "TASK-024", "EV-EXTRA"),
		indIn("EV-EXTRA", 6),
	})
	if err != nil {
		t.Fatalf("DeriveIndependenceCoverage(unmatched): %v", err)
	}
	if len(rep.Unmatched) != 1 || rep.Unmatched[0] != "EV-EXTRA" {
		t.Fatalf("Unmatched = %v, want [EV-EXTRA] once", rep.Unmatched)
	}
	if rep.Acceptances[0].Covered {
		t.Fatalf("unmatched input must not contribute to coverage")
	}

	// 同 binding 的重复当前 revision 输入是调用方错误(authoring provenance 每
	// binding 唯一):显式报错,不静默合并。
	dup := indSourceOK(frozen, "TASK-024", "EV-AC01-01")
	if _, err := DeriveIndependenceCoverage(frozen, indCtx(frozen), []IndependenceEvidence{dup, dup}); err == nil {
		t.Fatalf("duplicate current-revision input must error")
	}
}
