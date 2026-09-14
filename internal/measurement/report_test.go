package measurement

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

var reportBase = time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)

// reportEffort 构造一条通过校验的人工开销事实行。
func reportEffort(taskID string, kind HumanEffortKind, active, wall time.Duration) HumanEffort {
	return HumanEffort{TaskID: taskID, Kind: kind, Active: active, Wall: wall, RecordedAt: reportBase}
}

// reportUsage 构造一条通过校验的 LLM 使用事实行(用量已上报)。
func reportUsage(taskID, useCase string, plane CostPlane, costMicros int64) LLMUsage {
	u := validLLMUsage()
	u.TaskID = taskID
	u.UseCase = useCase
	u.CostPlane = plane
	u.UsageReported = true
	u.CostMicros = costMicros
	u.InputTokens = costMicros / 10
	u.OutputTokens = costMicros / 20
	u.RecordedAt = reportBase
	return u
}

// AC1:人工开销按条目 active/wall 双口径汇总;成本按写入时落成的 cost_plane
// 聚合;阶段墙钟与 idea_to_verified_latency 从事实行推导。
func TestBuildBaselineReportHumanCostPhasesLatency(t *testing.T) {
	t1 := TaskBaseline{
		TaskID:  "task-1",
		Profile: "standard",
		Efforts: []HumanEffort{
			reportEffort("task-1", EffortContractReview, 30*time.Minute, 45*time.Minute),
			reportEffort("task-1", EffortContractReview, 20*time.Minute, 15*time.Minute),
			reportEffort("task-1", EffortDiffReview, 10*time.Minute, 60*time.Minute),
		},
		Usages: []LLMUsage{
			reportUsage("task-1", "verify_contract_draft", CostPlaneControl, 100),
			reportUsage("task-1", "implement_patch", CostPlaneExecution, 300),
		},
		Spans: []PhaseSpan{
			{Phase: PhaseAgentExecution, Start: reportBase, End: reportBase.Add(30 * time.Minute)},
			{Phase: PhaseFinalValidation, Start: reportBase.Add(30 * time.Minute), End: reportBase.Add(50 * time.Minute)},
		},
		Milestones: MilestoneTimes{
			MilestoneIdeaReceived:          reportBase,
			MilestoneVerificationCompleted: reportBase.Add(2 * time.Hour),
		},
	}
	t2 := TaskBaseline{
		TaskID:  "task-2",
		Profile: "standard",
		Efforts: []HumanEffort{reportEffort("task-2", EffortManualCorrection, 90*time.Minute, 90*time.Minute)},
		Usages:  []LLMUsage{reportUsage("task-2", "implement_patch", CostPlaneExecution, 500)},
	}

	rep, err := BuildBaselineReport([]TaskBaseline{t1, t2}, reportBase.Add(time.Minute))
	if err != nil {
		t.Fatalf("BuildBaselineReport: %v", err)
	}
	if !rep.GeneratedAt.Equal(reportBase.Add(time.Minute).UTC()) {
		t.Errorf("GeneratedAt = %v, want %v", rep.GeneratedAt, reportBase.Add(time.Minute))
	}
	if rep.Tasks != 2 {
		t.Errorf("Tasks = %d, want 2", rep.Tasks)
	}

	// 人工开销:同条目分段行累加,双口径不合并。
	cr := rep.Human.ByKind[EffortContractReview]
	if cr.Active != 50*time.Minute || cr.Wall != 60*time.Minute {
		t.Errorf("contract_review = active %v wall %v, want 50m/60m", cr.Active, cr.Wall)
	}
	if cr.Samples.Count != 2 || !cr.Samples.Sufficient {
		t.Errorf("contract_review samples = %+v, want {2 true}", cr.Samples)
	}
	dr := rep.Human.ByKind[EffortDiffReview]
	if dr.Active != 10*time.Minute || dr.Wall != 60*time.Minute {
		t.Errorf("diff_review = active %v wall %v, want 10m/60m", dr.Active, dr.Wall)
	}
	if dr.Samples.Count != 1 || dr.Samples.Sufficient {
		t.Errorf("diff_review samples = %+v, want {1 false}(单样本不构成分布,AC4)", dr.Samples)
	}
	if got := rep.Human.ByKind[EffortSpecDefectDecision]; got.Active != 0 || got.Samples.Sufficient {
		t.Errorf("未观测条目应显式 0 且不足,得到 %+v", got)
	}
	if len(rep.Human.ByKind) != len(effortKinds) {
		t.Errorf("ByKind 应呈现全部 §25 条目,得到 %d 项, want %d", len(rep.Human.ByKind), len(effortKinds))
	}
	// 每任务 active 分钟样本:[60, 90]。
	pm := rep.Human.PerTaskActiveMinutes
	if pm.Count != 2 || !pm.Sufficient || pm.Min != 60 || pm.Median != 75 || pm.Max != 90 {
		t.Errorf("PerTaskActiveMinutes = %+v, want {2 true} 60/75/90", pm)
	}

	// 成本:按存储的 cost_plane 聚合,历史不重算。
	ctrl := rep.Cost.ByPlane[CostPlaneControl]
	exec := rep.Cost.ByPlane[CostPlaneExecution]
	if ctrl.CostMicros != 100 || ctrl.Calls != 1 || ctrl.InputTokens != 10 || ctrl.OutputTokens != 5 {
		t.Errorf("control plane = %+v, want cost 100 calls 1 tokens 10/5", ctrl)
	}
	if exec.CostMicros != 800 || exec.Calls != 2 {
		t.Errorf("execution plane = cost %d calls %d, want 800/2", exec.CostMicros, exec.Calls)
	}
	if !rep.Cost.RatioValid {
		t.Fatalf("总成本 > 0 时 ratio 必须有效")
	}
	if want := 100.0 / 900.0; rep.Cost.ControlRatio != want {
		t.Errorf("ControlRatio = %v, want %v", rep.Cost.ControlRatio, want)
	}
	pc := rep.Cost.PerTaskCostMicros
	if pc.Count != 2 || !pc.Sufficient || pc.Min != 400 || pc.Median != 450 || pc.Max != 500 {
		t.Errorf("PerTaskCostMicros = %+v, want {2 true} 400/450/500", pc)
	}

	// 阶段墙钟。
	ae := rep.Phases.WallByPhase[PhaseAgentExecution]
	fv := rep.Phases.WallByPhase[PhaseFinalValidation]
	if ae.Wall != 30*time.Minute || ae.Samples.Count != 1 || ae.Samples.Sufficient {
		t.Errorf("agent_execution = %+v, want wall 30m samples {1 false}", ae)
	}
	if fv.Wall != 20*time.Minute || fv.Samples.Count != 1 {
		t.Errorf("final_validation = %+v, want wall 20m samples 1", fv)
	}
	if got := rep.Phases.WallByPhase[PhaseIntake]; got.Wall != 0 || got.Samples.Count != 0 {
		t.Errorf("未观测阶段应显式 0,得到 %+v", got)
	}
	if len(rep.Phases.WallByPhase) != len(lifecyclePhases) {
		t.Errorf("WallByPhase 应呈现全部 §25 阶段,得到 %d 项, want %d", len(rep.Phases.WallByPhase), len(lifecyclePhases))
	}

	// idea_to_verified_latency:单样本 120 分钟,标记不足(AC4)。
	iv := rep.Latency.IdeaToVerifiedMinutes
	if iv.Count != 1 || iv.Sufficient || iv.Min != 120 || iv.Median != 120 || iv.Max != 120 {
		t.Errorf("IdeaToVerifiedMinutes = %+v, want {1 false} 120/120/120", iv)
	}
}

