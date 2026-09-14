// v14_rules_test.go 拥有 Rules 投影页的处理器测试(P3-06 验证面:治理与 Git
// 分歧的夹具)。核心 AC:UI 不得把仅存在于 DB 的提案显示为 Active;分歧与
// 不确定性一等可见。
package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"tally/internal/gitwriter"
	"tally/internal/policygovernance"
	"tally/internal/store"
	"tally/internal/web/handlers"
)

const p306RuleYAML = `id: POL-P306-EXAMPLE
enforcement: gate
rationale: P3-06 Rules 投影测试
trigger:
  risk_signals:
    - schema_migration
requires:
  - fresh_database
gate_condition:
  all_of:
    - fresh_database
`

// p306Server 构造带治理存储的 v14 服务器(v14NewServer + Governance 接线)。
func p306Server(t *testing.T, st *store.Store, repo string) (*httptest.Server, *policygovernance.Store) {
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
	s.RegisterV14Routes(r)
	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv, s.Governance
}

// p306SeedProposal 落一条治理提案(draft → submit)并返回重建值。
func p306SeedProposal(t *testing.T, gov *policygovernance.Store, id, yamlText string) policygovernance.Proposal {
	t.Helper()
	cand, err := policygovernance.NewCandidate([]byte(yamlText))
	if err != nil {
		t.Fatalf("NewCandidate: %v", err)
	}
	ctx := context.Background()
	ev := policygovernance.Event{
		ProposalID:        id,
		Kind:              policygovernance.EventDraft,
		CandidateRevision: cand.Revision(),
		CandidateYAML:     yamlText,
		ProposedBy:        "owner-a",
		OccurredAt:        time.Now().Add(-time.Hour).UTC(),
	}
	if _, err := gov.Append(ctx, ev); err != nil {
		t.Fatalf("Append(draft): %v", err)
	}
	sub := ev
	sub.Kind = policygovernance.EventSubmit
	sub.OccurredAt = time.Now().Add(-30 * time.Minute).UTC()
	sub.Evidence = []policygovernance.EvidenceRef{{Kind: policygovernance.EvidenceTask, Ref: "TASK-0001"}}
	if _, err := gov.Append(ctx, sub); err != nil {
		t.Fatalf("Append(submit): %v", err)
	}
	p, err := gov.Reconstruct(ctx, id)
	if err != nil {
		t.Fatalf("Reconstruct: %v", err)
	}
	return p
}

// p306CommitPolicy 把候选 YAML 经唯一写边界提交进 Git(基线 = 当前 HEAD)。
func p306CommitPolicy(t *testing.T, repo, yamlText string) {
	t.Helper()
	ctx := context.Background()
	w, err := gitwriter.New(repo)
	if err != nil {
		t.Fatalf("gitwriter.New: %v", err)
	}
	head, err := w.Head(ctx)
	if err != nil {
		t.Fatalf("Head: %v", err)
	}
	if _, err := w.Commit(ctx, gitwriter.CommitRequest{
		ExpectedBase: head,
		Message:      "tally: commit policy fixture",
		Files:        []gitwriter.FileUpdate{{Path: ".tally/policies/pol-p306-example.yaml", Content: []byte(yamlText)}},
	}); err != nil {
		t.Fatalf("Commit: %v", err)
	}
}

// govAppendActivate 在治理面补 activate 事件(夹具面;真实链走 P3-07 服务)。
func govAppendActivate(t *testing.T, gov *policygovernance.Store, p policygovernance.Proposal, approver string) {
	t.Helper()
	if _, err := gov.Append(context.Background(), policygovernance.Event{
		ProposalID:        p.ID,
		Kind:              policygovernance.EventActivate,
		CandidateRevision: p.Candidate.Revision(),
		CandidateYAML:     string(p.Candidate.Raw()),
		ProposedBy:        p.ProposedBy,
		ApprovedBy:        approver,
		OccurredAt:        time.Now().UTC(),
	}); err != nil {
		t.Fatalf("Append(activate): %v", err)
	}
}

// 夹具 A(治理领先):PROPOSED 提案存在、Git 无策略 → 出现在治理提案区,
// Active 区为空,且该策略绝不显示为 Active(AC)。
func TestV14RulesProposedNeverShownAsActive(t *testing.T) {
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	v13Seed(t, context.Background(), st.Queries(), repo, base, docs, code)
	srv, gov := p306Server(t, st, repo)
	p306SeedProposal(t, gov, "prop-p306-a", p306RuleYAML)

	_, body := v13Get(t, srv.URL, "/v14/rules")
	if !strings.Contains(body, "prop-p306-a") {
		t.Errorf("Proposed 区应呈现提案 prop-p306-a")
	}
	if !strings.Contains(body, "HEAD 快照中没有任何生效策略") {
		t.Errorf("Active 区应为空(HEAD 无策略):\n%s", body)
	}
	// AC:DB-only 提案绝不显示为 Active(其策略 id 只出现在治理提案区一次)。
	if got := strings.Count(body, "POL-P306-EXAMPLE"); got != 1 {
		t.Errorf("POL-P306-EXAMPLE 只应出现在治理提案区一次,得到 %d 次:\n%s", got, body)
	}
}

