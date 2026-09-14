//go:build integration

// P3-04 migration probe — the executable migration verifier for the PRD §24.2
// Database Migration closed-loop E2E (test/e2e/v14_migration_*).
//
// The P3-02 fixture contract (test/fixtures/v14/migration/**,
// catalog/v14/policies/database_migration.yaml) pins three required evidence
// keys — fresh_database, n_minus_1_upgrade, data_integrity — but its unit test
// explicitly defers real SQL execution to "the harness runtime task". This
// probe is that execution face: a minimal, deterministic PostgreSQL runner
// that the frozen VerificationContract bindings drive through the real
// fixture migration chain against disposable per-test databases.
//
// Subcommands (one verdict each; exit 0 = pass, exit 1 = deterministic fail):
//
//	fresh     -dsn D -dir DIR              recreate schema, apply 0001,0002,0003,
//	                                       assert orders.total_cents exists NOT NULL
//	n1        -dsn D -dir DIR -seed FILE   recreate schema, apply 0001,0002, apply the
//	                                       N-1 data seed, then upgrade with 0003
//	integrity -dsn D -dir DIR -seed FILE   n1 upgrade + protected row expectations:
//	                                       seeded ids/linkage/timestamps preserved and
//	                                       total_cents backfilled to 0 on every row
//
// Determinism discipline:
//   - every run recreates the public schema first, so repeated verdicts (the
//     repair loop re-verifies the same frozen protocol) are independent;
//   - SQL failures are printed as "MIGRATION-FAIL <sub>: <pg error>" on stderr
//     and exit 1 — the N-1 defect in the fixture chain fails PostgreSQL with
//     the exact message 'column "total_cents" of relation "orders" contains
//     null values' every time;
//   - structured argv only (frozen CommandSpec argv, no shell), AGENTS.md §11;
//   - expected rows mirror seed/n_minus_1.sql verbatim (ids, emails, linkage,
//     timestamps) so the data_integrity verdict cannot silently drift.
//
// Build tag integration keeps the probe out of non-integration package
// enumerations, matching the E2E suite that builds it.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// migrationChain is the fixed fixture revision order (P3-02 chain contract:
// 0001 → 0002 → 0003; 0003 is the candidate under verification, present in
// the workspace as either the defect or the repair content).
var migrationChain = []string{
	"0001_init.sql",
	"0002_add_orders.sql",
	"0003_orders_total_not_null.sql",
}

// expected rows of seed/n_minus_1.sql, mirrored verbatim (fixed ids and
// timestamps keep every run deterministic).
type seedAccount struct {
	id    int64
	email string
}

type seedOrder struct {
	id        int64
	accountID int64
	createdAt time.Time
}

var expectedAccounts = []seedAccount{
	{1, "seed-account-1@example.test"},
	{2, "seed-account-2@example.test"},
}

var expectedOrders = []seedOrder{
	{100, 1, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
	{101, 2, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)},
	{102, 1, time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)},
}

func main() {
	if len(os.Args) < 2 {
		fail("usage: v14migprobe fresh|n1|integrity -dsn D [-dir DIR] [-seed FILE]")
	}
	sub := os.Args[1]
	fs := flag.NewFlagSet(sub, flag.ContinueOnError)
	dsn := fs.String("dsn", "", "PostgreSQL DSN of the disposable per-binding database")
	dir := fs.String("dir", "migrations", "migrations directory (relative to the command cwd)")
	seed := fs.String("seed", "", "absolute path of the N-1 data seed SQL (n1/integrity)")
	timeout := fs.Duration("timeout", 120*time.Second, "wall clock budget for this probe run")
	if err := fs.Parse(os.Args[2:]); err != nil {
		fail("%s: flags: %v", sub, err)
	}
	if *dsn == "" {
		fail("%s: -dsn is required", sub)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	cfg, err := pgx.ParseConfig(*dsn)
	if err != nil {
		fail("%s: parse dsn: %v", sub, err)
	}
	// Simple protocol so one Exec carries the multi-statement fixture scripts
	// exactly as authored (CREATE/ALTER/INSERT with default Nextval bits).
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	cfg.ConnectTimeout = 15 * time.Second
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		fail("%s: connect: %v", sub, err)
	}
	defer conn.Close(context.WithoutCancel(ctx))

	switch sub {
	case "fresh", "n1", "integrity":
	default:
		fail("unknown subcommand %q", sub)
	}

	if err := resetSchema(ctx, conn); err != nil {
		fail("%s: reset schema: %v", sub, pgErr(err))
	}
	for _, rev := range migrationChain[:2] { // 0001, 0002
		if err := applyFile(ctx, conn, *dir, rev); err != nil {
			fail("%s: apply %s: %s", sub, rev, pgErr(err))
		}
	}
	if sub == "n1" || sub == "integrity" {
		if *seed == "" {
			fail("%s: -seed is required (N-1 upgrade needs real rows)", sub)
		}
		if err := applySeed(ctx, conn, *seed); err != nil {
			fail("%s: apply n-1 seed: %s", sub, pgErr(err))
		}
	}
	// Revision 0003 is the candidate under verification: the defect form
	// deterministically fails the N-1 path right here.
	if err := applyFile(ctx, conn, *dir, migrationChain[2]); err != nil {
		fail("%s: apply %s: %s", sub, migrationChain[2], pgErr(err))
	}

	switch sub {
	case "fresh":
		if err := assertFresh(ctx, conn); err != nil {
			fail("fresh: %v", err)
		}
	case "n1":
		// Success of the upgrade itself is the verdict.
	case "integrity":
		if err := assertIntegrity(ctx, conn); err != nil {
			fail("integrity: %v", err)
		}
	}
	fmt.Printf("MIGRATION-OK %s\n", sub)
}

