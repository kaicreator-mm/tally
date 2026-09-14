// Package taskstate 实现 PRD v1.3 §18.1 的 TaskExecutionState 推导引擎(P1-22)。
//
// 核心规则(PRD §18.1 / INV-STATE-01/02/03 / INV-AUTH-01/02):
//
//   - 状态不得由 Agent 或 UI 直接写入:本包没有任何 setter,TaskExecutionState
//     只能由 Derive 从「当前 desired revision + observed evidence」纯函数推导,
//     同一事实集的任意输入顺序得到完全相同的 Result(INV-STATE-01 可重建);
//   - Agent claim 不是 Project Fact:经 implementation_agent 或未知通道记录的
//     “事实”一律不参与推导,只作为 claim 列入 Result.IgnoredClaims,因此
//     Agent 的任何声明(包括“我验证通过了”)都不能产生 VERIFIED(AC1);
//   - VERIFIED 至少要求(PRD §18.1):当前 Desired/Verification revision 的所有
//     applicable required functional gates 通过(P1-19 功能覆盖报告 Complete);
//     planned/effective independence coverage 满足 Policy,或存在被允许且显式
//     记录的 Independence Waiver(§8.10,只覆盖运行期 independence 损失);
//     protected/holdout integrity 无 unresolved violation;没有 applicable
//     unresolved failed evidence(AC2);
//   - APPROVED 虽然来自人的决策,但实现方式仍然是 Owner decision →
//     HumanApprovalEvidence → State Engine → APPROVED:没有人类通道记录的
//     HumanApprovalEvidence,状态推不到 APPROVED(AC3);
//   - 验证结论区分 strict_satisfied / strict_waived(§8.10,AC4):两条路径都
//     可以到达 VERIFIED,但 Result.VerificationClass 永远可区分,strict_waived
//     不得在任何 UI/报表中表述为「Strict requirement 已满足」;
//   - desired intent 与 observed evidence 分别可追溯(INV-STATE-03):快照的
//     desired 侧是冻结 VerificationContract 身份,evidence 侧的覆盖报告 /
//     waiver / approval 都必须绑定该身份;绑定其它 revision 的历史事实对当前
//     状态不可见(§8.8.2 Validity ≠ Applicability)。
//
// 边界(P1):
//   - 本包是纯函数领域推导,不做 IO、不持锁、不开 goroutine;持久化投影由
//     调用方(数据库/集成层)从 append-only 事实表组装;
//   - 功能覆盖与 independence 覆盖推导分别归 P1-19 / P1-20
//     (internal/verification),本包消费其报告,不重新推导;
//   - P1-08:PRD §14 的 VerifiedBasis(Decision Lock 12:derived read model)
//     在本包作为纯派生投影实现(Result.VerifiedBasis,见 basis.go),不新增
//     任何状态表、状态机或 setter;
//   - 只建模 TaskExecutionState;DeliveryState 在 P1-P3 不激活(PRD §18.2)。
package taskstate

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"tally/internal/contract"
	"tally/internal/evidence"
	"tally/internal/verification"
)

// State 是 PRD §18.1 的 TaskExecutionState。
type State string

const (
	// 主线状态。
	StatePlanned    State = "PLANNED"    // TaskContract 已创建,尚无冻结验证契约事实
	StateReady      State = "READY"      // 验证契约 freeze 已记录,可进入执行
	StateRunning    State = "RUNNING"    // CodegenRun 进行中
	StateCandidate  State = "CANDIDATE"  // 存在当前候选 ChangeSet
	StateValidating State = "VALIDATING" // 当前候选的验证 Run 进行中
	StateVerified   State = "VERIFIED"   // 验证门全部满足
	StateApproved   State = "APPROVED"   // HumanApprovalEvidence 已记录
	StateMerged     State = "MERGED"     // 当前候选已记录合并事实

	// 异常态。异常不是独立写入的标志位,同样由事实推导:
	//   - CANCELLED:取消事实存在即终态(没有解除路径);
	//   - NEEDS_REVIEW:存在未被解除的 review 请求(如预算耗尽、SpecDefect 提案);
	//   - BLOCKED:存在未被解除的显式阻塞;
	//   - FAILED:最近的执行事件是一次终态 Run 失败;新 Run 的事实自然覆盖它。
	StateBlocked     State = "BLOCKED"
	StateFailed      State = "FAILED"
	StateNeedsReview State = "NEEDS_REVIEW"
	StateCancelled   State = "CANCELLED"
)

// IsException 报告该状态是否是 §18.1 的异常态(非主线推进位置)。
func (s State) IsException() bool {
	switch s {
	case StateBlocked, StateFailed, StateNeedsReview, StateCancelled:
		return true
	default:
		return false
	}
}

// VerificationClass 是 PRD §8.10 的验证结论分类:strict_satisfied(independence
// 覆盖被证据满足)与 strict_waived(经被允许且显式记录的 Waiver 放行)必须
// 可区分。未到 VERIFIED 及以上时为空。
type VerificationClass string

const (
	ClassNone            VerificationClass = ""
	ClassStrictSatisfied VerificationClass = "strict_satisfied"
	ClassStrictWaived    VerificationClass = "strict_waived"
)

// FactKind 是推导引擎消费的 append-only 生命周期事实种类。刻意不存在
// "verified"/"approved" 这类可直接宣告终态的事实种类:VERIFIED 只能由覆盖
// 门槛推导,APPROVED 只能由 HumanApprovalEvidence 推导,没有可被 Agent 直接
// 写入的旁路(AC1/AC3 的结构性保证)。
type FactKind string

