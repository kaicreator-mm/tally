package measurement

// recorder.go(P1-41;P0-06 演进用量可用性)测试。
//
// 两层,同一文件(Allowed paths 只给 recorder_test.go 一个测试文件):
//   - 纯单元层:阶段区间 start/end/wall 事实(AC1)、TokenUsage 的不可用
//     语义(AC4)、用量观测必须绑定存储(P0-06:未上报用量的调用也必须落行)
//     与并发阶段录制 —— 不触库,NewRecorder(nil) 即可;
//   - 数据库层:录制的人工计时/LLM 使用事实落库读回(AC1/AC2),未上报
//     用量调用的事实行与聚合口径(锁 16/P0-06 AC1-AC3),指标全部由事实
//     推导(AC3)。复用 store_test.go 的测试库装置(每测试独立 schema、
//     自跑迁移、数据库不可达即 SKIP,不产出假失败)。
//
// 全部断言基于确定性输入(types_test.go 的 testTime),不依赖墙钟。

import (
	"context"
	"sync"
	"testing"
	"time"
)

// ---- 纯单元层(无 DB) ----

// AC1:录制器按 start/end 落成阶段区间事实,End-Start 即该段墙钟时长;
// 同阶段多段先后累加,跨阶段互不吞并。
func TestRecorderPhaseStartEndFacts(t *testing.T) {
	rec := NewRecorder(nil)
	base := testTime

	if err := rec.PhaseStarted("T-41", PhaseAgentExecution, base); err != nil {
		t.Fatalf("PhaseStarted(agent_execution): %v", err)
	}
	span, err := rec.PhaseEnded("T-41", PhaseAgentExecution, base.Add(90*time.Second))
	if err != nil {
		t.Fatalf("PhaseEnded(agent_execution): %v", err)
	}
	if !span.Phase.IsValid() || !span.Start.Equal(base) || !span.End.Equal(base.Add(90*time.Second)) {
		t.Fatalf("区间事实不一致:%+v", span)
	}
	if got := span.Duration(); got != 90*time.Second {
		t.Fatalf("墙钟时长 = %v, want 90s(AC1)", got)
	}

	// 同阶段第二段(重试)与另一阶段:区间事实按序累积。
	if err := rec.PhaseStarted("T-41", PhaseAgentExecution, base.Add(2*time.Minute)); err != nil {
		t.Fatalf("PhaseStarted(第二段): %v", err)
	}
	if _, err := rec.PhaseEnded("T-41", PhaseAgentExecution, base.Add(3*time.Minute)); err != nil {
		t.Fatalf("PhaseEnded(第二段): %v", err)
	}
	if err := rec.PhaseStarted("T-41", PhaseFinalValidation, base.Add(3*time.Minute)); err != nil {
		t.Fatalf("PhaseStarted(final_validation): %v", err)
	}
	if _, err := rec.PhaseEnded("T-41", PhaseFinalValidation, base.Add(13*time.Minute)); err != nil {
		t.Fatalf("PhaseEnded(final_validation): %v", err)
	}

	timing, ok := rec.PhaseTiming("T-41")
	if !ok {
		t.Fatal("已录制区间但 PhaseTiming 报告无事实")
	}
	if err := timing.Validate(); err != nil {
		t.Fatalf("录制出的 PhaseTiming 未通过事实校验: %v", err)
	}
	if len(timing.Spans) != 3 || timing.TaskID != "T-41" {
		t.Fatalf("区间数 = %d(task=%s), want 3(T-41)", len(timing.Spans), timing.TaskID)
	}
	byPhase := timing.DurationByPhase()
	if got := byPhase[PhaseAgentExecution]; got != 90*time.Second+time.Minute {
		t.Errorf("agent_execution 汇总 = %v, want 2m30s(同阶段多段累加)", got)
	}
	if got := byPhase[PhaseFinalValidation]; got != 10*time.Minute {
		t.Errorf("final_validation 汇总 = %v, want 10m", got)
	}
	if _, present := byPhase[PhaseIntake]; present {
		t.Error("未录制的阶段不应出现在耗时分解里")
	}

	// 返回的是副本:调用方改动不得污染录制器内部事实。
	timing.Spans[0] = PhaseSpan{}
	again, _ := rec.PhaseTiming("T-41")
	if again.Spans[0].Phase != PhaseAgentExecution {
		t.Fatal("PhaseTiming 未返回副本,调用方修改会污染事实面")
	}

	if _, ok := rec.PhaseTiming("T-none"); ok {
		t.Fatal("无区间任务应报告无事实")
	}
}

