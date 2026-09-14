//go:build integration

// P2-05 — Human-centered UX gate(任务合同:验证结构化决策流是主路径、内部
// 状态机词汇不主导用户导航;三条 AC 可执行化)。
//
// 被测面 = 生产路由原样装配(handlers.Server.RegisterRoutes,与组合根同一
// 注册函数,末尾已组合 RegisterV14Routes;登录/CSRF 组的行为由既有 web 层
// 测试覆盖,与 P0-09 门禁同一注记)+ 真实内容仓库(p09Repo 同一引导)+
// 真实事实面(store.TestDB 每测试独立 schema)。
//
// 三条 AC 的可执行形态:
//
//	AC1 核心任务可不经 Chat-first 流创建/评审:launch 页的结构化意图表单是
//	    唯一入口 —— 一个有界 `<label for="v14-intent">意图(一句自然语言,
//	    唯一必填项)</label>` textarea + 服务端澄清闭环(P0-03/04 的有界
//	    schema),没有自由聊天 transcript 界面;评审面 = /v13/tasks(工作)+
//	    /v14/attention(需人工评审,逐条原因/依据/目标动作)。
//	AC2 键盘/基础表单可访问性:表单控件经 label[for] 关联;表格有 th 表头;
//	    页面无空文本链接(<a ...></a> 无可聚焦文本 = 键盘死角)。
//	AC3 无任意 LLM 生成 UI 执行:渲染面唯一的 <script> 是 base.tmpl 的静态
//	    htmx 引用(defer、仓库内静态文件、零模板插值);LLM 来源值一律经
//	    html/template 上下文感知转义作为数据渲染,不存在内联脚本执行缝。
//
// 门禁目标(状态机词汇不主导导航):导航轨道用固定人读标签(发起/工作/运行/
// 需人工评审),生命周期机器词面(NEEDS_REVIEW/VALIDATING/… )只允许作为
// 页面内容区的机器词面小标(dim),不得出现在导航段。
package e2e

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"tally/internal/store"
	"tally/internal/web/handlers"
)

