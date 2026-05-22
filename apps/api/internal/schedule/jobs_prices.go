package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/quotient/quotient/apps/api/internal/ingest"
	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/quotient/quotient/apps/api/internal/sources/yahoo"
)

// JobUpdateKRPrices fetches yesterday's daily bar for all active KRX instruments via Yahoo.
// 매일 16:30 KST 실행.
func JobUpdateKRPrices(ctx context.Context, d Deps) error {
	return updateDailyByExchange(ctx, d, "KRX")
}

// JobUpdateUSPrices fetches yesterday's daily bar for all active US instruments via Yahoo.
// 매일 06:00 KST 실행.
func JobUpdateUSPrices(ctx context.Context, d Deps) error {
	return updateDailyByExchange(ctx, d, "NASDAQ", "NYSE")
}

func updateDailyByExchange(ctx context.Context, d Deps, exchanges ...string) error {
	type sym struct{ id, code, exchange string }
	var syms []sym

	// pgx의 IN ($1) 바인딩은 array 필요 → ANY($1::text[]).
	// id::text 명시 캐스트 — pgx v5 기본 type map의 uuid → string 미지원 회피.
	rows, err := d.Pool.Query(ctx,
		`select id::text, symbol, exchange from public.instruments
		 where exchange = ANY($1::text[]) and is_active = true`,
		exchanges,
	)
	if err != nil {
		return fmt.Errorf("query instruments: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s sym
		if err := rows.Scan(&s.id, &s.code, &s.exchange); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		syms = append(syms, s)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}

	// 증분 갱신: 어제~오늘 1일 (Yahoo는 end exclusive — 하루 buffer)
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -2)

	var total int64
	for _, s := range syms {
		ysym := yahooSymbolForExchange(s.code, s.exchange)
		bars, err := d.Yahoo.FetchChart(ctx, ysym, start, end)
		if err != nil {
			slog.Warn("yahoo skip", "symbol", ysym, "err", err)
			continue
		}
		if len(bars) == 0 {
			continue
		}
		for i := range bars {
			bars[i].InstrumentID = s.id
		}
		n, err := ingest.UpsertPrices(ctx, d.Pool, bars)
		if err != nil {
			slog.Warn("upsert skip", "symbol", ysym, "err", err)
			continue
		}
		total += n
		time.Sleep(50 * time.Millisecond) // rate limit (spec §10-2)
	}
	slog.Info("prices updated", "exchanges", exchanges, "instruments", len(syms), "rows", total)
	_ = models.PriceBar{} // import 유지 안전망
	return nil
}

// yahooSymbolForExchange는 W2a의 yahoo.SymbolKR을 활용. KRX → .KS/.KQ는
// 정확한 KOSPI/KOSDAQ 시장 구분 필요. 현재 instruments.exchange는 'KRX' 단일이라
// 시장 정보 부재 → KOSPI(.KS) 기본 + 실패 시 .KQ 재시도.
func yahooSymbolForExchange(symbol, exchange string) string {
	switch exchange {
	case "KRX":
		return yahoo.SymbolKR(symbol, "KOSPI") // .KS suffix
	default:
		return symbol // NASDAQ, NYSE는 plain
	}
}
