package gitwriter

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

var testSig = Signature{Name: "Tally Test", Email: "test@tally.local"}

// runTestGit runs git for test setup and assertions.
func runTestGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out, errBuf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(errBuf.String()))
	}
	return out.String()
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runTestGit(t, dir, "init", "-q")
	return dir
}

func newTestWriter(t *testing.T, dir string) *Writer {
	t.Helper()
	w, err := New(dir)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return w
}

func headMust(t *testing.T, w *Writer) string {
	t.Helper()
	h, err := w.Head(context.Background())
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	return h
}

func statusMustBeClean(t *testing.T, dir string) {
	t.Helper()
	if out := strings.TrimSpace(runTestGit(t, dir, "status", "--porcelain")); out != "" {
		t.Fatalf("work tree/index not clean after operation:\n%s", out)
	}
}

func blobContent(t *testing.T, dir, rev, path string) string {
	t.Helper()
	return runTestGit(t, dir, "show", rev+":"+path)
}

func TestNewRejectsEmptyPath(t *testing.T) {
	for _, p := range []string{"", "   "} {
		if _, err := New(p); err == nil {
			t.Errorf("New(%q) succeeded, want error", p)
		}
	}
}

func TestHead(t *testing.T) {
	dir := initRepo(t)
	w := newTestWriter(t, dir)
	if h := headMust(t, w); h != "" {
		t.Fatalf("unborn HEAD = %q, want empty", h)
	}
	if _, err := newTestWriter(t, t.TempDir()).Head(context.Background()); !errors.Is(err, ErrNotRepository) {
		t.Fatalf("Head outside a repository: got %v, want ErrNotRepository", err)
	}

	res, err := w.Commit(context.Background(), CommitRequest{
		ExpectedBase: "",
		Message:      "seed",
		Files:        []FileUpdate{{Path: "f.txt", Content: []byte("v1")}},
		Signature:    &testSig,
	})
	if err != nil {
		t.Fatalf("seed commit: %v", err)
	}
	if h := headMust(t, w); h != res.Commit {
		t.Fatalf("Head = %q, want commit %q", h, res.Commit)
	}
	if res.Base != "" {
		t.Fatalf("res.Base = %q, want empty on unborn repo", res.Base)
	}
}

func TestCommitOnUnbornRepositoryRequiresUnbornBase(t *testing.T) {
	dir := initRepo(t)
	w := newTestWriter(t, dir)

	// A caller claiming a non-unborn base on an empty repo is rejected.
	_, err := w.Commit(context.Background(), CommitRequest{
		ExpectedBase: "0123456789abcdef0123456789abcdef01234567",
		Message:      "wrong base",
		Files:        []FileUpdate{{Path: "notes/a.txt", Content: []byte("hello-from-tally")}},
		Signature:    &testSig,
	})
	if !errors.Is(err, ErrBaseRevision) {
		t.Fatalf("wrong base on unborn repo: got %v, want ErrBaseRevision", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "notes")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("rejected commit must not write files: %v", err)
	}

	res, err := w.Commit(context.Background(), CommitRequest{
		ExpectedBase: "",
		Message:      "initial notes",
		Files:        []FileUpdate{{Path: "notes/a.txt", Content: []byte("hello-from-tally")}},
		Signature:    &testSig,
	})
	if err != nil {
		t.Fatalf("commit: %v", err)
	}
	if got := blobContent(t, dir, "HEAD", "notes/a.txt"); got != "hello-from-tally" {
		t.Fatalf("committed content = %q", got)
	}
	if res.Commit != headMust(t, w) {
		t.Fatal("result commit does not match HEAD")
	}
	statusMustBeClean(t, dir)
}

