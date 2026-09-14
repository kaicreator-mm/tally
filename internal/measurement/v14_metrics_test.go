package measurement

// v14_metrics_test.go(P3-06)覆盖 §31 闭环产品指标投影的验收标准:
//
//   - AC1:指标暴露样本数与缺失数据(充分性标注、MissingIntent、
//     UnboundAccepted、命名缺口);缺失绝不与 0 混同。
//   - AC2:Human 查看 Diff 不被计为介入(diff review 计时与不带豁免的
//     approve/reject 都不产生介入计数,也不打破 hands-off)。
//   - AC3:manual repair / rerun / waiver / clarification 等介入种类互相
//     可区分;无事实源的种类以命名缺口显式不可用,不合并、不补 0。
//   - AC4:14-day rework 用 outcome/取代事实派生,窗口 typed,归因局限
//     显式文档化。
//
// 全部投影为纯函数:同一输入断言逐字段相等(确定性),不触库。

import (
	"reflect"
	"testing"
	"time"

	"tally/internal/outcome"
)

var v14Base = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func v14At(days, hours int) time.Time {
	return v14Base.Add(time.Duration(days)*24*time.Hour + time.Duration(hours)*time.Hour)
}

func v14Event(kind outcome.Kind, taskID, cs string, at time.Time) outcome.Event {
	return outcome.Event{TaskID: taskID, ChangesetID: cs, Kind: kind, RecordedAt: at}
}

// v14AcceptedTask 是一个 hands-off 完成的最小任务:intent → run → accepted。
func v14AcceptedTask(id string) V14TaskLoopFacts {
	return V14TaskLoopFacts{
		TaskID:   id,
		IntentAt: v14Base,
		Runs:     []V14RunFact{{ID: 1, StartedAt: v14Base.Add(time.Hour)}},
		Outcomes: []outcome.Event{v14Event(outcome.KindAccepted, id, "cs-"+id, v14At(1, 0))},
	}
}

// ---- AC1:样本数与缺失数据 ----

func TestV14MetricsEmptyInputHasGapsAndNoRates(t *testing.T) {
	m, err := BuildV14LoopMetrics(nil, v14Base)
	if err != nil {
		t.Fatalf("空输入不应报错: %v", err)
	}
	if m.Tasks != 0 || m.TasksInScope != 0 {
		t.Errorf("空输入任务数应为 0,得到 %d/%d", m.Tasks, m.TasksInScope)
	}
	if m.HandsOff.RateValid {
		t.Errorf("无 in-scope 任务时 hands-off 比率必须未定义,不得补 0")
	}
	if m.Interventions.PerAcceptedChangeValid {
		t.Errorf("无 accepted change 时介入比率必须未定义,不得补 0")
	}
	if m.Rework.RateValid {
		t.Errorf("无 accepted change 时 rework 比率必须未定义,不得补 0")
	}
	if m.TimeToAccepted.Minutes.Count != 0 || m.TimeToAccepted.Minutes.Sufficient {
		t.Errorf("无样本的分布应为 n=0 且 insufficient,得到 %+v", m.TimeToAccepted.Minutes)
	}
	if m.HandsOff.Samples.Sufficient || m.Interventions.Samples.Sufficient {
		t.Errorf("空输入的充分性必须是 insufficient")
	}
	// 结构性命名缺口固定在账(AC1/AC3):缺失显式呈现,不靠省略。
	if len(m.Gaps) != 4 {
		t.Fatalf("结构性缺口应为 4 条,得到 %d", len(m.Gaps))
	}
	wantGaps := []V14DataGapKind{
		GapRunInitiatorNotCaptured, GapClarificationNotCaptured,
		GapTradeOffDecisionNotCaptured, GapProductionIncidentNotCaptured,
	}
	for i, g := range m.Gaps {
		if g.Kind != wantGaps[i] {
			t.Errorf("gaps[%d] = %q,应为 %q", i, g.Kind, wantGaps[i])
		}
		if g.Detail == "" {
			t.Errorf("缺口 %s 缺少事实性说明", g.Kind)
		}
	}
}

