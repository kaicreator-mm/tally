package verification

// visible_executor.go 实现 PRD §14.1 Incremental Validation 的可见增量验证执行器
// (P1-28):在 Agent 迭代过程中,Tally 只执行冻结 VerificationContract 里
// visible 的 required/protected binding,把结构化结果返回给调用方记录;
// held_out binding 从不执行、其 ref/内容/可识别判定也从不进入本执行器返回的
// Agent 可见输出(INV-VER-05;P1 采用 only-final held-out execution,
// held-out 的执行归 P1-29 的 final executor)。
//
// 规则来源:
//   - §14.1:每轮 Agent 修改后优先运行 focused visible required/protected
//     validation;held-out evidence 不作为 Implementation Agent 的迭代反馈;
//   - §14.3:Agent 的「我认为通过」只是 Claim;本执行器产出的是结构化观察
//     (structured command、exit code、stdout/stderr digest、耗时),经调用方
//     经 fact-capable 通道(RecorderVerificationRun/Validator)落库后才是
//     Evidence。本执行器不写任何 lifecycle 状态:Task/VERIFIED 与否由 State
//     Engine 基于覆盖推导(coverage_functional.go)另证,与增量结果无直连;
//   - §11.2 / 决策锁 6:命令只接受结构化 CommandSpec(argv/cwd/env/timeout),
//     执行预算受 ExecutionPolicy 约束;shell 语义在类型上不可表达;
//   - INV-EXEC-01:执行发生在调用方提供的 candidate workspace(一次性 Git
//     worktree)内;本执行器把相对 cwd 机械限制在该 workspace 之内,绝不运行
//     在 owner 主 worktree(owner 主 worktree 从不作为 workspace 传入)。
//
// 边界(P1):
//   - 本文件不持久化任何事实(证据落库归 evidence 存储任务),不推导
//     coverage/effective independence,不产生 §8.7 的 ExecutionVerdict
//     (verdict 绑定 immutable ChangeSet + 冻结环境身份,归 final executor);
//     增量结果是迭代反馈与可记录观察,不是最终 verdict;
//   - required_gates 的执行(§14.2)与 held-out 执行(§14.2)不在本文件;
//   - 网络 deny 等沙箱级约束由 runtime/workspace 层强制;本执行器消费已校验的
//     ExecutionPolicy 执行它本地可机械强制的部分:策略合法性、墙钟预算上限、
//     单命令超时上限、结构化 argv(无 shell)、workspace 内 cwd。

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"tally/internal/command"
	"tally/internal/policy"
	"tally/internal/runner"
)

// 哨兵错误:调用方用 errors.Is 分支,不做错误字符串匹配。
var (
	// ErrHeldOutNotIncremental — 请求把 held_out binding 作为增量验证执行。
	// §14.1/INV-VER-05:held-out 永不进入实现侧迭代反馈。错误文案不携带该
	// binding 的 id/store/ref,避免经错误通道泄露任何 held-out 线索。
	ErrHeldOutNotIncremental = errors.New("verification: 请求的 binding 不是 visible 增量验证,held_out 只能在最终验证执行")
	// ErrUnknownBinding — Focus 引用了契约中不存在的 binding id(契约外 id
	// 不是 held-out 线索,可以随错误回显)。
	ErrUnknownBinding = errors.New("verification: focus 引用了契约中不存在的 binding id")
	// ErrNoWorkspace — 未提供 candidate workspace;增量验证必须运行在一次性
	// candidate workspace 内(INV-EXEC-01),不允许落到进程默认 cwd。
	ErrNoWorkspace = errors.New("verification: 增量验证必须提供 candidate workspace 目录")
	// ErrBudgetExhausted — 本轮增量验证的墙钟预算(ExecutionPolicy
	// budgets.max_wall_time)已用尽,剩余 binding 不再执行。
	ErrBudgetExhausted = errors.New("verification: 增量验证墙钟预算已用尽")
)

