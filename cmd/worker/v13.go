// v13.go(P1-43)把 v1.3 Codegen/Verify 垂直路径接进 worker(PRD §36 P1
// Vertical Trust Loop),不动 legacy 环境/部署/失败诊断 job:
//
//	Tally 加载 Git HEAD 快照的 .tally 制品(TaskContract + 冻结
//	VerificationContract,P1-30 装载器)→ 一次性 worktree(P1-35)上跑
//	单 Task 顺序工作流内核(P1-39:CodegenRun → 不可变 ChangeSet,事实经
//	RunStoreSink 落 P1-33 观测面)→ 对固定候选做最终验证编排(P1-38,
//	visible 冻结协议 + required gates + held-out 注入;P2-01 起该组成抽为
//	internal/candidateverify 服务,worker 只做装配),VerificationRun /
//	Evidence 事实经该服务的 verification.RunSink→runstore 适配器落库 → 全程经
//	P1-41 Recorder 自动记录阶段耗时与 runtime 用量(P1-42 Select 产出的
//	context_hash 作为 CodegenRun 观测坐标)。
//
// 边界(与 AGENTS.md §3/§12 一致):
//   - 单 Task 垂直子集:没有 DAG、没有批量 autopilot;一次 job 恰好执行一个
//     standard TaskContract(PRD §36:P1 reference workflow 使用 standard Task)。
//   - Runtime 选择是配置(v13Config.Runtime → workerRuntimes 注册表选
//     fake|codex,P4-01),不是内核条件逻辑:workflow 内核只接收
//     agentruntime.Runtime 值,绝不按 runtime 名字分支(决策锁 13:
//     FakeRuntime 缺省,CodexRuntime 显式配置)。
//   - 候选执行与最终验证只发生在一次性 Git worktree / Tally 控制目录里,
//     owner 主 worktree 零写入(INV-EXEC-01);Git 装载走对象库只读快照。
//   - held-out 前提强制(AC5):契约含 held_out binding 时,holdout store
//     通道(决策锁 11)与 runtime 的 session.isolation 能力声明(Session
//     Isolation Canary 的前提,PRD §10.4/§35)缺一即拒绝执行,绝不静默跳过
//     held-out 最终验证(INV-VER-05)。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/artifacts"
	"tally/internal/candidateverify"
	"tally/internal/changeset"
	"tally/internal/contextpack"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/jobs"
	"tally/internal/measurement"
	"tally/internal/policy"
	"tally/internal/runner"
	"tally/internal/runstore"
	"tally/internal/store"
	"tally/internal/taskcontract"
	"tally/internal/verification"
	"tally/internal/workflow"
	"tally/internal/workspace"
)

// KindV13Task 是 v1.3 垂直路径的 job 类型。kind 字符串由本文件(设计期)定义,
// 复用同一 jobs 队列与租约生命周期;legacy 环境部署/诊断 job 的注册原样保留(AC2)。
// payload 只带坐标(V13TaskPayload),合同内容以 Git HEAD 快照为权威(INV-ART-01)。
const KindV13Task jobs.Kind = "v13_task"

// V13TaskPayload 是垂直路径 job 的入队参数:执行哪个 TaskContract。
type V13TaskPayload struct {
	Task string `json:"task"` // TaskContract id,如 TASK-023(合同本体在 Git)
}

// v13Config 是垂直路径的全部配置面(main.go 从环境装配,测试可直填)。
// 除 Repo/WorktreeRoot 外全部有保守缺省:缺省配置即 FakeRuntime 本地执行。
type v13Config struct {
	// Repo 是 owner 内容仓库(owner 主 worktree,只读:装载走 Git 对象库)。
	Repo string
	// WorktreeRoot 是一次性 worktree 的父目录;必须落在 owner checkout 之外
	// (workspace.New 把关,拒绝即失败 —— 主 worktree 零写入,AC4)。
	WorktreeRoot string
	// HoldoutRoot 是 Tally 控制的 holdout 文件系统根(决策锁 11);空 = 通道
	// 未接线,含 held_out binding 的契约会拒绝执行(AC5)。
	HoldoutRoot string
	// Runtime 是 runtime 适配器选择:fake(缺省)| codex(AC3:配置,不是
	// 内核条件逻辑)。
	Runtime string
	// CodexBin 显式指定 codex CLI 路径;空则由适配器按探针顺序发现。
	CodexBin string
	// DSH canary 配置面(P4-06):DSH developer-preview 适配器缺省禁用,
	// 只有显式配置( runtime=dsh + canary 开关 + 二进制路径)才可构造。
	// DSHModel/DSHProvider 是期望配置(--patch overlay),与实际生效模型
	// 是两个事实(runtimes/dsh 身份纪律)。
	DSHCanary   bool
	DSHBin      string
	DSHModel    string
	DSHProvider string
	// Profile / Model / Behavior 是 CodegenRun 的归因坐标(§11.2.1/§15.2)。
	Profile  string
	Model    string
	Behavior string
	// EnvName 是验证环境的稳定名称(与 EnvironmentIdentity 哈希合成环境身份)。
	EnvName string
}

