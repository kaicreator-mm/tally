package handlers_test

// P3-08 的 Launch 页测试(PRD §26.1 / §27)。
//
// 两层,与 v13_http_test.go 同一模式:
//   - 纯 HTTP 层(不需要数据库):空状态、意图校验、AI Runtime 未接入、
//     提案的草案/澄清/UNCERTAIN 流、草案确认边界、闭环启动入队(临时 Git
//     制品仓库 + 假队列);
//   - 全链路层(独立 schema 的真实 Postgres,不可达即 SKIP):列表行的
//     派生状态与可启动标记。
//
// v14 页面用 handlers.Server 直接挂路由(RegisterV14Routes;不经 internal/web
// 的登录/CSRF 组,那组行为由既有测试覆盖)。AI Runtime 推理出口经导出的
// SetV14ProposeExecutor 注入 airuntime.FakeExecutor,测试后还原。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"

	"tally/internal/airuntime"
	"tally/internal/jobs"
	"tally/internal/store"
	"tally/internal/web/handlers"
)

/* ---------- 装配 ---------- */

// v14FakeQueue 记录入队请求:Launch 的写合同只有「入队」,断言入的是什么。
type v14FakeQueue struct {
	kinds    []jobs.Kind
	payloads []string // payload 的 JSON 形态(线格式断言)
	err      error
}

