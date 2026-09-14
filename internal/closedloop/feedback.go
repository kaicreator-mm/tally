// feedback.go 实现 P2-04:面向 Repair Agent 的结构化安全 RepairFeedback 投影
// (PRD §17.1「failure evidence → Agent」段的合法载体;ADR-0001 §4;决策锁 23;
// INV-V14-07:声称严格信息独立时,held-out 原始内容绝不进入实现权威)。
//
// # Allowlist(ADR-0001 §4,固定;扩充需 ADR 修订)
//
//   - acceptance clause id + 语义文本(执行前对 Agent 本就可见);
//   - 失败类(类型化分类,failure.go 的封闭词表);
//   - 受影响行为描述(人类/权威侧撰写的措辞);
//   - 安全诊断摘要(权威侧已完成脱敏,无原始输出);
//   - 可见证据引用(仅实现侧本已可见的证据);
//   - 尝试谱系(run/changeset id)。
//
// # 结构性排除(投影类型里根本不存在这些字段)
//
// 受保护测试路径/源码、hidden test 内容、精确断言、期望值、fixture 内部、
// oracle 实现、held-out 原始 stdout/stderr、hidden 资产的内容摘要、held-out
// binding 在 pass/fail 类别之外的 verdict 细节 —— 本文件不提供任何能承载
// 它们的字段、参数或返回值(测试用反射守卫类型面,用 JSON/文本扫描守卫值面)。
//
// 结构保证的落点:
//
//   - 类型层:RepairFeedback 只含 allowlist 类型化字段;无 map、无 []byte、
//     无 interface/func 字段,不存在原始 blob 透传通道(反射测试固化);
//   - held-out 事实只能经 verification.RedactedHoldoutOutcome(P1-29 的唯一
//     投影出口)进入,本构造器不放宽它 —— 类别只允许 pass|fail;
//   - 可见证据只能经 VisibleEvidence 进入:该类型没有 store/ref 字段,
//     held-out 定位在类型层不可表达;
//   - 文本字段(语义文本/行为描述/诊断摘要)只经界界 + 摘要脱敏管线进入:
//     确定性按 rune 截断并带显式 [truncated] 标记;sha256:<64hex> 内容摘要
//     一律替换为显式 [redacted:digest] 标记(hidden 资产摘要被结构性排除);
//   - 身份字段(id/path/谱系)超界一律类型化报错,身份绝不被截断改写;
//   - fail-safe:不合法或不能落在 allowlist 内的输入 → 类型化 error,
//     Build 返回零值,绝不产出部分反馈、绝不静默吞掉(AC4);只有
//     FailureRepairable(PRD §17.4 唯一默认进入自动修复的类)才可构造,
//     其余类一律拒绝 —— 修复反馈在结构上不可能为升级/infra 路径伪造入口。
//
// 确定性:纯内存、无时钟、无随机、无 I/O;相同输入恒得相同反馈。
package closedloop

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"tally/internal/changeset"
	"tally/internal/verification"
)

// 哨兵错误:调用方用 errors.Is 判因,不做字符串匹配。
var (
	// ErrInvalidFeedback:构造请求不完整或自相矛盾(身份缺失、主体缺失或
	// 双重主体、诊断摘要缺失、held-out 投影与本任务身份不一致)。
	ErrInvalidFeedback = errors.New("closedloop: RepairFeedback 请求不合法")
	// ErrFailureNotRepairable:失败类缺席或不是 repairable。修复反馈只服务
	// 修复路径(PRD §17.4:仅 repairable 默认进入自动修复);其余类别的
	// 落点是 Router 的升级/infra 裁决,不是修复材料。
	ErrFailureNotRepairable = errors.New("closedloop: 非 repairable 失败不构造修复反馈")
	// ErrUnsafeFeedback:输入携带不能落入 allowlist 的材料(未知证据类型、
	// 非法 pass|fail 类别)。绝不静默改写或丢进文本字段。
	ErrUnsafeFeedback = errors.New("closedloop: 反馈输入携带 allowlist 之外的材料")
	// ErrFeedbackBoundExceeded:身份字段超长或事实条目超量。反馈规模有界
	// (AC3);身份绝不被截断,超界即显式失败(fail-safe)。
	ErrFeedbackBoundExceeded = errors.New("closedloop: 反馈规模超界")
)

