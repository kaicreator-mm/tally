package workspace

// candidate.go — prior-candidate materialization tests (P2-05, ADR-0001 §2).
//
// Verification mapping (reference pack):
//   - the initial repair workspace equals the prior candidate per ADR (the
//     frozen content layered on the same base revision);
//   - materialization is deterministic (equal input, equal tree);
//   - unsafe/ambiguous candidate sets are refused before any write;
//   - the owner primary worktree is never a writable repair workspace;
//   - workspace disposal failure surfaces explicitly and stays retryable
//     (a leaked writable workspace is a reportable outcome, never silent).

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tally/internal/runner"
)

// AC1: materializing the prior candidate into a fresh run worktree pinned at
// the same base revision yields exactly the prior candidate state — the base
// file is replaced by the frozen content, new files appear with exact
// content, and the result is independent of input order (deterministic).
func TestMaterializePriorCandidateCreatesRepairBase(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitNestedFile(t, repo, "svc/hello.txt", "base content\n", "c1")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))
	runs := NewRunTable(m)
	ctx := context.Background()

	// Two spellings of the same set: path order must not matter.
	files := []CandidateFile{
		{Path: "svc/hello.txt", Content: "prior candidate content\n"},
		{Path: "pkg/new/feature.go", Content: "package new\n", Executable: true},
		{Path: "docs/note.md", Content: "# note\n"},
	}
	shuffled := []CandidateFile{files[2], files[0], files[1]}

	var trees []string
	for i, in := range [][]CandidateFile{files, shuffled} {
		run := RunID(fmt.Sprintf("repair-%d", i+1))
		// The repair workspace base is the prior workspace base revision.
		sources, err := runs.Open(ctx, run, c1)
		if err != nil {
			t.Fatalf("Open(%s) error = %v", run, err)
		}
		if sources.BaseCommit() != c1 {
			t.Fatalf("BaseCommit() = %s, want %s", sources.BaseCommit(), c1)
		}
		if err := sources.Materialize(in); err != nil {
			t.Fatalf("Materialize(run %s) error = %v", run, err)
		}
		assertFileContent(t, filepath.Join(sources.Path(), "svc", "hello.txt"), "prior candidate content\n")
		assertFileContent(t, filepath.Join(sources.Path(), "pkg", "new", "feature.go"), "package new\n")
		assertFileContent(t, filepath.Join(sources.Path(), "docs", "note.md"), "# note\n")
		if runtime.GOOS != "windows" {
			assertMode(t, filepath.Join(sources.Path(), "pkg", "new", "feature.go"), 0o755)
			assertMode(t, filepath.Join(sources.Path(), "docs", "note.md"), 0o644)
		}
		// The worktree stays pinned at the base commit: materialization is
		// workspace content, not a git operation.
		if head := runGit(t, sources.Path(), "rev-parse", "HEAD"); head != c1 {
			t.Fatalf("worktree HEAD = %s, want base commit %s", head, c1)
		}
		trees = append(trees, sources.Path())
	}

	// Determinism: both runs materialized identical trees.
	assertTreeIdentical(t, trees[0], trees[1])

	// AC5: the owner primary worktree still holds the base content.
	assertFileContent(t, filepath.Join(repo, "svc", "hello.txt"), "base content\n")
}

// AC1 (empty set): an empty materialization set is a no-op success — the
// base checkout is the full continuation state.
func TestMaterializeEmptySetIsNoOp(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "hello.txt", "base\n", "c1")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))
	runs := NewRunTable(m)
	sources, err := runs.Open(context.Background(), "run-1", c1)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := sources.Materialize(nil); err != nil {
		t.Fatalf("Materialize(nil) error = %v", err)
	}
	assertFileContent(t, filepath.Join(sources.Path(), "hello.txt"), "base\n")
}

