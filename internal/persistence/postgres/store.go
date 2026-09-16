package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"tally/internal/controlplane"
	"tally/internal/engineering"
	"tally/internal/persistence"
)

const (
	kindProject      = "project"
	kindRequirements = "requirements"
	kindPlanning     = "planning"
	kindTaskGraph    = "task_graph"
	kindTaskContract = "task_contract"
	kindRun          = "run"
	kindFact         = "fact"
	kindEvidence     = "evidence"
	kindAction       = "action"
	kindOutcome      = "outcome"
)

type dbtx interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Store is the optional PostgreSQL implementation of the v1.5 persistence
// contract. PostgreSQL-specific behavior stays inside this package.
type Store struct {
	db *sql.DB
	q  dbtx
}

var (
	_ controlplane.Repository         = (*Store)(nil)
	_ persistence.ProjectStore        = (*Store)(nil)
	_ persistence.RequirementStore    = (*Store)(nil)
	_ persistence.PlanningStore       = (*Store)(nil)
	_ persistence.TaskGraphStore      = (*Store)(nil)
	_ persistence.TaskContractStore   = (*Store)(nil)
	_ persistence.RunStore            = (*Store)(nil)
	_ persistence.FactStore           = (*Store)(nil)
	_ persistence.EvidenceStore       = (*Store)(nil)
	_ persistence.ActionAuditStore    = (*Store)(nil)
	_ persistence.OutcomeStore        = (*Store)(nil)
	_ persistence.TransactionBoundary = (*Store)(nil)
)

// Open connects to an explicitly configured PostgreSQL database and applies
// the idempotent v1.5 adapter schema. The default SQLite profile never calls it.
func Open(ctx context.Context, dsn string) (*Store, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, errors.New("postgres persistence: DSN is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres persistence: open database: %w", err)
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)

	s := &Store{db: db, q: db}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres persistence: ping database: %w", err)
	}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close releases the PostgreSQL pool managed by database/sql.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS tally_v15_documents (
    kind TEXT NOT NULL,
    project_id TEXT NOT NULL,
    record_id TEXT NOT NULL,
    payload JSON NOT NULL,
    recorded_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (kind, project_id, record_id)
)`,
		`CREATE INDEX IF NOT EXISTS tally_v15_documents_project_kind
    ON tally_v15_documents(project_id, kind)`,
		`CREATE TABLE IF NOT EXISTS tally_v15_snapshots (
    project_id TEXT PRIMARY KEY,
    payload JSON NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
)`,
	}
	for _, statement := range statements {
		if _, err := s.q.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("postgres persistence: migrate schema: %w", err)
		}
	}
	return nil
}

// Stores returns the complete v1.5 port view backed by PostgreSQL.
func (s *Store) Stores() persistence.Stores {
	return persistence.Stores{
		Projects:      s,
		Requirements:  s,
		Planning:      s,
		TaskGraphs:    s,
		TaskContracts: s,
		Runs:          s,
		Facts:         s,
		Evidence:      s,
		Actions:       s,
		Outcomes:      s,
	}
}

// WithinTransaction hides *sql.Tx from Core while preserving caller-owned
// transaction boundaries.
func (s *Store) WithinTransaction(ctx context.Context, fn func(persistence.Stores) error) error {
	if fn == nil {
		return errors.New("postgres persistence: transaction callback is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres persistence: begin transaction: %w", err)
	}
	txs := &Store{db: s.db, q: tx}
	if err := fn(txs.Stores()); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("postgres persistence: rollback after %v: %w", err, rollbackErr)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres persistence: commit transaction: %w", err)
	}
	return nil
}

// Load implements the frozen v1.5.0 controlplane.Repository seam.
func (s *Store) Load(ctx context.Context, projectID engineering.ID) (controlplane.ProjectSnapshot, error) {
	if err := projectID.Validate("project"); err != nil {
		return controlplane.ProjectSnapshot{}, err
	}
	var payload []byte
	if err := s.q.QueryRowContext(ctx,
		`SELECT payload FROM tally_v15_snapshots WHERE project_id = $1`, string(projectID),
	).Scan(&payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return controlplane.ProjectSnapshot{}, controlplane.ErrProjectNotFound
		}
		return controlplane.ProjectSnapshot{}, fmt.Errorf("postgres persistence: load project snapshot: %w", err)
	}
	var snapshot controlplane.ProjectSnapshot
	if err := json.Unmarshal(payload, &snapshot); err != nil {
		return controlplane.ProjectSnapshot{}, fmt.Errorf("postgres persistence: decode project snapshot: %w", err)
	}
	if snapshot.ProjectID != projectID {
		return controlplane.ProjectSnapshot{}, errors.New("postgres persistence: snapshot project identity mismatch")
	}
	if err := controlplane.VerifySnapshot(snapshot); err != nil {
		return controlplane.ProjectSnapshot{}, fmt.Errorf("postgres persistence: invalid project snapshot: %w", err)
	}
	return snapshot, nil
}

// Save implements the frozen v1.5.0 controlplane.Repository seam.
func (s *Store) Save(ctx context.Context, snapshot controlplane.ProjectSnapshot) error {
	if err := controlplane.VerifySnapshot(snapshot); err != nil {
		return err
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("postgres persistence: encode project snapshot: %w", err)
	}
	_, err = s.q.ExecContext(ctx, `
