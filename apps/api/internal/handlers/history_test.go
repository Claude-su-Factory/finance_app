package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeHistoryRepo struct {
	prices     []PricePoint
	batch      map[string][]PricePoint
	indicators []IndicatorPoint
	fxPoints   []FxPoint
	err        error
}

func (f *fakeHistoryRepo) PriceBySymbol(ctx context.Context, symbol, rng string) ([]PricePoint, error) {
	return f.prices, f.err
}
func (f *fakeHistoryRepo) PriceByIDsBatch(ctx context.Context, ids []string, rng string) (map[string][]PricePoint, error) {
	return f.batch, f.err
}
func (f *fakeHistoryRepo) Indicator(ctx context.Context, code string, days int) ([]IndicatorPoint, error) {
	return f.indicators, f.err
}
func (f *fakeHistoryRepo) Fx(ctx context.Context, base, quote string, days int) ([]FxPoint, error) {
	return f.fxPoints, f.err
}

func TestPrices_InvalidRange(t *testing.T) {
	h := NewHistoryHandler(&fakeHistoryRepo{})
	w := httptest.NewRecorder()
	h.Prices(w, httptest.NewRequest(http.MethodGet, "/v1/prices/history?symbol=KOSPI&range=invalid", nil))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("got %d want 422", w.Code)
	}
}

func TestPrices_MissingSymbol(t *testing.T) {
	h := NewHistoryHandler(&fakeHistoryRepo{})
	w := httptest.NewRecorder()
	h.Prices(w, httptest.NewRequest(http.MethodGet, "/v1/prices/history", nil))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("got %d want 422", w.Code)
	}
}

func TestPrices_SingleSymbol(t *testing.T) {
	repo := &fakeHistoryRepo{prices: []PricePoint{{Date: "2026-05-20", Close: 100}}}
	h := NewHistoryHandler(repo)
	w := httptest.NewRecorder()
	h.Prices(w, httptest.NewRequest(http.MethodGet, "/v1/prices/history?symbol=KOSPI&range=1w", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, body=%s", w.Code, w.Body.String())
	}
	var got struct {
		Symbol string       `json:"symbol"`
		Points []PricePoint `json:"points"`
		Count  int          `json:"count"`
	}
	_ = json.NewDecoder(w.Body).Decode(&got)
	if got.Symbol != "KOSPI" || got.Count != 1 {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestPrices_BatchMode(t *testing.T) {
	repo := &fakeHistoryRepo{batch: map[string][]PricePoint{
		"id-1": {{Date: "2026-05-20", Close: 100}},
		"id-2": {{Date: "2026-05-20", Close: 200}},
	}}
	h := NewHistoryHandler(repo)
	w := httptest.NewRecorder()
	h.Prices(w, httptest.NewRequest(http.MethodGet, "/v1/prices/history?ids=id-1,id-2&range=1w", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("got %d", w.Code)
	}
	var got struct {
		Items map[string][]PricePoint `json:"items"`
	}
	_ = json.NewDecoder(w.Body).Decode(&got)
	if len(got.Items) != 2 {
		t.Errorf("got %d items, want 2", len(got.Items))
	}
}

func TestPrices_BatchTooMany(t *testing.T) {
	h := NewHistoryHandler(&fakeHistoryRepo{})
	parts := make([]string, 101)
	for i := range 101 {
		parts[i] = fmt.Sprintf("id%d", i)
	}
	ids := strings.Join(parts, ",")
	w := httptest.NewRecorder()
	h.Prices(w, httptest.NewRequest(http.MethodGet, "/v1/prices/history?ids="+ids+"&range=1w", nil))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("got %d want 422", w.Code)
	}
}

func TestIndicators_MissingCode(t *testing.T) {
	h := NewHistoryHandler(&fakeHistoryRepo{})
	w := httptest.NewRecorder()
	h.Indicators(w, httptest.NewRequest(http.MethodGet, "/v1/indicators/history", nil))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("got %d want 422", w.Code)
	}
}

func TestFx_MissingPair(t *testing.T) {
	h := NewHistoryHandler(&fakeHistoryRepo{})
	w := httptest.NewRecorder()
	h.Fx(w, httptest.NewRequest(http.MethodGet, "/v1/fx/history?base=USD", nil))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("got %d want 422", w.Code)
	}
}
