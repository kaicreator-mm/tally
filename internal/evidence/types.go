// Package evidence implements the P1 evidence primitives of PRD v1.3 §8.8:
// observed verification results are append-only machine facts whose
// historical validity never changes, while applicability to the current
// desired revision is a separate, derivable question (INV-FACT-01,
// INV-STATE-03).
//
// The product rules encoded here:
//
//   - LLM output is a Proposal and an agent result is never a Project Fact
//     (INV-AUTH-01, INV-AUTH-02): only a fact-capable recorder channel
//     (deterministic validator, executed verification run, human decision,
//     external system) can confirm a machine fact.
//   - Facts are append-only (INV-FACT-01): no domain operation deletes or
//     rewrites a recorded fact. Validity is only ever removed by appending a
//     provenance-backed ruling (confirmed holdout defect, SpecDefect or
//     invalid-evidence ruling, PRD §8.8.1).
//   - Validity and applicability are separate axes (PRD §8.8.2): a fact
//     stays valid until ruled invalid, and it only ever gates the exact
//     project/task/acceptance/verification-revision/ChangeSet combination it
//     was produced against.
//
// Standard library only, plus tally/internal/core shared identity types.
package evidence

import (
	"errors"
	"fmt"
	"time"
)

// Typed references. The rich domain types for task, verification and
// ChangeSet identities live in their own bounded packages; evidence needs
// only stable opaque identities to scope and trace facts.

// FactID identifies one immutable evidence fact.
type FactID string

// TaskID references a task identity authored as a TaskContract.
type TaskID string

// AcceptanceID references one acceptance clause inside a task.
type AcceptanceID string

// VerificationRevision references one frozen VerificationContract revision.
type VerificationRevision string

// ChangeSetID references one immutable ChangeSet snapshot.
type ChangeSetID string

// Verdict is the observed result a recorder confirms for an evidence fact.
type Verdict string

const (
	// VerdictPass records an observed pass of the verified subject.
	VerdictPass Verdict = "pass"
	// VerdictFail records an observed failure of the verified subject.
	VerdictFail Verdict = "fail"
	// VerdictError records an observed execution error (no usable verdict).
	VerdictError Verdict = "error"
)

// RecorderKind classifies the authority channel behind an observation.
// Implementation-authority channels (coding/debug agents) never confirm
// machine facts; they only produce proposals that an independent recorder
// must confirm.
type RecorderKind string

const (
	// RecorderValidator is a deterministic validator executed by Tally.
	RecorderValidator RecorderKind = "validator"
	// RecorderVerificationRun is an executed frozen verification protocol
	// against a concrete ChangeSet (PRD §8.13).
	RecorderVerificationRun RecorderKind = "verification_run"
	// RecorderHuman is a human decision or verification record.
	RecorderHuman RecorderKind = "human"
	// RecorderExternalSystem is a system outside the implementation authority.
	RecorderExternalSystem RecorderKind = "external_system"
	// RecorderAgent is the LLM/agent proposal channel. It cannot record facts.
	RecorderAgent RecorderKind = "implementation_agent"
)

// CanRecordFacts reports whether the channel may confirm machine facts.
// Unknown kinds are denied by default (conservative default).
func (k RecorderKind) CanRecordFacts() bool {
	switch k {
	case RecorderValidator, RecorderVerificationRun, RecorderHuman, RecorderExternalSystem:
		return true
	default:
		return false
	}
}

// Domain errors. Match with errors.Is, never by string.
var (
	// ErrRecorderCannotRecordFact is returned when a non-fact-capable
	// channel (agent proposal, unknown kind) tries to confirm a machine
	// fact or a validity ruling (INV-AUTH-01, INV-AUTH-02).
	ErrRecorderCannotRecordFact = errors.New("evidence: recorder kind cannot confirm machine facts")
	// ErrInvalidFact rejects fact construction with incomplete identity,
	// scope, verdict, time or recorder provenance.
	ErrInvalidFact = errors.New("evidence: invalid fact")
	// ErrInvalidRuling rejects rulings with unknown kind, missing reason,
	// zero time or missing provenance.
	ErrInvalidRuling = errors.New("evidence: invalid ruling")
	// ErrDuplicateFact is returned when recording a FactID twice. Fact
	// identities are unique forever; duplicates are never updates.
	ErrDuplicateFact = errors.New("evidence: fact id already recorded")
	// ErrUnknownFact is returned when appending a ruling to an unknown fact.
	ErrUnknownFact = errors.New("evidence: unknown fact id")
)

// Recorder is the provenance of one observation: the authority channel and
// the concrete actor within it (validator name, verification run reference,
// human identity, external system reference).
type Recorder struct {
	Kind  RecorderKind
	Actor string
}

