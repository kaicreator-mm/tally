package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"tally/internal/runner"
)

// newManagerAt returns a Manager rooted at the given repo with its disposable
// worktrees under root.
func newManagerAt(t *testing.T, repo, root string) *Manager {
	t.Helper()
	m, err := New(repo, root)
	if err != nil {
		t.Fatalf("New(%q, %q) error = %v", repo, root, err)
	}
	return m
}

// createWorktree creates a disposable worktree and fails the test on error.
func createWorktree(t *testing.T, m *Manager, base string) *Worktree {
	t.Helper()
	wt, err := m.Create(context.Background(), base)
	if err != nil {
		t.Fatalf("Create(%q) error = %v", base, err)
	}
	return wt
}

func assertWorktreeCount(t *testing.T, m *Manager, want int) {
	t.Helper()
	entries, err := m.git.worktreeList(context.Background(), m.repoDir)
	if err != nil {
		t.Fatalf("worktreeList: %v", err)
	}
	if len(entries) != want {
		t.Fatalf("repository has %d registered worktrees, want %d: %+v", len(entries), want, entries)
	}
}

func assertRootEmpty(t *testing.T, root string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		t.Fatalf("read disposable root %s: %v", root, err)
	}
	if len(entries) != 0 {
		t.Fatalf("disposable root %s is not empty: %d leftover entries", root, len(entries))
	}
}

// AC1: creates a worktree at an exact base revision.
func TestCreateWorktreeAtExactBaseRevision(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "hello.txt", "base\n", "c1")
	c2 := commitFile(t, repo, "hello.txt", "changed\n", "c2")

	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))

	// An exact SHA pins the worktree to that commit even though HEAD moved.
	wt := createWorktree(t, m, c1)
	if wt.BaseCommit != c1 {
		t.Fatalf("BaseCommit = %s, want %s", wt.BaseCommit, c1)
	}
	if head := runGit(t, wt.Path, "rev-parse", "HEAD"); head != c1 {
		t.Fatalf("worktree HEAD = %s, want base commit %s", head, c1)
	}
	if ref := runGit(t, wt.Path, "rev-parse", "--abbrev-ref", "HEAD"); ref != "HEAD" {
		t.Fatalf("worktree checked out ref %q, want detached HEAD", ref)
	}
	assertFileContent(t, filepath.Join(wt.Path, "hello.txt"), "base\n")

	// A symbolic ref resolves to the exact current commit.
	wt2 := createWorktree(t, m, "HEAD")
	if wt2.BaseCommit != c2 {
		t.Fatalf("BaseCommit = %s, want %s", wt2.BaseCommit, c2)
	}
	if head := runGit(t, wt2.Path, "rev-parse", "HEAD"); head != c2 {
		t.Fatalf("worktree HEAD = %s, want %s", head, c2)
	}
	assertWorktreeCount(t, m, 3)
}

// AC1/AC4: unknown revisions fail without leaving state behind.
func TestCreateUnknownRevision(t *testing.T) {
	repo := newTempRepo(t)
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))

	if _, err := m.Create(context.Background(), "definitely-not-a-revision"); !errors.Is(err, ErrUnknownRevision) {
		t.Fatalf("Create(unknown) error = %v, want ErrUnknownRevision", err)
	}
	assertWorktreeCount(t, m, 1)
}

// AC3: the owner primary worktree is never usable as the disposable root.
func TestNewRefusesDisposableRootInsideOwnerCheckout(t *testing.T) {
	repo := newTempRepo(t)
	for _, root := range []string{repo, filepath.Join(repo, "sub", "ws")} {
		if _, err := New(repo, root); !errors.Is(err, ErrPrimaryWorktree) {
			t.Fatalf("New(repo, %q) error = %v, want ErrPrimaryWorktree", root, err)
		}
	}
	if _, err := New(repo, filepath.Join(t.TempDir(), "ws")); err != nil {
		t.Fatalf("New(repo, outside root) error = %v, want nil", err)
	}
	if _, err := New(filepath.Join(t.TempDir(), "not-a-repo"), filepath.Join(t.TempDir(), "ws")); !errors.Is(err, ErrNotRepository) {
		t.Fatalf("New(non-repo) error = %v, want ErrNotRepository", err)
	}
}

