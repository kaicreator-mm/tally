package llm

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// 编译期断言:*HTTPClient 仍满足 Proposer,公开接口仍只有提案语义。
var _ Proposer = (*HTTPClient)(nil)

// 输出样例:问询提案的形状。注意没有"需求正文"字段 —— LLM 不产出需求正文,
// 这个形状由调用方(requirement)约束,llm 包只负责拿回候选。
type sampleSchema struct {
	Questions []struct {
		Gap        string   `json:"gap"`
		Question   string   `json:"question"`
		Candidates []string `json:"candidates"`
	} `json:"questions"`
}

// testRole 构造一个最小可用角色。角色与用途必须由调用方声明 ——
// 真实调用方(requirement/failure/harvest)各自声明各自的指令,本包不代选。
func testRole(name string) Role {
	return Role{Name: name, Instructions: "只做 " + name + " 的提案,不做其他角色的事。"}
}

// roleClient 返回已绑定角色的测试客户端(角色闸的合法前置)。
func roleClient(cfg Config, name string) *HTTPClient {
	return NewHTTPClient(cfg).WithRole(testRole(name))
}

// 正例:Propose 正确构造请求(端点、鉴权头、model、角色化 system 消息含 prompt),并解析响应 JSON。
func TestHTTPClientPropose(t *testing.T) {
	var gotAuth, gotPath, gotModel string
	var gotPrompt, gotSystem string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		var req chatRequest
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &req)
		gotModel = req.Model
		for _, m := range req.Messages {
			switch m.Role {
			case "user":
				gotPrompt = m.Content
			case "system":
				gotSystem = m.Content
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"questions\":[{\"gap\":\"验收机制\",\"question\":\"第 2 条用什么检查?\",\"candidates\":[\"自动化测试\",\"指标\"]}]}"}}]}`))
	}))
	defer srv.Close()

	c := roleClient(Config{BaseURL: srv.URL, APIKey: "test-key", Model: "test-model", Client: srv.Client()}, "requirement_interview")
	raw, err := c.Propose(context.Background(), "需求里有 2 条缺口", sampleSchema{})
	if err != nil {
		t.Fatalf("Propose 出错: %v", err)
	}

	if gotPath != "/chat/completions" {
		t.Errorf("请求路径 = %q, 期望 /chat/completions", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Errorf("鉴权头 = %q, 期望 Bearer test-key", gotAuth)
	}
	if gotModel != "test-model" {
		t.Errorf("model = %q, 期望 test-model", gotModel)
	}
	if !strings.Contains(gotPrompt, "需求里有 2 条缺口") {
		t.Errorf("用户消息未包含 prompt: %q", gotPrompt)
	}
	// 输出形状契约仍在(AC3):system 消息必须携带结构样例。
	if !strings.Contains(gotSystem, `{"questions":null}`) {
		t.Errorf("system 消息未包含输出结构样例: %q", gotSystem)
	}

	var out sampleSchema
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("返回的 JSON 无法解析: %v", err)
	}
	if len(out.Questions) != 1 || out.Questions[0].Gap != "验收机制" {
		t.Errorf("解析结果不符: %+v", out)
	}
}

// 角色闸:未声明角色(或角色不完整)就 Propose,必须拒绝且不触达服务端。
// 本包不提供默认角色 —— 角色/用途必须由调用方声明(P0-07)。
func TestProposeRequiresDeclaredRole(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		hit = true
	}))
	defer srv.Close()

	cfg := Config{BaseURL: srv.URL, Client: srv.Client()}
	for name, c := range map[string]*HTTPClient{
		"未绑定角色":         NewHTTPClient(cfg),
		"缺Name":         NewHTTPClient(cfg).WithRole(Role{Instructions: "有指令没名字"}),
		"缺Instructions": NewHTTPClient(cfg).WithRole(Role{Name: "只有名字"}),
	} {
		if _, err := c.Propose(context.Background(), "正常内容", sampleSchema{}); !errors.Is(err, ErrNoRole) {
			t.Errorf("%s: 应返回 ErrNoRole, got %v", name, err)
		}
	}
	if hit {
		t.Error("角色闸拒绝时不应触达服务端")
	}
}

// WithRole 返回副本,不污染原客户端:共享客户端被角色化后,原客户端仍然没有角色。
func TestWithRoleDoesNotMutateReceiver(t *testing.T) {
	base := NewHTTPClient(Config{BaseURL: "http://127.0.0.1:1", Client: &http.Client{}})
	bound := base.WithRole(testRole("failure_diagnosis"))
	if _, err := base.Propose(context.Background(), "x", sampleSchema{}); !errors.Is(err, ErrNoRole) {
		t.Errorf("原客户端应仍未绑定角色, got %v", err)
	}
	if _, err := bound.Propose(context.Background(), "x", sampleSchema{}); err == nil || errors.Is(err, ErrNoRole) {
		t.Errorf("绑定副本应可提案, got %v", err)
	}
}

// 角色声明(AC1):requirement/failure/harvest 各自声明的角色指令必须原样进入
// system 消息且互不相同;旧的硬编码"需求问询助手"人设必须消失。
func TestProposeSystemPromptUsesDeclaredRole(t *testing.T) {
	roles := []Role{
		{Name: "requirement_interview", Instructions: "把需求缺口转成问题,禁止重写正文。"},
		{Name: "failure_diagnosis", Instructions: "对失败片段提出修复候选,禁止修改测试。"},
		{Name: "harvest_service_definition", Instructions: "产出带健康检查的 compose 定义。"},
	}
	seen := map[string]string{}
	for _, role := range roles {
		var gotSystem string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req chatRequest
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &req)
			for _, m := range req.Messages {
				if m.Role == "system" {
					gotSystem = m.Content
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{}"}}]}`))
		}))

		c := NewHTTPClient(Config{BaseURL: srv.URL, Client: srv.Client()}).WithRole(role)
		if _, err := c.Propose(context.Background(), "正常内容", sampleSchema{}); err != nil {
			t.Fatalf("Propose(%s) 出错: %v", role.Name, err)
		}
		srv.Close()

		if !strings.Contains(gotSystem, role.Name) {
			t.Errorf("system 消息未包含角色名 %s: %q", role.Name, gotSystem)
		}
		if !strings.Contains(gotSystem, role.Instructions) {
			t.Errorf("system 消息未包含角色指令 %q: %q", role.Instructions, gotSystem)
		}
		if strings.Contains(gotSystem, "需求问询助手") {
			t.Errorf("system 消息仍含硬编码问询人设: %q", gotSystem)
		}
		if prev, dup := seen[gotSystem]; dup {
			t.Errorf("不同角色的 system 消息相同(%s 与 %s): %q", prev, role.Name, gotSystem)
		}
		seen[gotSystem] = role.Name
	}
}

