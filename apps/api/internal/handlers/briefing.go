package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
)

var ErrBriefingNotFound = errors.New("briefing not found")

type BriefingHandler struct {
	pool *pgxpool.Pool
}

func NewBriefingHandler(pool *pgxpool.Pool) *BriefingHandler {
	return &BriefingHandler{pool: pool}
}

// GET /v1/briefings/today
func (h *BriefingHandler) Today(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	loc, _ := time.LoadLocation("Asia/Seoul")
	today := time.Now().In(loc).Format("2006-01-02")
	b, err := loadBriefing(r.Context(), h.pool, uid, today)
	if err != nil {
		if errors.Is(err, ErrBriefingNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "no briefing for today")
			return
		}
		slog.Error("briefing load failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func loadBriefing(ctx context.Context, pool *pgxpool.Pool, uid, date string) (*models.AIBriefing, error) {
	var b models.AIBriefing
	row := pool.QueryRow(ctx, `
		select user_id::text, date::text, content_md, model, created_at
		from public.ai_briefings where user_id = $1 and date = $2
	`, uid, date)
	if err := row.Scan(&b.UserID, &b.Date, &b.ContentMD, &b.Model, &b.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBriefingNotFound
		}
		return nil, err
	}
	return &b, nil
}
