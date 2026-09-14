# Current Tally Code Evidence

These excerpts come from the user-provided `tally.zip` baseline and are local authority.

## internal/agentruntime/protocol.go:20-30

```text
   20: 	"errors"
   21: )
   22: 
   23: // SessionID 标识一次 Runtime 会话(session/thread 边界的协议句柄)。
   24: // Role separation(PRD §10.3)要求 Verification 与 Implementation 落在
   25: // 不同 Session 上,由上层用不同 StartRequest 保证;本包只携带标识。
   26: type SessionID string
   27: 
   28: // RunID 标识会话内的一次运行(一个 turn)。会话内唯一,由适配器生成,
   29: // 经 Execute 返回值与事件流携带,供 Cancel/Result 定位。
   30: type RunID string
```

## internal/agentruntime/protocol.go:44-115

```text
   44: 	RunCancelled RunStatus = "cancelled"
   45: )
   46: 
   47: // 协议级哨兵错误(结构化错误另见 capability.CapabilityError)。
   48: var (
   49: 	// ErrNoRuntimeName 表示 RuntimeDescriptor 缺少 Name;无名 runtime 无法归因。
   50: 	ErrNoRuntimeName = errors.New("agentruntime: RuntimeDescriptor 缺少 name")
   51: 	// ErrNoWorkspace 表示 StartRequest 缺少 workspace。workspace 必须是
   52: 	// 可丢弃 Git worktree —— agent 产生的可执行变更绝不在 owner 主
   53: 	// worktree 运行(INV-EXEC-01)。
   54: 	ErrNoWorkspace = errors.New("agentruntime: StartRequest 缺少 workspace(必须是可丢弃 Git worktree,INV-EXEC-01)")
   55: 	// ErrNoInstruction 表示 ExecuteRequest 缺少 instruction;空 turn 无意义。
   56: 	ErrNoInstruction = errors.New("agentruntime: ExecuteRequest 缺少 instruction")
   57: 	// ErrRunNotFound 表示 Cancel/Result 指定的 run 不存在(或已被遗忘)。
   58: 	ErrRunNotFound = errors.New("agentruntime: 指定的 run 不存在")
   59: 	// ErrSessionClosed 表示会话已 Close,一切操作都被拒绝。
   60: 	ErrSessionClosed = errors.New("agentruntime: 会话已关闭")
   61: )
   62: 
   63: // RuntimeDescriptor 是 runtime 的自述。Capabilities 是声明而不是承诺:
   64: // 声明的能力是否真实可用由 conformance(PRD §10.4,P1-25 harness)另行
   65: // 验证 —— 声明本身仍是 claim,conformance 结果才是证据。
   66: type RuntimeDescriptor struct {
   67: 	// Name 是 runtime 标识(如 "fake"、"codex"),只用于展示/遥测/审计归因。
   68: 	// 任何控制流都不得按 Name 分支 —— 按 CapabilitySet 的协商结果分支(PRD §10.3)。
   69: 	Name string
   70: 	// Version 是适配器/runtime 版本字符串,供归因与排障,可为空。
   71: 	Version string
   72: 	// Capabilities 是该 runtime 声明的能力集合(见 capability.go)。
   73: 	Capabilities CapabilitySet
   74: }
   75: 
   76: // Validate 校验描述符的最小结构:Name 必须非空。
   77: func (d RuntimeDescriptor) Validate() error {
   78: 	if d.Name == "" {
   79: 		return ErrNoRuntimeName
   80: 	}
   81: 	return nil
   82: }
   83: 
   84: // StartRequest 开启会话的请求。
   85: type StartRequest struct {
   86: 	// Workspace 是本次会话的源码执行目录,必须是可丢弃的 Git worktree
   87: 	// (INV-EXEC-01:agent 生成的可执行变更不在 owner 主 worktree 运行)。
   88: 	Workspace string
   89: 	// ResumeOf 非空表示续用既有会话而不是新开;需要 session.resume 能力。
   90: 	ResumeOf SessionID
   91: 	// ModelProfile 非空表示请求指定模型档位;需要 model.select 能力。
   92: 	ModelProfile string
   93: }
   94: 
   95: // RequiredCapabilities 由请求内容推导出的必需能力(纯数据,供 Negotiate
   96: // 使用):基础 Start 无额外要求;ResumeOf ⇒ session.resume;
   97: // ModelProfile ⇒ model.select。返回值去重并按字典序排序。
   98: func (r StartRequest) RequiredCapabilities() []Capability {
   99: 	var caps []Capability
  100: 	if r.ResumeOf != "" {
  101: 		caps = append(caps, CapSessionResume)
  102: 	}
  103: 	if r.ModelProfile != "" {
  104: 		caps = append(caps, CapModelSelect)
  105: 	}
  106: 	return SortedCapabilities(caps)
  107: }
  108: 
  109: // Validate 校验请求的最小结构。
  110: func (r StartRequest) Validate() error {
  111: 	if r.Workspace == "" {
  112: 		return ErrNoWorkspace
  113: 	}
  114: 	return nil
  115: }
```

