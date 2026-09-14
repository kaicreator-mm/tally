package risk

import (
	"errors"
	"sort"
)

// ErrMalformedSignal rejects structurally invalid signal input: blank or
// whitespace-padded IDs, missing or blank basis refs. Match with errors.Is,
// never by string.
var ErrMalformedSignal = errors.New("risk: malformed risk signal")

// Result is one deterministic derivation of the risk signal set (PRD §10
// detected_risk_signals as consumed by policy matching and assurance
// selection). It is a derived projection: recomputable from its inputs, with
// no setters and no authoritative score (PRD §22).
type Result struct {
	// Detected holds the inputs whose ID is in the frozen v1.4 enum, in
	// canonical (ID-lexicographic) order. Duplicate inputs for the same ID
	// are merged into one signal whose basis is the sorted, deduplicated
	// union of all input refs.
	Detected []Signal

	// Unknown holds the inputs whose ID is outside the frozen enum,
	// preserved verbatim in canonical order. They are never dropped,
	// renamed or treated as low risk (INV-V14-09 fail-safe).
	Unknown []Signal
}

// All returns every derived signal (detected and unknown) as one canonical
// sequence ordered by ID; it never hides unknown signals. The returned slice
// is a copy.
func (r Result) All() []Signal {
	out := make([]Signal, 0, len(r.Detected)+len(r.Unknown))
	out = append(out, r.Detected...)
	out = append(out, r.Unknown...)
	sortSignals(out)
	return out
}

// HasUnknown reports whether any input signal ID was outside the frozen
// enum. A true result must surface to the caller (attention, upgrade or an
// explicit limited verdict per PRD §10); it must never be read as "no risk".
func (r Result) HasUnknown() bool {
	return len(r.Unknown) > 0
}

// AboveMinimal reports whether the derived set rules out the minimal-risk
// default floor (Decision Lock 21): any detected or unknown signal does.
// This is a conservative predicate over the set, not an assurance decision —
// selecting levels or uplift mappings stays with the policy/assurance
// consumer (Decision Lock 11).
func (r Result) AboveMinimal() bool {
	return len(r.Detected) > 0 || len(r.Unknown) > 0
}

// Derive computes the risk signal set from append-input signal candidates.
//
// Pure function: the input slice and its basis refs are never modified, and
// any input order produces the same Result (PRD §22: derived models must be
// stably recomputable from underlying facts). Empty input derives an empty
// set — a legitimate result, not an error. Structurally invalid candidates
// are construction errors and fail the whole derivation; there is no silent
// partial output.
func Derive(signals []Signal) (Result, error) {
	res := Result{}

	type acc struct {
		known bool
		basis []string
	}
	// Known-ness is a pure function of the ID, so a single accumulator per
	// ID cannot conflict; multiple inputs for one ID merge their basis.
	merged := make(map[SignalID]*acc, len(signals))
	for _, s := range signals {
		if err := s.validate(); err != nil {
			return Result{}, err
		}
		a := merged[s.ID]
		if a == nil {
			a = &acc{known: IsKnown(s.ID)}
			merged[s.ID] = a
		}
		a.basis = append(a.basis, s.Basis...)
	}

	for id, a := range merged {
		sig := Signal{ID: id, Basis: normalizeRefs(a.basis)}
		if a.known {
			res.Detected = append(res.Detected, sig)
		} else {
			res.Unknown = append(res.Unknown, sig)
		}
	}
	sortSignals(res.Detected)
	sortSignals(res.Unknown)
	return res, nil
}

// normalizeRefs returns a sorted, deduplicated copy of refs. nil input stays
// nil; the input slice is never modified.
func normalizeRefs(refs []string) []string {
	if len(refs) == 0 {
		return nil
	}
	sorted := make([]string, len(refs))
	copy(sorted, refs)
	sort.Strings(sorted)
	out := sorted[:0]
	for i, ref := range sorted {
		if i == 0 || ref != sorted[i-1] {
			out = append(out, ref)
		}
	}
	return out
}

// sortSignals orders signals canonically by ID. IDs are unique per set
// (duplicate inputs are merged before sorting), so the order is total.
func sortSignals(s []Signal) {
	sort.Slice(s, func(i, j int) bool { return s[i].ID < s[j].ID })
}
