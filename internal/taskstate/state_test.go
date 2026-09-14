package taskstate

import (
	"errors"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"tally/internal/evidence"
	"tally/internal/verification"
)

// ---- 测试夹具 ---------------------------------------------------------------

var (
	base      = time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	humanR    = evidence.Recorder{Kind: evidence.RecorderHuman, Actor: "owner-1"}
	validator = evidence.Recorder{Kind: evidence.RecorderValidator, Actor: "tally-validator"}
	agentR    = evidence.Recorder{Kind: evidence.RecorderAgent, Actor: "coder-1"}
	external  = evidence.Recorder{Kind: evidence.RecorderExternalSystem, Actor: "ci"}

	cs1 = evidence.ChangeSetID("cs-1")
	cs2 = evidence.ChangeSetID("cs-2")
)

func at(step int) time.Time { return base.Add(time.Duration(step) * time.Hour) }

// frozenContract 冻结一份最小但合法的验证契约(AC-01 + 一条 visible binding)。
func frozenContract(t *testing.T, rev int) *verification.Frozen {
	t.Helper()
	c := &verification.Contract{
		TaskID:   "TASK-023",
		Revision: rev,
		Acceptances: []verification.Acceptance{{
			ID:        "AC-01",
			Statement: "stale leases can be reclaimed",
			EvidenceBindings: []verification.Binding{{
				ID:               "EV-1",
				AuthoringMode:    verification.AuthoringPreCommit,
				VisibilityPolicy: verification.VisibilityVisible,
				Evidence: verification.EvidenceRef{
					Type: verification.EvidenceTest,
					Path: "internal/jobs/lease_test.go",
				},
			}},
		}},
	}
	f, err := c.Freeze()
	if err != nil {
		t.Fatalf("冻结验证契约: %v", err)
	}
	return f
}

func funcReport(f *verification.Frozen, complete bool) *verification.FunctionalCoverageReport {
	return &verification.FunctionalCoverageReport{
		TaskID:           f.TaskID(),
		ContractRevision: f.Revision(),
		ContractHash:     f.Hash(),
		Acceptances: []verification.AcceptanceCoverage{{
			AcceptanceID: "AC-01",
			Covered:      complete,
			SatisfiedBy:  []string{"EV-1"},
		}},
		Complete: complete,
	}
}

func indepReport(f *verification.Frozen, covered bool) *verification.IndependenceCoverageReport {
	return &verification.IndependenceCoverageReport{
		TaskID:           f.TaskID(),
		ContractRevision: f.Revision(),
		ContractHash:     f.Hash(),
		Acceptances: []verification.AcceptanceIndependence{{
			AcceptanceID: "AC-01",
			Required:     verification.IndependenceNone,
			Covered:      covered,
		}},
		Complete: covered,
	}
}

// lifecycleFacts 是 PLANNED→READY→RUNNING→CANDIDATE→VALIDATING 的完整事实链
// (validator 通道,即事实 capable 的控制面通道)。
func lifecycleFacts(contractHash string) []Fact {
	return []Fact{
		{ID: "f-plan", Kind: FactTaskPlanned, At: at(0), Recorder: validator, RevisionRef: "sha256:task-contract"},
		{ID: "f-frz", Kind: FactVerificationFrozen, At: at(1), Recorder: validator, RevisionRef: contractHash},
		{ID: "f-run", Kind: FactCodegenRunStarted, At: at(2), Recorder: validator, RunID: "run-1"},
		{ID: "f-cand", Kind: FactChangeSetProduced, At: at(3), Recorder: validator, RunID: "run-1", ChangeSet: cs1},
		{ID: "f-vrun", Kind: FactVerificationRunStarted, At: at(4), Recorder: validator, RunID: "vrun-1", ChangeSet: cs1},
	}
}

func verifiedSnapshot(t *testing.T) Snapshot {
	t.Helper()
	f := frozenContract(t, 1)
	return Snapshot{
		Contract:     f,
		Facts:        lifecycleFacts(f.Hash()),
		Functional:   funcReport(f, true),
		Independence: indepReport(f, true),
	}
}

func approvalFor(cs evidence.ChangeSetID) ApprovalEvidence {
	return ApprovalEvidence{
		ID: "ap-1", Decision: DecisionApprove, ChangeSet: cs,
		Recorder: humanR, At: at(5),
	}
}

func waiverFor(f *verification.Frozen, cs evidence.ChangeSetID, acceptances ...string) WaiverEvidence {
	return WaiverEvidence{
		ID: "wv-1", ContractHash: f.Hash(), ChangeSet: cs,
		Acceptances: acceptances, ReasonCode: "runtime_exposure_accepted",
		Recorder: humanR, At: at(5),
	}
}

