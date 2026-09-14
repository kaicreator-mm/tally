# Current Tally Code Evidence

These excerpts come from the user-provided `tally.zip` baseline and are local authority.

## runtimes/codex/runtime.go:47-59

```text
   47: 
   48: // DescriptorName 是 CodexRuntime 的 runtime 标识。只用于展示/遥测/归因;
   49: // 任何控制流都不得按它分支(PRD §10.3)。
   50: const DescriptorName = "codex"
   51: 
   52: // DescriptorVersion 是本适配器(不是 CLI)的版本。CLI 版本由 Snapshot 经
   53: // `codex --version` 现场发现后记录,绝不硬编码(App 更新会改变它)。
   54: const DescriptorVersion = "1"
   55: 
   56: // 审批语义的文档化事实:现场复核(P1-37)实测 codex-cli 0.153.0 的
   57: // `codex exec` 拒绝 -a/--ask-for-approval(与探针记录 §2.1 的抄录不符,以
   58: // 实测表面为准,DECISION_LOCKS #13)。exec 是非交互一次性进程,无审批通道:
   59: // 本适配器不传任何审批旗标,也因此不声明 tool.approval(PRD §35 truthful)。
```

## runtimes/codex/runtime.go:82-92

```text
   82: 	)
   83: }
   84: 
   85: // Config 是 CodexRuntime 的构造配置。New 之后必须保持只读。
   86: type Config struct {
   87: 	// BinPath 显式指定 codex CLI 路径;空则按 ResolveBinaryPath 的观测顺序
   88: 	// 发现(env CODEX_CLI_PATH → config.toml → PATH)。测试注入替身二进制用。
   89: 	BinPath string
   90: 	// Sandbox 是 -s 的取值;空则 defaultSandbox。只接受 sandboxModes 中的
   91: 	// 探针记录枚举值,其它值在 Start 处拒绝。
   92: 	Sandbox string
```

## runtimes/codex/runtime.go:97-117

```text
   97: 	Env map[string]string
   98: 	// Now 返回事件时间戳;nil 则用 time.Now。测试可注入固定时钟。
   99: 	Now func() time.Time
  100: }
  101: 
  102: // RuntimeSnapshot 是 runtime 身份/版本/能力的一次可记录快照(P1-37 AC3):
  103: // 供上层把「跑的是哪个 codex、什么版本、声明了什么能力」落成审计记录。
  104: type RuntimeSnapshot struct {
  105: 	// Name 是 runtime 标识(DescriptorName)。
  106: 	Name string
  107: 	// AdapterVersion 是本适配器版本(DescriptorVersion)。
  108: 	AdapterVersion string
  109: 	// CLIVersion 是现场发现的 codex CLI 版本(如 "0.153.0"),非硬编码。
  110: 	CLIVersion string
  111: 	// BinaryPath 是现场解析出的 CLI 可执行文件路径。
  112: 	BinaryPath string
  113: 	// Capabilities 是声明能力(字典序)。声明由 conformance 另行验证。
  114: 	Capabilities []agentruntime.Capability
  115: }
  116: 
  117: // Runtime 是 CodexRuntime 的 agentruntime.Runtime 实现。
```

## runtimes/codex/runtime.go:132-161

```text
  132: // Descriptor 返回自述。Version 是适配器版本;CLI 版本属现场事实,由
  133: // Snapshot 发现并记录,不进 Descriptor(避免 Descriptor 触发进程)。
  134: func (r *Runtime) Descriptor() agentruntime.RuntimeDescriptor {
  135: 	return agentruntime.RuntimeDescriptor{
  136: 		Name:         DescriptorName,
  137: 		Version:      DescriptorVersion,
  138: 		Capabilities: Capabilities(),
  139: 	}
  140: }
  141: 
  142: // Snapshot 确保二进制与版本已发现,返回可记录的 RuntimeSnapshot(AC3)。
  143: func (r *Runtime) Snapshot(ctx context.Context) (RuntimeSnapshot, error) {
  144: 	bin, err := r.ensureBinary()
  145: 	if err != nil {
  146: 		return RuntimeSnapshot{}, err
  147: 	}
  148: 	ver, err := r.ensureVersion(ctx, bin)
  149: 	if err != nil {
  150: 		return RuntimeSnapshot{}, err
  151: 	}
  152: 	return RuntimeSnapshot{
  153: 		Name:           DescriptorName,
  154: 		AdapterVersion: DescriptorVersion,
  155: 		CLIVersion:     ver,
  156: 		BinaryPath:     bin,
  157: 		Capabilities:   Capabilities().Sorted(),
  158: 	}, nil
  159: }
  160: 
  161: // Start 开启会话。统一走 agentruntime.CheckStart:结构校验 + 能力协商
```

## runtimes/codex/runtime.go:246-256

```text
  246: 		r.mu.Lock()
  247: 		r.binPath = r.cfg.BinPath
  248: 		r.mu.Unlock()
  249: 		return r.cfg.BinPath, nil
  250: 	}
  251: 	bin, err := ResolveBinaryPath()
  252: 	if err != nil {
  253: 		return "", err
  254: 	}
  255: 	r.mu.Lock()
  256: 	r.binPath = bin
```

## internal/runstore/codegen.go:83-103

```text
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
```

## internal/runstore/codegen.go:113-125

```text
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

## internal/runstore/codegen.go:153-184

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
```

## internal/runstore/codegen.go:285-304

```text
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
  295: 	TaskRevision              string
  296: 	VerificationRevision      string
  297: 	ContextPackHash           string
  298: 	WorkspaceID               string
  299: 	AgentSessionID            string // 空 = 会话未及建立即崩溃
  300: 	StartedAt                 time.Time
  301: 	FinishedAt                time.Time // 零值 = 未结束(running)
  302: 	Status                    RunStatus
  303: 	Iterations                int32
  304: 	EscalatedFrom             int64  // 0 = 非升级 Run
```

## internal/runstore/codegen.go:344-354

```text
  344: 		ID:                        row.ID,
  345: 		Project:                   core.ProjectID(row.ProjectID),
  346: 		Task:                      row.TaskID,
  347: 		Role:                      row.Role,
  348: 		Runtime:                   row.Runtime,
  349: 		RuntimeVersion:            row.RuntimeVersion,
  350: 		Model:                     row.Model,
  351: 		BehaviorRevision:          row.BehaviorRevision,
  352: 		CostPlane:                 measurement.CostPlane(row.CostPlane),
  353: 		BaseCommit:                row.BaseCommit,
  354: 		TaskRevision:              row.TaskRevision,
```