func TestV14MetricsSampleSufficiencyFollowsCount(t *testing.T) {
	one := []V14TaskLoopFacts{v14AcceptedTask("t1")}
	m1, err := BuildV14LoopMetrics(one, v14Base)
	if err != nil {
		t.Fatalf("单任务投影不应报错: %v", err)
	}
	if m1.HandsOff.Samples.Count != 1 || m1.HandsOff.Samples.Sufficient {
		t.Errorf("单样本应 n=1 insufficient,得到 %+v", m1.HandsOff.Samples)
	}
	two := append(one, v14AcceptedTask("t2"))
	m2, err := BuildV14LoopMetrics(two, v14Base)
	if err != nil {
		t.Fatalf("双任务投影不应报错: %v", err)
	}
	if m2.HandsOff.Samples.Count != 2 || !m2.HandsOff.Samples.Sufficient {
		t.Errorf("双样本应 n=2 sufficient,得到 %+v", m2.HandsOff.Samples)
	}
	if !m2.HandsOff.RateValid || m2.HandsOff.Rate != 1 {
		t.Errorf("两个 hands-off 任务的比率应为 1,得到 %v(valid=%v)", m2.HandsOff.Rate, m2.HandsOff.RateValid)
	}
	if m2.TimeToAccepted.Minutes.Count != 2 {
		t.Errorf("两个任务的 time-to-accepted 样本应为 2,得到 %d", m2.TimeToAccepted.Minutes.Count)
	}
}

func TestV14MetricsMissingIntentAndUnboundAcceptedAreExplicit(t *testing.T) {
	// 有 accepted 事实但缺 intent 起点:不进 time-to-accepted 样本,也不把
	// 时长补成 0;accepted 事件缺 changeset 绑定:进分母但显式计数。
	task := V14TaskLoopFacts{
		TaskID:   "t-no-intent",
		Runs:     []V14RunFact{{ID: 7, StartedAt: v14Base}},
		Outcomes: []outcome.Event{v14Event(outcome.KindAccepted, "t-no-intent", "", v14At(2, 0))},
	}
	m, err := BuildV14LoopMetrics([]V14TaskLoopFacts{task}, v14Base)
	if err != nil {
		t.Fatalf("投影不应报错: %v", err)
	}
	if m.TimeToAccepted.MissingIntent != 1 {
		t.Errorf("MissingIntent 应为 1,得到 %d", m.TimeToAccepted.MissingIntent)
	}
	if m.TimeToAccepted.Minutes.Count != 0 {
		t.Errorf("缺 intent 的任务不得混入样本,得到 n=%d", m.TimeToAccepted.Minutes.Count)
	}
	if m.Rework.UnboundAccepted != 1 {
		t.Errorf("UnboundAccepted 应为 1,得到 %d", m.Rework.UnboundAccepted)
	}
	if m.Rework.AcceptedChanges != 1 || !m.Rework.RateValid {
		t.Errorf("缺绑定的 accepted 仍应进 rework 分母,得到 %+v", m.Rework)
	}
}

func TestV14MetricsOutOfScopeTaskExcludedFromDenominator(t *testing.T) {
	// 只有基线期输入(无 Run、无 outcome)的任务不构成 §31.1 样本。
	ghost := V14TaskLoopFacts{TaskID: "ghost", IntentAt: v14Base}
	m, err := BuildV14LoopMetrics([]V14TaskLoopFacts{ghost, v14AcceptedTask("t1")}, v14Base)
	if err != nil {
		t.Fatalf("投影不应报错: %v", err)
	}
	if m.Tasks != 2 || m.TasksInScope != 1 {
		t.Errorf("Tasks/TasksInScope 应为 2/1,得到 %d/%d", m.Tasks, m.TasksInScope)
	}
	if m.HandsOff.Rate != 1 {
		t.Errorf("分母只含 in-scope 任务,比率应为 1,得到 %v", m.HandsOff.Rate)
	}
}

// ---- AC2:Human 查看 Diff 不是介入 ----

