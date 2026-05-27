package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
)

type HoldingHandler struct {
	repo HoldingRepo
	pool *pgxpool.Pool // FX 환율 + asset_class 가드용
	run  txRunner
}

func NewHoldingHandler(repo HoldingRepo, pool *pgxpool.Pool) *HoldingHandler {
	h := &HoldingHandler{repo: repo, pool: pool}
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

// GET /v1/holdings
func (h *HoldingHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	var items []models.HoldingEnriched
	err := h.run(r.Context(), func(exec db.Executor) error {
		got, err := h.repo.List(r.Context(), exec, uid)
		if err != nil {
			return err
		}
		items = got
		return nil
	})
	if err != nil {
		slog.Error("holdings list failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}

	// FX 환산 + 평가액·수익률 계산 — 슈퍼유저 풀(공개 데이터)
	rates, err := FetchFXRates(r.Context(), h.pool)
	if err != nil {
		slog.Error("fx rates fetch failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "fx load failed")
		return
	}
	enriched := enrichHoldings(items, rates)

	if enriched == nil {
		enriched = []models.HoldingEnriched{}
	}
	writeJSON(w, http.StatusOK, enriched)
}

// POST /v1/holdings
func (h *HoldingHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var body struct {
		InstrumentID string  `json:"instrument_id"`
		Quantity     float64 `json:"quantity"`
		AvgCost      float64 `json:"avg_cost"`
		OpenedAt     *string `json:"opened_at,omitempty"` // "YYYY-MM-DD"
		Note         *string `json:"note,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if body.InstrumentID == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "instrument_id required")
		return
	}
	if body.Quantity <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "quantity must be > 0")
		return
	}
	if body.AvgCost < 0 {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "avg_cost must be >= 0")
		return
	}
	// asset_class 가드: INDEX·FX·CASH instrument는 holdings 대상이 아님 (W3 MVP).
	// instruments는 공개 데이터 — 슈퍼유저 풀로 직접 조회.
	if h.pool != nil {
		var assetClass string
		if err := h.pool.QueryRow(r.Context(), `select asset_class from public.instruments where id = $1`, body.InstrumentID).Scan(&assetClass); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "instrument not found")
			return
		}
		if assetClass == "INDEX" || assetClass == "FX" || assetClass == "CASH" {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "asset_class not supported for holdings: "+assetClass)
			return
		}
	}
	var out *models.Holding
	err := h.run(r.Context(), func(exec db.Executor) error {
		o, err := h.repo.Create(r.Context(), exec, uid, body.InstrumentID, body.Quantity, body.AvgCost, body.OpenedAt, body.Note)
		if err != nil {
			return err
		}
		out = o
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrHoldingConflict) {
			writeError(w, http.StatusConflict, "CONFLICT", "holding already exists for this instrument; use PATCH to update")
			return
		}
		slog.Error("holding create failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "create failed")
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// PATCH /v1/holdings/{id}
func (h *HoldingHandler) Patch(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "id required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var patch map[string]any
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if v, ok := patch["quantity"].(float64); ok && v <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "quantity must be > 0")
		return
	}
	if v, ok := patch["avg_cost"].(float64); ok && v < 0 {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "avg_cost must be >= 0")
		return
	}
	var out *models.Holding
	err := h.run(r.Context(), func(exec db.Executor) error {
		o, err := h.repo.Update(r.Context(), exec, uid, id, patch)
		if err != nil {
			return err
		}
		out = o
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrHoldingNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "holding not found")
			return
		}
		slog.Error("holding update failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "update failed")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// DELETE /v1/holdings/{id}
func (h *HoldingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "id required")
		return
	}
	err := h.run(r.Context(), func(exec db.Executor) error {
		return h.repo.Delete(r.Context(), exec, uid, id)
	})
	if err != nil {
		if errors.Is(err, ErrHoldingNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "holding not found")
			return
		}
		slog.Error("holding delete failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "delete failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// enrichHoldings는 raw List 결과에 KRW 환산·수익률·비중을 채워 반환.
func enrichHoldings(items []models.HoldingEnriched, rates map[string]float64) []models.HoldingEnriched {
	totalKRW := 0.0
	for i := range items {
		mv := items[i].Quantity * items[i].CurrentPrice
		cb := items[i].Quantity * items[i].AvgCost
		items[i].MarketValue = mv
		items[i].MarketValueKRW = ToKRW(mv, items[i].Currency, rates)
		items[i].CostBasisKRW = ToKRW(cb, items[i].Currency, rates)
		if cb > 0 && items[i].CurrentPrice > 0 {
			items[i].PnLPct = (mv - cb) / cb * 100.0
		}
		items[i].PnLKRW = items[i].MarketValueKRW - items[i].CostBasisKRW
		totalKRW += items[i].MarketValueKRW
	}
	if totalKRW > 0 {
		for i := range items {
			items[i].WeightPct = items[i].MarketValueKRW / totalKRW * 100.0
		}
	}
	return items
}
