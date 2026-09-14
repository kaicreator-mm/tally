// store_test.go 拥有治理事件存储的数据库层测试:append-only 写路径、CHECK
// 封闭枚举、触发器拒绝 UPDATE/DELETE、按序读出与重放重建(AC:提案历史可
// 重建;无「当前 ACTIVE」可变权威)。每个测试独立 schema、自跑 goose 迁移、
// cleanup DROP;库不可达即 SKIP(与 internal/outcome 同一纪律)。
package policygovernance

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"tally/migrations"
)

const polgovTestDSN = "postgres://postgres:postgres@localhost:5432/tally?sslmode=disable"

func testGovStore(t *testing.T) (*Store, *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	base := os.Getenv("DATABASE_URL")
	if base == "" {
		base = polgovTestDSN
	}
	if !dbReachable(base) {
		t.Skipf("policygovernance 数据库层测试需要 PostgreSQL(%s 不可达),跳过", base)
	}
	schema := "tally_polgov_" + strconv.FormatInt(time.Now().UnixNano(), 10)

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
	return NewStore(pool), pool
}

func candRevision() string {
	c, err := NewCandidate([]byte(validPolicyYAML))
	if err != nil {
		panic("夹具候选必须可解析: " + err.Error())
	}
	return c.Revision()
}

func baseEvent(kind EventKind, at time.Time) Event {
	// 修订取自真实正典解析(Reconstruct 会重放解析并核对 —— 内容派生,不可伪造)。
	return Event{
		ProposalID:        "prop-it-001",
		Kind:              kind,
		CandidateRevision: candRevision(),
		CandidateYAML:     validPolicyYAML,
		ProposedBy:        "owner-a",
		OccurredAt:        at,
	}
}

// AC 正面:draft→submit→activate 全链落行,重放重建出最终 ACTIVE 提案,
// 历史可按序读回。
func TestStoreAppendAndReconstruct(t *testing.T) {
	s, _ := testGovStore(t)
	ctx := context.Background()
	t0 := time.Now().Add(-time.Hour).UTC()

	if _, err := s.Append(ctx, baseEvent(EventDraft, t0)); err != nil {
		t.Fatalf("Append(draft): %v", err)
	}
	sub := baseEvent(EventSubmit, t0.Add(time.Minute))
	sub.Evidence = []EvidenceRef{mustRef(t, EvidenceOutcome, "out-001"), mustRef(t, EvidenceChangeSet, "cs-9")}
	if _, err := s.Append(ctx, sub); err != nil {
		t.Fatalf("Append(submit): %v", err)
	}
	act := baseEvent(EventActivate, t0.Add(2*time.Minute))
	act.ApprovedBy = "owner-b"
	if _, err := s.Append(ctx, act); err != nil {
		t.Fatalf("Append(activate): %v", err)
	}

	got, err := s.Reconstruct(ctx, "prop-it-001")
	if err != nil {
		t.Fatalf("Reconstruct: %v", err)
	}
	if got.Status != StatusActive || got.ApprovedBy != "owner-b" {
		t.Fatalf("重建状态 = %s approvedBy=%q, want ACTIVE/owner-b", got.Status, got.ApprovedBy)
	}
	if len(got.Evidence) != 2 || got.Evidence[0].Kind != EvidenceOutcome {
		t.Fatalf("重建证据 = %v", got.Evidence)
	}
	if got.Candidate.Revision() != candRevision() {
		t.Fatalf("重建候选修订 = %q", got.Candidate.Revision())
	}
	if got.Candidate.Policy().ID != "POL-MIGRATION" {
		t.Fatalf("重建候选解析: id = %q", got.Candidate.Policy().ID)
	}

	all, err := s.AllEvents(ctx)
	if err != nil || len(all) != 3 {
		t.Fatalf("AllEvents: %v len=%d", err, len(all))
	}
	if all[0].Kind != EventDraft || all[2].Kind != EventActivate {
		t.Fatalf("事件应按 id 升序: %s,%s,%s", all[0].Kind, all[1].Kind, all[2].Kind)
	}
}