// VisibleResultStatus — 一条 visible binding 增量执行的观察结果类别。
// pass/fail 是命令跑完后的退出码判定;error 表示「没有产生可判定 verdict 的
// 执行」(起不来、超时/取消、预算用尽、规格非法),它不是 FAIL,不得当成
// 验证失败事实记录。
type VisibleResultStatus string

const (
	VisibleStatusPass  VisibleResultStatus = "pass"
	VisibleStatusFail  VisibleResultStatus = "fail"
	VisibleStatusError VisibleResultStatus = "error"
)

// VisibleBindingResult — 一条 visible binding 的增量执行观察(§14.3 的结构化
// 结果投影)。纯值类型:记录它不改变任何 lifecycle 状态;Stdout/Stderr 是
// visible binding 的可见输出,允许进入实现侧迭代反馈。
type VisibleBindingResult struct {
	// BindingID — 契约中的 binding id(仅 visible binding 会出现)。
	BindingID string
	// AcceptanceID — 所属 Acceptance Clause id。
	AcceptanceID string
	// Status — pass | fail | error。
	Status VisibleResultStatus
	// ExitCode — 命令退出码;Status=error 时无意义(0)。
	ExitCode int
	// Stdout/Stderr — 可见原始输出(§14.3:digest 进证据,原文供迭代反馈)。
	Stdout string
	Stderr string
	// StdoutDigest/StderrDigest — sha256:<hex>,供证据记录而非原文搬运。
	StdoutDigest string
	StderrDigest string
	// Duration — 本次命令的实际耗时(用于墙钟预算记账)。
	Duration time.Duration
	// Err — Status=error 时的人类可读原因(pass/fail 时为空)。
	Err string
}

// EvidenceRecord 把该观察映射为可记录的功能覆盖证据记录(append-only 事实的
// 只读投影,供 DeriveFunctionalCoverage 消费)。Status=error 没有可声明的
// verdict,拒绝映射 —— 「执行出错」不得伪装成 pass/fail 事实。
func (r VisibleBindingResult) EvidenceRecord(revision int) (EvidenceRecord, error) {
	if revision < 1 {
		return EvidenceRecord{}, fmt.Errorf("verification: evidence record revision 必须 >= 1(当前 %d)", revision)
	}
	switch r.Status {
	case VisibleStatusPass:
		return EvidenceRecord{BindingID: r.BindingID, Revision: revision, Status: EvidenceStatusPass}, nil
	case VisibleStatusFail:
		return EvidenceRecord{BindingID: r.BindingID, Revision: revision, Status: EvidenceStatusFail}, nil
	default:
		return EvidenceRecord{}, fmt.Errorf("verification: binding %s 增量执行状态为 %q,没有可记录的 verdict", r.BindingID, r.Status)
	}
}

// IncrementalReport — 一轮可见增量验证的完整报告。它是 Agent 可见输出的唯一
// 载体:只含 visible binding 的结果;held-out 只以数量出现,任何 id/store/ref/
// 判定都不进入本类型(AC4)。
type IncrementalReport struct {
	TaskID           string
	ContractRevision int
	ContractHash     string
	// Results — 已执行 visible binding 的结果,按契约声明顺序。
	Results []VisibleBindingResult
	// SkippedHeldOut — 本轮选择中因 held_out 被跳过的 binding 数量。只有计数,
	// 没有任何可识别信息(AC4)。
	SkippedHeldOut int
	// BudgetExhausted — 本轮墙钟预算是否用尽(true 时尾部的 error 结果是
	// 预算原因,不是验证失败)。
	BudgetExhausted bool
	// TotalDuration — 已执行命令的实际耗时合计。
	TotalDuration time.Duration
}

