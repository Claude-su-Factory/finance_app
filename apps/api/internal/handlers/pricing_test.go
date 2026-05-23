package handlers

import "testing"

func TestToKRW(t *testing.T) {
	rates := map[string]float64{"KRW": 1.0, "USD": 1380.0, "JPY": 9.2}
	cases := []struct {
		amount   float64
		currency string
		want     float64
	}{
		{1000, "KRW", 1000},
		{10, "USD", 13800},
		{1000, "JPY", 9200},
		{100, "EUR", 100}, // 미정의 → 1.0 fallback (경고)
	}
	for _, c := range cases {
		got := ToKRW(c.amount, c.currency, rates)
		if got != c.want {
			t.Errorf("ToKRW(%v,%v) = %v, want %v", c.amount, c.currency, got, c.want)
		}
	}
}
