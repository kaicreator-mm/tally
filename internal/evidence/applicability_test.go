package evidence

import "testing"

// —— AC3: applicability is scoped to project/task/acceptance/verification/change revisions ——

func TestApplicableToRequiresFullScopeMatch(t *testing.T) {
	base := testScope(nil)

	if !base.ApplicableTo(testScope(nil)) {
		t.Fatal("identical scope must be applicable")
	}

	// Changing any single dimension breaks applicability.
	cases := map[string]func(*Scope){
		"project":               func(s *Scope) { s.Project = "proj-other" },
		"task":                  func(s *Scope) { s.Task = "task-other" },
		"acceptance":            func(s *Scope) { s.Acceptance = "AC-3" },
		"verification revision": func(s *Scope) { s.Verification = "verif-r2" },
		"changeset":             func(s *Scope) { s.ChangeSet = "cs-2" },
	}
	for name, override := range cases {
		if base.ApplicableTo(testScope(override)) {
			t.Fatalf("scope must not stay applicable after %s changes", name)
		}
	}
}

// —— AC2: validity and applicability are separate axes ——

func TestValidityUnaffectedByApplicability(t *testing.T) {
	// Old FAIL fact bound to verification revision r1 / candidate cs-1.
	f := mustFact(t, "fail-1", VerdictFail, testScope(nil), recRun)

	current := testScope(func(s *Scope) { s.Verification = "verif-r2"; s.ChangeSet = "cs-2" })
	if f.Scope().ApplicableTo(current) {
		t.Fatal("fact bound to superseded revisions must not be applicable to current context")
	}
	// Not applicable does not mean invalid: the FAIL stays a historical fact.
	if !f.IsValid() {
		t.Fatal("inapplicable fact must stay valid (validity is a separate axis)")
	}
}

func TestInvalidityUnaffectedByApplicability(t *testing.T) {
	f := mustFact(t, "f1", VerdictFail, testScope(nil), recValidator)
	r, err := NewRuling(RulingInvalidEvidence, "frozen protocol violated", testTime, recHuman)
	if err != nil {
		t.Fatalf("NewRuling: %v", err)
	}
	g := f.WithRuling(r)

	// Ruled invalid, but the scope binding is untouched.
	if !g.Scope().ApplicableTo(testScope(nil)) {
		t.Fatal("invalidating ruling must not change applicability")
	}
	if g.IsValid() {
		t.Fatal("fact should be invalid after ruling")
	}
	if g.Gates(testScope(nil)) {
		t.Fatal("invalid fact must not gate even when its scope matches")
	}
}

func TestGatesRequiresValidAndApplicable(t *testing.T) {
	current := testScope(nil)

	validApplicable := mustFact(t, "f1", VerdictPass, testScope(nil), recValidator)
	if !validApplicable.Gates(current) {
		t.Fatal("valid and applicable fact must gate")
	}

	validInapplicable := mustFact(t, "f2", VerdictPass, testScope(func(s *Scope) { s.ChangeSet = "cs-old" }), recValidator)
	if validInapplicable.Gates(current) {
		t.Fatal("inapplicable fact must not gate")
	}
	if !validInapplicable.IsValid() {
		t.Fatal("inapplicable fact must stay valid")
	}

	r, err := NewRuling(RulingInvalidSpecDefect, "spec defect", testTime, recHuman)
	if err != nil {
		t.Fatalf("NewRuling: %v", err)
	}
	invalidApplicable := validApplicable.WithRuling(r)
	if invalidApplicable.Gates(current) {
		t.Fatal("invalid fact must not gate even when applicable")
	}
}

// —— PRD §8.8.2 scenario: owner revision keeps the old FAIL as valid history
// while only current-revision evidence gates the State Engine. ——

func TestSupersededRevisionKeepsHistoryButStopsGating(t *testing.T) {
	l := NewLedger()

	oldScope := testScope(nil) // verif-r1 / cs-1
	oldFail := mustFact(t, "fail-1", VerdictFail, oldScope, recRun)
	if err := l.Record(oldFail); err != nil {
		t.Fatalf("Record(fail-1): %v", err)
	}

	// Owner formally revises the requirement: new verification revision and
	// a new candidate. The old FAIL is not erased by the new PASS binding.
	current := testScope(func(s *Scope) { s.Verification = "verif-r2"; s.ChangeSet = "cs-2" })
	newPass := mustFact(t, "pass-2", VerdictPass, current, recRun)
	if err := l.Record(newPass); err != nil {
		t.Fatalf("Record(pass-2): %v", err)
	}

	gating := l.FactsGating(current)
	if len(gating) != 1 || gating[0].ID() != FactID("pass-2") {
		t.Fatalf("FactsGating(current) = %v, want only pass-2", gating)
	}

	// The old FAIL remains recorded, valid, and separately traceable.
	stored, ok := l.Fact("fail-1")
	if !ok {
		t.Fatal("old FAIL fact must remain recorded (append-only history)")
	}
	if !stored.IsValid() {
		t.Fatal("old FAIL must stay valid; a revision is not an invalidation")
	}
	if stored.Verdict() != VerdictFail {
		t.Fatal("old FAIL verdict must be preserved")
	}
	if stored.Scope().ApplicableTo(current) {
		t.Fatal("old FAIL must only gate the revision it is bound to")
	}
}

func TestLedgerFactsGatingAcrossScopes(t *testing.T) {
	l := NewLedger()
	current := testScope(nil)

	otherTask := mustFact(t, "t2", VerdictPass, testScope(func(s *Scope) { s.Task = "task-2" }), recValidator)
	otherAcceptance := mustFact(t, "a3", VerdictPass, testScope(func(s *Scope) { s.Acceptance = "AC-3" }), recValidator)
	otherCandidate := mustFact(t, "c9", VerdictPass, testScope(func(s *Scope) { s.ChangeSet = "cs-9" }), recValidator)
	matchingFail := mustFact(t, "f1", VerdictFail, testScope(nil), recRun)
	matchingPass := mustFact(t, "f2", VerdictPass, testScope(nil), recValidator)

	for _, f := range []Fact{otherTask, otherAcceptance, otherCandidate, matchingFail, matchingPass} {
		if err := l.Record(f); err != nil {
			t.Fatalf("Record(%s): %v", f.ID(), err)
		}
	}

	got := l.FactsGating(current)
	if len(got) != 2 {
		t.Fatalf("FactsGating returned %d facts, want 2", len(got))
	}
	if got[0].ID() != FactID("f1") || got[1].ID() != FactID("f2") {
		t.Fatalf("FactsGating order/content wrong: %v", got)
	}
}

func TestLedgerFactsGatingExcludesRuledInvalidFact(t *testing.T) {
	l := NewLedger()
	if err := l.Record(mustFact(t, "f1", VerdictFail, testScope(nil), recValidator)); err != nil {
		t.Fatalf("Record: %v", err)
	}
	r, err := NewRuling(RulingInvalidHoldoutDefect, "holdout defect", testTime, recHuman)
	if err != nil {
		t.Fatalf("NewRuling: %v", err)
	}
	if _, err := l.AppendRuling("f1", r); err != nil {
		t.Fatalf("AppendRuling: %v", err)
	}

	if got := l.FactsGating(testScope(nil)); len(got) != 0 {
		t.Fatalf("FactsGating should exclude ruled-invalid fact, got %v", got)
	}
	// History is still queryable and still in the ledger.
	if f, ok := l.Fact("f1"); !ok || len(f.Rulings()) != 1 {
		t.Fatalf("ruled fact must remain stored with its ruling: ok=%v rulings=%d", ok, len(f.Rulings()))
	}
}
