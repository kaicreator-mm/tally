package measurement

import (
	"reflect"
	"testing"
	"time"
)

var testTime = time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)

// validLLMUsage 返回一条通过校验的最小 LLM 使用事实记录(用量未上报的
// 未知调用:数值全零 + UsageReported=false,锁 16:调用事实合法,数值未知)。
func validLLMUsage() LLMUsage {
	return LLMUsage{
		UseCase:    "implement_patch",
		CostPlane:  CostPlaneExecution,
		Status:     UsageSucceeded,
		RecordedAt: testTime,
	}
}

// AC1:cost_plane 只有 control|execution 两个合法值。
func TestCostPlaneEnum(t *testing.T) {
	valid := map[CostPlane]bool{
		CostPlaneControl:   true,
		CostPlaneExecution: true,
		"":                 false,
		"Control":          false,
		"execution ":       false,
		"planning":         false,
	}
	for p, want := range valid {
		if got := p.IsValid(); got != want {
			t.Errorf("CostPlane(%q).IsValid() = %v, want %v", string(p), got, want)
		}
	}
}

// AC1:cost_plane 在 LLM 使用记录上必填,缺省/非法值一律被 Validate 拒绝。
func TestLLMUsageValidate(t *testing.T) {
	ok := validLLMUsage()
	if err := ok.Validate(); err != nil {
		t.Fatalf("合法记录被拒绝: %v", err)
	}
	// 已上报观测:数值是事实(观测到 0 合法)。
	reported := validLLMUsage()
	reported.UsageReported = true
	reported.InputTokens, reported.OutputTokens, reported.CostMicros = 0, 0, 0
	if err := reported.Validate(); err != nil {
		t.Fatalf("已上报且观测到 0 的记录被拒绝(0 是合法观测): %v", err)
	}

	mutate := func(name string, f func(*LLMUsage)) {
		u := validLLMUsage()
		f(&u)
		if err := u.Validate(); err == nil {
			t.Errorf("%s: 非法记录被接受", name)
		}
	}
	mutate("cost_plane 缺失(空)", func(u *LLMUsage) { u.CostPlane = "" })
	mutate("cost_plane 非法值", func(u *LLMUsage) { u.CostPlane = "planning" })
	mutate("缺少 use_case", func(u *LLMUsage) { u.UseCase = "  " })
	mutate("status 非法", func(u *LLMUsage) { u.Status = "done" })
	mutate("status 缺失", func(u *LLMUsage) { u.Status = "" })
	mutate("input_tokens 为负", func(u *LLMUsage) { u.UsageReported = true; u.InputTokens = -1 })
	mutate("output_tokens 为负", func(u *LLMUsage) { u.UsageReported = true; u.OutputTokens = -1 })
	mutate("latency 为负", func(u *LLMUsage) { u.Latency = -time.Second })
	mutate("cost 为负", func(u *LLMUsage) { u.UsageReported = true; u.CostMicros = -1 })
	mutate("缺少 recorded_at", func(u *LLMUsage) { u.RecordedAt = time.Time{} })
	// 锁 16:未上报的未知用量不是「观测到 0」,携带数值等于伪装已知。
	mutate("未上报却携带 input_tokens", func(u *LLMUsage) { u.InputTokens = 1 })
	mutate("未上报却携带 output_tokens", func(u *LLMUsage) { u.OutputTokens = 1 })
	mutate("未上报却携带 cost", func(u *LLMUsage) { u.CostMicros = 1 })
}

