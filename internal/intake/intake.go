// Package intake 是 v1.4 的最小意图入口(PRD §6 / §27,P3-07):
// 一句自然语言(+可选 source_ref)进,一份**结构化的 Contract 草案**出。
//
// 边界与不变量:
//
//   - 自然语言优先、记录结构化(PRD §27):输入只有一句意图文本,可选外部来源
//     引用;低风险支持场景不要求任何手工必填字段(AC1)。
//   - 通用推理一律走 internal/airuntime 边界(DECISION_LOCKS #17 / ADR-0002 /
//     PRD §28):本包只声明 operation / intent / input / budget 加幂等键;
//     provider/model/tier/凭证是公共 AI Runtime 侧配置,不出现在本包。
//   - 推理只产草案(铁律 1 / INV-1 同款):返回的 Proposal 永远是给 owner 确认的
//     DRAFT,不是权威合同;验收冻结仍由既有 requirement/contract 路径负责
//     (INV-02),本包不提供任何冻结/接受路径。
//   - 未知/含糊意图产出澄清问题,绝不发明验收(AC3):推理输出要么是完整草案、
//     要么是澄清问题,二者混出或都不给 → 类型化错误,fail safe,不猜测。
//   - 只支持两个 MVP 任务类型:bug_fix(缺陷修复)/ db_migration(数据库迁移)
//     (AC4);其它任务类型一律拒绝,不静默降级。
//
// 语义校验(输出对不对)留在本调用方包内,airuntime 不做判定(INV-1);
// 输出解析用严格模式,形状非法即类型化报错,绝不做 best-effort 猜测。
package intake

import "errors"

// TaskType 是 MVP 支持的两个任务类型(PRD §24.1 / §24.2,AC4)。
// 取值是推理输出契约的一部分,保持稳定;超出集合的值一律拒绝(AC4 fail safe)。
type TaskType string

const (
	// TaskTypeBugFix 是缺陷修复闭环(PRD §24.1)。
	TaskTypeBugFix TaskType = "bug_fix"
	// TaskTypeMigration 是数据库迁移闭环(PRD §24.2)。
	TaskTypeMigration TaskType = "db_migration"
)

// taskTypeOf 把推理输出的字符串收敛为合法 TaskType;不在两个 MVP 值之内
// (含空串)一律不合法,调用方按 AC4 fail safe 拒绝,不猜。
func taskTypeOf(s string) (TaskType, bool) {
	switch t := TaskType(s); t {
	case TaskTypeBugFix, TaskTypeMigration:
		return t, true
	default:
		return "", false
	}
}

// Intent 是一次意图输入(PRD §6.1):一句自然语言 + 可选外部来源引用。
// 没有任何手工必填的结构化字段(AC1);外部来源只作参考,v1.4 不把外部
// Issue 搬进 Tally(PRD §6.1)。
type Intent struct {
	// Text 是那句自然语言意图(如「修复 logout 后 session 仍然有效的问题」)。
	Text string `json:"text"`
	// SourceRef 是可选的外部来源引用(如 "GitHub Issue #142");可空。
	SourceRef string `json:"source_ref,omitempty"`
}

// Scope 是草案声明的实现边界(与 taskcontract.Scope 同语义:仓库内斜杠风格
// 相对路径;通配语义由下游 scope 校验/执行方负责)。草案允许为空边界之外的
// 字段,但 allowed_paths 必须非空 —— 合同必须声明实现边界(PRD §7.3)。
type Scope struct {
	AllowedPaths   []string `json:"allowed_paths"`
	ForbiddenPaths []string `json:"forbidden_paths"`
}

// AcceptanceClause 是一条**草案**验收条款:只有 id 与可判定陈述。
// 刻意没有 independence_required / assurance 字段 —— 独立性与保证等级是
// 冻结/编译路径(intake policy 与 P1-07 compiler)的治理决策,不由推理发明。
type AcceptanceClause struct {
	// ID 是条款标识,草案内唯一,约定 AC-01 递增。
	ID string `json:"id"`
	// Statement 是可判定的验收陈述(能写出通过/不通过的检查)。
	Statement string `json:"statement"`
}

// ClarificationKind 是澄清字段的有界类型词表(DL-11 / PRD §3.2 受限
// Presentation Schema):动态表单只是**展示 schema**,词表是封闭枚举,
// 之外的类型一律 fail-safe 拒绝;没有 untyped custom control,也没有任何
// HTML/JS 通道(PRD §8.4)。
type ClarificationKind string

