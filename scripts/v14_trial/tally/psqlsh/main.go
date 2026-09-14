// psqlsh is a minimal psql-compatible PostgreSQL client for the P4-03 trial
// (Tally arm tooling, scripts/v14_trial/tally/**). The host has no psql
// binary; the ARM-NEUTRAL judge (scripts/v14_trial/bare judge, used by BOTH
// arms) takes a --psql-bin flag, and this program satisfies exactly the flag
// surface the judge exercises — nothing more:
//
//	psqlsh -X --dbname <uri/[db]> -c <sql>
//	psqlsh -X --dbname <uri/[db]> -1 -v ON_ERROR_STOP=1 -q -f <file>
//	psqlsh -X -A -t --dbname <uri/[db]> -c <query>     (scalar, tuples-only)
//
// Semantics mirror psql for this surface: -1 wraps a -f file in a single
// transaction; ON_ERROR_STOP stops at the first error with a nonzero exit;
// -A -t prints unaligned tuples-only rows (fields joined by "|"); exit 0 on
// success, 1 on SQL error, 2 on usage/connection failure.
//
// Connection strings are libpq URIs (postgres://user:pass@host:port/db) and
// are passed straight to pgx. Implemented over pgx/v5 (the repository's
// pinned PostgreSQL driver); no new dependency enters the module.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func main() {
	var (
		dbname    string
		singleTx  bool
		onErrorSt bool
		filePath  string
		sqlCmd    string
		tuples    bool // -A -t: unaligned, tuples-only output
	)
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch a {
		case "-X", "-q", "--quiet":
			// no-op: expanded output off / quiet (the judge's surface)
		case "-A", "--no-align", "-t", "--tuples-only":
			tuples = true
		case "-1", "--single-transaction":
			singleTx = true
		case "-v", "--set", "--variable":
			i++
			if i >= len(args) {
				fatalUsage("-v requires an argument")
			}
			kv := strings.TrimPrefix(args[i], "--")
			if !strings.EqualFold(kv, "ON_ERROR_STOP=1") && !strings.EqualFold(kv, "ON_ERROR_STOP=on") {
				fatalf("psqlsh: unsupported -v setting %q (only ON_ERROR_STOP=1)", args[i])
			}
			onErrorSt = true
		case "--dbname":
			i++
			if i >= len(args) {
				fatalUsage("--dbname requires an argument")
			}
			dbname = args[i]
		case "-c", "--command":
			i++
			if i >= len(args) {
				fatalUsage("-c requires an argument")
			}
			sqlCmd = args[i]
		case "-f", "--file":
			i++
			if i >= len(args) {
				fatalUsage("-f requires an argument")
			}
			filePath = args[i]
		default:
			fatalUsage("unsupported argument %q (psqlsh implements only the trial judge's flag surface)", a)
		}
	}
	if dbname == "" {
		fatalUsage("--dbname is required")
	}
	if (sqlCmd == "") == (filePath == "") {
		fatalUsage("exactly one of -c or -f is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cfg, err := pgx.ParseConfig(dbname)
	if err != nil {
		fatalf("psqlsh: parse connection string: %v", err)
	}
	// The judge's surface is psql's: simple protocol, so row-less commands
	// (CREATE/DROP DATABASE) and multi-statement files behave exactly like
	// they do under psql (no implicit extended-protocol transaction block).
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		fatalf("psqlsh: connect: %v", err)
	}
	defer conn.Close(ctx)

	if sqlCmd != "" {
		if err := runCommand(ctx, conn, sqlCmd, tuples); err != nil {
			fatalf("psqlsh: %v", err)
		}
		return
	}
	b, err := os.ReadFile(filePath)
	if err != nil {
		fatalf("psqlsh: read %s: %v", filePath, err)
	}
	run := func() error {
		_, err := conn.Exec(ctx, string(b))
		return err
	}
	if singleTx {
		tx, terr := conn.Begin(ctx)
		if terr != nil {
			fatalf("psqlsh: begin: %v", terr)
		}
		if _, err := tx.Exec(ctx, string(b)); err != nil && onErrorSt {
			_ = tx.Rollback(ctx)
			fatalf("psqlsh: %v", err)
		} else if err != nil {
			_ = tx.Rollback(ctx)
			fatalf("psqlsh: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			fatalf("psqlsh: commit: %v", err)
		}
		return
	}
	if err := run(); err != nil {
		if onErrorSt {
			fatalf("psqlsh: %v", err)
		}
		fatalf("psqlsh: %v", err)
	}
}

// runCommand executes one -c command via Query (which also accepts row-less
// commands like CREATE/DROP DATABASE). SQL that returns rows is printed in
// the -A -t shape (fields joined by "|", one line per row); row-less commands
// print nothing, like psql -q.
func runCommand(ctx context.Context, conn *pgx.Conn, sql string, tuples bool) error {
	rows, err := conn.Query(ctx, sql)
	if err != nil {
		return err
	}
	defer rows.Close()
	if tuples {
		for rows.Next() {
			vals := rows.RawValues()
			parts := make([]string, len(vals))
			for i, v := range vals {
				parts[i] = string(v)
			}
			fmt.Println(strings.Join(parts, "|"))
		}
		return rows.Err()
	}
	for rows.Next() {
	}
	return rows.Err()
}

func fatalUsage(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "psqlsh: usage error: "+format+"\n", a...)
	os.Exit(2)
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}