// AC3: cleanup refuses to destroy the owner primary worktree, whatever the
// caller claims the handle points at.
func TestCleanupRefusesOwnerPrimaryWorktree(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "hello.txt", "base\n", "c1")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))

	wt := &Worktree{Path: repo, BaseCommit: c1, repoDir: repo, git: m.git}
	if err := m.Cleanup(context.Background(), wt); !errors.Is(err, ErrPrimaryWorktree) {
		t.Fatalf("Cleanup(owner checkout) error = %v, want ErrPrimaryWorktree", err)
	}
	// The owner checkout is untouched.
	if head := runGit(t, repo, "rev-parse", "HEAD"); head != c1 {
		t.Fatalf("owner checkout HEAD = %s, want %s", head, c1)
	}
	assertFileContent(t, filepath.Join(repo, "hello.txt"), "base\n")

	// A primary worktree of any other repository is refused as well.
	other := newTempRepo(t)
	commitFile(t, other, "hello.txt", "base\n", "other c1")
	if _, err := IsPrimaryWorktree(other); err != nil {
		t.Fatalf("IsPrimaryWorktree(%q) error = %v", other, err)
	}
	otherWt := &Worktree{Path: other, repoDir: repo, git: m.git}
	if err := m.Cleanup(context.Background(), otherWt); !errors.Is(err, ErrPrimaryWorktree) {
		t.Fatalf("Cleanup(other primary worktree) error = %v, want ErrPrimaryWorktree", err)
	}
	assertFileContent(t, filepath.Join(other, "hello.txt"), "base\n")

	// A nil handle is a no-op, convenient for defer-based cleanup.
	if err := m.Cleanup(context.Background(), nil); err != nil {
		t.Fatalf("Cleanup(nil) error = %v, want nil", err)
	}
}

// AC4: only a linked worktree (gitfile .git) is considered disposable.
func TestIsPrimaryWorktree(t *testing.T) {
	repo := newTempRepo(t)
	commitFile(t, repo, "hello.txt", "base\n", "c1")
	primary, err := IsPrimaryWorktree(repo)
	if err != nil || !primary {
		t.Fatalf("IsPrimaryWorktree(owner checkout) = (%v, %v), want (true, nil)", primary, err)
	}

	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))
	wt := createWorktree(t, m, "HEAD")
	linked, err := IsPrimaryWorktree(wt.Path)
	if err != nil || linked {
		t.Fatalf("IsPrimaryWorktree(disposable) = (%v, %v), want (false, nil)", linked, err)
	}

	if _, err := IsPrimaryWorktree(filepath.Join(t.TempDir(), "gone")); !errors.Is(err, ErrNotWorktree) {
		t.Fatalf("IsPrimaryWorktree(missing) error = %v, want ErrNotWorktree", err)
	}
}

// AC2: cleanup is idempotent on the success path, including dirty worktrees.
func TestCleanupIdempotentAfterSuccess(t *testing.T) {
	repo := newTempRepo(t)
	commitFile(t, repo, "hello.txt", "base\n", "c1")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))

	wt := createWorktree(t, m, "HEAD")
	// Uncommitted changes must not block teardown.
	if err := os.WriteFile(filepath.Join(wt.Path, "scratch.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("dirty the worktree: %v", err)
	}

	if err := m.Cleanup(context.Background(), wt); err != nil {
		t.Fatalf("Cleanup() error = %v, want nil", err)
	}
	if pathExists(wt.Path) {
		t.Fatalf("worktree directory %s still exists after cleanup", wt.Path)
	}
	assertWorktreeCount(t, m, 1)

	// Repeated cleanup is a no-op success.
	for i := 0; i < 2; i++ {
		if err := m.Cleanup(context.Background(), wt); err != nil {
			t.Fatalf("Cleanup() call %d error = %v, want nil", i+2, err)
		}
	}
}

// AC2: cleanup recovers from a failed run that left stale state — the
// worktree directory vanished while the repository metadata stayed behind.
func TestCleanupAfterFailureWithStaleState(t *testing.T) {
	repo := newTempRepo(t)
	commitFile(t, repo, "hello.txt", "base\n", "c1")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))

	wt := createWorktree(t, m, "HEAD")
	// Simulate a crashed run: the directory is wiped underneath the manager.
	if err := os.RemoveAll(wt.Path); err != nil {
		t.Fatalf("simulate crashed worktree: %v", err)
	}

	if err := m.Cleanup(context.Background(), wt); err != nil {
		t.Fatalf("Cleanup() after failure error = %v, want nil", err)
	}
	assertWorktreeCount(t, m, 1)

	// A stale handle over the same path cleans up as a no-op too.
	stale := &Worktree{Path: wt.Path, repoDir: repo, git: m.git}
	if err := m.Cleanup(context.Background(), stale); err != nil {
		t.Fatalf("Cleanup() on stale handle error = %v, want nil", err)
	}
	assertWorktreeCount(t, m, 1)
}

