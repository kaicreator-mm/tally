package changeset

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"tally/internal/core"
)

func TestFreezeWorktreeCapturesModifiedAndUntrackedFiles(t *testing.T) {
	repo, base := initFreezeRepo(t)
	if err := os.WriteFile(filepath.Join(repo, "existing.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "new.sh"), []byte("#!/bin/sh\necho ok\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	candidate, ok, err := FreezeWorktree(context.Background(), WorktreeFreezeSpec{
		Project:       core.ProjectID("project-1"),
		BaseRevision:  base,
		ProducerRun:   "task-1/attempt/001",
		WorkspacePath: repo,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ok || candidate == nil {
		t.Fatal("expected immutable candidate")
	}
	files := candidate.Files()
	if len(files) != 2 || files[0].Path != "existing.txt" || files[1].Path != "new.sh" {
		t.Fatalf("files=%+v", files)
	}
	if files[0].Content != "changed\n" || files[1].Content != "#!/bin/sh\necho ok\n" {
		t.Fatalf("unexpected frozen content: %+v", files)
	}
	if candidate.BaseRevision() != base || candidate.ProducerRun() != "task-1/attempt/001" {
		t.Fatalf("candidate lineage base=%s run=%s", candidate.BaseRevision(), candidate.ProducerRun())
	}
}

func TestFreezeWorktreeNoChangesIsNotCandidate(t *testing.T) {
	repo, base := initFreezeRepo(t)
	candidate, ok, err := FreezeWorktree(context.Background(), WorktreeFreezeSpec{
		Project:       core.ProjectID("project-1"),
		BaseRevision:  base,
		ProducerRun:   "run-1",
		WorkspacePath: repo,
	})
	if err != nil {
		t.Fatal(err)
	}
	if ok || candidate != nil {
		t.Fatalf("expected no candidate, got ok=%v candidate=%v", ok, candidate)
	}
}

func TestFreezeWorktreeRejectsDeletion(t *testing.T) {
	repo, base := initFreezeRepo(t)
	if err := os.Remove(filepath.Join(repo, "existing.txt")); err != nil {
		t.Fatal(err)
	}
	_, _, err := FreezeWorktree(context.Background(), WorktreeFreezeSpec{
		Project:       core.ProjectID("project-1"),
		BaseRevision:  base,
		ProducerRun:   "run-1",
		WorkspacePath: repo,
	})
	if !errors.Is(err, ErrCandidateDeletion) {
		t.Fatalf("expected ErrCandidateDeletion, got %v", err)
	}
}

func initFreezeRepo(t *testing.T) (string, string) {
	t.Helper()
	repo := t.TempDir()
	runGitFreeze(t, repo, "init", "-q")
	runGitFreeze(t, repo, "config", "user.email", "tally@example.invalid")
	runGitFreeze(t, repo, "config", "user.name", "Tally Test")
	if err := os.WriteFile(filepath.Join(repo, "existing.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGitFreeze(t, repo, "add", "existing.txt")
	runGitFreeze(t, repo, "commit", "-q", "-m", "base")
	base := runGitFreeze(t, repo, "rev-parse", "HEAD")
	return repo, base
}

func runGitFreeze(t *testing.T, repo string, args ...string) string {
	t.Helper()
	argv := append([]string{"-C", repo}, args...)
	cmd := exec.Command("git", argv...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return string(bytesTrimSpace(out))
}

func bytesTrimSpace(in []byte) []byte {
	start, end := 0, len(in)
	for start < end && (in[start] == ' ' || in[start] == '\n' || in[start] == '\r' || in[start] == '\t') {
		start++
	}
	for end > start && (in[end-1] == ' ' || in[end-1] == '\n' || in[end-1] == '\r' || in[end-1] == '\t') {
		end--
	}
	return in[start:end]
}
