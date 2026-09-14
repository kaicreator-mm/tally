# Current Tally Code Evidence

These excerpts come from the user-provided `tally.zip` baseline and are local authority.

## internal/workflow/v13.go:1-7

```text
    1: // Package workflow 实现 P1 单 Task 垂直工作流内核(P1-39,PRD §17 P0/P1
    2: // vertical subset、§29 Debug 生命周期、§11.2.1 升级语义、§15 CodegenRun/ChangeSet)。
    3: //
    4: // 内核只做四件事,全部围绕**一个 Task 的顺序执行**:
    5: //
    6: //  1. Run 生命周期:一次 Task 执行 = 有限个顺序 CodegenRun;每个 Run 拥有一个
    7: //     一次性 Git worktree(P1-35)与一个 Runtime 会话(P1-04 协议),从 running
```

## internal/workflow/v13.go:10-29

```text
   10: //  2. 有限 retry/escalation 钩子:迭代次数上界与升级阈值来自 ExecutionPolicy
   11: //     (P1-03),内核只执行不定义;失败迭代达到阈值即升级边界 —— 有升级目标:
   12: //     当前 Run `status: escalated` 终止、新 Run 记 escalated_from(§11.2.1);
   13: //     无目标或目标非法:NEEDS_REVIEW(§29)。内核保证有限性:每 Run 迭代数有
   14: //     上界、升级目标 profile 不得重复(重复视为无目标)、升级链长度有绝对上限。
   15: //  3. 候选 ChangeSet 创建:到达候选固定点的 Run 经 FixCandidate 钩子固定
   16: //     恰好 0..1 个不可变候选快照(P1-05);crash/预算耗尽/升级终止的 Run
   17: //     不产生 ChangeSet(§11.2.1/§15.3)。
   18: //  4. 停止条件:策略列出的 stop condition 一旦被观测,立即终止当前 Run
   19: //     (无 ChangeSet)并把 Task 判为 NEEDS_REVIEW(§11.2/§29)。
   20: //
   21: // 刻意不存在的东西(AC4,§17「不得实现通用 workflow DSL」):Task DAG、
   22: // 就绪选择器、跨 Task 依赖、并行 Task 调度、通用编排语言。一个
   23: // TaskExecution 恰好驱动一个 Task,Execute 恰好可调用一次;任务状态由证据
   24: // 推导,内核不写 TaskExecutionState(§18.1),只产出可观测事实与结果值。
   25: //
   26: // 本包不执行进程、不发起网络请求、不解析 shell 字符串;所有外部确定性判断
   27: // (候选固定、停止观测、升级目标解析)都是调用方注入的显式钩子。
   28: package workflow
   29: 
```

## internal/workflow/v13.go:52-86

```text
   52: // 即便升级钩子不断产出新 profile 名,链长也不能超过此界。
   53: const escalationCap = 16
   54: 
   55: // 哨兵错误:调用方用 errors.Is 判因,不做字符串匹配。
   56: var (
   57: 	// ErrInvalidSpec 表示 TaskExecutionSpec / Hooks 不满足构造约束(缺必填、
   58: 	// 策略非法、钩子缺失等)。
   59: 	ErrInvalidSpec = errors.New("workflow: 非法的 TaskExecution 配置")
   60: 	// ErrAlreadyExecuted 表示对同一 TaskExecution 二次调用 Execute。一次执行
   61: 	// 对象只驱动一个 Task 的一次顺序执行,不可复用,也没有批量入口。
   62: 	ErrAlreadyExecuted = errors.New("workflow: TaskExecution 已执行,不可复用")
   63: 	// ErrUnregisteredStopCondition 表示停止观测钩子上报了当前 ExecutionPolicy
   64: 	// 未列出的条件:观测面与策略面不一致,按配置错误处理,绝不静默放行。
   65: 	ErrUnregisteredStopCondition = errors.New("workflow: 观测到策略未列出的停止条件")
   66: )
   67: 
   68: // RunID 是内核分配的 CodegenRun 标识。对 workspace(P1-35 的 worktree 归属)
   69: // 与 changeset(producer run 引用)透明;持久化主键由 RunSink 自行映射。
   70: type RunID string
   71: 
   72: // TaskStatus 是一次 Task 顺序执行的任务级结果。它只是内核的返回值,不是
   73: // TaskExecutionState 写入(§18.1:状态由证据推导,Agent/内核都不得直写)。
   74: type TaskStatus string
   75: 
   76: const (
   77: 	// TaskCompleted 表示某个 Run 固定了候选 ChangeSet,执行成功收束。
   78: 	TaskCompleted TaskStatus = "completed"
   79: 	// TaskNeedsReview 表示重试/升级阶梯耗尽、升级不可用、Run 中途夭折或触发
   80: 	// 停止条件 —— 按 §29 进入 NEEDS_REVIEW 等待人裁决。
   81: 	TaskNeedsReview TaskStatus = "needs_review"
   82: 	// TaskCancelled 表示调用方取消了执行。
   83: 	TaskCancelled TaskStatus = "cancelled"
   84: )
   85: 
   86: // NeedsReviewReason 说明 NEEDS_REVIEW 的成因(§29 阶梯落点)。
```

