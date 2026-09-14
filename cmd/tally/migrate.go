package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib" // 注册 pgx 的 database/sql 驱动

	"tally/internal/store"
	"tally/migrations"
)

const defaultDSN = "postgres://postgres:postgres@localhost:5432/tally?sslmode=disable"

func databaseURL() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	return defaultDSN
}

// openStore 打开业务库连接(store 层,非 migrate 用的 database/sql),统一「打开数据库」错误上下文。
// Close 留给调用方 defer —— defer 必须在调用方函数作用域内才有效。
func openStore(ctx context.Context) (*store.Store, error) {
	st, err := store.Open(ctx, databaseURL())
	if err != nil {
		return nil, fmt.Errorf("打开数据库: %w", err)
	}
	return st, nil
}

// runMigrate 确保目标库存在,然后跑全部未应用的迁移(goose 库 + 内嵌脚本,不依赖 goose CLI)。
func runMigrate(ctx context.Context) error {
	dsn := databaseURL()

	if err := ensureDatabase(ctx, dsn); err != nil {
		return fmt.Errorf("确保数据库存在: %w", err)
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("打开连接: %w", err)
	}
	defer db.Close()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		return fmt.Errorf("创建 goose provider: %w", err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("执行迁移: %w", err)
	}
	if len(results) == 0 {
		fmt.Println("migrations: 已是最新,无新迁移")
		return nil
	}
	for _, r := range results {
		if r.Error != nil {
			return fmt.Errorf("迁移 %s 失败: %w", r.Source.Path, r.Error)
		}
		fmt.Printf("applied: %s\n", r.Source.Path)
	}
	return nil
}

// ensureDatabase 连到同账号的维护库 postgres,目标库不存在则创建。
func ensureDatabase(ctx context.Context, dsn string) error {
	u, err := url.Parse(dsn)
	if err != nil {
		return fmt.Errorf("解析 DATABASE_URL: %w", err)
	}
	dbName := strings.TrimPrefix(u.Path, "/")
	if dbName == "" {
		return nil // 连接串没指定库名,假定已存在
	}

	maint := *u
	maint.Path = "/postgres"
	maintDB, err := sql.Open("pgx", maint.String())
	if err != nil {
		return fmt.Errorf("打开维护库连接: %w", err)
	}
	defer maintDB.Close()
	if err := maintDB.PingContext(ctx); err != nil {
		return fmt.Errorf("连接维护库 postgres(请确认 Postgres 已启动、账号可用): %w", err)
	}

	var exists bool
	err = maintDB.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, dbName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("查询数据库是否存在: %w", err)
	}
	if exists {
		return nil
	}
	// 库名不能作为参数绑定;dbName 取自 URL path,用引号标识符防注入
	if _, err := maintDB.ExecContext(ctx, `CREATE DATABASE `+quoteIdent(dbName)); err != nil {
		return fmt.Errorf("创建数据库 %s: %w", dbName, err)
	}
	fmt.Printf("created database %s\n", dbName)
	return nil
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
