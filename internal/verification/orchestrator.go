package verification

// orchestrator.go 实现 P1-38:一个 contract revision + 一个 immutable ChangeSet
// 的最终验证编排(PRD §14.2 Final Validation、§14.3、§8.7、§8.13、§36.3)。
//
// 职责:对(冻结 VerificationContract, immutable ChangeSet, environment identity)
// 这一条确定性验证路径,把 §14.2 要求的最终验证面一次跑完并落成事实:
//   1. 逐条执行全部 visible binding 的冻结执行协议(repetitions/all_pass/seed
//      经 ExecutionSession 机械强制,§8.7.2),verdict 绑定 §8.7.1 四元组
//      (contract revision + binding + candidate + frozen protocol + environment);
//   2. 执行 required_gates 的结构化命令(§14.2 required build/architecture/
//      security/schema gates;结构化 argv,无 shell 语义,决策锁 6);
//   3. 经 HoldoutExecutor(P1-29)把 held_out binding 注入 Tally 控制的最终
//      验证:一次性消耗、无重试(决策锁 12);
//   4. 汇总本轮 verdict 与「既往 valid 且 applicable 的 evidence」(经持久化
//      端口读回)推导最终功能覆盖(P1-19)与 effective independence 覆盖
//      (P1-20),由此给出 verified / failed / needs_review 结论;
//   5. 经 RunSink 端口恰好落一行 VerificationRun(绑定精确 contract/change/
//      environment 身份,§8.13)及其全部 evidence 事实(append-only,
//      INV-FACT-01),Recorder 通道为 fact-capable 的 verification_run(INV-AUTH-01)。
//
// 结论语义(§36.3:Verified / Failed / Needs Review,确定性判定顺序):
//   - 任一 binding/gate 执行出错(无可用 verdict)→ needs_review;
//   - 任一 held_out binding 存在 valid+applicable FAIL(本轮或既往)→
//     needs_review:非 VERIFIED 信号进入实现侧可见通道,但任何 held-out
//     诊断(ref/内容/输出/可识别 verdict)都不进入该通道(AC3,§8.4.1/§14.3);
//   - 功能覆盖不完整,或 required gate FAIL → failed;
//   - effective independence 覆盖不完整(planned ✓ → effective ✗)→
//     needs_review(Waiver 判定归 policy 层,§8.5.2);
//   - 其余 → verified。
//
// AC4(§8.8.1):既往 valid+applicable FAIL 是 append-only 事实。本编排器经
// RunSink 读回既往事实参与覆盖推导:新的 PASS 覆盖不了它,只有带 provenance
// 的正式裁定(invalidation)才使其失效——本包不存在任何裁定/改写端口,
// 「新 binding PASS 抹掉旧 FAIL」在结构上不可表达。
//
// 边界(P1):
//   - 持久化经 RunSink 窄端口(runstore 适配归集成接线;internal/runstore 反向
//     依赖本包,不能直接引用)。仅「完整推导出结论」的 Run 会落库;执行中断
//     (基础设施/取消/完整性失败)返回 error 且不落任何行——无 verdict 的
//     半次执行不得伪装成事实;
//   - Evidence Qualification 的 §8.3 final checks 与 protected 仓库文件哈希
//     归控制面(P1-16 评估器 / hunk inventory);本编排器只消费冻结协议执行;
//   - 实现侧可见投影只有 RedactedSignal(allowlist):结构上不含 held-out
//     内容/ref/digest/verdict;完整 HoldoutVerdictRecord 只在 Control/Review
//     通道(FinalOutcome)。

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"tally/internal/changeset"
	"tally/internal/command"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/holdout"
	"tally/internal/policy"
	"tally/internal/runner"
)

// FinalStatus — 一次最终验证编排的结论(§36.3)。
type FinalStatus string

const (
	FinalVerified    FinalStatus = "verified"     // 全部门通过
	FinalFailed      FinalStatus = "failed"       // 确定性验证失败(可见失败/required gate 失败)
	FinalNeedsReview FinalStatus = "needs_review" // 非 VERIFIED:需 Control/Review 侧人工/政策裁决
)

// RunStatus — VerificationRun 行上的观测结果(与 runstore 的 pass|fail|error
// 词表一致)。needs_review 在行上映射为 error:运行级没有干净的 pass/fail 结论。
type RunStatus string

const (
	RunStatusPass  RunStatus = "pass"
	RunStatusFail  RunStatus = "fail"
	RunStatusError RunStatus = "error"
)

// 哨兵错误:调用方用 errors.Is 判因,不做字符串匹配。
var (
	// ErrInvalidRequest — Run 请求不完整或不一致(nil 契约/候选、项目不匹配、
	// 环境身份非法等)。请求错误在任何执行发生前被拒绝。
	ErrInvalidRequest = errors.New("verification: 最终验证编排请求非法")
	// ErrHeldOutChannelMissing — 契约含 held_out binding,但编排器没有接
	// holdout VerificationReader / Tally 控制执行边界;没有注入通道就不存在
	// 合法的 held-out 最终验证(INV-VER-05),在执行前拒绝。
	ErrHeldOutChannelMissing = errors.New("verification: 契约含 held_out binding 但编排器缺少 holdout reader/执行边界")
)

// RunSpec — 落一行 VerificationRun 的输入(§8.13:精确绑定 contract/change/
// environment 身份)。字段与持久化适配层的 VerificationRun 行一一对应。
type RunSpec struct {
	Project          core.ProjectID
	Task             string
	ContractRevision int
	ContractHash     string
	ChangeSetID      changeset.ChangeSetID
	EnvironmentName  string // 验证环境的稳定名称
	EnvironmentHash  string // 环境语义身份内容哈希(EnvironmentIdentity.Hash)
	VersionCombo     core.VersionCombo
	Runtime          string // 执行后端身份
	StartedAt        time.Time
	FinishedAt       time.Time
	Status           RunStatus
}