func (r Recorder) validate() error {
	if !r.Kind.CanRecordFacts() {
		return fmt.Errorf("%w: kind %q", ErrRecorderCannotRecordFact, r.Kind)
	}
	if r.Actor == "" {
		return fmt.Errorf("%w: recorder actor is empty", ErrInvalidFact)
	}
	return nil
}

// Proposal is unconfirmed LLM/agent output: a claimed result plus the
// provenance of its producer. A proposal never becomes a fact by itself;
// Confirm is the only path from proposal to fact and it requires a
// fact-capable recorder that independently observed the result.
type Proposal struct {
	Summary        string
	Scope          Scope
	ClaimedVerdict Verdict
	ProducedBy     RecorderKind // normally RecorderAgent
}

// RulingKind enumerates the formal, provenance-backed rulings that remove a
// fact's validity (PRD §8.8.1). There is no ruling kind that erases a fact;
// facts only gain rulings.
type RulingKind string

const (
	// RulingInvalidHoldoutDefect confirms the held-out evidence itself was
	// defective.
	RulingInvalidHoldoutDefect RulingKind = "holdout_defect"
	// RulingInvalidSpecDefect confirms a SpecDefect invalidated the evidence.
	RulingInvalidSpecDefect RulingKind = "spec_defect"
	// RulingInvalidEvidence confirms the evidence execution was invalid.
	RulingInvalidEvidence RulingKind = "invalid_evidence"
)

// Invalidates reports whether the ruling kind removes validity. Unknown
// kinds never invalidate (conservative default); NewRuling rejects them.
func (k RulingKind) Invalidates() bool {
	switch k {
	case RulingInvalidHoldoutDefect, RulingInvalidSpecDefect, RulingInvalidEvidence:
		return true
	default:
		return false
	}
}

// Ruling is an append-only validity ruling carrying its own provenance. It
// never mutates the observed verdict; it only changes the fact's current
// validity (PRD §8.8.1).
type Ruling struct {
	Kind     RulingKind
	Reason   string
	Recorder Recorder
	RuledAt  time.Time
}

// NewRuling constructs a validity ruling. The ruling channel must itself be
// fact-capable: an agent proposal cannot invalidate evidence. RuledAt is
// normalized to UTC.
func NewRuling(kind RulingKind, reason string, ruledAt time.Time, recorder Recorder) (Ruling, error) {
	if !kind.Invalidates() {
		return Ruling{}, fmt.Errorf("%w: unknown ruling kind %q", ErrInvalidRuling, kind)
	}
	if reason == "" {
		return Ruling{}, fmt.Errorf("%w: reason is empty", ErrInvalidRuling)
	}
	if ruledAt.IsZero() {
		return Ruling{}, fmt.Errorf("%w: ruled-at time is zero", ErrInvalidRuling)
	}
	if err := recorder.validate(); err != nil {
		return Ruling{}, err
	}
	return Ruling{Kind: kind, Reason: reason, Recorder: recorder, RuledAt: ruledAt.UTC()}, nil
}

// Fact is one confirmed machine fact: an observed verdict bound to the exact
// desired context (Scope) it was produced against. Facts are immutable and
// append-only (INV-FACT-01): later facts never rewrite them, and validity is
// only removed by appending a provenance-backed ruling. Construct with
// NewFact or Confirm; the zero value is not usable.
type Fact struct {
	id         FactID
	scope      Scope
	verdict    Verdict
	recorder   Recorder
	recordedAt time.Time
	rulings    []Ruling // append-only, oldest first
}

// NewFact confirms a machine fact observed directly by a fact-capable
// recorder. It is the recorder path for validator, verification-run, human
// and external observations; agent results travel through Proposal and
// Confirm instead. recordedAt is normalized to UTC.
func NewFact(id FactID, scope Scope, verdict Verdict, recordedAt time.Time, recorder Recorder) (Fact, error) {
	f := Fact{
		id:         id,
		scope:      scope,
		verdict:    verdict,
		recorder:   recorder,
		recordedAt: recordedAt.UTC(),
	}
	if err := f.validate(); err != nil {
		return Fact{}, err
	}
	return f, nil
}

// Confirm is the only path from a Proposal to a confirmed machine fact. It
// records the proposal's claimed result as a fact only through a
// fact-capable recorder; the agent proposal channel itself can never
// confirm its own output (INV-AUTH-02).
func Confirm(id FactID, p Proposal, recordedAt time.Time, recorder Recorder) (Fact, error) {
	return NewFact(id, p.Scope, p.ClaimedVerdict, recordedAt, recorder)
}

