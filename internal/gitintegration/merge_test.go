package gitintegration

// 安全本地合并(P5-03)的测试:真实临时 Git 仓库,helper 与
// service_test.go / branch_test.go 的既有约定一致。覆盖任务验收要求的
// 干净合并 / 冲突 / base 漂移,以及快进、审计 ref 保全与全部失败路径。

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tally/internal/changeset"
	"tally/internal/gitwriter"
)

// mergeFixture 构造一个可合并的仓库形态:
//
//	base ── t1 (refs/heads/target 上的目标侧提交)
//	 └──── b1 (refs/heads/tally/integration/<hex> 上的候选侧提交)
//
// 两侧都在分离 HEAD 下构造,目标 ref 与集成分支都只经 update-ref 创建,
// 从不被检出 —— 这正是受控合并的前提形态。
type mergeFixture struct {
	dir    string
	base   string
	target string // refs/heads/target 当前指向的提交(t1)
	tip    string // 集成分支当前指向的提交(b1)
	ref    string // 确定性集成分支 ref
	cs     *changeset.ChangeSet
	fact   AcceptanceFact
}

func newMergeFixture(t *testing.T, run string, targetFiles, branchFiles map[string]string) *mergeFixture {
	t.Helper()
	dir := initRepo(t)
	base := seedCommit(t, dir)

	// 目标侧无文件时让目标 ref 停在 base 本身(真快进形态);
	// 有文件时在 base 之上造独立提交(真合并/冲突形态)。
	t1 := base
	if len(targetFiles) > 0 {
		t1 = commitDetached(t, dir, base, "target side", targetFiles)
	}
	runTestGit(t, dir, "update-ref", "refs/heads/target", t1)

	var files []changeset.ChangedFile
	for p, c := range branchFiles {
		files = append(files, changeset.ChangedFile{Path: p, Content: c})
	}
	b1 := commitDetached(t, dir, base, "branch side", branchFiles)
	cs, fact := acceptedChangeset(t, base, run, files...)
	ref := expectedBranchRef(t, cs)
	runTestGit(t, dir, "update-ref", ref, b1)

	return &mergeFixture{dir: dir, base: base, target: t1, tip: b1, ref: ref, cs: cs, fact: fact}
}

// commitDetached 在 parent 之上以分离 HEAD 造一个提交(写给定文件),
// 返回新提交 OID。工作树留在分离 HEAD 上 —— 合并从不依赖 HEAD。
func commitDetached(t *testing.T, dir, parent, msg string, files map[string]string) string {
	t.Helper()
	runTestGit(t, dir, "checkout", "-q", "--detach", parent)
	for p, c := range files {
		full := filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", p, err)
		}
		if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
			t.Fatalf("write %s: %v", p, err)
		}
		runTestGit(t, dir, "add", "--", p)
	}
	runTestGit(t, dir, "commit", "-q", "--allow-empty", "-m", msg)
	return strings.TrimSpace(runTestGit(t, dir, "rev-parse", "HEAD"))
}

// revIsAncestor 容忍「不是祖先」的探测(runTestGit 在失败时会 Fatal)。
func revIsAncestor(t *testing.T, dir, a, b string) bool {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "merge-base", "--is-ancestor", a, b)
	return cmd.Run() == nil
}

func refPointsAt(t *testing.T, dir, ref, want string) bool {
	t.Helper()
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--verify", "--quiet", ref)
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == want
}

