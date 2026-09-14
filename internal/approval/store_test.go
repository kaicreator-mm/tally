package approval

// store.go(P1-34)适配器测试。
//
// 两层,同一文件(Allowed paths 只给 store_test.go 一个测试文件):
//   - 纯单元层:waiver_refs 编码/重建往返、零值写入前拒绝(不触库)、
//     API 面无任务状态变更入口(AC4),`go test ./internal/approval/...`
//     无 DB 也全绿;
//   - 数据库层:写入/读回往返、追加式历史(AC1)、触发器拒绝路径。
//     每个测试独立 schema、自跑 goose 迁移、cleanup DROP(与 internal/runstore
//     同一隔离策略);数据库不可达时 SKIP,不产出假失败。
//
// 领域校验用 errors.Is 判因,不做字符串匹配。测试固定量复用 waiver_test.go
// 的 testProject/testTask/testCS/testAt/validWaiverSpec,保证全部断言基于
// 确定性输入。

import (
	"context"
	"database/sql"
	"errors"
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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"tally/internal/store"
	"tally/migrations"
)

const approvalTestDSN = "postgres://postgres:postgres@localhost:5432/tally?sslmode=disable"

// ---- 测试库:每测试独立 schema,自跑迁移,cleanup DROP;不可达即 SKIP ----

func testApprovalStore(t *testing.T) (*Store, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		base = approvalTestDSN
	}
	if !dbReachable(base) {
		t.Skipf("approval 数据库层测试需要 PostgreSQL(%s 不可达),跳过", base)
	}
	schema := "tally_appr_" + strconv.FormatInt(time.Now().UnixNano(), 10)

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

func countApprovals(t *testing.T, pool *pgxpool.Pool, query string, args ...any) int64 {
	t.Helper()
	var n int64
	if err := pool.QueryRow(context.Background(), query, args...).Scan(&n); err != nil {
		t.Fatalf("count %q: %v", query, err)
	}
	return n
}

// mustModifyRejected 断言一条改写/删除语句被 append-only 触发器拒绝(AC1 的
// 数据库侧防线;适配器无 API 只是第一道防线)。
func mustModifyRejected(t *testing.T, pool *pgxpool.Pool, query string, args ...any) {
	t.Helper()
	_, err := pool.Exec(context.Background(), query, args...)
	if err == nil {
		t.Fatalf("append-only 触发器未拦截:%s", query)
	}
}

func mustApproval(t *testing.T, spec ApprovalSpec) HumanApprovalEvidence {
	t.Helper()
	a, err := NewApproval(spec)
	if err != nil {
		t.Fatalf("NewApproval: %v", err)
	}
	return a
}

// ---- 纯单元层(无 DB) ----

func TestWaiverRefsJSONRoundTrip(t *testing.T) {
	w := mustWaiver(t, validWaiverSpec())
	data, err := waiverRefsJSON(w)
	if err != nil {
		t.Fatalf("waiverRefsJSON: %v", err)
	}
	// 形状合同:migrations/0021 要求 JSON 数组;无豁免编码为 []。
	if !strings.HasPrefix(string(data), "[") {
		t.Fatalf("waiver_refs 必须 JSON 数组,得到 %s", data)
	}
	if empty, err := waiverRefsJSON(nil); err != nil || string(empty) != "[]" {
		t.Fatalf("无豁免应编码为 [],得到 %s(%v)", empty, err)
	}

	got, err := waiverFromRefs(data, w.ReasonCode())
	if err != nil {
		t.Fatalf("waiverFromRefs: %v", err)
	}
	if got.Category() != w.Category() || got.ReasonCode() != w.ReasonCode() ||
		got.AcceptedBy() != w.AcceptedBy() || !got.AcceptedAt().Equal(w.AcceptedAt()) {
		t.Fatalf("标量 provenance 往返不一致: %+v", got)
	}
	if !reflect.DeepEqual(got.Gaps(), w.Gaps()) ||
		!reflect.DeepEqual(got.CompromisedAcceptances(), w.CompromisedAcceptances()) ||
		!reflect.DeepEqual(got.CompromisedBindings(), w.CompromisedBindings()) ||
		!reflect.DeepEqual(got.ExposureRefs(), w.ExposureRefs()) {
		t.Fatalf("列表 provenance 往返不一致: %+v", got)
	}

	// 编码确定性:同一豁免两次编码逐字节一致。
	again, err := waiverRefsJSON(w)
	if err != nil || string(again) != string(data) {
		t.Fatalf("编码应确定: %s vs %s (%v)", again, data, err)
	}

	// 未知条目 kind 是行破坏:拒绝而非静默跳过。
	tampered := strings.Replace(string(data), `"kind":"category"`, `"kind":"mystery"`, 1)
	if _, err := waiverFromRefs([]byte(tampered), w.ReasonCode()); err == nil {
		t.Fatal("未知条目 kind 应报错")
	}
	// 空数组(声称豁免却无 provenance)在重建即被领域构造器拒绝。
	if _, err := waiverFromRefs([]byte(`[]`), w.ReasonCode()); !errors.Is(err, ErrInvalidWaiver) {
		t.Fatalf("空 waiver_refs 应报 ErrInvalidWaiver,得到 %v", err)
	}
	// 缺失 reason code(列值为空)同样是行破坏。
	if _, err := waiverFromRefs(data, ""); !errors.Is(err, ErrInvalidWaiver) {
		t.Fatalf("缺 reason code 应报 ErrInvalidWaiver,得到 %v", err)
	}
	// 非法 JSON 同样拒绝。
	if _, err := waiverFromRefs([]byte(`not-json`), w.ReasonCode()); err == nil {
		t.Fatal("非法 JSON 应报错")
	}
}