// gapFor 返回指定 Requirement 的缺口是否出现。
func hasGap(res Result, req Requirement) bool {
	for _, g := range res.Gaps {
		if g.Requirement == req {
			return true
		}
	}
	return false
}

func hasClaim(res Result, ref, reason string) bool {
	for _, c := range res.IgnoredClaims {
		if c.Ref == ref && c.Reason == reason {
			return true
		}
	}
	return false
}

func mustDerive(t *testing.T, s Snapshot) Result {
	t.Helper()
	res, err := Derive(s)
	if err != nil {
		t.Fatalf("Derive: %v", err)
	}
	return res
}

// ---- 里程碑推进 -------------------------------------------------------------

func TestMilestoneProgressionFromFacts(t *testing.T) {
	f := frozenContract(t, 1)
	hash := f.Hash()

	steps := []struct {
		name  string
		facts []Fact
		want  State
	}{
		{"planned", lifecycleFacts(hash)[:1], StatePlanned},
		{"ready", lifecycleFacts(hash)[:2], StateReady},
		{"running", lifecycleFacts(hash)[:3], StateRunning},
		{"candidate", lifecycleFacts(hash)[:4], StateCandidate},
		{"validating", lifecycleFacts(hash)[:5], StateValidating},
	}
	for _, st := range steps {
		t.Run(st.name, func(t *testing.T) {
			res := mustDerive(t, Snapshot{Contract: f, Facts: st.facts})
			if res.State != st.want {
				t.Fatalf("state = %s, want %s", res.State, st.want)
			}
			// 未达 VERIFIED 时必须能解释为什么(可追溯,INV-STATE-02)。
			if !hasGap(res, ReqFunctionalCoverage) {
				t.Fatalf("缺少功能覆盖缺口说明: %+v", res.Gaps)
			}
		})
	}
}

func TestPlannedRequiresTaskFact(t *testing.T) {
	// 任何状态(包括 READY)都必须有 evidence-traceable 的事实基础:缺少
	// task_planned 创建事实时,推导显式失败而不是猜一个位置(INV-STATE-02)。
	if _, err := Derive(Snapshot{
		Contract: frozenContract(t, 1),
		Facts:    lifecycleFacts("sha256:x")[1:],
	}); !errors.Is(err, ErrNoTaskFact) {
		t.Fatalf("err = %v, want ErrNoTaskFact", err)
	}
	if _, err := Derive(Snapshot{Facts: lifecycleFacts("sha256:x")[1:]}); !errors.Is(err, ErrNoTaskFact) {
		t.Fatalf("err = %v, want ErrNoTaskFact", err)
	}
}

// ---- AC1:Agent claim 不能产生 VERIFIED --------------------------------------

func TestAgentClaimCannotProduceVerified(t *testing.T) {
	f := frozenContract(t, 1)
	// planned 之后的全部生命周期“事实”都由 implementation_agent 通道记录:
	// 它们是 claim 不是 fact,对状态推导不可见。
	facts := []Fact{
		{ID: "f-plan", Kind: FactTaskPlanned, At: at(0), Recorder: validator, RevisionRef: "sha256:task-contract"},
		{ID: "a-frz", Kind: FactVerificationFrozen, At: at(1), Recorder: agentR, RevisionRef: f.Hash()},
		{ID: "a-run", Kind: FactCodegenRunStarted, At: at(2), Recorder: agentR, RunID: "run-1"},
		{ID: "a-cand", Kind: FactChangeSetProduced, At: at(3), Recorder: agentR, RunID: "run-1", ChangeSet: cs1},
		{ID: "a-vrun", Kind: FactVerificationRunStarted, At: at(4), Recorder: agentR, RunID: "vrun-1", ChangeSet: cs1},
	}
	res := mustDerive(t, Snapshot{
		Contract:     f,
		Facts:        facts,
		Functional:   funcReport(f, true),
		Independence: indepReport(f, true),
	})
	if res.State != StatePlanned {
		t.Fatalf("agent claim 产生了状态推进: %s(AC1)", res.State)
	}
	if len(res.IgnoredClaims) != 4 {
		t.Fatalf("ignored claims = %+v, want 4 条", res.IgnoredClaims)
	}
	for _, id := range []string{"fact:a-frz", "fact:a-run", "fact:a-cand", "fact:a-vrun"} {
		if !hasClaim(res, id, ClaimReasonRecorderNotFactCapable) {
			t.Fatalf("缺少被拒绝 claim %s: %+v", id, res.IgnoredClaims)
		}
	}

	// 未知通道同样没有事实资格(保守默认, denied by default)。
	unknown := facts
	unknown[1].Recorder = evidence.Recorder{Kind: "mystery", Actor: "x"}
	res = mustDerive(t, Snapshot{Contract: f, Facts: unknown})
	if res.State != StatePlanned || !hasClaim(res, "fact:a-frz", ClaimReasonRecorderNotFactCapable) {
		t.Fatalf("未知通道被当成了事实: %s %+v", res.State, res.IgnoredClaims)
	}
}

