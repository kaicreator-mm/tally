//go:build integration

package store

// 集成测试的 schema 隔离基建(落实 docs/05-test-standards.md「每个测试自己建 schema,自己清理」)。
//
// 只参与测试:文件带 integration tag,不进生产构建,不对外开放 API(所以不放 doc.go)。
// 每个测试拿到一个独立 schema,迁移(含 INV-5 语句级触发器)已在其中跑过,互不共享表数据;
// 全局/全表断言在并行全跑下因此恢复确定性(治 tasks/BACKLOG.md 记的共享库并行污染)。

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
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

const testDefaultDSN = "postgres://postgres:postgres@localhost:5432/tally?sslmode=disable"

// TestDB 返回一个连到本测试独立 schema 的 *Store,迁移已应用;t.Cleanup 里 DROP SCHEMA CASCADE。
//
// 隔离方式:用 pgx 的 AfterConnect 钩子给每条新连接 SET search_path = <唯一 schema> ——
// goose 迁移与业务连接走同一 search_path,表、触发器函数、goose_db_version 全落在该 schema 内,
// 自包含,删 schema 即清干净,跨测试/跨包并行互不可见。
func TestDB(t *testing.T) *Store {
	t.Helper()
	ctx := context.Background()

	base := os.Getenv("DATABASE_URL")
	if base == "" {
		base = testDefaultDSN
	}
	schema := "tally_it_" + strings.NewReplacer("-", "", ".", "", ":", "", "+", "").Replace(
		fmt.Sprintf("%d", time.Now().UnixNano()))

	setSearchPath := func(ctx context.Context, c *pgconn.PgConn) error {
		_, err := c.Exec(ctx, "SET search_path = "+pgx.Identifier{schema}.Sanitize()).ReadAll()
		return err
	}

	// 引导连接:建 schema + 在其中跑全部迁移(连接经 AfterConnect 先 SET search_path 指向它)。
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

	// 业务连接池:同样钉在该 schema。
	pool, err := pgxpool.ParseConfig(base)
	if err != nil {
		t.Fatalf("解析 DATABASE_URL(连接池): %v", err)
	}
	pool.ConnConfig.Config.AfterConnect = setSearchPath
	st, err := openWithConfig(ctx, pool)
	if err != nil {
		t.Fatalf("打开测试 store: %v", err)
	}

	t.Cleanup(func() {
		st.Close()
		dropSchema(base, schema)
	})
	return st
}

// openWithConfig 用解析好的连接池配置建 Store(测试基建专用,Open 的底层)。
func openWithConfig(ctx context.Context, cfg *pgxpool.Config) (*Store, error) {
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("建立连接池: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping 数据库: %w", err)
	}
	return &Store{pool: pool, queries: New(pool)}, nil
}

// dropSchema 用一个连到 public 的干净连接删测试 schema(自身 schema 的连接已随 st.Close 关掉)。
func dropSchema(base, schema string) {
	u, err := url.Parse(base)
	if err != nil {
		return
	}
	q := u.Query()
	q.Del("search_path") // 防 base 自带 search_path,确保连到默认(public)
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