// TestZeroValueApprovalRejectedWithoutDB 固化写入前重校验合同:零值/矛盾
// 证据在触库之前被拒(RecordApproval 对 nil 查询集也必须先失败)。
func TestZeroValueApprovalRejectedWithoutDB(t *testing.T) {
	s := New(nil)
	if _, err := s.RecordApproval(context.Background(), HumanApprovalEvidence{}); !errors.Is(err, ErrInvalidApproval) {
		t.Fatalf("零值证据应报 ErrInvalidApproval,得到 %v", err)
	}
}

// TestApprovalStoreAPIHasNoTaskMutation 机械固化 AC4:适配器不存在任何
// 更新/删除 API,更没有触碰任务状态的入口(状态派生归 State Engine)。
func TestApprovalStoreAPIHasNoTaskMutation(t *testing.T) {
	typ := reflect.TypeOf(&Store{})
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		for _, banned := range []string{"Update", "Delete", "Rewrite", "Truncate", "Drop", "SetTask"} {
			if strings.HasPrefix(name, banned) {
				t.Errorf("决定证据面出现改写/删除/状态入口 API:%s(AC1/AC4)", name)
			}
		}
	}
}

// ---- 数据库层(不可达即 SKIP) ----

func TestApprovalRecordAndReadBackDB(t *testing.T) {
	st, _ := testApprovalStore(t)
	ctx := context.Background()

	a := mustApproval(t, ApprovalSpec{
		Project:     testProject,
		TaskID:      testTask,
		ChangeSetID: testCS,
		Decision:    DecisionApprove,
		Reason:      "unattributed=0,AC coverage complete",
		DecidedBy:   "owner-7",
		DecidedAt:   testAt,
		Waiver:      mustWaiver(t, validWaiverSpec()),
	})
	id, err := st.RecordApproval(ctx, a)
	if err != nil {
		t.Fatalf("RecordApproval: %v", err)
	}
	if id <= 0 {
		t.Fatalf("行 id 应为正,得到 %d", id)
	}

	got, err := st.Approval(ctx, id)
	if err != nil {
		t.Fatalf("Approval: %v", err)
	}
	if got.ID != id || !sameEvidence(got.Approval, a) {
		t.Fatalf("读回不一致: id=%d", got.ID)
	}
	if w := got.Approval.Waiver(); w == nil || w.ReasonCode() != "holdout_exposure_accepted" || len(w.Gaps()) != 2 {
		t.Fatalf("豁免 provenance 读回不一致: %+v", w)
	}

	latest, err := st.LatestApprovalForChangeSet(ctx, testProject, testTask, testCS)
	if err != nil || latest.ID != id {
		t.Fatalf("LatestApprovalForChangeSet: %v id=%d", err, latest.ID)
	}
	forTask, err := st.ApprovalsForTask(ctx, testProject, testTask)
	if err != nil || len(forTask) != 1 || forTask[0].ID != id {
		t.Fatalf("ApprovalsForTask: %v %d", err, len(forTask))
	}
	waived, err := st.WaivedApprovalsForProject(ctx, testProject)
	if err != nil || len(waived) != 1 || waived[0].ID != id || !waived[0].Approval.WaivedIndependence() {
		t.Fatalf("WaivedApprovalsForProject: %v %+v", err, waived)
	}

	if _, err := st.Approval(ctx, 987654); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("未知 id 应包 pgx.ErrNoRows,得到 %v", err)
	}
	// 无决定的候选:证据缺席(pgx.ErrNoRows),不是错误状态。
	if _, err := st.LatestApprovalForChangeSet(ctx, testProject, "T-other", "cs-none"); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("未决定候选应包 pgx.ErrNoRows,得到 %v", err)
	}
	if _, err := st.LatestApprovalForChangeSet(ctx, testProject, testTask, ""); err == nil {
		t.Fatal("空 changeset_id 应报错")
	}
}

