package workspace

// Prior-candidate materialization (P2-05, ADR-0001 §2): a repair attempt
// continues from the frozen prior candidate state. The continuation base is
// the latest non-superseded candidate's ChangeSet content applied on top of
// its workspace base revision — materialized into a NEW disposable worktree
// owned by the NEW CodegenRun. The prior candidate and its workspace are
// never mutated, and the owner primary worktree is never a materialization
// target: RunSources only ever wraps a manager-created disposable worktree
// and the entry guard re-checks that on every call.
//
// Discipline:
//   - Deterministic: files are written in sorted path order with their exact
//     frozen content (no diff application, no transformation), so equal input
//     always yields an equal workspace tree (AC1).
//   - Validate-then-write: the whole input set is validated before the first
//     write, so an unsafe or ambiguous set materializes nothing (no partial
//     state from bad input).
//   - Contained: candidate paths are repo-relative POSIX paths; absolute
//     paths, traversal elements, backslash/drive forms and git metadata are
//     refused before any write, and existing symlinks are never followed.
//   - Fail-safe: a terminated run's worktree no longer exists, so
//     materializing into it is refused rather than silently re-created.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Sentinel errors for prior-candidate materialization. Match with errors.Is,
// never by string.
var (
	// ErrUnsafeCandidatePath: a candidate file path cannot be materialized
	// safely inside the disposable worktree (absolute, traversal, git
	// metadata, or separator ambiguity). The file is never written.
	ErrUnsafeCandidatePath = errors.New("workspace: candidate file path is not safe to materialize")
	// ErrDuplicateCandidateFile: the same path appears twice in one
	// materialization set, so the resulting workspace state would be
	// ambiguous.
	ErrDuplicateCandidateFile = errors.New("workspace: candidate file path appears twice in the set")
	// ErrMaterializeTarget: an existing entry blocks materialization (a
	// non-regular file at the target path, or a symlink / non-directory in
	// the parent chain). Nothing is written through it and nothing is
	// removed to make room for it.
	ErrMaterializeTarget = errors.New("workspace: materialization target is blocked by an existing entry")
)

// CandidateFile is one file of a frozen prior candidate snapshot to be
// materialized into a fresh repair workspace (ADR-0001 §2: repair base =
// latest non-superseded candidate's ChangeSet content on its workspace base
// revision). It is a plain copyable value; the composition boundary typically
// maps it from the candidate snapshot's changed files.
type CandidateFile struct {
	// Path is the repo-relative slash path of the file ("pkg/serve.go").
	Path string
	// Content is the complete post-change file content (the frozen
	// candidate content, written verbatim — never transformed).
	Content string
	// Executable requests the executable file mode where the platform
	// represents one.
	Executable bool
}

// Materialize writes files into the run's disposable source worktree so this
// run's attempt starts from the frozen prior candidate state (P2-05,
// ADR-0001 §2). The worktree itself is created by RunTable.Open at the
// caller-chosen base revision; Materialize layers the prior candidate's
// content on top of it, byte for byte.
//
//   - An empty set is a no-op success (the base checkout is the full state).
//   - The whole set is validated (paths, duplicates) before the first write;
//     an invalid set materializes nothing.
//   - An existing regular file at a target path is replaced with the frozen
//     content; any other existing entry (directory, symlink, ...) at a
//     target or in its parent chain is refused (ErrMaterializeTarget) rather
//     than followed or removed.
//   - A materialization failure leaves the worktree partially written: the
//     worktree stays disposable, the attempt fails, and the caller surfaces
//     disposal through Terminate — a leaked writable workspace is a
//     reportable outcome, never a silent one.
func (r *RunSources) Materialize(files []CandidateFile) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.terminated {
		return fmt.Errorf("%w: %s", ErrRunTerminated, r.run)
	}
	// Defense in depth: the handle always wraps a manager-created
	// disposable worktree outside the owner checkout, but materialization
	// writes files — re-check the guard at the write boundary so the owner
	// primary worktree can never become a writable repair workspace.
	if err := guardDisposable(r.wt.Path, r.m.repoDir); err != nil {
		return err
	}
	plan, err := materializePlan(files, r.wt.Path)
	if err != nil {
		return err
	}
	for _, t := range plan {
		if err := writeCandidateFile(r.wt.Path, t); err != nil {
			return err
		}
	}
	return nil
}

// candidateTarget is one validated materialization entry: the frozen file
// plus its resolved absolute path inside the disposable worktree.
type candidateTarget struct {
	file CandidateFile
	abs  string
}