// AC2:qualification strength、authoring mode、visibility、strength×visibility
// 交叉与 acceptance 级 planned/effective independence 覆盖及缺口归因。
func TestBuildBaselineReportTrustDistributions(t *testing.T) {
	task := TaskBaseline{
		TaskID: "task-trust",
		Bindings: []BindingFact{
			{BindingID: "b1", AuthoringMode: ModePreCommit, Visibility: VisibilityVisible, Strength: StrengthAssertionStrong},
			{BindingID: "b2", AuthoringMode: ModePostHoc, Visibility: VisibilityVisible, Strength: StrengthCompileErrorWeak},
			{BindingID: "b3", AuthoringMode: ModePreCommit, Visibility: VisibilityHeldOut, Strength: StrengthNA},
			{BindingID: "b4", AuthoringMode: ModePreCommit, Visibility: VisibilityHeldOut, Strength: StrengthBaselinePass},
		},
		Acceptances: []AcceptanceFact{
			{AcceptanceID: "accA", Required: LevelSource, Covered: true},
			{AcceptanceID: "accB", Required: LevelSource, Covered: false},
			{AcceptanceID: "accC", Required: LevelInformation, Covered: true},
			// source 维度已有满足的 evidence,缺的是同一条 evidence 上的
			// information 维度(§8.5)→ information gap。
			{AcceptanceID: "accD", Required: LevelInformation, Covered: false, SourceSatisfied: true},
			{AcceptanceID: "accE", Required: LevelInformation, Covered: false, SourceSatisfied: false},
			// 未声明 = none(§7.3 缺省);none 不参与 effective 覆盖计数。
			{AcceptanceID: "accF", Covered: false},
		},
	}
	rep, err := BuildBaselineReport([]TaskBaseline{task}, reportBase)
	if err != nil {
		t.Fatalf("BuildBaselineReport: %v", err)
	}

	qs := rep.Trust.QualificationStrength
	if qs.Total != 4 || !qs.Sufficient {
		t.Fatalf("QualificationStrength = %+v, want total 4 sufficient", qs)
	}
	wantStrength := map[string]int{
		string(StrengthAssertionStrong):  1,
		string(StrengthCompileErrorWeak): 1,
		string(StrengthPanicErrorWeak):   0,
		string(StrengthErrorWeak):        0,
		string(StrengthBaselinePass):     1,
		string(StrengthNA):               1,
	}
	if !reflect.DeepEqual(qs.Counts, wantStrength) {
		t.Errorf("strength counts = %v, want %v", qs.Counts, wantStrength)
	}
	wantMode := map[string]int{string(ModePreCommit): 3, string(ModePostHoc): 1}
	if !reflect.DeepEqual(rep.Trust.AuthoringMode.Counts, wantMode) || !rep.Trust.AuthoringMode.Sufficient {
		t.Errorf("authoring mode counts = %v, want %v", rep.Trust.AuthoringMode.Counts, wantMode)
	}
	wantVis := map[string]int{string(VisibilityVisible): 2, string(VisibilityHeldOut): 2}
	if !reflect.DeepEqual(rep.Trust.Visibility.Counts, wantVis) || !rep.Trust.Visibility.Sufficient {
		t.Errorf("visibility counts = %v, want %v", rep.Trust.Visibility.Counts, wantVis)
	}
	cross := rep.Trust.StrengthByVisibility
	if cross.Total != 4 || !cross.Sufficient {
		t.Fatalf("StrengthByVisibility = total %d sufficient %v, want 4/true", cross.Total, cross.Sufficient)
	}
	if got := cross.Counts[string(StrengthAssertionStrong)][string(VisibilityVisible)]; got != 1 {
		t.Errorf("cross[assertion_strong][visible] = %d, want 1", got)
	}
	if got := cross.Counts[string(StrengthBaselinePass)][string(VisibilityHeldOut)]; got != 1 {
		t.Errorf("cross[baseline_pass][held_out] = %d, want 1", got)
	}
	if got := cross.Counts[string(StrengthAssertionStrong)][string(VisibilityHeldOut)]; got != 0 {
		t.Errorf("未观测组合应缺省 0,得到 %d", got)
	}

	wantPlanned := map[string]int{string(LevelNone): 1, string(LevelSource): 2, string(LevelInformation): 3}
	if !reflect.DeepEqual(rep.Trust.Independence.Planned.Counts, wantPlanned) {
		t.Errorf("planned counts = %v, want %v", rep.Trust.Independence.Planned.Counts, wantPlanned)
	}
	if rep.Trust.Independence.Planned.Total != 6 || !rep.Trust.Independence.Planned.Sufficient {
		t.Errorf("planned = %+v, want total 6 sufficient", rep.Trust.Independence.Planned)
	}

	src := rep.Trust.Independence.Source
	if src.Required != 2 || src.Covered != 1 || src.SourceGaps != 1 || src.InformationGaps != 0 || !src.Samples.Sufficient {
		t.Errorf("source coverage = %+v, want req 2 covered 1 source-gap 1 info-gap 0 sufficient", src)
	}
	info := rep.Trust.Independence.Information
	if info.Required != 3 || info.Covered != 1 || info.SourceGaps != 1 || info.InformationGaps != 1 || !info.Samples.Sufficient {
		t.Errorf("information coverage = %+v, want req 3 covered 1 source-gap 1 info-gap 1 sufficient", info)
	}
}

