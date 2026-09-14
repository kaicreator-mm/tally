package measurement

// store.go(P1-34)适配器测试。
//
// 两层,同一文件(Allowed paths 只给 store_test.go 一个测试文件):
//   - 纯单元层:cost_plane 聚合比率、*_minutes 换算、行重建校验(不触库),
//     `go test ./internal/measurement/...` 无 DB 也全绿;
//   - 数据库层:写入/读回往返、按存储 cost_plane 列聚合(AC2)、人工计时
//     跨行聚合(AC3)、触发器拒绝路径。每个测试独立 schema、自跑 goose
//     迁移、cleanup DROP(与 internal/runstore 同一隔离策略);数据库不可达
//     时 SKIP,不产出假失败。
//
// 测试固定量复用 types_test.go 的 testTime/validLLMUsage,全部断言基于
// 确定性输入,不依赖墙钟。

import (
	"context"
	"database/sql"
	"net"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"tally/internal/store"
	"tally/migrations"
)

const measurementTestDSN = "postgres://postgres:postgres@localhost:5432/tally?sslmode=disable"

// ---- 测试库:每测试独立 schema,自跑迁移,cleanup DROP;不可达即 SKIP ----

func testMeasurementStore(t *testing.T) (*Store, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		base = measurementTestDSN
	}
	if !dbReachable(base) {
		t.Skipf("measurement 数据库层测试需要 PostgreSQL(%s 不可达),跳过", base)
	}
	schema := "tally_meas_" + strconv.FormatInt(time.Now().UnixNano(), 10)

	setSearchPath := func(ctx context.Context, c *pgconn.PgConn) error {
		_, err := c.Exec(ctx, "SET search_path = "+pgx.Identifier{schema}.Sanitize()).ReadAll()
		return err
	}

	boot, err := pgxpool.ParseConfig(base)
	if err != nil {
		t.Fatalf("解析 DATABASE_URL: %v", err)
	}
	boot.MaxConns = 1
	boot.ConnConfig.Config.AfterConnect = setSearchPath
	bootDB := stdlib.OpenDB(*boot.ConnConfig)
	if _, err := bootDB.ExecContext(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		bootDB.Close()
		t.Fatalf("建测试 schema %s: %v", schema, err)
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, bootDB, migrations.FS)
	if err != nil {
		bootDB.Close()
		t.Fatalf("创建 goose provider: %v", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		bootDB.Close()
		t.Fatalf("在测试 schema %s 跑迁移: %v", schema, err)
	}
	bootDB.Close()

	cfg, err := pgxpool.ParseConfig(base)
	if err != nil {
		t.Fatalf("解析 DATABASE_URL(连接池): %v", err)
	}
	cfg.ConnConfig.Config.AfterConnect = setSearchPath
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("建立测试连接池: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping 测试库: %v", err)
	}
	t.Cleanup(func() {
		pool.Close()
		dropSchema(base, schema)
	})
	return New(store.New(pool)), pool
}

