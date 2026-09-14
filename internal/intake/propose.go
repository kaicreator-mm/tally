package intake

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"tally/internal/airuntime"
	"tally/internal/logpipe"
)

// maxIntentLen 是意图文本的长度上限(有界输入):一句自然语言远用不了这么长,
// 超限即拒绝,不让失控输入变成失控提示词。
const maxIntentLen = 20_000

// ProposeOperation 是意图结构化操作的稳定标识(契约字段 operation,minLength 1):
// 观测/审计按 operation 归因 intake 提案调用,保持不变(PRD §30)。
const ProposeOperation airuntime.Operation = "intake.propose"

// proposeIntent 是意图结构化调用的角色与用途声明(契约字段 intent;PRD §9/§30):
// 角色与用途由本域显式声明,composition root 不参与。
var proposeIntent = airuntime.Intent{
	Role: "intake_drafter",
	Instructions: "你是任务意图结构化助手:把一句自然语言意图整理成任务合同草案" +
		"(意图摘要/目标/边界/非目标/约束/初始验收/候选风险信号),或提出澄清问题。" +
		"你只产草案供人确认,绝不自评对错;验收冻结由既有流程负责,与你无关。" +
		"机密一律用 ${...} 占位符,绝不写死真实密钥。",
}

// proposeBudget 是一次意图结构化的有界预算(契约 Budget;域声明,Runtime 强制)。
// MaxAttempts=1:澄清-补答是新的逻辑调用(新意图输入、新幂等键),
// 不与传输层重试混用。
var proposeBudget = airuntime.Budget{
	MaxAttempts:       1,
	MaxCostCeilingUSD: 0.05,
	DeadlineMs:        60_000,
	MaxTokens:         4_096,
}

// wireProposal 是送进推理并解析回来的输出形状:草案字段组与 clarifications
// 二选一。解析用严格模式(未知字段/尾随内容即报错),不给推理"顺便多写点"
// 的余地(fail safe,不猜测)。clarifications 走有界 schema(DL-11):每项
// 可带封闭词表内的 kind 与 advisory 建议,词表之外的类型在语义校验层拒绝。
type wireProposal struct {
	TaskType       string             `json:"task_type"`
	Summary        string             `json:"summary"`
	Objective      string             `json:"objective"`
	Scope          Scope              `json:"scope"`
	NonScope       []string           `json:"non_scope"`
	Constraints    []string           `json:"constraints"`
	Acceptance     []AcceptanceClause `json:"acceptance"`
	RiskSignals    []string           `json:"risk_signals"`
	Clarifications []Clarification    `json:"clarifications"`
}

// Propose 把一句意图结构化成类型化提案:完整草案(DraftProposal)或澄清请求
// (ClarificationProposal)。
//
// 推理调用走 internal/airuntime 边界(P3-07 / DECISION_LOCKS #17 / ADR-0002 /
// PRD §28):本函数只声明 operation/intent/input/budget + 幂等键,先过本地
// 脱敏闸(logpipe.MustBeRedacted)再过 Request.Validate 域侧机械闸;输出的
// 解析与语义校验留在本包(INV-1:AI 产出永远是 Proposal)。
//
// 校验规则(fail safe,AC3/AC4):
//   - clarifications 非空 → 必须是纯澄清请求,任何草案字段出现即拒绝
//     (不得一边问一边发明验收);
//   - clarifications 为空 → 必须是完整草案:任务类型恰为两个 MVP 值之一、
//     摘要/目标非空、allowed_paths 非空、验收至少一条且 id 唯一;
//   - 形状非法(未知字段、空白条款、尾随内容、两者都不给)→ ErrInvalidProposal。
func Propose(ctx context.Context, exec airuntime.Executor, in Intent) (Proposal, error) {
	if exec == nil {
		return nil, ErrNoExecutor
	}
	if strings.TrimSpace(in.Text) == "" {
		return nil, ErrEmptyIntent
	}
	if len(in.Text) > maxIntentLen {
		return nil, fmt.Errorf("%w: len=%d max=%d", ErrIntentTooLong, len(in.Text), maxIntentLen)
	}

	prompt := intakePrompt(in)
	if err := logpipe.MustBeRedacted(prompt); err != nil {
		return nil, fmt.Errorf("意图文本疑似含密钥,拒绝送入 AI Runtime: %w", err)
	}
	shape, err := json.Marshal(wireProposal{})
	if err != nil {
		return nil, fmt.Errorf("序列化输出结构样例: %w", err)
	}
	req := airuntime.Request{
		Operation: ProposeOperation,
		Intent:    proposeIntent,
		Input: airuntime.Input{
			Prompt:      prompt,
			OutputShape: shape,
		},
		Budget:         proposeBudget,
		IdempotencyKey: proposeKey(in),
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("提案声明未过域侧闸: %w", err)
	}

	res, err := exec.Execute(ctx, req)
	if err != nil {
		// 传输/基础设施失败:没拿到 Operation 结果,如实上抛,不编造 Result。
		return nil, fmt.Errorf("提案执行失败(传输层,无 Operation 结果): %w", err)
	}
	raw, err := proposeOutput(res)
	if err != nil {
		return nil, err
	}
	wire, err := decodeStrict(raw)
	if err != nil {
		return nil, err
	}
	return toProposal(wire)
}

