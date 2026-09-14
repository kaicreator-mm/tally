package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"tally/internal/engineering"
)

var safeProjectID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type FileRepository struct {
	Root string
}

func NewFileRepository(root string) (*FileRepository, error) {
	if root == "" {
		return nil, errors.New("controlplane: state root is required")
	}
	return &FileRepository{Root: root}, nil
}

func (r *FileRepository) Load(ctx context.Context, id engineering.ID) (ProjectSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return ProjectSnapshot{}, err
	}
	prefix, err := r.prefixFor(id)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	matches, err := filepath.Glob(prefix + ".*.json")
	if err != nil {
		return ProjectSnapshot{}, fmt.Errorf("controlplane: scan snapshots: %w", err)
	}
	if len(matches) == 0 {
		return ProjectSnapshot{}, ErrProjectNotFound
	}
	type candidate struct {
		path     string
		revision int64
	}
	candidates := make([]candidate, 0, len(matches))
	for _, path := range matches {
		rev, ok := revisionFromPath(prefix, path)
		if ok {
			candidates = append(candidates, candidate{path: path, revision: rev})
		}
	}
	if len(candidates) == 0 {
		return ProjectSnapshot{}, ErrProjectNotFound
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].revision > candidates[j].revision })
	raw, err := os.ReadFile(candidates[0].path)
	if err != nil {
		return ProjectSnapshot{}, fmt.Errorf("controlplane: read snapshot: %w", err)
	}
	var s ProjectSnapshot
	if err := json.Unmarshal(raw, &s); err != nil {
		return ProjectSnapshot{}, fmt.Errorf("controlplane: corrupt latest snapshot JSON: %w", err)
	}
	if s.UpdatedAt.UnixNano() != candidates[0].revision {
		return ProjectSnapshot{}, errors.New("controlplane: snapshot filename revision does not match payload")
	}
	if err := validateSnapshot(s); err != nil {
		return ProjectSnapshot{}, err
	}
	return s, nil
}

func (r *FileRepository) Save(ctx context.Context, s ProjectSnapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateSnapshot(s); err != nil {
		return err
	}
	prefix, err := r.prefixFor(s.ProjectID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(r.Root, 0o700); err != nil {
		return fmt.Errorf("controlplane: create state root: %w", err)
	}
	revision := s.UpdatedAt.UnixNano()
	if current, err := r.latestRevision(prefix); err != nil {
		return err
	} else if revision <= current {
		// Repository publication owns the persistence revision. If a direct caller
		// reuses an UpdatedAt value, advance by one nanosecond rather than relying
		// on replace-existing rename semantics or silently overwriting newer state.
		revision = current + 1
		s.UpdatedAt = time.Unix(0, revision).UTC()
	}
	final := fmt.Sprintf("%s.%020d.json", prefix, revision)
	if _, err := os.Stat(final); err == nil {
		return errors.New("controlplane: snapshot revision already exists")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("controlplane: stat snapshot: %w", err)
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("controlplane: marshal snapshot: %w", err)
	}
	tmp, err := os.CreateTemp(r.Root, ".snapshot-*.tmp")
	if err != nil {
		return fmt.Errorf("controlplane: create temp snapshot: %w", err)
	}
	tmpName := tmp.Name()
	remove := true
	defer func() {
		_ = tmp.Close()
		if remove {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return fmt.Errorf("controlplane: chmod temp snapshot: %w", err)
	}
	if _, err := tmp.Write(raw); err != nil {
		return fmt.Errorf("controlplane: write temp snapshot: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("controlplane: fsync temp snapshot: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("controlplane: close temp snapshot: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Publish to a revision-unique destination. This preserves atomicity without
	// relying on replace-existing Rename semantics, which differ on Windows.
	if err := os.Rename(tmpName, final); err != nil {
		return fmt.Errorf("controlplane: publish snapshot: %w", err)
	}
	remove = false
	if dir, err := os.Open(r.Root); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	// Cleanup is best-effort and happens only after the new durable snapshot is
	// published. A crash during cleanup can leave older immutable generations,
	// but Load deterministically selects the newest revision.
	r.cleanupOlder(prefix, revision)
	return nil
}

func (r *FileRepository) prefixFor(id engineering.ID) (string, error) {
	if err := id.Validate("project"); err != nil {
		return "", err
	}
	if !safeProjectID.MatchString(string(id)) {
		return "", errors.New("controlplane: project id is not safe for filesystem persistence")
	}
	return filepath.Join(r.Root, string(id)), nil
}

func revisionFromPath(prefix, path string) (int64, bool) {
	base := strings.TrimSuffix(strings.TrimPrefix(path, prefix+"."), ".json")
	if base == path || base == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(base, 10, 64)
	return n, err == nil && n > 0
}

func (r *FileRepository) latestRevision(prefix string) (int64, error) {
	matches, err := filepath.Glob(prefix + ".*.json")
	if err != nil {
		return 0, fmt.Errorf("controlplane: scan snapshot revisions: %w", err)
	}
	var latest int64
	for _, path := range matches {
		if rev, ok := revisionFromPath(prefix, path); ok && rev > latest {
			latest = rev
		}
	}
	return latest, nil
}

func (r *FileRepository) cleanupOlder(prefix string, current int64) {
	matches, _ := filepath.Glob(prefix + ".*.json")
	for _, path := range matches {
		rev, ok := revisionFromPath(prefix, path)
		if ok && rev < current {
			_ = os.Remove(path)
		}
	}
}
