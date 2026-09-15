package gitengineering

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tally/internal/runner"
)

type fakeRunner struct {
	head     string
	base     string
	pushExit int
	calls    [][]string
}

func (f *fakeRunner) Run(_ context.Context, cmd runner.Command) (runner.Result, error) {
	f.calls = append(f.calls, append([]string(nil), cmd.Argv...))
	joined := strings.Join(cmd.Argv, " ")
	switch {
	case strings.Contains(joined, "check-ref-format --branch"):
		return runner.Result{}, nil
	case strings.Contains(joined, "rev-parse") && strings.Contains(joined, "HEAD^{commit}"):
		return runner.Result{Stdout: f.head + "\n"}, nil
	case strings.Contains(joined, "rev-parse"):
		return runner.Result{Stdout: f.base + "\n"}, nil
	case strings.Contains(joined, "push --porcelain"):
		return runner.Result{ExitCode: f.pushExit, Stderr: "push rejected"}, nil
	default:
		return runner.Result{}, nil
	}
}

func TestLocalPushRejectsStaleHeadBeforeNetwork(t *testing.T) {
	f := &fakeRunner{head: "actual", base: "base"}
	a := NewLocalGitAdapter(f)
	_, err := a.Push(context.Background(), "/repo", "origin", "feature", "expected")
	if !errors.Is(err, ErrStaleHead) {
		t.Fatalf("got %v", err)
	}
	for _, c := range f.calls {
		if len(c) > 1 && c[1] == "push" {
			t.Fatal("push executed after stale head")
		}
	}
}

func TestLocalPushFailureIsEvidenceBearingError(t *testing.T) {
	f := &fakeRunner{head: "head", pushExit: 1}
	a := NewLocalGitAdapter(f)
	_, err := a.Push(context.Background(), "/repo", "origin", "feature", "head")
	if !errors.Is(err, ErrGitCommand) {
		t.Fatalf("got %v", err)
	}
}

func TestLocalCreateBranchUsesResolvedBase(t *testing.T) {
	f := &fakeRunner{base: "base-sha"}
	a := NewLocalGitAdapter(f)
	rev, err := a.CreateBranch(context.Background(), "/repo", "feat/x", "main")
	if err != nil {
		t.Fatal(err)
	}
	if rev != "base-sha" {
		t.Fatalf("rev=%s", rev)
	}
	found := false
	for _, c := range f.calls {
		if strings.Join(c, " ") == "git branch --no-track feat/x base-sha" {
			found = true
		}
	}
	if !found {
		t.Fatalf("branch command not found: %+v", f.calls)
	}
}