func (f Fact) validate() error {
	if f.id == "" {
		return fmt.Errorf("%w: fact id is empty", ErrInvalidFact)
	}
	if err := f.scope.validate(); err != nil {
		return err
	}
	switch f.verdict {
	case VerdictPass, VerdictFail, VerdictError:
	default:
		return fmt.Errorf("%w: unknown verdict %q", ErrInvalidFact, f.verdict)
	}
	if f.recordedAt.IsZero() {
		return fmt.Errorf("%w: recorded-at time is zero", ErrInvalidFact)
	}
	return f.recorder.validate()
}

// ID returns the fact identity.
func (f Fact) ID() FactID { return f.id }

// Scope returns the desired context the fact was produced against.
func (f Fact) Scope() Scope { return f.scope }

// Verdict returns the observed verdict.
func (f Fact) Verdict() Verdict { return f.verdict }

// Recorder returns the recording provenance.
func (f Fact) Recorder() Recorder { return f.recorder }

// RecordedAt returns the normalized (UTC) recording time.
func (f Fact) RecordedAt() time.Time { return f.recordedAt }

// Rulings returns a copy of the append-only ruling sequence, oldest first.
// Mutating the returned slice cannot affect the fact.
func (f Fact) Rulings() []Ruling {
	if len(f.rulings) == 0 {
		return nil
	}
	out := make([]Ruling, len(f.rulings))
	copy(out, f.rulings)
	return out
}

// WithRuling returns a new fact view with the ruling appended. The receiver
// and every previously obtained fact value are unchanged; append is the only
// transition a fact ever undergoes.
func (f Fact) WithRuling(r Ruling) Fact {
	rulings := make([]Ruling, len(f.rulings), len(f.rulings)+1)
	copy(rulings, f.rulings)
	f.rulings = append(rulings, r)
	return f
}

// IsValid reports whether the fact still counts as valid evidence. A fact is
// born valid and loses validity only through a confirmed invalidating
// ruling; a later passing fact never restores or erases it (PRD §8.8.1).
func (f Fact) IsValid() bool {
	for _, r := range f.rulings {
		if r.Kind.Invalidates() {
			return false
		}
	}
	return true
}

// Gates reports whether the fact is gate input for the current desired
// context: still valid AND applicable to it. Both axes are required; whether
// an applicable pass satisfies or an applicable fail blocks is consumer
// policy, not part of the fact.
func (f Fact) Gates(current Scope) bool {
	return f.IsValid() && f.scope.ApplicableTo(current)
}

// Ledger is the append-only fact store primitive. Recording appends facts,
// AppendRuling appends validity rulings, and no other operations exist:
// there is no delete, update or rewrite path in the domain API. The database
// layer mirrors this with append-only fact tables guarded by triggers
// (P1-12 migration contract). Not safe for concurrent use.
type Ledger struct {
	facts []Fact
	index map[FactID]int
}

// NewLedger returns an empty append-only fact store.
func NewLedger() *Ledger {
	return &Ledger{index: make(map[FactID]int)}
}

// Record appends a fact. Fact identities are unique forever; recording an
// existing id is an error, never an update.
func (l *Ledger) Record(f Fact) error {
	if err := f.validate(); err != nil {
		return err
	}
	if _, ok := l.index[f.id]; ok {
		return fmt.Errorf("%w: %s", ErrDuplicateFact, f.id)
	}
	l.index[f.id] = len(l.facts)
	l.facts = append(l.facts, f)
	return nil
}

// AppendRuling appends a validity ruling to the stored fact and returns the
// updated fact view. Prior ledger entries are untouched; the ledger only
// ever grows.
func (l *Ledger) AppendRuling(id FactID, r Ruling) (Fact, error) {
	i, ok := l.index[id]
	if !ok {
		return Fact{}, fmt.Errorf("%w: %s", ErrUnknownFact, id)
	}
	updated := l.facts[i].WithRuling(r)
	l.facts[i] = updated
	return updated, nil
}

// Fact returns the current append-only view of one fact.
func (l *Ledger) Fact(id FactID) (Fact, bool) {
	i, ok := l.index[id]
	if !ok {
		return Fact{}, false
	}
	return l.facts[i], true
}

// Facts returns a copy of all recorded facts in append order.
func (l *Ledger) Facts() []Fact {
	out := make([]Fact, len(l.facts))
	copy(out, l.facts)
	return out
}

// FactsGating returns the valid facts applicable to the current desired
// context, in append order. Facts that are still valid but bound to
// superseded revisions or other candidates are excluded here while remaining
// in the ledger as history (PRD §8.8.2).
func (l *Ledger) FactsGating(current Scope) []Fact {
	var out []Fact
	for _, f := range l.facts {
		if f.Gates(current) {
			out = append(out, f)
		}
	}
	return out
}