INSERT INTO tally_v15_snapshots(project_id, payload, updated_at)
VALUES ($1, $2::json, $3)
ON CONFLICT(project_id) DO UPDATE SET
    payload = EXCLUDED.payload,
    updated_at = EXCLUDED.updated_at
`, string(snapshot.ProjectID), string(payload), snapshot.UpdatedAt.UTC())
	if err != nil {
		return fmt.Errorf("postgres persistence: save project snapshot: %w", err)
	}
	return nil
}

func (s *Store) saveDocument(ctx context.Context, kind string, d persistence.Document) error {
	if err := d.Verify(); err != nil {
		return err
	}
	_, err := s.q.ExecContext(ctx, `
INSERT INTO tally_v15_documents(kind, project_id, record_id, payload, recorded_at)
VALUES ($1, $2, $3, $4::json, $5)
ON CONFLICT(kind, project_id, record_id) DO UPDATE SET
    payload = EXCLUDED.payload,
    recorded_at = EXCLUDED.recorded_at
`, kind, string(d.ProjectID), string(d.RecordID), string(d.Payload), d.RecordedAt.UTC())
	if err != nil {
		return fmt.Errorf("postgres persistence: save %s: %w", kind, err)
	}
	return nil
}

func (s *Store) appendDocument(ctx context.Context, kind string, d persistence.Document) error {
	if err := d.Verify(); err != nil {
		return err
	}
	result, err := s.q.ExecContext(ctx, `
INSERT INTO tally_v15_documents(kind, project_id, record_id, payload, recorded_at)
VALUES ($1, $2, $3, $4::json, $5)
ON CONFLICT(kind, project_id, record_id) DO NOTHING
`, kind, string(d.ProjectID), string(d.RecordID), string(d.Payload), d.RecordedAt.UTC())
	if err != nil {
		return fmt.Errorf("postgres persistence: append %s: %w", kind, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres persistence: inspect append %s result: %w", kind, err)
	}
	if rows != 1 {
		return fmt.Errorf("%w: %s %s", persistence.ErrImmutable, kind, d.RecordID)
	}
	return nil
}

func (s *Store) loadDocument(ctx context.Context, kind string, projectID, recordID engineering.ID) (persistence.Document, error) {
	if err := projectID.Validate("persistence project"); err != nil {
		return persistence.Document{}, err
	}
	if err := recordID.Validate("persistence record"); err != nil {
		return persistence.Document{}, err
	}
	var payload []byte
	var recordedAt time.Time
	if err := s.q.QueryRowContext(ctx, `
