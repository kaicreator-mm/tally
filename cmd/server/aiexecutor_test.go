package main

// P0-05 的组合根接线测试:证明 v1.4 意图入口的生产 AI 执行路径端到端贯通,
// 且未配置/传输失败时显式、不 panic(P0-05 AC)。
//
// 三层:
//  1. 配置判定:有凭证(含 OPENAI_API_KEY 回退)或显式端点 → 执行器就绪;
//     全缺 → 类型化 errAIRuntimeNotConfigured,执行器为 nil(负路径);
//  2. 组合测试(假传输):环境指向 httptest 假 OpenAI 兼容端点,走与 main()
//     完全相同的生产装配(aiProposeExecutorFromEnv → handlers.
//     SetV14ProposeExecutor → 真实 v14 路由 → intake.Propose → Validated →
//     过渡 llm 传输),断言意图 POST 产出可确认草案;
//  3. 失败显式性:端点 500 → 类型化错误透出到页面,状态可区分,不 panic。
//
// 全局注入缝(handlers.SetV14ProposeExecutor)与进程环境均用 t.Cleanup /
// t.Setenv 还原,测试互不污染,也不依赖机器全局环境。

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"tally/cmd/internal/airuntimelegacy"
	"tally/internal/airuntime"
	"tally/internal/web/handlers"
)

// aiDraftJSON 是可通过 intake 严格校验的草案输出(最小完整字段组)。
const aiDraftJSON = `{
  "task_type": "bug_fix",
  "summary": "修复登出后会话残留",
  "objective": "登出后销毁全部服务端会话状态",
  "scope": {"allowed_paths": ["src/auth/**"], "forbidden_paths": []},
  "non_scope": [],
  "constraints": [],
  "acceptance": [{"id": "AC-01", "statement": "登出后旧会话 token 必须失效"}],
  "risk_signals": [],
  "clarifications": []
}`