// materializePlan validates the whole input set and resolves it to absolute
// targets in deterministic (sorted path) order. Nothing is written until the
// complete plan is accepted, so an invalid set has no filesystem effect.
func materializePlan(files []CandidateFile, root string) ([]candidateTarget, error) {
	sorted := make([]CandidateFile, len(files))
	copy(sorted, files)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Path < sorted[j].Path })

	plan := make([]candidateTarget, 0, len(sorted))
	for i, f := range sorted {
		if i > 0 && sorted[i-1].Path == f.Path {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateCandidateFile, f.Path)
		}
		if err := validateCandidatePath(f.Path); err != nil {
			return nil, err
		}
		abs := filepath.Join(root, filepath.FromSlash(f.Path))
		// Belt and braces: validation already excludes traversal, but the
		// resolved target must still name a path strictly inside the
		// disposable worktree.
		if !insidePath(root, abs) {
			return nil, fmt.Errorf("%w: %s escapes the disposable worktree", ErrUnsafeCandidatePath, f.Path)
		}
		plan = append(plan, candidateTarget{file: f, abs: abs})
	}
	return plan, nil
}

// validateCandidatePath refuses any path that cannot be materialized safely
// inside the disposable worktree. Candidate paths are repo-relative POSIX
// paths, so absolute forms, backslash separators, drive-like elements,
// traversal elements and git metadata are all refused before any write.
func validateCandidatePath(path string) error {
	switch {
	case strings.TrimSpace(path) == "":
		return fmt.Errorf("%w: path is empty", ErrUnsafeCandidatePath)
	case strings.Contains(path, "\\"):
		return fmt.Errorf("%w: %q contains a backslash separator", ErrUnsafeCandidatePath, path)
	case strings.HasPrefix(path, "/"), filepath.IsAbs(path):
		return fmt.Errorf("%w: %q is absolute", ErrUnsafeCandidatePath, path)
	}
	for _, comp := range strings.Split(path, "/") {
		switch {
		case comp == "", comp == ".", comp == "..":
			return fmt.Errorf("%w: %q has an empty, \".\" or \"..\" path element", ErrUnsafeCandidatePath, path)
		case strings.EqualFold(comp, ".git"):
			return fmt.Errorf("%w: %q touches git metadata", ErrUnsafeCandidatePath, path)
		case strings.Contains(comp, ":"):
			return fmt.Errorf("%w: %q contains a drive-like path element", ErrUnsafeCandidatePath, path)
		}
	}
	return nil
}

// writeCandidateFile materializes one file at its frozen content. The parent
// chain is verified before the write so nothing is ever written through a
// symlink; an existing regular file is replaced, anything else at the target
// is refused.
func writeCandidateFile(root string, t candidateTarget) error {
	if err := ensureCandidateDir(root, filepath.Dir(t.abs)); err != nil {
		return err
	}
	if fi, err := os.Lstat(t.abs); err == nil {
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("%w: %s is not a regular file", ErrMaterializeTarget, t.abs)
		}
		// Replace rather than truncate: the frozen content is the whole
		// file, and recreating avoids writing through any entry that
		// raced under the (regular-file) path.
		if err := os.Remove(t.abs); err != nil {
			return fmt.Errorf("workspace: replace %s: %w", t.abs, err)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("workspace: inspect %s: %w", t.abs, err)
	}
	perm := fs.FileMode(0o644)
	if t.file.Executable {
		perm = 0o755
	}
	if err := os.WriteFile(t.abs, []byte(t.file.Content), perm); err != nil {
		return fmt.Errorf("workspace: write %s: %w", t.abs, err)
	}
	return nil
}

// ensureCandidateDir verifies that the existing prefix of dir consists only
// of real directories (never symlinks or files) and creates the missing
// remainder. Writes must land inside the disposable worktree, never behind a
// redirect.
func ensureCandidateDir(root, dir string) error {
	if samePath(dir, root) {
		return nil
	}
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return fmt.Errorf("workspace: resolve %s: %w", dir, err)
	}
	cur := root
	for _, comp := range strings.Split(filepath.ToSlash(rel), "/") {
		cur = filepath.Join(cur, filepath.FromSlash(comp))
		fi, err := os.Lstat(cur)
		if errors.Is(err, fs.ErrNotExist) {
			break // the remainder is created fresh by MkdirAll below
		}
		if err != nil {
			return fmt.Errorf("workspace: inspect %s: %w", cur, err)
		}
		if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
			return fmt.Errorf("%w: %s is not a real directory", ErrMaterializeTarget, cur)
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("workspace: create %s: %w", dir, err)
	}
	return nil
}
