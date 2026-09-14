package evidence

import (
	"errors"
	"fmt"
	"time"

	"tally/internal/core"
)

// Exposure facts (PRD §8.6.1).
//
// When held-out content or diagnostics enter an Implementation-Authority
// visible channel, an append-only Exposure Fact is recorded. An exposure
// fact never rewrites the VerificationContract: visibility policy is frozen
// contract state, exposure is an observed fact (§8.4.1, §8.6.1). The
// database layer mirrors this with the append-only evidence_exposure table
// (P1-12 migration contract, migrations/0020).
//
// Compromise semantics (§8.6.1):
//
//   - Content entering a persistent/global channel (knowledge index,
//     persisted validation output) permanently destroys the information
//     independence of that binding — for every candidate, forever.
//   - A locally scoped exposure (one debug disclosure, one session, one
//     owner observation) pollutes at least the corresponding change
//     lineage. Where the fact carries no candidate anchor, the derivation
//     falls back to the conservative time rule: candidates not provably
//     fixed before the exposure are compromised.
//
// An Exposure fact carries no held-out content and no identifiable held-out
// verdict: the optional Reason is traceability provenance only (INV-VER-05).
// Owner disclosure follows §8.6.1: the owner is a decision role, not a
// naturally safe observer — an owner observation that feeds the
// implementation lineage (commit, prompt, revision, model choice) is
// recorded as an exposure fact, and the owner's subjective statement can
// only ever become a risk-acceptance annotation, never an independence
// input.

// ExposureID identifies one immutable exposure fact.
type ExposureID string

// ExposureChannel names the visible channel the held-out content or
// diagnostic entered. The set is open (PRD §8.6.1 lists typical channels
// only; the P1-12 SQL contract deliberately has no enum CHECK). Unknown
// channel names are classified conservatively as persistent by
// ChannelScopeOf.
type ExposureChannel string

const (
	// ChannelDebugDisclosure: held-out content or diagnostics surfaced as
	// implementation-side debug output. Local to the affected lineage.
	ChannelDebugDisclosure ExposureChannel = "debug_disclosure"
	// ChannelSessionLeak: held-out content reached a runtime session /
	// thread history of the implementation authority. Local to the affected
	// lineage.
	ChannelSessionLeak ExposureChannel = "session_leak"
	// ChannelValidationLog: held-out content or an identifiable verdict
	// landed in persisted validation output. Persistent: the output is kept
	// in Tally-controlled storage and stays retrievable beyond the local
	// lineage (PRD §14.3 raw output references).
	ChannelValidationLog ExposureChannel = "validation_log"
	// ChannelKnowledgeIndex: held-out content entered the knowledge index.
	// Persistent/global by definition.
	ChannelKnowledgeIndex ExposureChannel = "knowledge_index"
	// ChannelOwnerDisclosure: the owner observed hidden results. Owner
	// contributions to the implementation lineage are treated as
	// Implementation-Authority exposure (§8.6.1); the observation itself is
	// local until (and unless) it feeds the lineage.
	ChannelOwnerDisclosure ExposureChannel = "owner_disclosure"
	// ChannelManual: an exposure observed and recorded by hand through an
	// untracked channel. The real channel is unknown, so it is classified
	// conservatively as persistent.
	ChannelManual ExposureChannel = "manual"
)

// ChannelScope classifies the reach of an exposure channel (§8.6.1):
// persistent/global channels destroy the binding's information independence
// permanently; lineage-local channels pollute at least the affected change
// lineage.
type ChannelScope string

const (
	// ChannelScopePersistent: content entered a persistent/global channel;
	// the binding's information independence is permanently gone.
	ChannelScopePersistent ChannelScope = "persistent"
	// ChannelScopeLineage: content entered a locally scoped channel; at
	// least the corresponding change lineage is polluted.
	ChannelScopeLineage ChannelScope = "lineage"
)

// ChannelScopeOf classifies a channel. Known local channels are
// lineage-scoped; everything else — including unknown and empty names — is
// persistent (conservative default: an unclassified channel cannot be
// proven local).
func ChannelScopeOf(channel ExposureChannel) ChannelScope {
	switch channel {
	case ChannelDebugDisclosure, ChannelSessionLeak, ChannelOwnerDisclosure:
		return ChannelScopeLineage
	default:
		return ChannelScopePersistent
	}
}

// BindingRef is the identity of one evidence binding for holdout
// consumption and exposure accounting: project, frozen VerificationContract
// revision identity and the binding id inside it. The identity deliberately
// excludes the Task ID: re-keying work under a new task must never reset
// holdout consumption or exposure accounting (the binding ref is the
// consumption key; P1-18 source fact). The revision identity is the
// content-derived frozen contract identity, so a fresh binding on a new
// contract revision (PRD §8.6.3) is a distinct BindingRef while a re-keyed
// task on the same contract is not.
type BindingRef struct {
	Project      core.ProjectID
	Verification VerificationRevision
	BindingID    string
}

