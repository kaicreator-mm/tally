package closedloop

// phase.go 状态机测试:P2-02 AC1(无通用 workflow DSL —— 边集固定且可枚举)、
// AC2(显式终态,终态吸收)、转移表确定性。

import (
	"fmt"
	"testing"
)

// allEvents 是状态机事件词表的穷举(新增事件时此处必须同步)。
var allEvents = []Event{EventStart, EventCandidateReady, EventVerified, EventRepair, EventEscalate, EventCancel}

// TestPhaseClassification 校验状态的合法性/终态/活动态分类。
func TestPhaseClassification(t *testing.T) {
	terminal := []Phase{PhaseAccepted, PhaseEscalated, PhaseCancelled}
	active := []Phase{PhaseExecuting, PhaseVerifying, PhaseRepairing}
	entries := []Phase{PhasePending}

	for _, p := range append(append(append([]Phase{}, terminal...), active...), entries...) {
		if !p.IsValid() {
			t.Errorf("%s 应为合法状态", p)
		}
	}
	for _, p := range terminal {
		if !p.Terminal() {
			t.Errorf("%s 应为终态", p)
		}
		if p.Active() {
			t.Errorf("%s 终态不应是活动态", p)
		}
	}
	for _, p := range active {
		if !p.Active() {
			t.Errorf("%s 应为活动态", p)
		}
		if p.Terminal() {
			t.Errorf("%s 活动态不应是终态", p)
		}
	}
	for _, p := range []Phase{Phase("bogus"), "", Phase("Verified")} {
		if p.IsValid() {
			t.Errorf("%q 不应为合法状态", p)
		}
		if p.Terminal() || p.Active() {
			t.Errorf("%q 非法状态不应有终态/活动态分类", p)
		}
	}
}

// TestStepLegalEdges 穷举转移表的全部合法边(P2-02 AC1:边集固定,这是
// 完整清单;新增边必须显式出现在这里)。
func TestStepLegalEdges(t *testing.T) {
	legal := map[Phase]map[Event]Phase{
		PhasePending: {
			EventStart:  PhaseExecuting,
			EventCancel: PhaseCancelled,
		},
		PhaseExecuting: {
			EventCandidateReady: PhaseVerifying,
			EventEscalate:       PhaseEscalated,
			EventCancel:         PhaseCancelled,
		},
		PhaseVerifying: {
			EventVerified: PhaseAccepted,
			EventRepair:   PhaseRepairing,
			EventEscalate: PhaseEscalated,
			EventCancel:   PhaseCancelled,
		},
		PhaseRepairing: {
			EventCandidateReady: PhaseVerifying,
			EventEscalate:       PhaseEscalated,
			EventCancel:         PhaseCancelled,
		},
	}
	for from, edges := range legal {
		for ev, want := range edges {
			got, ok := step(from, ev)
			if !ok || got != want {
				t.Errorf("step(%s, %s) = (%s, %v), 期望 (%s, true)", from, ev, got, ok, want)
			}
			if !CanTransition(from, want) {
				t.Errorf("CanTransition(%s, %s) = false,但表中有边", from, want)
			}
		}
	}
}

// TestStepIllegalEdgesAreRejected 校验:终态吸收;活动态对未列出的事件拒绝;
// 非法状态拒绝一切。
func TestStepIllegalEdgesAreRejected(t *testing.T) {
	terminal := []Phase{PhaseAccepted, PhaseEscalated, PhaseCancelled}
	for _, p := range terminal {
		for _, ev := range allEvents {
			if got, ok := step(p, ev); ok {
				t.Errorf("终态 %s 对事件 %s 不应有出边,得到 %s", p, ev, got)
			}
			if CanTransition(p, p) {
				t.Errorf("终态 %s 不应有任何出边(含自环)", p)
			}
		}
	}

	// Executing 不允许直接 Accepted:候选必须先过独立验证(结构性禁止)。
	if _, ok := step(PhaseExecuting, EventVerified); ok {
		t.Error("Executing → Accepted 非法:候选必须先过独立验证")
	}
	// Executing 不允许直接 Repair:修复只能由 Verifying 的可信 FAIL 裁决进入。
	if _, ok := step(PhaseExecuting, EventRepair); ok {
		t.Error("Executing → Repairing 非法:修复只能从 Verifying 进入")
	}
	// Pending 不允许跳过执行直接验证/接受。
	for _, ev := range []Event{EventCandidateReady, EventVerified, EventRepair} {
		if _, ok := step(PhasePending, ev); ok {
			t.Errorf("Pending 对事件 %s 不应有出边", ev)
		}
	}
	// Repairing 不允许直接 Accepted(同 Executing:必须先验证)。
	if _, ok := step(PhaseRepairing, EventVerified); ok {
		t.Error("Repairing → Accepted 非法:修复候选必须先过独立验证")
	}
	if _, ok := step(Phase("bogus"), EventStart); ok {
		t.Error("非法状态不应有任何出边")
	}
	if CanTransition(PhasePending, PhaseAccepted) {
		t.Error("Pending → Accepted 非法:无验证即接受被结构性禁止")
	}
}

