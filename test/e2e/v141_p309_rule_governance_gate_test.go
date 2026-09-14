//go:build integration

// P3-09 — Rule-governance gate(任务合同:证明规则治理可以 评估 → 批准 →
// 激活 而不产生 split-brain 策略权威;P3 lane 的 release gate)。
//
// 被测链(全部在真实临时内容仓库 + 真实治理事件库上,生产装配原样):
//
//	提案      policygovernance.NewCandidate(出生正典校验)+ draft/submit 事件
//	重放      ReplayCandidate(证据边界:录得键 → would_have_passed;
//	            未录得键 → insufficient_evidence 一等,DL-10)
//	人本批准  (*Store).Activate(P3-07):gitwriter 期望基线 CAS 提交正典 YAML
//	            → artifacts.LoadPolicies 从确切新提交重载验证 → activate 事件
//	重载      LoadPolicies(HEAD) 观察到策略(AC1:Git 快照保持运行时权威)
//
// 分裂脑刻意注入(AC1 的对立面验证):给第二个提案只落 activate 治理事件、
// 不做 Git 提交 —— 治理面记 ACTIVE 而 HEAD 快照不可观察;运行时权威必须
// 跟随 Git(HEAD 无此策略),治理面不得成为第二权威(DL-09/BI-01)。
//
// 陈旧激活(AC4):基于过期基线的 Activate 被类型化拒绝,治理面保持
// PROPOSED,HEAD 不动。
//
// 治理字段(AC3):携带 governance_status 等治理词汇的策略 YAML 在运行时
// 装载面被整体拒绝(LoadPolicies → Rejected),权威面永不采纳。
package e2e

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"tally/internal/artifacts"
	"tally/internal/engineeringpolicy"
	"tally/internal/gitwriter"
	"tally/internal/policygovernance"
	"tally/internal/risk"
	"tally/internal/store"
)

const p309GateYAML = `id: POL-P309-GATE
enforcement: gate
rationale: P3-09 门禁:迁移类变更必须验证升级路径与数据完整性
trigger:
  risk_signals:
    - schema_migration
requires:
  - fresh_database
  - n1_upgrade
gate_condition:
  all_of:
    - fresh_database
    - n1_upgrade
`

func p309GateSetup(t *testing.T) (gov *policygovernance.Store, w *gitwriter.Writer, repoPath, expectedBase string) {
	t.Helper()
	base := e2eDBEnvOrDefault()
	if !e2eDBReachable(base) {
		t.Skipf("P3-09 门禁需要 PostgreSQL(%s 不可达),跳过", base)
	}
	t.Setenv("DATABASE_URL", base)
	st := store.TestDB(t)
	repoPath, expectedBase = p09Repo(t)
	gov = policygovernance.NewStore(st.Pool())
	w, err := gitwriter.New(repoPath)
	if err != nil {
		t.Fatalf("gitwriter.New: %v", err)
	}
	return gov, w, repoPath, expectedBase
}

func e2eDBEnvOrDefault() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	return clDSN
}

// p309MigrationRisk 构造录得触发输入(schema_migration,与候选触发轴同源;
// risk.Derive 是运行时同一派生入口)。
func p309MigrationRisk(t *testing.T) engineeringpolicy.MatchInput {
	t.Helper()
	res, err := risk.Derive([]risk.Signal{{ID: risk.SignalSchemaMigration, Basis: []string{"intent:t1"}}})
	if err != nil {
		t.Fatalf("risk.Derive: %v", err)
	}
	return engineeringpolicy.MatchInput{Risk: res}
}

