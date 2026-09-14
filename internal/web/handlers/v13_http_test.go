package handlers_test

// P1-44 的 v1.3 最小界面测试(PRD §26.2/§19)。
//
// 两层,同一文件:
//   - 纯 HTTP 层:数据源未接入时的空状态 / 明确失败(不需要数据库);
//   - 全链路层:临时 Git 制品仓库 + 独立 schema 的真实 Postgres,验证
//     Tasks 页派生状态(AC1)、Change 页 immutable ChangeSet / 验证证据 /
//     hunk 清单 / 覆盖(AC2)、Unattributed 阻断 Approve(AC3)、决定只写
//     HumanApprovalEvidence 而非状态(AC4)。数据库不可达时 SKIP,不产假失败。
//
// v13 页面用 handlers.Server 直接挂路由(不经 internal/web 的登录/CSRF 组;
// 那组行为由既有 handlers_http_test 覆盖)。写动作的防线在服务端 handler 本身。

import (
	"context"
	"database/sql"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	_ "github.com/jackc/pgx/v5/stdlib" // sql.Open("pgx") 驱动注册
	"github.com/pressly/goose/v3"

	"tally/internal/approval"
	"tally/internal/artifacts"
	"tally/internal/changeset"
	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/measurement"
	"tally/internal/runstore"
	"tally/internal/store"
	"tally/internal/verification"
	"tally/internal/web/handlers"
	"tally/migrations"
)

/* ---------- HTTP 层:直接挂 Server(无登录/CSRF 组) ---------- */