// AC1 的拒绝路径:非法阶段名/缺任务/缺时刻/重复开启/未开启先关闭/时刻倒流
// 一律报错,不落半截事实。
func TestRecorderPhaseErrors(t *testing.T) {
	rec := NewRecorder(nil)
	base := testTime

	if err := rec.PhaseStarted("", PhaseIntake, base); err == nil {
		t.Error("缺 task_id 的开启被接受")
	}
	if err := rec.PhaseStarted("T-41", "lunch", base); err == nil {
		t.Error("非法阶段名被接受")
	}
	if err := rec.PhaseStarted("T-41", PhaseIntake, time.Time{}); err == nil {
		t.Error("缺时刻的开启被接受")
	}
	if err := rec.PhaseStarted("T-41", PhaseAgentExecution, base); err != nil {
		t.Fatalf("合法开启被拒: %v", err)
	}
	if err := rec.PhaseStarted("T-41", PhaseAgentExecution, base.Add(time.Second)); err == nil {
		t.Error("同任务同阶段重复开启被接受")
	}
	if _, err := rec.PhaseEnded("T-41", PhaseIntake, base); err == nil {
		t.Error("未开启的阶段被关闭")
	}
	if _, err := rec.PhaseEnded("T-41", PhaseAgentExecution, base.Add(-time.Second)); err == nil {
		t.Error("结束时刻早于开启时刻被接受")
	}
	if _, err := rec.PhaseEnded("T-41", PhaseAgentExecution, time.Time{}); err == nil {
		t.Error("缺时刻的关闭被接受")
	}
	// 关闭成功后可再次开启同一阶段(分段计时)。
	if _, err := rec.PhaseEnded("T-41", PhaseAgentExecution, base.Add(time.Minute)); err != nil {
		t.Fatalf("合法关闭被拒: %v", err)
	}
	if err := rec.PhaseStarted("T-41", PhaseAgentExecution, base.Add(2*time.Minute)); err != nil {
		t.Fatalf("关闭后再次开启被拒: %v", err)
	}
}

// AC4:「provider 未上报用量」与「观测到 0」是两种状态,观测类型本身不可互化。
func TestTokenUsageUnavailableSemantics(t *testing.T) {
	if UnavailableTokens().Reported {
		t.Fatal("UnavailableTokens 不应报告已上报")
	}
	if err := UnavailableTokens().Validate(); err != nil {
		t.Fatalf("未上报观测被拒: %v", err)
	}
	zero := ReportedTokens(0, 0)
	if !zero.Reported {
		t.Fatal("ReportedTokens 应报告已上报")
	}
	if err := zero.Validate(); err != nil {
		t.Fatalf("观测到 0 是合法事实,被拒: %v", err)
	}
	if err := (TokenUsage{InputTokens: 5}).Validate(); err == nil {
		t.Fatal("未上报却携带数值被接受(等于把未知伪装成已知 0)")
	}
	if err := (TokenUsage{Reported: true, InputTokens: -1}).Validate(); err == nil {
		t.Fatal("负 token 被接受")
	}
}

