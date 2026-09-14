// replay_test.go 拥有证据边界重放的测试(P3-04 验证面):匹配三值(含
// insufficient_evidence 一等)、差量与逐键结局(只认录得事实)、冲突可见、
// 风险信号保真与确定性 AC(同快照重放恒等,双跑核对)。
package policygovernance

import (
	"reflect"
	"testing"

	"tally/internal/engineeringpolicy"
	"tally/internal/risk"
)

func replayCandidate(id, enforcement string, requires, allOf []string) engineeringpolicy.Policy {
	pol := engineeringpolicy.Policy{
		ID:          id,
		Enforcement: engineeringpolicy.Enforcement(enforcement),
		Trigger:     engineeringpolicy.Trigger{RiskSignals: []string{"schema_migration"}},
		Requires:    requires,
	}
	if enforcement == "gate" {
		pol.GateCondition = engineeringpolicy.GateCondition{AllOf: allOf}
	}
	return pol
}

func replayRiskWithUnknown(t *testing.T, unknownID string) engineeringpolicy.MatchInput {
	t.Helper()
	res, err := risk.Derive([]risk.Signal{
		{ID: risk.SignalSchemaMigration, Basis: []string{"intent:t1"}},
		{ID: risk.SignalID(unknownID), Basis: []string{"intent:t1"}},
	})
	if err != nil {
		t.Fatalf("risk.Derive: %v", err)
	}
	return engineeringpolicy.MatchInput{Risk: res}
}

func replayRisk(t *testing.T, extra ...risk.SignalID) engineeringpolicy.MatchInput {
	t.Helper()
	sigs := []risk.Signal{{ID: risk.SignalSchemaMigration, Basis: []string{"intent:t1"}}}
	for _, id := range extra {
		sigs = append(sigs, risk.Signal{ID: id, Basis: []string{"intent:t1"}})
	}
	res, err := risk.Derive(sigs)
	if err != nil {
		t.Fatalf("risk.Derive: %v", err)
	}
	return engineeringpolicy.MatchInput{Risk: res}
}

// 主链:候选命中(schema_migration)、新增门禁键 n1(历史未要求)、录得
// passed → would_have_passed;新增键 k9 无录得 → insufficient_evidence 一等。
func TestReplayMatchedWithDeltasAndOutcomes(t *testing.T) {
	in := ReplayInput{
		Candidate:       replayCandidate("CAND", "gate", []string{"fresh_database", "n1_upgrade"}, []string{"fresh_database", "n1_upgrade"}),
		Historical:      []engineeringpolicy.Policy{replayCandidate("HIST", "gate", []string{"fresh_database"}, []string{"fresh_database"})},
		TriggerRecorded: true,
		Match:           replayRisk(t),
		RecordedGateResults: map[string]string{
			"fresh_database": "passed",
			"n1_upgrade":     "passed",
		},
	}
	got, err := ReplayCandidate(in)
	if err != nil {
		t.Fatalf("ReplayCandidate: %v", err)
	}
	if got.Decision != ReplayMatched {
		t.Fatalf("Decision = %s, want matched", got.Decision)
	}
	if !reflect.DeepEqual(got.NewlyRequiredKeys, []string{"n1_upgrade"}) {
		t.Fatalf("NewlyRequiredKeys = %v, want [n1_upgrade]", got.NewlyRequiredKeys)
	}
	if !reflect.DeepEqual(got.UnchangedKeys, []string{"fresh_database"}) {
		t.Fatalf("UnchangedKeys = %v", got.UnchangedKeys)
	}
	if len(got.GateOutcomes) != 1 || got.GateOutcomes[0].Status != GateWouldPass || got.GateOutcomes[0].GateKey != "n1_upgrade" {
		t.Fatalf("GateOutcomes = %+v", got.GateOutcomes)
	}
	// 候选与历史 gate 同触发轴:叠加是合法语义,但作为 info 冲突发现可见(P3-05)。
	if len(got.Conflicts) != 1 || got.Conflicts[0].Kind != ConflictStackedGates {
		t.Fatalf("重放应呈现候选与历史 gate 的叠加发现: %+v", got.Conflicts)
	}
}

// 证据不足一等:门禁键无录得结局 → insufficient_evidence(不是错误、不是
// 静默省略,更不是猜测的 pass/fail)。
func TestReplayInsufficientGateEvidence(t *testing.T) {
	in := ReplayInput{
		Candidate:       replayCandidate("CAND", "gate", []string{"fresh_database", "k9"}, []string{"fresh_database", "k9"}),
		Historical:      []engineeringpolicy.Policy{replayCandidate("HIST", "gate", []string{"fresh_database"}, []string{"fresh_database"})},
		TriggerRecorded: true,
		Match:           replayRisk(t),
		RecordedGateResults: map[string]string{
			"fresh_database": "passed",
			// k9 无录得。
		},
	}
	got, err := ReplayCandidate(in)
	if err != nil {
		t.Fatalf("ReplayCandidate: %v", err)
	}
	var found bool
	for _, o := range got.GateOutcomes {
		if o.GateKey == "k9" {
			found = true
			if o.Status != GateInsufficientEvidence {
				t.Fatalf("k9 结局 = %s, want insufficient_evidence", o.Status)
			}
		}
	}
	if !found {
		t.Fatalf("k9 结局必须显式呈现(一等),得到 %+v", got.GateOutcomes)
	}
}

