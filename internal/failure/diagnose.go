package failure

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tally/internal/airuntime"
	"tally/internal/core"
	"tally/internal/logpipe"
)

// Candidate 是通用推理提的修复候选 —— 候选,不是结论(铁律 1)。
// 它算不算数由四道闸的确定性代码回答;推理的自我评估(root_cause、spec_defect 提议)
// 都只是提案,要过确定性闸门才参与后续分流。
type Candidate struct {
	Summary    string            `json:"summary"`     // 做了什么改动、为什么
	RootCause  string            `json:"root_cause"`  // 根因假设(候选,仅供参考)
	Changes    []core.FileChange `json:"changes"`     // 改哪些文件(改后完整内容)
	SpecDefect *SpecDefectHint   `json:"spec_defect"` // 「需求本身有问题」的提议
}

// SpecDefectHint 是推理对「需求本身有问题」的提议:指向条款、指出没写清什么。
// 只是提议;成不成立由确定性闸门验证(条款确实存在,见 SpecDefectSink)。
type SpecDefectHint struct {
	ClauseID  string `json:"clause_id"` // 哪条需求条款(如 REQ-12)
	Ambiguity string `json:"ambiguity"` // 什么没写清楚
}

// candidateSchema 是送进推理的输出结构样例。只有提案字段,没有任何判定字段
// (verified/passed/score 之类都没有)—— 推理没有判定角色,结构上就没地方落。
var candidateSchema = Candidate{}

// DiagnoseOperation 是失败诊断操作的稳定标识(契约字段 operation,minLength 1):
// 观测/审计按 operation 归因诊断调用,保持不变(PRD §30)。
const DiagnoseOperation airuntime.Operation = "failure.diagnose"

// diagnoseIntent 是诊断调用的角色与用途声明(契约字段 intent;PRD §9/§30)。
// 角色声明收进域(P0-11 迁移,同 P0-10):composition root 不再知道角色,
// 只负责接传输。Role 沿用 P0-07 起的稳定标识 "failure-diagnosis",观测归因不断档。
var diagnoseIntent = airuntime.Intent{
	Role: "failure-diagnosis",
	Instructions: "你负责诊断构建/部署失败:根据脱敏后的失败片段与相似历史线索," +
		"提出一个修复候选 —— 改哪些文件(给出改后完整内容)、做了什么改动、根因假设。" +
		"你只提案:修复算不算数由重跑验证的确定性闸门决定。" +
		"不得提议修改测试或验证用文件;" +
		"若你判断根因是需求条款本身没写清楚,给出条款编号与模糊点作为 spec_defect 提案," +
		"不得提议绕过验证。",
}

// diagnoseBudget 是一次诊断提案的有界预算(契约 Budget;域声明,Runtime 强制)。
// MaxAttempts=1:重试预算在 Handle(三轮 ConsultBudget),是外层修复轮次,
// 不与传输层重试混用 —— 每一轮是新的逻辑调用。
var diagnoseBudget = airuntime.Budget{
	MaxAttempts:       1,
	MaxCostCeilingUSD: 0.10,
	DeadlineMs:        120_000,
	MaxTokens:         8_192,
}

// 诊断调用对公共 AI Runtime 终态的类型化错误。三者必须可区分:取消、拿不准、
// Runtime 判失败不是同一件事,不得折叠成一个"诊断失败"(信任不变量:
// FAIL 与 ERROR/拿不准端到端可区分;取消/提供方错误不得编造成候选失败)。
var (
	// ErrDiagnoseUncertain:Runtime 自报拿不准(Status=UNCERTAIN)——结果不可采信。
	ErrDiagnoseUncertain = errors.New("诊断操作 UNCERTAIN:AI Runtime 拿不准,结果不采信")
	// ErrDiagnoseCancelled:操作被取消(Status=CANCELLED),不得编造成候选失败。
	ErrDiagnoseCancelled = errors.New("诊断操作被取消")
	// ErrDiagnoseFailed:Runtime 判定操作失败(Status=FAILED,含 PROVIDER_FAILURE 等终止原因)。
	ErrDiagnoseFailed = errors.New("诊断操作被 AI Runtime 判定失败")
)

