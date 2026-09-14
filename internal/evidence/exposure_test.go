package evidence

import (
	"errors"
	"testing"
	"time"
)

// —— exposure fact construction ——

func expBinding(rev VerificationRevision) BindingRef {
	return BindingRef{Project: "proj-1", Verification: rev, BindingID: "EV-AC02-H1"}
}

func mustExposure(t *testing.T, spec ExposureSpec) Exposure {
	t.Helper()
	e, err := NewExposure(spec)
	if err != nil {
		t.Fatalf("NewExposure(%s): unexpected error: %v", spec.ID, err)
	}
	return e
}

func TestNewExposureAcceptsAndNormalizes(t *testing.T) {
	local := time.Date(2026, 9, 6, 15, 0, 0, 0, time.FixedZone("CET", 2*60*60))
	e := mustExposure(t, ExposureSpec{
		ID:         "exp-1",
		Binding:    expBinding("verif-r1"),
		Acceptance: "AC-2",
		ChangeSet:  "cs-1",
		Channel:    ChannelDebugDisclosure,
		Reason:     "debug log cited hidden assertion name",
		ExposedAt:  local,
	})

	if e.ExposedAt() != local.UTC() {
		t.Fatal("exposed-at must be normalized to UTC")
	}
	if acc, ok := e.Acceptance(); !ok || acc != AcceptanceID("AC-2") {
		t.Fatalf("Acceptance() = %q, %v", acc, ok)
	}
	if cs, ok := e.ChangeSet(); !ok || cs != ChangeSetID("cs-1") {
		t.Fatalf("ChangeSet() = %q, %v", cs, ok)
	}
	if e.Reason() == "" {
		t.Fatal("reason must be preserved for traceability")
	}
}

func TestNewExposureRejectsIncompleteFact(t *testing.T) {
	valid := ExposureSpec{
		ID:        "exp-1",
		Binding:   expBinding("verif-r1"),
		Channel:   ChannelSessionLeak,
		ExposedAt: testTime,
	}
	cases := map[string]func(*ExposureSpec){
		"id":              func(s *ExposureSpec) { s.ID = "" },
		"project":         func(s *ExposureSpec) { s.Binding.Project = "" },
		"revision":        func(s *ExposureSpec) { s.Binding.Verification = "" },
		"binding id":      func(s *ExposureSpec) { s.Binding.BindingID = "" },
		"channel":         func(s *ExposureSpec) { s.Channel = "" },
		"exposed-at time": func(s *ExposureSpec) { s.ExposedAt = time.Time{} },
	}
	for name, breakIt := range cases {
		spec := valid
		breakIt(&spec)
		if _, err := NewExposure(spec); !errors.Is(err, ErrInvalidExposure) {
			t.Fatalf("%s: want ErrInvalidExposure, got %v", name, err)
		}
	}
}

func TestNewExposureAcceptsUnknownChannelAndUnanchoredFact(t *testing.T) {
	// The channel set is open (PRD §8.6.1 lists typical channels only); an
	// exposure observed before any concrete candidate may have no anchor.
	e := mustExposure(t, ExposureSpec{
		ID:        "exp-open",
		Binding:   expBinding("verif-r1"),
		Channel:   ExposureChannel("some_new_channel"),
		ExposedAt: testTime,
	})
	if cs, ok := e.ChangeSet(); ok {
		t.Fatalf("unanchored fact must have no changeset, got %q", cs)
	}
}

// —— channel scope classification (§8.6.1 persistent vs lineage-local) ——

func TestChannelScopeOfClassification(t *testing.T) {
	for ch, wantLocal := range map[ExposureChannel]bool{
		ChannelDebugDisclosure:       true,
		ChannelSessionLeak:           true,
		ChannelOwnerDisclosure:       true,
		ChannelValidationLog:         false,
		ChannelKnowledgeIndex:        false,
		ChannelManual:                false,
		ExposureChannel("brand_new"): false,
		ExposureChannel(""):          false,
	} {
		got := ChannelScopeOf(ch) == ChannelScopeLineage
		if got != wantLocal {
			t.Fatalf("ChannelScopeOf(%q) local = %v, want %v", ch, got, wantLocal)
		}
	}
}

// —— append-only exposure log ——

