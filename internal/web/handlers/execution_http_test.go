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
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"tally/internal/environment"
	"tally/internal/jobs"
	"tally/internal/store"
	"tally/internal/web"
	"tally/internal/web/handlers"
)

/* ---------- 假实现:队列记录入队;ExecSource 返回预置只读数据 ---------- */

// fakeQueue 记录每次入队的 kind 与 payload,供断言「界面 POST 只入队」。
type fakeQueue struct {
	mu       sync.Mutex
	kinds    []jobs.Kind
	payloads []any
}

func (f *fakeQueue) Enqueue(_ context.Context, kind jobs.Kind, payload any) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.kinds = append(f.kinds, kind)
	f.payloads = append(f.payloads, payload)
	return int64(len(f.kinds)), nil
}

func (f *fakeQueue) len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.kinds)
}

func (f *fakeQueue) at(i int) (jobs.Kind, any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.kinds[i], f.payloads[i]
}

// fakeExec 提供执行引擎页面的只读数据;每个方法返回预置值,不碰数据库。
type fakeExec struct {
	envRuns  []store.EnvironmentRun
	depRuns  []store.DeployRun
	stats    []store.FingerprintStat
	excerpts []store.ListLatestFailureExcerptsRow
	days     []store.CountFailuresByDayRow
	deps     []store.DependencyChange
	stat     store.FingerprintStat
	statErr  error
	events   []store.ListFailureEventsByFingerprintRow
	impact   []store.ViewFingerprintImpactRow
	fuzzy    []store.FingerprintStat
}

func (f fakeExec) LatestEnvironmentRuns(context.Context) ([]store.EnvironmentRun, error) {
	return f.envRuns, nil
}
func (f fakeExec) ListDeployRuns(context.Context) ([]store.DeployRun, error) { return f.depRuns, nil }
func (f fakeExec) ListFingerprintStats(context.Context) ([]store.FingerprintStat, error) {
	return f.stats, nil
}
func (f fakeExec) ListLatestFailureExcerpts(context.Context) ([]store.ListLatestFailureExcerptsRow, error) {
	return f.excerpts, nil
}
func (f fakeExec) CountFailuresByDay(context.Context) ([]store.CountFailuresByDayRow, error) {
	return f.days, nil
}
func (f fakeExec) ListDependencyChangesRecent(context.Context) ([]store.DependencyChange, error) {
	return f.deps, nil
}
func (f fakeExec) GetFingerprintStat(context.Context, string) (store.FingerprintStat, error) {
	return f.stat, f.statErr
}
func (f fakeExec) ViewFingerprintImpact(context.Context, string) ([]store.ViewFingerprintImpactRow, error) {
	return f.impact, nil
}
func (f fakeExec) ListFailureEventsByFingerprint(context.Context, string) ([]store.ListFailureEventsByFingerprintRow, error) {
	return f.events, nil
}
func (f fakeExec) ListFingerprintStatsByFuzzy(context.Context, string) ([]store.FingerprintStat, error) {
	return f.fuzzy, nil
}

var noFollow = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }

// newExecServer 起带假队列与假读源的测试服务器,返回 server、内容仓库路径、已登录客户端。
func newExecServer(t *testing.T, q handlers.JobQueue, ex handlers.ExecSource) (*httptest.Server, string, *http.Client) {
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
		Queue:        q,
		Exec:         ex,
	}
	srv := httptest.NewServer(web.NewRouter(deps))
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookiejar: %v", err)
	}
	u, _ := url.Parse(srv.URL)
	jar.SetCookies(u, []*http.Cookie{{Name: "tally_session", Value: sessionToken, Path: "/"}})
	client := &http.Client{Jar: jar, CheckRedirect: noFollow}
	return srv, repo, client
}

const envDefYAML = `name: %s
project: %s
kind: docker-compose
compose_file: ./compose.yml
health_checks:
  - name: api
    type: http
    url: http://127.0.0.1:1/health
wait:
  timeout: 1s
`

