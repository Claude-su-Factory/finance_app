package ingest

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/models"
)

func UpsertQuotes(ctx context.Context, pool *pgxpool.Pool, quotes []models.Quote) (int64, error) {
	if len(quotes) == 0 {
		return 0, nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var n int64
	for _, q := range quotes {
		if _, err := tx.Exec(ctx, `
			insert into public.quotes (instrument_id, price, change_abs, change_pct, updated_at)
			values ($1, $2, $3, $4, now())
			on conflict (instrument_id) do update set
				price = excluded.price,
				change_abs = excluded.change_abs,
				change_pct = excluded.change_pct,
				updated_at = now()
		`, q.InstrumentID, q.Price, q.ChangeAbs, q.ChangePct); err != nil {
			return n, err
		}
		n++
	}
	return n, tx.Commit(ctx)
}