// IncrementalRequest — 一轮增量验证的输入。
type IncrementalRequest struct {
	// Contract — 冻结的 VerificationContract(权威;作者侧契约不可执行)。
	Contract *Frozen
	// Workspace — candidate workspace 根目录(一次性 Git worktree;INV-EXEC-01)。
	// binding 命令的相对 cwd 在其内解析,逃出即拒绝。必须为绝对路径。
	Workspace string
	// Focus — 可选的 focused binding id 集合(§14.1:focused validation)。
	// 空 = 执行全部 visible binding。引用 held_out id 时整体拒绝(错误不回显
	// 该 id);引用未知 id 返回 ErrUnknownBinding。
	Focus []string
}

// VisibleExecutor — 可见增量验证执行器(P1-28)。构造时校验并冻结策略值;
// Run 对每次调用独立记账墙钟预算。
type VisibleExecutor struct {
	runner runner.Runner
	policy policy.ExecutionPolicy
	budget time.Duration // policy.Budgets.MaxWallTime,构造时已校验为正
}

// NewVisibleExecutor 创建执行器。策略非法时拒绝创建 —— 没有合法 ExecutionPolicy
// 就没有合法的执行边界,宁可完全不执行(§11.2:策略是执行边界的唯一权威)。
func NewVisibleExecutor(r runner.Runner, p policy.ExecutionPolicy) (*VisibleExecutor, error) {
	if r == nil {
		return nil, fmt.Errorf("verification: visible executor 需要 runner")
	}
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("verification: 执行策略非法,拒绝创建增量验证执行器: %w", err)
	}
	return &VisibleExecutor{runner: r, policy: p, budget: p.Budgets.MaxWallTime}, nil
}

// Run 执行一轮可见增量验证。只执行 visible binding(AC1);每条命令以结构化
// argv 运行并受 ExecutionPolicy 预算约束(AC2);结果只以值返回,由调用方记录,
// 不直接设置任何 Task 状态(AC3);held-out 不执行且不泄露进报告(AC4)。
//
// 选择规则:无 Focus 时按契约声明顺序执行全部 visible binding,held-out 计数
// 跳过;有 Focus 时只执行点名的 visible binding(保持契约顺序),点名 held-out
// 或未知 id 时返回错误且不执行任何命令。
//
// 单条 binding 的执行失败(如命令起不来)不中断整轮:它成为该 binding 的
// error 结果,其余 binding 继续在剩余预算内执行,给迭代反馈尽量完整的画面。
func (x *VisibleExecutor) Run(ctx context.Context, req IncrementalRequest) (IncrementalReport, error) {
	if req.Contract == nil {
		return IncrementalReport{}, fmt.Errorf("verification: 增量验证需要冻结的 VerificationContract")
	}
	if strings.TrimSpace(req.Workspace) == "" {
		return IncrementalReport{}, ErrNoWorkspace
	}
	if !filepath.IsAbs(req.Workspace) {
		return IncrementalReport{}, fmt.Errorf("verification: workspace 必须是绝对路径,当前 %q", req.Workspace)
	}

	selected, skippedHeldOut, err := selectVisibleBindings(req.Contract, req.Focus)
	if err != nil {
		return IncrementalReport{}, err
	}

	rep := IncrementalReport{
		TaskID:           req.Contract.TaskID(),
		ContractRevision: req.Contract.Revision(),
		ContractHash:     req.Contract.Hash(),
		SkippedHeldOut:   skippedHeldOut,
	}

	for _, sel := range selected {
		switch {
		case ctx.Err() != nil:
			rep.Results = append(rep.Results, sel.errorResult(ctx.Err().Error()))
		case rep.BudgetExhausted:
			rep.Results = append(rep.Results, sel.errorResult(ErrBudgetExhausted.Error()))
		default:
			remaining := x.budget - rep.TotalDuration // BudgetExhausted 未触发时必为正
			res := x.runBinding(ctx, req.Workspace, sel, remaining)
			rep.Results = append(rep.Results, res)
			rep.TotalDuration += res.Duration
			// 实际耗时合计达到墙钟预算后,剩余 binding 不再执行:预算是本轮
			// 全部命令的硬上限(AC2)。
			if rep.TotalDuration >= x.budget {
				rep.BudgetExhausted = true
			}
		}
	}
	return rep, nil
}

