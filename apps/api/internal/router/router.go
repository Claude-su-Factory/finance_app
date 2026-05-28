package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/quotient/quotient/apps/api/internal/auth"
	"github.com/quotient/quotient/apps/api/internal/handlers"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/observability"
)

func New(
	verifier *auth.Verifier,
	corsOrigin string,
	profileHandler *handlers.ProfileHandler,
	marketHandler *handlers.MarketHandler,
	instrumentHandler *handlers.InstrumentHandler,
	holdingHandler *handlers.HoldingHandler,
	watchlistHandler *handlers.WatchlistHandler,
	chatHandler *handlers.ChatHandler,
	briefingHandler *handlers.BriefingHandler,
	historyHandler *handlers.HistoryHandler,
	readyz http.HandlerFunc,
	alphaHandler *handlers.AlphaHandler,
) *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(observability.SentryMiddleware())
	r.Use(chimw.Recoverer)
	r.Use(middleware.CORS(corsOrigin))
	r.Get("/healthz", handlers.Healthz)
	r.Get("/readyz", readyz)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(verifier))
		r.Get("/v1/profile", profileHandler.Get)
		r.Patch("/v1/profile", profileHandler.Patch)
		r.Get("/v1/portfolio/alpha", alphaHandler.Get)
		r.Get("/v1/market/ticker", marketHandler.Ticker)
		r.Get("/v1/instruments/search", instrumentHandler.Search)
		r.Post("/v1/instruments/select", instrumentHandler.Select)

		r.Get("/v1/holdings", holdingHandler.List)
		r.Post("/v1/holdings", holdingHandler.Create)
		r.Patch("/v1/holdings/{id}", holdingHandler.Patch)
		r.Delete("/v1/holdings/{id}", holdingHandler.Delete)

		r.Get("/v1/watchlist", watchlistHandler.List)
		r.Post("/v1/watchlist", watchlistHandler.Add)
		r.Delete("/v1/watchlist/{instrument_id}", watchlistHandler.Remove)

		r.Post("/v1/chat", chatHandler.StreamChat)
		r.Get("/v1/chat/sessions", chatHandler.ListSessions)
		r.Delete("/v1/chat/sessions/{id}", chatHandler.DeleteSession)
		r.Get("/v1/chat/sessions/{id}/messages", chatHandler.ListMessages)
		r.Get("/v1/chat/sessions/{id}/unfinished", chatHandler.GetUnfinished)
		r.Get("/v1/chat/usage", chatHandler.GetUsage)
		r.Get("/v1/briefings/today", briefingHandler.Today)
		r.Get("/v1/prices/history", historyHandler.Prices)
		r.Get("/v1/indicators/history", historyHandler.Indicators)
		r.Get("/v1/fx/history", historyHandler.Fx)
	})

	return r
}