## internal/agentruntime/protocol.go:160-173

```text
  160: // Runtime 是适配器要实现的最小核心(PRD §10.2)。六件事的落点:
  161: // Descriptor/Start 在 Runtime 上;Execute/Cancel/Events/Result 在其开启的
  162: // Session 上。
  163: type Runtime interface {
  164: 	// Descriptor 返回自述(标识 + 声明能力)。声明由 conformance 另行验证。
  165: 	Descriptor() RuntimeDescriptor
  166: 	// Start 开启一个会话。实现应先过 CheckStart:校验请求结构,并按请求
  167: 	// 推导的必需能力与声明能力协商,缺任何一项即拒绝(AC2)。
  168: 	Start(ctx context.Context, req StartRequest) (Session, error)
  169: }
  170: 
  171: // Session 是一次已开启的协议会话。
  172: type Session interface {
  173: 	// Execute 同步执行一个 turn,返回 runtime 的 AgentRunResult(claim)。
```

## internal/agentruntime/protocol.go:192-202

```text
  192: }
  193: 
  194: // CheckStart 是 Start 实现的统一前置闸:校验请求结构,并按请求推导出的
  195: // 必需能力与声明能力做协商,缺任何一项返回 *CapabilityError。适配器不得
  196: // 跳过它自行放行,否则「按能力协商」就退化成了随意放行(AC2)。
  197: func CheckStart(d RuntimeDescriptor, req StartRequest) error {
  198: 	if err := d.Validate(); err != nil {
  199: 		return err
  200: 	}
  201: 	if err := req.Validate(); err != nil {
  202: 		return err
```

## internal/agentruntime/protocol.go:205-215

```text
  205: 	return err
  206: }
  207: 
  208: // CheckEvents 是 Events 实现的统一前置闸:未声明 event.streaming 的
  209: // runtime 直接拒绝(能力是协商出来的数据,不是接口存在性)。
  210: func CheckEvents(d RuntimeDescriptor) error {
  211: 	if !d.Capabilities.Has(CapEventStreaming) {
  212: 		return ErrStreamingNotDeclared
  213: 	}
  214: 	return nil
  215: }
```

## internal/workflow/v13.go:2-12

```text
    2: // vertical subset、§29 Debug 生命周期、§11.2.1 升级语义、§15 CodegenRun/ChangeSet)。
    3: //
    4: // 内核只做四件事,全部围绕**一个 Task 的顺序执行**:
    5: //
    6: //  1. Run 生命周期:一次 Task 执行 = 有限个顺序 CodegenRun;每个 Run 拥有一个
    7: //     一次性 Git worktree(P1-35)与一个 Runtime 会话(P1-04 协议),从 running
    8: //     迁移到唯一终态(completed/failed/escalated/cancelled),过程事实经
    9: //     RunSink 落库(P1-33 适配)。
   10: //  2. 有限 retry/escalation 钩子:迭代次数上界与升级阈值来自 ExecutionPolicy
   11: //     (P1-03),内核只执行不定义;失败迭代达到阈值即升级边界 —— 有升级目标:
   12: //     当前 Run `status: escalated` 终止、新 Run 记 escalated_from(§11.2.1);
```

## internal/workflow/v13.go:111-122

```text
  111: 	// ChangeSet 不跨 Run 累积,§15.3)。
  112: 	BaseRevision string
  113: 	// 首个 Run 的归因坐标(§11.2.1:runtime/model/behavior_revision 是 Run 内
  114: 	// 单值事实;升级 Run 的对应值来自 EscalationTarget)。
  115: 	Profile          string
  116: 	Runtime          agentruntime.Runtime
  117: 	Model            string
  118: 	BehaviorRevision string
  119: 	// VerificationRevision / ContextPackHash 是开跑即写定的观测面身份坐标
  120: 	// (§15.2;内核不解释,只为 RunSink 透传)。
  121: 	VerificationRevision string
  122: 	ContextPackHash      string
```

