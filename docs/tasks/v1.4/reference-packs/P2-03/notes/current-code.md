# Current Tally Code Evidence

These excerpts come from the user-provided `tally.zip` baseline and are local authority.

## internal/runstore/verification.go:23-44

```text
   23: 
   24: // VerificationStatus 是 verification_run.status:落行时一次写入的观测结果。
   25: type VerificationStatus string
   26: 
   27: const (
   28: 	// VerificationPass 观测通过。
   29: 	VerificationPass VerificationStatus = "pass"
   30: 	// VerificationFail 观测失败。
   31: 	VerificationFail VerificationStatus = "fail"
   32: 	// VerificationError 执行出错(无可用 verdict)。
   33: 	VerificationError VerificationStatus = "error"
   34: )
   35: 
   36: // IsValid 报告状态是否在迁移 0020 的 CHECK 取值内。
   37: func (s VerificationStatus) IsValid() bool {
   38: 	switch s {
   39: 	case VerificationPass, VerificationFail, VerificationError:
   40: 		return true
   41: 	default:
   42: 		return false
   43: 	}
   44: }
```

## internal/workflow/v13.go:81-102

```text
   81: 	TaskNeedsReview TaskStatus = "needs_review"
   82: 	// TaskCancelled 表示调用方取消了执行。
   83: 	TaskCancelled TaskStatus = "cancelled"
   84: )
   85: 
   86: // NeedsReviewReason 说明 NEEDS_REVIEW 的成因(§29 阶梯落点)。
   87: type NeedsReviewReason string
   88: 
   89: const (
   90: 	// ReviewEscalationUnavailable 到达升级边界但没有可用升级目标(无 target、
   91: 	// 目标非法,或目标 profile 已用过 —— 重复目标视为无目标,循环必须终止)。
   92: 	ReviewEscalationUnavailable NeedsReviewReason = "escalation_unavailable"
   93: 	// ReviewRunAborted Run 在升级边界前因 crash/预算耗尽/墙钟到期终止;策略
   94: 	// 未到升级点,内核不越权提前升级。
   95: 	ReviewRunAborted NeedsReviewReason = "run_aborted"
   96: 	// ReviewStopCondition 观测到策略停止条件,治理停止。
   97: 	ReviewStopCondition NeedsReviewReason = "stop_condition"
   98: )
   99: 
  100: // TaskExecutionSpec 是一次 Task 顺序执行的全部输入。
  101: type TaskExecutionSpec struct {
  102: 	// Project 项目边界。
```

## internal/workflow/v13.go:215-226

```text
  215: // Outcome 是一次 Task 顺序执行的结果值:Run 时间线 + 任务级落点 + (成功时)
  216: // 固定的候选 ChangeSet。
  217: type Outcome struct {
  218: 	Status TaskStatus
  219: 	// Reason 仅 Status==TaskNeedsReview 时非空。
  220: 	Reason NeedsReviewReason
  221: 	// StopConditions 仅 Reason==ReviewStopCondition 时非空:触发停止的已观测条件。
  222: 	StopConditions []policy.StopCondition
  223: 	// Runs 按开跑顺序排列(顺序执行的时间线,AC1)。
  224: 	Runs []RunRecord
  225: 	// ChangeSet 仅 Status==TaskCompleted 时非 nil:本执行固定的唯一候选。
  226: 	ChangeSet *changeset.ChangeSet
```

## internal/workflow/v13.go:348-367

```text
  348: 		case endCancelled:
  349: 			return Outcome{Status: TaskCancelled, Runs: e.records}, nil
  350: 		case endStopped:
  351: 			return Outcome{
  352: 				Status:         TaskNeedsReview,
  353: 				Reason:         ReviewStopCondition,
  354: 				StopConditions: res.stops,
  355: 				Runs:           e.records,
  356: 			}, nil
  357: 		case endEscalated:
  358: 			// §11.2.1:升级终止当前 Run;新 Run 记 escalated_from。
  359: 			escalatedFrom = res.rec.ID
  360: 			target = res.next
  361: 		case endAborted:
  362: 			return Outcome{Status: TaskNeedsReview, Reason: ReviewRunAborted, Runs: e.records}, nil
  363: 		case endNoEscalation:
  364: 			return Outcome{Status: TaskNeedsReview, Reason: ReviewEscalationUnavailable, Runs: e.records}, nil
  365: 		}
  366: 	}
  367: }
```

