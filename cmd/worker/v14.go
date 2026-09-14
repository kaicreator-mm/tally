// v14.go(P2-09)把 v1.4 外层闭环(PRD §5 的 Execute→Verify→Repair→
// Accept/Escalate 段)接进 worker 作业面 —— 与 v1.3 垂直路径(v13.go)并存,
// 同一 jobs 队列与租约生命周期;legacy 环境/部署/诊断 job 与 v13 job 原样保留
// (PRD §34 NOW:Auto Repair 与 v1.3 Trust Core 并行,NEXT 前 v13 基线不拆)。
//
// 组成(P2-08 已落地的包内组合根,internal/closedloop.NewLoop):
//
//	WorkspaceExecutor(一次性 worktree + 修复基物化)
//	→ ServiceVerifier(candidateverify 服务 → verdict + 有界 ERROR 复验)
//	→ FailureRouter(P2-03,分类源 = FinalOutcome 账面)
//	→ FeedbackSource(P2-04 安全投影,修复材料唯一通道)
//	→ RepairPolicy(P2-07,缺省 2 次,环境可覆盖)
//	→ LineageSink / ExposureSink(runstore append-only 适配,P2-05/06)
//
// 本文件只做装配与一个生产适配:闭环执行缝 closedloop.CodegenRunner。它把
// 一次外层尝试作为一次全新 CodegenRun 执行 —— 在 WorkspaceExecutor 打开的
// 一次性 worktree 内(修复尝试的 RepairBase 已由执行适配确定性物化),以
// attempt 推导的 RunID 作为该次 CodegenRun 身份(ADR-0001 §2 的幂等键),
// 内层迭代预算仍来自 ExecutionPolicy.Retry.MaxIterations(决策锁 7:内外
// 预算互不相通)。内层复用 v1.3 的既有件:候选固定钩子(fixCandidateFrom-
// Worktree)、ChangeSet 冻结(changeset.New)、Run 事实面(workflow.RunSink,
// 生产面 workflow.RunStoreSink)、runtime 用量观测(measuringRuntime)。
// v13 的内层升级阶梯(workflow.TaskExecution 的 ResolveEscalation)不进入
// 本路径:验证驱动的返工改由外层 Repair 承担(ADR-0001 §1),内层耗尽即
// 「无可固定候选」,由闭环 Engine 落 no_candidate 升级。
//
// 边界(与 AGENTS.md §3/§12 一致):
//   - 单 Task 垂直子集:一次 job 恰好执行一个 standard TaskContract 的闭环;
//   - runtime 选择是配置,闭环与适配层绝不按 runtime 名字分支(AC3):唯一
//     的条件逻辑是能力协商(CapModelSelect 是否携带 model profile,PRD §10.3);
//   - 候选执行与验证只发生在一次性 Git worktree(Tally 控制目录),owner 主
//     worktree 零写入(INV-EXEC-01);worktree 由 WorkspaceExecutor 开、
//     尝试结束即丢弃,candidateverify 的验证 workspace 自管自清;
//   - 修复预算来自配置(缺省 2,决策锁 23),零/负/超硬上限一律拒绝构造;
//   - job 取消沿 handler ctx 直达闭环 Engine(AC5):取消是一等终态,不产生
//     verdict、不消耗修复预算;事实落库用不继承取消的派生 context。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/artifacts"
	"tally/internal/candidateverify"
	"tally/internal/changeset"
	"tally/internal/closedloop"
	"tally/internal/contextpack"
	"tally/internal/contract"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/jobs"
	"tally/internal/measurement"
	"tally/internal/outcome"
	"tally/internal/policy"
	"tally/internal/runstore"
	"tally/internal/store"
	"tally/internal/taskcontract"
	"tally/internal/verification"
	"tally/internal/workflow"
	"tally/internal/workspace"
)

// KindV14Task 是 v1.4 闭环路径的 job 类型。kind 字符串由本文件(设计期)定义,
// 与 KindV13Task、legacy kind 互不重叠;jobs.job.kind 是自由 TEXT,无需迁移。
const KindV14Task jobs.Kind = "v14_task"

// V14TaskPayload 是闭环 job 的入队参数:执行哪个 TaskContract。
// 合同内容以 Git HEAD 快照为权威(INV-ART-01),payload 只带坐标。
type V14TaskPayload struct {
	Task string `json:"task"` // TaskContract id,如 TASK-023(合同本体在 Git)
}

// v14Config 是闭环路径的全部配置面(main.go 从环境装配,测试可直填)。
// 共享面(仓库/一次性根/holdout/runtime 选择/归因坐标/环境名)与 v13 同构,
// 复用其类型与环境装配习惯;v1.4 特有面只有修复预算。
type v14Config struct {
	// v13 是共享配置面(Repo/WorktreeRoot/HoldoutRoot/Runtime/CodexBin/
	// Profile/Model/Behavior/EnvName);字段语义见 v13.go。
	v13 v13Config
	// MaxRepair 是修复预算覆盖(TALLY_V14_MAX_REPAIR):0 = 缺省 2 次
	// (决策锁 23:策略默认值,不是架构常量);显式值经 NewRepairPolicy
	// 校验,零/负/超硬上限拒绝 —— 绝不静默放宽。
	MaxRepair int
}