func TestV14DiffViewingAndPlainRulingsAreNotInterventions(t *testing.T) {
	// 评审动作(view diff + 不带豁免的 approve/reject)不是 §31.3 介入,
	// 也不打破 hands-off;diff review 计时(EffortDiffReview)根本不是本
	// 投影的输入类型,不可能混入。
	task := V14TaskLoopFacts{
		TaskID:   "t-review-only",
		IntentAt: v14Base,
		Runs:     []V14RunFact{{ID: 3, StartedAt: v14Base.Add(time.Hour)}},
		Approvals: []V14ApprovalFact{
			{TaskID: "t-review-only", ChangesetID: "cs-1", Decision: V14DecisionApprove, DecidedAt: v14At(1, 2)},
			{TaskID: "t-review-only", ChangesetID: "cs-2", Decision: V14DecisionReject, DecidedAt: v14At(1, 3)},
		},
		Outcomes: []outcome.Event{
			v14Event(outcome.KindAccepted, "t-review-only", "cs-1", v14At(1, 0)),
			// diff_review 计时事实即使被误投进 Outcomes 也必须被校验拒绝,
			// 而不是被当成 outcome 事件计数(封闭枚举防线)。
		},
	}
	m, err := BuildV14LoopMetrics([]V14TaskLoopFacts{task}, v14Base)
	if err != nil {
		t.Fatalf("投影不应报错: %v", err)
	}
	for _, k := range v14InterventionKinds {
		if got := m.Interventions.ByKind[k].Count; got != 0 {
			t.Errorf("纯评审任务 %s 介入计数应为 0,得到 %s=%d", task.TaskID, k, got)
		}
	}
	if m.Interventions.CapturedTotal != 0 {
		t.Errorf("CapturedTotal 应为 0,得到 %d", m.Interventions.CapturedTotal)
	}
	if m.HandsOff.HandsOffAccepted != 1 {
		t.Errorf("查看 diff + 裁决的任务仍算 hands-off 完成,得到 %d", m.HandsOff.HandsOffAccepted)
	}
}

func TestV14HumanCorrectionCountsAsInterventionAndBreaksHandsOff(t *testing.T) {
	task := v14AcceptedTask("t-corrected")
	task.Outcomes = append(task.Outcomes,
		v14Event(outcome.KindHumanCorrection, "t-corrected", "cs-t-corrected", v14At(0, 12)))
	m, err := BuildV14LoopMetrics([]V14TaskLoopFacts{task}, v14Base)
	if err != nil {
		t.Fatalf("投影不应报错: %v", err)
	}
	if got := m.Interventions.ByKind[V14InterventionDirectCodeCorrection].Count; got != 1 {
		t.Errorf("human_correction 应计为 direct_code_correction=1,得到 %d", got)
	}
	if m.Interventions.CapturedTotal != 1 {
		t.Errorf("CapturedTotal 应为 1,得到 %d", m.Interventions.CapturedTotal)
	}
	if m.HandsOff.HandsOffAccepted != 0 {
		t.Errorf("直接改代码打破 hands-off,得到 HandsOffAccepted=%d", m.HandsOff.HandsOffAccepted)
	}
	if m.HandsOff.NotAccepted != 0 || m.HandsOff.AcceptedTasks != 1 {
		t.Errorf("打破 hands-off 不改变验收终点计数,得到 %+v", m.HandsOff)
	}
}

// ---- AC3:介入种类互相可区分,无事实源的种类显式不可用 ----

