package handlers

import (
	"context"
	"log/slog"
	"net/http"
)

// TickerItem은 헤더 티커용 한 행.
type TickerItem struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	ChangePct float64 `json:"change_pct"`
}

type MarketRepo interface {
	TickerSeed(ctx context.Context) ([]TickerItem, error)
}

type MarketHandler struct {
	repo MarketRepo
}

func NewMarketHandler(repo MarketRepo) *MarketHandler {
	return &MarketHandler{repo: repo}
}

// GET /v1/market/ticker → 시드 지수·환율 (KOSPI, SPX, USD_KRW)
func (h *MarketHandler) Ticker(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.TickerSeed(r.Context())
	if err != nil {
		slog.Error("ticker fetch failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "ticker fetch failed")
		return
	}
	if items == nil {
		items = []TickerItem{} // null이 아니라 []로 직렬화
	}
	writeJSON(w, http.StatusOK, items)
}