func TestMergeCleanTrueMerge(t *testing.T) {
	f := newMergeFixture(t, "mg-1",
		map[string]string{"g.txt": "ours"},
		map[string]string{"svc/f.txt": "theirs"},
	)
	svc := newTestService(t, t.TempDir())
	headBefore := strings.TrimSpace(runTestGit(t, f.dir, "rev-parse", "HEAD"))

	res, err := svc.Merge(context.Background(), MergeRequest{
		ChangeSet:            f.cs,
		Acceptance:           f.fact,
		RepositoryPath:       f.dir,
		TargetRef:            "refs/heads/target",
		Authorization:        testAuth(t, "refs/heads/target"),
		ExpectedTarget:       f.target,
		ExpectedBranchCommit: f.tip,
	})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if res.FastForward {
		t.Fatal("independent sides must produce a true merge, not a fast-forward")
	}
	if res.Commit == "" || res.OldTarget != f.target {
		t.Fatalf("unexpected result: %+v", res)
	}
	if !refPointsAt(t, f.dir, "refs/heads/target", res.Commit) {
		t.Fatalf("target ref is not at the merge commit %s", res.Commit)
	}
	// 双亲合并提交:两侧提交都是新提交的父,且历史只前进不重写。
	parents := strings.Fields(runTestGit(t, f.dir, "rev-list", "--parents", "-n", "1", res.Commit))
	if len(parents) != 3 || parents[1] != f.target || parents[2] != f.tip {
		t.Fatalf("merge commit parents = %v, want [%s %s]", parents, f.target, f.tip)
	}
	if !revIsAncestor(t, f.dir, f.target, res.Commit) || !revIsAncestor(t, f.dir, f.tip, res.Commit) {
		t.Fatal("both sides must remain ancestors (history advance, no rewrite)")
	}
	if got := runTestGit(t, f.dir, "show", res.Commit+":g.txt"); got != "ours" {
		t.Fatalf("merge tree g.txt = %q", got)
	}
	if got := runTestGit(t, f.dir, "show", res.Commit+":svc/f.txt"); got != "theirs" {
		t.Fatalf("merge tree svc/f.txt = %q", got)
	}
	// 提交信息语义化、确定性,引用候选身份。
	msg := strings.TrimSpace(runTestGit(t, f.dir, "log", "-1", "--format=%s", res.Commit))
	if msg != "tally: merge changeset "+string(f.cs.ID()) {
		t.Fatalf("merge message = %q", msg)
	}
	// 审计 ref 保全:集成分支原样保留;工作树/索引未被触碰。
	if !refPointsAt(t, f.dir, f.ref, f.tip) {
		t.Fatal("integration branch was modified by merge")
	}
	if h := strings.TrimSpace(runTestGit(t, f.dir, "rev-parse", "HEAD")); h != headBefore {
		t.Fatalf("HEAD moved to %q", h)
	}
	statusMustBeClean(t, f.dir)
}

func TestMergeFastForwardAdvancesWithoutNewCommit(t *testing.T) {
	f := newMergeFixture(t, "mg-2",
		map[string]string{}, // 目标停在 base
		map[string]string{"f.txt": "theirs"},
	)
	if f.target != f.base {
		t.Fatalf("fixture target %s, want base %s", f.target, f.base)
	}
	svc := newTestService(t, t.TempDir())

	res, err := svc.Merge(context.Background(), MergeRequest{
		ChangeSet:            f.cs,
		Acceptance:           f.fact,
		RepositoryPath:       f.dir,
		TargetRef:            "refs/heads/target",
		Authorization:        testAuth(t, "refs/heads/target"),
		ExpectedTarget:       f.target,
		ExpectedBranchCommit: f.tip,
	})
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if !res.FastForward || res.Commit != f.tip {
		t.Fatalf("result = %+v, want fast-forward to %s", res, f.tip)
	}
	if !refPointsAt(t, f.dir, "refs/heads/target", f.tip) {
		t.Fatal("target ref did not fast-forward to branch tip")
	}
	// 快进不造新提交:目标历史仍是 base + b1 两个提交。
	if n := strings.TrimSpace(runTestGit(t, f.dir, "rev-list", "--count", "refs/heads/target")); n != "2" {
		t.Fatalf("target history has %s commits, want 2 (no new commit)", n)
	}
	if !revIsAncestor(t, f.dir, f.base, f.tip) {
		t.Fatal("fast-forward must not rewrite history")
	}
	if !refPointsAt(t, f.dir, f.ref, f.tip) {
		t.Fatal("integration branch was modified by merge")
	}
	statusMustBeClean(t, f.dir)
}