// AC3:first-pass 按 visible_only / with_holdout 分桶,数据存在才计数。
func TestBuildBaselineReportFirstPassBuckets(t *testing.T) {
	fp := func(id string, withHoldout, passed bool) TaskBaseline {
		return TaskBaseline{TaskID: id, FirstPass: &FirstPassFact{TaskID: id, WithHoldout: withHoldout, Passed: passed}}
	}
	tasks := []TaskBaseline{
		fp("t-visible-1", false, true),
		fp("t-visible-2", false, false),
		fp("t-holdout-1", true, true),
		fp("t-holdout-2", true, false),
		fp("t-holdout-3", true, true),
	}
	rep, err := BuildBaselineReport(tasks, reportBase)
	if err != nil {
		t.Fatalf("BuildBaselineReport: %v", err)
	}
	vo := rep.FirstPass.VisibleOnly
	if vo.Tasks != 2 || vo.Passed != 1 || !vo.Samples.Sufficient {
		t.Errorf("VisibleOnly = %+v, want tasks 2 passed 1 sufficient", vo)
	}
	wh := rep.FirstPass.WithHoldout
	if wh.Tasks != 3 || wh.Passed != 2 || !wh.Samples.Sufficient {
		t.Errorf("WithHoldout = %+v, want tasks 3 passed 2 sufficient", wh)
	}
}