func TestAgentFailureClaimDoesNotBlockAndApprovalClaimDoesNotApprove(t *testing.T) {
	s := verifiedSnapshot(t)
	// Agent 记录的 FAIL 是 claim:既不能阻断 VERIFIED,也不能反向制造事实。
	s.Failures = []ApplicableFailure{{
		FactID: "af-1", BindingID: "EV-1", ChangeSet: cs1,
		Recorder: agentR, At: at(6),
	}}
	res := mustDerive(t, s)
	if res.State != StateVerified {
		t.Fatalf("agent FAIL claim 阻断了状态: %s", res.State)
	}
	if !hasClaim(res, "af-1", ClaimReasonFailureNotFactCapable) {
		t.Fatalf("缺少被拒绝的 agent FAIL claim: %+v", res.IgnoredClaims)
	}

	// Agent 记录的“批准”不产生 APPROVED(AC1/AC3)。
	s.Approvals = []ApprovalEvidence{{
		ID: "ap-agent", Decision: DecisionApprove, ChangeSet: cs1,
		Recorder: agentR, At: at(7),
	}}
	res = mustDerive(t, s)
	if res.State != StateVerified {
		t.Fatalf("agent 批准产生了 APPROVED: %s", res.State)
	}
	if !hasClaim(res, "approval:ap-agent", ClaimReasonApprovalNotHuman) {
		t.Fatalf("缺少被拒绝的 agent 批准: %+v", res.IgnoredClaims)
	}
}

// ---- AC2:VERIFIED 需要功能门与适用验证证据 ----------------------------------

func TestVerifiedRequiresFunctionalGatesAndApplicableEvidence(t *testing.T) {
	f := frozenContract(t, 1)
	facts := lifecycleFacts(f.Hash())

	t.Run("all gates pass", func(t *testing.T) {
		res := mustDerive(t, Snapshot{
			Contract: f, Facts: facts,
			Functional:   funcReport(f, true),
			Independence: indepReport(f, true),
		})
		if res.State != StateVerified || res.VerificationClass != ClassStrictSatisfied {
			t.Fatalf("state/class = %s/%s, want VERIFIED/strict_satisfied", res.State, res.VerificationClass)
		}
		if res.CurrentCandidate != cs1 || res.CurrentRun != "run-1" || res.ContractHash != f.Hash() {
			t.Fatalf("追溯字段不完整: %+v", res)
		}
	})

	t.Run("no functional report", func(t *testing.T) {
		res := mustDerive(t, Snapshot{Contract: f, Facts: facts, Independence: indepReport(f, true)})
		if res.State != StateValidating || !hasGap(res, ReqFunctionalCoverage) {
			t.Fatalf("state=%s gaps=%+v", res.State, res.Gaps)
		}
	})

	t.Run("functional incomplete", func(t *testing.T) {
		res := mustDerive(t, Snapshot{Contract: f, Facts: facts, Functional: funcReport(f, false), Independence: indepReport(f, true)})
		if res.State == StateVerified || !hasGap(res, ReqFunctionalCoverage) {
			t.Fatalf("state=%s gaps=%+v", res.State, res.Gaps)
		}
	})

	t.Run("report bound to superseded revision is invisible (INV-STATE-03/§8.8.2)", func(t *testing.T) {
		old := frozenContract(t, 1)
		f2 := frozenContract(t, 2) // 新 revision:哈希不同
		if old.Hash() == f2.Hash() {
			t.Fatal("两个 revision 的哈希不应相同")
		}
		res := mustDerive(t, Snapshot{
			Contract: f2, Facts: lifecycleFacts(f2.Hash()),
			Functional:   funcReport(old, true), // 旧 revision 的报告
			Independence: indepReport(old, true),
		})
		if res.State == StateVerified {
			t.Fatal("旧 revision 的覆盖报告验证了新 revision")
		}
		if !hasGap(res, ReqFunctionalCoverage) || !hasGap(res, ReqIndependenceCoverage) {
			t.Fatalf("gaps=%+v", res.Gaps)
		}
	})

	t.Run("no verification run for current candidate", func(t *testing.T) {
		res := mustDerive(t, Snapshot{
			Contract: f, Facts: lifecycleFacts(f.Hash())[:4], // 无 f-vrun
			Functional:   funcReport(f, true),
			Independence: indepReport(f, true),
		})
		if res.State != StateCandidate || !hasGap(res, ReqVerificationRun) {
			t.Fatalf("state=%s gaps=%+v", res.State, res.Gaps)
		}
	})

	t.Run("no candidate at all", func(t *testing.T) {
		res := mustDerive(t, Snapshot{
			Contract: f, Facts: lifecycleFacts(f.Hash())[:3],
			Functional:   funcReport(f, true),
			Independence: indepReport(f, true),
		})
		if res.State != StateRunning || !hasGap(res, ReqCandidate) {
			t.Fatalf("state=%s gaps=%+v", res.State, res.Gaps)
		}
	})

	t.Run("no frozen contract identity", func(t *testing.T) {
		res := mustDerive(t, Snapshot{Facts: lifecycleFacts(f.Hash())})
		if res.State != StateValidating {
			t.Fatalf("desired 契约缺失时仍应按事实位置推导,得到 %s", res.State)
		}
		if !hasGap(res, ReqVerificationContract) || !hasGap(res, ReqFunctionalCoverage) {
			t.Fatalf("gaps=%+v", res.Gaps)
		}
	})
}

