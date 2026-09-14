package evidence

import (
	"testing"
	"time"
)

// Shared scenario clock (testTime = 2026-09-06 12:00 UTC from types_test.go):
//
//	11:00  optional precommit frozen
//	11:30  candidate cs-1 fixed (root candidate)
//	11:45  first held-out verdict recorded on the binding
//	12:00  typical exposure observation / descendant cs-2 fixation
var (
	indPrecommitAt  = testTime.Add(-60 * time.Minute)
	indFixedAt      = testTime.Add(-30 * time.Minute)
	indFirstVerdict = testTime.Add(-15 * time.Minute)
)

func indBinding(rev VerificationRevision, id string) BindingRef {
	return BindingRef{Project: "proj-1", Verification: rev, BindingID: id}
}

// indVerdict projects one verdict record the way callers do from a Fact.
func indVerdict(b BindingRef, task TaskID, cs ChangeSetID, verdict Verdict, valid bool, at time.Time) VerdictRecord {
	return VerdictRecord{
		Binding:    b,
		Task:       task,
		ChangeSet:  cs,
		Verdict:    verdict,
		Valid:      valid,
		RecordedAt: at,
	}
}

// indBase returns a clean held-out scenario that must derive independent:
// held_out binding with a valid applicable verdict, candidate fixed before
// the verdict, no exposures, no precommit.
func indBase() InformationIndependenceInput {
	b := indBinding("verif-r1", "EV-AC02-H1")
	return InformationIndependenceInput{
		Binding:          b,
		Visibility:       VisibilityHeldOut,
		Candidate:        "cs-1",
		CandidateFixedAt: indFixedAt,
		Lineage:          []ChangeSetID{"cs-1"},
		Verdicts: []VerdictRecord{
			indVerdict(b, "task-1", "cs-1", VerdictPass, true, indFirstVerdict),
		},
	}
}

// —— visibility precondition (§8.4.1/§8.5) ——

func TestOnlyHeldOutVisibilityCanBeInformationIndependent(t *testing.T) {
	for _, vis := range []VisibilityPolicy{VisibilityVisible, VisibilityPolicy(""), VisibilityPolicy("public")} {
		in := indBase()
		in.Visibility = vis
		if EffectiveInformationIndependence(in) {
			t.Fatalf("visibility %q must never be information-independent", vis)
		}
	}
}

// —— AC1: held-out content exposure makes relevant lineage compromised ——

func TestContentExposureCompromisesRelevantLineage(t *testing.T) {
	b := indBinding("verif-r1", "EV-AC02-H1")

	persistent := mustExposure(t, ExposureSpec{ID: "exp-idx", Binding: b, Channel: ChannelKnowledgeIndex, ExposedAt: testTime})
	in := indBase()
	in.Exposures = []Exposure{persistent}
	if EffectiveInformationIndependence(in) {
		t.Fatal("persistent-channel exposure must permanently compromise the binding")
	}

	anchoredAncestor := mustExposure(t, ExposureSpec{ID: "exp-anc", Binding: b, ChangeSet: "cs-0", Channel: ChannelDebugDisclosure, ExposedAt: testTime})
	in = indBase()
	in.Lineage = []ChangeSetID{"cs-1", "cs-0"}
	in.Exposures = []Exposure{anchoredAncestor}
	if EffectiveInformationIndependence(in) {
		t.Fatal("exposure anchored into the candidate lineage must compromise the candidate")
	}

	unrelated := mustExposure(t, ExposureSpec{ID: "exp-other", Binding: b, ChangeSet: "cs-9", Channel: ChannelDebugDisclosure, ExposedAt: testTime})
	in = indBase()
	in.Exposures = []Exposure{unrelated}
	if !EffectiveInformationIndependence(in) {
		t.Fatal("exposure on an unrelated lineage must not compromise this candidate")
	}

	other := mustExposure(t, ExposureSpec{ID: "exp-ob", Binding: indBinding("verif-r1", "EV-OTHER"), Channel: ChannelKnowledgeIndex, ExposedAt: testTime})
	in = indBase()
	in.Exposures = []Exposure{other}
	if !EffectiveInformationIndependence(in) {
		t.Fatal("exposure facts of another binding must not leak across")
	}
}