// AC4:缺失/低样本读数标记不足而非补 0 或通过;空输入的全部维度都显式
// 「不足」,不产生任何可误读为 0/通过的结论。
func TestBuildBaselineReportInsufficientMarking(t *testing.T) {
	rep, err := BuildBaselineReport(nil, reportBase)
	if err != nil {
		t.Fatalf("BuildBaselineReport(nil): %v", err)
	}
	if rep.Tasks != 0 {
		t.Errorf("Tasks = %d, want 0", rep.Tasks)
	}
	for _, k := range effortKinds {
		if got := rep.Human.ByKind[k]; got.Samples.Count != 0 || got.Samples.Sufficient {
			t.Errorf("%s: 空输入应为 {{0 false}},得到 %+v", string(k), got.Samples)
		}
	}
	for _, p := range []CostPlane{CostPlaneControl, CostPlaneExecution} {
		if got := rep.Cost.ByPlane[p]; got.Samples.Count != 0 || got.Samples.Sufficient {
			t.Errorf("%s: 空输入应为样本不足,得到 %+v", string(p), got)
		}
	}
	if rep.Cost.RatioValid {
		t.Errorf("总成本为 0 时 ratio 未定义,RatioValid 应为 false(不猜,AC4)")
	}
	for _, ph := range lifecyclePhases {
		if got := rep.Phases.WallByPhase[ph]; got.Samples.Count != 0 || got.Samples.Sufficient {
			t.Errorf("%s: 空输入应为样本不足,得到 %+v", string(ph), got.Samples)
		}
	}
	for _, d := range []ScalarDistribution{rep.Human.PerTaskActiveMinutes, rep.Cost.PerTaskCostMicros, rep.Latency.IdeaToVerifiedMinutes} {
		if d.Count != 0 || d.Sufficient {
			t.Errorf("空输入的标量分布应为 {{0 false}},得到 %+v", d.Samples)
		}
	}
	for name, d := range map[string]CategoryDistribution{
		"strength":   rep.Trust.QualificationStrength,
		"mode":       rep.Trust.AuthoringMode,
		"visibility": rep.Trust.Visibility,
		"planned":    rep.Trust.Independence.Planned,
	} {
		if d.Total != 0 || d.Sufficient {
			t.Errorf("%s: 空输入应为不足,得到 %+v", name, d)
		}
	}
	if got := rep.Trust.QualificationStrength.Counts[string(StrengthAssertionStrong)]; got != 0 {
		t.Errorf("词表零计数取值应显式呈现 0,得到 %d", got)
	}
	if rep.Trust.StrengthByVisibility.Total != 0 || rep.Trust.StrengthByVisibility.Sufficient {
		t.Errorf("空交叉分布应为不足,得到 %+v", rep.Trust.StrengthByVisibility)
	}
	if rep.Trust.Independence.Source.Required != 0 || rep.Trust.Independence.Source.Samples.Sufficient {
		t.Errorf("空 effective coverage 应为不足,得到 %+v", rep.Trust.Independence.Source)
	}
	if rep.FirstPass.VisibleOnly.Tasks != 0 || rep.FirstPass.VisibleOnly.Samples.Sufficient ||
		rep.FirstPass.WithHoldout.Tasks != 0 || rep.FirstPass.WithHoldout.Samples.Sufficient {
		t.Errorf("空 first-pass 桶应为不足,得到 %+v / %+v", rep.FirstPass.VisibleOnly, rep.FirstPass.WithHoldout)
	}

	// 单样本:数值可呈现,但必须标记不足(个案不是分布)。
	single := TaskBaseline{
		TaskID:  "task-single",
		Efforts: []HumanEffort{reportEffort("task-single", EffortDiffReview, 42*time.Minute, 0)},
	}
	rep1, err := BuildBaselineReport([]TaskBaseline{single}, reportBase)
	if err != nil {
		t.Fatalf("BuildBaselineReport(single): %v", err)
	}
	pm := rep1.Human.PerTaskActiveMinutes
	if pm.Count != 1 || pm.Sufficient || pm.Min != 42 || pm.Median != 42 || pm.Max != 42 {
		t.Errorf("单样本分布 = %+v, want {{1 false}} 42/42/42", pm)
	}
}