// writeEnvDef 在内容仓库写一份环境定义并返回解析出的定义(带内容哈希)。
func writeEnvDef(t *testing.T, repo, file, name, project string) environment.Definition {
	t.Helper()
	dir := filepath.Join(repo, ".tally", "environments")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, file)
	if err := os.WriteFile(path, []byte(strings.Replace(strings.Replace(envDefYAML, "%s", name, 1), "%s", project, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	def, err := environment.ParseDefinition(path)
	if err != nil {
		t.Fatalf("解析环境定义: %v", err)
	}
	return def
}

// writeDeployDef 在内容仓库写一份部署定义。
func writeDeployDef(t *testing.T, repo, file, project, targetEnv string) {
	t.Helper()
	dir := filepath.Join(repo, ".tally", "deploys")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "project: " + project + "\n" +
		"target_env: " + targetEnv + "\n" +
		"tool: compose\n" +
		"artifact_digest: \"sha256:aaaa1111\"\n" +
		"config: ./deploy-compose.yml\n"
	if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func tstz(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

/* ---------- 动作边界:POST 只入队,不碰事实面 ---------- */

func TestEnvActionsEnqueueOnly(t *testing.T) {
	q := &fakeQueue{}
	srv, repo, client := newExecServer(t, q, fakeExec{})
	writeEnvDef(t, repo, "forge-dev.yaml", "dev", "forge")

	for _, path := range []string{"/environments/up", "/environments/down", "/environments/rebuild"} {
		resp := postForm(t, client, srv.URL, "/environments", path, url.Values{
			"def": {".tally/environments/forge-dev.yaml"}, "project": {"forge"}, "name": {"dev"},
		})
		loc := resp.Header.Get("Location")
		resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("POST %s 应 303(PRG),得到 %d", path, resp.StatusCode)
		}
		if !strings.Contains(loc, "kind=ok") || !strings.Contains(loc, "%E5%B7%B2%E5%85%A5%E9%98%9F") {
			t.Errorf("POST %s 后应带「已入队」提示重定向,得到 %q", path, loc)
		}
	}

	if q.len() != 3 {
		t.Fatalf("应入队 3 个 job,得到 %d", q.len())
	}
	// kind 与 payload 内容:worker 靠这些找到定义并执行。
	wantKinds := map[jobs.Kind]bool{
		jobs.KindEnvironmentUp: true, jobs.KindEnvironmentDown: true, jobs.KindEnvironmentRebuild: true,
	}
	for i := 0; i < 3; i++ {
		kind, payload := q.at(i)
		if !wantKinds[kind] {
			t.Errorf("第 %d 个 job 的 kind 不对: %s", i, kind)
		}
		p, ok := payload.(jobs.EnvActionPayload)
		if !ok {
			t.Fatalf("payload 应是 EnvActionPayload,得到 %T", payload)
		}
		if p.Project != "forge" || p.Name != "dev" || p.Def != ".tally/environments/forge-dev.yaml" {
			t.Errorf("payload 坐标不对: %+v", p)
		}
	}
}

func TestDeployRollbackEnqueuesOnly(t *testing.T) {
	q := &fakeQueue{}
	srv, repo, client := newExecServer(t, q, fakeExec{})
	writeDeployDef(t, repo, "forge-prod.yaml", "forge", "prod")

	resp := postForm(t, client, srv.URL, "/deploys", "/deploys/rollback", url.Values{
		"def": {".tally/deploys/forge-prod.yaml"}, "project": {"forge"},
		"target_env": {"prod"}, "digest": {"sha256:aaaa1111"},
	})
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("回滚 POST 应 303,得到 %d", resp.StatusCode)
	}
	if q.len() != 1 {
		t.Fatalf("应入队 1 个 job,得到 %d", q.len())
	}
	kind, payload := q.at(0)
	if kind != jobs.KindDeployRollback {
		t.Errorf("kind 应是 deploy_rollback,得到 %s", kind)
	}
	p := payload.(jobs.DeployRollbackPayload)
	if p.Project != "forge" || p.TargetEnv != "prod" || p.Digest != "sha256:aaaa1111" ||
		p.Def != ".tally/deploys/forge-prod.yaml" {
		t.Errorf("回滚 payload 不对: %+v", p)
	}
}

func TestDeployRollbackRejectsMissingTarget(t *testing.T) {
	q := &fakeQueue{}
	srv, repo, client := newExecServer(t, q, fakeExec{})
	writeDeployDef(t, repo, "forge-prod.yaml", "forge", "prod")

	resp := postForm(t, client, srv.URL, "/deploys", "/deploys/rollback", url.Values{
		"def": {".tally/deploys/forge-prod.yaml"}, "project": {"forge"},
		// 缺 target_env 与 digest
	})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(loc, "kind=bad") {
		t.Errorf("缺目标的回滚应带失败提示重定向,得到 %d %q", resp.StatusCode, loc)
	}
	if q.len() != 0 {
		t.Errorf("缺目标不应入队,入了 %d 个", q.len())
	}
}

func TestActionRejectsTraversalDefPath(t *testing.T) {
	q := &fakeQueue{}
	srv, _, client := newExecServer(t, q, fakeExec{})
	for _, bad := range []string{"../evil.yaml", "/abs.yaml", "", "a/../../b.yaml"} {
		resp := postForm(t, client, srv.URL, "/environments", "/environments/up", url.Values{
			"def": {bad}, "project": {"forge"}, "name": {"dev"},
		})
		loc := resp.Header.Get("Location")
		resp.Body.Close()
		if resp.StatusCode != http.StatusSeeOther || !strings.Contains(loc, "kind=bad") {
			t.Errorf("越界定义路径 %q 应带失败提示重定向,得到 %d %q", bad, resp.StatusCode, loc)
		}
	}
	if q.len() != 0 {
		t.Errorf("越界路径不应入队,入了 %d 个", q.len())
	}
}

func TestActionUnauthenticatedRejected(t *testing.T) {
	q := &fakeQueue{}
	srv, repo, _ := newExecServer(t, q, fakeExec{})
	writeEnvDef(t, repo, "forge-dev.yaml", "dev", "forge")

	// 无会话 cookie 的客户端
	client := &http.Client{CheckRedirect: noFollow}
	for _, path := range []string{"/environments/up", "/environments/down", "/environments/rebuild", "/deploys/rollback"} {
		resp, err := client.PostForm(srv.URL+path, url.Values{"def": {"x.yaml"}})
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 && resp.StatusCode != http.StatusSeeOther {
			t.Errorf("未登录 POST %s 不应成功,得到 %d", path, resp.StatusCode)
		}
		if resp.StatusCode == http.StatusSeeOther && resp.Header.Get("Location") != "/login" {
			t.Errorf("未登录 POST %s 应 303→/login,得到 %q", path, resp.Header.Get("Location"))
		}
	}
	if q.len() != 0 {
		t.Errorf("未登录不应入队,入了 %d 个", q.len())
	}
}

func TestActionMissingCSRFRejected(t *testing.T) {
	q := &fakeQueue{}
	srv, repo, client := newExecServer(t, q, fakeExec{})
	writeEnvDef(t, repo, "forge-dev.yaml", "dev", "forge")

	// 已登录但不带 csrf token 直接 POST
	resp, err := client.PostForm(srv.URL+"/environments/rebuild", url.Values{
		"def": {".tally/environments/forge-dev.yaml"}, "project": {"forge"}, "name": {"dev"},
	})
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("缺 CSRF 的动作 POST 应 403,得到 %d", resp.StatusCode)
	}
	if q.len() != 0 {
		t.Errorf("缺 CSRF 不应入队,入了 %d 个", q.len())
	}
}

/* ---------- 环境页:状态是算出来的,破坏性动作有二次确认,关 JS 可用 ---------- */

func TestEnvironmentsPageComputedStateAndDangerConfirm(t *testing.T) {
	ex := &fakeExec{} // 指针注入:先建服务器拿到仓库,再回填读取数据
	srv, repo, client := newExecServer(t, &fakeQueue{}, ex)
	forgeDef := writeEnvDef(t, repo, "forge-dev.yaml", "dev", "forge")
	writeEnvDef(t, repo, "hsk-stage.yaml", "stage", "hsk")
	ex.envRuns = []store.EnvironmentRun{
		{
			ID: 1, ProjectID: "forge", EnvName: "dev", DefinitionHash: forgeDef.Hash,
			StartedAt: tstz(time.Now().Add(-2 * time.Hour)), Status: "failed",
			VersionCombo: []byte(`{"kind":"docker-compose"}`),
		},
		{
			ID: 2, ProjectID: "hsk", EnvName: "stage", DefinitionHash: "sha256:whatever",
			StartedAt: tstz(time.Now().Add(-time.Hour)), Status: "up",
			VersionCombo: []byte(`{"kind":"docker-compose"}`),
		},
		// 定义已不在仓库的历史环境 → 只读行
		{
			ID: 3, ProjectID: "ghost", EnvName: "env", DefinitionHash: "sha256:ghost",
			StartedAt: tstz(time.Now().Add(-3 * time.Hour)), Status: "up",
			VersionCombo: []byte(`{}`),
		},
	}

	resp, err := client.Get(srv.URL + "/environments")
	if err != nil {
		t.Fatalf("GET /environments: %v", err)
	}
	body := bodyString(t, resp)

	// 状态从 environment_run 算出
	if !strings.Contains(body, "上次拉起失败") {
		t.Errorf("forge/dev 最近失败,应显示「上次拉起失败」")
	}
	if !strings.Contains(body, "在跑") {
		t.Errorf("hsk/stage 最近 up,应显示「在跑」")
	}
	// 定义已移除的环境:只读 + 说明
	if !strings.Contains(body, "定义已不在内容仓库") {
		t.Errorf("ghost/env 应标注定义已不在仓库")
	}
	// 三个动作表单都在
	for _, action := range []string{`action="/environments/up"`, `action="/environments/down"`, `action="/environments/rebuild"`} {
		if !strings.Contains(body, action) {
			t.Errorf("环境页应有动作表单 %s", action)
		}
	}
	// 破坏性动作:警示色 + 二次确认,文案说清后果(forge、hsk 两行各有推倒+重建)
	if n := strings.Count(body, `class="danger"`); n < 4 {
		t.Errorf("推倒/重建应是 danger 按钮(至少 4 个:两环境),得到 %d", n)
	}
	if n := strings.Count(body, "confirm("); n < 4 {
		t.Errorf("破坏性动作应带二次确认(至少 4 处),得到 %d", n)
	}
	if !strings.Contains(body, "未持久化到卷外的数据将丢失") {
		t.Errorf("推倒确认文案应说清后果")
	}
	if !strings.Contains(body, "期间服务不可用") {
		t.Errorf("重建确认文案应说清后果")
	}
	if !strings.Contains(body, "ghost") {
		t.Errorf("ghost 历史环境应展示")
	}
}

func TestEnvironmentsEmptyStateIsActionPrompt(t *testing.T) {
	srv, _, client := newExecServer(t, &fakeQueue{}, fakeExec{})
	resp, err := client.Get(srv.URL + "/environments")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body := bodyString(t, resp)
	if strings.Contains(body, "暂无数据") {
		t.Errorf("空状态不应是「暂无数据」")
	}
	if !strings.Contains(body, ".tally/environments") {
		t.Errorf("空状态应是行动提示,说明去哪里放定义")
	}
}

func TestDeploysPageHistoryAndRollbackDanger(t *testing.T) {
	srv, repo, client := newExecServer(t, &fakeQueue{}, fakeExec{depRuns: []store.DeployRun{
		{
			ID: 11, ProjectID: "forge", ArtifactDigest: "sha256:aaaa1111", TargetEnv: "prod", Tool: "compose",
			StartedAt: tstz(time.Now().Add(-time.Hour)), Status: "deployed",
			ExitCode: pgtype.Int4{Int32: 0, Valid: true}, VersionCombo: []byte(`{"tool":"compose"}`),
		},
		{
			ID: 10, ProjectID: "forge", ArtifactDigest: "sha256:bbbb2222", TargetEnv: "prod", Tool: "compose",
			StartedAt: tstz(time.Now().Add(-2 * time.Hour)), Status: "failed",
			ExitCode: pgtype.Int4{Int32: 1, Valid: true}, VersionCombo: []byte(`{"tool":"compose"}`),
		},
	}})
	writeDeployDef(t, repo, "forge-prod.yaml", "forge", "prod")

	resp, err := client.Get(srv.URL + "/deploys")
	if err != nil {
		t.Fatalf("GET /deploys: %v", err)
	}
	body := bodyString(t, resp)

	if !strings.Contains(body, "部署成功") || !strings.Contains(body, "部署失败") {
		t.Errorf("部署页应展示两种结果")
	}
	// 只有部署成功的行有回滚按钮,且是 danger + 二次确认 + 说清后果
	if n := strings.Count(body, "回滚到此版本"); n != 1 {
		t.Errorf("应只有 1 个回滚按钮(仅部署成功行),得到 %d", n)
	}
	if !strings.Contains(body, `action="/deploys/rollback"`) {
		t.Errorf("应有回滚表单")
	}
	if !strings.Contains(body, "confirm(") || !strings.Contains(body, "当前版本立即被替换") {
		t.Errorf("回滚应带二次确认且文案说清后果")
	}
	// 回滚表单带齐坐标隐藏域
	for _, want := range []string{`name="digest"`, `name="target_env"`, `name="def"`} {
		if !strings.Contains(body, want) {
			t.Errorf("回滚表单应有隐藏域 %s", want)
		}
	}
}

func TestDeploysEmptyStateIsActionPrompt(t *testing.T) {
	srv, _, client := newExecServer(t, &fakeQueue{}, fakeExec{})
	resp, err := client.Get(srv.URL + "/deploys")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body := bodyString(t, resp)
	if strings.Contains(body, "暂无数据") {
		t.Errorf("空状态不应是「暂无数据」")
	}
	if !strings.Contains(body, "tally deploy") {
		t.Errorf("空状态应是行动提示,说明怎么产生部署记录")
	}
}

/* ---------- 失败页:归并、排行(次数×项目数)、时间线与依赖对齐 ---------- */

func TestFailuresPageMergeRankAndTimeline(t *testing.T) {
	dayB := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	ex := fakeExec{
		stats: []store.FingerprintStat{
			{FpExact: "a3f9c2", FpFuzzy: "fz1", OccurrenceCount: 6,
				AffectedProjects: []string{"forge", "hsk", "restudio", "orienta"},
				Status:           "unresolved", LastSeen: tstz(time.Now().Add(-4 * time.Hour))},
			{FpExact: "7b1e04", FpFuzzy: "fz2", OccurrenceCount: 14,
				AffectedProjects: []string{"forge"},
				Status:           "fixed", LastSeen: tstz(time.Now().Add(-48 * time.Hour))},
		},
		excerpts: []store.ListLatestFailureExcerptsRow{
			{FpExact: "a3f9c2", Source: "build", Excerpt: pgtype.Text{String: "ABI 不匹配", Valid: true}},
			{FpExact: "7b1e04", Source: "build", Excerpt: pgtype.Text{String: "Prisma 迁移冲突", Valid: true}},
		},
		days: []store.CountFailuresByDayRow{
			{Day: pgtype.Date{Time: dayB, Valid: true}, Failures: 10},
		},
		deps: []store.DependencyChange{
			{ProjectID: "forge", Package: "node", ToVersion: pgtype.Text{String: "22", Valid: true},
				ChangedAt: tstz(dayB.Add(2 * time.Hour))},
		},
	}
	srv, _, client := newExecServer(t, &fakeQueue{}, ex)
	resp, err := client.Get(srv.URL + "/failures")
	if err != nil {
		t.Fatalf("GET /failures: %v", err)
	}
	body := bodyString(t, resp)

	// 归并 + 指纹条(同指纹同色由纯函数保证,这里断言两个指纹都有条)
	if n := strings.Count(body, "fp-strip"); n < 2 {
		t.Errorf("归并列表应给每个指纹渲染指纹条,fp-strip 出现 %d 次", n)
	}
	if !strings.Contains(body, "ABI 不匹配") || !strings.Contains(body, "Prisma 迁移冲突") {
		t.Errorf("归并列表应带摘要")
	}
	// 修复排行 = 次数 × 项目数:a3f9c2 = 6×4=24 排在 7b1e04 = 14×1=14 之前
	strongOf := func(v string) int { return strings.Index(body, "<strong>"+v+"</strong>") }
	r24, r14 := strongOf("24"), strongOf("14")
	if r24 < 0 || r14 < 0 {
		t.Fatalf("排行分应渲染成粗体单元格(24 与 14): %d / %d", r24, r14)
	}
	if r24 > r14 {
		t.Errorf("排行分 24(6×4)应排在 14(14×1)之前")
	}
	if !strings.Contains(body, "出现次数 × 影响项目数") {
		t.Errorf("排行区应说明算法")
	}
	// 时间线与依赖变更对齐:依赖刻度带项目·包·版本标题,失败柱有高度
	if !strings.Contains(body, "tl-marks") || !strings.Contains(body, "forge: node → 22") {
		t.Errorf("时间线应把依赖变更对齐进来并给出内容")
	}
	if !strings.Contains(body, "次失败") {
		t.Errorf("失败柱应带次数说明")
	}
	// 指纹链接到详情页
	if !strings.Contains(body, "/fingerprint?fp=a3f9c2") {
		t.Errorf("指纹应链到详情页")
	}
}

func TestFailuresEmptyStateIsActionPrompt(t *testing.T) {
	srv, _, client := newExecServer(t, &fakeQueue{}, fakeExec{})
	resp, err := client.Get(srv.URL + "/failures")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body := bodyString(t, resp)
	if strings.Contains(body, "暂无数据") {
		t.Errorf("空状态不应是「暂无数据」")
	}
	if !strings.Contains(body, "logpipe") {
		t.Errorf("空状态应是行动提示,说明失败记录怎么来")
	}
}

/* ---------- 指纹详情(视图 B 执行侧) ---------- */

func TestFingerprintDetailImpactFixDrift(t *testing.T) {
	fixAt := time.Now().Add(-72 * time.Hour)
	ex := fakeExec{
		stat: store.FingerprintStat{
			FpExact: "a3f9c2", FpFuzzy: "fz1",
			FirstSeen: tstz(fixAt.Add(-48 * time.Hour)), LastSeen: tstz(time.Now().Add(-4 * time.Hour)),
			OccurrenceCount: 6, AffectedProjects: []string{"forge", "hsk", "restudio", "orienta"},
			Status: "fixed", FixRef: pgtype.Text{String: "fixes/abi-node22.md", Valid: true},
			FixVerifiedAt: tstz(fixAt), FixVerifiedCombo: []byte(`{"node":"22.3.0"}`),
			DiagnoseAttempts: 1,
		},
		impact: []store.ViewFingerprintImpactRow{{
			FpExact: "a3f9c2", OccurrenceCount: 6, Status: "fixed",
			Drifted: pgtype.Bool{Bool: true, Valid: true}, // 修复后又复发
		}},
		events: []store.ListFailureEventsByFingerprintRow{
			{ID: 1, ProjectID: "hsk", Source: "build", OccurredAt: tstz(time.Now().Add(-4 * time.Hour)),
				Excerpt: pgtype.Text{String: "ABI 不匹配", Valid: true}, CommitSha: "e77a02ff"},
			{ID: 2, ProjectID: "forge", Source: "environment", OccurredAt: tstz(time.Now().Add(-40 * time.Hour)),
				Excerpt: pgtype.Text{String: "环境拉起失败", Valid: true}, CommitSha: ""}, // 环境来源无 commit
		},
		fuzzy: []store.FingerprintStat{
			{FpExact: "a3f9c2", FpFuzzy: "fz1", OccurrenceCount: 6, AffectedProjects: []string{"forge"}, Status: "fixed"},
			{FpExact: "a3f9d7", FpFuzzy: "fz1", OccurrenceCount: 3, AffectedProjects: []string{"forge"}, Status: "flaky",
				LastSeen: tstz(time.Now())},
		},
	}
	srv, _, client := newExecServer(t, &fakeQueue{}, ex)
	resp, err := client.Get(srv.URL + "/fingerprint?fp=a3f9c2")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body := bodyString(t, resp)

	for _, want := range []string{
		"forge hsk restudio orienta", // 影响项目
		"修复可能过期",                     // 漂移标记
		"fixes/abi-node22.md",        // 已验证修复引用
		"node=22.3.0",                // 验证时组合
		"e77a02",                     // 发生记录 commit 短显
		"a3f9d7",                     // 模糊邻居(排除自身后)
		"上限 3 轮",                     // 诊断预算
	} {
		if !strings.Contains(body, want) {
			t.Errorf("指纹详情应含 %q", want)
		}
	}
	// 环境来源的发生记录 commit 落「—」
	if !strings.Contains(body, "环境拉起失败") {
		t.Errorf("应展示环境来源的发生记录")
	}
}

func TestFingerprintNotFoundIsActionPrompt(t *testing.T) {
	ex := fakeExec{statErr: pgx.ErrNoRows}
	srv, _, client := newExecServer(t, &fakeQueue{}, ex)
	resp, err := client.Get(srv.URL + "/fingerprint?fp=不存在")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	body := bodyString(t, resp)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("找不到的指纹应正常渲染行动提示(200),得到 %d", resp.StatusCode)
	}
	if !strings.Contains(body, "还没有索引进库") || !strings.Contains(body, "tally reindex") {
		t.Errorf("空状态应是行动提示,说明如何重算索引")
	}
}