func TestUnanchoredLocalExposureJudgedByFixationTime(t *testing.T) {
	b := indBinding("verif-r1", "EV-AC02-H1")
	e := mustExposure(t, ExposureSpec{ID: "exp-1", Binding: b, Channel: ChannelSessionLeak, ExposedAt: testTime})

	before := indBase()
	before.Exposures = []Exposure{e}
	// base candidate was fixed at 11:30, before the 12:00 observation.
	if !EffectiveInformationIndependence(before) {
		t.Fatal("candidate provably fixed before the local exposure stays independent")
	}

	after := before
	after.CandidateFixedAt = testTime.Add(time.Minute)
	if EffectiveInformationIndependence(after) {
		t.Fatal("candidate fixed after an unanchored local exposure defaults to compromised")
	}
}

// —— AC2: first verdict starts the selection-exposure clock ——

func TestFirstVerdictExposesLaterUnfixedDescendants(t *testing.T) {
	b := indBinding("verif-r1", "EV-AC02-H1")
	verdicts := []VerdictRecord{indVerdict(b, "task-1", "cs-1", VerdictPass, true, indFirstVerdict)}

	// The original verdict candidate was fixed before its own verdict.
	in := InformationIndependenceInput{
		Binding:          b,
		Visibility:       VisibilityHeldOut,
		Candidate:        "cs-1",
		CandidateFixedAt: indFixedAt,
		Lineage:          []ChangeSetID{"cs-1"},
		Verdicts:         verdicts,
	}
	if !EffectiveInformationIndependence(in) {
		t.Fatal("candidate fixed before the first verdict must not be selection-exposed")
	}

	// A repaired descendant fixed after the verdict defaults to exposed.
	in.Candidate = "cs-2"
	in.CandidateFixedAt = testTime
	in.Lineage = []ChangeSetID{"cs-2", "cs-1"}
	in.Verdicts = append(verdicts, indVerdict(b, "task-1", "cs-2", VerdictPass, true, testTime))
	if EffectiveInformationIndependence(in) {
		t.Fatal("descendant fixed after the first held-out verdict must default selection-exposed")
	}

	// Unprovable fixation time defaults to the contaminated side.
	in.CandidateFixedAt = time.Time{}
	if EffectiveInformationIndependence(in) {
		t.Fatal("unprovable fixation time must default selection-exposed")
	}
}

func TestPrecommitExceptionRequiresMechanicalProof(t *testing.T) {
	b := indBinding("verif-r1", "EV-AC02-H1")
	in := indBase()
	in.Candidate = "cs-2"
	in.CandidateFixedAt = testTime
	in.Lineage = []ChangeSetID{"cs-2", "cs-1"}
	in.Verdicts = append(in.Verdicts, indVerdict(b, "task-1", "cs-2", VerdictPass, true, testTime))
	in.Precommit = SelectionPrecommit{
		FixedAt:                  indPrecommitAt,
		GenerationSchedule:       true,
		CandidateCount:           true,
		VisibleSelectionCriteria: true,
		RetryConditions:          true,
		Adhered:                  true,
	}
	if !EffectiveInformationIndependence(in) {
		t.Fatal("mechanically proven precommit frozen before the first verdict must preserve independence")
	}

	// Fixed at/after the first verdict proves nothing.
	late := in
	late.Precommit.FixedAt = indFirstVerdict
	if EffectiveInformationIndependence(late) {
		t.Fatal("precommit fixed at the first verdict must not count")
	}
	late.Precommit.FixedAt = indFirstVerdict.Add(time.Second)
	if EffectiveInformationIndependence(late) {
		t.Fatal("precommit fixed after the first verdict must not count")
	}

	// A missing commitment or unobserved adherence proves nothing.
	for name, weaken := range map[string]func(*SelectionPrecommit){
		"generation schedule": func(p *SelectionPrecommit) { p.GenerationSchedule = false },
		"candidate count":     func(p *SelectionPrecommit) { p.CandidateCount = false },
		"visible criteria":    func(p *SelectionPrecommit) { p.VisibleSelectionCriteria = false },
		"retry conditions":    func(p *SelectionPrecommit) { p.RetryConditions = false },
		"adherence":           func(p *SelectionPrecommit) { p.Adhered = false },
		"zero fixation time":  func(p *SelectionPrecommit) { p.FixedAt = time.Time{} },
	} {
		broken := in
		weaken(&broken.Precommit)
		if EffectiveInformationIndependence(broken) {
			t.Fatalf("precommit missing %s must default selection-exposed", name)
		}
	}
}