// aiClearRuntimeEnv 把 AI Runtime 传输的全部环境信号清空,隔离机器全局环境。
func aiClearRuntimeEnv(t *testing.T) {
	t.Helper()
	t.Setenv("TALLY_LLM_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("TALLY_LLM_BASE_URL", "")
	t.Setenv("TALLY_LLM_MODEL", "")
}

// aiOpenAIStub 是 OpenAI 兼容端点的假传输:chat/completions 请求按给定状态码
// 应答,200 时 content 为合法 JSON 草案字符串(遗留 llm 客户端契约)。
func aiOpenAIStub(t *testing.T, status int, content string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if status != http.StatusOK {
			_, _ = w.Write([]byte("boom"))
			return
		}
		resp := struct {
			Choices []struct {
				Message struct {
					Role    string `json:"role"`
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}{}
		resp.Choices = append(resp.Choices, struct {
			Message struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"message"`
		}{})
		resp.Choices[0].Message.Role = "assistant"
		resp.Choices[0].Message.Content = content
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// aiNewV14Server 直挂 handlers.Server 并只注册 v1.4 路由(与 handlers 包内
// 测试同一姿势,不经登录/CSRF 组)。
func aiNewV14Server(t *testing.T) *httptest.Server {
	t.Helper()
	s := &handlers.Server{
		Content: &handlers.ContentStore{
			RepoPath: t.TempDir(),
			Identity: handlers.Identity{Name: "test", Email: "test@example.com"},
		},
	}
	r := chi.NewRouter()
	s.RegisterV14Routes(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func aiPostForm(t *testing.T, srvURL, path string, form url.Values) (*http.Response, string) {
	t.Helper()
	resp, err := (&http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}).PostForm(srvURL+path, form)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("读响应体: %v", err)
	}
	return resp, string(body)
}

/* ---------- 配置判定(类型化 available/unavailable) ---------- */

func TestAIProposeExecutorNotConfiguredIsTypedAndNil(t *testing.T) {
	aiClearRuntimeEnv(t)
	x, err := aiProposeExecutorFromEnv()
	if x != nil {
		t.Fatalf("未配置时执行器必须为 nil(保持显式未接入),得到 %T", x)
	}
	if !errors.Is(err, errAIRuntimeNotConfigured) {
		t.Fatalf("未配置时必须返回类型化 errAIRuntimeNotConfigured,得到 %v", err)
	}
}

func TestAIProposeExecutorConfiguredDetection(t *testing.T) {
	cases := map[string]map[string]string{
		"显式 key":            {"TALLY_LLM_API_KEY": "k", "OPENAI_API_KEY": "", "TALLY_LLM_BASE_URL": ""},
		"OPENAI_API_KEY 回退": {"TALLY_LLM_API_KEY": "", "OPENAI_API_KEY": "k", "TALLY_LLM_BASE_URL": ""},
		"仅显式端点(自建网关)":       {"TALLY_LLM_API_KEY": "", "OPENAI_API_KEY": "", "TALLY_LLM_BASE_URL": "http://127.0.0.1:9/v1"},
	}
	for name, env := range cases {
		t.Run(name, func(t *testing.T) {
			aiClearRuntimeEnv(t)
			for k, v := range env {
				t.Setenv(k, v)
			}
			x, err := aiProposeExecutorFromEnv()
			if err != nil {
				t.Fatalf("已配置(%s)却返回错误: %v", name, err)
			}
			if x == nil {
				t.Fatalf("已配置(%s)却未建立执行器", name)
			}
		})
	}
}

/* ---------- 组合测试:生产装配链贯通(假传输) ---------- */

func TestAIProposeExecutorWiredEndToEnd(t *testing.T) {
	aiClearRuntimeEnv(t)
	endpoint := aiOpenAIStub(t, http.StatusOK, aiDraftJSON)
	t.Setenv("TALLY_LLM_BASE_URL", endpoint.URL)
	t.Setenv("TALLY_LLM_API_KEY", "test-key")

	// 与 main() 完全相同的生产装配:环境 → 执行器 → 组合根注入。
	x, err := aiProposeExecutorFromEnv()
	if err != nil {
		t.Fatalf("生产装配失败: %v", err)
	}
	handlers.SetV14ProposeExecutor(x)
	t.Cleanup(func() { handlers.SetV14ProposeExecutor(nil) })

	srv := aiNewV14Server(t)

	// AC:「startup establishes executor when configured」——接入后页面不再
	// 渲染未接入防线。
	resp, body := aiGetBody(t, srv.URL+"/v14/launch")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /v14/launch = %d, want 200", resp.StatusCode)
	}
	if strings.Contains(body, "AI Runtime 推理出口未接入") {
		t.Fatalf("执行器已注入,页面却仍渲染未接入防线")
	}

	// 全链贯通:handler → intake.Propose(域闸)→ Validated → 过渡传输 → 端点。
	resp, body = aiPostForm(t, srv.URL, "/v14/launch/intent",
		url.Values{"intent": {"修复登出后会话残留的问题"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST intent = %d, want 200;body: %s", resp.StatusCode, body)
	}
	if !strings.Contains(body, "合同草案") || !strings.Contains(body, "修复登出后会话残留") {
		t.Fatalf("意图未产出可确认草案(组合链断裂);body 片段: %s", aiSnippet(body))
	}
}

/* ---------- 失败显式性(不 panic、可区分) ---------- */

func TestAIProposeExecutorTransportFailureIsExplicit(t *testing.T) {
	aiClearRuntimeEnv(t)
	endpoint := aiOpenAIStub(t, http.StatusInternalServerError, "")
	t.Setenv("TALLY_LLM_BASE_URL", endpoint.URL)
	t.Setenv("TALLY_LLM_API_KEY", "test-key")

	x, err := aiProposeExecutorFromEnv()
	if err != nil {
		t.Fatalf("生产装配失败: %v", err)
	}
	handlers.SetV14ProposeExecutor(x)
	t.Cleanup(func() { handlers.SetV14ProposeExecutor(nil) })

	srv := aiNewV14Server(t)
	resp, body := aiPostForm(t, srv.URL, "/v14/launch/intent",
		url.Values{"intent": {"修复登出后会话残留的问题"}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("传输失败应显式拒绝(400),得到 %d", resp.StatusCode)
	}
	if !strings.Contains(body, "提案执行失败") || !strings.Contains(body, "LLM 返回 500") {
		t.Fatalf("传输层失败未如实透出(不得编造结果);body 片段: %s", aiSnippet(body))
	}
}

func TestUnconfiguredRuntimeKeepsLaunchExplicitlyUnavailable(t *testing.T) {
	aiClearRuntimeEnv(t)
	if _, err := aiProposeExecutorFromEnv(); !errors.Is(err, errAIRuntimeNotConfigured) {
		t.Fatalf("前置:环境未配置应返回类型化错误,得到 %v", err)
	}
	// 未配置 → 组合根不注入(handlers 保持 nil);页面/提交路径显式声明
	// 不可用,不 panic、不假装可用。
	srv := aiNewV14Server(t)
	resp, body := aiPostForm(t, srv.URL, "/v14/launch/intent",
		url.Values{"intent": {"修复登出后会话残留的问题"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("未配置时提交意图 = %d, want 200(显式不可用而非 5xx)", resp.StatusCode)
	}
	if !strings.Contains(body, "AI Runtime 推理出口未接入") {
		t.Fatalf("未配置时页面必须显式声明未接入;body 片段: %s", aiSnippet(body))
	}
}

/* ---------- connected integration(环境门控:有测试 Key 才跑) ---------- */

// TestAIProposeExecutorConnectedIntegration 用真实 OpenAI 兼容端点(DeepSeek,
// 凭证只从 DEEPSEEK_API_KEY 环境变量读取,绝不硬编码)验证生产装配链的
// connected integration:一次真实推理(受 intake 有界预算约束:单次、≤4096
// tokens、≤$0.05)必须命中类型化提案的一等产物 —— 完整草案(200)或澄清
// 请求(200),或命中 intake 的类型化 fail-safe 拒绝(400 + 明确原因:真实
// 模型输出可能违反严格形状契约,如 recommended 类型漂移,契约要求拒绝而非
// 猜测修正)。无论哪条,都必须是「已接线」的可区分结果:绝不 5xx、绝不
// panic、绝不渲染未接入防线。无 Key 时 SKIP(NOT_RUN,不算 PASS)。
func TestAIProposeExecutorConnectedIntegration(t *testing.T) {
	key := os.Getenv("DEEPSEEK_API_KEY")
	if key == "" {
		t.Skip("NOT_RUN: DEEPSEEK_API_KEY 未设置,跳过 connected integration")
	}
	aiClearRuntimeEnv(t)
	t.Setenv("TALLY_LLM_BASE_URL", "https://api.deepseek.com")
	t.Setenv("TALLY_LLM_API_KEY", key)
	t.Setenv("TALLY_LLM_MODEL", "deepseek-chat")

	x, err := aiProposeExecutorFromEnv()
	if err != nil {
		t.Fatalf("生产装配失败: %v", err)
	}
	handlers.SetV14ProposeExecutor(x)
	t.Cleanup(func() { handlers.SetV14ProposeExecutor(nil) })

	srv := aiNewV14Server(t)
	resp, body := aiPostForm(t, srv.URL, "/v14/launch/intent",
		url.Values{"intent": {"修复用户登出后 session 仍然有效的问题。复现步骤:登录子账号后登出," +
			"用旧 token 调 /api/me 仍返回 200。影响面:全部子账号,安全风险高。" +
			"涉及文件:internal/auth/session.go。这是缺陷修复,不是数据库迁移。"}})

	switch resp.StatusCode {
	case http.StatusOK:
		if !strings.Contains(body, "合同草案") && !strings.Contains(body, "澄清(意图含糊") {
			t.Fatalf("真实推理 200 却未产出类型化提案页面(草案/澄清);body 片段: %s", aiSnippet(body))
		}
	case http.StatusBadRequest:
		typed := []string{"提案执行失败", "形状非法", "拿不准", "被取消", "AI Runtime 判定失败"}
		found := false
		for _, m := range typed {
			if strings.Contains(body, m) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("400 未命中任何类型化提案失败原因(不得是无名 5xx 式失败);body 片段: %s", aiSnippet(body))
		}
	default:
		t.Fatalf("真实推理出现未约定状态 %d(接线不得产生 5xx);body 片段: %s", resp.StatusCode, aiSnippet(body))
	}
	if strings.Contains(body, "AI Runtime 推理出口未接入") {
		t.Fatal("执行器已注入,页面却仍渲染未接入防线")
	}
}

/* ---------- 辅助 ---------- */

func aiGetBody(t *testing.T, rawURL string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(rawURL)
	if err != nil {
		t.Fatalf("GET %s: %v", rawURL, err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("读响应体: %v", err)
	}
	return resp, string(body)
}

// aiSnippet 取正文前 400 字节用于失败信息,避免整页塞进测试输出。
func aiSnippet(body string) string {
	if len(body) <= 400 {
		return body
	}
	return body[:400] + "…"
}

// 编译期锚定:本包装配的执行器必须满足 intake 消费的 airuntime.Executor 缝
// (与 handlers.SetV14ProposeExecutor 的参数类型一致)。
var _ airuntime.Executor = airuntimelegacy.FromEnv()