// ---- AC3:APPROVED 需要 HumanApprovalEvidence --------------------------------

func TestApprovalRequiresHumanApprovalEvidence(t *testing.T) {
	s := verifiedSnapshot(t)

	res := mustDerive(t, s)
	if res.State != StateVerified {
		t.Fatalf("没有批准事实时得到 %s, want VERIFIED(AC3)", res.State)
	}

	s.Approvals = []ApprovalEvidence{approvalFor(cs1)}
	res = mustDerive(t, s)
	if res.State != StateApproved || res.VerificationClass != ClassStrictSatisfied {
		t.Fatalf("state/class = %s/%s, want APPROVED/strict_satisfied", res.State, res.VerificationClass)
	}

	// 批准绑定其它(历史)候选:不批准当前候选。
	s.Approvals = []ApprovalEvidence{approvalFor(cs2)}
	if res := mustDerive(t, s); res.State != StateVerified {
		t.Fatalf("历史候选的批准产生了 APPROVED: %s", res.State)
	}

	// reject 决定不产生 APPROVED。
	rej := approvalFor(cs1)
	rej.ID, rej.Decision = "ap-2", DecisionReject
	s.Approvals = []ApprovalEvidence{rej}
	if res := mustDerive(t, s); res.State != StateVerified {
		t.Fatalf("reject 决定产生了 APPROVED: %s", res.State)
	}
}

// ---- AC4:strict waived 与 strict satisfied 可区分 ----------------------------