func TestMergeConflictIsTypedWithoutPartialState(t *testing.T) {
	f := newMergeFixture(t, "mg-3",
		map[string]string{"c.txt": "ours"},
		map[string]string{"c.txt": "theirs"},
	)
	svc := newTestService(t, t.TempDir())
	headBefore := strings.TrimSpace(runTestGit(t, f.dir, "rev-parse", "HEAD"))

	_, err := svc.Merge(context.Background(), MergeRequest{
		ChangeSet:            f.cs,
		Acceptance:           f.fact,
		RepositoryPath:       f.dir,
		TargetRef:            "refs/heads/target",
		Authorization:        testAuth(t, "refs/heads/target"),
		ExpectedTarget:       f.target,
		ExpectedBranchCommit: f.tip,
	})
	if !errors.Is(err, ErrMergeConflict) {
		t.Fatalf("got %v, want ErrMergeConflict", err)
	}
	var conflict *MergeConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("got %v, want *MergeConflictError carrying paths", err)
	}
	if len(conflict.Paths) != 1 || conflict.Paths[0] != "c.txt" {
		t.Fatalf("conflict paths = %v, want [c.txt]", conflict.Paths)
	}
	// 失败安全:没有部分隐藏状态 —— 两个 ref 原地不动、HEAD 不动、
	// 工作树/索引干净、没有遗留的 MERGE_HEAD。
	if !refPointsAt(t, f.dir, "refs/heads/target", f.target) {
		t.Fatal("target ref moved despite conflict")
	}
	if !refPointsAt(t, f.dir, f.ref, f.tip) {
		t.Fatal("integration branch moved despite conflict")
	}
	if h := strings.TrimSpace(runTestGit(t, f.dir, "rev-parse", "HEAD")); h != headBefore {
		t.Fatalf("HEAD moved to %q", h)
	}
	statusMustBeClean(t, f.dir)
	if _, statErr := os.Stat(filepath.Join(f.dir, ".git", "MERGE_HEAD")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("leftover merge state: %v", statErr)
	}
}

func TestMergeRejectsTargetBaseDrift(t *testing.T) {
	f := newMergeFixture(t, "mg-4",
		map[string]string{"g.txt": "ours"},
		map[string]string{"f.txt": "theirs"},
	)
	svc := newTestService(t, t.TempDir())
	// 调用方观察之后目标被并发推进:类型化漂移,绝不静默合并。
	moved := commitDetached(t, f.dir, f.target, "concurrent", map[string]string{"h.txt": "x"})
	runTestGit(t, f.dir, "update-ref", "refs/heads/target", moved)

	_, err := svc.Merge(context.Background(), MergeRequest{
		ChangeSet:            f.cs,
		Acceptance:           f.fact,
		RepositoryPath:       f.dir,
		TargetRef:            "refs/heads/target",
		Authorization:        testAuth(t, "refs/heads/target"),
		ExpectedTarget:       f.target, // 过期的期望值
		ExpectedBranchCommit: f.tip,
	})
	if !errors.Is(err, ErrStaleBase) {
		t.Fatalf("got %v, want ErrStaleBase", err)
	}
	if !refPointsAt(t, f.dir, "refs/heads/target", moved) {
		t.Fatal("drifted target must stay untouched")
	}
	if !refPointsAt(t, f.dir, f.ref, f.tip) {
		t.Fatal("integration branch must stay untouched")
	}
}

func TestMergeRejectsBranchBaseDrift(t *testing.T) {
	f := newMergeFixture(t, "mg-5",
		map[string]string{"g.txt": "ours"},
		map[string]string{"f.txt": "theirs"},
	)
	svc := newTestService(t, t.TempDir())
	// 集成分支在调用方观察之后被外部移动:拒绝,不合并未知内容。
	moved := commitDetached(t, f.dir, f.tip, "external", map[string]string{"z.txt": "z"})
	runTestGit(t, f.dir, "update-ref", f.ref, moved)

	_, err := svc.Merge(context.Background(), MergeRequest{
		ChangeSet:            f.cs,
		Acceptance:           f.fact,
		RepositoryPath:       f.dir,
		TargetRef:            "refs/heads/target",
		Authorization:        testAuth(t, "refs/heads/target"),
		ExpectedTarget:       f.target,
		ExpectedBranchCommit: f.tip, // 过期的期望值
	})
	if !errors.Is(err, ErrStaleBase) {
		t.Fatalf("got %v, want ErrStaleBase", err)
	}
	if !refPointsAt(t, f.dir, "refs/heads/target", f.target) {
		t.Fatal("target must stay untouched on branch drift")
	}
	if !refPointsAt(t, f.dir, f.ref, moved) {
		t.Fatal("integration branch must stay at its externally moved value")
	}
}

