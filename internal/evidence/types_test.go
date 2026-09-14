package evidence

import (
	"errors"
	"testing"
	"time"
)

var (
	testTime = time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

	recValidator = Recorder{Kind: RecorderValidator, Actor: "go-test-validator"}
	recRun       = Recorder{Kind: RecorderVerificationRun, Actor: "run-17"}
	recHuman     = Recorder{Kind: RecorderHuman, Actor: "owner-1"}
	recExternal  = Recorder{Kind: RecorderExternalSystem, Actor: "ci.example.com"}
	recAgent     = Recorder{Kind: RecorderAgent, Actor: "coding-agent-1"}
)

// testScope returns a fully qualified scope with one dimension overridden.
func testScope(override func(*Scope)) Scope {
	s := Scope{
		Project:      "proj-1",
		Task:         "task-1",
		Acceptance:   "AC-2",
		Verification: "verif-r1",
		ChangeSet:    "cs-1",
	}
	if override != nil {
		override(&s)
	}
	return s
}

func mustFact(t *testing.T, id FactID, verdict Verdict, scope Scope, rec Recorder) Fact {
	t.Helper()
	f, err := NewFact(id, scope, verdict, testTime, rec)
	if err != nil {
		t.Fatalf("NewFact(%s): unexpected error: %v", id, err)
	}
	return f
}

// —— AC1: facts are append-only; no delete or rewrite path ——

func TestLedgerAppendOnlyNoDeleteOrRewrite(t *testing.T) {
	l := NewLedger()
	fail := mustFact(t, "f1", VerdictFail, testScope(nil), recValidator)
	pass := mustFact(t, "f2", VerdictPass, testScope(nil), recRun)
	if err := l.Record(fail); err != nil {
		t.Fatalf("Record(f1): %v", err)
	}
	if err := l.Record(pass); err != nil {
		t.Fatalf("Record(f2): %v", err)
	}

	// Appending a ruling grows the ruling sequence only; the fact set stays
	// at two entries and the other fact is untouched.
	ruling, err := NewRuling(RulingInvalidEvidence, "baseline hash mismatch", testTime.Add(time.Minute), recHuman)
	if err != nil {
		t.Fatalf("NewRuling: %v", err)
	}
	updated, err := l.AppendRuling("f1", ruling)
	if err != nil {
		t.Fatalf("AppendRuling: %v", err)
	}
	if got := len(l.Facts()); got != 2 {
		t.Fatalf("ledger size after ruling = %d, want 2 (append-only, no delete)", got)
	}
	if updated.IsValid() {
		t.Fatal("updated fact view should be invalid after invalidating ruling")
	}
	if got, ok := l.Fact("f1"); !ok || got.IsValid() {
		t.Fatalf("stored fact f1 view: ok=%v valid=%v, want ok=true valid=false", ok, got.IsValid())
	}
	if got, ok := l.Fact("f2"); !ok || !got.IsValid() {
		t.Fatalf("fact f2 must be untouched: ok=%v valid=%v", ok, got.IsValid())
	}

	// Earlier fact views are immutable values: the pre-ruling copy stays valid.
	if !fail.IsValid() || len(fail.Rulings()) != 0 {
		t.Fatalf("original fact value mutated: valid=%v rulings=%d", fail.IsValid(), len(fail.Rulings()))
	}

	// Recorded fact identity is unique forever; re-record is an error, not update.
	if err := l.Record(mustFact(t, "f1", VerdictPass, testScope(nil), recValidator)); !errors.Is(err, ErrDuplicateFact) {
		t.Fatalf("re-record f1 err = %v, want ErrDuplicateFact", err)
	}
}

func TestFactWithRulingDoesNotMutateReceiver(t *testing.T) {
	f := mustFact(t, "f1", VerdictFail, testScope(nil), recValidator)
	r, err := NewRuling(RulingInvalidHoldoutDefect, "holdout defect confirmed", testTime, recHuman)
	if err != nil {
		t.Fatalf("NewRuling: %v", err)
	}

	g := f.WithRuling(r)
	if !f.IsValid() || len(f.Rulings()) != 0 {
		t.Fatalf("receiver mutated: valid=%v rulings=%d", f.IsValid(), len(f.Rulings()))
	}
	if g.IsValid() || len(g.Rulings()) != 1 {
		t.Fatalf("new view wrong: valid=%v rulings=%d", g.IsValid(), len(g.Rulings()))
	}
	// Appending twice from the same base must not cross-contaminate views.
	h := f.WithRuling(r)
	if len(g.Rulings()) != 1 || len(h.Rulings()) != 1 {
		t.Fatalf("branching appends contaminated views: g=%d h=%d", len(g.Rulings()), len(h.Rulings()))
	}
}

