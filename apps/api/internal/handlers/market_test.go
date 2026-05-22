package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeMarketRepo struct {
	items []TickerItem
	err   error
}

func (f *fakeMarketRepo) TickerSeed(ctx context.Context) ([]TickerItem, error) {
	return f.items, f.err
}

func TestMarketHandler_Ticker_OK(t *testing.T) {
	repo := &fakeMarketRepo{items: []TickerItem{
		{Symbol: "KOSPI", Name: "KOSPI 종합", Price: 2700.5, ChangePct: 0.42},
		{Symbol: "SPX", Name: "S&P 500", Price: 4800, ChangePct: -0.1},
	}}
	h := NewMarketHandler(repo)
	w := httptest.NewRecorder()
	h.Ticker(w, httptest.NewRequest(http.MethodGet, "/v1/market/ticker", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got []TickerItem
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 || got[0].Symbol != "KOSPI" {
		t.Errorf("got %+v", got)
	}
}

func TestMarketHandler_Ticker_NilToEmptyArray(t *testing.T) {
	h := NewMarketHandler(&fakeMarketRepo{items: nil})
	w := httptest.NewRecorder()
	h.Ticker(w, httptest.NewRequest(http.MethodGet, "/v1/market/ticker", nil))
	if got := w.Body.String(); got != "[]\n" {
		t.Errorf("nil items should serialize as [], got %q", got)
	}
}

func TestMarketHandler_Ticker_Error(t *testing.T) {
	h := NewMarketHandler(&fakeMarketRepo{err: errors.New("db down")})
	w := httptest.NewRecorder()
	h.Ticker(w, httptest.NewRequest(http.MethodGet, "/v1/market/ticker", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}
