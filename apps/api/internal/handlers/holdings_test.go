package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
)

func TestEnrichHoldings(t *testing.T) {
	rates := map[string]float64{"KRW": 1.0, "USD": 1380.0}
	items := []models.HoldingEnriched{
		{
			Holding:      models.Holding{Quantity: 10, AvgCost: 70000},
			Currency:     "KRW",
			CurrentPrice: 80000,
		},
		{
			Holding:      models.Holding{Quantity: 5, AvgCost: 150},
			Currency:     "USD",
			CurrentPrice: 180,
		},
	}
	got := enrichHoldings(items, rates)

	if got[0].MarketValueKRW != 800000 {
		t.Errorf("KR market_value_krw: got %v want 800000", got[0].MarketValueKRW)
	}
	if got[0].PnLPct < 14.28 || got[0].PnLPct > 14.29 {
		t.Errorf("KR pnl_pct: got %v want ~14.2857", got[0].PnLPct)
	}
	if got[1].MarketValueKRW != 1242000 {
		t.Errorf("US market_value_krw: got %v want 1242000", got[1].MarketValueKRW)
	}
	total := 800000.0 + 1242000.0
	wantW0 := 800000 / total * 100
	if got[0].WeightPct < wantW0-0.01 || got[0].WeightPct > wantW0+0.01 {
		t.Errorf("KR weight: got %v want ~%v", got[0].WeightPct, wantW0)
	}
}

type fakeHoldingRepo struct {
	createErr error
	deleteErr error
	created   *models.Holding
}

func (f *fakeHoldingRepo) List(ctx context.Context, exec db.Executor, userID string) ([]models.HoldingEnriched, error) {
	_ = exec
	return []models.HoldingEnriched{}, nil
}
func (f *fakeHoldingRepo) Create(ctx context.Context, exec db.Executor, userID, instrumentID string, qty, avgCost float64, openedAt *string, note *string) (*models.Holding, error) {
	_ = exec
	if f.createErr != nil {
		return nil, f.createErr
	}
	h := &models.Holding{ID: "fake-id", InstrumentID: instrumentID, Quantity: qty, AvgCost: avgCost}
	f.created = h
	return h, nil
}
func (f *fakeHoldingRepo) Update(ctx context.Context, exec db.Executor, userID, id string, patch map[string]any) (*models.Holding, error) {
	_ = exec
	return &models.Holding{ID: id}, nil
}
func (f *fakeHoldingRepo) Delete(ctx context.Context, exec db.Executor, userID, id string) error {
	_ = exec
	return f.deleteErr
}

func reqWithUID(method, target, body, uid string) *http.Request {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r = r.WithContext(middleware.WithUserID(r.Context(), uid))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestHoldingCreate_ValidationBefore_AssetClassGuard(t *testing.T) {
	// asset_class 가드는 DB 호출 필요라 fake repo로 도달 못 함.
	// 여기선 검증 단계까지만 — InstrumentID 누락·quantity·avg_cost·인증.
	repo := &fakeHoldingRepo{}
	h := NewHoldingHandler(repo, nil, nil)

	cases := []struct {
		name string
		body string
		want int
	}{
		{"missing instrument", `{"quantity":1,"avg_cost":100}`, http.StatusUnprocessableEntity},
		{"zero quantity", `{"instrument_id":"x","quantity":0,"avg_cost":100}`, http.StatusUnprocessableEntity},
		{"negative avg_cost", `{"instrument_id":"x","quantity":1,"avg_cost":-1}`, http.StatusUnprocessableEntity},
		{"no auth", `{"instrument_id":"x","quantity":1,"avg_cost":100}`, http.StatusUnauthorized},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var r *http.Request
			if c.name == "no auth" {
				r = httptest.NewRequest(http.MethodPost, "/v1/holdings", strings.NewReader(c.body))
			} else {
				r = reqWithUID(http.MethodPost, "/v1/holdings", c.body, "user-1")
			}
			w := httptest.NewRecorder()
			h.Create(w, r)
			if w.Code != c.want {
				t.Errorf("got %d want %d, body=%s", w.Code, c.want, w.Body.String())
			}
		})
	}
}

func TestHoldingDelete_NotFound(t *testing.T) {
	repo := &fakeHoldingRepo{deleteErr: ErrHoldingNotFound}
	h := NewHoldingHandler(repo, nil, nil)
	r := reqWithUID(http.MethodDelete, "/v1/holdings/abc", "", "user-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	r = r.WithContext(middleware.WithUserID(r.Context(), "user-1"))

	w := httptest.NewRecorder()
	h.Delete(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("got %d want 404", w.Code)
	}
}

func TestCreateHolding_WithReason_CreatesAutoEntry(t *testing.T) {
	repo := &fakeHoldingRepo{}
	jrepo := &fakeJournalRepo{}
	h := NewHoldingHandler(repo, jrepo, nil)
	body := `{"instrument_id":"x","quantity":1,"avg_cost":100,"reason":"실적 회복 기대"}`
	w := httptest.NewRecorder()
	h.Create(w, reqWithUID(http.MethodPost, "/v1/holdings", body, "user-1"))
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	if jrepo.lastCreateContent != "실적 회복 기대" {
		t.Errorf("auto entry not created with reason: %v", jrepo.lastCreateContent)
	}
}

// suppress unused imports if not all used
var _ = bytes.NewReader
var _ = json.NewDecoder
