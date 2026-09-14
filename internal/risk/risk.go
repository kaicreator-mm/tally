// Package risk implements the PRD v1.4 derived RiskSignals model
// (PRD §10 Policy Match Coverage, §13.3 level selection, §14 VerifiedBasis,
// §22 Derived Model).
//
// Risk is a derived projection, never a persisted business object
// (PRD §22): there is no RiskProfile table, no lifecycle state and no
// mutable score anywhere in this package. The signal set is recomputed
// deterministically from its declared append-input candidates by Derive, so
// every consumer (policy matching, assurance selection) sees the same result
// for the same inputs regardless of input order.
//
// The product rules encoded here:
//
//   - Signal identities are stable machine enums (PRD §10
//     detected_risk_signals): the frozen v1.4 set is explicit and compared
//     by exact ID; nothing matches on display strings.
//   - Every signal carries basis refs (PRD §10 match_basis, §14
//     VerifiedBasis): a signal without at least one source reference is a
//     construction error, not data.
//   - Unknown signals remain visible (INV-V14-09 fail-safe): an input whose
//     ID is outside the frozen enum is never dropped, renamed or coerced to
//     a low-risk interpretation; it is returned verbatim in Result.Unknown
//     and keeps the derived set above the minimal-risk floor.
//   - No risk score exists to become authoritative (PRD §22, §31: no single
//     Trust Score): this package classifies nothing beyond known/unknown and
//     set-empty/non-empty. Mapping signals to assurance levels is a consumer
//     decision (Decision Locks 11 and 21), never made here.
//
// Standard library only; basis refs are opaque strings owned by callers, so
// the package has no internal dependencies.
package risk

import (
	"fmt"
	"strings"
)

// SignalID identifies one kind of risk signal. The constants below are the
// frozen v1.4 enum: stable machine identities that policy triggers and
// assurance mappings may depend on. Extending the set is a task-contract
// level change, not runtime data; IDs observed at runtime that are not in
// the enum stay unknown (and visible) by design.
type SignalID string

const (
	// SignalAuthSymbolChange is the PRD §14 example signal: the task or
	// change touches an authentication/authorization symbol or surface.
	SignalAuthSymbolChange SignalID = "auth_symbol_change"

	// SignalSchemaMigration is the PRD §13.1 migration-correctness /
	// §24.2 Database Migration risk: the task or change touches database
	// schema or ships a migration.
	SignalSchemaMigration SignalID = "schema_migration"

	// SignalProtectedSurfaceChange is the PRD §4.8 / §13.1 risk that makes
	// held-out assurance the risk-adjusted uplift (Decision Lock 21): the
	// task or change touches a surface guarded by protected or held-out
	// verifiers.
	SignalProtectedSurfaceChange SignalID = "protected_surface_change"
)

// knownSignals is the frozen v1.4 enum, in stable (lexicographic) order.
var knownSignals = []SignalID{
	SignalAuthSymbolChange,
	SignalProtectedSurfaceChange,
	SignalSchemaMigration,
}

// KnownSignals returns a copy of the frozen v1.4 signal enum in stable
// (lexicographic) order.
func KnownSignals() []SignalID {
	out := make([]SignalID, len(knownSignals))
	copy(out, knownSignals)
	return out
}

// IsKnown reports whether id is a member of the frozen v1.4 enum. The match
// is exact; there is no normalization, so an unknown ID is never silently
// mapped onto a known one (INV-V14-09 fail-safe).
func IsKnown(id SignalID) bool {
	for _, k := range knownSignals {
		if k == id {
			return true
		}
	}
	return false
}

// Signal is one risk signal projection together with its basis: the inputs
// it was derived from. Signals are plain values; derivation never mutates
// them.
type Signal struct {
	// ID is the signal kind. IDs outside the frozen enum are preserved
	// verbatim as unknown signals (INV-V14-09 fail-safe).
	ID SignalID

	// Basis references the inputs this signal was derived from, in
	// "<kind>:<id>" form (for example "intent:diff-17", "fact:e-42",
	// "contract:TASK-023"). At least one non-blank ref is mandatory: an
	// untraceable signal is a construction error, not data. Derive returns
	// basis refs sorted and deduplicated.
	Basis []string
}

// validate enforces the structural contract. Blank or whitespace-padded IDs
// and blank basis refs are construction bugs and are rejected instead of
// being silently normalized.
func (s Signal) validate() error {
	id := string(s.ID)
	switch {
	case strings.TrimSpace(id) == "":
		return fmt.Errorf("%w: signal id is blank", ErrMalformedSignal)
	case strings.TrimSpace(id) != id:
		return fmt.Errorf("%w: signal id %q has surrounding whitespace", ErrMalformedSignal, s.ID)
	case len(s.Basis) == 0:
		return fmt.Errorf("%w: signal %s carries no basis ref", ErrMalformedSignal, s.ID)
	}
	for _, ref := range s.Basis {
		if strings.TrimSpace(ref) == "" {
			return fmt.Errorf("%w: signal %s carries a blank basis ref", ErrMalformedSignal, s.ID)
		}
	}
	return nil
}