func TestNoVerdictMeansNoSelectionExposure(t *testing.T) {
	b := indBinding("verif-r1", "EV-AC02-H1")
	// Without any recorded verdict there is nothing to select against; the
	// clock rule in isolation does not fire (the coverage rule would still
	// reject this evidence downstream).
	if SelectionExposed(nil, b, testTime, SelectionPrecommit{}) {
		t.Fatal("without any recorded verdict there is no selection exposure")
	}
}

// —— AC3: a new Task ID alone does not reset lineage ——

func TestTaskRekeyDoesNotResetHoldoutConsumption(t *testing.T) {
	b := indBinding("verif-r1", "EV-AC02-H1")

	// Runs originally keyed under task-1; the task is later re-keyed as
	// task-2 on the same frozen contract. The binding ref — the holdout
	// consumption key — is unchanged, so the first verdict still governs.
	verdicts := []VerdictRecord{
		indVerdict(b, "task-1", "cs-1", VerdictPass, true, indFirstVerdict),
	}
	in := InformationIndependenceInput{
		Binding:          b,
		Visibility:       VisibilityHeldOut,
		Candidate:        "cs-2", // produced under the re-keyed task
		CandidateFixedAt: testTime,
		Lineage:          []ChangeSetID{"cs-2", "cs-1"},
		Verdicts:         append(verdicts, indVerdict(b, "task-2", "cs-2", VerdictPass, true, testTime)),
	}
	if EffectiveInformationIndependence(in) {
		t.Fatal("re-keying the task must not reset the first-verdict clock")
	}

	// The clock is insensitive to the traceability-only Task field: the
	// same records with every Task label changed derive identically.
	for i := range in.Verdicts {
		in.Verdicts[i].Task = "task-9"
	}
	if EffectiveInformationIndependence(in) {
		t.Fatal("the Task field is traceability only and must not influence the derivation")
	}

	// What does reset the clock is a genuinely fresh binding on a new
	// contract revision (§8.6.3) — with its own valid verdict for a
	// candidate fixed before that verdict.
	fresh := indBinding("verif-r2", "EV-AC02-H2")
	in.Binding = fresh
	in.CandidateFixedAt = testTime.Add(-time.Minute)
	in.Lineage = []ChangeSetID{"cs-2"}
	in.Verdicts = []VerdictRecord{indVerdict(fresh, "task-2", "cs-2", VerdictPass, true, testTime)}
	if !EffectiveInformationIndependence(in) {
		t.Fatal("a fresh binding on a new contract revision starts with a clean clock")
	}
}

// —— AC4: validity/applicability revision scope is respected ——

