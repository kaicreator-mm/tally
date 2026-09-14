// Command worker 是 tally 的后台任务执行器:拉环境、部署、跑验证、调 LLM。
//
// 无状态;从 Postgres 队列(internal/jobs)取任务。T05b 接上了执行引擎的动作 job:
// 环境拉起/推倒/重建与部署回滚 —— 界面只入队,事实面(environment_run / deploy_run)
// 由这里的执行端在动作执行时 INSERT(INV-5 的正路)。T10 接上了失败录入与诊断:
// 动作失败经 ingest 录进 failure_event;诊断 job 按指纹装载失败,交 failure.Handle 跑一轮。
//
// P0-21(串行集成点):把 P0 基建接进生产装配 ——
//   - 队列带租约生命周期(owner 认领 + 心跳续租 + stale 回收,PRD §17.1);
//   - 诊断的候选执行只在一次性 Git worktree(INV-EXEC-01 / PRD §21.5),绝不碰主 workdir;
//   - spec-defect 出口接 requirement(诊断指「需求本身有问题」→ NEEDS_REVIEW 提案,INV-VER-04);
//   - 推理调用按调用点声明角色与用途(P0-07 起按调用点分人设;P0-11 起角色
//     声明收进 failure 域、调用走公共 AI Runtime 边界)。
//
// P1-43:同一队列再注册 v1.3 Codegen/Verify 垂直路径 job(见 v13.go)——
// 单个 approved standard TaskContract 经 CodegenRun→ChangeSet→Verification,
// 全程只在一次性 worktree;legacy 环境/部署/失败诊断 job 原样保留。
//
// P0-11:失败诊断的通用推理改走 internal/airuntime 边界 —— operation/intent/
// budget 由 failure 域声明,本进程只在 composition root 接过渡传输(airuntimelegacy adapter);
// 直连供应商路径不再进入任何域包。
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"tally/cmd/internal/airuntimelegacy"
	"tally/internal/environment"
	"tally/internal/exec"
	"tally/internal/failure"
	"tally/internal/jobs"
	"tally/internal/knowledge"
	"tally/internal/logpipe"
	"tally/internal/requirement"
	"tally/internal/runner"
	"tally/internal/store"
	"tally/internal/workspace"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.Open(ctx, databaseURL())
	if err != nil {
		slog.Error("连接数据库失败", "err", err)
		os.Exit(1)
	}
	defer st.Close()

	// 执行端:按 payload 里的定义路径(相对内容仓库)读定义,调 environment/deploy 真正执行。
	// 与 server 共用一个内容仓库路径(单机自部署,默认 ./tally-content)。
	// DB 接上后,环境/部署失败经 ingest 录进 failure_event(接上生产链:失败 → 指纹 → 诊断)。
	root := envOr("TALLY_CONTENT_REPO", "./tally-content")
	// T11e:TALLY_DIAGNOSE_AUTO=1 → 失败录入后自动入队 diagnose(默认关,D-e1)。
	autoDiagnose := os.Getenv("TALLY_DIAGNOSE_AUTO") == "1"
	// 先建队列再填 handler 表:exec 的自动诊断要入队到自己所在的队列,
	// jobs.New 与 exec.Handlers 互相引用,用同一个 map 两步装配。
	// 租约生命周期(PRD §17.1)在 jobs.Queue 内实现(P0-10);这里把租约参数接上,
	// 长任务(分钟级构建跑 G1/G2)可经环境覆盖,缺省值归 jobs 包。
	handlers := map[jobs.Kind]jobs.Handler{}
	queue := jobs.New(st, handlers, leaseOptions()...)
	for kind, h := range exec.Handlers(exec.Config{
		Root:   root,
		Store:  st.Queries(),
		DB:     st,
		Docker: runner.NewDockerRunner(),
		NewSSH: func(s environment.SSHDef) environment.SSHRunner {
			return runner.NewSSHRunner(environment.SSHRunnerConfig(s))
		},
		Exec: runner.LocalRunner{},
		Diagnose: exec.DiagnoseDeps{
			// P0-11:推理走 airuntime 边界(DECISION_LOCKS #17 / PRD §28)——
			// operation/intent/budget 由 failure 域声明,这里只接传输;真实 AI
			// Runtime SDK 接线落地前用 airuntimelegacy adapter 过渡(见其注释)。
			// 推理只提案(INV-1),判定在 failure 四道闸。
			Executor: airuntimelegacy.FromEnv(),
			// P0-21:候选执行只在一次性 worktree(INV-EXEC-01 / PRD §21.5),
			// 成功/失败/取消一律清理,绝不写主 workdir。
			Runner: failureRunner{
				local: runner.LocalRunner{},
				root:  envOr("TALLY_WORKTREE_ROOT", filepath.Join(os.TempDir(), "tally-worktrees")),
			},
			KB: knowledge.NewStore(root, st.Queries()), // 与 server 同一个内容仓库
			// P0-14:spec-defect 出口接生产(failure.SpecDefectSink 的 requirement 侧实现)。
			// 诊断指「需求本身有问题」时经确定性条款校验留痕 NEEDS_REVIEW 提案,
			// 不是静默丢弃;留痕提交身份沿 cmd/tally 的签署身份约定,环境可覆盖。
			SpecSink: requirement.SpecDefectSink{
				RepoPath: root,
				Identity: requirement.Identity{
					Name:  envOr("TALLY_SIGN_NAME", "tally-worker"),
					Email: envOr("TALLY_SIGN_EMAIL", "tally-worker@localhost"),
				},
			},
		},
		DiagnoseAuto:  autoDiagnose,
		DiagnoseQueue: queue,
	}) {
		handlers[kind] = h
	}

	// P1-43:v1.3 Codegen/Verify 垂直路径。kind 由 worker 侧定义(cmd/worker/v13.go),
	// 复用同一队列与租约生命周期;runtime 选择(fake|codex)是配置(AC3),
	// 缺省 FakeRuntime(决策锁 13)。legacy job 注册不受影响(AC2)。
	if kind, h := v13Handler(v13ConfigFromEnv(root), st.Queries()); h != nil {
		handlers[kind] = h
	}

	// P2-09:v1.4 外层闭环 job(见 v14.go)—— 同一队列与租约生命周期,经
	// internal/closedloop 的 NewLoop 组合根驱动 Execute→Verify→Repair→
	// Accept/Escalate;v1.3 垂直路径与 legacy job 原样保留(PRD §34:两条
	// 路径并存,v13 基线不拆)。
	if kind, h := v14Handler(v14ConfigFromEnv(root), st.Queries()); h != nil {
		handlers[kind] = h
	}

	// 队列消费:阻塞式 worker pool 放后台;非取消导致的退出视为致命错误。
	go func() {
		if err := queue.Run(ctx); err != nil && ctx.Err() == nil {
			slog.Error("队列消费异常退出", "err", err)
			os.Exit(1)
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	addr := envOr("TALLY_WORKER_ADDR", ":8081")
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	slog.Info("worker 启动,消费执行动作队列", "health_addr", addr, "content_repo", root)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("worker 出错", "err", err)
		os.Exit(1)
	}
	slog.Info("worker 已停止")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// leaseOptions 把租约参数接到队列(PRD §17.1:worker 心跳续租,stale 租约安全回收)。
// 默认窗口/心跳归 jobs 包(60s / 20s,间隔非法时按租约/3 收紧),这里只透传环境覆盖:
// TALLY_JOB_LEASE / TALLY_JOB_HEARTBEAT(Go duration)。
func leaseOptions() []jobs.Option {
	var opts []jobs.Option
	if d, err := time.ParseDuration(os.Getenv("TALLY_JOB_LEASE")); err == nil && d > 0 {
		opts = append(opts, jobs.WithLease(d))
	}
	if d, err := time.ParseDuration(os.Getenv("TALLY_JOB_HEARTBEAT")); err == nil && d > 0 {
		opts = append(opts, jobs.WithHeartbeatInterval(d))
	}
	return opts
}

// Legacy LLM transport is centralized in cmd/internal/airuntimelegacy.

// failureRunner 把 runner.Runner 适配成 failure.Runner(诊断的验证执行器)。
//
// P0-21 / INV-EXEC-01 / PRD §21.5:候选执行只发生在一次性 Git worktree 里 ——
// 从失败命令记录的工作目录解析出仓库根,以仓库当前 HEAD 为基线每次新建干净 worktree
// (failure_event 未携带失败时的 commit,当前 HEAD 是数据模型里可派生的最近似
// 「failure-related commit」固定基线),上一个 candidate 污染不到下一个;
// 成功 / 失败 / 取消一律清理(workspace.Manager.Cleanup 幂等且不继承取消)。
//
// spec.Command 是结构化 CommandSpec(PRD §21.6):Argv 原样执行,不做 strings.Fields,
// 跑什么(Argv)、在哪跑(Cwd,重映射进 worktree)、带什么环境(Env)全按 spec 来;
// Shell 语义本执行器不支持,显式拒绝。Fix 照写进 worktree(core.FileChange 约定:
// 执行端照写,不引 patch 引擎),越出 worktree 的路径一律拒绝。Target 非空 = G3:
// spec.Command 已由闸门换成该目标自己的构建命令,Target 只是目标名标签,原样跑。
// local 是候选命令执行器(生产注 LocalRunner,测试注 fake);仓库根解析走真 git
// (只读查询,与 workspace 包同一套 CLI 姿势)。
type failureRunner struct {
	local runner.Runner
	root  string // 一次性 worktree 的父目录;必须落在 owner checkout 之外(workspace.New 把关)
}

func (r failureRunner) Run(ctx context.Context, spec failure.RunSpec) (failure.RunResult, error) {
	cmd := spec.Command
	if cmd.Shell {
		return failure.RunResult{}, fmt.Errorf("候选执行拒绝 shell 语义命令(PRD §21.6 只认结构化 argv): %v", cmd.Argv)
	}
	if err := cmd.Validate(); err != nil {
		return failure.RunResult{}, fmt.Errorf("候选命令不合法: %w", err)
	}
	// Fix 路径先做静态闸:越出一次性 worktree 的写入一律拒绝 —— 否则等于变相改
	// owner 主 worktree(INV-EXEC-01)。校验先于建 worktree,坏输入零副作用。
	for _, c := range spec.Fix {
		if err := guardWorktreeRelPath(c.Path); err != nil {
			return failure.RunResult{}, err
		}
	}
	if cmd.Cwd == "" {
		return failure.RunResult{}, errors.New("候选命令缺工作目录,无从派生一次性 worktree")
	}

	repo, err := repoRoot(ctx, cmd.Cwd)
	if err != nil {
		return failure.RunResult{}, err
	}
	mgr, err := workspace.New(repo, r.root)
	if err != nil {
		return failure.RunResult{}, fmt.Errorf("建一次性 workspace: %w", err)
	}
	wt, err := mgr.Create(ctx, "HEAD")
	if err != nil {
		return failure.RunResult{}, fmt.Errorf("建一次性 worktree: %w", err)
	}
	// 成功/失败/取消一律清理(PRD §21.5:失败 candidate 不得污染下一个)。
	// Cleanup 幂等,且不继承调用方的取消 —— ctx 已取消时也能收尾。
	defer func() {
		if err := mgr.Cleanup(ctx, wt); err != nil {
			slog.Warn("清理一次性 worktree 失败", "path", wt.Path, "err", err)
		}
	}()

	// 记录的工作目录重映射进 worktree(仓库根 → worktree 根,相对位置保留);
	// 指向仓库之外的工作目录拒绝执行 —— 候选执行绝不离开一次性 worktree。
	dir, err := remapInWorktree(wt.Path, repo, cmd.Cwd)
	if err != nil {
		return failure.RunResult{}, err
	}

	for _, c := range spec.Fix {
		p := filepath.Join(wt.Path, filepath.FromSlash(filepath.Clean(c.Path)))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return failure.RunResult{}, fmt.Errorf("应用修复(建目录)%s: %w", c.Path, err)
		}
		if err := os.WriteFile(p, []byte(c.Content), 0o644); err != nil {
			return failure.RunResult{}, fmt.Errorf("应用修复 %s: %w", c.Path, err)
		}
	}

	// Timeout 是 CommandSpec 的建议超时(PRD §21.6):有值就按次执行限时,
	// 超时经 LocalRunner 以 ctx 错误上抛,由闸门/队列按取消语义处理。
	if cmd.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cmd.Timeout)
		defer cancel()
	}

	res, err := r.local.Run(ctx, runner.Command{Argv: cmd.Argv, Dir: dir, Env: flattenEnv(cmd.Env)})
	if err != nil {
		return failure.RunResult{}, err
	}
	return failure.RunResult{
		Green:  res.ExitCode == 0,
		Output: logpipe.RedactCoarse(res.Stdout + "\n" + res.Stderr),
	}, nil
}

