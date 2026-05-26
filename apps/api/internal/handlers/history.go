package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

type HistoryHandler struct {
	repo HistoryRepo
}

func NewHistoryHandler(repo HistoryRepo) *HistoryHandler {
	return &HistoryHandler{repo: repo}
}

func (h *HistoryHandler) Prices(w http.ResponseWriter, r *http.Request) {
	rng := r.URL.Query().Get("range")
	if rng == "" {
		rng = "1mo"
	}
	if rangeToInterval(rng) == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid range")
		return
	}

	if idsStr := r.URL.Query().Get("ids"); idsStr != "" {
		ids := splitCSV(idsStr)
		if len(ids) > 100 {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "too many ids (max 100)")
			return
		}
		items, err := h.repo.PriceByIDsBatch(r.Context(), ids, rng)
		if err != nil {
			slog.Error("price batch failed", "err", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "range": rng})
		return
	}

	sym := r.URL.Query().Get("symbol")
	if sym == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "symbol or ids required")
		return
	}
	points, err := h.repo.PriceBySymbol(r.Context(), sym, rng)
	if err != nil {
		slog.Error("price history failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if points == nil {
		points = []PricePoint{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"symbol": sym, "range": rng, "points": points, "count": len(points)})
}

func (h *HistoryHandler) Indicators(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "code required")
		return
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	points, err := h.repo.Indicator(r.Context(), code, days)
	if err != nil {
		slog.Error("indicator history failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if points == nil {
		points = []IndicatorPoint{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"code": code, "points": points, "count": len(points)})
}

func (h *HistoryHandler) Fx(w http.ResponseWriter, r *http.Request) {
	base := r.URL.Query().Get("base")
	quote := r.URL.Query().Get("quote")
	if base == "" || quote == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "base and quote required")
		return
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	points, err := h.repo.Fx(r.Context(), base, quote, days)
	if err != nil {
		slog.Error("fx history failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if points == nil {
		points = []FxPoint{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"base": base, "quote": quote, "points": points, "count": len(points)})
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
