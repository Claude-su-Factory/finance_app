// Package backfill — KIND/Yahoo에서 가격 히스토리를 채우는 시장별 백필 로직.
// cmd/backfill(CLI)과 cmd/server(부팅 시드)가 공유한다.
package backfill

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/ingest"
	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/quotient/quotient/apps/api/internal/schedule"
	"github.com/quotient/quotient/apps/api/internal/sources/kind"
	"github.com/quotient/quotient/apps/api/internal/sources/yahoo"
)

// nasdaqSeed는 backfill 시드 30종목. Phase 2에서 S&P 100 전체로 확장.
var nasdaqSeed = []struct{ Symbol, Name string }{
	{"AAPL", "Apple Inc."}, {"MSFT", "Microsoft"}, {"GOOGL", "Alphabet Class A"},
	{"AMZN", "Amazon"}, {"NVDA", "NVIDIA"}, {"META", "Meta Platforms"},
	{"TSLA", "Tesla"}, {"AVGO", "Broadcom"}, {"NFLX", "Netflix"},
	{"AMD", "Advanced Micro Devices"}, {"INTC", "Intel"}, {"ORCL", "Oracle"},
	{"CRM", "Salesforce"}, {"ADBE", "Adobe"}, {"QCOM", "Qualcomm"},
	{"TXN", "Texas Instruments"}, {"COST", "Costco"}, {"PEP", "PepsiCo"},
	{"CSCO", "Cisco"}, {"TMUS", "T-Mobile US"}, {"INTU", "Intuit"},
	{"AMAT", "Applied Materials"}, {"BKNG", "Booking Holdings"},
	{"ISRG", "Intuitive Surgical"}, {"REGN", "Regeneron"},
	{"VRTX", "Vertex Pharmaceuticals"}, {"LRCX", "Lam Research"},
	{"PANW", "Palo Alto Networks"}, {"ADP", "Automatic Data Processing"},
	{"GILD", "Gilead Sciences"},
}

// RunKR은 KIND에서 종목 마스터를 받아 KOSPI/KOSDAQ 전 종목의 years년 일봉을 백필한다.
func RunKR(ctx context.Context, pool *pgxpool.Pool, market string, years, limit int) error {
	kc := kind.NewClient("")
	yc := yahoo.NewClient()

	items, err := kc.FetchInstruments(ctx, market)
	if err != nil {
		return fmt.Errorf("kind: %w", err)
	}
	slog.Info("instruments fetched", "market", market, "count", len(items))
	if _, err := ingest.UpsertInstruments(ctx, pool, items); err != nil {
		return fmt.Errorf("upsert instruments: %w", err)
	}

	rows, err := pool.Query(ctx,
		`select id::text, symbol from public.instruments where exchange = 'KRX' and is_active = true order by symbol`)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	type s struct{ id, code string }
	var syms []s
	for rows.Next() {
		var x s
		if err := rows.Scan(&x.id, &x.code); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		syms = append(syms, x)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate: %w", err)
	}
	if limit > 0 && len(syms) > limit {
		syms = syms[:limit]
	}

	end := time.Now().UTC()
	start := end.AddDate(-years, 0, 0)
	for i, sym := range syms {
		ysym := yahoo.SymbolKR(sym.code, market)
		n, err := backfillSymbol(ctx, pool, yc, sym.id, ysym, start, end)
		if err != nil {
			slog.Warn("kr skip", "symbol", ysym, "err", err)
			continue
		}
		slog.Info("backfilled", "i", i+1, "total", len(syms), "symbol", ysym, "rows", n)
		time.Sleep(200 * time.Millisecond) // Yahoo rate limit 보호
	}
	return nil
}