// v14ConfigFromEnv 从环境装配配置。TALLY_V14_* 系列;Repo 沿用 worker 的
// 内容仓库根(TALLY_CONTENT_REPO)。
func v14ConfigFromEnv(repo string) v14Config {
	maxRepair := 0
	if v := os.Getenv("TALLY_V14_MAX_REPAIR"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			slog.Warn("v14: TALLY_V14_MAX_REPAIR 非整数,按缺省修复预算执行", "value", v)
		} else {
			maxRepair = n
		}
	}
	return v14Config{
		v13: v13Config{
			Repo:         repo,
			WorktreeRoot: envOr("TALLY_V14_WORKTREE_ROOT", filepath.Join(os.TempDir(), "tally-v14-worktrees")),
			HoldoutRoot:  os.Getenv("TALLY_V14_HOLDOUT_ROOT"),
			Runtime:      envOr("TALLY_V14_RUNTIME", "fake"),
			CodexBin:     os.Getenv("TALLY_V14_CODEX_BIN"),
			Profile:      envOr("TALLY_V14_PROFILE", "default-coding"),
			Model:        envOr("TALLY_V14_MODEL", "default"),
			Behavior:     envOr("TALLY_V14_BEHAVIOR", "v1"),
			EnvName:      envOr("TALLY_V14_ENV", "worker-local"),
		},
		MaxRepair: maxRepair,
	}
}

// v14RepairPolicy 解析修复预算:0 = 缺省 2(决策锁 23);显式值必须落在
// NewRepairPolicy 的硬界内,非法值拒绝构造(fail-safe:宁可不起跑)。
func v14RepairPolicy(max int) (closedloop.RepairPolicy, error) {
	if max == 0 {
		return closedloop.DefaultRepairPolicy(), nil
	}
	return closedloop.NewRepairPolicy(max)
}

// v14Handler 构造闭环路径的队列 handler。返回 kind 与 handler,由 main.go 注册
// 进同一 handlers 表 —— 注册本身没有开关:worker 恒能执行 v14 job,与 v13/legacy
// job 并存(AC2)。
func v14Handler(cfg v14Config, q *store.Queries) (jobs.Kind, jobs.Handler) {
	return KindV14Task, func(ctx context.Context, payload json.RawMessage) error {
		var p V14TaskPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return fmt.Errorf("v14: 解析 payload: %w", err)
		}
		taskID := strings.TrimSpace(p.Task)
		if taskID == "" {
			return errors.New("v14: payload 缺少 task")
		}
		res, err := runV14Task(ctx, v14Deps{cfg: cfg, queries: q}, taskID)
		if err != nil {
			return err
		}
		// 领域终态(Accepted/Escalated/Cancelled)不是 job 失败:事实已落库。
		// 取消由队列按执行 ctx 自行裁决(保持 running 等回收),这里只留痕。
		switch res.Phase {
		case closedloop.PhaseAccepted:
			slog.Info("v14 闭环接受",
				"task", taskID,
				"changeset", string(res.Basis.ChangeSet),
				"verify_run", res.Basis.VerifyRun,
				"attempt", int(res.Basis.Attempt))
		case closedloop.PhaseEscalated:
			slog.Warn("v14 闭环升级",
				"task", taskID,
				"reason", string(res.Escalation.Reason),
				"detail", res.Escalation.Detail,
				"attempts", len(res.Attempts))
		case closedloop.PhaseCancelled:
			slog.Warn("v14 闭环取消", "task", taskID, "attempts", len(res.Attempts))
		}
		return nil
	}
}

// v14Deps 是闭环路径的依赖面。queries 为生产事实查询集;runtime/wfSink/verify/
// lineage/exposure 是测试注入缝(nil = 生产路径:按配置选 runtime、经 runstore
// 适配器落全部事实)。
type v14Deps struct {
	cfg     v14Config
	queries *store.Queries

	runtime  agentruntime.Runtime             // 缺省:workerRuntimes.build(cfg.v13)
	wfSink   workflow.RunSink                 // codegen 事实面;缺省 workflow.NewRunStoreSink
	verify   closedloop.CandidateVerification // 缺省:candidateverify 生产服务
	lineage  closedloop.LineageSink           // 缺省:closedloop.NewStoreLineageSink
	exposure closedloop.ExposureSink          // 缺省:closedloop.NewStoreExposureSink
	accept   closedloop.AcceptSink            // 缺省:closedloop.NewStoreAcceptSink(P6-01)
}

