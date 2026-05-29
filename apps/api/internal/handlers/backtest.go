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
	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

// BacktestRunner는 portfolio.BacktestService의 인터페이스 (테스트 fake용).
type BacktestRunner interface {
	Run(ctx context.Context, pool db.Executor, req portfolio.BacktestRequest) (*portfolio.BacktestResult, error)
}

type BacktestHandler struct {
	svc  BacktestRunner
	pool *pgxpool.Pool
}

// NewBacktestHandler — pool은 슈퍼유저 공개 read용. 백테스트는 사용자 데이터 미접근 → txRunner 불필요.
func NewBacktestHandler(svc BacktestRunner, pool *pgxpool.Pool) *BacktestHandler {
	return &BacktestHandler{svc: svc, pool: pool}
}

// POST /v1/backtest/run
func (h *BacktestHandler) Run(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	var req portfolio.BacktestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if msg := validateBacktestReq(req); msg != "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", msg)
		return
	}

	result, err := h.svc.Run(r.Context(), h.pool, req)
	if err != nil {
		var ve *portfolio.ValidationError
		if errors.As(err, &ve) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error": map[string]any{"code": ve.Code, "message": ve.Message},
			})
			return
		}
		var ie *portfolio.InsufficientDataError
		if errors.As(err, &ie) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error": map[string]any{
					"code":         "INSUFFICIENT_DATA",
					"reason":       ie.Reason,
					"message":      "기간이 너무 짧습니다 (최소 30영업일)",
					"min_days":     ie.MinDays,
					"current_days": ie.CurrentDays,
				},
			})
			return
		}
		slog.Error("backtest run failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "backtest failed")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// validateBacktestReq — 구조적 검증만(요청 형태). 빈 문자열 반환 = 통과.
func validateBacktestReq(req portfolio.BacktestRequest) string {
	if len(req.Basket) == 0 || len(req.Basket) > 10 {
		return "종목을 1~10개 선택하세요"
	}
	if req.InitialCash <= 0 {
		return "초기 자금은 0보다 커야 합니다"
	}
	if req.Monthly < 0 {
		return "월 적립금은 0 이상이어야 합니다"
	}
	for _, b := range req.Basket {
		if b.InstrumentID == "" {
			return "종목을 찾을 수 없습니다"
		}
		if b.Weight <= 0 {
			return "비중은 0보다 커야 합니다"
		}
	}
	return ""
}