func TestMergeRejectsAlreadyMergedReplay(t *testing.T) {
	f := newMergeFixture(t, "mg-6",
		map[string]string{"g.txt": "ours"},
		map[string]string{"f.txt": "theirs"},
	)
	svc := newTestService(t, t.TempDir())
	first, err := svc.Merge(context.Background(), MergeRequest{
		ChangeSet:            f.cs,
		Acceptance:           f.fact,
		RepositoryPath:       f.dir,
		TargetRef:            "refs/heads/target",
		Authorization:        testAuth(t, "refs/heads/target"),
		ExpectedTarget:       f.target,
		ExpectedBranchCommit: f.tip,
	})
	if err != nil {
		t.Fatalf("first merge: %v", err)
	}

	// 重放:以新目标期望值再合并同一分支 —— 已包含,类型化非成功。
	_, err = svc.Merge(context.Background(), MergeRequest{
		ChangeSet:            f.cs,
		Acceptance:           f.fact,
		RepositoryPath:       f.dir,
		TargetRef:            "refs/heads/target",
		Authorization:        testAuth(t, "refs/heads/target"),
		ExpectedTarget:       first.Commit,
		ExpectedBranchCommit: f.tip,
	})
	if !errors.Is(err, ErrAlreadyMerged) {
		t.Fatalf("got %v, want ErrAlreadyMerged", err)
	}
	if !refPointsAt(t, f.dir, "refs/heads/target", first.Commit) {
		t.Fatal("target must stay at the first merge commit")
	}
}

func TestMergeRejectsCheckedOutTarget(t *testing.T) {
	dir := initRepo(t)
	base := seedCommit(t, dir)
	defaultBranch := strings.TrimSpace(runTestGit(t, dir, "symbolic-ref", "--short", "HEAD"))
	cs, fact := acceptedCandidate(t, base, "mg-7")
	svc := newTestService(t, t.TempDir())
	runTestGit(t, dir, "update-ref", expectedBranchRef(t, cs), base)

	// 目标是当前检出的默认分支:在其下移动 ref 会让工作树失真,拒绝。
	_, err := svc.Merge(context.Background(), MergeRequest{
		ChangeSet:            cs,
		Acceptance:           fact,
		RepositoryPath:       dir,
		TargetRef:            "refs/heads/" + defaultBranch,
		ExpectedTarget:       base,
		ExpectedBranchCommit: base,
		Authorization:        testAuth(t, "refs/heads/"+defaultBranch),
	})
	if !errors.Is(err, ErrTargetCheckedOut) {
		t.Fatalf("got %v, want ErrTargetCheckedOut", err)
	}
	if h := strings.TrimSpace(runTestGit(t, dir, "rev-parse", "HEAD")); h != base {
		t.Fatal("HEAD moved on checked-out-target rejection")
	}
}

func TestMergeRejectsBranchNotCarryingCandidateBase(t *testing.T) {
	f := newMergeFixture(t, "mg-8",
		map[string]string{"g.txt": "ours"},
		map[string]string{"f.txt": "theirs"},
	)
	svc := newTestService(t, t.TempDir())
	// 把集成分支移到一个不承载候选 base 的无关根提交上
	// (SHA-1 空树对象是内容寻址的常量,任何空仓库都可解析)。
	const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"
	orphan := strings.TrimSpace(runTestGit(t, f.dir, "commit-tree", emptyTree, "-m", "orphan"))
	runTestGit(t, f.dir, "update-ref", f.ref, orphan)

	_, err := svc.Merge(context.Background(), MergeRequest{
		ChangeSet:            f.cs,
		Acceptance:           f.fact,
		RepositoryPath:       f.dir,
		TargetRef:            "refs/heads/target",
		Authorization:        testAuth(t, "refs/heads/target"),
		ExpectedTarget:       f.target,
		ExpectedBranchCommit: orphan,
	})
	if !errors.Is(err, ErrStaleBase) {
		t.Fatalf("got %v, want ErrStaleBase (branch does not carry candidate base)", err)
	}
	if !refPointsAt(t, f.dir, "refs/heads/target", f.target) {
		t.Fatal("target must stay untouched")
	}
}