## internal/workflow/v13.go:95-116

```text
   95: 	ReviewRunAborted NeedsReviewReason = "run_aborted"
   96: 	// ReviewStopCondition 观测到策略停止条件,治理停止。
   97: 	ReviewStopCondition NeedsReviewReason = "stop_condition"
   98: )
   99: 
  100: // TaskExecutionSpec 是一次 Task 顺序执行的全部输入。
  101: type TaskExecutionSpec struct {
  102: 	// Project 项目边界。
  103: 	Project core.ProjectID
  104: 	// Task 被执行的合同(身份与指令来源);内核要求它已过 Validate。
  105: 	Task taskcontract.TaskContract
  106: 	// Role 是本执行的逻辑角色(PRD §9;如 "coding")。
  107: 	Role string
  108: 	// Policy 是执行边界的唯一权威(PRD §11.2),必须过 Validate。
  109: 	Policy policy.ExecutionPolicy
  110: 	// BaseRevision 所有 Run 的共同 base revision(每次执行从同一基线出发;
  111: 	// ChangeSet 不跨 Run 累积,§15.3)。
  112: 	BaseRevision string
  113: 	// 首个 Run 的归因坐标(§11.2.1:runtime/model/behavior_revision 是 Run 内
  114: 	// 单值事实;升级 Run 的对应值来自 EscalationTarget)。
  115: 	Profile          string
  116: 	Runtime          agentruntime.Runtime
```

## internal/workflow/v13.go:120-130

```text
  120: 	// (§15.2;内核不解释,只为 RunSink 透传)。
  121: 	VerificationRevision string
  122: 	ContextPackHash      string
  123: 	// Worktrees 提供 Run 的一次性源码 worktree(P1-35)。
  124: 	Worktrees *workspace.RunTable
  125: 	// Sink 接收 Run 生命周期与 ChangeSet 事实(P1-33 的进程内适配点)。
  126: 	Sink RunSink
  127: 	// Hooks 是确定性集成钩子;FixCandidate 必填,其余可空(语义见 Hooks)。
  128: 	Hooks Hooks
  129: 	// Now 返回事实时刻;nil 则用 time.Now。测试注入固定时钟。
  130: 	Now func() time.Time
```

## internal/policy/execution.go:61-73

```text
   61: // 编码 agent」(PRD §32.1)由类型形状保证 —— 没有可置位的生产作用域。
   62: type CredentialPolicy struct {
   63: 	ScopedSecrets []string // capability-scoped secret 引用名;空 = 不授予任何凭据
   64: }
   65: 
   66: // RetryPolicy 约束同一 CodegenRun 内的重试次数。必须有限(>=1)。
   67: type RetryPolicy struct {
   68: 	MaxIterations int
   69: }
   70: 
   71: // Budgets 约束一次执行的资源上限。两项都必须有限且为正。
   72: type Budgets struct {
   73: 	MaxTokens   int64
```