// runV14Task 执行一个 approved standard TaskContract 的 v1.4 闭环(PRD §5:
// Execute→Verify→Repair→Accept/Escalate)。执行前置顺序(P1-04,PRD §16):
// 同源 bundle 装载(DL-04)→ profile 准入 → **合同编译权威(P1-03
// CompileAuthority)**——编译先于任何 runtime 会话与一次性 workspace 创建,
// 编译权威错误类型化上抛,不开工。返回 error 仅表示基础设施/配置/编译权威
// 故障;领域终态(Escalated/Cancelled)是正常结果(job 成功,事实已落库)。
func runV14Task(ctx context.Context, deps v14Deps, taskID string) (closedloop.Result, error) {
	cfg := deps.cfg

	// 1. 同源执行 bundle(P1-01,DL-04):一次 revision 解析把同一任务身份的
	// FrozenIntent、TaskContract、冻结 VerificationContract 与策略快照全部装
	// 到同一个 pinned 提交 —— 混合来源在构造上不可能。Git 是 desired 侧唯一
	// 来源;项目身份由同一棵提交树的注册表声明(worker 运行事实坐标,空身份
	// fail-closed),profile 必须 standard。
	bundle, err := artifacts.LoadBundle(cfg.v13.Repo, "HEAD", taskID)
	if err != nil {
		return closedloop.Result{}, fmt.Errorf("v14: 装载 %s 的同源执行 bundle(HEAD): %w", cfg.v13.Repo, err)
	}
	if bundle.ProjectID == "" {
		return closedloop.Result{}, fmt.Errorf("v14: 提交 %s 缺少 %s(项目身份必须由 Git 注册表声明,PRD §6.4.1/§33.3)",
			bundle.SourceCommit, artifacts.ManifestPath)
	}
	task := bundle.Task.Contract
	if task.Profile != taskcontract.ProfileStandard {
		return closedloop.Result{}, fmt.Errorf("v14: task %s profile=%q;闭环路径只执行 standard TaskContract(PRD §36)", taskID, task.Profile)
	}
	frozen := bundle.Verification.Frozen

	// 2. 合同编译权威(P1-03,PRD §16「Worker 不允许绕过 Compiler」):编译是
	// 执行的第一道门,必须在任何 runtime 会话与一次性 workspace 创建之前完成。
	// 任一 fail-safe(拒收策略/无效风险依据/来源不一致/合同或编译拒收)都不
	// 产出 Compiled;类型化错误经 %w 原样上抛,调用方以 errors.Is 判类。这是
	// 执行前置失败,不是候选失败 —— 绝不进入闭环、绝不伪装成 verdict、绝不
	// 消耗修复预算(§5 信任不变量)。
	authority, err := bundle.CompileAuthority()
	if err != nil {
		return closedloop.Result{}, fmt.Errorf("v14: 编译 %s 的执行权威: %w", taskID, err)
	}
	// 编译身份入运行观测:task/verification/base 修订进 RunBegin 事实(下方
	// 同一 bundle 的 pinned 坐标);bundle 身份与编译哈希是这些坐标的确定性
	// 纯函数,可从同一提交独立复算(P1-03 确定性)。policy_requires 计数是
	// 防御上下文(P1-05)暴露面的观测坐标。
	slog.Info("v14 执行权威已编译",
		"task", authority.TaskID,
		"source_commit", authority.SourceCommit,
		"bundle", authority.BundleIdentity,
		"compiled", authority.Compiled.Hash(),
		"policy_requires", len(authority.Compiled.Requires().Items),
		"risk_signals", len(authority.Risk.Detected),
		"unknown_risk_signals", len(authority.Risk.Unknown))

	// 3. Runtime 选择(配置面:workerRuntimes 注册表按名字选适配器,P4-01,
	// 与 v13 同一缝)+ held-out 前提强制(复用 v13 同一合同)。
	rt := deps.runtime
	if rt == nil {
		rt, err = workerRuntimes.build(cfg.v13)
		if err != nil {
			return closedloop.Result{}, fmt.Errorf("v14: %w", err)
		}
	}
	if err := enforceHeldOutPrerequisites(cfg.v13, rt, frozen); err != nil {
		return closedloop.Result{}, err
	}

	// 4. 一次性 worktree(workspace.New 拒绝把一次性根放进 owner checkout:
	// 配置错误在开跑前失败,而不是中途污染主 worktree,INV-EXEC-01)。
	mgr, err := workspace.New(cfg.v13.Repo, cfg.v13.WorktreeRoot)
	if err != nil {
		return closedloop.Result{}, fmt.Errorf("v14: 建 workspace manager: %w", err)
	}
	worktrees := workspace.NewRunTable(mgr)

	// 5. ContextPack(P1-42 Select 复用;P1-05 compiled-aware):实现权威可见
	// 的上下文里注入编译权威的执行前 requires(PRD §8.3 Prevention)—— 防御
	// 要求来自命中策略的公开声明,是冻结合同派生的必现小节,纳入 context_hash。
	// 从此 Run 事实的 ContextPackHash 是编译权威内容的敏感函数:同一任务在策略
	// 换版后重跑/修复,即便候选面与合同内容不变,上下文坐标也必然改变 ——
	// 修复/重放不能在事实面上悄悄换权威。
	packHash, err := v14ContextPack(cfg.v13.Repo, bundle.SourceCommit, task, frozen, authority.Compiled.Requires())
	if err != nil {
		return closedloop.Result{}, err
	}

	// 6. 事实面与闭环端口(生产路径全部经 runstore append-only 适配)。
	runs := (*runstore.Store)(nil)
	if deps.queries != nil {
		runs = runstore.New(deps.queries)
	}
	wfSink := deps.wfSink
	if wfSink == nil {
		if runs == nil {
			return closedloop.Result{}, errors.New("v14: 缺少事实存储且未注入 workflow sink")
		}
		wfSink = workflow.NewRunStoreSink(runs)
	}
	verify := deps.verify
	if verify == nil {
		if deps.queries == nil {
			return closedloop.Result{}, errors.New("v14: 缺少事实存储且未注入验证服务")
		}
		verify = candidateverify.New(candidateverify.Config{
			Repo:         cfg.v13.Repo,
			WorktreeRoot: cfg.v13.WorktreeRoot,
			HoldoutRoot:  cfg.v13.HoldoutRoot,
			EnvName:      cfg.v13.EnvName,
			Recorder:     evidence.Recorder{Kind: evidence.RecorderVerificationRun, Actor: "tally-worker/v14"},
		}, candidateverify.Deps{Queries: deps.queries})
	}
	lineage := deps.lineage
	if lineage == nil && runs != nil {
		if lineage, err = closedloop.NewStoreLineageSink(runs); err != nil {
			return closedloop.Result{}, fmt.Errorf("v14: 接线取代事实出口: %w", err)
		}
	}
	exposure := deps.exposure
	if exposure == nil && runs != nil {
		if exposure, err = closedloop.NewStoreExposureSink(runs, frozen.Revision()); err != nil {
			return closedloop.Result{}, fmt.Errorf("v14: 接线暴露事实出口: %w", err)
		}
	}
	// accepted outcome 出口(P6-01):accept 转移落成之前追加 kind=accepted
	// 事件(幂等:同一 task/changeset 恰一条);写入失败显式 outcome_
	// unavailable 升级,绝不静默接受、绝不静默丢事件。事实存储缺席且未注入
	// 出口 = 未绑定(v1.4 行为;生产 worker 恒有事实存储,恒绑定)。
	accept := deps.accept
	if accept == nil && deps.queries != nil {
		if accept, err = closedloop.NewStoreAcceptSink(outcome.New(deps.queries)); err != nil {
			return closedloop.Result{}, fmt.Errorf("v14: 接线 accepted outcome 出口: %w", err)
		}
	}
	// Run 开始前已成立的暴露事实(append-only 投影,组合根读出后注入;
	// 谱系单调的 accept 守卫以同一账面裁决,INV-V14-08)。
	var seeds []evidence.Exposure
	if runs != nil {
		if seeds, err = runs.ExposuresForTask(ctx, core.ProjectID(bundle.ProjectID), taskID); err != nil {
			return closedloop.Result{}, fmt.Errorf("v14: 读暴露事实种子: %w", err)
		}
	}

	// 7. 度量观测缝(P1-41 复用):runtime 用量观测挂在每次内层 Execute。
	rec := measurement.NewRecorder(nil)
	if deps.queries != nil {
		rec = measurement.NewRecorder(measurement.New(deps.queries))
	}

	// 8. 内层 codegen 执行缝:一次调用 = 一次全新 CodegenRun(attempt 推导
	// 的 RunID;WorkspaceExecutor 打开的一次性 worktree 内执行)。执行权威
	// 已在第 2 步编译就位(PRD §16 no-bypass):runner 持有它,任何会话开启
	// 都晚于编译成功。
	runner := &v14CodegenRunner{
		project:   core.ProjectID(bundle.ProjectID),
		task:      task,
		role:      "coding",
		policy:    policy.DefaultExecutionPolicy(),
		profile:   cfg.v13.Profile,
		runtime:   newMeasuringRuntime(rt, rec, taskID, cfg.v13),
		model:     cfg.v13.Model,
		behavior:  cfg.v13.Behavior,
		verifRev:  frozen.Hash(),
		packHash:  packHash,
		authority: authority,
		sink:      wfSink,
		fix:       fixCandidateFromWorktree,
		now:       time.Now,
	}

	// 9. 修复预算(策略值,缺省 2)+ 闭环组合根(P2-08 NewLoop)。
	repair, err := v14RepairPolicy(cfg.MaxRepair)
	if err != nil {
		return closedloop.Result{}, fmt.Errorf("v14: %w", err)
	}
	loop, err := closedloop.NewLoop(closedloop.LoopConfig{
		Codegen:  runner,
		Verify:   verify,
		Contract: frozen,
		// 验收守卫(P1-06,DL-07/A-03):编译权威(P1-04 已必达)在此成为
		// accept 权威的输入 —— FinalVerified 不再单独授权普通 Accept,
		// 必需保证档未达成/未匹配风险/豁免条款 fail-safe 升级
		// (assurance_insufficient),绝不伪装成候选失败、绝不耗修复预算。
		Authority: authority.Compiled,
		Runtime:   rt.Descriptor().Name,
		Worktrees: worktrees,
		BaseRevision: func(context.Context, string) (string, error) {
			// 首次尝试基线 = bundle 来源提交(与编译权威同一 pinned 提交,
			// Git 权威);修复尝试的基线由冻结前候选携带(WorkspaceExecutor 裁量)。
			return bundle.SourceCommit, nil
		},
		Repair:    repair,
		Lineage:   lineage,
		Exposure:  exposure,
		Exposures: seeds,
		// accepted outcome 出口(P6-01):生产路径恒绑定(有事实存储即接线);
		// accept 转移落成前追加 accepted 事件,失败显式升级。
		AcceptSink: accept,
	})
	if err != nil {
		return closedloop.Result{}, fmt.Errorf("v14: 组装闭环: %w", err)
	}

	// 10. 执行闭环(handler ctx 直达 Engine:队列取消/租约丢失即闭环取消终态,
	// 不产生 verdict、不消耗修复预算,AC5)。
	res, err := loop.Run(ctx, taskID)
	if err != nil {
		return closedloop.Result{}, fmt.Errorf("v14: 闭环执行: %w", err)
	}
	return res, nil
}