const (
	FactTaskPlanned            FactKind = "task_planned"             // TaskContract 已创建(RevisionRef = 任务合同内容哈希)
	FactVerificationFrozen     FactKind = "verification_frozen"      // 验证契约已冻结(RevisionRef = 冻结契约内容哈希)
	FactCodegenRunStarted      FactKind = "codegen_run_started"      // 实现 Run 开始(RunID)
	FactChangeSetProduced      FactKind = "changeset_produced"       // 产生候选 ChangeSet(RunID + ChangeSet)
	FactVerificationRunStarted FactKind = "verification_run_started" // 候选的验证 Run 开始(RunID + ChangeSet)
	FactCodegenRunFailed       FactKind = "codegen_run_failed"       // 实现 Run 终态失败(RunID)
	FactReviewRequested        FactKind = "review_requested"         // 请求人工 review(如预算耗尽/SpecDefect 提案)
	FactReviewResolved         FactKind = "review_resolved"          // 解除某次 review 请求(Resolves)
	FactTaskBlocked            FactKind = "task_blocked"             // 显式阻塞
	FactTaskUnblocked          FactKind = "task_unblocked"           // 解除某次阻塞(Resolves)
	FactTaskCancelled          FactKind = "task_cancelled"           // 取消(终态)
	FactMergeRecorded          FactKind = "merge_recorded"           // 候选已合并(ChangeSet)
)

// Fact 是一条生命周期事实的只读投影。所有事实都必须携带 fact-capable 的
// Recorder provenance;implementation_agent 与未知通道的“事实”不参与推导
// (INV-AUTH-01/02)。事实只追加,不修改;引擎对同一集合按 (At, Kind, ID)
// 规范排序后推导,与输入顺序无关。
type Fact struct {
	// ID 是事实身份,快照内唯一,用于 Result.Basis 追溯(INV-STATE-02)。
	ID string
	// Kind 是事实种类;未知种类是构造错误,不是数据。
	Kind FactKind
	// At 是事实发生时间(零值非法)。同一时刻的事实按 Kind、ID 稳定排序。
	At time.Time
	// Recorder 是记录 provenance;kind 必须 fact-capable。
	Recorder evidence.Recorder

	// RevisionRef 是事实绑定的 desired 制品内容身份:task_planned 绑定
	// TaskContract 内容哈希,verification_frozen 绑定冻结 VerificationContract
	// 内容哈希;其余种类必须为空(INV-STATE-03:desired 侧单独可追溯)。
	RevisionRef string
	// RunID 是 run 相关事实的实现 Run 身份;非 run 种类必须为空。
	RunID string
	// ChangeSet 是候选相关事实的 immutable ChangeSet 身份;非候选种类必须为空。
	ChangeSet evidence.ChangeSetID
	// Resolves 是解除类事实指向的被解除事实 id;非解除种类必须为空。
	Resolves string
}

// ApprovalDecision 是 HumanApprovalEvidence 的决定方向(PRD §18.1:Owner
// decision 是一种 Evidence,不是 UPDATE task SET status)。
type ApprovalDecision string

const (
	DecisionApprove ApprovalDecision = "approve"
	DecisionReject  ApprovalDecision = "reject"
)

// ApprovalEvidence 是一条 HumanApprovalEvidence 的只读投影。只有 decision=
// approve、绑定当前候选 ChangeSet、且经人类通道(recorder kind = human)记录
// 的证据才能推导 APPROVED;其它通道的“批准”是 claim,不是 Gate Authority 的
// 决定事实(PRD §2/§18.1)。
type ApprovalEvidence struct {
	ID        string
	Decision  ApprovalDecision
	ChangeSet evidence.ChangeSetID
	Recorder  evidence.Recorder
	At        time.Time
}

// WaiverEvidence 是一条 Independence Waiver 的只读投影(PRD §8.10):Waiver
// 只能覆盖运行期 independence 损失,必须绑定当前冻结契约 revision 与候选
// ChangeSet,枚举被放行的 compromised acceptances,并带 reason code 与 Owner
// 风险接受 provenance(人类通道)。Waiver 不改变 independence 事实,也覆盖
// 不了功能门、integrity 或 freeze 时的 planned coverage 缺失。
type WaiverEvidence struct {
	ID           string
	ContractHash string // 必须等于当前冻结契约内容哈希
	ChangeSet    evidence.ChangeSetID
	Acceptances  []string // 被放行的 acceptance id 集合
	ReasonCode   string
	Recorder     evidence.Recorder
	At           time.Time
}

// IntegrityKind 是 protected/holdout integrity 违规的种类(PRD §18.1 的
// integrity 条件;INV-VER-03 / INV-VER-05)。
type IntegrityKind string

const (
	// IntegrityProtectedBaseline:protected evidence 被其所授权的实现 Run 修改
	// (INV-VER-03)。
	IntegrityProtectedBaseline IntegrityKind = "protected_baseline_violation"
	// IntegrityHoldoutExposure:held-out 内容经实现侧通道暴露(§8.6,
	// INV-VER-05)。
	IntegrityHoldoutExposure IntegrityKind = "holdout_exposure"
)

// IntegrityViolation 是一条 integrity 违规事实的只读投影。ResolvedBy 非空
// 表示已有带 provenance 的正式解除记录;unresolved violation 阻断 VERIFIED,
// 且不可被 Waiver 覆盖(§8.10:Waiver 只覆盖 independence 损失)。
type IntegrityViolation struct {
	ID         string
	Kind       IntegrityKind
	Ref        string // 违规对象引用(binding id、exposure ref 等)
	ResolvedBy string // 解除记录 id;空 = 未解除
	Recorder   evidence.Recorder
	At         time.Time
}