func TestV14InterventionKindsDistinguishableWithNamedGaps(t *testing.T) {
	task := V14TaskLoopFacts{
		TaskID:   "t-mixed",
		IntentAt: v14Base,
		Runs:     []V14RunFact{{ID: 9, StartedAt: v14Base.Add(time.Hour)}},
		Approvals: []V14ApprovalFact{
			{TaskID: "t-mixed", Decision: V14DecisionApprove, WaivedIndependence: true, DecidedAt: v14At(1, 1)},
			// 不带豁免的决定不产生 waiver 计数。
			{TaskID: "t-mixed", Decision: V14DecisionApprove, DecidedAt: v14At(1, 2)},
		},
		Outcomes: []outcome.Event{
			v14Event(outcome.KindAccepted, "t-mixed", "cs-a", v14At(1, 0)),
			v14Event(outcome.KindHumanCorrection, "t-mixed", "cs-a", v14At(1, 5)),
		},
	}
	m, err := BuildV14LoopMetrics([]V14TaskLoopFacts{task}, v14Base)
	if err != nil {
		t.Fatalf("投影不应报错: %v", err)
	}
	if len(m.Interventions.ByKind) != len(v14InterventionKinds) {
		t.Fatalf("介入词表应全量出现,得到 %d/%d", len(m.Interventions.ByKind), len(v14InterventionKinds))
	}
	// 有事实源的种类按事实计数。
	if got := m.Interventions.ByKind[V14InterventionWaiver]; got.Count != 1 || !got.FactSourceCaptured || got.Gap != "" {
		t.Errorf("waiver 应 count=1/captured,得到 %+v", got)
	}
	if got := m.Interventions.ByKind[V14InterventionDirectCodeCorrection]; got.Count != 1 || !got.FactSourceCaptured {
		t.Errorf("direct_code_correction 应 count=1/captured,得到 %+v", got)
	}
	// 无事实源的种类:各自独立、各自命名缺口,count=0 是「未捕获」不是 0 次。
	wantGaps := map[V14InterventionKind]V14DataGapKind{
		V14InterventionManualRepair:     GapRunInitiatorNotCaptured,
		V14InterventionManualRerun:      GapRunInitiatorNotCaptured,
		V14InterventionForcedRetry:      GapRunInitiatorNotCaptured,
		V14InterventionClarification:    GapClarificationNotCaptured,
		V14InterventionTradeOffDecision: GapTradeOffDecisionNotCaptured,
	}
	for k, wantGap := range wantGaps {
		got := m.Interventions.ByKind[k]
		if got.Count != 0 {
			t.Errorf("%s 无事实源却出现计数 %d(缺失被当成了 0 次观测)", k, got.Count)
		}
		if got.FactSourceCaptured {
			t.Errorf("%s 的 FactSourceCaptured 应为 false", k)
		}
		if got.Gap != wantGap {
			t.Errorf("%s 应指向缺口 %q,得到 %q", k, wantGap, got.Gap)
		}
	}
	if m.Interventions.CapturedTotal != 2 {
		t.Errorf("CapturedTotal 只含已捕获种类,应为 2,得到 %d", m.Interventions.CapturedTotal)
	}
	if !m.Interventions.PerAcceptedChangeValid || m.Interventions.PerAcceptedChange != 2 {
		t.Errorf("2 次介入 / 1 个 accepted change 应为 2,得到 %v", m.Interventions.PerAcceptedChange)
	}
}

// ---- repair attempts(决策锁 8/23 的事实形态;原因互相可区分) ----

func TestV14RepairAttemptsFromReworkSupersessions(t *testing.T) {
	task := V14TaskLoopFacts{
		TaskID:   "t-repairs",
		IntentAt: v14Base,
		Runs:     []V14RunFact{{ID: 1, StartedAt: v14Base}},
		Outcomes: []outcome.Event{v14Event(outcome.KindAccepted, "t-repairs", "cs-final", v14At(2, 0))},
		Supersessions: []V14SupersessionFact{
			{Old: "cs-1", New: "cs-2", Reason: V14SupersedeRework, At: v14At(0, 10)},
			{Old: "cs-2", New: "cs-3", Reason: V14SupersedeRework, At: v14At(1, 10)},
			{Old: "cs-3", New: "cs-final", Reason: V14SupersedeSpecDefect, At: v14At(1, 20)},
			{Old: "cs-old", New: "cs-new", Reason: V14SupersedeEscalation, At: v14At(0, 5)},
		},
	}
	m, err := BuildV14LoopMetrics([]V14TaskLoopFacts{task}, v14Base)
	if err != nil {
		t.Fatalf("投影不应报错: %v", err)
	}
	if m.Repairs.Total != 2 || m.Repairs.TasksWithRepairs != 1 {
		t.Errorf("repair attempts 应为 2(reason=rework),得到 total=%d tasks=%d", m.Repairs.Total, m.Repairs.TasksWithRepairs)
	}
	// 三种原因互相可区分;escalation 不计入 repair attempts。
	if m.Repairs.ByReason[V14SupersedeRework] != 2 ||
		m.Repairs.ByReason[V14SupersedeSpecDefect] != 1 ||
		m.Repairs.ByReason[V14SupersedeEscalation] != 1 {
		t.Errorf("取代原因计数应可区分,得到 %v", m.Repairs.ByReason)
	}
	if m.Repairs.PerTask.Count != 1 || m.Repairs.PerTask.Max != 2 {
		t.Errorf("per-task repair 分布应 n=1 max=2,得到 %+v", m.Repairs.PerTask)
	}
}

