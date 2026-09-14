//go:build integration

package store

// TestDB 的隔离性测试:每个测试拿到独立 schema,迁移(含 INV-5 语句级触发器)已在其中应用,
// 互不共享表数据。t.Cleanup 由 TestDB 自己登记(DROP SCHEMA CASCADE),这里不重复清理。

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// 两个 TestDB 互不可见:各自的 fingerprint_stat 是两张表(不同 schema),
// 一边 upsert 不影响另一边;全局计数也是各自 schema 的「全局」,互不污染。
func TestDBSchemasAreIsolated(t *testing.T) {
	ctx := context.Background()
	a := TestDB(t)
	b := TestDB(t)
	at := time.Now().UTC()

	if _, err := a.Queries().UpsertFingerprintStat(ctx, UpsertFingerprintStatParams{
		FpExact: "fpaaa", FpFuzzy: "fpaaaf", FirstSeen: pgtype.Timestamptz{Time: at, Valid: true}, Project: "demo",
	}); err != nil {
		t.Fatalf("a upsert: %v", err)
	}

	// b 的 fingerprint_stat 应为空:a 的写入不可见。
	var nb int
	if err := b.Pool().QueryRow(ctx, `SELECT count(*) FROM fingerprint_stat`).Scan(&nb); err != nil {
		t.Fatalf("b 数 fingerprint_stat: %v", err)
	}
	if nb != 0 {
		t.Errorf("b 不应看到 a 的行:fingerprint_stat 有 %d 行", nb)
	}
	// a 自己看得到自己那行。
	var na int
	if err := a.Pool().QueryRow(ctx, `SELECT count(*) FROM fingerprint_stat`).Scan(&na); err != nil {
		t.Fatalf("a 数 fingerprint_stat: %v", err)
	}
	if na != 1 {
		t.Errorf("a 应有自己的 1 行:fingerprint_stat 有 %d 行", na)
	}
}

// TestDB 的 schema 里迁移完整:INV-5 语句级触发器就位(0011),
// 对事实面的 0 行 UPDATE 也应被拒(语句级,不命中行也触发)。
func TestDBSchemaHasAppendOnlyTriggers(t *testing.T) {
	st := TestDB(t)
	if _, err := st.Pool().Exec(context.Background(),
		`UPDATE failure_event SET excerpt = 'x' WHERE project_id = 'no-such'`); err == nil {
		t.Error("TestDB schema 的事实面 UPDATE 应被语句级触发器拒绝(INV-5)")
	}
}

// TestDB 的 schema 每次唯一:同一进程内开两个,current_schema 不同。
func TestDBSchemasAreUnique(t *testing.T) {
	ctx := context.Background()
	a := TestDB(t)
	b := TestDB(t)
	var sa, sb string
	if err := a.Pool().QueryRow(ctx, `SELECT current_schema()`).Scan(&sa); err != nil {
		t.Fatalf("a current_schema: %v", err)
	}
	if err := b.Pool().QueryRow(ctx, `SELECT current_schema()`).Scan(&sb); err != nil {
		t.Fatalf("b current_schema: %v", err)
	}
	if sa == sb {
		t.Errorf("两个 TestDB 应在不同 schema,都是 %q", sa)
	}
}
