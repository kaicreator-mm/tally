// outcome_evidence_test.go 拥有 P6-03 投影桥的测试:outcome 事件 → 治理证据
// 引用的映射(封闭 kind + 至少一个绑定 + 显式时刻,fail-closed)、次序保真、
// 引用稳定性(同一事件恒同一引用),以及运行时匹配面与 Outcome 的零耦合
// 边界守卫(AC:governance 读 append-only 事件;matching 不查 Outcome)。
package policygovernance

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"tally/internal/outcome"
)

func p603Event(kind outcome.Kind, task, cs string, at time.Time) outcome.Event {
	return outcome.Event{TaskID: task, ChangesetID: cs, Kind: kind, RecordedAt: at}
}

// AC 正面:合法 outcome 事件 → EvidenceOutcome 引用;引用含绑定/kind/时刻
// 三元组(自然键,append-only 日志内唯一)。
func TestOutcomeEvidenceRef(t *testing.T) {
	at := time.Date(2026, 9, 13, 4, 0, 0, 0, time.UTC)
	ref, err := OutcomeEvidenceRef(p603Event(outcome.KindAccepted, "TASK-1", "", at))
	if err != nil {
		t.Fatalf("OutcomeEvidenceRef: %v", err)
	}
	if ref.Kind != EvidenceOutcome {
		t.Fatalf("Kind = %s, want outcome", ref.Kind)
	}
	want := "outcome:TASK-1/accepted/2026-09-13T04:00:00Z"
	if got := ref.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
	// 同一事件恒同一引用(稳定性);绑定优先级:task 优先于 changeset。
	again, _ := OutcomeEvidenceRef(p603Event(outcome.KindAccepted, "TASK-1", "", at))
	if again.Ref != ref.Ref {
		t.Fatalf("同一事件引用漂移: %q vs %q", again.Ref, ref.Ref)
	}
	csOnly, err := OutcomeEvidenceRef(p603Event(outcome.KindRegressionFound, "", "cs-7", at))
	if err != nil {
		t.Fatalf("OutcomeEvidenceRef(changeset 绑定): %v", err)
	}
	if got := csOnly.String(); !strings.HasPrefix(got, "outcome:cs-7/regression_found/") {
		t.Fatalf("changeset 绑定引用 = %q", got)
	}
}

// fail-closed:kind 词表外/无绑定/缺时刻的事件不产生证据引用。
func TestOutcomeEvidenceNegatives(t *testing.T) {
	at := time.Now().UTC()
	if _, err := OutcomeEvidenceRef(p603Event("deployed", "TASK-1", "", at)); err == nil {
		t.Fatal("词表外 kind 应拒绝(五类封闭枚举)")
	}
	if _, err := OutcomeEvidenceRef(p603Event(outcome.KindAccepted, "", "", at)); err == nil {
		t.Fatal("完全无绑定应拒绝")
	}
	if _, err := OutcomeEvidenceRef(p603Event(outcome.KindAccepted, "TASK-1", "", time.Time{})); err == nil {
		t.Fatal("缺时刻应拒绝")
	}
	// ProposalEvidence:任一事件不合法 → 整体失败,不静默裁剪(证据面纪律)。
	events := []outcome.Event{
		p603Event(outcome.KindAccepted, "TASK-1", "", at),
		p603Event("deployed", "TASK-1", "", at),
	}
	if _, err := ProposalEvidence(events); err == nil {
		t.Fatal("含不合法事件的批次应整体失败")
	}
}

// 次序保真:投影保持事件日志先后次序(append-only 读序即证据序)。
func TestProposalEvidenceOrderPreserved(t *testing.T) {
	t0 := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	events := []outcome.Event{
		p603Event(outcome.KindAccepted, "TASK-1", "", t0),
		p603Event(outcome.KindRegressionFound, "TASK-1", "", t0.Add(time.Hour)),
		p603Event(outcome.KindHumanCorrection, "TASK-1", "", t0.Add(2*time.Hour)),
	}
	refs, err := ProposalEvidence(events)
	if err != nil {
		t.Fatalf("ProposalEvidence: %v", err)
	}
	if len(refs) != 3 {
		t.Fatalf("refs = %d, want 3", len(refs))
	}
	for i, want := range []string{"accepted", "regression_found", "human_correction"} {
		if !strings.Contains(refs[i].Ref, "/"+want+"/") {
			t.Errorf("refs[%d] 应对应 %s,得到 %q", i, want, refs[i].Ref)
		}
	}
}

// AC 边界守卫:运行时策略匹配面(engineeringpolicy)不得导入 outcome ——
// 治理经投影桥读事件,匹配面与 Outcome 零耦合。静态源码扫描(与 arch 守卫
// 同型;本包内测试避免触碰 test/**)。
func TestEngineeringPolicyMustNotImportOutcome(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("无法定位测试源文件")
	}
	dir := filepath.Join(filepath.Dir(thisFile), "..", "engineeringpolicy")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读 engineeringpolicy 目录: %v", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("读 %s: %v", e.Name(), err)
		}
		if strings.Contains(string(raw), OutcomeImportBoundary) {
			t.Errorf("%s 导入了 %s:运行时策略匹配面不得直接查询 Outcome(AC)", e.Name(), OutcomeImportBoundary)
		}
	}
}
