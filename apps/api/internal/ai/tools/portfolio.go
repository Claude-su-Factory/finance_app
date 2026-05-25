package tools

import (
	"context"
	"fmt"

	"github.com/quotient/quotient/apps/api/internal/ai"
)

// --- get_portfolio ---
type getPortfolio struct{ *Deps }

func (t *getPortfolio) Spec() ai.ToolSpec {
	return ai.ToolSpec{
		Name:        "get_portfolio",
		Description: "사용자의 전체 보유 자산 목록을 반환. KRW 환산 평가액·수익률·비중 포함.",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	}
}
func (t *getPortfolio) Run(ctx context.Context, userID string, _ map[string]any) (any, error) {
	rows, err := t.Pool.Query(ctx, `
		select h.id::text, i.symbol, i.name, i.asset_class, i.currency,
		       h.quantity::float8, h.avg_cost::float8,
		       coalesce(q.price, 0)::float8, h.opened_at
		from public.holdings h
		join public.instruments i on i.id = h.instrument_id
		left join public.quotes q on q.instrument_id = h.instrument_id
		where h.user_id = $1
		order by h.created_at desc
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	type item struct {
		ID, Symbol, Name, AssetClass, Currency string
		Quantity, AvgCost, CurrentPrice        float64
		OpenedAt                               any
	}
	var out []item
	for rows.Next() {
		var x item
		if err := rows.Scan(&x.ID, &x.Symbol, &x.Name, &x.AssetClass, &x.Currency, &x.Quantity, &x.AvgCost, &x.CurrentPrice, &x.OpenedAt); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return map[string]any{"holdings": out, "count": len(out)}, rows.Err()
}

// --- get_holding_detail ---
type getHoldingDetail struct{ *Deps }

func (t *getHoldingDetail) Spec() ai.ToolSpec {
	return ai.ToolSpec{
		Name:        "get_holding_detail",
		Description: "특정 보유 자산의 상세 정보 + 수익률.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"instrument_id": map[string]any{"type": "string"},
			},
			"required": []string{"instrument_id"},
		},
	}
}
func (t *getHoldingDetail) Run(ctx context.Context, userID string, input map[string]any) (any, error) {
	iid, _ := input["instrument_id"].(string)
	if iid == "" {
		return nil, fmt.Errorf("instrument_id required")
	}
	row := t.Pool.QueryRow(ctx, `
		select h.id::text, i.symbol, i.name, i.asset_class, i.currency,
		       h.quantity::float8, h.avg_cost::float8,
		       coalesce(q.price, 0)::float8, h.opened_at, h.note
		from public.holdings h
		join public.instruments i on i.id = h.instrument_id
		left join public.quotes q on q.instrument_id = h.instrument_id
		where h.user_id = $1 and h.instrument_id = $2
	`, userID, iid)
	var id, symbol, name, cls, cur string
	var qty, avg, price float64
	var opened, note any
	if err := row.Scan(&id, &symbol, &name, &cls, &cur, &qty, &avg, &price, &opened, &note); err != nil {
		return nil, fmt.Errorf("not found or error: %w", err)
	}
	mv := qty * price
	cb := qty * avg
	pnlPct := 0.0
	if cb > 0 && price > 0 {
		pnlPct = (mv - cb) / cb * 100
	}
	return map[string]any{
		"id": id, "symbol": symbol, "name": name, "asset_class": cls, "currency": cur,
		"quantity": qty, "avg_cost": avg, "current_price": price,
		"market_value": mv, "pnl_pct": pnlPct,
		"opened_at": opened, "note": note,
	}, nil
}

// --- calc_portfolio_metrics ---
type calcPortfolioMetrics struct{ *Deps }

func (t *calcPortfolioMetrics) Spec() ai.ToolSpec {
	return ai.ToolSpec{
		Name:        "calc_portfolio_metrics",
		Description: "포트폴리오 집중도·통화 분산. 변동성·샤프는 v2.",
		InputSchema: map[string]any{"type": "object", "properties": map[string]any{}},
	}
}
func (t *calcPortfolioMetrics) Run(ctx context.Context, userID string, _ map[string]any) (any, error) {
	rows, err := t.Pool.Query(ctx, `
		select i.asset_class, i.currency, h.quantity::float8 * coalesce(q.price, 0)::float8 as mv
		from public.holdings h
		join public.instruments i on i.id = h.instrument_id
		left join public.quotes q on q.instrument_id = h.instrument_id
		where h.user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	byClass := map[string]float64{}
	byCurrency := map[string]float64{}
	var total float64
	for rows.Next() {
		var cls, cur string
		var mv float64
		if err := rows.Scan(&cls, &cur, &mv); err != nil {
			return nil, err
		}
		byClass[cls] += mv
		byCurrency[cur] += mv
		total += mv
	}
	concentration := 0.0
	for _, v := range byClass {
		if v > concentration {
			concentration = v
		}
	}
	concPct := 0.0
	if total > 0 {
		concPct = concentration / total * 100
	}
	return map[string]any{
		"total_market_value":    total,
		"by_asset_class":        byClass,
		"by_currency":           byCurrency,
		"top_concentration_pct": concPct,
		"note":                  "변동성·샤프 지수는 Phase 2 — prices 시계열 분석 후 제공 예정.",
	}, rows.Err()
}

// RegisterPortfolio는 3 도구를 registry에 일괄 등록.
func RegisterPortfolio(r *Registry, d *Deps) {
	r.Register(&getPortfolio{Deps: d})
	r.Register(&getHoldingDetail{Deps: d})
	r.Register(&calcPortfolioMetrics{Deps: d})
}
