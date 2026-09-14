//go:build integration

package store

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/core"
)

// 集成测试用真实 Postgres(不 mock 数据库,见 docs/05-test-standards.md)。
// 每个测试用 TestDB 拿独立 schema。

func openStore(t *testing.T) *Store {
	t.Helper()
	return TestDB(t)
}

// uniqueFP 每次测试用独立指纹:同库内还有别的种子数据,唯一值让断言锁定本测试种的那行。
func uniqueFP(t *testing.T) string {
	t.Helper()
	return "t06" + time.Now().Format("150405.000000000")[:13]
}

func TestUpsertFingerprintStatAccumulates(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	fp := uniqueFP(t)
	at := time.Now().UTC()

	row, err := st.Queries().UpsertFingerprintStat(ctx, UpsertFingerprintStatParams{
		FpExact: fp, FpFuzzy: fp + "f", FirstSeen: pgtype.Timestamptz{Time: at, Valid: true}, Project: "demo",
	})
	if err != nil {
		t.Fatalf("首次 upsert: %v", err)
	}
	if row.OccurrenceCount != 1 || len(row.AffectedProjects) != 1 || row.Status != "unresolved" {
		t.Errorf("新行应是 1 次/1 项目/unresolved: %+v", row)
	}

	// 同指纹再来:次数 +1,同项目不重复。
	row, err = st.Queries().UpsertFingerprintStat(ctx, UpsertFingerprintStatParams{
		FpExact: fp, FpFuzzy: fp + "f", FirstSeen: pgtype.Timestamptz{Time: at, Valid: true}, Project: "demo",
	})
	if err != nil {
		t.Fatalf("二次 upsert: %v", err)
	}
	if row.OccurrenceCount != 2 || len(row.AffectedProjects) != 1 {
		t.Errorf("应是 2 次/1 项目(同项目不重复): %+v", row)
	}

	// 另一个项目命中同指纹:并入受影响项目(跨项目信号)。
	row, err = st.Queries().UpsertFingerprintStat(ctx, UpsertFingerprintStatParams{
		FpExact: fp, FpFuzzy: fp + "f", FirstSeen: pgtype.Timestamptz{Time: at, Valid: true}, Project: "other",
	})
	if err != nil {
		t.Fatalf("三次 upsert: %v", err)
	}
	if row.OccurrenceCount != 3 || len(row.AffectedProjects) != 2 {
		t.Errorf("应是 3 次/2 项目: %+v", row)
	}
}

