//go:build integration

// P0-09 — Intent-to-Freeze E2E gate(任务合同:在真实临时 Git 仓库上证明
// 「人类意图 → 动态澄清 → 不可变冻结」的完整权威链,解除 P0 Confirm→Freeze
// hard blocker 的运行证据)。
//
// 被测纵切 = 生产路由原样装配:handlers.Server + RegisterV14Routes(与
// cmd/server 组合根挂载的同一注册函数)+ handlers.EnsureRepo(生产内容仓库
// 引导)+ SetV14ProposeExecutor(生产 AI Runtime 注入缝)。全链经 HTTP:
//
//	POST /v14/launch/intent  提案(澄清请求或草案)
//	POST /v14/launch/intent  澄清答案按有界 schema 复核折叠 → 新提案 → 草案
//	POST /v14/launch/confirm 确认 → intakefreeze.Freeze 准入 →
//	                         artifacts.FreezeCommit 单一期望基线原子提交(gitwriter)
//	artifacts.Load / LoadIntent 从 pinned 提交确定性读回(P0-02/P0-08 生产装载器)
//
// AI Runtime 环节用确定性测试替身 airuntime.FakeExecutor 挂在生产注入缝上:
// gate 必须可复现,真实端点连通性由 P0-05 的 DEEPSEEK_API_KEY 门控连接测试
// 单独覆盖(非确定性输入不进 gate 判定)。
//
// 覆盖面(任务合同 AC + Reference Pack 负面用例):
//
//	AC1 bug_fix 场景:澄清 → 草案 → 确认 → 冻结 → pinned 读回一致(含 DL-06:
//	   仅 owner 勾选保留的风险提示进入冻结证据,候选绝不自动晋升)
//	AC1 db_migration 场景:同链冻结;冻结提交恰含三件套三个新文件(原子性)
//	AC2 缺失 planned evidence 阻断冻结(400,零提交);半声明、不可承载语义
//	   (non_scope)同样类型化阻断
//	AC3 全部制品从 pinned 提交装载(Load + LoadIntent,身份三元组一致;
//	   冻结前基线提交里 intent 不存在 —— 制品确实来自冻结提交)
//	负面:非法澄清答案被服务端复核阻断(无草案、无第二次 AI 调用);
//	   重复确认内容幂等(ErrNoChanges,零新提交);陈旧基线类型化冲突
//	   (ErrBaseRevision,零部分写入,以当前基线重试成功)
//
// 陈旧基线用例说明:V14ConfirmDraft 在请求内先读实时 HEAD 再提交,HTTP 层
// 无法无损注入并发推进窗口(P0-08 报告同一分区);该负面在 handler 委托的
// **同一生产事务** artifacts.FreezeCommit 上以显式陈旧 ExpectedBase 驱动,
// 冻结产物经同一准入入口 intakefreeze.Freeze 构造,仓库为同一真实 e2e 仓库。
package e2e

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"

	"tally/internal/airuntime"
	"tally/internal/artifacts"
	"tally/internal/gitwriter"
	"tally/internal/intake"
	"tally/internal/intakefreeze"
	"tally/internal/intentevidence"
	"tally/internal/store"
	"tally/internal/taskcontract"
	"tally/internal/verification"
	"tally/internal/web/handlers"
)

// ---- 装配:生产路由 + 真实临时内容仓库 -------------------------------------

// p09Identity 是 e2e 内容仓库与冻结提交的固定身份。
var p09Identity = handlers.Identity{Name: "tally-p009-e2e", Email: "tally-p009-e2e@example.invalid"}

// p09Repo 按 cmd/server 的生产引导建真实临时内容仓库(EnsureRepo init + 首提
// 交),再经 gitwriter(唯一写边界,DL-03)提交项目注册表,使 artifacts.Load
// 可从冻结提交整树装载。返回仓库路径与冻结前基线 sha。
func p09Repo(t *testing.T) (repo, base string) {
	t.Helper()
	dir := t.TempDir()
	if err := handlers.EnsureRepo(dir, p09Identity); err != nil {
		t.Fatalf("EnsureRepo: %v", err)
	}
	w, err := gitwriter.New(dir)
	if err != nil {
		t.Fatalf("gitwriter.New: %v", err)
	}
	head, err := w.Head(context.Background())
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	res, err := w.Commit(context.Background(), gitwriter.CommitRequest{
		ExpectedBase: head,
		Message:      "p0-09 e2e: 项目注册表",
		Files: []gitwriter.FileUpdate{
			{Path: ".tally/project.yaml", Content: []byte("project: e2e-p009-gate\n")},
		},
		Signature: &gitwriter.Signature{Name: p09Identity.Name, Email: p09Identity.Email},
	})
	if err != nil {
		t.Fatalf("提交项目注册表: %v", err)
	}
	return dir, res.Commit
}