// ---- 内层 codegen 执行缝(closedloop.CodegenRunner 的生产适配)----

// v14CodegenRunner 把一次外层尝试执行为一次全新 CodegenRun。实现闭环端口契约
// (internal/closedloop/executor.go):在 WorkspaceExecutor 打开的一次性
// worktree 内执行(修复尝试的 RepairBase 已物化,内容对会话可见);以
// req.RunID(attempt 幂等键)作为 CodegenRun 身份;至多固化一个不可变候选;
// error 只表示进程/基础设施/提供方失败,「内层收束但没有候选」返回 nil。
//
// 内层迭代预算来自 ExecutionPolicy.Retry.MaxIterations(决策锁 7:与外层
// RepairPolicy 零耦合);预算/墙钟耗尽与「无可固定候选」都是 nil 候选落点,
// 由闭环 Engine 裁决为升级。Run 事实经 workflow.RunSink 落库(生产面
// workflow.NewRunStoreSink):codegen_run 以 attempt id 开跑,父链接经
// RunBegin.EscalatedFrom 携带(闭环 Run 链的谱系槽位,ADR-0001 §2
// attempt-numbered, parent-linked)。
type v14CodegenRunner struct {
	project  core.ProjectID
	task     taskcontract.TaskContract
	role     string
	policy   policy.ExecutionPolicy
	profile  string
	runtime  agentruntime.Runtime
	model    string
	behavior string
	verifRev string
	packHash string
	// authority 是本次执行的已编译合同权威(P1-04):编译先于任何 runtime
	// 会话开启(runV14Task 第 2 步);runner 持有它供运行事实与防御上下文
	// (P1-05)取用 —— 权威只在执行前编译一次,runtime/候选侧绝不改写
	// (PRD §16:runtime 输出不是 Verification/Acceptance)。同一权威也是
	// 验收守卫的输入(P1-06):经 LoopConfig.Authority 进入闭环组合根,
	// accept 转移按它裁决(DL-07)。
	authority *artifacts.CompiledAuthority
	sink      workflow.RunSink
	fix       workflow.CandidateFixFunc
	now       func() time.Time
}

