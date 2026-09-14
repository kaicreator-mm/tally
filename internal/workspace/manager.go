// Package workspace owns the disposable Git worktree lifecycle: it creates a
// worktree at an exact base revision for one execution and removes it
// idempotently afterwards — on success, failure or cancellation.
//
// Agent-generated executable changes never run in the owner primary worktree
// (PRD v1.3 INV-EXEC-01, §13 Workspace, §21.5 Failure Diagnosis): the manager
// refuses to place disposable worktrees inside the owner checkout and refuses
// to destroy the owner primary worktree, whatever the caller claims about a
// given path.
package workspace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Sentinel errors. Match with errors.Is, never by string.
var (
	// ErrPrimaryWorktree: the operation would treat the owner primary
	// worktree as disposable, either by creating state inside it or by
	// destroying it.
	ErrPrimaryWorktree = errors.New("workspace: owner primary worktree is not disposable")
	// ErrNotRepository: the given path is not a git repository checkout.
	ErrNotRepository = errors.New("workspace: not a git repository checkout")
	// ErrNotWorktree: the given path is not a checked-out git worktree
	// (no .git entry at all).
	ErrNotWorktree = errors.New("workspace: not a checked-out git worktree")
	// ErrUnknownRevision: the requested base revision does not resolve to
	// a commit in the owner repository.
	ErrUnknownRevision = errors.New("workspace: base revision does not resolve to a commit")
)

// worktreeCleanupTimeout bounds the git calls of a teardown. Teardown must
// complete even when the owning operation was cancelled, so it never
// inherits the caller's cancellation — this timeout is the only bound.
const worktreeCleanupTimeout = 30 * time.Second

// Manager creates and destroys disposable worktrees for one owner repository.
type Manager struct {
	// repoDir is the owner repository checkout. It is the reference point
	// for every git operation and is never used as a disposable target.
	repoDir string
	// root is the Tally-owned directory that holds disposable worktrees,
	// guaranteed to lie outside repoDir.
	root string
	git  *gitClient
}

// New returns a Manager for the repository checked out at repoDir. Disposable
// worktrees are created under root, which must lie outside the owner
// checkout; New fails with ErrPrimaryWorktree otherwise (INV-EXEC-01).
func New(repoDir, root string) (*Manager, error) {
	absRepo, err := filepath.Abs(repoDir)
	if err != nil {
		return nil, fmt.Errorf("workspace: resolve repo dir: %w", err)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("workspace: resolve disposable root: %w", err)
	}
	if _, err := os.Stat(filepath.Join(absRepo, ".git")); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrNotRepository, absRepo)
	}
	if samePath(absRepo, absRoot) || insidePath(absRepo, absRoot) {
		return nil, fmt.Errorf("%w: disposable root %s is inside owner checkout %s",
			ErrPrimaryWorktree, absRoot, absRepo)
	}
	return &Manager{repoDir: absRepo, root: absRoot, git: newGitClient()}, nil
}

// Create checks out a new disposable worktree at exactly baseRevision. The
// revision is resolved to a commit first; the worktree is created detached at
// that exact commit, never at a moving ref.
//
// If creation fails — including when the context is cancelled while git runs
// — any partially created state is removed before the error is returned, so a
// failed Create never leaks a worktree.
func (m *Manager) Create(ctx context.Context, baseRevision string) (*Worktree, error) {
	commit, err := m.git.resolveRevision(ctx, m.repoDir, baseRevision)
	if err != nil {
		return nil, err
	}
	path, err := m.reservePath(commit)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(m.root, 0o755); err != nil {
		return nil, fmt.Errorf("workspace: create disposable root %s: %w", m.root, err)
	}
	if err := m.git.worktreeAdd(ctx, m.repoDir, path, commit); err != nil {
		m.abandon(ctx, path)
		return nil, err
	}
	return &Worktree{Path: path, BaseCommit: commit, repoDir: m.repoDir, git: m.git}, nil
}

// Cleanup destroys the worktree previously returned by Create.
//
// Cleanup is idempotent: repeated and concurrent calls converge to the same
// state and a call on an already-removed worktree succeeds. It is equally
// valid after success, after failure (including a worktree directory that
// vanished underneath the manager, leaving stale git metadata) and after
// cancellation: teardown deliberately does not inherit the caller's
// cancellation, so a cancelled operation can always be torn down; internal
// steps are bounded by worktreeCleanupTimeout instead.
//
// Cleanup refuses to destroy the owner primary worktree: if wt.Path is the
// manager's owner checkout or the primary worktree of its repository, it
// returns ErrPrimaryWorktree and touches nothing.
func (m *Manager) Cleanup(ctx context.Context, wt *Worktree) error {
	if wt == nil {
		return nil
	}
	if err := guardDisposable(wt.Path, m.repoDir); err != nil {
		return err
	}

	wt.mu.Lock()
	defer wt.mu.Unlock()
	if wt.cleaned {
		return nil
	}
	if err := wt.remove(ctx); err != nil {
		return err
	}
	wt.cleaned = true
	return nil
}