// v13NewServer 把 handlers.Server 直挂 chi(仅注册路由),返回测试服务器。
func v13NewServer(t *testing.T, q *store.Queries, repoPath string) *httptest.Server {
	t.Helper()
	s := &handlers.Server{
		Content: &handlers.ContentStore{
			RepoPath: repoPath,
			Identity: handlers.Identity{Name: "test", Email: "test@example.com"},
		},
		Trace: q,
	}
	r := chi.NewRouter()
	s.RegisterRoutes(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

func v13Get(t *testing.T, srvURL, path string) (int, string) {
	t.Helper()
	resp, err := http.Get(srvURL + path)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("读 %s 响应体: %v", path, err)
	}
	return resp.StatusCode, string(body)
}

// v13PostForm POST 表单且不跟随重定向(便于断言 303/Location)。
func v13PostForm(t *testing.T, srvURL, path string, form url.Values) (*http.Response, string) {
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

func TestV13PagesRenderEmptyStateWithoutSources(t *testing.T) {
	// Trace 与 RepoPath 都缺席:页面给「未接入」空状态,不 panic、不 500。
	srv := v13NewServer(t, nil, "")
	for _, path := range []string{"/v13/tasks", "/v13/runs", "/v13/changes/abc", "/v13/review/abc"} {
		status, body := v13Get(t, srv.URL, path)
		if status != http.StatusOK {
			t.Fatalf("GET %s = %d, want 200", path, status)
		}
		if !strings.Contains(body, "未接入") {
			t.Errorf("GET %s 应渲染未接入空状态", path)
		}
	}
	// 决定入口在数据源缺席时 303 回评审页并带明确提示(不落任何东西)。
	resp, _ := v13PostForm(t, srv.URL, "/v13/review/abc/decide", url.Values{"decision": {"approve"}})
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(resp.Header.Get("Location"), "kind=bad") {
		t.Fatalf("未接入时决定应 303 + kind=bad,得到 %d → %s", resp.StatusCode, resp.Header.Get("Location"))
	}
}

/* ---------- 测试制品仓库:base → docs 候选 → code 候选 ---------- */

const v13TaskYAML = `id: TASK-0001
profile: standard
title: 演示任务
objective: 验证 v1.3 界面装配
product_refs:
  - .tally/product/PRD.md#demo
requirement_refs:
  - .tally/requirements/REQ-0001.md
scope:
  allowed_paths:
    - docs/**
    - src/**
  forbidden_paths:
    - vendor/**
acceptance:
  - id: AC-01
    statement: 演示验收
    independence_required: none
`

const v13VerifYAML = `task_id: TASK-0001
revision: 1
acceptances:
  - id: AC-01
    statement: 演示验收
    independence_required: none
    evidence_bindings:
      - id: EV-AC01-01
        authoring_mode: pre_commit
        visibility_policy: visible
        evidence:
          type: validator
          validator: demo-v1
        qualification:
          change_overlap: required
`

// v13Repo 建一个临时制品仓库:base(.tally 制品)→ docs 候选(只改 docs)
// → code 候选(加一个 scope 内的实现代码文件)。三个提交都带固定过去时刻,
// 保证「制品事实早于运行事实」的推导顺序确定。
func v13Repo(t *testing.T) (dir, base, docs, code string) {
	t.Helper()
	dir = t.TempDir()
	r, err := git.PlainInit(dir, false)
	if err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	wt, err := r.Worktree()
	if err != nil {
		t.Fatalf("Worktree: %v", err)
	}
	when := time.Now().Add(-time.Hour)
	commit := func(msg string) string {
		t.Helper()
		h, err := wt.Commit(msg, &git.CommitOptions{
			Author:    &object.Signature{Name: "test", Email: "test@example.com", When: when},
			Committer: &object.Signature{Name: "test", Email: "test@example.com", When: when},
		})
		if err != nil {
			t.Fatalf("提交 %s: %v", msg, err)
		}
		return h.String()
	}
	write := func(path, content string) {
		t.Helper()
		full := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("建目录 %s: %v", path, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("写 %s: %v", path, err)
		}
		if _, err := wt.Add(path); err != nil {
			t.Fatalf("add %s: %v", path, err)
		}
	}

	write(".tally/project.yaml", "project: demo-proj\n")
	write(".tally/tasks/TASK-0001.yaml", v13TaskYAML)
	write(".tally/verifications/TASK-0001.yaml", v13VerifYAML)
	base = commit("base: .tally 制品")

	write("docs/guide.md", "# guide\n\ndemo\n")
	docs = commit("docs: 演示文档改动")

	write("src/app/handler.go", "package app\n\nfunc Handler() int { return 1 }\n")
	code = commit("feat: 实现代码改动(规则无法归账的残差)")

	return dir, base, docs, code
}

/* ---------- 数据库层:每测试独立 schema,不可达即 SKIP ---------- */

func v13TestDB(t *testing.T) *store.Store {
	t.Helper()
	ctx := context.Background()
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		base = "postgres://postgres:postgres@localhost:5432/tally?sslmode=disable"
	}
	if !v13DBReachable(base) {
		t.Skipf("v13 界面数据库测试需要 PostgreSQL(%s 不可达),跳过", base)
	}
	schema := "tally_v13_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	dsn := base + sep + "search_path=" + schema

	boot, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("打开引导连接: %v", err)
	}
	if _, err := boot.ExecContext(ctx, "CREATE SCHEMA "+schema); err != nil {
		boot.Close()
		t.Fatalf("建测试 schema %s: %v", schema, err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, boot, migrations.FS)
	if err != nil {
		boot.Close()
		t.Fatalf("创建 goose provider: %v", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		boot.Close()
		t.Fatalf("在测试 schema 跑迁移: %v", err)
	}
	boot.Close()

	st, err := store.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("打开测试 store: %v", err)
	}
	t.Cleanup(func() {
		st.Close()
		v13DropSchema(base, schema)
	})
	return st
}

func v13DBReachable(base string) bool {
	u, err := url.Parse(base)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "" {
		host = "localhost"
	}
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func v13DropSchema(base, schema string) {
	db, err := sql.Open("pgx", base)
	if err != nil {
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = db.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE")
}

/* ---------- 事实面种子:Run / ChangeSet / 验证 Run / 证据 ---------- */

// v13Seed 开两个实现 Run(docs / code 各一;producer_run_id UNIQUE = 一 Run
// 至多一个候选快照),各自固定一个候选并对候选各记一次验证 Run(绑定冻结契约)
// 与一条 pass 证据。返回候选身份与冻结契约哈希。
func v13Seed(t *testing.T, ctx context.Context, q *store.Queries, repo, base, docs, code string) (docsCS, codeCS changeset.ChangeSetID, frozenHash string) {
	t.Helper()
	rs := runstore.New(q)

	startRun := func(workspace string) int64 {
		t.Helper()
		runID, err := rs.StartCodegenRun(ctx, runstore.StartRunSpec{
			Project:                   "demo-proj",
			Task:                      "TASK-0001",
			Role:                      "coder",
			Runtime:                   "fake",
			RuntimeVersion:            "v0",
			Model:                     "demo-model",
			BehaviorRevision:          "b1",
			CostPlane:                 measurement.CostPlaneControl,
			BaseCommit:                base,
			TaskRevision:              "sha256:taskrev",
			VerificationRevision:      "sha256:verifrev",
			ContextPackHash:           "sha256:cp",
			WorkspaceID:               workspace,
			RuntimeCapabilitySnapshot: []byte(`{"runtime":"fake"}`),
		})
		if err != nil {
			t.Fatalf("StartCodegenRun: %v", err)
		}
		return runID
	}
	docsProducer := strconv.FormatInt(startRun("ws-docs"), 10)
	codeProducer := strconv.FormatInt(startRun("ws-code"), 10)

	frozenHash = v13FrozenHash(t, repo, base)

	fix := func(cand, producer string) changeset.ChangeSetID {
		t.Helper()
		cs, err := changeset.New(changeset.Spec{
			Project:           "demo-proj",
			BaseRevision:      base,
			CandidateRevision: cand,
			ProducerRun:       producer,
		})
		if err != nil {
			t.Fatalf("构造候选(%s): %v", cand, err)
		}
		if err := rs.SaveChangeSet(ctx, cs); err != nil {
			t.Fatalf("落候选快照: %v", err)
		}
		return cs.ID()
	}
	docsCS = fix(docs, docsProducer)
	codeCS = fix(code, codeProducer)

	seedRun := func(cs changeset.ChangeSetID, factID string) {
		t.Helper()
		vrID, err := rs.RecordVerificationRun(ctx, runstore.VerificationRunSpec{
			Project:           "demo-proj",
			Task:              "TASK-0001",
			ContractRevision:  1,
			ContractHash:      frozenHash,
			ChangeSetID:       cs,
			EnvName:           "env@sha256:envdef",
			EnvDefinitionHash: "sha256:envdef",
			VersionCombo:      core.VersionCombo{"tool": "v1"},
			Runtime:           "fake",
			Status:            runstore.VerificationPass,
		})
		if err != nil {
			t.Fatalf("RecordVerificationRun: %v", err)
		}
		if _, err := rs.RecordEvidence(ctx, "demo-proj", runstore.EvidenceRecord{
			FactID:           evidence.FactID(factID),
			RunID:            vrID,
			Task:             "TASK-0001",
			Acceptance:       "AC-01",
			BindingID:        "EV-AC01-01",
			VisibilityPolicy: verification.VisibilityVisible,
			Verdict:          evidence.VerdictPass,
			Recorder:         evidence.Recorder{Kind: evidence.RecorderValidator, Actor: "demo-v1"},
		}); err != nil {
			t.Fatalf("RecordEvidence: %v", err)
		}
	}
	seedRun(docsCS, "fact-docs-1")
	seedRun(codeCS, "fact-code-1")
	return docsCS, codeCS, frozenHash
}

// v13FrozenHash 从制品仓库 base 提交装载冻结契约,取其内容哈希。
func v13FrozenHash(t *testing.T, repo, rev string) string {
	t.Helper()
	snap, err := artifacts.Load(repo, rev)
	if err != nil {
		t.Fatalf("装载制品: %v", err)
	}
	if len(snap.Verifications) != 1 {
		t.Fatalf("制品快照应有 1 份验证契约,得到 %d", len(snap.Verifications))
	}
	return snap.Verifications[0].Frozen.Hash()
}

/* ---------- AC1:Tasks 页派生状态 / 当前 Run / 验证摘要 ---------- */

func TestV13TaskPageShowsDerivedStateRunAndVerificationSummary(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	docsCS, codeCS, _ := v13Seed(t, ctx, st.Queries(), repo, base, docs, code)

	srv := v13NewServer(t, st.Queries(), repo)
	status, body := v13Get(t, srv.URL, "/v13/tasks")
	if status != http.StatusOK {
		t.Fatalf("GET /v13/tasks = %d, want 200", status)
	}
	for _, want := range []string{
		"<h1>工作</h1>", // P2-02:工作列表以人类词呈现(PRD §30.1),不再是内部阶段词
		"TASK-0001",
		`href="/v14/tasks/TASK-0001"`, // 每行任务可点开任务详情(Work → Task,PRD §29)
		"演示任务",
		"VALIDATING",               // 派生状态:候选已产生且其验证 Run 已记录
		"pass 2 · fail 0",          // 验证摘要(两个候选各一次 pass 验证)
		"功能覆盖",                     // 门槛缺口:无适用功能覆盖报告时如实说明
		v13ShortCS(string(codeCS)), // 当前候选 = 派生指针指向的最新候选
	} {
		if !strings.Contains(body, want) {
			t.Errorf("Tasks 页应包含 %q", want)
		}
	}
	_ = docsCS
	if strings.Contains(body, ">APPROVED<") {
		t.Errorf("只记录了审批之外的事实时不得派生 APPROVED")
	}
}

/* ---------- AC2:Change 页 immutable 快照 / 证据 / hunk 清单 / 覆盖 ---------- */

func TestV13ChangePageShowsChangeSetEvidenceInventoryAndCoverage(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	docsCS, _, _ := v13Seed(t, ctx, st.Queries(), repo, base, docs, code)

	srv := v13NewServer(t, st.Queries(), repo)
	status, body := v13Get(t, srv.URL, "/v13/changes/"+string(docsCS))
	if status != http.StatusOK {
		t.Fatalf("GET change 页 = %d, want 200", status)
	}
	for _, want := range []string{
		"不可变快照",
		string(docsCS)[:20],     // 完整候选身份(截断防转义差异)
		"EV-AC01-01",            // 证据行
		"fact-docs-1",           // 证据事实 id
		"pass",                  // 判定
		"确定性 Hunk 清单",           // 清单标题
		"docs/guide.md",         // 清单里的文件
		"覆盖(功能 + independence)", // 覆盖区
	} {
		if !strings.Contains(body, want) {
			t.Errorf("Change 页应包含 %q", want)
		}
	}
	if !strings.Contains(body, "覆盖</span>") {
		t.Errorf("匹配契约的功能/independence 覆盖应显示为「覆盖」,body:\n%s", body)
	}
}

/* ---------- AC3 + AC4:Unattributed 阻断 approve;决定只写证据 ---------- */

func TestV13ReviewBlocksApproveOnUnattributedChange(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	_, codeCS, _ := v13Seed(t, ctx, st.Queries(), repo, base, docs, code)

	srv := v13NewServer(t, st.Queries(), repo)
	status, body := v13Get(t, srv.URL, "/v13/review/"+string(codeCS))
	if status != http.StatusOK {
		t.Fatalf("GET review 页 = %d, want 200", status)
	}
	for _, want := range []string{
		"unattributed",       // 残差 bucket 如实呈现
		"src/app/handler.go", // 未归账文件
		"Approve 被阻断",        // 阻断提示
	} {
		if !strings.Contains(body, want) {
			t.Errorf("Review 页应包含 %q", want)
		}
	}
	// 页面呈现层:approve 选项禁用,否决永远可用。
	if !strings.Contains(body, `value="approve" disabled`) {
		t.Errorf("Unattributed 非空时 approve 选项应禁用")
	}

	// 服务端防线:即使绕过页面直接 POST approve,也不落任何决定证据。
	resp, _ := v13PostForm(t, srv.URL, "/v13/review/"+string(codeCS)+"/decide",
		url.Values{"decision": {"approve"}, "reason": {"试一下"}, "decided_by": {"owner"}})
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(resp.Header.Get("Location"), "kind=bad") {
		t.Fatalf("被阻断的 approve 应 303 + kind=bad,得到 %d → %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	if rows := v13ApprovalCount(t, ctx, st.Queries(), "TASK-0001"); rows != 0 {
		t.Fatalf("被阻断的 approve 不应落 HumanApprovalEvidence,得到 %d 行", rows)
	}

	// 否决不受阻断限制:显式记 reject 证据。
	resp, _ = v13PostForm(t, srv.URL, "/v13/review/"+string(codeCS)+"/decide",
		url.Values{"decision": {"reject"}, "reason": {"有未归账改动"}, "decided_by": {"owner"}})
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(resp.Header.Get("Location"), "kind=ok") {
		t.Fatalf("reject 应 303 + kind=ok,得到 %d → %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	rows := v13ApprovalsForTask(t, ctx, st.Queries(), "TASK-0001")
	if len(rows) != 1 || string(rows[0].Approval.Decision()) != "reject" {
		t.Fatalf("应恰好记录一条 reject 证据,得到 %+v", rows)
	}
}

func TestV13ApproveWritesHumanApprovalEvidenceNotStatus(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	docsCS, _, _ := v13Seed(t, ctx, st.Queries(), repo, base, docs, code)

	srv := v13NewServer(t, st.Queries(), repo)
	_, body := v13Get(t, srv.URL, "/v13/review/"+string(docsCS))
	if !strings.Contains(body, "没有阻断项") {
		t.Errorf("docs-only 候选归账应无阻断项(body 不含「没有阻断项」):\n%s", body)
	}
	if strings.Contains(body, `value="approve" disabled`) {
		t.Errorf("无阻断时 approve 选项应可用")
	}

	resp, _ := v13PostForm(t, srv.URL, "/v13/review/"+string(docsCS)+"/decide",
		url.Values{"decision": {"approve"}, "reason": {"归账完整,验收无缺口"}, "decided_by": {"owner"}})
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(resp.Header.Get("Location"), "kind=ok") {
		t.Fatalf("approve 应 303 + kind=ok,得到 %d → %s", resp.StatusCode, resp.Header.Get("Location"))
	}

	// AC4:落的是 HumanApprovalEvidence(决定方向 / 原因 / 裁决人 / 候选绑定),
	// 不是任何状态字段 —— v1.3 事实面根本没有可写的任务状态列,状态由引擎派生。
	rows := v13ApprovalsForTask(t, ctx, st.Queries(), "TASK-0001")
	if len(rows) != 1 {
		t.Fatalf("应恰好一条审批证据,得到 %d", len(rows))
	}
	a := rows[0].Approval
	if csID, ok := a.ChangeSetID(); !ok || csID != string(docsCS) {
		t.Fatalf("审批证据应绑定当前候选 %s,得到 %s(ok=%v)", docsCS, csID, ok)
	}
	if a.DecidedBy() != "owner" || a.Reason() != "归账完整,验收无缺口" {
		t.Fatalf("审批证据应原样记录裁决人与原因,得到 by=%q reason=%q", a.DecidedBy(), a.Reason())
	}

	// 任务页派生状态不因审批证据而变成 APPROVED(功能覆盖报告缺席,VERIFIED
	// 不可达;决定只是证据,INV-STATE-01)。
	_, taskBody := v13Get(t, srv.URL, "/v13/tasks")
	if !strings.Contains(taskBody, "approve · owner") {
		t.Errorf("Tasks 页应显示最近审批记录(记录式呈现)")
	}
	if strings.Contains(taskBody, ">APPROVED<") {
		t.Errorf("审批证据不得直接改写派生状态为 APPROVED")
	}
}

/* ---------- 辅助 ---------- */

// v13ShortCS 与界面 short 函数同一约定:去 sha256: 前缀取前 12 位。
func v13ShortCS(s string) string {
	s = strings.TrimPrefix(s, "sha256:")
	if len(s) > 12 {
		s = s[:12]
	}
	return s
}

func v13ApprovalCount(t *testing.T, ctx context.Context, q *store.Queries, taskID string) int {
	t.Helper()
	return len(v13ApprovalsForTask(t, ctx, q, taskID))
}

func v13ApprovalsForTask(t *testing.T, ctx context.Context, q *store.Queries, taskID string) []approval.RecordedApproval {
	t.Helper()
	rows, err := approval.New(q).ApprovalsForTask(ctx, "demo-proj", taskID)
	if err != nil {
		t.Fatalf("读审批证据: %v", err)
	}
	return rows
}