// 反馈规模界(AC3)。文本界按 rune 计(UTF-8 安全);条目界按元素计。
// 超界的落点见各字段注释:散文截断带标记,身份报错。
const (
	// MaxFeedbackIdentityLen 是单个身份字段(id/path/selector/validator/
	// gate 名/验证运行引用)的 rune 上限。身份是精确标识,超界报错、绝不截断。
	MaxFeedbackIdentityLen = 256
	// MaxFeedbackStatementLen 是 acceptance 语义文本的 rune 上限(超出截断)。
	MaxFeedbackStatementLen = 1000
	// MaxFeedbackBehaviorLen 是受影响行为描述的 rune 上限(超出截断)。
	MaxFeedbackBehaviorLen = 400
	// MaxFeedbackDiagnosticLen 是安全诊断摘要的 rune 上限(超出截断)。
	MaxFeedbackDiagnosticLen = 800
	// MaxFeedbackEvidenceFacts 是可见证据事实条数上限(超出报错)。
	MaxFeedbackEvidenceFacts = 16
	// MaxFeedbackHoldoutFacts 是 held-out 投影条数上限(超出报错)。
	MaxFeedbackHoldoutFacts = 16
)

// 截断与摘要脱敏的显式标记(稳定 ASCII,测试固化):被裁剪/被脱敏的内容
// 绝不静默消失,也不静默泄漏 —— 读者始终能看到「这里发生过什么」。
const (
	feedbackTruncatedMark  = " [truncated]"
	feedbackDigestRedacted = "[redacted:digest]"
)

// feedbackDigestPattern 匹配内容摘要形态(sha256:<64 位十六进制>)。
// hidden 资产的内容摘要在投影中被结构性排除;散文字段里出现的任何该形态
// 串一律替换为显式标记,构造器不猜它是谁的摘要。
var feedbackDigestPattern = regexp.MustCompile(`sha256:[0-9a-fA-F]{64}`)

// EscalationFeedbackUnavailable 是修复反馈缝的 fail-safe 升级原因:反馈
// 来源缺席构造、构造失败(不合法/不安全/规模超界)或反馈身份与本次失败
// 候选不一致 —— 没有安全反馈就不开始修复 Run,宁可升级也绝不把未落在
// allowlist 内的材料送进实现权威(AC4:insufficient safe feedback
// escalates rather than leaks;INV-V14-07)。
const EscalationFeedbackUnavailable EscalationReason = "feedback_unavailable"

// FeedbackSource 是修复反馈的组合层来源端口:由持有冻结契约与类型化验证
// 事实的一方(P2-08 组合层)以 P2-04 的 FeedbackBuilder 构造本次可信 FAIL
// 的安全投影。Engine 在裁决为修复后、开始新 Run 前咨询一次;构造路径的
// 任何失败(含 ErrFailureNotRepairable / ErrUnsafeFeedback /
// ErrFeedbackBoundExceeded)都落 EscalationFeedbackUnavailable 升级 ——
// 反馈材料只有「安全的完整投影」与「不修复」两种命运,绝无降格版本。
// 蓄意诊断暴露(超出 allowlist 的材料被刻意放进投影)由来源侧先经
// ExposureLedger 记账再返回;账实不符的暴露治理归 evidence 侧推导与
// engine 的 accept 守卫。
type FeedbackSource interface {
	RepairFeedback(ctx context.Context, failure VerifyFailure) (RepairFeedback, error)
}

// FeedbackSubject 是反馈指认的失败主体(AC1:acceptance/gate 可识别)。
// 恰好一种形态成立:acceptance(id + 冻结契约的语义文本,执行前对实现侧
// 本就可见)或 required gate 名。两者互斥由构造器机械强制。
type FeedbackSubject struct {
	// AcceptanceID 是失败的验收条款 id;gate 失败时为空。
	AcceptanceID string
	// Statement 是该验收条款的语义文本(冻结契约原文,allowlist:
	// "acceptance_clause id + semantic text")。gate 失败时为空。
	Statement string
	// Gate 是失败的 required gate 名(如 build);acceptance 失败时为空。
	Gate string
}