// v13ConfigFromEnv 从环境装配配置。TALLY_V13_* 系列;Repo 沿用 worker 的
// 内容仓库根(TALLY_CONTENT_REPO)。
func v13ConfigFromEnv(repo string) v13Config {
	return v13Config{
		Repo:         repo,
		WorktreeRoot: envOr("TALLY_V13_WORKTREE_ROOT", filepath.Join(os.TempDir(), "tally-v13-worktrees")),
		HoldoutRoot:  os.Getenv("TALLY_V13_HOLDOUT_ROOT"),
		Runtime:      envOr("TALLY_V13_RUNTIME", "fake"),
		CodexBin:     os.Getenv("TALLY_V13_CODEX_BIN"),
		DSHCanary:    os.Getenv("TALLY_DSH_CANARY") == "1",
		DSHBin:       os.Getenv("TALLY_DSH_BIN"),
		DSHModel:     os.Getenv("TALLY_DSH_MODEL"),
		DSHProvider:  os.Getenv("TALLY_DSH_PROVIDER"),
		Profile:      envOr("TALLY_V13_PROFILE", "default-coding"),
		Model:        envOr("TALLY_V13_MODEL", "default"),
		Behavior:     envOr("TALLY_V13_BEHAVIOR", "v1"),
		EnvName:      envOr("TALLY_V13_ENV", "worker-local"),
	}
}

// v13Handler 构造垂直路径的队列 handler。返回 kind 与 handler,由 main.go 注册
// 进同一 handlers 表 —— 注册本身没有开关:worker 恒能执行 v13 job,与 legacy job 并存。
func v13Handler(cfg v13Config, q *store.Queries) (jobs.Kind, jobs.Handler) {
	return KindV13Task, func(ctx context.Context, payload json.RawMessage) error {
		var p V13TaskPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("v13: 解析 payload: %w", err)
		}
		taskID := strings.TrimSpace(p.Task)
		if taskID == "" {
			return errors.New("v13: payload 缺少 task")
		}
		res, err := runV13Task(ctx, v13Deps{cfg: cfg, queries: q}, taskID)
		if err != nil {
			return err
		}
		if res.Verify != nil {
			slog.Info("v13 垂直路径完成",
				"task", taskID,
				"changeset", string(res.Codegen.ChangeSet.ID()),
				"verification", string(res.Verify.Status),
				"verification_run", res.Verify.RunID)
		}
		return nil
	}
}

// v13Deps 是垂直路径的依赖面。queries 为生产事实查询集;runtime 与两个 sink 是
// 测试注入缝(nil = 生产路径:按配置选 runtime、用 runstore 适配器)。
type v13Deps struct {
	cfg     v13Config
	queries *store.Queries

	runtime      agentruntime.Runtime
	workflowSink workflow.RunSink
	verifySink   verification.RunSink
}

// v13Result 是一次垂直路径执行的结果值:codegen 侧 Outcome + (固定候选时)
// 最终验证结论。二者都只是观测结果;生命周期状态由证据另行推导(§18.1)。
type v13Result struct {
	Codegen workflow.Outcome
	Verify  *verification.FinalOutcome
}

