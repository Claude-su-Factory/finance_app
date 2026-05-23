package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/quotient/quotient/apps/api/internal/ingest"
	"github.com/quotient/quotient/apps/api/internal/models"
)

const quotesCacheTTL = 60 * time.Second

// JobUpdateMarketQuotes refreshes quotes for INDEX ∪ holdings ∪ watchlist (dedup).
// 시세 TTL 60초 (spec §10-2). KR/US 장중 판정은 asset_class·exchange 기반.
func JobUpdateMarketQuotes(ctx context.Context, d Deps) error {
	type row struct {
		id, symbol, exchange, assetClass string
		updatedAt                        *time.Time // nullable: 신규 종목은 quotes 행 없음
	}

	// 1) 폴링 대상 = INDEX ∪ holdings ∪ watchlist (dedup via UNION).
	rs, err := d.Pool.Query(ctx, `
		with targets as (
			select id from public.instruments where is_active = true and asset_class = 'INDEX'
			union
			select instrument_id as id from public.holdings
			union
			select instrument_id as id from public.watchlist
		)
		select i.id::text, i.symbol, i.exchange, i.asset_class, q.updated_at
		from targets t
		join public.instruments i on i.id = t.id
		left join public.quotes q on q.instrument_id = i.id
		where i.is_active = true
	`)
	if err != nil {
		return fmt.Errorf("query market quotes targets: %w", err)
	}
	defer rs.Close()

	var rows []row
	for rs.Next() {
		var r row
		if err := rs.Scan(&r.id, &r.symbol, &r.exchange, &r.assetClass, &r.updatedAt); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		rows = append(rows, r)
	}
	if err := rs.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}

	now := time.Now()
	quotes := make([]models.Quote, 0, len(rows))
	for _, r := range rows {
		// TTL 캐시 (spec §10-2)
		if r.updatedAt != nil && now.Sub(*r.updatedAt) < quotesCacheTTL {
			continue
		}

		// 장중 판정: asset_class·exchange 기반
		switch r.assetClass {
		case "INDEX":
			if isKRIndex(r.symbol) && !IsKRMarketOpen(now) {
				continue
			}
			if isUSIndex(r.symbol) && !IsUSMarketOpen(now) {
				continue
			}
		case "KR_STOCK", "ETF":
			// ETF는 KR/US 모두 가능 — exchange로 분기
			if isKRExchange(r.exchange) && !IsKRMarketOpen(now) {
				continue
			}
			if isUSExchange(r.exchange) && !IsUSMarketOpen(now) {
				continue
			}
		case "US_STOCK":
			if !IsUSMarketOpen(now) {
				continue
			}
		case "CASH":
			continue // CASH 시세 폴링 대상 아님
		case "FX":
			continue // FX 별도 잡(5분 주기)이 처리
		}

		// INDEX는 IndexYahooSymbol, 그 외는 StockYahooSymbol
		var ysym string
		if r.assetClass == "INDEX" {
			ysym = IndexYahooSymbol(r.symbol, r.exchange)
		} else {
			ysym = StockYahooSymbol(r.symbol, r.exchange)
		}
		if ysym == "" {
			continue
		}
		q, err := d.Yahoo.FetchQuote(ctx, ysym)
		if err != nil {
			slog.Warn("yahoo quote skip", "symbol", ysym, "err", err)
			continue
		}
		q.InstrumentID = r.id
		quotes = append(quotes, q)
		time.Sleep(50 * time.Millisecond)
	}

	if len(quotes) == 0 {
		return nil
	}
	n, err := ingest.UpsertQuotes(ctx, d.Pool, quotes)
	if err != nil {
		return fmt.Errorf("upsert quotes: %w", err)
	}
	slog.Info("market quotes updated", "count", n)
	return nil
}

func isKRIndex(symbol string) bool {
	return symbol == "KOSPI" || symbol == "KOSDAQ"
}

func isUSIndex(symbol string) bool {
	return strings.HasPrefix(symbol, "SP") || symbol == "NDX"
}