// repoRoot 解析 dir 所在的 git 仓库根(git rev-parse --show-toplevel,结构化 argv 无 shell)。
// 不是仓库检出 → 报错拒绝执行(fail-closed):没有仓库就派生不出一次性 worktree,
// 绝不退回「直接在原目录跑」的旧语义(INV-EXEC-01)。环境隔离与 internal/workspace
// 的 gitClient 同一合同(不带全局/系统配置、不交互;那边未导出构造,这里按同一字面写)。
func repoRoot(ctx context.Context, dir string) (string, error) {
	res, err := (runner.LocalRunner{}).Run(ctx, runner.Command{
		Argv: []string{"git", "-C", dir, "rev-parse", "--show-toplevel"},
		Env: []string{
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_CONFIG_GLOBAL=" + os.DevNull,
			"GIT_CONFIG_SYSTEM=" + os.DevNull,
			"GIT_TERMINAL_PROMPT=0",
		},
	})
	if err != nil {
		return "", fmt.Errorf("解析失败工作目录的仓库根 %s: %w", dir, err)
	}
	if res.ExitCode != 0 {
		return "", fmt.Errorf("失败工作目录 %s 不是 git 仓库检出,无从派生一次性 worktree: %s",
			dir, strings.TrimSpace(res.Stderr))
	}
	return filepath.Clean(filepath.FromSlash(strings.TrimSpace(res.Stdout))), nil
}

