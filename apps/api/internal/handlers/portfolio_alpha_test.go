package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

type fakeAlpha struct {
	res *portfolio.AlphaResult
	err error
}

func (f *fakeAlpha) Compute(_ context.Context, _ db.Executor, _ db.Executor, _ string, _ portfolio.Period) (*portfolio.AlphaResult, error) {
	return f.res, f.err
}

func reqAlpha(period, uid string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/v1/portfolio/alpha?period="+period, nil)
	if uid != "" {
		r = r.WithContext(middleware.WithUserID(r.Context(), uid))
	}
	return r
}

func TestAlphaHandler_OK(t *testing.T) {
	svc := &fakeAlpha{res: &portfolio.AlphaResult{Period: portfolio.Period90D, DaysUsed: 90}}
	h := NewAlphaHandler(svc, nil)
	w := httptest.NewRecorder()
	h.Get(w, reqAlpha("90d", "user-1"))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["period"] != "90d" {
		t.Errorf("period=%v", got["period"])
	}
}

func TestAlphaHandler_BadPeriod(t *testing.T) {
	h := NewAlphaHandler(&fakeAlpha{}, nil)
	w := httptest.NewRecorder()
	h.Get(w, reqAlpha("FOO", "user-1"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400", w.Code)
	}
}

func TestAlphaHandler_NoAuth(t *testing.T) {
	h := NewAlphaHandler(&fakeAlpha{}, nil)
	w := httptest.NewRecorder()
	h.Get(w, reqAlpha("90d", ""))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d, want 401", w.Code)
	}
}

func TestAlphaHandler_AccountTooYoung(t *testing.T) {
	svc := &fakeAlpha{err: &portfolio.InsufficientDataError{Reason: "account_too_young", MinDays: 7, CurrentDays: 3}}
	h := NewAlphaHandler(svc, nil)
	w := httptest.NewRecorder()
	h.Get(w, reqAlpha("90d", "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", w.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	errBlock := got["error"].(map[string]any)
	if errBlock["reason"] != "account_too_young" || errBlock["current_days"].(float64) != 3 {
		t.Errorf("error=%+v", errBlock)
	}
}

func TestAlphaHandler_NoHoldings(t *testing.T) {
	svc := &fakeAlpha{err: &portfolio.InsufficientDataError{Reason: "no_holdings"}}
	h := NewAlphaHandler(svc, nil)
	w := httptest.NewRecorder()
	h.Get(w, reqAlpha("1y", "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d", w.Code)
	}
}