// TestEventString 覆盖事件规范名与未知值。
func TestEventString(t *testing.T) {
	names := map[Event]string{
		EventStart:          "start",
		EventCandidateReady: "candidate_ready",
		EventVerified:       "verified",
		EventRepair:         "repair",
		EventEscalate:       "escalate",
		EventCancel:         "cancel",
		Event(99):           "Event(unknown)",
	}
	for ev, want := range names {
		if got := fmt.Sprint(ev); got != want {
			t.Errorf("Event(%d).String() = %q, 期望 %q", uint8(ev), got, want)
		}
	}
}

// TestRouteString 覆盖路由规范名与未知值。
func TestRouteString(t *testing.T) {
	if got := RouteRepair.String(); got != "repair" {
		t.Errorf("RouteRepair.String() = %q", got)
	}
	if got := RouteEscalate.String(); got != "escalate" {
		t.Errorf("RouteEscalate.String() = %q", got)
	}
	if got := Route(42).String(); got != "Route(unknown)" {
		t.Errorf("Route(42).String() = %q", got)
	}
}

// TestAttemptNoVocabulary 校验尝试序号词表(1 = 首次,≥2 = 修复)。
func TestAttemptNoVocabulary(t *testing.T) {
	if (AttemptNo(0)).IsValid() || (AttemptNo(-1)).IsValid() {
		t.Error("AttemptNo < 1 不应合法")
	}
	if !(AttemptNo(1)).IsValid() || (AttemptNo(1)).IsRepair() {
		t.Error("AttemptNo 1 = 首次执行,不是修复")
	}
	if !(AttemptNo(2)).IsRepair() || !(AttemptNo(7)).IsRepair() {
		t.Error("AttemptNo >= 2 应为修复尝试")
	}
}

// TestRunIDForDeterminism 校验尝试幂等键(ADR-0001 §2:身份对
// (task, attempt_no) 幂等;不同键互异)。
func TestRunIDForDeterminism(t *testing.T) {
	a1 := RunIDFor("TASK-A", 1)
	if a1 != RunIDFor("TASK-A", 1) {
		t.Fatal("同 (task, attempt) 必须推导出同一 RunID(幂等键)")
	}
	distinct := []RunID{
		a1,
		RunIDFor("TASK-A", 2),
		RunIDFor("TASK-A", 3),
		RunIDFor("TASK-B", 1),
	}
	seen := map[RunID]bool{}
	for _, id := range distinct {
		if id == "" {
			t.Fatal("RunIDFor 不应返回空 id")
		}
		if seen[id] {
			t.Errorf("RunID %q 重复:不同 (task, attempt) 必须互异", id)
		}
		seen[id] = true
	}
}

// TestVerdictVocabulary 校验 verdict 词表(FAIL 与 Inconclusive 端到端不同)。
func TestVerdictVocabulary(t *testing.T) {
	for _, v := range []Verdict{VerdictVerified, VerdictFailed, VerdictInconclusive} {
		if !v.IsValid() {
			t.Errorf("verdict %q 应合法", v)
		}
	}
	for _, v := range []Verdict{"", "passed", "error", "VERIFIED"} {
		if v.IsValid() {
			t.Errorf("verdict %q 不应合法(词表外)", v)
		}
	}
	if VerdictFailed == VerdictInconclusive {
		t.Fatal("FAIL 与 Inconclusive 必须是不同 verdict(ADR-0001 §3)")
	}
}

// TestEscalationReasonValidity 校验升级原因最低门槛(非空)。
func TestEscalationReasonValidity(t *testing.T) {
	if (EscalationReason("")).IsValid() || (EscalationReason("   ")).IsValid() {
		t.Error("空升级原因不应合法")
	}
	if !(EscalationReason("contract_defect")).IsValid() {
		t.Error("非空升级原因应合法(词表归产出方,循环不解释)")
	}
}