// 输入校验是硬边界:映射错误报错,不静默归类、不丢行。
func TestBuildBaselineReportInputValidation(t *testing.T) {
	valid := TaskBaseline{
		TaskID:  "task-ok",
		Efforts: []HumanEffort{reportEffort("task-ok", EffortContractReview, time.Minute, time.Minute)},
		Usages:  []LLMUsage{reportUsage("task-ok", "implement_patch", CostPlaneExecution, 5)},
		Spans: []PhaseSpan{
			{Phase: PhaseAgentExecution, Start: reportBase, End: reportBase.Add(time.Minute)},
		},
		Milestones: MilestoneTimes{MilestoneIdeaReceived: reportBase},
		Bindings: []BindingFact{
			{BindingID: "b", AuthoringMode: ModePreCommit, Visibility: VisibilityVisible, Strength: StrengthAssertionStrong},
		},
		Acceptances: []AcceptanceFact{{AcceptanceID: "a", Required: LevelSource, Covered: true}},
		FirstPass:   &FirstPassFact{TaskID: "task-ok", WithHoldout: true, Passed: false},
	}

	mutate := func(name string, f func(t *TaskBaseline)) {
		in := valid
		f(&in)
		if _, err := BuildBaselineReport([]TaskBaseline{in}, reportBase); err == nil {
			t.Errorf("%s: 非法输入被接受", name)
		}
	}
	mutate("缺少 task_id", func(t *TaskBaseline) { t.TaskID = "  " })
	mutate("effort 归属任务不一致", func(t *TaskBaseline) {
		t.Efforts[0].TaskID = "task-other"
	})
	mutate("effort kind 非法", func(t *TaskBaseline) {
		t.Efforts[0].Kind = HumanEffortKind("bogus")
	})
	mutate("usage 归属任务不一致", func(t *TaskBaseline) {
		t.Usages[0].TaskID = "task-other"
	})
	mutate("usage cost_plane 非法", func(t *TaskBaseline) {
		t.Usages[0].CostPlane = CostPlane("planning")
	})
	mutate("phase span end 早于 start", func(t *TaskBaseline) {
		t.Spans[0].End = reportBase
	})
	mutate("phase span 阶段名非法", func(t *TaskBaseline) {
		t.Spans[0].Phase = LifecyclePhase("bogus")
	})
	mutate("里程碑名非法", func(t *TaskBaseline) {
		t.Milestones = MilestoneTimes{Milestone("bogus"): reportBase}
	})
	mutate("里程碑缺时刻", func(t *TaskBaseline) {
		t.Milestones = MilestoneTimes{MilestoneIdeaReceived: {}}
	})
	mutate("binding 缺 id", func(t *TaskBaseline) {
		t.Bindings[0].BindingID = ""
	})
	mutate("binding strength 非法", func(t *TaskBaseline) {
		t.Bindings[0].Strength = QualificationStrength("super_strong")
	})
	mutate("binding visibility 非法", func(t *TaskBaseline) {
		t.Bindings[0].Visibility = Visibility("secret")
	})
	mutate("binding authoring mode 非法", func(t *TaskBaseline) {
		t.Bindings[0].AuthoringMode = AuthoringMode("ad_hoc")
	})
	mutate("acceptance 缺 id", func(t *TaskBaseline) {
		t.Acceptances[0].AcceptanceID = ""
	})
	mutate("acceptance 级别非法", func(t *TaskBaseline) {
		t.Acceptances[0].Required = IndependenceLevel("absolute")
	})
	mutate("first-pass 归属任务不一致", func(t *TaskBaseline) {
		t.FirstPass.TaskID = "task-other"
	})

	if _, err := BuildBaselineReport([]TaskBaseline{valid, valid}, reportBase); err == nil {
		t.Errorf("重复任务 id 被接受")
	}
	if _, err := BuildBaselineReport([]TaskBaseline{valid}, time.Time{}); err == nil {
		t.Errorf("缺少生成时刻被接受")
	}
}

