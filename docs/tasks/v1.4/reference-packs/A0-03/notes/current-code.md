# Current Tally Code Evidence

These excerpts come from the user-provided `tally.zip` baseline and are local authority.

## internal/verification/holdout_executor.go:4-43

```text
    4: // Validation、§14.3、§8.4.1、§8.7、INV-VER-05,决策锁 11/12)。
    5: //
    6: // 职责:把 held-out 证据只注入 Tally 控制的最终验证 —— 对一个 immutable
    7: // ChangeSet 与冻结的 execution context(§8.7.1 四元组)执行冻结契约中的
    8: // held_out binding,产出:
    9: //   - Control/Review Authority 侧的判定事实(HoldoutVerdictRecord):FAIL
   10: //     可记录,并经 DeriveFunctionalCoverage 阻断 VERIFIED(AC3);
   11: //   - Implementation Agent 可见通道的唯一合法投影(RedactedHoldoutOutcome):
   12: //     结构上不存在 path/selector/assertion/expected/actual/ref/digest/verdict
   13: //     字段,按 §8.4.3 只暴露 acceptance 语义与 evidence type(AC2)。
   14: //
   15: // 机械保证(P1-29 AC 映射):
   16: //   - AC1:内容只经 holdout.VerificationReader 窄能力进入本包,再只以内存字节
   17: //     传给注入的 Tally 控制执行边界(HoldoutRunFunc);本文件不导入任何文件系统
   18: //     /进程能力(test/arch 层面另有守卫),不存在 workspace 可读 held-out 内容
   19: //     的路径。执行边界收到的内容也不得落进实现权威可见通道 —— 那是 §8.6.1
   20: //     exposure accounting 的管辖,本包不提供任何这样的通道。
   21: //   - AC2:RedactHoldoutRecord 是唯一的投影出口,allowlist 构造;
   22: //     HoldoutRunOutput 只返回结构化 status + 输出摘要,原始失败输出(§14.3:
   23: //     不得进入实现侧事件流)留在 Tally 控制的边界内,本包类型层不接收。
   24: //   - AC3:FAIL 落成 append-only 判定事实;EvidenceRecord() 投影进功能覆盖
   25: //     推导,valid + applicable 的 FAIL 使 acceptance 保持未覆盖 → 不能 VERIFIED。
   26: //   - AC4:Execute 拒绝 candidate 不匹配的 execution context;ChangeSet 身份由
   27: //     内容推导(P1-05),可执行代码一变即新 ChangeSet ID → 旧 context 被拒、
   28: //     新 context 身份哈希必变 → 任何可执行变更都需要新 ChangeSet/新 verdict
   29: //     (§8.7.1)。
   30: //   - AC5:每个 binding 在本执行器上一次性消耗(verdict 与否);不提供 rerun/
   31: //     retry/recovery/probe 之类的任何 API(决策锁 12:P1 只有隔离 + 最终执行 +
   32: //     exposure accounting,自动恢复梯子属 P2)。重做只能走权威侧流程:新
   33: //     VerificationContract revision 的 fresh binding(§8.6.3),并另行接受
   34: //     selection-exposure 会计(internal/evidence)。
   35: //
   36: // 边界(P1):
   37: //   - 本包不执行进程、不落盘、不持有 Store:真实执行由控制面把
   38: //     holdout.VerificationReader + HoldoutRunFunc 接到一次性验证 workspace 与
   39: //     结构化命令 runner(internal/command CommandSpec 语义)完成;
   40: //   - visible binding 不属于本注入点(增量验证归 visible 执行器任务);
   41: //   - 不推导 effective independence、不记录 exposure fact:那些归
   42: //     internal/evidence;本执行器只是 INV-VER-05 的 held-out 注入点。
   43: 
```

## internal/verification/holdout_executor.go:51-61

```text
   51: 	"tally/internal/holdout"
   52: )
   53: 
   54: // 哨兵错误:调用方用 errors.Is 判因,不做字符串匹配。
   55: var (
   56: 	// ErrNilDependency 构造 HoldoutExecutor 时缺少必需依赖(nil 契约/candidate/
   57: 	// reader/执行边界)。保守失败,不构造半成品执行器。
   58: 	ErrNilDependency = errors.New("verification: holdout executor 缺少必需依赖")
   59: 	// ErrNoHeldOutBinding 契约不含任何 held_out binding:本执行器只为 held-out
   60: 	// 注入而存在,没有注入对象说明接线错误。
   61: 	ErrNoHeldOutBinding = errors.New("verification: 契约不含 held_out binding,holdout executor 无可注入对象")
```

