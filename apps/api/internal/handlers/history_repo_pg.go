package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PricePoint struct {
	Date  string  `json:"date"`
	Close float64 `json:"close"`
}

type IndicatorPoint struct {
	ObservedAt string  `json:"observed_at"`
	Value      float64 `json:"value"`
}

type FxPoint struct {
	ObservedAt string  `json:"observed_at"`
	Rate       float64 `json:"rate"`
}

type HistoryRepo interface {
	PriceBySymbol(ctx context.Context, symbol, rng string) ([]PricePoint, error)
	PriceByIDsBatch(ctx context.Context, ids []string, rng string) (map[string][]PricePoint, error)
	Indicator(ctx context.Context, code string, days int) ([]IndicatorPoint, error)
	Fx(ctx context.Context, base, quote string, days int) ([]FxPoint, error)
}

type PgHistoryRepo struct {
	pool *pgxpool.Pool
}

func NewPgHistoryRepo(pool *pgxpool.Pool) *PgHistoryRepo {
	return &PgHistoryRepo{pool: pool}
}

func rangeToInterval(rng string) string {
	switch rng {
	case "1w":
		return "7 days"
	case "1mo":
		return "30 days"
	case "6mo":
		return "180 days"
	case "1y":
		return "365 days"
	case "5y":
		return "1825 days"
	}
	return ""
}

func (r *PgHistoryRepo) PriceBySymbol(ctx context.Context, symbol, rng string) ([]PricePoint, error) {
	interval := rangeToInterval(rng)
	if interval == "" {
		return nil, fmt.Errorf("invalid range")
	}
	// p.date는 DATE 타입이므로 명시적 DATE cast로 비교 (timezone 영향 회피)
	rows, err := r.pool.Query(ctx, `
		select to_char(p.date, 'YYYY-MM-DD'), p.close::float8
		from public.prices p
		join public.instruments i on i.id = p.instrument_id
		where i.symbol = $1 and i.is_active = true
		  and p.date >= (current_date - $2::interval)::date
		order by p.date
	`, symbol, interval)
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

// PriceByIDsBatch returns last `rng` days of close prices for each instrument_id.
// UUID 직접 비교로 prices_date_idx 인덱스 활용.
func (r *PgHistoryRepo) PriceByIDsBatch(ctx context.Context, ids []string, rng string) (map[string][]PricePoint, error) {
	interval := rangeToInterval(rng)
	if interval == "" {
		return nil, fmt.Errorf("invalid range")
	}
	if len(ids) == 0 {
		return map[string][]PricePoint{}, nil
	}
	rows, err := r.pool.Query(ctx, `
		select p.instrument_id::text, to_char(p.date, 'YYYY-MM-DD'), p.close::float8
		from public.prices p
		where p.instrument_id = any($1::uuid[])
		  and p.date >= (current_date - $2::interval)::date
		order by p.instrument_id, p.date
	`, ids, interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]PricePoint{}
	for rows.Next() {
		var iid string
		var p PricePoint
		if err := rows.Scan(&iid, &p.Date, &p.Close); err != nil {
			return nil, err
		}
		out[iid] = append(out[iid], p)
	}
	return out, rows.Err()
}

func (r *PgHistoryRepo) Indicator(ctx context.Context, code string, days int) ([]IndicatorPoint, error) {
	if days <= 0 || days > 3650 {
		days = 90
	}
	// observed_at은 timestamptz. now() - 일 단위 interval 명시 (timezone 안전)
	rows, err := r.pool.Query(ctx, `
		select to_char(observed_at, 'YYYY-MM-DD'), value::float8
		from public.economic_indicators
		where code = $1
		  and observed_at >= now() - ($2::int * interval '1 day')
		order by observed_at
	`, code, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IndicatorPoint
	for rows.Next() {
		var p IndicatorPoint
		if err := rows.Scan(&p.ObservedAt, &p.Value); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PgHistoryRepo) Fx(ctx context.Context, base, quote string, days int) ([]FxPoint, error) {
	if days <= 0 || days > 3650 {
		days = 30
	}
	base = strings.ToUpper(base)
	quote = strings.ToUpper(quote)
	rows, err := r.pool.Query(ctx, `
		select to_char(observed_at, 'YYYY-MM-DD'), rate::float8
		from public.fx_rates
		where base = $1 and quote = $2
		  and observed_at >= now() - ($3::int * interval '1 day')
		order by observed_at
	`, base, quote, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FxPoint
	for rows.Next() {
		var p FxPoint
		if err := rows.Scan(&p.ObservedAt, &p.Rate); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