// CodegenRunner 端口编译期钉住。
var _ closedloop.CodegenRunner = (*v14CodegenRunner)(nil)

// RunCodegen 实现 closedloop.CodegenRunner。终态词汇与 v13 codegen_run 一致
// (completed/failed/cancelled);事实落库用不继承调用方取消的派生 context
// (取消是执行终态,不是事实丢失理由)。
func (r *v14CodegenRunner) RunCodegen(ctx context.Context, req closedloop.CodegenRunRequest) (*changeset.ChangeSet, error) {
	// 编译权威是 runner 的存在前提(P1-04 组合不变量,PRD §16 no-bypass):
	// 缺权威就无法投影防御上下文,宁可拒绝执行也绝不无声地在不被告知验收
	// 条件的情况下开跑(fail-safe,不是降级)。
	if r.authority == nil || r.authority.Compiled == nil {
		return nil, errors.New("v14: codegen runner 缺少编译权威,拒绝执行(PRD §16:不允许绕过编译器)")
	}
	runID := workflow.RunID(req.RunID)
	now := r.now
	if now == nil {
		now = time.Now
	}
	factCtx := context.WithoutCancel(ctx)

	// runtime 会话绑定本次尝试的一次性 worktree;可执行变更绝不在 owner 主
	// worktree 发生。期望 model 档位只在 runtime 声明 model.select 时携带
	// (能力协商分支,不按 runtime 名字分支,AC3)。
	desc := r.runtime.Descriptor()
	startReq := agentruntime.StartRequest{Workspace: req.WorkspacePath}
	if desc.Capabilities.Has(agentruntime.CapModelSelect) {
		startReq.ModelProfile = r.model
	}
	session, err := r.runtime.Start(ctx, startReq)
	if err != nil {
		return nil, fmt.Errorf("v14: %s 开启 runtime 会话: %w", req.RunID, err)
	}
	// 实际身份是可选观察:未实现 IdentityReporter = 未观察,零值降级,
	// 绝不用期望值补齐(决策锁 15)。
	var observed agentruntime.ObservedIdentity
	if rep, ok := session.(agentruntime.IdentityReporter); ok {
		observed = rep.Identity()
	}

	// 开跑事实:attempt 推导的 Run 身份 + Run 链父链接 + 观测坐标。
	// CapabilitySnapshot 是 runtime 声明能力的非空 JSON 快照(观测面要求)。
	capSnap, err := v14CapabilitySnapshot(desc)
	if err != nil {
		_ = session.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("v14: %s 观测能力快照: %w", req.RunID, err)
	}
	runRef, err := r.sink.BeginRun(factCtx, workflow.RunBegin{
		ID:                   runID,
		EscalatedFrom:        workflow.RunID(req.ParentRun),
		Project:              r.project,
		TaskID:               r.task.ID,
		TaskRevision:         r.task.Hash,
		Role:                 r.role,
		Profile:              r.profile,
		RuntimeName:          desc.Name,
		RuntimeVersion:       desc.Version,
		Model:                r.model,
		BehaviorRevision:     r.behavior,
		ObservedIdentity:     observed,
		BaseRevision:         req.BaseRevision,
		WorkspaceID:          req.WorkspacePath,
		CapabilitySnapshot:   capSnap,
		StartedAt:            now(),
		VerificationRevision: r.verifRev,
		ContextPackHash:      r.packHash,
	})
	if err != nil {
		_ = session.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("v14: %s 记录开跑: %w", req.RunID, err)
	}
	if strings.TrimSpace(runRef) == "" {
		_ = session.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("v14: %s 的 sink 未返回持久化 Run 引用", req.RunID)
	}

	// 墙钟预算:每个 Run 一个 deadline(§11.2 budgets.max_wall_time)。
	runCtx, cancelWall := context.WithTimeout(ctx, r.policy.Budgets.MaxWallTime)
	defer cancelWall()

	// 防御上下文(P1-05,PRD §8.3 Prevention):编译权威的执行前 requires
	// 上下文的确定性渲染行,随每次迭代指令告知实现权威「最终会按这些条件
	// 验收」。内容只来自命中策略的公开声明(contract.Bind 保证,INV-V14-07),
	// held-out 验证材料(store/ref、selector、测试内容)在该投影类型上不存在;
	// 与验证证据投影(Pack.Verification)和修复反馈(Repair context)是分离的
	// 小节 —— Prevention 与验证证据/holdouts 分面,互不混载。
	requires := r.authority.Compiled.Requires().Lines()

	var (
		iterations int
		inTokens   int64
		outTokens  int64
		changeSet  *changeset.ChangeSet
		status     runstore.RunStatus
		why        string
		retErr     error
		lastClaim  agentruntime.AgentRunResult // runtime 自述(claim,非事实)
	)

	for iterations < r.policy.Retry.MaxIterations {
		iterations++
		res, execErr := session.Execute(runCtx, agentruntime.ExecuteRequest{
			Instruction: v14Instruction(r.task, iterations, req.Feedback, requires),
		})
		lastClaim = res
		inTokens += res.Usage.InputTokens
		outTokens += res.Usage.OutputTokens
		if pErr := r.sink.RecordProgress(factCtx, runID, workflow.RunProgress{
			Iterations:  int32(iterations),
			InputTokens: &inTokens, OutputTokens: &outTokens,
		}); pErr != nil {
			status, why, retErr = runstore.RunFailed, "record_progress_failed", pErr
			break
		}

		// 1) 调用方取消优先于一切(队列关闭/租约丢失/任务取消)。
		if ctx.Err() != nil {
			status, why, retErr = runstore.RunCancelled, "cancelled", ctx.Err()
			break
		}
		// 2) 预算耗尽(墙钟/令牌):Run 终止且不产生候选(§11.2.1)。
		if runCtx.Err() != nil {
			status, why = runstore.RunFailed, "wall_time_budget_exhausted"
			break
		}
		if inTokens+outTokens > r.policy.Budgets.MaxTokens {
			status, why = runstore.RunFailed, "token_budget_exhausted"
			break
		}
		// 3) runtime 会话级故障(进程/提供方失败):会话已不可信,按基础设施
		// 错误上抛(闭环 Engine 落 infrastructure 升级,绝不伪装成候选事实,
		// 也绝不消耗外层修复预算,ADR-0001 §3)。
		if execErr != nil {
			status, why, retErr = runstore.RunFailed, "runtime_error", execErr
			break
		}
		// 4) runtime 自述失败/被取消:非产出迭代,计入有限重试。
		if res.Status != agentruntime.RunCompleted {
			continue
		}
		// 5) 完成声明 → 候选固定点:恰好 0..1 个不可变 ChangeSet。修复尝试的
		// worktree 里已有物化的冻结前候选,快照按 worktree 当前全部改动收取。
		snap, ok, fixErr := r.fix(ctx, workflow.IterationView{
			Run:           runID,
			Profile:       r.profile,
			WorkspacePath: req.WorkspacePath,
			BaseRevision:  req.BaseRevision,
			Iteration:     iterations,
			Claim:         res,
		})
		if fixErr != nil {
			status, why, retErr = runstore.RunFailed, "fix_candidate_failed", fixErr
			break
		}
		if !ok {
			continue
		}
		cs, csErr := changeset.New(changeset.Spec{
			Project:           r.project,
			BaseRevision:      req.BaseRevision,
			CandidateRevision: snap.CandidateRevision,
			Files:             snap.Files,
			Entities:          snap.Entities,
			ProducerRun:       runRef, // 持久化 Run 引用:快照回指存储层的 Run 行
			CreatedAt:         now(),
		})
		if csErr != nil {
			status, why, retErr = runstore.RunFailed, "changeset_failed", csErr
			break
		}
		if rErr := r.sink.RecordChangeSet(factCtx, runID, cs); rErr != nil {
			status, why, retErr = runstore.RunFailed, "record_changeset_failed", rErr
			break
		}
		changeSet = cs
		status = runstore.RunCompleted
		break
	}
	if status == "" {
		// 迭代预算耗尽:内层收束但没有可固定候选 —— 领域落点(nil 候选),
		// 由闭环 Engine 裁决为 no_candidate 升级(验证驱动的返工归外层 Repair)。
		status, why = runstore.RunFailed, "iterations_exhausted"
	}

	if finErr := r.sink.FinishRun(factCtx, workflow.RunFinish{
		ID:              runID,
		Status:          status,
		Iterations:      int32(iterations),
		InputTokens:     &inTokens,
		OutputTokens:    &outTokens,
		ResultPatchHash: v14ChangeSetID(changeSet),
		AgentClaim:      lastClaim.Summary,
		FailureReason:   why,
		FinishedAt:      now(),
	}); finErr != nil && retErr == nil {
		// 终态事实写不进去而执行侧又没有更早的错误:按基础设施失败上抛,
		// 绝不静默丢事实。
		_ = session.Close(context.WithoutCancel(ctx))
		return nil, fmt.Errorf("v14: %s 记录终态: %w", req.RunID, finErr)
	}
	// 会话关闭是资源回收,失败不遮蔽执行结论;worktree 由 WorkspaceExecutor
	// 统一销毁(尝试结束即丢弃,成功/失败/取消一律)。
	_ = session.Close(context.WithoutCancel(ctx))
	if retErr != nil {
		return nil, retErr
	}
	return changeSet, nil
}

