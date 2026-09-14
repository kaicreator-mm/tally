package requirement

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tally/internal/airuntime"
)

// 问询循环(PRD 第六部分规则 ②):通用推理(AI Runtime)负责问,不负责写。
//
// 两类缺口分开对待:
//   - 结构缺口 —— linter 报的(L1–L5),硬闸,不清零不许签(见 lint.go / sign.go)。
//   - 实质缺口 —— 从条款里读出的模糊点(边界、异常、量级),只建议;
//     可一键驳回,**不阻塞签署**;驳回留痕,存 git。
//
// 推理只做"缺口 → 问题"的转换,尽量给候选答案(选择题比问答题好答)。
// **全程不产出需求正文** —— 提案的 JSON 结构里没有正文字段(见 proposal)。
//
// 推理调用走 internal/airuntime 边界(P0-10 / DECISION_LOCKS #17 / PRD §28):
// 本包只声明 operation / intent / input / budget 与幂等键;provider/model/tier
// 是公共 AI Runtime 侧的配置,不出现在这里。传输由 composition root 注入
// (airuntime.Executor);测试注入 airuntime.FakeExecutor。

// Question 是通用推理针对一个实质缺口提出的问题,尽量带候选答案。
type Question struct {
	Gap        string   `json:"gap"`        // 哪个缺口(定位到条款/概念)
	Question   string   `json:"question"`   // 问用户什么
	Candidates []string `json:"candidates"` // 候选答案
}

// proposal 是送进推理的输出结构样例。
//
// 注意它只有 questions(gap/question/candidates),**没有任何"需求正文"字段** ——
// 这是机器保证:推理想写正文也没地方落。测试会反射断言这一点。
type proposal struct {
	Questions []Question `json:"questions"`
}

// InterviewOperation 是问询操作的稳定标识(契约字段 operation,minLength 1):
// 观测/审计按 operation 归因问询调用,保持不变(PRD §30)。
const InterviewOperation airuntime.Operation = "requirement.interview"

// interviewIntent 是问询调用的角色与用途声明(契约字段 intent;PRD §9/§30)。
// 旧直连路径上 CLI 从未绑定角色(llm.WithRole),`tally req interview` 必然
// ErrNoRole 失败;本迁移把角色声明收进域 —— composition root 不再知道角色,
// 只负责接传输。
var interviewIntent = airuntime.Intent{
	Role: "requirement_interviewer",
	Instructions: "你是需求问询助手:把需求条款中的实质性模糊点或缺口转成问用户的" +
		"问题,尽量给出候选答案(选择题优于问答题)。你只提问,绝不重写、补全或" +
		"改写任何需求正文;结构缺口由确定性 linter 负责,不在你的职责内。",
}

// interviewBudget 是一次问询提案的有界预算(契约 Budget;域声明,Runtime 强制)。
// MaxAttempts=1:问询没有域内重试轮;要不要在传输层重试属于公共 Runtime。
var interviewBudget = airuntime.Budget{
	MaxAttempts:       1,
	MaxCostCeilingUSD: 0.05,
	DeadlineMs:        60_000,
	MaxTokens:         4_096,
}

// 问询调用对公共 AI Runtime 终态的类型化错误。三者必须可区分:
// 取消、拿不准、Runtime 判失败不是同一件事,不得折叠成一个"失败"
// (信任不变量:FAIL 与 ERROR/拿不准端到端可区分)。
var (
	// ErrInterviewUncertain:Runtime 自报拿不准(Status=UNCERTAIN)——结果不可
	// 采信,也不是失败判定;由调用方决定重试或人工介入。
	ErrInterviewUncertain = errors.New("问询操作 UNCERTAIN:AI Runtime 拿不准,结果不采信")
	// ErrInterviewCancelled:操作被取消(Status=CANCELLED),不得编造成候选失败。
	ErrInterviewCancelled = errors.New("问询操作被取消")
	// ErrInterviewFailed:Runtime 判定操作失败(Status=FAILED,含 PROVIDER_FAILURE
	// 等终止原因)。
	ErrInterviewFailed = errors.New("问询操作被 AI Runtime 判定失败")
)

// Interview 让通用推理把文档里的实质缺口转成问题。结构缺口不进这里(那是 linter 的硬闸)。
//
// exec 是 composition root 注入的 airuntime.Executor(真实传输或测试 Fake)。
// 本函数只声明 operation/intent/input/budget + 幂等键,先过 Request.Validate
// 域侧机械闸(操作/意图/幂等键/预算/输入/脱敏),再执行;输出解析与语义校验
// 留在域内(INV-1:AI 产出永远是 Proposal)。
func Interview(ctx context.Context, exec airuntime.Executor, doc *Document) ([]Question, error) {
	prompt := interviewPrompt(doc)
	shape, err := json.Marshal(proposal{})
	if err != nil {
		return nil, fmt.Errorf("序列化问询输出结构样例: %w", err)
	}
	req := airuntime.Request{
		Operation: InterviewOperation,
		Intent:    interviewIntent,
		Input: airuntime.Input{
			Prompt:      prompt,
			OutputShape: shape,
		},
		Budget:         interviewBudget,
		IdempotencyKey: interviewKey(prompt),
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("问询声明未过域侧闸: %w", err)
	}

	res, err := exec.Execute(ctx, req)
	if err != nil {
		// 传输/基础设施失败:没拿到 Operation 结果,如实上抛,不编造 Result。
		return nil, fmt.Errorf("问询执行失败(传输层,无 Operation 结果): %w", err)
	}
	raw, err := interviewOutput(res)
	if err != nil {
		return nil, err
	}

	var out proposal
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("解析提案: %w", err)
	}
	return out.Questions, nil
}