// VisibleEvidence 是一条实现侧已可见证据的引用事实(allowlist:"visible
// evidence refs")。类型上不存在 store/ref 字段:held-out 定位不可表达;
// Path/Selector 只描述实现 workspace 内本就可见的定位。
type VisibleEvidence struct {
	// AcceptanceID 是该证据所属验收条款 id(gate 相关事实可为空)。
	AcceptanceID string
	// BindingID 是冻结契约中的 evidence binding id。
	BindingID string
	// EvidenceType 是证据类型(封闭词表,同冻结契约的 evidence.type)。
	EvidenceType verification.EvidenceType
	// Path 是实现侧可见证据的 repo 内定位(可见 binding 专属;可空)。
	Path string
	// Selector 是 test/选择器定位(可见 binding 专属;可空)。
	Selector string
	// Validator 是具名 validator 定位(可空)。
	Validator string
	// Status 是该证据的观察类别(仅 pass|fail;与 verdict 词表一致)。
	Status verification.EvidenceStatus
}

// HoldoutFailure 是一条 held-out binding 的投影事实:组合 P1-29 的唯一投影
// 出口 verification.RedactedHoldoutOutcome(不放宽:无 path/ref/digest/
// 可识别 verdict),外加 pass|fail 类别(ADR-0001 §4:held-out binding 的
// verdict 细节中,恰好类别这一粒度是允许的)。
type HoldoutFailure struct {
	// Outcome 是实现侧可见投影原样(RedactHoldoutRecord 的产物)。
	Outcome verification.RedactedHoldoutOutcome
	// Status 是该 binding 的观察类别(仅 pass|fail)。
	Status verification.EvidenceStatus
}

// RepairFeedback 是交给 Repair Agent 的有界结构化反馈(ADR-0001 §4 allowlist
// 的完整投影)。全部字段类型化;无 map/[]byte/interface,不存在原始透传
// 通道。只能经 FeedbackBuilder 构造,不存在绕过 allowlist 管线的赋值入口
// (字段保持导出以供消费方读取;构造纪律由测试反射守卫)。
type RepairFeedback struct {
	// TaskID 是闭环任务 id。
	TaskID string
	// Attempt 是产出失败候选的尝试序号(1 起)。
	Attempt AttemptNo
	// RunID 是该次尝试的 CodegenRun 身份(RunIDFor 确定性推导)。
	RunID RunID
	// ParentRun 是紧邻前一次尝试的 Run 身份;首次尝试为空(谱系父链接,
	// ADR-0001 §2)。
	ParentRun RunID
	// ChangeSet 是未通过验证的不可变候选快照身份(verdict 绑定对象)。
	ChangeSet changeset.ChangeSetID
	// VerifyRunRef 是给出 FAIL 判定的验证运行引用(验证服务如实报告;可空)。
	VerifyRunRef string

	// Subject 指认失败的 acceptance 或 required gate(AC1)。
	Subject FeedbackSubject
	// Failure 是类型化失败类(恒为 FailureRepairable,构造器强制)。
	Failure FailureClass

	// AffectedBehavior 是受影响行为描述(人类/权威侧撰写;可空;有界)。
	AffectedBehavior string
	// DiagnosticSummary 是安全诊断摘要(权威侧脱敏 + 构造器再防线;必填;
	// 有界;sha256 摘要形态已被显式标记替换)。
	DiagnosticSummary string

	// Evidence 是可见证据引用事实(仅实现侧本已可见;条数有界)。
	Evidence []VisibleEvidence
	// Holdout 是 held-out 投影事实(RedactedHoldoutOutcome + pass|fail 类别;
	// 条数有界;条目类型上不存在内容/定位/digest)。
	Holdout []HoldoutFailure
}

