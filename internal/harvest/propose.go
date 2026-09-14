package harvest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"tally/internal/airuntime"
	"tally/internal/logpipe"
)

// 哨兵错误:提案/定义为什么不合格(确定性结论)。
var (
	// ErrNoHealthCheck:定义缺少健康检查。没有它,「拉起成功」的记录可能是假的(PRD 第一部分)。
	ErrNoHealthCheck = errors.New("服务定义缺少健康检查")
	// ErrNoCompose:推理没给 compose 内容。
	ErrNoCompose = errors.New("提案缺少 compose 内容")
	// ErrProposalInvalid:提案结构合法但内容校验不过(如缺健康检查)。
	ErrProposalInvalid = errors.New("提案校验未通过")
)

// Proposal 是通用推理产出的候选服务定义——候选,不是结论(铁律 1)。
// 算不算数由「实际拉起过健康检查」的确定性代码回答(verify.go)。
//
// 结构里**没有任何判定字段**(verified/passed/ok/score/valid 之类都没有):
// 推理只提候选,test 用反射断言这一点(仿 requirement 的提案结构检查)。
type Proposal struct {
	Summary      string        `json:"summary"`       // 这个服务是什么、怎么用
	Compose      string        `json:"compose"`       // 完整 docker-compose YAML
	HealthChecks []HealthCheck `json:"health_checks"` // **必须**含健康检查,否则提案失败
	Wait         WaitPolicy    `json:"wait"`          // 等待策略
}

// proposalSchema 是送进推理的输出结构样例。只有提案字段,无任何判定字段。
var proposalSchema = Proposal{}

// ProposeOperation 是服务定义提案操作的稳定标识(契约字段 operation,
// minLength 1):观测/审计按 operation 归因 harvest 提案调用,保持不变(PRD §30)。
const ProposeOperation airuntime.Operation = "harvest.propose"

// proposeIntent 是提案调用的角色与用途声明(契约字段 intent;PRD §9/§30)。
// 旧直连路径上 CLI 从未绑定角色(llm.WithRole),`tally harvest` 必然 ErrNoRole
// 失败;本迁移把角色声明收进域 —— composition root 不再知道角色,只负责接传输。
var proposeIntent = airuntime.Intent{
	Role: "harvest_proposer",
	Instructions: "你是自托管服务定义抽取助手:为给定服务产出可验证的 docker-compose " +
		"定义与健康检查候选,必须包含至少一项健康检查;机密一律用 ${SERVICE_PASSWORD_XXX} " +
		"这类占位符,绝不写死真实密钥。你只提候选,对不对由实际拉起跑健康检查的" +
		"确定性代码决定,不要自评是否通过。",
}

// proposeBudget 是一次提案的有界预算(契约 Budget;域声明,Runtime 强制)。
// MaxAttempts=1:轮次预算在 Harvester(MaxRounds,提案→验证的外层循环),
// 不与传输层重试混用 —— 每一轮是新的逻辑调用(提示词带上一轮失败原因)。
var proposeBudget = airuntime.Budget{
	MaxAttempts:       1,
	MaxCostCeilingUSD: 0.10,
	DeadlineMs:        120_000,
	MaxTokens:         8_192,
}

// 提案调用对公共 AI Runtime 终态的类型化错误。三者必须可区分:取消、拿不准、
// Runtime 判失败不是同一件事,不得折叠成一个"提案失败"(信任不变量:
// FAIL 与 ERROR/拿不准端到端可区分)。
var (
	// ErrProposeUncertain:Runtime 自报拿不准(Status=UNCERTAIN)——结果不可采信。
	ErrProposeUncertain = errors.New("提案操作 UNCERTAIN:AI Runtime 拿不准,结果不采信")
	// ErrProposeCancelled:操作被取消(Status=CANCELLED),不得编造成候选失败。
	ErrProposeCancelled = errors.New("提案操作被取消")
	// ErrProposeFailed:Runtime 判定操作失败(Status=FAILED,含 PROVIDER_FAILURE
	// 等终止原因)。
	ErrProposeFailed = errors.New("提案操作被 AI Runtime 判定失败")
)