// AC4:成本归属只读写入时落成的 cost_plane;use_case/provider/model 等字段
// 无论如何变化都不影响归属与比率 —— 包内不需要任何查询期 role→plane 重映射。
func TestCostAttributionUsesStoredPlaneOnly(t *testing.T) {
	usages := []LLMUsage{
		{UseCase: "intake", CostPlane: CostPlaneControl, CostMicros: 300, Status: UsageSucceeded, RecordedAt: testTime},
		{UseCase: "plan_contract", CostPlane: CostPlaneControl, CostMicros: 100, Status: UsageSucceeded, RecordedAt: testTime},
		{UseCase: "implement_patch", CostPlane: CostPlaneExecution, CostMicros: 600, Status: UsageSucceeded, RecordedAt: testTime},
	}

	control, execution := CostByPlane(usages)
	if control != 400 || execution != 600 {
		t.Fatalf("CostByPlane = (%d, %d), want (400, 600)", control, execution)
	}
	ratio, ok := ControlPlaneCostRatio(usages)
	if !ok {
		t.Fatal("ControlPlaneCostRatio 报告未定义,但总成本 > 0")
	}
	if ratio != 0.4 {
		t.Errorf("control_plane_llm_cost_ratio = %v, want 0.4", ratio)
	}

	// 改写 use_case/provider/model(等价于未来 role 命名变化):历史归属不变。
	relabeled := make([]LLMUsage, len(usages))
	for i, u := range usages {
		u.UseCase = u.UseCase + "-v2"
		u.Provider = "someone"
		u.Model = "some-model"
		relabeled[i] = u
	}
	control2, execution2 := CostByPlane(relabeled)
	if control2 != control || execution2 != execution {
		t.Errorf("重命名 use_case 后归属变化:(%d, %d) → (%d, %d);历史归属不得重算",
			control, execution, control2, execution2)
	}
	if r2, ok := ControlPlaneCostRatio(relabeled); !ok || r2 != ratio {
		t.Errorf("重命名 use_case 后比率变化:%v → %v", ratio, r2)
	}

	// cost_plane 非法的记录不参与归属(保守跳过,不猜)。
	bad := LLMUsage{UseCase: "x", CostMicros: 999, Status: UsageSucceeded, RecordedAt: testTime}
	control3, execution3 := CostByPlane(append(relabeled, bad))
	if control3 != control || execution3 != execution {
		t.Errorf("非法 cost_plane 记录被计入归属:(%d, %d)", control3, execution3)
	}
}

// AC4 的比率边界:总成本为 0 时比率未定义,返回 (0, false) 而不是猜一个值。
func TestControlPlaneCostRatioUndefinedWithoutCost(t *testing.T) {
	if ratio, ok := ControlPlaneCostRatio(nil); ok || ratio != 0 {
		t.Errorf("空记录集的比率 = (%v, %v), want (0, false)", ratio, ok)
	}
	usages := []LLMUsage{
		{UseCase: "a", CostPlane: CostPlaneControl, Status: UsageSucceeded, RecordedAt: testTime},
		{UseCase: "b", CostPlane: CostPlaneExecution, Status: UsageFailed, RecordedAt: testTime},
	}
	if ratio, ok := ControlPlaneCostRatio(usages); ok || ratio != 0 {
		t.Errorf("全 0 成本的比率 = (%v, %v), want (0, false)", ratio, ok)
	}
}

// AC2:人工开销记录上 active 与 wall 是两个独立字段,互不吞并。
func TestHumanEffortActiveAndWallDistinct(t *testing.T) {
	h := HumanEffort{
		TaskID:     "TASK-023",
		Kind:       EffortDiffReview,
		Active:     2 * time.Minute,
		Wall:       40 * time.Minute,
		RecordedAt: testTime,
	}
	if err := h.Validate(); err != nil {
		t.Fatalf("合法记录被拒绝: %v", err)
	}
	if h.Active == h.Wall {
		t.Fatal("active 与 wall 不应相等")
	}
	if got := h.ActiveMinutes(); got != 2 {
		t.Errorf("ActiveMinutes = %v, want 2", got)
	}
	if got := h.WallMinutes(); got != 40 {
		t.Errorf("WallMinutes = %v, want 40", got)
	}

	// 允许只测到其中一个口径(单侧为 0 不与另一侧混同)。
	activeOnly := HumanEffort{TaskID: "TASK-023", Kind: EffortContractReview, Active: 5 * time.Minute, RecordedAt: testTime}
	if err := activeOnly.Validate(); err != nil {
		t.Errorf("只有 active 的记录被拒绝: %v", err)
	}
	wallOnly := HumanEffort{TaskID: "TASK-023", Kind: EffortDiffReview, Wall: 15 * time.Minute, RecordedAt: testTime}
	if err := wallOnly.Validate(); err != nil {
		t.Errorf("只有 wall 的记录被拒绝: %v", err)
	}
}