SELECT payload, recorded_at
FROM tally_v15_documents
WHERE kind = $1 AND project_id = $2 AND record_id = $3
`, kind, string(projectID), string(recordID)).Scan(&payload, &recordedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return persistence.Document{}, persistence.ErrNotFound
		}
		return persistence.Document{}, fmt.Errorf("postgres persistence: load %s: %w", kind, err)
	}
	return persistence.NewDocument(projectID, recordID, json.RawMessage(payload), recordedAt.UTC())
}

func (s *Store) LoadProject(ctx context.Context, projectID engineering.ID) (persistence.Document, error) {
	return s.loadDocument(ctx, kindProject, projectID, projectID)
}

func (s *Store) SaveProject(ctx context.Context, d persistence.Document) error {
	if d.ProjectID != d.RecordID {
		return errors.New("postgres persistence: project record id must equal project id")
	}
	return s.saveDocument(ctx, kindProject, d)
}

func (s *Store) LoadRequirements(ctx context.Context, projectID, recordID engineering.ID) (persistence.Document, error) {
	return s.loadDocument(ctx, kindRequirements, projectID, recordID)
}

func (s *Store) SaveRequirements(ctx context.Context, d persistence.Document) error {
	return s.saveDocument(ctx, kindRequirements, d)
}

func (s *Store) LoadPlanning(ctx context.Context, projectID, recordID engineering.ID) (persistence.Document, error) {
	return s.loadDocument(ctx, kindPlanning, projectID, recordID)
}

func (s *Store) SavePlanning(ctx context.Context, d persistence.Document) error {
	return s.saveDocument(ctx, kindPlanning, d)
}

func (s *Store) LoadTaskGraph(ctx context.Context, projectID, recordID engineering.ID) (persistence.Document, error) {
	return s.loadDocument(ctx, kindTaskGraph, projectID, recordID)
}

func (s *Store) SaveTaskGraph(ctx context.Context, d persistence.Document) error {
	return s.saveDocument(ctx, kindTaskGraph, d)
}

func (s *Store) LoadTaskContract(ctx context.Context, projectID, recordID engineering.ID) (persistence.Document, error) {
	return s.loadDocument(ctx, kindTaskContract, projectID, recordID)
}

func (s *Store) SaveTaskContract(ctx context.Context, d persistence.Document) error {
	return s.saveDocument(ctx, kindTaskContract, d)
}

func (s *Store) LoadRun(ctx context.Context, projectID, recordID engineering.ID) (persistence.Document, error) {
	return s.loadDocument(ctx, kindRun, projectID, recordID)
}

func (s *Store) SaveRun(ctx context.Context, d persistence.Document) error {
	return s.saveDocument(ctx, kindRun, d)
}

func (s *Store) LoadFact(ctx context.Context, projectID, recordID engineering.ID) (persistence.Document, error) {
	return s.loadDocument(ctx, kindFact, projectID, recordID)
}

func (s *Store) AppendFact(ctx context.Context, d persistence.Document) error {
	return s.appendDocument(ctx, kindFact, d)
}

func (s *Store) LoadEvidence(ctx context.Context, projectID, recordID engineering.ID) (persistence.Document, error) {
	return s.loadDocument(ctx, kindEvidence, projectID, recordID)
}

func (s *Store) AppendEvidence(ctx context.Context, d persistence.Document) error {
	return s.appendDocument(ctx, kindEvidence, d)
}

func (s *Store) LoadAction(ctx context.Context, projectID, recordID engineering.ID) (persistence.Document, error) {
	return s.loadDocument(ctx, kindAction, projectID, recordID)
}

func (s *Store) AppendAction(ctx context.Context, d persistence.Document) error {
	return s.appendDocument(ctx, kindAction, d)
}

func (s *Store) LoadOutcome(ctx context.Context, projectID, recordID engineering.ID) (persistence.Document, error) {
	return s.loadDocument(ctx, kindOutcome, projectID, recordID)
}

func (s *Store) AppendOutcome(ctx context.Context, d persistence.Document) error {
	return s.appendDocument(ctx, kindOutcome, d)
}