// FactSpec — 落一条 evidence 事实的输入(§14.3:结构化命令、判定、摘要、
// recorder provenance)。FactID 由编排器按 (run, binding) 确定性派生:
// 事实身份唯一且永不复用,同一 Run 的同一 binding 二次落库是错误而非更新。
type FactSpec struct {
	FactID           evidence.FactID
	RunID            int64
	Task             string
	Acceptance       string
	BindingID        string
	VisibilityPolicy VisibilityPolicy
	Verdict          EvidenceStatus // pass|fail(编排器只落有 verdict 的事实)
	Recorder         evidence.Recorder
	Command          *CommandSpec // 冻结协议声明的结构化命令
	ExitCode         int          // 末次 repetition 的退出码
	StdoutDigest     string
	StderrDigest     string
	RecordedAt       time.Time
}

// PriorFact — 既往 valid 且 applicable 的 evidence 事实投影(RunSink 读回)。
// 只有 valid 事实会到达这里:被正式裁定失效的事实不再 gate(§8.8.1),
// 过滤责任在持久化端口的实现侧(它是 validity 的权威)。
type PriorFact struct {
	BindingID  string
	Status     EvidenceStatus // pass|fail(error 判定无可用 verdict,适配层省略)
	RecordedAt time.Time      // 原始记录时刻(selection exposure 时钟输入,§8.6.2)
}

// RunSink — 最终验证编排的持久化窄端口。实现侧由控制面接线到 append-only
// 事实存储(internal/runstore);端口刻意不含任何更新/删除/裁定方法:
// evidence 落库后不存在改写路径(INV-FACT-01,AC4 的结构层落点)。
type RunSink interface {
	// RecordRun 恰好落一行 VerificationRun,返回其行 id。
	RecordRun(ctx context.Context, spec RunSpec) (int64, error)
	// RecordFact 落一条由该 Run 产生的 evidence 事实。
	RecordFact(ctx context.Context, project core.ProjectID, fact FactSpec) error
	// ValidApplicableFacts 返回既往 valid 且精确适用该 scope(project/task/
	// acceptance/contract revision/ChangeSet)的 evidence 事实(§8.8.1/8.8.2)。
	ValidApplicableFacts(ctx context.Context, scope evidence.Scope) ([]PriorFact, error)
}

// OrchestratorConfig — NewOrchestrator 的依赖面。
type OrchestratorConfig struct {
	// Runner 执行 visible binding 与 required gate 的结构化命令(一次性
	// candidate workspace 内,INV-EXEC-01)。
	Runner runner.Runner
	// Policy 是执行边界(墙钟预算上限);非法策略拒绝构造执行器。
	Policy policy.ExecutionPolicy
	// HoldoutReader / HoldoutRun 是 held-out 注入通道(P1-29);契约不含
	// held_out binding 时可以不接。实现权威侧组件的类型签名里不存在它们。
	HoldoutReader holdout.VerificationReader
	HoldoutRun    HoldoutRunFunc
	// Sink 是持久化端口(见 RunSink)。
	Sink RunSink
	// Recorder 是落库 provenance;必须是 fact-capable 通道(INV-AUTH-01/02)。
	Recorder evidence.Recorder
}

// Orchestrator — 一个 contract revision × 一个 immutable ChangeSet 的最终
// 验证编排器。每次 Run 恰好产生一行 VerificationRun;同一 (contract, ChangeSet)
// 的多次 Run 各自成行,禁止 cherry-pick 靠全量留痕(§8.13)。
type Orchestrator struct {
	runner     runner.Runner
	budget     time.Duration // policy.Budgets.MaxWallTime,构造时已校验为正
	reader     holdout.VerificationReader
	holdoutRun HoldoutRunFunc
	sink       RunSink
	recorder   evidence.Recorder
}

// NewOrchestrator 构造编排器。依赖缺失或策略非法时不产生编排器(保守失败,
// 不构造半成品验证路径)。
func NewOrchestrator(cfg OrchestratorConfig) (*Orchestrator, error) {
	switch {
	case cfg.Runner == nil:
		return nil, fmt.Errorf("%w: 命令 runner 为 nil", ErrNilDependency)
	case cfg.Sink == nil:
		return nil, fmt.Errorf("%w: 持久化端口(RunSink)为 nil", ErrNilDependency)
	}
	if err := cfg.Policy.Validate(); err != nil {
		return nil, fmt.Errorf("verification: 执行策略非法,拒绝构造最终验证编排器: %w", err)
	}
	if !cfg.Recorder.Kind.CanRecordFacts() || strings.TrimSpace(cfg.Recorder.Actor) == "" {
		return nil, fmt.Errorf("%w: recorder %q/%q 不是 fact-capable 通道(INV-AUTH-01/02)", ErrNilDependency, cfg.Recorder.Kind, cfg.Recorder.Actor)
	}
	return &Orchestrator{
		runner:     cfg.Runner,
		budget:     cfg.Policy.Budgets.MaxWallTime,
		reader:     cfg.HoldoutReader,
		holdoutRun: cfg.HoldoutRun,
		sink:       cfg.Sink,
		recorder:   cfg.Recorder,
	}, nil
}