/* ---------- 重诊断:入队 jobs,三轮上限由后端守(INV-5 + PRD 规则⑤) ---------- */

func TestDiagnoseTriggerEnqueuesWhenBudgetLeft(t *testing.T) {
	q := &fakeQueue{}
	ex := fakeExec{stat: store.FingerprintStat{
		FpExact: "a3f9c2", Status: "unresolved", DiagnoseAttempts: 1, AffectedProjects: []string{"forge"},
	}}
	srv, _, client := newExecServer(t, q, ex)

	// 状态页:预算未用尽,显示触发按钮。
	resp, err := client.Get(srv.URL + "/diagnose?fp=a3f9c2")
	if err != nil {
		t.Fatalf("GET /diagnose: %v", err)
	}
	body := bodyString(t, resp)
	if !strings.Contains(body, `action="/diagnose"`) {
		t.Errorf("预算未用尽应显示「触发重诊断」表单")
	}
	if !strings.Contains(body, "已诊断 1 / 3 轮") {
		t.Errorf("应显示第几轮:已诊断 1 / 3 轮")
	}

	// 触发:真入队一个 diagnose job。
	resp = postForm(t, client, srv.URL, "/diagnose?fp=a3f9c2", "/diagnose", url.Values{
		"fp": {"a3f9c2"}, "project": {"forge"},
	})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(loc, "kind=ok") {
		t.Fatalf("触发重诊断应带「已入队」重定向,得到 %d %q", resp.StatusCode, loc)
	}
	if q.len() != 1 {
		t.Fatalf("应入队 1 个 diagnose job,得到 %d", q.len())
	}
	kind, payload := q.at(0)
	if kind != jobs.KindDiagnose {
		t.Errorf("kind 应是 diagnose,得到 %s", kind)
	}
	if p, ok := payload.(jobs.DiagnosePayload); !ok || p.Fingerprint != "a3f9c2" {
		t.Errorf("payload 应带指纹 a3f9c2: %+v", payload)
	}
}

