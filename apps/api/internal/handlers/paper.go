package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

type PaperHandler struct {
	repo        PaperRepo
	journalRepo JournalRepo
	equity      *portfolio.EquityComputer
	pool        *pgxpool.Pool
	run         txRunner
}

func NewPaperHandler(repo PaperRepo, journalRepo JournalRepo, equity *portfolio.EquityComputer, pool *pgxpool.Pool) *PaperHandler {
	h := &PaperHandler{repo: repo, journalRepo: journalRepo, equity: equity, pool: pool}
	if pool == nil {
		h.run = func(ctx context.Context, fn func(db.Executor) error) error { return fn(nil) }
		return h
	}
	h.run = func(ctx context.Context, fn func(db.Executor) error) error {
		uid := middleware.UserID(ctx)
		return db.AsUser(ctx, pool, uid, fn)
	}
	return h
}

// POST /v1/paper/transactions
func (h *PaperHandler) Trade(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var body models.PaperTransactionCreate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if body.InstrumentID == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "instrument_id required")
		return
	}
	if body.Action != "buy" && body.Action != "sell" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "action must be buy or sell")
		return
	}
	if body.Quantity <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "quantity must be > 0")
		return
	}

	if h.pool == nil {
		writeError(w, http.StatusServiceUnavailable, "INTERNAL", "pool unavailable")
		return
	}
	var assetClass, currency string
	if err := h.pool.QueryRow(r.Context(),
		`select asset_class, currency from public.instruments where id = $1::uuid`, body.InstrumentID,
	).Scan(&assetClass, &currency); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "instrument not found")
		return
	}
	if assetClass == "INDEX" || assetClass == "FX" || assetClass == "CASH" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error": map[string]any{
				"code": "ASSET_NOT_SUPPORTED", "message": "지수·환율은 매매 불가",
			},
		})
		return
	}

	var price float64
	if err := h.pool.QueryRow(r.Context(),
		`select coalesce(price, 0)::float8 from public.quotes where instrument_id = $1::uuid`, body.InstrumentID,
	).Scan(&price); err != nil || price <= 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error": map[string]any{
				"code": "NO_QUOTE", "message": "현재 시세 없음 — 잠시 후 시도",
			},
		})
		return
	}

	fxToKRW := 1.0
	if currency != "KRW" {
		var fx float64
		if err := h.pool.QueryRow(r.Context(), `
			select rate::float8 from public.fx_rates
			where base = $1 and quote = 'KRW'
			order by observed_at desc limit 1
		`, currency).Scan(&fx); err != nil || fx <= 0 {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error": map[string]any{
					"code": "NO_QUOTE", "message": "환율 데이터 없음",
				},
			})
			return
		}
		fxToKRW = fx
	}

	totalKRW := body.Quantity * price * fxToKRW

	var responseBody map[string]any
	err := h.run(r.Context(), func(exec db.Executor) error {
		if _, err := h.repo.GetOrCreateAccount(r.Context(), exec, uid); err != nil {
			return err
		}

		var newHolding *models.PaperHolding

		if body.Action == "buy" {
			oldH, err := h.repo.GetHolding(r.Context(), exec, uid, body.InstrumentID)
			if err != nil && !errors.Is(err, ErrPaperHoldingNotFound) {
				return err
			}
			var oldQty, oldAvg float64
			if oldH != nil {
				oldQty, oldAvg = oldH.Quantity, oldH.AvgCost
			}
			newQty, newAvg := CalcBuyAvgCost(oldQty, oldAvg, body.Quantity, price)
			if err := h.repo.UpsertHolding(r.Context(), exec, uid, body.InstrumentID, newQty, newAvg); err != nil {
				return err
			}
			if _, err := h.repo.ApplyCashDelta(r.Context(), exec, uid, -totalKRW); err != nil {
				if errors.Is(err, ErrInsufficientCash) {
					curBal := 0.0
					_ = h.pool.QueryRow(r.Context(), `select cash_balance::float8 from public.paper_account where user_id = $1`, uid).Scan(&curBal)
					return &paperBusinessErr{
						Code:    "INSUFFICIENT_CASH",
						Detail:  map[string]any{"need_krw": totalKRW, "have_krw": curBal},
						Message: "잔액 부족",
					}
				}
				return err
			}
			newHolding = &models.PaperHolding{InstrumentID: body.InstrumentID, Quantity: newQty, AvgCost: newAvg}
		} else { // sell
			oldH, err := h.repo.GetHolding(r.Context(), exec, uid, body.InstrumentID)
			if err != nil {
				if errors.Is(err, ErrPaperHoldingNotFound) {
					return &paperBusinessErr{Code: "INSUFFICIENT_HOLDING", Message: "보유 수량 없음"}
				}
				return err
			}
			if oldH.Quantity < body.Quantity {
				return &paperBusinessErr{
					Code:    "INSUFFICIENT_HOLDING",
					Detail:  map[string]any{"need_qty": body.Quantity, "have_qty": oldH.Quantity},
					Message: "보유 수량 부족",
				}
			}
			newQty := oldH.Quantity - body.Quantity
			if newQty <= 0 {
				if err := h.repo.DeleteHolding(r.Context(), exec, uid, body.InstrumentID); err != nil {
					return err
				}
				newHolding = nil
			} else {
				if err := h.repo.UpsertHolding(r.Context(), exec, uid, body.InstrumentID, newQty, oldH.AvgCost); err != nil {
					return err
				}
				newHolding = &models.PaperHolding{InstrumentID: body.InstrumentID, Quantity: newQty, AvgCost: oldH.AvgCost}
			}
			if _, err := h.repo.ApplyCashDelta(r.Context(), exec, uid, totalKRW); err != nil {
				return err
			}
		}

		tx := &models.PaperTransaction{
			UserID:       uid,
			InstrumentID: body.InstrumentID,
			Action:       body.Action,
			Quantity:     body.Quantity,
			Price:        price,
			Currency:     currency,
			FxToKRW:      fxToKRW,
			TotalKRW:     totalKRW,
			Active:       true,
		}
		if err := h.repo.InsertTransaction(r.Context(), exec, tx); err != nil {
			return err
		}

		if body.Reason != nil && *body.Reason != "" && h.journalRepo != nil {
			reason := *body.Reason
			if len([]rune(reason)) > 200 {
				reason = string([]rune(reason)[:200])
			}
			action := body.Action
			var sym string
			_ = h.pool.QueryRow(r.Context(), `select symbol from public.instruments where id = $1::uuid`, body.InstrumentID).Scan(&sym)
			symbols := []string{}
			if sym != "" {
				symbols = []string{sym}
			}
			_, _ = h.journalRepo.CreateEntry(r.Context(), exec, uid, "auto", models.JournalEntryCreate{
				Action: &action, RelatedSymbols: symbols, Content: reason,
			}, nil)
		}

		newAccount, err := h.repo.GetOrCreateAccount(r.Context(), exec, uid)
		if err != nil {
			return err
		}
		responseBody = map[string]any{
			"transaction":      tx,
			"new_cash_balance": newAccount.CashBalance,
			"holding":          newHolding,
		}
		return nil
	})

	if err != nil {
		var bizErr *paperBusinessErr
		if errors.As(err, &bizErr) {
			payload := map[string]any{
				"code": bizErr.Code, "message": bizErr.Message,
			}
			maps.Copy(payload, bizErr.Detail)
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": payload})
			return
		}
		slog.Error("paper trade failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "trade failed")
		return
	}
	writeJSON(w, http.StatusCreated, responseBody)
}

