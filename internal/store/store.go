package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store 持有连接池与 sqlc 生成的查询。不放业务逻辑。
// 事实面表只暴露 Insert 和 Select,不暴露 Update/Delete(INV-5)。
type Store struct {
	pool    *pgxpool.Pool
	queries *Queries
}

// Open 建立连接池并确认可连通。
func Open(ctx context.Context, dsn string) (*Store, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("建立连接池: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping 数据库: %w", err)
	}
	return &Store{pool: pool, queries: New(pool)}, nil
}

// Queries 返回 sqlc 生成的查询集。
func (s *Store) Queries() *Queries { return s.queries }

// Pool 返回底层连接池。
func (s *Store) Pool() *pgxpool.Pool { return s.pool }

// InTx 在单个事务里执行 fn;fn 返回 nil 则提交,否则回滚。
// 事务边界在业务包,不在 store(docs/04-code-standards.md)。
func (s *Store) InTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开启事务: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(s.queries.WithTx(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交事务: %w", err)
	}
	return nil
}

// Close 关闭连接池。
func (s *Store) Close() { s.pool.Close() }