## internal/verification/holdout_executor.go:75-128

```text
   75: 	// 不一致(§8.7.1/§15.4:可执行代码变更产生新 ChangeSet,旧 verdict 不适用)。
   76: 	ErrCandidateMismatch = errors.New("verification: execution context 的 candidate 与给定 ChangeSet 不一致")
   77: 	// ErrProtocolMismatch execution context 携带的执行协议与契约为该 binding
   78: 	// 冻结的协议不一致(§8.7.2:协议必须在首次执行前随契约冻结,不能现编)。
   79: 	ErrProtocolMismatch = errors.New("verification: execution context 的执行协议与契约冻结协议不一致")
   80: 	// ErrBindingAlreadyExecuted 该 binding 的 held-out 最终执行已消耗。P1 最终
   81: 	// 验证一次性:无论成功、失败还是基础设施错误,都不提供自动重试/恢复
   82: 	// (AC5,决策锁 12)。
   83: 	ErrBindingAlreadyExecuted = errors.New("verification: 该 binding 的 held-out 最终执行已消耗(P1 无自动恢复)")
   84: 	// ErrHoldoutIntegrity held-out 内容完整性校验失败(Read 内容与 Store 元数据
   85: 	// 摘要不符,或与冻结 protection.baseline_hash 不符,INV-VER-03/§14.2)。
   86: 	ErrHoldoutIntegrity = errors.New("verification: held-out 内容完整性校验失败")
   87: 	// ErrInvalidRunOutput 执行边界返回的观察不合法(status 非 pass|fail,或输出
   88: 	// 摘要不是 sha256:<hex> 形态)。没有合法观察就不能记录 repetition。
   89: 	ErrInvalidRunOutput = errors.New("verification: 执行边界返回的观察不合法")
   90: )
   91: 
   92: // HoldoutRunRequest 是 Tally 控制的最终验证执行边界的一次输入。
   93: //
   94: // Content 是 held-out 证据内容,只以内存字节存在:边界把它注入一次性验证
   95: // workspace(如临时 materialize 后执行冻结命令),绝不写入实现权威可达的
   96: // 任何位置;原始输出也留在边界内(Control/Review Authority 侧),只以摘要
   97: // 形态返回。Command/Seed 来自冻结执行协议,边界不得改写语义。
   98: type HoldoutRunRequest struct {
   99: 	BindingID string       // 正在注入的 binding id(契约标识,非敏感)
  100: 	Command   *CommandSpec // 冻结协议命令(结构化 argv/cwd/env/timeout)
  101: 	Seed      string       // 冻结协议随机 seed
  102: 	Content   []byte       // held-out 内容(内存字节;边界用后即弃)
  103: }
  104: 
  105: // HoldoutRunOutput 是执行边界对一次 repetition 的结构化观察。原始 stdout/
  106: // stderr 不回到本包(§14.3);OutputDigest 供 Control/Review 侧追溯。
  107: type HoldoutRunOutput struct {
  108: 	Status       EvidenceStatus // pass | fail
  109: 	OutputDigest string         // 原始输出的 sha256:<hex> 摘要
  110: }
  111: 
  112: // HoldoutRunFunc 是 Tally 控制的最终验证执行边界:把请求中的 held-out 内容
  113: // 注入一次性验证环境并执行冻结命令。只有控制面持有并接线该函数(实现权威
  114: // 侧的类型签名里不存在它);它属于验证权威通道,其内部对原始输出的留存归
  115: // §14.3 的 VerificationRun 事实存储管辖。
  116: type HoldoutRunFunc func(ctx context.Context, req HoldoutRunRequest) (HoldoutRunOutput, error)
  117: 
  118: // HoldoutVerdictRecord 是一条 held-out binding 对 immutable ChangeSet 的最终
  119: // 验证判定事实(§14.2:结果只进入允许的 Control/Review Authority)。它是
  120: // append-only 事实的值投影:记录后不可改写;后续 PASS 不抹掉历史 FAIL
  121: // (§8.8.1)。它不携带 held-out 内容,也不携带原始失败输出;ContentDigest 与
  122: // OutputDigests 是完整性/追溯锚点,只随本记录进入 Control/Review 通道。
  123: type HoldoutVerdictRecord struct {
  124: 	TaskID           string
  125: 	ContractRevision int
  126: 	ContractHash     string
  127: 	AcceptanceID     string
  128: 	BindingID        string
```

