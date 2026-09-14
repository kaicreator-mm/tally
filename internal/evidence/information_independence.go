package evidence

import "time"

// Effective information independence derivation (PRD §8.5, §8.6).
//
// Information independence is the held-out dimension of the Acceptance-level
// independence model (§8.5): the evidence could not be iterated against by
// the Implementation Authority, because its content and identifiable
// verdicts never entered an implementation-visible channel and its verdict
// was never usable as a candidate selection oracle. A planned held_out
// binding (§8.5.1, freeze-time check in internal/verification) can lose that
// property at runtime; this file derives the effective property per binding
// and candidate from append-only facts:
//
//   - exposure facts (exposure.go, §8.6.1): content compromise;
//   - held-out verdict records projected from the run/evidence ledger
//     (Fact, Ledger): the selection-oracle clock (§8.6.2);
//   - the candidate's mechanical fixation time and change lineage
//     (immutable ChangeSet snapshot, P1-05);
//   - a mechanically verifiable selection precommit (§8.6.2).
//
// The derivation is a deterministic function of these provenance inputs:
// same inputs, same answer, for every input combination, with conservative
// defaults on every unknown — when uninfluenced cannot be mechanically
// proven, the contaminated side wins (§8.6.2). It has no error path, in the
// same style as EffectiveSourceIndependence.
//
// Two hard boundaries:
//
//   - Policy cannot redefine compromised as independent (§8.10, AC5): the
//     input deliberately carries no policy, waiver or tolerance field. An
//     Independence Waiver is a separate risk-acceptance annotation that can
//     only downgrade a verdict to strict_waived; it can never flip this
//     derivation, and neither can any visibility or evidence-strength
//     reading (§8.5.3: strength is orthogonal to independence).
//   - Holdout consumption is keyed by BindingRef, which excludes the Task
//     ID (AC3): re-keying the same contract under a new task id cannot
//     reset the first-verdict clock or exposure accounting. The Task field
//     on VerdictRecord exists for traceability only and no rule reads it.

// VisibilityPolicy is the binding's frozen visibility contract policy
// (PRD §8.4.1). It mirrors the frozen VerificationContract vocabulary
// (internal/verification VisibilityPolicy) and is re-declared here so this
// package stays free of a dependency on the contract package; the string
// forms are identical, so callers convert with a plain string conversion.
type VisibilityPolicy string

const (
	// VisibilityVisible: the Implementation Authority may read the
	// evidence; it can never provide information independence.
	VisibilityVisible VisibilityPolicy = "visible"
	// VisibilityHeldOut: content and identifiable verdicts are kept out of
	// every Tally-controlled and Runtime-Protocol-observable
	// Implementation-Authority channel (INV-VER-05).
	VisibilityHeldOut VisibilityPolicy = "held_out"
)

// VerdictRecord is one held-out verdict occurrence projected from the
// append-only run/evidence records (verification_run / verification_evidence
// facts, Fact). It is a plain input record for the pure derivation; callers
// project it, including the validity axis (Fact.IsValid) and the observed
// recording time (Fact.RecordedAt).
type VerdictRecord struct {
	// Binding is the binding the verdict was produced against. This is the
	// holdout consumption identity: it deliberately excludes the Task ID,
	// so a new Task ID alone never resets lineage (AC3).
	Binding BindingRef
	// Task records the task the run was keyed under, for traceability
	// only. No derivation rule reads it.
	Task TaskID
	// ChangeSet is the immutable candidate the verdict was bound to
	// (§8.7.1).
	ChangeSet ChangeSetID
	// Verdict is the observed result.
	Verdict Verdict
	// Valid projects the validity axis (Fact.IsValid): a provenance-backed
	// invalidating ruling (§8.8.1) removes the record from coverage.
	Valid bool
	// RecordedAt is the observed recording time; it starts the
	// selection-exposure clock (§8.6.2).
	RecordedAt time.Time
}

