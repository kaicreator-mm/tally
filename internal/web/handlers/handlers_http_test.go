package handlers_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"golang.org/x/crypto/bcrypt"

	"tally/internal/gitwriter"
	"tally/internal/knowledge"
	"tally/internal/web"
	"tally/internal/web/handlers"
)

/* ---------- 手写假实现(不引 mock 框架,见 docs/05-test-standards.md) ---------- */

// fakeSessions 是内存会话存储,供单元层测试绕过真实 Postgres 完成鉴权。
type fakeSessions struct {
	mu     sync.Mutex
	tokens map[string]bool
}

func (f *fakeSessions) CreateSession(_ context.Context, token string, _ time.Duration) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tokens[token] = true
	return nil
}
func (f *fakeSessions) SessionExists(_ context.Context, token string) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.tokens[token], nil
}
func (f *fakeSessions) DeleteSession(_ context.Context, token string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.tokens, token)
	return nil
}

// fakeTasks 提供算出的任务行,验证任务页只读渲染(无控件)。
type fakeTasks struct{ rows []handlers.TaskRow }

func (f fakeTasks) TaskRows(context.Context) ([]handlers.TaskRow, error) { return f.rows, nil }

// fakeKnowledge 提供预置的已验证修复条目,验证验证库页只读渲染(无编辑入口)。
type fakeKnowledge struct{ entries []knowledge.Entry }

func (f fakeKnowledge) ListVerified(context.Context) ([]knowledge.Entry, error) {
	return f.entries, nil
}

const sessionToken = "sess-token"

// newTestServer 起一个带临时 git 内容仓库 + 假会话的测试服务器。
// 返回 server、内容仓库路径、带 cookie jar 且不自动跟随重定向的客户端。
func newTestServer(t *testing.T, tasks handlers.TaskSource) (*httptest.Server, string, *http.Client, *url.URL) {
	t.Helper()
	return newTestServerFull(t, tasks, nil, nil)
}

// newTestServerFull 是 newTestServer 的可注入版:队列 / 验证库来源可给假实现。
func newTestServerFull(t *testing.T, tasks handlers.TaskSource, q handlers.JobQueue, kb handlers.KnowledgeSource) (*httptest.Server, string, *http.Client, *url.URL) {
	t.Helper()
	repo := t.TempDir()
	id := handlers.Identity{Name: "test", Email: "test@example.com"}
	if err := handlers.EnsureRepo(repo, id); err != nil {
		t.Fatalf("EnsureRepo: %v", err)
	}
	pw, err := bcrypt.GenerateFromPassword([]byte("pw"), 4)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	deps := web.Deps{
		Sessions:     &fakeSessions{tokens: map[string]bool{sessionToken: true}},
		PasswordHash: string(pw),
		ContentRepo:  repo,
		Identity:     id,
		Tasks:        tasks,
		Queue:        q,
		Knowledge:    kb,
	}
	srv := httptest.NewServer(web.NewRouter(deps))
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	u, _ := url.Parse(srv.URL)
	// 预置会话 cookie = 已登录。
	jar.SetCookies(u, []*http.Cookie{{Name: "tally_session", Value: sessionToken, Path: "/"}})

	client := &http.Client{
		Jar: jar,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse // 不跟随重定向,便于断言 303/Location
		},
	}
	return srv, repo, client, u
}

// primeCSRF 访问一个受保护 GET 页面,让中间件下发 csrf cookie 并存进 jar,返回其值。
func primeCSRF(t *testing.T, client *http.Client, srvURL, page string) string {
	t.Helper()
	resp, err := client.Get(srvURL + page)
	if err != nil {
		t.Fatalf("GET %s: %v", page, err)
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return csrfFromJar(t, client, srvURL)
}

func csrfFromJar(t *testing.T, client *http.Client, srvURL string) string {
	t.Helper()
	u, _ := url.Parse(srvURL)
	for _, c := range client.Jar.Cookies(u) {
		if c.Name == "tally_csrf" {
			return c.Value
		}
	}
	return ""
}

// postForm 带 CSRF 提交表单(PRGR:先 GET 拿 csrf,再 POST)。返回响应。
func postForm(t *testing.T, client *http.Client, srvURL, primePage, postPath string, form url.Values) *http.Response {
	t.Helper()
	token := primeCSRF(t, client, srvURL, primePage)
	if token == "" {
		t.Fatalf("拿不到 CSRF token(prime %s)", primePage)
	}
	if form == nil {
		form = url.Values{}
	}
	form.Set("csrf_token", token)
	resp, err := client.PostForm(srvURL+postPath, form)
	if err != nil {
		t.Fatalf("POST %s: %v", postPath, err)
	}
	return resp
}

func bodyString(t *testing.T, resp *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("读响应体: %v", err)
	}
	return string(b)
}

