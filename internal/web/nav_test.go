package web

import (
	"bytes"
	"html/template"
	"io/fs"
	"strings"
	"testing"

	assets "tally/web"
)

/* ---------- P2-01 人类导航投影:base.tmpl 模板级合同 ---------- */

// navPageData 是 base.tmpl 外壳所需的最小页面数据(content 有空默认定义,
// 只渲染外壳即可,不需要具体页面结构体)。
type navPageData struct {
	Title  string
	Active string
	CSRF   string
	Flash  struct {
		Kind string
		Msg  string
	}
}

// renderBase 渲染 base.tmpl 外壳,返回 HTML。模板解析失败直接 Fatal:
// 语法破坏会让 web.NewRouter 启动即 panic,属于不可恢复错误。
func renderBase(t *testing.T, active string) string {
	t.Helper()
	base, err := template.New("base").ParseFS(assets.TemplateFS, "templates/base.tmpl")
	if err != nil {
		t.Fatalf("解析 base.tmpl: %v", err)
	}
	var data navPageData
	data.Title = "t"
	data.Active = active
	data.CSRF = "tok"
	var buf bytes.Buffer
	if err := base.ExecuteTemplate(&buf, "base", data); err != nil {
		t.Fatalf("渲染 base: %v", err)
	}
	return buf.String()
}

// navSection 截取 <nav>…</nav> 之间的侧栏片段。
func navSection(t *testing.T, html string) string {
	t.Helper()
	start := strings.Index(html, `<nav class="rail">`)
	end := strings.Index(html, `</nav>`)
	if start < 0 || end < 0 || end < start {
		t.Fatalf("渲染结果里找不到导航区")
	}
	return html[start:end]
}

// navLine 返回导航区里含给定 href 的那一行(每个条目独占一行,便于精确断言
// aria-current 落点)。没有该条目则 Fatal。
func navLine(t *testing.T, nav, href string) string {
	t.Helper()
	for _, line := range strings.Split(nav, "\n") {
		if strings.Contains(line, `href="`+href+`"`) {
			return line
		}
	}
	t.Fatalf("导航里没有 %s 条目", href)
	return ""
}

// 研发导航条目:人类心智模型(PRD §6.1)到既有路由的映射。每一条都必须指向
// 真实存在、由真实读模型数据支撑的页面(AC:导航映射到既有/读模型数据)。
var humanNavEntries = map[string]string{
	"/v14/launch":    "发起",
	"/v13/tasks":     "工作",
	"/v13/runs":      "运行", // P2-03:实现 Run 过程事实时间线(/v13/runs 只读投影)
	"/v14/attention": "需人工",
}

// TestNavProjectsHumanEntriesOntoExistingRoutes:研发导航把人类词(发起/工作/需人工)
// 投影到既有路由上;条目文本与 href 一一对应,不允许空 href 或纯占位。
func TestNavProjectsHumanEntriesOntoExistingRoutes(t *testing.T) {
	nav := navSection(t, renderBase(t, ""))
	for href, label := range humanNavEntries {
		line := navLine(t, nav, href)
		if !strings.Contains(line, ">"+label+"<") {
			t.Errorf("导航 %s 条目的人读标签应为 %q,得到行:%s", href, label, line)
		}
		if strings.Contains(line, `href=""`) {
			t.Errorf("导航条目不允许空 href:%s", line)
		}
	}
}

// TestNavHighlightFollowsActiveKey:aria-current 只落在 Active 键对应的条目上,
// 其余条目不得同时高亮(一页只能有一个当前项)。
func TestNavHighlightFollowsActiveKey(t *testing.T) {
	cases := []struct{ active, href string }{
		{"v13", "/v13/tasks"},
		{"v13-runs", "/v13/runs"},
		{"v14-launch", "/v14/launch"},
		{"v14-attention", "/v14/attention"},
	}
	for _, tc := range cases {
		nav := navSection(t, renderBase(t, tc.active))
		for href := range humanNavEntries {
			line := navLine(t, nav, href)
			want := href == tc.href
			got := strings.Contains(line, `aria-current="page"`)
			if got != want {
				t.Errorf("Active=%q 时 %s 条目 aria-current 应为 %v,得到 %v(行:%s)", tc.active, href, want, got, line)
			}
		}
	}
}

