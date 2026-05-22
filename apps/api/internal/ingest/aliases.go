package ingest

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SeedAlias adds (or upserts source=seed) an alias.
func SeedAlias(ctx context.Context, pool *pgxpool.Pool, alias, instrumentID string) error {
	_, err := pool.Exec(ctx, `
		insert into public.instrument_aliases (alias, instrument_id, source)
		values ($1, $2, 'seed')
		on conflict (alias) do update set instrument_id = excluded.instrument_id
	`, strings.ToLower(alias), instrumentID)
	return err
}

// LearnAlias records a user search→selection mapping (non-overwriting).
func LearnAlias(ctx context.Context, pool *pgxpool.Pool, alias, instrumentID string) error {
	_, err := pool.Exec(ctx, `
		insert into public.instrument_aliases (alias, instrument_id, source)
		values ($1, $2, 'learned')
		on conflict (alias) do nothing
	`, strings.ToLower(alias), instrumentID)
	return err
}
