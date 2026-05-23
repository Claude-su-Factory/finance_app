package models

import "time"

// Holding은 보유 자산 한 건의 raw 데이터.
type Holding struct {
	ID           string     `json:"id"`
	InstrumentID string     `json:"instrument_id"`
	Quantity     float64    `json:"quantity"`
	AvgCost      float64    `json:"avg_cost"`
	OpenedAt     *time.Time `json:"opened_at,omitempty"`
	Note         *string    `json:"note,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// HoldingEnriched는 instrument·quote join + KRW 환산 평가액·수익률·비중.
type HoldingEnriched struct {
	Holding
	Symbol         string  `json:"symbol"`
	Exchange       string  `json:"exchange"`
	Name           string  `json:"name"`
	AssetClass     string  `json:"asset_class"`
	Currency       string  `json:"currency"`         // 원본 통화
	CurrentPrice   float64 `json:"current_price"`    // quotes.price (원본 통화), 없으면 0
	MarketValue    float64 `json:"market_value"`     // qty * price (원본 통화)
	MarketValueKRW float64 `json:"market_value_krw"` // KRW 환산
	CostBasisKRW   float64 `json:"cost_basis_krw"`   // qty * avg_cost in KRW
	PnLKRW         float64 `json:"pnl_krw"`          // market_value_krw - cost_basis_krw
	PnLPct         float64 `json:"pnl_pct"`          // (market_value - cost_basis) / cost_basis * 100, 원본 통화 기준
	WeightPct      float64 `json:"weight_pct"`       // 전체 KRW 평가 합계 대비 비중
}
