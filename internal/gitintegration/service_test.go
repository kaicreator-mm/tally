package gitintegration

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/gitwriter"
)

// runTestGit / initRepo / 各 helper 与 internal/gitwriter/writer_test.go
// 的既有约定一致:真实临时 Git 仓库,不用 mock。

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
	runTestGit(t, dir, "config", "user.name", "Tally Test")
	runTestGit(t, dir, "config", "user.email", "test@tally.local")
	return dir
}

func seedCommit(t *testing.T, dir string) string {
	t.Helper()
	runTestGit(t, dir, "commit", "--allow-empty", "-q", "-m", "seed")
	return strings.TrimSpace(runTestGit(t, dir, "rev-parse", "HEAD"))
}

// acceptedChangeset 构造一个带验收事实的已识别 ChangeSet。
func acceptedChangeset(t *testing.T, base, run string, files ...changeset.ChangedFile) (*changeset.ChangeSet, AcceptanceFact) {
	t.Helper()
	cs, err := changeset.New(changeset.Spec{
		Project:           core.ProjectID("proj"),
		BaseRevision:      base,
		ProducerRun:       run,
		Files:             files,
		CandidateRevision: "candidate-" + run,
	})
	if err != nil {
		t.Fatalf("changeset.New: %v", err)
	}
	return cs, AcceptanceFact{
		ChangeSetID: cs.ID(),
		AcceptedAt:  time.Now().UTC(),
		Basis:       "acceptance-event:test",
	}
}

func newTestService(t *testing.T, primary string) *Service {
	t.Helper()
	s, err := New(Config{PrimaryWorktree: primary})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func statusMustBeClean(t *testing.T, dir string) {
	t.Helper()
	if out := strings.TrimSpace(runTestGit(t, dir, "status", "--porcelain")); out != "" {
		t.Fatalf("work tree/index not clean after rejected integration:\n%s", out)
	}
}

func TestNewRequiresPrimaryWorktree(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("New without PrimaryWorktree succeeded, want error")
	}
}

func TestIntegrateAcceptedChangeset(t *testing.T) {
	dir := initRepo(t)
	base := seedCommit(t, dir)

	cs, fact := acceptedChangeset(t, base, "run-1",
		changeset.ChangedFile{Path: "svc/main.go", Content: "package main\n"},
		changeset.ChangedFile{Path: "svc/util.go", Content: "package svc\n"},
	)
	svc := newTestService(t, t.TempDir()) // 主工作树是另一个目录

	res, err := svc.Integrate(context.Background(), IntegrateRequest{
		ChangeSet:      cs,
		Acceptance:     fact,
		RepositoryPath: dir,
		ExpectedBase:   base,
	})
	if err != nil {
		t.Fatalf("Integrate: %v", err)
	}
	if res.Base != base {
		t.Fatalf("res.Base = %q, want %q", res.Base, base)
	}
	if res.ChangeSetID != cs.ID() {
		t.Fatalf("res.ChangeSetID = %q, want %q", res.ChangeSetID, cs.ID())
	}
	if res.Commit != strings.TrimSpace(runTestGit(t, dir, "rev-parse", "HEAD")) {
		t.Fatalf("res.Commit %q is not the new HEAD", res.Commit)
	}
	if got := runTestGit(t, dir, "show", res.Commit+":svc/main.go"); got != "package main\n" {
		t.Fatalf("integrated content = %q", got)
	}
	// 提交信息是语义化且确定性的,含候选身份。
	msg := runTestGit(t, dir, "log", "-1", "--format=%s")
	if !strings.Contains(msg, string(cs.ID())) {
		t.Fatalf("commit message %q must cite the changeset id", msg)
	}
	statusMustBeClean(t, dir)
}

func TestIntegrateRejectsNilChangeset(t *testing.T) {
	dir := initRepo(t)
	base := seedCommit(t, dir)
	svc := newTestService(t, t.TempDir())

	_, err := svc.Integrate(context.Background(), IntegrateRequest{
		RepositoryPath: dir,
		ExpectedBase:   base,
	})
	if !errors.Is(err, ErrCandidateUnidentified) {
		t.Fatalf("nil ChangeSet: got %v, want ErrCandidateUnidentified", err)
	}
	if h := strings.TrimSpace(runTestGit(t, dir, "rev-parse", "HEAD")); h != base {
		t.Fatalf("HEAD moved to %q after rejection", h)
	}
	statusMustBeClean(t, dir)
}

