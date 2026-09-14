// Package workflow 实现 P1 单 Task 垂直工作流内核(P1-39,PRD §17 P0/P1
// vertical subset、§29 Debug 生命周期、§11.2.1 升级语义、§15 CodegenRun/ChangeSet)。
//
// 内核只做四件事,全部围绕**一个 Task 的顺序执行**:
//
//  1. Run 生命周期:一次 Task 执行 = 有限个顺序 CodegenRun;每个 Run 拥有一个
//     一次性 Git worktree(P1-35)与一个 Runtime 会话(P1-04 协议),从 running
//     迁移到唯一终态(completed/failed/escalated/cancelled),过程事实经
//     RunSink 落库(P1-33 适配)。
//  2. 有限 retry/escalation 钩子:迭代次数上界与升级阈值来自 ExecutionPolicy
//     (P1-03),内核只执行不定义;失败迭代达到阈值即升级边界 —— 有升级目标:
//     当前 Run `status: escalated` 终止、新 Run 记 escalated_from(§11.2.1);
//     无目标或目标非法:NEEDS_REVIEW(§29)。内核保证有限性:每 Run 迭代数有
//     上界、升级目标 profile 不得重复(重复视为无目标)、升级链长度有绝对上限。
//  3. 候选 ChangeSet 创建:到达候选固定点的 Run 经 FixCandidate 钩子固定
//     恰好 0..1 个不可变候选快照(P1-05);crash/预算耗尽/升级终止的 Run
//     不产生 ChangeSet(§11.2.1/§15.3)。
//  4. 停止条件:策略列出的 stop condition 一旦被观测,立即终止当前 Run
//     (无 ChangeSet)并把 Task 判为 NEEDS_REVIEW(§11.2/§29)。
//
// 刻意不存在的东西(AC4,§17「不得实现通用 workflow DSL」):Task DAG、
// 就绪选择器、跨 Task 依赖、并行 Task 调度、通用编排语言。一个
// TaskExecution 恰好驱动一个 Task,Execute 恰好可调用一次;任务状态由证据
// 推导,内核不写 TaskExecutionState(§18.1),只产出可观测事实与结果值。
//
// 本包不执行进程、不发起网络请求、不解析 shell 字符串;所有外部确定性判断
// (候选固定、停止观测、升级目标解析)都是调用方注入的显式钩子。
package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"tally/internal/agentruntime"
	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/measurement"
	"tally/internal/policy"
	"tally/internal/runstore"
	"tally/internal/taskcontract"
	"tally/internal/workspace"
)

// escalationCap 是一条升级链的绝对长度上限(§29「不得无限循环」的最后防线):
// 即便升级钩子不断产出新 profile 名,链长也不能超过此界。
const escalationCap = 16

// 哨兵错误:调用方用 errors.Is 判因,不做字符串匹配。
var (
	// ErrInvalidSpec 表示 TaskExecutionSpec / Hooks 不满足构造约束(缺必填、
	// 策略非法、钩子缺失等)。
	ErrInvalidSpec = errors.New("workflow: 非法的 TaskExecution 配置")
	// ErrAlreadyExecuted 表示对同一 TaskExecution 二次调用 Execute。一次执行
	// 对象只驱动一个 Task 的一次顺序执行,不可复用,也没有批量入口。
	ErrAlreadyExecuted = errors.New("workflow: TaskExecution 已执行,不可复用")
	// ErrUnregisteredStopCondition 表示停止观测钩子上报了当前 ExecutionPolicy
	// 未列出的条件:观测面与策略面不一致,按配置错误处理,绝不静默放行。
	ErrUnregisteredStopCondition = errors.New("workflow: 观测到策略未列出的停止条件")
)

// RunID 是内核分配的 CodegenRun 标识。对 workspace(P1-35 的 worktree 归属)
// 与 changeset(producer run 引用)透明;持久化主键由 RunSink 自行映射。
type RunID string

// TaskStatus 是一次 Task 顺序执行的任务级结果。它只是内核的返回值,不是
// TaskExecutionState 写入(§18.1:状态由证据推导,Agent/内核都不得直写)。
type TaskStatus string

const (
	// TaskCompleted 表示某个 Run 固定了候选 ChangeSet,执行成功收束。
	TaskCompleted TaskStatus = "completed"
	// TaskNeedsReview 表示重试/升级阶梯耗尽、升级不可用、Run 中途夭折或触发
	// 停止条件 —— 按 §29 进入 NEEDS_REVIEW 等待人裁决。
	TaskNeedsReview TaskStatus = "needs_review"
	// TaskCancelled 表示调用方取消了执行。
	TaskCancelled TaskStatus = "cancelled"
)

// NeedsReviewReason 说明 NEEDS_REVIEW 的成因(§29 阶梯落点)。
type NeedsReviewReason string

const (
	// ReviewEscalationUnavailable 到达升级边界但没有可用升级目标(无 target、
	// 目标非法,或目标 profile 已用过 —— 重复目标视为无目标,循环必须终止)。
	ReviewEscalationUnavailable NeedsReviewReason = "escalation_unavailable"
	// ReviewRunAborted Run 在升级边界前因 crash/预算耗尽/墙钟到期终止;策略
	// 未到升级点,内核不越权提前升级。
	ReviewRunAborted NeedsReviewReason = "run_aborted"
	// ReviewStopCondition 观测到策略停止条件,治理停止。
	ReviewStopCondition NeedsReviewReason = "stop_condition"
)

