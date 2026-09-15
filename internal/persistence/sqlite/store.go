// Package sqlite implements the v1.5 persistence ports with SQLite. It is the
// default local persistence adapter and intentionally contains all SQLite-
// specific SQL and driver configuration.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

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

// Store is the SQLite implementation of the v1.5 persistence contract.
type Store struct {
	db *sql.DB
	q  dbtx
}

var (
	_ controlplane.Repository          = (*Store)(nil)
	_ persistence.ProjectStore         = (*Store)(nil)
	_ persistence.RequirementStore     = (*Store)(nil)
	_ persistence.PlanningStore        = (*Store)(nil)
	_ persistence.TaskGraphStore       = (*Store)(nil)
	_ persistence.TaskContractStore    = (*Store)(nil)
	_ persistence.RunStore             = (*Store)(nil)
	_ persistence.FactStore            = (*Store)(nil)
	_ persistence.EvidenceStore        = (*Store)(nil)
	_ persistence.ActionAuditStore     = (*Store)(nil)
	_ persistence.OutcomeStore         = (*Store)(nil)
	_ persistence.TransactionBoundary  = (*Store)(nil)
)

// Open creates or opens a SQLite database and applies the v1.5 adapter schema.
func Open(ctx context.Context, path string) (*Store, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("sqlite persistence: database path is required")
	}
	if path != ":memory:" && !strings.HasPrefix(path, "file:") {
		dir := filepath.Dir(path)
		if dir != "." && dir != "" {
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return nil, fmt.Errorf("sqlite persistence: create database directory: %w", err)
			}
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite persistence: open database: %w", err)
	}
	// A single connection gives deterministic writer serialization and ensures
	// connection-local PRAGMAs apply to all operations. Application task
	// parallelism remains independent from database writer concurrency.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	s := &Store{db: db, q: db}
	if err := s.configure(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite persistence: ping database: %w", err)
	}
	return s, nil
}

// Close releases the SQLite database connection.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) configure(ctx context.Context) error {
	for _, statement := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
	} {
		if _, err := s.q.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("sqlite persistence: configure %q: %w", statement, err)
		}
	}
	return nil
}

func (s *Store) migrate(ctx context.Context) error {
	const schema = `
CREATE TABLE IF NOT EXISTS tally_v15_documents (
    kind TEXT NOT NULL,
    project_id TEXT NOT NULL,
    record_id TEXT NOT NULL,
    payload TEXT NOT NULL,
    recorded_at TEXT NOT NULL,
    PRIMARY KEY (kind, project_id, record_id)
) WITHOUT ROWID;
CREATE INDEX IF NOT EXISTS tally_v15_documents_project_kind
    ON tally_v15_documents(project_id, kind);
CREATE TABLE IF NOT EXISTS tally_v15_snapshots (
    project_id TEXT PRIMARY KEY,
    payload TEXT NOT NULL,
    updated_at TEXT NOT NULL
) WITHOUT ROWID;
`
	if _, err := s.q.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("sqlite persistence: migrate schema: %w", err)
	}
	return nil
}

// Stores returns the complete port view backed by this adapter.
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

// WithinTransaction executes one caller-owned atomic mutation without exposing
// *sql.Tx through the persistence contract.
func (s *Store) WithinTransaction(ctx context.Context, fn func(persistence.Stores) error) error {
	if fn == nil {
		return errors.New("sqlite persistence: transaction callback is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlite persistence: begin transaction: %w", err)
	}
	txs := &Store{db: s.db, q: tx}
	if err := fn(txs.Stores()); err != nil {
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("sqlite persistence: rollback after %v: %w", err, rollbackErr)
		}
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlite persistence: commit transaction: %w", err)
	}
	return nil
}