// BindingAttestation — 一条 binding 的实现决策 lineage 接触 attestation
// (§8.2.2 活动接触;effective independence 推导输入)。缺省 = 全 false:
// 缺失的 provenance 保守取依赖侧。
type BindingAttestation struct {
	// CandidateBlind — post_hoc authoring 的 candidate-blind 证明(§8.2.1;
	// 仅 post_hoc 被消费,pre_commit 由 mode 自证)。
	CandidateBlind bool
	// AuthoredInLineage — 该证据作为当前实现决策 lineage 的一部分被首次创作。
	AuthoredInLineage bool
	// ChangedInLineage — 既存证据内容在当前实现决策 lineage 内被修改。
	ChangedInLineage bool
}

// FinalRunRequest — 一次最终验证的输入。除执行所需边界外,还携带 effective
// independence 推导(§8.5.2)的控制面 provenance 输入:它们是 append-only
// 事实/attestation 的投影,编排器不发明、不改写。
type FinalRunRequest struct {
	// Project — 项目边界;必须与 ChangeSet 所属项目一致。
	Project core.ProjectID
	// Contract — 冻结的 VerificationContract(权威;作者侧契约不可执行)。
	Contract *Frozen
	// ChangeSet — 被验证的 immutable ChangeSet(verdict 绑定对象,§8.7.1)。
	ChangeSet *changeset.ChangeSet
	// EnvironmentName / Environment / VersionCombo / Runtime — 验证环境身份:
	// 稳定名称、语义身份(全字段参与身份哈希)、版本组合快照与执行后端身份。
	EnvironmentName string
	Environment     EnvironmentIdentity
	VersionCombo    core.VersionCombo
	Runtime         string
	// Workspace — candidate workspace 根目录(一次性 Git worktree;INV-EXEC-01)。
	// visible binding 与 gate 命令的相对 cwd 在其内解析,逃出即拒绝。必须为绝对路径。
	Workspace string
	// CandidateFixedAt — 候选快照固化时刻(selection exposure 时钟,§8.6.2);
	// 零值 = 无法证明固化,选择暴露取污染侧(保守默认)。
	CandidateFixedAt time.Time
	// Lineage — 候选 change lineage:候选自身在前,同实现决策内的祖先随后。
	Lineage []evidence.ChangeSetID
	// Implementation — 候选的实际实现权威 provenance;为空时 source 独立性
	// 保守取依赖侧(§8.2.2)。
	Implementation evidence.ImplementationProvenance
	// Precommit — 该候选的机械可验证选择 precommit(§8.6.2);零值 = 无。
	Precommit evidence.SelectionPrecommit
	// Attestations — 逐 binding 的 lineage 接触 attestation;缺省 = 全 false。
	Attestations map[string]BindingAttestation
	// Exposures — 全部既往 exposure facts(§8.6.1);推导按 BindingRef 选择。
	Exposures []evidence.Exposure
}

// GateResult — 一个 required gate 的执行观察(§14.3 结构化结果投影)。
type GateResult struct {
	Name         string
	Required     bool
	Status       VisibleResultStatus // pass | fail | error
	ExitCode     int
	StdoutDigest string
	StderrDigest string
	Duration     time.Duration
	Err          string // Status=error 时的人类可读原因
}

// BindingOutcome — 一条 visible binding 的最终执行观察:冻结协议完整执行后
// 的 §8.7.1 四元组 verdict(Err 为空时 Status 有效)。
type BindingOutcome struct {
	BindingID    string
	AcceptanceID string
	Visibility   VisibilityPolicy
	ContextHash  string // execution context 四元组身份哈希
	Command      *CommandSpec
	Status       EvidenceStatus // pass|fail(Err 为空时有意义)
	Repetitions  int
	Failures     int
	ExitCode     int // 末次 repetition
	StdoutDigest string
	StderrDigest string
	Duration     time.Duration
	Err          string // 非空 = 无可用 verdict(执行出错/预算耗尽/协议缺失)
}

// FinalOutcome — 一次最终验证编排的完整结论。Control/Review Authority 侧
// 事实载体:含完整 verdict、覆盖推导与既往事实合并结果。实现侧只见
// RedactedSignal(AC3)。
type FinalOutcome struct {
	RunID            int64
	TaskID           string
	ContractRevision int
	ContractHash     string
	ChangeSetID      changeset.ChangeSetID
	EnvironmentHash  string
	Status           FinalStatus
	// Bindings — visible binding 的最终执行观察(契约声明顺序)。
	Bindings []BindingOutcome
	// Gates — 已执行 gate 的观察(名称字典序)。
	Gates []GateResult
	// Functional / Independence — 最终覆盖推导(P1-19/P1-20;已合并既往
	// valid+applicable 事实)。覆盖结论可逐 acceptance 追溯到具体 binding
	// (INV-STATE-02)。
	Functional   FunctionalCoverageReport
	Independence IndependenceCoverageReport
	// Holdout — held-out 判定事实(仅 Control/Review 通道;§14.3 原始失败
	// 输出从不进入任何返回值)。
	Holdout []HoldoutVerdictRecord
	// FactIDs — 本次落库的 evidence 事实身份(按落库顺序)。
	FactIDs []evidence.FactID
	// StartedAt / FinishedAt — 本轮编排的实际起止时刻。
	StartedAt  time.Time
	FinishedAt time.Time
}

// RedactedFinalSignal — 实现侧可见通道的唯一合法投影(AC3):生命周期信号
// (verified/failed/needs_review)与 held-out 执行完成的 allowlist 事实;
// 结构上不含 held-out 内容/ref/digest/可识别 verdict(§8.4.1/§8.4.3)。
type RedactedFinalSignal struct {
	TaskID           string
	ContractRevision int
	Status           FinalStatus
	Holdouts         []RedactedHoldoutOutcome
}