// TaskExecutionSpec 是一次 Task 顺序执行的全部输入。
type TaskExecutionSpec struct {
	// Project 项目边界。
	Project core.ProjectID
	// Task 被执行的合同(身份与指令来源);内核要求它已过 Validate。
	Task taskcontract.TaskContract
	// Role 是本执行的逻辑角色(PRD §9;如 "coding")。
	Role string
	// Policy 是执行边界的唯一权威(PRD §11.2),必须过 Validate。
	Policy policy.ExecutionPolicy
	// BaseRevision 所有 Run 的共同 base revision(每次执行从同一基线出发;
	// ChangeSet 不跨 Run 累积,§15.3)。
	BaseRevision string
	// 首个 Run 的归因坐标(§11.2.1:runtime/model/behavior_revision 是 Run 内
	// 单值事实;升级 Run 的对应值来自 EscalationTarget)。
	Profile          string
	Runtime          agentruntime.Runtime
	Model            string
	BehaviorRevision string
	// VerificationRevision / ContextPackHash 是开跑即写定的观测面身份坐标
	// (§15.2;内核不解释,只为 RunSink 透传)。
	VerificationRevision string
	ContextPackHash      string
	// Worktrees 提供 Run 的一次性源码 worktree(P1-35)。
	Worktrees *workspace.RunTable
	// Sink 接收 Run 生命周期与 ChangeSet 事实(P1-33 的进程内适配点)。
	Sink RunSink
	// Hooks 是确定性集成钩子;FixCandidate 必填,其余可空(语义见 Hooks)。
	Hooks Hooks
	// Now 返回事实时刻;nil 则用 time.Now。测试注入固定时钟。
	Now func() time.Time
}

// Hooks 是内核的确定性集成缝。内核不猜:候选是否固定、停止条件是否出现、
// 升级目标是谁,全部由钩子显式回答;内核只执行策略给出的有限预算。
type Hooks struct {
	// FixCandidate 在一个完成声明后尝试固定候选;ok=false 表示本轮没有可固定
	// 候选(该迭代按失败迭代计入 retry/escalation)。必填。
	FixCandidate CandidateFixFunc
	// StopProbe 每个迭代后观测停止条件;nil = 无观测。返回的条件若不在当前
	// ExecutionPolicy.StopConditions 内,按 ErrUnregisteredStopCondition 报错。
	StopProbe StopProbeFunc
	// ResolveEscalation 在升级边界解析升级目标;入参是本次执行已用的 profile
	// 链(按使用顺序)。nil 或 ok=false = 无升级目标(§29:NEEDS_REVIEW)。
	ResolveEscalation EscalationResolveFunc
}

// IterationView 是刚结束的一个迭代的只读视图,供钩子观测。
type IterationView struct {
	Run           RunID
	Profile       string
	WorkspacePath string
	BaseRevision  string
	// Iteration 是刚结束的迭代序号(1 起)。
	Iteration int
	// Claim 是 Runtime 对该迭代的自述(agentruntime.AgentRunResult:只是
	// claim,不是事实,INV-AUTH-01)。
	Claim agentruntime.AgentRunResult
}

// CandidateSnapshot 是钩子观测到的候选内容;内核补齐 project/base/producer/
// created_at 后经 changeset.New 固定(候选身份由内容推导,P1-05)。
type CandidateSnapshot struct {
	// CandidateRevision 固定的 candidate revision(如 worktree HEAD);
	// 与 Files 至少提供其一。
	CandidateRevision string
	Files             []changeset.ChangedFile
	Entities          []string
}

// CandidateFixFunc 固定一次候选;ok=false = 本迭代无可固定候选。
type CandidateFixFunc func(ctx context.Context, v IterationView) (CandidateSnapshot, bool, error)

// StopProbeFunc 观测一次停止条件集合(只上报观测,不裁决;裁决权在策略)。
type StopProbeFunc func(ctx context.Context, v IterationView) ([]policy.StopCondition, error)

// EscalationTarget 是升级 Run 的归因坐标(§11.2.1:升级即换 Run,单值事实
// 随 Run 更换,绝不在同一 Run 内替换 runtime/model)。
type EscalationTarget struct {
	Profile          string
	Runtime          agentruntime.Runtime
	Model            string
	BehaviorRevision string
}

// EscalationResolveFunc 解析升级目标;usedProfiles 是已用 profile 链。
// ok=false = 无可用目标。
type EscalationResolveFunc func(ctx context.Context, usedProfiles []string) (EscalationTarget, bool, error)

// runTarget 是待开 Run 的归因坐标(首个 Run 来自 spec,升级 Run 来自钩子)。
type runTarget struct {
	profile          string
	runtime          agentruntime.Runtime
	model            string
	behaviorRevision string
}