// AC4 + P0-06 锁 16:未上报用量的调用也必须落成事实行,因此未绑定存储的
// 录制器对任何 runtime 用量观测(含未上报)都报错且不建聚合 —— 调用事实
// 无法持久化就不计录,不退回「只在内存计数」的静默丢事实路径。
func TestRecorderRuntimeUsageRequiresStoreForEveryCall(t *testing.T) {
	rec := NewRecorder(nil)
	ctx := context.Background()

	if _, err := rec.RecordRuntimeUsage(ctx, RuntimeUsage{
		TaskID: "T-41", UseCase: "implement_patch", CostPlane: CostPlaneExecution,
		Runtime: "fake", Status: UsageSucceeded, Latency: time.Minute,
		Tokens: UnavailableTokens(), RecordedAt: testTime,
	}); err == nil {
		t.Fatal("未绑定存储的未上报观测应报错(未知用量的调用也必须落行)")
	}
	if _, err := rec.RecordRuntimeUsage(ctx, RuntimeUsage{
		TaskID: "T-41", UseCase: "implement_patch", CostPlane: CostPlaneExecution,
		Status: UsageSucceeded, Tokens: ReportedTokens(10, 5), RecordedAt: testTime.Add(time.Second),
	}); err == nil {
		t.Fatal("未绑定存储的已上报观测应报错")
	}
	if _, ok := rec.RuntimeUsageForTask("T-41"); ok {
		t.Fatal("落库失败的观测不应建聚合(事实未落成不计入)")
	}

	// 非法观测在录制前被拒(含:用量未上报却携带成本)。
	bad := []RuntimeUsage{
		{TaskID: "", UseCase: "x", CostPlane: CostPlaneControl, Status: UsageSucceeded, Tokens: UnavailableTokens(), RecordedAt: testTime},
		{TaskID: "T-41", UseCase: " ", CostPlane: CostPlaneControl, Status: UsageSucceeded, Tokens: UnavailableTokens(), RecordedAt: testTime},
		{TaskID: "T-41", UseCase: "x", CostPlane: CostPlane("planning"), Status: UsageSucceeded, Tokens: UnavailableTokens(), RecordedAt: testTime},
		{TaskID: "T-41", UseCase: "x", CostPlane: CostPlaneControl, Status: "done", Tokens: UnavailableTokens(), RecordedAt: testTime},
		{TaskID: "T-41", UseCase: "x", CostPlane: CostPlaneControl, Status: UsageSucceeded, Tokens: TokenUsage{InputTokens: 3}, RecordedAt: testTime},
		{TaskID: "T-41", UseCase: "x", CostPlane: CostPlaneControl, Status: UsageSucceeded, Tokens: UnavailableTokens()},
		{TaskID: "T-41", UseCase: "x", CostPlane: CostPlaneControl, Status: UsageSucceeded, Tokens: UnavailableTokens(), CostMicros: 5, RecordedAt: testTime},
	}
	for i, o := range bad {
		if _, err := rec.RecordRuntimeUsage(ctx, o); err == nil {
			t.Fatalf("非法观测 %d 被接受", i)
		}
	}
	if other, ok := rec.RuntimeUsageForTask("T-none"); ok {
		t.Fatalf("非法观测不应建聚合:%+v", other)
	}
}

// 并发录制(纯事实路径):不同阶段并行 start/end 互不丢事实(race 检测下
// 同时验证内部锁)。用量观测的并发路径需要落库,见 DB 层
// TestRecorderConcurrentUsageRecordingDB。
func TestRecorderConcurrentPhaseRecording(t *testing.T) {
	rec := NewRecorder(nil)
	phases := LifecyclePhases()

	var wg sync.WaitGroup
	for _, p := range phases {
		wg.Add(1)
		go func(p LifecyclePhase) {
			defer wg.Done()
			if err := rec.PhaseStarted("T-41", p, testTime); err != nil {
				t.Errorf("并发 PhaseStarted(%s): %v", string(p), err)
				return
			}
			if _, err := rec.PhaseEnded("T-41", p, testTime.Add(time.Minute)); err != nil {
				t.Errorf("并发 PhaseEnded(%s): %v", string(p), err)
			}
		}(p)
	}
	wg.Wait()

	timing, ok := rec.PhaseTiming("T-41")
	if !ok || len(timing.Spans) != len(phases) {
		t.Fatalf("并发阶段录制丢事实:%d 段, want %d", len(timing.Spans), len(phases))
	}
}

// ---- 数据库层(不可达即 SKIP) ----