// SelectionPrecommit is a mechanically verifiable selection precommit
// (PRD §8.6.2): the only exception that can keep a post-verdict candidate
// selection-exposure-free. Every listed commitment must have been frozen
// before the first held-out verdict on the binding, and the observed
// process must have strictly followed them. A precommit that misses any
// commitment, was fixed at/after the first verdict, or whose adherence was
// not mechanically observed proves nothing and defaults to the contaminated
// side. Adhered must come from mechanical conformance observation (declared
// schedule/count/criteria/retry vs observed process), never from an
// attestation alone.
type SelectionPrecommit struct {
	// FixedAt is when the precommit was frozen. It must precede the first
	// held-out verdict.
	FixedAt time.Time
	// GenerationSchedule: the candidate generation schedule was frozen.
	GenerationSchedule bool
	// CandidateCount: the candidate count was frozen.
	CandidateCount bool
	// VisibleSelectionCriteria: selection criteria were frozen and use
	// visible criteria only.
	VisibleSelectionCriteria bool
	// RetryConditions: retry conditions were frozen.
	RetryConditions bool
	// Adhered: mechanical conformance observation confirmed the process
	// strictly followed the frozen commitments.
	Adhered bool
}

// provenBefore reports whether this is a mechanically valid precommit
// against a first held-out verdict recorded at firstVerdict.
func (p SelectionPrecommit) provenBefore(firstVerdict time.Time) bool {
	if p.FixedAt.IsZero() || !p.FixedAt.Before(firstVerdict) {
		return false
	}
	return p.GenerationSchedule &&
		p.CandidateCount &&
		p.VisibleSelectionCriteria &&
		p.RetryConditions &&
		p.Adhered
}

// firstVerdictTime returns the earliest recorded verdict time for binding b.
// The clock is binding-wide and verdict-record-wide: every recorded
// occurrence counts, regardless of candidate, later validity rulings or the
// task the run was keyed under. That is the conservative reading of the
// §8.6.2 oracle rule — a recorded verdict is an observed information flow,
// and neither a ruling (§8.8.1: validity never erases history) nor a task
// re-key (AC3) can un-observe it.
func firstVerdictTime(records []VerdictRecord, b BindingRef) (time.Time, bool) {
	var first time.Time
	found := false
	for _, v := range records {
		if v.Binding != b {
			continue
		}
		if !found || v.RecordedAt.Before(first) {
			first = v.RecordedAt
			found = true
		}
	}
	return first, found
}

// SelectionExposed implements the §8.6.2 mechanical rule: after the first
// held-out verdict on binding b, any candidate not provably fixed before
// that verdict defaults selection-exposed — the verdict must not be usable
// as a candidate selection oracle. The only exception is a mechanically
// proven precommit frozen before the first verdict and strictly adhered to
// (§8.6.2). With no recorded verdict yet there is nothing to select
// against.
func SelectionExposed(records []VerdictRecord, b BindingRef, candidateFixedAt time.Time, pre SelectionPrecommit) bool {
	first, ok := firstVerdictTime(records, b)
	if !ok {
		return false
	}
	if !candidateFixedAt.IsZero() && candidateFixedAt.Before(first) {
		// Provably fixed before the verdict existed: it could not have been
		// selected, repaired or regenerated using it. This covers the
		// original verdict candidate itself (§8.7.1: the verdict binds an
		// immutable candidate that necessarily predates it) and protocol
		// re-runs of it (§8.7.2).
		return false
	}
	// Unprovable fixation time defaults to the contaminated side (§8.6.2).
	return !pre.provenBefore(first)
}