// runV13Task 执行一个 approved standard TaskContract 的垂直路径(AC1):
// CodegenRun→ChangeSet→Verification。返回 error 仅表示基础设施/配置故障;
// 领域内的 NEEDS_REVIEW / cancelled 是正常结果(job 成功,事实已落库)。
//
// 「approved」的语义:合同与冻结验证契约都取自 Git HEAD 快照(P1-30 装载器,
// 含全部静态校验与 P1 standard freeze)—— Git 是 desired 侧唯一来源,工作区
// 未提交内容不可见、不可执行;profile 必须 standard(PRD §36)。
func runV13Task(ctx context.Context, deps v13Deps, taskID string) (v13Result, error) {
	cfg := deps.cfg

	// 1. Git HEAD 快照装载:TaskContract + 冻结 VerificationContract。
	snap, err := artifacts.Load(cfg.Repo, "HEAD")
	if err != nil {
		return v13Result{}, fmt.Errorf("v13: 装载 %s 的 .tally 制品(HEAD): %w", cfg.Repo, err)
	}
	task, ok := findV13Task(snap, taskID)
	if !ok {
		return v13Result{}, fmt.Errorf("v13: HEAD 快照(%s)没有有效的 task %s;Git 是 desired 侧唯一来源,不执行未提交或未通过静态校验的合同%s",
			snap.SourceCommit, taskID, rejectionSummary(snap))
	}
	if task.Contract.Profile != taskcontract.ProfileStandard {
		return v13Result{}, fmt.Errorf("v13: task %s profile=%q;P1 垂直路径只执行 standard TaskContract(PRD §36)", taskID, task.Contract.Profile)
	}
	frozen, err := findV13Verification(snap, taskID)
	if err != nil {
		return v13Result{}, err
	}

	// 2. Runtime 选择(配置面,AC3:workerRuntimes 注册表按名字选适配器,
	// P4-01)+ held-out 前提强制(AC5)。
	rt := deps.runtime
	if rt == nil {
		rt, err = workerRuntimes.build(cfg)
		if err != nil {
			return v13Result{}, fmt.Errorf("v13: %w", err)
		}
	}
	if err := enforceHeldOutPrerequisites(cfg, rt, frozen); err != nil {
		return v13Result{}, err
	}

	// 3. 一次性 worktree 表(P1-35)。workspace.New 拒绝把一次性根放进 owner
	// checkout:配置错误在开跑前失败,而不是中途污染主 worktree。
	mgr, err := workspace.New(cfg.Repo, cfg.WorktreeRoot)
	if err != nil {
		return v13Result{}, fmt.Errorf("v13: 建 workspace manager: %w", err)
	}

	// 4. ContextPack(P1-42 Select):context_hash 是 CodegenRun 的观测坐标。
	packHash, err := v13ContextPack(cfg.Repo, snap.SourceCommit, task.Contract, frozen)
	if err != nil {
		return v13Result{}, err
	}

	// 5. 事实面 sink(P1-33 适配)+ 度量录制器(P1-41)。
	runs := (*runstore.Store)(nil)
	if deps.queries != nil {
		runs = runstore.New(deps.queries)
	}
	wfSink := deps.workflowSink
	if wfSink == nil {
		if runs == nil {
			return v13Result{}, errors.New("v13: 缺少事实存储且未注入 workflow sink")
		}
		wfSink = workflow.NewRunStoreSink(runs)
	}
	rec := measurement.NewRecorder(nil)
	if deps.queries != nil {
		rec = measurement.NewRecorder(measurement.New(deps.queries))
	}

	// 6. 单 Task 顺序执行(P1-39 内核):CodegenRun → 0..1 个不可变 ChangeSet。
	spec := workflow.TaskExecutionSpec{
		Project:              core.ProjectID(snap.ProjectID),
		Task:                 task.Contract,
		Role:                 "coding",
		Policy:               policy.DefaultExecutionPolicy(),
		BaseRevision:         snap.SourceCommit,
		Profile:              cfg.Profile,
		Runtime:              newMeasuringRuntime(rt, rec, taskID, cfg),
		Model:                cfg.Model,
		BehaviorRevision:     cfg.Behavior,
		VerificationRevision: frozen.Hash(),
		ContextPackHash:      packHash,
		Worktrees:            workspace.NewRunTable(mgr),
		Sink:                 wfSink,
		Hooks:                workflow.Hooks{FixCandidate: fixCandidateFromWorktree},
	}
	texec, err := workflow.NewTaskExecution(spec)
	if err != nil {
		return v13Result{}, fmt.Errorf("v13: 构造 TaskExecution: %w", err)
	}

	phaseStart(rec, taskID, measurement.PhaseAgentExecution)
	out, execErr := texec.Execute(ctx)
	phaseEnd(rec, taskID, measurement.PhaseAgentExecution)
	if execErr != nil {
		return v13Result{}, fmt.Errorf("v13: codegen 执行失败: %w", execErr)
	}
	res := v13Result{Codegen: out}
	if out.Status != workflow.TaskCompleted || out.ChangeSet == nil {
		// 领域内落点(NEEDS_REVIEW / cancelled):事实已在 codegen_run,job 成功。
		slog.Warn("v13 codegen 未固定候选",
			"task", taskID, "status", string(out.Status), "reason", string(out.Reason))
		return res, nil
	}

	// 7. 最终验证(P1-38 编排的控制面接线,组成在 internal/candidateverify):
	// visible 冻结协议 + gates + held-out 注入。
	phaseStart(rec, taskID, measurement.PhaseFinalValidation)
	vout, verr := runV13Verification(ctx, deps, cfg, frozen, out.ChangeSet, rt)
	phaseEnd(rec, taskID, measurement.PhaseFinalValidation)
	if verr != nil {
		return res, verr
	}
	res.Verify = vout
	return res, nil
}

