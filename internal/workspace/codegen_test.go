package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

const testPlatform = "windows-amd64"

// AC1/AC2: one run owns exactly one disposable worktree pinned to the exact
// base commit, and every iteration reuses the same directory until the run
// terminates.
func TestRunOpenCreatesOneWorktreeReusedAcrossIterations(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "hello.txt", "base\n", "c1")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))
	runs := NewRunTable(m)
	ctx := context.Background()

	run := RunID("run-1")
	sources, err := runs.Open(ctx, run, c1)
	if err != nil {
		t.Fatalf("Open(%s, %q) error = %v", run, c1, err)
	}
	if sources.Run() != run {
		t.Fatalf("Run() = %s, want %s", sources.Run(), run)
	}
	if sources.BaseCommit() != c1 {
		t.Fatalf("BaseCommit() = %s, want %s", sources.BaseCommit(), c1)
	}
	if head := runGit(t, sources.Path(), "rev-parse", "HEAD"); head != c1 {
		t.Fatalf("worktree HEAD = %s, want base commit %s", head, c1)
	}
	assertFileContent(t, filepath.Join(sources.Path(), "hello.txt"), "base\n")

	// Iteration 1 mutates the worktree in place.
	if err := os.WriteFile(filepath.Join(sources.Path(), "patch.txt"), []byte("iter1"), 0o644); err != nil {
		t.Fatalf("iteration 1 write: %v", err)
	}

	// Iteration 2 (and every later Open) gets the same worktree back: no
	// second checkout is created and iteration 1's state is still there.
	for i := 0; i < 3; i++ {
		reopened, err := runs.Open(ctx, run, c1)
		if err != nil {
			t.Fatalf("Open() iteration %d error = %v", i+2, err)
		}
		if reopened != sources {
			t.Fatalf("Open() iteration %d returned a different handle", i+2)
		}
		assertFileContent(t, filepath.Join(reopened.Path(), "patch.txt"), "iter1")
	}
	assertWorktreeCount(t, m, 2)

	// A symbolic ref that resolves to the same commit reuses the worktree.
	if reopened, err := runs.Open(ctx, run, "HEAD"); err != nil || reopened != sources {
		t.Fatalf("Open(HEAD) = (%p, %v), want the same handle, nil", reopened, err)
	}
	assertWorktreeCount(t, m, 2)

	// Different runs own different worktrees.
	other, err := runs.Open(ctx, RunID("run-2"), c1)
	if err != nil {
		t.Fatalf("Open(run-2) error = %v", err)
	}
	if samePath(other.Path(), sources.Path()) {
		t.Fatalf("runs %s and %s share worktree %s", run, "run-2", other.Path())
	}
	assertWorktreeCount(t, m, 3)
}

// AC1: iterations of one run must agree on the base commit; a conflicting
// base is refused instead of silently reusing the old worktree.
func TestRunOpenBaseConflict(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "hello.txt", "base\n", "c1")
	c2 := commitFile(t, repo, "hello.txt", "changed\n", "c2")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))
	runs := NewRunTable(m)
	ctx := context.Background()

	sources, err := runs.Open(ctx, "run-1", c1)
	if err != nil {
		t.Fatalf("Open(%q) error = %v", c1, err)
	}
	if _, err := runs.Open(ctx, "run-1", c2); !errors.Is(err, ErrRunBaseConflict) {
		t.Fatalf("Open(conflicting base) error = %v, want ErrRunBaseConflict", err)
	}
	// The original worktree is untouched and still the run's only one.
	assertFileContent(t, filepath.Join(sources.Path(), "hello.txt"), "base\n")
	assertWorktreeCount(t, m, 2)
}

// AC1/AC2: termination is the end of the run's worktree — idempotent, and a
// terminated run never gets a second worktree.
func TestRunTerminateIdempotentNoSecondWorktree(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "hello.txt", "base\n", "c1")
	root := filepath.Join(t.TempDir(), "ws")
	m := newManagerAt(t, repo, root)
	runs := NewRunTable(m)
	ctx := context.Background()

	run := RunID("run-1")
	sources, err := runs.Open(ctx, run, c1)
	if err != nil {
		t.Fatalf("Open(%s) error = %v", run, err)
	}
	if err := runs.Terminate(ctx, run); err != nil {
		t.Fatalf("Terminate(%s) error = %v", run, err)
	}
	if pathExists(sources.Path()) {
		t.Fatalf("worktree directory %s still exists after termination", sources.Path())
	}
	// Termination is idempotent, on both the table and the handle.
	if err := runs.Terminate(ctx, run); err != nil {
		t.Fatalf("repeated Terminate(%s) error = %v", run, err)
	}
	if err := sources.Terminate(ctx); err != nil {
		t.Fatalf("Terminate() on terminated handle error = %v", err)
	}
	// Terminating a run that never opened is a no-op success.
	if err := runs.Terminate(ctx, "never-opened"); err != nil {
		t.Fatalf("Terminate(unknown run) error = %v", err)
	}
	assertRootEmpty(t, root)
	assertWorktreeCount(t, m, 1)

	// A late Open for the terminated run fails instead of minting a second
	// worktree for the same run.
	if _, err := runs.Open(ctx, run, c1); !errors.Is(err, ErrRunTerminated) {
		t.Fatalf("Open(terminated run) error = %v, want ErrRunTerminated", err)
	}
	assertWorktreeCount(t, m, 1)
}