// 安全边界:prompt 疑似含密钥时,必须拒绝发送,且不触达服务端。
func TestProposeBlocksUnredactedSecret(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		hit = true
	}))
	defer srv.Close()

	c := roleClient(Config{BaseURL: srv.URL, APIKey: "k", Client: srv.Client()}, "requirement_interview")
	_, err := c.Propose(context.Background(), "clone 用 ghp_aB3dE5gH7jK9mN1pQ3sT5vW7xY9z 这个令牌", sampleSchema{})
	if err == nil {
		t.Fatal("含密钥的 prompt 应被拒绝")
	}
	if hit {
		t.Error("拒绝发送时不应触达服务端")
	}
}

// 负例:HTTP 非 2xx 返回错误。
func TestProposeHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := roleClient(Config{BaseURL: srv.URL, Client: srv.Client()}, "failure_diagnosis")
	if _, err := c.Propose(context.Background(), "正常内容", sampleSchema{}); err == nil {
		t.Fatal("5xx 应返回错误")
	}
}

// 负例:LLM 输出不是合法 JSON 时返回错误(调用方没法安全反序列化)。
func TestProposeInvalidJSONContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"这不是 JSON"}}]}`))
	}))
	defer srv.Close()

	c := roleClient(Config{BaseURL: srv.URL, Client: srv.Client()}, "harvest_service_definition")
	if _, err := c.Propose(context.Background(), "正常内容", sampleSchema{}); err == nil {
		t.Fatal("非法 JSON 输出应返回错误")
	}
}

// 负例:API 层 error 字段非空时返回错误。
func TestProposeAPIErrorField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":{"message":"rate limited"}}`))
	}))
	defer srv.Close()

	c := roleClient(Config{BaseURL: srv.URL, Client: srv.Client()}, "requirement_interview")
	_, err := c.Propose(context.Background(), "正常内容", sampleSchema{})
	if err == nil || !strings.Contains(err.Error(), "rate limited") {
		t.Fatalf("应返回含 rate limited 的错误, got %v", err)
	}
}