func TestStrictWaivedDistinguishableFromStrictSatisfied(t *testing.T) {
	f := frozenContract(t, 1)
	facts := lifecycleFacts(f.Hash())
	good := Snapshot{Contract: f, Facts: facts, Functional: funcReport(f, true)}

	t.Run("strict satisfied", func(t *testing.T) {
		res := mustDerive(t, withIndep(good, indepReport(f, true)))
		if res.State != StateVerified || res.VerificationClass != ClassStrictSatisfied {
			t.Fatalf("%s/%s", res.State, res.VerificationClass)
		}
	})

	t.Run("strict waived", func(t *testing.T) {
		s := withIndep(good, indepReport(f, false))
		s.WaiverPermitted = true
		s.Waivers = []WaiverEvidence{waiverFor(f, cs1, "AC-01")}
		res := mustDerive(t, s)
		if res.State != StateVerified || res.VerificationClass != ClassStrictWaived {
			t.Fatalf("%s/%s, want VERIFIED/strict_waived", res.State, res.VerificationClass)
		}
		if !hasClaim(res, "nonexistent", "x") && len(res.IgnoredClaims) != 0 {
			t.Fatalf("waived 路径不应有 ignored claims: %+v", res.IgnoredClaims)
		}
		// Waiver 路径必须在 Basis 中显式可追溯。
		foundWaiver, foundPolicy := false, false
		for _, b := range res.Basis {
			switch b {
			case "waiver:wv-1":
				foundWaiver = true
			case "policy:waiver_permitted":
				foundPolicy = true
			}
		}
		if !foundWaiver || !foundPolicy {
			t.Fatalf("waiver basis 不完整: %v", res.Basis)
		}
	})

	t.Run("waiver not permitted by policy", func(t *testing.T) {
		s := withIndep(good, indepReport(f, false))
		s.Waivers = []WaiverEvidence{waiverFor(f, cs1, "AC-01")}
		res := mustDerive(t, s)
		if res.State == StateVerified || !hasGap(res, ReqIndependenceCoverage) {
			t.Fatalf("policy 禁止时 waiver 仍放行: %s gaps=%+v", res.State, res.Gaps)
		}
	})

	t.Run("waiver misses compromised acceptance", func(t *testing.T) {
		s := withIndep(good, indepReport(f, false))
		s.WaiverPermitted = true
		s.Waivers = []WaiverEvidence{waiverFor(f, cs1, "AC-99")}
		if res := mustDerive(t, s); res.State == StateVerified {
			t.Fatal("未覆盖缺口的 waiver 产生了 VERIFIED")
		}
	})

	t.Run("waiver bound to superseded contract revision", func(t *testing.T) {
		f2 := frozenContract(t, 2)
		s := Snapshot{
			Contract: f2, Facts: lifecycleFacts(f2.Hash()),
			Functional:      funcReport(f2, true),
			Independence:    indepReport(f2, false),
			WaiverPermitted: true,
			Waivers:         []WaiverEvidence{waiverFor(f, cs1, "AC-01")}, // 旧 revision 哈希
		}
		if res := mustDerive(t, s); res.State == StateVerified {
			t.Fatal("旧 revision 的 waiver 放行了新 revision")
		}
	})

	t.Run("waiver bound to other candidate", func(t *testing.T) {
		s := withIndep(good, indepReport(f, false))
		s.WaiverPermitted = true
		s.Waivers = []WaiverEvidence{waiverFor(f, cs2, "AC-01")}
		if res := mustDerive(t, s); res.State == StateVerified {
			t.Fatal("其它候选的 waiver 放行了当前候选")
		}
	})

	t.Run("validator-recorded waiver is a claim", func(t *testing.T) {
		s := withIndep(good, indepReport(f, false))
		s.WaiverPermitted = true
		w := waiverFor(f, cs1, "AC-01")
		w.Recorder = validator
		s.Waivers = []WaiverEvidence{w}
		res := mustDerive(t, s)
		if res.State == StateVerified {
			t.Fatal("非人类通道的 waiver 放行了 VERIFIED")
		}
		if !hasClaim(res, "waiver:wv-1", ClaimReasonWaiverNotHuman) {
			t.Fatalf("缺少被拒绝的 waiver claim: %+v", res.IgnoredClaims)
		}
	})

	t.Run("waiver never covers functional gaps", func(t *testing.T) {
		s := good
		s.Functional = funcReport(f, false)
		s.Independence = indepReport(f, true)
		s.WaiverPermitted = true
		s.Waivers = []WaiverEvidence{waiverFor(f, cs1, "AC-01")}
		if res := mustDerive(t, s); res.State == StateVerified {
			t.Fatal("waiver 覆盖了功能门(§8.10 禁止)")
		}
	})
}

func withIndep(s Snapshot, r *verification.IndependenceCoverageReport) Snapshot {
	s.Independence = r
	return s
}

// ---- AC5:同一事实集的重建是确定性的 ------------------------------------------