func dbReachable(base string) bool {
	u, err := url.Parse(base)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "" {
		host = "localhost"
	}
	port := u.Port()
	if port == "" {
		port = "5432"
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 2*time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func dropSchema(base, schema string) {
	u, err := url.Parse(base)
	if err != nil {
		return
	}
	q := u.Query()
	q.Del("search_path")
	u.RawQuery = q.Encode()
	db, err := sql.Open("pgx", u.String())
	if err != nil {
		return
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, _ = db.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+pgx.Identifier{schema}.Sanitize()+" CASCADE")
}

func countRows(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("count %q: %v", query, err)
	}
	return n
}

// mustModifyRejected 断言一条改写/删除语句被 append-only 触发器拒绝
// (INV-FACT-01 的数据库侧防线;适配器无 API 只是第一道防线)。
func mustModifyRejected(t *testing.T, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	_, err := pool.Exec(context.Background(), query, args...)
	if err == nil {
		t.Fatalf("append-only 触发器未拦截:%s", query)
	}
}

// ---- 纯单元层(无 DB) ----

// AC2 的读侧:比率只由存储的 cost_plane 分组值推导,总额为 0 时不定义。
func TestControlPlaneRatioFromStoredAggregates(t *testing.T) {
	both := []PlaneCost{
		{Plane: CostPlaneControl, CostMicros: 30},
		{Plane: CostPlaneExecution, CostMicros: 70},
	}
	if got, ok := ControlPlaneRatio(both); !ok || got != 0.3 {
		t.Fatalf("比率应 0.3,得到 %v %v", got, ok)
	}
	execOnly := []PlaneCost{{Plane: CostPlaneExecution, CostMicros: 10}}
	if got, ok := ControlPlaneRatio(execOnly); !ok || got != 0 {
		t.Fatalf("纯 execution 比率应 0,得到 %v %v", got, ok)
	}
	if got, ok := ControlPlaneRatio(nil); ok || got != 0 {
		t.Fatalf("无记录比率未定义,得到 %v %v", got, ok)
	}
	if got, ok := ControlPlaneRatio([]PlaneCost{{Plane: CostPlaneControl}}); ok || got != 0 {
		t.Fatalf("零成本比率未定义,得到 %v %v", got, ok)
	}
}

func TestEffortSummaryMinutesConversion(t *testing.T) {
	s := EffortSummary{Kind: EffortDiffReview, Active: 90 * time.Second, Wall: 30 * time.Minute, Samples: 3}
	if got := s.ActiveMinutes(); got != 1.5 {
		t.Fatalf("ActiveMinutes 应 1.5,得到 %v", got)
	}
	if got := s.WallMinutes(); got != 30 {
		t.Fatalf("WallMinutes 应 30,得到 %v", got)
	}
}

// 重建即验证:status 越出 CHECK 取值的行(行破坏)必须报错,不被当真;
// usage_reported 与 NULL/非 NULL 数值的不自洽(0022 CHECK 的读侧镜像)同样
// 视为行破坏。
func TestLLMUsageRowReconstructionRejectsCorruption(t *testing.T) {
	row := store.ListLLMUsageByTaskRow{
		ID: 1, UseCase: "implement_patch", CostPlane: "execution",
		InputTokens:   pgtype.Int8{Int64: 10, Valid: true},
		OutputTokens:  pgtype.Int8{Int64: 5, Valid: true},
		LatencyMicros: 1500,
		CostMicros:    pgtype.Int8{Int64: 7, Valid: true},
		UsageReported: true,
		Status:        "succeeded", RecordedAt: pgtype.Timestamptz{Time: testTime, Valid: true},
	}
	got, err := llmUsageFromRow(row)
	if err != nil || got.Latency != 1500*time.Microsecond || got.CostPlane != CostPlaneExecution || !got.UsageReported {
		t.Fatalf("合法行重建不一致: %v %+v", err, got)
	}
	bad := row
	bad.Status = "green"
	if _, err := llmUsageFromRow(bad); err == nil {
		t.Fatal("status 越界应报错")
	}
	bad = row
	bad.CostPlane = "galactic"
	if _, err := llmUsageFromRow(bad); err == nil {
		t.Fatal("cost_plane 越界应报错")
	}
	bad = row
	bad.LatencyMicros = -1
	if _, err := llmUsageFromRow(bad); err == nil {
		t.Fatal("负 latency 应报错")
	}
	// 未知用量行:usage_reported=false + 三列 NULL,重建为可用性 false 的领域行。
	unknown := row
	unknown.UsageReported = false
	unknown.InputTokens, unknown.OutputTokens, unknown.CostMicros = pgtype.Int8{}, pgtype.Int8{}, pgtype.Int8{}
	gotUnknown, err := llmUsageFromRow(unknown)
	if err != nil || gotUnknown.UsageReported || gotUnknown.InputTokens != 0 || gotUnknown.CostMicros != 0 {
		t.Fatalf("未知用量行重建不一致: %v %+v", err, gotUnknown)
	}
	// 行破坏:未上报却携带数值(库内正常被 0022 CHECK 拒,读侧镜像兜底)。
	bad = unknown
	bad.CostMicros = pgtype.Int8{Int64: 1, Valid: true}
	if _, err := llmUsageFromRow(bad); err == nil {
		t.Fatal("未上报却携带数值的行应报错")
	}
	// 行破坏:已上报却缺数值。
	bad = row
	bad.OutputTokens = pgtype.Int8{}
	if _, err := llmUsageFromRow(bad); err == nil {
		t.Fatal("已上报却缺数值的行应报错")
	}
}

// AC4:适配器不存在任何更新/删除 API,更没有触碰任务状态的入口。
func TestMeasurementStoreAPIHasNoTaskMutation(t *testing.T) {
	typ := reflect.TypeOf(&Store{})
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		for _, banned := range []string{"Update", "Delete", "Rewrite", "Truncate", "Drop", "SetTask"} {
			if strings.HasPrefix(name, banned) {
				t.Errorf("度量事实面出现改写/删除/状态入口 API:%s(AC4)", name)
			}
		}
	}
}

