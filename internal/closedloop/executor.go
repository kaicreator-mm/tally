// executor.go 实现 P2-08 的 Executor 生产适配器(ports.go 的 Executor 端口):
// 把 v1.3 codegen 内循环作为不透明 CodegenRunner 注入,本适配器只负责
// ADR-0001 §2 的工作区纪律 ——
//
//   - 每次外层尝试(RunIDFor(task, attempt) 推导的幂等 Run 身份)在
//     internal/workspace 的 RunTable 下打开**全新**一次性 worktree(owner
//     主 worktree 永远不是执行沙箱,INV-EXEC-01;RunTable 语义:一个 Run
//     恰好一个一次性 worktree,Run 身份按尝试推导 ⇒ 每次尝试必然新沙箱);
//   - 修复尝试(Attempt ≥ 2)先以 RepairBase 的 base revision 为基线检出,
//     再把冻结前候选内容经 RunSources.Materialize **确定性物化**为修复起点
//     (P2-05;前候选与其工作区绝不被改写 —— 物化只写全新 worktree);
//   - 尝试结束(无论成败/取消)一律丢弃一次性 worktree;清理不继承调用方
//     取消(与 internal/candidateverify / workspace.Manager 同纪律);
//   - 本适配器**不做** infra/provider 重试:执行侧的进程/提供方失败重试是
//     v1.3 内循环语义(ADR-0001 §1:inner retry ≠ outer Repair,
//     ExecutionPolicy.Retry 归 workflow),适配器错误原样上抛,由 Engine
//     fail-safe 升级 —— 绝不伪装成候选失败,也绝不消耗外层修复预算。
package closedloop

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"tally/internal/changeset"
	"tally/internal/workspace"
)

// ErrInvalidExecutor:执行适配器的请求或构造输入不合法(task id / run id /
// base revision 缺失,依赖为 nil)。errors.Is 判因,不做字符串匹配。
var ErrInvalidExecutor = errors.New("closedloop: 执行适配器输入不合法")

// CodegenRunRequest 是一次内层 codegen 执行的全部输入(v1.3 内循环的入口
// 载荷;内层迭代预算与重试语义完全归 runner 自持,对外层不可见,AC3)。
type CodegenRunRequest struct {
	// TaskID 是闭环任务 id。
	TaskID string
	// RunID 是本次尝试的 CodegenRun 身份(RunIDFor 推导的幂等键;
	// ADR-0001 §2:适配器语义要求 runner 以它作为该次 CodegenRun 的身份)。
	RunID RunID
	// ParentRun 是紧邻前一次尝试的 Run 身份(首次尝试为空);runner 据此
	// 落 Run 链父链接事实。
	ParentRun RunID
	// WorkspacePath 是本次尝试的**一次性** codegen worktree 绝对路径
	// (WorkspaceExecutor 已按 base revision 检出,修复尝试已物化修复起点;
	// runner 只在该目录内执行,owner 主 worktree 永不是沙箱)。
	WorkspacePath string
	// BaseRevision 是工作区检出所用的精确 base commit。
	BaseRevision string
	// RepairBase 是修复尝试(≥2)继续的冻结前候选(仅候选内容;首次尝试
	// 恒为 nil)。runner 不得改写它,也不得把它当作验收/保证材料。
	RepairBase *changeset.ChangeSet
	// Feedback 是修复尝试交给实现权威的唯一失败材料通道(P2-04 安全投影;
	// 首次尝试恒为 nil)。runner 只能把投影内容放进编码上下文,没有任何
	// 第二条失败材料来路。
	Feedback *RepairFeedback
}

// CodegenRunner 是内层 codegen 执行的消费侧窄缝(v1.3 workflow.TaskExecution
// 或等价服务经小适配实现;P2-09 组合根接线)。生产 runner 的实现契约:
//
//   - 每次调用 = 一次全新 CodegenRun(至多固化一个候选);
//   - 返回的候选是冻结的不可变快照,nil 表示本次尝试没有可固定候选
//     (半途工作已随一次性 worktree 丢弃);
//   - error 只表示进程/基础设施/提供方意义上的失败 —— 候选质量结论是
//     验证权威的事,runner 绝不伪造 verdict(与端口契约同纪律)。
type CodegenRunner interface {
	RunCodegen(ctx context.Context, req CodegenRunRequest) (*changeset.ChangeSet, error)
}

// BaseRevisionFunc 解析首次尝试的工作区基线 revision(任务在组合根已知的
// base commit / branch);修复尝试的基线改由冻结前候选携带,不经此解析。
type BaseRevisionFunc func(ctx context.Context, taskID string) (string, error)

// WorkspaceExecutorConfig 是 NewWorkspaceExecutor 的依赖面,全部必填。
type WorkspaceExecutorConfig struct {
	// Worktrees 是一次性 codegen worktree 表(共享一个 workspace.Manager)。
	Worktrees *workspace.RunTable
	// Runner 是内层 codegen 执行缝。
	Runner CodegenRunner
	// BaseRevision 解析首次尝试的工作区基线。
	BaseRevision BaseRevisionFunc
}