// ---- 装载辅助 ----

func findV13Task(snap *artifacts.Snapshot, taskID string) (artifacts.TaskArtifact, bool) {
	for _, a := range snap.Tasks {
		if a.Contract.ID == taskID {
			return a, true
		}
	}
	return artifacts.TaskArtifact{}, false
}

// findV13Verification 按 task_id 定位冻结验证契约。垂直路径必须有验证契约:
// 没有 independent 验证面的执行不构成 v1.3 垂直路径(§36.1 A.17-20)。
func findV13Verification(snap *artifacts.Snapshot, taskID string) (*verification.Frozen, error) {
	for _, v := range snap.Verifications {
		if v.Frozen.TaskID() == taskID {
			return v.Frozen, nil
		}
	}
	return nil, fmt.Errorf("v13: HEAD 快照没有 task %s 的有效 VerificationContract(垂直路径必须有冻结验证契约,§36.1)%s",
		taskID, rejectionSummary(snap))
}

// rejectionSummary 把装载器拒收的制品(路径 + 确定性原因)拼进错误信息:
// 「没有这份合同」与「合同写坏了」对操作者是两种不同的修复动作。
func rejectionSummary(snap *artifacts.Snapshot) string {
	if len(snap.Rejected) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(";被拒收的制品:")
	for i, r := range snap.Rejected {
		if i > 0 {
			b.WriteString(" | ")
		}
		b.WriteString(r.Path + ": " + r.Reason)
	}
	return b.String()
}

// enforceHeldOutPrerequisites 强制 held-out 前提(AC5)。契约含 held_out binding
// 时,执行前必须同时满足:
//  1. holdout store 通道已接线(Tally 控制文件系统根,决策锁 11)—— 缺失即拒绝,
//     绝不「降级为只跑 visible」:跳过 held-out 最终验证等于伪造 VERIFIED 路径;
//  2. runtime 声明 session.isolation —— 该能力声明以 Session Isolation Canary
//     通过为前提(PRD §10.4/§35)。FakeRuntime 声明它;CodexRuntime v1 因共享
//     CODEX_HOME 的 rollout 可检索风险不声明,因此当前不能承载含 held_out 的契约
//     (这是能力协商事实,不是本文件的策略发明)。
func enforceHeldOutPrerequisites(cfg v13Config, rt agentruntime.Runtime, f *verification.Frozen) error {
	if !candidateverify.ContractHasHeldOut(f) {
		return nil
	}
	if strings.TrimSpace(cfg.HoldoutRoot) == "" {
		return errors.New("v13: 契约含 held_out binding 但 holdout 通道未接线(设 TALLY_V13_HOLDOUT_ROOT);拒绝执行(INV-VER-05,AC5)")
	}
	d := rt.Descriptor()
	if !d.Capabilities.Has(agentruntime.CapSessionIsolation) {
		return fmt.Errorf("v13: runtime %s 未声明 session.isolation(Session Isolation Canary 前提不满足),拒绝执行含 held_out binding 的契约", d.Name)
	}
	return nil
}

// ---- ContextPack(P1-42)----