// ---- 数据库层(不可达即 SKIP) ----

// AC2:cost_plane 写入时落成事实字段,读回/聚合都按存储值,无查询期重算。
// P0-06:未知用量的调用照落行(usage_reported=false + NULL 数值),聚合的
// total/reported/missing 三列可重建,未知行不进数值合计(锁 16)。
func TestLLMUsageRecordReadBackAndCostPlaneAggregateDB(t *testing.T) {
	st, pool := testMeasurementStore(t)
	ctx := context.Background()

	exec := LLMUsage{
		TaskID: "T-1", UseCase: "implement_patch", CostPlane: CostPlaneExecution,
		Provider: "anthropic", Runtime: "codex", Model: "test-model",
		PromptRevision: "prompt-3", BehaviorRevision: "behavior-1", ContextHash: "sha256:pack",
		InputTokens: 120, OutputTokens: 45, Latency: 1500 * time.Millisecond,
		CostMicros: 7_000_000, UsageReported: true,
		Status: UsageSucceeded, HumanIntervened: true,
		AuthoringProvenance: "authoring-v2", RecordedAt: testTime,
	}
	if _, err := st.RecordLLMUsage(ctx, exec); err != nil {
		t.Fatalf("RecordLLMUsage(exec): %v", err)
	}
	ctrl := LLMUsage{
		UseCase: "intake_interview", CostPlane: CostPlaneControl,
		Model: "test-model", InputTokens: 10, OutputTokens: 2,
		Latency: 200 * time.Millisecond, CostMicros: 3_000_000, UsageReported: true,
		Status: UsageSucceeded, RecordedAt: testTime.Add(time.Minute),
	}
	if _, err := st.RecordLLMUsage(ctx, ctrl); err != nil {
		t.Fatalf("RecordLLMUsage(ctrl,任务外控制面): %v", err)
	}
	// 失败调用按实际发生值照记(§30)。
	failed := LLMUsage{
		TaskID: "T-1", UseCase: "verify_draft", CostPlane: CostPlaneControl,
		InputTokens: 5, OutputTokens: 0, CostMicros: 1_000_000, UsageReported: true,
		Status: UsageFailed, FailureType: "rate_limited", RecordedAt: testTime.Add(2 * time.Minute),
	}
	if _, err := st.RecordLLMUsage(ctx, failed); err != nil {
		t.Fatalf("RecordLLMUsage(failed): %v", err)
	}
	// 未知用量的调用照样落行(锁 16):latency 与终态是 Tally 自观测,照记;
	// token/cost 未知,落 usage_reported=false + NULL,不伪装成 0。
	unknown := LLMUsage{
		TaskID: "T-1", UseCase: "implement_patch", CostPlane: CostPlaneExecution,
		Runtime: "codex", Latency: 800 * time.Millisecond,
		Status: UsageCancelled, RecordedAt: testTime.Add(3 * time.Minute),
	}
	unknownID, err := st.RecordLLMUsage(ctx, unknown)
	if err != nil {
		t.Fatalf("RecordLLMUsage(unknown): %v", err)
	}
	if unknownID == 0 {
		t.Fatal("未知用量的调用应有事实行 id(调用事实必须保留)")
	}

	forTask, err := st.LLMUsageForTask(ctx, "T-1")
	if err != nil || len(forTask) != 3 {
		t.Fatalf("LLMUsageForTask: %v %d", err, len(forTask))
	}
	if !sameUsage(forTask[0], exec) {
		t.Fatalf("execution 行读回不一致:\n got %+v\nwant %+v", forTask[0], exec)
	}
	if !sameUsage(forTask[1], failed) {
		t.Fatalf("failed 行读回不一致:\n got %+v\nwant %+v", forTask[1], failed)
	}
	if !sameUsage(forTask[2], unknown) {
		t.Fatalf("未知用量行读回不一致:\n got %+v\nwant %+v", forTask[2], unknown)
	}
	// 库内事实:未知行的三列确实是 NULL、usage_reported 确实是 false。
	var allNull, usageReported bool
	if err := pool.QueryRow(context.Background(),
		`SELECT input_tokens IS NULL AND output_tokens IS NULL AND cost_micros IS NULL, usage_reported
		 FROM llm_usage WHERE id = $1`, unknownID).Scan(&allNull, &usageReported); err != nil {
		t.Fatalf("读未知行原始列: %v", err)
	}
	if !allNull || usageReported {
		t.Fatalf("未知用量行应为 NULL 数值 + usage_reported=false: null=%v reported=%v", allNull, usageReported)
	}
	// 任务外控制面调用:task_id 为空,不进任何任务投影。
	outOfTask, err := st.LLMUsageForTask(ctx, "T-missing")
	if err != nil || len(outOfTask) != 0 {
		t.Fatalf("其他任务投影应为空: %v %d", err, len(outOfTask))
	}

	// 全库聚合:只按存储 cost_plane 分组,control 在前;total/reported/missing
	// 三列计数,数值合计只含已上报观测(未知行不进合计)。
	byPlane, err := st.LLMCostByPlane(ctx)
	if err != nil {
		t.Fatalf("LLMCostByPlane: %v", err)
	}
	want := []PlaneCost{
		{Plane: CostPlaneControl, CostMicros: 4_000_000, InputTokens: 15, OutputTokens: 2,
			Calls: 2, CallsReported: 2, CallsMissingUsage: 0},
		{Plane: CostPlaneExecution, CostMicros: 7_000_000, InputTokens: 120, OutputTokens: 45,
			Calls: 2, CallsReported: 1, CallsMissingUsage: 1},
	}
	if !reflect.DeepEqual(byPlane, want) {
		t.Fatalf("按面聚合不一致:\n got %+v\nwant %+v", byPlane, want)
	}
	if got, ok := ControlPlaneRatio(byPlane); !ok || got != 4_000_000.0/11_000_000.0 {
		t.Fatalf("control_plane_llm_cost_ratio 应 4M/11M≈0.3636,得到 %v %v", got, ok)
	}
	forTaskCost, err := st.LLMCostByPlaneForTask(ctx, "T-1")
	if err != nil {
		t.Fatalf("LLMCostByPlaneForTask: %v", err)
	}
	// T-1 的 verify_draft 是控制面调用(intake 那条不属于任何任务)。
	wantTask := []PlaneCost{
		{Plane: CostPlaneControl, CostMicros: 1_000_000, InputTokens: 5, OutputTokens: 0,
			Calls: 1, CallsReported: 1, CallsMissingUsage: 0},
		{Plane: CostPlaneExecution, CostMicros: 7_000_000, InputTokens: 120, OutputTokens: 45,
			Calls: 2, CallsReported: 1, CallsMissingUsage: 1},
	}
	if !reflect.DeepEqual(forTaskCost, wantTask) {
		t.Fatalf("单任务聚合不一致:\n got %+v\nwant %+v", forTaskCost, wantTask)
	}

	// AC2 第一道防线:非法归属面/缺 use_case/负 token/未上报却携带数值
	// (锁 16)在写入前拒绝,不落行。
	baseline := countRows(t, pool, "SELECT count(*) FROM llm_usage")
	bad := validLLMUsage()
	bad.CostPlane = CostPlane("galactic")
	if _, err := st.RecordLLMUsage(ctx, bad); err == nil {
		t.Fatal("非法 cost_plane 应报错")
	}
	missingUseCase := validLLMUsage()
	missingUseCase.UseCase = " "
	if _, err := st.RecordLLMUsage(ctx, missingUseCase); err == nil {
		t.Fatal("空 use_case 应报错")
	}
	negative := validLLMUsage()
	negative.UsageReported = true
	negative.CostMicros = -1
	if _, err := st.RecordLLMUsage(ctx, negative); err == nil {
		t.Fatal("负 cost 应报错")
	}
	unreportedWithValues := validLLMUsage()
	unreportedWithValues.InputTokens = 7
	if _, err := st.RecordLLMUsage(ctx, unreportedWithValues); err == nil {
		t.Fatal("未上报却携带数值应报错(未知不是 0,锁 16)")
	}
	if n := countRows(t, pool, "SELECT count(*) FROM llm_usage"); n != baseline {
		t.Fatalf("非法输入不应落行:baseline %d,实际 %d", baseline, n)
	}
}