// ApplicableFailure 是一条 valid 且 applicable(绑定当前 desired 上下文)的
// FAIL evidence 的只读投影(§8.8.1/§8.8.2)。ResolvedBy 非空表示已有带
// provenance 的正式裁定(holdout_defect / spec_defect / invalid_evidence)
// 使其失去 gating 效力;unresolved 的 applicable FAIL 阻断 VERIFIED,新 PASS
// 不能抹掉它。经非 fact-capable 通道“记录”的 FAIL 是 claim,既不阻断也不
// 参与推导(INV-AUTH-01)。
type ApplicableFailure struct {
	FactID     evidence.FactID
	BindingID  string
	ChangeSet  evidence.ChangeSetID
	ResolvedBy string
	Recorder   evidence.Recorder
	At         time.Time
}

// Requirement 是 VERIFIED 门槛未满足时的稳定机器词表(调用方按 Requirement
// 分支,不做字符串匹配推断)。
type Requirement string

const (
	ReqVerificationContract Requirement = "verification_contract" // 无当前冻结验证契约身份
	ReqFunctionalCoverage   Requirement = "functional_coverage"   // 功能覆盖未满足(P1-19 报告缺失/过期/不完整)
	ReqIndependenceCoverage Requirement = "independence_coverage" // independence 覆盖既未满足也无有效 Waiver
	ReqIntegrity            Requirement = "integrity"             // 存在 unresolved integrity violation
	ReqNoApplicableFailures Requirement = "applicable_failures"   // 存在 applicable unresolved failed evidence
	ReqCandidate            Requirement = "candidate"             // 无当前候选 ChangeSet
	ReqVerificationRun      Requirement = "verification_run"      // 当前候选没有验证 Run 事实
)

// Gap 是一条未满足的 VERIFIED 门槛要求(INV-STATE-02:状态结论可追溯到
// 具体缺口与证据)。
type Gap struct {
	Requirement Requirement
	Detail      string
	Refs        []string
}

// IgnoredClaim 是一条被引擎拒绝消费的声明:结构完整但 provenance 通道不具
// 备事实资格(INV-AUTH-01/02)。列出而非静默丢弃,使被拒绝的 Agent claim
// 可审计。
type IgnoredClaim struct {
	Ref    string
	Reason string
}

// IgnoredClaim 的稳定原因词表。
const (
	ClaimReasonRecorderNotFactCapable = "recorder_not_fact_capable"         // 生命周期事实经非事实通道记录
	ClaimReasonFailureNotFactCapable  = "failure_recorder_not_fact_capable" // FAIL 投影经非事实通道记录
	ClaimReasonApprovalNotHuman       = "approval_recorder_not_human"       // “批准”非人类通道记录
	ClaimReasonWaiverNotHuman         = "waiver_recorder_not_human"         // “Waiver”非人类通道记录
)

// Snapshot 是一次状态推导的输入投影:desired 侧(冻结验证契约)与 observed
// 侧(事实与证据投影)分别可追溯(INV-STATE-03)。
type Snapshot struct {
	// Contract 是当前 desired VerificationContract revision(冻结值)。nil =
	// 尚无冻结验证契约,主线上限为 PLANNED/READY 之前的推导路径。
	Contract *verification.Frozen

	// WaiverPermitted 是 Project Verification Policy 对 Independence Waiver
	// 的许可投影(§8.10:Policy 决定 Waiver 是否可用)。false 时任何 Waiver
	// 证据都不产生 strict_waived。
	WaiverPermitted bool

	// Facts 是 append-only 生命周期事实投影(顺序无关,引擎规范化排序)。
	Facts []Fact

	// Functional 是 P1-19 的功能覆盖报告(必须绑定当前契约身份才被消费)。
	Functional *verification.FunctionalCoverageReport
	// Independence 是 P1-20 的 effective independence 覆盖报告(同上)。
	Independence *verification.IndependenceCoverageReport

	// Approvals / Waivers / Violations / Failures 是对应证据事实的只读投影。
	Approvals  []ApprovalEvidence
	Waivers    []WaiverEvidence
	Violations []IntegrityViolation
	Failures   []ApplicableFailure

	// Compiled 是 v1.4 的已编译合同(P1-07 seam 的可选投影输入;P1-08)。
	// nil = 未接入,保持 v1.3 推导,Result.VerifiedBasis 为 nil(向后兼容)。
	// 非 nil 时必须与 Contract 是同一冻结 VerificationContract 内容修订
	// (§8.8.2:不跨 revision 投影),否则 Derive 返回
	// ErrCompiledMismatch。VerifiedBasis 投影只读消费它,不引入任何新的
	// 状态权威(Decision Lock 12)。
	Compiled *contract.Compiled
}

// Result 是一次确定性推导的输出。
type Result struct {
	// State 是推导出的 TaskExecutionState(PRD §18.1)。
	State State
	// VerificationClass 是当前候选的验证结论(§8.10);未到 VERIFIED 及以上
	// 时为空。strict_waived 不得在任何 UI/报表中表述为「Strict 已满足」。
	VerificationClass VerificationClass
	// CurrentRun / CurrentCandidate 是规范顺序下最新的实现 Run 与候选身份。
	CurrentRun       string
	CurrentCandidate evidence.ChangeSetID
	// ContractHash 是当前 desired 验证契约内容身份;无冻结契约时为空
	// (INV-STATE-03:desired 侧单独可追溯)。
	ContractHash string
	// Basis 是支撑本次状态结论的事实/证据引用(fact:<id> / approval:<id> /
	// waiver:<id> / functional_report:<hash> / independence_report:<hash> /
	// policy:waiver_permitted / compiled:<hash>,按字典序稳定排序
	// (INV-STATE-02)。
	Basis []string
	// Gaps 说明当前候选为何还未 VERIFIED(仅在主线未达 VERIFIED 时非空;
	// 异常态下为空)。按 Requirement 稳定顺序排列。
	Gaps []Gap
	// IgnoredClaims 是被拒绝消费的声明(非 fact-capable 通道),按 Ref 稳定
	// 排序(INV-AUTH-01/02 的可审计落点)。
	IgnoredClaims []IgnoredClaim
	// VerifiedBasis 是 PRD §14 的 v1.4 验证结论投影(P1-08;Decision Lock 12:
	// 派生只读模型,不是新的状态权威)。仅在 Snapshot.Compiled 非 nil 时产出;
	// nil 表示本次推导未接入 v1.4 合同编译面(v1.3 兼容)。VERIFIED 及以上
	// 状态若存在未匹配风险、被豁免条款或保证缺口,Verdict 必须是
	// VERIFIED_LIMITED,不得隐藏(INV-07:satisfied 与 waived 永不合并;
	// INV-11:verified-like 结论必须带依据)。
	VerifiedBasis *VerifiedBasis
}