func v14ChangeSetID(cs *changeset.ChangeSet) string {
	if cs == nil {
		return ""
	}
	return string(cs.ID())
}

// v14CapabilitySnapshot 把 runtime 自述能力编码成非空 JSON 对象(观测面要求,
// 与 workflow 内核同一形态)。
func v14CapabilitySnapshot(d agentruntime.RuntimeDescriptor) (json.RawMessage, error) {
	caps := d.Capabilities.Sorted()
	names := make([]string, 0, len(caps))
	for _, c := range caps {
		names = append(names, string(c))
	}
	data, err := json.Marshal(struct {
		Name         string   `json:"name"`
		Version      string   `json:"version"`
		Capabilities []string `json:"capabilities"`
	}{Name: d.Name, Version: d.Version, Capabilities: names})
	if err != nil {
		return nil, fmt.Errorf("v14: 编码 runtime 能力快照: %w", err)
	}
	return data, nil
}

// v14Instruction 组装一次内层迭代的确定性指令:合同目标 + 验收条款;requires
// 非空时附编译权威的防御上下文(P1-05,PRD §8.3 Prevention:每轮迭代都被告知
// 最终验收条件;行内容只来自命中策略的公开声明,held-out 验证材料类型上不在
// 渲染输入里);修复尝试附 P2-04 安全投影(RepairFeedback 是失败材料的唯一
// 通道,类型上不存在 allowlist 之外的字段 —— 这里只渲染投影,绝不接触原始
// 验证输出);第 N>1 轮非修复迭代附注重试语义。指令只是输入,产出仍是提案。
func v14Instruction(task taskcontract.TaskContract, iteration int, fb *closedloop.RepairFeedback, requires []string) string {
	var b strings.Builder
	b.WriteString("Task " + task.ID)
	if task.Title != "" {
		b.WriteString(": " + task.Title)
	}
	b.WriteString("\n\nObjective:\n" + task.Objective + "\n")
	if len(task.Acceptance) > 0 {
		b.WriteString("\nAcceptance:\n")
		for _, a := range task.Acceptance {
			b.WriteString("- " + a.ID + ": " + a.Statement + "\n")
		}
	}
	if len(requires) > 0 {
		b.WriteString("\nPrevention requirements (compiled contract):\n")
		for _, line := range requires {
			b.WriteString("- " + line + "\n")
		}
	}
	if fb != nil {
		// 修复反馈:只含 allowlist 投影(acceptance id + 冻结语义文本 / gate 名、
		// 失败类、权威侧脱敏的诊断摘要、可见证据定位、尝试谱系)。held-out 的
		// 内容/定位/摘要在投影类型上不存在,本渲染无从泄漏(INV-V14-07)。
		b.WriteString("\nRepair context (verification feedback):\n")
		b.WriteString("- failed attempt: " + strconv.Itoa(int(fb.Attempt)) +
			" (run " + string(fb.RunID) + ")\n")
		if fb.Subject.AcceptanceID != "" {
			b.WriteString("- failed acceptance " + fb.Subject.AcceptanceID + ": " + fb.Subject.Statement + "\n")
		}
		if fb.Subject.Gate != "" {
			b.WriteString("- failed required gate: " + fb.Subject.Gate + "\n")
		}
		b.WriteString("- failure class: " + string(fb.Failure) + "\n")
		b.WriteString("- diagnostic summary: " + fb.DiagnosticSummary + "\n")
		for _, ev := range fb.Evidence {
			b.WriteString("- visible evidence " + ev.BindingID)
			if ev.Path != "" {
				b.WriteString(" at " + ev.Path)
			}
			if ev.Selector != "" {
				b.WriteString(" selector " + ev.Selector)
			}
			b.WriteString(" status " + string(ev.Status) + "\n")
		}
		b.WriteString("\nThis is repair attempt " + strconv.Itoa(iteration) +
			"; the previous candidate is already materialized in this workspace. " +
			"Re-work the change so the failed acceptance passes under the same frozen contract.\n")
	} else if iteration > 1 {
		b.WriteString("\nThis is retry iteration " + strconv.Itoa(iteration) +
			"; the previous iteration did not produce an acceptable candidate. Re-work the change in the same workspace.\n")
	}
	return b.String()
}