// resetSchema recreates public so each verdict is independent of prior runs
// (the repair loop re-runs the same frozen protocol against the same DSN).
func resetSchema(ctx context.Context, conn *pgx.Conn) error {
	_, err := conn.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public;")
	return err
}

// applyFile applies one fixture revision script verbatim (simple protocol,
// multi-statement safe; the fixture files are single-statement each).
func applyFile(ctx context.Context, conn *pgx.Conn, dir, name string) error {
	script, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if _, err := conn.Exec(ctx, string(script)); err != nil {
		return err
	}
	return nil
}

// applySeed applies the N-1 data seed so 0003 upgrades a database that
// carries rows (the seed is data setup, never part of the migration chain).
func applySeed(ctx context.Context, conn *pgx.Conn, path string) error {
	script, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if _, err := conn.Exec(ctx, string(script)); err != nil {
		return err
	}
	return nil
}

// assertFresh pins the fresh-DB outcome: the candidate revision must leave
// orders.total_cents present and NOT NULL (the PRD §24.2 false-confidence
// path: this passes on the defect candidate too — an empty table never
// violates the NOT NULL backfill).
func assertFresh(ctx context.Context, conn *pgx.Conn) error {
	var nullable string
	err := conn.QueryRow(ctx,
		"SELECT is_nullable FROM information_schema.columns WHERE table_schema='public' AND table_name='orders' AND column_name='total_cents'",
	).Scan(&nullable)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("orders.total_cents missing after fresh migration")
	}
	if err != nil {
		return err
	}
	if nullable != "NO" {
		return fmt.Errorf("orders.total_cents is_nullable=%q, want NO", nullable)
	}
	return nil
}

// assertIntegrity pins the protected row expectations (PRD §24.2 data
// integrity): every seeded account and order survives the upgrade with its
// identity, linkage and timestamp intact, and total_cents is backfilled to 0
// on every pre-existing row.
func assertIntegrity(ctx context.Context, conn *pgx.Conn) error {
	rows, err := conn.Query(ctx, "SELECT id, email FROM accounts ORDER BY id")
	if err != nil {
		return err
	}
	var gotAccounts []seedAccount
	for rows.Next() {
		var a seedAccount
		if err := rows.Scan(&a.id, &a.email); err != nil {
			rows.Close()
			return err
		}
		gotAccounts = append(gotAccounts, a)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if err := expectRows("accounts", len(expectedAccounts), len(gotAccounts)); err != nil {
		return err
	}
	for i, want := range expectedAccounts {
		if gotAccounts[i] != want {
			return fmt.Errorf("accounts[%d] = %+v, want %+v (seed data lost or drifted)", i, gotAccounts[i], want)
		}
	}

	rows, err = conn.Query(ctx, "SELECT id, account_id, created_at, total_cents FROM orders ORDER BY id")
	if err != nil {
		return err
	}
	defer rows.Close()
	var gotOrders []seedOrder
	var totals []int64
	for rows.Next() {
		var o seedOrder
		var total int64
		if err := rows.Scan(&o.id, &o.accountID, &o.createdAt, &total); err != nil {
			return err
		}
		gotOrders = append(gotOrders, o)
		totals = append(totals, total)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := expectRows("orders", len(expectedOrders), len(gotOrders)); err != nil {
		return err
	}
	for i, want := range expectedOrders {
		got := gotOrders[i]
		if got.id != want.id || got.accountID != want.accountID || !got.createdAt.Equal(want.createdAt) {
			return fmt.Errorf("orders[%d] = {id:%d account:%d at:%s}, want {id:%d account:%d at:%s} (seed data lost or drifted)",
				i, gotOrders[i].id, gotOrders[i].accountID, gotOrders[i].createdAt.Format(time.RFC3339),
				want.id, want.accountID, want.createdAt.Format(time.RFC3339))
		}
		if totals[i] != 0 {
			return fmt.Errorf("orders id %d total_cents=%d, want backfilled 0", gotOrders[i].id, totals[i])
		}
	}
	return nil
}

func expectRows(table string, want, got int) error {
	if got != want {
		return fmt.Errorf("%s row count = %d, want %d (seed data lost or drifted)", table, got, want)
	}
	return nil
}

// pgErr renders a PostgreSQL failure compactly and deterministically: the
// N-1 defect fails with SQLSTATE 23502 and the exact message 'column
// "total_cents" of relation "orders" contains null values'.
func pgErr(err error) string {
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) {
		return fmt.Sprintf("pg %s: %s", pgerr.Code, pgerr.Message)
	}
	return err.Error()
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "MIGRATION-FAIL "+format+"\n", args...)
	os.Exit(1)
}