## internal/policy/execution.go:104-114

```text
  104: // 构造后必须通过 Validate 才能视为有效。
  105: type ExecutionPolicy struct {
  106: 	Network        NetworkPolicy
  107: 	Shell          ShellPolicy
  108: 	Credentials    CredentialPolicy
  109: 	Retry          RetryPolicy
  110: 	Budgets        Budgets
  111: 	Escalation     EscalationPolicy
  112: 	StopConditions []StopCondition
  113: }
  114: 
```

## internal/policy/execution.go:119-129

```text
  119: func DefaultExecutionPolicy() ExecutionPolicy {
  120: 	return ExecutionPolicy{
  121: 		Network:        NetworkPolicy{Mode: NetworkDeny},
  122: 		Shell:          ShellPolicy{AllowShell: false},
  123: 		Credentials:    CredentialPolicy{},
  124: 		Retry:          RetryPolicy{MaxIterations: 4},
  125: 		Budgets:        Budgets{MaxTokens: 180000, MaxWallTime: 30 * time.Minute},
  126: 		Escalation:     EscalationPolicy{AfterFailedIterations: 2, TargetProfile: "strong-coding"},
  127: 		StopConditions: slices.Clone(hardStopConditions),
  128: 	}
  129: }
```

## internal/policy/execution.go:136-147

```text
  136: 		return err
  137: 	}
  138: 	if err := validateCredentials(p.Credentials); err != nil {
  139: 		return err
  140: 	}
  141: 	if p.Retry.MaxIterations < 1 {
  142: 		return fmt.Errorf("%w: retry.max_iterations=%d, 必须 >=1(预算必须有限)", ErrInvalidPolicy, p.Retry.MaxIterations)
  143: 	}
  144: 	if p.Budgets.MaxTokens < 1 {
  145: 		return fmt.Errorf("%w: budgets.max_tokens=%d, 必须 >=1", ErrInvalidPolicy, p.Budgets.MaxTokens)
  146: 	}
  147: 	if p.Budgets.MaxWallTime <= 0 {
```

## internal/policy/execution.go:151-161

```text
  151: 		return fmt.Errorf("%w: escalation.after_failed_iterations=%d, 必须 >=1", ErrInvalidPolicy, p.Escalation.AfterFailedIterations)
  152: 	}
  153: 	if p.Escalation.TargetProfile == "" {
  154: 		return fmt.Errorf("%w: escalation.target_profile 为空", ErrInvalidPolicy)
  155: 	}
  156: 	if p.Escalation.AfterFailedIterations > p.Retry.MaxIterations {
  157: 		return fmt.Errorf("%w: escalation.after_failed_iterations=%d 超过 retry.max_iterations=%d, 升级永远不会触发",
  158: 			ErrInvalidPolicy, p.Escalation.AfterFailedIterations, p.Retry.MaxIterations)
  159: 	}
  160: 	return validateStopConditions(p.StopConditions)
  161: }
```

## internal/verification/orchestrator.go:8-18

```text
    8: //   1. 逐条执行全部 visible binding 的冻结执行协议(repetitions/all_pass/seed
    9: //      经 ExecutionSession 机械强制,§8.7.2),verdict 绑定 §8.7.1 四元组
   10: //      (contract revision + binding + candidate + frozen protocol + environment);
   11: //   2. 执行 required_gates 的结构化命令(§14.2 required build/architecture/
   12: //      security/schema gates;结构化 argv,无 shell 语义,决策锁 6);
   13: //   3. 经 HoldoutExecutor(P1-29)把 held_out binding 注入 Tally 控制的最终
   14: //      验证:一次性消耗、无重试(决策锁 12);
   15: //   4. 汇总本轮 verdict 与「既往 valid 且 applicable 的 evidence」(经持久化
   16: //      端口读回)推导最终功能覆盖(P1-19)与 effective independence 覆盖
   17: //      (P1-20),由此给出 verified / failed / needs_review 结论;
   18: //   5. 经 RunSink 端口恰好落一行 VerificationRun(绑定精确 contract/change/
```