const (
	// KindChoice 单选:从 Candidates 里选一个(旧版简单候选澄清的归一形态)。
	KindChoice ClarificationKind = "choice"
	// KindMultiChoice 多选:从 Candidates 里选多个。
	KindMultiChoice ClarificationKind = "multichoice"
	// KindBoolean 是/否二元澄清。
	KindBoolean ClarificationKind = "boolean"
	// KindNumber 数值澄清,可选 min/max 闭区间。
	KindNumber ClarificationKind = "number"
	// KindRepository 仓库路径澄清(选项即候选仓库路径)。
	KindRepository ClarificationKind = "repository"
	// KindBranch 分支澄清(选项即候选分支)。
	KindBranch ClarificationKind = "branch"
	// KindRiskChoice 风险权衡澄清(选项即候选取舍;推荐仍是 advisory)。
	KindRiskChoice ClarificationKind = "risk_choice"
	// KindConfirmation 确认型澄清(owner 显式确认/否决一个既定提法)。
	KindConfirmation ClarificationKind = "confirmation"
	// KindFreeText 自由文本澄清 —— 自然语言逃生舱(PRD §8.1):无法结构化
	// 的上下文永远可以用一句自然语言补充。
	KindFreeText ClarificationKind = "free_text"
)

// clarificationKindOf 把 wire 上的 kind 字符串收敛为合法 ClarificationKind;
// 不在封闭词表之内(含空串)一律不合法,由调用方 fail-safe 拒绝(DL-11)。
func clarificationKindOf(s string) (ClarificationKind, bool) {
	switch k := ClarificationKind(s); k {
	case KindChoice, KindMultiChoice, KindBoolean, KindNumber,
		KindRepository, KindBranch, KindRiskChoice, KindConfirmation, KindFreeText:
		return k, true
	default:
		return "", false
	}
}

// Clarification 是针对意图含糊处向 owner 提出的一个问题,尽量带候选答案
// (选择题优于问答题,与 requirement 问询一致)。澄清不是失败,是 intake
// 的一等产物(AC3):owner 回答后才可能产出草案。
//
// v1.4.1 起(DL-11 / PRD §3.2)它同时是有界 Dynamic Form 的展示 schema:
// Kind 收敛封闭词表,Recommended/RecommendationReason 只是 advisory 的生成
// 建议(PRD §8.3),字段值永远是纯文本数据 —— 结构里没有任何标记/脚本通道,
// 不存在 arbitrary HTML/JS 的承载面(PRD §8.4);它也不定义任何领域契约
// (DL-11:presentation ≠ domain schema),权威字段面仍由 Draft/既有合同
// 路径承担。
type Clarification struct {
	// Question 是问 owner 的问题本身。
	Question string `json:"question"`
	// Candidates 是候选答案,可为空(开放问题)。
	Candidates []string `json:"candidates,omitempty"`
	// Kind 是有界字段类型(PRD §3.2 封闭词表)。零值空串由校验层按旧形态
	// 显式迁移:带 candidates → choice,不带 → free_text(AC:旧简单候选
	// 澄清保持兼容);词表之外的值一律 ErrUnknownClarificationKind。
	Kind ClarificationKind `json:"kind,omitempty"`
	// Recommended 是 advisory 推荐项在 Candidates 中的下标(PRD §8.3):
	// 仅对带候选的类型有意义,越界/用在不匹配的类型上一律拒绝。建议不是
	// 决定 —— 最终裁决来自 owner 或明确 policy。
	Recommended *int `json:"recommended,omitempty"`
	// RecommendationReason 是推荐理由(advisory,纯文本;PRD §8.3),必须
	// 伴随 Recommended 出现。
	RecommendationReason string `json:"recommendation_reason,omitempty"`
	// Min/Max 是 KindNumber 的可选闭区间;同时给出且 Min > Max 一律拒绝。
	Min *float64 `json:"min,omitempty"`
	Max *float64 `json:"max,omitempty"`
}