func TestFingerprintStatLifecycle(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	fp := uniqueFP(t)
	at := time.Now().UTC()

	if _, err := st.Queries().GetFingerprintStat(ctx, fp); err == nil {
		t.Fatal("不存在的行应报错")
	} else if err != pgx.ErrNoRows {
		t.Fatalf("应是 ErrNoRows, got %v", err)
	}

	if _, err := st.Queries().UpsertFingerprintStat(ctx, UpsertFingerprintStatParams{
		FpExact: fp, FpFuzzy: fp + "f", FirstSeen: pgtype.Timestamptz{Time: at, Valid: true}, Project: "demo",
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// 预算:诊断次数累加。
	for i := 0; i < 3; i++ {
		if err := st.Queries().BumpDiagnoseAttempts(ctx, fp); err != nil {
			t.Fatalf("bump: %v", err)
		}
	}
	row, err := st.Queries().GetFingerprintStat(ctx, fp)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if row.DiagnoseAttempts != 3 {
		t.Errorf("诊断次数 = %d, want 3", row.DiagnoseAttempts)
	}

	// 归类出口(指纹索引 status 由 failure 归类函数写,INV-3 登记的例外)。
	err = st.Queries().UpdateFingerprintStatClassification(ctx, UpdateFingerprintStatClassificationParams{
		FpExact: fp, Status: "flaky",
	})
	if err != nil {
		t.Fatalf("归类: %v", err)
	}
	if row, _ = st.Queries().GetFingerprintStat(ctx, fp); row.Status != "flaky" {
		t.Errorf("状态应是 flaky, got %s", row.Status)
	}

	// 修复入库回写:status=fixed + 引用 + 验证边界。
	combo, _ := json.Marshal(core.VersionCombo{"go": "1.23.0"})
	err = st.Queries().UpdateFingerprintStatFix(ctx, UpdateFingerprintStatFixParams{
		FpExact:          fp,
		FixRef:           pgtype.Text{String: "deadbeef", Valid: true},
		FixVerifiedAt:    pgtype.Timestamptz{Time: at, Valid: true},
		FixVerifiedCombo: combo,
	})
	if err != nil {
		t.Fatalf("写修复索引: %v", err)
	}
	row, err = st.Queries().GetFingerprintStat(ctx, fp)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if row.Status != "fixed" || row.FixRef.String != "deadbeef" {
		t.Errorf("应是 fixed/deadbeef: %+v", row)
	}
	var got core.VersionCombo
	if err := json.Unmarshal(row.FixVerifiedCombo, &got); err != nil || got["go"] != "1.23.0" {
		t.Errorf("验证组合应回读: %v %v", got, err)
	}
}

func TestListFingerprintStatsByFuzzy(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	fuzzy := uniqueFP(t) + "fz"
	at := time.Now().UTC()

	for _, exact := range []string{fuzzy[:16] + "a", fuzzy[:16] + "b"} {
		if _, err := st.Queries().UpsertFingerprintStat(ctx, UpsertFingerprintStatParams{
			FpExact: exact, FpFuzzy: fuzzy, FirstSeen: pgtype.Timestamptz{Time: at, Valid: true}, Project: "demo",
		}); err != nil {
			t.Fatalf("upsert: %v", err)
		}
	}

	rows, err := st.Queries().ListFingerprintStatsByFuzzy(ctx, fuzzy)
	if err != nil {
		t.Fatalf("按模糊指纹查: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("应召回 2 行, got %d", len(rows))
	}
}

func TestInsertFailureEventAndDepChanges(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	fp := uniqueFP(t)
	// 项目 id 也要每次唯一:dependency_change 是事实面(只追加,清不掉),
	// 固定 id 会让窗口计数在重复运行间累积,和指纹用 uniqueFP 是同一个理由。
	proj := "demo-" + fp
	at := time.Now().UTC()
	combo, _ := json.Marshal(core.VersionCombo{"go": "1.23.0"})

	// failure_event 的来源外键需要真实行:先记一次环境拉起。
	envID, err := st.Queries().InsertEnvironmentRun(ctx, InsertEnvironmentRunParams{
		ProjectID:      proj,
		EnvName:        "dev",
		DefinitionHash: "sha256:def",
		StartedAt:      pgtype.Timestamptz{Time: at, Valid: true},
		Status:         "failed",
		VersionCombo:   combo,
	})
	if err != nil {
		t.Fatalf("记环境拉起: %v", err)
	}

	eventID, err := st.Queries().InsertFailureEvent(ctx, InsertFailureEventParams{
		ProjectID:           proj,
		Source:              "environment",
		EnvironmentRunID:    pgtype.Int8{Int64: envID, Valid: true},
		Excerpt:             pgtype.Text{String: "health check failed", Valid: true},
		FpExact:             fp,
		FpFuzzy:             fp + "f",
		PipelineVersion:     "1",
		OccurredAt:          pgtype.Timestamptz{Time: at, Valid: true},
		VersionCombo:        combo,
		OtherTargetCommands: []byte("[]"),
		ReproCommand:        []byte("null"),
	})
	if err != nil {
		t.Fatalf("记失败事件: %v", err)
	}
	if eventID == 0 {
		t.Error("事件 id 应非零")
	}

	// 依赖变更时间线(上游归类信号)。
	if _, err := st.Queries().InsertDependencyChange(ctx, InsertDependencyChangeParams{
		ProjectID: proj, CommitSha: "abc", Package: "libfoo",
		ToVersion: pgtype.Text{String: "2.0.0", Valid: true},
		ChangedAt: pgtype.Timestamptz{Time: at.Add(-time.Hour), Valid: true},
	}); err != nil {
		t.Fatalf("记依赖变更: %v", err)
	}

	n, err := st.Queries().CountDependencyChangesSince(ctx, CountDependencyChangesSinceParams{
		ProjectID: proj, ChangedAt: pgtype.Timestamptz{Time: at.Add(-24 * time.Hour), Valid: true},
	})
	if err != nil || n != 1 {
		t.Errorf("窗口内应有 1 条依赖变更: n=%d err=%v", n, err)
	}
	n, err = st.Queries().CountDependencyChangesSince(ctx, CountDependencyChangesSinceParams{
		ProjectID: proj, ChangedAt: pgtype.Timestamptz{Time: at.Add(time.Hour), Valid: true},
	})
	if err != nil || n != 0 {
		t.Errorf("窗口外应无依赖变更: n=%d err=%v", n, err)
	}
}

// 事实面只追加:对 failure_event 的 UPDATE/DELETE 应被数据库触发器挡下(INV-5 双保险)。
func TestFailureEventAppendOnlyEnforced(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	if _, err := st.Pool().Exec(ctx,
		`UPDATE failure_event SET excerpt = 'x' WHERE project_id = 'no-such-project'`); err == nil {
		t.Error("事实面 UPDATE 应被触发器拒绝(INV-5)")
	}
}