// p09Server 按生产挂载方式装一台北端服务器:handlers.Server + RegisterV14Routes
// (与组合根同一注册函数;登录/CSRF 组的行为由既有 web 层测试覆盖)。Trace 用
// 零值 *store.Queries 表达「事实面已接入且本链路不读库」(Intent→Freeze 的
// 权威面只在 Git)。
func p09Server(t *testing.T, repoPath string) *httptest.Server {
	t.Helper()
	s := &handlers.Server{
		Content: &handlers.ContentStore{RepoPath: repoPath, Identity: p09Identity},
		Trace:   &store.Queries{},
	}
	r := chi.NewRouter()
	s.RegisterV14Routes(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// p09SetExecutor 把确定性 AI Runtime 替身挂上生产注入缝,测试后还原为未接入。
func p09SetExecutor(t *testing.T, x airuntime.Executor) {
	t.Helper()
	handlers.SetV14ProposeExecutor(x)
	t.Cleanup(func() { handlers.SetV14ProposeExecutor(nil) })
}

// p09Post 以不跟随重定向的客户端 POST 表单(生产 PRG 契约的观察点)。
func p09Post(t *testing.T, srvURL, path string, form url.Values) (*http.Response, string) {
	t.Helper()
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
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

// p09Flash 解码 303 Location 里的 kind/flash(生产 PRG flash 契约)。
func p09Flash(t *testing.T, resp *http.Response) (string, string) {
	t.Helper()
	u, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		t.Fatalf("解析 Location %q: %v", resp.Header.Get("Location"), err)
	}
	q := u.Query()
	return q.Get("kind"), q.Get("flash")
}

// p09Head 返回制品仓库当前 HEAD(完整 sha)。
func p09Head(t *testing.T, repo string) string {
	t.Helper()
	w, err := gitwriter.New(repo)
	if err != nil {
		t.Fatalf("gitwriter.New: %v", err)
	}
	h, err := w.Head(context.Background())
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	return h
}

// ---- 渲染页表单回收(e2e 保真:确认提交的是生产页面渲染出的内容)-----------

func p09InputVal(body, name string) (string, bool) {
	m := regexp.MustCompile(`name="` + regexp.QuoteMeta(name) + `" value="([^"]*)"`).FindStringSubmatch(body)
	if m == nil {
		return "", false
	}
	return m[1], true
}

func p09TextareaVal(body, name string) (string, bool) {
	m := regexp.MustCompile(`(?s)<textarea name="` + regexp.QuoteMeta(name) + `"[^>]*>(.*?)</textarea>`).FindStringSubmatch(body)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// p09Field 从渲染页回收一个表单字段值(input value= 或 textarea 正文)。
func p09Field(t *testing.T, body, name string) string {
	t.Helper()
	if v, ok := p09InputVal(body, name); ok {
		return v
	}
	if v, ok := p09TextareaVal(body, name); ok {
		return v
	}
	t.Fatalf("渲染页缺少字段 %q", name)
	return ""
}

// p09HiddenAll 回收渲染页里同名 hidden 域的全部取值(候选风险信号回传)。
func p09HiddenAll(body, name string) []string {
	ms := regexp.MustCompile(`name="`+regexp.QuoteMeta(name)+`" value="([^"]*)"`).FindAllStringSubmatch(body, -1)
	out := make([]string, 0, len(ms))
	for _, m := range ms {
		out = append(out, m[1])
	}
	return out
}

// ---- Git 树与冻结产物读回断言 ----------------------------------------------

// p09TreeFiles 列出 rev 提交树里的全部文件路径(Git 对象库直读,工作区不可见)。
func p09TreeFiles(t *testing.T, repo, rev string) map[string]bool {
	t.Helper()
	r, err := git.PlainOpen(repo)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	h, err := r.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		t.Fatalf("ResolveRevision %q: %v", rev, err)
	}
	commit, err := r.CommitObject(*h)
	if err != nil {
		t.Fatalf("CommitObject: %v", err)
	}
	tree, err := commit.Tree()
	if err != nil {
		t.Fatalf("Tree: %v", err)
	}
	files := map[string]bool{}
	if err := tree.Files().ForEach(func(f *object.File) error { files[f.Name] = true; return nil }); err != nil {
		t.Fatalf("遍历提交树: %v", err)
	}
	return files
}

// p09AssertSHA256 断言值是 sha256:<64hex> 形式的内容修订标识。
func p09AssertSHA256(t *testing.T, what, v string) {
	t.Helper()
	if !regexp.MustCompile(`^sha256:[0-9a-f]{64}$`).MatchString(v) {
		t.Fatalf("%s = %q,期望 sha256:<64hex> 形式的内容修订标识", what, v)
	}
}

// ---- AI Runtime 确定性替身的提案输出(经生产 intake.Propose 校验的形状)----

// p09ClarifyResult 是第一轮提案:纯澄清请求(risk_choice,含 advisory 推荐)。
// 含糊时绝不发明验收 —— 澄清轮不得附带草案。
func p09ClarifyResult(t *testing.T) airuntime.Result {
	t.Helper()
	return airuntime.Result{
		OperationID: "p009-clarify-1",
		Status:      airuntime.StatusSucceeded,
		Output: []byte(`{
		  "task_type": "", "summary": "", "objective": "",
		  "scope": {"allowed_paths": null, "forbidden_paths": null},
		  "non_scope": null, "constraints": null, "acceptance": null, "risk_signals": null,
		  "clarifications": [
		    {"question": "采用哪种修复策略?", "kind": "risk_choice",
		     "candidates": ["热修复", "回滚再修"], "recommended": 1,
		     "recommendation_reason": "回滚风险更低"}
		  ]
		}`),
	}
}

// p09BugFixDraftResult 是第二轮提案:完整 bug_fix 草案(两个候选风险信号;
// 无 non_scope/constraints —— 本草案无需 owner 先解析不可承载语义)。
func p09BugFixDraftResult(t *testing.T) airuntime.Result {
	t.Helper()
	return airuntime.Result{
		OperationID: "p009-draft-bugfix",
		Status:      airuntime.StatusSucceeded,
		Output: []byte(`{
		  "task_type": "bug_fix",
		  "summary": "修复登出后会话残留",
		  "objective": "登出后销毁全部服务端会话状态",
		  "scope": {"allowed_paths": ["src/auth/**"], "forbidden_paths": ["vendor/**"]},
		  "non_scope": [], "constraints": [],
		  "acceptance": [{"id": "AC-01", "statement": "登出后旧会话 token 必须失效"}],
		  "risk_signals": ["auth_session", "token_cache"],
		  "clarifications": []
		}`),
	}
}

// p09MigrationDraftResult 是 db_migration 草案(一个候选风险信号,schema_check
// 证据面)。
func p09MigrationDraftResult(t *testing.T) airuntime.Result {
	t.Helper()
	return airuntime.Result{
		OperationID: "p009-draft-migration",
		Status:      airuntime.StatusSucceeded,
		Output: []byte(`{
		  "task_type": "db_migration",
		  "summary": "为 orders 表增加 status 索引",
		  "objective": "orders 按 status 过滤的查询走索引扫描",
		  "scope": {"allowed_paths": ["migrations/**"], "forbidden_paths": []},
		  "non_scope": [], "constraints": [],
		  "acceptance": [{"id": "AC-01", "statement": "迁移后 EXPLAIN 确认命中 idx_orders_status"}],
		  "risk_signals": ["long_lock"],
		  "clarifications": []
		}`),
	}
}

// ---- AC1/AC3:bug_fix 全链(澄清 → 草案 → 确认 → 冻结 → pinned 读回)-------

func TestV141IntentToFreezeGateBugFixEndToEnd(t *testing.T) {
	repo, base := p09Repo(t)
	srv := p09Server(t, repo)

	// AI Runtime 替身:第 1 次调用回澄清请求,第 2 次回草案。
	calls := 0
	fake := &airuntime.FakeExecutor{
		Responder: func(ctx context.Context, req airuntime.Request) (airuntime.Result, error) {
			calls++
			if calls == 1 {
				return p09ClarifyResult(t), nil
			}
			return p09BugFixDraftResult(t), nil
		},
	}
	p09SetExecutor(t, fake)

	// 1) 提案 → 澄清请求(纯问题,无草案)。
	resp, body := p09Post(t, srv.URL, "/v14/launch/intent", url.Values{
		"intent":     {"修复登出后会话残留"},
		"source_ref": {"GitHub Issue #901"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("提案应 200,得到 %d", resp.StatusCode)
	}
	for _, want := range []string{"澄清", "采用哪种修复策略?", `name="v14a.0"`, "回滚再修"} {
		if !strings.Contains(body, want) {
			t.Errorf("澄清页应包含 %q", want)
		}
	}
	if strings.Contains(body, `name="acstmt.0"`) {
		t.Errorf("澄清轮不得附带草案(含糊时绝不发明验收)")
	}

	// 2) 澄清答案按有界 schema 折叠 → 新提案 → 草案(可确认/可编辑)。
	resp, body = p09Post(t, srv.URL, "/v14/launch/intent", url.Values{
		"intent":     {"修复登出后会话残留"},
		"source_ref": {"GitHub Issue #901"},
		"v14q.0":     {"采用哪种修复策略?"},
		"v14k.0":     {"risk_choice"},
		"v14c.0":     {"热修复", "回滚再修"},
		"v14r.0":     {"1"},
		"v14rr.0":    {"回滚风险更低"},
		"v14a.0":     {"回滚再修"},
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("澄清折叠后的提案应 200,得到 %d", resp.StatusCode)
	}
	if !strings.Contains(body, "合同草案(可编辑后确认冻结)") {
		t.Fatalf("回答澄清后应产出草案")
	}
	// 链路连续性:第二次 AI 调用的意图文本必须携带折叠后的澄清问答。
	if n := len(fake.Requests()); n != 2 {
		t.Fatalf("应恰好两次提案调用,得到 %d", n)
	}
	prompt := string(fake.Requests()[1].Input.Prompt)
	if !strings.Contains(prompt, "Q: 采用哪种修复策略?") || !strings.Contains(prompt, "A: 回滚再修") {
		t.Errorf("第二次提案的意图文本应携带澄清问答,prompt:\n%s", prompt)
	}

	// 3) owner 确认:提交内容取自生产页面渲染值(AI 侧字段) + owner 冻结输入
	//    (任务 id / 治理等级 / 逐条款证据 / 勾选保留的风险提示)。
	if got := p09Field(t, body, "task_type"); got != "bug_fix" {
		t.Errorf("渲染的 task_type = %q", got)
	}
	if got := p09Field(t, body, "summary"); got != "修复登出后会话残留" {
		t.Errorf("渲染的 summary = %q", got)
	}
	if got := p09Field(t, body, "acid.0"); got != "AC-01" {
		t.Errorf("渲染的验收条款 id = %q", got)
	}
	form := url.Values{
		"task_type":       {p09Field(t, body, "task_type")},
		"source_ref":      {p09Field(t, body, "source_ref")},
		"summary":         {p09Field(t, body, "summary")},
		"objective":       {p09Field(t, body, "objective")},
		"allowed_paths":   {p09Field(t, body, "allowed_paths")},
		"forbidden_paths": {p09Field(t, body, "forbidden_paths")},
		"acid.0":          {p09Field(t, body, "acid.0")},
		"acstmt.0":        {p09Field(t, body, "acstmt.0")},
	}
	for _, r := range p09HiddenAll(body, "risk") {
		form.Add("risk", r) // 页面 hidden 回传候选信号(与浏览器行为一致)
	}
	form.Set("task_id", "TASK-0901")
	form.Set("profile", "standard")
	form.Set("ev_kind.0", "test")
	form.Set("ev_path.0", "src/auth/logout_test.go")
	form.Set("ev_selector.0", "TestLogoutInvalidatesSession")
	form.Set("risk_hint", "auth_session") // 仅勾选保留一条;token_cache 只是候选

	before := p09Head(t, repo)
	resp, _ = p09Post(t, srv.URL, "/v14/launch/confirm", form)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("确认冻结应 303(PRG),得到 %d", resp.StatusCode)
	}
	kind, flash := p09Flash(t, resp)
	if kind != "ok" || !strings.Contains(flash, "冻结完成") {
		t.Fatalf("确认应成功冻结,得到 kind=%s flash=%q", kind, flash)
	}

	// 4) 恰一个新提交;成功提示携带该提交 sha;它就是新的 pinned 基线。
	pinned := p09Head(t, repo)
	if pinned == "" || pinned == before {
		t.Fatalf("确认应产生新冻结提交,before=%s after=%s", before, pinned)
	}
	if !strings.Contains(flash, pinned[:12]) {
		t.Errorf("成功提示应携带提交 sha %s,得到 %q", pinned[:12], flash)
	}

	// 5) AC3:三件套全部从 pinned 提交(完整 sha,不是 HEAD 符号)确定性读回。
	snap, err := artifacts.Load(repo, pinned)
	if err != nil {
		t.Fatalf("Load(pinned): %v", err)
	}
	if snap.SourceCommit != pinned {
		t.Fatalf("快照来源提交 = %s,期望 pinned %s", snap.SourceCommit, pinned)
	}
	var task taskcontract.TaskContract
	found := false
	for i := range snap.Tasks {
		if snap.Tasks[i].Contract.ID == "TASK-0901" {
			found, task = true, snap.Tasks[i].Contract
			if snap.Tasks[i].Path != ".tally/tasks/TASK-0901.yaml" {
				t.Errorf("task 制品路径 = %s", snap.Tasks[i].Path)
			}
		}
	}
	if !found {
		t.Fatalf("pinned 快照里没有 TASK-0901 的 TaskContract")
	}
	if task.Profile != "standard" {
		t.Errorf("冻结 profile = %q,期望 owner 显式选择的 standard", task.Profile)
	}
	if task.Title != "修复登出后会话残留" || task.Objective != "登出后销毁全部服务端会话状态" {
		t.Errorf("冻结 title/objective 与确认内容不一致:%q / %q", task.Title, task.Objective)
	}
	if len(task.Scope.AllowedPaths) != 1 || task.Scope.AllowedPaths[0] != "src/auth/**" ||
		len(task.Scope.ForbiddenPaths) != 1 || task.Scope.ForbiddenPaths[0] != "vendor/**" {
		t.Errorf("冻结 scope 与确认内容不一致:%+v", task.Scope)
	}
	if len(task.Acceptance) != 1 || task.Acceptance[0].ID != "AC-01" ||
		task.Acceptance[0].Statement != "登出后旧会话 token 必须失效" {
		t.Errorf("冻结验收条款与确认内容不一致:%+v", task.Acceptance)
	}
	p09AssertSHA256(t, "TaskContract.Hash", task.Hash)
	if task.Hash != task.RevisionHash() {
		t.Errorf("TaskContract.Hash %s 应等于内容重算 %s(哈希来自重算,不来自声明)", task.Hash, task.RevisionHash())
	}

	var verif verification.Frozen
	found = false
	for i := range snap.Verifications {
		if snap.Verifications[i].Frozen.TaskID() == "TASK-0901" {
			found, verif = true, *snap.Verifications[i].Frozen
			if snap.Verifications[i].Path != ".tally/verifications/TASK-0901.yaml" {
				t.Errorf("verification 制品路径 = %s", snap.Verifications[i].Path)
			}
		}
	}
	if !found {
		t.Fatalf("pinned 快照里没有 TASK-0901 的冻结 VerificationContract")
	}
	if verif.Revision() != 1 {
		t.Errorf("冻结 verification revision = %d,期望首次作者化为 1", verif.Revision())
	}
	p09AssertSHA256(t, "VerificationContract.Hash", verif.Hash())
	vc := verif.Contract()
	if len(vc.Acceptances) != 1 || len(vc.Acceptances[0].EvidenceBindings) != 1 {
		t.Fatalf("冻结验证契约应与确认验收 1:1(1 条款 1 证据绑定),得到 %+v", vc.Acceptances)
	}
	b := vc.Acceptances[0].EvidenceBindings[0]
	if b.ID != "AC-01-EV01" || b.AuthoringMode != verification.AuthoringPreCommit ||
		b.VisibilityPolicy != verification.VisibilityVisible {
		t.Errorf("证据绑定身份/provenance/可见性不一致:%+v", b)
	}
	if b.Evidence.Type != "test" || b.Evidence.Path != "src/auth/logout_test.go" ||
		b.Evidence.Selector != "TestLogoutInvalidatesSession" {
		t.Errorf("证据定位与 owner 声明不一致:%+v", b.Evidence)
	}

	ia, err := artifacts.LoadIntent(repo, pinned, "TASK-0901")
	if err != nil {
		t.Fatalf("LoadIntent(pinned, TASK-0901): %v", err)
	}
	if ia.SourceCommit != pinned || ia.Path != ".tally/intents/TASK-0901.yaml" {
		t.Errorf("intent 制品来源/路径 = %s / %s", ia.SourceCommit, ia.Path)
	}
	if ia.Intent.TaskType() != intentevidence.TaskTypeBugFix {
		t.Errorf("冻结意图 task_type = %v,期望 bug_fix", ia.Intent.TaskType())
	}
	if ia.Intent.SourceRef() != "GitHub Issue #901" {
		t.Errorf("冻结意图 SourceRef = %q", ia.Intent.SourceRef())
	}
	// DL-06:进入冻结证据的只有 owner 勾选保留的提示;未勾选的候选绝不自动晋升。
	if hints := ia.Intent.RiskHints(); len(hints) != 1 || hints[0] != "auth_session" {
		t.Errorf("冻结意图 RiskHints = %v,期望恰为 owner 确认保留的 [auth_session]", hints)
	}
	p09AssertSHA256(t, "FrozenIntent.Revision", ia.Intent.Revision())

	// 身份三元组一致(DL-04 的写侧对应):三份制品共享同一任务身份。
	if ia.Intent.TaskID() != task.ID || vc.TaskID != task.ID {
		t.Errorf("冻结三件套身份不一致:intent=%s task=%s verification=%s",
			ia.Intent.TaskID(), task.ID, vc.TaskID)
	}

	// 确定性:同一 pinned 提交重复装载产出逐字段相等的结果。
	snap2, err := artifacts.Load(repo, pinned)
	if err != nil {
		t.Fatalf("Load(pinned) 第二次: %v", err)
	}
	ia2, err := artifacts.LoadIntent(repo, pinned, "TASK-0901")
	if err != nil {
		t.Fatalf("LoadIntent(pinned) 第二次: %v", err)
	}
	if snap2.Tasks[0].Contract.Hash != task.Hash ||
		snap2.Verifications[0].Frozen.Hash() != verif.Hash() ||
		!ia2.Intent.Equal(ia.Intent) {
		t.Errorf("同一 pinned 提交的重复装载结果不一致(确定性破坏)")
	}

	// 制品确实来自冻结提交:冻结前基线提交里 intent 不存在。
	if _, err := artifacts.LoadIntent(repo, base, "TASK-0901"); !errors.Is(err, artifacts.ErrIntentNotFound) {
		t.Errorf("冻结前基线装载 intent 应 ErrIntentNotFound,得到 %v", err)
	}

	// gate 证据:记录提交/制品哈希(执行报告引用)。
	t.Logf("GATE bug_fix: freeze_commit=%s task_hash=%s verification_hash=%s intent_revision=%s",
		pinned, task.Hash, verif.Hash(), ia.Intent.Revision())
}

// ---- AC1:db_migration 场景 + 冻结提交原子性(恰三件套三个新文件)-----------

func TestV141IntentToFreezeGateMigrationScenarioFreezesAtomically(t *testing.T) {
	repo, base := p09Repo(t)
	srv := p09Server(t, repo)
	p09SetExecutor(t, &airuntime.FakeExecutor{Result: p09MigrationDraftResult(t)})

	// 提案直接产出草案(无澄清轮)。
	resp, body := p09Post(t, srv.URL, "/v14/launch/intent", url.Values{
		"intent":     {"为 orders 表增加 status 索引"},
		"source_ref": {"GitHub Issue #902"},
	})
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "合同草案(可编辑后确认冻结)") {
		t.Fatalf("迁移意图应直接产出草案,得到 %d", resp.StatusCode)
	}

	// owner 确认(页面渲染值 + 冻结输入;勾选保留 long_lock 风险提示)。
	form := url.Values{
		"task_type":     {p09Field(t, body, "task_type")},
		"source_ref":    {p09Field(t, body, "source_ref")},
		"summary":       {p09Field(t, body, "summary")},
		"objective":     {p09Field(t, body, "objective")},
		"allowed_paths": {p09Field(t, body, "allowed_paths")},
		"acid.0":        {p09Field(t, body, "acid.0")},
		"acstmt.0":      {p09Field(t, body, "acstmt.0")},
	}
	for _, r := range p09HiddenAll(body, "risk") {
		form.Add("risk", r)
	}
	form.Set("task_id", "TASK-0902")
	form.Set("profile", "standard")
	form.Set("ev_kind.0", "schema_check")
	form.Set("ev_path.0", "migrations/0009_orders_status_index.yaml")
	form.Set("risk_hint", "long_lock")

	resp, _ = p09Post(t, srv.URL, "/v14/launch/confirm", form)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("确认冻结应 303,得到 %d", resp.StatusCode)
	}
	if kind, flash := p09Flash(t, resp); kind != "ok" || !strings.Contains(flash, "冻结完成") {
		t.Fatalf("迁移场景应冻结成功,得到 kind/flash=%s/%q", kind, flash)
	}
	pinned := p09Head(t, repo)

	// 原子性:冻结提交相对父提交恰新增三件套三个文件,别无其他。
	pinnedFiles := p09TreeFiles(t, repo, pinned)
	baseFiles := p09TreeFiles(t, repo, base)
	want := map[string]bool{
		".tally/intents/TASK-0902.yaml":       false,
		".tally/tasks/TASK-0902.yaml":         false,
		".tally/verifications/TASK-0902.yaml": false,
	}
	for p := range pinnedFiles {
		if !baseFiles[p] {
			if _, ok := want[p]; !ok {
				t.Errorf("冻结提交出现三件套之外的文件 %q", p)
			}
			want[p] = true
		}
	}
	for p, seen := range want {
		if !seen {
			t.Errorf("冻结提交缺少三件套文件 %q", p)
		}
	}

	// pinned 读回:迁移类型 + intake 风险提示以不可变证据存活(DL-06)。
	snap, err := artifacts.Load(repo, pinned)
	if err != nil {
		t.Fatalf("Load(pinned): %v", err)
	}
	var task taskcontract.TaskContract
	found := false
	for i := range snap.Tasks {
		if snap.Tasks[i].Contract.ID == "TASK-0902" {
			found, task = true, snap.Tasks[i].Contract
		}
	}
	if !found {
		t.Fatalf("pinned 快照里没有 TASK-0902 的 TaskContract")
	}
	if task.Profile != "standard" || len(task.Acceptance) != 1 {
		t.Errorf("迁移冻结合同 profile/条款数不符:%q / %d 条", task.Profile, len(task.Acceptance))
	}
	var verif *verification.Frozen
	for i := range snap.Verifications {
		if snap.Verifications[i].Frozen.TaskID() == "TASK-0902" {
			verif = snap.Verifications[i].Frozen
		}
	}
	if verif == nil {
		t.Fatalf("pinned 快照里没有 TASK-0902 的冻结 VerificationContract")
	}
	vb := verif.Contract().Acceptances[0].EvidenceBindings[0]
	if vb.Evidence.Type != "schema_check" || vb.Evidence.Path != "migrations/0009_orders_status_index.yaml" {
		t.Errorf("迁移证据绑定与 owner 声明不一致:%+v", vb.Evidence)
	}
	ia, err := artifacts.LoadIntent(repo, pinned, "TASK-0902")
	if err != nil {
		t.Fatalf("LoadIntent(pinned, TASK-0902): %v", err)
	}
	if ia.Intent.TaskType() != intentevidence.TaskTypeMigration {
		t.Errorf("冻结意图 task_type = %v,期望 db_migration", ia.Intent.TaskType())
	}
	if hints := ia.Intent.RiskHints(); len(hints) != 1 || hints[0] != "long_lock" {
		t.Errorf("冻结意图 RiskHints = %v,期望 [long_lock]", hints)
	}

	t.Logf("GATE db_migration: freeze_commit=%s task_hash=%s verification_hash=%s intent_revision=%s",
		pinned, task.Hash, verif.Hash(), ia.Intent.Revision())
}

// ---- AC2 + 负面:冻结准入阻断(缺失证据 / 半声明 / 不可承载语义)-------------

func TestV141IntentToFreezeGateAdmissionBlocksFreeze(t *testing.T) {
	repo, _ := p09Repo(t)
	srv := p09Server(t, repo)
	before := p09Head(t, repo)

	valid := func() url.Values {
		return url.Values{
			"task_type":     {"bug_fix"},
			"summary":       {"修复登出后会话残留"},
			"objective":     {"登出后销毁全部服务端会话状态"},
			"allowed_paths": {"src/auth/**"},
			"acid.0":        {"AC-01"},
			"acstmt.0":      {"登出后旧会话 token 必须失效"},
			"task_id":       {"TASK-0905"},
			"profile":       {"standard"},
			"ev_kind.0":     {"test"},
			"ev_path.0":     {"src/auth/logout_test.go"},
		}
	}

	// AC2:缺失 planned evidence 阻断执行就绪冻结,类型化列出缺失条款。
	resp, body := p09Post(t, srv.URL, "/v14/launch/confirm", func() url.Values {
		f := valid()
		f.Del("ev_kind.0")
		f.Del("ev_path.0")
		return f
	}())
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("缺证据的确认应 400,得到 %d", resp.StatusCode)
	}
	if !strings.Contains(body, "planned evidence") || !strings.Contains(body, "AC-01") {
		t.Errorf("应类型化列出缺失证据的条款(AC-01),body:\n%s", body[:min(len(body), 800)])
	}

	// 半声明(只有类型没有路径)是写错的输入,类型化拒绝。
	resp, body = p09Post(t, srv.URL, "/v14/launch/confirm", func() url.Values {
		f := valid()
		f.Del("ev_path.0")
		return f
	}())
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("证据半声明应 400,得到 %d", resp.StatusCode)
	}
	if !strings.Contains(body, "必须同时声明") {
		t.Errorf("半声明应类型化说明证据类型与路径必须同时声明,body:\n%s", body[:min(len(body), 800)])
	}

	// 不可承载语义(non_scope)绝不静默丢弃:冻结准入类型化退回。
	resp, body = p09Post(t, srv.URL, "/v14/launch/confirm", func() url.Values {
		f := valid()
		f.Set("non_scope", "不重构登录页")
		return f
	}())
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("携带 non_scope 的确认应 400,得到 %d", resp.StatusCode)
	}
	if !strings.Contains(body, "无法承载的语义") {
		t.Errorf("应类型化说明冻结制品无字段承载 non_scope,body:\n%s", body[:min(len(body), 800)])
	}

	// 全部阻断路径零 Git 副作用:HEAD 不动,树上没有半套制品。
	if now := p09Head(t, repo); now != before {
		t.Fatalf("准入失败不得产生提交,HEAD %s → %s", before, now)
	}
	for p := range p09TreeFiles(t, repo, before) {
		if strings.HasPrefix(p, ".tally/intents/") || strings.HasPrefix(p, ".tally/tasks/") || strings.HasPrefix(p, ".tally/verifications/") {
			t.Errorf("准入失败不得留下冻结制品,发现 %q", p)
		}
	}
}