// v13ContextPack 组装最小 ContextPack 并返回 context_hash。候选面 = base 提交
// 树里 scope.allowed_paths 之下的 tracked 文件 + visible test binding 指定的
// 路径(Select 自己裁决 forbidden/held-out/非测试代码并登记台账)。候选 Content
// 留空:P1 不设 token 预算,内容只用于预算计量,包内只携带路径与合同指定 refs。
func v13ContextPack(repo, base string, task taskcontract.TaskContract, f *verification.Frozen) (string, error) {
	tracked, err := gitTreeFiles(context.Background(), repo, base)
	if err != nil {
		return "", fmt.Errorf("v13: 列举 base %s 的 tracked 文件: %w", base, err)
	}
	visible := visibleTestPaths(f)
	roots := task.Scope.AllowedPaths

	seen := map[string]bool{}
	var candidates []contextpack.Candidate
	add := func(p string) {
		if p == "" || seen[p] {
			return
		}
		seen[p] = true
		candidates = append(candidates, contextpack.Candidate{Path: p})
	}
	for _, p := range tracked {
		if underAnyRoot(roots, p) {
			add(p)
		}
	}
	for _, p := range visible {
		add(p)
	}

	res, err := contextpack.Select(contextpack.SelectInput{
		Task:         task,
		Verification: f,
		Candidates:   candidates,
	})
	if err != nil {
		return "", fmt.Errorf("v13: 组装 ContextPack: %w", err)
	}
	return res.Pack.Hash(), nil
}

// visibleTestPaths 收集冻结契约中 visible test binding 指定的 repo 路径。
func visibleTestPaths(f *verification.Frozen) []string {
	var out []string
	c := f.Contract()
	for _, acc := range c.Acceptances {
		for _, b := range acc.EvidenceBindings {
			if b.VisibilityPolicy == verification.VisibilityVisible &&
				b.Evidence.Type == verification.EvidenceTest && b.Evidence.Path != "" {
				out = append(out, b.Evidence.Path)
			}
		}
	}
	return out
}

// underAnyRoot 报告 path 是否等于任一根或位于任一根之下(目录式前缀;与
// contextpack 的 scope 匹配同一语义)。
func underAnyRoot(roots []string, path string) bool {
	for _, root := range roots {
		root = strings.TrimSuffix(root, "/")
		if root == "" {
			continue
		}
		if path == root || strings.HasPrefix(path, root+"/") {
			return true
		}
	}
	return false
}

// gitTreeFiles 用结构化 argv 列举一个提交树的全部文件(只读查询,无 shell;
// 不读 owner 工作区 —— 未提交内容对执行面不可见,INV-ART-01)。
func gitTreeFiles(ctx context.Context, repo, rev string) ([]string, error) {
	res, err := (runner.LocalRunner{}).Run(ctx, runner.Command{
		Argv: []string{"git", "-C", repo, "ls-tree", "-r", "--name-only", "-z", rev},
		Env:  gitIsolationEnv(),
	})
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("git ls-tree %s: %s", rev, strings.TrimSpace(res.Stderr))
	}
	var out []string
	for _, p := range strings.Split(res.Stdout, "\x00") {
		if p != "" {
			out = append(out, p)
		}
	}
	return out, nil
}

// ---- 候选固定(workflow.FixCandidate 钩子)----