// p309SeedProposal 落 draft+submit(候选出生即正典校验;提交必须携带证据)。
func p309SeedProposal(t *testing.T, gov *policygovernance.Store, id, yamlText string) policygovernance.Proposal {
	t.Helper()
	ctx := context.Background()
	cand, err := policygovernance.NewCandidate([]byte(yamlText))
	if err != nil {
		t.Fatalf("NewCandidate(%s): %v", id, err)
	}
	ev := policygovernance.Event{
		ProposalID:        id,
		Kind:              policygovernance.EventDraft,
		CandidateRevision: cand.Revision(),
		CandidateYAML:     yamlText,
		ProposedBy:        "owner-a",
		OccurredAt:        time.Now().Add(-time.Hour).UTC(),
	}
	if _, err := gov.Append(ctx, ev); err != nil {
		t.Fatalf("Append(draft %s): %v", id, err)
	}
	sub := ev
	sub.Kind = policygovernance.EventSubmit
	sub.OccurredAt = time.Now().Add(-30 * time.Minute).UTC()
	sub.Evidence = []policygovernance.EvidenceRef{{Kind: policygovernance.EvidenceOutcome, Ref: "out-p309-1"}}
	if _, err := gov.Append(ctx, sub); err != nil {
		t.Fatalf("Append(submit %s): %v", id, err)
	}
	p, err := gov.Reconstruct(ctx, id)
	if err != nil {
		t.Fatalf("Reconstruct(%s): %v", id, err)
	}
	return p
}