// ---- 负面:非法澄清答案被服务端复核阻断(无草案、无第二次 AI 调用)---------

func TestV141IntentToFreezeGateInvalidClarificationBlocksChain(t *testing.T) {
	repo, _ := p09Repo(t)
	srv := p09Server(t, repo)
	fake := &airuntime.FakeExecutor{Result: p09ClarifyResult(t)}
	p09SetExecutor(t, fake)
	before := p09Head(t, repo)

	// 第一轮:合法提案 → 澄清请求(消耗一次 AI 调用)。
	if resp, body := p09Post(t, srv.URL, "/v14/launch/intent", url.Values{
		"intent": {"修复登出后会话残留"},
	}); resp.StatusCode != http.StatusOK || !strings.Contains(body, `name="v14a.0"`) {
		t.Fatalf("首轮提案应 200 且产出澄清,得到 %d", resp.StatusCode)
	}

	// 第二轮:携带上一轮渲染 schema 的答案不在声明候选集内 —— 有界 schema
	// 服务端复核 fail-safe 拒绝,绝不猜测修正,也不再触发任何新的 AI 调用。
	resp, body := p09Post(t, srv.URL, "/v14/launch/intent", url.Values{
		"intent": {"修复登出后会话残留"},
		"v14q.0": {"采用哪种修复策略?"},
		"v14k.0": {"risk_choice"},
		"v14c.0": {"热修复", "回滚再修"},
		"v14a.0": {"立即热修"},
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("候选集外的澄清答案应 400,得到 %d", resp.StatusCode)
	}
	if !strings.Contains(body, "不在候选") || !strings.Contains(body, "澄清答案未通过服务端复核") {
		t.Errorf("应类型化说明答案未通过复核,body:\n%s", body[:min(len(body), 800)])
	}
	if strings.Contains(body, `name="acstmt.0"`) {
		t.Errorf("复核失败不得产出草案")
	}
	if n := len(fake.Requests()); n != 1 {
		t.Fatalf("复核失败不得触发新的 AI 调用(应停在首轮提案的 1 次),得到 %d 次", n)
	}
	if now := p09Head(t, repo); now != before {
		t.Fatalf("澄清复核失败不得产生提交,HEAD %s → %s", before, now)
	}
}

// ---- 负面:重复确认内容幂等(ErrNoChanges,零新提交)------------------------

func TestV141IntentToFreezeGateDuplicateConfirmIdempotent(t *testing.T) {
	repo, _ := p09Repo(t)
	srv := p09Server(t, repo)
	form := url.Values{
		"task_type":     {"bug_fix"},
		"summary":       {"修复登出后会话残留"},
		"objective":     {"登出后销毁全部服务端会话状态"},
		"allowed_paths": {"src/auth/**"},
		"acid.0":        {"AC-01"},
		"acstmt.0":      {"登出后旧会话 token 必须失效"},
		"task_id":       {"TASK-0906"},
		"profile":       {"standard"},
		"ev_kind.0":     {"test"},
		"ev_path.0":     {"src/auth/logout_test.go"},
	}

	resp, _ := p09Post(t, srv.URL, "/v14/launch/confirm", form)
	if kind, _ := p09Flash(t, resp); kind != "ok" {
		t.Fatalf("首次确认应成功,得到 %s", kind)
	}
	first := p09Head(t, repo)

	// 完全相同的确认再提交:内容与 HEAD 一致 → 幂等,不产生新提交。
	resp, _ = p09Post(t, srv.URL, "/v14/launch/confirm", form)
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("重复确认应 303,得到 %d", resp.StatusCode)
	}
	kind, flash := p09Flash(t, resp)
	if kind != "ok" || !strings.Contains(flash, "未产生新提交") {
		t.Errorf("重复确认应幂等并如实说明,得到 kind=%s flash=%q", kind, flash)
	}
	if now := p09Head(t, repo); now != first {
		t.Fatalf("重复确认不得产生新提交,HEAD %s → %s", first, now)
	}

	// 幂等之后制品仍然可从 pinned 提交完整读回。
	if _, err := artifacts.Load(repo, first); err != nil {
		t.Fatalf("幂等后 Load(pinned): %v", err)
	}
	if _, err := artifacts.LoadIntent(repo, first, "TASK-0906"); err != nil {
		t.Fatalf("幂等后 LoadIntent(pinned): %v", err)
	}
}

