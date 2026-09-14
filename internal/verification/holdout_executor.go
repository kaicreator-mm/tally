package verification

// 本文件实现 P1-29:Held-out final validation executor(PRD §14.2 Final
// Validation、§14.3、§8.4.1、§8.7、INV-VER-05,决策锁 11/12)。
//
// 职责:把 held-out 证据只注入 Tally 控制的最终验证 —— 对一个 immutable
// ChangeSet 与冻结的 execution context(§8.7.1 四元组)执行冻结契约中的
// held_out binding,产出:
//   - Control/Review Authority 侧的判定事实(HoldoutVerdictRecord):FAIL
//     可记录,并经 DeriveFunctionalCoverage 阻断 VERIFIED(AC3);
//   - Implementation Agent 可见通道的唯一合法投影(RedactedHoldoutOutcome):
//     结构上不存在 path/selector/assertion/expected/actual/ref/digest/verdict
//     字段,按 §8.4.3 只暴露 acceptance 语义与 evidence type(AC2)。
//
// 机械保证(P1-29 AC 映射):
//   - AC1:内容只经 holdout.VerificationReader 窄能力进入本包,再只以内存字节
//     传给注入的 Tally 控制执行边界(HoldoutRunFunc);本文件不导入任何文件系统
//     /进程能力(test/arch 层面另有守卫),不存在 workspace 可读 held-out 内容
//     的路径。执行边界收到的内容也不得落进实现权威可见通道 —— 那是 §8.6.1
//     exposure accounting 的管辖,本包不提供任何这样的通道。
//   - AC2:RedactHoldoutRecord 是唯一的投影出口,allowlist 构造;
//     HoldoutRunOutput 只返回结构化 status + 输出摘要,原始失败输出(§14.3:
//     不得进入实现侧事件流)留在 Tally 控制的边界内,本包类型层不接收。
//   - AC3:FAIL 落成 append-only 判定事实;EvidenceRecord() 投影进功能覆盖
//     推导,valid + applicable 的 FAIL 使 acceptance 保持未覆盖 → 不能 VERIFIED。
//   - AC4:Execute 拒绝 candidate 不匹配的 execution context;ChangeSet 身份由
//     内容推导(P1-05),可执行代码一变即新 ChangeSet ID → 旧 context 被拒、
//     新 context 身份哈希必变 → 任何可执行变更都需要新 ChangeSet/新 verdict
//     (§8.7.1)。
//   - AC5:每个 binding 在本执行器上一次性消耗(verdict 与否);不提供 rerun/
//     retry/recovery/probe 之类的任何 API(决策锁 12:P1 只有隔离 + 最终执行 +
//     exposure accounting,自动恢复梯子属 P2)。重做只能走权威侧流程:新
//     VerificationContract revision 的 fresh binding(§8.6.3),并另行接受
//     selection-exposure 会计(internal/evidence)。
//
// 边界(P1):
//   - 本包不执行进程、不落盘、不持有 Store:真实执行由控制面把
//     holdout.VerificationReader + HoldoutRunFunc 接到一次性验证 workspace 与
//     结构化命令 runner(internal/command CommandSpec 语义)完成;
//   - visible binding 不属于本注入点(增量验证归 visible 执行器任务);
//   - 不推导 effective independence、不记录 exposure fact:那些归
//     internal/evidence;本执行器只是 INV-VER-05 的 held-out 注入点。

import (
	"context"
	"errors"
	"fmt"
	"time"

	"tally/internal/changeset"
	"tally/internal/holdout"
)