// remapInWorktree 把记录的工作目录映射进一次性 worktree:仓库根换成 worktree 根,
// 相对位置保留。cwd 越出仓库(解析出 .. / 绝对路径)→ 拒绝执行 —— 候选执行
// 不落到 worktree 之外(含 owner 主 worktree,INV-EXEC-01)。
func remapInWorktree(wtPath, repoRoot, cwd string) (string, error) {
	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("解析候选工作目录 %s: %w", cwd, err)
	}
	rel, err := filepath.Rel(repoRoot, absCwd)
	if err != nil || rel == ".." || filepath.IsAbs(rel) ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("候选工作目录 %s 不在一次性 worktree 的仓库内,拒绝执行", cwd)
	}
	if rel == "." {
		return wtPath, nil
	}
	return filepath.Join(wtPath, rel), nil
}

// guardWorktreeRelPath 拒绝越出一次性 worktree 的仓库相对修复路径(空、绝对、.. 穿越)。
func guardWorktreeRelPath(p string) error {
	rel := filepath.Clean(filepath.FromSlash(p))
	if p == "" || rel == "." || rel == ".." || filepath.IsAbs(rel) ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("修复路径 %q 越出一次性 worktree,拒绝执行(INV-EXEC-01)", p)
	}
	return nil
}

// flattenEnv 把 CommandSpec 的环境追加项转成 runner 的 KEY=VALUE 列表。
// 键序固定(排序),同一条 spec 每次执行的环境一致。
func flattenEnv(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}
	out := make([]string, 0, len(env))
	for _, k := range slices.Sorted(maps.Keys(env)) {
		out = append(out, k+"="+env[k])
	}
	return out
}

func databaseURL() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	return "postgres://postgres:postgres@localhost:5432/tally?sslmode=disable"
}