// abandon best-effort removes a partially created worktree after a failed
// Create. Errors are swallowed: the original creation error is what the
// caller must see.
func (m *Manager) abandon(ctx context.Context, path string) {
	_ = (&Worktree{Path: path, repoDir: m.repoDir, git: m.git}).remove(ctx)
}

// Worktree is one disposable checkout owned by a Manager.
type Worktree struct {
	// Path is the absolute directory of the disposable checkout.
	Path string
	// BaseCommit is the exact commit the checkout was created at.
	BaseCommit string

	repoDir string
	git     *gitClient

	mu      sync.Mutex
	cleaned bool
}

// remove tears the worktree down. Steps are detached from the caller's
// cancellation (bounded by worktreeCleanupTimeout) so teardown completes
// after success, failure or cancellation alike.
//
// The directory is removed via `git worktree remove --force`; if git reports
// failure the observed state is verified (worktree list plus directory
// existence) rather than trusting stderr text, so an already-vanished
// worktree is not an error. Any leftover directory is deleted directly and
// stale metadata is pruned.
func (wt *Worktree) remove(ctx context.Context) error {
	hctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), worktreeCleanupTimeout)
	defer cancel()

	if pathExists(wt.Path) {
		err := wt.git.worktreeRemove(hctx, wt.repoDir, wt.Path)
		if err != nil {
			entries, listErr := wt.git.worktreeList(hctx, wt.repoDir)
			if listErr == nil && worktreeRegistered(entries, wt.Path) && pathExists(wt.Path) {
				return err
			}
		}
		if err := os.RemoveAll(wt.Path); err != nil {
			return fmt.Errorf("workspace: remove worktree directory %s: %w", wt.Path, err)
		}
	}
	if err := wt.git.worktreePrune(hctx, wt.repoDir); err != nil {
		return fmt.Errorf("workspace: prune worktree metadata for %s: %w", wt.Path, err)
	}
	return nil
}

// guardDisposable refuses to treat path as disposable when it is the owner
// checkout or the primary worktree of its repository. It inspects the
// filesystem, not caller-supplied metadata, so a forged or stale handle
// cannot bypass the guard.
func guardDisposable(path, repoDir string) error {
	if samePath(path, repoDir) {
		return fmt.Errorf("%w: %s is the owner repository checkout", ErrPrimaryWorktree, path)
	}
	primary, err := IsPrimaryWorktree(path)
	if err != nil {
		// Not a checked-out worktree at all: either already removed
		// (teardown continues to prune stale metadata) or a path this
		// package never created. Both are safe to clean.
		if errors.Is(err, ErrNotWorktree) {
			return nil
		}
		return err
	}
	if primary {
		return fmt.Errorf("%w: %s is the primary worktree of its repository", ErrPrimaryWorktree, path)
	}
	return nil
}

// IsPrimaryWorktree reports whether dir is the primary (main) worktree of its
// repository: the checkout whose .git is the repository directory itself
// rather than a gitfile pointing into <main>/.git/worktrees/. It fails with
// ErrNotWorktree when dir has no .git entry.
func IsPrimaryWorktree(dir string) (bool, error) {
	fi, err := os.Lstat(filepath.Join(dir, ".git"))
	if err != nil {
		if os.IsNotExist(err) {
			return false, fmt.Errorf("%w: %s", ErrNotWorktree, dir)
		}
		return false, fmt.Errorf("workspace: inspect %s: %w", dir, err)
	}
	return fi.IsDir(), nil
}

// reservePath allocates a unique path under the disposable root, named after
// the base commit plus a random suffix so concurrent Creates never collide.
func (m *Manager) reservePath(commit string) (string, error) {
	short := commit
	if len(short) > 8 {
		short = short[:8]
	}
	for i := 0; i < 5; i++ {
		suffix, err := randomSuffix()
		if err != nil {
			return "", fmt.Errorf("workspace: allocate worktree path: %w", err)
		}
		path := filepath.Join(m.root, "wt-"+short+"-"+suffix)
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			return path, nil
		}
	}
	return "", fmt.Errorf("workspace: could not allocate a unique worktree path under %s", m.root)
}

// randomSuffix returns 8 random hex characters.
func randomSuffix() (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

// insidePath reports whether child lies strictly underneath parent.
func insidePath(parent, child string) bool {
	rel, err := filepath.Rel(filepath.Clean(parent), filepath.Clean(child))
	if err != nil {
		return false
	}
	if filepath.IsAbs(rel) {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// pathExists reports whether path exists (files, directories and symlinks).
func pathExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