// 领域错误。一律用 errors.Is 匹配,不做字符串匹配。
var (
	// ErrEmptySnapshot:事实投影为空,没有任何可推导的任务事实基础。
	ErrEmptySnapshot = errors.New("taskstate: 事实投影为空,无法推导任务状态")
	// ErrNoTaskFact:缺少 task_planned 事实;没有已创建的任务合同事实,一切
	// 状态推导都无事实基础(INV-STATE-02)。
	ErrNoTaskFact = errors.New("taskstate: 缺少 task_planned 事实")
	// ErrMalformedFact:事实/证据投影结构非法(空 id、零时间、字段与种类
	// 不符等)。构造错误显式报错,不做静默修正。
	ErrMalformedFact = errors.New("taskstate: 非法事实投影")
	// ErrDuplicateFactID:事实 id 重复。事实身份唯一,重复是调用方错误。
	ErrDuplicateFactID = errors.New("taskstate: 事实 id 重复")
	// ErrUnknownFactKind:未知事实种类。未知种类是代码错误,不是数据。
	ErrUnknownFactKind = errors.New("taskstate: 未知事实种类")
	// ErrCompiledMismatch(P1-08):Snapshot.Compiled 绑定的冻结
	// VerificationContract 与当前冻结契约不是同一内容修订(或缺失当前冻结
	// 契约身份)。VerifiedBasis 不跨 revision 投影(§8.8.2),缺失身份显式
	// 报错,不做静默降级。
	ErrCompiledMismatch = errors.New("taskstate: Compiled 与当前冻结验证契约非同一内容修订")
)