// RedactedSignal 把 Control/Review 侧结论投影为实现侧可见信号。投影逐字段
// allowlist 构造:held-out 判定经 RedactHoldoutRecord(P1-29 的唯一出口)。
func (o *FinalOutcome) RedactedSignal() RedactedFinalSignal {
	if o == nil {
		return RedactedFinalSignal{}
	}
	sig := RedactedFinalSignal{
		TaskID:           o.TaskID,
		ContractRevision: o.ContractRevision,
		Status:           o.Status,
	}
	for _, h := range o.Holdout {
		sig.Holdouts = append(sig.Holdouts, RedactHoldoutRecord(h))
	}
	return sig
}

// contractBinding — 冻结契约声明顺序下的一条 binding 及其所属 acceptance。
type contractBinding struct {
	acceptanceID string
	binding      Binding
}

// frozenBindings 返回冻结契约的全部 binding(声明顺序)。
func frozenBindings(f *Frozen) []contractBinding {
	c := f.Contract()
	out := make([]contractBinding, 0, len(c.Acceptances))
	for i := range c.Acceptances {
		acc := &c.Acceptances[i]
		for j := range acc.EvidenceBindings {
			out = append(out, contractBinding{acceptanceID: acc.ID, binding: acc.EvidenceBindings[j]})
		}
	}
	return out
}

// frozenHeldOutBindingIDs 返回契约中全部 held_out binding id(声明顺序)。
func frozenHeldOutBindingIDs(f *Frozen) []string {
	var ids []string
	for _, cb := range frozenBindings(f) {
		if cb.binding.VisibilityPolicy == VisibilityHeldOut {
			ids = append(ids, cb.binding.ID)
		}
	}
	return ids
}

// Run 执行一次最终验证编排:执行全部 visible binding 冻结协议与 required
// gates、注入全部 held-out binding、合并既往 valid+applicable 事实推导最终
// 覆盖、经 RunSink 恰好落一行 VerificationRun 与全部 evidence 事实,返回完整
// 结论(Control/Review 侧)。
//
// 求值失败(基础设施错误、取消、holdout 完整性失败、sink 读写失败)返回
// error 且不落任何行:无 verdict 的半次执行不伪装成事实。验证结论本身
// (failed/needs_review)不是 error——失败 Run 恰恰必须被记录(§8.13/AC4)。
func (o *Orchestrator) Run(ctx context.Context, req FinalRunRequest) (*FinalOutcome, error) {
	if err := o.validateRequest(&req); err != nil {
		return nil, err
	}
	f := req.Contract
	envHash, err := req.Environment.Hash()
	if err != nil {
		return nil, fmt.Errorf("%w: 环境身份哈希: %v", ErrInvalidRequest, err)
	}
	task := f.TaskID()
	csID := req.ChangeSet.ID()
	startedAt := time.Now().UTC()

	// ---- 1. visible bindings:冻结协议完整执行(§8.7.2) ----
	var bindings []BindingOutcome
	used := time.Duration(0)
	for _, cb := range frozenBindings(f) {
		if cb.binding.VisibilityPolicy != VisibilityVisible {
			continue
		}
		bo := o.executeVisibleBinding(ctx, &req, f, cb, &used)
		bindings = append(bindings, bo)
	}

	// ---- 2. required gates(§14.2;名称字典序,确定性顺序) ----
	gates, gateErr := o.executeGates(ctx, &req, &used)

	// ---- 3. held-out 注入(P1-29;一次性消耗) ----
	holdoutRecs, holdoutFatal, err := o.executeHeldOut(ctx, &req, f)
	if err != nil {
		return nil, err
	}

	// ---- 4. 合并既往 valid+applicable 事实并推导覆盖 ----
	priors, err := o.loadPriorFacts(ctx, &req, f)
	if err != nil {
		return nil, err
	}
	finishedAt := time.Now().UTC()

	records := make([]EvidenceRecord, 0, len(bindings)+len(holdoutRecs)+len(priors))
	for _, bo := range bindings {
		if bo.Err != "" {
			continue // 无 verdict 的执行不产生事实
		}
		records = append(records, EvidenceRecord{BindingID: bo.BindingID, Revision: f.Revision(), Status: bo.Status, Validity: EvidenceValidityValid})
	}
	for i := range holdoutRecs {
		records = append(records, holdoutRecs[i].EvidenceRecord())
	}
	for _, p := range priors {
		records = append(records, EvidenceRecord{BindingID: p.BindingID, Revision: f.Revision(), Status: p.Status, Validity: EvidenceValidityValid})
	}
	functional, err := DeriveFunctionalCoverage(f, records)
	if err != nil {
		return nil, fmt.Errorf("verification: 推导最终功能覆盖: %w", err)
	}
	independence, err := o.deriveIndependence(&req, f, bindings, holdoutRecs, priors, finishedAt)
	if err != nil {
		return nil, err
	}

	// ---- 5. 结论(确定性判定顺序,见文件头) ----
	bindingErr := false
	for _, bo := range bindings {
		if bo.Err != "" {
			bindingErr = true
		}
	}
	requiredGateFailed := false
	for _, g := range gates {
		if g.Required && g.Status == VisibleStatusFail {
			requiredGateFailed = true
		}
	}
	executionIncomplete := bindingErr || gateErr || holdoutFatal
	status := FinalVerified
	switch {
	case executionIncomplete:
		status = FinalNeedsReview
	case len(heldOutBlockedBindings(f, functional)) > 0:
		status = FinalNeedsReview
	case !functional.Complete || requiredGateFailed:
		status = FinalFailed
	case !independence.Complete:
		status = FinalNeedsReview
	}

	// ---- 6. 持久化:恰好一行 VerificationRun + 全部 evidence 事实 ----
	runID, err := o.sink.RecordRun(ctx, RunSpec{
		Project:          req.Project,
		Task:             task,
		ContractRevision: f.Revision(),
		ContractHash:     f.Hash(),
		ChangeSetID:      csID,
		EnvironmentName:  req.EnvironmentName,
		EnvironmentHash:  envHash,
		VersionCombo:     req.VersionCombo,
		Runtime:          req.Runtime,
		StartedAt:        startedAt,
		FinishedAt:       finishedAt,
		Status:           runStatusFor(status),
	})
	if err != nil {
		return nil, fmt.Errorf("verification: 落 VerificationRun(task=%s changeset=%s): %w", task, csID, err)
	}
	facts := buildFactSpecs(runID, task, finishedAt, o.recorder, bindings, holdoutRecs)
	factIDs := make([]evidence.FactID, 0, len(facts))
	for _, fact := range facts {
		if err := o.sink.RecordFact(ctx, req.Project, fact); err != nil {
			return nil, fmt.Errorf("verification: 落 evidence %s: %w", fact.FactID, err)
		}
		factIDs = append(factIDs, fact.FactID)
	}

	return &FinalOutcome{
		RunID:            runID,
		TaskID:           task,
		ContractRevision: f.Revision(),
		ContractHash:     f.Hash(),
		ChangeSetID:      csID,
		EnvironmentHash:  envHash,
		Status:           status,
		Bindings:         bindings,
		Gates:            gates,
		Functional:       functional,
		Independence:     independence,
		Holdout:          holdoutRecs,
		FactIDs:          factIDs,
		StartedAt:        startedAt,
		FinishedAt:       finishedAt,
	}, nil
}

