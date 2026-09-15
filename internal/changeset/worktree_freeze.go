package changeset

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tally/internal/core"
	"tally/internal/runner"
)

var (
	ErrUnsafeWorktreePath = errors.New("changeset: unsafe worktree candidate path")
	ErrCandidateDeletion  = errors.New("changeset: candidate deletion is not representable as full post-change content")
)

// WorktreeFreezeSpec identifies one completed implementation attempt whose
// Tally-owned disposable Git worktree should be frozen into an immutable
// ChangeSet. The owner worktree is never accepted here: callers are expected
// to pass a workspace created by internal/workspace.
type WorktreeFreezeSpec struct {
	Project       core.ProjectID
	BaseRevision  string
	ProducerRun   string
	WorkspacePath string
	CreatedAt     time.Time
}

// FreezeWorktree captures the complete post-change content of every modified
// or untracked file in a disposable Git worktree and returns one immutable
// ChangeSet. No changes means (nil, false, nil): a completed Agent claim alone
// is not a candidate. File deletions remain fail-closed because the current
// ChangeSet model represents changed files by complete post-change content.
func FreezeWorktree(ctx context.Context, spec WorktreeFreezeSpec) (*ChangeSet, bool, error) {
	if strings.TrimSpace(spec.WorkspacePath) == "" {
		return nil, false, errors.New("changeset: workspace path is required")
	}
	if strings.TrimSpace(spec.BaseRevision) == "" {
		return nil, false, ErrNoBaseRevision
	}
	if strings.TrimSpace(spec.ProducerRun) == "" {
		return nil, false, ErrNoProducerRun
	}

	local := runner.LocalRunner{}
	status, err := local.Run(ctx, runner.Command{
		Argv: []string{"git", "-C", spec.WorkspacePath, "status", "--porcelain=v1", "-z", "--no-renames", "--untracked-files=all"},
	})
	if err != nil {
		return nil, false, fmt.Errorf("changeset: inspect worktree status: %w", err)
	}
	if status.ExitCode != 0 {
		return nil, false, fmt.Errorf("changeset: git status failed: %s", strings.TrimSpace(status.Stderr))
	}

	entries, err := parseWorktreeStatus(status.Stdout)
	if err != nil {
		return nil, false, err
	}
	if len(entries) == 0 {
		return nil, false, nil
	}

	files := make([]ChangedFile, 0, len(entries))
	for _, entry := range entries {
		if strings.Contains(entry.code, "D") {
			return nil, false, fmt.Errorf("%w: %s", ErrCandidateDeletion, entry.path)
		}
		if err := validateWorktreeCandidatePath(entry.path); err != nil {
			return nil, false, err
		}
		path := filepath.Join(spec.WorkspacePath, filepath.FromSlash(entry.path))
		info, err := os.Lstat(path)
		if err != nil {
			return nil, false, fmt.Errorf("changeset: inspect candidate file %s: %w", entry.path, err)
		}
		if !info.Mode().IsRegular() {
			return nil, false, fmt.Errorf("%w: %s is not a regular file", ErrUnsafeWorktreePath, entry.path)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, false, fmt.Errorf("changeset: read candidate file %s: %w", entry.path, err)
		}
		files = append(files, ChangedFile{
			Path:       entry.path,
			Content:    string(content),
			Executable: info.Mode()&0o111 != 0,
		})
	}

	head, err := local.Run(ctx, runner.Command{Argv: []string{"git", "-C", spec.WorkspacePath, "rev-parse", "HEAD"}})
	if err != nil {
		return nil, false, fmt.Errorf("changeset: resolve worktree HEAD: %w", err)
	}
	if head.ExitCode != 0 {
		return nil, false, fmt.Errorf("changeset: git rev-parse HEAD failed: %s", strings.TrimSpace(head.Stderr))
	}

	candidate, err := New(Spec{
		Project:           spec.Project,
		BaseRevision:      spec.BaseRevision,
		CandidateRevision: strings.TrimSpace(head.Stdout),
		Files:             files,
		ProducerRun:       spec.ProducerRun,
		CreatedAt:         spec.CreatedAt,
	})
	if err != nil {
		return nil, false, err
	}
	return candidate, true, nil
}

type worktreeStatusEntry struct {
	code string
	path string
}

func parseWorktreeStatus(out string) ([]worktreeStatusEntry, error) {
	var entries []worktreeStatusEntry
	for _, record := range strings.Split(out, "\x00") {
		if record == "" {
			continue
		}
		if len(record) < 4 || record[2] != ' ' {
			return nil, fmt.Errorf("changeset: malformed git status record %q", record)
		}
		entries = append(entries, worktreeStatusEntry{code: record[:2], path: record[3:]})
	}
	return entries, nil
}

func validateWorktreeCandidatePath(path string) error {
	if strings.TrimSpace(path) == "" || strings.Contains(path, "\\") || strings.HasPrefix(path, "/") {
		return fmt.Errorf("%w: %q", ErrUnsafeWorktreePath, path)
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if clean == "." || clean == ".." || filepath.IsAbs(clean) || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: %q", ErrUnsafeWorktreePath, path)
	}
	for _, part := range strings.Split(path, "/") {
		if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") || strings.Contains(part, ":") {
			return fmt.Errorf("%w: %q", ErrUnsafeWorktreePath, path)
		}
	}
	return nil
}