// fixCandidateFromWorktree 把一次「完成声明」固定为候选快照:对 run 的一次性
// worktree 跑 git status(结构化 argv,-z 机器格式,关 rename 检测),改动文件
// 以改后完整内容收进快照(候选身份由内容推导,P1-05),candidate revision 取
// worktree HEAD。没有任何改动 = 本轮无可固定候选(ok=false,内核按失败迭代计)。
// 只读 worktree,绝不写 owner 主 worktree(INV-EXEC-01)。
func fixCandidateFromWorktree(ctx context.Context, v workflow.IterationView) (workflow.CandidateSnapshot, bool, error) {
	local := runner.LocalRunner{}
	res, err := local.Run(ctx, runner.Command{
		Argv: []string{"git", "-C", v.WorkspacePath, "status", "--porcelain=v1", "-z", "--no-renames", "--untracked-files=all"},
		Env:  gitIsolationEnv(),
	})
	if err != nil {
		return workflow.CandidateSnapshot{}, false, fmt.Errorf("git status %s: %w", v.WorkspacePath, err)
	}
	if res.ExitCode != 0 {
		return workflow.CandidateSnapshot{}, false, fmt.Errorf("git status %s: %s", v.WorkspacePath, strings.TrimSpace(res.Stderr))
	}
	entries, err := parseGitStatusZ(res.Stdout)
	if err != nil {
		return workflow.CandidateSnapshot{}, false, err
	}

	var files []changeset.ChangedFile
	for _, e := range entries {
		if strings.Contains(e.code, "D") {
			return workflow.CandidateSnapshot{}, false, fmt.Errorf(
				"候选包含删除改动 %s:P1 候选固定只收改后完整内容,删除文件无法以内容语义进入不可变快照", e.path)
		}
		if err := guardWorktreeRelPath(e.path); err != nil {
			return workflow.CandidateSnapshot{}, false, err
		}
		content, err := os.ReadFile(filepath.Join(v.WorkspacePath, filepath.FromSlash(filepath.Clean(e.path))))
		if err != nil {
			return workflow.CandidateSnapshot{}, false, fmt.Errorf("读取候选文件 %s: %w", e.path, err)
		}
		files = append(files, changeset.ChangedFile{Path: e.path, Content: string(content)})
	}
	if len(files) == 0 {
		return workflow.CandidateSnapshot{}, false, nil
	}

	rev, err := local.Run(ctx, runner.Command{
		Argv: []string{"git", "-C", v.WorkspacePath, "rev-parse", "HEAD"},
		Env:  gitIsolationEnv(),
	})
	if err != nil || rev.ExitCode != 0 {
		return workflow.CandidateSnapshot{}, false, fmt.Errorf("解析 worktree HEAD: %s", strings.TrimSpace(rev.Stderr))
	}
	return workflow.CandidateSnapshot{
		CandidateRevision: strings.TrimSpace(rev.Stdout),
		Files:             files,
	}, true, nil
}

// statusEntry 是 git status --porcelain -z 的一条记录。
type statusEntry struct {
	code string // XY 两字符
	path string
}

// parseGitStatusZ 解析 -z 机器格式:NUL 分隔的 "XY PATH" 记录(--no-renames
// 保证不出现 -> 形态;-z 保证路径不做 quotePath 转义)。
func parseGitStatusZ(out string) ([]statusEntry, error) {
	var entries []statusEntry
	for _, rec := range strings.Split(out, "\x00") {
		if rec == "" {
			continue
		}
		if len(rec) < 4 || rec[2] != ' ' {
			return nil, fmt.Errorf("v13: 无法解析 git status 记录 %q", rec)
		}
		entries = append(entries, statusEntry{code: rec[:2], path: rec[3:]})
	}
	return entries, nil
}

// ---- 最终验证(P1-38 编排的控制面接线;组成在 internal/candidateverify)----

// runV13Verification 装配并调用可复用候选验证服务(P2-01):候选 materialize +
// 最终验证编排组成(visible 冻结协议 + required gates + held-out 注入 + RunSink
// 落库)全部在 internal/candidateverify;本文件只装配 worker 的生产依赖(owner
// 仓库与一次性根、holdout 通道根、验证环境名、evidence provenance 通道、事实
// 存储与测试注入 sink)。行为与抽取前完全同构。
func runV13Verification(ctx context.Context, deps v13Deps, cfg v13Config, frozen *verification.Frozen, cs *changeset.ChangeSet, rt agentruntime.Runtime) (*verification.FinalOutcome, error) {
	svc := candidateverify.New(candidateverify.Config{
		Repo:         cfg.Repo,
		WorktreeRoot: cfg.WorktreeRoot,
		HoldoutRoot:  cfg.HoldoutRoot,
		EnvName:      cfg.EnvName,
		Recorder:     evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-worker/v13"},
	}, candidateverify.Deps{Queries: deps.queries, Sink: deps.verifySink})
	return svc.Verify(ctx, candidateverify.Request{
		Contract:  frozen,
		ChangeSet: cs,
		Runtime:   rt.Descriptor().Name,
	})
}

// ---- 度量观测缝(P1-41)----

// measuringRuntime 把 P1-41 的 runtime 用量观测缝接到任意 runtime 适配器:
// 每次 Execute 的墙钟耗时、终态与自报用量经 Recorder 观测(AC4:runtime 自报
// 用量是 claim;provider 未上报与观测到 0 是两种状态 —— 零值 Usage 按未上报
// 处理,绝不伪造 0 token 事实)。观测失败只告警:度量是观测面,不阻断执行。
// 透传的 Descriptor 保证能力协商与身份归因不受包装影响。
type measuringRuntime struct {
	inner    agentruntime.Runtime
	rec      *measurement.Recorder
	taskID   string
	model    string
	behavior string
}