func TestP309RuleGovernanceGate(t *testing.T) {
	gov, w, repoPath, expectedBase := p309GateSetup(t)
	ctx := context.Background()
	cand, err := policygovernance.NewCandidate([]byte(p309GateYAML))
	if err != nil {
		t.Fatalf("NewCandidate: %v", err)
	}

	// ---- 提案 + 重放(证据边界) ----
	p := p309SeedProposal(t, gov, "prop-p309-gate", p309GateYAML)
	if p.Status != policygovernance.StatusProposed {
		t.Fatalf("提案状态 = %s, want PROPOSED", p.Status)
	}
	// 录得的门禁结局:验证绑定结果按 gate 键投影(fresh_database 录得 passed;
	// n1_upgrade 未录得 → 证据不足一等)。
	replay, err := policygovernance.ReplayCandidate(policygovernance.ReplayInput{
		Candidate:       p.Candidate.Policy(),
		TriggerRecorded: true,
		Match:           p309MigrationRisk(t),
		RecordedGateResults: map[string]string{
			"fresh_database": "passed",
		},
	})
	if err != nil {
		t.Fatalf("ReplayCandidate: %v", err)
	}
	if replay.Decision != policygovernance.ReplayMatched {
		t.Fatalf("重放判定 = %s, want matched", replay.Decision)
	}
	insufficient := 0
	for _, o := range replay.GateOutcomes {
		if o.Status == policygovernance.GateInsufficientEvidence {
			insufficient++ // n1_upgrade 无录得:insufficient_evidence 必须一等可见(AC2)
		}
	}
	if insufficient == 0 {
		t.Fatalf("重放应含 insufficient_evidence 结局(AC2),得到 %+v", replay.GateOutcomes)
	}

	// ---- 人本批准 → GitWriter 提交 → 重载(AC1:Git 快照保持运行时权威) ----
	res, err := gov.Activate(ctx, policygovernance.ActivationRequest{
		ProposalID:   p.ID,
		Approver:     "owner-b",
		ExpectedBase: expectedBase,
		RepoPath:     repoPath,
	}, w)
	if err != nil {
		t.Fatalf("Activate: %v", err)
	}
	headSnap, err := artifacts.LoadPolicies(repoPath, "HEAD")
	if err != nil {
		t.Fatalf("LoadPolicies(HEAD): %v", err)
	}
	observed := false
	for _, pa := range headSnap.Policies {
		if pa.Policy.ID == "POL-P309-GATE" && pa.Policy.Revision == cand.Revision() {
			observed = true
		}
	}
	if !observed {
		t.Fatalf("AC1:激活后 HEAD 快照必须可观察同 id 同修订策略(HEAD=%s)", res.Commit)
	}
	got, err := gov.Reconstruct(ctx, p.ID)
	if err != nil || got.Status != policygovernance.StatusActive || got.ApprovedBy != "owner-b" {
		t.Fatalf("治理面: %v status=%s", err, got.Status)
	}

	// ---- 分裂脑注入:第二个提案只落治理 ACTIVE 事件、无 Git 提交 ----
	dual := p309SeedProposal(t, gov, "prop-p309-split", strings.Replace(p309GateYAML, "POL-P309-GATE", "POL-P309-SPLIT", 1))
	if _, err := gov.Append(ctx, policygovernance.Event{
		ProposalID:        dual.ID,
		Kind:              policygovernance.EventActivate,
		CandidateRevision: dual.Candidate.Revision(),
		CandidateYAML:     string(dual.Candidate.Raw()),
		ProposedBy:        "owner-a",
		ApprovedBy:        "owner-rogue",
		OccurredAt:        time.Now().UTC(),
	}); err != nil {
		t.Fatalf("Append(rogue activate): %v", err)
	}
	dualState, err := gov.Reconstruct(ctx, dual.ID)
	if err != nil || dualState.Status != policygovernance.StatusActive {
		t.Fatalf("治理面应记录 ACTIVE(分裂脑夹具): %v", err)
	}
	// AC1:运行时权威跟随 Git —— HEAD 快照不得观察到未提交的策略。
	headSnap2, err := artifacts.LoadPolicies(repoPath, "HEAD")
	if err != nil {
		t.Fatalf("LoadPolicies(HEAD) after split: %v", err)
	}
	for _, pa := range headSnap2.Policies {
		if pa.Policy.ID == "POL-P309-SPLIT" {
			t.Fatalf("AC1 破坏:仅治理面 ACTIVE 的策略出现在 HEAD 快照(分裂脑)")
		}
	}

	// ---- 陈旧激活(AC4):第三个提案 + 过期基线 ----
	stale := p309SeedProposal(t, gov, "prop-p309-stale", strings.Replace(p309GateYAML, "POL-P309-GATE", "POL-P309-STALE", 1))
	_, err = gov.Activate(ctx, policygovernance.ActivationRequest{
		ProposalID:   stale.ID,
		Approver:     "owner-b",
		ExpectedBase: expectedBase, // HEAD 已因第一次激活推进 → 过期
		RepoPath:     repoPath,
	}, w)
	if err == nil {
		t.Fatalf("AC4:陈旧基线激活必须被拒绝")
	}
	staleState, rerr := gov.Reconstruct(ctx, stale.ID)
	if rerr != nil || staleState.Status != policygovernance.StatusProposed {
		t.Fatalf("陈旧激活后治理面必须保持 PROPOSED: %v status=%s", rerr, staleState.Status)
	}

	// ---- AC3:治理词汇混入策略 YAML → 运行时装载面整体拒绝 ----
	tainted := strings.Replace(p309GateYAML, "rationale: P3-09 门禁:迁移类变更必须验证升级路径与数据完整性",
		"rationale: tainted\ngovernance_status: ACTIVE", 1)
	if _, err := w.Commit(ctx, gitwriter.CommitRequest{
		ExpectedBase: headSnap2.SourceCommit,
		Message:      "tally: tainted policy fixture",
		Files:        []gitwriter.FileUpdate{{Path: ".tally/policies/pol-p309-tainted.yaml", Content: []byte(tainted)}},
	}); err != nil {
		t.Fatalf("投毒提交: %v", err)
	}
	taintedSnap, err := artifacts.LoadPolicies(repoPath, "HEAD")
	if err != nil {
		t.Fatalf("LoadPolicies(tainted): %v", err)
	}
	gateStillActive := false
	for _, pa := range taintedSnap.Policies {
		if pa.Policy.ID == "POL-P309-GATE" {
			gateStillActive = true
		}
		if strings.Contains(strings.ToLower(pa.Path), "tainted") {
			t.Fatalf("AC3:治理字段污染的制品不得进入生效集: %s", pa.Path)
		}
	}
	if !gateStillActive {
		t.Fatalf("AC3:合法策略应仍在生效集")
	}
	rejected := false
	for _, rj := range taintedSnap.Rejected {
		if strings.Contains(strings.ToLower(rj.Path), "tainted") {
			rejected = true
		}
	}
	if !rejected {
		t.Fatalf("AC3:治理字段污染的制品应进入 Rejected(运行时权威永不采纳),得到 %+v", taintedSnap.Rejected)
	}
}
