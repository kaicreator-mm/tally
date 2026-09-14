//go:build integration

package handlers_test

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"tally/internal/store"
	"tally/internal/web"
	"tally/internal/web/handlers"
)

// 集成测试用真实 Postgres 的 session 表跑完整登录链路(不 mock 数据库,见 docs/05-test-standards.md)。
// 每测试 store.TestDB 拿独立 schema。

func TestLoginFlowWithRealPostgresSession(t *testing.T) {
	ctx := context.Background()
	st := store.TestDB(t)

	repo := t.TempDir()
	id := handlers.Identity{Name: "itest", Email: "itest@example.com"}
	if err := handlers.EnsureRepo(repo, id); err != nil {
		t.Fatalf("EnsureRepo: %v", err)
	}
	pw, err := bcrypt.GenerateFromPassword([]byte("s3cret"), 4)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	deps := web.Deps{
		Sessions:     web.PGSessions(st.Queries()), // 真实 session 表
		PasswordHash: string(pw),
		ContentRepo:  repo,
		Identity:     id,
	}
	srv := httptest.NewServer(web.NewRouter(deps))
	t.Cleanup(srv.Close)

	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// 未登录访问受保护页 → 重定向 /login。
	resp, err := client.Get(srv.URL + "/projects")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Fatalf("未登录应 303→/login,得到 %d→%s", resp.StatusCode, resp.Header.Get("Location"))
	}

	// 密码错 → 回登录页带 error,且未签发会话。
	resp, err = client.PostForm(srv.URL+"/login", url.Values{"password": {"错的"}})
	if err != nil {
		t.Fatalf("登录: %v", err)
	}
	resp.Body.Close()
	if loc := resp.Header.Get("Location"); !strings.Contains(loc, "/login") || !strings.Contains(loc, "error") {
		t.Errorf("密码错应重定向 /login?error,得到 %q", loc)
	}
	if hasSession(jar, srv.URL) {
		t.Errorf("密码错不应签发会话")
	}

	// 密码对 → 签发会话(session 存真实 Postgres 表)。
	resp, err = client.PostForm(srv.URL+"/login", url.Values{"password": {"s3cret"}})
	if err != nil {
		t.Fatalf("登录: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/" {
		t.Fatalf("登录成功应 303→/,得到 %d→%s", resp.StatusCode, resp.Header.Get("Location"))
	}
	if !hasSession(jar, srv.URL) {
		t.Fatalf("登录成功应签发会话 cookie")
	}

	// 带真实会话访问受保护页 → 200。
	resp, err = client.Get(srv.URL + "/projects")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("已登录访问 /projects 应 200,得到 %d", resp.StatusCode)
	}

	// 已登录 + CSRF,整页提交建项目 → 落 git。
	postForm(t, client, srv.URL, "/projects/new", "/projects", url.Values{
		"slug": {"forge"}, "name": {"Forge"}, "description": {"集成测试"},
	}).Body.Close()
	if _, err := os.Stat(filepath.Join(repo, "projects", "forge.md")); err != nil {
		t.Errorf("真实会话下建项目应落 git: %v", err)
	}

	// 登出 → 会话从真实 Postgres 表删除,再访问受保护页又被挡。
	token := sessionValue(jar, srv.URL)
	resp, err = client.PostForm(srv.URL+"/logout", url.Values{"csrf_token": {csrfFromJar(t, client, srv.URL)}})
	if err != nil {
		t.Fatalf("登出: %v", err)
	}
	resp.Body.Close()
	// 直接查库:该 session 应已删除(查不到)。
	if _, err := st.Queries().GetSession(ctx, token); err == nil {
		t.Errorf("登出后 session 应已从 Postgres 删除,但仍能查到")
	}
	// 用旧会话再访问受保护页 → 又被重定向到 /login。
	resp, err = client.Get(srv.URL + "/projects")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Errorf("登出后访问受保护页应再次 303→/login,得到 %d→%s", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func hasSession(jar *cookiejar.Jar, srvURL string) bool {
	return sessionValue(jar, srvURL) != ""
}

func sessionValue(jar *cookiejar.Jar, srvURL string) string {
	u, _ := url.Parse(srvURL)
	for _, c := range jar.Cookies(u) {
		if c.Name == "tally_session" && c.Value != "" {
			return c.Value
		}
	}
	return ""
}
