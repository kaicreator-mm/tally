package handlers_test

import (
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

/* ---------- P2-01 人类导航投影:HTTP 层合同(真实路由可达性 + 高亮接线) ---------- */

// navRail 截取已渲染页面里 <nav class="rail">…</nav> 的侧栏片段。
func navRail(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, `<nav class="rail">`)
	end := strings.Index(body, `</nav>`)
	if start < 0 || end < 0 || end < start {
		t.Fatalf("页面缺少导航区")
	}
	return body[start:end]
}

// navHrefLine 在导航片段里找含给定 href 的行(每条目独占一行)。
func navHrefLine(t *testing.T, nav, href string) string {
	t.Helper()
	for _, line := range strings.Split(nav, "\n") {
		if strings.Contains(line, `href="`+href+`"`) {
			return line
		}
	}
	t.Fatalf("导航里没有 %s 条目", href)
	return ""
}

var navHrefs = regexp.MustCompile(`href="(/[^"]*)"`)

// humanNavEntries 是研发导航的四个条目(发起/工作/运行/需人工),全部映射到
// 既有只读路由(/v14/launch、/v13/tasks、/v13/runs、/v14/attention)。
var humanNavEntries = []string{"/v14/launch", "/v13/tasks", "/v13/runs", "/v14/attention"}

// TestNavLinksResolveToRealRoutes(AC:既有路由保持有效;导航映射到既有/读模型数据):
// 把导航区渲染出的每一个 href 都经真实路由器 GET 一遍 —— 404/405/500 即失败。
// 这是负路径守卫:任何人往导航里加了没有真实路由的占位条目,本测试先红。
// 空数据源(nil)下各页按既有合同渲染「未接入」空状态,仍是 200,不是占位页。
func TestNavLinksResolveToRealRoutes(t *testing.T) {
	srv, _, client, _ := newTestServer(t, nil)

	resp, err := client.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	body := bodyString(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET / 应 200,得到 %d", resp.StatusCode)
	}
	nav := navRail(t, body)

	for _, href := range humanNavEntries {
		if !strings.Contains(nav, `href="`+href+`"`) {
			t.Errorf("研发导航缺少条目 %s", href)
		}
	}

	seen := map[string]bool{}
	for _, m := range navHrefs.FindAllStringSubmatch(nav, -1) {
		path := m[1]
		if seen[path] {
			continue
		}
		seen[path] = true
		r, err := client.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET 导航目标 %s: %v", path, err)
		}
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
		if r.StatusCode == http.StatusNotFound || r.StatusCode == http.StatusMethodNotAllowed || r.StatusCode >= 500 {
			t.Errorf("导航条目 %s 指向的路由不可用:HTTP %d(导航不得指向占位/不存在路由)", path, r.StatusCode)
		}
	}
	if len(seen) < 10 {
		t.Fatalf("导航解析出的路由过少(%d),提取逻辑可能失效", len(seen))
	}
}

// TestNavHighlightWiringOverHTTP:研发导航条目的 aria-current 接线 ——
// 当前页对应条目高亮,同组其余条目不得同时高亮(每页至多一个当前项)。
func TestNavHighlightWiringOverHTTP(t *testing.T) {
	srv, _, client, _ := newTestServer(t, nil)

	cases := []struct{ page, activeHref string }{
		{"/v13/tasks", "/v13/tasks"},
		{"/v13/runs", "/v13/runs"},
		{"/v14/launch", "/v14/launch"},
		{"/v14/attention", "/v14/attention"},
	}
	for _, tc := range cases {
		resp, err := client.Get(srv.URL + tc.page)
		if err != nil {
			t.Fatalf("GET %s: %v", tc.page, err)
		}
		body := bodyString(t, resp)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET %s 应 200,得到 %d", tc.page, resp.StatusCode)
		}
		nav := navRail(t, body)
		for _, href := range humanNavEntries {
			line := navHrefLine(t, nav, href)
			want := href == tc.activeHref
			if got := strings.Contains(line, `aria-current="page"`); got != want {
				t.Errorf("GET %s:条目 %s 的 aria-current 应为 %v,得到 %v(行:%s)", tc.page, href, want, got, line)
			}
		}
	}
}
