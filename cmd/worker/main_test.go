package main

// failureRunner 的接线测试(P0-21):结构化 CommandSpec 原样执行(全程无 strings.Fields,
// 含空格的参数不得被拆)、Target 非空跑目标自己的构建命令、Fix 写进一次性 worktree
// 而不是主 worktree(INV-EXEC-01)、成功/失败/取消后 worktree 一律清理(PRD §21.5)。
// worktree 生命周期走真 git(workspace.Manager 的生产姿势),候选命令执行用 fake 离线断言。

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"tally/internal/command"
	"tally/internal/core"
	"tally/internal/failure"
	"tally/internal/runner"
)

// recRunner 记录收到的每条命令并回固定结果。inspect 在执行现场同步跑:
// 一次性 worktree 随 Run 结束即被清理,依赖 worktree 内容的断言必须在当下做。
type recRunner struct {
	got     []runner.Command
	res     runner.Result
	err     error
	inspect func(runner.Command)
}

func (r *recRunner) Run(_ context.Context, cmd runner.Command) (runner.Result, error) {
	r.got = append(r.got, cmd)
	if r.inspect != nil {
		r.inspect(cmd)
	}
	return r.res, r.err
}

// hangRunner 挂起直到 ctx 被取消:模拟候选执行中途取消(队列关闭 / 租约丢失)。
type hangRunner struct{ started chan struct{} }

func (r *hangRunner) Run(ctx context.Context, _ runner.Command) (runner.Result, error) {
	close(r.started)
	<-ctx.Done()
	return runner.Result{}, ctx.Err()
}

// newGitRepo 建一个带首个提交的真 git 仓库(HEAD 可解析,一次性 worktree 可派生)。
func newGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	gitRun(t, dir, "init")
	gitRun(t, dir, "config", "user.name", "test")
	gitRun(t, dir, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("demo\n"), 0o644); err != nil {
		t.Fatalf("写 README: %v", err)
	}
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "init")
	return dir
}

// gitRun 跑一条 git 命令(结构化 argv),失败即测试失败。
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	res, err := (runner.LocalRunner{}).Run(context.Background(), runner.Command{
		Argv: append([]string{"git", "-C", dir}, args...),
	})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("git %v: err=%v res=%+v", args, err, res)
	}
}

// worktreeCount 数仓库登记的 worktree(含主 worktree):清理后应回到 1。
func worktreeCount(t *testing.T, repo string) int {
	t.Helper()
	res, err := (runner.LocalRunner{}).Run(context.Background(), runner.Command{
		Argv: []string{"git", "-C", repo, "worktree", "list", "--porcelain"},
	})
	if err != nil || res.ExitCode != 0 {
		t.Fatalf("worktree list: err=%v res=%+v", err, res)
	}
	n := 0
	for _, line := range strings.Split(res.Stdout, "\n") {
		if strings.HasPrefix(strings.TrimRight(line, "\r"), "worktree ") {
			n++
		}
	}
	return n
}

// assertInsideRoot 断言 path 落在一次性根目录之内(候选执行不离开 disposable root)。
func assertInsideRoot(t *testing.T, root, path string) {
	t.Helper()
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || filepath.IsAbs(rel) ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		t.Errorf("执行目录 %s 应落在一次性根 %s 之内", path, root)
	}
}

// 验收(G3):Target 非空 → spec.Command 已由闸门换成该目标自己的构建命令,
// 执行器原样跑 Command、不对 Command 拆改,Target 只是标签。
func TestFailureRunnerTargetRunsAsFullCommand(t *testing.T) {
	repo := newGitRepo(t)
	rec := &recRunner{res: runner.Result{ExitCode: 0, Stdout: "ok"}}
	fr := failureRunner{local: rec, root: t.TempDir()}

	res, err := fr.Run(context.Background(), failure.RunSpec{
		Command: command.CommandSpec{Argv: []string{"make", "build-module-b"}, Cwd: repo},
		Target:  "build-module-b", // G3 目标名标签
	})
	if err != nil {
		t.Fatalf("Target 非空不该报错(G3 已接线): %v", err)
	}
	if !res.Green {
		t.Errorf("退出码 0 应是绿: %+v", res)
	}
	if len(rec.got) != 1 {
		t.Fatalf("应恰好执行一次候选命令: %d", len(rec.got))
	}
	if want := []string{"make", "build-module-b"}; !slices.Equal(rec.got[0].Argv, want) {
		t.Errorf("G3 应原样跑目标构建命令: %v", rec.got[0].Argv)
	}
	assertInsideRoot(t, fr.root, rec.got[0].Dir)
}