// Propose 向通用推理要一个服务定义候选,并做确定性校验。
//
// 铁律 1:推理只提候选。**提案必须包含合法的健康检查**,否则视为提案失败、不进验证——
// 健康检查写错了,后面所有「拉起成功」都是假的(PRD 第一部分)。
//
// attempt / prevErr:第几轮、上一轮为何失败,喂回提示词让推理针对性改
// (三轮预算在 harvest.go)。
//
// 推理调用走 internal/airuntime 边界(P0-10 / DECISION_LOCKS #17 / PRD §28):
// 本函数只声明 operation/intent/input/budget + 幂等键,先过本地脱敏闸
// (logpipe.MustBeRedacted)再过 Request.Validate 域侧机械闸 —— 换了
// airuntime.FakeExecutor(测试)也没人能替它守。
func Propose(ctx context.Context, exec airuntime.Executor, mat Material, attempt int, prevErr error) (Proposal, error) {
	prompt := buildPrompt(mat, attempt, prevErr)
	if err := logpipe.MustBeRedacted(prompt); err != nil {
		return Proposal{}, fmt.Errorf("提案提示词未脱敏干净,拒绝送出: %w", err)
	}
	shape, err := json.Marshal(proposalSchema)
	if err != nil {
		return Proposal{}, fmt.Errorf("序列化输出结构样例: %w", err)
	}
	req := airuntime.Request{
		Operation: ProposeOperation,
		Intent:    proposeIntent,
		Input: airuntime.Input{
			Prompt:      prompt,
			OutputShape: shape,
		},
		Budget:         proposeBudget,
		IdempotencyKey: proposeKey(mat, attempt),
	}
	if err := req.Validate(); err != nil {
		return Proposal{}, fmt.Errorf("提案声明未过域侧闸: %w", err)
	}

	res, err := exec.Execute(ctx, req)
	if err != nil {
		// 传输/基础设施失败:没拿到 Operation 结果,如实上抛,不编造 Result。
		return Proposal{}, fmt.Errorf("提案执行失败(传输层,无 Operation 结果): %w", err)
	}
	raw, err := proposeOutput(res)
	if err != nil {
		return Proposal{}, err
	}

	var prop Proposal
	if err := json.Unmarshal(raw, &prop); err != nil {
		return Proposal{}, fmt.Errorf("解析定义候选: %w", err)
	}
	if err := validateProposal(prop); err != nil {
		return Proposal{}, err
	}
	return prop, nil
}

// proposeOutput 把 Runtime 的 Operation 完成事实映射成可解析的原始输出:
// 只有 SUCCEEDED 且输出非空才放行;UNCERTAIN/CANCELLED/FAILED 各归到可区分的
// 类型化错误(带 OperationID 与终止原因,保住按 operation/trace 归因);
// 未知终态按拿不准处理(fail safe,不静默采信)。
func proposeOutput(res airuntime.Result) (json.RawMessage, error) {
	switch res.Status {
	case airuntime.StatusSucceeded:
		if len(res.Output) == 0 {
			return nil, fmt.Errorf("提案 SUCCEEDED 但无输出(operation=%s):没有可解析的候选", res.OperationID)
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
// operation + 服务指纹 + 轮次。轮次进键 —— 第 N 轮是新的逻辑调用(提示词带
// 上一轮失败原因),不复用第 N-1 轮的幂等键;同一服务同一轮重试则复用同一键,
// Runtime 幂等返回既有 Operation。sha256 截断保证键长恒定且远低于 200 上限
// (Ref 可能是长 URL,不能直接进键)。
func proposeKey(mat Material, attempt int) string {
	sum := sha256.Sum256([]byte(mat.Service.Source + "\x00" + mat.Service.Name + "\x00" + mat.Service.Ref))
	return string(ProposeOperation) + ":" + hex.EncodeToString(sum[:16]) + ":" + strconv.Itoa(attempt)
}

// validateProposal 是提案的确定性校验:compose 非空 + 健康检查合法。
// 缺健康检查 → ErrProposalInvalid 包 ErrNoHealthCheck(提案失败,不进验证)。
func validateProposal(p Proposal) error {
	if strings.TrimSpace(p.Compose) == "" {
		return fmt.Errorf("%w:LLM 未给出 compose", ErrNoCompose)
	}
	if err := validateChecks(p.HealthChecks); err != nil {
		return fmt.Errorf("%w: %w", ErrProposalInvalid, err)
	}
	return nil
}

// buildPrompt 组装提示词。明确要求「必须含健康检查」「只提候选」「机密用占位符」。
func buildPrompt(mat Material, attempt int, prevErr error) string {
	var b strings.Builder
	b.WriteString("为下面这个自托管服务产出一份可验证的 docker-compose 定义和健康检查。\n")
	b.WriteString("**你的产出必须包含至少一项健康检查**(tcp / http / log-pattern);没有健康检查的定义一律作废。\n")
	b.WriteString("**你只提候选,对不对由实际拉起跑健康检查决定,不要自评是否通过。**\n")
	b.WriteString("compose 里机密用 ${SERVICE_PASSWORD_XXX} 这类占位,不要写死任何真实密钥。\n\n")
	fmt.Fprintf(&b, "服务: %s(来源 %s)\n", mat.Service.Name, mat.Service.Source)
	if mat.Service.Hints != "" {
		fmt.Fprintf(&b, "抽取提示: %s\n", mat.Service.Hints)
	}
	if mat.Raw != "" {
		// 原始材料防御性再脱敏:来源理应干净,不信任上游。
		fmt.Fprintf(&b, "\n参考的原始定义/文档(已脱敏,可能不完整,以上游实际为准):\n%s\n", logpipe.RedactFine(mat.Raw))
	}
	if attempt > 1 && prevErr != nil {
		fmt.Fprintf(&b, "\n这是第 %d 次尝试。上一轮失败原因: %v。请针对性修正后重新产出完整定义。\n", attempt, prevErr)
	}
	return b.String()
}