func TestDiagnoseTriggerRejectedAtCap(t *testing.T) {
	q := &fakeQueue{}
	ex := fakeExec{stat: store.FingerprintStat{
		FpExact: "b8e1d5", Status: "unresolved", DiagnoseAttempts: 3, AffectedProjects: []string{"forge"},
	}}
	srv, _, client := newExecServer(t, q, ex)

	// 状态页:三轮用尽,显示「已达上限,不再诊断」,不给触发按钮。
	resp, err := client.Get(srv.URL + "/diagnose?fp=b8e1d5")
	if err != nil {
		t.Fatalf("GET /diagnose: %v", err)
	}
	body := bodyString(t, resp)
	if !strings.Contains(body, "已达上限,不再诊断") {
		t.Errorf("三轮用尽应显示「已达上限,不再诊断」")
	}
	if strings.Contains(body, `action="/diagnose"`) {
		t.Errorf("三轮用尽不应显示触发按钮")
	}

	// 即便手工 POST,后端也拒绝入队。
	resp = postForm(t, client, srv.URL, "/diagnose?fp=b8e1d5", "/diagnose", url.Values{
		"fp": {"b8e1d5"}, "project": {"forge"},
	})
	loc := resp.Header.Get("Location")
	resp.Body.Close()
	if !strings.Contains(loc, "kind=warn") {
		t.Errorf("三轮用尽 POST 应带「已达上限」警告重定向,得到 %q", loc)
	}
	if q.len() != 0 {
		t.Errorf("三轮用尽不应入队,入了 %d 个(后端没守住)", q.len())
	}
}