// AC1: empty run identifiers are refused without touching git.
func TestRunOpenEmptyID(t *testing.T) {
	repo := newTempRepo(t)
	commitFile(t, repo, "hello.txt", "base\n", "c1")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))
	runs := NewRunTable(m)

	if _, err := runs.Open(context.Background(), "", "HEAD"); !errors.Is(err, ErrEmptyRunID) {
		t.Fatalf("Open(\"\") error = %v, want ErrEmptyRunID", err)
	}
	assertWorktreeCount(t, m, 1)
}

// AC1/AC2: concurrent Opens of the same run observe one shared worktree —
// exactly one is created.
func TestRunTableConcurrentOpenSameRun(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "hello.txt", "base\n", "c1")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))
	runs := NewRunTable(m)

	const callers = 6
	var wg sync.WaitGroup
	handles := make(chan *RunSources, callers)
	errs := make(chan error, callers)
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sources, err := runs.Open(context.Background(), "run-1", c1)
			if err != nil {
				errs <- err
				return
			}
			handles <- sources
		}()
	}
	wg.Wait()
	close(handles)
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent Open() error = %v", err)
	}
	var first *RunSources
	for h := range handles {
		if first == nil {
			first = h
			continue
		}
		if h != first {
			t.Fatal("concurrent Opens of one run returned different handles")
		}
	}
	assertWorktreeCount(t, m, 2)
}

// AC3: cache paths are deterministic per key and never inside the source
// worktree (nor is the worktree inside the cache).
func TestDependencyCacheKeyableOutsideWorktree(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "hello.txt", "base\n", "c1")
	cacheRoot := filepath.Join(t.TempDir(), "cache")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))
	cache, err := m.NewDependencyCache(cacheRoot)
	if err != nil {
		t.Fatalf("NewDependencyCache(%q) error = %v", cacheRoot, err)
	}
	if !samePath(cache.Root(), cacheRoot) {
		t.Fatalf("Root() = %s, want %s", cache.Root(), cacheRoot)
	}

	runs := NewRunTable(m)
	sources, err := runs.Open(context.Background(), "run-1", c1)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	key := CacheKey{Toolchain: "go", LockHash: "3bf1c2", Platform: testPlatform}
	path, err := cache.Path(key)
	if err != nil {
		t.Fatalf("Path(go key) error = %v", err)
	}
	again, err := cache.Path(key)
	if err != nil || path != again {
		t.Fatalf("Path(go key) = (%q, %v) then (%q, %v), want identical paths", path, err, again, err)
	}
	// The key dimensions nest unambiguously and stay under the cache root.
	if rel, err := filepath.Rel(cache.Root(), path); err != nil || rel == ".." || filepath.IsAbs(rel) {
		t.Fatalf("cache path %s is not under root %s (rel %q, err %v)", path, cache.Root(), rel, err)
	}
	if insidePath(sources.Path(), path) || insidePath(path, sources.Path()) {
		t.Fatalf("cache path %s and source worktree %s nest", path, sources.Path())
	}

	// Different keys never share a directory, in any dimension.
	for _, other := range []CacheKey{
		{Toolchain: "node", LockHash: "3bf1c2", Platform: testPlatform},
		{Toolchain: "go", LockHash: "other", Platform: testPlatform},
		{Toolchain: "go", LockHash: "3bf1c2", Platform: "linux-amd64"},
	} {
		p, err := cache.Path(other)
		if err != nil {
			t.Fatalf("Path(%+v) error = %v", other, err)
		}
		if samePath(p, path) {
			t.Fatalf("key %+v collides with %+v at %s", other, key, p)
		}
	}
}

// AC3: malformed key components are refused before they can reach the
// filesystem.
func TestDependencyCacheKeyValidation(t *testing.T) {
	repo := newTempRepo(t)
	commitFile(t, repo, "hello.txt", "base\n", "c1")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))
	cache, err := m.NewDependencyCache(filepath.Join(t.TempDir(), "cache"))
	if err != nil {
		t.Fatalf("NewDependencyCache() error = %v", err)
	}

	for _, key := range []CacheKey{
		{Toolchain: "", LockHash: "abc", Platform: testPlatform},
		{Toolchain: "go", LockHash: "", Platform: testPlatform},
		{Toolchain: "go", LockHash: "abc", Platform: ""},
		{Toolchain: "go/cache", LockHash: "abc", Platform: testPlatform},
		{Toolchain: "..", LockHash: "abc", Platform: testPlatform},
		{Toolchain: "go", LockHash: "a b", Platform: testPlatform},
		{Toolchain: "-lead", LockHash: "abc", Platform: testPlatform},
		{Toolchain: "go", LockHash: string(make([]byte, 65)), Platform: testPlatform},
	} {
		if _, err := cache.Path(key); !errors.Is(err, ErrInvalidCacheKey) {
			t.Fatalf("Path(%+v) error = %v, want ErrInvalidCacheKey", key, err)
		}
	}
}

