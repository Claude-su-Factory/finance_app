//go:build integration
// +build integration

package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupPG(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	pg, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)

	// minimal schema (real migrations are tested elsewhere)
	_, err = pool.Exec(ctx, `
		create extension if not exists "pgcrypto";
		create table instruments (
			id uuid primary key default gen_random_uuid(),
			symbol text, exchange text, isin text, name text,
			asset_class text, currency text, is_active boolean default true,
			created_at timestamptz default now(), updated_at timestamptz default now(),
			unique (symbol, exchange)
		);
		create table prices (
			instrument_id uuid, date date,
			open numeric, high numeric, low numeric, close numeric, volume bigint,
			primary key (instrument_id, date)
		);
		create table quotes (
			instrument_id uuid primary key,
			price numeric, change_abs numeric default 0, change_pct numeric default 0,
			updated_at timestamptz default now()
		);
		create table economic_indicators (
			code text, observed_at timestamptz, name text, value numeric, unit text,
			primary key (code, observed_at)
		);
		create table fx_rates (
			base text, quote text, observed_at timestamptz, rate numeric,
			primary key (base, quote, observed_at)
		);
		create table instrument_aliases (
			alias text primary key, instrument_id uuid, source text default 'seed',
			created_at timestamptz default now()
		);
	`)
	require.NoError(t, err)
	return pool
}

func TestUpsertInstruments(t *testing.T) {
	pool := setupPG(t)
	defer pool.Close()
	ctx := context.Background()
	isin := "KR7005930003"
	items := []models.Instrument{
		{Symbol: "005930", Exchange: "KRX", ISIN: &isin, Name: "삼성전자", AssetClass: models.AssetKRStock, Currency: "KRW"},
		{Symbol: "AAPL", Exchange: "NASDAQ", Name: "Apple", AssetClass: models.AssetUSStock, Currency: "USD"},
	}
	n, err := UpsertInstruments(ctx, pool, items)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)

	// 재실행 멱등성
	n2, err := UpsertInstruments(ctx, pool, items)
	require.NoError(t, err)
	require.Equal(t, int64(2), n2)

	var count int
	require.NoError(t, pool.QueryRow(ctx, `select count(*) from instruments`).Scan(&count))
	require.Equal(t, 2, count)
}

func TestUpsertPrices_COPY(t *testing.T) {
	pool := setupPG(t)
	defer pool.Close()
	ctx := context.Background()

	var id uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `
		insert into instruments (symbol, exchange, name, asset_class, currency)
		values ('AAPL', 'NASDAQ', 'Apple', 'US_STOCK', 'USD') returning id
	`).Scan(&id))

	bars := []models.PriceBar{
		{InstrumentID: id.String(), Date: time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC), Open: 200, High: 202, Low: 199, Close: 201, Volume: 1e6},
		{InstrumentID: id.String(), Date: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), Open: 201, High: 203, Low: 200, Close: 202, Volume: 1.1e6},
	}
	n, err := UpsertPrices(ctx, pool, bars)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)

	var count int
	require.NoError(t, pool.QueryRow(ctx, `select count(*) from prices where instrument_id=$1`, id).Scan(&count))
	require.Equal(t, 2, count)
}

func TestUpsertFXRates(t *testing.T) {
	pool := setupPG(t)
	defer pool.Close()
	ctx := context.Background()
	now := time.Now().UTC()
	rates := []models.FXRate{
		{Base: "USD", Quote: "KRW", ObservedAt: now, Rate: 1450.5},
		{Base: "USD", Quote: "EUR", ObservedAt: now, Rate: 0.92},
	}
	n, err := UpsertFXRates(ctx, pool, rates)
	require.NoError(t, err)
	require.Equal(t, int64(2), n)
}