// RunRecord 是一个已终结 Run 的生命周期事实投影(结果值,§29 调试时间线)。
type RunRecord struct {
	ID            RunID
	EscalatedFrom RunID // 空 = 非升级 Run
	Profile       string
	Status        runstore.RunStatus // 终态(running 不会出现在结果里)
	Iterations    int
	FailureReason string
	// ChangeSetID 非空表示该 Run 固定了候选;升级/失败/取消的 Run 为空
	// (§11.2.1:没有固定 candidate 的 Run 不产生 ChangeSet)。
	ChangeSetID changeset.ChangeSetID
	// WorkspacePath 是该 Run 的一次性 worktree 路径(已随 Run 终止销毁)。
	WorkspacePath string

	// changeSet 仅供 Execute 回传 Outcome.ChangeSet,不经 API 暴露。
	changeSet *changeset.ChangeSet
}

// Outcome 是一次 Task 顺序执行的结果值:Run 时间线 + 任务级落点 + (成功时)
// 固定的候选 ChangeSet。
type Outcome struct {
	Status TaskStatus
	// Reason 仅 Status==TaskNeedsReview 时非空。
	Reason NeedsReviewReason
	// StopConditions 仅 Reason==ReviewStopCondition 时非空:触发停止的已观测条件。
	StopConditions []policy.StopCondition
	// Runs 按开跑顺序排列(顺序执行的时间线,AC1)。
	Runs []RunRecord
	// ChangeSet 仅 Status==TaskCompleted 时非 nil:本执行固定的唯一候选。
	ChangeSet *changeset.ChangeSet
}

// TaskExecution 驱动一个 Task 的顺序执行。一次性使用:Execute 恰好可调用
// 一次(AC4:没有批量/并发/跨 Task 入口)。
type TaskExecution struct {
	spec TaskExecutionSpec
	now  func() time.Time

	mu      sync.Mutex
	done    bool
	runSeq  int
	used    []string // 已用 profile 链(含首个)
	records []RunRecord
}

// NewTaskExecution 校验并构造一次 Task 执行。
func NewTaskExecution(spec TaskExecutionSpec) (*TaskExecution, error) {
	switch {
	case spec.Project == "":
		return nil, fmt.Errorf("%w: 缺少 project", ErrInvalidSpec)
	case strings.TrimSpace(spec.Task.ID) == "":
		return nil, fmt.Errorf("%w: 缺少 task", ErrInvalidSpec)
	case strings.TrimSpace(spec.Role) == "":
		return nil, fmt.Errorf("%w: 缺少 role", ErrInvalidSpec)
	case strings.TrimSpace(spec.BaseRevision) == "":
		return nil, fmt.Errorf("%w: 缺少 base revision", ErrInvalidSpec)
	case strings.TrimSpace(spec.Profile) == "":
		return nil, fmt.Errorf("%w: 缺少首个 Run 的 profile", ErrInvalidSpec)
	case spec.Runtime == nil:
		return nil, fmt.Errorf("%w: 缺少首个 Run 的 runtime", ErrInvalidSpec)
	case strings.TrimSpace(spec.Model) == "":
		return nil, fmt.Errorf("%w: 缺少首个 Run 的 model", ErrInvalidSpec)
	case strings.TrimSpace(spec.BehaviorRevision) == "":
		return nil, fmt.Errorf("%w: 缺少首个 Run 的 behavior revision", ErrInvalidSpec)
	case strings.TrimSpace(spec.VerificationRevision) == "":
		return nil, fmt.Errorf("%w: 缺少 verification revision", ErrInvalidSpec)
	case strings.TrimSpace(spec.ContextPackHash) == "":
		return nil, fmt.Errorf("%w: 缺少 context pack hash", ErrInvalidSpec)
	case spec.Worktrees == nil:
		return nil, fmt.Errorf("%w: 缺少 worktree 表", ErrInvalidSpec)
	case spec.Sink == nil:
		return nil, fmt.Errorf("%w: 缺少 RunSink", ErrInvalidSpec)
	case spec.Hooks.FixCandidate == nil:
		return nil, fmt.Errorf("%w: 缺少 FixCandidate 钩子", ErrInvalidSpec)
	}
	if err := spec.Policy.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSpec, err)
	}
	if err := spec.Task.Validate(); err != nil {
		return nil, fmt.Errorf("%w: task contract: %v", ErrInvalidSpec, err)
	}
	if err := spec.Runtime.Descriptor().Validate(); err != nil {
		return nil, fmt.Errorf("%w: runtime descriptor: %v", ErrInvalidSpec, err)
	}
	now := spec.Now
	if now == nil {
		now = time.Now
	}
	return &TaskExecution{spec: spec, now: now}, nil
}

// runEnd 是一个 Run 的终局分类(内核内部裁决词汇)。
type runEnd uint8

const (
	endCompleted    runEnd = iota // 固定了候选,任务成功收束
	endEscalated                  // 升级边界到达且有目标:Run 记 escalated,开新 Run
	endNoEscalation               // 升级边界到达但无目标:Run 记 failed,NEEDS_REVIEW
	endAborted                    // 边界前夭折(crash/预算/墙钟):Run 记 failed,NEEDS_REVIEW
	endStopped                    // 观测到策略停止条件:Run 记 failed,NEEDS_REVIEW
	endCancelled                  // 调用方取消:Run 记 cancelled
)

