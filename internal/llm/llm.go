package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"tally/internal/logpipe"
)

// Proposer 是 llm 包唯一导出的接口,只有一个方法。
//
// 只提案,不判定:没有 Judge/Verify/Score/Evaluate/Check/Validate(INV-1,
// test/arch 用反射强制)。"这个提案对不对"永远由调用方的确定性代码回答。
//
// schema 是期望输出形状的样例值(其 JSON 结构即目标结构),用于约束模型输出;
// 返回模型产出的原始 JSON,由调用方自行反序列化并验证。
type Proposer interface {
	Propose(ctx context.Context, prompt string, schema any) (json.RawMessage, error)
}

// Role 是调用方声明的提案角色(PRD §9 Agent Roles / §30 use_case/role):
// 一次 LLM 提案调用的职责与用途,由调用方声明,llm 包不内置任何业务人设。
//
// 曾经硬编码在本包的"需求问询助手"系统提示词被 requirement/harvest/failure
// 三处共用 —— 失败诊断顶着问询助手的人设干活,角色与用途无从区分(P0-07 修复)。
//
//	Name 是稳定角色标识(供观测/审计按 use_case/role 归因,PRD §30);
//	Instructions 是该角色的系统指令(职责与行为边界),每个调用方各不相同。
type Role struct {
	Name         string
	Instructions string
}

// ErrNoRole 表示客户端未绑定调用方声明的提案角色(或角色不完整)就发起了 Propose。
// 本包不提供默认角色:角色/用途必须由调用方显式声明,不能由 llm 包代选。
var ErrNoRole = errors.New("未声明提案角色:必须先 WithRole 声明角色与用途(PRD §9)")

// Config 是 OpenAI 兼容客户端的注入配置。端点/key/model 可由此注入,
// 也可经 ConfigFromEnv 从环境读取。
type Config struct {
	// BaseURL 是 OpenAI 兼容 API 的根(不含 /chat/completions)。
	BaseURL string
	// APIKey 放在 Authorization: Bearer 头里。
	APIKey string
	// Model 是模型名。
	Model string
	// Client 可注入(测试用 httptest);nil 时用带超时的默认客户端。
	Client *http.Client
}