// sameUsage/sameEffort 逐字段比较;时刻用 Equal(pgx 读回的 timestamptz
// 携带本地时区位置,同一时刻的 struct 相等不成立)。
func sameUsage(a, b LLMUsage) bool {
	return a.TaskID == b.TaskID && a.UseCase == b.UseCase && a.CostPlane == b.CostPlane &&
		a.Provider == b.Provider && a.Runtime == b.Runtime && a.Model == b.Model &&
		a.PromptRevision == b.PromptRevision && a.BehaviorRevision == b.BehaviorRevision &&
		a.ContextHash == b.ContextHash && a.InputTokens == b.InputTokens &&
		a.OutputTokens == b.OutputTokens && a.Latency == b.Latency &&
		a.CostMicros == b.CostMicros && a.UsageReported == b.UsageReported && a.Status == b.Status &&
		a.FailureType == b.FailureType && a.HumanIntervened == b.HumanIntervened &&
		a.AuthoringProvenance == b.AuthoringProvenance && a.RecordedAt.Equal(b.RecordedAt)
}

func sameEffort(a, b HumanEffort) bool {
	return a.TaskID == b.TaskID && a.Kind == b.Kind && a.Active == b.Active && a.Wall == b.Wall && a.RecordedAt.Equal(b.RecordedAt)
}

