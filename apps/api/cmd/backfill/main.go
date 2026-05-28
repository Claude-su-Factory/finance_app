package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/config"
	"github.com/quotient/quotient/apps/api/internal/db"
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

func main() {
	years := flag.Int("years", 5, "백필 기간 (연 단위)")
	market := flag.String("market", "KOSPI", "KOSPI | KOSDAQ | NASDAQ | INDICES")
	limit := flag.Int("limit", 0, "최대 종목 수 (0=전체). 디버깅용.")
	flag.Parse()

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	switch *market {
	case "KOSPI", "KOSDAQ":
		if err := runKR(ctx, pool, *market, *years, *limit); err != nil {
			slog.Error("kr backfill", "err", err)
			os.Exit(1)
		}
	case "NASDAQ":
		if err := runUS(ctx, pool, *years, *limit); err != nil {
			slog.Error("us backfill", "err", err)
			os.Exit(1)
		}
	case "INDICES":
		if err := runIndices(ctx, pool, *years, *limit); err != nil {
			slog.Error("indices backfill", "err", err)
			os.Exit(1)
		}
	default:
		slog.Error("unknown market", "market", *market)
		os.Exit(1)
	}
	slog.Info("backfill done", "market", *market, "years", *years)
}

func runKR(ctx context.Context, pool *pgxpool.Pool, market string, years, limit int) error {
	kc := kind.NewClient("")
	yc := yahoo.NewClient()

	// 1) 종목 마스터
	items, err := kc.FetchInstruments(ctx, market)
	if err != nil {
		return fmt.Errorf("kind: %w", err)
	}
	slog.Info("instruments fetched", "market", market, "count", len(items))
	if _, err := ingest.UpsertInstruments(ctx, pool, items); err != nil {
		return fmt.Errorf("upsert instruments: %w", err)
	}

	// 2) DB에서 id+symbol 조회 (지수·FX 제외, KRX exchange)
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
	if limit > 0 && len(syms) > limit {
		syms = syms[:limit]
	}

	end := time.Now().UTC()
	start := end.AddDate(-years, 0, 0)
	for i, sym := range syms {
		ysym := yahoo.SymbolKR(sym.code, market) // KOSPI→.KS, KOSDAQ→.KQ
		bars, err := yc.FetchChart(ctx, ysym, start, end)
		if err != nil {
			slog.Warn("yahoo skip", "symbol", ysym, "err", err)
			continue
		}
		if len(bars) == 0 {
			continue
		}
		for j := range bars {
			bars[j].InstrumentID = sym.id
		}
		n, err := ingest.UpsertPrices(ctx, pool, bars)
		if err != nil {
			slog.Warn("upsert skip", "symbol", ysym, "err", err)
			continue
		}
		slog.Info("backfilled", "i", i+1, "total", len(syms), "symbol", ysym, "rows", n)
		time.Sleep(200 * time.Millisecond) // Yahoo rate limit 보호
	}
	return nil
}

func runUS(ctx context.Context, pool *pgxpool.Pool, years, limit int) error {
	yc := yahoo.NewClient()

	// 1) instruments에 시드 upsert
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

	// 2) DB에서 id 조회
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
	if limit > 0 && len(syms) > limit {
		syms = syms[:limit]
	}

	end := time.Now().UTC()
	start := end.AddDate(-years, 0, 0)
	for i, sym := range syms {
		bars, err := yc.FetchChart(ctx, sym.code, start, end)
		if err != nil {
			slog.Warn("yahoo skip", "symbol", sym.code, "err", err)
			continue
		}
		if len(bars) == 0 {
			continue
		}
		for j := range bars {
			bars[j].InstrumentID = sym.id
		}
		n, err := ingest.UpsertPrices(ctx, pool, bars)
		if err != nil {
			slog.Warn("upsert skip", "symbol", sym.code, "err", err)
			continue
		}
		slog.Info("backfilled", "i", i+1, "total", len(syms), "symbol", sym.code, "rows", n)
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

// runIndices는 asset_class='INDEX' 종목(KOSPI/KOSDAQ/SPX/NDX/DJI 등)의 5년 일봉을 Yahoo에서 백필.
// 알파 카드(90D/1Y)가 의존하므로 MVP 출시 전 1회 실행 필요.
func runIndices(ctx context.Context, pool *pgxpool.Pool, years, limit int) error {
	yc := yahoo.NewClient()

	// instruments에서 활성 인덱스만 조회. exchange는 KRX-IDX/NYSE-IDX/NASDAQ-IDX 형태.
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
		bars, err := yc.FetchChart(ctx, ysym, start, end)
		if err != nil {
			slog.Warn("yahoo skip", "symbol", ysym, "err", err)
			continue
		}
		if len(bars) == 0 {
			continue
		}
		for j := range bars {
			bars[j].InstrumentID = x.id
		}
		n, err := ingest.UpsertPrices(ctx, pool, bars)
		if err != nil {
			slog.Warn("upsert skip", "symbol", ysym, "err", err)
			continue
		}
		slog.Info("backfilled", "i", i+1, "total", len(list), "symbol", ysym, "rows", n)
		time.Sleep(200 * time.Millisecond) // Yahoo rate limit 보호
	}
	return nil
}