## internal/verification/orchestrator.go:61-87

```text
   61: 	"tally/internal/holdout"
   62: 	"tally/internal/policy"
   63: 	"tally/internal/runner"
   64: )
   65: 
   66: // FinalStatus — 一次最终验证编排的结论(§36.3)。
   67: type FinalStatus string
   68: 
   69: const (
   70: 	FinalVerified    FinalStatus = "verified"     // 全部门通过
   71: 	FinalFailed      FinalStatus = "failed"       // 确定性验证失败(可见失败/required gate 失败)
   72: 	FinalNeedsReview FinalStatus = "needs_review" // 非 VERIFIED:需 Control/Review 侧人工/政策裁决
   73: )
   74: 
   75: // RunStatus — VerificationRun 行上的观测结果(与 runstore 的 pass|fail|error
   76: // 词表一致)。needs_review 在行上映射为 error:运行级没有干净的 pass/fail 结论。
   77: type RunStatus string
   78: 
   79: const (
   80: 	RunStatusPass  RunStatus = "pass"
   81: 	RunStatusFail  RunStatus = "fail"
   82: 	RunStatusError RunStatus = "error"
   83: )
   84: 
   85: // 哨兵错误:调用方用 errors.Is 判因,不做字符串匹配。
   86: var (
   87: 	// ErrInvalidRequest — Run 请求不完整或不一致(nil 契约/候选、项目不匹配、
```

## internal/verification/orchestrator.go:105-115

```text
  105: 	EnvironmentHash  string // 环境语义身份内容哈希(EnvironmentIdentity.Hash)
  106: 	VersionCombo     core.VersionCombo
  107: 	Runtime          string // 执行后端身份
  108: 	StartedAt        time.Time
  109: 	FinishedAt       time.Time
  110: 	Status           RunStatus
  111: }
  112: 
  113: // FactSpec — 落一条 evidence 事实的输入(§14.3:结构化命令、判定、摘要、
  114: // recorder provenance)。FactID 由编排器按 (run, binding) 确定性派生:
  115: // 事实身份唯一且永不复用,同一 Run 的同一 binding 二次落库是错误而非更新。
```

## internal/verification/orchestrator.go:118-128

```text
  118: 	RunID            int64
  119: 	Task             string
  120: 	Acceptance       string
  121: 	BindingID        string
  122: 	VisibilityPolicy VisibilityPolicy
  123: 	Verdict          EvidenceStatus // pass|fail(编排器只落有 verdict 的事实)
  124: 	Recorder         evidence.Recorder
  125: 	Command          *CommandSpec // 冻结协议声明的结构化命令
  126: 	ExitCode         int          // 末次 repetition 的退出码
  127: 	StdoutDigest     string
  128: 	StderrDigest     string
```

## internal/verification/orchestrator.go:132-142

```text
  132: // PriorFact — 既往 valid 且 applicable 的 evidence 事实投影(RunSink 读回)。
  133: // 只有 valid 事实会到达这里:被正式裁定失效的事实不再 gate(§8.8.1),
  134: // 过滤责任在持久化端口的实现侧(它是 validity 的权威)。
  135: type PriorFact struct {
  136: 	BindingID  string
  137: 	Status     EvidenceStatus // pass|fail(error 判定无可用 verdict,适配层省略)
  138: 	RecordedAt time.Time      // 原始记录时刻(selection exposure 时钟输入,§8.6.2)
  139: }
  140: 
  141: // RunSink — 最终验证编排的持久化窄端口。实现侧由控制面接线到 append-only
  142: // 事实存储(internal/runstore);端口刻意不含任何更新/删除/裁定方法:
```

## internal/verification/orchestrator.go:255-265

```text
  255: 
  256: // GateResult — 一个 required gate 的执行观察(§14.3 结构化结果投影)。
  257: type GateResult struct {
  258: 	Name         string
  259: 	Required     bool
  260: 	Status       VisibleResultStatus // pass | fail | error
  261: 	ExitCode     int
  262: 	StdoutDigest string
  263: 	StderrDigest string
  264: 	Duration     time.Duration
  265: 	Err          string // Status=error 时的人类可读原因
```