## internal/verification/orchestrator.go:40-50

```text
   40: //     (基础设施/取消/完整性失败)返回 error 且不落任何行——无 verdict 的
   41: //     半次执行不得伪装成事实;
   42: //   - Evidence Qualification 的 §8.3 final checks 与 protected 仓库文件哈希
   43: //     归控制面(P1-16 评估器 / hunk inventory);本编排器只消费冻结协议执行;
   44: //   - 实现侧可见投影只有 RedactedSignal(allowlist):结构上不含 held-out
   45: //     内容/ref/digest/verdict;完整 HoldoutVerdictRecord 只在 Control/Review
   46: //     通道(FinalOutcome)。
   47: 
   48: import (
   49: 	"context"
   50: 	"errors"
```

## internal/verification/orchestrator.go:156-169

```text
  156: 	// Runner 执行 visible binding 与 required gate 的结构化命令(一次性
  157: 	// candidate workspace 内,INV-EXEC-01)。
  158: 	Runner runner.Runner
  159: 	// Policy 是执行边界(墙钟预算上限);非法策略拒绝构造执行器。
  160: 	Policy policy.ExecutionPolicy
  161: 	// HoldoutReader / HoldoutRun 是 held-out 注入通道(P1-29);契约不含
  162: 	// held_out binding 时可以不接。实现权威侧组件的类型签名里不存在它们。
  163: 	HoldoutReader holdout.VerificationReader
  164: 	HoldoutRun    HoldoutRunFunc
  165: 	// Sink 是持久化端口(见 RunSink)。
  166: 	Sink RunSink
  167: 	// Recorder 是落库 provenance;必须是 fact-capable 通道(INV-AUTH-01/02)。
  168: 	Recorder evidence.Recorder
  169: }
```

## internal/verification/orchestrator.go:173-183

```text
  173: // 的多次 Run 各自成行,禁止 cherry-pick 靠全量留痕(§8.13)。
  174: type Orchestrator struct {
  175: 	runner     runner.Runner
  176: 	budget     time.Duration // policy.Budgets.MaxWallTime,构造时已校验为正
  177: 	reader     holdout.VerificationReader
  178: 	holdoutRun HoldoutRunFunc
  179: 	sink       RunSink
  180: 	recorder   evidence.Recorder
  181: }
  182: 
  183: // NewOrchestrator 构造编排器。依赖缺失或策略非法时不产生编排器(保守失败,
```

## internal/verification/orchestrator.go:196-206

```text
  196: 		return nil, fmt.Errorf("%w: recorder %q/%q 不是 fact-capable 通道(INV-AUTH-01/02)", ErrNilDependency, cfg.Recorder.Kind, cfg.Recorder.Actor)
  197: 	}
  198: 	return &Orchestrator{
  199: 		runner:     cfg.Runner,
  200: 		budget:     cfg.Policy.Budgets.MaxWallTime,
  201: 		reader:     cfg.HoldoutReader,
  202: 		holdoutRun: cfg.HoldoutRun,
  203: 		sink:       cfg.Sink,
  204: 		recorder:   cfg.Recorder,
  205: 	}, nil
  206: }
```

## internal/runstore/changeset.go:94-183

