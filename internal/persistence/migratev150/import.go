// Package migratev150 imports the v1.5.0 compatibility FileRepository
// snapshots into a v1.5.1 Repository. Source files remain untouched so the
// migration is retryable and rollback-friendly.
package migratev150

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tally/internal/controlplane"
	"tally/internal/engineering"
)

type Report struct {
	SourceRoot string           `json:"source_root"`
	Imported   []engineering.ID `json:"imported,omitempty"`
	Skipped    []engineering.ID `json:"skipped_existing,omitempty"`
}

// Import validates every source snapshot before mutating the target, then
// imports only projects that do not already exist. It is intentionally
// idempotent and never overwrites a newer v1.5.1 target snapshot.
func Import(ctx context.Context, sourceRoot string, target controlplane.Repository) (Report, error) {
	report := Report{SourceRoot: sourceRoot}
	if target == nil {
		return report, errors.New("v1.5.0 migration: target repository is required")
	}
	sourceRoot = strings.TrimSpace(sourceRoot)
	if sourceRoot == "" {
		return report, errors.New("v1.5.0 migration: source root is required")
	}
	report.SourceRoot = sourceRoot

	entries, err := os.ReadDir(sourceRoot)
	if err != nil {
		return report, fmt.Errorf("v1.5.0 migration: read source root: %w", err)
	}
	type candidate struct {
		path     string
		snapshot controlplane.ProjectSnapshot
	}
	var candidates []candidate
	for _, entry := range entries {
		if entry.IsDir() || !entry.Type().IsRegular() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(sourceRoot, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return report, fmt.Errorf("v1.5.0 migration: read %s: %w", entry.Name(), err)
		}
		var snapshot controlplane.ProjectSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			return report, fmt.Errorf("v1.5.0 migration: decode %s: %w", entry.Name(), err)
		}
		if err := controlplane.VerifySnapshot(snapshot); err != nil {
			return report, fmt.Errorf("v1.5.0 migration: invalid %s: %w", entry.Name(), err)
		}
		candidates = append(candidates, candidate{path: path, snapshot: snapshot})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].snapshot.ProjectID < candidates[j].snapshot.ProjectID })

	for _, candidate := range candidates {
		id := candidate.snapshot.ProjectID
		if _, err := target.Load(ctx, id); err == nil {
			report.Skipped = append(report.Skipped, id)
			continue
		} else if !errors.Is(err, controlplane.ErrProjectNotFound) {
			return report, fmt.Errorf("v1.5.0 migration: inspect target project %s: %w", id, err)
		}
		if err := target.Save(ctx, candidate.snapshot); err != nil {
			return report, fmt.Errorf("v1.5.0 migration: import project %s from %s: %w", id, filepath.Base(candidate.path), err)
		}
		report.Imported = append(report.Imported, id)
	}
	return report, nil
}
