package handlers

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FetchFXRates는 현재 quotes에서 통화별 KRW 환율을 반환.
// 키: 원본 통화(예: "USD"). 값: KRW 환산 배수(예: 1380.5).
// KRW는 항상 1.0. quotes 미존재 통화는 1.0 fallback + 경고.
func FetchFXRates(ctx context.Context, pool *pgxpool.Pool) (map[string]float64, error) {
	// FX instruments의 symbol은 "USD_KRW", "EUR_KRW", "JPY_KRW" 형식.
	// "<CURRENCY>_KRW" 패턴에서 첫 토큰을 키로 추출.
	rows, err := pool.Query(ctx, `
		select i.symbol, coalesce(q.price, 0)::float8
		from public.instruments i
		left join public.quotes q on q.instrument_id = i.id
		where i.asset_class = 'FX' and i.symbol like '%\_KRW' escape '\'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rates := map[string]float64{"KRW": 1.0}
	for rows.Next() {
		var sym string
		var price float64
		if err := rows.Scan(&sym, &price); err != nil {
			return nil, err
		}
		// "USD_KRW" → "USD"
		base := sym
		for i, c := range sym {
			if c == '_' {
				base = sym[:i]
				break
			}
		}
		if price <= 0 {
			slog.Warn("FX rate missing, fallback 1.0", "symbol", sym)
			rates[base] = 1.0
			continue
		}
		rates[base] = price
	}
	return rates, rows.Err()
}

// ToKRW는 amount를 원본 통화에서 KRW로 환산.
// 알 수 없는 통화는 1.0(KRW 가정) + 경고 로그.
func ToKRW(amount float64, currency string, rates map[string]float64) float64 {
	r, ok := rates[currency]
	if !ok {
		slog.Warn("unknown currency, treating as KRW", "currency", currency)
		return amount
	}
	return amount * r
}