// runResult 是 runOne 的内部结果:记录 + 终局 + (升级时)下一个目标。
type runResult struct {
	rec   RunRecord
	end   runEnd
	stops []policy.StopCondition
	next  runTarget // 仅 end==endEscalated 时有效
}

// Execute 顺序执行该 Task 的 Run 链直至任务级终态(AC1)。返回 error 仅表示
// 配置/基础设施故障(钩子出错、sink 写入失败、worktree 回收失败等);领域内
// 的阶梯耗尽/停止/取消都是正常 Outcome(§29)。
func (e *TaskExecution) Execute(ctx context.Context) (Outcome, error) {
	e.mu.Lock()
	if e.done {
		e.mu.Unlock()
		return Outcome{}, ErrAlreadyExecuted
	}
	e.done = true
	e.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return Outcome{Status: TaskCancelled}, nil
	}

	target := runTarget{
		profile:          e.spec.Profile,
		runtime:          e.spec.Runtime,
		model:            e.spec.Model,
		behaviorRevision: e.spec.BehaviorRevision,
	}
	e.used = []string{target.profile}
	var escalatedFrom RunID

	for {
		res, err := e.runOne(ctx, target, escalatedFrom)
		if res.rec.ID != "" {
			e.records = append(e.records, res.rec)
		}
		if err != nil {
			return Outcome{}, err
		}
		switch res.end {
		case endCompleted:
			return Outcome{
				Status:    TaskCompleted,
				Runs:      e.records,
				ChangeSet: res.rec.changeSet,
			}, nil
		case endCancelled:
			return Outcome{Status: TaskCancelled, Runs: e.records}, nil
		case endStopped:
			return Outcome{
				Status:         TaskNeedsReview,
				Reason:         ReviewStopCondition,
				StopConditions: res.stops,
				Runs:           e.records,
			}, nil
		case endEscalated:
			// §11.2.1:升级终止当前 Run;新 Run 记 escalated_from。
			escalatedFrom = res.rec.ID
			target = res.next
		case endAborted:
			return Outcome{Status: TaskNeedsReview, Reason: ReviewRunAborted, Runs: e.records}, nil
		case endNoEscalation:
			return Outcome{Status: TaskNeedsReview, Reason: ReviewEscalationUnavailable, Runs: e.records}, nil
		}
	}
}

// resolveEscalation 在升级边界解析下一个升级目标。目标非法(缺 profile/runtime、
// model/behavior 为空)或 profile 已用过,一律按「无目标」处理(§29:无目标 →
// NEEDS_REVIEW;重复目标不得形成循环)。链长超上限同样无目标。
func (e *TaskExecution) resolveEscalation(ctx context.Context) (runTarget, bool, error) {
	if len(e.used) >= escalationCap {
		return runTarget{}, false, nil
	}
	if e.spec.Hooks.ResolveEscalation == nil {
		return runTarget{}, false, nil
	}
	tgt, ok, err := e.spec.Hooks.ResolveEscalation(ctx, slices.Clone(e.used))
	if err != nil {
		return runTarget{}, false, fmt.Errorf("workflow: 解析升级目标: %w", err)
	}
	if !ok {
		return runTarget{}, false, nil
	}
	switch {
	case strings.TrimSpace(tgt.Profile) == "":
		return runTarget{}, false, nil
	case slices.Contains(e.used, tgt.Profile):
		return runTarget{}, false, nil
	case tgt.Runtime == nil:
		return runTarget{}, false, nil
	case strings.TrimSpace(tgt.Model) == "":
		return runTarget{}, false, nil
	case strings.TrimSpace(tgt.BehaviorRevision) == "":
		return runTarget{}, false, nil
	}
	e.used = append(e.used, tgt.Profile)
	return runTarget{
		profile:          tgt.Profile,
		runtime:          tgt.Runtime,
		model:            tgt.Model,
		behaviorRevision: tgt.BehaviorRevision,
	}, true, nil
}

