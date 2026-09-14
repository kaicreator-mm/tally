package main

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// runHealth 检查数据库连通性:连上并 ping 成功打印 ok。
func runHealth(ctx context.Context) error {
	db, err := sql.Open("pgx", databaseURL())
	if err != nil {
		return fmt.Errorf("打开连接: %w", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("数据库不可达: %w", err)
	}
	fmt.Println("ok")
	return nil
}
