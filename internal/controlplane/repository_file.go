package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

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
	path, err := r.pathFor(id)
	if err != nil {
		return ProjectSnapshot{}, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ProjectSnapshot{}, ErrProjectNotFound
	}
	if err != nil {
		return ProjectSnapshot{}, fmt.Errorf("controlplane: read snapshot: %w", err)
	}
	var s ProjectSnapshot
	if err := json.Unmarshal(raw, &s); err != nil {
		return ProjectSnapshot{}, fmt.Errorf("controlplane: corrupt snapshot JSON: %w", err)
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
	path, err := r.pathFor(s.ProjectID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(r.Root, 0o700); err != nil {
		return fmt.Errorf("controlplane: create state root: %w", err)
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
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("controlplane: replace snapshot: %w", err)
	}
	remove = false
	if dir, err := os.Open(r.Root); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}

func (r *FileRepository) pathFor(id engineering.ID) (string, error) {
	if err := id.Validate("project"); err != nil {
		return "", err
	}
	if !safeProjectID.MatchString(string(id)) {
		return "", errors.New("controlplane: project id is not safe for filesystem persistence")
	}
	return filepath.Join(r.Root, string(id)+".json"), nil
}
