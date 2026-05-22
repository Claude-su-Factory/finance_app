package ingest

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/models"
)

const chunkSize = 1000

// UpsertPrices uses COPY into temp table + INSERT ... ON CONFLICT to bulk-upsert.
func UpsertPrices(ctx context.Context, pool *pgxpool.Pool, bars []models.PriceBar) (int64, error) {
	if len(bars) == 0 {
		return 0, nil
	}
	var total int64
	for i := 0; i < len(bars); i += chunkSize {
		end := i + chunkSize
		if end > len(bars) {
			end = len(bars)
		}
		n, err := copyChunk(ctx, pool, bars[i:end])
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

func copyChunk(ctx context.Context, pool *pgxpool.Pool, bars []models.PriceBar) (int64, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		create temp table tmp_prices (
			instrument_id uuid, date date,
			open numeric, high numeric, low numeric, close numeric, volume bigint
		) on commit drop`); err != nil {
		return 0, err
	}

	rows := make([][]any, len(bars))
	for i, b := range bars {
		rows[i] = []any{b.InstrumentID, b.Date, b.Open, b.High, b.Low, b.Close, b.Volume}
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"tmp_prices"},
		[]string{"instrument_id", "date", "open", "high", "low", "close", "volume"},
		pgx.CopyFromRows(rows)); err != nil {
		return 0, err
	}

	tag, err := tx.Exec(ctx, `
		insert into public.prices (instrument_id, date, open, high, low, close, volume)
		select instrument_id, date, open, high, low, close, volume from tmp_prices
		on conflict (instrument_id, date) do update set
			open = excluded.open, high = excluded.high, low = excluded.low,
			close = excluded.close, volume = excluded.volume
	`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), tx.Commit(ctx)
}