func commitCount(t *testing.T, repo string) int {
	t.Helper()
	r, err := git.PlainOpen(repo)
	if err != nil {
		t.Fatalf("打开仓库: %v", err)
	}
	it, err := r.Log(&git.LogOptions{})
	if err != nil {
		t.Fatalf("日志: %v", err)
	}
	n := 0
	_ = it.ForEach(func(*object.Commit) error { n++; return nil })
	return n
}

/* ---------- 登录与写边界(INV-9 / CSRF) ---------- */

func TestUnauthenticatedProtectedRouteRedirectsToLogin(t *testing.T) {
	srv, _, _, _ := newTestServer(t, nil)
	// 无会话 cookie 的客户端。
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	for _, path := range []string{"/", "/projects", "/tasks", "/wiki", "/docs", "/requirements"} {
		resp, err := client.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("未登录访问 %s 应 303,得到 %d", path, resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != "/login" {
			t.Errorf("未登录访问 %s 应重定向 /login,得到 %q", path, loc)
		}
	}
}

func TestUnauthenticatedWriteRejected(t *testing.T) {
	srv, _, _, _ := newTestServer(t, nil)
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.PostForm(srv.URL+"/projects", url.Values{"slug": {"x"}, "name": {"X"}})
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		t.Errorf("未登录写请求不应成功,得到 %d", resp.StatusCode)
	}
}

func TestPostWithoutCSRFRejected(t *testing.T) {
	srv, _, client, _ := newTestServer(t, nil)
	// 已登录(有 session cookie),但不带 csrf token 直接 POST。
	resp, err := client.PostForm(srv.URL+"/projects", url.Values{"slug": {"x"}, "name": {"X"}})
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("缺 CSRF token 的 POST 应 403,得到 %d", resp.StatusCode)
	}
}

func TestWrongCSRFRejected(t *testing.T) {
	srv, _, client, _ := newTestServer(t, nil)
	primeCSRF(t, client, srv.URL, "/projects/new") // 让中间件下发 cookie
	resp, err := client.PostForm(srv.URL+"/projects", url.Values{
		"slug": {"x"}, "name": {"X"}, "csrf_token": {"错的-token"},
	})
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("CSRF 不匹配应 403,得到 %d", resp.StatusCode)
	}
}

/* ---------- 项目管理:写 git + commit ---------- */