// ---- AC4:14-day rework(窗口 typed、归因局限文档化) ----

func TestV14ReworkWindowIsTypedAndDocumented(t *testing.T) {
	if V14ReworkWindow != 14*24*time.Hour {
		t.Errorf("rework 窗口应恒为 14 天,得到 %v", V14ReworkWindow)
	}
	if V14ReworkAttributionNote == "" {
		t.Fatal("rework 归因局限声明不得为空(AC4)")
	}
	m, err := BuildV14LoopMetrics(nil, v14Base)
	if err != nil {
		t.Fatalf("投影不应报错: %v", err)
	}
	if m.Rework.Window != V14ReworkWindow {
		t.Errorf("投影的窗口应携带 typed 常量,得到 %v", m.Rework.Window)
	}
}

func TestV14ReworkInsideAndOutsideWindow(t *testing.T) {
	inside := V14TaskLoopFacts{
		TaskID:   "t-inside",
		IntentAt: v14Base,
		Runs:     []V14RunFact{{ID: 1, StartedAt: v14Base}},
		Outcomes: []outcome.Event{v14Event(outcome.KindAccepted, "t-inside", "cs-i", v14Base)},
		Supersessions: []V14SupersessionFact{
			{Old: "cs-i", New: "cs-i2", Reason: V14SupersedeRework, At: v14At(13, 23)},
		},
	}
	outside := V14TaskLoopFacts{
		TaskID:   "t-outside",
		IntentAt: v14Base,
		Runs:     []V14RunFact{{ID: 2, StartedAt: v14Base}},
		Outcomes: []outcome.Event{
			v14Event(outcome.KindAccepted, "t-outside", "cs-o", v14Base),
			// 验收前发生的取代是闭环内正常 repair,不是已验收 change 的 rework。
			v14Event(outcome.KindRegressionFound, "t-outside", "cs-early", v14At(0, 1)),
		},
		Supersessions: []V14SupersessionFact{
			{Old: "cs-o", New: "cs-o2", Reason: V14SupersedeRework, At: v14At(15, 0)},
		},
	}
	m, err := BuildV14LoopMetrics([]V14TaskLoopFacts{inside, outside}, v14Base)
	if err != nil {
		t.Fatalf("投影不应报错: %v", err)
	}
	if m.Rework.AcceptedChanges != 2 {
		t.Fatalf("分母应为 2 个 accepted change,得到 %d", m.Rework.AcceptedChanges)
	}
	if m.Rework.ReworkedChanges != 1 {
		t.Errorf("窗口内 1 例、窗口外 0 例,reworked 应为 1,得到 %d", m.Rework.ReworkedChanges)
	}
	if m.Rework.Rate != 0.5 || !m.Rework.RateValid {
		t.Errorf("rework rate 应为 0.5,得到 %v(valid=%v)", m.Rework.Rate, m.Rework.RateValid)
	}
	if got := m.Rework.SignalCounts[V14ReworkSupersededRework]; got != 1 {
		t.Errorf("superseded_rework 信号应计 1,得到 %d", got)
	}
	// 验收前的取代照常计入 repair attempts(与 rework 判定互不吞并)。
	if m.Repairs.Total != 2 {
		t.Errorf("两条 rework 取代事实都应计入 repair attempts,得到 %d", m.Repairs.Total)
	}
}

