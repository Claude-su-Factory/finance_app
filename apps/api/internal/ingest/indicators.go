package ingest

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/models"
)

func UpsertIndicators(ctx context.Context, pool *pgxpool.Pool, items []models.Indicator) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)
	var n int64
	for _, it := range items {
		if _, err := tx.Exec(ctx, `
			insert into public.economic_indicators (code, observed_at, name, value, unit)
			values ($1, $2, $3, $4, $5)
			on conflict (code, observed_at) do update set
				name = excluded.name, value = excluded.value, unit = excluded.unit
		`, it.Code, it.ObservedAt, it.Name, it.Value, it.Unit); err != nil {
			return n, err
		}
		n++
	}
	return n, tx.Commit(ctx)
}
