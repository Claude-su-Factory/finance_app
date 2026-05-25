package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/quotient/quotient/apps/api/internal/ai"
)

type searchInstrument struct{ *Deps }

func (t *searchInstrument) Spec() ai.ToolSpec {
	return ai.ToolSpec{
		Name:        "search_instrument",
		Description: "한글·영문·티커로 종목 검색. alias 우선, 없으면 name/symbol ILIKE.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{"query": map[string]any{"type": "string"}},
			"required":   []string{"query"},
		},
	}
}
func (t *searchInstrument) Run(ctx context.Context, _ string, input map[string]any) (any, error) {
	q, _ := input["query"].(string)
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, fmt.Errorf("query required")
	}
	rows, err := t.Pool.Query(ctx, `
		select i.id::text, i.symbol, i.exchange, i.name, i.asset_class, i.currency
		from public.instrument_aliases a
		join public.instruments i on i.id = a.instrument_id
		where a.alias = $1 and i.is_active = true
		limit 10
	`, strings.ToLower(q))
	if err != nil {
		return nil, err
	}
	results, err := scanInstruments(rows)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		pat := "%" + strings.ToLower(q) + "%"
		rows2, err := t.Pool.Query(ctx, `
			select id::text, symbol, exchange, name, asset_class, currency from public.instruments
			where (lower(name) like $1 or lower(symbol) like $1) and is_active = true
			order by length(symbol) asc
			limit 10
		`, pat)
		if err != nil {
			return nil, err
		}
		results, err = scanInstruments(rows2)
		if err != nil {
			return nil, err
		}
	}
	return map[string]any{"results": results, "count": len(results)}, nil
}

func scanInstruments(rows interface {
	Next() bool
	Scan(...any) error
	Close()
	Err() error
}) ([]map[string]any, error) {
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id, sym, ex, name, cls, cur string
		if err := rows.Scan(&id, &sym, &ex, &name, &cls, &cur); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"id": id, "symbol": sym, "exchange": ex, "name": name,
			"asset_class": cls, "currency": cur,
		})
	}
	return out, rows.Err()
}

type getWatchlist struct{ *Deps }

func (t *getWatchlist) Spec() ai.ToolSpec {
	return ai.ToolSpec{
		Name:        "get_watchlist",
		Description: "사용자의 관심 종목 목록 + 현재 시세.",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	}
}
func (t *getWatchlist) Run(ctx context.Context, userID string, _ map[string]any) (any, error) {
	rows, err := t.Pool.Query(ctx, `
		select i.symbol, i.name, i.currency,
		       coalesce(q.price, 0)::float8, coalesce(q.change_pct, 0)::float8
		from public.watchlist w
		join public.instruments i on i.id = w.instrument_id
		left join public.quotes q on q.instrument_id = w.instrument_id
		where w.user_id = $1
		order by w.added_at desc
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type item struct {
		Symbol, Name, Currency string
		Price, ChangePct       float64
	}
	var out []item
	for rows.Next() {
		var x item
		if err := rows.Scan(&x.Symbol, &x.Name, &x.Currency, &x.Price, &x.ChangePct); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return map[string]any{"watchlist": out, "count": len(out)}, rows.Err()
}

type getEconomicIndicator struct{ *Deps }

func (t *getEconomicIndicator) Spec() ai.ToolSpec {
	return ai.ToolSpec{
		Name:        "get_economic_indicator",
		Description: "특정 경제 지표의 최근 값·추이. code 예: DFF (Fed Funds Rate), DGS10 (10Y Treasury), 722Y001 (BOK 기준금리).",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"code": map[string]any{"type": "string"}},
			"required":   []string{"code"},
		},
	}
}
func (t *getEconomicIndicator) Run(ctx context.Context, _ string, input map[string]any) (any, error) {
	code, _ := input["code"].(string)
	if code == "" {
		return nil, fmt.Errorf("code required")
	}
	rows, err := t.Pool.Query(ctx, `
		select observed_at, value::float8
		from public.economic_indicators
		where code = $1
		order by observed_at desc
		limit 30
	`, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type pt struct {
		At  any
		Val float64
	}
	var out []pt
	for rows.Next() {
		var p pt
		if err := rows.Scan(&p.At, &p.Val); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("indicator %q not found", code)
	}
	return map[string]any{"code": code, "points": out, "latest": out[0].Val}, rows.Err()
}

func RegisterSearch(r *Registry, d *Deps) {
	r.Register(&searchInstrument{Deps: d})
	r.Register(&getWatchlist{Deps: d})
	r.Register(&getEconomicIndicator{Deps: d})
}