// Draft 是推理产出的结构化 Contract 草案,字段一一对应 PRD §6.2 的
// 结构化 Diff(intent summary / initial acceptance / scope / non-scope /
// constraints / task type / candidate risk signals)。
//
// 它是给 owner 确认的 DRAFT:没有任何判定字段(verified/frozen/approved
// 都不存在),算不算数由 owner 确认后的既有冻结路径决定(AC2,INV-02)。
type Draft struct {
	// TaskType 是草案归入的 MVP 任务类型(AC4:只有两个合法值)。
	TaskType TaskType `json:"task_type"`
	// Summary 是意图摘要(一句话说清要做什么)。
	Summary string `json:"summary"`
	// Objective 是要达成什么(WHY/WHAT),对齐 taskcontract.Objective。
	Objective string `json:"objective"`
	// Scope 是实现边界;AllowedPaths 必须非空。
	Scope Scope `json:"scope"`
	// NonScope 是明确不做什么(目标层面的非目标)。
	NonScope []string `json:"non_scope,omitempty"`
	// Constraints 是约束(兼容性、性能、安全等)。
	Constraints []string `json:"constraints,omitempty"`
	// Acceptance 是初始验收条款草案,至少一条(给 owner 一个可确认的 diff)。
	Acceptance []AcceptanceClause `json:"acceptance"`
	// RiskSignals 是候选风险信号(供下游风险分级参考;只是候选,不作数)。
	RiskSignals []string `json:"risk_signals,omitempty"`
}

// Proposal 是一次意图结构化的类型化结果:要么是给 owner 确认的草案
// (DraftProposal),要么是澄清请求(ClarificationProposal)。二者互斥,
// 由 Go 类型保证 —— 不存在「半个草案 + 半个问题」的中间态(AC3)。
type Proposal interface {
	isProposal()
}

// DraftProposal 携带完整草案。owner 确认并经既有冻结路径之后它才成为权威
// 合同;在本包里它永远只是提案(AC2)。
type DraftProposal struct {
	Draft Draft
}

// ClarificationProposal 携带澄清问题:意图未知/含糊时的产物,
// 绝不附带任何发明出来的验收(AC3)。
type ClarificationProposal struct {
	Questions []Clarification
}

func (DraftProposal) isProposal()         {}
func (ClarificationProposal) isProposal() {}

// 包内类型化错误的哨兵值:调用方用 errors.Is/errors.As 判类,不匹配错误文本。
var (
	// ErrEmptyIntent:意图文本为空/全空白。没有意图就没有 intake(AC1 的
	// 唯一必填项就是这句话)。
	ErrEmptyIntent = errors.New("intake: 意图文本为空")
	// ErrIntentTooLong:意图文本超过长度上限(有界输入,防失控提示词)。
	ErrIntentTooLong = errors.New("intake: 意图文本超过长度上限")
	// ErrNoExecutor:composition root 未注入 airuntime.Executor。
	ErrNoExecutor = errors.New("intake: 未注入 airuntime.Executor")
	// ErrUnsupportedTaskType:推理给出的任务类型不在两个 MVP 类型之内
	// (AC4:只支持 bug_fix / db_migration,其余一律 fail safe 拒绝)。
	ErrUnsupportedTaskType = errors.New("intake: 任务类型不在 MVP 支持范围内(bug_fix/db_migration)")
	// ErrUnknownClarificationKind:推理给出的澄清字段类型不在有界词表之内
	// (DL-11 / PRD §8.4:没有 untyped custom control,未知类型 fail safe
	// 拒绝,绝不猜测成某个已知类型)。
	ErrUnknownClarificationKind = errors.New("intake: 澄清字段类型不在有界词表内")
	// ErrInvalidProposal:推理输出形状非法(缺必填草案字段、澄清与草案混出、
	// 条款重复/空白、未知字段、尾随内容等)。fail safe:宁可拒绝,不猜测。
	ErrInvalidProposal = errors.New("intake: 提案输出形状非法")
	// ErrProposeUncertain:Runtime 自报拿不准(Status=UNCERTAIN)——结果不采信。
	ErrProposeUncertain = errors.New("intake: 提案操作 UNCERTAIN,AI Runtime 拿不准,结果不采信")
	// ErrProposeCancelled:操作被取消(Status=CANCELLED),不得编造成提案失败。
	ErrProposeCancelled = errors.New("intake: 提案操作被取消")
	// ErrProposeFailed:Runtime 判定操作失败(Status=FAILED,含 PROVIDER_FAILURE
	// 等终止原因)。
	ErrProposeFailed = errors.New("intake: 提案操作被 AI Runtime 判定失败")
)