// runOne 开一个 Run(一次性 worktree + Runtime 会话 + sink 开跑记录),顺序
// 执行有限个迭代,迁移到唯一终态并回收资源。事实写入用不继承调用方取消的
// 派生 context:取消是执行终态,不是事实丢失理由。
func (e *TaskExecution) runOne(ctx context.Context, target runTarget, escalatedFrom RunID) (runResult, error) {
	e.runSeq++
	id := RunID(fmt.Sprintf("run-%04d", e.runSeq))

	// 一次性源码 worktree:一个 Run 恰好拥有一个(P1-35/INV-EXEC-01)。
	sources, err := e.spec.Worktrees.Open(ctx, workspace.RunID(id), e.spec.BaseRevision)
	if err != nil {
		return runResult{}, fmt.Errorf("workflow: %s 打开源 worktree: %w", id, err)
	}

	// Runtime 会话绑定该 worktree;可执行变更绝不在 owner 主 worktree 发生。
	desc := target.runtime.Descriptor()
	// 期望 model 档位随会话请求下发(P0-01 AC1),但只在 runtime 声明
	// model.select 时携带 —— 控制流按能力协商结果分支,不按 runtime 名字
	// 分支(PRD §10.3)。未声明的 runtime 以自身默认模型启动;期望值仍如实
	// 记录为 RunBegin.Model,与实际观察分开(Decision Lock 15),不互相冒充。
	startReq := agentruntime.StartRequest{Workspace: sources.Path()}
	if desc.Capabilities.Has(agentruntime.CapModelSelect) {
		startReq.ModelProfile = target.model
	}
	session, err := target.runtime.Start(ctx, startReq)
	if err != nil {
		if termErr := e.teardown(ctx, id, session); termErr != nil {
			return runResult{}, termErr
		}
		return runResult{}, fmt.Errorf("workflow: %s 开启 runtime 会话: %w", id, err)
	}

	// 实际身份是可选观察(P0-01 AC3):会话未实现 IdentityReporter = 完全
	// 未观察,零值降级,绝不用期望值补齐(Decision Lock 15:actual != desired)。
	var observed agentruntime.ObservedIdentity
	if rep, ok := session.(agentruntime.IdentityReporter); ok {
		observed = rep.Identity()
	}

	snapshot, err := capabilitySnapshot(desc)
	if err != nil {
		if termErr := e.teardown(ctx, id, session); termErr != nil {
			return runResult{}, termErr
		}
		return runResult{}, err
	}

	factCtx := context.WithoutCancel(ctx)
	runRef, err := e.spec.Sink.BeginRun(factCtx, RunBegin{
		ID:                   id,
		EscalatedFrom:        escalatedFrom,
		Project:              e.spec.Project,
		TaskID:               e.spec.Task.ID,
		TaskRevision:         e.spec.Task.Hash,
		Role:                 e.spec.Role,
		Profile:              target.profile,
		RuntimeName:          desc.Name,
		RuntimeVersion:       desc.Version,
		Model:                target.model,
		BehaviorRevision:     target.behaviorRevision,
		ObservedIdentity:     observed,
		BaseRevision:         e.spec.BaseRevision,
		WorkspaceID:          sources.Path(),
		CapabilitySnapshot:   snapshot,
		StartedAt:            e.now(),
		VerificationRevision: e.spec.VerificationRevision,
		ContextPackHash:      e.spec.ContextPackHash,
	})
	if err != nil {
		if termErr := e.teardown(ctx, id, session); termErr != nil {
			return runResult{}, termErr
		}
		return runResult{}, fmt.Errorf("workflow: %s 记录开跑: %w", id, err)
	}
	if strings.TrimSpace(runRef) == "" {
		if termErr := e.teardown(ctx, id, session); termErr != nil {
			return runResult{}, termErr
		}
		return runResult{}, fmt.Errorf("workflow: %s 的 sink 未返回持久化 Run 引用", id)
	}

	// 墙钟预算:每个 Run 一个 deadline(§11.2 budgets.max_wall_time)。
	runCtx, cancelWall := context.WithTimeout(ctx, e.spec.Policy.Budgets.MaxWallTime)
	defer cancelWall()

	rec := RunRecord{ID: id, EscalatedFrom: escalatedFrom, Profile: target.profile, WorkspacePath: sources.Path()}
	var (
		iterations   int
		failedIters  int
		inTokens     int64
		outTokens    int64
		lastClaim    agentruntime.AgentRunResult
		changeSet    *changeset.ChangeSet
		observedStop []policy.StopCondition
		nextTarget   runTarget
	)

	// release 幂等回收 Run 资源;任何错误退出路径都必须先经过它,
	// 一次性 worktree 绝不泄漏。
	var torn bool
	release := func() error {
		if torn {
			return nil
		}
		torn = true
		return e.teardown(ctx, id, session)
	}
	// fail 在错误退出前回收资源;回收失败优先上报(现场未清理是更严重的错)。
	fail := func(err error) (runResult, error) {
		if termErr := release(); termErr != nil {
			return runResult{}, termErr
		}
		return runResult{}, err
	}

	// endRun 落终态事实、回收资源,返回终局结果。changeSet 只在 completed 路径
	// 之前被赋值;failed/escalated/cancelled 路径保持 nil(§11.2.1)。
	endRun := func(status runstore.RunStatus, kind runEnd, why string) (runResult, error) {
		if err := e.spec.Sink.FinishRun(factCtx, RunFinish{
			ID:              id,
			Status:          status,
			Iterations:      int32(iterations),
			InputTokens:     &inTokens,
			OutputTokens:    &outTokens,
			ResultPatchHash: changeSetHash(changeSet),
			AgentClaim:      lastClaim.Summary,
			FailureReason:   why,
			FinishedAt:      e.now(),
		}); err != nil {
			return fail(fmt.Errorf("workflow: %s 记录终态: %w", id, err))
		}
		rec.Status = status
		rec.Iterations = iterations
		rec.FailureReason = why
		rec.changeSet = changeSet
		if changeSet != nil {
			rec.ChangeSetID = changeSet.ID()
		}
		if err := release(); err != nil {
			return runResult{}, err
		}
		return runResult{rec: rec, end: kind, stops: observedStop, next: nextTarget}, nil
	}

	// escalateEnd 在失败迭代达到阈值(升级边界)时裁决并终结 Run(§11.2/§29):
	// 有目标 → escalated + 新目标;无目标 → failed + NEEDS_REVIEW。
	// 未达阈值时返回 done=false,由调用方决定继续重试还是按夭折终止。
	escalateEnd := func() (runResult, bool, error) {
		if failedIters < e.spec.Policy.Escalation.AfterFailedIterations {
			return runResult{}, false, nil
		}
		next, ok, err := e.resolveEscalation(ctx)
		if err != nil {
			r, ferr := fail(err)
			return r, false, ferr
		}
		if ok {
			nextTarget = next
			res, err := endRun(runstore.RunEscalated, endEscalated, "escalated_to:"+next.profile)
			return res, true, err
		}
		res, err := endRun(runstore.RunFailed, endNoEscalation, "escalation_unavailable")
		return res, true, err
	}

	// abortEnd 处理 Run 的必然终止(crash/预算/墙钟):仍先过升级边界 ——
	// 例如 crash 迭代恰好跨过阈值时,按 §11.2.1 记 escalated 并升级;
	// 未达阈值则记 aborted(内核不越权提前升级)。
	abortEnd := func(why string) (runResult, error) {
		if res, done, err := escalateEnd(); err != nil || done {
			return res, err
		}
		return endRun(runstore.RunFailed, endAborted, why)
	}

	for iterations < e.spec.Policy.Retry.MaxIterations {
		iterations++
		res, err := session.Execute(runCtx, agentruntime.ExecuteRequest{
			Instruction: instructionFor(e.spec.Task, iterations),
		})
		lastClaim = res
		inTokens += res.Usage.InputTokens
		outTokens += res.Usage.OutputTokens

		if err := e.spec.Sink.RecordProgress(factCtx, id, RunProgress{
			Iterations:   int32(iterations),
			InputTokens:  &inTokens,
			OutputTokens: &outTokens,
		}); err != nil {
			return fail(fmt.Errorf("workflow: %s 记录进度: %w", id, err))
		}

		// 1) 调用方取消优先于一切(终止,不裁决升级)。
		if ctx.Err() != nil {
			return endRun(runstore.RunCancelled, endCancelled, "cancelled")
		}

		// 2) 治理停止条件:观测面与策略面不一致是配置错误,绝不静默。
		if e.spec.Hooks.StopProbe != nil {
			observed, err := e.spec.Hooks.StopProbe(ctx, iterationView(id, target, sources, e.spec.BaseRevision, iterations, res))
			if err != nil {
				return fail(fmt.Errorf("workflow: %s 停止观测: %w", id, err))
			}
			for _, c := range observed {
				if !slices.Contains(e.spec.Policy.StopConditions, c) {
					return fail(fmt.Errorf("%w: %q 不在当前 ExecutionPolicy 中",
						ErrUnregisteredStopCondition, c))
				}
			}
			if len(observed) > 0 {
				observedStop = observed
				names := make([]string, len(observed))
				for i, c := range observed {
					names[i] = string(c)
				}
				return endRun(runstore.RunFailed, endStopped, "stop_condition:"+strings.Join(names, ","))
			}
		}

		// 3) 预算耗尽(墙钟/令牌):Run 必然终止且不产生 ChangeSet(§11.2.1)。
		if runCtx.Err() != nil {
			return abortEnd("wall_time_budget_exhausted")
		}
		if inTokens+outTokens > e.spec.Policy.Budgets.MaxTokens {
			return abortEnd("token_budget_exhausted")
		}

		// 4) Runtime 会话级故障(如 crash):本轮按失败迭代计;会话已不可信,
		// Run 终止,不得在同一 Run 内重试(§29)。
		if err != nil {
			failedIters++
			return abortEnd("runtime_error")
		}

		// 5) Runtime 自述失败/被取消:失败迭代,进入 retry/escalation 计数。
		if res.Status != agentruntime.RunCompleted {
			failedIters++
			if r, done, err := escalateEnd(); err != nil || done {
				return r, err
			}
			continue
		}

		// 6) 完成声明 → 候选固定点:恰好 0..1 个不可变 ChangeSet(AC2)。
		snap, ok, err := e.spec.Hooks.FixCandidate(ctx, iterationView(id, target, sources, e.spec.BaseRevision, iterations, res))
		if err != nil {
			return fail(fmt.Errorf("workflow: %s 固定候选: %w", id, err))
		}
		if !ok {
			// 完成声明但无可固定候选 = 非产出迭代,计入失败迭代。
			failedIters++
			if r, done, err := escalateEnd(); err != nil || done {
				return r, err
			}
			continue
		}
		cs, err := changeset.New(changeset.Spec{
			Project:           e.spec.Project,
			BaseRevision:      e.spec.BaseRevision,
			CandidateRevision: snap.CandidateRevision,
			Files:             snap.Files,
			Entities:          snap.Entities,
			ProducerRun:       runRef, // 持久化 Run 引用:快照回指存储层的 Run 行
			CreatedAt:         e.now(),
		})
		if err != nil {
			return fail(fmt.Errorf("workflow: %s 构造 ChangeSet: %w", id, err))
		}
		if err := e.spec.Sink.RecordChangeSet(factCtx, id, cs); err != nil {
			return fail(fmt.Errorf("workflow: %s 记录 ChangeSet: %w", id, err))
		}
		changeSet = cs
		return endRun(runstore.RunCompleted, endCompleted, "")
	}

	// 迭代预算耗尽。每次未收束迭代都已计入 failedIters,且
	// AfterFailedIterations <= MaxIterations(policy.Validate 保证),所以正常
	// 不会走到这里;万一到达,按升级边界/夭折收束,绝不无界循环。
	return abortEnd("iterations_exhausted")
}