// Derive 从当前 desired revision 与 observed evidence 推导 TaskExecutionState。
// 纯函数:不修改输入;同一事实集的任意输入顺序产生完全相同的 Result
// (INV-STATE-01)。所有结构性错误显式返回;除此之外一切结论都在 Result 中,
// 不做静默修正。
func Derive(s Snapshot) (Result, error) {
	res := Result{}

	if len(s.Facts) == 0 {
		return Result{}, ErrEmptySnapshot
	}

	// ---- 0. v1.4 投影输入的适用性(§8.8.2):Compiled 必须绑定与当前冻结契约
	// 相同的内容修订,否则拒绝推导(不跨 revision 投影,不静默降级)。
	if s.Compiled != nil {
		compiledHash := ""
		if cs := s.Compiled.Snapshot(); cs != nil && cs.Verification() != nil {
			compiledHash = cs.Verification().Hash()
		}
		currentHash := ""
		if s.Contract != nil {
			currentHash = s.Contract.Hash()
		}
		if compiledHash == "" || currentHash == "" || compiledHash != currentHash {
			return Result{}, fmt.Errorf("%w: compiled %q vs 当前冻结契约 %q", ErrCompiledMismatch, compiledHash, currentHash)
		}
	}

	// ---- 1. 校验并分拣事实:fact-capable 通道进入推导,其余是 ignored claim。
	claims := make([]Fact, 0, len(s.Facts))
	seen := map[string]bool{}
	for _, f := range s.Facts {
		if err := f.validate(); err != nil {
			return Result{}, err
		}
		if seen[f.ID] {
			return Result{}, fmt.Errorf("%w: %s", ErrDuplicateFactID, f.ID)
		}
		seen[f.ID] = true
		if !f.Recorder.Kind.CanRecordFacts() {
			res.IgnoredClaims = append(res.IgnoredClaims, IgnoredClaim{
				Ref:    factRef(f.ID),
				Reason: ClaimReasonRecorderNotFactCapable,
			})
			continue
		}
		claims = append(claims, f)
	}
	if len(claims) == 0 {
		return Result{}, ErrNoTaskFact
	}
	sortFacts(claims)

	// task_planned 是一切状态推导的事实基础(INV-STATE-02:PLANNED 也必须有
	// 证据可追溯的创建事实)。
	hasPlanned := false
	for i := range claims {
		if claims[i].Kind == FactTaskPlanned {
			hasPlanned = true
			break
		}
	}
	if !hasPlanned {
		return Result{}, ErrNoTaskFact
	}

	// ---- 2. 校验证据投影。
	approvals := make([]ApprovalEvidence, 0, len(s.Approvals))
	for _, a := range s.Approvals {
		if err := a.validate(); err != nil {
			return Result{}, err
		}
		if a.Recorder.Kind != evidence.RecorderHuman {
			res.IgnoredClaims = append(res.IgnoredClaims, IgnoredClaim{
				Ref:    approvalRef(a.ID),
				Reason: ClaimReasonApprovalNotHuman,
			})
			continue
		}
		approvals = append(approvals, a)
	}
	waivers := make([]WaiverEvidence, 0, len(s.Waivers))
	for _, w := range s.Waivers {
		if err := w.validate(); err != nil {
			return Result{}, err
		}
		if w.Recorder.Kind != evidence.RecorderHuman {
			res.IgnoredClaims = append(res.IgnoredClaims, IgnoredClaim{
				Ref:    waiverRef(w.ID),
				Reason: ClaimReasonWaiverNotHuman,
			})
			continue
		}
		waivers = append(waivers, w)
	}
	violations := make([]IntegrityViolation, 0, len(s.Violations))
	for _, v := range s.Violations {
		if err := v.validate(); err != nil {
			return Result{}, err
		}
		violations = append(violations, v)
	}
	failures := make([]ApplicableFailure, 0, len(s.Failures))
	for _, fl := range s.Failures {
		if err := fl.validate(); err != nil {
			return Result{}, err
		}
		if !fl.Recorder.Kind.CanRecordFacts() {
			res.IgnoredClaims = append(res.IgnoredClaims, IgnoredClaim{
				Ref:    string(fl.FactID),
				Reason: ClaimReasonFailureNotFactCapable,
			})
			continue
		}
		failures = append(failures, fl)
	}

	// ---- 3. 规范顺序扫描:得到“当前”指针与 open holds。
	var (
		curPlanned   *Fact
		curFrozen    *Fact
		curRun       *Fact
		curCandidate *Fact
		curVerifRun  *Fact
		curMerge     *Fact
		curCancelled *Fact
		// latestExecKind 是最近一次执行位置事实的种类(FAILED 推导用)。
		latestExecKind FactKind
		latestExec     *Fact
	)
	var openReviews []string // 未解除 review 请求的事实 id(规范顺序)
	var openBlocks []string  // 未解除阻塞的事实 id(规范顺序)
	reviewOpen := map[string]bool{}
	blockOpen := map[string]bool{}
	for i := range claims {
		f := &claims[i]
		switch f.Kind {
		case FactTaskPlanned:
			curPlanned = f
		case FactVerificationFrozen:
			curFrozen = f
		case FactCodegenRunStarted:
			curRun = f
			latestExecKind, latestExec = f.Kind, f
		case FactChangeSetProduced:
			curCandidate = f
			curVerifRun = nil // 新候选使旧候选的验证 Run 事实不再是“当前候选的”
			latestExecKind, latestExec = f.Kind, f
		case FactVerificationRunStarted:
			// 只有针对当前候选的验证 Run 才推进 VALIDATING 位置;针对历史候选
			// 的是历史事实(§8.8.2),不参与当前位置。
			if curCandidate != nil && f.ChangeSet == curCandidate.ChangeSet {
				curVerifRun = f
				latestExecKind, latestExec = f.Kind, f
			}
		case FactCodegenRunFailed:
			latestExecKind, latestExec = f.Kind, f
		case FactReviewRequested:
			reviewOpen[f.ID] = true
			openReviews = append(openReviews, f.ID)
		case FactReviewResolved:
			// 解除只能关闭规范顺序在它之前的请求;之后的请求不受影响。
			if reviewOpen[f.Resolves] {
				delete(reviewOpen, f.Resolves)
				openReviews = removeID(openReviews, f.Resolves)
			}
		case FactTaskBlocked:
			blockOpen[f.ID] = true
			openBlocks = append(openBlocks, f.ID)
		case FactTaskUnblocked:
			if blockOpen[f.Resolves] {
				delete(blockOpen, f.Resolves)
				openBlocks = removeID(openBlocks, f.Resolves)
			}
		case FactTaskCancelled:
			curCancelled = f
		case FactMergeRecorded:
			curMerge = f
		}
	}

	// ---- 4. desired 身份与覆盖报告的适用性(INV-STATE-03 / §8.8.2):
	// 报告必须绑定当前冻结契约身份,否则是历史/过期投影,对当前状态不可见。
	var (
		contract      = s.Contract
		functionalOK  bool
		independence  bool // strict 满足(未经 Waiver)
		incomplete    []string
		functionalRef string
		indepRef      string
	)
	if contract != nil {
		res.ContractHash = contract.Hash()
	}
	functionalMatched := contract != nil && s.Functional != nil &&
		s.Functional.TaskID == contract.TaskID() &&
		s.Functional.ContractRevision == contract.Revision() &&
		s.Functional.ContractHash == contract.Hash()
	if functionalMatched {
		functionalOK = s.Functional.Complete
		functionalRef = functionalReportRef(contract.Hash())
	}
	indepMatched := contract != nil && s.Independence != nil &&
		s.Independence.TaskID == contract.TaskID() &&
		s.Independence.ContractRevision == contract.Revision() &&
		s.Independence.ContractHash == contract.Hash()
	if indepMatched {
		indepRef = independenceReportRef(contract.Hash())
		if s.Independence.Complete {
			independence = true
		} else {
			for _, a := range s.Independence.Acceptances {
				if !a.Covered {
					incomplete = append(incomplete, a.AcceptanceID)
				}
			}
			sort.Strings(incomplete)
		}
	}

	// 当前候选上下文内的 applicable unresolved 失败与 integrity 违规。
	var blockingFailures []string
	if curCandidate != nil {
		for _, fl := range failures {
			if fl.ChangeSet == curCandidate.ChangeSet && fl.ResolvedBy == "" {
				blockingFailures = append(blockingFailures, string(fl.FactID))
			}
		}
	}
	sort.Strings(blockingFailures)
	var unresolvedViolations []string
	for _, v := range violations {
		if v.ResolvedBy == "" {
			unresolvedViolations = append(unresolvedViolations, v.ID)
		}
	}
	sort.Strings(unresolvedViolations)

	// Waiver 评估(§8.10):只在「报告适用且存在缺口」时才有可放行的对象;
	// 放行条件 = Policy 允许 + 人类通道证据 + 绑定当前契约哈希与当前候选 +
	// 覆盖全部缺口 acceptance。Waiver 永远覆盖不了功能门与 integrity。
	waived := false
	var waiverRefs []string
	// waivedAcceptances 是被有效 Waiver 放行的缺口 acceptance id(P1-08
	// VerifiedBasis 投影用;与 satisfied 永不合并,PRD §13.5)。incomplete
	// 已升序,输出保持升序。
	var waivedAcceptances []string
	if s.WaiverPermitted && len(incomplete) > 0 && curCandidate != nil && contract != nil {
		covered := map[string]bool{}
		for _, w := range waivers {
			if w.ContractHash != contract.Hash() || w.ChangeSet != curCandidate.ChangeSet {
				continue
			}
			waiverRefs = append(waiverRefs, waiverRef(w.ID))
			for _, a := range w.Acceptances {
				covered[a] = true
			}
		}
		waived = true
		for _, id := range incomplete {
			if !covered[id] {
				waived = false
				continue
			}
			waivedAcceptances = append(waivedAcceptances, id)
		}
	}
	sort.Strings(waiverRefs)

	// ---- 5. VERIFIED 门槛(§18.1 四条 + 生命周期一致性)。
	verified := contract != nil && functionalOK && (independence || waived) &&
		len(unresolvedViolations) == 0 && len(blockingFailures) == 0 &&
		curCandidate != nil && curVerifRun != nil

	// 审批与合并资格:绑定当前候选。
	var approvalRefs []string
	approved := false
	if verified && curCandidate != nil {
		for _, a := range approvals {
			if a.Decision == DecisionApprove && a.ChangeSet == curCandidate.ChangeSet {
				approved = true
				approvalRefs = append(approvalRefs, approvalRef(a.ID))
			}
		}
	}
	sort.Strings(approvalRefs)
	merged := false
	var mergeRef string
	if approved && curMerge != nil && curMerge.ChangeSet == curCandidate.ChangeSet {
		merged = true
		mergeRef = factRef(curMerge.ID)
	}

	// ---- 6. 状态推导:异常态优先,主线取最新执行位置。
	var basis []string
	basis = append(basis, factRef(curPlanned.ID))
	switch {
	case curCancelled != nil:
		// 取消是终态事实:任何后续事实都不能把任务推出 CANCELLED。
		res.State = StateCancelled
		basis = append(basis, factRef(curCancelled.ID))
	case len(openReviews) > 0:
		res.State = StateNeedsReview
		for _, id := range openReviews {
			basis = append(basis, factRef(id))
		}
	case len(openBlocks) > 0:
		res.State = StateBlocked
		for _, id := range openBlocks {
			basis = append(basis, factRef(id))
		}
	case latestExecKind == FactCodegenRunFailed:
		res.State = StateFailed
		basis = append(basis, factRef(latestExec.ID))
	default:
		switch latestExecKind {
		case FactVerificationRunStarted:
			res.State = StateValidating
		case FactChangeSetProduced:
			res.State = StateCandidate
		case FactCodegenRunStarted:
			res.State = StateRunning
		default:
			if curFrozen != nil {
				res.State = StateReady
			} else {
				res.State = StatePlanned
			}
		}
		if verified {
			res.State = StateVerified
			if approved {
				res.State = StateApproved
				if merged {
					res.State = StateMerged
				}
			}
		}
	}

	// ---- 7. 结论追溯(INV-STATE-02)。
	res.VerificationClass = ClassNone
	if res.State == StateVerified || res.State == StateApproved || res.State == StateMerged {
		if waived {
			res.VerificationClass = ClassStrictWaived
		} else {
			res.VerificationClass = ClassStrictSatisfied
		}
	}
	if curRun != nil && (res.State == StateRunning || res.State == StateCandidate ||
		res.State == StateValidating || res.State == StateVerified ||
		res.State == StateApproved || res.State == StateMerged) {
		basis = append(basis, factRef(curRun.ID))
	}
	if curCandidate != nil && (res.State == StateCandidate || res.State == StateValidating ||
		res.State == StateVerified || res.State == StateApproved || res.State == StateMerged) {
		basis = append(basis, factRef(curCandidate.ID))
	}
	if curVerifRun != nil && (res.State == StateValidating || res.State == StateVerified ||
		res.State == StateApproved || res.State == StateMerged) {
		basis = append(basis, factRef(curVerifRun.ID))
	}
	if curFrozen != nil && (res.State == StateReady || res.State == StateRunning ||
		res.State == StateCandidate || res.State == StateValidating ||
		res.State == StateVerified || res.State == StateApproved || res.State == StateMerged) {
		basis = append(basis, factRef(curFrozen.ID))
	}
	if res.State == StateMerged {
		basis = append(basis, mergeRef)
	}
	if res.State == StateApproved || res.State == StateMerged {
		basis = append(basis, approvalRefs...)
	}
	if res.State == StateVerified || res.State == StateApproved || res.State == StateMerged {
		basis = append(basis, functionalRef, indepRef)
		if waived {
			basis = append(basis, waiverRefs...)
			basis = append(basis, "policy:waiver_permitted")
		}
	}

	// ---- 7b. v1.4 VerifiedBasis 投影(PRD §14 / P1-08;Decision Lock 12):
	// 仅当调用方接入已编译合同时产出,纯派生只读模型;结论可经 compiled:<hash>
	// 追溯到已编译合同身份(INV-STATE-02)。
	if s.Compiled != nil {
		res.VerifiedBasis = projectVerifiedBasis(basisInput{
			compiled:             s.Compiled,
			contract:             contract,
			functional:           s.Functional,
			functionalMatched:    functionalMatched,
			state:                res.State,
			waivedAcceptances:    waivedAcceptances,
			unresolvedViolations: unresolvedViolations,
			blockingFailures:     blockingFailures,
		})
		basis = append(basis, "compiled:"+s.Compiled.Hash())
	}
	sort.Strings(basis)
	res.Basis = basis

	// ---- 8. 未满足门槛(仅主线未达 VERIFIED 时;异常态不解释门槛)。
	if !res.State.IsException() && !verified {
		if contract == nil {
			res.Gaps = append(res.Gaps, Gap{
				Requirement: ReqVerificationContract,
				Detail:      "快照没有当前冻结 VerificationContract,无法评估任何验证门",
			})
		}
		switch {
		case contract == nil:
			// 功能/independence 门无从评估,连同候选/Run 一起列出结构性缺口。
			res.Gaps = append(res.Gaps, Gap{
				Requirement: ReqFunctionalCoverage,
				Detail:      "无当前契约身份,功能覆盖报告不适用",
			})
			res.Gaps = append(res.Gaps, Gap{
				Requirement: ReqIndependenceCoverage,
				Detail:      "无当前契约身份,independence 覆盖报告不适用",
			})
		case !functionalOK:
			g := Gap{Requirement: ReqFunctionalCoverage}
			if !functionalMatched {
				g.Detail = "功能覆盖报告缺失或绑定其它 revision/任务(§8.8.2),对当前状态不可见"
			} else {
				g.Detail = "功能覆盖不完整:存在未被 applicable required evidence 覆盖或被 valid FAIL 阻断的 acceptance"
				for _, a := range s.Functional.Acceptances {
					if !a.Covered {
						g.Refs = append(g.Refs, acceptanceRef(a.AcceptanceID))
					}
				}
			}
			res.Gaps = append(res.Gaps, g)
		}
		if contract != nil && !independence && !waived {
			g := Gap{Requirement: ReqIndependenceCoverage}
			if !indepMatched {
				g.Detail = "independence 覆盖报告缺失或绑定其它 revision/任务(§8.8.2),对当前状态不可见"
			} else {
				g.Detail = "independence 覆盖未满足且无可放行 Waiver"
				g.Refs = append(g.Refs, acceptanceRefs(incomplete)...)
			}
			res.Gaps = append(res.Gaps, g)
		}
		if len(unresolvedViolations) > 0 {
			res.Gaps = append(res.Gaps, Gap{
				Requirement: ReqIntegrity,
				Detail:      "存在 unresolved protected/holdout integrity violation",
				Refs:        unresolvedViolations,
			})
		}
		if len(blockingFailures) > 0 {
			res.Gaps = append(res.Gaps, Gap{
				Requirement: ReqNoApplicableFailures,
				Detail:      "当前候选存在 applicable unresolved failed evidence(新 PASS 不抹掉合法 FAIL,§8.8.1)",
				Refs:        blockingFailures,
			})
		}
		if curCandidate == nil {
			res.Gaps = append(res.Gaps, Gap{
				Requirement: ReqCandidate,
				Detail:      "没有当前候选 ChangeSet 事实",
			})
		} else if curVerifRun == nil {
			res.Gaps = append(res.Gaps, Gap{
				Requirement: ReqVerificationRun,
				Detail:      "当前候选没有验证 Run 事实",
				Refs:        []string{factRef(curCandidate.ID)},
			})
		}
	}

	// ---- 9. 输出规范化(确定性)。
	res.CurrentRun = ""
	if curRun != nil {
		res.CurrentRun = curRun.RunID
	}
	res.CurrentCandidate = ""
	if curCandidate != nil {
		res.CurrentCandidate = curCandidate.ChangeSet
	}
	sortClaims(res.IgnoredClaims)
	return res, nil
}