// 录得 failed → would_have_failed(录得事实允许的确定性判定)。
func TestReplayRecordedFail(t *testing.T) {
	in := ReplayInput{
		Candidate:           replayCandidate("CAND", "gate", []string{"k9"}, []string{"k9"}),
		TriggerRecorded:     true,
		Match:               replayRisk(t),
		RecordedGateResults: map[string]string{"k9": "failed"},
	}
	got, err := ReplayCandidate(in)
	if err != nil {
		t.Fatalf("ReplayCandidate: %v", err)
	}
	if len(got.GateOutcomes) != 1 || got.GateOutcomes[0].Status != GateWouldFail {
		t.Fatalf("GateOutcomes = %+v, want would_have_failed", got.GateOutcomes)
	}
}

// 触发轴未录得 → 匹配判定 INSUFFICIENT(不猜 matched/unmatched)。
func TestReplayTriggerNotRecorded(t *testing.T) {
	in := ReplayInput{
		Candidate:  replayCandidate("CAND", "gate", []string{"k1"}, []string{"k1"}),
		Historical: []engineeringpolicy.Policy{replayCandidate("HIST", "gate", []string{"k1"}, []string{"k1"})},
	}
	got, err := ReplayCandidate(in)
	if err != nil {
		t.Fatalf("ReplayCandidate: %v", err)
	}
	if got.Decision != ReplayInsufficient {
		t.Fatalf("Decision = %s, want insufficient_evidence", got.Decision)
	}
	if len(got.GateOutcomes) != 0 || len(got.NewlyRequiredKeys) != 0 {
		t.Fatalf("未录得触发轴时不得给出差量/结局(证据边界)")
	}
}

// 未命中:录得的触发轴与候选不交 → unmatched(合法结论,非证据不足)。
func TestReplayUnmatched(t *testing.T) {
	// 候选只触发 protected_surface_change,而录得轴只有 schema_migration →
	// 录得事实支持「未命中」的确定性结论。
	cand := replayCandidate("CAND", "gate", []string{"k1"}, []string{"k1"})
	cand.Trigger.RiskSignals = []string{"protected_surface_change"}
	in := ReplayInput{
		Candidate:       cand,
		TriggerRecorded: true,
		Match:           replayRisk(t),
	}
	got, err := ReplayCandidate(in)
	if err != nil {
		t.Fatalf("ReplayCandidate: %v", err)
	}
	if got.Decision != ReplayUnmatched {
		t.Fatalf("Decision = %s, want unmatched", got.Decision)
	}
}

// 风险信号保真:unknown 信号原样进入输出(INV-V14-09 同纪律)。
func TestReplayUnknownSignalFidelity(t *testing.T) {
	in := ReplayInput{
		Candidate:       replayCandidate("CAND", "guidance", []string{"k1"}, nil),
		TriggerRecorded: true,
		Match:           replayRiskWithUnknown(t, "audit_trail_missing"),
	}
	got, err := ReplayCandidate(in)
	if err != nil {
		t.Fatalf("ReplayCandidate: %v", err)
	}
	if !reflect.DeepEqual(got.UnknownRiskSignals, []string{"audit_trail_missing"}) {
		t.Fatalf("UnknownRiskSignals = %v, want [audit_trail_missing]", got.UnknownRiskSignals)
	}
}

// 冲突可见:候选与历史集合同 id 冲突(身份冲突)出现在重放发现里。
func TestReplaySurfacesConflicts(t *testing.T) {
	in := ReplayInput{
		Candidate:       replayCandidate("HIST", "gate", []string{"k1", "k9"}, []string{"k1", "k9"}),
		Historical:      []engineeringpolicy.Policy{replayCandidate("HIST", "gate", []string{"k1"}, []string{"k1"})},
		TriggerRecorded: true,
		Match:           replayRisk(t),
	}
	got, err := ReplayCandidate(in)
	if err != nil {
		t.Fatalf("ReplayCandidate: %v", err)
	}
	if len(got.Conflicts) == 0 {
		t.Fatal("同 id 冲突应出现在重放发现里")
	}
}

// 确定性 AC:同一证据快照重放两次,输出恒等。
func TestReplayDeterministic(t *testing.T) {
	in := ReplayInput{
		Candidate:           replayCandidate("CAND", "gate", []string{"fresh_database", "n1_upgrade", "k9"}, []string{"fresh_database", "n1_upgrade"}),
		Historical:          []engineeringpolicy.Policy{replayCandidate("HIST", "gate", []string{"fresh_database"}, []string{"fresh_database"})},
		TriggerRecorded:     true,
		Match:               replayRisk(t),
		RecordedGateResults: map[string]string{"n1_upgrade": "passed"},
	}
	a, err := ReplayCandidate(in)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	b, err := ReplayCandidate(in)
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("同一证据快照重放必须恒等(AC)")
	}
}

// 无效候选 fail-closed。
func TestReplayInvalidCandidate(t *testing.T) {
	bad := replayCandidate("CAND", "guidance", []string{"k1"}, nil)
	bad.GateCondition = engineeringpolicy.GateCondition{AllOf: []string{"k1"}} // guidance 携带 gate_condition 非法
	in := ReplayInput{Candidate: bad, TriggerRecorded: true}
	if _, err := ReplayCandidate(in); err == nil {
		t.Fatal("无效候选应拒绝")
	}
}
