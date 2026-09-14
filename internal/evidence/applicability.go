package evidence

import (
	"fmt"

	"tally/internal/core"
)

// Scope identifies the exact desired context one evidence fact was produced
// against: the project, the task, the acceptance clause, the frozen
// VerificationContract revision and the immutable ChangeSet candidate
// (PRD §8.8.2, INV-STATE-03: desired intent and observed evidence remain
// separately traceable). All five dimensions are required: P1 facts are
// candidate-bound observations (PRD §8.7.1, §8.13), and a partially
// specified scope could silently gate the wrong revision.
type Scope struct {
	Project      core.ProjectID
	Task         TaskID
	Acceptance   AcceptanceID
	Verification VerificationRevision
	ChangeSet    ChangeSetID
}

// validate rejects partially specified scopes.
func (s Scope) validate() error {
	switch {
	case s.Project == "":
		return fmt.Errorf("%w: scope project is empty", ErrInvalidFact)
	case s.Task == "":
		return fmt.Errorf("%w: scope task is empty", ErrInvalidFact)
	case s.Acceptance == "":
		return fmt.Errorf("%w: scope acceptance is empty", ErrInvalidFact)
	case s.Verification == "":
		return fmt.Errorf("%w: scope verification revision is empty", ErrInvalidFact)
	case s.ChangeSet == "":
		return fmt.Errorf("%w: scope changeset is empty", ErrInvalidFact)
	default:
		return nil
	}
}

// ApplicableTo reports whether this scope gates the given current desired
// context. Applicability requires every dimension to match: a fact only ever
// gates the exact project/task/acceptance/verification-revision/ChangeSet
// combination it was produced against. A superseded requirement or
// acceptance, a new VerificationContract revision, or a new candidate
// ChangeSet therefore makes a fact inapplicable to the current context while
// the fact itself stays valid (PRD §8.8.2). Applicability says nothing about
// validity, and validity says nothing about applicability.
func (s Scope) ApplicableTo(current Scope) bool {
	return s == current
}
