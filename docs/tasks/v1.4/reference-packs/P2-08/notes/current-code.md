# Current Tally Code Evidence

These excerpts come from the user-provided `tally.zip` baseline and are local authority.

## internal/workflow/v13.go:72-88

```text
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
   87: type NeedsReviewReason string
   88: 
```

## internal/workflow/v13.go:214-230

```text
  214: 
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
  227: }
  228: 
  229: // TaskExecution 驱动一个 Task 的顺序执行。一次性使用:Execute 恰好可调用
  230: // 一次(AC4:没有批量/并发/跨 Task 入口)。
```

## internal/workflow/v13.go:306-326

```text
  306: }
  307: 
  308: // Execute 顺序执行该 Task 的 Run 链直至任务级终态(AC1)。返回 error 仅表示
  309: // 配置/基础设施故障(钩子出错、sink 写入失败、worktree 回收失败等);领域内
  310: // 的阶梯耗尽/停止/取消都是正常 Outcome(§29)。
  311: func (e *TaskExecution) Execute(ctx context.Context) (Outcome, error) {
  312: 	e.mu.Lock()
  313: 	if e.done {
  314: 		e.mu.Unlock()
  315: 		return Outcome{}, ErrAlreadyExecuted
  316: 	}
  317: 	e.done = true
  318: 	e.mu.Unlock()
  319: 
  320: 	if err := ctx.Err(); err != nil {
  321: 		return Outcome{Status: TaskCancelled}, nil
  322: 	}
  323: 
  324: 	target := runTarget{
  325: 		profile:          e.spec.Profile,
  326: 		runtime:          e.spec.Runtime,
```

## internal/workflow/v13.go:339-369

```text
  339: 			return Outcome{}, err
  340: 		}
  341: 		switch res.end {
  342: 		case endCompleted:
  343: 			return Outcome{
  344: 				Status:    TaskCompleted,
  345: 				Runs:      e.records,
  346: 				ChangeSet: res.rec.changeSet,
  347: 			}, nil
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
  368: 
  369: // resolveEscalation 在升级边界解析下一个升级目标。目标非法(缺 profile/runtime、
```

## internal/verification/orchestrator.go:70-87

```text
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

## internal/verification/orchestrator.go:105-123

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
  116: type FactSpec struct {
  117: 	FactID           evidence.FactID
  118: 	RunID            int64
  119: 	Task             string
  120: 	Acceptance       string
  121: 	BindingID        string
  122: 	VisibilityPolicy VisibilityPolicy
  123: 	Verdict          EvidenceStatus // pass|fail(编排器只落有 verdict 的事实)
```

## internal/verification/orchestrator.go:216-229

```text
  216: 	AuthoredInLineage bool
  217: 	// ChangedInLineage — 既存证据内容在当前实现决策 lineage 内被修改。
  218: 	ChangedInLineage bool
  219: }
  220: 
  221: // FinalRunRequest — 一次最终验证的输入。除执行所需边界外,还携带 effective
  222: // independence 推导(§8.5.2)的控制面 provenance 输入:它们是 append-only
  223: // 事实/attestation 的投影,编排器不发明、不改写。
  224: type FinalRunRequest struct {
  225: 	// Project — 项目边界;必须与 ChangeSet 所属项目一致。
  226: 	Project core.ProjectID
  227: 	// Contract — 冻结的 VerificationContract(权威;作者侧契约不可执行)。
  228: 	Contract *Frozen
  229: 	// ChangeSet — 被验证的 immutable ChangeSet(verdict 绑定对象,§8.7.1)。
```

## internal/verification/orchestrator.go:285-295

```text
  285: 
  286: // FinalOutcome — 一次最终验证编排的完整结论。Control/Review Authority 侧
  287: // 事实载体:含完整 verdict、覆盖推导与既往事实合并结果。实现侧只见
  288: // RedactedSignal(AC3)。
  289: type FinalOutcome struct {
  290: 	RunID            int64
  291: 	TaskID           string
  292: 	ContractRevision int
  293: 	ContractHash     string
  294: 	ChangeSetID      changeset.ChangeSetID
  295: 	EnvironmentHash  string
```

## internal/verification/orchestrator.go:376-386

```text
  376: // 结论(Control/Review 侧)。
  377: //
  378: // 求值失败(基础设施错误、取消、holdout 完整性失败、sink 读写失败)返回
  379: // error 且不落任何行:无 verdict 的半次执行不伪装成事实。验证结论本身
  380: // (failed/needs_review)不是 error——失败 Run 恰恰必须被记录(§8.13/AC4)。
  381: func (o *Orchestrator) Run(ctx context.Context, req FinalRunRequest) (*FinalOutcome, error) {
  382: 	if err := o.validateRequest(&req); err != nil {
  383: 		return nil, err
  384: 	}
  385: 	f := req.Contract
  386: 	envHash, err := req.Environment.Hash()
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