// Diagnose 向通用推理要一个修复候选。
//
// 脱敏边界(docs/04-code-standards.md「脱敏」):片段先过 RedactFine,整条提示词
// 送出前再过 MustBeRedacted 自检 —— 命中疑似密钥就拒绝发送。Request.Validate 里
// 还有同一道闸;这里是失败侧自己的保证:换了 FakeExecutor(测试)也没人能替它守。
//
// 模糊指纹召回的线索会进提示词,但明确标注「仅供参考」:模糊命中只能提示方向,
// 不能替代验证(PRD 第五部分)。
//
// 推理调用走 internal/airuntime 边界(P0-11 / DECISION_LOCKS #17 / PRD §28):
// 本函数只声明 operation/intent/input/budget + 幂等键,先过本地脱敏闸再过
// Request.Validate 域侧机械闸;输出解析与语义校验留在域内(INV-1:AI 产出
// 永远是 Proposal)。exec 是 composition root 注入的 airuntime.Executor
// (真实传输或测试 Fake)。
func Diagnose(ctx context.Context, exec airuntime.Executor, ev Event, leads []core.Lead) (Candidate, error) {
	prompt := buildPrompt(ev, leads)
	if err := logpipe.MustBeRedacted(prompt); err != nil {
		return Candidate{}, fmt.Errorf("诊断提示词未脱敏干净,拒绝送出: %w", err)
	}
	shape, err := json.Marshal(candidateSchema)
	if err != nil {
		return Candidate{}, fmt.Errorf("序列化诊断输出结构样例: %w", err)
	}
	req := airuntime.Request{
		Operation: DiagnoseOperation,
		Intent:    diagnoseIntent,
		Input: airuntime.Input{
			Prompt:      prompt,
			OutputShape: shape,
		},
		Budget:         diagnoseBudget,
		IdempotencyKey: diagnoseKey(prompt),
	}
	if err := req.Validate(); err != nil {
		return Candidate{}, fmt.Errorf("诊断声明未过域侧闸: %w", err)
	}

	res, err := exec.Execute(ctx, req)
	if err != nil {
		// 传输/基础设施失败:没拿到 Operation 结果,如实上抛,不编造 Result。
		return Candidate{}, fmt.Errorf("诊断执行失败(传输层,无 Operation 结果): %w", err)
	}
	raw, err := diagnoseOutput(res)
	if err != nil {
		return Candidate{}, err
	}
	var cand Candidate
	if err := json.Unmarshal(raw, &cand); err != nil {
		return Candidate{}, fmt.Errorf("解析修复候选: %w", err)
	}
	return cand, nil
}

// diagnoseOutput 把 Runtime 的 Operation 完成事实映射成可解析的原始输出:
// 只有 SUCCEEDED 且输出非空才放行;UNCERTAIN/CANCELLED/FAILED 各归到可区分的
// 类型化错误(带 OperationID 与终止原因,保住按 operation/trace 归因);
// 未知终态按拿不准处理(fail safe,不静默采信)。
func diagnoseOutput(res airuntime.Result) (json.RawMessage, error) {
	switch res.Status {
	case airuntime.StatusSucceeded:
		if len(res.Output) == 0 {
			return nil, fmt.Errorf("诊断 SUCCEEDED 但无输出(operation=%s):没有可解析的修复候选", res.OperationID)
		}
		return res.Output, nil
	case airuntime.StatusUncertain:
		return nil, fmt.Errorf("%w: operation=%s termination=%s", ErrDiagnoseUncertain, res.OperationID, res.TerminationReason)
	case airuntime.StatusCancelled:
		return nil, fmt.Errorf("%w: operation=%s termination=%s", ErrDiagnoseCancelled, res.OperationID, res.TerminationReason)
	case airuntime.StatusFailed:
		return nil, fmt.Errorf("%w: operation=%s termination=%s", ErrDiagnoseFailed, res.OperationID, res.TerminationReason)
	default:
		return nil, fmt.Errorf("%w: 未知终态 %q operation=%s termination=%s",
			ErrDiagnoseUncertain, res.Status, res.OperationID, res.TerminationReason)
	}
}

// diagnoseKey 构造诊断的幂等键(契约 Idempotency-Key 头,1..200 字符):
// operation + 提示词内容指纹。同一失败片段同一批线索的重试复用同一键
// (Runtime 幂等返回既有 Operation);片段或线索一变,键随之变(新调用)。
// sha256 截断保证键长恒定且远低于 200 上限。
func diagnoseKey(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return string(DiagnoseOperation) + ":" + hex.EncodeToString(sum[:16])
}

func buildPrompt(ev Event, leads []core.Lead) string {
	excerpt := logpipe.RedactFine(ev.Excerpt) // 防御性再脱敏:excerpt 理应已洗过,不信任上游
	var b strings.Builder
	b.WriteString("下面是一次失败的关键片段(已脱敏)和模糊指纹召回的相似历史失败。\n")
	b.WriteString("请提出一个修复方案:改哪些文件(给出改后完整内容)、做了什么改动、为什么、根因猜测。\n")
	b.WriteString("若判断根因是需求本身没写清楚,填写 spec_defect 并指向条款编号。\n")
	b.WriteString("**你只提候选,对不对由重跑验证决定。不得修改测试/验证用文件。**\n\n")
	fmt.Fprintf(&b, "失败来源: %s\n失败片段:\n%s\n", ev.Source, excerpt)
	if len(leads) > 0 {
		b.WriteString("\n相似线索(模糊指纹命中,仅供参考,不是答案):\n")
		for _, l := range leads {
			fmt.Fprintf(&b, "- %s status=%s", l.ExactFP, l.Status)
			if l.Stale {
				b.WriteString("(可能过期)")
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}