## internal/workflow/v13.go:150-160

```text
  150: 	Profile       string
  151: 	WorkspacePath string
  152: 	BaseRevision  string
  153: 	// Iteration 是刚结束的迭代序号(1 起)。
  154: 	Iteration int
  155: 	// Claim 是 Runtime 对该迭代的自述(agentruntime.AgentRunResult:只是
  156: 	// claim,不是事实,INV-AUTH-01)。
  157: 	Claim agentruntime.AgentRunResult
  158: }
  159: 
  160: // CandidateSnapshot 是钩子观测到的候选内容;内核补齐 project/base/producer/
```

## internal/workflow/v13.go:175-197

```text
  175: 
  176: // EscalationTarget 是升级 Run 的归因坐标(§11.2.1:升级即换 Run,单值事实
  177: // 随 Run 更换,绝不在同一 Run 内替换 runtime/model)。
  178: type EscalationTarget struct {
  179: 	Profile          string
  180: 	Runtime          agentruntime.Runtime
  181: 	Model            string
  182: 	BehaviorRevision string
  183: }
  184: 
  185: // EscalationResolveFunc 解析升级目标;usedProfiles 是已用 profile 链。
  186: // ok=false = 无可用目标。
  187: type EscalationResolveFunc func(ctx context.Context, usedProfiles []string) (EscalationTarget, bool, error)
  188: 
  189: // runTarget 是待开 Run 的归因坐标(首个 Run 来自 spec,升级 Run 来自钩子)。
  190: type runTarget struct {
  191: 	profile          string
  192: 	runtime          agentruntime.Runtime
  193: 	model            string
  194: 	behaviorRevision string
  195: }
  196: 
  197: // RunRecord 是一个已终结 Run 的生命周期事实投影(结果值,§29 调试时间线)。
```

## internal/workflow/v13.go:250-260

```text
  250: 		return nil, fmt.Errorf("%w: 缺少 role", ErrInvalidSpec)
  251: 	case strings.TrimSpace(spec.BaseRevision) == "":
  252: 		return nil, fmt.Errorf("%w: 缺少 base revision", ErrInvalidSpec)
  253: 	case strings.TrimSpace(spec.Profile) == "":
  254: 		return nil, fmt.Errorf("%w: 缺少首个 Run 的 profile", ErrInvalidSpec)
  255: 	case spec.Runtime == nil:
  256: 		return nil, fmt.Errorf("%w: 缺少首个 Run 的 runtime", ErrInvalidSpec)
  257: 	case strings.TrimSpace(spec.Model) == "":
  258: 		return nil, fmt.Errorf("%w: 缺少首个 Run 的 model", ErrInvalidSpec)
  259: 	case strings.TrimSpace(spec.BehaviorRevision) == "":
  260: 		return nil, fmt.Errorf("%w: 缺少首个 Run 的 behavior revision", ErrInvalidSpec)
```

## internal/runstore/codegen.go:82-106

```text
   82: // 新 Run 记 EscalatedFrom(§11.2.1)。
   83: type StartRunSpec struct {
   84: 	Project              core.ProjectID
   85: 	Task                 string // TaskContract 身份(Git 持有意图,这里只存引用)
   86: 	Role                 string
   87: 	Runtime              string
   88: 	RuntimeVersion       string
   89: 	Model                string
   90: 	BehaviorRevision     string
   91: 	CostPlane            measurement.CostPlane // 成本归属面,与 internal/measurement 取值一致(PRD §30)
   92: 	BaseCommit           string
   93: 	TaskRevision         string // 执行所对 TaskContract 修订哈希
   94: 	VerificationRevision string // 执行所对 VerificationContract 修订哈希
   95: 	ContextPackHash      string
   96: 	WorkspaceID          string // 一次性 Git worktree(P0-05/P1-35)
   97: 	// AgentSessionID 是 Runtime 会话身份;会话未及建立即崩溃时为空。
   98: 	AgentSessionID string
   99: 	// RuntimeCapabilitySnapshot 是运行时声明能力快照(§10.3),必须是非空 JSON
  100: 	// 对象;原样存 JSONB,本包不解释其内容。
  101: 	RuntimeCapabilitySnapshot json.RawMessage
  102: 	// EscalatedFrom 是被升级终止的旧 Run id;0 = 非升级 Run。
  103: 	EscalatedFrom int64
  104: 	// StartedAt 为零值时取当下(UTC)。
  105: 	StartedAt time.Time
  106: }
```

