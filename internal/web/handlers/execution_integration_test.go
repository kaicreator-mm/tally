//go:build integration

package handlers_test

import (
	"context"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"tally/internal/jobs"
	"tally/internal/knowledge"
	"tally/internal/store"
	"tally/internal/web"
	"tally/internal/web/handlers"
)

// 集成测试(真 Postgres,不 mock 数据库,见 docs/05-test-standards.md):
// 核心断言是动作边界 —— POST 动作真往 job 表入队,且 handler 没往事实面写任何行(INV-5)。
// 每测试 store.TestDB 拿独立 schema,断言可断全表(不被它包并行写入污染)。

func openExecStore(t *testing.T) *store.Store {
	t.Helper()
	return store.TestDB(t)
}

// newExecServerReal 起带真会话表 + 真队列 + 真读源的服务器,返回已登录客户端。
func newExecServerReal(t *testing.T, st *store.Store) (*httptest.Server, string, *http.Client) {
	t.Helper()
	repo := t.TempDir()
	id := handlers.Identity{Name: "itest", Email: "itest@example.com"}
	if err := handlers.EnsureRepo(repo, id); err != nil {
		t.Fatalf("EnsureRepo: %v", err)
	}
	pw, err := bcrypt.GenerateFromPassword([]byte("pw"), 4)
	if err != nil {
		t.Fatalf("bcrypt: %v", err)
	}
	deps := web.Deps{
		Sessions:     web.PGSessions(st.Queries()), // 真会话表
		PasswordHash: string(pw),
		ContentRepo:  repo,
		Identity:     id,
		Queue:        jobs.New(st, nil), // 真队列:Enqueue 真写 job 表;handlers 表为空(不执行)
		Exec:         st.Queries(),      // 真读源
		// T05c:任务算出状态 / 已验证修复库(同内容仓库)/ 追溯与覆盖率(真索引面)。
		Tasks:     handlers.NewExecTaskSource(st.Queries()),
		Knowledge: knowledge.NewStore(repo, st.Queries()),
		Trace:     st.Queries(),
	}
	srv := httptest.NewServer(web.NewRouter(deps))
	t.Cleanup(srv.Close)

	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, CheckRedirect: noFollow}

	// 完整登录链路(真 Postgres session)
	resp, err := client.PostForm(srv.URL+"/login", url.Values{"password": {"pw"}})
	if err != nil {
		t.Fatalf("登录: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("登录应 303,得到 %d", resp.StatusCode)
	}
	return srv, repo, client
}

// countRows 数某事实面表的全表行数。独立 schema 下没有它包写入,断全表即严格断言
// 「界面没写事实面」(INV-5)。
func countRows(t *testing.T, st *store.Store, table string) int64 {
	t.Helper()
	var n int64
	if err := st.Pool().QueryRow(context.Background(),
		"SELECT count(*) FROM "+table).Scan(&n); err != nil {
		t.Fatalf("数 %s: %v", table, err)
	}
	return n
}

func countJobs(t *testing.T, st *store.Store, kind string) int64 {
	t.Helper()
	var n int64
	if err := st.Pool().QueryRow(context.Background(),
		"SELECT count(*) FROM job WHERE kind = $1", kind).Scan(&n); err != nil {
		t.Fatalf("数 job(%s): %v", kind, err)
	}
	return n
}

/* ---------- 动作边界:只入队 job,不写事实面(INV-5 核心) ---------- */

func TestEnvActionEnqueuesJobAndFactsUntouched(t *testing.T) {
	st := openExecStore(t)
	srv, repo, client := newExecServerReal(t, st)
	project := "forge-env"
	writeEnvDef(t, repo, "forge-dev.yaml", "dev", project)

	envBefore := countRows(t, st, "environment_run")
	depBefore := countRows(t, st, "deploy_run")
	jobBefore := countJobs(t, st, string(jobs.KindEnvironmentUp))

	resp := postForm(t, client, srv.URL, "/environments", "/environments/up", url.Values{
		"def": {".tally/environments/forge-dev.yaml"}, "project": {project}, "name": {"dev"},
	})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(loc, "kind=ok") {
		t.Fatalf("拉起 POST 应带「已入队」提示重定向,得到 %d %q", resp.StatusCode, loc)
	}

	// jobs 表真有一行(pending,等 worker 取)
	if got := countJobs(t, st, string(jobs.KindEnvironmentUp)); got != jobBefore+1 {
		t.Errorf("job 表应多一行 environment_up: before=%d after=%d", jobBefore, got)
	}
	var status, payloadJSON string
	if err := st.Pool().QueryRow(context.Background(),
		"SELECT status, payload::text FROM job WHERE kind = $1 ORDER BY id DESC LIMIT 1",
		string(jobs.KindEnvironmentUp)).Scan(&status, &payloadJSON); err != nil {
		t.Fatalf("读回 job: %v", err)
	}
	if status != "pending" {
		t.Errorf("新入队的 job 应是 pending,得到 %q", status)
	}
	if !strings.Contains(payloadJSON, "forge-dev.yaml") {
		t.Errorf("job payload 应带定义坐标: %s", payloadJSON)
	}

	// 事实面全表一行没多:handler 只入队,写事实面是 worker 的事(INV-5)
	if got := countRows(t, st, "environment_run"); got != envBefore {
		t.Errorf("environment_run 不应被界面写: before=%d after=%d(INV-5)", envBefore, got)
	}
	if got := countRows(t, st, "deploy_run"); got != depBefore {
		t.Errorf("deploy_run 不应被界面写: before=%d after=%d(INV-5)", depBefore, got)
	}
}

