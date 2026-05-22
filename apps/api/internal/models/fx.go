package models

import "time"

type FXRate struct {
	Base       string    `json:"base"`
	Quote      string    `json:"quote"`
	ObservedAt time.Time `json:"observed_at"`
	Rate       float64   `json:"rate"`
}
