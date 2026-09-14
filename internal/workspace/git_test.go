package workspace

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tally/internal/runner"
)

func TestResolveRevision(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "hello.txt", "base\n", "c1")
	g := newGitClient()
	ctx := context.Background()

	if got, err := g.resolveRevision(ctx, repo, "HEAD"); err != nil || got != c1 {
		t.Fatalf("resolveRevision(HEAD) = (%q, %v), want (%q, nil)", got, err, c1)
	}
	// A short SHA resolves to the full commit.
	if got, err := g.resolveRevision(ctx, repo, c1[:8]); err != nil || got != c1 {
		t.Fatalf("resolveRevision(%q) = (%q, %v), want (%q, nil)", c1[:8], got, err, c1)
	}
}

func TestResolveRevisionUnknown(t *testing.T) {
	repo := newTempRepo(t)
	g := newGitClient()

	if _, err := g.resolveRevision(context.Background(), repo, "no-such-revision"); !errors.Is(err, ErrUnknownRevision) {
		t.Fatalf("resolveRevision(unknown) error = %v, want ErrUnknownRevision", err)
	}
}

func TestResolveRevisionContextCancellation(t *testing.T) {
	repo := newTempRepo(t)
	g := newGitClient()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := g.resolveRevision(ctx, repo, "HEAD")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("resolveRevision(canceled) error = %v, want context.Canceled", err)
	}
	if errors.Is(err, ErrUnknownRevision) {
		t.Fatalf("cancellation must not be reported as ErrUnknownRevision: %v", err)
	}
}

func TestWorktreeRoundTripAtExactCommit(t *testing.T) {
	repo := newTempRepo(t)
	c1 := commitFile(t, repo, "hello.txt", "base\n", "c1")
	commitFile(t, repo, "hello.txt", "changed\n", "c2")

	g := newGitClient()
	ctx := context.Background()

	entries, err := g.worktreeList(ctx, repo)
	if err != nil {
		t.Fatalf("worktreeList: %v", err)
	}
	if len(entries) != 1 || !samePath(entries[0].Path, repo) {
		t.Fatalf("initial worktree list = %+v, want only the main checkout %s", entries, repo)
	}
	if entries[0].Detached {
		t.Fatalf("main checkout reported detached: %+v", entries[0])
	}

	wtPath := filepath.Join(t.TempDir(), "roundtrip")
	if err := g.worktreeAdd(ctx, repo, wtPath, c1); err != nil {
		t.Fatalf("worktreeAdd: %v", err)
	}
	if head := runGit(t, wtPath, "rev-parse", "HEAD"); head != c1 {
		t.Fatalf("worktree HEAD = %s, want %s", head, c1)
	}
	assertFileContent(t, filepath.Join(wtPath, "hello.txt"), "base\n")

	entries, err = g.worktreeList(ctx, repo)
	if err != nil {
		t.Fatalf("worktreeList: %v", err)
	}
	if len(entries) != 2 || !worktreeRegistered(entries, wtPath) {
		t.Fatalf("worktree list = %+v, want the registered worktree %s", entries, wtPath)
	}
	for _, e := range entries {
		if samePath(e.Path, wtPath) && !e.Detached {
			t.Fatalf("worktree %s not reported detached: %+v", wtPath, e)
		}
	}

	if err := g.worktreeRemove(ctx, repo, wtPath); err != nil {
		t.Fatalf("worktreeRemove: %v", err)
	}
	if pathExists(wtPath) {
		t.Fatalf("worktree directory %s still exists after remove", wtPath)
	}
	if err := g.worktreePrune(ctx, repo); err != nil {
		t.Fatalf("worktreePrune: %v", err)
	}
	entries, err = g.worktreeList(ctx, repo)
	if err != nil {
		t.Fatalf("worktreeList: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("worktree list after teardown = %+v, want only the main checkout", entries)
	}
}

func TestParseWorktreeList(t *testing.T) {
	out := "worktree C:/repo main\n" +
		"HEAD 1111111111111111111111111111111111111111\n" +
		"branch refs/heads/main\n" +
		"\n" +
		"worktree C:/temp/wt one\n" +
		"HEAD 2222222222222222222222222222222222222222\n" +
		"detached\n"

	entries := parseWorktreeList(out)
	if len(entries) != 2 {
		t.Fatalf("parseWorktreeList returned %d entries, want 2: %+v", len(entries), entries)
	}
	if entries[0].Path != "C:/repo main" ||
		entries[0].Head != "1111111111111111111111111111111111111111" ||
		entries[0].Detached {
		t.Fatalf("entry[0] = %+v, want main checkout on a branch", entries[0])
	}
	if entries[1].Path != "C:/temp/wt one" ||
		entries[1].Head != "2222222222222222222222222222222222222222" ||
		!entries[1].Detached {
		t.Fatalf("entry[1] = %+v, want detached worktree", entries[1])
	}
}

func TestParseWorktreeListEmpty(t *testing.T) {
	if entries := parseWorktreeList(""); len(entries) != 0 {
		t.Fatalf("parseWorktreeList(\"\") = %+v, want none", entries)
	}
}

func TestWorktreeRegisteredPathNormalization(t *testing.T) {
	entries := []worktreeInfo{{Path: "C:/Temp/wt-abc123"}}
	if !worktreeRegistered(entries, `C:\Temp\wt-abc123`) {
		t.Fatal("registered worktree not found across separator/case styles")
	}
	if worktreeRegistered(entries, `C:\Temp\other`) {
		t.Fatal("unrelated path reported as registered")
	}
}

// runGit executes one git command in dir and fails the test on any error.
func runGit(t *testing.T, dir string, argv ...string) string {
	t.Helper()
	local := runner.LocalRunner{}
	res, err := local.Run(context.Background(), runner.Command{
		Argv: append([]string{"git"}, argv...),
		Dir:  dir,
		Env:  gitEnv(),
	})
	if err != nil {
		t.Fatalf("git %v: %v", argv, err)
	}
	if res.ExitCode != 0 {
		t.Fatalf("git %v: exit %d: %s", argv, res.ExitCode, res.Stderr)
	}
	return strings.TrimSpace(res.Stdout)
}

// newTempRepo creates a throwaway local git repository with two-configured
// identity (global/system config is disabled by gitEnv) and no remote.
func newTempRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.name", "tally-test")
	runGit(t, dir, "config", "user.email", "tally-test@example.invalid")
	return dir
}

// commitFile writes a file, commits everything and returns the new HEAD SHA.
func commitFile(t *testing.T, repo, name, content, msg string) string {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", msg)
	return runGit(t, repo, "rev-parse", "HEAD")
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(data) != want {
		t.Fatalf("content of %s = %q, want %q", path, data, want)
	}
}