type paperBusinessErr struct {
	Code    string
	Message string
	Detail  map[string]any
}

func (e *paperBusinessErr) Error() string { return e.Code + ": " + e.Message }

// CalcBuyAvgCost는 매수 시 가중 평균 매수가 계산 — 순수 함수로 unit 검증 가능.
//
//	newQty = oldQty + addQty
//	newAvg = (oldQty * oldAvg + addQty * addPrice) / newQty
//
// oldQty==0인 신규 매수도 처리.
func CalcBuyAvgCost(oldQty, oldAvg, addQty, addPrice float64) (newQty, newAvg float64) {
	newQty = oldQty + addQty
	if newQty <= 0 {
		return newQty, 0
	}
	newAvg = (oldQty*oldAvg + addQty*addPrice) / newQty
	return
}

// GET /v1/paper/portfolio?period=90d
func (h *PaperHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	period := r.URL.Query().Get("period")
	days := 90
	switch period {
	case "1m":
		days = 30
	case "90d", "":
		days = 90
	case "1y":
		days = 365
	case "all":
		days = 365 * 5
	default:
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "period must be 1m|90d|1y|all")
		return
	}

	var resp models.PaperPortfolioResponse
	err := h.run(r.Context(), func(exec db.Executor) error {
		account, err := h.repo.GetOrCreateAccount(r.Context(), exec, uid)
		if err != nil {
			return err
		}
		holdings, err := h.repo.ListHoldings(r.Context(), exec, uid)
		if err != nil {
			return err
		}
		totalEquity := account.CashBalance
		for i := range holdings {
			holdings[i] = enrichHolding(r.Context(), h.pool, holdings[i])
			totalEquity += holdings[i].MarketValueKRW
		}

		// equity computer 부재 시 (테스트 등) — 빈 시리즈
		var series []models.EquityPoint
		if h.equity != nil {
			txs, err := h.repo.ListActiveTransactionsSince(r.Context(), exec, uid,
				time.Now().AddDate(0, 0, -days))
			if err != nil {
				return err
			}
			series, err = h.equity.Compute(r.Context(), h.pool, account, txs, holdings, days)
			if err != nil {
				return err
			}
		}

		resp = models.PaperPortfolioResponse{
			Account:      *account,
			Holdings:     holdings,
			EquitySeries: series,
			Summary: models.PaperSummary{
				TotalEquityKRW: totalEquity,
				TotalPnLKRW:    totalEquity - account.InitialCash,
				TotalPnLPct:    pctChange(account.InitialCash, totalEquity),
			},
		}
		return nil
	})
	if err != nil {
		slog.Error("paper portfolio failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if resp.Holdings == nil {
		resp.Holdings = []models.PaperHolding{}
	}
	if resp.EquitySeries == nil {
		resp.EquitySeries = []models.EquityPoint{}
	}
	writeJSON(w, http.StatusOK, resp)
}