## internal/runstore/codegen.go:111-125

```text
  111: 		return errors.New("runstore: 缺少 project")
  112: 	case spec.Task == "":
  113: 		return errors.New("runstore: 缺少 task")
  114: 	case spec.Role == "":
  115: 		return errors.New("runstore: 缺少 role")
  116: 	case spec.Runtime == "":
  117: 		return errors.New("runstore: 缺少 runtime")
  118: 	case spec.RuntimeVersion == "":
  119: 		return errors.New("runstore: 缺少 runtime_version")
  120: 	case spec.Model == "":
  121: 		return errors.New("runstore: 缺少 model")
  122: 	case spec.BehaviorRevision == "":
  123: 		return errors.New("runstore: 缺少 behavior_revision")
  124: 	case !spec.CostPlane.IsValid():
  125: 		return fmt.Errorf("runstore: 非法 cost_plane %q", spec.CostPlane)
```

## internal/runstore/codegen.go:131-143

```text
  131: 		return errors.New("runstore: 缺少 verification_revision")
  132: 	case spec.ContextPackHash == "":
  133: 		return errors.New("runstore: 缺少 context_pack_hash")
  134: 	case spec.WorkspaceID == "":
  135: 		return errors.New("runstore: 缺少 workspace_id")
  136: 	case len(spec.RuntimeCapabilitySnapshot) == 0 || !json.Valid(spec.RuntimeCapabilitySnapshot):
  137: 		return errors.New("runstore: runtime_capability_snapshot 必须是非空 JSON")
  138: 	case !isJSONObject(spec.RuntimeCapabilitySnapshot):
  139: 		return errors.New("runstore: runtime_capability_snapshot 必须是 JSON 对象")
  140: 	case spec.EscalatedFrom < 0:
  141: 		return errors.New("runstore: escalated_from 不能为负")
  142: 	default:
  143: 		return nil
```

## internal/runstore/codegen.go:153-185

```text
  153: 	startedAt := spec.StartedAt
  154: 	if startedAt.IsZero() {
  155: 		startedAt = time.Now()
  156: 	}
  157: 	session := pgtype.Text{}
  158: 	if spec.AgentSessionID != "" {
  159: 		session = pgtype.Text{String: spec.AgentSessionID, Valid: true}
  160: 	}
  161: 	var escalated pgtype.Int8
  162: 	if spec.EscalatedFrom != 0 {
  163: 		escalated = pgtype.Int8{Int64: spec.EscalatedFrom, Valid: true}
  164: 	}
  165: 	id, err := s.q.InsertCodegenRun(ctx, store.InsertCodegenRunParams{
  166: 		ProjectID:                 string(spec.Project),
  167: 		TaskID:                    spec.Task,
  168: 		Role:                      spec.Role,
  169: 		Runtime:                   spec.Runtime,
  170: 		RuntimeVersion:            spec.RuntimeVersion,
  171: 		Model:                     spec.Model,
  172: 		BehaviorRevision:          spec.BehaviorRevision,
  173: 		CostPlane:                 string(spec.CostPlane),
  174: 		BaseCommit:                spec.BaseCommit,
  175: 		TaskRevision:              spec.TaskRevision,
  176: 		VerificationRevision:      spec.VerificationRevision,
  177: 		ContextPackHash:           spec.ContextPackHash,
  178: 		WorkspaceID:               spec.WorkspaceID,
  179: 		AgentSessionID:            session,
  180: 		RuntimeCapabilitySnapshot: []byte(spec.RuntimeCapabilitySnapshot),
  181: 		EscalatedFrom:             escalated,
  182: 		StartedAt:                 tsPtr(startedAt),
  183: 	})
  184: 	if err != nil {
  185: 		return 0, fmt.Errorf("runstore: 写入 codegen_run(project=%s task=%s): %w", spec.Project, spec.Task, err)
```

## internal/runstore/codegen.go:284-294

```text
  284: type Run struct {
  285: 	ID                        int64
  286: 	Project                   core.ProjectID
  287: 	Task                      string
  288: 	Role                      string
  289: 	Runtime                   string
  290: 	RuntimeVersion            string
  291: 	Model                     string
  292: 	BehaviorRevision          string
  293: 	CostPlane                 measurement.CostPlane
  294: 	BaseCommit                string
```