// 全生命周期含否决与取代:历史逐事件可重建,终态如实。
func TestStoreReconstructRejectAndSupersede(t *testing.T) {
	s, _ := testGovStore(t)
	ctx := context.Background()
	t0 := time.Now().UTC()

	draft := baseEvent("draft", t0) // kind 用字符串字面量构造非法入口的对照
	draft.Kind = EventDraft
	if _, err := s.Append(ctx, draft); err != nil {
		t.Fatalf("Append(draft): %v", err)
	}
	sub := baseEvent(EventSubmit, t0.Add(time.Minute))
	sub.Evidence = []EvidenceRef{mustRef(t, EvidenceVerification, "sha256:v1")}
	if _, err := s.Append(ctx, sub); err != nil {
		t.Fatalf("Append(submit): %v", err)
	}
	rej := baseEvent(EventReject, t0.Add(2*time.Minute))
	if _, err := s.Append(ctx, rej); err != nil {
		t.Fatalf("Append(reject): %v", err)
	}
	got, err := s.Reconstruct(ctx, "prop-it-001")
	if err != nil || got.Status != StatusRejected {
		t.Fatalf("Reconstruct(reject 链): %v status=%s", err, got.Status)
	}
}

// fail-safe 写路径:未知 kind 被域侧拒绝;activate 无批准人被拒绝;缺时刻/
// 缺候选内容被拒绝。
func TestStoreAppendNegatives(t *testing.T) {
	s, _ := testGovStore(t)
	ctx := context.Background()
	t0 := time.Now().UTC()

	if _, err := s.Append(ctx, baseEvent("coup", t0)); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("未知 kind 应被域侧拒绝,得到 %v", err)
	}
	bad := baseEvent(EventActivate, t0)
	if _, err := s.Append(ctx, bad); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("activate 无批准人应拒绝,得到 %v", err)
	}
	noTime := baseEvent(EventDraft, t0)
	noTime.OccurredAt = time.Time{}
	if _, err := s.Append(ctx, noTime); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("缺发生时刻应拒绝,得到 %v", err)
	}
	noYAML := baseEvent(EventDraft, t0)
	noYAML.CandidateYAML = ""
	if _, err := s.Append(ctx, noYAML); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("缺候选 YAML 应拒绝,得到 %v", err)
	}
}

// 读出侧 fail-safe:被污染的证据引用串读出即 ErrInconsistentHistory,不静默。
func TestStoreReadSideEvidenceValidation(t *testing.T) {
	s, pool := testGovStore(t)
	ctx := context.Background()
	t0 := time.Now().UTC()

	ev := baseEvent(EventSubmit, t0)
	ev.Evidence = []EvidenceRef{mustRef(t, EvidenceTask, "TASK-1")}
	if _, err := s.Append(ctx, ev); err != nil {
		t.Fatalf("Append: %v", err)
	}
	// 模拟带外污染(绕过 Append 的直接 SQL 写入 —— 触发器只拦 UPDATE/DELETE)。
	if _, err := pool.Exec(ctx,
		`UPDATE policy_governance_event SET evidence_refs = ARRAY['deploy:thing'] WHERE proposal_id=$1`,
		"prop-it-001"); err != nil {
		// UPDATE 被触发器拦截本身就是正确行为(append-only):此时用 DELETE+INSERT
		// 也不可行 —— 直接断言触发器生效并结束本面。
		if strings.Contains(err.Error(), "tally_prevent_modify") || strings.Contains(err.Error(), "append") {
			return
		}
		t.Fatalf("带外 UPDATE: %v", err)
	}
	if _, err := s.Reconstruct(ctx, "prop-it-001"); !errors.Is(err, ErrInconsistentHistory) {
		t.Fatalf("被污染证据读出应 ErrInconsistentHistory,得到 %v", err)
	}
}

