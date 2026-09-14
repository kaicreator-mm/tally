// v1.3 权威模型不变量测试(P1-45,PRD §3.2 稳定 ID)。
//
// 覆盖 P1 激活的权威面不变量:
//   - INV-AUTH-01 · P1 — Agent Claim is never Project Fact;
//   - INV-AUTH-02 · P1 — LLM output is Proposal unless independently
//     validated or explicitly approved。
//
// 按 PRD §3.2「对应领域未激活时不要求提前实现无主语的测试」:P2/P5 治理类
// 不变量(INV-GOV-01/02)与 P2 视图不变量(INV-VIEW-01)不在本文件内,
// 也不为其伪造任何无主语断言(任务 AC2)。
package arch

import (
	"errors"
	"testing"
	"time"

	"tally/internal/core"
	"tally/internal/evidence"
	"tally/internal/taskstate"
)

// authAt 是固定时钟:断言不依赖真实时间。
var authAt = time.Date(2026, 9, 6, 8, 0, 0, 0, time.UTC)

// authScope 返回一个结构完整的证据 scope(五维全部非空)。
func authScope() evidence.Scope {
	return evidence.Scope{
		Project:      core.ProjectID("proj-arch"),
		Task:         evidence.TaskID("TASK-ARCH"),
		Acceptance:   evidence.AcceptanceID("AC-01"),
		Verification: evidence.VerificationRevision("sha256:contract-1"),
		ChangeSet:    evidence.ChangeSetID("sha256:cs-1"),
	}
}

// authRecorder 构造带具体 actor 的记录 provenance。
func authRecorder(kind evidence.RecorderKind, actor string) evidence.Recorder {
	return evidence.Recorder{Kind: kind, Actor: actor}
}

// INV-AUTH-01:实现权威通道(Agent)与一切未知通道都不能确认机器事实,
// 也不能做裁定 —— 保守默认是拒绝。
func TestV13AgentRecorderCannotRecordFacts(t *testing.T) {
	if evidence.RecorderAgent.CanRecordFacts() {
		t.Fatal("implementation_agent 通道可以确认机器事实(INV-AUTH-01:Agent Claim is never Project Fact)")
	}
	if evidence.RecorderKind("mystery-channel").CanRecordFacts() {
		t.Fatal("未知记录通道被允许确认事实(必须保守默认拒绝,INV-AUTH-01)")
	}

	agent := authRecorder(evidence.RecorderAgent, "coding-agent")
	if _, err := evidence.NewFact("fact-agent-1", authScope(), evidence.VerdictPass, authAt, agent); !errors.Is(err, evidence.ErrRecorderCannotRecordFact) {
		t.Fatalf("Agent 通道直接记录事实: err = %v, want ErrRecorderCannotRecordFact(INV-AUTH-01)", err)
	}
	// 裁定是事实面操作:Agent 同样不能借助裁定改写证据有效性。
	if _, err := evidence.NewRuling(evidence.RulingInvalidEvidence, "self-declared invalid", authAt, agent); !errors.Is(err, evidence.ErrRecorderCannotRecordFact) {
		t.Fatalf("Agent 通道做裁定: err = %v, want ErrRecorderCannotRecordFact(INV-AUTH-01)", err)
	}
}

// INV-AUTH-02:LLM/Agent 输出只是 Proposal;Confirm 是 proposal → fact 的
// 唯一路径,且必须经 fact-capable 的独立记录者。事实的 provenance 是那个
// 独立记录者,绝不是 Agent 自己。
func TestV13ProposalBecomesFactOnlyViaIndependentRecorder(t *testing.T) {
	proposal := evidence.Proposal{
		Summary:        "agent claims tests pass",
		Scope:          authScope(),
		ClaimedVerdict: evidence.VerdictPass,
		ProducedBy:     evidence.RecorderAgent,
	}
	agent := authRecorder(evidence.RecorderAgent, "coding-agent")
	if _, err := evidence.Confirm("fact-confirm-1", proposal, authAt, agent); !errors.Is(err, evidence.ErrRecorderCannotRecordFact) {
		t.Fatalf("Agent 经 Confirm 自我确认 proposal: err = %v, want ErrRecorderCannotRecordFact(INV-AUTH-02)", err)
	}

	validator := authRecorder(evidence.RecorderValidator, "tally/validator")
	fact, err := evidence.Confirm("fact-confirm-1", proposal, authAt, validator)
	if err != nil {
		t.Fatalf("独立验证者确认合法 proposal 失败: %v", err)
	}
	if fact.Recorder().Kind == evidence.RecorderAgent {
		t.Fatal("确认后事实的 provenance 仍是 Agent 通道(INV-AUTH-02)")
	}
	if fact.Recorder().Kind != evidence.RecorderValidator || fact.Recorder().Actor != "tally/validator" {
		t.Fatalf("事实 provenance = %+v, want 独立 validator 记录者(INV-AUTH-02)", fact.Recorder())
	}
	if fact.Verdict() != proposal.ClaimedVerdict || fact.Scope() != proposal.Scope {
		t.Fatalf("确认保留观察内容失败: verdict=%s scope=%+v", fact.Verdict(), fact.Scope())
	}
}