// validateRequest 在任何执行发生前校验请求(含 held-out 注入通道完整性)。
func (o *Orchestrator) validateRequest(req *FinalRunRequest) error {
	switch {
	case req.Contract == nil:
		return fmt.Errorf("%w: 冻结 VerificationContract 为 nil", ErrInvalidRequest)
	case req.ChangeSet == nil:
		return fmt.Errorf("%w: immutable ChangeSet 为 nil", ErrInvalidRequest)
	case req.Project == "":
		return fmt.Errorf("%w: 缺少 project", ErrInvalidRequest)
	case req.Project != req.ChangeSet.Project():
		return fmt.Errorf("%w: ChangeSet %s 属于项目 %s,与请求项目 %s 不一致", ErrInvalidRequest, req.ChangeSet.ID(), req.ChangeSet.Project(), req.Project)
	case strings.TrimSpace(req.EnvironmentName) == "":
		return fmt.Errorf("%w: 缺少验证环境名称", ErrInvalidRequest)
	case strings.TrimSpace(req.Runtime) == "":
		return fmt.Errorf("%w: 缺少执行后端身份(runtime)", ErrInvalidRequest)
	case len(req.VersionCombo) == 0:
		return fmt.Errorf("%w: version_combo 不能为空(执行环境必须自包含)", ErrInvalidRequest)
	case strings.TrimSpace(req.Workspace) == "":
		return ErrNoWorkspace
	case !filepath.IsAbs(req.Workspace):
		return fmt.Errorf("%w: workspace 必须是绝对路径,当前 %q", ErrInvalidRequest, req.Workspace)
	}
	if err := req.Environment.Validate(); err != nil {
		return fmt.Errorf("%w: 环境身份: %v", ErrInvalidRequest, err)
	}
	if len(frozenHeldOutBindingIDs(req.Contract)) > 0 && (o.reader == nil || o.holdoutRun == nil) {
		return ErrHeldOutChannelMissing
	}
	return nil
}