func enrichHolding(ctx context.Context, pool *pgxpool.Pool, h models.PaperHolding) models.PaperHolding {
	if pool == nil {
		return h
	}
	var price float64
	_ = pool.QueryRow(ctx, `select coalesce(price, 0)::float8 from public.quotes where instrument_id = $1::uuid`, h.InstrumentID).Scan(&price)
	h.CurrentPrice = price
	h.MarketValue = h.Quantity * price
	fx := 1.0
	if h.Currency != "KRW" {
		_ = pool.QueryRow(ctx, `
			select rate::float8 from public.fx_rates
			where base = $1 and quote = 'KRW' order by observed_at desc limit 1
		`, h.Currency).Scan(&fx)
	}
	h.MarketValueKRW = h.MarketValue * fx
	costKRW := h.Quantity * h.AvgCost * fx
	h.PnLKRW = h.MarketValueKRW - costKRW
	if costKRW > 0 {
		h.PnLPct = h.PnLKRW / costKRW * 100
	}
	return h
}

func pctChange(start, end float64) float64 {
	if start == 0 {
		return 0
	}
	return (end - start) / start * 100
}

// GET /v1/paper/transactions
func (h *PaperHandler) ListTx(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 50
	}
	var before *time.Time
	if s := r.URL.Query().Get("before"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			before = &t
		}
	}
	var txs []models.PaperTransaction
	var hasMore bool
	err := h.run(r.Context(), func(exec db.Executor) error {
		ts, more, err := h.repo.ListTransactions(r.Context(), exec, uid, limit, before)
		if err != nil {
			return err
		}
		txs = ts
		hasMore = more
		return nil
	})
	if err != nil {
		slog.Error("paper list tx failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if txs == nil {
		txs = []models.PaperTransaction{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"transactions": txs, "has_more": hasMore})
}

// POST /v1/paper/reset
func (h *PaperHandler) Reset(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var body models.PaperResetRequest
	_ = json.NewDecoder(r.Body).Decode(&body)
	initialCash := 10000000.0
	if body.InitialCash != nil {
		if *body.InitialCash < 0 || *body.InitialCash > 1e15 {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "initial_cash out of range")
			return
		}
		initialCash = *body.InitialCash
	}

	var account *models.PaperAccount
	err := h.run(r.Context(), func(exec db.Executor) error {
		if _, err := h.repo.GetOrCreateAccount(r.Context(), exec, uid); err != nil {
			return err
		}
		if err := h.repo.InactivateAllTransactions(r.Context(), exec, uid); err != nil {
			return err
		}
		if err := h.repo.DeleteAllHoldings(r.Context(), exec, uid); err != nil {
			return err
		}
		a, err := h.repo.ResetAccount(r.Context(), exec, uid, initialCash)
		if err != nil {
			return err
		}
		account = a
		return nil
	})
	if err != nil {
		slog.Error("paper reset failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "reset failed")
		return
	}
	writeJSON(w, http.StatusOK, account)
}
