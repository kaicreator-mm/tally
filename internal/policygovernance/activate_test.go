// activate_test.go 拥有激活链的集成测试(P3-07 验证面:success / stale base /
// invalid candidate / 不可观察 fail-safe):真实临时内容仓库(p09 同款生产引导)
// + 真实治理事件库(每测试独立 schema),经唯一 Git 写边界(gitwriter)提交并
// 从确切新源提交重载验证(AC:提案只有在其承诺策略可从源快照观察后才算运行
// 时生效)。
package policygovernance

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"

	"tally/internal/artifacts"
	"tally/internal/gitwriter"
)

const activatePolicyYAML = `id: POL-ACTIVATE-TEST
enforcement: gate
rationale: P3-07 激活链测试
trigger:
  risk_signals:
    - schema_migration
requires:
  - fresh_database
gate_condition:
  all_of:
    - fresh_database
`

// activateFixture 建真实临时内容仓库(PlainInit + gitwriter 空基线首提交)
// + 治理存储 + 已提交评审的提案(draft → submit,证据齐备),返回期望基线与
// 写边界。LoadPolicies 只裁决 .tally/policies 制品面,无需项目注册表。
func activateFixture(t *testing.T) (s *Store, w *gitwriter.Writer, repoPath, expectedBase, proposalID string) {
	t.Helper()
	dir := t.TempDir()
	if _, err := git.PlainInit(dir, false); err != nil {
		t.Fatalf("PlainInit: %v", err)
	}
	w, err := gitwriter.New(dir)
	if err != nil {
		t.Fatalf("gitwriter.New: %v", err)
	}
	base, err := w.Commit(context.Background(), gitwriter.CommitRequest{
		Message: "tally: policy repo bootstrap",
		Files:   []gitwriter.FileUpdate{{Path: "README.md", Content: []byte("policy fixture repo")}},
	})
	if err != nil {
		t.Fatalf("首提交(空基线): %v", err)
	}
	s, _ = testGovStore(t)

	cand, err := NewCandidate([]byte(activatePolicyYAML))
	if err != nil {
		t.Fatalf("NewCandidate: %v", err)
	}
	proposalID = "prop-act-001"
	ev := Event{
		ProposalID:        proposalID,
		Kind:              EventDraft,
		CandidateRevision: cand.Revision(),
		CandidateYAML:     string(cand.Raw()),
		ProposedBy:        "owner-a",
		OccurredAt:        time.Now().Add(-time.Hour).UTC(),
	}
	if _, err := s.Append(context.Background(), ev); err != nil {
		t.Fatalf("Append(draft): %v", err)
	}
	sub := ev
	sub.Kind = EventSubmit
	sub.OccurredAt = time.Now().Add(-30 * time.Minute).UTC()
	sub.Evidence = []EvidenceRef{mustRef(t, EvidenceOutcome, "out-act-1")}
	if _, err := s.Append(context.Background(), sub); err != nil {
		t.Fatalf("Append(submit): %v", err)
	}
	return s, w, dir, base.Commit, proposalID
}

// 成功链:提交 → 从确切新提交重载观察到同 id 同修订 → 治理记录 ACTIVE。
func TestActivateSuccess(t *testing.T) {
	s, w, repoPath, base, proposalID := activateFixture(t)
	ctx := context.Background()
	wantRevision := candRevisionOf(t, activatePolicyYAML)

	res, err := s.Activate(ctx, ActivationRequest{
		ProposalID:   proposalID,
		Approver:     "owner-b",
		ExpectedBase: base,
		RepoPath:     repoPath,
	}, w)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	if res.Commit == "" || !strings.HasPrefix(res.PolicyPath, ".tally/policies/") {
		t.Fatalf("激活产物不完整: commit=%q path=%q", res.Commit, res.PolicyPath)
	}
	// AC:承诺策略可从确切新源提交观察(独立经 LoadPolicies 复核)。
	snap, err := artifacts.LoadPolicies(repoPath, res.Commit)
	if err != nil {
		t.Fatalf("LoadPolicies: %v", err)
	}
	found := false
	for _, pa := range snap.Policies {
		if pa.Policy.ID == "POL-ACTIVATE-TEST" && pa.Policy.Revision == wantRevision {
			found = true
		}
	}
	if !found {
		t.Fatalf("新提交快照未观察到激活策略(修订 %s): %+v", wantRevision, snap.Policies)
	}
	// 治理面:重放重建 ACTIVE + 批准人。
	got, err := s.Reconstruct(ctx, proposalID)
	if err != nil || got.Status != StatusActive || got.ApprovedBy != "owner-b" {
		t.Fatalf("Reconstruct: %v status=%s approvedBy=%q", err, got.Status, got.ApprovedBy)
	}
}

