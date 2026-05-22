package fx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchRates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "base=USD")
		assert.Contains(t, r.URL.RawQuery, "symbols=KRW,EUR,JPY")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"date": "2025-12-30",
			"base": "USD",
			"rates": map[string]float64{"KRW": 1450.5, "EUR": 0.92, "JPY": 148.3},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	rates, err := c.FetchRates(context.Background(), "USD", []string{"KRW", "EUR", "JPY"})
	require.NoError(t, err)
	assert.Len(t, rates, 3)

	// Find KRW
	for _, r := range rates {
		if r.Quote == "KRW" {
			assert.Equal(t, "USD", r.Base)
			assert.Equal(t, 1450.5, r.Rate)
		}
	}
}