func TestMergeRejectsMissingBranch(t *testing.T) {
	f := newMergeFixture(t, "mg-9",
		map[string]string{"g.txt": "ours"},
		map[string]string{"f.txt": "theirs"},
	)
	runTestGit(t, f.dir, "update-ref", "-d", f.ref)
	svc := newTestService(t, t.TempDir())

	_, err := svc.Merge(context.Background(), MergeRequest{
		ChangeSet:            f.cs,
		Acceptance:           f.fact,
		RepositoryPath:       f.dir,
		TargetRef:            "refs/heads/target",
		Authorization:        testAuth(t, "refs/heads/target"),
		ExpectedTarget:       f.target,
		ExpectedBranchCommit: f.tip,
	})
	if !errors.Is(err, ErrIntegrationBranchMissing) {
		t.Fatalf("got %v, want ErrIntegrationBranchMissing", err)
	}
	if !refPointsAt(t, f.dir, "refs/heads/target", f.target) {
		t.Fatal("target must stay untouched")
	}
}

func TestMergeValidatesLikeIntegrate(t *testing.T) {
	f := newMergeFixture(t, "mg-10",
		map[string]string{"g.txt": "ours"},
		map[string]string{"f.txt": "theirs"},
	)
	target := "refs/heads/target"

	cases := []struct {
		name    string
		req     MergeRequest
		protect bool
		wantErr error
	}{
		{name: "nil changeset", req: MergeRequest{
			RepositoryPath: f.dir, TargetRef: target,
			ExpectedTarget: f.target, ExpectedBranchCommit: f.tip,
		}, wantErr: ErrCandidateUnidentified},
		{name: "zero acceptance", req: MergeRequest{
			ChangeSet: f.cs, RepositoryPath: f.dir, TargetRef: target,
			ExpectedTarget: f.target, ExpectedBranchCommit: f.tip,
			Authorization: testAuth(t, target),
		}, wantErr: ErrNotAccepted},
		{name: "mismatched acceptance", req: MergeRequest{
			ChangeSet: f.cs,
			Acceptance: AcceptanceFact{
				ChangeSetID: changeset.ChangeSetID("sha256:deadbeef"),
				AcceptedAt:  f.fact.AcceptedAt,
				Basis:       "acceptance-event:other",
			},
			RepositoryPath: f.dir, TargetRef: target,
			ExpectedTarget: f.target, ExpectedBranchCommit: f.tip,
			Authorization: testAuth(t, target),
		}, wantErr: ErrAcceptanceMismatch},
		{name: "missing repository path", req: MergeRequest{
			ChangeSet: f.cs, Acceptance: f.fact, TargetRef: target,
			ExpectedTarget: f.target, ExpectedBranchCommit: f.tip,
		}, wantErr: ErrNoRepository},
		{name: "protected primary worktree", req: MergeRequest{
			ChangeSet: f.cs, Acceptance: f.fact, RepositoryPath: f.dir, TargetRef: target,
			ExpectedTarget: f.target, ExpectedBranchCommit: f.tip,
		}, protect: true, wantErr: ErrProtectedTarget},
		{name: "missing expected target", req: MergeRequest{
			ChangeSet: f.cs, Acceptance: f.fact, RepositoryPath: f.dir, TargetRef: target,
			ExpectedBranchCommit: f.tip,
			Authorization:        testAuth(t, target),
		}, wantErr: ErrNoExpectedBase},
		{name: "missing expected branch commit", req: MergeRequest{
			ChangeSet: f.cs, Acceptance: f.fact, RepositoryPath: f.dir, TargetRef: target,
			ExpectedTarget: f.target,
			Authorization:  testAuth(t, target),
		}, wantErr: ErrNoExpectedBase},
		{name: "relative ref name", req: MergeRequest{
			ChangeSet: f.cs, Acceptance: f.fact, RepositoryPath: f.dir, TargetRef: "master",
			ExpectedTarget: f.target, ExpectedBranchCommit: f.tip,
		}, wantErr: ErrInvalidTargetRef},
		{name: "audit namespace target", req: MergeRequest{
			ChangeSet: f.cs, Acceptance: f.fact, RepositoryPath: f.dir, TargetRef: f.ref,
			ExpectedTarget: f.target, ExpectedBranchCommit: f.tip,
		}, wantErr: ErrInvalidTargetRef},
		{name: "dotdot in ref name", req: MergeRequest{
			ChangeSet: f.cs, Acceptance: f.fact, RepositoryPath: f.dir, TargetRef: "refs/heads/a..b",
			ExpectedTarget: f.target, ExpectedBranchCommit: f.tip,
		}, wantErr: ErrInvalidTargetRef},
		{name: "ref lock suffix", req: MergeRequest{
			ChangeSet: f.cs, Acceptance: f.fact, RepositoryPath: f.dir, TargetRef: "refs/heads/evil.lock",
			ExpectedTarget: f.target, ExpectedBranchCommit: f.tip,
		}, wantErr: ErrInvalidTargetRef},
	}
	for _, tc := range cases {
		primary := t.TempDir()
		if tc.protect {
			primary = f.dir
		}
		svc := newTestService(t, primary)
		_, err := svc.Merge(context.Background(), tc.req)
		if !errors.Is(err, tc.wantErr) {
			t.Fatalf("%s: got %v, want %v", tc.name, err, tc.wantErr)
		}
		// 每一次拒绝都不移动任何 ref。
		if !refPointsAt(t, f.dir, target, f.target) {
			t.Fatalf("%s: target ref moved on rejection", tc.name)
		}
		if !refPointsAt(t, f.dir, f.ref, f.tip) {
			t.Fatalf("%s: integration branch moved on rejection", tc.name)
		}
	}
}

