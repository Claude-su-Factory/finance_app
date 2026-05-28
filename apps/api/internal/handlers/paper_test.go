package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

type fakePaperRepo struct {
	account     *models.PaperAccount
	holding     *models.PaperHolding
	upsertCalls []upsertCall
	deleteCalls []string
	insertedTx  *models.PaperTransaction
	txList      []models.PaperTransaction
}

type upsertCall struct {
	InstrumentID string
	Quantity     float64
	AvgCost      float64
}

func (f *fakePaperRepo) GetOrCreateAccount(_ context.Context, _ db.Executor, uid string) (*models.PaperAccount, error) {
	if f.account == nil {
		f.account = &models.PaperAccount{UserID: uid, InitialCash: 10000000, CashBalance: 10000000, BaseCurrency: "KRW", CreatedAt: time.Now()}
	}
	return f.account, nil
}
func (f *fakePaperRepo) ApplyCashDelta(_ context.Context, _ db.Executor, _ string, delta float64) (float64, error) {
	if f.account == nil {
		return 0, ErrPaperAccountNotFound
	}
	if f.account.CashBalance+delta < 0 {
		return 0, ErrInsufficientCash
	}
	f.account.CashBalance += delta
	return f.account.CashBalance, nil
}
func (f *fakePaperRepo) ResetAccount(_ context.Context, _ db.Executor, _ string, initialCash float64) (*models.PaperAccount, error) {
	f.account.InitialCash = initialCash
	f.account.CashBalance = initialCash
	return f.account, nil
}
func (f *fakePaperRepo) ListHoldings(_ context.Context, _ db.Executor, _ string) ([]models.PaperHolding, error) {
	if f.holding != nil {
		return []models.PaperHolding{*f.holding}, nil
	}
	return nil, nil
}
func (f *fakePaperRepo) GetHolding(_ context.Context, _ db.Executor, _, iid string) (*models.PaperHolding, error) {
	if f.holding != nil && f.holding.InstrumentID == iid {
		return f.holding, nil
	}
	return nil, ErrPaperHoldingNotFound
}
func (f *fakePaperRepo) UpsertHolding(_ context.Context, _ db.Executor, _, iid string, qty, avgCost float64) error {
	f.upsertCalls = append(f.upsertCalls, upsertCall{iid, qty, avgCost})
	f.holding = &models.PaperHolding{InstrumentID: iid, Quantity: qty, AvgCost: avgCost}
	return nil
}
func (f *fakePaperRepo) DeleteHolding(_ context.Context, _ db.Executor, _, iid string) error {
	f.deleteCalls = append(f.deleteCalls, iid)
	f.holding = nil
	return nil
}
func (f *fakePaperRepo) DeleteAllHoldings(_ context.Context, _ db.Executor, _ string) error {
	f.holding = nil
	return nil
}
func (f *fakePaperRepo) InsertTransaction(_ context.Context, _ db.Executor, t *models.PaperTransaction) error {
	t.ID = "fake-tx-id"
	t.CreatedAt = time.Now()
	f.insertedTx = t
	return nil
}
func (f *fakePaperRepo) ListTransactions(_ context.Context, _ db.Executor, _ string, _ int, _ *time.Time) ([]models.PaperTransaction, bool, error) {
	return f.txList, false, nil
}
func (f *fakePaperRepo) ListActiveTransactionsSince(_ context.Context, _ db.Executor, _ string, _ time.Time) ([]models.PaperTransaction, error) {
	return f.txList, nil
}
func (f *fakePaperRepo) InactivateAllTransactions(_ context.Context, _ db.Executor, _ string) error {
	return nil
}

func reqPaper(method, target, body, uid string) *http.Request {
	r := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	if uid != "" {
		r = r.WithContext(middleware.WithUserID(r.Context(), uid))
	}
	return r
}

func newPaperHandlerForTest(repo *fakePaperRepo) *PaperHandler {
	// pool=nil이라 asset_class·quotes·fx_rates 조회 단계에서 503 가드 — Task 5에서 GetPortfolio·ListTx·Reset 추가 unit
	return NewPaperHandler(repo, &fakeJournalRepo{}, nil, nil)
}

func TestTrade_NoAuth_401(t *testing.T) {
	h := newPaperHandlerForTest(&fakePaperRepo{})
	w := httptest.NewRecorder()
	h.Trade(w, reqPaper(http.MethodPost, "/v1/paper/transactions", `{}`, ""))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d", w.Code)
	}
}

func TestTrade_BadAction_422(t *testing.T) {
	h := newPaperHandlerForTest(&fakePaperRepo{})
	w := httptest.NewRecorder()
	h.Trade(w, reqPaper(http.MethodPost, "/v1/paper/transactions",
		`{"instrument_id":"x","action":"foo","quantity":1}`, "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d", w.Code)
	}
}

func TestTrade_ZeroQuantity_422(t *testing.T) {
	h := newPaperHandlerForTest(&fakePaperRepo{})
	w := httptest.NewRecorder()
	h.Trade(w, reqPaper(http.MethodPost, "/v1/paper/transactions",
		`{"instrument_id":"x","action":"buy","quantity":0}`, "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d", w.Code)
	}
}

func TestTrade_NoPool_503(t *testing.T) {
	h := newPaperHandlerForTest(&fakePaperRepo{})
	w := httptest.NewRecorder()
	h.Trade(w, reqPaper(http.MethodPost, "/v1/paper/transactions",
		`{"instrument_id":"x","action":"buy","quantity":1}`, "user-1"))
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("status=%d", w.Code)
	}
}

func TestCalcBuyAvgCost_NewBuy(t *testing.T) {
	q, a := CalcBuyAvgCost(0, 0, 10, 70000)
	if q != 10 || a != 70000 {
		t.Errorf("q=%v a=%v, want 10 70000", q, a)
	}
}

func TestCalcBuyAvgCost_AdditionalBuy(t *testing.T) {
	q, a := CalcBuyAvgCost(10, 70000, 10, 80000)
	if q != 20 || a != 75000 {
		t.Errorf("q=%v a=%v, want 20 75000", q, a)
	}
}

func TestCalcBuyAvgCost_PartialQuantity(t *testing.T) {
	q, a := CalcBuyAvgCost(0.5, 60000, 0.5, 70000)
	if q != 1 || a != 65000 {
		t.Errorf("q=%v a=%v, want 1 65000", q, a)
	}
}

// 빌드 가드
var _ = portfolio.NewEquityComputer
var _ json.RawMessage