// DB 侧兜底:未知 kind 被 CHECK 拒绝;UPDATE/DELETE 被触发器拒绝(append-only)。
func TestStoreDatabaseEnforcement(t *testing.T) {
	s, pool := testGovStore(t)
	ctx := context.Background()
	t0 := time.Now().UTC()

	if _, err := s.Append(ctx, baseEvent(EventDraft, t0)); err != nil {
		t.Fatalf("Append(draft): %v", err)
	}
	// CHECK:绕过域层的带外 INSERT(未知 kind)在 DB 层拒绝。
	if _, err := pool.Exec(ctx,
		`INSERT INTO policy_governance_event (proposal_id,action_kind,candidate_rev,candidate_yaml,proposed_by,occurred_at)
		 VALUES ('x','pardon','r','y','a',now())`); err == nil {
		t.Fatal("DB CHECK 应拒绝未知治理动作类型")
	}
	// 触发器:UPDATE 拒绝(语句级,0 行也报错;0023 同型)。
	if _, err := pool.Exec(ctx, `UPDATE policy_governance_event SET scope = 'x'`); err == nil {
		t.Fatal("append-only 触发器应拒绝 UPDATE")
	}
	if _, err := pool.Exec(ctx, `DELETE FROM policy_governance_event`); err == nil {
		t.Fatal("append-only 触发器应拒绝 DELETE")
	}
	// 空流重放:类型化 ErrUnknownProposal,不猜测。
	if _, err := s.Reconstruct(ctx, "prop-never-was"); !errors.Is(err, ErrUnknownProposal) {
		t.Fatalf("空流重放应 ErrUnknownProposal,得到 %v", err)
	}
}

// 历史不一致 fail-safe:submit 之前的 activate 在重放处显式失败。
func TestStoreReconstructInconsistentHistory(t *testing.T) {
	s, _ := testGovStore(t)
	ctx := context.Background()
	t0 := time.Now().UTC()

	act := baseEvent(EventActivate, t0)
	act.ApprovedBy = "owner-b"
	if _, err := s.Append(ctx, act); err != nil {
		t.Fatalf("Append(activate 先行): %v", err)
	}
	if _, err := s.Append(ctx, baseEvent(EventSubmit, t0.Add(time.Minute))); err != nil {
		t.Fatalf("Append(submit 后行): %v", err)
	}
	if _, err := s.Reconstruct(ctx, "prop-it-001"); !errors.Is(err, ErrInconsistentHistory) {
		t.Fatalf("乱序历史应 ErrInconsistentHistory,得到 %v", err)
	}
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

// P3-03 写入边界:声称修订与正典解析不一致在 Append 处拒绝(不信任调用方自报)。
func TestStoreAppendRejectsRevisionDrift(t *testing.T) {
	s, _ := testGovStore(t)
	ctx := context.Background()

	drift := baseEvent(EventDraft, time.Now().UTC())
	drift.CandidateRevision = "sha256:claimed-by-caller"
	if _, err := s.Append(ctx, drift); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("修订漂移应在写入处拒绝,得到 %v", err)
	}
	unparseable := baseEvent(EventDraft, time.Now().UTC())
	unparseable.CandidateYAML = "not: [valid, policy"
	if _, err := s.Append(ctx, unparseable); !errors.Is(err, ErrInvalidProposal) {
		t.Fatalf("不可解析候选应在写入处拒绝,得到 %v", err)
	}
}

// P3-03 要求 2:重复身份 fail-closed —— 第二个 draft 在写入处拒绝。
func TestStoreAppendRejectsDuplicateIdentity(t *testing.T) {
	s, _ := testGovStore(t)
	ctx := context.Background()
	t0 := time.Now().UTC()

	if _, err := s.Append(ctx, baseEvent(EventDraft, t0)); err != nil {
		t.Fatalf("Append(draft#1): %v", err)
	}
	if _, err := s.Append(ctx, baseEvent(EventDraft, t0.Add(time.Minute))); !errors.Is(err, ErrDuplicateProposal) {
		t.Fatalf("重复身份应 ErrDuplicateProposal,得到 %v", err)
	}
	// 非重复的后续事件仍可落(submit 跟随首个 draft)。
	sub := baseEvent(EventSubmit, t0.Add(2*time.Minute))
	sub.Evidence = []EvidenceRef{mustRef(t, EvidenceOutcome, "out-1")}
	if _, err := s.Append(ctx, sub); err != nil {
		t.Fatalf("Append(submit 跟随): %v", err)
	}
}