func TestExposureLogAppendOnly(t *testing.T) {
	l := NewExposureLog()
	e1 := mustExposure(t, ExposureSpec{ID: "exp-1", Binding: expBinding("verif-r1"), Channel: ChannelSessionLeak, ExposedAt: testTime})
	e2 := mustExposure(t, ExposureSpec{ID: "exp-2", Binding: expBinding("verif-r1"), Channel: ChannelKnowledgeIndex, ExposedAt: testTime.Add(time.Minute)})

	if err := l.Append(e1); err != nil {
		t.Fatalf("Append(exp-1): %v", err)
	}
	if err := l.Append(e2); err != nil {
		t.Fatalf("Append(exp-2): %v", err)
	}
	if err := l.Append(e1); !errors.Is(err, ErrDuplicateExposure) {
		t.Fatalf("duplicate append must be ErrDuplicateExposure, got %v", err)
	}
	// A zero-value exposure is rejected, never silently appended.
	if err := l.Append(Exposure{}); !errors.Is(err, ErrInvalidExposure) {
		t.Fatalf("zero-value append must be ErrInvalidExposure, got %v", err)
	}

	all := l.Exposures()
	if len(all) != 2 || all[0].ID() != ExposureID("exp-1") || all[1].ID() != ExposureID("exp-2") {
		t.Fatalf("Exposures() = %v", all)
	}
	// Mutating the returned copy cannot touch the log.
	all[0] = Exposure{}
	if got := l.Exposures()[0].ID(); got != ExposureID("exp-1") {
		t.Fatalf("Exposures must return a copy, first id now %q", got)
	}
}

func TestExposureLogBindingExposuresFiltersByBinding(t *testing.T) {
	l := NewExposureLog()
	mine := expBinding("verif-r1")
	other := expBinding("verif-r2")
	for _, e := range []Exposure{
		mustExposure(t, ExposureSpec{ID: "a", Binding: mine, Channel: ChannelSessionLeak, ExposedAt: testTime}),
		mustExposure(t, ExposureSpec{ID: "b", Binding: other, Channel: ChannelSessionLeak, ExposedAt: testTime}),
		mustExposure(t, ExposureSpec{ID: "c", Binding: mine, Channel: ChannelDebugDisclosure, ExposedAt: testTime}),
	} {
		if err := l.Append(e); err != nil {
			t.Fatalf("Append(%s): %v", e.ID(), err)
		}
	}
	got := l.BindingExposures(mine)
	if len(got) != 2 || got[0].ID() != ExposureID("a") || got[1].ID() != ExposureID("c") {
		t.Fatalf("BindingExposures = %v, want a,c", got)
	}
}

// —— compromise semantics (§8.6.1) ——

func TestPersistentChannelCompromisesBindingWide(t *testing.T) {
	// Persistent/global channels destroy information independence for the
	// binding permanently: every candidate is compromised, even one
	// finalized before the exposure and on a different lineage.
	for _, ch := range []ExposureChannel{ChannelKnowledgeIndex, ChannelValidationLog, ChannelManual, ExposureChannel("undocumented")} {
		e := mustExposure(t, ExposureSpec{ID: ExposureID("exp-" + string(ch)), Binding: expBinding("verif-r1"), Channel: ch, ExposedAt: testTime})
		if !e.BindingWide() {
			t.Fatalf("channel %q must be binding-wide", ch)
		}
		if !e.Compromises(nil, testTime.Add(-time.Hour)) {
			t.Fatalf("channel %q must compromise even a candidate fixed before the exposure", ch)
		}
		if !e.Compromises([]ChangeSetID{"cs-other"}, testTime) {
			t.Fatalf("channel %q must compromise unrelated lineages too", ch)
		}
	}
}

func TestLocalExposurePollutesAnchoredLineage(t *testing.T) {
	// Debug disclosure anchored to cs-1 compromises cs-1 and its descendant
	// cs-2 (lineage candidate first), but not a sibling lineage.
	e := mustExposure(t, ExposureSpec{ID: "exp-1", Binding: expBinding("verif-r1"), ChangeSet: "cs-1", Channel: ChannelDebugDisclosure, ExposedAt: testTime})
	if e.BindingWide() {
		t.Fatal("local exposure must not be binding-wide")
	}
	if !e.Compromises([]ChangeSetID{"cs-1"}, testTime) {
		t.Fatal("anchor candidate must be compromised")
	}
	if !e.Compromises([]ChangeSetID{"cs-2", "cs-1"}, testTime) {
		t.Fatal("descendant of the anchor must be compromised")
	}
	if e.Compromises([]ChangeSetID{"cs-9"}, testTime) {
		t.Fatal("unrelated lineage must not be compromised by the anchored local exposure")
	}
}

func TestUnanchoredLocalExposureUsesConservativeTimeRule(t *testing.T) {
	// An exposure observed before any concrete candidate anchors to no
	// lineage; candidates not provably fixed before it are compromised.
	e := mustExposure(t, ExposureSpec{ID: "exp-1", Binding: expBinding("verif-r1"), Channel: ChannelSessionLeak, ExposedAt: testTime})

	if e.Compromises(nil, testTime.Add(-time.Hour)) {
		t.Fatal("candidate provably fixed before the exposure must not be compromised")
	}
	if !e.Compromises(nil, testTime) {
		t.Fatal("candidate fixed exactly at exposure time must default to compromised")
	}
	if !e.Compromises(nil, testTime.Add(time.Hour)) {
		t.Fatal("candidate fixed after the exposure must default to compromised")
	}
	// Zero fixation time cannot prove anything: contaminated side wins.
	if !e.Compromises(nil, time.Time{}) {
		t.Fatal("unprovable fixation time must default to compromised")
	}
}