// teardown 收尾一个 Run:先关会话(尽力而为),再销毁一次性 worktree
// (P1-35:Terminate 幂等、不继承调用方取消;失败保留现场可重试)。
func (e *TaskExecution) teardown(ctx context.Context, id RunID, session agentruntime.Session) error {
	if session != nil {
		_ = session.Close(ctx) // 会话关闭是资源回收,失败不遮蔽后续 teardown
	}
	return e.spec.Worktrees.Terminate(ctx, workspace.RunID(id))
}

func iterationView(id RunID, target runTarget, sources *workspace.RunSources, baseRevision string, iteration int, claim agentruntime.AgentRunResult) IterationView {
	return IterationView{
		Run:           id,
		Profile:       target.profile,
		WorkspacePath: sources.Path(),
		BaseRevision:  baseRevision,
		Iteration:     iteration,
		Claim:         claim,
	}
}

// instructionFor 组装一个迭代的确定性指令:合同目标 + 验收条款;第 N>1 轮
// 附注重试语义(上一轮未产出可接受候选)。指令只是输入,产出仍是提案。
func instructionFor(c taskcontract.TaskContract, iteration int) string {
	var b strings.Builder
	b.WriteString("Task " + c.ID)
	if c.Title != "" {
		b.WriteString(": " + c.Title)
	}
	b.WriteString("\n\nObjective:\n" + c.Objective + "\n")
	if len(c.Acceptance) > 0 {
		b.WriteString("\nAcceptance:\n")
		for _, a := range c.Acceptance {
			b.WriteString("- " + a.ID + ": " + a.Statement + "\n")
		}
	}
	if iteration > 1 {
		b.WriteString("\nThis is retry iteration " + strconv.Itoa(iteration) +
			"; the previous iteration did not produce an acceptable candidate. Re-work the change in the same workspace.\n")
	}
	return b.String()
}

