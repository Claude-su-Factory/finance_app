package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/quotient/quotient/apps/api/internal/auth"
	"github.com/quotient/quotient/apps/api/internal/handlers"
	"github.com/quotient/quotient/apps/api/internal/middleware"
)

func New(verifier *auth.Verifier, corsOrigin string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.CORS(corsOrigin))
	r.Get("/healthz", handlers.Healthz)
	r.Get("/readyz", handlers.Readyz)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(verifier))
		// /v1 라우트는 Task 6에서 추가
	})

	return r
}
