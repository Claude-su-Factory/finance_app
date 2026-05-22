package models

import "time"

type AssetClass string

const (
	AssetKRStock AssetClass = "KR_STOCK"
	AssetUSStock AssetClass = "US_STOCK"
	AssetETF     AssetClass = "ETF"
	AssetCash    AssetClass = "CASH"
	AssetIndex   AssetClass = "INDEX"
	AssetFX      AssetClass = "FX"
)

type Instrument struct {
	ID         string     `json:"id"`
	Symbol     string     `json:"symbol"`
	Exchange   string     `json:"exchange"`
	ISIN       *string    `json:"isin,omitempty"`
	Name       string     `json:"name"`
	AssetClass AssetClass `json:"asset_class"`
	Currency   string     `json:"currency"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type Quote struct {
	InstrumentID string    `json:"instrument_id"`
	Price        float64   `json:"price"`
	ChangeAbs    float64   `json:"change_abs"`
	ChangePct    float64   `json:"change_pct"`
	UpdatedAt    time.Time `json:"updated_at"`
}
