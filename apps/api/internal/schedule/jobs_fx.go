package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/quotient/quotient/apps/api/internal/ingest"
	"github.com/quotient/quotient/apps/api/internal/models"
)

// JobUpdateFXRates는 frankfurter.dev에서 USD 기준 KRW/EUR/JPY 환율을 받아
// fx_rates(시계열) + quotes(현재값) 양쪽에 적재한다.
// 5분 cron, 24/7.
func JobUpdateFXRates(ctx context.Context, d Deps) error {
	rates, err := d.FX.FetchRates(ctx, "USD", []string{"KRW", "EUR", "JPY"})
	if err != nil {
		return fmt.Errorf("frankfurter: %w", err)
	}

	// 1) fx_rates 적재 (시계열)
	nFX, err := ingest.UpsertFXRates(ctx, d.Pool, rates)
	if err != nil {
		return fmt.Errorf("upsert fx_rates: %w", err)
	}

	// 2) quotes 갱신 (USD_KRW 등 instrument에 매핑)
	rateMap := map[string]float64{}
	for _, r := range rates {
		rateMap[fmt.Sprintf("%s_%s", r.Base, r.Quote)] = r.Rate
	}
	// frankfurter는 base=USD로 USD→{KRW,EUR,JPY}만 반환하므로 EUR_KRW/JPY_KRW는 derived 계산.
	// EUR_KRW = USD_KRW / USD_EUR, JPY_KRW = USD_KRW / USD_JPY.
	if usdKRW, ok := rateMap["USD_KRW"]; ok && usdKRW > 0 {
		if usdEUR, ok := rateMap["USD_EUR"]; ok && usdEUR > 0 {
			rateMap["EUR_KRW"] = usdKRW / usdEUR
		}
		if usdJPY, ok := rateMap["USD_JPY"]; ok && usdJPY > 0 {
			rateMap["JPY_KRW"] = usdKRW / usdJPY
		}
	}

	type instRow struct{ id, symbol string }
	rs, err := d.Pool.Query(ctx, `
		select id::text, symbol from public.instruments
		where asset_class = 'FX' and is_active = true
	`)
	if err != nil {
		return fmt.Errorf("query fx instruments: %w", err)
	}
	defer rs.Close()

	var quotes []models.Quote
	prevRates := previousRates(ctx, d, time.Now().UTC())
	// prev에도 동일 derived 계산 적용 (EUR_KRW/JPY_KRW change_pct 정상화)
	if usdKRW, ok := prevRates["USD_KRW"]; ok && usdKRW > 0 {
		if usdEUR, ok := prevRates["USD_EUR"]; ok && usdEUR > 0 {
			prevRates["EUR_KRW"] = usdKRW / usdEUR
		}
		if usdJPY, ok := prevRates["USD_JPY"]; ok && usdJPY > 0 {
			prevRates["JPY_KRW"] = usdKRW / usdJPY
		}
	}
	for rs.Next() {
		var r instRow
		if err := rs.Scan(&r.id, &r.symbol); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		v, ok := rateMap[r.symbol]
		if !ok {
			continue
		}
		var changeAbs, changePct float64
		if prev, ok := prevRates[r.symbol]; ok && prev > 0 {
			changeAbs = v - prev
			changePct = (changeAbs / prev) * 100.0
		}
		quotes = append(quotes, models.Quote{
			InstrumentID: r.id,
			Price:        v,
			ChangeAbs:    changeAbs,
			ChangePct:    changePct,
		})
	}
	if err := rs.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}

	nQ, err := ingest.UpsertQuotes(ctx, d.Pool, quotes)
	if err != nil {
		return fmt.Errorf("upsert fx quotes: %w", err)
	}

	slog.Info("fx updated", "fx_rates", nFX, "quotes", nQ)
	return nil
}

// previousRates는 오늘 미만 최신 영업일의 환율을 USD_XXX 키로 반환.
// 휴일·주말로 어제 데이터 부재 시 그 이전 가장 최근 영업일을 자동 선택.
// 첫 배포로 fx_rates에 오늘 행만 있으면 빈 맵 반환 → 호출자가 change_pct=0 처리 (의도).
func previousRates(ctx context.Context, d Deps, today time.Time) map[string]float64 {
	out := map[string]float64{}
	rs, err := d.Pool.Query(ctx, `
		select distinct on (base, quote) base, quote, rate
		from public.fx_rates
		where observed_at::date < $1::date
		order by base, quote, observed_at desc
	`, today.Format("2006-01-02"))
	if err != nil {
		return out
	}
	defer rs.Close()
	for rs.Next() {
		var base, q string
		var r float64
		if err := rs.Scan(&base, &q, &r); err == nil {
			out[fmt.Sprintf("%s_%s", base, q)] = r
		}
	}
	return out
}
