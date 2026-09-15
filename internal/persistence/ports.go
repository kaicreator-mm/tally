// Package persistence defines database-independent storage ports for the v1.5
// Engineering Control Plane. Database adapters implement these interfaces;
// Core callers must not depend on adapter-specific SQL types or locking rules.
package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"tally/internal/engineering"
)

var (
	ErrNotFound  = errors.New("persistence: record not found")
	ErrConflict  = errors.New("persistence: record identity conflict")
	ErrImmutable = errors.New("persistence: immutable record cannot be replaced")
)

// Document is the database-neutral persistence envelope used by the fine-
// grained v1.5 ports. Payload contains serialized Tally-owned domain state; it
// is opaque to adapters and therefore does not make JSON query behavior part of
// Core correctness.
type Document struct {
	ProjectID  engineering.ID `json:"project_id"`
	RecordID   engineering.ID `json:"record_id"`
	Payload    json.RawMessage `json:"payload"`
	RecordedAt time.Time       `json:"recorded_at"`
}

// NewDocument creates a defensive database-neutral persistence value.
func NewDocument(projectID, recordID engineering.ID, payload json.RawMessage, recordedAt time.Time) (Document, error) {
	d := Document{
		ProjectID:  projectID,
		RecordID:   recordID,
		Payload:    append(json.RawMessage(nil), payload...),
		RecordedAt: recordedAt.UTC(),
	}
	if err := d.Verify(); err != nil {
		return Document{}, err
	}
	return d, nil
}

// Verify applies only database-independent envelope invariants.
func (d Document) Verify() error {
	if err := d.ProjectID.Validate("persistence project"); err != nil {
		return err
	}
	if err := d.RecordID.Validate("persistence record"); err != nil {
		return err
	}
	if len(d.Payload) == 0 || !json.Valid(d.Payload) {
		return errors.New("persistence: payload must be valid JSON")
	}
	if d.RecordedAt.IsZero() {
		return errors.New("persistence: recorded_at is required")
	}
	return nil
}

// Clone returns a defensive copy suitable for crossing adapter boundaries.
func (d Document) Clone() Document {
	d.Payload = append(json.RawMessage(nil), d.Payload...)
	d.RecordedAt = d.RecordedAt.UTC()
	return d
}

type ProjectStore interface {
	LoadProject(context.Context, engineering.ID) (Document, error)
	SaveProject(context.Context, Document) error
}

type RequirementStore interface {
	LoadRequirements(context.Context, engineering.ID, engineering.ID) (Document, error)
	SaveRequirements(context.Context, Document) error
}

type PlanningStore interface {
	LoadPlanning(context.Context, engineering.ID, engineering.ID) (Document, error)
	SavePlanning(context.Context, Document) error
}

type TaskGraphStore interface {
	LoadTaskGraph(context.Context, engineering.ID, engineering.ID) (Document, error)
	SaveTaskGraph(context.Context, Document) error
}

type TaskContractStore interface {
	LoadTaskContract(context.Context, engineering.ID, engineering.ID) (Document, error)
	SaveTaskContract(context.Context, Document) error
}

type RunStore interface {
	LoadRun(context.Context, engineering.ID, engineering.ID) (Document, error)
	SaveRun(context.Context, Document) error
}

type FactStore interface {
	LoadFact(context.Context, engineering.ID, engineering.ID) (Document, error)
	AppendFact(context.Context, Document) error
}

type EvidenceStore interface {
	LoadEvidence(context.Context, engineering.ID, engineering.ID) (Document, error)
	AppendEvidence(context.Context, Document) error
}

type ActionAuditStore interface {
	LoadAction(context.Context, engineering.ID, engineering.ID) (Document, error)
	AppendAction(context.Context, Document) error
}

type OutcomeStore interface {
	LoadOutcome(context.Context, engineering.ID, engineering.ID) (Document, error)
	AppendOutcome(context.Context, Document) error
}

// Stores exposes the same ports inside and outside a transaction without
// leaking the adapter's transaction type.
type Stores struct {
	Projects     ProjectStore
	Requirements RequirementStore
	Planning     PlanningStore
	TaskGraphs   TaskGraphStore
	TaskContracts TaskContractStore
	Runs         RunStore
	Facts        FactStore
	Evidence     EvidenceStore
	Actions      ActionAuditStore
	Outcomes     OutcomeStore
}

// Verify confirms that a transaction received a complete adapter view.
func (s Stores) Verify() error {
	if s.Projects == nil || s.Requirements == nil || s.Planning == nil || s.TaskGraphs == nil ||
		s.TaskContracts == nil || s.Runs == nil || s.Facts == nil || s.Evidence == nil || s.Actions == nil || s.Outcomes == nil {
		return fmt.Errorf("persistence: transaction stores are incomplete")
	}
	return nil
}

// TransactionBoundary owns adapter transaction mechanics while the caller owns
// what engineering state must change atomically.
type TransactionBoundary interface {
	WithinTransaction(context.Context, func(Stores) error) error
}