func TestFactRulingsReturnsDefensiveCopy(t *testing.T) {
	f := mustFact(t, "f1", VerdictFail, testScope(nil), recValidator)
	r, err := NewRuling(RulingInvalidEvidence, "execution invalid", testTime, recValidator)
	if err != nil {
		t.Fatalf("NewRuling: %v", err)
	}
	g := f.WithRuling(r)

	rulings := g.Rulings()
	rulings[0].Reason = "tampered"
	if got := g.Rulings()[0].Reason; got != "execution invalid" {
		t.Fatalf("ruling mutable through returned slice: reason=%q", got)
	}
}

// —— AC4: LLM proposal cannot construct a confirmed fact without a recorder path ——

func TestNewFactRejectsAgentRecorder(t *testing.T) {
	_, err := NewFact("f1", testScope(nil), VerdictPass, testTime, recAgent)
	if !errors.Is(err, ErrRecorderCannotRecordFact) {
		t.Fatalf("agent recorder err = %v, want ErrRecorderCannotRecordFact", err)
	}
	// Unknown recorder kinds are denied by default.
	_, err = NewFact("f1", testScope(nil), VerdictPass, testTime, Recorder{Kind: RecorderKind("mystery"), Actor: "x"})
	if !errors.Is(err, ErrRecorderCannotRecordFact) {
		t.Fatalf("unknown recorder err = %v, want ErrRecorderCannotRecordFact", err)
	}
}

func TestNewFactAcceptsFactCapableRecorders(t *testing.T) {
	for name, rec := range map[string]Recorder{
		"validator":        recValidator,
		"verification_run": recRun,
		"human":            recHuman,
		"external_system":  recExternal,
	} {
		f, err := NewFact("f1", testScope(nil), VerdictFail, testTime, rec)
		if err != nil {
			t.Fatalf("recorder %s: NewFact: %v", name, err)
		}
		if f.Recorder() != rec {
			t.Fatalf("recorder %s: provenance not preserved", name)
		}
	}
}

func TestConfirmProposalRequiresFactCapableRecorder(t *testing.T) {
	p := Proposal{
		Summary:        "all tests pass",
		Scope:          testScope(nil),
		ClaimedVerdict: VerdictPass,
		ProducedBy:     RecorderAgent,
	}

	// The agent channel cannot confirm its own claim.
	if _, err := Confirm("f1", p, testTime, recAgent); !errors.Is(err, ErrRecorderCannotRecordFact) {
		t.Fatalf("self-confirmed proposal err = %v, want ErrRecorderCannotRecordFact", err)
	}

	// An independent validator confirms it through the recorder path.
	f, err := Confirm("f1", p, testTime, recValidator)
	if err != nil {
		t.Fatalf("Confirm with validator: %v", err)
	}
	if f.Verdict() != VerdictPass || f.ID() != FactID("f1") {
		t.Fatalf("confirmed fact = verdict %q id %q", f.Verdict(), f.ID())
	}
	if !f.IsValid() {
		t.Fatal("confirmed fact should be born valid")
	}
}

func TestNewFactRejectsIncompleteProvenance(t *testing.T) {
	cases := []struct {
		name    string
		id      FactID
		scope   Scope
		verdict Verdict
		at      time.Time
		rec     Recorder
		wantErr error
	}{
		{
			name: "empty id", id: "", scope: testScope(nil), verdict: VerdictPass, at: testTime, rec: recValidator, wantErr: ErrInvalidFact,
		},
		{
			name: "unknown verdict", id: "f1", scope: testScope(nil), verdict: Verdict("green"), at: testTime, rec: recValidator, wantErr: ErrInvalidFact,
		},
		{
			name: "zero time", id: "f1", scope: testScope(nil), verdict: VerdictPass, at: time.Time{}, rec: recValidator, wantErr: ErrInvalidFact,
		},
		{
			name: "empty actor", id: "f1", scope: testScope(nil), verdict: VerdictPass, at: testTime, rec: Recorder{Kind: RecorderValidator}, wantErr: ErrInvalidFact,
		},
		{
			name: "missing scope dimension", id: "f1", scope: testScope(func(s *Scope) { s.ChangeSet = "" }), verdict: VerdictPass, at: testTime, rec: recValidator, wantErr: ErrInvalidFact,
		},
	}
	for _, tc := range cases {
		_, err := NewFact(tc.id, tc.scope, tc.verdict, tc.at, tc.rec)
		if !errors.Is(err, tc.wantErr) {
			t.Fatalf("%s: err = %v, want %v", tc.name, err, tc.wantErr)
		}
	}
}

