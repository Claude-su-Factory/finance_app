package models

import "time"

type PaperAccount struct {
	UserID       string    `json:"user_id"`
	InitialCash  float64   `json:"initial_cash"`
	CashBalance  float64   `json:"cash_balance"`
	BaseCurrency string    `json:"base_currency"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PaperHolding struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	InstrumentID   string    `json:"instrument_id"`
	Symbol         string    `json:"symbol,omitempty"`
	Name           string    `json:"name,omitempty"`
	Currency       string    `json:"currency,omitempty"`
	Quantity       float64   `json:"quantity"`
	AvgCost        float64   `json:"avg_cost"`
	CurrentPrice   float64   `json:"current_price,omitempty"`
	MarketValue    float64   `json:"market_value,omitempty"`
	MarketValueKRW float64   `json:"market_value_krw,omitempty"`
	PnLKRW         float64   `json:"pnl_krw,omitempty"`
	PnLPct         float64   `json:"pnl_pct,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type PaperTransaction struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	InstrumentID string    `json:"instrument_id"`
	Symbol       string    `json:"symbol,omitempty"`
	Action       string    `json:"action"` // "buy" | "sell"
	Quantity     float64   `json:"quantity"`
	Price        float64   `json:"price"`
	Currency     string    `json:"currency"`
	FxToKRW      float64   `json:"fx_to_krw"`
	TotalKRW     float64   `json:"total_krw"`
	Active       bool      `json:"active"`
	CreatedAt    time.Time `json:"created_at"`
}

// EquityPoint — 일자별 KRW 평가액 (시계열용)
type EquityPoint struct {
	Date      string  `json:"date"`       // YYYY-MM-DD
	EquityKRW float64 `json:"equity_krw"`
}

// PaperTransactionCreate — POST body
type PaperTransactionCreate struct {
	InstrumentID string  `json:"instrument_id"`
	Action       string  `json:"action"`
	Quantity     float64 `json:"quantity"`
	Reason       *string `json:"reason,omitempty"`
}

// PaperPortfolioResponse — GET /v1/paper/portfolio 응답
type PaperPortfolioResponse struct {
	Account      PaperAccount  `json:"account"`
	Holdings     []PaperHolding `json:"holdings"`
	Summary      PaperSummary  `json:"summary"`
	EquitySeries []EquityPoint `json:"equity_series"`
}

type PaperSummary struct {
	TotalEquityKRW float64 `json:"total_equity_krw"`
	TotalPnLKRW    float64 `json:"total_pnl_krw"`
	TotalPnLPct    float64 `json:"total_pnl_pct"`
}

// PaperResetRequest — POST /v1/paper/reset body
type PaperResetRequest struct {
	InitialCash *float64 `json:"initial_cash,omitempty"`
}