// validate 校验生命周期事实的结构合法性。字段与种类不匹配视为构造错误
// (写错的字段是 bug,不是数据)。
func (f Fact) validate() error {
	switch {
	case strings.TrimSpace(f.ID) == "":
		return fmt.Errorf("%w: 事实 id 为空", ErrMalformedFact)
	case f.At.IsZero():
		return fmt.Errorf("%w: 事实 %s 缺少发生时间", ErrMalformedFact, f.ID)
	}
	kindFields := func(run, cs, ref, resolves bool) error {
		if run && strings.TrimSpace(f.RunID) == "" {
			return fmt.Errorf("%w: 事实 %s(%s)缺少 run_id", ErrMalformedFact, f.ID, f.Kind)
		}
		if !run && f.RunID != "" {
			return fmt.Errorf("%w: 事实 %s(%s)不应携带 run_id", ErrMalformedFact, f.ID, f.Kind)
		}
		if cs && f.ChangeSet == "" {
			return fmt.Errorf("%w: 事实 %s(%s)缺少 changeset", ErrMalformedFact, f.ID, f.Kind)
		}
		if !cs && f.ChangeSet != "" {
			return fmt.Errorf("%w: 事实 %s(%s)不应携带 changeset", ErrMalformedFact, f.ID, f.Kind)
		}
		if ref && strings.TrimSpace(f.RevisionRef) == "" {
			return fmt.Errorf("%w: 事实 %s(%s)缺少 revision_ref", ErrMalformedFact, f.ID, f.Kind)
		}
		if !ref && f.RevisionRef != "" {
			return fmt.Errorf("%w: 事实 %s(%s)不应携带 revision_ref", ErrMalformedFact, f.ID, f.Kind)
		}
		if resolves && strings.TrimSpace(f.Resolves) == "" {
			return fmt.Errorf("%w: 事实 %s(%s)缺少 resolves", ErrMalformedFact, f.ID, f.Kind)
		}
		if !resolves && f.Resolves != "" {
			return fmt.Errorf("%w: 事实 %s(%s)不应携带 resolves", ErrMalformedFact, f.ID, f.Kind)
		}
		return nil
	}
	switch f.Kind {
	case FactTaskPlanned:
		return kindFields(false, false, true, false)
	case FactVerificationFrozen:
		return kindFields(false, false, true, false)
	case FactCodegenRunStarted, FactCodegenRunFailed:
		return kindFields(true, false, false, false)
	case FactChangeSetProduced, FactVerificationRunStarted:
		return kindFields(true, true, false, false)
	case FactReviewRequested, FactTaskBlocked, FactTaskCancelled:
		return kindFields(false, false, false, false)
	case FactReviewResolved, FactTaskUnblocked:
		return kindFields(false, false, false, true)
	case FactMergeRecorded:
		return kindFields(false, true, false, false)
	default:
		return fmt.Errorf("%w: %q", ErrUnknownFactKind, f.Kind)
	}
}