// WorkspaceExecutor 是 Executor 端口的生产适配:工作区纪律 + 不透明内循环。
type WorkspaceExecutor struct {
	table  *workspace.RunTable
	runner CodegenRunner
	base   BaseRevisionFunc
}

// Executor 端口编译期钉住。
var _ Executor = (*WorkspaceExecutor)(nil)

// NewWorkspaceExecutor 构造执行适配器;依赖缺失时拒绝构造(ErrInvalidExecutor)。
func NewWorkspaceExecutor(cfg WorkspaceExecutorConfig) (*WorkspaceExecutor, error) {
	switch {
	case cfg.Worktrees == nil:
		return nil, fmt.Errorf("%w: 一次性 worktree 表为 nil", ErrInvalidExecutor)
	case cfg.Runner == nil:
		return nil, fmt.Errorf("%w: 内层 codegen 执行缝为 nil", ErrInvalidExecutor)
	case cfg.BaseRevision == nil:
		return nil, fmt.Errorf("%w: 首次尝试基线解析为 nil", ErrInvalidExecutor)
	}
	return &WorkspaceExecutor{table: cfg.Worktrees, runner: cfg.Runner, base: cfg.BaseRevision}, nil
}

// ExecuteCandidate 实现 Executor 端口:打开本次尝试的一次性 worktree →
// (修复尝试)物化冻结前候选 → 调用不透明内循环 → 交付候选。
// error 原样上抛(Engine fail-safe 升级;取消优先由 Engine 在调用前后裁决)。
func (x *WorkspaceExecutor) ExecuteCandidate(ctx context.Context, req ExecuteRequest) (ExecuteOutcome, error) {
	switch {
	case strings.TrimSpace(req.TaskID) == "":
		return ExecuteOutcome{}, fmt.Errorf("%w: task id 为空", ErrInvalidExecutor)
	case strings.TrimSpace(string(req.RunID)) == "":
		return ExecuteOutcome{}, fmt.Errorf("%w: run id 为空", ErrInvalidExecutor)
	}

	// 继续基线(ADR-0001 §2):修复尝试用冻结前候选的 base revision;
	// 首次尝试由组合根解析。
	base := ""
	if req.RepairBase != nil {
		base = req.RepairBase.BaseRevision()
	} else {
		resolved, err := x.base(ctx, req.TaskID)
		if err != nil {
			return ExecuteOutcome{}, fmt.Errorf("closedloop: 解析首次尝试基线: %w", err)
		}
		base = resolved
	}
	if strings.TrimSpace(base) == "" {
		return ExecuteOutcome{}, fmt.Errorf("%w: base revision 为空", ErrInvalidExecutor)
	}

	rs, err := x.table.Open(ctx, workspace.RunID(req.RunID), base)
	if err != nil {
		return ExecuteOutcome{}, fmt.Errorf("closedloop: 打开一次性 codegen workspace(run=%s): %w", req.RunID, err)
	}
	// 尝试结束一律丢弃一次性 worktree(成功:候选已是冻结快照;失败/取消:
	// 半途工作不遗留)。清理不继承调用方取消;幂等;失败保留现场以便重试,
	// 只告警不改写尝试结论。
	defer func() {
		if err := rs.Terminate(context.WithoutCancel(ctx)); err != nil {
			slog.Warn("closedloop: 清理一次性 codegen workspace 失败", "run", string(req.RunID), "err", err)
		}
	}()

	if req.RepairBase != nil {
		if err := materializeRepairBase(rs, req.RepairBase); err != nil {
			return ExecuteOutcome{}, err
		}
	}

	cand, err := x.runner.RunCodegen(ctx, CodegenRunRequest{
		TaskID:        req.TaskID,
		RunID:         req.RunID,
		ParentRun:     req.ParentRun,
		WorkspacePath: rs.Path(),
		BaseRevision:  rs.BaseCommit(),
		RepairBase:    req.RepairBase,
		Feedback:      req.Feedback,
	})
	if err != nil {
		return ExecuteOutcome{}, err
	}
	return ExecuteOutcome{Candidate: cand}, nil
}

// materializeRepairBase 把冻结前候选内容确定性物化为修复起点(P2-05,
// ADR-0001 §2):候选改动文件逐字映射为 CandidateFile,排序/校验/整集
// validate-then-write 归 workspace.RunSources.Materialize(内容原样,无
// patch 引擎;越界路径被拒,绝不写出 worktree)。
func materializeRepairBase(rs *workspace.RunSources, base *changeset.ChangeSet) error {
	files := base.Files()
	cfs := make([]workspace.CandidateFile, 0, len(files))
	for _, f := range files {
		cfs = append(cfs, workspace.CandidateFile{Path: f.Path, Content: f.Content, Executable: f.Executable})
	}
	if err := rs.Materialize(cfs); err != nil {
		return fmt.Errorf("closedloop: 物化修复继续基 %s: %w", base.ID(), err)
	}
	return nil
}