```text
   94: 		out = append(out, cs)
   95: 	}
   96: 	return out, nil
   97: }
   98: 
   99: // AppendSupersededFact 追加一条取代事实(append-only,PRD §15.3,AC1):
  100: // old 被 new 取代。同一事实(同 old/new/reason/时刻)重复追加按 SQL 合同
  101: // 幂等忽略;不同时刻各自成行。
  102: func (s *Store) AppendSupersededFact(ctx context.Context, f changeset.SupersededFact) error {
  103: 	if err := validateSupersededFact(f); err != nil {
  104: 		return err
  105: 	}
  106: 	if err := s.q.InsertChangeSetSupersededFact(ctx, store.InsertChangeSetSupersededFactParams{
  107: 		OldChangesetID: string(f.Old),
  108: 		NewChangesetID: string(f.New),
  109: 		Reason:         string(f.Reason),
  110: 		SupersededAt:   tsPtr(f.At),
  111: 	}); err != nil {
  112: 		return fmt.Errorf("runstore: 追加取代事实 %s→%s: %w", f.Old, f.New, err)
  113: 	}
  114: 	return nil
  115: }
  116: 
  117: // ChangeSetSupersessions 返回某快照参与的取代事实(作为 old 或 new),
  118: // 按发生时刻定序:派生 superseded_by / 取代链的投影源。
  119: func (s *Store) ChangeSetSupersessions(ctx context.Context, id changeset.ChangeSetID) ([]changeset.SupersededFact, error) {
  120: 	rows, err := s.q.ListChangeSetSupersessionsForChangeSet(ctx, string(id))
  121: 	if err != nil {
  122: 		return nil, fmt.Errorf("runstore: 查询 changeset %s 的取代事实: %w", id, err)
  123: 	}
  124: 	out := make([]changeset.SupersededFact, 0, len(rows))
  125: 	for _, r := range rows {
  126: 		f, err := supersededFactFromRow(r.OldChangesetID, r.NewChangesetID, r.Reason, r.SupersededAt)
  127: 		if err != nil {
  128: 			return nil, err
  129: 		}
  130: 		out = append(out, f)
  131: 	}
  132: 	return out, nil
  133: }
  134: 
  135: // AllChangeSetSupersessions 返回全部取代事实(排序确定):
  136: // 取代投影整体重建 / 对账用(INV-STATE-01 可重建)。
  137: func (s *Store) AllChangeSetSupersessions(ctx context.Context) ([]changeset.SupersededFact, error) {
  138: 	rows, err := s.q.ListChangeSetSupersessions(ctx)
  139: 	if err != nil {
  140: 		return nil, fmt.Errorf("runstore: 查询全部取代事实: %w", err)
  141: 	}
  142: 	out := make([]changeset.SupersededFact, 0, len(rows))
  143: 	for _, r := range rows {
  144: 		f, err := supersededFactFromRow(r.OldChangesetID, r.NewChangesetID, r.Reason, r.SupersededAt)
  145: 		if err != nil {
  146: 			return nil, err
  147: 		}
  148: 		out = append(out, f)
  149: 	}
  150: 	return out, nil
  151: }
  152: 
  153: func supersededFactFromRow(oldID, newID, reason string, at pgtype.Timestamptz) (changeset.SupersededFact, error) {
  154: 	f, err := changeset.NewSupersededFact(
  155: 		changeset.ChangeSetID(oldID),
  156: 		changeset.ChangeSetID(newID),
  157: 		changeset.SupersedeReason(reason),
  158: 		tsVal(at),
  159: 	)
  160: 	if err != nil {
  161: 		return changeset.SupersededFact{}, fmt.Errorf("runstore: 重建取代事实 %s→%s(%s): %w", oldID, newID, reason, err)
  162: 	}
  163: 	return f, nil
  164: }
  165: 
  166: func validateSupersededFact(f changeset.SupersededFact) error {
  167: 	switch {
  168: 	case f.Old == "" || f.New == "":
  169: 		return fmt.Errorf("%w: old/new 为空", changeset.ErrBadSupersede)
  170: 	case f.Old == f.New:
  171: 		return fmt.Errorf("%w: old 与 new 相同", changeset.ErrBadSupersede)
  172: 	case f.At.IsZero():
  173: 		return fmt.Errorf("%w: 时刻为零值", changeset.ErrBadSupersede)
  174: 	}
  175: 	switch f.Reason {
  176: 	case changeset.SupersedeEscalation, changeset.SupersedeRework, changeset.SupersedeSpecDefect:
  177: 	default:
  178: 		return fmt.Errorf("%w: 未知原因 %q", changeset.ErrBadSupersede, f.Reason)
  179: 	}
  180: 	return nil
  181: }
  182: 
  183: // changeSetFromRow 从行重建领域快照并核对内容身份(AC3/AC4)。
```