func (a ApprovalEvidence) validate() error {
	switch {
	case strings.TrimSpace(a.ID) == "":
		return fmt.Errorf("%w: 审批证据 id 为空", ErrMalformedFact)
	case a.At.IsZero():
		return fmt.Errorf("%w: 审批证据 %s 缺少时间", ErrMalformedFact, a.ID)
	case a.ChangeSet == "":
		return fmt.Errorf("%w: 审批证据 %s 缺少 changeset", ErrMalformedFact, a.ID)
	}
	switch a.Decision {
	case DecisionApprove, DecisionReject:
	default:
		return fmt.Errorf("%w: 审批证据 %s 决定非法: %q(approve|reject)", ErrMalformedFact, a.ID, a.Decision)
	}
	return nil
}

func (w WaiverEvidence) validate() error {
	switch {
	case strings.TrimSpace(w.ID) == "":
		return fmt.Errorf("%w: waiver 证据 id 为空", ErrMalformedFact)
	case w.At.IsZero():
		return fmt.Errorf("%w: waiver 证据 %s 缺少时间", ErrMalformedFact, w.ID)
	case strings.TrimSpace(w.ContractHash) == "":
		return fmt.Errorf("%w: waiver 证据 %s 缺少契约哈希", ErrMalformedFact, w.ID)
	case w.ChangeSet == "":
		return fmt.Errorf("%w: waiver 证据 %s 缺少 changeset", ErrMalformedFact, w.ID)
	case strings.TrimSpace(w.ReasonCode) == "":
		return fmt.Errorf("%w: waiver 证据 %s 缺少 reason_code(§8.10)", ErrMalformedFact, w.ID)
	case len(w.Acceptances) == 0:
		return fmt.Errorf("%w: waiver 证据 %s 未枚举任何 compromised acceptance(§8.10)", ErrMalformedFact, w.ID)
	}
	for _, a := range w.Acceptances {
		if strings.TrimSpace(a) == "" {
			return fmt.Errorf("%w: waiver 证据 %s 含空白 acceptance id", ErrMalformedFact, w.ID)
		}
	}
	return nil
}