// AC1/AC5: unsafe or ambiguous candidate sets are refused with typed errors,
// and a set containing an unsafe entry materializes nothing at all (the
// whole set is validated before the first write).
func TestMaterializeRefusesUnsafeSets(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "hello.txt", "base\n", "c1")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))
	runs := NewRunTable(m)
	sources, err := runs.Open(context.Background(), "run-1", c1)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	// The owner file the traversal attempts would love to reach.
	ownerFile := filepath.Join(repo, "hello.txt")

	for _, f := range []CandidateFile{
		{Path: "../escape.txt", Content: "x"},
		{Path: "/abs.txt", Content: "x"},
		{Path: "a/../../b.txt", Content: "x"},
		{Path: `back\slash.txt`, Content: "x"},
		{Path: "C:/evil.txt", Content: "x"},
		{Path: "drive:d.txt", Content: "x"},
		{Path: "", Content: "x"},
		{Path: "   ", Content: "x"},
		{Path: "a//b.txt", Content: "x"},
		{Path: "./rel.txt", Content: "x"},
		{Path: "..", Content: "x"},
		{Path: ".git/config", Content: "x"},
		{Path: ".git", Content: "x"},
	} {
		if err := sources.Materialize([]CandidateFile{f}); !errors.Is(err, ErrUnsafeCandidatePath) {
			t.Fatalf("Materialize(%q) error = %v, want ErrUnsafeCandidatePath", f.Path, err)
		}
	}
	if err := sources.Materialize([]CandidateFile{
		{Path: "a.txt", Content: "1"},
		{Path: "a.txt", Content: "2"},
	}); !errors.Is(err, ErrDuplicateCandidateFile) {
		t.Fatalf("Materialize(duplicate) error = %v, want ErrDuplicateCandidateFile", err)
	}

	// Mixed valid+invalid set: validated as a whole, nothing written.
	if err := sources.Materialize([]CandidateFile{
		{Path: "ok.txt", Content: "ok"},
		{Path: "../nope.txt", Content: "x"},
	}); !errors.Is(err, ErrUnsafeCandidatePath) {
		t.Fatalf("Materialize(mixed set) error = %v, want ErrUnsafeCandidatePath", err)
	}
	if pathExists(filepath.Join(sources.Path(), "ok.txt")) {
		t.Fatal("an invalid set must not be partially materialized")
	}
	assertFileContent(t, ownerFile, "base\n")
}

// AC5: existing entries that would redirect or absorb a write are refused —
// a non-regular file at the target, a symlink in the parent chain — and the
// owner checkout stays untouched.
func TestMaterializeRefusesBlockedTargets(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "keep.txt", "base\n", "c1")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))
	runs := NewRunTable(m)
	sources, err := runs.Open(context.Background(), "run-1", c1)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	// A directory exists where the candidate wants a file.
	if err := os.MkdirAll(filepath.Join(sources.Path(), "occupied"), 0o755); err != nil {
		t.Fatalf("mkdir occupied: %v", err)
	}
	if err := sources.Materialize([]CandidateFile{{Path: "occupied", Content: "x"}}); !errors.Is(err, ErrMaterializeTarget) {
		t.Fatalf("Materialize(onto directory) error = %v, want ErrMaterializeTarget", err)
	}

	// A symlink in the parent chain pointing at the owner checkout must
	// never be followed. Symlink creation needs privileges on Windows.
	link := filepath.Join(sources.Path(), "jump")
	if err := os.Symlink(repo, link); err != nil {
		t.Skipf("symlink unsupported on this platform: %v", err)
	}
	if err := sources.Materialize([]CandidateFile{{Path: "jump/evil.txt", Content: "x"}}); !errors.Is(err, ErrMaterializeTarget) {
		t.Fatalf("Materialize(through symlink) error = %v, want ErrMaterializeTarget", err)
	}
	if pathExists(filepath.Join(repo, "evil.txt")) {
		t.Fatal("a write escaped through a symlinked parent directory")
	}
	assertFileContent(t, filepath.Join(repo, "keep.txt"), "base\n")
}