// 哨兵错误:调用方用 errors.Is 判因,不做字符串匹配。
var (
	// ErrNilDependency 构造 HoldoutExecutor 时缺少必需依赖(nil 契约/candidate/
	// reader/执行边界)。保守失败,不构造半成品执行器。
	ErrNilDependency = errors.New("verification: holdout executor 缺少必需依赖")
	// ErrNoHeldOutBinding 契约不含任何 held_out binding:本执行器只为 held-out
	// 注入而存在,没有注入对象说明接线错误。
	ErrNoHeldOutBinding = errors.New("verification: 契约不含 held_out binding,holdout executor 无可注入对象")
	// ErrRuntimeRequirementsUnmet 冻结契约未声明 INV-VER-05 所需的实现侧 runtime
	// 能力(session_isolation + held_out_unreachable + conformance_class)。
	// Freeze 对含 held_out binding 的契约已强制;此处重复检查是纵深防御。
	ErrRuntimeRequirementsUnmet = errors.New("verification: 契约未声明 INV-VER-05 runtime 能力(session_isolation/held_out_unreachable/conformance_class)")
	// ErrBindingNotFound bindingID 不在冻结契约中。
	ErrBindingNotFound = errors.New("verification: binding 不在冻结契约中")
	// ErrBindingNotHeldOut 目标 binding 不是 held_out:本执行器只注入 held-out
	// 证据,visible binding 属于增量验证通道。
	ErrBindingNotHeldOut = errors.New("verification: binding 不是 held_out")
	// ErrExecutionContextMismatch execution context 与冻结契约 revision/哈希或
	// 目标 binding 不一致(§8.7.1:verdict 绑定契约 revision + binding)。
	ErrExecutionContextMismatch = errors.New("verification: execution context 与冻结契约/binding 不一致")
	// ErrCandidateMismatch execution context 绑定的 candidate 与给定 ChangeSet
	// 不一致(§8.7.1/§15.4:可执行代码变更产生新 ChangeSet,旧 verdict 不适用)。
	ErrCandidateMismatch = errors.New("verification: execution context 的 candidate 与给定 ChangeSet 不一致")
	// ErrProtocolMismatch execution context 携带的执行协议与契约为该 binding
	// 冻结的协议不一致(§8.7.2:协议必须在首次执行前随契约冻结,不能现编)。
	ErrProtocolMismatch = errors.New("verification: execution context 的执行协议与契约冻结协议不一致")
	// ErrBindingAlreadyExecuted 该 binding 的 held-out 最终执行已消耗。P1 最终
	// 验证一次性:无论成功、失败还是基础设施错误,都不提供自动重试/恢复
	// (AC5,决策锁 12)。
	ErrBindingAlreadyExecuted = errors.New("verification: 该 binding 的 held-out 最终执行已消耗(P1 无自动恢复)")
	// ErrHoldoutIntegrity held-out 内容完整性校验失败(Read 内容与 Store 元数据
	// 摘要不符,或与冻结 protection.baseline_hash 不符,INV-VER-03/§14.2)。
	ErrHoldoutIntegrity = errors.New("verification: held-out 内容完整性校验失败")
	// ErrInvalidRunOutput 执行边界返回的观察不合法(status 非 pass|fail,或输出
	// 摘要不是 sha256:<hex> 形态)。没有合法观察就不能记录 repetition。
	ErrInvalidRunOutput = errors.New("verification: 执行边界返回的观察不合法")
)

// HoldoutRunRequest 是 Tally 控制的最终验证执行边界的一次输入。
//
// Content 是 held-out 证据内容,只以内存字节存在:边界把它注入一次性验证
// workspace(如临时 materialize 后执行冻结命令),绝不写入实现权威可达的
// 任何位置;原始输出也留在边界内(Control/Review Authority 侧),只以摘要
// 形态返回。Command/Seed 来自冻结执行协议,边界不得改写语义。
type HoldoutRunRequest struct {
	BindingID string       // 正在注入的 binding id(契约标识,非敏感)
	Command   *CommandSpec // 冻结协议命令(结构化 argv/cwd/env/timeout)
	Seed      string       // 冻结协议随机 seed
	Content   []byte       // held-out 内容(内存字节;边界用后即弃)
}

// HoldoutRunOutput 是执行边界对一次 repetition 的结构化观察。原始 stdout/
// stderr 不回到本包(§14.3);OutputDigest 供 Control/Review 侧追溯。
type HoldoutRunOutput struct {
	Status       EvidenceStatus // pass | fail
	OutputDigest string         // 原始输出的 sha256:<hex> 摘要
}

// HoldoutRunFunc 是 Tally 控制的最终验证执行边界:把请求中的 held-out 内容
// 注入一次性验证环境并执行冻结命令。只有控制面持有并接线该函数(实现权威
// 侧的类型签名里不存在它);它属于验证权威通道,其内部对原始输出的留存归
// §14.3 的 VerificationRun 事实存储管辖。
type HoldoutRunFunc func(ctx context.Context, req HoldoutRunRequest) (HoldoutRunOutput, error)

