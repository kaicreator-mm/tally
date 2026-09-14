package web

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"tally/internal/store"
)

// PGSessions 把 store.Queries 适配成 SessionStore。
func PGSessions(q *store.Queries) SessionStore { return pgSessions{q: q} }

type pgSessions struct{ q *store.Queries }

func (p pgSessions) CreateSession(ctx context.Context, token string, ttl time.Duration) error {
	return p.q.InsertSession(ctx, store.InsertSessionParams{
		Token:     token,
		ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(ttl), Valid: true},
	})
}

func (p pgSessions) SessionExists(ctx context.Context, token string) (bool, error) {
	if _, err := p.q.GetSession(ctx, token); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (p pgSessions) DeleteSession(ctx context.Context, token string) error {
	return p.q.DeleteSession(ctx, token)
}