// AC1(active/wall)+ AC2 + AC3:人工计时与 LLM 使用事实经录制器落库读回,
// *_minutes 与 control_plane 比率只从事实行推导 —— 全程没有任何手工任务字段。
func TestRecorderPersistsEffortAndUsageFactsDB(t *testing.T) {
	st, pool := testMeasurementStore(t)
	rec := NewRecorder(st)
	ctx := context.Background()

	// AC1:active/wall 双口径人工计时事实,读侧聚合出 *_minutes。
	effort := HumanEffort{
		TaskID: "T-41", Kind: EffortDiffReview,
		Active: 10 * time.Minute, Wall: 15 * time.Minute, RecordedAt: testTime,
	}
	if _, err := rec.RecordEffort(ctx, effort); err != nil {
		t.Fatalf("RecordEffort: %v", err)
	}
	rows, err := st.HumanEffortForTask(ctx, "T-41")
	if err != nil || len(rows) != 1 || !sameEffort(rows[0], effort) {
		t.Fatalf("人工计时事实读回不一致: %v %+v", err, rows)
	}
	sums, err := st.HumanEffortByKindForTask(ctx, "T-41")
	if err != nil || len(sums) != 1 {
		t.Fatalf("HumanEffortByKindForTask: %v %d", err, len(sums))
	}
	if sums[0].ActiveMinutes() != 10 || sums[0].WallMinutes() != 15 {
		t.Fatalf("*_minutes 换算不一致: active=%v wall=%v(AC1/AC3)", sums[0].ActiveMinutes(), sums[0].WallMinutes())
	}

	// AC2:cost_plane 在调用发生时随录制落成事实字段。
	ctrl := LLMUsage{
		TaskID: "T-41", UseCase: "verify_draft", CostPlane: CostPlaneControl,
		CostMicros: 2_000_000, UsageReported: true,
		Status: UsageSucceeded, RecordedAt: testTime,
	}
	exec := LLMUsage{
		TaskID: "T-41", UseCase: "implement_patch", CostPlane: CostPlaneExecution,
		Provider: "anthropic", Runtime: "codex",
		InputTokens: 100, OutputTokens: 40, CostMicros: 8_000_000, UsageReported: true,
		Status: UsageSucceeded, RecordedAt: testTime.Add(time.Minute),
	}
	if _, err := rec.RecordLLMUsage(ctx, ctrl); err != nil {
		t.Fatalf("RecordLLMUsage(control): %v", err)
	}
	if _, err := rec.RecordLLMUsage(ctx, exec); err != nil {
		t.Fatalf("RecordLLMUsage(execution): %v", err)
	}
	costs, err := st.LLMCostByPlaneForTask(ctx, "T-41")
	if err != nil {
		t.Fatalf("LLMCostByPlaneForTask: %v", err)
	}
	if ratio, ok := ControlPlaneRatio(costs); !ok || ratio != 0.2 {
		t.Fatalf("control_plane_llm_cost_ratio = (%v, %v), want 0.2(AC2/AC3:只按存储 cost_plane 推导)", ratio, ok)
	}

	// 非法归属面经录制器同样在写入前被拒,不落行。
	baseline := countRows(t, pool, "SELECT count(*) FROM llm_usage")
	bad := validLLMUsage()
	bad.TaskID = "T-41"
	bad.CostPlane = "planning"
	if _, err := rec.RecordLLMUsage(ctx, bad); err == nil {
		t.Fatal("非法 cost_plane 应报错")
	}
	if n := countRows(t, pool, "SELECT count(*) FROM llm_usage"); n != baseline {
		t.Fatalf("非法输入不应落行:baseline %d,实际 %d", baseline, n)
	}
}