// HoldoutVerdictRecord 是一条 held-out binding 对 immutable ChangeSet 的最终
// 验证判定事实(§14.2:结果只进入允许的 Control/Review Authority)。它是
// append-only 事实的值投影:记录后不可改写;后续 PASS 不抹掉历史 FAIL
// (§8.8.1)。它不携带 held-out 内容,也不携带原始失败输出;ContentDigest 与
// OutputDigests 是完整性/追溯锚点,只随本记录进入 Control/Review 通道。
type HoldoutVerdictRecord struct {
	TaskID           string
	ContractRevision int
	ContractHash     string
	AcceptanceID     string
	BindingID        string
	CandidateID      string // immutable ChangeSet id(verdict 绑定对象,§8.7.1)
	ContextHash      string // execution context 四元组身份哈希
	EvidenceType     EvidenceType
	ContentDigest    string // held-out 内容摘要(sha256:<hex>)
	Status           EvidenceStatus
	Repetitions      int
	Failures         int
	OutputDigests    []string // 各 repetition 输出摘要,执行顺序
	CompletedAt      time.Time
}

// EvidenceRecord 把判定投影为功能覆盖推导(DeriveFunctionalCoverage)的输入:
// valid + applicable(Revision == 冻结契约 revision)。FAIL 由此阻断 VERIFIED
// (AC3);历史 FAIL 不被后续 PASS 抹掉由覆盖推导与 append-only 事实保证。
func (r HoldoutVerdictRecord) EvidenceRecord() EvidenceRecord {
	return EvidenceRecord{
		BindingID: r.BindingID,
		Revision:  r.ContractRevision,
		Status:    r.Status,
		Validity:  EvidenceValidityValid,
	}
}

// RedactedHoldoutOutcome 是进入 Implementation Agent 可见通道的唯一合法形态
// (AC2)。字段是 §8.4.3 allowlist:acceptance 语义(id)与 evidence type 之外
// 一无所有 —— 结构上不存在 path/selector/assertion/expected/actual/ref/digest,
// 也不存在 verdict(§8.4.1:可识别 verdict 不得进入实现侧通道;最终 VERIFIED
// 状态是 State Engine 的派生生命周期状态,不是本投影)。
type RedactedHoldoutOutcome struct {
	TaskID           string
	ContractRevision int
	AcceptanceID     string
	EvidenceType     EvidenceType
	CompletedAt      time.Time
}

// RedactHoldoutRecord 是 Control/Review 判定事实到实现侧可见投影的唯一出口:
// allowlist 逐字段拷贝,任何新增字段都必须显式进入本函数与 allowlist 测试,
// 不能靠"顺带"漏进实现侧通道(AC2)。
func RedactHoldoutRecord(r HoldoutVerdictRecord) RedactedHoldoutOutcome {
	return RedactedHoldoutOutcome{
		TaskID:           r.TaskID,
		ContractRevision: r.ContractRevision,
		AcceptanceID:     r.AcceptanceID,
		EvidenceType:     r.EvidenceType,
		CompletedAt:      r.CompletedAt,
	}
}

// HoldoutExecutor 把冻结契约中的 held_out binding 注入 Tally 控制的最终验证:
// 每条 binding 对一个 immutable ChangeSet、按冻结 execution context 执行一次
// (含冻结 repetitions)。实例按(契约, ChangeSet)构造;一个 binding 至多
// 消耗一次注入(AC5)。
type HoldoutExecutor struct {
	contract  *Frozen
	candidate *changeset.ChangeSet
	// reader 是 Holdout Evidence Store 的窄读能力(holdout.VerificationReader):
	// 只有验证权威侧代码持有;实现权威侧组件的类型签名里没有它(AC1 的类型层
	// 落点,与 internal/holdout 的能力拆分一致)。
	reader holdout.VerificationReader
	// run 是 Tally 控制的执行边界;内容只经它进入一次性验证环境。
	run HoldoutRunFunc

	records   []HoldoutVerdictRecord // 已产出的判定事实,append-only
	attempted map[string]bool        // binding id → 注入已消耗(无论结果)
}