// 夹具 B(分歧):治理面 ACTIVE 但 HEAD 不可观察 → 分歧区显式呈现 + 运行时
// 不受影响;Active 区为空(AC)。
func TestV14RulesDivergedActiveNotShownAsActive(t *testing.T) {
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	v13Seed(t, context.Background(), st.Queries(), repo, base, docs, code)
	srv, gov := p306Server(t, st, repo)
	p := p306SeedProposal(t, gov, "prop-p306-b", p306RuleYAML)
	govAppendActivate(t, gov, p, "owner-b") // 只落治理事件,不做 Git 提交 —— 面间分歧

	_, body := v13Get(t, srv.URL, "/v14/rules")
	if !strings.Contains(body, "治理分歧(治理面 ACTIVE 但 HEAD 不可观察)") {
		t.Errorf("分歧区应呈现:\n%s", body)
	}
	if !strings.Contains(body, "HEAD 快照中不存在该 id 的策略") || !strings.Contains(body, "运行时不受其影响") {
		t.Errorf("分歧原因与 DL-09 注记应如实呈现:\n%s", body)
	}
	// AC:分歧提案绝不进入 Active 列表(Git 面不变 → Active 区仍为空)。
	if !strings.Contains(body, "HEAD 快照中没有任何生效策略") {
		t.Errorf("Active 区应为空(分歧提案不改变 Git 面):\n%s", body)
	}
}

// 夹具 C(一致):Git 提交 + 治理 ACTIVE 同修订 → Active 行带一致性标注,
// 无分歧行。
func TestV14RulesMatchedActive(t *testing.T) {
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	v13Seed(t, context.Background(), st.Queries(), repo, base, docs, code)
	srv, gov := p306Server(t, st, repo)
	p306CommitPolicy(t, repo, p306RuleYAML)
	p := p306SeedProposal(t, gov, "prop-p306-c", p306RuleYAML)
	govAppendActivate(t, gov, p, "owner-b")

	_, body := v13Get(t, srv.URL, "/v14/rules")
	if !strings.Contains(body, "POL-P306-EXAMPLE") || !strings.Contains(body, "治理提案 prop-p306-c 一致(ACTIVE)") {
		t.Errorf("Active 行应呈现并带治理一致性标注:\n%s", body)
	}
	if strings.Contains(body, "治理分歧(治理面 ACTIVE 但 HEAD 不可观察)") {
		t.Errorf("一致场景不得出现分歧区:\n%s", body)
	}
}

// 夹具 D(不确定性):Git 有策略但治理面无记录 → Active 行携带不确定标注,
// 且 Git 权威面照常呈现。
func TestV14RulesUncertaintyVisible(t *testing.T) {
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	v13Seed(t, context.Background(), st.Queries(), repo, base, docs, code)
	srv, _ := p306Server(t, st, repo)
	p306CommitPolicy(t, repo, p306RuleYAML)

	_, body := v13Get(t, srv.URL, "/v14/rules")
	if !strings.Contains(body, "不确定:治理面无对应 ACTIVE 提案记录") {
		t.Errorf("治理面无记录的策略应携带不确定标注(一等可见):\n%s", body)
	}
	if !strings.Contains(body, "POL-P306-EXAMPLE") {
		t.Errorf("Git 面策略应照常呈现在 Active 区(权威面不因记录面缺席而缺席)")
	}
}

// 治理未接入:Active 照常(权威面),记录面按未接入诚实呈现。
func TestV14RulesWithoutGovernance(t *testing.T) {
	st := v13TestDB(t)
	repo, base, docs, code := v13Repo(t)
	v13Seed(t, context.Background(), st.Queries(), repo, base, docs, code)
	p306CommitPolicy(t, repo, p306RuleYAML)

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

	status, body := v13Get(t, srv.URL, "/v14/rules")
	if status != http.StatusOK {
		t.Fatalf("GET /v14/rules = %d", status)
	}
	if !strings.Contains(body, "治理存储未接入") {
		t.Errorf("治理未接入应诚实呈现")
	}
	if !strings.Contains(body, "POL-P306-EXAMPLE") {
		t.Errorf("治理未接入时 Active 照常呈现(权威面不因记录面缺席而缺席)")
	}
}