func TestCommitValidatesExpectedBase(t *testing.T) {
	dir := initRepo(t)
	w := newTestWriter(t, dir)
	base := headMust(t, w)
	first, err := w.Commit(context.Background(), CommitRequest{
		ExpectedBase: base,
		Message:      "one",
		Files:        []FileUpdate{{Path: "a.txt", Content: []byte("v1")}},
		Signature:    &testSig,
	})
	if err != nil {
		t.Fatalf("first commit: %v", err)
	}

	// Stale base: rejected before any file is touched.
	_, err = w.Commit(context.Background(), CommitRequest{
		ExpectedBase: first.Base, // == base, no longer HEAD
		Message:      "stale",
		Files:        []FileUpdate{{Path: "late.txt", Content: []byte("must-not-exist")}},
		Signature:    &testSig,
	})
	if !errors.Is(err, ErrBaseRevision) {
		t.Fatalf("stale base: got %v, want ErrBaseRevision", err)
	}
	if _, serr := os.Stat(filepath.Join(dir, "late.txt")); !errors.Is(serr, os.ErrNotExist) {
		t.Fatalf("rejected commit wrote files: %v", serr)
	}
	statusMustBeClean(t, dir)
	if headMust(t, w) != first.Commit {
		t.Fatal("HEAD moved on rejected commit")
	}

	second, err := w.Commit(context.Background(), CommitRequest{
		ExpectedBase: first.Commit,
		Message:      "two",
		Files:        []FileUpdate{{Path: "b.txt", Content: []byte("v2")}},
		Signature:    &testSig,
	})
	if err != nil {
		t.Fatalf("second commit: %v", err)
	}
	if second.Base != first.Commit || second.Commit != headMust(t, w) {
		t.Fatalf("unexpected result %+v", second)
	}
	if out := runTestGit(t, dir, "rev-list", "--count", "HEAD"); strings.TrimSpace(out) != "2" {
		t.Fatalf("commit count = %s, want 2", out)
	}
	statusMustBeClean(t, dir)
}

func TestCommitDeletesFiles(t *testing.T) {
	dir := initRepo(t)
	w := newTestWriter(t, dir)
	base := headMust(t, w)
	first, err := w.Commit(context.Background(), CommitRequest{
		ExpectedBase: base,
		Message:      "add",
		Files: []FileUpdate{
			{Path: "keep.txt", Content: []byte("keep")},
			{Path: "gone.txt", Content: []byte("gone")},
		},
		Signature: &testSig,
	})
	if err != nil {
		t.Fatalf("add commit: %v", err)
	}

	if _, err := w.Commit(context.Background(), CommitRequest{
		ExpectedBase: first.Commit,
		Message:      "remove gone.txt",
		Files:        []FileUpdate{{Path: "gone.txt", Delete: true}},
		Signature:    &testSig,
	}); err != nil {
		t.Fatalf("delete commit: %v", err)
	}
	if _, serr := os.Stat(filepath.Join(dir, "gone.txt")); !errors.Is(serr, os.ErrNotExist) {
		t.Fatalf("deleted file still in work tree: %v", serr)
	}
	if _, err := exec.Command("git", "-C", dir, "show", "HEAD:gone.txt").Output(); err == nil {
		t.Fatal("deleted file still in HEAD")
	}
	if got := blobContent(t, dir, "HEAD", "keep.txt"); got != "keep" {
		t.Fatalf("unrelated file changed: %q", got)
	}
	statusMustBeClean(t, dir)
}

func TestCommitNothingToCommit(t *testing.T) {
	dir := initRepo(t)
	w := newTestWriter(t, dir)
	base := headMust(t, w)
	first, err := w.Commit(context.Background(), CommitRequest{
		ExpectedBase: base,
		Message:      "one",
		Files:        []FileUpdate{{Path: "a.txt", Content: []byte("v1")}},
		Signature:    &testSig,
	})
	if err != nil {
		t.Fatalf("first commit: %v", err)
	}

	_, err = w.Commit(context.Background(), CommitRequest{
		ExpectedBase: first.Commit,
		Message:      "no-op",
		Files:        []FileUpdate{{Path: "a.txt", Content: []byte("v1")}},
		Signature:    &testSig,
	})
	if !errors.Is(err, ErrNoChanges) {
		t.Fatalf("no-change commit: got %v, want ErrNoChanges", err)
	}
	if headMust(t, w) != first.Commit {
		t.Fatal("HEAD moved on no-change commit")
	}
	statusMustBeClean(t, dir)
}

func TestCommitRejectsUnsafePaths(t *testing.T) {
	dir := initRepo(t)
	w := newTestWriter(t, dir)
	for _, p := range []string{
		"",
		".",
		"..",
		"../escape.txt",
		"/abs.txt",
		"C:/abs.txt",
		"sub/../../out.txt",
	} {
		if _, err := w.Commit(context.Background(), CommitRequest{
			ExpectedBase: headMust(t, w),
			Message:      "bad path",
			Files:        []FileUpdate{{Path: p, Content: []byte("x")}},
			Signature:    &testSig,
		}); err == nil {
			t.Errorf("path %q accepted", p)
		}
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "escape.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("file escaped the repository: %v", err)
	}
	statusMustBeClean(t, dir)
}

