package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/ai"
	"github.com/quotient/quotient/apps/api/internal/ai/tools"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
)

const (
	maxJournalContent   = 2000
	maxJournalSymbols   = 10
	maxJournalSymbolLen = 20
	maxJournalTitle     = 100
)

type JournalHandler struct {
	repo     JournalRepo
	chatRepo ChatRepo // 사용량 한도 차감 + 체크
	pool     *pgxpool.Pool
	client   ai.Client // analyze 도구 LLM 호출
	registry *tools.Registry
	run      txRunner
}

func NewJournalHandler(repo JournalRepo, chatRepo ChatRepo, pool *pgxpool.Pool, client ai.Client, registry *tools.Registry) *JournalHandler {
	h := &JournalHandler{repo: repo, chatRepo: chatRepo, pool: pool, client: client, registry: registry}
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

// GET /v1/journal/entries
func (h *JournalHandler) List(w http.ResponseWriter, r *http.Request) {
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
	var entries []models.JournalEntry
	var hasMore bool
	err := h.run(r.Context(), func(exec db.Executor) error {
		es, more, err := h.repo.ListEntries(r.Context(), exec, uid, limit, before)
		if err != nil {
			return err
		}
		entries = es
		hasMore = more
		return nil
	})
	if err != nil {
		slog.Error("journal list failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if entries == nil {
		entries = []models.JournalEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries, "has_more": hasMore})
}

// POST /v1/journal/entries — manual entry
func (h *JournalHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	var body models.JournalEntryCreate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if err := validateEntryCreate(&body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	var out *models.JournalEntry
	err := h.run(r.Context(), func(exec db.Executor) error {
		e, err := h.repo.CreateEntry(r.Context(), exec, uid, "manual", body, nil)
		if err != nil {
			return err
		}
		out = e
		return nil
	})
	if err != nil {
		slog.Error("journal create failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "create failed")
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// PATCH /v1/journal/entries/{id}
func (h *JournalHandler) Patch(w http.ResponseWriter, r *http.Request) {
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
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	var body models.JournalEntryPatch
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if err := validateEntryPatch(&body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	var out *models.JournalEntry
	err := h.run(r.Context(), func(exec db.Executor) error {
		e, err := h.repo.UpdateEntry(r.Context(), exec, uid, id, body)
		if err != nil {
			return err
		}
		out = e
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrJournalEntryNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "entry not found")
			return
		}
		if errors.Is(err, ErrJournalEntryNotMutable) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error": map[string]any{
					"code":    "CANNOT_MODIFY_AUTO",
					"message": "자동 기록은 수정할 수 없습니다",
				},
			})
			return
		}
		slog.Error("journal patch failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "update failed")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// DELETE /v1/journal/entries/{id}
func (h *JournalHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	id := chi.URLParam(r, "id")
	err := h.run(r.Context(), func(exec db.Executor) error {
		return h.repo.DeleteEntry(r.Context(), exec, uid, id)
	})
	if err != nil {
		if errors.Is(err, ErrJournalEntryNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "entry not found")
			return
		}
		slog.Error("journal delete failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "delete failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /v1/journal/analyze — on-demand
func (h *JournalHandler) Analyze(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var body struct {
		PeriodDays int `json:"period_days"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.PeriodDays <= 0 {
		body.PeriodDays = 90
	}
	if body.PeriodDays < 7 || body.PeriodDays > 365 {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "period_days must be 7..365")
		return
	}

	// 사용량 한도 체크 (chat 한도와 통합)
	if err := h.run(r.Context(), func(exec db.Executor) error {
		return h.chatRepo.CheckLimits(r.Context(), exec, uid, false)
	}); err != nil {
		if errors.Is(err, ErrUsageExceeded) {
			writeJSON(w, http.StatusTooManyRequests, map[string]any{
				"error": map[string]any{
					"code":    "USAGE_EXCEEDED",
					"reason":  "monthly_chat_limit",
					"message": "월 30회 한도 도달",
				},
			})
			return
		}
		slog.Error("journal analyze limit check failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "check failed")
		return
	}

	// LLM 도구 호출 (tools.ExecuteAndSerialize)
	result := tools.ExecuteAndSerialize(r.Context(), h.registry, h.pool, "analyze_journal", uid, map[string]any{
		"period_days": body.PeriodDays,
	})

	// result 파싱 — 도구가 JSON 응답
	var toolOut struct {
		EntriesCount int    `json:"entries_count"`
		ContentMD    string `json:"content_md"`
		Model        string `json:"model"`
		Error        string `json:"error,omitempty"`
	}
	_ = json.Unmarshal([]byte(result), &toolOut)
	if toolOut.Error != "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error": map[string]any{
				"code":    "INSUFFICIENT_DATA",
				"reason":  "no_entries",
				"message": toolOut.Error,
			},
		})
		return
	}

	// 한도 차감 + analysis_runs 저장 (같은 트랜잭션)
	periodEnd := time.Now()
	periodStart := periodEnd.AddDate(0, 0, -body.PeriodDays)
	var run *models.AnalysisRun
	err := h.run(r.Context(), func(exec db.Executor) error {
		if err := h.chatRepo.IncrementUsage(r.Context(), exec, uid, 0, 0, false); err != nil {
			return err
		}
		a, err := h.repo.InsertAnalysis(r.Context(), exec, uid, "on_demand",
			periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02"),
			toolOut.EntriesCount, toolOut.ContentMD, toolOut.Model)
		if err != nil {
			return err
		}
		run = a
		return nil
	})
	if err != nil {
		slog.Error("journal analyze persist failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "persist failed")
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// GET /v1/journal/analyses
func (h *JournalHandler) ListAnalyses(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	var out []models.AnalysisRun
	err := h.run(r.Context(), func(exec db.Executor) error {
		as, err := h.repo.ListAnalyses(r.Context(), exec, uid, limit)
		if err != nil {
			return err
		}
		out = as
		return nil
	})
	if err != nil {
		slog.Error("journal analyses list failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if out == nil {
		out = []models.AnalysisRun{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"analyses": out})
}

// --- validation ---

func validateEntryCreate(b *models.JournalEntryCreate) error {
	if len(b.Content) < 1 || len(b.Content) > maxJournalContent {
		return errors.New("content length must be 1..2000")
	}
	if len(b.RelatedSymbols) > maxJournalSymbols {
		return errors.New("too many related_symbols (max 10)")
	}
	for _, s := range b.RelatedSymbols {
		if len(s) > maxJournalSymbolLen {
			return errors.New("symbol too long (max 20)")
		}
	}
	if b.Title != nil && len(*b.Title) > maxJournalTitle {
		return errors.New("title too long (max 100)")
	}
	if b.Action != nil {
		switch *b.Action {
		case "buy", "sell", "observation", "other":
		default:
			return errors.New("invalid action")
		}
	}
	return nil
}

func validateEntryPatch(b *models.JournalEntryPatch) error {
	if b.Content != nil && (len(*b.Content) < 1 || len(*b.Content) > maxJournalContent) {
		return errors.New("content length must be 1..2000")
	}
	if b.RelatedSymbols != nil {
		if len(*b.RelatedSymbols) > maxJournalSymbols {
			return errors.New("too many related_symbols")
		}
		for _, s := range *b.RelatedSymbols {
			if len(s) > maxJournalSymbolLen {
				return errors.New("symbol too long")
			}
		}
	}
	if b.Title != nil && len(*b.Title) > maxJournalTitle {
		return errors.New("title too long")
	}
	if b.Action != nil {
		switch *b.Action {
		case "buy", "sell", "observation", "other":
		default:
			return errors.New("invalid action")
		}
	}
	return nil
}