// executeVisibleBinding 对一条 visible binding 完整执行其冻结协议:
// NewExecutionSession 先冻结 §8.7.1 四元组与停止规则,逐 repetition 以结构化
// argv 在 candidate workspace 内执行并 append-only 记录,最后聚合 verdict。
// 任何失败都不产生 verdict(Err 非空),也不中断其余 binding(给 Review 完整
// 画面);墙钟预算为整轮硬上限。
func (o *Orchestrator) executeVisibleBinding(ctx context.Context, req *FinalRunRequest, f *Frozen, cb contractBinding, used *time.Duration) BindingOutcome {
	b := &cb.binding
	out := BindingOutcome{BindingID: b.ID, AcceptanceID: cb.acceptanceID, Visibility: b.VisibilityPolicy}
	proto := b.ExecutionProtocol
	if proto == nil || proto.Command == nil {
		out.Err = "binding 没有冻结执行协议,最终验证无法产生 verdict(§8.7.1)"
		return out
	}
	xctx := ExecutionContext{
		ContractRevision: f.Revision(),
		ContractHash:     f.Hash(),
		BindingID:        b.ID,
		CandidateID:      string(req.ChangeSet.ID()),
		Protocol:         *proto,
		Environment:      req.Environment,
	}
	session, err := NewExecutionSession(xctx)
	if err != nil {
		out.Err = err.Error()
		return out
	}
	out.ContextHash = session.ContextHash()
	out.Repetitions = proto.Repetitions
	out.Command = cloneCommandSpec(proto.Command)

	_, env, cs, specErr := prepareCommand(req.Workspace, proto.Command)
	if specErr != nil {
		out.Err = specErr.Error()
		return out
	}

	for i := 1; i <= proto.Repetitions; i++ {
		if err := ctx.Err(); err != nil {
			out.Err = fmt.Sprintf("repetition %d 前取消(无 verdict): %v", i, err)
			return out
		}
		remaining := o.budget - *used
		if remaining <= 0 {
			out.Err = ErrBudgetExhausted.Error()
			return out
		}
		raw, runErr := o.runWithinBudget(ctx, cs, proto.Command, env, remaining)
		*used += raw.Duration
		out.Duration += raw.Duration
		if runErr != nil {
			out.Err = fmt.Sprintf("repetition %d 执行失败(无 verdict): %v", i, runErr)
			return out
		}
		out.ExitCode = raw.ExitCode
		out.StdoutDigest = sha256Digest(raw.Stdout)
		out.StderrDigest = sha256Digest(raw.Stderr)
		// P6-05:冻结协议声明的 infra 退出码 = 文档化的 wiring/infra 错误
		// (内容无关)。命中的执行无 verdict、不落事实、结论 needs_review —
		// verifier 错误不得伪造为候选失败(AGENTS.md §5 信任不变量)。
		// 未声明的退出码语义不变:0 = pass,其余(含 1)= 可信 FAIL。
		if proto.Command.isInfraExitCode(raw.ExitCode) {
			out.Err = fmt.Sprintf("repetition %d 命中冻结协议声明的 infra 退出码 %d(文档化 wiring/infra 错误,无 verdict,不伪造为候选 FAIL)", i, raw.ExitCode)
			return out
		}
		st := EvidenceStatusFail
		if raw.ExitCode == 0 {
			st = EvidenceStatusPass
		}
		if st == EvidenceStatusFail {
			out.Failures++
		}
		if err := session.Record(RunObservation{Index: i, Status: st, Seed: xctx.Protocol.Seed, Environment: req.Environment}); err != nil {
			out.Err = fmt.Sprintf("repetition %d 记录: %v", i, err)
			return out
		}
	}
	verdict, err := session.Verdict()
	if err != nil {
		out.Err = err.Error()
		return out
	}
	out.Status = verdict.Status
	return out
}