// decodeStrict 严格解析推理输出:未知字段、尾随内容、非法 JSON 一律按
// ErrInvalidProposal 拒绝 —— 不做 best-effort 猜测(AC3 fail safe)。
func decodeStrict(raw json.RawMessage) (wireProposal, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var w wireProposal
	if err := dec.Decode(&w); err != nil {
		return wireProposal{}, fmt.Errorf("%w: %s", ErrInvalidProposal, err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return wireProposal{}, fmt.Errorf("%w: 输出含尾随内容", ErrInvalidProposal)
	}
	return w, nil
}

// toProposal 把解析后的形状收敛为互斥的类型化提案(AC3):要么纯澄清,
// 要么完整草案;混出或缺项都是 ErrInvalidProposal,任务类型越界是
// ErrUnsupportedTaskType(AC4)。
func toProposal(w wireProposal) (Proposal, error) {
	if len(w.Clarifications) > 0 {
		if hasDraftFields(w) {
			return nil, fmt.Errorf("%w: 澄清与草案混出(不得一边问一边发明验收)", ErrInvalidProposal)
		}
		// 有界 schema 校验与归一(DL-11):旧简单形态显式迁移,未知字段
		// 类型与形状漂移 fail safe 拒绝(详见 clarification_schema.go)。
		qs, err := validateClarifications(w.Clarifications)
		if err != nil {
			return nil, err
		}
		return ClarificationProposal{Questions: qs}, nil
	}

	tt, ok := taskTypeOf(w.TaskType)
	if !ok {
		return nil, fmt.Errorf("%w: task_type=%q", ErrUnsupportedTaskType, w.TaskType)
	}
	if strings.TrimSpace(w.Summary) == "" {
		return nil, fmt.Errorf("%w: 缺少 summary", ErrInvalidProposal)
	}
	if strings.TrimSpace(w.Objective) == "" {
		return nil, fmt.Errorf("%w: 缺少 objective", ErrInvalidProposal)
	}
	if err := validatePaths("scope.allowed_paths", w.Scope.AllowedPaths); err != nil {
		return nil, err
	}
	if len(w.Scope.AllowedPaths) == 0 {
		return nil, fmt.Errorf("%w: scope.allowed_paths 为空(合同必须声明实现边界)", ErrInvalidProposal)
	}
	if err := validatePaths("scope.forbidden_paths", w.Scope.ForbiddenPaths); err != nil {
		return nil, err
	}
	if len(w.Acceptance) == 0 {
		return nil, fmt.Errorf("%w: 验收条款为空(草案必须给出可确认的初始验收)", ErrInvalidProposal)
	}
	seen := map[string]bool{}
	for i, a := range w.Acceptance {
		if strings.TrimSpace(a.ID) == "" {
			return nil, fmt.Errorf("%w: 验收条款 #%d 缺少 id", ErrInvalidProposal, i+1)
		}
		if seen[a.ID] {
			return nil, fmt.Errorf("%w: 验收条款 id %q 重复", ErrInvalidProposal, a.ID)
		}
		seen[a.ID] = true
		if strings.TrimSpace(a.Statement) == "" {
			return nil, fmt.Errorf("%w: 验收条款 %s 缺少 statement", ErrInvalidProposal, a.ID)
		}
	}
	if err := validatePaths("non_scope", w.NonScope); err != nil {
		return nil, err
	}
	if err := validatePaths("constraints", w.Constraints); err != nil {
		return nil, err
	}
	if err := validatePaths("risk_signals", w.RiskSignals); err != nil {
		return nil, err
	}
	return DraftProposal{Draft: Draft{
		TaskType:    tt,
		Summary:     w.Summary,
		Objective:   w.Objective,
		Scope:       w.Scope,
		NonScope:    w.NonScope,
		Constraints: w.Constraints,
		Acceptance:  w.Acceptance,
		RiskSignals: w.RiskSignals,
	}}, nil
}

// hasDraftFields 报告 wire 里是否出现任何草案字段:纯澄清请求不允许夹带草案。
func hasDraftFields(w wireProposal) bool {
	return w.TaskType != "" ||
		strings.TrimSpace(w.Summary) != "" ||
		strings.TrimSpace(w.Objective) != "" ||
		len(w.Scope.AllowedPaths) > 0 ||
		len(w.Scope.ForbiddenPaths) > 0 ||
		len(w.NonScope) > 0 ||
		len(w.Constraints) > 0 ||
		len(w.Acceptance) > 0 ||
		len(w.RiskSignals) > 0
}

// validatePaths 拒绝列表里的空白条目(空条目是写错了,不是数据;
// 与 taskcontract.Validate 同规则)。
func validatePaths(name string, vals []string) error {
	for _, v := range vals {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("%w: %s 含空白条目", ErrInvalidProposal, name)
		}
	}
	return nil
}