func newMeasuringRuntime(inner agentruntime.Runtime, rec *measurement.Recorder, taskID string, cfg v13Config) agentruntime.Runtime {
	return &measuringRuntime{inner: inner, rec: rec, taskID: taskID, model: cfg.Model, behavior: cfg.Behavior}
}

func (m *measuringRuntime) Descriptor() agentruntime.RuntimeDescriptor { return m.inner.Descriptor() }

func (m *measuringRuntime) Start(ctx context.Context, req agentruntime.StartRequest) (agentruntime.Session, error) {
	s, err := m.inner.Start(ctx, req)
	if err != nil {
		return nil, err
	}
	return &measuringSession{rt: m, inner: s}, nil
}

type measuringSession struct {
	rt    *measuringRuntime
	inner agentruntime.Session
}

func (s *measuringSession) Execute(ctx context.Context, req agentruntime.ExecuteRequest) (agentruntime.AgentRunResult, error) {
	start := time.Now()
	res, err := s.inner.Execute(ctx, req)
	usage := measurement.RuntimeUsage{
		TaskID:           s.rt.taskID,
		UseCase:          "codegen_iteration",
		CostPlane:        measurement.CostPlaneExecution,
		Runtime:          s.rt.inner.Descriptor().Name,
		Model:            s.rt.model,
		BehaviorRevision: s.rt.behavior,
		Latency:          time.Since(start),
		Status:           usageStatusFor(res.Status, err, ctx),
		RecordedAt:       time.Now(),
		Tokens:           tokenObservation(res.Usage),
	}
	if _, uerr := s.rt.rec.RecordRuntimeUsage(ctx, usage); uerr != nil {
		slog.Warn("v13: 记录 runtime 用量观测失败(不影响执行)", "task", s.rt.taskID, "err", uerr)
	}
	return res, err
}

func (s *measuringSession) Cancel(ctx context.Context, id agentruntime.RunID) error {
	return s.inner.Cancel(ctx, id)
}

func (s *measuringSession) Events(ctx context.Context) (agentruntime.EventStream, error) {
	return s.inner.Events(ctx)
}

func (s *measuringSession) Result(ctx context.Context, id agentruntime.RunID) (agentruntime.AgentRunResult, error) {
	return s.inner.Result(ctx, id)
}

func (s *measuringSession) Close(ctx context.Context) error { return s.inner.Close(ctx) }

func usageStatusFor(st agentruntime.RunStatus, err error, ctx context.Context) measurement.UsageStatus {
	if err != nil {
		if ctx.Err() != nil {
			return measurement.UsageCancelled
		}
		return measurement.UsageFailed
	}
	switch st {
	case agentruntime.RunCompleted:
		return measurement.UsageSucceeded
	case agentruntime.RunCancelled:
		return measurement.UsageCancelled
	default:
		return measurement.UsageFailed
	}
}

// tokenObservation 把 runtime 自报用量转成 AC4 观测:零值 = 未上报(不可用),
// 非零 = 已上报的观测值。
func tokenObservation(u agentruntime.Usage) measurement.TokenUsage {
	if u.InputTokens == 0 && u.OutputTokens == 0 {
		return measurement.UnavailableTokens()
	}
	return measurement.ReportedTokens(u.InputTokens, u.OutputTokens)
}

// ---- 阶段计时(P1-41 AC1)----

func phaseStart(rec *measurement.Recorder, taskID string, phase measurement.LifecyclePhase) {
	if err := rec.PhaseStarted(taskID, phase, time.Now()); err != nil {
		slog.Warn("v13: 开启阶段计时失败", "task", taskID, "phase", string(phase), "err", err)
	}
}

func phaseEnd(rec *measurement.Recorder, taskID string, phase measurement.LifecyclePhase) {
	if _, err := rec.PhaseEnded(taskID, phase, time.Now()); err != nil {
		slog.Warn("v13: 关闭阶段计时失败", "task", taskID, "phase", string(phase), "err", err)
	}
}

// ---- git 纪律(与 internal/workspace 同一隔离合同)----

// gitIsolationEnv 让 git 只读查询不带全局/系统配置、不交互(与 internal/workspace
// 的 gitClient 同一合同)。
func gitIsolationEnv() []string {
	return []string{
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_SYSTEM=" + os.DevNull,
		"GIT_TERMINAL_PROMPT=0",
	}
}