func TestMergeOnNonRepositoryFails(t *testing.T) {
	plain := t.TempDir()
	cs, fact := acceptedCandidate(t, "0123456789abcdef0123456789abcdef01234567", "mg-11")
	svc := newTestService(t, t.TempDir())

	_, err := svc.Merge(context.Background(), MergeRequest{
		ChangeSet:            cs,
		Acceptance:           fact,
		RepositoryPath:       plain,
		TargetRef:            "refs/heads/target",
		ExpectedTarget:       "0123456789abcdef0123456789abcdef01234567",
		ExpectedBranchCommit: "0123456789abcdef0123456789abcdef01234567",
		Authorization:        testAuth(t, "refs/heads/target"),
	})
	if !errors.Is(err, gitwriter.ErrNotRepository) {
		t.Fatalf("got %v, want gitwriter.ErrNotRepository", err)
	}
}

func TestMergeCancelledContext(t *testing.T) {
	f := newMergeFixture(t, "mg-12",
		map[string]string{"g.txt": "ours"},
		map[string]string{"f.txt": "theirs"},
	)
	svc := newTestService(t, t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := svc.Merge(ctx, MergeRequest{
		ChangeSet:            f.cs,
		Acceptance:           f.fact,
		RepositoryPath:       f.dir,
		TargetRef:            "refs/heads/target",
		Authorization:        testAuth(t, "refs/heads/target"),
		ExpectedTarget:       f.target,
		ExpectedBranchCommit: f.tip,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	// 取消不得被伪造成合并结果,也不移动任何 ref。
	if !refPointsAt(t, f.dir, "refs/heads/target", f.target) {
		t.Fatal("target ref moved on cancelled context")
	}
	if !refPointsAt(t, f.dir, f.ref, f.tip) {
		t.Fatal("integration branch moved on cancelled context")
	}
}