// ---- 负面:陈旧基线类型化冲突(零部分写入;以当前基线重试成功)---------------

func TestV141IntentToFreezeGateStaleBaseConflictFailsSafe(t *testing.T) {
	repo, _ := p09Repo(t)
	srv := p09Server(t, repo)

	// 先经生产 HTTP 链冻结 TASK-0907,推进 HEAD。
	resp, _ := p09Post(t, srv.URL, "/v14/launch/confirm", url.Values{
		"task_type":     {"bug_fix"},
		"summary":       {"修复排序翻转"},
		"objective":     {"列表按创建时间升序"},
		"allowed_paths": {"src/list/**"},
		"acid.0":        {"AC-01"},
		"acstmt.0":      {"列表默认按创建时间升序"},
		"task_id":       {"TASK-0907"},
		"profile":       {"standard"},
		"ev_kind.0":     {"test"},
		"ev_path.0":     {"src/list/list_test.go"},
	})
	if kind, _ := p09Flash(t, resp); kind != "ok" {
		t.Fatalf("前置冻结应成功,得到 %s", kind)
	}
	frozen := p09Head(t, repo)

	// 以陈旧基线(前置冻结之前的 HEAD)驱动 handler 委托的同一生产事务:
	// 类型化 ErrBaseRevision,绝不静默改基,零部分写入。
	preFreezeBase := firstParentOf(t, repo, frozen)

	frozen0908, ferr := intakefreeze.Freeze(
		intakefreeze.ConfirmedDraft{
			TaskID:  "TASK-0908",
			Profile: "standard",
			Draft: intake.Draft{
				TaskType:  intake.TaskTypeBugFix,
				Summary:   "冲突探测任务",
				Objective: "验证基线冲突 fail-safe",
				Scope:     intake.Scope{AllowedPaths: []string{"src/x/**"}},
				Acceptance: []intake.AcceptanceClause{
					{ID: "AC-01", Statement: "x 行为正确"},
				},
			},
		},
		intakefreeze.VerificationPlan{
			Evidence: []intakefreeze.EvidenceDeclaration{{
				AcceptanceID: "AC-01",
				Kind:         verification.EvidenceType("test"),
				Path:         "src/x/x_test.go",
			}},
		})
	if ferr != nil {
		t.Fatalf("构造冻结产物: %v", ferr)
	}
	_, cerr := artifacts.FreezeCommit(context.Background(), repo, artifacts.FreezeCommitRequest{
		ExpectedBase: preFreezeBase, // 陈旧:HEAD 已被 TASK-0907 推进
		Artifacts:    frozen0908,
		Message:      "freeze: TASK-0908(应被基线冲突拒绝)",
		Signature:    &gitwriter.Signature{Name: p09Identity.Name, Email: p09Identity.Email},
	})
	if !errors.Is(cerr, gitwriter.ErrBaseRevision) {
		t.Fatalf("陈旧基线应类型化 ErrBaseRevision,得到 %v", cerr)
	}
	if now := p09Head(t, repo); now != frozen {
		t.Fatalf("基线冲突不得改变 HEAD,%s → %s", frozen, now)
	}
	for p := range p09TreeFiles(t, repo, frozen) {
		if strings.Contains(p, "TASK-0908") {
			t.Errorf("基线冲突不得留下部分制品,发现 %q", p)
		}
	}

	// 以当前基线重试同一冻结:成功 —— 冲突没有破坏仓库,恢复路径可用。
	res, cerr := artifacts.FreezeCommit(context.Background(), repo, artifacts.FreezeCommitRequest{
		ExpectedBase: frozen,
		Artifacts:    frozen0908,
		Message:      "freeze: TASK-0908(当前基线重试)",
		Signature:    &gitwriter.Signature{Name: p09Identity.Name, Email: p09Identity.Email},
	})
	if cerr != nil {
		t.Fatalf("以当前基线重试应成功: %v", cerr)
	}
	if res.Commit == "" || res.Base != frozen {
		t.Fatalf("重试提交结果异常:%+v", res)
	}
	if _, err := artifacts.LoadIntent(repo, res.Commit, "TASK-0908"); err != nil {
		t.Fatalf("重试冻结后 LoadIntent: %v", err)
	}
	t.Logf("GATE stale_base: conflicted_base=%s recovered_commit=%s", preFreezeBase, res.Commit)
}

// firstParentOf 返回 rev 提交的第一个父提交 sha(rev 必须有父提交)。
func firstParentOf(t *testing.T, repo, rev string) string {
	t.Helper()
	r, err := git.PlainOpen(repo)
	if err != nil {
		t.Fatalf("PlainOpen: %v", err)
	}
	h, err := r.ResolveRevision(plumbing.Revision(rev))
	if err != nil {
		t.Fatalf("ResolveRevision %q: %v", rev, err)
	}
	commit, err := r.CommitObject(*h)
	if err != nil {
		t.Fatalf("CommitObject: %v", err)
	}
	if commit.NumParents() == 0 {
		t.Fatalf("提交 %s 没有父提交", rev)
	}
	parent, err := commit.Parent(0)
	if err != nil {
		t.Fatalf("Parent(0): %v", err)
	}
	return parent.Hash.String()
}