// AC3/AC4: the cache root may not nest with the owner checkout or the
// disposable worktree root in either direction.
func TestNewDependencyCacheBoundary(t *testing.T) {
	repo := newTempRepo(t)
	commitFile(t, repo, "hello.txt", "base\n", "c1")
	base := t.TempDir()
	wsRoot := filepath.Join(base, "ws")
	m := newManagerAt(t, repo, wsRoot)

	for _, tc := range []struct {
		name string
		root string
		want error
	}{
		{"inside owner checkout", filepath.Join(repo, "cache"), ErrPrimaryWorktree},
		{"owner checkout itself", repo, ErrPrimaryWorktree},
		{"inside disposable root", filepath.Join(wsRoot, "cache"), ErrCacheBoundary},
		{"disposable root itself", wsRoot, ErrCacheBoundary},
		{"contains disposable root", base, ErrCacheBoundary},
		{"contains owner checkout", filepath.Dir(base), ErrCacheBoundary},
		{"empty", "", ErrCacheBoundary},
	} {
		if _, err := m.NewDependencyCache(tc.root); !errors.Is(err, tc.want) {
			t.Fatalf("%s: NewDependencyCache(%q) error = %v, want %v", tc.name, tc.root, err, tc.want)
		}
	}

	// A sibling of the disposable root is a valid cache home.
	if _, err := m.NewDependencyCache(filepath.Join(base, "cache")); err != nil {
		t.Fatalf("NewDependencyCache(sibling) error = %v", err)
	}
}

// AC4: run termination deletes only the run's source worktree — a populated
// shared cache survives untouched.
func TestRunTerminatePreservesSharedCache(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "hello.txt", "base\n", "c1")
	base := t.TempDir()
	m := newManagerAt(t, repo, filepath.Join(base, "ws"))
	cache, err := m.NewDependencyCache(filepath.Join(base, "cache"))
	if err != nil {
		t.Fatalf("NewDependencyCache() error = %v", err)
	}

	// An agent populated the shared cache during an earlier run.
	key := CacheKey{Toolchain: "go", LockHash: "3bf1c2", Platform: testPlatform}
	cachePath, err := cache.Path(key)
	if err != nil {
		t.Fatalf("Path() error = %v", err)
	}
	if err := os.MkdirAll(cachePath, 0o755); err != nil {
		t.Fatalf("populate cache: %v", err)
	}
	sentinel := filepath.Join(cachePath, "module.zip")
	if err := os.WriteFile(sentinel, []byte("cached"), 0o644); err != nil {
		t.Fatalf("populate cache: %v", err)
	}

	runs := NewRunTable(m)
	sources, err := runs.Open(context.Background(), "run-1", c1)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	// The worktree stayed dirty, as after a real run.
	if err := os.WriteFile(filepath.Join(sources.Path(), "scratch.txt"), []byte("dirty"), 0o644); err != nil {
		t.Fatalf("dirty the worktree: %v", err)
	}

	if err := runs.Terminate(context.Background(), "run-1"); err != nil {
		t.Fatalf("Terminate() error = %v", err)
	}
	if pathExists(sources.Path()) {
		t.Fatalf("worktree %s still exists after termination", sources.Path())
	}
	assertFileContent(t, sentinel, "cached")
	entries, err := os.ReadDir(cache.Root())
	if err != nil {
		t.Fatalf("read cache root: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("cache root is empty after run termination")
	}
}

// AC5: the whole run lifecycle never touches the owner primary worktree.
func TestRunLifecycleLeavesOwnerWorktreeUntouched(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "hello.txt", "owner content\n", "c1")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))
	runs := NewRunTable(m)
	ctx := context.Background()

	sources, err := runs.Open(ctx, "run-1", "HEAD")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	// Iteration writes land in the disposable worktree only.
	if err := os.WriteFile(filepath.Join(sources.Path(), "hello.txt"), []byte("mutated\n"), 0o644); err != nil {
		t.Fatalf("mutate worktree: %v", err)
	}
	if err := runs.Terminate(ctx, "run-1"); err != nil {
		t.Fatalf("Terminate() error = %v", err)
	}

	assertFileContent(t, filepath.Join(repo, "hello.txt"), "owner content\n")
	if head := runGit(t, repo, "rev-parse", "HEAD"); head != c1 {
		t.Fatalf("owner HEAD = %s, want %s", head, c1)
	}
	assertWorktreeCount(t, m, 1)
	if primary, err := IsPrimaryWorktree(repo); err != nil || !primary {
		t.Fatalf("IsPrimaryWorktree(owner) = (%v, %v), want (true, nil)", primary, err)
	}
}