// TestApprovalAppendOnlyHistoryDB(AC1):同一候选后来的新决定追加新行,
// 历史行永不改写、仍可回查;最新行决定当前投影,但更早行不被抹掉。
func TestApprovalAppendOnlyHistoryDB(t *testing.T) {
	st, pool := testApprovalStore(t)
	ctx := context.Background()

	approve := mustApproval(t, ApprovalSpec{
		Project: testProject, TaskID: testTask, ChangeSetID: testCS,
		Decision: DecisionApprove, Reason: "first pass",
		DecidedBy: "owner-7", DecidedAt: testAt,
	})
	approveID, err := st.RecordApproval(ctx, approve)
	if err != nil {
		t.Fatalf("RecordApproval(approve): %v", err)
	}
	reject := mustApproval(t, ApprovalSpec{
		Project: testProject, TaskID: testTask, ChangeSetID: testCS,
		Decision: DecisionReject, Reason: "scope violation found",
		DecidedBy: "owner-7", DecidedAt: testAt.Add(time.Hour),
	})
	rejectID, err := st.RecordApproval(ctx, reject)
	if err != nil {
		t.Fatalf("RecordApproval(reject): %v", err)
	}
	if approveID == rejectID {
		t.Fatal("两次决定不应共享行 id")
	}

	list, err := st.ApprovalsForTask(ctx, testProject, testTask)
	if err != nil || len(list) != 2 {
		t.Fatalf("ApprovalsForTask 应有 2 行历史: %v %d", err, len(list))
	}
	if list[0].ID != rejectID || list[0].Approval.Decision() != DecisionReject {
		t.Fatalf("最新行应为 reject: %+v", list[0])
	}
	// 最早的 approve 行仍可按 id 回查:历史证据不被后来的决定洗掉。
	first, err := st.Approval(ctx, approveID)
	if err != nil || first.Approval.Decision() != DecisionApprove || first.Approval.Reason() != "first pass" {
		t.Fatalf("历史 approve 行应原样可回查: %v %+v", err, first)
	}
	// 无豁免的 reject 不进豁免投影。
	waived, err := st.WaivedApprovalsForProject(ctx, testProject)
	if err != nil || len(waived) != 0 {
		t.Fatalf("无豁免行时豁免投影应为空: %v %d", err, len(waived))
	}

	mustModifyRejected(t, pool, "UPDATE human_approval SET decision='approve' WHERE id=$1", rejectID)
	mustModifyRejected(t, pool, "DELETE FROM human_approval WHERE id=$1", approveID)
}

func TestApprovalInvalidInputRejectedDB(t *testing.T) {
	st, pool := testApprovalStore(t)
	ctx := context.Background()
	baseline := countApprovals(t, pool, "SELECT count(*) FROM human_approval")

	// 零值证据。
	if _, err := st.RecordApproval(ctx, HumanApprovalEvidence{}); !errors.Is(err, ErrInvalidApproval) {
		t.Fatalf("零值证据应报 ErrInvalidApproval,得到 %v", err)
	}
	// reject 携带豁免是构造期矛盾:领域构造器拒绝,不可能落库(字段私有,
	// 无法在构造后注入非法组合,写路径重校验因此只对零值/外部破坏生效)。

	// 非候选查询缺坐标。
	if _, err := st.LatestApprovalForChangeSet(ctx, testProject, testTask, ""); err == nil {
		t.Fatal("空 changeset 查询应报错")
	}

	if n := countApprovals(t, pool, "SELECT count(*) FROM human_approval"); n != baseline {
		t.Fatalf("非法输入不应落行:baseline %d,实际 %d", baseline, n)
	}
}

// sameEvidence 比较两条决定证据的全部领域访问器(含豁免)。
func sameEvidence(a, b HumanApprovalEvidence) bool {
	if a.Project() != b.Project() || a.TaskID() != b.TaskID() ||
		a.Decision() != b.Decision() || a.Reason() != b.Reason() ||
		a.DecidedBy() != b.DecidedBy() || !a.DecidedAt().Equal(b.DecidedAt()) ||
		a.WaivedIndependence() != b.WaivedIndependence() {
		return false
	}
	csA, okA := a.ChangeSetID()
	csB, okB := b.ChangeSetID()
	if okA != okB || csA != csB {
		return false
	}
	wa, wb := a.Waiver(), b.Waiver()
	if (wa == nil) != (wb == nil) {
		return false
	}
	if wa == nil {
		return true
	}
	return wa.Category() == wb.Category() && wa.ReasonCode() == wb.ReasonCode() &&
		wa.AcceptedBy() == wb.AcceptedBy() && wa.AcceptedAt().Equal(wb.AcceptedAt()) &&
		reflect.DeepEqual(wa.Gaps(), wb.Gaps()) &&
		reflect.DeepEqual(wa.CompromisedAcceptances(), wb.CompromisedAcceptances()) &&
		reflect.DeepEqual(wa.CompromisedBindings(), wb.CompromisedBindings()) &&
		reflect.DeepEqual(wa.ExposureRefs(), wb.ExposureRefs())
}
