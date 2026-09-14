//go:build integration

package failure

// failure 对 store 的真实接线在集成测试里覆盖(单测用内存 fake,见 budget_test.go):
// RecordEvent 同事务写事实面 + 累加索引面;statStore 包装 store.Queries 的五个方法。
// 用真实 Postgres,不 mock 数据库(docs/05-test-standards.md);每个测试用 store.TestDB 拿独立 schema。

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/command"
	"tally/internal/core"
	"tally/internal/store"
)

func openStore(t *testing.T) *store.Store {
	t.Helper()
	return store.TestDB(t)
}

// uniqueFP 每次测试用不同指纹,避免跨运行污染(事实面只追加,清不掉)。
func uniqueFP(t *testing.T, salt int) core.Fingerprint {
	t.Helper()
	return core.Fingerprint("06" + time.Now().Format("150405.000000000")[:13] + string(rune('a'+salt)))
}

// uniqueProject 项目 id 同样要唯一:dependency_change 等事实面按项目查询,
// 固定 id 会让其他包的集成测试在重复运行间互相串扰。
func uniqueProject(fp core.Fingerprint) string { return "demo-" + string(fp) }

// eventWithEnvRun 记一次环境拉起(失败事件来源外键需要真实行),返回带好了的 Event。
func eventWithEnvRun(t *testing.T, st *store.Store, fp core.Fingerprint) Event {
	t.Helper()
	ctx := context.Background()
	at := time.Now().UTC()
	envID, err := st.Queries().InsertEnvironmentRun(ctx, store.InsertEnvironmentRunParams{
		ProjectID:      uniqueProject(fp),
		EnvName:        "verify",
		DefinitionHash: "sha256:def",
		StartedAt:      pgtype.Timestamptz{Time: at, Valid: true},
		Status:         "failed",
		VersionCombo:   []byte(`{"go":"1.23.0"}`),
	})
	if err != nil {
		t.Fatalf("记环境拉起: %v", err)
	}
	return Event{
		ProjectID:        uniqueProject(fp),
		Source:           "environment",
		EnvironmentRunID: envID,
		Excerpt:          "health check failed",
		Normalized:       "health check failed",
		PipelineVersion:  "1",
		Exact:            fp,
		Fuzzy:            core.Fingerprint(string(fp) + "f"),
		Combo:            core.VersionCombo{"go": "1.23.0"},
		OccurredAt:       at,
	}
}

// RecordEvent:事实面 INSERT 与索引面累加同一事务;事件 id 非零,索引行随之而生。
func TestRecordEventWritesEventAndStat(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	fp := uniqueFP(t, 0)

	id, err := RecordEvent(ctx, st, eventWithEnvRun(t, st, fp))
	if err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	if id == 0 {
		t.Error("事件 id 应非零")
	}

	row, err := st.Queries().GetFingerprintStat(ctx, string(fp))
	if err != nil {
		t.Fatalf("索引行应与事件同事务写入: %v", err)
	}
	if row.OccurrenceCount != 1 || row.Status != "unresolved" || row.DiagnoseAttempts != 0 {
		t.Errorf("新指纹应是 1 次/unresolved/0 轮: %+v", row)
	}

	// 同指纹再来一次:次数 +1(索引累加,事实面追加第二条)。
	if _, err := RecordEvent(ctx, st, eventWithEnvRun(t, st, fp)); err != nil {
		t.Fatalf("再次 RecordEvent: %v", err)
	}
	row, err = st.Queries().GetFingerprintStat(ctx, string(fp))
	if err != nil {
		t.Fatalf("取索引行: %v", err)
	}
	if row.OccurrenceCount != 2 {
		t.Errorf("同指纹应是 2 次: %+v", row)
	}
}

// RecordEvent:三个来源外键必须恰好一个非空 —— 0 个或 2 个都拒绝,不碰数据库。
func TestRecordEventRejectsBadSourceCount(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()

	ev := eventWithEnvRun(t, st, uniqueFP(t, 1))
	ev.EnvironmentRunID = 0 // 0 个来源
	if _, err := RecordEvent(ctx, st, ev); err == nil {
		t.Error("零来源应拒绝")
	}
	ev.EnvironmentRunID = 1
	ev.BuildRunID = 1 // 2 个来源
	if _, err := RecordEvent(ctx, st, ev); err == nil {
		t.Error("两个来源应拒绝")
	}
}