// ConfigFromEnv 从环境变量构造配置,缺省给合理默认值。
//
//	TALLY_LLM_BASE_URL(默认 https://api.openai.com/v1)
//	TALLY_LLM_API_KEY(回退 OPENAI_API_KEY)
//	TALLY_LLM_MODEL(默认 gpt-4o-mini)
func ConfigFromEnv() Config {
	apiKey := os.Getenv("TALLY_LLM_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	return Config{
		BaseURL: envOr("TALLY_LLM_BASE_URL", "https://api.openai.com/v1"),
		APIKey:  apiKey,
		Model:   envOr("TALLY_LLM_MODEL", "gpt-4o-mini"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// HTTPClient 用标准库 net/http 直连 OpenAI 兼容 API(不用 SDK,见技术栈)。
//
// 客户端只负责"提案"的机械部分:角色闸、脱敏闸、输出形状约束、OpenAI 报文。
// 角色不是客户端的内置人设,必须经 WithRole 由调用方声明。
type HTTPClient struct {
	cfg  Config
	role *Role // 调用方声明的提案角色;nil = 未声明,Propose 拒绝执行(ErrNoRole)
}

// NewHTTPClient 构造客户端;补齐 BaseURL 与默认 http.Client。
// 新构造的客户端未绑定角色;调用方须再链上 WithRole 声明角色与用途。
//
// Deprecated:遗留直连供应商路径。消费方已全部迁到 internal/airuntime 边界
// (P0-09/P0-10/P0-11);本包仅剩 cmd/tally 与 cmd/worker composition root 的
// llmExecutor 过渡适配器在用,真实 AI Runtime SDK 接线后随包一起删除。
// 新代码不要调用本构造器。
func NewHTTPClient(cfg Config) *HTTPClient {
	if cfg.Client == nil {
		cfg.Client = &http.Client{Timeout: 60 * time.Second}
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.openai.com/v1"
	}
	return &HTTPClient{cfg: cfg}
}

// WithRole 绑定调用方声明的提案角色,返回绑定后的客户端副本(不改原客户端,
// 副本可安全并发共享)。这是既有调用方的最小迁移适配器:
//
//	llm.NewHTTPClient(llm.ConfigFromEnv()).WithRole(llm.Role{Name: "…", Instructions: "…"})
//
// 角色不完整(Name/Instructions 为空)不在绑定时报错,在 Propose 时拒绝:
// 让"没把角色声明好"成为调用时可见的事实,而不是绑定时的猜测。
func (c *HTTPClient) WithRole(role Role) *HTTPClient {
	r := role
	return &HTTPClient{cfg: c.cfg, role: &r}
}

// chatMessage / chatRequest / chatResponse 是 OpenAI chat/completions 的最小报文。
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string            `json:"model"`
	Messages       []chatMessage     `json:"messages"`
	ResponseFormat map[string]string `json:"response_format,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ErrRedacted 表示送进来的内容疑似含密钥,被安全边界拒绝(INV-1 的脱敏前置)。
var ErrRedacted = fmt.Errorf("内容疑似含密钥,拒绝送入 LLM")

// Propose 把 prompt 送给模型,要求按 schema 的 JSON 结构作答,返回原始 JSON。
//
// 前置两道闸:
//   - 角色闸:必须先 WithRole 声明提案角色;llm 包不替调用方选角色、不内置
//     任何业务人设(P0-07:共用"需求问询助手"人设的缺陷已移除)。
//   - 脱敏闸:prompt 必须先过 logpipe.MustBeRedacted,命中疑似密钥即拒绝发送 ——
//     脱敏漏网时这是最后一道闸(docs/04-code-standards.md「脱敏」)。
//
// 接口里没有判定方法:Propose 只负责"拿回候选",候选对不对由调用方验
// (INV-AUTH-02:LLM 产出永远是 Proposal)。
func (c *HTTPClient) Propose(ctx context.Context, prompt string, schema any) (json.RawMessage, error) {
	if c.role == nil {
		return nil, ErrNoRole
	}
	if c.role.Name == "" || c.role.Instructions == "" {
		return nil, fmt.Errorf("%w: 角色 Name/Instructions 不完整(Name=%q)", ErrNoRole, c.role.Name)
	}

	if err := logpipe.MustBeRedacted(prompt); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrRedacted, err)
	}

	shape, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("序列化输出结构样例: %w", err)
	}

	// 系统提示词 = 调用方声明的角色指令 + 本包固有的输出形状契约。
	// 后者是机械契约(怎么答),不是人设(是谁/干什么)——人设只来自 Role。
	system := "你是 " + c.role.Name + "。" + c.role.Instructions + "\n" +
		"只输出 JSON,不要任何解释文字。" +
		"输出必须与如下 JSON 结构同形(字段名、层级一致,值按问题填写):\n" + string(shape)
	body, err := json.Marshal(chatRequest{
		Model: c.cfg.Model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: prompt},
		},
		ResponseFormat: map[string]string{"type": "json_object"},
	})
	if err != nil {
		return nil, fmt.Errorf("序列化请求: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.cfg.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("构造请求: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.cfg.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用 LLM: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读响应体: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("LLM 返回 %d: %s", resp.StatusCode, truncate(raw, 512))
	}

	var parsed chatResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("解析响应: %w", err)
	}
	if parsed.Error != nil {
		return nil, fmt.Errorf("LLM 报错: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("LLM 响应无 choices")
	}

	content := parsed.Choices[0].Message.Content
	if !json.Valid([]byte(content)) {
		return nil, fmt.Errorf("LLM 输出不是合法 JSON: %s", truncate([]byte(content), 256))
	}
	return json.RawMessage(content), nil
}

// truncate 截断过长报文用于错误信息,避免把整段(可能很大的)响应塞进错误。
func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "...(截断)"
}
