package models

import "time"

type WatchlistItem struct {
	InstrumentID string    `json:"instrument_id"`
	AddedAt      time.Time `json:"added_at"`
	Symbol       string    `json:"symbol"`
	Exchange     string    `json:"exchange"`
	Name         string    `json:"name"`
	AssetClass   string    `json:"asset_class"`
	Currency     string    `json:"currency"`
	Price        float64   `json:"price"`      // quotes.price, 없으면 0
	ChangePct    float64   `json:"change_pct"`
}
