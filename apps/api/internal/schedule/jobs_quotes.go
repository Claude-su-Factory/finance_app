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

// JobUpdateIndexQuotes refreshes quotes for indices (KOSPI, KOSDAQ, SPX, NDX) every minute during market hours.
// 시세 TTL 60초 (spec §10-2) — updated_at이 60초 미만이면 skip.
// W3에서 holdings/watchlist union으로 확장 (현재는 INDEX 전용).
func JobUpdateIndexQuotes(ctx context.Context, d Deps) error {
	type row struct {
		id, symbol, exchange string
		updatedAt            *time.Time // nullable: 신규 종목은 quotes 행 없음
	}

	// 1) 폴링 대상 + 마지막 적재 시각 (id::text — pgx v5 uuid scan 안전)
	rs, err := d.Pool.Query(ctx, `
		select i.id::text, i.symbol, i.exchange, q.updated_at
		from public.instruments i
		left join public.quotes q on q.instrument_id = i.id
		where i.is_active = true and i.asset_class = 'INDEX'
	`)
	if err != nil {
		return fmt.Errorf("query indices: %w", err)
	}
	defer rs.Close()

	var rows []row
	for rs.Next() {
		var r row
		if err := rs.Scan(&r.id, &r.symbol, &r.exchange, &r.updatedAt); err != nil {
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

		// 장중 판정 (KOSPI/KOSDAQ → KR, SPX/NDX → US)
		if isKRIndex(r.symbol) && !IsKRMarketOpen(now) {
			continue
		}
		if isUSIndex(r.symbol) && !IsUSMarketOpen(now) {
			continue
		}

		ysym := IndexYahooSymbol(r.symbol, r.exchange)
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
	slog.Info("index quotes updated", "count", n)
	return nil
}

func isKRIndex(symbol string) bool {
	return symbol == "KOSPI" || symbol == "KOSDAQ"
}

func isUSIndex(symbol string) bool {
	return strings.HasPrefix(symbol, "SP") || symbol == "NDX"
}
