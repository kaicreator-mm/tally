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
	"strconv"
	"strings"

	"tally/internal/controlplane"
	"tally/internal/engineering"
)

type Report struct {
	SourceRoot string           `json:"source_root"`
	Imported   []engineering.ID `json:"imported,omitempty"`
	Skipped    []engineering.ID `json:"skipped_existing,omitempty"`
}

type candidate struct {
	path     string
	revision int64
	snapshot controlplane.ProjectSnapshot
}

// Import validates every source snapshot before mutating the target, selects
// the latest immutable FileRepository generation per project, then imports only
// projects that do not already exist. It is idempotent and never overwrites a
// newer v1.5.1 target snapshot.
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

	latest := make(map[engineering.ID]candidate)
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
		revision, err := sourceRevision(entry.Name(), snapshot)
		if err != nil {
			return report, err
		}
		current, ok := latest[snapshot.ProjectID]
		if !ok || revision > current.revision {
			latest[snapshot.ProjectID] = candidate{path: path, revision: revision, snapshot: snapshot}
		}
	}

	ids := make([]engineering.ID, 0, len(latest))
	for id := range latest {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, id := range ids {
		selected := latest[id]
		if _, err := target.Load(ctx, id); err == nil {
			report.Skipped = append(report.Skipped, id)
			continue
		} else if !errors.Is(err, controlplane.ErrProjectNotFound) {
			return report, fmt.Errorf("v1.5.0 migration: inspect target project %s: %w", id, err)
		}
		if err := target.Save(ctx, selected.snapshot); err != nil {
			return report, fmt.Errorf("v1.5.0 migration: import project %s from %s: %w", id, filepath.Base(selected.path), err)
		}
		report.Imported = append(report.Imported, id)
	}
	return report, nil
}

func sourceRevision(name string, snapshot controlplane.ProjectSnapshot) (int64, error) {
	base := strings.TrimSuffix(name, ".json")
	separator := strings.LastIndexByte(base, '.')
	if separator <= 0 || separator == len(base)-1 {
		return 0, fmt.Errorf("v1.5.0 migration: invalid snapshot filename %q", name)
	}
	project := base[:separator]
	if project != string(snapshot.ProjectID) {
		return 0, fmt.Errorf("v1.5.0 migration: snapshot filename project %q does not match payload %q", project, snapshot.ProjectID)
	}
	revision, err := strconv.ParseInt(base[separator+1:], 10, 64)
	if err != nil || revision <= 0 {
		return 0, fmt.Errorf("v1.5.0 migration: invalid snapshot revision in %q", name)
	}
	if snapshot.UpdatedAt.UnixNano() != revision {
		return 0, fmt.Errorf("v1.5.0 migration: snapshot filename revision does not match payload for %q", name)
	}
	return revision, nil
}
