package tools

import (
	"testing"
)

func TestAllToolsRegistered(t *testing.T) {
	r := NewRegistry()
	d := &Deps{Pool: nil}
	RegisterPortfolio(r, d)
	RegisterQuote(r, d)
	RegisterSearch(r, d)

	want := []string{
		"get_portfolio", "get_holding_detail", "calc_portfolio_metrics",
		"get_quote", "get_price_history", "get_market_overview",
		"search_instrument", "get_watchlist", "get_economic_indicator",
	}
	for _, name := range want {
		if _, ok := r.Get(name); !ok {
			t.Errorf("tool %s not registered", name)
		}
	}
	if got := len(r.Specs()); got != 9 {
		t.Errorf("got %d specs, want 9", got)
	}
	// 모든 도구 InputSchema.type == "object"
	for _, s := range r.Specs() {
		if s.InputSchema["type"] != "object" {
			t.Errorf("tool %s schema type != object", s.Name)
		}
	}
}
