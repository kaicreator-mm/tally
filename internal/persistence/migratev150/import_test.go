package migratev150

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"tally/internal/controlplane"
	"tally/internal/engineering"
	"tally/internal/persistence/sqlite"
)

func snapshot(id engineering.ID, at time.Time) controlplane.ProjectSnapshot {
	return controlplane.ProjectSnapshot{
		SchemaVersion: controlplane.SnapshotSchemaVersion,
		ProjectID:     id,
		Contracts:     map[engineering.ID]engineering.CompletionContract{},
		UpdatedAt:     at,
	}
}

func TestImportIsIdempotentAndDoesNotOverwriteExistingTarget(t *testing.T) {
	ctx := context.Background()
	source, err := controlplane.NewFileRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	old := snapshot("project-1", time.Unix(100, 0).UTC())
	if err := source.Save(ctx, old); err != nil {
		t.Fatal(err)
	}
	target, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "tally.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()

	report, err := Import(ctx, source.Root, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Imported) != 1 || report.Imported[0] != "project-1" || len(report.Skipped) != 0 {
		t.Fatalf("first report=%+v", report)
	}
	got, err := target.Load(ctx, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if !got.UpdatedAt.Equal(old.UpdatedAt) {
		t.Fatalf("imported updated_at=%s, want %s", got.UpdatedAt, old.UpdatedAt)
	}

	newer := snapshot("project-1", time.Unix(200, 0).UTC())
	if err := target.Save(ctx, newer); err != nil {
		t.Fatal(err)
	}
	report, err = Import(ctx, source.Root, target)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Imported) != 0 || len(report.Skipped) != 1 || report.Skipped[0] != "project-1" {
		t.Fatalf("second report=%+v", report)
	}
	got, err = target.Load(ctx, "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if !got.UpdatedAt.Equal(newer.UpdatedAt) {
		t.Fatalf("migration overwrote existing target: updated_at=%s want=%s", got.UpdatedAt, newer.UpdatedAt)
	}
}

func TestImportValidatesAllSourceSnapshotsBeforeWritingTarget(t *testing.T) {
	ctx := context.Background()
	source, err := controlplane.NewFileRepository(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := source.Save(ctx, snapshot("project-valid", time.Unix(100, 0).UTC())); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source.Root, "zzz-corrupt.json"), []byte(`{"schema_version":1,"project_id":"broken"`), 0o600); err != nil {
		t.Fatal(err)
	}
	target, err := sqlite.Open(ctx, filepath.Join(t.TempDir(), "tally.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()

	if _, err := Import(ctx, source.Root, target); err == nil {
		t.Fatal("expected corrupt source migration to fail")
	}
	if _, err := target.Load(ctx, "project-valid"); !errors.Is(err, controlplane.ErrProjectNotFound) {
		t.Fatalf("target mutated before source validation completed: %v", err)
	}
}