func TestExposureLogCompromised(t *testing.T) {
	l := NewExposureLog()
	b := expBinding("verif-r1")
	other := expBinding("verif-r2")
	if err := l.Append(mustExposure(t, ExposureSpec{ID: "exp-1", Binding: other, Channel: ChannelKnowledgeIndex, ExposedAt: testTime})); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if l.Compromised(b, []ChangeSetID{"cs-1"}, testTime) {
		t.Fatal("exposure on another binding must not compromise this binding")
	}
	if err := l.Append(mustExposure(t, ExposureSpec{ID: "exp-2", Binding: b, ChangeSet: "cs-0", Channel: ChannelDebugDisclosure, ExposedAt: testTime})); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if !l.Compromised(b, []ChangeSetID{"cs-1", "cs-0"}, testTime) {
		t.Fatal("ancestor-anchored exposure must compromise the lineage")
	}
	if l.Compromised(b, []ChangeSetID{"cs-1"}, testTime) {
		t.Fatal("anchored exposure must not compromise when the anchor is outside the lineage")
	}
}

// —— repair-lineage compromise bound (INV-V14-08 / ADR-0001 §5) ——

func TestCompromiseScopeEmptyWhenNothingCompromises(t *testing.T) {
	b := expBinding("verif-r1")
	facts := []Exposure{
		// Anchored outside the candidate lineage: not compromising.
		mustExposure(t, ExposureSpec{ID: "exp-1", Binding: b, ChangeSet: "cs-other", Channel: ChannelDebugDisclosure, ExposedAt: testTime}),
		// Unanchored, but the candidate is provably fixed before it.
		mustExposure(t, ExposureSpec{ID: "exp-2", Binding: b, Channel: ChannelSessionLeak, ExposedAt: testTime.Add(time.Hour)}),
	}
	if _, ok := CompromiseScope(facts, []ChangeSetID{"cs-1"}, testTime); ok {
		t.Fatal("no compromising fact must leave assurance unbounded by exposure")
	}
	if _, ok := CompromiseScope(nil, nil, testTime); ok {
		t.Fatal("empty fact set must yield no scope")
	}
}

func TestCompromiseScopeLineageLocal(t *testing.T) {
	b := expBinding("verif-r1")
	facts := []Exposure{
		mustExposure(t, ExposureSpec{ID: "exp-1", Binding: b, ChangeSet: "cs-0", Channel: ChannelDebugDisclosure, ExposedAt: testTime}),
	}
	scope, ok := CompromiseScope(facts, []ChangeSetID{"cs-1", "cs-0"}, testTime)
	if !ok || scope != ChannelScopeLineage {
		t.Fatalf("scope = %q, %v; want lineage-bound fact to bind at lineage scope", scope, ok)
	}
}

func TestCompromiseScopePersistentDominates(t *testing.T) {
	b := expBinding("verif-r1")
	local := mustExposure(t, ExposureSpec{ID: "exp-loc", Binding: b, ChangeSet: "cs-0", Channel: ChannelDebugDisclosure, ExposedAt: testTime})
	persist := mustExposure(t, ExposureSpec{ID: "exp-per", Binding: b, Channel: ChannelKnowledgeIndex, ExposedAt: testTime.Add(time.Minute)})
	lineage := []ChangeSetID{"cs-1", "cs-0"}

	// The bound is order-independent: the worst fact caps the result no
	// matter which was appended first.
	for _, facts := range [][]Exposure{{local, persist}, {persist, local}} {
		scope, ok := CompromiseScope(facts, lineage, testTime)
		if !ok || scope != ChannelScopePersistent {
			t.Fatalf("scope = %q, %v; want persistent bound (order-independent)", scope, ok)
		}
	}
}

func TestCompromiseScopeIgnoresOtherBindingsOnlyForScopedCallers(t *testing.T) {
	// CompromiseScope is deliberately binding-agnostic (repair loops cannot
	// scope by binding); per-binding accounting stays on ExposureLog.Compromised.

	other := expBinding("verif-r2")
	facts := []Exposure{
		mustExposure(t, ExposureSpec{ID: "exp-1", Binding: other, Channel: ChannelValidationLog, ExposedAt: testTime}),
	}
	if scope, ok := CompromiseScope(facts, []ChangeSetID{"cs-1"}, testTime); !ok || scope != ChannelScopePersistent {
		t.Fatalf("scope = %q, %v; want facts of any binding to bind the candidate", scope, ok)
	}
}