## internal/verification/holdout_executor.go:138-222

```text
  138: }
  139: 
  140: // EvidenceRecord 把判定投影为功能覆盖推导(DeriveFunctionalCoverage)的输入:
  141: // valid + applicable(Revision == 冻结契约 revision)。FAIL 由此阻断 VERIFIED
  142: // (AC3);历史 FAIL 不被后续 PASS 抹掉由覆盖推导与 append-only 事实保证。
  143: func (r HoldoutVerdictRecord) EvidenceRecord() EvidenceRecord {
  144: 	return EvidenceRecord{
  145: 		BindingID: r.BindingID,
  146: 		Revision:  r.ContractRevision,
  147: 		Status:    r.Status,
  148: 		Validity:  EvidenceValidityValid,
  149: 	}
  150: }
  151: 
  152: // RedactedHoldoutOutcome 是进入 Implementation Agent 可见通道的唯一合法形态
  153: // (AC2)。字段是 §8.4.3 allowlist:acceptance 语义(id)与 evidence type 之外
  154: // 一无所有 —— 结构上不存在 path/selector/assertion/expected/actual/ref/digest,
  155: // 也不存在 verdict(§8.4.1:可识别 verdict 不得进入实现侧通道;最终 VERIFIED
  156: // 状态是 State Engine 的派生生命周期状态,不是本投影)。
  157: type RedactedHoldoutOutcome struct {
  158: 	TaskID           string
  159: 	ContractRevision int
  160: 	AcceptanceID     string
  161: 	EvidenceType     EvidenceType
  162: 	CompletedAt      time.Time
  163: }
  164: 
  165: // RedactHoldoutRecord 是 Control/Review 判定事实到实现侧可见投影的唯一出口:
  166: // allowlist 逐字段拷贝,任何新增字段都必须显式进入本函数与 allowlist 测试,
  167: // 不能靠"顺带"漏进实现侧通道(AC2)。
  168: func RedactHoldoutRecord(r HoldoutVerdictRecord) RedactedHoldoutOutcome {
  169: 	return RedactedHoldoutOutcome{
  170: 		TaskID:           r.TaskID,
  171: 		ContractRevision: r.ContractRevision,
  172: 		AcceptanceID:     r.AcceptanceID,
  173: 		EvidenceType:     r.EvidenceType,
  174: 		CompletedAt:      r.CompletedAt,
  175: 	}
  176: }
  177: 
  178: // HoldoutExecutor 把冻结契约中的 held_out binding 注入 Tally 控制的最终验证:
  179: // 每条 binding 对一个 immutable ChangeSet、按冻结 execution context 执行一次
  180: // (含冻结 repetitions)。实例按(契约, ChangeSet)构造;一个 binding 至多
  181: // 消耗一次注入(AC5)。
  182: type HoldoutExecutor struct {
  183: 	contract  *Frozen
  184: 	candidate *changeset.ChangeSet
  185: 	// reader 是 Holdout Evidence Store 的窄读能力(holdout.VerificationReader):
  186: 	// 只有验证权威侧代码持有;实现权威侧组件的类型签名里没有它(AC1 的类型层
  187: 	// 落点,与 internal/holdout 的能力拆分一致)。
  188: 	reader holdout.VerificationReader
  189: 	// run 是 Tally 控制的执行边界;内容只经它进入一次性验证环境。
  190: 	run HoldoutRunFunc
  191: 
  192: 	records   []HoldoutVerdictRecord // 已产出的判定事实,append-only
  193: 	attempted map[string]bool        // binding id → 注入已消耗(无论结果)
  194: }
  195: 
  196: // NewHoldoutExecutor 构造执行器。校验:
  197: //   - 契约、candidate、reader、执行边界非 nil(AC1:没有 reader 能力就没有
  198: //     内容来源;没有 Tally 控制边界就没有合法执行);
  199: //   - 契约已声明 INV-VER-05 runtime 能力(纵深防御;Freeze 已强制);
  200: //   - 契约至少含一条 held_out binding(本执行器只做 held-out 注入)。
  201: //
  202: // 任何校验失败都不产生执行器。
  203: func NewHoldoutExecutor(contract *Frozen, candidate *changeset.ChangeSet, reader holdout.VerificationReader, run HoldoutRunFunc) (*HoldoutExecutor, error) {
  204: 	switch {
  205: 	case contract == nil:
  206: 		return nil, fmt.Errorf("%w: 冻结 VerificationContract 为 nil", ErrNilDependency)
  207: 	case candidate == nil:
  208: 		return nil, fmt.Errorf("%w: immutable ChangeSet 为 nil", ErrNilDependency)
  209: 	case reader == nil:
  210: 		return nil, fmt.Errorf("%w: holdout VerificationReader 能力为 nil(AC1:内容只能经窄能力进入)", ErrNilDependency)
  211: 	case run == nil:
  212: 		return nil, fmt.Errorf("%w: Tally 控制的执行边界为 nil", ErrNilDependency)
  213: 	}
  214: 	if r := contract.RuntimeRequirements(); !r.SessionIsolation || !r.HeldOutUnreachable || r.ConformanceClass == "" {
  215: 		return nil, fmt.Errorf("%w: 当前 %+v", ErrRuntimeRequirementsUnmet, r)
  216: 	}
  217: 	e := &HoldoutExecutor{
  218: 		contract:  contract,
  219: 		candidate: candidate,
  220: 		reader:    reader,
  221: 		run:       run,
  222: 		attempted: make(map[string]bool),
```