func TestHumanEffortValidate(t *testing.T) {
	ok := HumanEffort{TaskID: "TASK-023", Kind: EffortManualCorrection, Active: time.Minute, RecordedAt: testTime}
	if err := ok.Validate(); err != nil {
		t.Fatalf("合法记录被拒绝: %v", err)
	}

	mutate := func(name string, f func(*HumanEffort)) {
		h := ok
		f(&h)
		if err := h.Validate(); err == nil {
			t.Errorf("%s: 非法记录被接受", name)
		}
	}
	mutate("缺少 task_id", func(h *HumanEffort) { h.TaskID = "" })
	mutate("kind 非法", func(h *HumanEffort) { h.Kind = "coffee_break" })
	mutate("active 为负", func(h *HumanEffort) { h.Active = -time.Minute })
	mutate("wall 为负", func(h *HumanEffort) { h.Wall = -time.Minute })
	mutate("缺少 recorded_at", func(h *HumanEffort) { h.RecordedAt = time.Time{} })
}

// AC3:生命周期阶段名是稳定枚举,顺序与 PRD §25 分解一一对应。
func TestLifecyclePhasesStableEnum(t *testing.T) {
	want := []LifecyclePhase{
		PhaseIntake,
		PhaseProductPlanning,
		PhaseVerificationPlanning,
		PhaseQueue,
		PhaseWorkspaceBootstrap,
		PhaseAgentExecution,
		PhaseIncrementalValidation,
		PhaseFinalValidation,
		PhaseHumanReview,
		PhaseIntegration,
	}
	got := LifecyclePhases()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("LifecyclePhases() = %v, want %v(PRD §25 顺序即契约)", got, want)
	}

	seen := map[LifecyclePhase]bool{}
	for _, p := range got {
		if seen[p] {
			t.Errorf("阶段名 %q 重复", string(p))
		}
		seen[p] = true
		if !p.IsValid() {
			t.Errorf("阶段名 %q 未通过自身枚举校验", string(p))
		}
	}

	// 返回的是副本:调用方改动不得污染枚举本身。
	got[0] = "tampered"
	if LifecyclePhases()[0] != PhaseIntake {
		t.Fatal("LifecyclePhases() 未返回副本,调用方修改会污染稳定枚举")
	}

	for _, bad := range []LifecyclePhase{"", "Intake", "human_review_v2"} {
		if bad.IsValid() {
			t.Errorf("非法阶段名 %q 通过校验", string(bad))
		}
	}
}

func TestMilestoneEnum(t *testing.T) {
	valid := []Milestone{
		MilestoneIdeaReceived, MilestoneTriageCompleted, MilestonePlanCompleted,
		MilestoneAgentStarted, MilestoneAgentCompleted,
		MilestoneVerificationStarted, MilestoneVerificationCompleted,
		MilestoneReviewStarted, MilestoneReviewCompleted,
		MilestoneMerged, MilestoneReleased, MilestoneDeployed, MilestoneStable,
	}
	seen := map[Milestone]bool{}
	for _, m := range valid {
		if !m.IsValid() {
			t.Errorf("里程碑 %q 未通过校验", string(m))
		}
		if seen[m] {
			t.Errorf("里程碑名 %q 重复", string(m))
		}
		seen[m] = true
	}
	for _, bad := range []Milestone{"", "IDEA_RECEIVED", "deployed_v2"} {
		if bad.IsValid() {
			t.Errorf("非法里程碑 %q 通过校验", string(bad))
		}
	}
}