func TestValidityRulesCoverageButNotTheClock(t *testing.T) {
	b := indBinding("verif-r1", "EV-AC02-H1")

	// A fact later ruled invalid (confirmed holdout defect, §8.8.1) is
	// projected with Valid=false: it provides no coverage. It was recorded
	// at 11:45, before the later candidate existed.
	fact, err := NewFact("h1", testScope(nil), VerdictPass, indFirstVerdict, recRun)
	if err != nil {
		t.Fatalf("NewFact: %v", err)
	}
	ruling, err := NewRuling(RulingInvalidHoldoutDefect, "held-out test itself defective", testTime, recHuman)
	if err != nil {
		t.Fatalf("NewRuling: %v", err)
	}
	ruled := fact.WithRuling(ruling)
	invalidRecord := VerdictRecord{
		Binding:    b,
		Task:       ruled.Scope().Task,
		ChangeSet:  ruled.Scope().ChangeSet,
		Verdict:    ruled.Verdict(),
		Valid:      ruled.IsValid(),
		RecordedAt: ruled.RecordedAt(),
	}
	if invalidRecord.Valid {
		t.Fatal("sanity: ruled fact must project invalid")
	}

	// A later candidate cs-2 (fixed 11:50) with its own valid verdict
	// recorded at 12:00.
	cs2Verdict := indVerdict(b, "task-1", "cs-2", VerdictPass, true, testTime)
	in := indBase()
	in.Candidate = "cs-2"
	in.CandidateFixedAt = testTime.Add(-10 * time.Minute)
	in.Lineage = []ChangeSetID{"cs-2", "cs-1"}
	in.Verdicts = []VerdictRecord{invalidRecord, cs2Verdict}

	if EffectiveInformationIndependence(in) {
		t.Fatal("a ruled-invalid verdict still started the selection clock (history is not erased)")
	}

	// Control: without the ruled-invalid occurrence the clock would start
	// at the candidate's own 12:00 verdict, which the 11:50 fixation
	// precedes.
	in.Verdicts = []VerdictRecord{cs2Verdict}
	if !EffectiveInformationIndependence(in) {
		t.Fatal("control: with a clean clock the candidate fixed before its verdict stays independent")
	}

	// And an invalid occurrence alone provides no coverage.
	in.Verdicts = []VerdictRecord{invalidRecord}
	in.Candidate = "cs-1"
	in.CandidateFixedAt = indFixedAt
	in.Lineage = []ChangeSetID{"cs-1"}
	if EffectiveInformationIndependence(in) {
		t.Fatal("an invalid verdict provides no coverage")
	}
}

func TestRevisionScopeIsolatesBindings(t *testing.T) {
	old := indBinding("verif-r1", "EV-AC02-H1")
	r1 := indBase() // verdict on r1 at 11:45

	// Owner formally revises the requirement: fresh binding EV-H2 on
	// revision r2 with its own verdict. The r1 verdict must not consume the
	// r2 clock, and r1 exposures must not leak across (§8.8.2 applicability
	// scope, §8.6.3 fresh binding).
	fresh := indBinding("verif-r2", "EV-AC02-H2")
	in := InformationIndependenceInput{
		Binding:          fresh,
		Visibility:       VisibilityHeldOut,
		Candidate:        "cs-2",
		CandidateFixedAt: testTime.Add(-time.Minute),
		Lineage:          []ChangeSetID{"cs-2"},
		Verdicts: append(
			r1.Verdicts,
			indVerdict(fresh, "task-1", "cs-2", VerdictPass, true, testTime),
		),
		Exposures: []Exposure{
			mustExposure(t, ExposureSpec{ID: "exp-r1", Binding: old, Channel: ChannelKnowledgeIndex, ExposedAt: testTime}),
		},
	}
	if !EffectiveInformationIndependence(in) {
		t.Fatal("old-revision verdicts and exposures must not compromise the fresh binding")
	}

	// Conversely the fresh revision's verdict does nothing for the old
	// binding: it holds no valid verdict applicable to the repaired
	// candidate cs-2 (§8.8.2: facts only gate the exact revision/candidate
	// combination they were produced against).
	in.Binding = old
	in.Exposures = nil
	if EffectiveInformationIndependence(in) {
		t.Fatal("the fresh revision's verdict must not cover the old binding")
	}
}

func TestCoverageRequiresSameCandidateAndUsableVerdict(t *testing.T) {
	b := indBinding("verif-r1", "EV-AC02-H1")

	// Verdict for a different candidate: not applicable to this one.
	in := indBase()
	in.Verdicts = []VerdictRecord{indVerdict(b, "task-1", "cs-other", VerdictPass, true, indFirstVerdict)}
	if EffectiveInformationIndependence(in) {
		t.Fatal("a verdict for another candidate provides no coverage here")
	}

	// VerdictError is no usable verdict.
	in.Verdicts = []VerdictRecord{indVerdict(b, "task-1", "cs-1", VerdictError, true, indFirstVerdict)}
	if EffectiveInformationIndependence(in) {
		t.Fatal("an error verdict provides no coverage")
	}
}

// —— AC5: policy cannot redefine compromised as independent ——

