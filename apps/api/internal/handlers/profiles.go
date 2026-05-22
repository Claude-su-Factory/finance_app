package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/quotient/quotient/apps/api/internal/middleware"
)

type ProfileRepo interface {
	Get(ctx context.Context, userID string) (map[string]any, error)
	Update(ctx context.Context, userID string, patch map[string]any) (map[string]any, error)
}

type ProfileHandler struct {
	repo ProfileRepo
}

func NewProfileHandler(repo ProfileRepo) *ProfileHandler {
	return &ProfileHandler{repo: repo}
}

func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	p, err := h.repo.Get(r.Context(), uid)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "profile not found")
			return
		}
		slog.Error("profile get failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to load profile")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *ProfileHandler) Patch(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096) // 4KB max for profile patch
	var patch map[string]any
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if v, ok := patch["base_currency"].(string); ok && v != "KRW" && v != "USD" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "base_currency must be KRW or USD")
		return
	}
	if v, ok := patch["ui_intensity"].(string); ok && v != "vivid" && v != "standard" && v != "subtle" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "ui_intensity invalid")
		return
	}
	updated, err := h.repo.Update(r.Context(), uid, patch)
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "profile not found")
			return
		}
		slog.Error("profile update failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to update profile")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": msg},
	})
}
