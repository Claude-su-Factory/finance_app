package ingest

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/models"
)

func UpsertFXRates(ctx context.Context, pool *pgxpool.Pool, rates []models.FXRate) (int64, error) {
	if len(rates) == 0 {
		return 0, nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var n int64
	for _, r := range rates {
		if _, err := tx.Exec(ctx, `
			insert into public.fx_rates (base, quote, observed_at, rate)
			values ($1, $2, $3, $4)
			on conflict (base, quote, observed_at) do update set rate = excluded.rate
		`, r.Base, r.Quote, r.ObservedAt, r.Rate); err != nil {
			return n, err
		}
		n++
	}
	return n, tx.Commit(ctx)
}
