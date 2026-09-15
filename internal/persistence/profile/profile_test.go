package profile

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"tally/internal/controlplane"
)

func TestOpenDefaultsToSQLiteWithoutPostgres(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tally.db")
	adapter, name, err := Open(context.Background(), Config{SQLitePath: path})
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()
	if name != SQLite {
		t.Fatalf("profile=%s, want sqlite", name)
	}
	if _, err := adapter.Load(context.Background(), "missing-project"); !errors.Is(err, controlplane.ErrProjectNotFound) {
		t.Fatalf("missing snapshot error=%v", err)
	}
}

func TestPostgresProfileRequiresExplicitDSN(t *testing.T) {
	if _, _, err := Open(context.Background(), Config{Name: Postgres}); err == nil {
		t.Fatal("expected explicit PostgreSQL DSN error")
	}
}

func TestUnknownProfileFailsClosed(t *testing.T) {
	if _, _, err := Open(context.Background(), Config{Name: "mysql"}); err == nil {
		t.Fatal("expected unsupported profile error")
	}
}
