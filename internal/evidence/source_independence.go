package evidence

// Effective source-independence derivation (PRD §8.2.2).
//
// An evidence binding only *declares* a desired provenance basis
// (independence_basis.source). The binding that actually counts for
// acceptance is effective_source_independence(authoring, implementation):
// derived relative to the provenance of the actual implementation decision
// lineage behind the ChangeSet under evaluation, not compared by actor
// identity (§8.2.2: actor category ≠ authority independence; Authority model
// §3.1: authority is a property of the activity context, never a permanent
// identity label on a natural person).
//
// The inputs here are deliberately activity-shaped:
//
//   - Was the evidence authored, and was its content changed, inside the
//     current implementation decision lineage (the runs/decisions that
//     produced the current candidate)? That is the contamination question.
//     Identity equality alone answers nothing: a solo developer who authored
//     a test historically and only later implements code does not destroy
//     that test's independence, while a completely different person writing
//     a test while looking at the candidate is still source-dependent.
//   - authoring_mode provenance (INV-VER-01, §8.2.1): pre_commit authoring
//     happened before the authorized run with the candidate absent;
//     post_hoc authoring preserves independence only along the
//     candidate-blind governance-upgrade path.
//   - The declared basis is only a claim: implementation_authority is
//     source-dependent by declaration; an unknown or undeclared basis cannot
//     ground an independence claim. A declared non-implementation basis
//     still loses independence the moment lineage contact is provenanced.
//   - Evidence authored against historical desired revisions is not
//     disqualified by age or revision distance; whether it gates the current
//     revision at all is the separate applicability question
//     (Scope.ApplicableTo, §8.8.2). Only implementation-decision lineage
//     contact is judged here.
//
// Actor names and roles (human owner, coding agent, ...) are recorded for
// traceability and are intentionally never decision inputs: a human role or
// a shared name neither grants nor forfeits independence. The derivation is
// a deterministic function of the provenance inputs: same inputs, same
// answer, for every input combination (no error path; every unknown or
// incomplete provenance conservatively derives to dependent).
//
// The declared basis and authoring mode values mirror the frozen
// VerificationContract vocabulary (internal/verification IndependenceBasis /
// AuthoringMode). They are re-declared here so this package stays free of a
// dependency on the contract package; the string forms are identical, so
// callers convert with a plain string conversion.

// SourceBasis is the declared desired provenance basis of an evidence item
// (independence_basis.source, PRD §8.2.2).
type SourceBasis string

const (
	// BasisVerificationAuthority declares verification-authority authorship.
	BasisVerificationAuthority SourceBasis = "verification_authority"
	// BasisPreExisting declares evidence that predates the current task.
	BasisPreExisting SourceBasis = "pre_existing"
	// BasisHuman declares human-authored evidence.
	BasisHuman SourceBasis = "human"
	// BasisExternalSystem declares evidence from a system outside the
	// implementation authority.
	BasisExternalSystem SourceBasis = "external_system"
	// BasisImplementationAuthority declares implementation-authority
	// authorship. Such evidence is source-dependent by declaration.
	BasisImplementationAuthority SourceBasis = "implementation_authority"
)

// AuthoringMode is the declared authoring provenance of the verification
// authority (INV-VER-01, PRD §8.2.1).
type AuthoringMode string

const (
	// AuthoringPreCommit: defined and frozen before the authorized
	// implementation run started; the authoring context did not contain the
	// candidate implementation.
	AuthoringPreCommit AuthoringMode = "pre_commit"
	// AuthoringPostHoc: defined only after a mid-run governance upgrade,
	// admissible solely as a candidate-blind process with explicit review
	// and a new authorized run (§8.2.1).
	AuthoringPostHoc AuthoringMode = "post_hoc"
)

// Contributor records one actor within an authority context. Name and role
// exist for traceability only; no derivation in this file reads them.
type Contributor struct {
	// Name is the actor identity (person, agent session, system).
	Name string
	// Role is the authority context the actor acted in (e.g. human_owner,
	// coding_agent, verification_planner).
	Role string
}

