package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
)

type ProfileRepo interface {
	Get(ctx context.Context, exec db.Executor, uid string) (map[string]any, error)
	Update(ctx context.Context, exec db.Executor, uid string, patch map[string]any) (map[string]any, error)
}

// txRunner는 핸들러가 사용하는 트랜잭션 wrap 추상화.
// production: db.AsUser(pool, uid, fn). test: passthrough(nil exec).
type txRunner func(ctx context.Context, fn func(db.Executor) error) error

type ProfileHandler struct {
	repo ProfileRepo
	run  txRunner
}

// NewProfileHandler는 production용. pool == nil이면 test passthrough.
func NewProfileHandler(repo ProfileRepo, pool *pgxpool.Pool) *ProfileHandler {
	h := &ProfileHandler{repo: repo}
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

func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	var out map[string]any
	err := h.run(r.Context(), func(exec db.Executor) error {
		p, err := h.repo.Get(r.Context(), exec, uid)
		if err != nil {
			return err
		}
		out = p
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "profile not found")
			return
		}
		slog.Error("profile get failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "failed to load profile")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *ProfileHandler) Patch(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
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
	var updated map[string]any
	err := h.run(r.Context(), func(exec db.Executor) error {
		u, err := h.repo.Update(r.Context(), exec, uid, patch)
		if err != nil {
			return err
		}
		updated = u
		return nil
	})
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
