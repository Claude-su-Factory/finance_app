package portfolio

import (
	"context"
	"strings"
	"time"

	"github.com/quotient/quotient/apps/api/internal/db"
)

// NewService는 production용 Service. Pg 구현 주입.
func NewService() *Service {
	return &Service{deps: &PgDeps{}, now: time.Now}
}

// PgDeps는 Service의 production 구현. Pure SQL.
// 모든 일자 비교는 `::date` 직접 비교 — 인덱스 활용 + timezone 안전.
type PgDeps struct{}

func (PgDeps) UserCreatedAt(ctx context.Context, exec db.Executor, uid string) (time.Time, error) {
	var t time.Time
	err := exec.QueryRow(ctx, `select created_at from public.profiles where id = $1`, uid).Scan(&t)
	return t, err
}

func (PgDeps) UserHoldings(ctx context.Context, exec db.Executor, uid string) ([]HoldingRow, error) {
	rows, err := exec.Query(ctx, `
		select h.instrument_id::text, i.symbol, i.currency, h.quantity::float8
		from public.holdings h
		join public.instruments i on i.id = h.instrument_id
		where h.user_id = $1
	`, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HoldingRow
	for rows.Next() {
		var r HoldingRow
		if err := rows.Scan(&r.InstrumentID, &r.Symbol, &r.Currency, &r.Quantity); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// TradingDays는 KOSPI ∪ SPX prices distinct date 합집합.
// KR·US 영업일 캘린더 차이를 모두 포괄 → 종목 가격·fx는 forward-fill로 채움.
func (PgDeps) TradingDays(ctx context.Context, pool db.Executor, since, until time.Time) ([]string, error) {
	rows, err := pool.Query(ctx, `
		select distinct to_char(p.date, 'YYYY-MM-DD')
		from public.prices p
		join public.instruments i on i.id = p.instrument_id
		where i.symbol in ('KOSPI', 'SPX')
		  and p.date >= $1::date and p.date <= $2::date
		order by 1
	`, since, until)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// InstrumentClosesOnDates — p.date를 date[]와 직접 비교 (인덱스 활용).
func (PgDeps) InstrumentClosesOnDates(ctx context.Context, pool db.Executor, iid string, dates []string) (map[string]float64, error) {
	if len(dates) == 0 {
		return map[string]float64{}, nil
	}
	rows, err := pool.Query(ctx, `
		select to_char(p.date, 'YYYY-MM-DD'), p.close::float8
		from public.prices p
		where p.instrument_id = $1::uuid and p.date = any($2::date[])
	`, iid, dates)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]float64{}
	for rows.Next() {
		var d string
		var v float64
		if err := rows.Scan(&d, &v); err != nil {
			return nil, err
		}
		out[d] = v
	}
	return out, rows.Err()
}

// FxRatesOnDates — observed_at::date 비교 (timezone 안전).
// 첫 일자 fx 누락 가드: range 시작 직전 가장 최근 1개 추가 조회 → 결과 맵에 "__before" 키로 저장.
func (PgDeps) FxRatesOnDates(ctx context.Context, pool db.Executor, currency string, dates []string) (map[string]float64, error) {
	if len(dates) == 0 || currency == "KRW" {
		return map[string]float64{}, nil
	}
	currency = strings.ToUpper(currency)
	rows, err := pool.Query(ctx, `
		select to_char(observed_at::date, 'YYYY-MM-DD'), rate::float8
		from public.fx_rates
		where base = $1 and quote = 'KRW' and observed_at::date = any($2::date[])
	`, currency, dates)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]float64{}
	for rows.Next() {
		var d string
		var v float64
		if err := rows.Scan(&d, &v); err != nil {
			return nil, err
		}
		out[d] = v
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// before fallback — range 첫 일자 직전 가장 최근 fx 1점
	firstDate := dates[0]
	var beforeRate float64
	err = pool.QueryRow(ctx, `
		select rate::float8
		from public.fx_rates
		where base = $1 and quote = 'KRW' and observed_at::date < $2::date
		order by observed_at desc limit 1
	`, currency, firstDate).Scan(&beforeRate)
	if err == nil && beforeRate > 0 {
		out["__before"] = beforeRate
	}
	return out, nil
}

func (PgDeps) BenchmarkSeries(ctx context.Context, pool db.Executor, symbol string, dates []string) ([]PricePoint, error) {
	if len(dates) == 0 {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		select to_char(p.date, 'YYYY-MM-DD'), p.close::float8
		from public.prices p
		join public.instruments i on i.id = p.instrument_id
		where i.symbol = $1 and p.date = any($2::date[])
		order by p.date
	`, symbol, dates)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PricePoint
	for rows.Next() {
		var p PricePoint
		if err := rows.Scan(&p.Date, &p.Close); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// InstrumentsMeta — 바스켓 종목 메타(symbol·name·currency·asset_class) 일괄 조회.
func (PgDeps) InstrumentsMeta(ctx context.Context, pool db.Executor, ids []string) (map[string]InstrumentMeta, error) {
	if len(ids) == 0 {
		return map[string]InstrumentMeta{}, nil
	}
	rows, err := pool.Query(ctx, `
		select id::text, symbol, name, currency, asset_class
		from public.instruments where id = any($1::uuid[])
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]InstrumentMeta{}
	for rows.Next() {
		var id string
		var m InstrumentMeta
		if err := rows.Scan(&id, &m.Symbol, &m.Name, &m.Currency, &m.AssetClass); err != nil {
			return nil, err
		}
		out[id] = m
	}
	return out, rows.Err()
}
