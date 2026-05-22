package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

type SearchResult struct {
	ID       string `json:"id"`
	Symbol   string `json:"symbol"`
	Exchange string `json:"exchange"`
	Name     string `json:"name"`
}

type InstrumentRepo interface {
	SearchByAlias(ctx context.Context, query string) ([]SearchResult, error)
	SearchByText(ctx context.Context, query string) ([]SearchResult, error)
	LearnAlias(ctx context.Context, alias, instrumentID string) error
}

type InstrumentHandler struct {
	repo InstrumentRepo
}

func NewInstrumentHandler(repo InstrumentRepo) *InstrumentHandler {
	return &InstrumentHandler{repo: repo}
}

// GET /v1/instruments/search?q=
func (h *InstrumentHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusOK, []SearchResult{})
		return
	}

	// 1차: alias 정확 매칭
	results, err := h.repo.SearchByAlias(r.Context(), q)
	if err != nil {
		slog.Error("alias search failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "search failed")
		return
	}

	// 2차: name/symbol ILIKE (alias 매칭 없을 때만)
	if len(results) == 0 {
		results, err = h.repo.SearchByText(r.Context(), q)
		if err != nil {
			slog.Error("text search failed", "err", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "search failed")
			return
		}
	}

	if results == nil {
		results = []SearchResult{}
	}
	writeJSON(w, http.StatusOK, results)
}

// POST /v1/instruments/select {query, instrument_id} → alias 학습
func (h *InstrumentHandler) Select(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var body struct {
		Query        string `json:"query"`
		InstrumentID string `json:"instrument_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if strings.TrimSpace(body.Query) == "" || strings.TrimSpace(body.InstrumentID) == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "query and instrument_id required")
		return
	}
	if err := h.repo.LearnAlias(r.Context(), body.Query, body.InstrumentID); err != nil {
		slog.Error("learn alias failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "learn failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"learned": true})
}