## internal/verification/holdout_executor.go:228-238

```text
  228: }
  229: 
  230: // HeldOutBindingIDs 返回契约中全部 held_out binding id(契约声明顺序),
  231: // 供控制面核对最终验证必须注入的完整集合(§14.2:held-out execution 是
  232: // VERIFIED 前的必检项)。
  233: func (e *HoldoutExecutor) HeldOutBindingIDs() []string {
  234: 	if e == nil || e.contract == nil {
  235: 		return nil
  236: 	}
  237: 	var ids []string
  238: 	c := e.contract.Contract()
```

## internal/verification/coverage_independence.go:20-35

```text
   20: // 规则来源:
   21: //   - §8.5/§7.3:none 不要求任何独立证据(AC1);source 要求至少一条 applicable
   22: //     evidence 的 effective_source_independence = true(AC2);information 要求
   23: //     至少一条 applicable evidence 同时满足 effective source independence 与
   24: //     effective information independence(AC3)。两个维度必须在同一条 evidence
   25: //     上同时成立:Information Independence 永远不能补救 Source Dependence,
   26: //     不允许跨 evidence OR;
   27: //   - §8.5.2:effective coverage 在执行后计算,只消费 applicable evidence set
   28: //     —— 即 §8.8.2 意义下适用于当前契约 revision、且在求值对象 candidate 上
   29: //     存在 valid 且可用判定记录的 evidence。运行期 exposure/selection
   30: //     contamination 使 planned ✓ → effective ✗ 在此被推导暴露;Independence
   31: //     Waiver 是后续 policy 层的独立风险接受标注,本推导没有 waiver/policy
   32: //     输入(compromised 是 observed fact,不能被重新标注为 independent,§8.10);
   33: //   - §8.2.2 / §8.6:per-binding 的 effective source independence(P1-17)与
   34: //     effective information independence(P1-18)直接复用 internal/evidence 的
   35: //     确定性推导。binding 的 declared basis 与 authoring_mode 以冻结契约为权威
```

## internal/verification/coverage_independence.go:50-62

```text
   50: //   - planned coverage(§8.5.1)在 freeze 校验中(types.go validatePlannedCoverage);
   51: //     本文件只做执行后的 effective coverage;
   52: //   - 不设置 lifecycle 状态,不解释 qualification(§8.3)与功能 PASS/FAIL;
   53: //     只产出供 State Engine 消费的覆盖事实。
   54: 
   55: // IndependenceContext — effective coverage 的求值上下文(§8.5.2:coverage 是
   56: // coverage(acceptance, changeset),相对候选的实际实现权威 provenance 推导)。
   57: type IndependenceContext struct {
   58: 	// Project — holdout 消费键(evidence.BindingRef)的 project 分量。
   59: 	Project core.ProjectID
   60: 	// Verification — 被求值冻结契约的内容身份(即 Frozen.Hash()),BindingRef
   61: 	// 的 revision 分量;与契约不一致时判定/exposure 永远无法匹配,显式报错。
   62: 	Verification evidence.VerificationRevision
```

