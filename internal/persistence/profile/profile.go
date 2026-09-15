// Package profile owns v1.5 persistence adapter selection. Core callers depend
// on the neutral Repository/Stores contracts; backend selection stays at the
// composition edge.
package profile

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"tally/internal/controlplane"
	"tally/internal/persistence"
	"tally/internal/persistence/postgres"
	"tally/internal/persistence/sqlite"
)

type Name string

const (
	SQLite   Name = "sqlite"
	Postgres Name = "postgres"

	DefaultSQLitePath = "./tally-state/v1.5/tally.db"
)

type Config struct {
	Name        Name
	SQLitePath  string
	PostgresDSN string
}

// Adapter is the complete application-facing persistence composition used by
// v1.5. It intentionally exposes no database/sql, pgx, SQL dialect or locking
// primitive to Core.
type Adapter interface {
	controlplane.Repository
	persistence.TransactionBoundary
	Stores() persistence.Stores
	Close() error
}

func (c Config) normalized() Config {
	if strings.TrimSpace(string(c.Name)) == "" {
		c.Name = SQLite
	}
	if strings.TrimSpace(c.SQLitePath) == "" {
		c.SQLitePath = DefaultSQLitePath
	}
	return c
}

// Open selects SQLite by default. PostgreSQL is opened only when the caller
// explicitly selects the postgres profile and supplies its DSN.
func Open(ctx context.Context, cfg Config) (Adapter, Name, error) {
	cfg = cfg.normalized()
	switch cfg.Name {
	case SQLite:
		store, err := sqlite.Open(ctx, cfg.SQLitePath)
		if err != nil {
			return nil, "", err
		}
		return store, SQLite, nil
	case Postgres:
		if strings.TrimSpace(cfg.PostgresDSN) == "" {
			return nil, "", errors.New("persistence profile: postgres requires an explicit DSN")
		}
		store, err := postgres.Open(ctx, cfg.PostgresDSN)
		if err != nil {
			return nil, "", err
		}
		return store, Postgres, nil
	default:
		return nil, "", fmt.Errorf("persistence profile: unsupported profile %q (want sqlite|postgres)", cfg.Name)
	}
}