func (q *v14FakeQueue) Enqueue(ctx context.Context, kind jobs.Kind, payload any) (int64, error) {
	if q.err != nil {
		return 0, q.err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	q.kinds = append(q.kinds, kind)
	q.payloads = append(q.payloads, string(b))
	return int64(len(q.kinds)), nil
}

// v14NewServer 直挂 handlers.Server 并只注册 v1.4 路由。q 为 nil 时渲染
// 「未接入」;非 nil 的 *store.Queries 即真实读取来源(闭环启动 POST 只做
// Git 校验 + 入队,不触库,可用零值 Queries 表达「已接入但本路径不读」)。
func v14NewServer(t *testing.T, q *store.Queries, repoPath string, queue handlers.JobQueue) *httptest.Server {
	t.Helper()
	s := &handlers.Server{
		Content: &handlers.ContentStore{
			RepoPath: repoPath,
			Identity: handlers.Identity{Name: "test", Email: "test@example.com"},
		},
		Trace: q,
		Queue: queue,
	}
	r := chi.NewRouter()
	s.RegisterV14Routes(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// v14PostForm 是 v13PostForm 的别名语义(不跟随重定向)。
func v14PostForm(t *testing.T, srvURL, path string, form url.Values) (*http.Response, string) {
	t.Helper()
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.PostForm(srvURL+path, form)
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

// v14SetExecutor 注入 FakeExecutor 并在测试结束后还原为未接入。
func v14SetExecutor(t *testing.T, x airuntime.Executor) {
	t.Helper()
	handlers.SetV14ProposeExecutor(x)
	t.Cleanup(func() { handlers.SetV14ProposeExecutor(nil) })
}

// v14DraftResult 构造一份通过 intake 严格校验的草案输出(完整字段组)。
func v14DraftResult(t *testing.T) airuntime.Result {
	t.Helper()
	raw := json.RawMessage(`{
	  "task_type": "bug_fix",
	  "summary": "修复登出后会话残留",
	  "objective": "登出后销毁全部服务端会话状态",
	  "scope": {"allowed_paths": ["src/auth/**"], "forbidden_paths": ["vendor/**"]},
	  "non_scope": ["不重构登录页"],
	  "constraints": ["保持 API 兼容"],
	  "acceptance": [{"id": "AC-01", "statement": "登出后旧会话 token 必须失效"}],
	  "risk_signals": ["auth_session"],
	  "clarifications": []
	}`)
	return airuntime.Result{
		OperationID: "op-draft-1",
		Status:      airuntime.StatusSucceeded,
		Output:      raw,
	}
}

func v14ClarifyResult(t *testing.T) airuntime.Result {
	t.Helper()
	raw := json.RawMessage(`{
	  "task_type": "",
	  "summary": "",
	  "objective": "",
	  "scope": {"allowed_paths": null, "forbidden_paths": null},
	  "non_scope": null,
	  "constraints": null,
	  "acceptance": null,
	  "risk_signals": null,
	  "clarifications": [
	    {"question": "复现步骤是什么?", "candidates": ["登录后立即登出", "等待 10 分钟后登出"]}
	  ]
	}`)
	return airuntime.Result{
		OperationID: "op-clarify-1",
		Status:      airuntime.StatusSucceeded,
		Output:      raw,
	}
}

/* ---------- 空状态与意图入口防线 ---------- */

func TestV14LaunchRendersUnavailableWithoutSources(t *testing.T) {
	srv := v14NewServer(t, nil, "", nil)
	status, body := v13Get(t, srv.URL, "/v14/launch")
	if status != http.StatusOK {
		t.Fatalf("GET /v14/launch = %d, want 200", status)
	}
	if !strings.Contains(body, "未接入") {
		t.Errorf("无数据源时应渲染未接入空状态")
	}
}

func TestV14ProposeRejectsEmptyIntent(t *testing.T) {
	srv := v14NewServer(t, nil, "", nil)
	resp, body := v14PostForm(t, srv.URL, "/v14/launch/intent", url.Values{"intent": {"   "}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("空意图应 400,得到 %d", resp.StatusCode)
	}
	if !strings.Contains(body, "意图文本为空") {
		t.Errorf("应说明意图文本为空是唯一拒绝原因,body:\n%s", body)
	}
}

func TestV14ProposeWithoutExecutorStatesUnwired(t *testing.T) {
	srv := v14NewServer(t, nil, "", nil)
	resp, body := v14PostForm(t, srv.URL, "/v14/launch/intent", url.Values{"intent": {"修复登出后会话残留"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("未接入时渲染状态页,得到 %d", resp.StatusCode)
	}
	if !strings.Contains(body, "AI Runtime 推理出口未接入") {
		t.Errorf("应如实说明 AI Runtime 未接入,不假装提案可用,body:\n%s", body)
	}
	if strings.Contains(body, `name="acstmt.0"`) {
		t.Errorf("未接入时不得产出草案(验收编辑表单不应出现)")
	}
}

/* ---------- 提案流:草案可确认/可编辑(AC2)+ 澄清(AC3)+ 拿不准如实呈现 ---------- */

func TestV14ProposeRendersEditableDraftAndConfirmBoundary(t *testing.T) {
	v14SetExecutor(t, &airuntime.FakeExecutor{Result: v14DraftResult(t)})
	srv := v14NewServer(t, nil, "", nil)

	resp, body := v14PostForm(t, srv.URL, "/v14/launch/intent",
		url.Values{"intent": {"修复登出后会话残留"}, "source_ref": {"GitHub Issue #142"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("提案应 200,得到 %d", resp.StatusCode)
	}
	for _, want := range []string{
		"合同草案(可编辑后确认冻结)",  // 编辑态标题
		"修复登出后会话残留",       // 摘要回填
		`name="acstmt.0"`, // 验收陈述可编辑(AC2)
		"src/auth/**",     // 边界回填
		"bug_fix",         // 任务类型展示
		`name="summary"`,  // 可编辑字段
		`name="allowed_paths"`,
		// P0-08:确认表单携带冻结输入(任务 id / 治理等级 / planned evidence)。
		`name="task_id"`, `name="profile"`, `name="ev_kind.0"`, `name="ev_path.0"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("草案页应包含 %q", want)
		}
	}
	if strings.Contains(body, "草案已确认") {
		t.Errorf("确认前不得显示已确认回执")
	}

	// 确认表单缺冻结必填输入(治理等级/证据声明):冻结准入类型化拒绝并退回
	// 表单 —— handler 绝不替 owner 发明治理属性或证据绑定。
	resp, body = v14PostForm(t, srv.URL, "/v14/launch/confirm", url.Values{
		"task_type":     {"bug_fix"},
		"summary":       {"修复登出后会话残留"},
		"objective":     {"登出后销毁全部服务端会话状态"},
		"allowed_paths": {"src/auth/**\ninternal/session/**"},
		"acid.0":        {"AC-01"},
		"acstmt.0":      {"登出后旧会话 token 必须失效"},
		"source_ref":    {"GitHub Issue #142"},
		"task_id":       {"TASK-0042"},
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("缺治理等级的确认应 400,得到 %d", resp.StatusCode)
	}
	if !strings.Contains(body, "治理等级") || !strings.Contains(body, "light/standard/strict") {
		t.Errorf("应类型化说明治理等级必须显式选择,body:\n%s", body)
	}
	// 编辑内容不丢:退回的表单保留已填字段。
	for _, want := range []string{"修复登出后会话残留", `value="TASK-0042"`, "登出后旧会话 token 必须失效"} {
		if !strings.Contains(body, want) {
			t.Errorf("准入失败退回表单应保留 %q", want)
		}
	}

	// 确认表单缺必填字段:类型化拒绝,不静默通过。
	resp, body = v14PostForm(t, srv.URL, "/v14/launch/confirm", url.Values{
		"task_type":     {"bug_fix"},
		"objective":     {"x"},
		"allowed_paths": {"src/**"},
		"acid.0":        {"AC-01"},
		"acstmt.0":      {"y"},
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("缺摘要的确认应 400,得到 %d", resp.StatusCode)
	}
	if !strings.Contains(body, "草案字段非法") {
		t.Errorf("应给出类型化的字段非法原因,body:\n%s", body)
	}
}

func TestV14ProposeClarificationLoopFoldsAnswersIntoNewIntent(t *testing.T) {
	calls := 0
	fake := &airuntime.FakeExecutor{
		Responder: func(ctx context.Context, req airuntime.Request) (airuntime.Result, error) {
			calls++
			if calls == 1 {
				return v14ClarifyResult(t), nil
			}
			return v14DraftResult(t), nil
		},
	}
	v14SetExecutor(t, fake)
	srv := v14NewServer(t, nil, "", nil)

	// 第一次:澄清请求(纯问题,无草案 —— 含糊时绝不发明验收)。
	_, body := v14PostForm(t, srv.URL, "/v14/launch/intent", url.Values{"intent": {"修一下会话问题"}})
	for _, want := range []string{"澄清", "复现步骤是什么?", `name="v14a.0"`, "登录后立即登出"} {
		if !strings.Contains(body, want) {
			t.Errorf("澄清页应包含 %q", want)
		}
	}
	if strings.Contains(body, `name="acstmt.0"`) {
		t.Errorf("澄清请求不得附带草案(验收编辑表单不应出现)")
	}

	// 第二次:结构化选项优先(PRD §27);答案折叠进新意图 → 新提案。
	_, body = v14PostForm(t, srv.URL, "/v14/launch/intent", url.Values{
		"intent": {"修一下会话问题"},
		"v14q.0": {"复现步骤是什么?"},
		"v14a.0": {"登录后立即登出"},
	})
	if !strings.Contains(body, "合同草案(可编辑后确认冻结)") {
		t.Errorf("回答澄清后应产出草案,body:\n%s", body)
	}
	if calls != 2 {
		t.Fatalf("应恰好发生两次提案调用,得到 %d", calls)
	}
	if len(fake.Requests()) != 2 {
		t.Fatalf("FakeExecutor 应记录两次请求,得到 %d", len(fake.Requests()))
	}
	prompt := string(fake.Requests()[1].Input.Prompt)
	if !strings.Contains(prompt, "Q: 复现步骤是什么?") || !strings.Contains(prompt, "A: 登录后立即登出") {
		t.Errorf("第二次提案的意图文本应携带澄清问答,prompt:\n%s", prompt)
	}
}

/* ---------- P0-04:有界 Dynamic Form 渲染(全部字段类型)+ 提交值服务端复核 ---------- */

// v14EightKindsClarifyResult 覆盖封闭词表前 8 种字段类型的澄清输出(含
// advisory 推荐与 number 区间)。有界上限是 8 问题/提案,free_text 与旧简单
// 形态由 v14FreeTextClarifyResult 单独一轮覆盖。
func v14EightKindsClarifyResult(t *testing.T) airuntime.Result {
	t.Helper()
	raw := json.RawMessage(`{
	  "task_type": "", "summary": "", "objective": "",
	  "scope": {"allowed_paths": null, "forbidden_paths": null},
	  "non_scope": null, "constraints": null, "acceptance": null, "risk_signals": null,
	  "clarifications": [
	    {"question": "选一个修复策略?", "kind": "choice", "candidates": ["热修复", "回滚"], "recommended": 1, "recommendation_reason": "回滚风险更低"},
	    {"question": "哪些模块受影响?", "kind": "multichoice", "candidates": ["auth", "session", "billing"], "recommended": 0, "recommendation_reason": "会话残留多在 auth"},
	    {"question": "是否需要数据迁移?", "kind": "boolean"},
	    {"question": "保留多少天日志?", "kind": "number", "min": 1, "max": 90},
	    {"question": "改动落在哪个仓库?", "kind": "repository", "candidates": ["tally-core", "tally-web"]},
	    {"question": "基于哪个分支修?", "kind": "branch", "candidates": ["main", "release/1.4"]},
	    {"question": "可用性还是一致性优先?", "kind": "risk_choice", "candidates": ["可用性", "一致性"], "recommended": 0},
	    {"question": "确认按上述边界执行?", "kind": "confirmation"}
	  ]
	}`)
	return airuntime.Result{OperationID: "op-clarify-8k", Status: airuntime.StatusSucceeded, Output: raw}
}

// v14FreeTextClarifyResult 覆盖 free_text(自然语言逃生舱)与未携带 kind 的
// 旧简单候选形态(校验层显式迁移为 choice)。
func v14FreeTextClarifyResult(t *testing.T) airuntime.Result {
	t.Helper()
	raw := json.RawMessage(`{
	  "task_type": "", "summary": "", "objective": "",
	  "scope": {"allowed_paths": null, "forbidden_paths": null},
	  "non_scope": null, "constraints": null, "acceptance": null, "risk_signals": null,
	  "clarifications": [
	    {"question": "还有什么背景需要补充?", "kind": "free_text"},
	    {"question": "旧形态候选问题?", "candidates": ["甲", "乙"]}
	  ]
	}`)
	return airuntime.Result{OperationID: "op-clarify-ft", Status: airuntime.StatusSucceeded, Output: raw}
}

func TestV14ClarificationFormsRenderAllFieldKinds(t *testing.T) {
	srv := v14NewServer(t, nil, "", nil)

	// 第一轮:choice/multichoice/boolean/number/repository/branch/risk_choice/confirmation。
	v14SetExecutor(t, &airuntime.FakeExecutor{Result: v14EightKindsClarifyResult(t)})
	resp, body := v14PostForm(t, srv.URL, "/v14/launch/intent", url.Values{"intent": {"修一下会话问题"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("提案应 200,得到 %d", resp.StatusCode)
	}
	for _, want := range []string{
		// 每种类型渲染为对应原生控件:单选(radio)/多选(checkbox)/是否/
		// 数值(number + 区间)/确认。
		`type="radio" name="v14a.0" value="热修复"`,
		`type="checkbox" name="v14a.1" value="auth"`,
		`type="radio" name="v14a.2" value="是"`,
		`type="radio" name="v14a.2" value="否"`,
		`type="number" name="v14f.3"`, `min="1"`, `max="90"`,
		`type="radio" name="v14a.4" value="tally-core"`,
		`type="radio" name="v14a.5" value="main"`,
		`type="radio" name="v14a.6" value="一致性"`,
		`type="radio" name="v14a.7" value="确认"`,
		`type="radio" name="v14a.7" value="否决"`,
		// schema 随表单携带(供服务端复核),类型有人读标注。
		`name="v14k.0" value="choice"`, `name="v14k.1" value="multichoice"`,
		`name="v14k.3" value="number"`, `name="v14k.6" value="risk_choice"`,
		`name="v14min.3" value="1"`, `name="v14max.3" value="90"`,
		`[单选]`, `[多选]`, `[数值]`, `[确认]`,
		// advisory 推荐只标注不预选:原因可见,没有 checked。
		"回滚风险更低", "(推荐)", "仅供参考,建议不是决定",
		`name="v14r.0" value="1"`,
		// 可访问性:fieldset/legend 分组 + label 关联;自由文本逃生舱可选。
		"<fieldset", "<legend", `for="v14f-3"`, "其他(填写后优先于选项)",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("澄清表单应包含 %q", want)
		}
	}
	if strings.Contains(body, "checked") {
		t.Errorf("推荐是 advisory,不得预选任何选项(advisory 不是决定)")
	}
	if strings.Contains(body, "hx-post") || strings.Contains(body, "hx-get") {
		t.Errorf("动态表单不得依赖 htmx 属性(关 JS 必须可用)")
	}

	// 第二轮:free_text(自然语言逃生舱)+ 旧简单候选形态(迁移为 choice)。
	v14SetExecutor(t, &airuntime.FakeExecutor{Result: v14FreeTextClarifyResult(t)})
	resp, body = v14PostForm(t, srv.URL, "/v14/launch/intent", url.Values{"intent": {"再补充一点"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("第二次提案应 200,得到 %d", resp.StatusCode)
	}
	for _, want := range []string{
		`[自由文本]`, `type="text" name="v14f.0"`,
		`[单选]`, `type="radio" name="v14a.1" value="甲"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("澄清表单应包含 %q", want)
		}
	}
}

func TestV14ClarificationRenderedValuesStayEscapedData(t *testing.T) {
	// 模型给出的问题/候选/推荐理由含 HTML/脚本形态文本:必须按数据转义呈现,
	// 不存在任何执行面(DL-11 / 任务 AC:No raw generated HTML is executed)。
	raw := json.RawMessage(`{
	  "task_type": "", "summary": "", "objective": "",
	  "scope": {"allowed_paths": null, "forbidden_paths": null},
	  "non_scope": null, "constraints": null, "acceptance": null, "risk_signals": null,
	  "clarifications": [
	    {"question": "<script>alert('q')</script> 选哪个?", "kind": "choice",
	     "candidates": ["<img src=x onerror=alert(1)>", "正常"], "recommended": 0,
	     "recommendation_reason": "</fieldset><script>alert('r')</script>"}
	  ]
	}`)
	v14SetExecutor(t, &airuntime.FakeExecutor{Result: airuntime.Result{
		OperationID: "op-clarify-xss", Status: airuntime.StatusSucceeded, Output: raw,
	}})
	srv := v14NewServer(t, nil, "", nil)
	resp, body := v14PostForm(t, srv.URL, "/v14/launch/intent", url.Values{"intent": {"修一下"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("提案应 200,得到 %d", resp.StatusCode)
	}
	for _, bad := range []string{"<script>alert", "<img src=x", "</fieldset><script>"} {
		if strings.Contains(body, bad) {
			t.Errorf("生成文本不得以可执行形态出现:%q", bad)
		}
	}
	for _, want := range []string{"&lt;script&gt;", "&lt;img src=x onerror=alert(1)&gt;"} {
		if !strings.Contains(body, want) {
			t.Errorf("注入形态值应按转义数据呈现,缺少 %q", want)
		}
	}
}

func TestV14ClarificationFoldRevalidatesAndFoldsPerKind(t *testing.T) {
	// 第二轮提交:答案 + 渲染时的 schema 隐藏域(模拟真实表单);服务端按
	// kind 逐一复核,复核通过的问答折叠进新意图,第二次提案返回草案。
	fake := &airuntime.FakeExecutor{Result: v14DraftResult(t)}
	v14SetExecutor(t, fake)
	srv := v14NewServer(t, nil, "", nil)

	_, body := v14PostForm(t, srv.URL, "/v14/launch/intent", url.Values{
		"intent": {"修一下会话问题"},
		"v14q.0": {"选一个修复策略?"}, "v14k.0": {"choice"},
		"v14c.0": {"热修复", "回滚"}, "v14a.0": {"回滚"},
		"v14q.1": {"哪些模块受影响?"}, "v14k.1": {"multichoice"},
		"v14c.1": {"auth", "session", "billing"}, "v14a.1": {"auth", "billing"},
		"v14q.2": {"是否需要数据迁移?"}, "v14k.2": {"boolean"}, "v14a.2": {"是"},
		"v14q.3": {"保留多少天日志?"}, "v14k.3": {"number"},
		"v14min.3": {"1"}, "v14max.3": {"90"}, "v14f.3": {"30"},
		"v14q.4": {"改动落在哪个仓库?"}, "v14k.4": {"repository"},
		"v14c.4": {"tally-core", "tally-web"}, "v14f.4": {"跨仓库 tally-cli"},
		"v14q.5": {"基于哪个分支修?"}, "v14k.5": {"branch"},
		"v14c.5": {"main", "release/1.4"}, "v14a.5": {"main"},
		"v14q.6": {"可用性还是一致性优先?"}, "v14k.6": {"risk_choice"},
		"v14c.6": {"可用性", "一致性"}, "v14a.6": {"一致性"},
		"v14q.7": {"确认按上述边界执行?"}, "v14k.7": {"confirmation"}, "v14a.7": {"确认"},
		"v14q.8": {"还有什么背景需要补充?"}, "v14k.8": {"free_text"},
		"v14f.8": {"补充背景:会话表按天分区"},
	})
	if !strings.Contains(body, "合同草案(可编辑后确认冻结)") {
		t.Fatalf("复核通过的澄清答案应折叠进新意图并产出草案,body:\n%s", body)
	}
	if len(fake.Requests()) != 1 {
		t.Fatalf("应恰好发生一次提案调用,得到 %d", len(fake.Requests()))
	}
	prompt := string(fake.Requests()[0].Input.Prompt)
	for _, want := range []string{
		"Q: 选一个修复策略?\nA: 回滚",
		"A: auth、billing",
		"Q: 是否需要数据迁移?\nA: 是",
		"Q: 保留多少天日志?\nA: 30",
		"Q: 改动落在哪个仓库?\nA: 跨仓库 tally-cli", // 自由文本逃生舱优先于选项
		"A: main",
		"A: 一致性",
		"A: 确认",
		"A: 补充背景:会话表按天分区",
	} {
		if !strings.Contains(prompt, want) {
			t.Errorf("提案的意图文本应携带复核后的问答 %q,prompt:\n%s", want, prompt)
		}
	}
}

func TestV14ClarificationFoldRejectsInvalidAnswers(t *testing.T) {
	srv := v14NewServer(t, nil, "", nil)

	cases := []struct {
		name string
		form url.Values
		want string // 类型化拒绝原因的一部分
	}{
		{
			name: "数值超出声明区间",
			form: url.Values{"v14q.3": {"保留多少天日志?"}, "v14k.3": {"number"},
				"v14min.3": {"1"}, "v14max.3": {"90"}, "v14f.3": {"900"}},
			want: "高于声明上界",
		},
		{
			name: "数值不是合法数字",
			form: url.Values{"v14q.3": {"保留多少天日志?"}, "v14k.3": {"number"},
				"v14min.3": {"1"}, "v14max.3": {"90"}, "v14f.3": {"abc"}},
			want: "不是合法数字",
		},
		{
			name: "词表外字段类型 fail-safe 拒绝",
			form: url.Values{"v14q.0": {"选一个修复策略?"}, "v14k.0": {"html"},
				"v14c.0": {"热修复"}, "v14a.0": {"热修复"}},
			want: "不在有界词表内",
		},
		{
			name: "单选答案不在候选集",
			form: url.Values{"v14q.0": {"选一个修复策略?"}, "v14k.0": {"choice"},
				"v14c.0": {"热修复", "回滚"}, "v14a.0": {"降级"}},
			want: "不在候选中",
		},
		{
			name: "多选答案不在候选集",
			form: url.Values{"v14q.1": {"哪些模块受影响?"}, "v14k.1": {"multichoice"},
				"v14c.1": {"auth"}, "v14a.1": {"billing"}},
			want: "不在候选中",
		},
		{
			name: "是否答案取非规范值",
			form: url.Values{"v14q.2": {"是否需要数据迁移?"}, "v14k.2": {"boolean"},
				"v14a.2": {"maybe"}},
			want: "「是」或「否」",
		},
		{
			name: "携带的声明区间倒挂",
			form: url.Values{"v14q.3": {"保留多少天日志?"}, "v14k.3": {"number"},
				"v14min.3": {"90"}, "v14max.3": {"1"}, "v14f.3": {"30"}},
			want: "数值区间非法",
		},
		{
			name: "问题未回答则要求重答",
			form: url.Values{"v14q.0": {"选一个修复策略?"}, "v14k.0": {"choice"},
				"v14c.0": {"热修复", "回滚"}},
			want: "还有澄清问题未回答",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, body := v14PostForm(t, srv.URL, "/v14/launch/intent", tc.form)
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("非法提交应 400,得到 %d", resp.StatusCode)
			}
			if !strings.Contains(body, tc.want) {
				t.Errorf("应给出类型化原因 %q,body:\n%s", tc.want, body)
			}
		})
	}

	// 复核失败重渲染时,问题与 schema 保留(可重答),且绝不产生草案。
	resp, body := v14PostForm(t, srv.URL, "/v14/launch/intent", url.Values{
		"v14q.3": {"保留多少天日志?"}, "v14k.3": {"number"},
		"v14min.3": {"1"}, "v14max.3": {"90"}, "v14f.3": {"900"},
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("越界数值应 400,得到 %d", resp.StatusCode)
	}
	for _, want := range []string{"保留多少天日志?", `name="v14k.3" value="number"`, `max="90"`} {
		if !strings.Contains(body, want) {
			t.Errorf("复核失败应保留问题与 schema 供重答,缺少 %q", want)
		}
	}
	if strings.Contains(body, `name="acstmt.0"`) {
		t.Errorf("复核失败不得产出草案")
	}
}

func TestV14ProposeUncertainAndCancelledAreNotFabricated(t *testing.T) {
	// UNCERTAIN:Runtime 拿不准 → 结果不采信,如实呈现,不折叠成失败。
	v14SetExecutor(t, &airuntime.FakeExecutor{Result: airuntime.Result{
		OperationID: "op-u", Status: airuntime.StatusUncertain,
		TerminationReason: airuntime.TerminationPolicyRejected,
	}})
	srv := v14NewServer(t, nil, "", nil)
	resp, body := v14PostForm(t, srv.URL, "/v14/launch/intent", url.Values{"intent": {"修一下"}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("UNCERTAIN 应 400,得到 %d", resp.StatusCode)
	}
	if !strings.Contains(body, "拿不准") || !strings.Contains(body, "未采信") {
		t.Errorf("UNCERTAIN 应如实呈现为拿不准且未采信,body:\n%s", body)
	}

	// CANCELLED:取消是一等终态,绝不编造成提案失败。
	v14SetExecutor(t, &airuntime.FakeExecutor{Result: airuntime.Result{
		OperationID: "op-c", Status: airuntime.StatusCancelled,
		TerminationReason: airuntime.TerminationClientCancelled,
	}})
	resp, body = v14PostForm(t, srv.URL, "/v14/launch/intent", url.Values{"intent": {"修一下"}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("CANCELLED 应 400,得到 %d", resp.StatusCode)
	}
	if !strings.Contains(body, "取消") || strings.Contains(body, "判定失败") {
		t.Errorf("取消不得被表述成失败,body:\n%s", body)
	}
}

/* ---------- 闭环启动:入队线格式与防线(AC1/AC3 的执行入口) ---------- */

func TestV14LaunchRunEnqueuesV14TaskJob(t *testing.T) {
	repo, _, _, _ := v13Repo(t)
	queue := &v14FakeQueue{}
	srv := v14NewServer(t, &store.Queries{}, repo, queue)

	resp, _ := v14PostForm(t, srv.URL, "/v14/launch/run", url.Values{"task": {"TASK-0001"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("启动闭环应 303(PRG),得到 %d", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); !strings.Contains(loc, "/v14/launch") || !strings.Contains(loc, "kind=ok") {
		t.Fatalf("应 303 回 Launch 页并带成功提示,得到 %s", loc)
	}
	if len(queue.kinds) != 1 || queue.kinds[0] != "v14_task" {
		t.Fatalf("应入队一条 v14_task job,得到 %v", queue.kinds)
	}
	if queue.payloads[0] != `{"task":"TASK-0001"}` {
		t.Fatalf("payload 应与 worker 的 V14TaskPayload 线格式一致,得到 %s", queue.payloads[0])
	}
}

func TestV14LaunchRunGuards(t *testing.T) {
	repo, _, _, _ := v13Repo(t)

	// HEAD 快照里没有这个任务:拒绝入队。
	queue := &v14FakeQueue{}
	srv := v14NewServer(t, &store.Queries{}, repo, queue)
	resp, _ := v14PostForm(t, srv.URL, "/v14/launch/run", url.Values{"task": {"TASK-9999"}})
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(resp.Header.Get("Location"), "kind=bad") {
		t.Fatalf("未知任务应 303 + kind=bad,得到 %d → %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	if len(queue.kinds) != 0 {
		t.Fatalf("未知任务不得入队,得到 %v", queue.kinds)
	}

	// 队列未接入:明确失败提示,不 panic。
	srv2 := v14NewServer(t, &store.Queries{}, repo, nil)
	resp, _ = v14PostForm(t, srv2.URL, "/v14/launch/run", url.Values{"task": {"TASK-0001"}})
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(resp.Header.Get("Location"), "kind=bad") {
		t.Fatalf("队列未接入应 303 + kind=bad,得到 %d → %s", resp.StatusCode, resp.Header.Get("Location"))
	}
}

// v14RepoWithoutFrozen 把制品仓库改成 HEAD 无冻结验证契约(用于「未冻结不可
// 启动」防线):提交一次删除 .tally/verifications/TASK-0001.yaml。
func v14RepoWithoutFrozen(t *testing.T) string {
	t.Helper()
	repo, _, _, _ := v13Repo(t)
	r, err := git.PlainOpen(repo)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	wt, err := r.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}
	if _, err := wt.Remove(".tally/verifications/TASK-0001.yaml"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	when := time.Now().Add(-time.Hour)
	if _, err := wt.Commit("chore: 移除冻结验证契约", &git.CommitOptions{
		Author:    &object.Signature{Name: "test", Email: "test@example.com", When: when},
		Committer: &object.Signature{Name: "test", Email: "test@example.com", When: when},
	}); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	return repo
}

func TestV14LaunchRunBlocksWithoutFrozenVerification(t *testing.T) {
	repo := v14RepoWithoutFrozen(t)
	queue := &v14FakeQueue{}
	srv := v14NewServer(t, &store.Queries{}, repo, queue)
	resp, _ := v14PostForm(t, srv.URL, "/v14/launch/run", url.Values{"task": {"TASK-0001"}})
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(resp.Header.Get("Location"), "kind=bad") {
		t.Fatalf("无冻结验证契约应 303 + kind=bad,得到 %d → %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	if !strings.Contains(resp.Header.Get("Location"), "INV-02") {
		t.Errorf("拒绝原因应指向验收先冻结的不变量,得到 %s", resp.Header.Get("Location"))
	}
	if len(queue.kinds) != 0 {
		t.Fatalf("不得入队,得到 %v", queue.kinds)
	}
}

/* ---------- 全链路:列表行派生状态(需要 Postgres,不可达即 SKIP) ---------- */

func TestV14LaunchListsDerivedRowsWithLaunchAction(t *testing.T) {
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	v13Seed(t, context.Background(), st.Queries(), repo, base, docs, code)

	queue := &v14FakeQueue{}
	srv := v14NewServer(t, st.Queries(), repo, queue)
	status, body := v13Get(t, srv.URL, "/v14/launch")
	if status != http.StatusOK {
		t.Fatalf("GET /v14/launch = %d, want 200", status)
	}
	for _, want := range []string{
		"TASK-0001",
		"演示任务",
		"启动闭环",       // 可启动动作(standard + 已冻结)
		"VALIDATING", // 派生状态来自事实(v13Seed 的候选 + 验证 Run)
		"状态列由 taskstate 引擎从证据派生", // 无手动状态控件
	} {
		if !strings.Contains(body, want) {
			t.Errorf("Launch 列表应包含 %q", want)
		}
	}
}