## internal/verification/coverage_independence.go:72-109

```text
   72: 	Implementation evidence.ImplementationProvenance
   73: 	// Precommit — 该候选的机械可验证选择 precommit(§8.6.2);零值 = 无。
   74: 	Precommit evidence.SelectionPrecommit
   75: }
   76: 
   77: // IndependenceEvidence — 一条契约 binding 的执行后独立性求值输入。binding 的
   78: // declared source basis 与 authoring_mode 以冻结契约为权威,不在此重复声明
   79: // (INV-VER-01/02);调用方只提供活动接触 attestation 与 append-only 事实投影。
   80: type IndependenceEvidence struct {
   81: 	// BindingID — 契约中的 binding id。
   82: 	BindingID string
   83: 	// Revision — 该求值输入针对的契约 revision(§8.8.2:evidence 只适用于它
   84: 	// 绑定的 revision;其他 revision 的输入是历史事实,不参与当前推导)。
   85: 	Revision int
   86: 	// CandidateBlind — post_hoc authoring 的 candidate-blind attestation
   87: 	// (§8.2.1;仅 post_hoc 被消费,pre_commit 的 candidate 缺席由 mode 自证)。
   88: 	CandidateBlind bool
   89: 	// AuthoredInLineage — 该证据作为当前实现决策 lineage 的一部分被首次创作
   90: 	// (§8.2.2 活动接触;与 actor 身份无关)。
   91: 	AuthoredInLineage bool
   92: 	// ChangedInLineage — 既存证据内容在当前实现决策 lineage 内被修改(§8.2.2)。
   93: 	ChangedInLineage bool
   94: 	// Verdicts — 该 binding 的判定记录投影(全部 candidate;推导按 BindingRef +
   95: 	// 当前 candidate 选择)。visible binding 同样需要:applicable evidence set
   96: 	// 的成员资格由当前 candidate 上 valid 且可用的判定记录确立。
   97: 	Verdicts []evidence.VerdictRecord
   98: 	// Exposures — 全部 exposure facts(§8.6.1);推导按 BindingRef 选择,其他
   99: 	// binding 的 exposure 永远不会跨 binding 泄漏。
  100: 	Exposures []evidence.Exposure
  101: }
  102: 
  103: // BindingIndependence — 单条 binding 的独立性推导明细(INV-STATE-02 可追溯)。
  104: type BindingIndependence struct {
  105: 	BindingID string
  106: 	// Applicable — 该 binding 属于当前 applicable evidence set:在求值对象
  107: 	// candidate 上存在 valid 且可用的判定记录(§8.5.2/§8.8.2)。
  108: 	Applicable bool
  109: 	// SourceIndependent — effective_source_independence(§8.2.2,P1-17 推导)。
```

## internal/verification/coverage_independence.go:115-186