// NewHoldoutExecutor 构造执行器。校验:
//   - 契约、candidate、reader、执行边界非 nil(AC1:没有 reader 能力就没有
//     内容来源;没有 Tally 控制边界就没有合法执行);
//   - 契约已声明 INV-VER-05 runtime 能力(纵深防御;Freeze 已强制);
//   - 契约至少含一条 held_out binding(本执行器只做 held-out 注入)。
//
// 任何校验失败都不产生执行器。
func NewHoldoutExecutor(contract *Frozen, candidate *changeset.ChangeSet, reader holdout.VerificationReader, run HoldoutRunFunc) (*HoldoutExecutor, error) {
	switch {
	case contract == nil:
		return nil, fmt.Errorf("%w: 冻结 VerificationContract 为 nil", ErrNilDependency)
	case candidate == nil:
		return nil, fmt.Errorf("%w: immutable ChangeSet 为 nil", ErrNilDependency)
	case reader == nil:
		return nil, fmt.Errorf("%w: holdout VerificationReader 能力为 nil(AC1:内容只能经窄能力进入)", ErrNilDependency)
	case run == nil:
		return nil, fmt.Errorf("%w: Tally 控制的执行边界为 nil", ErrNilDependency)
	}
	if r := contract.RuntimeRequirements(); !r.SessionIsolation || !r.HeldOutUnreachable || r.ConformanceClass == "" {
		return nil, fmt.Errorf("%w: 当前 %+v", ErrRuntimeRequirementsUnmet, r)
	}
	e := &HoldoutExecutor{
		contract:  contract,
		candidate: candidate,
		reader:    reader,
		run:       run,
		attempted: make(map[string]bool),
	}
	if len(e.HeldOutBindingIDs()) == 0 {
		return nil, ErrNoHeldOutBinding
	}
	return e, nil
}

// HeldOutBindingIDs 返回契约中全部 held_out binding id(契约声明顺序),
// 供控制面核对最终验证必须注入的完整集合(§14.2:held-out execution 是
// VERIFIED 前的必检项)。
func (e *HoldoutExecutor) HeldOutBindingIDs() []string {
	if e == nil || e.contract == nil {
		return nil
	}
	var ids []string
	c := e.contract.Contract()
	for i := range c.Acceptances {
		for j := range c.Acceptances[i].EvidenceBindings {
			if c.Acceptances[i].EvidenceBindings[j].VisibilityPolicy == VisibilityHeldOut {
				ids = append(ids, c.Acceptances[i].EvidenceBindings[j].ID)
			}
		}
	}
	return ids
}

// Records 返回已产出的判定事实(执行顺序的副本)。append-only:本执行器没有
// 任何删除/改写路径;历史 FAIL 不被后续结果抹掉(§8.8.1,AC3)。
func (e *HoldoutExecutor) Records() []HoldoutVerdictRecord {
	out := make([]HoldoutVerdictRecord, len(e.records))
	copy(out, e.records)
	return out
}