// AC4 + P0-06:上报用量的 runtime 观测落成携带数值的事实行;未上报的照样
// 落行(usage_reported=false + NULL 数值)并把聚合标记为不可用 —— 落库面
// 与聚合面都不把「未观测」写成「观测到 0」;UsageAvailable 由缺失计数单调
// 推导,之后的已上报调用不会把它翻回 true(P0-06 bug a)。
func TestRecorderRuntimeUsageDBPersistenceUnavailableNotZero(t *testing.T) {
	st, pool := testMeasurementStore(t)
	rec := NewRecorder(st)
	ctx := context.Background()

	reported := RuntimeUsage{
		TaskID: "T-41", UseCase: "implement_patch", CostPlane: CostPlaneExecution,
		Provider: "anthropic", Runtime: "codex", Model: "test-model",
		BehaviorRevision: "behavior-1", ContextHash: "sha256:pack",
		Latency: 2 * time.Minute, Status: UsageSucceeded,
		Tokens: ReportedTokens(120, 45), CostMicros: 5_000_000, RecordedAt: testTime,
	}
	id, err := rec.RecordRuntimeUsage(ctx, reported)
	if err != nil || id == 0 {
		t.Fatalf("RecordRuntimeUsage(上报): %v id=%d", err, id)
	}

	// AC4/锁 16:未上报用量 —— 调用事实照落,数值 NULL(不是 0)。
	unavailable := reported
	unavailable.Tokens = UnavailableTokens()
	unavailable.CostMicros = 0
	unavailable.Status = UsageCancelled
	unavailable.Latency = time.Minute
	unavailable.RecordedAt = testTime.Add(time.Minute)
	id2, err := rec.RecordRuntimeUsage(ctx, unavailable)
	if err != nil {
		t.Fatalf("RecordRuntimeUsage(未上报): %v", err)
	}
	if id2 == 0 {
		t.Fatal("未上报用量的调用应有事实行 id(未知用量仍是一次调用,锁 16)")
	}
	if n := countRows(t, pool, "SELECT count(*) FROM llm_usage WHERE task_id=$1", "T-41"); n != 2 {
		t.Fatalf("T-41 应有 2 条事实行(上报 + 未上报),得到 %d", n)
	}
	rows, err := st.LLMUsageForTask(ctx, "T-41")
	if err != nil || len(rows) != 2 {
		t.Fatalf("LLMUsageForTask: %v %d", err, len(rows))
	}
	if !sameUsage(rows[0], reported.fact()) {
		t.Fatalf("上报事实行读回不一致:\n got %+v\nwant %+v", rows[0], reported.fact())
	}
	if !sameUsage(rows[1], unavailable.fact()) {
		t.Fatalf("未上报事实行读回不一致:\n got %+v\nwant %+v", rows[1], unavailable.fact())
	}
	if rows[1].UsageReported || rows[1].InputTokens != 0 || rows[1].CostMicros != 0 {
		t.Fatalf("未上报事实行携带了数值或误标已上报:%+v(未知不是 0)", rows[1])
	}
	var nullValues, usageReported bool
	if err := pool.QueryRow(ctx,
		`SELECT input_tokens IS NULL AND output_tokens IS NULL AND cost_micros IS NULL, usage_reported
		 FROM llm_usage WHERE id = $1`, id2).Scan(&nullValues, &usageReported); err != nil {
		t.Fatalf("读未上报行原始列: %v", err)
	}
	if !nullValues || usageReported {
		t.Fatalf("未上报行应为 NULL 数值 + usage_reported=false: null=%v reported=%v", nullValues, usageReported)
	}

	// P0-06 bug a:缺失调用之后再出现已上报调用,UsageAvailable 不得翻回 true;
	// 数值合计继续累计已上报观测,缺失计数单调不减。
	later := reported
	later.Tokens = ReportedTokens(30, 15)
	later.CostMicros = 1_000_000
	later.RecordedAt = testTime.Add(2 * time.Minute)
	if _, err := rec.RecordRuntimeUsage(ctx, later); err != nil {
		t.Fatalf("RecordRuntimeUsage(缺失后再上报): %v", err)
	}
	totals, ok := rec.RuntimeUsageForTask("T-41")
	if !ok {
		t.Fatal("已录制观测但聚合报告无数据")
	}
	want := RuntimeUsageTotals{
		Calls: 3, CallsMissingUsage: 1,
		InputTokens: 150, OutputTokens: 60, CostMicros: 6_000_000,
		UsageAvailable: false,
	}
	if totals != want {
		t.Fatalf("聚合不一致:\n got %+v\nwant %+v", totals, want)
	}

	// 全部上报的任务:合计完整可用,缺失计数为 0。
	full := reported
	full.TaskID = "T-42"
	full.Tokens = ReportedTokens(10, 4)
	full.CostMicros = 100
	if _, err := rec.RecordRuntimeUsage(ctx, full); err != nil {
		t.Fatalf("RecordRuntimeUsage(T-42): %v", err)
	}
	if got, ok := rec.RuntimeUsageForTask("T-42"); !ok || got != (RuntimeUsageTotals{
		Calls: 1, InputTokens: 10, OutputTokens: 4, CostMicros: 100, UsageAvailable: true,
	}) {
		t.Fatalf("完整观测聚合不一致: %+v ok=%v", got, ok)
	}

	// 存储聚合与录制器聚合同口径:total/reported/missing 三列可重建。
	costs, err := st.LLMCostByPlaneForTask(ctx, "T-41")
	if err != nil || len(costs) != 1 {
		t.Fatalf("LLMCostByPlaneForTask(T-41): %v %d", err, len(costs))
	}
	c := costs[0]
	if c.Plane != CostPlaneExecution || c.Calls != 3 || c.CallsReported != 2 || c.CallsMissingUsage != 1 ||
		c.CostMicros != 6_000_000 || c.InputTokens != 150 || c.OutputTokens != 60 {
		t.Fatalf("存储聚合与录制器聚合不一致(AC1 不可重建): %+v", c)
	}

	// 非法观测:不落行、不改聚合。
	baseline := countRows(t, pool, "SELECT count(*) FROM llm_usage")
	badObs := reported
	badObs.CostPlane = "planning"
	if _, err := rec.RecordRuntimeUsage(ctx, badObs); err == nil {
		t.Fatal("非法 cost_plane 观测应报错")
	}
	if n := countRows(t, pool, "SELECT count(*) FROM llm_usage"); n != baseline {
		t.Fatalf("非法观测不应落行:baseline %d,实际 %d", baseline, n)
	}
	if got, _ := rec.RuntimeUsageForTask("T-41"); got != want {
		t.Fatalf("非法观测改变了聚合:%+v", got)
	}
}