func TestV14ReworkSignalsFromOutcomeEvents(t *testing.T) {
	task := V14TaskLoopFacts{
		TaskID:   "t-signals",
		IntentAt: v14Base,
		Runs:     []V14RunFact{{ID: 1, StartedAt: v14Base}},
		Outcomes: []outcome.Event{
			v14Event(outcome.KindAccepted, "t-signals", "cs-1", v14Base),
			v14Event(outcome.KindAccepted, "t-signals", "cs-2", v14At(2, 0)),
			v14Event(outcome.KindRegressionFound, "t-signals", "cs-1", v14At(3, 0)),
			v14Event(outcome.KindHotfixed, "t-signals", "cs-1", v14At(4, 0)),
			// 仅任务绑定(缺 changeset):按 §31.4 WorkItem 级归因到最近一次
			// 先于事件的验收(这里归 cs-2)。
			v14Event(outcome.KindReverted, "t-signals", "", v14At(5, 0)),
			// 同为任务绑定,归到更早的 cs-1。
			v14Event(outcome.KindRegressionFound, "t-signals", "", v14At(1, 0)),
			// 早于任何验收的任务绑定事件:无法归因,显式计数,不静默丢弃。
			v14Event(outcome.KindRegressionFound, "t-signals", "", v14Base.Add(-2*time.Hour)),
			// human_correction 是介入事实,不归 rework 信号。
			v14Event(outcome.KindHumanCorrection, "t-signals", "cs-1", v14At(2, 1)),
		},
	}
	m, err := BuildV14LoopMetrics([]V14TaskLoopFacts{task}, v14Base)
	if err != nil {
		t.Fatalf("投影不应报错: %v", err)
	}
	// cs-1(回归+热修)与 cs-2(回退)各自成为一例 rework。
	if m.Rework.ReworkedChanges != 2 {
		t.Errorf("两个 accepted change 各自出现再修改信号,应为 2,得到 %d", m.Rework.ReworkedChanges)
	}
	want := map[V14ReworkSignal]int{
		V14ReworkRegressionFound:      2,
		V14ReworkHotfixed:             1,
		V14ReworkReverted:             1,
		V14ReworkSupersededRework:     0,
		V14ReworkSupersededSpecDefect: 0,
	}
	for s, n := range want {
		if got := m.Rework.SignalCounts[s]; got != n {
			t.Errorf("信号 %s 应为 %d,得到 %d", s, n, got)
		}
	}
	if m.Rework.UnattributedSignals != 1 {
		t.Errorf("无法归因的信号应显式计 1,得到 %d", m.Rework.UnattributedSignals)
	}
	if got := m.Interventions.ByKind[V14InterventionDirectCodeCorrection].Count; got != 1 {
		t.Errorf("human_correction 应归 §31.3 介入而非 rework,得到 %d", got)
	}
	// slow guardrails(§31.5):同一批事实的 guardrail 视角按事件类别全量
	// 计数(含窗口外与无法归因的事件 —— guardrail 只问事实发生过)。
	if m.Guardrails.ByKind[outcome.KindRegressionFound].Count != 3 ||
		m.Guardrails.ByKind[outcome.KindHotfixed].Count != 1 ||
		m.Guardrails.ByKind[outcome.KindReverted].Count != 1 {
		t.Errorf("guardrail 计数应按事件类别可区分,得到 %v", m.Guardrails.ByKind)
	}
	if !m.Guardrails.ByKind[outcome.KindRegressionFound].Samples.Sufficient {
		t.Errorf("guardrail 计数应带样本充分性(n=2 sufficient)")
	}
}

func TestV14SpecDefectSupersessionIsReworkSignal(t *testing.T) {
	task := V14TaskLoopFacts{
		TaskID:   "t-spec",
		IntentAt: v14Base,
		Runs:     []V14RunFact{{ID: 1, StartedAt: v14Base}},
		Outcomes: []outcome.Event{v14Event(outcome.KindAccepted, "t-spec", "cs-s", v14Base)},
		Supersessions: []V14SupersessionFact{
			{Old: "cs-s", New: "cs-s2", Reason: V14SupersedeSpecDefect, At: v14At(7, 0)},
			// escalation 是升级重做,不是已验收 change 的缺陷返工。
			{Old: "cs-s", New: "cs-s3", Reason: V14SupersedeEscalation, At: v14At(8, 0)},
		},
	}
	m, err := BuildV14LoopMetrics([]V14TaskLoopFacts{task}, v14Base)
	if err != nil {
		t.Fatalf("投影不应报错: %v", err)
	}
	if m.Rework.ReworkedChanges != 1 || m.Rework.SignalCounts[V14ReworkSupersededSpecDefect] != 1 {
		t.Errorf("spec_defect 取代应是 rework 信号,得到 %+v", m.Rework)
	}
	if m.Rework.SignalCounts[V14ReworkSupersededRework] != 0 {
		t.Errorf("escalation 不得混入 rework 信号,得到 %d", m.Rework.SignalCounts[V14ReworkSupersededRework])
	}
}

