// v13_propose_rule_test.go 拥有 P3-08 评审动作的投影测试:仅此一次决定走既有
// 审批证据语义且绝不触碰治理面;提议新规只落治理提案草稿且绝不写审批证据、
// 绝不自动批准;治理存储未接入时显式失败。AC:一次性决定绝不自动改策略。
package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"tally/internal/policygovernance"
	"tally/internal/store"
	"tally/internal/web/handlers"
)

const p308RuleYAML = `id: POL-P308-EXAMPLE
enforcement: gate
rationale: P3-08 测试规则:此类情况应当有门
trigger:
  risk_signals:
    - schema_migration
requires:
  - fresh_database
gate_condition:
  all_of:
    - fresh_database
`

// p308Server 构造带治理存储的评审服务器(v13NewServer + Governance 接线)。
func p308Server(t *testing.T, st *store.Store, repo string) (*handlers.Server, *httptest.Server) {
	t.Helper()
	s := &handlers.Server{
		Content: &handlers.ContentStore{
			RepoPath: repo,
			Identity: handlers.Identity{Name: "test", Email: "test@example.com"},
		},
		Trace:      st.Queries(),
		Governance: policygovernance.NewStore(st.Pool()),
	}
	r := chi.NewRouter()
	s.RegisterRoutes(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return s, srv
}

// p308SoleProposalID 取治理事件流中唯一提案的 ID(测试面)。
func p308SoleProposalID(t *testing.T, s *handlers.Server) string {
	t.Helper()
	all, err := s.Governance.AllEvents(context.Background())
	if err != nil || len(all) == 0 {
		t.Fatalf("治理事件流应非空: %v len=%d", err, len(all))
	}
	return all[0].ProposalID
}

func TestV13ProposeRuleCreatesGovernanceDraftOnly(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	docsCS, _, _ := v13Seed(t, ctx, st.Queries(), repo, base, docs, code)

	s, srv := p308Server(t, st, repo)
	resp, _ := v13PostForm(t, srv.URL, "/v13/review/"+string(docsCS)+"/propose-rule",
		url.Values{
			"rule_yaml":   {p308RuleYAML},
			"proposed_by": {"owner-a"},
			"rule_note":   {"针对此类迁移缺口"},
		})
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(resp.Header.Get("Location"), "kind=ok") {
		t.Fatalf("propose-rule 应 303 + kind=ok,得到 %d → %s", resp.StatusCode, resp.Header.Get("Location"))
	}

	// 治理面:草稿事件可重建,证据引用指向本案(候选 + 任务)。
	got, err := s.Governance.Reconstruct(ctx, p308SoleProposalID(t, s))
	if err != nil {
		t.Fatalf("Reconstruct: %v", err)
	}
	if got.Status != policygovernance.StatusDraft {
		t.Fatalf("提案应为 DRAFT,得到 %s", got.Status)
	}
	if got.Candidate.Policy().ID != "POL-P308-EXAMPLE" {
		t.Fatalf("提案候选解析: id = %q", got.Candidate.Policy().ID)
	}
	kinds := map[policygovernance.EvidenceKind]bool{}
	for _, e := range got.Evidence {
		kinds[e.Kind] = true
	}
	if !kinds[policygovernance.EvidenceChangeSet] || !kinds[policygovernance.EvidenceTask] {
		t.Fatalf("提案证据应引用本案候选与任务,得到 %v", got.Evidence)
	}

	// AC 负面:提议新规绝不写审批证据(本案决定事实不变)。
	if rows := v13ApprovalsForTask(t, ctx, st.Queries(), "TASK-0001"); len(rows) != 0 {
		t.Fatalf("提议新规不得写审批证据,得到 %d 条", len(rows))
	}
}

func TestV13ProposeRuleNegatives(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	docsCS, _, _ := v13Seed(t, ctx, st.Queries(), repo, base, docs, code)
	s, srv := p308Server(t, st, repo)

	// 不可解析候选 fail-closed:零治理事件、零审批证据。
	tainted := strings.Replace(p308RuleYAML, "enforcement: gate", "enforcement: gate\ngovernance_status: PROPOSED", 1)
	resp, _ := v13PostForm(t, srv.URL, "/v13/review/"+string(docsCS)+"/propose-rule",
		url.Values{"rule_yaml": {tainted}, "proposed_by": {"owner-a"}})
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(resp.Header.Get("Location"), "kind=bad") {
		t.Fatalf("治理字段污染候选应 303 + kind=bad,得到 %d → %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	if _, err := s.Governance.Reconstruct(ctx, "anything"); !errors.Is(err, policygovernance.ErrUnknownProposal) {
		t.Fatalf("拒绝的提议不得留治理事件,得到 %v", err)
	}
	if rows := v13ApprovalsForTask(t, ctx, st.Queries(), "TASK-0001"); len(rows) != 0 {
		t.Fatalf("拒绝的提议不得写审批证据,得到 %d 条", len(rows))
	}

	// 缺提案人 fail-closed。
	resp, _ = v13PostForm(t, srv.URL, "/v13/review/"+string(docsCS)+"/propose-rule",
		url.Values{"rule_yaml": {p308RuleYAML}})
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(resp.Header.Get("Location"), "kind=bad") {
		t.Fatalf("缺提案人应 303 + kind=bad,得到 %d → %s", resp.StatusCode, resp.Header.Get("Location"))
	}
}

// AC 正面(仅此一次):既有 decide 流照旧写审批证据,且绝不产生治理事件 ——
// 一次性决定永不自动改策略。
func TestV13ApplyOnceNeverTouchesPolicy(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	docsCS, _, _ := v13Seed(t, ctx, st.Queries(), repo, base, docs, code)
	s, srv := p308Server(t, st, repo)

	resp, _ := v13PostForm(t, srv.URL, "/v13/review/"+string(docsCS)+"/decide",
		url.Values{"decision": {"approve"}, "reason": {"仅此一次:本案无缺口"}, "decided_by": {"owner"}})
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(resp.Header.Get("Location"), "kind=ok") {
		t.Fatalf("decide 应 303 + kind=ok,得到 %d → %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	if rows := v13ApprovalsForTask(t, ctx, st.Queries(), "TASK-0001"); len(rows) != 1 {
		t.Fatalf("仅此一次决定应写恰好一条审批证据,得到 %d 条", len(rows))
	}
	// 治理面零事件:决定绝不自动创建/修改策略。
	all, err := s.Governance.AllEvents(ctx)
	if err != nil {
		t.Fatalf("AllEvents: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("仅此一次决定不得产生治理事件,得到 %d 条", len(all))
	}
}

// 治理存储未接入:提议新规显式失败(不静默假装成功);仅此一次决定不受影响。
func TestV13ProposeRuleUnavailableWithoutGovernance(t *testing.T) {
	ctx := context.Background()
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	docsCS, _, _ := v13Seed(t, ctx, st.Queries(), repo, base, docs, code)
	srv := v13NewServer(t, st.Queries(), repo)

	resp, _ := v13PostForm(t, srv.URL, "/v13/review/"+string(docsCS)+"/propose-rule",
		url.Values{"rule_yaml": {p308RuleYAML}, "proposed_by": {"owner-a"}})
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(resp.Header.Get("Location"), "kind=bad") {
		t.Fatalf("治理未接入应 303 + kind=bad,得到 %d → %s", resp.StatusCode, resp.Header.Get("Location"))
	}

	resp, _ = v13PostForm(t, srv.URL, "/v13/review/"+string(docsCS)+"/decide",
		url.Values{"decision": {"reject"}, "reason": {"暂不批准"}, "decided_by": {"owner"}})
	if resp.StatusCode != http.StatusSeeOther || !strings.Contains(resp.Header.Get("Location"), "kind=ok") {
		t.Fatalf("治理未接入时 decide 不受影响,得到 %d → %s", resp.StatusCode, resp.Header.Get("Location"))
	}
}