func TestIntegrateRejectsMissingOrIncompleteAcceptance(t *testing.T) {
	dir := initRepo(t)
	base := seedCommit(t, dir)
	cs, fact := acceptedChangeset(t, base, "run-2",
		changeset.ChangedFile{Path: "a.txt", Content: "a"},
	)
	svc := newTestService(t, t.TempDir())
	ctx := context.Background()

	cases := map[string]AcceptanceFact{
		"zero fact":    {},
		"no id":        {AcceptedAt: fact.AcceptedAt, Basis: fact.Basis},
		"no timestamp": {ChangeSetID: fact.ChangeSetID, Basis: fact.Basis},
		"no basis":     {ChangeSetID: fact.ChangeSetID, AcceptedAt: fact.AcceptedAt},
	}
	for name, f := range cases {
		_, err := svc.Integrate(ctx, IntegrateRequest{
			ChangeSet:      cs,
			Acceptance:     f,
			RepositoryPath: dir,
			ExpectedBase:   base,
		})
		if !errors.Is(err, ErrNotAccepted) {
			t.Fatalf("%s: got %v, want ErrNotAccepted", name, err)
		}
	}
	if h := strings.TrimSpace(runTestGit(t, dir, "rev-parse", "HEAD")); h != base {
		t.Fatalf("HEAD moved after rejected integrations")
	}
	statusMustBeClean(t, dir)
}

func TestIntegrateRejectsAcceptanceMismatch(t *testing.T) {
	dir := initRepo(t)
	base := seedCommit(t, dir)
	cs, _ := acceptedChangeset(t, base, "run-3",
		changeset.ChangedFile{Path: "a.txt", Content: "a"},
	)
	// 验收事实指向另一个候选身份。
	fact := AcceptanceFact{
		ChangeSetID: changeset.ChangeSetID("sha256:deadbeef"),
		AcceptedAt:  time.Now().UTC(),
		Basis:       "acceptance-event:other",
	}
	svc := newTestService(t, t.TempDir())

	_, err := svc.Integrate(context.Background(), IntegrateRequest{
		ChangeSet:      cs,
		Acceptance:     fact,
		RepositoryPath: dir,
		ExpectedBase:   base,
	})
	if !errors.Is(err, ErrAcceptanceMismatch) {
		t.Fatalf("got %v, want ErrAcceptanceMismatch", err)
	}
	if h := strings.TrimSpace(runTestGit(t, dir, "rev-parse", "HEAD")); h != base {
		t.Fatalf("HEAD moved after mismatch rejection")
	}
}

func TestIntegrateRejectsStaleBase(t *testing.T) {
	dir := initRepo(t)
	base := seedCommit(t, dir)
	cs, fact := acceptedChangeset(t, base, "run-4",
		changeset.ChangedFile{Path: "a.txt", Content: "a"},
	)
	svc := newTestService(t, t.TempDir())

	// 目标在快照固定之后前进了一步:基线过期是预期类型化冲突。
	runTestGit(t, dir, "commit", "--allow-empty", "-q", "-m", "concurrent")
	_, err := svc.Integrate(context.Background(), IntegrateRequest{
		ChangeSet:      cs,
		Acceptance:     fact,
		RepositoryPath: dir,
		ExpectedBase:   base,
	})
	if !errors.Is(err, ErrStaleBase) {
		t.Fatalf("got %v, want ErrStaleBase", err)
	}
	if !errors.Is(err, gitwriter.ErrBaseRevision) {
		t.Fatalf("ErrStaleBase must wrap gitwriter.ErrBaseRevision, got %v", err)
	}
}

func TestIntegrateRejectsEmptyExpectedBase(t *testing.T) {
	dir := initRepo(t)
	base := seedCommit(t, dir)
	cs, fact := acceptedChangeset(t, base, "run-5",
		changeset.ChangedFile{Path: "a.txt", Content: "a"},
	)
	svc := newTestService(t, t.TempDir())

	_, err := svc.Integrate(context.Background(), IntegrateRequest{
		ChangeSet:      cs,
		Acceptance:     fact,
		RepositoryPath: dir,
		ExpectedBase:   "",
	})
	if !errors.Is(err, ErrNoExpectedBase) {
		t.Fatalf("got %v, want ErrNoExpectedBase", err)
	}
}

func TestIntegrateRejectsUnbornTargetWithoutBase(t *testing.T) {
	dir := initRepo(t) // 无任何提交
	cs, fact := acceptedChangeset(t, "0123456789abcdef0123456789abcdef01234567", "run-6",
		changeset.ChangedFile{Path: "a.txt", Content: "a"},
	)
	svc := newTestService(t, t.TempDir())

	_, err := svc.Integrate(context.Background(), IntegrateRequest{
		ChangeSet:      cs,
		Acceptance:     fact,
		RepositoryPath: dir,
		ExpectedBase:   strings.TrimSpace(""), // 显式空 base 同样拒绝
	})
	if !errors.Is(err, ErrNoExpectedBase) {
		t.Fatalf("got %v, want ErrNoExpectedBase", err)
	}
}