// ---- ContextPack(P1-05 compiled-aware)----

// v14ContextPack 组装 v14 路径的 ContextPack 并返回 compiled-aware
// context_hash(P1-05)。候选面与 v13ContextPack 同一规则:base 提交树里
// scope.allowed_paths 之下的 tracked 文件 + visible test binding 指定的路径
// (Select 自己裁决 forbidden/held-out/非测试代码并登记台账);差异只有一处 ——
// Requires 注入编译权威(contract.Bind)的执行前 requires 上下文(PRD §8.3
// Prevention):防御要求是冻结合同派生的必现小节,不参与候选预算,纳入
// context_hash。因此该哈希是编译权威内容的敏感函数 —— 同一任务在策略换版后
// 重跑/修复,即便候选面与合同内容都不变,上下文坐标也必然改变(修复/重放不能
// 悄悄换权威)。v13 路径不编译权威,沿用 v13ContextPack(零值 Requires,哈希
// 不变;零策略仓库的 v14 组装与 v13 逐字节一致,由测试钉住)。
func v14ContextPack(repo, base string, task taskcontract.TaskContract, f *verification.Frozen, requires contract.RequiresContext) (string, error) {
	tracked, err := gitTreeFiles(context.Background(), repo, base)
	if err != nil {
		return "", fmt.Errorf("v14: 列举 base %s 的 tracked 文件: %w", base, err)
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
		Requires:     requires,
	})
	if err != nil {
		return "", fmt.Errorf("v14: 组装 ContextPack: %w", err)
	}
	return res.Pack.Hash(), nil
}
