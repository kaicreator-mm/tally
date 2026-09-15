package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"tally/internal/controlplane"
	"tally/internal/engineering"
	"tally/internal/persistence/persistencetest"
)

func TestSharedConformance(t *testing.T) {
	store := openTestStore(t)
	persistencetest.Run(t, store)
}

func TestSnapshotRoundTrip(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	projectID := engineering.ID(fmt.Sprintf("postgres-snapshot-%d", time.Now().UTC().UnixNano()))
	at := time.Date(2026, 9, 15, 10, 30, 0, 0, time.UTC)
	snapshot := controlplane.ProjectSnapshot{
		SchemaVersion: controlplane.SnapshotSchemaVersion,
		ProjectID:     projectID,
		Contracts:     map[engineering.ID]engineering.CompletionContract{},
		UpdatedAt:     at,
	}
	if err := store.Save(ctx, snapshot); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load(ctx, projectID)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.ProjectID != snapshot.ProjectID || !got.UpdatedAt.Equal(snapshot.UpdatedAt) {
		t.Fatalf("snapshot mismatch: got=%+v want=%+v", got, snapshot)
	}
}

func TestMigrationIsIdempotent(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()
	first, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	second, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	if err := second.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestMissingSnapshotMapsToControlPlaneNotFound(t *testing.T) {
	store := openTestStore(t)
	missing := engineering.ID(fmt.Sprintf("postgres-missing-%d", time.Now().UTC().UnixNano()))
	if _, err := store.Load(context.Background(), missing); !errors.Is(err, controlplane.ErrProjectNotFound) {
		t.Fatalf("Load missing = %v, want ErrProjectNotFound", err)
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(context.Background(), testDSN(t))
	if err != nil {
		t.Fatalf("Open PostgreSQL test adapter: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("Close PostgreSQL test adapter: %v", err)
		}
	})
	return store
}

func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TALLY_TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("TALLY_TEST_POSTGRES_URL is not configured; PostgreSQL conformance requires an explicit test database")
	}
	return dsn
}
