package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/quotient/quotient/apps/api/internal/handlers"
)

func New() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/healthz", handlers.Healthz)
	r.Get("/readyz", handlers.Readyz)
	return r
}