// FeedbackBuilder 从类型化事实构造 RepairFeedback。粘滞错误:任一步失败
// 即记录首个类型化 error,后续调用成为 no-op,Build 时返回 —— 绝不产出
// 部分反馈,绝不静默吞错(AC4 的构造器侧落点)。零 I/O,确定性。
type FeedbackBuilder struct {
	err error
	fb  RepairFeedback

	subjectSet bool
	failureSet bool
}

// NewFeedbackBuilder 以失败尝试的谱系身份开建反馈(allowlist:"attempt
// lineage"):RunID 按 (task, attempt) 确定性推导,修复尝试(≥2)同时推导
// 父 Run 身份。身份缺失/非法记粘滞错误。
func NewFeedbackBuilder(taskID string, attempt AttemptNo, cs changeset.ChangeSetID, verifyRunRef string) *FeedbackBuilder {
	b := &FeedbackBuilder{}
	switch {
	case strings.TrimSpace(taskID) == "":
		b.err = fmt.Errorf("%w: task id 不能为空", ErrInvalidFeedback)
	case !attempt.IsValid():
		b.err = fmt.Errorf("%w: 尝试序号非法 %d", ErrInvalidFeedback, int(attempt))
	case strings.TrimSpace(string(cs)) == "":
		b.err = fmt.Errorf("%w: ChangeSet 身份不能为空", ErrInvalidFeedback)
	case utf8.RuneCountInString(verifyRunRef) > MaxFeedbackIdentityLen:
		b.err = fmt.Errorf("%w: 验证运行引用超长(> %d)", ErrFeedbackBoundExceeded, MaxFeedbackIdentityLen)
	}
	if b.err != nil {
		return b
	}
	b.fb.TaskID = taskID
	b.fb.Attempt = attempt
	b.fb.RunID = RunIDFor(taskID, attempt)
	if attempt.IsRepair() {
		b.fb.ParentRun = RunIDFor(taskID, attempt-1)
	}
	b.fb.ChangeSet = cs
	b.fb.VerifyRunRef = verifyRunRef
	return b
}

// ForAcceptance 指认失败主体为验收条款(id + 冻结契约语义文本)。与
// ForGate 互斥;重复设置主体或 statement 缺失记粘滞错误。
func (b *FeedbackBuilder) ForAcceptance(id, statement string) *FeedbackBuilder {
	if b.err != nil {
		return b
	}
	if b.subjectSet {
		b.err = fmt.Errorf("%w: 失败主体重复设置", ErrInvalidFeedback)
		return b
	}
	if strings.TrimSpace(id) == "" {
		b.err = fmt.Errorf("%w: acceptance id 不能为空", ErrInvalidFeedback)
		return b
	}
	if strings.TrimSpace(statement) == "" {
		b.err = fmt.Errorf("%w: acceptance %s 缺少语义文本", ErrInvalidFeedback, id)
		return b
	}
	if utf8.RuneCountInString(id) > MaxFeedbackIdentityLen {
		b.err = fmt.Errorf("%w: acceptance id 超长(> %d)", ErrFeedbackBoundExceeded, MaxFeedbackIdentityLen)
		return b
	}
	b.fb.Subject = FeedbackSubject{
		AcceptanceID: id,
		Statement:    boundProse(statement, MaxFeedbackStatementLen),
	}
	b.subjectSet = true
	return b
}

// ForGate 指认失败主体为 required gate(如 build)。与 ForAcceptance 互斥;
// 重复设置或名称缺失/超长记粘滞错误。
func (b *FeedbackBuilder) ForGate(name string) *FeedbackBuilder {
	if b.err != nil {
		return b
	}
	if b.subjectSet {
		b.err = fmt.Errorf("%w: 失败主体重复设置", ErrInvalidFeedback)
		return b
	}
	if strings.TrimSpace(name) == "" {
		b.err = fmt.Errorf("%w: gate 名不能为空", ErrInvalidFeedback)
		return b
	}
	if utf8.RuneCountInString(name) > MaxFeedbackIdentityLen {
		b.err = fmt.Errorf("%w: gate 名超长(> %d)", ErrFeedbackBoundExceeded, MaxFeedbackIdentityLen)
		return b
	}
	b.fb.Subject = FeedbackSubject{Gate: name}
	b.subjectSet = true
	return b
}