// TestNavHasNoPlaceholderEntries(负路径):没有真实路由背书的概念绝不出现导航条目
// —— Rules 尚无 EngineeringPolicy 读取路由(属主 P3),从未存在的 /runs 与
// /v14/runs 路径也不得出现(Runs 的真实路由是 /v13/runs);同时导航区不得出现
// 占位话术。
func TestNavHasNoPlaceholderEntries(t *testing.T) {
	nav := navSection(t, renderBase(t, ""))
	for _, phantom := range []string{
		`href="/v14/runs"`, `href="/runs"`, `href="/rules"`, `href="/policies"`, `href="/reviews"`,
	} {
		if strings.Contains(nav, phantom) {
			t.Errorf("导航出现了没有真实路由背书的占位条目 %s(AC:禁止假功能占位)", phantom)
		}
	}
	for _, marker := range []string{"即将上线", "敬请期待", "coming soon", "占位"} {
		if strings.Contains(nav, marker) {
			t.Errorf("导航出现占位话术 %q", marker)
		}
	}
}

// TestNavHrefsAreKnownRoutes(负路径):导航区里出现的每一个 href 都必须在已知
// 真实路由集合内 —— 新增导航条目而忘记接真实路由时,本测试先红。
// 路由的 HTTP 可达性由 internal/web/handlers 的导航巡检测试另行实测。
func TestNavHrefsAreKnownRoutes(t *testing.T) {
	// 已存在的真实 GET 路由(web.NewRouter → handlers.RegisterRoutes / v13 / v14)。
	known := map[string]bool{
		"/": true, "/projects": true, "/requirements": true, "/tasks": true, "/coverage": true,
		"/docs": true, "/wiki": true,
		"/environments": true, "/deploys": true, "/failures": true,
		"/knowledge": true,
		// P2-01 研发导航条目(Runs 由 P2-03 落地 /v13/runs;Rules 由 P3-06 落地 /v14/rules)
		"/v14/launch": true, "/v13/tasks": true, "/v13/runs": true, "/v14/attention": true,
		"/v14/rules": true, "/v14/integration": true,
	}
	nav := navSection(t, renderBase(t, ""))
	for _, line := range strings.Split(nav, "\n") {
		i := strings.Index(line, `href="`)
		if i < 0 {
			continue
		}
		rest := line[i+len(`href="`):]
		j := strings.Index(rest, `"`)
		if j < 0 {
			t.Fatalf("导航行 href 语法异常:%s", line)
		}
		href := rest[:j]
		if strings.HasPrefix(href, "/static/") || href == "" {
			continue // 静态资源与空串不算页面导航条目
		}
		if !known[href] {
			t.Errorf("导航 href %q 不在已知真实路由集合内(新增条目必须先有真实路由)", href)
		}
	}
}

// TestNavTemplateParseFSIsComplete:base.tmpl 定义了 base 与 content 两个模板,
// 缺一个页面克隆就会在运行期才炸。
func TestNavTemplateParseFSIsComplete(t *testing.T) {
	base, err := template.New("base").ParseFS(assets.TemplateFS, "templates/base.tmpl")
	if err != nil {
		t.Fatalf("解析 base.tmpl: %v", err)
	}
	for _, name := range []string{"base", "content"} {
		if t2 := base.Lookup(name); t2 == nil {
			t.Errorf("base.tmpl 缺少模板定义 %q", name)
		}
	}
	// 模板文件必须真实存在于内嵌 FS(防路径改名后 ParseFS 静默空匹配)。
	if _, err := fs.Stat(assets.TemplateFS, "templates/base.tmpl"); err != nil {
		t.Fatalf("内嵌 FS 里找不到 templates/base.tmpl: %v", err)
	}
}