// AC3:人工计时按事实行落库,跨行聚合在读侧完成;任务上没有任何手工
// 记账字段参与计算。
func TestHumanEffortAggregateWithoutManualTaskFieldDB(t *testing.T) {
	st, pool := testMeasurementStore(t)
	ctx := context.Background()

	segments := []HumanEffort{
		{TaskID: "T-1", Kind: EffortDiffReview, Active: 10 * time.Minute, Wall: 15 * time.Minute, RecordedAt: testTime},
		{TaskID: "T-1", Kind: EffortDiffReview, Active: 5 * time.Minute, Wall: 10 * time.Minute, RecordedAt: testTime.Add(time.Minute)},
		{TaskID: "T-1", Kind: EffortContractReview, Active: 12 * time.Minute, Wall: 12 * time.Minute, RecordedAt: testTime.Add(2 * time.Minute)},
		{TaskID: "T-1", Kind: EffortManualCorrection, Active: 3 * time.Minute, Wall: 0, RecordedAt: testTime.Add(3 * time.Minute)},
	}
	for i, h := range segments {
		if _, err := st.RecordHumanEffort(ctx, h); err != nil {
			t.Fatalf("RecordHumanEffort(%d): %v", i, err)
		}
	}

	rows, err := st.HumanEffortForTask(ctx, "T-1")
	if err != nil || len(rows) != 4 {
		t.Fatalf("HumanEffortForTask: %v %d", err, len(rows))
	}
	for i := range segments {
		if !sameEffort(rows[i], segments[i]) {
			t.Fatalf("分段行 %d 读回不一致: %+v != %+v", i, rows[i], segments[i])
		}
	}

	sums, err := st.HumanEffortByKindForTask(ctx, "T-1")
	if err != nil {
		t.Fatalf("HumanEffortByKindForTask: %v", err)
	}
	want := []EffortSummary{
		{Kind: EffortContractReview, Active: 12 * time.Minute, Wall: 12 * time.Minute, Samples: 1},
		{Kind: EffortDiffReview, Active: 15 * time.Minute, Wall: 25 * time.Minute, Samples: 2},
		{Kind: EffortManualCorrection, Active: 3 * time.Minute, Wall: 0, Samples: 1},
	}
	if !reflect.DeepEqual(sums, want) {
		t.Fatalf("按 kind 聚合不一致:\n got %+v\nwant %+v", sums, want)
	}
	diff := sums[1]
	if diff.ActiveMinutes() != 15 || diff.WallMinutes() != 25 {
		t.Fatalf("*_minutes 换算不一致: active=%v wall=%v", diff.ActiveMinutes(), diff.WallMinutes())
	}
	if other, err := st.HumanEffortByKindForTask(ctx, "T-missing"); err != nil || len(other) != 0 {
		t.Fatalf("无计时任务聚合应为空: %v %d", err, len(other))
	}

	// 非法输入在写入前拒绝,不落行。
	baseline := countRows(t, pool, "SELECT count(*) FROM human_effort")
	bad := []HumanEffort{
		{TaskID: "", Kind: EffortDiffReview, Active: time.Minute, RecordedAt: testTime},
		{TaskID: "T-1", Kind: HumanEffortKind("meeting_minutes"), Active: time.Minute, RecordedAt: testTime},
		{TaskID: "T-1", Kind: EffortDiffReview, Active: -time.Minute, RecordedAt: testTime},
		{TaskID: "T-1", Kind: EffortDiffReview, Active: time.Minute},
	}
	for i, h := range bad {
		if _, err := st.RecordHumanEffort(ctx, h); err == nil {
			t.Fatalf("非法人工计时输入 %d 应报错", i)
		}
	}
	if n := countRows(t, pool, "SELECT count(*) FROM human_effort"); n != baseline {
		t.Fatalf("非法输入不应落行:baseline %d,实际 %d", baseline, n)
	}
}

