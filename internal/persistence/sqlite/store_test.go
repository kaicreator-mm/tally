package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"tally/internal/controlplane"
	"tally/internal/engineering"
	"tally/internal/persistence"
)

func TestStoreRoundTripAndRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "tally.db")
	store, err := Open(ctx, path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	at := time.Date(2026, 9, 15, 9, 0, 0, 0, time.UTC)
	doc := mustDocument(t, "project-1", "run-1", `{"status":"running"}`, at)
	if err := store.SaveRun(ctx, doc); err != nil {
		t.Fatalf("SaveRun: %v", err)
	}
	got, err := store.LoadRun(ctx, "project-1", "run-1")
	if err != nil {
		t.Fatalf("LoadRun: %v", err)
	}
	if string(got.Payload) != string(doc.Payload) || !got.RecordedAt.Equal(at) {
		t.Fatalf("round trip mismatch: got=%+v want=%+v", got, doc)
	}

	snapshot := controlplane.ProjectSnapshot{
		SchemaVersion: controlplane.SnapshotSchemaVersion,
		ProjectID:     engineering.ID("project-1"),
		Contracts:     map[engineering.ID]engineering.CompletionContract{},
		UpdatedAt:     at,
	}
	if err := store.Save(ctx, snapshot); err != nil {
		t.Fatalf("Save snapshot: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	store, err = Open(ctx, path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer store.Close()
	gotSnapshot, err := store.Load(ctx, "project-1")
	if err != nil {
		t.Fatalf("Load snapshot after restart: %v", err)
	}
	if gotSnapshot.ProjectID != snapshot.ProjectID || !gotSnapshot.UpdatedAt.Equal(snapshot.UpdatedAt) {
		t.Fatalf("snapshot restart mismatch: got=%+v want=%+v", gotSnapshot, snapshot)
	}
}

func TestAppendOnlyRecordsRejectReplacement(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "tally.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	first := mustDocument(t, "project-1", "fact-1", `{"value":1}`, time.Now().UTC())
	if err := store.AppendFact(ctx, first); err != nil {
		t.Fatalf("AppendFact: %v", err)
	}
	second := mustDocument(t, "project-1", "fact-1", `{"value":2}`, time.Now().UTC().Add(time.Second))
	if err := store.AppendFact(ctx, second); !errors.Is(err, persistence.ErrImmutable) {
		t.Fatalf("duplicate append = %v, want ErrImmutable", err)
	}
	got, err := store.LoadFact(ctx, "project-1", "fact-1")
	if err != nil {
		t.Fatalf("LoadFact: %v", err)
	}
	if string(got.Payload) != `{"value":1}` {
		t.Fatalf("immutable payload changed: %s", got.Payload)
	}
}

func TestTransactionRollbackIsAtomic(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "tally.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	rollback := errors.New("rollback requested")
	doc := mustDocument(t, "project-1", "evidence-1", `{"passed":true}`, time.Now().UTC())
	err = store.WithinTransaction(ctx, func(stores persistence.Stores) error {
		if err := stores.Verify(); err != nil {
			return err
		}
		if err := stores.Evidence.AppendEvidence(ctx, doc); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("WithinTransaction = %v, want rollback sentinel", err)
	}
	if _, err := store.LoadEvidence(ctx, "project-1", "evidence-1"); !errors.Is(err, persistence.ErrNotFound) {
		t.Fatalf("rolled-back evidence load = %v, want ErrNotFound", err)
	}
}

func TestConcurrentAppendUsesBoundedSQLiteWriter(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "tally.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	docs := make([]persistence.Document, 8)
	for i := range docs {
		docs[i] = mustDocument(
			t,
			"project-1",
			engineering.ID(fmt.Sprintf("evidence-%02d", i)),
			`{"ok":true}`,
			time.Now().UTC(),
		)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, len(docs))
	for _, doc := range docs {
		doc := doc
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := store.AppendEvidence(ctx, doc); err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatalf("concurrent append: %v", err)
	}
}

func TestMissingSnapshotMapsToControlPlaneNotFound(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, filepath.Join(t.TempDir(), "tally.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()
	if _, err := store.Load(ctx, "missing-project"); !errors.Is(err, controlplane.ErrProjectNotFound) {
		t.Fatalf("Load missing snapshot = %v, want ErrProjectNotFound", err)
	}
}

func mustDocument(t *testing.T, projectID, recordID engineering.ID, raw string, at time.Time) persistence.Document {
	t.Helper()
	doc, err := persistence.NewDocument(projectID, recordID, json.RawMessage(raw), at)
	if err != nil {
		t.Fatalf("NewDocument: %v", err)
	}
	return doc
}