// WithFailure 设置类型化失败类。仅 FailureRepairable 可构造修复反馈
// (PRD §17.4);缺席/非法/其余类别一律 ErrFailureNotRepairable(fail-safe:
// 升级与 infra 材料绝不从修复通道流出)。
func (b *FeedbackBuilder) WithFailure(cls FailureClass) *FeedbackBuilder {
	if b.err != nil {
		return b
	}
	if b.failureSet {
		b.err = fmt.Errorf("%w: 失败类重复设置", ErrInvalidFeedback)
		return b
	}
	if cls != FailureRepairable {
		b.err = fmt.Errorf("%w: 失败类 %q", ErrFailureNotRepairable, string(cls))
		return b
	}
	b.fb.Failure = cls
	b.failureSet = true
	return b
}

// WithAffectedBehavior 设置受影响行为描述(人类/权威侧措辞;可省略)。
// 文本经界界 + 摘要脱敏管线(确定性截断带标记;sha256 摘要形态替换为
// 显式标记)。
func (b *FeedbackBuilder) WithAffectedBehavior(text string) *FeedbackBuilder {
	if b.err != nil {
		return b
	}
	b.fb.AffectedBehavior = boundProse(text, MaxFeedbackBehaviorLen)
	return b
}

// WithDiagnosticSummary 设置安全诊断摘要(权威侧已完成脱敏;构造器再设
// 防线:界界截断 + sha256 摘要形态显式替换)。必填:反馈没有可诊断摘要
// 即不成立(AC1)。
func (b *FeedbackBuilder) WithDiagnosticSummary(text string) *FeedbackBuilder {
	if b.err != nil {
		return b
	}
	if strings.TrimSpace(text) == "" {
		b.err = fmt.Errorf("%w: 安全诊断摘要不能为空", ErrInvalidFeedback)
		return b
	}
	b.fb.DiagnosticSummary = boundProse(text, MaxFeedbackDiagnosticLen)
	return b
}

// AddVisibleEvidence 追加一条可见证据引用事实。类型/类别不在封闭词表、
// 身份超长或条数超界的落点都是类型化错误 —— 绝不静默改写或丢弃。
func (b *FeedbackBuilder) AddVisibleEvidence(fact VisibleEvidence) *FeedbackBuilder {
	if b.err != nil {
		return b
	}
	if !validEvidenceType(fact.EvidenceType) {
		b.err = fmt.Errorf("%w: 未知证据类型 %q", ErrUnsafeFeedback, string(fact.EvidenceType))
		return b
	}
	switch fact.Status {
	case verification.EvidenceStatusPass, verification.EvidenceStatusFail:
	default:
		b.err = fmt.Errorf("%w: 可见证据类别 %q(仅 pass|fail)", ErrUnsafeFeedback, string(fact.Status))
		return b
	}
	for _, ident := range [...]struct {
		name, val string
	}{
		{"AcceptanceID", fact.AcceptanceID},
		{"BindingID", fact.BindingID},
		{"Path", fact.Path},
		{"Selector", fact.Selector},
		{"Validator", fact.Validator},
	} {
		if utf8.RuneCountInString(ident.val) > MaxFeedbackIdentityLen {
			b.err = fmt.Errorf("%w: 可见证据 %s 超长(> %d)", ErrFeedbackBoundExceeded, ident.name, MaxFeedbackIdentityLen)
			return b
		}
	}
	if strings.TrimSpace(fact.BindingID) == "" {
		b.err = fmt.Errorf("%w: 可见证据缺少 binding id", ErrInvalidFeedback)
		return b
	}
	if len(b.fb.Evidence) >= MaxFeedbackEvidenceFacts {
		b.err = fmt.Errorf("%w: 可见证据条数超界(> %d)", ErrFeedbackBoundExceeded, MaxFeedbackEvidenceFacts)
		return b
	}
	b.fb.Evidence = append(b.fb.Evidence, fact)
	return b
}

