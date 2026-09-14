// v141_freeze_commit_test.go 钉住 P0-08 confirm→freeze Git 事务的合同:
//   - 成功路径:三件套**一个提交**落库,且可从 pinned 提交经既有装载器确定性
//     读回(任务 AC2);提交后工作区/索引干净;
//   - stale base:期望基线过期时类型化冲突(ErrBaseRevision),绝不静默改基、
//     绝不写入任何文件(任务 AC1);
//   - 部分失败回滚:三件套要么全部落库、要么全部不落(部分应用被 gitwriter
//     回滚,不留半套制品);
//   - 重复确认幂等:同内容再提交 → ErrNoChanges,不产生第二个提交;
//   - 身份门:三件套任务身份不一致的装配 fail-closed 拒收。

package artifacts

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"tally/internal/gitwriter"
	"tally/internal/intake"
	"tally/internal/intakefreeze"
	"tally/internal/verification"
)

// freezeSig 是事务测试的显式提交身份(不依赖环境 git 配置)。
var freezeSig = gitwriter.Signature{Name: "freeze-test", Email: "freeze@test.local"}

// freezeArtifacts 经真实的冻结准入(P0-07 intakefreeze.Freeze)构造一份三件套
// 夹具:与 handler 将提交的形态同源,而不是手工拼装。
func freezeArtifacts(t *testing.T, taskID string) intakefreeze.FrozenArtifacts {
	t.Helper()
	fa, err := intakefreeze.Freeze(intakefreeze.ConfirmedDraft{
		TaskID:    taskID,
		Profile:   "standard",
		SourceRef: "GitHub Issue #142",
		Draft: intake.Draft{
			TaskType:  intake.TaskTypeBugFix,
			Summary:   "修复 logout 后 session 仍然有效的问题",
			Objective: "logout 必须使既有 session 立即失效",
			Scope: intake.Scope{
				AllowedPaths:   []string{"internal/session/**"},
				ForbiddenPaths: []string{"internal/legacy/**"},
			},
			Acceptance: []intake.AcceptanceClause{
				{ID: "AC-01", Statement: "logout 后旧 session token 的请求返回 401"},
			},
		},
		RiskHints: []string{"auth_session"},
	}, intakefreeze.VerificationPlan{
		Evidence: []intakefreeze.EvidenceDeclaration{{
			AcceptanceID: "AC-01",
			Kind:         verification.EvidenceTest,
			Path:         "internal/session/logout_test.go",
		}},
	})
	if err != nil {
		t.Fatalf("冻结夹具 %s: %v", taskID, err)
	}
	return fa
}

// gitOut 在 dir 里跑一条只读 git 命令并返回 stdout(测试断言用)。
func gitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var out, errBuf strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(errBuf.String()))
	}
	return out.String()
}

// freezeStatusMustBeClean 断言事务之后工作区与索引没有半套残留。
func freezeStatusMustBeClean(t *testing.T, dir string) {
	t.Helper()
	if out := strings.TrimSpace(gitOut(t, dir, "status", "--porcelain")); out != "" {
		t.Fatalf("事务后工作区/索引不干净:\n%s", out)
	}
}