func TestCompromiseCannotBeRelabeledIndependent(t *testing.T) {
	b := indBinding("verif-r1", "EV-AC02-H1")
	in := indBase()
	in.Exposures = []Exposure{
		mustExposure(t, ExposureSpec{ID: "exp-1", Binding: b, Channel: ChannelKnowledgeIndex, ExposedAt: testTime}),
	}

	// Every benign variation of the remaining inputs — valid precommit,
	// fail verdict, exposure observed before the candidate was even fixed,
	// unrelated lineage — still derives compromised. The derivation
	// deliberately has no policy, waiver or tolerance field that could
	// relabel the result (§8.10): a waiver can only annotate strict_waived
	// downstream.
	in.Precommit = SelectionPrecommit{
		FixedAt:                  indPrecommitAt,
		GenerationSchedule:       true,
		CandidateCount:           true,
		VisibleSelectionCriteria: true,
		RetryConditions:          true,
		Adhered:                  true,
	}
	in.Verdicts[0].Verdict = VerdictFail
	in.Exposures[0] = mustExposure(t, ExposureSpec{ID: "exp-1", Binding: b, Channel: ChannelKnowledgeIndex, ExposedAt: testTime.Add(-time.Hour)})
	if EffectiveInformationIndependence(in) {
		t.Fatal("persistent content exposure must stay compromised under every benign variation")
	}

	// The same holds for selection compromise: an adhered precommit cannot
	// cure it when the precommit itself missed the first verdict.
	clean := indBase()
	clean.Candidate = "cs-2"
	clean.CandidateFixedAt = testTime
	clean.Lineage = []ChangeSetID{"cs-2", "cs-1"}
	clean.Precommit = SelectionPrecommit{
		FixedAt:                  indFirstVerdict.Add(time.Minute),
		GenerationSchedule:       true,
		CandidateCount:           true,
		VisibleSelectionCriteria: true,
		RetryConditions:          true,
		Adhered:                  true,
	}
	if EffectiveInformationIndependence(clean) {
		t.Fatal("a late precommit cannot relabel a selection-exposed candidate independent")
	}
}

// —— §8.8.3 scenario: HOLDOUT-1 compromised, HOLDOUT-2 independent ——

func TestAcceptanceLevelHoldout1Holdout2(t *testing.T) {
	h1 := indBinding("verif-r1", "EV-AC02-H1")
	h2 := indBinding("verif-r2", "EV-AC02-H2")

	// HOLDOUT-1 passed functionally on the repaired ChangeSet but its
	// content leaked through the validation log on the first attempt.
	repair := ChangeSetID("cs-2")
	records := []VerdictRecord{
		indVerdict(h1, "task-1", "cs-1", VerdictFail, true, indFirstVerdict),
		indVerdict(h1, "task-1", repair, VerdictPass, true, testTime),
		indVerdict(h2, "task-1", repair, VerdictPass, true, testTime),
	}
	exposures := []Exposure{
		mustExposure(t, ExposureSpec{ID: "exp-h1", Binding: h1, ChangeSet: "cs-1", Channel: ChannelValidationLog, ExposedAt: indFirstVerdict}),
	}

	h1In := InformationIndependenceInput{
		Binding:          h1,
		Visibility:       VisibilityHeldOut,
		Candidate:        repair,
		CandidateFixedAt: testTime.Add(-time.Minute),
		Lineage:          []ChangeSetID{repair, "cs-1"},
		Exposures:        exposures,
		Verdicts:         records,
	}
	if EffectiveInformationIndependence(h1In) {
		t.Fatal("HOLDOUT-1 must be compromised on the repaired ChangeSet")
	}

	// HOLDOUT-2 keeps its regression value for HOLDOUT-1's failure history
	// while HOLDOUT-2 satisfies acceptance-level information independence.
	h2In := InformationIndependenceInput{
		Binding:          h2,
		Visibility:       VisibilityHeldOut,
		Candidate:        repair,
		CandidateFixedAt: testTime.Add(-time.Minute),
		Lineage:          []ChangeSetID{repair, "cs-1"},
		Exposures:        exposures,
		Verdicts:         records,
	}
	if !EffectiveInformationIndependence(h2In) {
		t.Fatal("HOLDOUT-2 must remain information-independent")
	}
}