// 验收(结构化命令):Target 为空 → 原样跑 spec.Command.Argv;含空格的参数必须
// 完整保留 —— 证明这条路径上没有 strings.Fields(PRD §21.6)。
func TestFailureRunnerEmptyTargetRunsCommand(t *testing.T) {
	repo := newGitRepo(t)
	rec := &recRunner{res: runner.Result{ExitCode: 1, Stderr: "boom"}}
	fr := failureRunner{local: rec, root: t.TempDir()}

	res, err := fr.Run(context.Background(), failure.RunSpec{
		Command: command.CommandSpec{Argv: []string{"make", "build", "arg with space"}, Cwd: repo},
	})
	if err != nil {
		t.Fatalf("非零退出码不是 error: %v", err)
	}
	if res.Green {
		t.Errorf("退出码 1 应是红: %+v", res)
	}
	want := []string{"make", "build", "arg with space"}
	if !slices.Equal(rec.got[0].Argv, want) {
		t.Errorf("结构化 argv 应原样传递、不得拆分: got=%v want=%v", rec.got[0].Argv, want)
	}
	if rec.got[0].Dir == repo {
		t.Errorf("候选不得在主 worktree 执行(INV-EXEC-01): %s", rec.got[0].Dir)
	}
}

// 验收(Fix):先照写改后文件再跑命令;写进一次性 worktree,主 worktree 零痕迹。
func TestFailureRunnerAppliesFixInsideDisposableWorktree(t *testing.T) {
	repo := newGitRepo(t)
	rec := &recRunner{res: runner.Result{ExitCode: 0}}
	fr := failureRunner{local: rec, root: t.TempDir()}

	// worktree 在 Run 返回后即清理(defer Cleanup),文件内容在命令执行时刻捕获 ——
	// 这同时证明「先写修复、后跑命令」的次序。
	var got string
	rec.inspect = func(cmd runner.Command) {
		b, err := os.ReadFile(filepath.Join(cmd.Dir, "go.mod"))
		if err != nil {
			t.Errorf("修复应先于命令写进一次性 worktree 的执行目录: %v", err)
			return
		}
		got = string(b)
	}
	if _, err := fr.Run(context.Background(), failure.RunSpec{
		Command: command.CommandSpec{Argv: []string{"make", "build"}, Cwd: repo},
		Fix:     []core.FileChange{{Path: "go.mod", Content: "module demo\n"}},
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(rec.got) != 1 {
		t.Fatalf("应恰好执行一次候选命令: %d", len(rec.got))
	}
	if got != "module demo\n" {
		t.Errorf("改后文件内容不对: %q", got)
	}

	assertInsideRoot(t, fr.root, rec.got[0].Dir)
	if _, err := os.Stat(filepath.Join(repo, "go.mod")); !os.IsNotExist(err) {
		t.Errorf("主 worktree 不得被写入修复文件(INV-EXEC-01): %v", err)
	}
}

// 记录的工作目录是仓库子目录时,命令应重映射进 worktree 的对应子目录。
func TestFailureRunnerRemapsSubdirCwdIntoWorktree(t *testing.T) {
	repo := newGitRepo(t)
	if err := os.MkdirAll(filepath.Join(repo, "cmd", "app"), 0o755); err != nil {
		t.Fatalf("建子目录: %v", err)
	}
	rec := &recRunner{res: runner.Result{ExitCode: 0}}
	fr := failureRunner{local: rec, root: t.TempDir()}

	if _, err := fr.Run(context.Background(), failure.RunSpec{
		Command: command.CommandSpec{Argv: []string{"make", "build"}, Cwd: filepath.Join(repo, "cmd", "app")},
	}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if filepath.Base(rec.got[0].Dir) != "app" {
		t.Errorf("应在 worktree 的对应子目录执行: %q", rec.got[0].Dir)
	}
	assertInsideRoot(t, fr.root, rec.got[0].Dir)
}

// 验收(清理):成功(绿)与失败(红)跑完,一次性 worktree 都得清掉:
// 仓库登记回到只剩主 worktree,一次性根目录不留残骸。
func TestFailureRunnerCleansUpWorktreeAfterRun(t *testing.T) {
	repo := newGitRepo(t)
	root := t.TempDir()
	for _, tc := range []struct {
		name string
		res  runner.Result
	}{
		{"绿", runner.Result{ExitCode: 0}},
		{"红", runner.Result{ExitCode: 1, Stderr: "boom"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := &recRunner{res: tc.res}
			fr := failureRunner{local: rec, root: root}
			if _, err := fr.Run(context.Background(),
				failure.RunSpec{Command: command.CommandSpec{Argv: []string{"make", "build"}, Cwd: repo}}); err != nil {
				t.Fatalf("非零退出码不是 error: %v", err)
			}
			if n := worktreeCount(t, repo); n != 1 {
				t.Errorf("跑完应只剩主 worktree: %d", n)
			}
			if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
				t.Errorf("一次性根目录应清空: entries=%d err=%v", len(entries), err)
			}
		})
	}
}

// 验收(清理/取消):候选执行中途被取消,worktree 同样必须清掉(PRD §21.5:
// 失败 candidate 不得污染下一个;队列关闭/租约丢失都走取消路径)。
func TestFailureRunnerCleansUpAfterCancellation(t *testing.T) {
	repo := newGitRepo(t)
	root := t.TempDir()
	hang := &hangRunner{started: make(chan struct{})}
	fr := failureRunner{local: hang, root: root}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-hang.started
		cancel()
	}()
	if _, err := fr.Run(ctx, failure.RunSpec{
		Command: command.CommandSpec{Argv: []string{"make", "build"}, Cwd: repo},
	}); !errors.Is(err, context.Canceled) {
		t.Fatalf("取消应作为错误上抛: %v", err)
	}
	if n := worktreeCount(t, repo); n != 1 {
		t.Errorf("取消后仍应清理,只剩主 worktree: %d", n)
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Errorf("取消后一次性根目录应清空: entries=%d err=%v", len(entries), err)
	}
}

// 验收(fail-closed):越出一次性 worktree 的修复路径直接拒绝,且不执行任何命令 ——
// 否则等于变相改 owner 主 worktree(INV-EXEC-01)。
func TestFailureRunnerRejectsFixOutsideWorktree(t *testing.T) {
	repo := newGitRepo(t)
	rec := &recRunner{res: runner.Result{ExitCode: 0}}
	fr := failureRunner{local: rec, root: t.TempDir()}

	_, err := fr.Run(context.Background(), failure.RunSpec{
		Command: command.CommandSpec{Argv: []string{"make", "build"}, Cwd: repo},
		Fix:     []core.FileChange{{Path: "../escape.txt", Content: "no"}},
	})
	if err == nil {
		t.Fatal("越出 worktree 的修复路径应拒绝")
	}
	if len(rec.got) != 0 {
		t.Errorf("拒绝后不得执行任何命令: %v", rec.got)
	}
}

// 验收(fail-closed):记录的工作目录不是 git 仓库检出 → 拒绝执行,
// 绝不退回「直接在原目录跑」的旧语义(INV-EXEC-01)。
func TestFailureRunnerRejectsNonRepoWorkdir(t *testing.T) {
	plain := t.TempDir() // 无 .git:t.TempDir 各自独立,不是仓库
	rec := &recRunner{res: runner.Result{ExitCode: 0}}
	fr := failureRunner{local: rec, root: t.TempDir()}

	if _, err := fr.Run(context.Background(), failure.RunSpec{
		Command: command.CommandSpec{Argv: []string{"make", "build"}, Cwd: plain},
	}); err == nil {
		t.Fatal("非仓库工作目录应拒绝执行")
	}
	if len(rec.got) != 0 {
		t.Errorf("拒绝后不得执行任何命令: %v", rec.got)
	}
}

// flattenEnv 键序固定:同一条 spec 每次执行的环境一致。
func TestFlattenEnvSorted(t *testing.T) {
	got := flattenEnv(map[string]string{"B": "2", "A": "1"})
	if want := []string{"A=1", "B=2"}; !slices.Equal(got, want) {
		t.Errorf("flattenEnv 应按键序稳定输出: got=%v want=%v", got, want)
	}
	if v := flattenEnv(nil); v != nil {
		t.Errorf("空 env 应返回 nil: %v", v)
	}
}
