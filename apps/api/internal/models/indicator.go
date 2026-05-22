package models

import "time"

type Indicator struct {
	Code       string    `json:"code"`
	ObservedAt time.Time `json:"observed_at"`
	Name       string    `json:"name"`
	Value      float64   `json:"value"`
	Unit       *string   `json:"unit,omitempty"`
}