func changeSetHash(cs *changeset.ChangeSet) string {
	if cs == nil {
		return ""
	}
	return string(cs.ID())
}

// capabilitySnapshot 把 runtime 自述编码成非空 JSON 对象(观测面要求,
// P1-33 StartRunSpec.RuntimeCapabilitySnapshot)。
func capabilitySnapshot(d agentruntime.RuntimeDescriptor) (json.RawMessage, error) {
	caps := d.Capabilities.Sorted()
	names := make([]string, len(caps))
	for i, c := range caps {
		names[i] = string(c)
	}
	data, err := json.Marshal(struct {
		Name         string   `json:"name"`
		Version      string   `json:"version"`
		Capabilities []string `json:"capabilities"`
	}{Name: d.Name, Version: d.Version, Capabilities: names})
	if err != nil {
		return nil, fmt.Errorf("workflow: 编码 runtime 能力快照: %w", err)
	}
	return data, nil
}

// ---- Run 事实出口(内核与持久化之间唯一的编码边界) ----

// RunBegin 是开跑事实(P1-33 StartRunSpec 的内核侧投影)。
type RunBegin struct {
	ID               RunID
	EscalatedFrom    RunID // 空 = 非升级 Run
	Project          core.ProjectID
	TaskID           string
	TaskRevision     string
	Role             string
	Profile          string
	RuntimeName      string
	RuntimeVersion   string
	Model            string
	BehaviorRevision string
	// ObservedIdentity 是 runtime 自报的会话实际身份(可选观察,零值 = 未
	// 观察);与期望的 Model/RuntimeName/RuntimeVersion 分开记录,互不回填
	//(Decision Lock 15:实际身份 != 期望配置)。
	ObservedIdentity agentruntime.ObservedIdentity
	BaseRevision     string
	WorkspaceID      string
	// CapabilitySnapshot 是 runtime 声明能力快照(非空 JSON 对象)。
	CapabilitySnapshot json.RawMessage
	StartedAt          time.Time
	// VerificationRevision / ContextPackHash 是观测面身份坐标,内核透传。
	VerificationRevision string
	ContextPackHash      string
}

// RunProgress 是一次运行中进度上报(run-control 语义,仅对 running 生效)。
type RunProgress struct {
	Iterations   int32
	InputTokens  *int64
	OutputTokens *int64
}

// RunFinish 是终态迁移事实(一次性,不可回退)。
type RunFinish struct {
	ID               RunID
	Status           runstore.RunStatus // 必须是终态
	Iterations       int32
	InputTokens      *int64
	OutputTokens     *int64
	ResultPatchHash  string // 仅固定了候选的 Run 非空
	AgentClaim       string // Runtime 自述原样保留(claim,非事实)
	ValidationResult string
	FailureReason    string
	FinishedAt       time.Time
}