// Execute 把一条 held_out binding 注入最终验证:经 reader 读取内容(AC1),
// 完整性核对后,按冻结 execution context 执行全部 repetitions,聚合出判定
// (§8.7.2:任一 repetition FAIL 即 FAIL,无 cherry-pick)。
//
// x 必须与冻结契约、目标 binding、给定 ChangeSet 完全一致(§8.7.1 四元组);
// 协议必须就是契约为该 binding 冻结的协议(ErrProtocolMismatch 拒绝现编)。
//
// 一次性(AC5):从注入开始(reader.Read)起该 binding 即被消耗;内容缺失、
// 完整性失败、边界错误、取消等任何失败都不产生 verdict,也不提供重试。
// binding 不存在/不是 held_out/context 不匹配属于接线错误,发生在注入前,
// 不消耗该 binding。
//
// 成功时返回判定事实(Control/Review 通道)与其实现侧可见投影(实现通道)。
func (e *HoldoutExecutor) Execute(ctx context.Context, bindingID string, x ExecutionContext) (HoldoutVerdictRecord, RedactedHoldoutOutcome, error) {
	binding, acceptanceID, err := e.heldOutBinding(bindingID)
	if err != nil {
		return HoldoutVerdictRecord{}, RedactedHoldoutOutcome{}, err
	}
	if err := e.checkContext(binding, x); err != nil {
		return HoldoutVerdictRecord{}, RedactedHoldoutOutcome{}, err
	}
	if e.attempted[bindingID] {
		return HoldoutVerdictRecord{}, RedactedHoldoutOutcome{}, fmt.Errorf("%w: binding %s(重做只能经新契约 revision 的 fresh binding,§8.6.3)", ErrBindingAlreadyExecuted, bindingID)
	}
	// 自此注入已发生:一次性消耗(AC5)。
	e.attempted[bindingID] = true

	// AC1:held-out ref 只是不透明引用;内容只经 VerificationReader 能力解析,
	// 以内存字节存在,本包不落盘、不外显。
	ref, err := holdout.ParseRef(binding.Evidence.Ref)
	if err != nil {
		return HoldoutVerdictRecord{}, RedactedHoldoutOutcome{}, fmt.Errorf("verification: binding %s 的 held-out ref 非法: %w", bindingID, err)
	}
	content, err := e.reader.Read(ctx, ref)
	if err != nil {
		return HoldoutVerdictRecord{}, RedactedHoldoutOutcome{}, fmt.Errorf("verification: binding %s 读取 held-out 内容: %w", bindingID, err)
	}
	meta, err := e.reader.Stat(ctx, ref)
	if err != nil {
		return HoldoutVerdictRecord{}, RedactedHoldoutOutcome{}, fmt.Errorf("verification: binding %s 读取 held-out 元数据: %w", bindingID, err)
	}
	digest := holdout.ContentDigest(content)
	if digest != meta.Digest {
		return HoldoutVerdictRecord{}, RedactedHoldoutOutcome{}, fmt.Errorf("%w: binding %s 内容摘要与 Store 元数据不符", ErrHoldoutIntegrity, bindingID)
	}
	// §14.2 protected evidence hash:冻结时固定了 baseline_hash 的 binding,
	// 注入内容必须与之逐字节一致(INV-VER-03)。
	if binding.Protection != nil && binding.Protection.BaselineHash != "" && digest != binding.Protection.BaselineHash {
		return HoldoutVerdictRecord{}, RedactedHoldoutOutcome{}, fmt.Errorf("%w: binding %s 内容摘要与冻结 protection.baseline_hash 不一致", ErrHoldoutIntegrity, bindingID)
	}

	// 冻结执行协议(§8.7.2):session 构造即冻结 context,此后 repetitions/
	// seed/环境不可扩大或替换。
	session, err := NewExecutionSession(x)
	if err != nil {
		return HoldoutVerdictRecord{}, RedactedHoldoutOutcome{}, fmt.Errorf("verification: binding %s 冻结执行协议: %w", bindingID, err)
	}

	digests := make([]string, 0, x.Protocol.Repetitions)
	for i := 1; i <= x.Protocol.Repetitions; i++ {
		if err := ctx.Err(); err != nil {
			return HoldoutVerdictRecord{}, RedactedHoldoutOutcome{}, fmt.Errorf("verification: binding %s repetition %d 前取消(一次性注入,无自动重试): %w", bindingID, i, err)
		}
		out, err := e.run(ctx, HoldoutRunRequest{
			BindingID: bindingID,
			Command:   cloneCommandSpec(x.Protocol.Command),
			Seed:      x.Protocol.Seed,
			Content:   content,
		})
		if err != nil {
			return HoldoutVerdictRecord{}, RedactedHoldoutOutcome{}, fmt.Errorf("verification: binding %s repetition %d 执行边界失败(一次性注入,失败不自动重试,AC5): %w", bindingID, i, err)
		}
		switch out.Status {
		case EvidenceStatusPass, EvidenceStatusFail:
		default:
			return HoldoutVerdictRecord{}, RedactedHoldoutOutcome{}, fmt.Errorf("%w: binding %s repetition %d status %q(只支持 pass|fail)", ErrInvalidRunOutput, bindingID, i, out.Status)
		}
		if !sha256HashRe.MatchString(out.OutputDigest) {
			return HoldoutVerdictRecord{}, RedactedHoldoutOutcome{}, fmt.Errorf("%w: binding %s repetition %d 输出摘要必须形如 sha256:<64 位十六进制>,当前 %q", ErrInvalidRunOutput, bindingID, i, out.OutputDigest)
		}
		if err := session.Record(RunObservation{
			Index:       i,
			Status:      out.Status,
			Seed:        x.Protocol.Seed,
			Environment: x.Environment,
		}); err != nil {
			return HoldoutVerdictRecord{}, RedactedHoldoutOutcome{}, fmt.Errorf("verification: binding %s repetition %d 记录: %w", bindingID, i, err)
		}
		digests = append(digests, out.OutputDigest)
	}

	verdict, err := session.Verdict()
	if err != nil {
		return HoldoutVerdictRecord{}, RedactedHoldoutOutcome{}, fmt.Errorf("verification: binding %s 聚合判定: %w", bindingID, err)
	}
	rec := HoldoutVerdictRecord{
		TaskID:           e.contract.TaskID(),
		ContractRevision: e.contract.Revision(),
		ContractHash:     e.contract.Hash(),
		AcceptanceID:     acceptanceID,
		BindingID:        bindingID,
		CandidateID:      x.CandidateID,
		ContextHash:      verdict.ContextHash,
		EvidenceType:     binding.Evidence.Type,
		ContentDigest:    digest,
		Status:           verdict.Status,
		Repetitions:      verdict.Repetitions,
		Failures:         verdict.Failures,
		OutputDigests:    digests,
		CompletedAt:      time.Now().UTC(),
	}
	e.records = append(e.records, rec)
	return rec, RedactHoldoutRecord(rec), nil
}

