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

type WatchlistHandler struct {
	repo WatchlistRepo
	pool *pgxpool.Pool
	run  txRunner
}

func NewWatchlistHandler(repo WatchlistRepo, pool *pgxpool.Pool) *WatchlistHandler {
	h := &WatchlistHandler{repo: repo, pool: pool}
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

func (h *WatchlistHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	var items []models.WatchlistItem
	err := h.run(r.Context(), func(exec db.Executor) error {
		got, err := h.repo.List(r.Context(), exec, uid)
		if err != nil {
			return err
		}
		items = got
		return nil
	})
	if err != nil {
		slog.Error("watchlist list failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if items == nil {
		items = []models.WatchlistItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *WatchlistHandler) Add(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var body struct {
		InstrumentID string `json:"instrument_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if body.InstrumentID == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "instrument_id required")
		return
	}
	// asset_class 가드 — instruments는 공개 데이터, 슈퍼유저 풀로 조회.
	if h.pool != nil {
		var assetClass string
		if err := h.pool.QueryRow(r.Context(), `select asset_class from public.instruments where id = $1`, body.InstrumentID).Scan(&assetClass); err != nil {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "instrument not found")
			return
		}
		if assetClass == "INDEX" || assetClass == "FX" || assetClass == "CASH" {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "asset_class not supported for watchlist: "+assetClass)
			return
		}
	}
	err := h.run(r.Context(), func(exec db.Executor) error {
		return h.repo.Add(r.Context(), exec, uid, body.InstrumentID)
	})
	if err != nil {
		if errors.Is(err, ErrWatchlistConflict) {
			writeError(w, http.StatusConflict, "CONFLICT", "already in watchlist")
			return
		}
		if errors.Is(err, ErrInstrumentRefMissing) {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "instrument not found")
			return
		}
		slog.Error("watchlist add failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "add failed")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *WatchlistHandler) Remove(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	iid := chi.URLParam(r, "instrument_id")
	if iid == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "instrument_id required")
		return
	}
	err := h.run(r.Context(), func(exec db.Executor) error {
		return h.repo.Remove(r.Context(), exec, uid, iid)
	})
	if err != nil {
		if errors.Is(err, ErrWatchlistNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "not in watchlist")
			return
		}
		slog.Error("watchlist remove failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "remove failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