// IsZero reports whether the reference is unset.
func (b BindingRef) IsZero() bool { return b == BindingRef{} }

func (b BindingRef) validate() error {
	switch {
	case b.Project == "":
		return fmt.Errorf("%w: binding ref project is empty", ErrInvalidExposure)
	case b.Verification == "":
		return fmt.Errorf("%w: binding ref verification revision is empty", ErrInvalidExposure)
	case b.BindingID == "":
		return fmt.Errorf("%w: binding ref binding id is empty", ErrInvalidExposure)
	default:
		return nil
	}
}

// Domain errors. Match with errors.Is, never by string.
var (
	// ErrInvalidExposure rejects exposure facts with incomplete identity,
	// binding reference, channel or time.
	ErrInvalidExposure = errors.New("evidence: invalid exposure fact")
	// ErrDuplicateExposure is returned when appending an ExposureID twice.
	// Exposure identities are unique forever; duplicates are never updates.
	ErrDuplicateExposure = errors.New("evidence: exposure id already recorded")
)

// ExposureSpec is the construction input for one exposure fact.
type ExposureSpec struct {
	// ID is the unique exposure fact identity.
	ID ExposureID
	// Binding is the binding whose held-out content was exposed.
	Binding BindingRef
	// Acceptance optionally records the acceptance clause the binding
	// belongs to.
	Acceptance AcceptanceID
	// ChangeSet optionally anchors the exposure to the candidate whose
	// lineage it polluted. It may be empty: content exposure can be
	// observed before any concrete candidate exists (P1-12 contract).
	ChangeSet ChangeSetID
	// Channel is the visible channel the content entered.
	Channel ExposureChannel
	// Reason is optional traceability provenance. It must never carry
	// held-out content or an identifiable held-out verdict (INV-VER-05).
	Reason string
	// ExposedAt is when the exposure was observed. Normalized to UTC.
	ExposedAt time.Time
}

// Exposure is one append-only exposure fact: held-out content or
// diagnostics entered an Implementation-Authority visible channel (PRD
// §8.6.1). Exposure facts are never deleted or rewritten; they carry no
// held-out content. Construct with NewExposure; the zero value is not
// usable.
type Exposure struct {
	id         ExposureID
	binding    BindingRef
	acceptance AcceptanceID
	changeSet  ChangeSetID
	channel    ExposureChannel
	reason     string
	exposedAt  time.Time
}

// NewExposure records the construction of one exposure fact. It rejects
// incomplete identity, binding reference, channel or time; unknown channel
// names are accepted (the channel set is open) and classified
// conservatively. exposedAt is normalized to UTC.
func NewExposure(spec ExposureSpec) (Exposure, error) {
	e := Exposure{
		id:         spec.ID,
		binding:    spec.Binding,
		acceptance: spec.Acceptance,
		changeSet:  spec.ChangeSet,
		channel:    spec.Channel,
		reason:     spec.Reason,
		exposedAt:  spec.ExposedAt.UTC(),
	}
	if err := e.validate(); err != nil {
		return Exposure{}, err
	}
	return e, nil
}

func (e Exposure) validate() error {
	if e.id == "" {
		return fmt.Errorf("%w: exposure id is empty", ErrInvalidExposure)
	}
	if err := e.binding.validate(); err != nil {
		return err
	}
	if e.channel == "" {
		return fmt.Errorf("%w: exposure channel is empty", ErrInvalidExposure)
	}
	if e.exposedAt.IsZero() {
		return fmt.Errorf("%w: exposed-at time is zero", ErrInvalidExposure)
	}
	return nil
}

// ID returns the exposure fact identity.
func (e Exposure) ID() ExposureID { return e.id }

// Binding returns the binding whose content was exposed.
func (e Exposure) Binding() BindingRef { return e.binding }

// Acceptance returns the acceptance clause, if recorded.
func (e Exposure) Acceptance() (AcceptanceID, bool) { return e.acceptance, e.acceptance != "" }

// ChangeSet returns the candidate anchor, if recorded. An unanchored fact
// was observed before any concrete candidate (or the anchor is unknown).
func (e Exposure) ChangeSet() (ChangeSetID, bool) { return e.changeSet, e.changeSet != "" }

// Channel returns the channel the content entered.
func (e Exposure) Channel() ExposureChannel { return e.channel }

// Reason returns the traceability provenance, if recorded. It never carries
// held-out content (INV-VER-05).
func (e Exposure) Reason() string { return e.reason }