// INV-AUTH-01/02(状态面):经 Agent 通道“记录”的生命周期事实不参与状态
// 推导 —— Agent 无法用一条 merge/verified 声明直接改写 lifecycle 状态;
// 被拒绝的 claim 列入 IgnoredClaims 可审计,而不是静默丢弃。
func TestV13AgentLifecycleClaimsDoNotDriveState(t *testing.T) {
	valid := authRecorder(evidence.RecorderValidator, "tally")
	facts := []taskstate.Fact{
		{ID: "f-planned", Kind: taskstate.FactTaskPlanned, At: authAt, Recorder: valid, RevisionRef: "sha256:task-rev-1"},
		{ID: "f-run", Kind: taskstate.FactCodegenRunStarted, At: authAt.Add(time.Second), Recorder: valid, RunID: "run-1"},
		{ID: "f-cand", Kind: taskstate.FactChangeSetProduced, At: authAt.Add(2 * time.Second), Recorder: valid, RunID: "run-1", ChangeSet: "sha256:cs-1"},
		// Agent 自称“已合并”:不是事实,推不出 MERGED。
		{ID: "f-merge-claim", Kind: taskstate.FactMergeRecorded, At: authAt.Add(3 * time.Second), Recorder: authRecorder(evidence.RecorderAgent, "coding-agent"), ChangeSet: "sha256:cs-1"},
	}
	res, err := taskstate.Derive(taskstate.Snapshot{Facts: facts})
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if res.State != taskstate.StateCandidate {
		t.Fatalf("Agent merge claim 使状态 = %s, want CANDIDATE(Agent claim 不是事实,推不出 MERGED;INV-AUTH-01)", res.State)
	}
	found := false
	for _, c := range res.IgnoredClaims {
		if c.Ref == "fact:f-merge-claim" && c.Reason == taskstate.ClaimReasonRecorderNotFactCapable {
			found = true
		}
	}
	if !found {
		t.Fatalf("被拒绝的 Agent claim 未进入 IgnoredClaims: %+v(INV-AUTH-01/02 必须可审计)", res.IgnoredClaims)
	}
}

// INV-AUTH-02(审批面):批准是 Gate Authority 的人类决定;Agent 通道的
// “批准”不是事实,推不出 APPROVED,只能作为被拒绝的 claim 留痕。
func TestV13AgentApprovalIsNotGateFact(t *testing.T) {
	facts := []taskstate.Fact{
		{ID: "f-planned", Kind: taskstate.FactTaskPlanned, At: authAt, Recorder: authRecorder(evidence.RecorderValidator, "tally"), RevisionRef: "sha256:task-rev-1"},
	}
	snap := taskstate.Snapshot{
		Facts: facts,
		Approvals: []taskstate.ApprovalEvidence{{
			ID: "appr-agent-1", Decision: taskstate.DecisionApprove,
			ChangeSet: "sha256:cs-1", Recorder: authRecorder(evidence.RecorderAgent, "coding-agent"), At: authAt,
		}},
	}
	res, err := taskstate.Derive(snap)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	if res.State == taskstate.StateApproved {
		t.Fatal("Agent 通道的批准证据推导出 APPROVED(INV-AUTH-02:LLM 输出未经显式人类批准不是事实)")
	}
	if res.State != taskstate.StatePlanned {
		t.Fatalf("状态 = %s, want PLANNED(唯一合法事实只有 task_planned)", res.State)
	}
	found := false
	for _, c := range res.IgnoredClaims {
		if c.Ref == "approval:appr-agent-1" && c.Reason == taskstate.ClaimReasonApprovalNotHuman {
			found = true
		}
	}
	if !found {
		t.Fatalf("Agent 批准未进入 IgnoredClaims: %+v(INV-AUTH-02 必须可审计)", res.IgnoredClaims)
	}
}

// 结构边界:事实/状态/验证/合同/评审面不得依赖 LLM(Proposal Authority)。
// LLM 只能提案;事实记录、状态推导与验证权威的定义路径上出现 LLM 依赖即
// 意味着 Claim 可以直接塑形 Fact(INV-AUTH-01/02 的结构面)。
func TestV13FactAndVerificationSurfacesDoNotImportLLM(t *testing.T) {
	graph := importGraph(t)
	for _, pkg := range []string{"evidence", "taskstate", "runstore", "verification", "changeset", "review", "taskcontract"} {
		if graph[pkg]["llm"] {
			t.Errorf("internal/%s 导入了 internal/llm:Proposal Authority 不得出现在事实/状态/验证权威的依赖路径上(INV-AUTH-01/02)", pkg)
		}
	}
}