// bindingSelection — 选中的一条 visible binding 及其所属 acceptance。
type bindingSelection struct {
	acceptanceID string
	binding      Binding
}

// errorResult — 选中但未真正执行的 binding 的 error 结果(Duration=0)。
func (s bindingSelection) errorResult(reason string) VisibleBindingResult {
	return VisibleBindingResult{
		BindingID:    s.binding.ID,
		AcceptanceID: s.acceptanceID,
		Status:       VisibleStatusError,
		Err:          reason,
	}
}

// selectVisibleBindings 解析执行选择:契约声明顺序下的 visible binding 子集。
// 无 focus = 全部 visible,held-out 计数跳过;有 focus = 点名子集,点名
// held-out 时整体拒绝(文案不带 id),点名未知 id 报 ErrUnknownBinding。
func selectVisibleBindings(f *Frozen, focus []string) ([]bindingSelection, int, error) {
	focusSet := make(map[string]bool, len(focus))
	for _, id := range focus {
		focusSet[id] = true
	}
	useFocus := len(focus) > 0

	// 先做全量校验,再执行:任何选择错误都在运行前暴露。
	known := map[string]bool{}
	heldOutRequested := false
	for i := range f.c.Acceptances {
		acc := &f.c.Acceptances[i]
		for j := range acc.EvidenceBindings {
			b := &acc.EvidenceBindings[j]
			known[b.ID] = true
			if useFocus && focusSet[b.ID] && b.VisibilityPolicy == VisibilityHeldOut {
				heldOutRequested = true
			}
		}
	}
	if heldOutRequested {
		// INV-VER-05:错误文案不携带任何 held-out id/store/ref。
		return nil, 0, ErrHeldOutNotIncremental
	}
	if useFocus {
		for _, id := range focus {
			if !known[id] {
				return nil, 0, fmt.Errorf("%w: %q", ErrUnknownBinding, id)
			}
		}
	}

	var selected []bindingSelection
	skipped := 0
	for i := range f.c.Acceptances {
		acc := &f.c.Acceptances[i]
		for j := range acc.EvidenceBindings {
			b := &acc.EvidenceBindings[j]
			if useFocus && !focusSet[b.ID] {
				continue
			}
			if b.VisibilityPolicy != VisibilityVisible {
				skipped++ // 只计数:数量之外的一切 held-out 信息都不进入报告(AC4)
				continue
			}
			selected = append(selected, bindingSelection{acceptanceID: acc.ID, binding: cloneBinding(*b)})
		}
	}
	return selected, skipped, nil
}