// p205Server 按生产挂载方式装配全页面服务器(RegisterRoutes 末尾已组合
// RegisterV14Routes)。
func p205Server(t *testing.T, repoPath string, q *store.Queries) *httptest.Server {
	t.Helper()
	s := &handlers.Server{
		Content: &handlers.ContentStore{RepoPath: repoPath, Identity: p09Identity},
		Trace:   q,
	}
	r := chi.NewRouter()
	s.RegisterRoutes(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// p205Get 是跟随重定向的普通 GET(四页均为只读投影页)。
func p205Get(t *testing.T, srvURL, path string) string {
	t.Helper()
	resp, err := http.Get(srvURL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("读响应体: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s = %d, want 200", path, resp.StatusCode)
	}
	return string(body)
}

// p205NavSegment 截取导航轨道段(第一个 <nav …> 到其闭合标签)。
func p205NavSegment(body string) string {
	start := strings.Index(body, "<nav")
	if start < 0 {
		return ""
	}
	end := strings.Index(body[start:], "</nav>")
	if end < 0 {
		return ""
	}
	return body[start : start+end]
}

// p205TemplatesDir 用 runtime.Caller 定位仓库根的模板目录(与 go test 的
// 工作目录无关)。
func p205TemplatesDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位测试源文件路径")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "web", "templates")
}

func TestP205HumanCenteredUXGate(t *testing.T) {
	// 事实面:每测试独立 schema(不可达则跳过,与闭环 harness 同一纪律)。
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		base = clDSN
	}
	if !e2eDBReachable(base) {
		t.Skipf("P2-05 UX 门禁需要 PostgreSQL(%s 不可达),跳过", base)
	}
	t.Setenv("DATABASE_URL", base)
	st := store.TestDB(t)

	repo, _ := p09Repo(t)
	srv := p205Server(t, repo, st.Queries())

	// 每页的固定人读导航标签(研发组;P2-01 落地,基准导航不含机器词面)。
	navHuman := []string{"发起", "工作", "运行", "需人工"}
	// 生命周期机器词面不得出现在导航段(门禁目标;页面内容区的机器词面小标
	// 不在本判定内 —— 它们是事实的 dim 词面,不是导航)。
	navMachine := []string{"NEEDS_REVIEW", "VALIDATING", "AWAITING_APPROVAL", "CODEGEN", "CHANGESET"}
	// AC3:渲染面唯一允许的 <script> 是 base.tmpl 的静态 htmx 引用。
	const staticScript = `<script src="/static/htmx.min.js" defer></script>`
	// AC2:无空文本链接(键盘死角)。
	emptyAnchor := regexp.MustCompile(`<a\b[^>]*>\s*</a>`)

	for _, page := range []struct {
		path string
		want string // 页面人本主标题片段
	}{
		{"/v14/launch", "意图(一句自然语言,唯一必填项)"},
		{"/v13/tasks", "工作"},
		{"/v13/runs", "运行"},
		{"/v14/attention", "需人工评审"},
	} {
		body := p205Get(t, srv.URL, page.path)

		if nav := p205NavSegment(body); nav != "" {
			for _, label := range navHuman {
				if !strings.Contains(nav, label) {
					t.Errorf("%s:导航应包含人读标签 %q,导航段:\n%s", page.path, label, nav)
				}
			}
			for _, machine := range navMachine {
				if strings.Contains(nav, machine) {
					t.Errorf("%s:导航不得包含生命周期机器词面 %q(门禁目标),导航段:\n%s", page.path, machine, nav)
				}
			}
		} else {
			t.Errorf("%s:应渲染导航轨道", page.path)
		}

		if got := strings.Count(body, "<script"); got != 1 || !strings.Contains(body, staticScript) {
			t.Errorf("%s:渲染面应恰含静态 htmx 脚本引用、零内联脚本(AC3),实际 %d 个 <script>", page.path, got)
		}
		if emptyAnchor.MatchString(body) {
			t.Errorf("%s:存在空文本链接(AC2 键盘可访问性):\n%s", page.path, emptyAnchor.FindString(body))
		}
		if !strings.Contains(body, page.want) {
			t.Errorf("%s:应包含人本主标题片段 %q", page.path, page.want)
		}
	}

	// AC1:launch 页结构化优先 —— 唯一入口是有界意图表单(一句自然语言 +
	// 服务端澄清闭环),不存在 chat transcript 式自由多轮界面。
	launch := p205Get(t, srv.URL, "/v14/launch")
	for _, want := range []string{
		`<form method="post" action="/v14/launch/intent"`,
		`<label for="v14-intent">意图(一句自然语言,唯一必填项)</label>`,
		`id="v14-intent"`,
	} {
		if !strings.Contains(launch, want) {
			t.Errorf("launch 应包含结构化入口要素 %q", want)
		}
	}

	// AC2(模板静态面):P2 六个模板零内联脚本;任务表有 th 表头;意图控件
	// 经 label[for] 关联。
	dir := p205TemplatesDir(t)
	for _, name := range []string{"base.tmpl", "v14_launch.tmpl", "v13_tasks.tmpl", "v13_runs.tmpl", "v14_attention.tmpl"} {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("读模板 %s: %v", name, err)
		}
		src := string(raw)
		if name == "base.tmpl" {
			if strings.Count(src, "<script") != 1 || !strings.Contains(src, staticScript) {
				t.Errorf("base.tmpl 应恰含静态 htmx 引用(AC3)")
			}
			continue
		}
		if strings.Contains(src, "<script") {
			t.Errorf("%s 不得包含脚本(AC3:LLM 来源值只作数据渲染)", name)
		}
	}
	tasksTmpl, err := os.ReadFile(filepath.Join(dir, "v13_tasks.tmpl"))
	if err != nil {
		t.Fatalf("读 v13_tasks.tmpl: %v", err)
	}
	if !strings.Contains(string(tasksTmpl), "<th>") {
		t.Errorf("工作列表表格应有 th 表头(AC2)")
	}
}