func TestDeployRollbackEnqueuesJobAndFactsUntouched(t *testing.T) {
	st := openExecStore(t)
	srv, repo, client := newExecServerReal(t, st)
	project := "forge-rb"
	writeDeployDef(t, repo, "forge-prod.yaml", project, "prod")

	depBefore := countRows(t, st, "deploy_run")
	jobBefore := countJobs(t, st, string(jobs.KindDeployRollback))

	resp := postForm(t, client, srv.URL, "/deploys", "/deploys/rollback", url.Values{
		"def": {".tally/deploys/forge-prod.yaml"}, "project": {project},
		"target_env": {"prod"}, "digest": {"sha256:aaaa1111"},
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("回滚 POST 应 303,得到 %d", resp.StatusCode)
	}
	if got := countJobs(t, st, string(jobs.KindDeployRollback)); got != jobBefore+1 {
		t.Errorf("job 表应多一行 deploy_rollback: before=%d after=%d", jobBefore, got)
	}
	if got := countRows(t, st, "deploy_run"); got != depBefore {
		t.Errorf("deploy_run 不应被界面写: before=%d after=%d(INV-5)", depBefore, got)
	}
}

func TestActionRejectedWithoutAuthOrCSRF(t *testing.T) {
	st := openExecStore(t)
	srv, repo, client := newExecServerReal(t, st)
	writeEnvDef(t, repo, "forge-dev.yaml", "dev", "forge")
	jobBefore := countJobs(t, st, string(jobs.KindEnvironmentRebuild))

	// 未登录:303→/login,不入队
	anon := &http.Client{CheckRedirect: noFollow}
	resp, err := anon.PostForm(srv.URL+"/environments/rebuild", url.Values{
		"def": {".tally/environments/forge-dev.yaml"}, "project": {"forge"}, "name": {"dev"},
	})
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
		t.Errorf("未登录动作 POST 应 303→/login,得到 %d→%q", resp.StatusCode, resp.Header.Get("Location"))
	}

	// 已登录无 CSRF:403,不入队
	resp, err = client.PostForm(srv.URL+"/environments/rebuild", url.Values{
		"def": {".tally/environments/forge-dev.yaml"}, "project": {"forge"}, "name": {"dev"},
	})
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("缺 CSRF 的动作 POST 应 403,得到 %d", resp.StatusCode)
	}

	if got := countJobs(t, st, string(jobs.KindEnvironmentRebuild)); got != jobBefore {
		t.Errorf("被拒的请求不应入队: before=%d after=%d", jobBefore, got)
	}
}

/* ---------- 页面从真库读:状态算出、归并、排行、时间线对齐 ---------- */