```text
  115: 	// none 时任何 applicable evidence 均视为满足(无独立性约束);source 时
  116: 	// 要求 Applicable && SourceIndependent;information 时要求三者同时成立。
  117: 	Qualifies bool
  118: }
  119: 
  120: // AcceptanceIndependence — 单条 Acceptance Clause 的 independence 覆盖推导结果。
  121: type AcceptanceIndependence struct {
  122: 	AcceptanceID string
  123: 	// Required — 归一化后的独立性要求(空声明 = none,PRD §7.3)。
  124: 	Required IndependenceRequired
  125: 	// Covered — none 恒为 true(不要求任何独立证据,AC1);source/information
  126: 	// 要求至少一条 qualifying 的 applicable evidence。
  127: 	Covered bool
  128: 	// SatisfiedBy — 满足独立性要求的 binding id(契约声明顺序);none 恒为空
  129: 	// (没有独立性要求,也就没有任何 binding「提供独立性」)。
  130: 	SatisfiedBy []string
  131: 	// Bindings — 逐 binding 推导明细(契约声明顺序)。
  132: 	Bindings []BindingIndependence
  133: }
  134: 
  135: // IndependenceCoverageReport — 整份冻结契约的 effective independence coverage。
  136: type IndependenceCoverageReport struct {
  137: 	TaskID           string
  138: 	ContractRevision int
  139: 	ContractHash     string
  140: 	Acceptances      []AcceptanceIndependence
  141: 	// Unmatched — 引用了契约外 binding id 的求值输入(supplementary 或调用方
  142: 	// 错误),不参与推导;按首次出现顺序去重列出(INV-STATE-02)。
  143: 	Unmatched []string
  144: 	// Complete — 所有 acceptance 的 independence_required 均被满足。false 表示
  145: 	// 存在 planned ✓ → effective ✗ 的 acceptance;Waiver 判定归 policy 层,
  146: 	// 不在本推导。
  147: 	Complete bool
  148: }
  149: 
  150: // DeriveIndependenceCoverage 对冻结契约推导 Acceptance 级 effective independence
  151: // coverage(P1-20)。逐 Acceptance Clause 判定其 independence_required 是否由
  152: // 当前 applicable evidence set 满足;per-binding 的两个独立性维度直接复用
  153: // internal/evidence 的确定性推导,本函数只做 Acceptance 级组合。
  154: //
  155: // f、ctx 为 nil/不完整,或求值输入形态非法时返回 error;其余一切结论都在报告中
  156: // (包括不覆盖),不做静默修正。
  157: func DeriveIndependenceCoverage(f *Frozen, ctx IndependenceContext, records []IndependenceEvidence) (IndependenceCoverageReport, error) {
  158: 	if f == nil {
  159: 		return IndependenceCoverageReport{}, fmt.Errorf("verification: contract 为 nil,无法推导 independence coverage")
  160: 	}
  161: 	if ctx.Project == "" {
  162: 		return IndependenceCoverageReport{}, fmt.Errorf("verification: independence 求值上下文缺少 project(holdout 消费键分量)")
  163: 	}
  164: 	if ctx.Verification == "" {
  165: 		return IndependenceCoverageReport{}, fmt.Errorf("verification: independence 求值上下文缺少 verification revision 身份")
  166: 	}
  167: 	// BindingRef 的 revision 分量就是冻结契约的内容哈希(internal/evidence
  168: 	// BindingRef 契约);与被求值契约不一致时判定/exposure 永远无法命中,显式
  169: 	// 报错而不是静默推出全不覆盖。
  170: 	if ctx.Verification != evidence.VerificationRevision(f.Hash()) {
  171: 		return IndependenceCoverageReport{}, fmt.Errorf("verification: 求值上下文 verification 身份 %s 与被求值契约哈希 %s 不一致", ctx.Verification, f.Hash())
  172: 	}
  173: 	if ctx.ChangeSet == "" {
  174: 		return IndependenceCoverageReport{}, fmt.Errorf("verification: independence 求值上下文缺少候选 ChangeSet(§8.5.2:coverage 是 coverage(acceptance, changeset))")
  175: 	}
  176: 
  177: 	rep := IndependenceCoverageReport{
  178: 		TaskID:           f.TaskID(),
  179: 		ContractRevision: f.Revision(),
  180: 		ContractHash:     f.Hash(),
  181: 		Acceptances:      make([]AcceptanceIndependence, len(f.c.Acceptances)),
  182: 		Complete:         true,
  183: 	}
  184: 
  185: 	// 契约 binding id 集合。Contract.Validate 保证 binding id 全局唯一且跨
  186: 	// acceptance 不重复,因此按 id 分派无归属歧义。
```

## internal/verification/coverage_independence.go:193-203

```text
  193: 	}
  194: 
  195: 	// 分派求值输入:契约外 id 列入 Unmatched;历史 revision 的输入按 §8.8.2
  196: 	// 跳过;同 binding 的重复输入是调用方错误(authoring provenance 每 binding
  197: 	// 唯一),显式报错而不是静默合并。
  198: 	inputs := map[string]*IndependenceEvidence{}
  199: 	unmatchedSeen := map[string]bool{}
  200: 	for k := range records {
  201: 		r := &records[k]
  202: 		if strings.TrimSpace(r.BindingID) == "" {
  203: 			return IndependenceCoverageReport{}, fmt.Errorf("verification: independence 求值输入缺少 binding id")
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