// 并发用量录制(DB 层):同任务并行观测(上报/未上报混合)互不丢事实,
// 聚合的 Calls/CallsMissingUsage 由缺失计数单调收敛(race 检测下同时验证
// 内部锁)。
func TestRecorderConcurrentUsageRecordingDB(t *testing.T) {
	st, pool := testMeasurementStore(t)
	rec := NewRecorder(st)
	ctx := context.Background()

	const reportedCalls, missingCalls = 6, 10
	var wg sync.WaitGroup
	for i := 0; i < reportedCalls; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := rec.RecordRuntimeUsage(ctx, RuntimeUsage{
				TaskID: "T-41", UseCase: "implement_patch", CostPlane: CostPlaneExecution,
				Status: UsageSucceeded, Tokens: ReportedTokens(10, 4), CostMicros: 100,
				RecordedAt: testTime.Add(time.Duration(i) * time.Second),
			}); err != nil {
				t.Errorf("并发已上报观测: %v", err)
			}
		}(i)
	}
	for i := 0; i < missingCalls; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if _, err := rec.RecordRuntimeUsage(ctx, RuntimeUsage{
				TaskID: "T-41", UseCase: "implement_patch", CostPlane: CostPlaneExecution,
				Status: UsageSucceeded, Tokens: UnavailableTokens(),
				RecordedAt: testTime.Add(time.Duration(i) * time.Second),
			}); err != nil {
				t.Errorf("并发未上报观测: %v", err)
			}
		}(i)
	}
	wg.Wait()

	total := int64(reportedCalls + missingCalls)
	totals, ok := rec.RuntimeUsageForTask("T-41")
	if !ok || totals.Calls != total || totals.CallsMissingUsage != missingCalls || totals.UsageAvailable {
		t.Fatalf("并发用量聚合不一致: %+v ok=%v(存在未上报调用,UsageAvailable 必须为 false)", totals, ok)
	}
	if n := countRows(t, pool, "SELECT count(*) FROM llm_usage WHERE task_id=$1", "T-41"); n != total {
		t.Fatalf("并发落行丢事实: %d, want %d", n, total)
	}
	if n := countRows(t, pool,
		"SELECT count(*) FROM llm_usage WHERE task_id=$1 AND NOT usage_reported", "T-41"); n != missingCalls {
		t.Fatalf("并发未上报行计数不一致: %d, want %d", n, missingCalls)
	}
}