// 陈旧基线 fail-safe:提交被 CAS 拒绝 → 类型化失败、治理面保持 PROPOSED、
// HEAD 不动(零部分写入、零激活事件)。
func TestActivateStaleBase(t *testing.T) {
	s, w, repoPath, base, proposalID := activateFixture(t)
	ctx := context.Background()

	adv, err := w.Commit(ctx, gitwriter.CommitRequest{
		ExpectedBase: base,
		Message:      "tally: unrelated advance",
		Files:        []gitwriter.FileUpdate{{Path: "docs/other.md", Content: []byte("x")}},
	})
	if err != nil {
		t.Fatalf("推进 HEAD: %v", err)
	}
	_, err = s.Activate(ctx, ActivationRequest{
		ProposalID:   proposalID,
		Approver:     "owner-b",
		ExpectedBase: base,
		RepoPath:     repoPath,
	}, w)
	if err == nil || !strings.Contains(err.Error(), "保持 PROPOSED") {
		t.Fatalf("陈旧基线应类型化失败且保持 PROPOSED,得到 %v", err)
	}
	got, rerr := s.Reconstruct(ctx, proposalID)
	if rerr != nil || got.Status != StatusProposed {
		t.Fatalf("治理面必须保持 PROPOSED: %v status=%s", rerr, got.Status)
	}
	w2, err := gitwriter.New(repoPath)
	if err != nil {
		t.Fatalf("gitwriter.New(复核): %v", err)
	}
	head, herr := w2.Head(ctx)
	if herr != nil || head != adv.Commit {
		t.Fatalf("HEAD 应停在推进提交 %s,得到 %s(%v)", adv.Commit, head, herr)
	}
}

// 前置 fail-safe:批准人必填;DRAFT(未提交评审)不可激活;未知提案类型化拒绝。
func TestActivatePreflightNegatives(t *testing.T) {
	s, w, repoPath, _, proposalID := activateFixture(t)
	ctx := context.Background()

	if _, err := s.Activate(ctx, ActivationRequest{ProposalID: proposalID, RepoPath: repoPath}, w); err == nil {
		t.Fatal("缺批准人应拒绝")
	}
	cand2, err := NewCandidate([]byte(strings.Replace(activatePolicyYAML, "POL-ACTIVATE-TEST", "POL-ACTIVATE-DRAFT", 1)))
	if err != nil {
		t.Fatalf("NewCandidate: %v", err)
	}
	if _, err := s.Append(ctx, Event{
		ProposalID:        "prop-act-draft",
		Kind:              EventDraft,
		CandidateRevision: cand2.Revision(),
		CandidateYAML:     string(cand2.Raw()),
		ProposedBy:        "owner-a",
		OccurredAt:        time.Now().UTC(),
	}); err != nil {
		t.Fatalf("Append(draft2): %v", err)
	}
	if _, err := s.Activate(ctx, ActivationRequest{
		ProposalID: "prop-act-draft", Approver: "owner-b", ExpectedBase: "0", RepoPath: repoPath,
	}, w); err == nil || !strings.Contains(err.Error(), "PROPOSED") {
		t.Fatalf("DRAFT 提案应拒绝激活并指明状态,得到 %v", err)
	}
	if _, err := s.Activate(ctx, ActivationRequest{
		ProposalID: "prop-never", Approver: "owner-b", ExpectedBase: "0", RepoPath: repoPath,
	}, w); !errors.Is(err, ErrInvalidActivation) {
		t.Fatalf("未知提案应 ErrInvalidActivation,得到 %v", err)
	}
}

// candRevisionOf 独立解析出内容修订(与被测代码同语义,交叉核对)。
func candRevisionOf(t *testing.T, raw string) string {
	t.Helper()
	c, err := NewCandidate([]byte(raw))
	if err != nil {
		t.Fatalf("NewCandidate: %v", err)
	}
	return c.Revision()
}