// RunUS는 nasdaqSeed 30종목을 시드한 뒤 years년 일봉을 백필한다.
func RunUS(ctx context.Context, pool *pgxpool.Pool, years, limit int) error {
	yc := yahoo.NewClient()

	insts := make([]models.Instrument, 0, len(nasdaqSeed))
	for _, s := range nasdaqSeed {
		insts = append(insts, models.Instrument{
			Symbol: s.Symbol, Exchange: "NASDAQ", Name: s.Name,
			AssetClass: models.AssetUSStock, Currency: "USD", IsActive: true,
		})
	}
	if _, err := ingest.UpsertInstruments(ctx, pool, insts); err != nil {
		return fmt.Errorf("upsert seed: %w", err)
	}

	rows, err := pool.Query(ctx,
		`select id::text, symbol from public.instruments where exchange = 'NASDAQ' and is_active = true order by symbol`)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	type s struct{ id, code string }
	var syms []s
	for rows.Next() {
		var x s
		if err := rows.Scan(&x.id, &x.code); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		syms = append(syms, x)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate: %w", err)
	}
	if limit > 0 && len(syms) > limit {
		syms = syms[:limit]
	}

	end := time.Now().UTC()
	start := end.AddDate(-years, 0, 0)
	for i, sym := range syms {
		n, err := backfillSymbol(ctx, pool, yc, sym.id, sym.code, start, end)
		if err != nil {
			slog.Warn("us skip", "symbol", sym.code, "err", err)
			continue
		}
		slog.Info("backfilled", "i", i+1, "total", len(syms), "symbol", sym.code, "rows", n)
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

// RunIndices는 asset_class='INDEX' 활성 종목의 years년 일봉을 Yahoo에서 백필한다.
// 알파 카드(90D/1Y)가 의존한다.
func RunIndices(ctx context.Context, pool *pgxpool.Pool, years, limit int) error {
	yc := yahoo.NewClient()

	rows, err := pool.Query(ctx,
		`select id::text, symbol, exchange from public.instruments
		 where asset_class = 'INDEX' and is_active = true
		 order by symbol`)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	type idx struct{ id, symbol, exchange string }
	var list []idx
	for rows.Next() {
		var x idx
		if err := rows.Scan(&x.id, &x.symbol, &x.exchange); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		list = append(list, x)
	}
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}

	end := time.Now().UTC()
	start := end.AddDate(-years, 0, 0)
	for i, x := range list {
		ysym := schedule.IndexYahooSymbol(x.symbol, x.exchange)
		if ysym == "" {
			slog.Warn("no yahoo symbol for index", "symbol", x.symbol, "exchange", x.exchange)
			continue
		}
		n, err := backfillSymbol(ctx, pool, yc, x.id, ysym, start, end)
		if err != nil {
			slog.Warn("indices skip", "symbol", ysym, "err", err)
			continue
		}
		slog.Info("backfilled", "i", i+1, "total", len(list), "symbol", ysym, "rows", n)
		time.Sleep(200 * time.Millisecond) // Yahoo rate limit 보호
	}
	return nil
}

// backfillSymbol은 한 instrument의 [start,end] 일봉을 fetch·upsert한다. 호출자가 rate-limit sleep 담당.
func backfillSymbol(ctx context.Context, pool *pgxpool.Pool, yc *yahoo.Client,
	instrumentID, yahooSymbol string, start, end time.Time) (int64, error) {
	bars, err := yc.FetchChart(ctx, yahooSymbol, start, end)
	if err != nil {
		return 0, err
	}
	if len(bars) == 0 {
		return 0, nil
	}
	for j := range bars {
		bars[j].InstrumentID = instrumentID
	}
	return ingest.UpsertPrices(ctx, pool, bars)
}

// SeedIfEmpty는 부팅 시 비어 있는 INDEX·NASDAQ 시리즈만 채운다(멱등, per-series).
// NASDAQ instrument를 멱등 시드하고(INDEX는 마이그레이션이 시드), 각 대상의 봉 존재 여부를
// 확인해 0행인 시리즈만 백필한다. 부분 실패는 다음 부팅에서 실패분만 재시도된다(cross-boot self-heal).
// 하나라도 실패하면 집계 에러를 반환한다(호출자가 Sentry 보고용).
func SeedIfEmpty(ctx context.Context, pool *pgxpool.Pool, years int) error {
	// 1) NASDAQ instrument 시드 보장 (멱등 upsert).
	insts := make([]models.Instrument, 0, len(nasdaqSeed))
	for _, s := range nasdaqSeed {
		insts = append(insts, models.Instrument{
			Symbol: s.Symbol, Exchange: "NASDAQ", Name: s.Name,
			AssetClass: models.AssetUSStock, Currency: "USD", IsActive: true,
		})
	}
	if _, err := ingest.UpsertInstruments(ctx, pool, insts); err != nil {
		return fmt.Errorf("seed nasdaq instruments: %w", err)
	}

	// 2) 대상 조회: INDEX 전체 + NASDAQ 종목.
	rows, err := pool.Query(ctx,
		`select id::text, symbol, exchange, asset_class from public.instruments
		 where (asset_class = 'INDEX' or exchange = 'NASDAQ') and is_active = true
		 order by symbol`)
	if err != nil {
		return fmt.Errorf("query targets: %w", err)
	}
	defer rows.Close()
	type target struct{ id, symbol, exchange, assetClass string }
	var targets []target
	for rows.Next() {
		var t target
		if err := rows.Scan(&t.id, &t.symbol, &t.exchange, &t.assetClass); err != nil {
			return fmt.Errorf("scan target: %w", err)
		}
		targets = append(targets, t)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate targets: %w", err)
	}

	yc := yahoo.NewClient()
	end := time.Now().UTC()
	start := end.AddDate(-years, 0, 0)
	var seeded, failures int
	for _, t := range targets {
		has, err := hasBars(ctx, pool, t.id)
		if err != nil {
			slog.Warn("seed: hasBars failed", "symbol", t.symbol, "err", err)
			failures++
			continue
		}
		if has {
			continue // 이미 채워짐 — skip
		}
		ysym := seedYahooSymbol(t.assetClass, t.symbol, t.exchange)
		if ysym == "" {
			slog.Warn("seed: no yahoo symbol", "symbol", t.symbol, "exchange", t.exchange)
			continue
		}
		n, err := backfillSymbol(ctx, pool, yc, t.id, ysym, start, end)
		if err != nil {
			slog.Warn("seed: backfill failed", "symbol", ysym, "err", err)
			failures++
			continue
		}
		slog.Info("seed: backfilled", "symbol", ysym, "rows", n)
		seeded++
		time.Sleep(200 * time.Millisecond) // Yahoo rate limit 보호
	}
	slog.Info("seed: done", "seeded", seeded, "failures", failures, "targets", len(targets))
	if failures > 0 {
		return fmt.Errorf("seed: %d series failed (will retry next boot)", failures)
	}
	return nil
}

// hasBars는 instrument에 봉이 1개라도 있으면 true.
func hasBars(ctx context.Context, pool *pgxpool.Pool, instrumentID string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx,
		`select exists(select 1 from public.prices where instrument_id = $1)`,
		instrumentID).Scan(&exists)
	return exists, err
}

// seedYahooSymbol은 INDEX면 IndexYahooSymbol, NASDAQ 종목이면 심볼 그대로(RunUS와 동일).
func seedYahooSymbol(assetClass, symbol, exchange string) string {
	if assetClass == string(models.AssetIndex) {
		return schedule.IndexYahooSymbol(symbol, exchange)
	}
	return symbol
}