func TestCommitRollsBackWhenApplyFails(t *testing.T) {
	dir := initRepo(t)
	w := newTestWriter(t, dir)
	// "blocker" is a tracked regular file, so creating "blocker/x.txt"
	// fails midway through apply.
	seed, err := w.Commit(context.Background(), CommitRequest{
		ExpectedBase: headMust(t, w),
		Message:      "seed blocker",
		Files:        []FileUpdate{{Path: "blocker", Content: []byte("i am a file")}},
		Signature:    &testSig,
	})
	if err != nil {
		t.Fatalf("seed blocker: %v", err)
	}

	_, err = w.Commit(context.Background(), CommitRequest{
		ExpectedBase: seed.Commit,
		Message:      "atomicity probe",
		Files: []FileUpdate{
			{Path: "a.txt", Content: []byte("written first")},
			{Path: "blocker/x.txt", Content: []byte("cannot exist")},
		},
		Signature: &testSig,
	})
	if err == nil {
		t.Fatal("commit must fail when apply fails")
	}
	if _, serr := os.Stat(filepath.Join(dir, "a.txt")); !errors.Is(serr, os.ErrNotExist) {
		t.Fatalf("partial apply not rolled back: %v", serr)
	}
	if headMust(t, w) != seed.Commit {
		t.Fatal("HEAD moved on failed commit")
	}
	statusMustBeClean(t, dir)
}

func TestCommitPreservesMessage(t *testing.T) {
	dir := initRepo(t)
	w := newTestWriter(t, dir)
	msg := "subject line\n\nbody text"
	if _, err := w.Commit(context.Background(), CommitRequest{
		ExpectedBase: headMust(t, w),
		Message:      msg,
		Files:        []FileUpdate{{Path: "f.txt", Content: []byte("v")}},
		Signature:    &testSig,
	}); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if got := strings.TrimSpace(runTestGit(t, dir, "log", "-1", "--format=%B")); got != msg {
		t.Fatalf("message = %q, want %q", got, msg)
	}
}

// AC4: locks are released on the error paths.
func TestCommitReleasesLocksAfterBaseError(t *testing.T) {
	dir := initRepo(t)
	w := newTestWriter(t, dir)
	first, err := w.Commit(context.Background(), CommitRequest{
		ExpectedBase: headMust(t, w),
		Message:      "one",
		Files:        []FileUpdate{{Path: "a.txt", Content: []byte("v1")}},
		Signature:    &testSig,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := w.Commit(context.Background(), CommitRequest{
		ExpectedBase: first.Base, // stale
		Message:      "fails on base",
		Files:        []FileUpdate{{Path: "b.txt", Content: []byte("v")}},
		Signature:    &testSig,
	}); !errors.Is(err, ErrBaseRevision) {
		t.Fatalf("expected ErrBaseRevision, got %v", err)
	}

	second, err := w.Commit(context.Background(), CommitRequest{
		ExpectedBase: first.Commit,
		Message:      "two",
		Files:        []FileUpdate{{Path: "b.txt", Content: []byte("v")}},
		Signature:    &testSig,
	})
	if err != nil {
		t.Fatalf("commit after failed one: %v", err)
	}
	if second.Commit != headMust(t, w) {
		t.Fatal("HEAD mismatch after retry")
	}
}

func TestCommitReleasesLocksAfterCanceledContext(t *testing.T) {
	dir := initRepo(t)
	w := newTestWriter(t, dir)
	base := headMust(t, w)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := w.Commit(ctx, CommitRequest{
		ExpectedBase: base,
		Message:      "canceled",
		Files:        []FileUpdate{{Path: "a.txt", Content: []byte("v")}},
		Signature:    &testSig,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled commit: got %v, want context.Canceled", err)
	}
	if h := headMust(t, w); h != base {
		t.Fatal("HEAD moved on canceled commit")
	}

	res, err := w.Commit(context.Background(), CommitRequest{
		ExpectedBase: base,
		Message:      "after cancel",
		Files:        []FileUpdate{{Path: "a.txt", Content: []byte("v")}},
		Signature:    &testSig,
	})
	if err != nil {
		t.Fatalf("commit after canceled one: %v", err)
	}
	if res.Commit != headMust(t, w) {
		t.Fatal("HEAD mismatch after retry")
	}
	statusMustBeClean(t, dir)
}

// AC1: two writers for the same repo serialize.
func TestConcurrentCommitsSerializeSameRepo(t *testing.T) {
	dir := initRepo(t)
	w := newTestWriter(t, dir)

	const workers, perWorker = 4, 3
	var wg sync.WaitGroup
	var mu sync.Mutex
	var failures []error
	for g := 0; g < workers; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				// Losing the base race is expected: retry with a fresh
				// base, which is the intended caller behavior on
				// ErrBaseRevision.
				committed := false
				for attempt := 0; attempt < 100 && !committed; attempt++ {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					head, err := w.Head(ctx)
					if err == nil {
						_, err = w.Commit(ctx, CommitRequest{
							ExpectedBase: head,
							Message:      "concurrent",
							Files:        []FileUpdate{{Path: "g" + string(rune('0'+g)) + "-" + string(rune('0'+i)) + ".txt", Content: []byte("x")}},
							Signature:    &testSig,
						})
					}
					cancel()
					switch {
					case errors.Is(err, ErrBaseRevision):
						continue
					case err != nil:
						mu.Lock()
						failures = append(failures, err)
						mu.Unlock()
						return
					default:
						committed = true
					}
				}
				if !committed {
					mu.Lock()
					failures = append(failures, errors.New("too many ErrBaseRevision retries"))
					mu.Unlock()
					return
				}
			}
		}(g)
	}
	wg.Wait()
	if len(failures) > 0 {
		t.Fatalf("concurrent commits failed: %v", failures)
	}
	// Seed HEAD is unborn, so exactly workers*perWorker commits must exist.
	if out := strings.TrimSpace(runTestGit(t, dir, "rev-list", "--count", "HEAD")); out != "12" {
		t.Fatalf("commit count = %s, want 12", out)
	}
	statusMustBeClean(t, dir)
}

