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

type navPageData struct {
	Title  string
	Active string
	CSRF   string
	Flash  struct {
		Kind string
		Msg  string
	}
	ControlPlane bool
}

func renderBase(t *testing.T, active string) string {
	t.Helper()
	return renderBaseWithControlPlane(t, active, true)
}

func renderBaseWithControlPlane(t *testing.T, active string, controlPlane bool) string {
	t.Helper()
	base, err := template.New("base").ParseFS(assets.TemplateFS, "templates/base.tmpl")
	if err != nil {
		t.Fatalf("解析 base.tmpl: %v", err)
	}
	var data navPageData
	data.Title = "t"
	data.Active = active
	data.CSRF = "tok"
	data.ControlPlane = controlPlane
	var buf bytes.Buffer
	if err := base.ExecuteTemplate(&buf, "base", data); err != nil {
		t.Fatalf("渲染 base: %v", err)
	}
	return buf.String()
}

func navSection(t *testing.T, html string) string {
	t.Helper()
	start := strings.Index(html, `<nav class="rail">`)
	end := strings.Index(html, `</nav>`)
	if start < 0 || end < 0 || end < start {
		t.Fatalf("渲染结果里找不到导航区")
	}
	return html[start:end]
}

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

// 研发导航条目:人类心智模型到真实路由的映射。v1.5 Engineering Control
// Plane is now a real protected route backed by the Active TaskDAG read model.
var humanNavEntries = map[string]string{
	"/v15/control-plane": "工程控制台",
	"/v14/launch":        "发起",
	"/v13/tasks":         "工作",
	"/v13/runs":          "运行",
	"/v14/attention":     "需人工",
}

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

// 控制台未挂载(ControlPlane=false)时导航不得出现工程控制台条目:
// 没有真实路由背书的条目宁可不出现,绝不放占位入口;其余研发条目照常。
func TestNavHidesControlPlaneEntryWhenDisabled(t *testing.T) {
	nav := navSection(t, renderBaseWithControlPlane(t, "", false))
	if strings.Contains(nav, `href="/v15/control-plane"`) {
		t.Errorf("控制台未挂载时导航仍出现 /v15/control-plane 占位条目")
	}
	for href, label := range map[string]string{
		"/v14/launch":    "发起",
		"/v13/tasks":     "工作",
		"/v13/runs":      "运行",
		"/v14/attention": "需人工",
	} {
		line := navLine(t, nav, href)
		if !strings.Contains(line, ">"+label+"<") {
			t.Errorf("禁用控制台后 %s 条目应保留且标签为 %q,得到:%s", href, label, line)
		}
	}
}

func TestNavHighlightFollowsActiveKey(t *testing.T) {
	cases := []struct{ active, href string }{
		{"v15-control-plane", "/v15/control-plane"},
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

func TestNavHrefsAreKnownRoutes(t *testing.T) {
	known := map[string]bool{
		"/": true, "/projects": true, "/requirements": true, "/tasks": true, "/coverage": true,
		"/docs": true, "/wiki": true,
		"/environments": true, "/deploys": true, "/failures": true,
		"/knowledge": true,
		"/v15/control-plane": true,
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
			continue
		}
		if !known[href] {
			t.Errorf("导航 href %q 不在已知真实路由集合内(新增条目必须先有真实路由)", href)
		}
	}
}

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
	if _, err := fs.Stat(assets.TemplateFS, "templates/base.tmpl"); err != nil {
		t.Fatalf("内嵌 FS 里找不到 templates/base.tmpl: %v", err)
	}
}