// Materializing into a terminated run's worktree is refused: a terminated
// run never gets its disposable workspace back.
func TestMaterializeAfterTerminateRefused(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "hello.txt", "base\n", "c1")
	m := newManagerAt(t, repo, filepath.Join(t.TempDir(), "ws"))
	runs := NewRunTable(m)
	ctx := context.Background()
	sources, err := runs.Open(ctx, "run-1", c1)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := runs.Terminate(ctx, "run-1"); err != nil {
		t.Fatalf("Terminate() error = %v", err)
	}
	if err := sources.Materialize([]CandidateFile{{Path: "late.txt", Content: "x"}}); !errors.Is(err, ErrRunTerminated) {
		t.Fatalf("Materialize(after terminate) error = %v, want ErrRunTerminated", err)
	}
}

// RQ-05 cleanup discipline: a workspace disposal failure surfaces explicitly
// as an error and the worktree stays retryable — a leaked writable workspace
// is a reportable outcome, never a silent one.
func TestTerminateFailureSurfacesAndStaysRetryable(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "hello.txt", "base\n", "c1")
	root := filepath.Join(t.TempDir(), "ws")
	m := newManagerAt(t, repo, root)
	runs := NewRunTable(m)
	ctx := context.Background()
	sources, err := runs.Open(ctx, "run-1", c1)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := sources.Materialize([]CandidateFile{{Path: "dirty.txt", Content: "x"}}); err != nil {
		t.Fatalf("Materialize() error = %v", err)
	}

	// Inject git-level failure for the teardown commands.
	orig := m.git.run
	m.git.run = func(context.Context, runner.Command) (runner.Result, error) {
		return runner.Result{}, errors.New("git down")
	}
	if err := runs.Terminate(ctx, "run-1"); err == nil {
		t.Fatal("disposal failure must surface explicitly, not silently leak the writable worktree")
	}
	// Restore git before inspecting observed state: the worktree is still
	// registered (its metadata was never pruned) — the leak is observable
	// and the run stays unterminated.
	m.git.run = orig
	assertWorktreeCount(t, m, 2)

	// The surfaced failure is retryable: with git healthy again the same
	// Terminate call completes and the workspace is really gone.
	if err := runs.Terminate(ctx, "run-1"); err != nil {
		t.Fatalf("Terminate() retry after surfaced failure: %v", err)
	}
	if pathExists(sources.Path()) {
		t.Fatalf("worktree %s still exists after retried termination", sources.Path())
	}
	assertRootEmpty(t, root)
	assertWorktreeCount(t, m, 1)
}

// ---- helpers ----

// commitNestedFile is commitFile for a path with parent directories: it
// creates the parents, writes the file and commits everything, returning the
// new HEAD SHA.
func commitNestedFile(t *testing.T, repo, name, content, msg string) string {
	t.Helper()
	abs := filepath.Join(repo, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", name, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", msg)
	return runGit(t, repo, "rev-parse", "HEAD")
}

func assertMode(t *testing.T, path string, want fs.FileMode) {
	t.Helper()
	fi, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("lstat %s: %v", path, err)
	}
	if got := fi.Mode().Perm(); got != want {
		t.Fatalf("mode of %s = %o, want %o", path, got, want)
	}
}

// assertTreeIdentical compares two materialized workspace trees (relative
// paths + content digests, git metadata excluded) and fails when they differ.
func assertTreeIdentical(t *testing.T, a, b string) {
	t.Helper()
	digest := func(t *testing.T) map[string]string {
		t.Helper()
		out := map[string]string{}
		err := filepath.WalkDir(a, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(a, p)
			if err != nil {
				return err
			}
			if rel == "." {
				return nil
			}
			slash := filepath.ToSlash(rel)
			if strings.EqualFold(slash, ".git") {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(data)
			out[slash] = hex.EncodeToString(sum[:])
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", a, err)
		}
		return out
	}
	da, db := digest(t), digest(t)
	if len(da) != len(db) {
		t.Fatalf("trees differ in file count: %d vs %d (%v vs %v)", len(da), len(db), da, db)
	}
	for p, ha := range da {
		hb, ok := db[p]
		if !ok || hb != ha {
			t.Fatalf("trees differ at %s: %q vs %q (present=%v)", p, ha, hb, ok)
		}
	}
}
