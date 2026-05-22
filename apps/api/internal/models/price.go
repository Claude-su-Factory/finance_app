package models

import "time"

type PriceBar struct {
	InstrumentID string    `json:"instrument_id"`
	Date         time.Time `json:"date"`
	Open         float64   `json:"open"`
	High         float64   `json:"high"`
	Low          float64   `json:"low"`
	Close        float64   `json:"close"`
	Volume       int64     `json:"volume"`
}