func TestCreateProjectWritesGitCommit(t *testing.T) {
	srv, repo, client, _ := newTestServer(t, nil)
	before := commitCount(t, repo)

	resp := postForm(t, client, srv.URL, "/projects/new", "/projects", url.Values{
		"slug": {"forge"}, "name": {"Forge"}, "description": {"一个构建工具"},
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("建项目应 303 重定向,得到 %d", resp.StatusCode)
	}

	// 文件真落盘。
	b, err := os.ReadFile(filepath.Join(repo, "projects", "forge.md"))
	if err != nil {
		t.Fatalf("项目文件应存在: %v", err)
	}
	if !strings.Contains(string(b), "# Forge") {
		t.Errorf("项目文件内容不对: %s", b)
	}
	// git 里真有新 commit。
	if after := commitCount(t, repo); after != before+1 {
		t.Errorf("建项目后 commit 应 +1: before=%d after=%d", before, after)
	}
}

func TestUpdateProjectWritesGitCommit(t *testing.T) {
	srv, repo, client, _ := newTestServer(t, nil)
	postForm(t, client, srv.URL, "/projects/new", "/projects", url.Values{
		"slug": {"forge"}, "name": {"Forge"}, "description": {"旧说明"},
	}).Body.Close()
	before := commitCount(t, repo)

	resp := postForm(t, client, srv.URL, "/projects/forge/edit", "/projects/forge", url.Values{
		"name": {"Forge 改"}, "description": {"新说明"},
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("改项目应 303,得到 %d", resp.StatusCode)
	}
	b, _ := os.ReadFile(filepath.Join(repo, "projects", "forge.md"))
	if !strings.Contains(string(b), "新说明") {
		t.Errorf("项目未更新: %s", b)
	}
	if after := commitCount(t, repo); after != before+1 {
		t.Errorf("改项目后 commit 应 +1: before=%d after=%d", before, after)
	}
}

func TestCreateProjectRejectsBadSlug(t *testing.T) {
	srv, repo, client, _ := newTestServer(t, nil)
	// 越界的 slug(路径穿越)必须被拒,不落盘。
	resp := postForm(t, client, srv.URL, "/projects/new", "/projects", url.Values{
		"slug": {"../evil"}, "name": {"Evil"},
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("越界 slug 应 400,得到 %d", resp.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(repo, "projects", "..", "evil.md")); !os.IsNotExist(err) {
		t.Errorf("越界 slug 不应落盘")
	}
}

// 关掉 JS(不发任何 htmx 头)整页提交仍能完成写入 —— htmx 只是渐进增强。
func TestNoJSFullPageSubmitStillWrites(t *testing.T) {
	srv, repo, client, _ := newTestServer(t, nil)
	// client 从不设置 HX-* 头:等价于禁用了 JS 的浏览器。
	resp := postForm(t, client, srv.URL, "/projects/new", "/projects", url.Values{
		"slug": {"hsk"}, "name": {"HSK"}, "description": {"无 JS 提交"},
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("无 JS 整页提交应 303,得到 %d", resp.StatusCode)
	}
	if _, err := os.Stat(filepath.Join(repo, "projects", "hsk.md")); err != nil {
		t.Errorf("无 JS 提交后文件应存在: %v", err)
	}
}

/* ---------- 需求:写 git,复用 requirement 包展示 ---------- */

func TestCreateRequirementWritesGit(t *testing.T) {
	srv, repo, client, _ := newTestServer(t, nil)
	content := "# 多租户\n\n## REQ-1 隔离\n\nWHEN 请求带租户标识, THE 系统 SHALL 只返回该租户数据。\n\n验收: 自动化测试\n- 全绿\n\n## 非目标\n- 不做迁移\n\n## 术语表\n- 租户: 组织账号\n"
	postForm(t, client, srv.URL, "/requirements", "/requirements", url.Values{
		"project": {"forge"}, "file": {"multi-tenant.md"}, "content": {content},
	}).Body.Close()

	path := filepath.Join(repo, ".tally", "requirements", "forge", "multi-tenant.md")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("需求文件应存在: %v", err)
	}
	// 编辑页展示解析出的条款与内容哈希(只读)。
	resp, err := client.Get(srv.URL + "/requirements/forge/multi-tenant.md")
	if err != nil {
		t.Fatalf("GET 需求: %v", err)
	}
	body := bodyString(t, resp)
	if !strings.Contains(body, "REQ-1") {
		t.Errorf("编辑页应展示解析出的条款 REQ-1")
	}
	if !strings.Contains(body, "内容哈希") {
		t.Errorf("编辑页应展示内容哈希(INV-4)")
	}
}

/* ---------- 文档 / Wiki:写 git,Wiki 标「笔记·未验证」,markdown 安全渲染 ---------- */

func TestDocAndWikiWriteGit(t *testing.T) {
	srv, repo, client, _ := newTestServer(t, nil)
	postForm(t, client, srv.URL, "/docs/new", "/docs", url.Values{
		"slug": {"arch"}, "content": {"# 架构\n\n正文 <script>x</script>"},
	}).Body.Close()
	postForm(t, client, srv.URL, "/wiki/new", "/wiki", url.Values{
		"slug": {"idea"}, "content": {"# 一个想法\n\n还没验证。"},
	}).Body.Close()

	if _, err := os.Stat(filepath.Join(repo, "docs", "arch.md")); err != nil {
		t.Errorf("文档应落 git: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "wiki", "idea.md")); err != nil {
		t.Errorf("笔记应落 git: %v", err)
	}

	// 文档渲染:markdown 生效,且原始 <script> 被转义。
	resp, _ := client.Get(srv.URL + "/docs/arch")
	body := bodyString(t, resp)
	if !strings.Contains(body, "<h1>架构</h1>") {
		t.Errorf("文档应渲染 H1")
	}
	if strings.Contains(body, "<script>x</script>") {
		t.Errorf("文档渲染不应放出原始 <script>(XSS)")
	}

	// Wiki 渲染:必须有「笔记·未验证」标。
	resp2, _ := client.Get(srv.URL + "/wiki/idea")
	body2 := bodyString(t, resp2)
	if !strings.Contains(body2, "笔记 · 未验证") {
		t.Errorf("Wiki 页应标「笔记·未验证」")
	}
}

/* ---------- 任务:状态算出只读,无任何状态控件(INV-3) ---------- */

func TestTasksPageHasNoStatusControls(t *testing.T) {
	// 即便有算出的任务行,页面也不得出现勾选框 / 下拉 / 状态表单。
	tasks := fakeTasks{rows: []handlers.TaskRow{
		{Project: "forge", Title: "登录改造", Merged: "已合并", Built: "通过", Tested: "通过", Deployed: "成功", Stable: "稳定"},
	}}
	srv, _, client, _ := newTestServer(t, tasks)
	resp, err := client.Get(srv.URL + "/tasks")
	if err != nil {
		t.Fatalf("GET /tasks: %v", err)
	}
	body := bodyString(t, resp)
	if !strings.Contains(body, "登录改造") {
		t.Errorf("任务页应展示算出的任务行")
	}
	for _, banned := range []string{`type="checkbox"`, `<select`, `name="status"`, "/status", `type="radio"`} {
		if strings.Contains(body, banned) {
			t.Errorf("任务页不应出现状态控件 %q(INV-3)", banned)
		}
	}
}

func TestTasksPageEmptyStateIsActionPrompt(t *testing.T) {
	srv, _, client, _ := newTestServer(t, nil) // 无 TaskSource → 空
	resp, err := client.Get(srv.URL + "/tasks")
	if err != nil {
		t.Fatalf("GET /tasks: %v", err)
	}
	body := bodyString(t, resp)
	if strings.Contains(body, "暂无数据") {
		t.Errorf("空状态不应是「暂无数据」")
	}
	if !strings.Contains(body, "执行数据") && !strings.Contains(body, "构建") {
		t.Errorf("空状态应是行动提示,说明状态怎么来")
	}
}

/* ---------- 空状态是行动提示(项目列表) ---------- */

func TestProjectsEmptyStateIsActionPrompt(t *testing.T) {
	srv, _, client, _ := newTestServer(t, nil)
	resp, err := client.Get(srv.URL + "/projects")
	if err != nil {
		t.Fatalf("GET /projects: %v", err)
	}
	body := bodyString(t, resp)
	if strings.Contains(body, "暂无数据") {
		t.Errorf("空状态不应是「暂无数据」")
	}
	if !strings.Contains(body, "建第一个项目") {
		t.Errorf("项目空状态应是行动提示")
	}
}

/* ---------- 已验证修复库:只读无编辑入口(INV-2),与 Wiki 笔记层视觉明确可分 ---------- */

// 知识 Wiki 双层(核心):验证库页面不许出现任何编辑 / 新增入口,只能经失败处理流程入库。
func TestKnowledgePageHasNoEditEntry(t *testing.T) {
	kb := fakeKnowledge{entries: []knowledge.Entry{
		{Fingerprint: "a3f9c2d4", Repro: "go build ./...", Summary: "升级 node 到 22",
			VerifiedCombo: map[string]string{"node": "22.3.0"}},
	}}
	srv, _, client, _ := newTestServerFull(t, nil, nil, kb)
	resp, err := client.Get(srv.URL + "/knowledge")
	if err != nil {
		t.Fatalf("GET /knowledge: %v", err)
	}
	body := bodyString(t, resp)

	// 条目内容渲染:指纹条、复现命令、修复内容、验证组合。
	if !strings.Contains(body, "fp-strip") {
		t.Errorf("验证库条目应渲染指纹条")
	}
	if !strings.Contains(body, "go build ./...") {
		t.Errorf("验证库条目应展示复现命令")
	}
	if !strings.Contains(body, "node=22.3.0") {
		t.Errorf("验证库条目应展示验证时版本组合")
	}

	// INV-2:无任何编辑 / 新增入口 —— 断言内容区(<main>)无表单、无写路由、无编辑/新增按钮。
	// 断言收窄到内容区:base.tmpl 的登出表单在 <main> 之外,不算验证库的编辑入口。
	mainStart := strings.Index(body, "<main>")
	if mainStart < 0 {
		t.Fatalf("页面应有 <main> 内容区")
	}
	content := body[mainStart:]
	for _, banned := range []string{
		`<form`, `method="post"`, `action="/knowledge`,
		"新增", "编辑", `type="submit"`, `class="btn`,
	} {
		if strings.Contains(content, banned) {
			t.Errorf("验证库内容区不应出现编辑/新增入口 %q(INV-2)", banned)
		}
	}
}

// 两层徽章视觉明确可分:验证库 = --verified 绿 +「已验证」(badge-verified);
// Wiki 笔记 = --stale 赭石 +「笔记·未验证」(badge-note)。两页不得共用同一徽章。
func TestKnowledgeVsWikiBadgeDistinct(t *testing.T) {
	kb := fakeKnowledge{entries: []knowledge.Entry{
		{Fingerprint: "a3f9c2d4", Repro: "r", Summary: "s"},
	}}
	srv, _, client, _ := newTestServerFull(t, nil, nil, kb)

	// 验证库页:badge-verified + 已验证,且无 badge-note。
	kresp, err := client.Get(srv.URL + "/knowledge")
	if err != nil {
		t.Fatalf("GET /knowledge: %v", err)
	}
	kbody := bodyString(t, kresp)
	if !strings.Contains(kbody, "badge-verified") || !strings.Contains(kbody, "已验证") {
		t.Errorf("验证库页应带 badge-verified「已验证」标")
	}
	if strings.Contains(kbody, "badge-note") {
		t.Errorf("验证库页不应出现笔记层的 badge-note")
	}

	// Wiki 笔记页:badge-note + 笔记·未验证,且无 badge-verified。
	postForm(t, client, srv.URL, "/wiki/new", "/wiki", url.Values{
		"slug": {"idea"}, "content": {"# 想法\n\n未验证"},
	}).Body.Close()
	wresp, err := client.Get(srv.URL + "/wiki/idea")
	if err != nil {
		t.Fatalf("GET /wiki/idea: %v", err)
	}
	wbody := bodyString(t, wresp)
	if !strings.Contains(wbody, "badge-note") || !strings.Contains(wbody, "笔记 · 未验证") {
		t.Errorf("笔记页应带 badge-note「笔记·未验证」标")
	}
	if strings.Contains(wbody, "badge-verified") {
		t.Errorf("笔记页不应出现验证库的 badge-verified")
	}
}

func TestKnowledgeEmptyStateIsActionPrompt(t *testing.T) {
	srv, _, client, _ := newTestServerFull(t, nil, nil, fakeKnowledge{}) // 空库
	resp, err := client.Get(srv.URL + "/knowledge")
	if err != nil {
		t.Fatalf("GET /knowledge: %v", err)
	}
	body := bodyString(t, resp)
	if strings.Contains(body, "暂无数据") {
		t.Errorf("空状态不应是「暂无数据」")
	}
	if !strings.Contains(body, "失败处理流程") || !strings.Contains(body, "四道闸") {
		t.Errorf("空状态应是行动提示,说明修复怎么入库")
	}
}

/* ---------- P0-16:ContentStore 提交统一走 gitwriter 单一写边界 ---------- */

// gitDirOf 返回仓库的 git dir(gitwriter 放跨进程锁文件的位置)。
func gitDirOf(t *testing.T, repo string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", repo, "rev-parse", "--absolute-git-dir")
	var out, errBuf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("定位 git dir: %v: %s", err, strings.TrimSpace(errBuf.String()))
	}
	return strings.TrimSpace(out.String())
}

// gitStatusPorcelain 返回 git status --porcelain 输出(空 = 工作区/暂存区干净)。
func gitStatusPorcelain(t *testing.T, repo string) string {
	t.Helper()
	cmd := exec.Command("git", "-C", repo, "status", "--porcelain")
	var out, errBuf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errBuf
	if err := cmd.Run(); err != nil {
		t.Fatalf("git status: %v: %s", err, strings.TrimSpace(errBuf.String()))
	}
	return strings.TrimSpace(out.String())
}

// TestContentStoreHasNoDirectGitCommit(P0-16 AC1 的回归闸):content.go 不得再出现
// go-git worktree 直接提交;提交必须经 internal/gitwriter。
func TestContentStoreHasNoDirectGitCommit(t *testing.T) {
	// go test 的工作目录即本包目录。
	src, err := os.ReadFile("content.go")
	if err != nil {
		t.Fatalf("读 content.go: %v", err)
	}
	s := string(src)
	for _, banned := range []string{
		"Worktree()",      // 直接开 worktree 写入
		"CommitOptions",   // go-git 提交选项
		"plumbing/object", // go-git 签名/对象构造
	} {
		if strings.Contains(s, banned) {
			t.Errorf("content.go 不应再出现直接 git 写入痕迹 %q(P0-16 AC1)", banned)
		}
	}
	if !strings.Contains(s, "gitwriter.Commit") {
		t.Errorf("ContentStore 的提交应经 gitwriter(P0-16 AC1)")
	}
}

// TestContentStoreCommitRejectsStaleBase(P0-16 AC2,确定性):基线已前进后,
// 携带旧基线的写入必须被 gitwriter 以 ErrBaseRevision 拒绝,HEAD 不被覆盖、
// 文件不落盘、工作区不留残渣 —— 冲突显式暴露,绝不静默叠写。
func TestContentStoreCommitRejectsStaleBase(t *testing.T) {
	repo := t.TempDir()
	id := handlers.Identity{Name: "test", Email: "test@example.com"}
	if err := handlers.EnsureRepo(repo, id); err != nil {
		t.Fatalf("EnsureRepo: %v", err)
	}
	w, err := gitwriter.New(repo)
	if err != nil {
		t.Fatalf("gitwriter.New: %v", err)
	}
	ctx := context.Background()
	staleBase, err := w.Head(ctx)
	if err != nil {
		t.Fatalf("Head: %v", err)
	}

	cs := &handlers.ContentStore{RepoPath: repo, Identity: id}
	if _, err := cs.Commit("projects/first.md", []byte("first"), "feat: first"); err != nil {
		t.Fatalf("首次提交: %v", err)
	}

	// 模拟与首次提交并发编写、仍以旧基线提交的变更 → 必须被拒。
	_, err = w.Commit(ctx, gitwriter.CommitRequest{
		ExpectedBase: staleBase,
		Message:      "feat: stale",
		Files:        []gitwriter.FileUpdate{{Path: "projects/stale.md", Content: []byte("stale")}},
		Signature:    &gitwriter.Signature{Name: id.Name, Email: id.Email},
	})
	if !errors.Is(err, gitwriter.ErrBaseRevision) {
		t.Fatalf("旧基线提交应报 ErrBaseRevision,得到: %v", err)
	}
	head, err := w.Head(ctx)
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	if head == staleBase {
		t.Errorf("被拒后 HEAD 不应停在旧基线(首次提交不应被回滚)")
	}
	if _, err := os.Stat(filepath.Join(repo, "projects", "stale.md")); !os.IsNotExist(err) {
		t.Errorf("被拒提交的文件不应落盘")
	}
	if out := gitStatusPorcelain(t, repo); out != "" {
		t.Errorf("被拒后工作区应干净,得到: %s", out)
	}
}

// TestContentStoreCommitSurfacesBaseMismatch(P0-16 AC2,经 ContentStore 并发路径):
// 两个写入者以同一基线并发提交时,恰好一方成功,另一方必须以基线冲突
// (gitwriter.ErrBaseRevision,经 ContentStore 包装后仍可 errors.Is 识别)显式失败
// 并完整回滚,绝不静默叠写。极端调度下双方可能恰好先后串行(后者重读到新基线),
// 这种轮次直接换一轮重试;若并发叠写回归(负方悄悄成功),任何一轮都不会出现冲突。
func TestContentStoreCommitSurfacesBaseMismatch(t *testing.T) {
	const rounds = 5
	for i := 1; i <= rounds; i++ {
		surfaced, err := raceTwoSameBaseCommits(t)
		if err != nil {
			t.Fatalf("第 %d 轮: %v", i, err)
		}
		if surfaced {
			return
		}
	}
	t.Fatalf("%d 轮同基线并发提交均未出现基线冲突:期望一方以 gitwriter.ErrBaseRevision 失败", rounds)
}

// raceTwoSameBaseCommits 让两个 ContentStore 写入者并发提交,并手工占住 gitwriter 的
// 跨进程锁文件(名字同 internal/gitwriter/lock.go 的 lockFileName)把 HEAD 冻结在初始
// 提交上,保证双方同基线出发。返回是否观察到「恰好一方成功,另一方以
// gitwriter.ErrBaseRevision 失败且无任何残留」。
func raceTwoSameBaseCommits(t *testing.T) (bool, error) {
	t.Helper()
	repo := t.TempDir()
	id := handlers.Identity{Name: "test", Email: "test@example.com"}
	if err := handlers.EnsureRepo(repo, id); err != nil {
		return false, fmt.Errorf("EnsureRepo: %w", err)
	}
	cs := &handlers.ContentStore{RepoPath: repo, Identity: id}

	lockPath := filepath.Join(gitDirOf(t, repo), "tally-gitwriter.lock")
	if err := os.WriteFile(lockPath, []byte("{}"), 0o666); err != nil {
		return false, err
	}
	defer os.Remove(lockPath) // 锁内容无关紧要,仅存在即冻结 HEAD;TempDir 兜底清理

	type result struct {
		path string
		err  error
	}
	started := make(chan struct{})
	results := make(chan result, 2)
	race := func(path, content string) {
		go func() {
			started <- struct{}{}
			_, err := cs.Commit(path, []byte(content), "feat: "+path)
			results <- result{path: path, err: err}
		}()
	}
	race("projects/racer-a.md", "A")
	race("projects/racer-b.md", "B")
	<-started
	<-started // 双方已动身;此刻才放行第一个提交,二者只可能读到同一基线
	if err := os.Remove(lockPath); err != nil {
		return false, err
	}

	wins, baseErrs := 0, 0
	loserPath := ""
	for i := 0; i < 2; i++ {
		r := <-results
		switch {
		case r.err == nil:
			wins++
		case errors.Is(r.err, gitwriter.ErrBaseRevision):
			baseErrs++
			loserPath = r.path
		default:
			return false, fmt.Errorf("写入者 %s 意外失败: %w", r.path, r.err)
		}
	}
	if wins != 1 || baseErrs != 1 {
		return false, nil // 本轮双双先后串行成功:非冲突路径,换一轮再试
	}

	// 负方绝不落盘;仓库恰好 +1 提交;工作区无叠写残渣。
	if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(loserPath))); !os.IsNotExist(err) {
		return false, fmt.Errorf("基线冲突方的文件 %s 不应落盘", loserPath)
	}
	if n := commitCount(t, repo); n != 2 {
		return false, fmt.Errorf("初始提交+1 次成功提交后应有 2 个提交,得到 %d", n)
	}
	if out := gitStatusPorcelain(t, repo); out != "" {
		return false, fmt.Errorf("工作区应干净: %s", out)
	}
	return true, nil
}