// AddHoldoutOutcome 追加一条 held-out 投影事实。outcome 必须来自
// RedactHoldoutRecord(类型上已无内容/定位/digest);类别仅 pass|fail;
// 投影携带的任务 id 与本反馈任务 id 矛盾时按接线错误失败(fail-safe,
// 绝不静默改写身份)。
func (b *FeedbackBuilder) AddHoldoutOutcome(outcome verification.RedactedHoldoutOutcome, status verification.EvidenceStatus) *FeedbackBuilder {
	if b.err != nil {
		return b
	}
	switch status {
	case verification.EvidenceStatusPass, verification.EvidenceStatusFail:
	default:
		b.err = fmt.Errorf("%w: held-out 类别 %q(仅 pass|fail)", ErrUnsafeFeedback, string(status))
		return b
	}
	if strings.TrimSpace(outcome.AcceptanceID) == "" {
		b.err = fmt.Errorf("%w: held-out 投影缺少 acceptance id", ErrInvalidFeedback)
		return b
	}
	if !validEvidenceType(outcome.EvidenceType) {
		b.err = fmt.Errorf("%w: held-out 投影证据类型非法 %q", ErrUnsafeFeedback, string(outcome.EvidenceType))
		return b
	}
	if outcome.TaskID != "" && outcome.TaskID != b.fb.TaskID {
		b.err = fmt.Errorf("%w: held-out 投影任务 id %q 与反馈任务 id %q 不一致", ErrInvalidFeedback, outcome.TaskID, b.fb.TaskID)
		return b
	}
	if len(b.fb.Holdout) >= MaxFeedbackHoldoutFacts {
		b.err = fmt.Errorf("%w: held-out 投影条数超界(> %d)", ErrFeedbackBoundExceeded, MaxFeedbackHoldoutFacts)
		return b
	}
	b.fb.Holdout = append(b.fb.Holdout, HoldoutFailure{Outcome: outcome, Status: status})
	return b
}

// Build 校验完整性并返回反馈。任何粘滞错误或缺项(主体/失败类/诊断摘要)
// 都返回零值 + 类型化 error:调用方的落点是升级(AC4),绝不产出部分或
// 降格反馈。幂等纯函数:相同输入重复调用恒得相同结果。
func (b *FeedbackBuilder) Build() (RepairFeedback, error) {
	if b.err != nil {
		return RepairFeedback{}, b.err
	}
	switch {
	case !b.subjectSet:
		return RepairFeedback{}, fmt.Errorf("%w: 缺少失败主体(acceptance 或 gate)", ErrInvalidFeedback)
	case !b.failureSet:
		return RepairFeedback{}, fmt.Errorf("%w: 缺少失败类", ErrFailureNotRepairable)
	case b.fb.DiagnosticSummary == "":
		return RepairFeedback{}, fmt.Errorf("%w: 缺少安全诊断摘要", ErrInvalidFeedback)
	}
	out := b.fb
	out.Evidence = append([]VisibleEvidence(nil), b.fb.Evidence...)
	out.Holdout = append([]HoldoutFailure(nil), b.fb.Holdout...)
	return out, nil
}

// boundProse 是散文字段的唯一入口:先做摘要脱敏(sha256 摘要形态 → 显式
// 标记),再按 rune 截断并追加显式标记。确定性;UTF-8 安全;绝不静默裁剪。
func boundProse(s string, maxRunes int) string {
	s = feedbackDigestPattern.ReplaceAllString(s, feedbackDigestRedacted)
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:maxRunes]) + feedbackTruncatedMark
}

// validEvidenceType 复核证据类型的封闭词表(与冻结契约 evidence.type 同集;
// verification 未提供导出校验器,此处封闭开关由测试守卫同步)。
func validEvidenceType(t verification.EvidenceType) bool {
	switch t {
	case verification.EvidenceTest,
		verification.EvidenceValidator,
		verification.EvidenceBuild,
		verification.EvidenceSchemaCheck,
		verification.EvidenceStaticRule,
		verification.EvidenceHumanVerification,
		verification.EvidenceDeploymentVerification,
		verification.EvidenceRuntimeObservation:
		return true
	default:
		return false
	}
}