// runBinding 执行单条 visible binding:把冻结协议里的结构化命令转换为
// internal/command.CommandSpec(无 shell 语义),单命令超时受「binding 声明的
// timeout 与本轮剩余墙钟预算的较小者」约束,经 runner 执行并投影为观察结果。
// remaining 必须为正(Run 保证:预算用尽后不再进入本函数)。
func (x *VisibleExecutor) runBinding(ctx context.Context, workspace string, sel bindingSelection, remaining time.Duration) VisibleBindingResult {
	out := VisibleBindingResult{BindingID: sel.binding.ID, AcceptanceID: sel.acceptanceID}
	specErr := func(err error) VisibleBindingResult {
		out.Status = VisibleStatusError
		out.Err = err.Error()
		return out
	}

	proto := sel.binding.ExecutionProtocol
	if proto == nil || proto.Command == nil {
		return specErr(errors.New("binding 没有冻结的可执行命令(execution_protocol.command 缺失),增量验证无法执行"))
	}
	spec := proto.Command
	if err := spec.Validate(); err != nil {
		return specErr(err)
	}
	dir, err := resolveWithinWorkspace(workspace, spec.Dir)
	if err != nil {
		return specErr(err)
	}
	env, err := bindingEnv(spec.Env)
	if err != nil {
		return specErr(err)
	}

	// internal/command.CommandSpec:Shell 恒为 false —— 增量验证命令只接受
	// 结构化 argv,shell 语义在转换中不可表达(决策锁 6)。
	cs := command.CommandSpec{Argv: append([]string(nil), spec.Argv...), Cwd: dir, Shell: false}
	if err := cs.Validate(); err != nil {
		return specErr(err)
	}

	// ExecutionPolicy 墙钟预算是本轮硬上限(AC2):单命令超时取 binding 声明
	// 的 timeout 与剩余预算的较小者;binding 未声明超时时直接使用剩余预算。
	timeout := time.Duration(spec.Timeout)
	if timeout == 0 || timeout > remaining {
		timeout = remaining
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	raw, runErr := x.runner.Run(runCtx, runner.Command{Argv: cs.Argv, Dir: cs.Cwd, Env: env})

	out.Stdout = raw.Stdout
	out.Stderr = raw.Stderr
	out.StdoutDigest = sha256Digest(raw.Stdout)
	out.StderrDigest = sha256Digest(raw.Stderr)
	out.Duration = raw.Duration
	switch {
	case runErr != nil:
		// runner 的 error 只表示「没跑起来/被取消」:无 verdict,归 error。
		out.Status = VisibleStatusError
		out.Err = runErr.Error()
	case raw.ExitCode != 0:
		out.Status = VisibleStatusFail
		out.ExitCode = raw.ExitCode
	default:
		out.Status = VisibleStatusPass
	}
	return out
}

// resolveWithinWorkspace 把 binding 命令的相对 cwd 解析到 candidate workspace
// 之内;绝对路径与逃出 workspace 的相对路径一律拒绝(INV-EXEC-01:增量验证
// 不运行在 candidate workspace 之外的任何位置)。词法检查(无 symlink 解析):
// workspace 由 Tally 一次性创建管理,深度逃逸防御归 workspace 层。
func resolveWithinWorkspace(workspace, dir string) (string, error) {
	if strings.TrimSpace(dir) == "" {
		return workspace, nil
	}
	if filepath.IsAbs(dir) {
		return "", fmt.Errorf("verification: binding 命令 cwd 必须是 workspace 内相对路径,当前 %q", dir)
	}
	p := filepath.Clean(filepath.Join(workspace, dir))
	rel, err := filepath.Rel(workspace, p)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("verification: binding 命令 cwd %q 逃出 candidate workspace", dir)
	}
	return p, nil
}

// bindingEnv 校验追加环境变量的 KEY=VALUE 形态:KEY 非空且不重复。追加语义与
// runner.Command.Env 一致(叠加在执行环境之上,不是完整快照)。
func bindingEnv(env []string) ([]string, error) {
	if len(env) == 0 {
		return nil, nil
	}
	seen := make(map[string]bool, len(env))
	out := make([]string, 0, len(env))
	for _, kv := range env {
		i := strings.Index(kv, "=")
		if i <= 0 {
			return nil, fmt.Errorf("verification: binding 命令 env 项必须是 KEY=VALUE 且 KEY 非空,当前 %q", kv)
		}
		k := kv[:i]
		if seen[k] {
			return nil, fmt.Errorf("verification: binding 命令 env KEY 重复: %s", k)
		}
		seen[k] = true
		out = append(out, kv)
	}
	return out, nil
}

// sha256Digest 返回内容的 sha256:<hex> 摘要(§14.3:证据记录 digest)。
func sha256Digest(s string) string {
	sum := sha256.Sum256([]byte(s))
	return "sha256:" + hex.EncodeToString(sum[:])
}
