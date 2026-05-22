package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/quotient/quotient/apps/api/internal/auth"
	"github.com/quotient/quotient/apps/api/internal/handlers"
	"github.com/quotient/quotient/apps/api/internal/middleware"
)

func New(
	verifier *auth.Verifier,
	corsOrigin string,
	profileHandler *handlers.ProfileHandler,
	marketHandler *handlers.MarketHandler,
	instrumentHandler *handlers.InstrumentHandler,
	readyz http.HandlerFunc,
) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.CORS(corsOrigin))
	r.Get("/healthz", handlers.Healthz)
	r.Get("/readyz", readyz)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(verifier))
		r.Get("/v1/profile", profileHandler.Get)
		r.Patch("/v1/profile", profileHandler.Patch)
		r.Get("/v1/market/ticker", marketHandler.Ticker)
		r.Get("/v1/instruments/search", instrumentHandler.Search)
		r.Post("/v1/instruments/select", instrumentHandler.Select)
	})

	return r
}