// interviewOutput 把 Runtime 的 Operation 完成事实映射成可解析的原始输出:
// 只有 SUCCEEDED 且输出非空才放行;UNCERTAIN/CANCELLED/FAILED 各归到可区分的
// 类型化错误(带 OperationID 与终止原因,保住按 operation/trace 归因);
// 未知终态按拿不准处理(fail safe,不静默采信)。
func interviewOutput(res airuntime.Result) (json.RawMessage, error) {
	switch res.Status {
	case airuntime.StatusSucceeded:
		if len(res.Output) == 0 {
			return nil, fmt.Errorf("问询 SUCCEEDED 但无输出(operation=%s):没有可解析的提案", res.OperationID)
		}
		return res.Output, nil
	case airuntime.StatusUncertain:
		return nil, fmt.Errorf("%w: operation=%s termination=%s", ErrInterviewUncertain, res.OperationID, res.TerminationReason)
	case airuntime.StatusCancelled:
		return nil, fmt.Errorf("%w: operation=%s termination=%s", ErrInterviewCancelled, res.OperationID, res.TerminationReason)
	case airuntime.StatusFailed:
		return nil, fmt.Errorf("%w: operation=%s termination=%s", ErrInterviewFailed, res.OperationID, res.TerminationReason)
	default:
		return nil, fmt.Errorf("%w: 未知终态 %q operation=%s termination=%s",
			ErrInterviewUncertain, res.Status, res.OperationID, res.TerminationReason)
	}
}

// interviewKey 构造问询的幂等键(契约 Idempotency-Key 头,1..200 字符):
// operation + 提示词内容指纹。同一文档重复问询复用同一键(Runtime 幂等返回
// 既有 Operation);条款一变,提示词变,键随之变(新调用)。sha256 截断保证
// 键长恒定且远低于 200 上限。
func interviewKey(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return string(InterviewOperation) + ":" + hex.EncodeToString(sum[:16])
}

// interviewPrompt 构造问询提示词:给条款,要求把缺口转成带候选答案的问题,
// 明确禁止重写正文。
func interviewPrompt(doc *Document) string {
	var b strings.Builder
	b.WriteString("下面是需求文档的条款。请找出其中**实质性的模糊点或缺口**")
	b.WriteString("(边界情况、异常处理、量级/上限、与相邻条款的交互等),")
	b.WriteString("把每个缺口转成一个问用户的问题,并尽量给出候选答案。\n")
	b.WriteString("**只提问,不要重写、补全或改写任何需求正文。**\n\n条款:\n")
	for _, c := range doc.Clauses {
		fmt.Fprintf(&b, "- %s: %s", c.ID, c.Statement)
		if c.Mechanism != "" {
			fmt.Fprintf(&b, " [检查机制: %s]", c.Mechanism)
		}
		if len(c.Criteria) > 0 {
			fmt.Fprintf(&b, " [验收: %s]", strings.Join(c.Criteria, ";"))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// CanSign 报告是否可以签署:只看结构缺口(linter)。实质缺口(问询建议)不阻塞签署。
func CanSign(doc *Document) bool {
	return !HasStructuralGap(Lint(doc))
}

// RejectionFile 是驳回留痕的文件(在 git 里)。
const RejectionFile = ".tally/rejections.md"

// Reject 驳回一个实质缺口建议,并把驳回记录追加进 git(留痕)。
//
// 驳回只留痕、不改需求、不阻塞签署:它写一条"某问题被驳回 + 理由"的记录并提交。
// 记录是追加式(铁律 5 的精神:用户内容面的历史可追溯)。
func Reject(repoPath string, q Question, reason string, id Identity) (commit string, err error) {
	fsPath := filepath.Join(repoPath, filepath.FromSlash(RejectionFile))
	existing, err := os.ReadFile(fsPath)
	if err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("读驳回记录: %w", err)
	}

	when := id.When
	var b strings.Builder
	b.Write(existing)
	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		b.WriteString("\n")
	}
	if len(existing) == 0 {
		b.WriteString("# 问询驳回归档\n\n被驳回的是 LLM 的实质缺口建议;驳回不阻塞签署,仅留痕。\n\n")
	}
	fmt.Fprintf(&b, "## 驳回 %s\n", when.Format("2006-01-02T15:04:05Z07:00"))
	fmt.Fprintf(&b, "- 缺口: %s\n", q.Gap)
	fmt.Fprintf(&b, "- 问题: %s\n", q.Question)
	fmt.Fprintf(&b, "- 理由: %s\n\n", strings.TrimSpace(reason))

	msg := "reject: 驳回问询建议(" + q.Gap + ")\n"
	hash, err := commitFile(repoPath, RejectionFile, []byte(b.String()), msg, id, nil)
	if err != nil {
		return "", fmt.Errorf("提交驳回记录: %w", err)
	}
	return hash.String(), nil
}