func TestRebuildDeterministicAcrossFactOrder(t *testing.T) {
	f := frozenContract(t, 1)
	facts := lifecycleFacts(f.Hash())
	facts = append(facts,
		Fact{ID: "f-run2", Kind: FactCodegenRunStarted, At: at(2), Recorder: external, RunID: "run-0"},
		Fact{ID: "f-cand2", Kind: FactChangeSetProduced, At: at(3), Recorder: validator, RunID: "run-1", ChangeSet: cs2},
		Fact{ID: "f-review", Kind: FactReviewRequested, At: at(4), Recorder: validator},
		Fact{ID: "f-review-x", Kind: FactReviewResolved, At: at(5), Recorder: humanR, Resolves: "f-review"},
		Fact{ID: "f-blk", Kind: FactTaskBlocked, At: at(4), Recorder: humanR},
		Fact{ID: "f-blk-x", Kind: FactTaskUnblocked, At: at(6), Recorder: validator, Resolves: "f-blk"},
	)
	// 同一时刻、不同种类的事实:规范排序的并列破除必须稳定。
	facts = append(facts,
		Fact{ID: "f-tie-a", Kind: FactCodegenRunStarted, At: at(7), Recorder: validator, RunID: "run-2"},
		Fact{ID: "f-tie-b", Kind: FactChangeSetProduced, At: at(7), Recorder: validator, RunID: "run-2", ChangeSet: cs1},
		Fact{ID: "f-tie-c", Kind: FactVerificationRunStarted, At: at(7), Recorder: validator, RunID: "vr-2", ChangeSet: cs1},
	)
	snap := Snapshot{
		Contract:        f,
		Facts:           facts,
		Functional:      funcReport(f, true),
		Independence:    indepReport(f, false),
		WaiverPermitted: true,
		Waivers:         []WaiverEvidence{waiverFor(f, cs1, "AC-01")},
		Approvals:       []ApprovalEvidence{approvalFor(cs1), approvalFor(cs2)},
		Violations: []IntegrityViolation{{
			ID: "vio-1", Kind: IntegrityHoldoutExposure, Ref: "EV-1",
			ResolvedBy: "res-1", Recorder: validator, At: at(8),
		}},
		Failures: []ApplicableFailure{
			{FactID: "fail-1", BindingID: "EV-1", ChangeSet: cs2, ResolvedBy: "rule-1", Recorder: validator, At: at(8)},
			{FactID: "fail-2", BindingID: "EV-1", ChangeSet: cs1, Recorder: validator, At: at(8)},
		},
	}

	want := mustDerive(t, snap)
	rnd := rand.New(rand.NewSource(42))
	for i := 0; i < 25; i++ {
		shuffled := make([]Fact, len(snap.Facts))
		copy(shuffled, snap.Facts)
		rnd.Shuffle(len(shuffled), func(a, b int) { shuffled[a], shuffled[b] = shuffled[b], shuffled[a] })
		s := snap
		s.Facts = shuffled
		got := mustDerive(t, s)
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("第 %d 次重排得到不同结果(INV-STATE-01):\nwant %+v\ngot  %+v", i, want, got)
		}
	}

	// 完全逆序同样一致。
	rev := make([]Fact, len(snap.Facts))
	for i := range snap.Facts {
		rev[len(rev)-1-i] = snap.Facts[i]
	}
	s := snap
	s.Facts = rev
	if got := mustDerive(t, s); !reflect.DeepEqual(want, got) {
		t.Fatalf("逆序输入得到不同结果:\nwant %+v\ngot  %+v", want, got)
	}
}

// ---- 异常态推导 ---------------------------------------------------------------

func TestExceptionStatesAreDerived(t *testing.T) {
	f := frozenContract(t, 1)
	hash := f.Hash()

	t.Run("cancelled is terminal", func(t *testing.T) {
		facts := lifecycleFacts(hash)
		facts = append(facts,
			Fact{ID: "f-cancel", Kind: FactTaskCancelled, At: at(5), Recorder: humanR},
			Fact{ID: "f-late", Kind: FactChangeSetProduced, At: at(6), Recorder: validator, RunID: "run-1", ChangeSet: cs1},
		)
		res := mustDerive(t, Snapshot{Contract: f, Facts: facts})
		if res.State != StateCancelled {
			t.Fatalf("state = %s, want CANCELLED", res.State)
		}
	})

	t.Run("review requested then resolved", func(t *testing.T) {
		facts := lifecycleFacts(hash)
		facts = append(facts,
			Fact{ID: "f-review", Kind: FactReviewRequested, At: at(5), Recorder: validator},
		)
		res := mustDerive(t, Snapshot{Contract: f, Facts: facts})
		if res.State != StateNeedsReview {
			t.Fatalf("state = %s, want NEEDS_REVIEW", res.State)
		}
		facts = append(facts, Fact{ID: "f-review-x", Kind: FactReviewResolved, At: at(6), Recorder: humanR, Resolves: "f-review"})
		res = mustDerive(t, Snapshot{Contract: f, Facts: facts})
		if res.State != StateValidating {
			t.Fatalf("解除后 state = %s, want VALIDATING", res.State)
		}
		// 解除未知 id:无效果。
		facts[len(facts)-1].Resolves = "f-unknown"
		res = mustDerive(t, Snapshot{Contract: f, Facts: facts})
		if res.State != StateNeedsReview {
			t.Fatalf("对未知请求的解除改变了状态: %s", res.State)
		}
	})

	t.Run("blocked then unblocked", func(t *testing.T) {
		facts := lifecycleFacts(hash)
		facts = append(facts,
			Fact{ID: "f-blk", Kind: FactTaskBlocked, At: at(5), Recorder: humanR},
		)
		res := mustDerive(t, Snapshot{Contract: f, Facts: facts})
		if res.State != StateBlocked {
			t.Fatalf("state = %s, want BLOCKED", res.State)
		}
		facts = append(facts, Fact{ID: "f-blk-x", Kind: FactTaskUnblocked, At: at(6), Recorder: validator, Resolves: "f-blk"})
		res = mustDerive(t, Snapshot{Contract: f, Facts: facts})
		if res.State != StateValidating {
			t.Fatalf("解除后 state = %s, want VALIDATING", res.State)
		}
	})

	t.Run("review outranks open block", func(t *testing.T) {
		facts := lifecycleFacts(hash)
		facts = append(facts,
			Fact{ID: "f-blk", Kind: FactTaskBlocked, At: at(5), Recorder: humanR},
			Fact{ID: "f-review", Kind: FactReviewRequested, At: at(6), Recorder: validator},
		)
		res := mustDerive(t, Snapshot{Contract: f, Facts: facts})
		if res.State != StateNeedsReview {
			t.Fatalf("state = %s, want NEEDS_REVIEW", res.State)
		}
	})

	t.Run("failed run then new run", func(t *testing.T) {
		facts := lifecycleFacts(hash)
		facts = append(facts,
			Fact{ID: "f-fail", Kind: FactCodegenRunFailed, At: at(5), Recorder: validator, RunID: "run-1"},
		)
		res := mustDerive(t, Snapshot{Contract: f, Facts: facts})
		if res.State != StateFailed {
			t.Fatalf("state = %s, want FAILED", res.State)
		}
		facts = append(facts, Fact{ID: "f-run2", Kind: FactCodegenRunStarted, At: at(6), Recorder: validator, RunID: "run-2"})
		res = mustDerive(t, Snapshot{Contract: f, Facts: facts})
		if res.State != StateRunning {
			t.Fatalf("新 Run 后 state = %s, want RUNNING", res.State)
		}
	})
}