func TestFreezeCommitCommitsTripleAndReloads(t *testing.T) {
	rf := initRepo(t)
	base := rf.commit(t, map[string]string{ManifestPath: manifestYAML}, "base: 注册表")
	fa := freezeArtifacts(t, "TASK-0042")

	res, err := FreezeCommit(context.Background(), rf.dir, FreezeCommitRequest{
		ExpectedBase: base,
		Artifacts:    fa,
		Message:      "freeze: 确认冻结任务 TASK-0042",
		Signature:    &freezeSig,
	})
	if err != nil {
		t.Fatalf("FreezeCommit: %v", err)
	}
	if res.Base != base || res.Commit == "" || res.Commit == base {
		t.Fatalf("CommitResult = %+v,应校验基线 %q 并产生新提交", res, base)
	}
	if head := strings.TrimSpace(gitOut(t, rf.dir, "rev-parse", "HEAD")); head != res.Commit {
		t.Fatalf("HEAD = %q,want 新提交 %q", head, res.Commit)
	}
	freezeStatusMustBeClean(t, rf.dir)

	// AC2:成功确认可从提交的制品确定性地读回 —— 任务/验证契约经 Load,
	// 意图证据经 LoadIntent,哈希/修订全部来自内容重算且与冻结值相等。
	snap, err := Load(rf.dir, "HEAD")
	if err != nil {
		t.Fatalf("Load(HEAD): %v", err)
	}
	var task *TaskArtifact
	for i := range snap.Tasks {
		if snap.Tasks[i].Contract.ID == "TASK-0042" {
			task = &snap.Tasks[i]
			break
		}
	}
	if task == nil {
		t.Fatalf("HEAD 快照里没有 TASK-0042 制品(路径 %v)", pathsOf(snap.Tasks))
	}
	if task.Path != TasksDir+"/TASK-0042.yaml" {
		t.Errorf("task 制品路径 = %q", task.Path)
	}
	if task.Contract.Hash != fa.Task.Hash {
		t.Errorf("重载合同哈希 = %q,want 冻结值 %q(往返必须保内容)", task.Contract.Hash, fa.Task.Hash)
	}
	var verif *VerificationArtifact
	for i := range snap.Verifications {
		if snap.Verifications[i].Frozen.TaskID() == "TASK-0042" {
			verif = &snap.Verifications[i]
			break
		}
	}
	if verif == nil {
		t.Fatalf("HEAD 快照里没有 TASK-0042 的冻结验证契约")
	}
	if verif.Path != VerificationsDir+"/TASK-0042.yaml" {
		t.Errorf("verification 制品路径 = %q", verif.Path)
	}
	if verif.Frozen.Hash() != fa.Verification.Hash() {
		t.Errorf("重载验证契约哈希 = %q,want 冻结值 %q", verif.Frozen.Hash(), fa.Verification.Hash())
	}

	ia, err := LoadIntent(rf.dir, res.Commit, "TASK-0042")
	if err != nil {
		t.Fatalf("LoadIntent(%s): %v", res.Commit, err)
	}
	if ia.Path != ".tally/intents/TASK-0042.yaml" || ia.SourceCommit != res.Commit {
		t.Errorf("IntentArtifact = %+v", ia)
	}
	if ia.Intent.Revision() != fa.Intent.Revision() {
		t.Errorf("重载 intent 修订 = %q,want 冻结值 %q", ia.Intent.Revision(), fa.Intent.Revision())
	}
	if ia.Intent.SourceRef() != "GitHub Issue #142" {
		t.Errorf("重载 intent SourceRef = %q", ia.Intent.SourceRef())
	}
	if hints := ia.Intent.RiskHints(); len(hints) != 1 || hints[0] != "auth_session" {
		t.Errorf("重载 intent RiskHints = %v", hints)
	}
}

func TestFreezeCommitStaleBaseConflicts(t *testing.T) {
	rf := initRepo(t)
	base := rf.commit(t, map[string]string{ManifestPath: manifestYAML}, "base: 注册表")
	// 期望基线之后仓库被并发推进:stale base 必须冲突,而不是叠写。
	head := rf.commit(t, map[string]string{"docs/note.md": "# note\n"}, "docs: 并发推进")
	fa := freezeArtifacts(t, "TASK-0042")

	res, err := FreezeCommit(context.Background(), rf.dir, FreezeCommitRequest{
		ExpectedBase: base, // 已过期
		Artifacts:    fa,
		Message:      "freeze: 应被基线校验拒绝",
		Signature:    &freezeSig,
	})
	if !errors.Is(err, gitwriter.ErrBaseRevision) {
		t.Fatalf("stale base 应返回 ErrBaseRevision,得到 %v", err)
	}
	if res.Commit != "" {
		t.Errorf("冲突不得产生提交,得到 %+v", res)
	}
	if now := strings.TrimSpace(gitOut(t, rf.dir, "rev-parse", "HEAD")); now != head {
		t.Fatalf("冲突后 HEAD = %q,want 保持 %q", now, head)
	}
	freezeStatusMustBeClean(t, rf.dir)
	// 三件套一份都不许出现在工作区(冲突发生在任何文件写入之前)。
	for _, p := range []string{".tally/intents/TASK-0042.yaml", ".tally/tasks/TASK-0042.yaml", ".tally/verifications/TASK-0042.yaml"} {
		if _, serr := os.Stat(rf.dir + "/" + p); !errors.Is(serr, os.ErrNotExist) {
			t.Errorf("冲突后 %s 不应存在: %v", p, serr)
		}
	}
}