// ExposedAt returns the normalized (UTC) observation time.
func (e Exposure) ExposedAt() time.Time { return e.exposedAt }

// BindingWide reports whether this fact compromises every candidate of the
// binding: content entered a persistent/global channel, which destroys the
// binding's information independence permanently (§8.6.1).
func (e Exposure) BindingWide() bool {
	return ChannelScopeOf(e.channel) == ChannelScopePersistent
}

// Compromises reports whether this fact compromises a candidate fixed at
// fixedAt whose change lineage is lineage (the candidate itself first,
// nearest ancestor last). Persistent-channel facts compromise every
// candidate. Lineage-scope facts compromise the candidate when their anchor
// is in its lineage or — for unanchored facts — when the candidate is not
// provably fixed before the exposure (conservative time rule; the task's
// conservative time-based mechanics, no causal claims). An unprovable (zero)
// fixation time defaults to the compromised side.
func (e Exposure) Compromises(lineage []ChangeSetID, fixedAt time.Time) bool {
	if e.BindingWide() {
		return true
	}
	if e.changeSet != "" {
		for _, cs := range lineage {
			if cs == e.changeSet {
				return true
			}
		}
		return false
	}
	if fixedAt.IsZero() {
		return true
	}
	return !fixedAt.Before(e.exposedAt)
}

// CompromiseScope derives the strongest channel scope among the recorded
// exposure facts that compromise a candidate fixed at fixedAt whose change
// lineage is lineage (candidate itself first, nearest ancestor last). It is
// the repair-lineage bound of INV-V14-08 / ADR-0001 §5: the worst exposure
// in the lineage bounds the assurance of every descendant — a
// persistent-channel fact anywhere in scope caps the result at persistent,
// otherwise any compromising lineage-local fact yields lineage, and with no
// compromising fact there is no bound (ok=false) and assurance is
// unaffected by exposure. The result is independent of fact order
// (append order is irrelevant to the bound); all bindings are considered —
// a repair loop that evaluates candidate trustworthiness cannot scope by
// binding, and per-binding accounting stays on ExposureLog.Compromised.
// Completing the lineage is caller provenance duty, as in Compromised.
func CompromiseScope(facts []Exposure, lineage []ChangeSetID, fixedAt time.Time) (ChannelScope, bool) {
	worst := ChannelScope("")
	found := false
	for _, e := range facts {
		if !e.Compromises(lineage, fixedAt) {
			continue
		}
		scope := ChannelScopeOf(e.Channel())
		if !found || (worst == ChannelScopeLineage && scope == ChannelScopePersistent) {
			worst = scope
		}
		found = true
	}
	return worst, found
}

// ExposureLog is the append-only exposure fact store primitive. Append is
// the only transition; there is no delete, update or rewrite path
// (INV-FACT-01, PRD §8.6.1). The database layer mirrors this with the
// append-only evidence_exposure table (P1-12). Not safe for concurrent use.
type ExposureLog struct {
	entries []Exposure
	index   map[ExposureID]int
}

// NewExposureLog returns an empty append-only exposure store.
func NewExposureLog() *ExposureLog {
	return &ExposureLog{index: make(map[ExposureID]int)}
}

// Append appends an exposure fact. Exposure identities are unique forever;
// appending an existing id is an error, never an update.
func (l *ExposureLog) Append(e Exposure) error {
	if err := e.validate(); err != nil {
		return err
	}
	if _, ok := l.index[e.id]; ok {
		return fmt.Errorf("%w: %s", ErrDuplicateExposure, e.id)
	}
	l.index[e.id] = len(l.entries)
	l.entries = append(l.entries, e)
	return nil
}

// Exposures returns a copy of all recorded exposure facts in append order.
func (l *ExposureLog) Exposures() []Exposure {
	out := make([]Exposure, len(l.entries))
	copy(out, l.entries)
	return out
}

// BindingExposures returns the exposure facts recorded for one binding, in
// append order.
func (l *ExposureLog) BindingExposures(b BindingRef) []Exposure {
	var out []Exposure
	for _, e := range l.entries {
		if e.binding == b {
			out = append(out, e)
		}
	}
	return out
}

// Compromised reports whether any recorded exposure fact compromises a
// candidate of binding b fixed at fixedAt whose change lineage is lineage
// (candidate first, nearest ancestor last). See Exposure.Compromises for
// the per-fact rule. Completing the lineage is caller provenance duty: an
// incomplete chain can only under-report anchored compromises.
func (l *ExposureLog) Compromised(b BindingRef, lineage []ChangeSetID, fixedAt time.Time) bool {
	for _, e := range l.entries {
		if e.binding == b && e.Compromises(lineage, fixedAt) {
			return true
		}
	}
	return false
}
