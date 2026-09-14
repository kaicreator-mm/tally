// pg.go owns the probe's live-PostgreSQL evidence path (PRD §24.2): one
// disposable database per subcommand, created from the maintenance DSN and
// always dropped (deferred), chain/seed application from the verification
// workspace, and scalar assertion queries. Same execution discipline as the
// arm-neutral judge's psql path (single-transaction file application with
// stop-on-first-error), implemented over pgx so the probe has no external
// client dependency.
package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
)

// dbConn builds a connection string for the named database from the
// maintenance URI (which carries no database name, mirroring the judge's
// pgConfig surface).
func dbConn(dsn, db string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("parse DSN: %w", err)
	}
	if u.Path != "" && u.Path != "/" && strings.Trim(u.Path, "/") != "postgres" {
		return "", fmt.Errorf("maintenance DSN must not name a non-postgres database, got %q", u.Path)
	}
	u.Path = "/" + db
	return u.String(), nil
}

func connect(ctx context.Context, dsn, db string) (*pgx.Conn, error) {
	connStr, err := dbConn(dsn, db)
	if err != nil {
		return nil, err
	}
	cfg, err := pgx.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}
	// psql-equivalent behavior for row-less commands (CREATE DATABASE) and
	// multi-statement chain files: simple protocol, like the judge's psql.
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func execSQL(ctx context.Context, dsn, db, sql string) error {
	conn, err := connect(ctx, dsn, db)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	_, err = conn.Exec(ctx, sql)
	return err
}

func queryScalar(ctx context.Context, dsn, db, query string) (string, error) {
	conn, err := connect(ctx, dsn, db)
	if err != nil {
		return "", err
	}
	defer conn.Close(ctx)
	rows, err := conn.Query(ctx, query)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return "", err
		}
		return "", fmt.Errorf("query returned no rows")
	}
	values := rows.RawValues()
	if len(values) != 1 {
		return "", fmt.Errorf("query returned %d columns, want 1", len(values))
	}
	// The judge compares trimmed textual single values (psql -A -t shape).
	return strings.TrimSpace(string(values[0])), nil
}

// psqlFile applies one SQL file inside a single implicit transaction with
// stop-on-first-error (the psql `-1 -v ON_ERROR_STOP=1` discipline the
// arm-neutral judge uses).
func psqlFile(ctx context.Context, dsn, db, rel string) error {
	b, err := os.ReadFile(filepath.FromSlash(rel))
	if err != nil {
		return fmt.Errorf("read %s: %w", rel, err)
	}
	conn, err := connect(ctx, dsn, db)
	if err != nil {
		return err
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, string(b)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