// executeGates 执行全部带结构化命令的 gate(required 与 advisory;只有
// required 的 FAIL 参与结论)。名称字典序保证确定性执行与报告顺序。
func (o *Orchestrator) executeGates(ctx context.Context, req *FinalRunRequest, used *time.Duration) ([]GateResult, bool) {
	names := make([]string, 0, len(req.Contract.Contract().RequiredGates))
	for name, gate := range req.Contract.Contract().RequiredGates {
		if gate.Command != nil {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	errSeen := false
	out := make([]GateResult, 0, len(names))
	for _, name := range names {
		gate := req.Contract.Contract().RequiredGates[name]
		g := GateResult{Name: name, Required: gate.Required}
		_, env, cs, specErr := prepareCommand(req.Workspace, gate.Command)
		if specErr != nil {
			g.Status = VisibleStatusError
			g.Err = specErr.Error()
			errSeen = true
			out = append(out, g)
			continue
		}
		if err := ctx.Err(); err != nil {
			g.Status = VisibleStatusError
			g.Err = fmt.Sprintf("执行前取消: %v", err)
			errSeen = true
			out = append(out, g)
			continue
		}
		remaining := o.budget - *used
		if remaining <= 0 {
			g.Status = VisibleStatusError
			g.Err = ErrBudgetExhausted.Error()
			errSeen = true
			out = append(out, g)
			continue
		}
		raw, runErr := o.runWithinBudget(ctx, cs, gate.Command, env, remaining)
		*used += raw.Duration
		g.Duration = raw.Duration
		g.ExitCode = raw.ExitCode
		g.StdoutDigest = sha256Digest(raw.Stdout)
		g.StderrDigest = sha256Digest(raw.Stderr)
		switch {
		case runErr != nil:
			g.Status = VisibleStatusError
			g.Err = runErr.Error()
			errSeen = true
		case gate.Command.isInfraExitCode(raw.ExitCode):
			// P6-05:与 visible binding 同一 infra 通道 — 声明的 infra 退出码
			// 是 wiring 错误(error → needs_review),不是可信 gate FAIL。
			g.Status = VisibleStatusError
			g.Err = fmt.Sprintf("gate 命中冻结协议声明的 infra 退出码 %d(文档化 wiring/infra 错误,不伪造为 gate FAIL)", raw.ExitCode)
			errSeen = true
		case raw.ExitCode != 0:
			g.Status = VisibleStatusFail
		default:
			g.Status = VisibleStatusPass
		}
		out = append(out, g)
	}
	return out, errSeen
}

// executeHeldOut 把契约中全部 held_out binding 经 HoldoutExecutor 注入 Tally
// 控制的最终验证。任一注入失败(内容缺失/完整性失败/边界错误/取消)返回
// fatal error:注入一次性消耗,失败不产生 verdict 也不自动重试(AC5,决策锁 12),
// 半次执行不得落库。
func (o *Orchestrator) executeHeldOut(ctx context.Context, req *FinalRunRequest, f *Frozen) ([]HoldoutVerdictRecord, bool, error) {
	ids := frozenHeldOutBindingIDs(f)
	if len(ids) == 0 {
		return nil, false, nil
	}
	hx, err := NewHoldoutExecutor(f, req.ChangeSet, o.reader, o.holdoutRun)
	if err != nil {
		return nil, false, fmt.Errorf("verification: 构造 holdout executor: %w", err)
	}
	recs := make([]HoldoutVerdictRecord, 0, len(ids))
	for _, id := range ids {
		b, _, err := findFrozenBinding(f, id)
		if err != nil {
			return nil, false, err // 不可能:frozenHeldOutBindingIDs 已保证存在
		}
		xctx := ExecutionContext{
			ContractRevision: f.Revision(),
			ContractHash:     f.Hash(),
			BindingID:        id,
			CandidateID:      string(req.ChangeSet.ID()),
			Protocol:         *b.ExecutionProtocol,
			Environment:      req.Environment,
		}
		rec, _, err := hx.Execute(ctx, id, xctx)
		if err != nil {
			return nil, false, fmt.Errorf("verification: binding %s 的 held-out 最终注入失败(一次性,无自动重试): %w", id, err)
		}
		recs = append(recs, rec)
	}
	return recs, false, nil
}

// loadPriorFacts 经 RunSink 读回既往 valid 且 applicable 的 evidence 事实
// (逐 acceptance 精确 scope,§8.8.1/8.8.2)。读失败 = 基础设施失败:中止,
// 不落库——在看不到既往事实的情况下推导覆盖会伪造结论。
func (o *Orchestrator) loadPriorFacts(ctx context.Context, req *FinalRunRequest, f *Frozen) ([]PriorFact, error) {
	c := f.Contract()
	var out []PriorFact
	for i := range c.Acceptances {
		scope := evidence.Scope{
			Project:      req.Project,
			Task:         evidence.TaskID(f.TaskID()),
			Acceptance:   evidence.AcceptanceID(c.Acceptances[i].ID),
			Verification: evidence.VerificationRevision(f.Hash()),
			ChangeSet:    evidence.ChangeSetID(req.ChangeSet.ID()),
		}
		facts, err := o.sink.ValidApplicableFacts(ctx, scope)
		if err != nil {
			return nil, fmt.Errorf("verification: 读回既往 evidence(acceptance=%s): %w", c.Acceptances[i].ID, err)
		}
		out = append(out, facts...)
	}
	return out, nil
}

// deriveIndependence 组装 §8.5.2 的求值输入并推导 effective independence
// coverage:binding 声明(basis/authoring mode/visibility)以冻结契约为权威
// (INV-VER-01/02),attestation/判定投影/exposure 由控制面提供。判定投影合并
// 本轮 verdict 与既往 valid 事实(selection exposure 时钟因此跨 Run 连续,
// §8.6.2)。
func (o *Orchestrator) deriveIndependence(req *FinalRunRequest, f *Frozen, bindings []BindingOutcome, holdoutRecs []HoldoutVerdictRecord, priors []PriorFact, finishedAt time.Time) (IndependenceCoverageReport, error) {
	ictx := IndependenceContext{
		Project:          req.Project,
		Verification:     evidence.VerificationRevision(f.Hash()),
		ChangeSet:        evidence.ChangeSetID(req.ChangeSet.ID()),
		CandidateFixedAt: req.CandidateFixedAt,
		Lineage:          req.Lineage,
		Implementation:   req.Implementation,
		Precommit:        req.Precommit,
	}

	task := evidence.TaskID(f.TaskID())
	verdicts := make([]evidence.VerdictRecord, 0, len(bindings)+len(holdoutRecs)+len(priors))
	for _, bo := range bindings {
		if bo.Err != "" {
			continue
		}
		verdicts = append(verdicts, evidence.VerdictRecord{
			Binding:    evidence.BindingRef{Project: req.Project, Verification: evidence.VerificationRevision(f.Hash()), BindingID: bo.BindingID},
			Task:       task,
			ChangeSet:  evidence.ChangeSetID(req.ChangeSet.ID()),
			Verdict:    evidence.Verdict(bo.Status),
			Valid:      true,
			RecordedAt: finishedAt,
		})
	}
	for i := range holdoutRecs {
		verdicts = append(verdicts, evidence.VerdictRecord{
			Binding:    evidence.BindingRef{Project: req.Project, Verification: evidence.VerificationRevision(f.Hash()), BindingID: holdoutRecs[i].BindingID},
			Task:       task,
			ChangeSet:  evidence.ChangeSetID(req.ChangeSet.ID()),
			Verdict:    evidence.Verdict(holdoutRecs[i].Status),
			Valid:      true,
			RecordedAt: holdoutRecs[i].CompletedAt,
		})
	}
	for _, p := range priors {
		verdicts = append(verdicts, evidence.VerdictRecord{
			Binding:    evidence.BindingRef{Project: req.Project, Verification: evidence.VerificationRevision(f.Hash()), BindingID: p.BindingID},
			Task:       task,
			ChangeSet:  evidence.ChangeSetID(req.ChangeSet.ID()),
			Verdict:    evidence.Verdict(p.Status),
			Valid:      true,
			RecordedAt: p.RecordedAt,
		})
	}

	inputs := make([]IndependenceEvidence, 0, 8)
	for _, cb := range frozenBindings(f) {
		att := req.Attestations[cb.binding.ID]
		inputs = append(inputs, IndependenceEvidence{
			BindingID:         cb.binding.ID,
			Revision:          f.Revision(),
			CandidateBlind:    att.CandidateBlind,
			AuthoredInLineage: att.AuthoredInLineage,
			ChangedInLineage:  att.ChangedInLineage,
			Verdicts:          verdicts,
			Exposures:         req.Exposures,
		})
	}
	rep, err := DeriveIndependenceCoverage(f, ictx, inputs)
	if err != nil {
		return IndependenceCoverageReport{}, fmt.Errorf("verification: 推导最终 independence 覆盖: %w", err)
	}
	return rep, nil
}

// heldOutBlockedBindings 返回功能覆盖被 valid+applicable FAIL 阻断的
// held-out binding id(本轮或既往事实;§8.8.1)。这些 binding 的任何失败细节
// 都不得进入实现侧通道(AC3)。
func heldOutBlockedBindings(f *Frozen, rep FunctionalCoverageReport) []string {
	visibility := map[string]VisibilityPolicy{}
	for _, cb := range frozenBindings(f) {
		visibility[cb.binding.ID] = cb.binding.VisibilityPolicy
	}
	var out []string
	for _, ac := range rep.Acceptances {
		if ac.Gap != GapBlockingFailure {
			continue
		}
		for _, id := range ac.Blocking {
			if visibility[id] == VisibilityHeldOut {
				out = append(out, id)
			}
		}
	}
	return out
}

// findFrozenBinding 在冻结契约中按 id 定位 binding。
func findFrozenBinding(f *Frozen, id string) (*Binding, string, error) {
	for _, cb := range frozenBindings(f) {
		if cb.binding.ID == id {
			b := cb.binding
			return &b, cb.acceptanceID, nil
		}
	}
	return nil, "", fmt.Errorf("%w: %s", ErrBindingNotFound, id)
}

// prepareCommand 把冻结的结构化命令准备为可执行形态:cwd 解析到 candidate
// workspace 之内(逃出即拒绝)、追加环境变量校验 KEY=VALUE、转换为
// internal/command.CommandSpec(Shell 恒为 false,shell 语义不可表达,决策锁 6)。
func prepareCommand(workspace string, spec *CommandSpec) (string, []string, command.CommandSpec, error) {
	if spec == nil {
		return "", nil, command.CommandSpec{}, errors.New("命令规格为 nil")
	}
	if err := spec.Validate(); err != nil {
		return "", nil, command.CommandSpec{}, err
	}
	dir, err := resolveWithinWorkspace(workspace, spec.Dir)
	if err != nil {
		return "", nil, command.CommandSpec{}, err
	}
	env, err := bindingEnv(spec.Env)
	if err != nil {
		return "", nil, command.CommandSpec{}, err
	}
	cs := command.CommandSpec{Argv: append([]string(nil), spec.Argv...), Cwd: dir, Shell: false}
	if err := cs.Validate(); err != nil {
		return "", nil, command.CommandSpec{}, err
	}
	return dir, env, cs, nil
}

// runWithinBudget 在剩余墙钟预算内执行一条结构化命令:单命令超时取「binding
// 声明的 timeout 与剩余预算的较小者」。
func (o *Orchestrator) runWithinBudget(ctx context.Context, cs command.CommandSpec, spec *CommandSpec, env []string, remaining time.Duration) (runner.Result, error) {
	timeout := time.Duration(spec.Timeout)
	if timeout == 0 || timeout > remaining {
		timeout = remaining
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return o.runner.Run(runCtx, runner.Command{Argv: cs.Argv, Dir: cs.Cwd, Env: env})
}

// buildFactSpecs 把本轮 verdict 汇总为 evidence 事实(append-only 投影):
// visible binding 的观察 + held-out 判定事实。FactID 按 (run, binding) 确定性
// 派生:同一 Run 的同一 binding 二次落库是身份冲突错误,不是更新(INV-FACT-01)。
func buildFactSpecs(runID int64, task string, recordedAt time.Time, recorder evidence.Recorder, bindings []BindingOutcome, holdoutRecs []HoldoutVerdictRecord) []FactSpec {
	out := make([]FactSpec, 0, len(bindings)+len(holdoutRecs))
	for _, bo := range bindings {
		if bo.Err != "" {
			continue // 无 verdict 的执行不产生事实
		}
		out = append(out, FactSpec{
			FactID:           evidence.FactID(fmt.Sprintf("vrun-%d-%s", runID, bo.BindingID)),
			RunID:            runID,
			Task:             task,
			Acceptance:       bo.AcceptanceID,
			BindingID:        bo.BindingID,
			VisibilityPolicy: bo.Visibility,
			Verdict:          bo.Status,
			Recorder:         recorder,
			Command:          bo.Command,
			ExitCode:         bo.ExitCode,
			StdoutDigest:     bo.StdoutDigest,
			StderrDigest:     bo.StderrDigest,
			RecordedAt:       recordedAt,
		})
	}
	for i := range holdoutRecs {
		h := &holdoutRecs[i]
		spec := FactSpec{
			FactID:           evidence.FactID(fmt.Sprintf("vrun-%d-%s", runID, h.BindingID)),
			RunID:            runID,
			Task:             task,
			Acceptance:       h.AcceptanceID,
			BindingID:        h.BindingID,
			VisibilityPolicy: VisibilityHeldOut,
			Verdict:          h.Status,
			Recorder:         recorder,
			RecordedAt:       recordedAt,
		}
		if n := len(h.OutputDigests); n > 0 {
			spec.StdoutDigest = h.OutputDigests[n-1] // 末次 repetition 的输出摘要
		}
		out = append(out, spec)
	}
	return out
}

// runStatusFor 把编排结论映射为 VerificationRun 行上的观测结果:needs_review
// 在行上映射为 error(运行级没有干净的 pass/fail 结论)。
func runStatusFor(s FinalStatus) RunStatus {
	switch s {
	case FinalVerified:
		return RunStatusPass
	case FinalFailed:
		return RunStatusFail
	default:
		return RunStatusError
	}
}
