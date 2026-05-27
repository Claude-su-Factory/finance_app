package tools

import (
	"context"
	"fmt"

	"github.com/quotient/quotient/apps/api/internal/ai"
	"github.com/quotient/quotient/apps/api/internal/db"
)

type getQuote struct{ *Deps }

func (t *getQuote) Spec() ai.ToolSpec {
	return ai.ToolSpec{
		Name:        "get_quote",
		Description: "특정 종목의 현재 시세 (15분 지연). symbol 또는 instrument_id로 조회.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"symbol":        map[string]any{"type": "string"},
				"instrument_id": map[string]any{"type": "string"},
			},
		},
	}
}
func (t *getQuote) RequiresUserContext() bool { return false }
func (t *getQuote) Run(ctx context.Context, exec db.Executor, _ string, input map[string]any) (any, error) {
	sym, _ := input["symbol"].(string)
	iid, _ := input["instrument_id"].(string)
	if sym == "" && iid == "" {
		return nil, fmt.Errorf("symbol or instrument_id required")
	}
	var row [5]any
	var err error
	if iid != "" {
		err = exec.QueryRow(ctx, `
			select i.symbol, i.name, i.currency,
			       coalesce(q.price, 0)::float8, coalesce(q.change_pct, 0)::float8
			from public.instruments i
			left join public.quotes q on q.instrument_id = i.id
			where i.id = $1
		`, iid).Scan(&row[0], &row[1], &row[2], &row[3], &row[4])
	} else {
		err = exec.QueryRow(ctx, `
			select i.symbol, i.name, i.currency,
			       coalesce(q.price, 0)::float8, coalesce(q.change_pct, 0)::float8
			from public.instruments i
			left join public.quotes q on q.instrument_id = i.id
			where i.symbol = $1 limit 1
		`, sym).Scan(&row[0], &row[1], &row[2], &row[3], &row[4])
	}
	if err != nil {
		return nil, fmt.Errorf("not found: %w", err)
	}
	return map[string]any{
		"symbol": row[0], "name": row[1], "currency": row[2],
		"price": row[3], "change_pct": row[4],
		"delay_minutes": 15,
	}, nil
}

type getPriceHistory struct{ *Deps }

func (t *getPriceHistory) Spec() ai.ToolSpec {
	return ai.ToolSpec{
		Name:        "get_price_history",
		Description: "일봉 가격 시계열. range: 1w | 1mo | 6mo | 1y | 5y.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"symbol": map[string]any{"type": "string"},
				"range":  map[string]any{"type": "string", "enum": []string{"1w", "1mo", "6mo", "1y", "5y"}},
			},
			"required": []string{"symbol", "range"},
		},
	}
}
func (t *getPriceHistory) RequiresUserContext() bool { return false }
func (t *getPriceHistory) Run(ctx context.Context, exec db.Executor, _ string, input map[string]any) (any, error) {
	sym, _ := input["symbol"].(string)
	rng, _ := input["range"].(string)
	if sym == "" || rng == "" {
		return nil, fmt.Errorf("symbol and range required")
	}
	interval := map[string]string{
		"1w": "7 days", "1mo": "30 days", "6mo": "180 days",
		"1y": "365 days", "5y": "1825 days",
	}[rng]
	if interval == "" {
		return nil, fmt.Errorf("invalid range")
	}
	rows, err := exec.Query(ctx, `
		select p.date, p.close::float8
		from public.prices p
		join public.instruments i on i.id = p.instrument_id
		where i.symbol = $1 and p.date >= current_date - $2::interval
		order by p.date
	`, sym, interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type pt struct{ Date, Close any }
	var out []pt
	for rows.Next() {
		var p pt
		if err := rows.Scan(&p.Date, &p.Close); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return map[string]any{"symbol": sym, "range": rng, "points": out, "count": len(out)}, rows.Err()
}

type getMarketOverview struct{ *Deps }

func (t *getMarketOverview) Spec() ai.ToolSpec {
	return ai.ToolSpec{
		Name:        "get_market_overview",
		Description: "주요 지수·환율 현재 스냅샷 (KOSPI, KOSDAQ, S&P 500, NASDAQ, USD/KRW, EUR/KRW, JPY/KRW).",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	}
}
func (t *getMarketOverview) RequiresUserContext() bool { return false }
func (t *getMarketOverview) Run(ctx context.Context, exec db.Executor, _ string, _ map[string]any) (any, error) {
	rows, err := exec.Query(ctx, `
		select i.symbol, i.name, i.asset_class,
		       coalesce(q.price, 0)::float8, coalesce(q.change_pct, 0)::float8
		from public.instruments i
		left join public.quotes q on q.instrument_id = i.id
		where i.asset_class in ('INDEX', 'FX') and i.is_active = true
		order by case i.asset_class when 'INDEX' then 1 when 'FX' then 2 end, i.symbol
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type item struct {
		Symbol, Name, AssetClass string
		Price, ChangePct         float64
	}
	var out []item
	for rows.Next() {
		var x item
		if err := rows.Scan(&x.Symbol, &x.Name, &x.AssetClass, &x.Price, &x.ChangePct); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return map[string]any{"items": out, "count": len(out)}, rows.Err()
}

func RegisterQuote(r *Registry, d *Deps) {
	r.Register(&getQuote{Deps: d})
	r.Register(&getPriceHistory{Deps: d})
	r.Register(&getMarketOverview{Deps: d})
}