// Load implements controlplane.Repository.
func (s *Store) Load(ctx context.Context, projectID engineering.ID) (controlplane.ProjectSnapshot, error) {
	if err := projectID.Validate("project"); err != nil {
		return controlplane.ProjectSnapshot{}, err
	}
	var payload string
	if err := s.q.QueryRowContext(ctx,
		`SELECT payload FROM tally_v15_snapshots WHERE project_id = ?`, string(projectID),
	).Scan(&payload); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return controlplane.ProjectSnapshot{}, controlplane.ErrProjectNotFound
		}
		return controlplane.ProjectSnapshot{}, fmt.Errorf("sqlite persistence: load project snapshot: %w", err)
	}
	var snapshot controlplane.ProjectSnapshot
	if err := json.Unmarshal([]byte(payload), &snapshot); err != nil {
		return controlplane.ProjectSnapshot{}, fmt.Errorf("sqlite persistence: decode project snapshot: %w", err)
	}
	if snapshot.ProjectID != projectID {
		return controlplane.ProjectSnapshot{}, errors.New("sqlite persistence: snapshot project identity mismatch")
	}
	if err := controlplane.VerifySnapshot(snapshot); err != nil {
		return controlplane.ProjectSnapshot{}, fmt.Errorf("sqlite persistence: invalid project snapshot: %w", err)
	}
	return snapshot, nil
}

// Save implements controlplane.Repository using application-owned timestamps.
func (s *Store) Save(ctx context.Context, snapshot controlplane.ProjectSnapshot) error {
	if err := controlplane.VerifySnapshot(snapshot); err != nil {
		return err
	}
	payload, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("sqlite persistence: encode project snapshot: %w", err)
	}
	_, err = s.q.ExecContext(ctx, `
INSERT INTO tally_v15_snapshots(project_id, payload, updated_at)
VALUES (?, ?, ?)
ON CONFLICT(project_id) DO UPDATE SET
    payload = excluded.payload,
    updated_at = excluded.updated_at
`, string(snapshot.ProjectID), string(payload), snapshot.UpdatedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("sqlite persistence: save project snapshot: %w", err)
	}
	return nil
}

func (s *Store) saveDocument(ctx context.Context, kind string, d persistence.Document) error {
	if err := d.Verify(); err != nil {
		return err
	}
	_, err := s.q.ExecContext(ctx, `
INSERT INTO tally_v15_documents(kind, project_id, record_id, payload, recorded_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(kind, project_id, record_id) DO UPDATE SET
    payload = excluded.payload,
    recorded_at = excluded.recorded_at
`, kind, string(d.ProjectID), string(d.RecordID), string(d.Payload), d.RecordedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("sqlite persistence: save %s: %w", kind, err)
	}
	return nil
}

func (s *Store) appendDocument(ctx context.Context, kind string, d persistence.Document) error {
	if err := d.Verify(); err != nil {
		return err
	}
	result, err := s.q.ExecContext(ctx, `
INSERT OR IGNORE INTO tally_v15_documents(kind, project_id, record_id, payload, recorded_at)
VALUES (?, ?, ?, ?, ?)
`, kind, string(d.ProjectID), string(d.RecordID), string(d.Payload), d.RecordedAt.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("sqlite persistence: append %s: %w", kind, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("sqlite persistence: inspect append %s result: %w", kind, err)
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
	var payload, recordedAt string
	if err := s.q.QueryRowContext(ctx, `
SELECT payload, recorded_at
FROM tally_v15_documents
WHERE kind = ? AND project_id = ? AND record_id = ?
`, kind, string(projectID), string(recordID)).Scan(&payload, &recordedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return persistence.Document{}, persistence.ErrNotFound
		}
		return persistence.Document{}, fmt.Errorf("sqlite persistence: load %s: %w", kind, err)
	}
	at, err := time.Parse(time.RFC3339Nano, recordedAt)
	if err != nil {
		return persistence.Document{}, fmt.Errorf("sqlite persistence: decode %s timestamp: %w", kind, err)
	}
	return persistence.NewDocument(projectID, recordID, json.RawMessage(payload), at)
}

func (s *Store) LoadProject(ctx context.Context, projectID engineering.ID) (persistence.Document, error) {
	return s.loadDocument(ctx, kindProject, projectID, projectID)
}

func (s *Store) SaveProject(ctx context.Context, d persistence.Document) error {
	if d.ProjectID != d.RecordID {
		return errors.New("sqlite persistence: project record id must equal project id")
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