// RecordEvent(T11b):复现上下文三列随失败持久化,GetLatestFailureEventByFingerprint
// 原样读回。留空也合法(NOT NULL DEFAULT,旧失败走「跳过 G1/G2」路径)。
func TestRecordEventPersistsReproContext(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	fp := uniqueFP(t, 3)

	ev := eventWithEnvRun(t, st, fp)
	ev.Repro = command.CommandSpec{Argv: []string{"make", "build"}, Cwd: "/tmp/demo"}
	ev.OtherTargets = []OtherTarget{
		{Target: "other-a", Command: command.CommandSpec{Argv: []string{"make", "other-a"}}},
		{Target: "other-b", Command: command.CommandSpec{Argv: []string{"make", "other-b"}}},
	}
	if _, err := RecordEvent(ctx, st, ev); err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}

	row, err := st.Queries().GetLatestFailureEventByFingerprint(ctx, string(fp))
	if err != nil {
		t.Fatalf("读回失败事件: %v", err)
	}
	// 0016 双轨合同:结构化写入 repro_command/other_target_commands,旧文本列不再写。
	var gotSpec command.CommandSpec
	if err := json.Unmarshal(row.ReproCommand, &gotSpec); err != nil {
		t.Fatalf("repro_command 应是 CommandSpec JSON: %v", err)
	}
	if gotSpec.Cwd != "/tmp/demo" || strings.Join(gotSpec.Argv, " ") != "make build" {
		t.Errorf("复现上下文应原样落库: %+v", gotSpec)
	}
	var gotTargets []OtherTarget
	if err := json.Unmarshal(row.OtherTargetCommands, &gotTargets); err != nil {
		t.Fatalf("other_target_commands 应是 JSON 数组: %v", err)
	}
	if len(gotTargets) != 2 || strings.Join(gotTargets[0].Command.Argv, " ") != "make other-a" ||
		strings.Join(gotTargets[1].Command.Argv, " ") != "make other-b" {
		t.Errorf("其余构建目标应原样落库: %v", gotTargets)
	}

	// 留空的失败(部署/环境语义):三列取默认值,不报错。
	fp2 := uniqueFP(t, 4)
	if _, err := RecordEvent(ctx, st, eventWithEnvRun(t, st, fp2)); err != nil {
		t.Fatalf("无复现上下文的 RecordEvent: %v", err)
	}
	row, err = st.Queries().GetLatestFailureEventByFingerprint(ctx, string(fp2))
	if err != nil {
		t.Fatalf("读回无复现上下文的失败: %v", err)
	}
	if row.Workdir != "" || row.Repro != "" {
		t.Errorf("留空应是默认值: workdir=%q repro=%q", row.Workdir, row.Repro)
	}
	gotTargets = nil
	if err := json.Unmarshal(row.OtherTargets, &gotTargets); err != nil || len(gotTargets) != 0 {
		t.Errorf("留空应是空 JSON 数组: %s (err=%v)", row.OtherTargets, err)
	}
}

// statStore(NewStatStore 包装 store.Queries)五个方法的真实接线。
func TestStatStoreWiring(t *testing.T) {
	st := openStore(t)
	ctx := context.Background()
	fp := uniqueFP(t, 2)
	ss := NewStatStore(st.Queries())

	// 行不存在 → ErrNoStat(哨兵,预算逻辑靠它分流)。
	if _, err := ss.GetStat(ctx, fp); !errors.Is(err, ErrNoStat) {
		t.Fatalf("不存在的行应 ErrNoStat, got %v", err)
	}

	if _, err := RecordEvent(ctx, st, eventWithEnvRun(t, st, fp)); err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	stat, err := ss.GetStat(ctx, fp)
	if err != nil {
		t.Fatalf("GetStat: %v", err)
	}
	if stat.Status != "unresolved" || stat.DiagnoseAttempts != 0 || len(stat.AffectedProjects) != 1 {
		t.Errorf("新行应是 unresolved/0 轮/1 项目: %+v", stat)
	}

	// 预算累加:三轮上限的地基。
	for i := 0; i < 3; i++ {
		if err := ss.BumpAttempts(ctx, fp); err != nil {
			t.Fatalf("BumpAttempts: %v", err)
		}
	}
	if stat, _ = ss.GetStat(ctx, fp); stat.DiagnoseAttempts != 3 {
		t.Errorf("诊断次数应累加到 3, got %d", stat.DiagnoseAttempts)
	}

	// 归类出口(INV-3 登记的例外:status 只由 failure 归类函数写)。
	if err := ss.SetClassification(ctx, fp, OutcomeFlaky); err != nil {
		t.Fatalf("SetClassification: %v", err)
	}
	if stat, _ = ss.GetStat(ctx, fp); stat.Status != "flaky" {
		t.Errorf("归类后状态应是 flaky, got %s", stat.Status)
	}

	// 修复入库回写:引用 + 验证边界(时刻、版本组合)。
	at := time.Now().UTC()
	combo := core.VersionCombo{"go": "1.23.0"}
	if err := ss.SetFix(ctx, fp, "deadbeef", at, combo); err != nil {
		t.Fatalf("SetFix: %v", err)
	}
	if stat, _ = ss.GetStat(ctx, fp); stat.Status != "fixed" || stat.FixRef != "deadbeef" {
		t.Errorf("入库后应是 fixed/deadbeef: %+v", stat)
	}

	// 依赖变更时间线(upstream 归类的确定性信号)。
	proj := uniqueProject(fp)
	since := at.Add(-time.Hour)
	if got, err := ss.HasRecentDepChange(ctx, proj, since); err != nil || got {
		t.Errorf("窗口内无依赖变更应是 false: got=%v err=%v", got, err)
	}
	if _, err := st.Queries().InsertDependencyChange(ctx, store.InsertDependencyChangeParams{
		ProjectID: proj, CommitSha: "abc123", Package: "libfoo",
		ToVersion: pgtype.Text{String: "2.0.0", Valid: true},
		ChangedAt: pgtype.Timestamptz{Time: at.Add(-30 * time.Minute), Valid: true},
	}); err != nil {
		t.Fatalf("记依赖变更: %v", err)
	}
	if got, err := ss.HasRecentDepChange(ctx, proj, since); err != nil || !got {
		t.Errorf("窗口内有依赖变更应是 true: got=%v err=%v", got, err)
	}
}
