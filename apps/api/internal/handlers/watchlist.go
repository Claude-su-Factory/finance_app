package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
)

type WatchlistHandler struct {
	repo WatchlistRepo
}

func NewWatchlistHandler(repo WatchlistRepo) *WatchlistHandler {
	return &WatchlistHandler{repo: repo}
}

func (h *WatchlistHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	items, err := h.repo.List(r.Context(), uid)
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
	if err := h.repo.Add(r.Context(), uid, body.InstrumentID); err != nil {
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
	if err := h.repo.Remove(r.Context(), uid, iid); err != nil {
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