// heldOutBinding 在冻结契约中定位 bindingID,并要求它是 held_out(AC1:本
// 执行器只注入 held-out 证据)。返回 binding 指针(指向冻结内容的深拷贝,
// 只读)与其所属 acceptance id。
func (e *HoldoutExecutor) heldOutBinding(bindingID string) (*Binding, string, error) {
	c := e.contract.Contract()
	for i := range c.Acceptances {
		for j := range c.Acceptances[i].EvidenceBindings {
			b := &c.Acceptances[i].EvidenceBindings[j]
			if b.ID != bindingID {
				continue
			}
			if b.VisibilityPolicy != VisibilityHeldOut {
				return nil, "", fmt.Errorf("%w: binding %s 是 %s;visible binding 走增量验证通道", ErrBindingNotHeldOut, bindingID, b.VisibilityPolicy)
			}
			return b, c.Acceptances[i].ID, nil
		}
	}
	return nil, "", fmt.Errorf("%w: %s", ErrBindingNotFound, bindingID)
}

// checkContext 校验 execution context 与冻结契约、目标 binding、immutable
// ChangeSet 的一致性(§8.7.1 四元组;AC4)。
func (e *HoldoutExecutor) checkContext(b *Binding, x ExecutionContext) error {
	if x.ContractRevision != e.contract.Revision() || x.ContractHash != e.contract.Hash() || x.BindingID != b.ID {
		return fmt.Errorf("%w: context(rev %d,hash %s,binding %s)与冻结契约(rev %d,hash %s)/目标 binding %s 不一致", ErrExecutionContextMismatch, x.ContractRevision, x.ContractHash, x.BindingID, e.contract.Revision(), e.contract.Hash(), b.ID)
	}
	if x.CandidateID != string(e.candidate.ID()) {
		return fmt.Errorf("%w: context 绑定 candidate %s,给定 ChangeSet 是 %s(可执行代码变更必须形成新 ChangeSet 并绑定新 verdict,§8.7.1)", ErrCandidateMismatch, x.CandidateID, e.candidate.ID())
	}
	if b.ExecutionProtocol == nil {
		return fmt.Errorf("%w: binding %s 没有冻结执行协议,不存在可绑定的 verdict(§8.7.1)", ErrProtocolMismatch, b.ID)
	}
	want, err := protocolHash(b.ExecutionProtocol)
	if err != nil {
		return fmt.Errorf("verification: binding %s 冻结协议哈希: %w", b.ID, err)
	}
	got, err := protocolHash(&x.Protocol)
	if err != nil {
		return fmt.Errorf("verification: execution context 协议哈希: %w", err)
	}
	if want != got {
		return fmt.Errorf("%w: execution context 携带的协议与契约冻结协议不一致(§8.7.2:协议必须首次执行前随契约冻结)", ErrProtocolMismatch)
	}
	return nil
}