func TestFreezeCommitPartialFailureLeavesNoArtifacts(t *testing.T) {
	rf := initRepo(t)
	base := rf.commit(t, map[string]string{ManifestPath: manifestYAML}, "base: 注册表")
	fa := freezeArtifacts(t, "TASK-0042")
	// 让三件套中的第二份(task)落盘失败:.tally/tasks/TASK-0042.yaml 是一个
	// 目录 → gitwriter 在已写入 intent 之后 apply 失败,必须整体回滚。
	if mkErr := os.MkdirAll(rf.dir+"/.tally/tasks/TASK-0042.yaml", 0o755); mkErr != nil {
		t.Fatalf("构造阻塞目录: %v", mkErr)
	}

	_, err := FreezeCommit(context.Background(), rf.dir, FreezeCommitRequest{
		ExpectedBase: base,
		Artifacts:    fa,
		Message:      "freeze: 中途失败",
		Signature:    &freezeSig,
	})
	if err == nil {
		t.Fatalf("task 制品落盘失败必须让事务失败")
	}
	if errors.Is(err, gitwriter.ErrBaseRevision) {
		t.Fatalf("失败应发生在基线校验之后的落盘阶段,得到 %v", err)
	}
	if now := strings.TrimSpace(gitOut(t, rf.dir, "rev-parse", "HEAD")); now != base {
		t.Fatalf("失败后 HEAD = %q,want 保持基线 %q", now, base)
	}
	freezeStatusMustBeClean(t, rf.dir)
	// 已写入的 intent 必须被回滚:失败不留半套可装载制品。
	if _, serr := os.Stat(rf.dir + "/.tally/intents/TASK-0042.yaml"); !errors.Is(serr, os.ErrNotExist) {
		t.Errorf("部分失败必须回滚已应用文件,intent 残留: %v", serr)
	}
	// 提交树里同样一份都不许有(失败没有产生任何提交)。
	if _, lerr := LoadIntent(rf.dir, base, "TASK-0042"); !errors.Is(lerr, ErrIntentNotFound) {
		t.Errorf("失败基线上不应能装载 intent,得到 %v", lerr)
	}
}

func TestFreezeCommitDuplicateConfirmIsErrNoChanges(t *testing.T) {
	rf := initRepo(t)
	base := rf.commit(t, map[string]string{ManifestPath: manifestYAML}, "base: 注册表")
	fa := freezeArtifacts(t, "TASK-0042")
	req := FreezeCommitRequest{ExpectedBase: base, Artifacts: fa, Message: "freeze: 确认冻结任务 TASK-0042", Signature: &freezeSig}

	first, err := FreezeCommit(context.Background(), rf.dir, req)
	if err != nil {
		t.Fatalf("首次 FreezeCommit: %v", err)
	}
	// 重复确认(同内容):幂等 —— 不产生第二个提交,类型化 ErrNoChanges。
	req.ExpectedBase = first.Commit
	second, err := FreezeCommit(context.Background(), rf.dir, req)
	if !errors.Is(err, gitwriter.ErrNoChanges) {
		t.Fatalf("重复确认应返回 ErrNoChanges,得到 %v", err)
	}
	if second.Commit != "" {
		t.Errorf("重复确认不得产生新提交,得到 %+v", second)
	}
	if now := strings.TrimSpace(gitOut(t, rf.dir, "rev-parse", "HEAD")); now != first.Commit {
		t.Fatalf("重复确认后 HEAD = %q,want 保持 %q", now, first.Commit)
	}
	freezeStatusMustBeClean(t, rf.dir)
	if n := strings.TrimSpace(gitOut(t, rf.dir, "rev-list", "--count", "HEAD")); n != "2" {
		t.Errorf("仓库应有 2 个提交(注册表 + 一次冻结),得到 %s", n)
	}
}

func TestFreezeCommitRejectsIdentityMismatch(t *testing.T) {
	rf := initRepo(t)
	base := rf.commit(t, map[string]string{ManifestPath: manifestYAML}, "base: 注册表")
	a := freezeArtifacts(t, "TASK-0042")
	b := freezeArtifacts(t, "TASK-0043")
	// 手工装配出身份不一致的三件套:intent 是 0043,task/verification 是 0042。
	mixed := intakefreeze.FrozenArtifacts{Intent: b.Intent, Task: a.Task, Verification: a.Verification}

	res, err := FreezeCommit(context.Background(), rf.dir, FreezeCommitRequest{
		ExpectedBase: base,
		Artifacts:    mixed,
		Message:      "freeze: 身份不一致",
		Signature:    &freezeSig,
	})
	if !errors.Is(err, ErrFreezeIdentityMismatch) {
		t.Fatalf("身份不一致应类型化拒收,得到 %v", err)
	}
	if res.Commit != "" {
		t.Errorf("拒收不得产生提交,得到 %+v", res)
	}
	freezeStatusMustBeClean(t, rf.dir)
}