// P0-08 AC3:覆盖期间从输入事实的观测时刻推导(含 span 端点与里程碑),
// 统一呈现 UTC;没有任何带时刻的事实时期间未知(SourcePeriodValid=false),
// 不得用 GeneratedAt 伪充(AC2)。
func TestBuildBaselineReportSourcePeriod(t *testing.T) {
	plus8 := time.FixedZone("plus8", 8*60*60)
	spanStart := time.Date(2026, 9, 1, 9, 0, 0, 0, plus8)  // 01:00Z
	milestone := time.Date(2026, 9, 1, 8, 30, 0, 0, plus8) // 00:30Z,全量最早
	effortAt := time.Date(2026, 9, 2, 10, 0, 0, 0, time.UTC)
	usageAt := time.Date(2026, 9, 3, 18, 25, 0, 0, time.UTC) // 全量最晚
	eff := reportEffort("task-p", EffortContractReview, time.Minute, time.Minute)
	eff.RecordedAt = effortAt
	usg := reportUsage("task-p", "implement_patch", CostPlaneExecution, 5)
	usg.RecordedAt = usageAt
	rep, err := BuildBaselineReport([]TaskBaseline{{
		TaskID:  "task-p",
		Efforts: []HumanEffort{eff},
		Usages:  []LLMUsage{usg},
		Spans: []PhaseSpan{
			{Phase: PhaseAgentExecution, Start: spanStart, End: spanStart.Add(30 * time.Minute)},
		},
		Milestones: MilestoneTimes{MilestoneIdeaReceived: milestone},
	}}, reportBase)
	if err != nil {
		t.Fatalf("BuildBaselineReport: %v", err)
	}
	if !rep.SourcePeriodValid {
		t.Fatalf("有带时刻的事实,SourcePeriodValid 应为 true")
	}
	if want := milestone.UTC(); !rep.SourcePeriodFrom.Equal(want) {
		t.Errorf("SourcePeriodFrom = %v, want %v(含里程碑时刻且换算 UTC)", rep.SourcePeriodFrom, want)
	}
	if want := usageAt.UTC(); !rep.SourcePeriodTo.Equal(want) {
		t.Errorf("SourcePeriodTo = %v, want %v(usage recorded_at 是最晚观测)", rep.SourcePeriodTo, want)
	}

	// 无任何带时刻的事实:期间未知,显式 false。
	empty, err := BuildBaselineReport(nil, reportBase)
	if err != nil {
		t.Fatalf("BuildBaselineReport(nil): %v", err)
	}
	if empty.SourcePeriodValid {
		t.Errorf("空输入 SourcePeriodValid 应为 false(期间未知,不用 GeneratedAt 伪充)")
	}
	// 仅 first-pass/binding 等无时间戳投影:期间同样未知。
	fpOnly, err := BuildBaselineReport([]TaskBaseline{{
		TaskID:    "task-fp",
		FirstPass: &FirstPassFact{TaskID: "task-fp", Passed: true},
	}}, reportBase)
	if err != nil {
		t.Fatalf("BuildBaselineReport(fpOnly): %v", err)
	}
	if fpOnly.SourcePeriodValid {
		t.Errorf("仅 first-pass 投影不含观测时刻,SourcePeriodValid 应为 false")
	}
}