// ---- integrity / applicable failures ----------------------------------------

func TestIntegrityViolationBlocksVerified(t *testing.T) {
	s := verifiedSnapshot(t)
	s.Violations = []IntegrityViolation{{
		ID: "vio-1", Kind: IntegrityProtectedBaseline, Ref: "EV-1",
		Recorder: validator, At: at(6),
	}}
	res := mustDerive(t, s)
	if res.State == StateVerified || !hasGap(res, ReqIntegrity) {
		t.Fatalf("unresolved violation 未阻断: %s gaps=%+v", res.State, res.Gaps)
	}
	// waiver 覆盖不了 integrity(即使 independence 也不缺)。
	s.WaiverPermitted = true
	s.Waivers = []WaiverEvidence{waiverFor(frozenContract(t, 1), cs1, "AC-01")}
	if res := mustDerive(t, s); res.State == StateVerified {
		t.Fatal("waiver 放行了 integrity violation")
	}
	s.Violations[0].ResolvedBy = "res-1"
	if res := mustDerive(t, s); res.State != StateVerified {
		t.Fatalf("解除后 state = %s, want VERIFIED", res.State)
	}
}

func TestApplicableFailureBlocksVerifiedUntilResolved(t *testing.T) {
	s := verifiedSnapshot(t)
	s.Failures = []ApplicableFailure{{
		FactID: "fail-1", BindingID: "EV-1", ChangeSet: cs1,
		Recorder: validator, At: at(6),
	}}
	res := mustDerive(t, s)
	if res.State == StateVerified || !hasGap(res, ReqNoApplicableFailures) {
		t.Fatalf("applicable FAIL 未阻断: %s gaps=%+v", res.State, res.Gaps)
	}
	// 绑定历史候选的 FAIL 不适用于当前候选(§8.8.2)。
	s.Failures[0].ChangeSet = cs2
	res = mustDerive(t, s)
	if res.State != StateVerified {
		t.Fatalf("历史候选的 FAIL 阻断了当前候选: %s", res.State)
	}
	// 正式裁定解除后不再阻断。
	s.Failures[0].ChangeSet = cs1
	s.Failures[0].ResolvedBy = "ruling-1"
	if res := mustDerive(t, s); res.State != StateVerified {
		t.Fatalf("裁定后 state = %s, want VERIFIED", res.State)
	}
}

// ---- 候选 supersession --------------------------------------------------------