func TestRecordRejectsZeroFact(t *testing.T) {
	l := NewLedger()
	if err := l.Record(Fact{}); !errors.Is(err, ErrInvalidFact) {
		t.Fatalf("Record(zero fact) err = %v, want ErrInvalidFact", err)
	}
}

// —— Validity: append-only rulings with provenance ——

func TestFactValidUntilInvalidatingRuling(t *testing.T) {
	f := mustFact(t, "f1", VerdictFail, testScope(nil), recValidator)
	if !f.IsValid() {
		t.Fatal("fact must be born valid")
	}

	first, err := NewRuling(RulingInvalidHoldoutDefect, "holdout defect confirmed by owner", testTime, recHuman)
	if err != nil {
		t.Fatalf("NewRuling: %v", err)
	}
	g := f.WithRuling(first)
	if g.IsValid() {
		t.Fatal("fact should lose validity after confirmed holdout_defect ruling")
	}

	// More rulings can be appended, but validity never comes back.
	second, err := NewRuling(RulingInvalidSpecDefect, "spec defect record", testTime.Add(time.Minute), recValidator)
	if err != nil {
		t.Fatalf("NewRuling: %v", err)
	}
	h := g.WithRuling(second)
	if h.IsValid() || len(h.Rulings()) != 2 {
		t.Fatalf("validity must not be restored: valid=%v rulings=%d", h.IsValid(), len(h.Rulings()))
	}
}

func TestNewRulingRequiresProvenanceAndKnownKind(t *testing.T) {
	if _, err := NewRuling(RulingKind("erased"), "try to erase", testTime, recHuman); !errors.Is(err, ErrInvalidRuling) {
		t.Fatalf("unknown ruling kind err = %v, want ErrInvalidRuling", err)
	}
	if _, err := NewRuling(RulingInvalidEvidence, "", testTime, recHuman); !errors.Is(err, ErrInvalidRuling) {
		t.Fatalf("empty reason err = %v, want ErrInvalidRuling", err)
	}
	if _, err := NewRuling(RulingInvalidEvidence, "r", time.Time{}, recHuman); !errors.Is(err, ErrInvalidRuling) {
		t.Fatalf("zero time err = %v, want ErrInvalidRuling", err)
	}
	// An agent proposal cannot invalidate evidence either.
	if _, err := NewRuling(RulingInvalidEvidence, "agent says so", testTime, recAgent); !errors.Is(err, ErrRecorderCannotRecordFact) {
		t.Fatalf("agent ruling err = %v, want ErrRecorderCannotRecordFact", err)
	}
}

func TestLedgerAppendRulingUnknownFact(t *testing.T) {
	l := NewLedger()
	r, err := NewRuling(RulingInvalidEvidence, "invalid", testTime, recValidator)
	if err != nil {
		t.Fatalf("NewRuling: %v", err)
	}
	if _, err := l.AppendRuling("missing", r); !errors.Is(err, ErrUnknownFact) {
		t.Fatalf("err = %v, want ErrUnknownFact", err)
	}
}

func TestFactsReturnsCopyInAppendOrder(t *testing.T) {
	l := NewLedger()
	if err := l.Record(mustFact(t, "f1", VerdictFail, testScope(nil), recValidator)); err != nil {
		t.Fatalf("Record(f1): %v", err)
	}
	if err := l.Record(mustFact(t, "f2", VerdictPass, testScope(nil), recRun)); err != nil {
		t.Fatalf("Record(f2): %v", err)
	}

	facts := l.Facts()
	if len(facts) != 2 || facts[0].ID() != FactID("f1") || facts[1].ID() != FactID("f2") {
		t.Fatalf("Facts order/length wrong: %v", facts)
	}
	facts[0] = Fact{}
	if again := l.Facts(); again[0].ID() != FactID("f1") {
		t.Fatal("Facts() result is mutable from outside")
	}
}