// INV-FACT-01 双保险:两张度量事实表的 UPDATE/DELETE 由行级触发器拦截。
func TestMeasurementAppendOnlyTriggersDB(t *testing.T) {
	st, pool := testMeasurementStore(t)
	ctx := context.Background()

	usageID, err := st.RecordLLMUsage(ctx, validLLMUsage())
	if err != nil {
		t.Fatalf("RecordLLMUsage: %v", err)
	}
	effortID, err := st.RecordHumanEffort(ctx, HumanEffort{
		TaskID: "T-1", Kind: EffortDiffReview, Active: time.Minute, RecordedAt: testTime,
	})
	if err != nil {
		t.Fatalf("RecordHumanEffort: %v", err)
	}

	mustModifyRejected(t, pool, "UPDATE llm_usage SET cost_micros=1 WHERE id=$1", usageID)
	mustModifyRejected(t, pool, "DELETE FROM llm_usage WHERE id=$1", usageID)
	mustModifyRejected(t, pool, "UPDATE human_effort SET active_micros=0 WHERE id=$1", effortID)
	mustModifyRejected(t, pool, "DELETE FROM human_effort WHERE id=$1", effortID)
	// 双保险兜底:改任务状态类语句本就不存在 API 面;这里固化任何直接
	// UPDATE 事实表都不可行(含全部行)。
	mustModifyRejected(t, pool, "UPDATE llm_usage SET status='succeeded'")
	mustModifyRejected(t, pool, "DELETE FROM human_effort")
}
