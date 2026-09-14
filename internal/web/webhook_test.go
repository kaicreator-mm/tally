package web

// webhook 的离线单测(httptest,不碰数据库):鉴权与尺寸边界在录入之前就拦下,
// 所以替身 store 都不需要。真录入链路在 webhook_integration_test.go(真 Postgres)。

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// webhookRouter 只挂 webhook 路由的基座(405 依赖 chi 的方法匹配)。
func webhookRouter(h http.HandlerFunc) *chi.Mux {
	r := chi.NewRouter()
	r.Post("/webhook/build", h)
	return r
}

// TestWebhookBuildDisabled503:未配置 token → 503(明确「未启用」,
// 不做默认放行也不做默认拒绝歧义,D-d1)。
func TestWebhookBuildDisabled503(t *testing.T) {
	h := Deps{}.webhookBuild
	req := httptest.NewRequest(http.MethodPost, "/webhook/build", strings.NewReader("{}"))
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("未配置 token 状态码 = %d,想要 503", rec.Code)
	}
}

// TestWebhookBuildBadToken401:无 token / 错 token / 非 Bearer 前缀 → 一律 401。
func TestWebhookBuildBadToken401(t *testing.T) {
	h := Deps{WebhookToken: "secret"}.webhookBuild
	for _, hdr := range []string{"", "Bearer wrong", "Token secret"} {
		req := httptest.NewRequest(http.MethodPost, "/webhook/build", strings.NewReader("{}"))
		if hdr != "" {
			req.Header.Set("Authorization", hdr)
		}
		rec := httptest.NewRecorder()
		h(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("Authorization=%q:状态码 = %d,想要 401", hdr, rec.Code)
		}
	}
}

// TestWebhookBuildNonPost405:路由只接受 POST(chi 方法匹配在 handler 之前)。
func TestWebhookBuildNonPost405(t *testing.T) {
	srv := httptest.NewServer(webhookRouter(Deps{WebhookToken: "secret"}.webhookBuild))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/webhook/build")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("GET 状态码 = %d,想要 405", resp.StatusCode)
	}
}

// TestWebhookBuildOversizedLog413:log 超 10MB → 413 且不录入
// (尺寸检查在入库之前,替身 store 用 nil 也不会被碰到)。
func TestWebhookBuildOversizedLog413(t *testing.T) {
	h := Deps{WebhookToken: "secret", LogDir: t.TempDir()}.webhookBuild
	big := strings.Repeat("x", maxWebhookLogBytes+1)
	body := `{"project":"p1","commit":"c1","log":"` + big + `"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook/build", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("超大 log 状态码 = %d,想要 413", rec.Code)
	}
}

// TestWebhookBuildOversizedBody413:请求体超 12MB 封顶 → 413(MaxBytesReader)。
func TestWebhookBuildOversizedBody413(t *testing.T) {
	h := Deps{WebhookToken: "secret"}.webhookBuild
	big := strings.Repeat("x", maxWebhookBodyBytes+1)
	req := httptest.NewRequest(http.MethodPost, "/webhook/build", strings.NewReader(big))
	req.Header.Set("Authorization", "Bearer secret")
	rec := httptest.NewRecorder()
	h(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("超大请求体状态码 = %d,想要 413", rec.Code)
	}
}

// TestWebhookBuildBadRequest400:坏 JSON / 缺必填 / 不安全的项目名(防路径穿越)→ 400。
func TestWebhookBuildBadRequest400(t *testing.T) {
	h := Deps{WebhookToken: "secret", LogDir: t.TempDir()}.webhookBuild
	for name, body := range map[string]string{
		"坏 JSON":     `{not json`,
		"缺 project":  `{"commit":"c1"}`,
		"缺 commit":   `{"project":"p1"}`,
		"非法 status":  `{"project":"p1","commit":"c1","status":"weird"}`,
		"项目名带斜杠":     `{"project":"../etc","commit":"c1"}`,
		"commit 带空格": `{"project":"p1","commit":"c 1"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/webhook/build", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer secret")
		rec := httptest.NewRecorder()
		h(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s:状态码 = %d,想要 400", name, rec.Code)
		}
	}
}
