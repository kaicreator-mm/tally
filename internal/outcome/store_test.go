package outcome

// store.go(P3-05)适配器测试。
//
// 两层,同一文件(与 internal/measurement 同一惯例):
//   - 纯单元层:事件行重建校验、API 面无改写/删除入口(不触库),
//     `go test ./internal/outcome/...` 无 DB 也全绿;
//   - 数据库层:写入/读回往返、append-only 触发器拒绝路径、未知 kind 的
//     CHECK 拒绝路径。每个测试独立 schema、自跑 goose 迁移、cleanup DROP;
//     数据库不可达时 SKIP,不产出假失败。

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

const outcomeTestDSN = "postgres://postgres:postgres@localhost:5432/tally?sslmode=disable"

// ---- 测试库:每测试独立 schema,自跑迁移,cleanup DROP;不可达即 SKIP ----

func testOutcomeStore(t *testing.T) (*Store, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		base = outcomeTestDSN
	}
	if !dbReachable(base) {
		t.Skipf("outcome 数据库层测试需要 PostgreSQL(%s 不可达),跳过", base)
	}
	schema := "tally_outcome_" + strconv.FormatInt(time.Now().UnixNano(), 10)

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

// mustModifyRejected 断言一条改写/删除语句被 append-only 触发器拒绝
// (AC3 的数据库侧防线;适配器无 API 只是第一道防线)。
func mustModifyRejected(t *testing.T, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	_, err := pool.Exec(context.Background(), query, args...)
	if err == nil {
		t.Fatalf("append-only 触发器未拦截:%s", query)
	}
}

// ---- 纯单元层(无 DB) ----

// AC1 的读侧镜像:kind 越出 0023 CHECK 取值的行(行破坏)重建时报错,
// 不被当真、更不被静默折叠进已知 kind(fail-safe)。
func TestEventRowReconstructionRejectsCorruption(t *testing.T) {
	row := store.OutcomeEvent{
		ID:          1,
		TaskID:      pgtype.Text{String: "T-1", Valid: true},
		ChangesetID: pgtype.Text{String: "cs-abc", Valid: true},
		EventKind:   "accepted",
		RecordedAt:  pgtype.Timestamptz{Time: testTime, Valid: true},
	}
	got, err := eventFromRow(row)
	if err != nil || got.Kind != KindAccepted || got.TaskID != "T-1" || got.ChangesetID != "cs-abc" {
		t.Fatalf("合法行重建不一致: %v %+v", err, got)
	}

	// 完全无绑定的行(NULL 双空):领域侧视为行破坏 —— ForTask/ForChangeset
	// 的 WHERE 保证读路径不会命中它,但若出现在重建路径上必须报错而不是
	// 静默收下孤儿事件(至少绑定其一)。
	unbound := row
	unbound.TaskID, unbound.ChangesetID = pgtype.Text{}, pgtype.Text{}
	if _, err := eventFromRow(unbound); err == nil {
		t.Fatal("完全无绑定的行应报错(不被静默收下孤儿事件)")
	}

	// 行破坏:kind 越出 CHECK 取值(库内正常被 CHECK 拦,读侧兜底)。
	bad := row
	bad.EventKind = "merged"
	if _, err := eventFromRow(bad); err == nil {
		t.Fatal("未知 kind 的行应报错(不被静默折叠)")
	}
}

// AC3/AC5:适配器不存在任何更新/删除/状态入口 API —— outcome 事实落库后
// 只读,「当前 outcome」由读侧投影,不在本包落可变权威。
func TestOutcomeStoreAPIHasNoMutation(t *testing.T) {
	typ := reflect.TypeOf(&Store{})
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		for _, banned := range []string{"Update", "Delete", "Rewrite", "Truncate", "Drop", "SetTask", "SetStatus", "Mark"} {
			if strings.HasPrefix(name, banned) {
				t.Errorf("outcome 事实面出现改写/删除/状态入口 API:%s(AC3/AC5)", name)
			}
		}
	}
}

// ---- 数据库层(不可达即 SKIP) ----

