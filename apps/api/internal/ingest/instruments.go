package ingest

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/models"
)

// UpsertInstruments uses pgx.Batch to avoid N round-trips.
func UpsertInstruments(ctx context.Context, pool *pgxpool.Pool, items []models.Instrument) (int64, error) {
	if len(items) == 0 {
		return 0, nil
	}
	b := &pgx.Batch{}
	for _, it := range items {
		b.Queue(`
			insert into public.instruments (symbol, exchange, isin, name, asset_class, currency)
			values ($1, $2, $3, $4, $5, $6)
			on conflict (symbol, exchange) do update set
				isin = excluded.isin, name = excluded.name,
				asset_class = excluded.asset_class, currency = excluded.currency,
				is_active = true, updated_at = now()
		`, it.Symbol, it.Exchange, it.ISIN, it.Name, string(it.AssetClass), it.Currency)
	}
	br := pool.SendBatch(ctx, b)
	defer br.Close()
	var n int64
	for range items {
		if _, err := br.Exec(); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
