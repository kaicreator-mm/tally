package airuntime

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"tally/internal/logpipe"
)

// Operation 标识一次通用推理操作(契约字段 "operation",minLength 1)。
// 稳定字符串 ID 由调用方声明并保持不变,供观测/审计按 operation 归因
// (如 requirement 问询、harvest 提案、failure 诊断各自的稳定 ID)。
type Operation string

// Intent 是调用方声明的业务意图(契约字段 "intent",开放对象;此处类型化,
// 不用 map[string]any 承载核心状态)。迁移自 internal/llm.Role:一次提案调用
// 的职责与用途必须由调用方显式声明,公共 Runtime / 本包都不代选角色。
type Intent struct {
	// Role 是稳定角色标识(对应 llm.Role.Name,供 use_case/role 归因)。
	Role string `json:"role"`
	// Instructions 是该角色的系统指令(对应 llm.Role.Instructions)。
	Instructions string `json:"instructions"`
}

// Input 是结构化操作输入(契约字段 "input",任意 JSON;此处收窄为 Tally
// 通用推理消费方实际使用的形状:提示词 + 期望输出形状)。
type Input struct {
	// Prompt 是调用方的提示词(对应 llm.Propose 的 prompt 参数)。
	Prompt string `json:"prompt"`
	// OutputShape 是期望输出形状的样例 JSON(对应 llm.Propose 的 schema
	// 参数的结构);nil 表示不约束输出形状。
	OutputShape json.RawMessage `json:"output_shape,omitempty"`
}

// Budget 镜像公共 AI Runtime 契约的 Budget(budget.schema.json):
// 每次操作的有界预算,由域声明、Runtime 强制执行。
// JSON 字段名与契约一致,四个下界来自契约 minimum。
type Budget struct {
	// MaxAttempts 是本次操作内 Runtime 侧的最大尝试次数(>=1)。
	MaxAttempts int `json:"max_attempts"`
	// MaxCostCeilingUSD 是最坏情况成本上界,美元(>=0);Runtime 内部换算
	// 整数 micro-USD 执行。
	MaxCostCeilingUSD float64 `json:"max_cost_ceiling"`
	// DeadlineMs 是本次操作的墙钟时限,毫秒(>=1)。
	DeadlineMs int `json:"deadline_ms"`
	// MaxTokens 是本次操作的 token 上界(>=1)。
	MaxTokens int `json:"max_tokens"`
}

// idempotencyKeyMaxLen 是契约 Idempotency-Key 头的 maxLength。
const idempotencyKeyMaxLen = 200

// Request 是 Tally 域侧对一次通用推理操作的完整声明:只有
// operation / intent / input / budget 加幂等键 —— 没有 provider/model/tier。
//
// 幂等键是契约要求的调用方稳定键(传输层放 Idempotency-Key 头,不在请求体,
// 故 json:"-"):同一逻辑调用重试时复用同一键,Runtime 幂等返回既有 Operation。
// 键的构成(如 任务ID+操作+尝试号)由调用方定义,本包不生成。
type Request struct {
	Operation      Operation `json:"operation"`
	Intent         Intent    `json:"intent"`
	Input          Input     `json:"input"`
	Budget         Budget    `json:"budget"`
	IdempotencyKey string    `json:"-"`
}

// 域侧机械闸的哨兵错误。语义验证(输出对不对)不在这里 —— 那永远属于调用方。
var (
	// ErrNoOperation 表示未声明操作 ID。
	ErrNoOperation = errors.New("未声明操作 ID:Request.Operation 不能为空(契约 operation minLength=1)")
	// ErrNoIntent 表示未声明业务意图(迁移自 llm.ErrNoRole 语义)。
	ErrNoIntent = errors.New("未声明业务意图:Intent.Role/Instructions 不能为空,角色与用途由调用方显式声明")
	// ErrInvalidIdempotencyKey 表示幂等键为空或超过契约上限。
	ErrInvalidIdempotencyKey = errors.New("幂等键非法:必须由调用方稳定提供且不超过 200 字符(契约 Idempotency-Key)")
	// ErrInvalidBudget 表示预算越过了契约下界。
	ErrInvalidBudget = errors.New("预算非法:MaxAttempts/DeadlineMs/MaxTokens>=1 且 MaxCostCeilingUSD>=0(契约 Budget)")
	// ErrInvalidInput 表示输入为空或不是合法 JSON。
	ErrInvalidInput = errors.New("输入非法:Prompt 不能为空,OutputShape 若提供必须是合法 JSON")
	// ErrRedacted 表示送入内容疑似含密钥,被域侧脱敏闸拒绝
	// (迁移自 llm.ErrRedacted;logpipe.MustBeRedacted 的最后一道闸)。
	ErrRedacted = errors.New("内容疑似含密钥,拒绝送入 AI Runtime")
)

// Validate 执行域侧机械校验(AC2:域侧验证留在 Tally):
// 结构完整性(操作/意图/幂等键/预算/输入) + 脱敏闸。
// 它不做任何输出判定 —— 接口形状里没有判定方法(INV-1)。
func (r Request) Validate() error {
	if strings.TrimSpace(string(r.Operation)) == "" {
		return ErrNoOperation
	}
	if r.Intent.Role == "" || r.Intent.Instructions == "" {
		return fmt.Errorf("%w: Role=%q", ErrNoIntent, r.Intent.Role)
	}
	if r.IdempotencyKey == "" || len(r.IdempotencyKey) > idempotencyKeyMaxLen {
		return fmt.Errorf("%w: 长度=%d", ErrInvalidIdempotencyKey, len(r.IdempotencyKey))
	}
	if r.Budget.MaxAttempts < 1 || r.Budget.DeadlineMs < 1 ||
		r.Budget.MaxTokens < 1 || r.Budget.MaxCostCeilingUSD < 0 {
		return fmt.Errorf("%w: %+v", ErrInvalidBudget, r.Budget)
	}
	if strings.TrimSpace(r.Input.Prompt) == "" {
		return fmt.Errorf("%w: Prompt 为空", ErrInvalidInput)
	}
	if len(r.Input.OutputShape) > 0 && !json.Valid(r.Input.OutputShape) {
		return fmt.Errorf("%w: OutputShape 不是合法 JSON", ErrInvalidInput)
	}

	// 脱敏闸:所有自由文本通道过一遍 logpipe.MustBeRedacted,
	// 命中疑似密钥即拒绝发送(宁可误伤,不可漏一个)。
	for _, s := range []string{
		r.Intent.Role,
		r.Intent.Instructions,
		r.Input.Prompt,
		string(r.Input.OutputShape),
	} {
		if err := logpipe.MustBeRedacted(s); err != nil {
			return fmt.Errorf("%w: %w", ErrRedacted, err)
		}
	}
	return nil
}
