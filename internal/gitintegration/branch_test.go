package gitintegration

// 集成分支创建(P5-02)的测试:真实临时 Git 仓库,helper 与
// service_test.go / internal/gitwriter/writer_test.go 的既有约定一致。

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/gitwriter"
)

// acceptedCandidate 构造一个只有 candidate revision、无文件内容的已接受
// 候选:分支创建只依赖身份与 base,不依赖可应用内容(内容的应用是
// Integrate 的事)。
func acceptedCandidate(t *testing.T, base, run string) (*changeset.ChangeSet, AcceptanceFact) {
	t.Helper()
	cs, err := changeset.New(changeset.Spec{
		Project:           core.ProjectID("proj"),
		BaseRevision:      base,
		CandidateRevision: "candidate-" + run,
		ProducerRun:       run,
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

// expectedBranchRef 按包内命名约定推导期望 ref 名,与实现互为印证。
func expectedBranchRef(t *testing.T, cs *changeset.ChangeSet) string {
	t.Helper()
	hex := strings.TrimPrefix(string(cs.ID()), "sha256:")
	if len(hex) != 64 {
		t.Fatalf("unexpected changeset id %q", cs.ID())
	}
	return "refs/heads/tally/integration/" + hex
}

// refExists 容忍 ref 不存在的探测(runTestGit 在失败时会 Fatal)。
func refExists(t *testing.T, dir, ref string) bool {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--verify", "--quiet", ref)
	return cmd.Run() == nil
}

func TestEnsureIntegrationBranchCreatesAtBase(t *testing.T) {
	dir := initRepo(t)
	base := seedCommit(t, dir)
	cs, fact := acceptedCandidate(t, base, "br-1")
	svc := newTestService(t, t.TempDir())

	res, err := svc.EnsureIntegrationBranch(context.Background(), IntegrationBranchRequest{
		ChangeSet:      cs,
		Acceptance:     fact,
		RepositoryPath: dir,
		Authorization:  testAuth(t, expectedBranchRef(t, cs)),
	})
	if err != nil {
		t.Fatalf("EnsureIntegrationBranch: %v", err)
	}
	if !res.Created {
		t.Fatal("first ensure should create the branch")
	}
	if res.ChangeSetID != cs.ID() {
		t.Fatalf("res.ChangeSetID = %q, want %q", res.ChangeSetID, cs.ID())
	}
	if res.Branch != expectedBranchRef(t, cs) {
		t.Fatalf("res.Branch = %q, want deterministic %q", res.Branch, expectedBranchRef(t, cs))
	}
	if res.Base != base {
		t.Fatalf("res.Base = %q, want %q", res.Base, base)
	}
	if got := strings.TrimSpace(runTestGit(t, dir, "rev-parse", "--verify", res.Branch)); got != base {
		t.Fatalf("branch points at %q, want base %q", got, base)
	}
	// 分支创建不触碰 HEAD、不检出任何内容。
	if h := strings.TrimSpace(runTestGit(t, dir, "rev-parse", "HEAD")); h != base {
		t.Fatalf("HEAD moved to %q", h)
	}
	statusMustBeClean(t, dir)
}

func TestEnsureIntegrationBranchIsIdempotent(t *testing.T) {
	dir := initRepo(t)
	base := seedCommit(t, dir)
	cs, fact := acceptedCandidate(t, base, "br-2")
	svc := newTestService(t, t.TempDir())
	req := IntegrationBranchRequest{ChangeSet: cs, Acceptance: fact, RepositoryPath: dir, Authorization: testAuth(t, expectedBranchRef(t, cs))}

	first, err := svc.EnsureIntegrationBranch(context.Background(), req)
	if err != nil || !first.Created {
		t.Fatalf("first ensure: created=%v err=%v", first.Created, err)
	}

	// 同一确定性输入的第二次创建是幂等确认,不是错误,也不改 ref。
	second, err := svc.EnsureIntegrationBranch(context.Background(), req)
	if err != nil {
		t.Fatalf("second ensure: %v", err)
	}
	if second.Created {
		t.Fatal("second ensure must not report creation")
	}
	if second.Branch != first.Branch || second.Base != first.Base {
		t.Fatalf("second ensure drifted: %+v vs %+v", second, first)
	}
	if got := strings.TrimSpace(runTestGit(t, dir, "rev-parse", "--verify", first.Branch)); got != base {
		t.Fatalf("branch moved to %q, want %q", got, base)
	}
	statusMustBeClean(t, dir)
}

func TestEnsureIntegrationBranchRejectsDriftedBranch(t *testing.T) {
	dir := initRepo(t)
	base := seedCommit(t, dir)
	cs, fact := acceptedCandidate(t, base, "br-3")
	svc := newTestService(t, t.TempDir())
	ref := expectedBranchRef(t, cs)

	// 目标分支已存在且演进到别处:这是预期的类型化冲突。
	runTestGit(t, dir, "commit", "--allow-empty", "-q", "-m", "later")
	other := strings.TrimSpace(runTestGit(t, dir, "rev-parse", "HEAD"))
	runTestGit(t, dir, "update-ref", ref, other)

	_, err := svc.EnsureIntegrationBranch(context.Background(), IntegrationBranchRequest{
		ChangeSet:      cs,
		Acceptance:     fact,
		RepositoryPath: dir,
		Authorization:  testAuth(t, ref),
	})
	if !errors.Is(err, ErrIntegrationBranchExists) {
		t.Fatalf("got %v, want ErrIntegrationBranchExists", err)
	}
	// 不覆盖:ref 仍指向 other,主工作树 HEAD 也不动。
	if cur := strings.TrimSpace(runTestGit(t, dir, "rev-parse", "--verify", ref)); cur != other {
		t.Fatalf("branch was overwritten: %q, want %q", cur, other)
	}
	if h := strings.TrimSpace(runTestGit(t, dir, "rev-parse", "HEAD")); h != other {
		t.Fatalf("HEAD moved to %q", h)
	}
}

func TestEnsureIntegrationBranchRejectsMissingBase(t *testing.T) {
	dir := initRepo(t)
	seedCommit(t, dir)
	cs, fact := acceptedCandidate(t, "0123456789abcdef0123456789abcdef01234567", "br-4")
	svc := newTestService(t, t.TempDir())

	_, err := svc.EnsureIntegrationBranch(context.Background(), IntegrationBranchRequest{
		ChangeSet:      cs,
		Acceptance:     fact,
		RepositoryPath: dir,
		Authorization:  testAuth(t, expectedBranchRef(t, cs)),
	})
	if !errors.Is(err, ErrBaseNotFound) {
		t.Fatalf("got %v, want ErrBaseNotFound", err)
	}
	// 失败安全:分支没有被创建。
	if refExists(t, dir, expectedBranchRef(t, cs)) {
		t.Fatal("branch was created despite missing base")
	}
}

func TestEnsureIntegrationBranchValidatesLikeIntegrate(t *testing.T) {
	dir := initRepo(t)
	base := seedCommit(t, dir)
	cs, fact := acceptedCandidate(t, base, "br-5")
	ref := expectedBranchRef(t, cs)

	cases := []struct {
		name    string
		req     IntegrationBranchRequest
		protect bool // 主工作树声明为目标本身,验证受保护拒绝
		wantErr error
	}{
		{name: "nil changeset", req: IntegrationBranchRequest{Acceptance: fact, RepositoryPath: dir}, wantErr: ErrCandidateUnidentified},
		{name: "zero acceptance", req: IntegrationBranchRequest{ChangeSet: cs, RepositoryPath: dir, Authorization: testAuth(t, ref)}, wantErr: ErrNotAccepted},
		{name: "mismatched acceptance", req: IntegrationBranchRequest{ChangeSet: cs, Authorization: testAuth(t, ref), Acceptance: AcceptanceFact{
			ChangeSetID: changeset.ChangeSetID("sha256:deadbeef"),
			AcceptedAt:  time.Now().UTC(),
			Basis:       "acceptance-event:other",
		}, RepositoryPath: dir}, wantErr: ErrAcceptanceMismatch},
		{name: "missing repository path", req: IntegrationBranchRequest{ChangeSet: cs, Acceptance: fact}, wantErr: ErrNoRepository},
		{name: "protected primary worktree", req: IntegrationBranchRequest{ChangeSet: cs, Acceptance: fact, RepositoryPath: dir}, protect: true, wantErr: ErrProtectedTarget},
	}
	for _, tc := range cases {
		primary := t.TempDir() // 其余用例:主工作树是独立目录
		if tc.protect {
			primary = dir
		}
		svc := newTestService(t, primary)
		_, err := svc.EnsureIntegrationBranch(context.Background(), tc.req)
		if !errors.Is(err, tc.wantErr) {
			t.Fatalf("%s: got %v, want %v", tc.name, err, tc.wantErr)
		}
		// 每一次拒绝都不得创建分支或移动 HEAD。
		if refExists(t, dir, ref) {
			t.Fatalf("%s: branch was created on rejection", tc.name)
		}
		if h := strings.TrimSpace(runTestGit(t, dir, "rev-parse", "HEAD")); h != base {
			t.Fatalf("%s: HEAD moved to %q", tc.name, h)
		}
	}
}

func TestEnsureIntegrationBranchOnNonRepositoryFails(t *testing.T) {
	plain := t.TempDir() // 非 Git 仓库
	cs, fact := acceptedCandidate(t, "0123456789abcdef0123456789abcdef01234567", "br-6")
	svc := newTestService(t, t.TempDir())

	_, err := svc.EnsureIntegrationBranch(context.Background(), IntegrationBranchRequest{
		ChangeSet:      cs,
		Acceptance:     fact,
		RepositoryPath: plain,
		Authorization:  testAuth(t, expectedBranchRef(t, cs)),
	})
	if !errors.Is(err, gitwriter.ErrNotRepository) {
		t.Fatalf("got %v, want gitwriter.ErrNotRepository", err)
	}
}

func TestEnsureIntegrationBranchCancelledContext(t *testing.T) {
	dir := initRepo(t)
	base := seedCommit(t, dir)
	cs, fact := acceptedCandidate(t, base, "br-7")
	svc := newTestService(t, t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.EnsureIntegrationBranch(ctx, IntegrationBranchRequest{
		ChangeSet:      cs,
		Acceptance:     fact,
		RepositoryPath: dir,
		Authorization:  testAuth(t, expectedBranchRef(t, cs)),
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	// 取消不得被伪造成分支结果,也不得留下分支。
	if refExists(t, dir, expectedBranchRef(t, cs)) {
		t.Fatal("branch was created on cancelled context")
	}
}