// P0-08 AC1-AC3:运维呈现是确定性的;样本数与充分性随行标注;词表全量取值
// 出现;比率未定义、无样本分布输出显式文字而不是数值;缺失呈现为
// insufficient,不得被误读为 0 或通过。
func TestRenderBaselineReport(t *testing.T) {
	t1 := TaskBaseline{
		TaskID: "task-1",
		Efforts: []HumanEffort{
			reportEffort("task-1", EffortContractReview, 30*time.Minute, 45*time.Minute),
			reportEffort("task-1", EffortContractReview, 20*time.Minute, 15*time.Minute),
			reportEffort("task-1", EffortDiffReview, 10*time.Minute, 60*time.Minute),
		},
		Usages: []LLMUsage{
			reportUsage("task-1", "verify_contract_draft", CostPlaneControl, 100),
			reportUsage("task-1", "implement_patch", CostPlaneExecution, 300),
		},
		Spans: []PhaseSpan{
			{Phase: PhaseAgentExecution, Start: reportBase, End: reportBase.Add(30 * time.Minute)},
		},
		Milestones: MilestoneTimes{
			MilestoneIdeaReceived:          reportBase,
			MilestoneVerificationCompleted: reportBase.Add(2 * time.Hour),
		},
		Bindings: []BindingFact{
			{BindingID: "b1", AuthoringMode: ModePreCommit, Visibility: VisibilityVisible, Strength: StrengthAssertionStrong},
			{BindingID: "b2", AuthoringMode: ModePreCommit, Visibility: VisibilityHeldOut, Strength: StrengthBaselinePass},
		},
		Acceptances: []AcceptanceFact{
			{AcceptanceID: "a1", Required: LevelSource, Covered: true},
			{AcceptanceID: "a2", Required: LevelInformation, Covered: false, SourceSatisfied: true},
		},
		FirstPass: &FirstPassFact{TaskID: "task-1", WithHoldout: true, Passed: true},
	}
	rep, err := BuildBaselineReport([]TaskBaseline{t1}, reportBase.Add(time.Hour))
	if err != nil {
		t.Fatalf("BuildBaselineReport: %v", err)
	}

	out := RenderBaselineReport(rep)
	if again := RenderBaselineReport(rep); again != out {
		t.Errorf("RenderBaselineReport 应逐字节确定(可 diff、可存档)")
	}
	for _, want := range []string{
		"tasks: 1",
		"source_period: " + reportBase.Format(time.RFC3339) + " .. " + reportBase.Add(2*time.Hour).Format(time.RFC3339),
		"contract_review: active=50.0m wall=60.0m samples=2 sufficient",
		"spec_defect_decision: active=0.0m wall=0.0m samples=0 insufficient", // 未观测条目显式 0 + 不足
		"per_task_active_minutes: n=1 min=60.0 median=60.0 max=60.0 insufficient",
		"control: cost=100micros in=10 out=5 calls=1 samples=1 insufficient",
		"execution: cost=300micros in=30 out=15 calls=1 samples=1 insufficient",
		"control_plane_llm_cost_ratio: 0.2500",
		"intake: wall=0.0m samples=0 insufficient",
		"agent_execution: wall=30.0m samples=1 insufficient",
		"idea_to_verified_minutes: n=1 min=120.0 median=120.0 max=120.0 insufficient",
		"assertion_strong=1",
		"pre_commit=2",
		"held_out=1",
		"strength_x_visibility: total=2 sufficient | assertion_strong/visible=1 baseline_pass/held_out=1",
		"independence_planned: total=2 sufficient | none=0 source=1 information=1",
		"independence_source: required=1 covered=1 source_gaps=0 information_gaps=0 insufficient",
		"independence_information: required=1 covered=0 source_gaps=0 information_gaps=1 insufficient",
		"visible_only: tasks=0 passed=0 insufficient",
		"with_holdout: tasks=1 passed=1 insufficient",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("呈现应包含 %q,得到:\n%s", want, out)
		}
	}

	// 空报告:期间未知、比率未定义、无样本分布都以显式文字呈现(AC2)。
	empty, err := BuildBaselineReport(nil, reportBase)
	if err != nil {
		t.Fatalf("BuildBaselineReport(nil): %v", err)
	}
	emptyOut := RenderBaselineReport(empty)
	for _, want := range []string{
		"tasks: 0",
		"source_period: 未知(输入不含任何带观测时刻的事实,AC2:不伪造期间)",
		"per_task_active_minutes: n=0 insufficient(无样本)",
		"control_plane_llm_cost_ratio: 未定义(总成本为 0;不猜,AC2)",
		"strength_x_visibility: total=0 insufficient(无已观测组合)",
	} {
		if !strings.Contains(emptyOut, want) {
			t.Errorf("空报告呈现应包含 %q,得到:\n%s", want, emptyOut)
		}
	}
}