func TestPagesReadFromRealFacts(t *testing.T) {
	st := openExecStore(t)
	srv, _, client := newExecServerReal(t, st)
	ctx := context.Background()
	q := st.Queries()
	now := time.Now().UTC()
	project := "web-itest"

	// 环境事实:一行 up
	if _, err := q.InsertEnvironmentRunOutcome(ctx, store.InsertEnvironmentRunOutcomeParams{
		ProjectID: project, EnvName: "web", DefinitionHash: "sha256:def0",
		StartedAt:  pgtype.Timestamptz{Time: now.Add(-time.Hour), Valid: true},
		FinishedAt: pgtype.Timestamptz{Time: now.Add(-55 * time.Minute), Valid: true},
		Status:     "up", HealthCheckResult: []byte(`{"ok":true}`),
		VersionCombo: []byte(`{"kind":"docker-compose"}`),
	}); err != nil {
		t.Fatalf("种 environment_run: %v", err)
	}

	// 失败事实:同指纹两条事件(environment 来源)+ 依赖变更。
	// 独立 schema 起是空的,出现次数就是种的 2。
	runID, err := q.InsertEnvironmentRunOutcome(ctx, store.InsertEnvironmentRunOutcomeParams{
		ProjectID: project, EnvName: "web", DefinitionHash: "sha256:def0",
		StartedAt:  pgtype.Timestamptz{Time: now.Add(-2 * time.Hour), Valid: true},
		FinishedAt: pgtype.Timestamptz{Time: now.Add(-2 * time.Hour), Valid: true},
		Status:     "failed", HealthCheckResult: []byte(`{"ok":false}`),
		VersionCombo: []byte(`{"kind":"docker-compose"}`),
	})
	if err != nil {
		t.Fatalf("种失败来源 environment_run: %v", err)
	}
	for i := 0; i < 2; i++ {
		if _, err := q.InsertFailureEvent(ctx, store.InsertFailureEventParams{
			ProjectID: project, Source: "environment",
			EnvironmentRunID: pgtype.Int8{Int64: runID, Valid: true},
			Excerpt:          pgtype.Text{String: "集成测试超时:连接被重置", Valid: true},
			Normalized:       pgtype.Text{String: "集成测试超时:连接被重置", Valid: true},
			FpExact:          "it05b1aa", FpFuzzy: "fz-it05",
			PipelineVersion:     "v4",
			OccurredAt:          pgtype.Timestamptz{Time: now.Add(-time.Duration(i+1) * time.Hour), Valid: true},
			VersionCombo:        []byte(`{"kind":"docker-compose"}`),
			OtherTargetCommands: []byte("[]"),
			ReproCommand:        []byte("null"),
		}); err != nil {
			t.Fatalf("种 failure_event: %v", err)
		}
		if _, err := q.UpsertFingerprintStat(ctx, store.UpsertFingerprintStatParams{
			FpExact: "it05b1aa", FpFuzzy: "fz-it05",
			FirstSeen: pgtype.Timestamptz{Time: now.Add(-time.Duration(i+1) * time.Hour), Valid: true},
			Project:   project,
		}); err != nil {
			t.Fatalf("种 fingerprint_stat: %v", err)
		}
	}
	if _, err := q.InsertDependencyChange(ctx, store.InsertDependencyChangeParams{
		ProjectID: project, CommitSha: "c0ffee", Package: "left-pad",
		ToVersion: pgtype.Text{String: "2.0.0", Valid: true},
		ChangedAt: pgtype.Timestamptz{Time: now.Add(-90 * time.Minute), Valid: true},
	}); err != nil {
		t.Fatalf("种 dependency_change: %v", err)
	}

	// 环境页:状态从事实面算出(定义不在仓库 → 只读行也展示)
	resp, err := client.Get(srv.URL + "/environments")
	if err != nil {
		t.Fatalf("GET /environments: %v", err)
	}
	envBody := bodyString(t, resp)
	eidx := strings.Index(envBody, project)
	if eidx < 0 {
		t.Fatalf("环境页应从真库读出 %s", project)
	}
	eseg := envBody[eidx:]
	if end := strings.Index(eseg, "</tr>"); end > 0 {
		eseg = eseg[:end]
	}
	if !strings.Contains(eseg, "在跑") {
		t.Errorf("%s 最近一次拉起是 up,该行应显示「在跑」: %s", project, eseg)
	}
	if !strings.Contains(envBody, "定义已不在内容仓库") {
		t.Errorf("无定义的历史环境应只读展示")
	}

	// 失败页:按指纹归并(两条事件一行)、排行、时间线与依赖对齐
	resp, err = client.Get(srv.URL + "/failures")
	if err != nil {
		t.Fatalf("GET /failures: %v", err)
	}
	failBody := bodyString(t, resp)
	idx := strings.Index(failBody, "fp=it05b1aa")
	if idx < 0 {
		t.Fatalf("失败页应按指纹 it05b1aa 归并展示")
	}
	// 归并:两条事件在归并列表里是一行,次数 = 2(排行分算法 次数×项目数 由单测钉住)
	seg := failBody[idx:]
	if end := strings.Index(seg, "</tr>"); end > 0 {
		seg = seg[:end]
	}
	if !strings.Contains(seg, `<td class="num">2</td>`) {
		t.Errorf("it05b1aa 归并行应显示出现 2 次: %s", seg)
	}
	if !strings.Contains(failBody, "集成测试超时") {
		t.Errorf("失败页应带脱敏摘要")
	}
	if !strings.Contains(failBody, "left-pad") {
		t.Errorf("时间线应把依赖变更 left-pad 对齐进来")
	}
	if !strings.Contains(failBody, "出现次数 × 影响项目数") {
		t.Errorf("失败页应有修复排行区")
	}

	// 指纹详情:从真库读影响面
	resp, err = client.Get(srv.URL + "/fingerprint?fp=it05b1aa")
	if err != nil {
		t.Fatalf("GET /fingerprint: %v", err)
	}
	fpBody := bodyString(t, resp)
	if !strings.Contains(fpBody, project) || !strings.Contains(fpBody, "影响面") {
		t.Errorf("指纹详情应展示影响项目")
	}
	if strings.Contains(fpBody, "还没有索引进库") {
		t.Errorf("已种索引的指纹不应落「未索引」空状态")
	}
}
