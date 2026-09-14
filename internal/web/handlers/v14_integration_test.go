// v14_integration_test.go 拥有集成审计投影页的处理器测试(P5-05 验证面)。
// 核心 AC:UI 永不把候选验收等同于成功集成 —— 两分区独立来源、独立呈现。
package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"tally/internal/gitwriter"
	"tally/internal/store"
	"tally/internal/web/handlers"
)

// p505Server 构造集成审计页服务器(v14 路由原样装配)。
func p505Server(t *testing.T, st *store.Store, repo string) *httptest.Server {
	t.Helper()
	s := &handlers.Server{
		Content: &handlers.ContentStore{
			RepoPath: repo,
			Identity: handlers.Identity{Name: "test", Email: "test@example.com"},
		},
		Trace: st.Queries(),
	}
	r := chi.NewRouter()
	s.RegisterV14Routes(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// p505SeedAcceptance 经生产 decide 流录得一条验收决定(既有审批证据语义)。
func p505SeedAcceptance(t *testing.T, st *store.Store, repo, docsCS string) {
	t.Helper()
	decideSrv := v13NewServer(t, st.Queries(), repo)
	resp, _ := v13PostForm(t, decideSrv.URL, "/v13/review/"+docsCS+"/decide",
		map[string][]string{"decision": {"approve"}, "reason": {"归账完整"}, "decided_by": {"owner"}})
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(resp.Header.Get("Location"), "kind=ok") {
		t.Fatalf("decide 应 303 + kind=ok,得到 %d → %s", resp.StatusCode, resp.Header.Get("Location"))
	}
}

// p505SeedIntegrationRef 经 git update-ref 落一条集成分支事实(审计列举的
// 读面夹具;生产写面是 P5-02/P5-03 的具名操作)。
func p505SeedIntegrationRef(t *testing.T, repo, name, at string) {
	t.Helper()
	if _, err := gitwriter.RunGit(context.Background(), repo, "", "update-ref", name, at); err != nil {
		t.Fatalf("update-ref: %v", err)
	}
}

// 仅验收:审批证据已录得、无集成分支 → Accepted 区如实呈现,集成标注
// 「尚未集成」,Integrated 区为空(AC:验收绝不等同于集成)。
func TestV14IntegrationAcceptedWithoutIntegration(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	docsCS, _, _ := v13Seed(t, ctx, st.Queries(), repo, base, docs, code)
	srv := p505Server(t, st, repo)
	p505SeedAcceptance(t, st, repo, string(docsCS))

	_, body := v13Get(t, srv.URL, "/v14/integration")
	if !strings.Contains(body, "已验收(Accepted") || !strings.Contains(body, "TASK-0001") {
		t.Errorf("Accepted 区应呈现验收决定:\n%s", body)
	}
	if !strings.Contains(body, "尚未集成") {
		t.Errorf("无集成分支的验收必须标注「尚未集成」(AC):\n%s", body)
	}
	if !strings.Contains(body, "尚无集成分支事实") {
		t.Errorf("Integrated 区应为空并如实呈现")
	}
}

// 验收 + 集成并存:集成分支事实落地 → Integrated 区呈现;两分区独立,
// 无对应候选的集成分支不改变任何验收行的语义。
func TestV14IntegrationAcceptedAndIntegrated(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	docsCS, _, _ := v13Seed(t, ctx, st.Queries(), repo, base, docs, code)
	srv := p505Server(t, st, repo)
	p505SeedAcceptance(t, st, repo, string(docsCS))

	// 一条不对应任何已验收候选的集成分支(64 位十六进制命名)。
	p505SeedIntegrationRef(t, repo,
		"refs/heads/tally/integration/"+strings.Repeat("ab", 32), base)

	_, body := v13Get(t, srv.URL, "/v14/integration")
	if !strings.Contains(body, "已集成(Integrated") || !strings.Contains(body, "refs/heads/tally/integration/") {
		t.Errorf("Integrated 区应呈现集成分支事实:\n%s", body)
	}
	// AC:验收行的集成标注仍由确定性对账决定 —— 不对应的分支不得把验收行
	// 翻转为「已集成」。
	if !strings.Contains(body, "尚未集成") {
		t.Errorf("验收行的集成标注必须经确定性命名对账,不得被无关分支污染:\n%s", body)
	}
}

// 治理命名空间之外的引用不见于集成事实(列举范围锁定)。
func TestV14IntegrationIgnoresForeignRefs(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	docsCS, _, _ := v13Seed(t, ctx, st.Queries(), repo, base, docs, code)
	srv := p505Server(t, st, repo)
	p505SeedAcceptance(t, st, repo, string(docsCS))

	p505SeedIntegrationRef(t, repo, "refs/heads/feature/random", base)

	_, body := v13Get(t, srv.URL, "/v14/integration")
	if strings.Contains(body, "feature/random") {
		t.Errorf("治理命名空间之外的引用不得出现在集成事实区:\n%s", body)
	}
}