func TestNewCandidateSupersedesValidation(t *testing.T) {
	f := frozenContract(t, 1)
	facts := lifecycleFacts(f.Hash())
	res := mustDerive(t, Snapshot{Contract: f, Facts: facts, Functional: funcReport(f, true), Independence: indepReport(f, true)})
	if res.State != StateVerified {
		t.Fatalf("state = %s, want VERIFIED", res.State)
	}

	// 新候选产生:验证 Run 属于旧候选,当前候选回到未验证位置。
	facts = append(facts, Fact{ID: "f-cand2", Kind: FactChangeSetProduced, At: at(5), Recorder: validator, RunID: "run-1", ChangeSet: cs2})
	res = mustDerive(t, Snapshot{Contract: f, Facts: facts, Functional: funcReport(f, true), Independence: indepReport(f, true)})
	if res.State != StateCandidate || res.CurrentCandidate != cs2 {
		t.Fatalf("state/candidate = %s/%s, want CANDIDATE/cs-2", res.State, res.CurrentCandidate)
	}
	if !hasGap(res, ReqVerificationRun) {
		t.Fatalf("gaps=%+v", res.Gaps)
	}

	// 旧候选上的批准不批准新候选。
	facts = append(facts,
		Fact{ID: "f-vrun2", Kind: FactVerificationRunStarted, At: at(6), Recorder: validator, RunID: "vrun-2", ChangeSet: cs2},
	)
	res = mustDerive(t, Snapshot{
		Contract: f, Facts: facts, Functional: funcReport(f, true), Independence: indepReport(f, true),
		Approvals: []ApprovalEvidence{approvalFor(cs1)},
	})
	if res.State != StateVerified {
		t.Fatalf("state = %s, want VERIFIED(批准属于旧候选)", res.State)
	}
}

// ---- 合并推导 ------------------------------------------------------------------

func TestMergedRequiresApprovalAndCurrentCandidate(t *testing.T) {
	f := frozenContract(t, 1)
	facts := lifecycleFacts(f.Hash())

	merge := Fact{ID: "f-merge", Kind: FactMergeRecorded, At: at(6), Recorder: validator, ChangeSet: cs1}

	// 未批准的合并事实:停在 VERIFIED。
	res := mustDerive(t, Snapshot{Contract: f, Facts: append(facts, merge), Functional: funcReport(f, true), Independence: indepReport(f, true)})
	if res.State != StateVerified {
		t.Fatalf("state = %s, want VERIFIED", res.State)
	}

	// 批准 + 合并 → MERGED。
	res = mustDerive(t, Snapshot{
		Contract: f, Facts: append(facts, merge), Functional: funcReport(f, true), Independence: indepReport(f, true),
		Approvals: []ApprovalEvidence{approvalFor(cs1)},
	})
	if res.State != StateMerged {
		t.Fatalf("state = %s, want MERGED", res.State)
	}

	// 合并的是历史候选:不产生 MERGED。
	merge.ChangeSet = cs2
	res = mustDerive(t, Snapshot{
		Contract: f, Facts: append(facts, merge), Functional: funcReport(f, true), Independence: indepReport(f, true),
		Approvals: []ApprovalEvidence{approvalFor(cs1)},
	})
	if res.State != StateApproved {
		t.Fatalf("state = %s, want APPROVED", res.State)
	}
}

// ---- 结构性错误 ----------------------------------------------------------------

func TestMalformedInputsAreErrors(t *testing.T) {
	f := frozenContract(t, 1)
	good := lifecycleFacts(f.Hash())

	cases := []struct {
		name string
		snap Snapshot
		want error
	}{
		{"empty facts", Snapshot{}, ErrEmptySnapshot},
		{"no planned fact", Snapshot{Facts: good[1:]}, ErrNoTaskFact},
		{
			"duplicate fact id",
			Snapshot{Facts: append(append([]Fact{}, good...), good[0])},
			ErrDuplicateFactID,
		},
		{
			"unknown kind",
			Snapshot{Facts: append(append([]Fact{}, good...), Fact{ID: "f-x", Kind: "verified_because_i_said_so", At: at(9), Recorder: validator})},
			ErrUnknownFactKind,
		},
		{
			"zero time",
			Snapshot{Facts: append(append([]Fact{}, good...), Fact{ID: "f-x", Kind: FactTaskBlocked, Recorder: validator})},
			ErrMalformedFact,
		},
		{
			"changeset on planned fact",
			Snapshot{Facts: []Fact{{ID: "f-plan", Kind: FactTaskPlanned, At: at(0), Recorder: validator, RevisionRef: "h", ChangeSet: cs1}}},
			ErrMalformedFact,
		},
		{
			"waiver without acceptances",
			Snapshot{Facts: good, Waivers: []WaiverEvidence{{ID: "w", ContractHash: f.Hash(), ChangeSet: cs1, ReasonCode: "r", Recorder: humanR, At: at(1)}}},
			ErrMalformedFact,
		},
		{
			"approval with bad decision",
			Snapshot{Facts: good, Approvals: []ApprovalEvidence{{ID: "a", Decision: "maybe", ChangeSet: cs1, Recorder: humanR, At: at(1)}}},
			ErrMalformedFact,
		},
		{
			"failure without binding",
			Snapshot{Facts: good, Failures: []ApplicableFailure{{FactID: "x", ChangeSet: cs1, Recorder: validator, At: at(1)}}},
			ErrMalformedFact,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Derive(tc.snap); !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}