func TestDiagnoseTriggerRejectedWhenConcluded(t *testing.T) {
	q := &fakeQueue{}
	ex := fakeExec{stat: store.FingerprintStat{
		FpExact: "c2a4e6", Status: "fixed", DiagnoseAttempts: 1, AffectedProjects: []string{"forge"},
	}}
	srv, _, client := newExecServer(t, q, ex)

	resp, err := client.Get(srv.URL + "/diagnose?fp=c2a4e6")
	if err != nil {
		t.Fatalf("GET /diagnose: %v", err)
	}
	body := bodyString(t, resp)
	if !strings.Contains(body, "已有定论") {
		t.Errorf("终态出口应显示「已有定论」")
	}

	resp = postForm(t, client, srv.URL, "/diagnose?fp=c2a4e6", "/diagnose", url.Values{
		"fp": {"c2a4e6"}, "project": {"forge"},
	})
	resp.Body.Close()
	if q.len() != 0 {
		t.Errorf("已有定论不应入队,入了 %d 个", q.len())
	}
}

/* ---------- 关 JS:表单是原生整页提交,htmx 只是渐进增强 ---------- */
func TestActionFormsArePlainNoJSUsable(t *testing.T) {
	srv, repo, client := newExecServer(t, &fakeQueue{}, fakeExec{})
	writeEnvDef(t, repo, "forge-dev.yaml", "dev", "forge")
	writeDeployDef(t, repo, "forge-prod.yaml", "forge", "prod")

	resp, err := client.Get(srv.URL + "/environments")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	envBody := bodyString(t, resp)
	// 没有任何 hx-* 属性:表单退化为整页提交,关掉 JS 照样可用。
	for _, attr := range []string{"hx-post", "hx-get", "hx-swap", "hx-target"} {
		if strings.Contains(envBody, attr) {
			t.Errorf("动作表单不应依赖 htmx 属性 %q(关 JS 必须可用)", attr)
		}
	}
	// 每个动作 form 自带 method/action/csrf,是能独立工作的整页提交。
	if n := strings.Count(envBody, `method="post"`); n < 3 {
		t.Errorf("环境页动作应是原生 POST 表单,得到 %d 个", n)
	}
	if !strings.Contains(envBody, `name="csrf_token"`) {
		t.Errorf("动作表单应自带 CSRF 隐藏域")
	}
}