func TestPhaseSpanValidate(t *testing.T) {
	start := testTime
	span := PhaseSpan{Phase: PhaseAgentExecution, Start: start, End: start.Add(10 * time.Minute)}
	if err := span.Validate(); err != nil {
		t.Fatalf("合法区间被拒绝: %v", err)
	}
	if got := span.Duration(); got != 10*time.Minute {
		t.Errorf("Duration = %v, want 10m", got)
	}

	mutate := func(name string, f func(*PhaseSpan)) {
		s := span
		f(&s)
		if err := s.Validate(); err == nil {
			t.Errorf("%s: 非法区间被接受", name)
		}
	}
	mutate("phase 非法", func(s *PhaseSpan) { s.Phase = "lunch" })
	mutate("缺 start", func(s *PhaseSpan) { s.Start = time.Time{} })
	mutate("缺 end", func(s *PhaseSpan) { s.End = time.Time{} })
	mutate("end 早于 start", func(s *PhaseSpan) { s.End = s.Start.Add(-time.Minute) })
}

func TestPhaseTimingValidateAndAggregate(t *testing.T) {
	start := testTime
	timing := PhaseTiming{
		TaskID: "TASK-023",
		Spans: []PhaseSpan{
			{Phase: PhaseAgentExecution, Start: start, End: start.Add(5 * time.Minute)},
			{Phase: PhaseAgentExecution, Start: start.Add(10 * time.Minute), End: start.Add(18 * time.Minute)},
			{Phase: PhaseHumanReview, Start: start.Add(20 * time.Minute), End: start.Add(50 * time.Minute)},
		},
	}
	if err := timing.Validate(); err != nil {
		t.Fatalf("合法记录被拒绝: %v", err)
	}

	byPhase := timing.DurationByPhase()
	if got := byPhase[PhaseAgentExecution]; got != 13*time.Minute {
		t.Errorf("agent_execution 汇总 = %v, want 13m(同阶段多区间累加)", got)
	}
	if got := byPhase[PhaseHumanReview]; got != 30*time.Minute {
		t.Errorf("human_review 汇总 = %v, want 30m", got)
	}
	if _, present := byPhase[PhaseIntake]; present {
		t.Error("没有区间的阶段不应出现在汇总里")
	}

	if err := (PhaseTiming{TaskID: ""}).Validate(); err == nil {
		t.Error("缺少 task_id 的记录被接受")
	}
	broken := PhaseTiming{TaskID: "TASK-023", Spans: []PhaseSpan{{Phase: "lunch", Start: start, End: start}}}
	if err := broken.Validate(); err == nil {
		t.Error("含非法区间的记录被接受")
	}
}

func TestIdeaToVerifiedLatency(t *testing.T) {
	start := testTime
	ms := MilestoneTimes{
		MilestoneIdeaReceived:          start,
		MilestoneVerificationCompleted: start.Add(40 * time.Minute),
	}
	got, ok := IdeaToVerifiedLatency(ms)
	if !ok || got != 40*time.Minute {
		t.Errorf("IdeaToVerifiedLatency = (%v, %v), want (40m, true)", got, ok)
	}

	if _, ok := IdeaToVerifiedLatency(MilestoneTimes{MilestoneIdeaReceived: start}); ok {
		t.Error("缺 verification_completed 时不该给出跨度")
	}
	if _, ok := IdeaToVerifiedLatency(MilestoneTimes{MilestoneVerificationCompleted: start}); ok {
		t.Error("缺 idea_received 时不该给出跨度")
	}
	if _, ok := IdeaToVerifiedLatency(MilestoneTimes{
		MilestoneIdeaReceived:          time.Time{},
		MilestoneVerificationCompleted: start,
	}); ok {
		t.Error("零值时间戳不该参与跨度计算")
	}
}

// Minutes 是 PRD *_minutes 口径的换算。
func TestMinutes(t *testing.T) {
	if got := Minutes(90 * time.Second); got != 1.5 {
		t.Errorf("Minutes(90s) = %v, want 1.5", got)
	}
	if got := Minutes(0); got != 0 {
		t.Errorf("Minutes(0) = %v, want 0", got)
	}
}