// AC2: different repositories do not block one another.
func TestDifferentRepositoriesDoNotBlock(t *testing.T) {
	dirA, dirB := initRepo(t), initRepo(t)
	wA, wB := newTestWriter(t, dirA), newTestWriter(t, dirB)

	// Hold every lock layer of repo A exactly like a mid-operation writer.
	muA := repoMutexFor(dirA)
	if err := muA.lock(context.Background()); err != nil {
		t.Fatalf("lock A: %v", err)
	}
	flA, err := acquireFileLock(context.Background(), lockPathIn(t, dirA), 0)
	if err != nil {
		muA.unlock()
		t.Fatalf("file lock A: %v", err)
	}

	// Repo B must succeed while A is fully locked.
	ctxB, cancelB := context.WithTimeout(context.Background(), 10*time.Second)
	resB, err := wB.Commit(ctxB, CommitRequest{
		ExpectedBase: headMust(t, wB),
		Message:      "independent repo B",
		Files:        []FileUpdate{{Path: "b.txt", Content: []byte("b")}},
		Signature:    &testSig,
	})
	cancelB()
	if err != nil {
		t.Fatalf("repo B blocked by repo A: %v", err)
	}
	if resB.Commit != headMust(t, wB) {
		t.Fatal("repo B HEAD mismatch")
	}

	// Sanity: repo A really is blocked while held.
	ctxA, cancelA := context.WithTimeout(context.Background(), 150*time.Millisecond)
	_, err = wA.Commit(ctxA, CommitRequest{
		ExpectedBase: headMust(t, wA),
		Message:      "blocked",
		Files:        []FileUpdate{{Path: "a.txt", Content: []byte("a")}},
		Signature:    &testSig,
	})
	cancelA()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("repo A commit while locked: got %v, want DeadlineExceeded", err)
	}

	if err := flA.release(); err != nil {
		t.Fatalf("release file lock A: %v", err)
	}
	muA.unlock()

	resA, err := wA.Commit(context.Background(), CommitRequest{
		ExpectedBase: headMust(t, wA),
		Message:      "after release",
		Files:        []FileUpdate{{Path: "a.txt", Content: []byte("a")}},
		Signature:    &testSig,
	})
	if err != nil {
		t.Fatalf("repo A after release: %v", err)
	}
	if resA.Commit != headMust(t, wA) {
		t.Fatal("repo A HEAD mismatch after release")
	}
}

// AC4: cross-process lock release on error, exercised through the public API.
func TestCommitReleasesCrossProcessLockOnStaleTakeover(t *testing.T) {
	dir := initRepo(t)
	w := newTestWriter(t, dir)
	base := headMust(t, w)

	// Simulate a foreign crashed process holding the lock file.
	writeLockFile(t, lockPathIn(t, dir), fileLockInfo{
		PID:       os.Getpid(),
		Host:      "some-other-host", // not probeable: only the age rule applies
		CreatedAt: time.Now().UTC().Add(-time.Hour),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	w.StaleLockAfter = 30 * time.Second
	if _, err := w.Commit(ctx, CommitRequest{
		ExpectedBase: base,
		Message:      "after takeover",
		Files:        []FileUpdate{{Path: "a.txt", Content: []byte("v")}},
		Signature:    &testSig,
	}); err != nil {
		t.Fatalf("commit with stale foreign lock: %v", err)
	}
	statusMustBeClean(t, dir)
}
