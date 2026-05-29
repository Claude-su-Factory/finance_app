package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

type fakeBacktestRunner struct {
	res *portfolio.BacktestResult
	err error
}

func (f *fakeBacktestRunner) Run(_ context.Context, _ db.Executor, _ portfolio.BacktestRequest) (*portfolio.BacktestResult, error) {
	return f.res, f.err
}

func validBacktestBody() portfolio.BacktestRequest {
	return portfolio.BacktestRequest{
		Period: "3Y", InitialCash: 10_000_000, Monthly: 0, Rebalance: "none",
		Basket: []portfolio.BasketItem{{InstrumentID: "id1", Weight: 100}},
	}
}

func reqBacktest(t *testing.T, body portfolio.BacktestRequest, uid string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatalf("encode: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, "/v1/backtest/run", &buf)
	if uid != "" {
		r = r.WithContext(middleware.WithUserID(r.Context(), uid))
	}
	return r
}

func TestBacktestHandler_OK(t *testing.T) {
	svc := &fakeBacktestRunner{res: &portfolio.BacktestResult{ClampedStart: "2023-05-30", End: "2026-05-29"}}
	h := NewBacktestHandler(svc, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, validBacktestBody(), "user-1"))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["clamped_start"] != "2023-05-30" {
		t.Errorf("clamped_start=%v", got["clamped_start"])
	}
}

func TestBacktestHandler_NoAuth(t *testing.T) {
	h := NewBacktestHandler(&fakeBacktestRunner{}, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, validBacktestBody(), ""))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d, want 401", w.Code)
	}
}

func TestBacktestHandler_EmptyBasket(t *testing.T) {
	b := validBacktestBody()
	b.Basket = nil
	h := NewBacktestHandler(&fakeBacktestRunner{}, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, b, "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d, want 422", w.Code)
	}
}

func TestBacktestHandler_TooManyBasket(t *testing.T) {
	b := validBacktestBody()
	b.Basket = nil
	for i := 0; i < 11; i++ {
		b.Basket = append(b.Basket, portfolio.BasketItem{InstrumentID: "id", Weight: 1})
	}
	h := NewBacktestHandler(&fakeBacktestRunner{}, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, b, "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d, want 422", w.Code)
	}
}

func TestBacktestHandler_BadInitialCash(t *testing.T) {
	b := validBacktestBody()
	b.InitialCash = 0
	h := NewBacktestHandler(&fakeBacktestRunner{}, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, b, "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d, want 422", w.Code)
	}
}

func TestBacktestHandler_NegativeMonthly(t *testing.T) {
	b := validBacktestBody()
	b.Monthly = -1
	h := NewBacktestHandler(&fakeBacktestRunner{}, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, b, "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d, want 422", w.Code)
	}
}

func TestBacktestHandler_ZeroWeight(t *testing.T) {
	b := validBacktestBody()
	b.Basket = []portfolio.BasketItem{{InstrumentID: "id1", Weight: 0}}
	h := NewBacktestHandler(&fakeBacktestRunner{}, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, b, "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d, want 422", w.Code)
	}
}

func TestBacktestHandler_AssetNotSupported(t *testing.T) {
	svc := &fakeBacktestRunner{err: &portfolio.ValidationError{Code: "ASSET_NOT_SUPPORTED", Message: "지수·환율은 백테스트 불가"}}
	h := NewBacktestHandler(svc, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, validBacktestBody(), "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", w.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	errBlock, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error block: %+v", got)
	}
	if errBlock["code"] != "ASSET_NOT_SUPPORTED" {
		t.Errorf("code=%v", errBlock["code"])
	}
}

func TestBacktestHandler_InsufficientData(t *testing.T) {
	svc := &fakeBacktestRunner{err: &portfolio.InsufficientDataError{Reason: "backtest_window_too_short", MinDays: 30, CurrentDays: 12}}
	h := NewBacktestHandler(svc, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, validBacktestBody(), "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", w.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	errBlock, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("missing error block: %+v", got)
	}
	if errBlock["code"] != "INSUFFICIENT_DATA" || errBlock["current_days"].(float64) != 12 {
		t.Errorf("error=%+v", errBlock)
	}
}

func TestBacktestHandler_BadJSON(t *testing.T) {
	h := NewBacktestHandler(&fakeBacktestRunner{}, nil)
	r := httptest.NewRequest(http.MethodPost, "/v1/backtest/run", strings.NewReader("{bad"))
	r = r.WithContext(middleware.WithUserID(r.Context(), "user-1"))
	w := httptest.NewRecorder()
	h.Run(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400", w.Code)
	}
}