// InformationIndependenceInput is the provenance input for one binding and
// candidate evaluation.
type InformationIndependenceInput struct {
	// Binding is the binding under evaluation.
	Binding BindingRef
	// Visibility is the binding's frozen visibility policy (§8.4.1).
	Visibility VisibilityPolicy
	// Candidate is the immutable ChangeSet under evaluation.
	Candidate ChangeSetID
	// CandidateFixedAt is when the candidate snapshot was finalized. A zero
	// time means the fixation cannot be proven and defaults to the
	// contaminated side for selection exposure.
	CandidateFixedAt time.Time
	// Lineage is the candidate's change lineage: the candidate itself
	// first, then its ancestors within the same implementation decision,
	// nearest ancestor first. Completing it is caller provenance duty: an
	// incomplete chain can only under-report anchored content exposures.
	Lineage []ChangeSetID
	// Exposures are the recorded exposure facts (all of them; the
	// derivation selects those bound to Binding).
	Exposures []Exposure
	// Verdicts are the projected held-out verdict records (all of them;
	// the derivation selects those bound to Binding).
	Verdicts []VerdictRecord
	// Precommit is the mechanically verifiable selection precommit for
	// this candidate, if one exists. Zero value = none.
	Precommit SelectionPrecommit
}

// EffectiveInformationIndependence derives PRD §8.5/§8.6
// effective information independence for one binding and candidate:
// whether the binding still provides an information boundary for this
// candidate, or whether content exposure or selection exposure has
// compromised it. The result composes with the source dimension (§8.5:
// information independence never remedies source dependence); the
// Acceptance-level coverage decision consumes both, never this alone.
//
// The rules, in deterministic order — every unknown defaults to the
// contaminated side:
//
//  1. Only a held_out binding can be information-independent (§8.4.1,
//     §8.5). A visible or undeclared visibility policy never qualifies.
//  2. Coverage precondition (§8.5.2, validity/applicability scope, AC4):
//     the binding must hold a valid, applicable, usable verdict for this
//     candidate — same BindingRef and ChangeSet, Valid, and not
//     VerdictError. A verdict for another candidate, on a superseded
//     contract revision (a different BindingRef: fresh-binding scope,
//     §8.6.3/§8.8.2), ruled invalid, or without a usable result provides no
//     coverage. Validity ruling out coverage never erases the record: the
//     occurrence still drives the §8.6.2 clock below.
//  3. Content exposure (§8.6.1, AC1): any exposure fact for this binding
//     that is binding-wide (persistent channel) or that compromises this
//     candidate's lineage makes the binding compromised for this
//     candidate. Persistent-channel compromise is permanent; exposure
//     facts for other bindings never leak across.
//  4. Selection exposure (§8.6.2, AC2/AC3): if the candidate is not
//     provably fixed before this binding's first held-out verdict and no
//     mechanically proven precommit exists, the verdict has been available
//     as a selection oracle and the candidate is selection-exposed. The
//     clock is binding-wide, so a new Task ID alone cannot reset it (AC3),
//     and a fresh binding on a new contract revision starts clean (§8.6.3)
//     while still facing every rule above — fresh is not automatically
//     independent.
//
// There is no policy, waiver or tolerance input: a compromised result is a
// fact and cannot be relabeled independent (§8.10, AC5). A waiver can only
// produce a separate strict_waived acceptance annotation downstream.
func EffectiveInformationIndependence(in InformationIndependenceInput) bool {
	// (1) Held-out visibility is the necessary contract precondition.
	if in.Visibility != VisibilityHeldOut {
		return false
	}

	// (2) Coverage precondition: a valid, applicable, usable verdict of this
	// binding for this candidate (§8.5.2; validity/applicability scope).
	covered := false
	for _, v := range in.Verdicts {
		if v.Binding == in.Binding &&
			v.ChangeSet == in.Candidate &&
			v.Valid &&
			v.Verdict != VerdictError {
			covered = true
			break
		}
	}
	if !covered {
		return false
	}

	// (3) Content exposure facts (§8.6.1).
	for _, e := range in.Exposures {
		if e.binding == in.Binding && e.Compromises(in.Lineage, in.CandidateFixedAt) {
			return false
		}
	}

	// (4) Selection exposure (§8.6.2).
	return !SelectionExposed(in.Verdicts, in.Binding, in.CandidateFixedAt, in.Precommit)
}