// EvidenceAuthorship is the authoring provenance of one evidence item
// relative to the current implementation decision lineage.
type EvidenceAuthorship struct {
	// Basis is the declared desired provenance basis.
	Basis SourceBasis
	// AuthoringMode is the declared authoring provenance (INV-VER-01).
	AuthoringMode AuthoringMode
	// CandidateBlind attests, for post_hoc authoring only, that the
	// authoring process could not observe the current candidate
	// (§8.2.1). Ignored for pre_commit authoring, where the candidate is
	// absent by definition.
	CandidateBlind bool
	// AuthoredInLineage records that the evidence was first authored as
	// part of the current implementation decision lineage (created inside
	// the runs/decisions that produced the current candidate).
	AuthoredInLineage bool
	// ChangedInLineage records that pre-existing evidence content was
	// changed as part of the current implementation decision lineage.
	ChangedInLineage bool
	// Authors records who authored the evidence, for traceability. It is
	// deliberately not a decision input: names and roles neither grant nor
	// forfeit independence.
	Authors []Contributor
}

// ImplementationProvenance is the observed provenance of the actual
// implementation authority behind the ChangeSet under evaluation.
type ImplementationProvenance struct {
	// Contributors are the actual contributors of the current candidate,
	// recorded for traceability. Identity is deliberately never compared
	// against evidence author identity. An empty record means the
	// implementation side is unprovenanced, and no independence claim can
	// be established against an unknown implementation.
	Contributors []Contributor
}

// EffectiveSourceIndependence derives PRD §8.2.2
// effective_source_independence(authorship, changeset): whether the
// authoring activity behind one evidence item is independent of the current
// implementation decision lineage under evaluation.
//
// The rules, in deterministic order — any single source-dependent fact
// makes the whole derivation dependent:
//
//  1. An unprovenanced implementation side (no recorded contributors)
//     cannot ground an independence claim (conservative default).
//  2. A declared implementation_authority basis is source-dependent by
//     declaration; an unknown or empty basis is dependent as well.
//  3. The authoring mode must be declared (INV-VER-01). pre_commit
//     authoring preserves independence; post_hoc preserves it only under
//     the candidate-blind attestation; an unknown mode is dependent.
//  4. Evidence authored or changed inside the current implementation
//     decision lineage is source-dependent, regardless of who authored it.
//
// The result is a deterministic function of the provenance inputs: every
// input combination yields an answer, so there is no error path.
func EffectiveSourceIndependence(authorship EvidenceAuthorship, implementation ImplementationProvenance) bool {
	// (1) Unknown implementation provenance: the derivation is relative to
	// the actual implementation contributors; without them there is nothing
	// to be independent of that could be trusted.
	if len(implementation.Contributors) == 0 {
		return false
	}

	// (2) The declared basis is only a claim, but an implementation_authority
	// declaration is binding against the declarer, and an unknown declaration
	// cannot ground independence (conservative default).
	switch authorship.Basis {
	case BasisVerificationAuthority, BasisPreExisting, BasisHuman, BasisExternalSystem:
		// Known non-implementation basis: fall through to activity rules.
	case BasisImplementationAuthority:
		return false
	default:
		return false
	}

	// (3) Declared authoring provenance decides the candidate-exposure
	// question at authoring time.
	independentAtAuthoring := false
	switch authorship.AuthoringMode {
	case AuthoringPreCommit:
		// Authored before the authorized run; candidate absent by
		// definition (§8.2.1).
		independentAtAuthoring = true
	case AuthoringPostHoc:
		// Only the candidate-blind governance-upgrade path preserves
		// independence (§8.2.1); post_hoc with candidate exposure is the
		// textbook dependent case — even for a different person.
		independentAtAuthoring = authorship.CandidateBlind
	default:
		// Undeclared authoring provenance (INV-VER-01 violation):
		// conservative default.
		return false
	}
	if !independentAtAuthoring {
		return false
	}

	// (4) The activity question: contact with the current implementation
	// decision lineage — authoring inside it or changing pre-existing
	// content inside it — makes the evidence source-dependent whoever the
	// actor was. Identity equality or difference never enters here.
	return !authorship.AuthoredInLineage && !authorship.ChangedInLineage
}