// proposeOutput 把 Runtime 的 Operation 完成事实映射成可解析的原始输出:
// 只有 SUCCEEDED 且输出非空才放行;UNCERTAIN/CANCELLED/FAILED 各归到可区分的
// 类型化错误(带 OperationID 与终止原因,保住按 operation/trace 归因);
// 未知终态按拿不准处理(fail safe,不静默采信)。
func proposeOutput(res airuntime.Result) (json.RawMessage, error) {
	switch res.Status {
	case airuntime.StatusSucceeded:
		if len(res.Output) == 0 {
			return nil, fmt.Errorf("提案 SUCCEEDED 但无输出(operation=%s):没有可解析的草案", res.OperationID)
		}
		return res.Output, nil
	case airuntime.StatusUncertain:
		return nil, fmt.Errorf("%w: operation=%s termination=%s", ErrProposeUncertain, res.OperationID, res.TerminationReason)
	case airuntime.StatusCancelled:
		return nil, fmt.Errorf("%w: operation=%s termination=%s", ErrProposeCancelled, res.OperationID, res.TerminationReason)
	case airuntime.StatusFailed:
		return nil, fmt.Errorf("%w: operation=%s termination=%s", ErrProposeFailed, res.OperationID, res.TerminationReason)
	default:
		return nil, fmt.Errorf("%w: 未知终态 %q operation=%s termination=%s",
			ErrProposeUncertain, res.Status, res.OperationID, res.TerminationReason)
	}
}

// proposeKey 构造提案的幂等键(契约 Idempotency-Key 头,1..200 字符):
// operation + 意图内容指纹。同一意图(含来源引用)重复提案复用同一键,
// Runtime 幂等返回既有 Operation;意图一变,键随之变(新调用)。sha256 截断
// 保证键长恒定且远低于 200 上限。
func proposeKey(in Intent) string {
	sum := sha256.Sum256([]byte(in.Text + "\x00" + in.SourceRef))
	return string(ProposeOperation) + ":" + hex.EncodeToString(sum[:16])
}

// intakePrompt 组装提示词:给一句意图(+可选来源),要求产出草案字段或
// 澄清问题(二选一),明确禁止在含糊时发明验收。
func intakePrompt(in Intent) string {
	var b strings.Builder
	b.WriteString("下面是一句自然语言任务意图。请把它结构化成任务合同草案,")
	b.WriteString("字段为 task_type/summary/objective/scope/non_scope/constraints/acceptance/risk_signals。\n")
	b.WriteString("规则:\n")
	b.WriteString("1. task_type 只能取 \"bug_fix\"(缺陷修复)或 \"db_migration\"(数据库迁移);\n")
	b.WriteString("   意图不属于这两类、或拿不准属于哪类时,不要硬分类。\n")
	b.WriteString("2. 二选一:信息足够 → 填草案字段且 clarifications 留空;缺关键信息\n")
	b.WriteString("   (复现步骤、影响面、涉及文件、迁移基线等)→ 只给 clarifications,\n")
	b.WriteString("   草案字段全部留空。**含糊时绝不发明验收条款。**\n")
	b.WriteString("3. acceptance 每条 statement 必须可判定(能写出通过/不通过的检查),id 用 AC-01 起递增且唯一。\n")
	b.WriteString("4. scope 用仓库内相对路径(斜杠风格),allowed_paths 必须非空。\n")
	b.WriteString("5. 你只产草案,算不算数由人确认与既有冻结流程决定,不要自评。\n")
	b.WriteString("6. clarifications 每项可带 kind(封闭词表):choice/multichoice/boolean/number/\n")
	b.WriteString("   repository/branch/risk_choice/confirmation/free_text;省略 kind 时,带 candidates\n")
	b.WriteString("   视为 choice,不带视为 free_text(自由文本逃生舱)。词表之外的 kind 会被整体拒绝。\n")
	b.WriteString("   recommended/recommendation_reason 只是建议(advisory);问题最多 8 个、候选最多 16 个。\n\n")
	fmt.Fprintf(&b, "意图: %s\n", in.Text)
	if in.SourceRef != "" {
		b.WriteString("外部来源(仅参考,不搬进 Tally): ")
		b.WriteString(in.SourceRef)
		b.WriteString("\n")
	}
	return b.String()
}