// ---- 确定性(同事实 → 同数字) ----

func TestV14MetricsDeterministic(t *testing.T) {
	tasks := []V14TaskLoopFacts{v14AcceptedTask("t1"), v14AcceptedTask("t2")}
	a, err := BuildV14LoopMetrics(tasks, v14Base)
	if err != nil {
		t.Fatalf("投影不应报错: %v", err)
	}
	b, err := BuildV14LoopMetrics(tasks, v14Base)
	if err != nil {
		t.Fatalf("投影不应报错: %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Error("同一事实的两次投影必须逐字段相等")
	}
}

// ---- 输入校验:映射错误一律报错,不静默归类 ----

func TestV14MetricsInputValidation(t *testing.T) {
	valid := v14AcceptedTask("t1")
	cases := []struct {
		name  string
		tasks []V14TaskLoopFacts
	}{
		{"空 task id", []V14TaskLoopFacts{{IntentAt: v14Base}}},
		{"重复任务", []V14TaskLoopFacts{valid, v14AcceptedTask("t1")}},
		{"run 缺 id", []V14TaskLoopFacts{{TaskID: "t", Runs: []V14RunFact{{StartedAt: v14Base}}}}},
		{"run 缺开始时刻", []V14TaskLoopFacts{{TaskID: "t", Runs: []V14RunFact{{ID: 1}}}}},
		{"决定归属错任务", []V14TaskLoopFacts{{TaskID: "t", Approvals: []V14ApprovalFact{{TaskID: "other", Decision: V14DecisionApprove, DecidedAt: v14Base}}}}},
		{"非法决定", []V14TaskLoopFacts{{TaskID: "t", Approvals: []V14ApprovalFact{{TaskID: "t", Decision: "maybe", DecidedAt: v14Base}}}}},
		{"决定缺时刻", []V14TaskLoopFacts{{TaskID: "t", Approvals: []V14ApprovalFact{{TaskID: "t", Decision: V14DecisionApprove}}}}},
		{"outcome 归属错任务", []V14TaskLoopFacts{{TaskID: "t", Outcomes: []outcome.Event{v14Event(outcome.KindAccepted, "other", "cs", v14Base)}}}},
		{"outcome 非法", []V14TaskLoopFacts{{TaskID: "t", Outcomes: []outcome.Event{{TaskID: "t", Kind: "vibes", RecordedAt: v14Base}}}}},
		{"取代 old=new", []V14TaskLoopFacts{{TaskID: "t", Supersessions: []V14SupersessionFact{{Old: "cs", New: "cs", Reason: V14SupersedeRework, At: v14Base}}}}},
		{"取代非法原因", []V14TaskLoopFacts{{TaskID: "t", Supersessions: []V14SupersessionFact{{Old: "a", New: "b", Reason: "vibes", At: v14Base}}}}},
		{"取代缺时刻", []V14TaskLoopFacts{{TaskID: "t", Supersessions: []V14SupersessionFact{{Old: "a", New: "b", Reason: V14SupersedeRework}}}}},
	}
	for _, tc := range cases {
		if _, err := BuildV14LoopMetrics(tc.tasks, v14Base); err == nil {
			t.Errorf("%s:应报错却得到 nil", tc.name)
		}
	}
	// 生成时刻缺失的独立用例(nil 任务 + 零时刻)。
	if _, err := BuildV14LoopMetrics(nil, time.Time{}); err == nil {
		t.Error("缺少生成时刻:应报错却得到 nil")
	}
	// 验收早于 intent 起点:事实映射矛盾,报错。
	contradiction := V14TaskLoopFacts{
		TaskID:   "t-backwards",
		IntentAt: v14At(5, 0),
		Runs:     []V14RunFact{{ID: 1, StartedAt: v14Base}},
		Outcomes: []outcome.Event{v14Event(outcome.KindAccepted, "t-backwards", "cs", v14Base)},
	}
	if _, err := BuildV14LoopMetrics([]V14TaskLoopFacts{contradiction}, v14Base); err == nil {
		t.Error("accepted 早于 intent:应报错却得到 nil")
	}
}