// RunSink 接收一次执行产出的 Run 生命周期与 ChangeSet 事实。它是内核的
// 可替换持久化缝(P1-33 的进程内适配点):生产用 RunStoreSink,测试用内存
// 实现。实现必须保持事实面 append-only:不存在更新/删除语义。
type RunSink interface {
	// BeginRun 落一条开跑事实,并返回该 Run 的**持久化引用**(事实面主键的
	// 字符串形态;RunStoreSink 返回 codegen_run 数字主键的十进制串)。
	// 候选 ChangeSet 的 producer run 引用必须使用它,保证快照能回指
	// 存储层的 Run 行(P1-33 合同);返回空串按 sink 违约处理。
	BeginRun(ctx context.Context, b RunBegin) (string, error)
	RecordProgress(ctx context.Context, id RunID, p RunProgress) error
	FinishRun(ctx context.Context, f RunFinish) error
	RecordChangeSet(ctx context.Context, id RunID, cs *changeset.ChangeSet) error
}

// RunStoreSink 用 P1-33 持久化适配器(*runstore.Store)实现 RunSink。
// 它在一次 Task 执行内维护内核 RunID → 存储主键的映射,因此一个实例只服务
// 一次 TaskExecution(顺序使用;内部有锁,但语义上不是并发入口)。
type RunStoreSink struct {
	runs *runstore.Store

	mu  sync.Mutex
	ids map[RunID]int64
}

// NewRunStoreSink 构造适配器。
func NewRunStoreSink(runs *runstore.Store) *RunStoreSink {
	return &RunStoreSink{runs: runs, ids: make(map[RunID]int64)}
}

// BeginRun 落一条 codegen_run 过程记录(status 恒以 running 起步),返回
// 数字主键的十进制串作为持久化 Run 引用。
func (s *RunStoreSink) BeginRun(ctx context.Context, b RunBegin) (string, error) {
	var escalated int64
	if b.EscalatedFrom != "" {
		s.mu.Lock()
		prev, ok := s.ids[b.EscalatedFrom]
		s.mu.Unlock()
		if !ok {
			return "", fmt.Errorf("workflow: 升级来源 Run %s 未曾开跑", b.EscalatedFrom)
		}
		escalated = prev
	}
	id, err := s.runs.StartCodegenRun(ctx, runstore.StartRunSpec{
		Project:                   b.Project,
		Task:                      b.TaskID,
		Role:                      b.Role,
		Runtime:                   b.RuntimeName,
		RuntimeVersion:            b.RuntimeVersion,
		Model:                     b.Model,
		BehaviorRevision:          b.BehaviorRevision,
		CostPlane:                 measurement.CostPlaneExecution,
		BaseCommit:                b.BaseRevision,
		TaskRevision:              b.TaskRevision,
		VerificationRevision:      b.VerificationRevision,
		ContextPackHash:           b.ContextPackHash,
		WorkspaceID:               b.WorkspaceID,
		AgentSessionID:            string(b.ObservedIdentity.SessionID),
		RuntimeCapabilitySnapshot: b.CapabilitySnapshot,
		EscalatedFrom:             escalated,
		StartedAt:                 b.StartedAt,
	})
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	s.ids[b.ID] = id
	s.mu.Unlock()
	return strconv.FormatInt(id, 10), nil
}

// RecordProgress 上报迭代进度与用量(run-control 迁移)。
func (s *RunStoreSink) RecordProgress(ctx context.Context, id RunID, p RunProgress) error {
	dbID, ok := s.lookup(id)
	if !ok {
		return fmt.Errorf("workflow: Run %s 未曾开跑", id)
	}
	ok, err := s.runs.UpdateRunProgress(ctx, runstore.RunProgress{
		ID:           dbID,
		Iterations:   p.Iterations,
		InputTokens:  p.InputTokens,
		OutputTokens: p.OutputTokens,
	})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("workflow: Run %s 进度上报落空(已终态?)", id)
	}
	return nil
}

// FinishRun 执行一次性终态迁移。
func (s *RunStoreSink) FinishRun(ctx context.Context, f RunFinish) error {
	dbID, ok := s.lookup(f.ID)
	if !ok {
		return fmt.Errorf("workflow: Run %s 未曾开跑", f.ID)
	}
	ok, err := s.runs.FinishCodegenRun(ctx, runstore.FinishRunSpec{
		ID:               dbID,
		Status:           f.Status,
		Iterations:       f.Iterations,
		InputTokens:      f.InputTokens,
		OutputTokens:     f.OutputTokens,
		ResultPatchHash:  f.ResultPatchHash,
		AgentClaim:       f.AgentClaim,
		ValidationResult: f.ValidationResult,
		FailureReason:    f.FailureReason,
		FinishedAt:       f.FinishedAt,
	})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("workflow: Run %s 终态迁移落空(已终态?)", f.ID)
	}
	return nil
}

// RecordChangeSet 以 insert-once 语义固定候选快照(不可变,INV-CHANGE-02)。
func (s *RunStoreSink) RecordChangeSet(ctx context.Context, id RunID, cs *changeset.ChangeSet) error {
	return s.runs.SaveChangeSet(ctx, cs)
}

func (s *RunStoreSink) lookup(id RunID) (int64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.ids[id]
	return v, ok
}