// AC2:写入/读回往返 —— 事件、绑定坐标与写入时间一次落成;同一任务的完整
// 历史旧→新可重放(append-only 事件流,读侧投影「当前 outcome」)。
func TestOutcomeEventRoundtrip(t *testing.T) {
	st, pool := testOutcomeStore(t)
	ctx := context.Background()

	accepted := Event{TaskID: "T-42", ChangesetID: "cs-1", Kind: KindAccepted, RecordedAt: testTime}
	id1, err := st.Record(ctx, accepted)
	if err != nil {
		t.Fatalf("写入 accepted: %v", err)
	}
	if id1 <= 0 {
		t.Fatalf("新行 id 应为正: %d", id1)
	}

	// 同一任务后来的回归与人工修正:两行事实,历史完整可重放(锁 14:
	// 不是 UPDATE 一个可变状态列)。
	later := testTime.Add(time.Hour)
	id2, err := st.Record(ctx, Event{TaskID: "T-42", ChangesetID: "cs-1", Kind: KindRegressionFound, RecordedAt: later})
	if err != nil {
		t.Fatalf("写入 regression_found: %v", err)
	}
	id3, err := st.Record(ctx, Event{TaskID: "T-42", Kind: KindHumanCorrection, RecordedAt: later.Add(time.Minute)})
	if err != nil {
		t.Fatalf("写入 human_correction(仅 task 绑定): %v", err)
	}
	if !(id1 < id2 && id2 < id3) {
		t.Fatalf("事件 id 应单调递增: %d %d %d", id1, id2, id3)
	}

	events, err := st.ForTask(ctx, "T-42")
	if err != nil {
		t.Fatalf("读回任务事件: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("应读回 3 行,得到 %d: %+v", len(events), events)
	}
	wantKinds := []Kind{KindAccepted, KindRegressionFound, KindHumanCorrection}
	for i, e := range events {
		if e.Kind != wantKinds[i] || e.TaskID != "T-42" {
			t.Errorf("events[%d] = %+v, want kind=%s task=T-42(旧→新)", i, e, wantKinds[i])
		}
	}
	if events[0].ChangesetID != "cs-1" || events[1].ChangesetID != "cs-1" || events[2].ChangesetID != "" {
		t.Errorf("changeset 绑定读回不一致: %+v", events)
	}
	if !events[0].RecordedAt.Equal(testTime) {
		t.Errorf("recorded_at 往返不一致: %v want %v", events[0].RecordedAt, testTime)
	}

	// 候选维度的读路径:cs-1 关联两行,cs-2 无事件。
	byCS, err := st.ForChangeset(ctx, "cs-1")
	if err != nil {
		t.Fatalf("读回候选事件: %v", err)
	}
	if len(byCS) != 2 {
		t.Fatalf("cs-1 应关联 2 行,得到 %d", len(byCS))
	}
	if empty, err := st.ForChangeset(ctx, "cs-none"); err != nil || len(empty) != 0 {
		t.Fatalf("无事件候选应返回空切片: %v %v", empty, err)
	}

	if n := countOutcomeRows(t, pool); n != 3 {
		t.Fatalf("outcome_event 应恰有 3 行,得到 %d", n)
	}
}

// AC2:上下文外事件(事故热修)允许只有 changeset 绑定 —— 采集点先于
// 归档完善,拒绝落行等于永久丢失无法补采的事实(§23 动机)。
func TestOutcomeEventChangesetOnlyBindingRoundtrip(t *testing.T) {
	st, _ := testOutcomeStore(t)
	ctx := context.Background()

	id, err := st.Record(ctx, Event{ChangesetID: "cs-hotfix", Kind: KindHotfixed, RecordedAt: testTime})
	if err != nil {
		t.Fatalf("仅 changeset 绑定的事件被拒绝: %v", err)
	}
	events, err := st.ForChangeset(ctx, "cs-hotfix")
	if err != nil || len(events) != 1 {
		t.Fatalf("读回热修事件: %v %+v", err, events)
	}
	if events[0].Kind != KindHotfixed || events[0].TaskID != "" || events[0].ChangesetID != "cs-hotfix" || id <= 0 {
		t.Errorf("热修事件往返不一致: %+v (id=%d)", events[0], id)
	}
}

// AC1:未知 kind 在写入路径被显式拒绝 —— 领域 Validate 先拦(不触库),
// 绕过领域直接写库时 0023 的 CHECK 拦(fail-safe 双防线,不静默折叠)。
func TestOutcomeEventUnknownKindRejected(t *testing.T) {
	st, pool := testOutcomeStore(t)
	ctx := context.Background()

	if _, err := st.Record(ctx, Event{TaskID: "T-9", Kind: Kind("merged"), RecordedAt: testTime}); err == nil {
		t.Fatal("未知 kind 被领域校验放行")
	}
	if _, err := pool.Exec(ctx,
		`INSERT INTO outcome_event (task_id, changeset_id, event_kind, recorded_at) VALUES ('T-9', NULL, 'merged', $1)`,
		pgtype.Timestamptz{Time: testTime, Valid: true}); err == nil {
		t.Fatal("未知 kind 绕过领域写入时未被 0023 CHECK 拒绝")
	}
}

// AC3:append-only 双保险 —— 数据库层拒绝 UPDATE/DELETE(语句级触发器:
// 0 行命中的改写语句本身也报错,0011 口径)。
func TestOutcomeEventAppendOnlyTrigger(t *testing.T) {
	st, pool := testOutcomeStore(t)
	ctx := context.Background()

	if _, err := st.Record(ctx, Event{TaskID: "T-7", ChangesetID: "cs-7", Kind: KindReverted, RecordedAt: testTime}); err != nil {
		t.Fatalf("写入 reverted: %v", err)
	}

	mustModifyRejected(t, pool, `UPDATE outcome_event SET event_kind = 'accepted' WHERE task_id = 'T-7'`)
	mustModifyRejected(t, pool, `UPDATE outcome_event SET event_kind = 'accepted' WHERE task_id = 'no-such-task'`)
	mustModifyRejected(t, pool, `DELETE FROM outcome_event WHERE task_id = 'T-7'`)
	mustModifyRejected(t, pool, `DELETE FROM outcome_event WHERE task_id = 'no-such-task'`)

	// 触发器拦截后行原样保留(历史不被破坏)。
	events, err := st.ForTask(ctx, "T-7")
	if err != nil || len(events) != 1 || events[0].Kind != KindReverted {
		t.Fatalf("触发器拦截后事件应原样保留: %v %+v", err, events)
	}
}

func countOutcomeRows(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	var n int64
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM outcome_event`).Scan(&n); err != nil {
		t.Fatalf("count outcome_event: %v", err)
	}
	return n
}
