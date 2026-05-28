package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

// AlphaComputer는 portfolio.Service의 인터페이스 (테스트 fake용).
type AlphaComputer interface {
	Compute(ctx context.Context, exec db.Executor, pool db.Executor, uid string, period portfolio.Period) (*portfolio.AlphaResult, error)
}

type AlphaHandler struct {
	svc  AlphaComputer
	pool *pgxpool.Pool
	run  txRunner
}

// NewAlphaHandler는 production용. pool == nil이면 test passthrough.
func NewAlphaHandler(svc AlphaComputer, pool *pgxpool.Pool) *AlphaHandler {
	h := &AlphaHandler{svc: svc, pool: pool}
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

func (h *AlphaHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	period, err := portfolio.ParsePeriod(r.URL.Query().Get("period"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "period must be 1m|90d|1y|all")
		return
	}

	var result *portfolio.AlphaResult
	err = h.run(r.Context(), func(exec db.Executor) error {
		res, e := h.svc.Compute(r.Context(), exec, h.pool, uid, period)
		if e != nil {
			return e
		}
		result = res
		return nil
	})
	if err != nil {
		var ie *portfolio.InsufficientDataError
		if errors.As(err, &ie) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error": map[string]any{
					"code":         "INSUFFICIENT_DATA",
					"reason":       ie.Reason,
					"message":      insufficientMessage(ie),
					"min_days":     ie.MinDays,
					"current_days": ie.CurrentDays,
				},
			})
			return
		}
		slog.Error("alpha compute failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "compute failed")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func insufficientMessage(e *portfolio.InsufficientDataError) string {
	switch e.Reason {
	case "account_too_young":
		return "7일 이상 보유 후 표시됩니다"
	case "no_holdings":
		return "보유 자산 추가 후 표시됩니다"
	}
	return "데이터 부족"
}