// AC2: a cancelled Create never leaks a worktree, even when git already
// finished registering it when the cancellation was observed.
func TestCreateCancellationLeavesNoWorktree(t *testing.T) {
	repo := newTempRepo(t)
	commitFile(t, repo, "hello.txt", "base\n", "c1")
	root := filepath.Join(t.TempDir(), "ws")

	local := runner.LocalRunner{}
	added := make(chan struct{})
	m := &Manager{
		repoDir: repo,
		root:    root,
		git: &gitClient{run: func(ctx context.Context, cmd runner.Command) (runner.Result, error) {
			if len(cmd.Argv) >= 3 && cmd.Argv[1] == "worktree" && cmd.Argv[2] == "add" {
				// Run the real registration to completion, then report
				// the context cancellation: the real race between a
				// killed process and git having done its work.
				res, err := local.Run(context.WithoutCancel(ctx), cmd)
				if err != nil {
					return res, err
				}
				close(added)
				<-ctx.Done()
				return res, ctx.Err()
			}
			return local.Run(ctx, cmd)
		}},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-added
		cancel()
	}()

	if _, err := m.Create(ctx, "HEAD"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Create(canceled) error = %v, want context.Canceled", err)
	}
	assertRootEmpty(t, root)
	assertWorktreeCount(t, m, 1)
}

// AC2: cleanup completes even when the caller's context is already
// cancelled — teardown must not inherit the cancellation that killed the
// owning operation.
func TestCleanupAfterCancellation(t *testing.T) {
	repo := newTempRepo(t)
	commitFile(t, repo, "hello.txt", "base\n", "c1")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))

	wt := createWorktree(t, m, "HEAD")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := m.Cleanup(ctx, wt); err != nil {
		t.Fatalf("Cleanup(canceled ctx) error = %v, want nil", err)
	}
	if pathExists(wt.Path) {
		t.Fatalf("worktree directory %s still exists after cleanup", wt.Path)
	}
	assertWorktreeCount(t, m, 1)

	// Still idempotent on the cancelled context.
	if err := m.Cleanup(ctx, wt); err != nil {
		t.Fatalf("repeated Cleanup(canceled ctx) error = %v, want nil", err)
	}
}

// AC2: concurrent cleanup of the same worktree is safe; exactly one teardown
// happens and every caller observes success.
func TestConcurrentCleanup(t *testing.T) {
	repo := newTempRepo(t)
	commitFile(t, repo, "hello.txt", "base\n", "c1")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))

	wt := createWorktree(t, m, "HEAD")

	const callers = 8
	var wg sync.WaitGroup
	errs := make(chan error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- m.Cleanup(context.Background(), wt)
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Cleanup() error = %v", err)
		}
	}
	if pathExists(wt.Path) {
		t.Fatalf("worktree directory %s still exists after concurrent cleanup", wt.Path)
	}
	assertWorktreeCount(t, m, 1)
}

// AC1/AC4: concurrent Creates never collide on paths and all pin their base.
func TestCreateConcurrentDistinctPaths(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "hello.txt", "base\n", "c1")
	root := filepath.Join(t.TempDir(), "ws")
	m := newManagerAt(t, repo, root)

	const creates = 4
	var wg sync.WaitGroup
	paths := make(chan string, creates)
	errs := make(chan error, creates)
	for i := 0; i < creates; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			wt, err := m.Create(context.Background(), c1)
			if err != nil {
				errs <- err
				return
			}
			paths <- wt.Path
		}()
	}
	wg.Wait()
	close(paths)
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent Create() error = %v", err)
	}
	seen := make(map[string]bool, creates)
	for p := range paths {
		if seen[p] {
			t.Fatalf("duplicate worktree path %q", p)
		}
		seen[p] = true
		if head := runGit(t, p, "rev-parse", "HEAD"); head != c1 {
			t.Fatalf("worktree %s HEAD = %s, want %s", p, head, c1)
		}
	}
	assertWorktreeCount(t, m, creates+1)
}