func (v IntegrityViolation) validate() error {
	switch {
	case strings.TrimSpace(v.ID) == "":
		return fmt.Errorf("%w: integrity 违规 id 为空", ErrMalformedFact)
	case v.At.IsZero():
		return fmt.Errorf("%w: integrity 违规 %s 缺少时间", ErrMalformedFact, v.ID)
	case strings.TrimSpace(v.Ref) == "":
		return fmt.Errorf("%w: integrity 违规 %s 缺少对象引用", ErrMalformedFact, v.ID)
	}
	switch v.Kind {
	case IntegrityProtectedBaseline, IntegrityHoldoutExposure:
	default:
		return fmt.Errorf("%w: integrity 违规 %s 种类非法: %q", ErrMalformedFact, v.ID, v.Kind)
	}
	return nil
}

func (fl ApplicableFailure) validate() error {
	switch {
	case fl.FactID == "":
		return fmt.Errorf("%w: 失败证据缺少 fact id", ErrMalformedFact)
	case strings.TrimSpace(fl.BindingID) == "":
		return fmt.Errorf("%w: 失败证据 %s 缺少 binding id", ErrMalformedFact, fl.FactID)
	case fl.ChangeSet == "":
		return fmt.Errorf("%w: 失败证据 %s 缺少 changeset", ErrMalformedFact, fl.FactID)
	case fl.At.IsZero():
		return fmt.Errorf("%w: 失败证据 %s 缺少时间", ErrMalformedFact, fl.FactID)
	}
	return nil
}

// sortFacts 是事实的规范全序:(At, Kind, ID)。id 在快照内唯一,因此全序
// 唯一,任何输入顺序都得到同一处理序列(INV-STATE-01)。
func sortFacts(fs []Fact) {
	sort.Slice(fs, func(i, j int) bool {
		a, b := fs[i].At.UTC(), fs[j].At.UTC()
		if !a.Equal(b) {
			return a.Before(b)
		}
		if fs[i].Kind != fs[j].Kind {
			return fs[i].Kind < fs[j].Kind
		}
		return fs[i].ID < fs[j].ID
	})
}

func sortClaims(claims []IgnoredClaim) {
	sort.Slice(claims, func(i, j int) bool { return claims[i].Ref < claims[j].Ref })
}

// removeID 从保持顺序的 id 切表中移除第一个等于 id 的条目;不存在时原样返回。
func removeID(ids []string, id string) []string {
	for k, v := range ids {
		if v == id {
			return append(ids[:k], ids[k+1:]...)
		}
	}
	return ids
}

func acceptanceRefs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, acceptanceRef(id))
	}
	return out
}

func acceptanceRef(id string) string        { return "acceptance:" + id }
func factRef(id string) string              { return "fact:" + id }
func approvalRef(id string) string          { return "approval:" + id }
func waiverRef(id string) string            { return "waiver:" + id }
func functionalReportRef(h string) string   { return "functional_report:" + h }
func independenceReportRef(h string) string { return "independence_report:" + h }