func TestIntegrateRejectsProtectedPrimaryWorktree(t *testing.T) {
	dir := initRepo(t)
	base := seedCommit(t, dir)
	cs, fact := acceptedChangeset(t, base, "run-7",
		changeset.ChangedFile{Path: "a.txt", Content: "a"},
	)
	// 主工作树声明为集成目标本身 —— 必须拒绝。
	svc := newTestService(t, dir)

	_, err := svc.Integrate(context.Background(), IntegrateRequest{
		ChangeSet:      cs,
		Acceptance:     fact,
		RepositoryPath: dir,
		ExpectedBase:   base,
	})
	if !errors.Is(err, ErrProtectedTarget) {
		t.Fatalf("got %v, want ErrProtectedTarget", err)
	}
	if h := strings.TrimSpace(runTestGit(t, dir, "rev-parse", "HEAD")); h != base {
		t.Fatalf("HEAD moved on protected worktree")
	}
}

func TestIntegrateRejectsNoApplicableContent(t *testing.T) {
	dir := initRepo(t)
	base := seedCommit(t, dir)
	// 只有 candidate revision、无文件内容的快照:无法经全量内容模型应用。
	cs, err := changeset.New(changeset.Spec{
		Project:           core.ProjectID("proj"),
		BaseRevision:      base,
		CandidateRevision: "candidate-ref-only",
		ProducerRun:       "run-8",
	})
	if err != nil {
		t.Fatalf("changeset.New: %v", err)
	}
	fact := AcceptanceFact{
		ChangeSetID: cs.ID(),
		AcceptedAt:  time.Now().UTC(),
		Basis:       "acceptance-event:test",
	}
	svc := newTestService(t, t.TempDir())

	_, err = svc.Integrate(context.Background(), IntegrateRequest{
		ChangeSet:      cs,
		Acceptance:     fact,
		RepositoryPath: dir,
		ExpectedBase:   base,
	})
	if !errors.Is(err, ErrNoApplicableContent) {
		t.Fatalf("got %v, want ErrNoApplicableContent", err)
	}
}

func TestIntegrateRequiresRepositoryPath(t *testing.T) {
	dir := initRepo(t)
	base := seedCommit(t, dir)
	cs, fact := acceptedChangeset(t, base, "run-9",
		changeset.ChangedFile{Path: "a.txt", Content: "a"},
	)
	svc := newTestService(t, t.TempDir())

	if _, err := svc.Integrate(context.Background(), IntegrateRequest{
		ChangeSet:    cs,
		Acceptance:   fact,
		ExpectedBase: base,
	}); !errors.Is(err, ErrNoRepository) {
		t.Fatalf("got %v, want ErrNoRepository", err)
	}
}

func TestIntegrateOnNonRepositoryFails(t *testing.T) {
	plain := t.TempDir() // 非 Git 仓库
	cs, fact := acceptedChangeset(t, "0123456789abcdef0123456789abcdef01234567", "run-10",
		changeset.ChangedFile{Path: "a.txt", Content: "a"},
	)
	svc := newTestService(t, t.TempDir())

	_, err := svc.Integrate(context.Background(), IntegrateRequest{
		ChangeSet:      cs,
		Acceptance:     fact,
		RepositoryPath: plain,
		ExpectedBase:   "0123456789abcdef0123456789abcdef01234567",
	})
	if !errors.Is(err, gitwriter.ErrNotRepository) {
		t.Fatalf("got %v, want gitwriter.ErrNotRepository", err)
	}
	// 失败不得在目标目录留下任何文件。
	if _, statErr := os.Stat(filepath.Join(plain, "a.txt")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("rejected integration wrote files: %v", statErr)
	}
}

func TestIntegrateNoChangesSurfacesTypedNoChanges(t *testing.T) {
	dir := initRepo(t)
	runTestGit(t, dir, "commit", "--allow-empty", "-q", "-m", "seed with file")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), nil, 0o644); err != nil {
		t.Fatalf("write a.txt: %v", err)
	}
	runTestGit(t, dir, "add", "a.txt")
	runTestGit(t, dir, "commit", "-q", "-m", "add a.txt")
	base := strings.TrimSpace(runTestGit(t, dir, "rev-parse", "HEAD"))

	// 快照内容与目标当前内容一致 → gitwriter 报 ErrNoChanges,原样透传。
	cs, fact := acceptedChangeset(t, base, "run-11",
		changeset.ChangedFile{Path: "a.txt", Content: ""},
	)
	svc := newTestService(t, t.TempDir())
	_, err := svc.Integrate(context.Background(), IntegrateRequest{
		ChangeSet:      cs,
		Acceptance:     fact,
		RepositoryPath: dir,
		ExpectedBase:   base,
	})
	if !errors.Is(err, gitwriter.ErrNoChanges) {
		t.Fatalf("got %v, want gitwriter.ErrNoChanges", err)
	}
	statusMustBeClean(t, dir)
}
