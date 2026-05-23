package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
)

type fakeWatchlistRepo struct {
	addErr    error
	removeErr error
}

func (f *fakeWatchlistRepo) List(ctx context.Context, userID string) ([]models.WatchlistItem, error) {
	return []models.WatchlistItem{{InstrumentID: "iid-1", Symbol: "KOSPI"}}, nil
}
func (f *fakeWatchlistRepo) Add(ctx context.Context, userID, instrumentID string) error {
	return f.addErr
}
func (f *fakeWatchlistRepo) Remove(ctx context.Context, userID, instrumentID string) error {
	return f.removeErr
}

func TestWatchlistAdd_Conflict(t *testing.T) {
	repo := &fakeWatchlistRepo{addErr: ErrWatchlistConflict}
	h := NewWatchlistHandler(repo)
	body := `{"instrument_id":"abc"}`
	r := httptest.NewRequest(http.MethodPost, "/v1/watchlist", strings.NewReader(body))
	r = r.WithContext(middleware.WithUserID(r.Context(), "user-1"))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.Add(w, r)
	if w.Code != http.StatusConflict {
		t.Errorf("got %d want 409", w.Code)
	}
}

func TestWatchlistRemove_NotFound(t *testing.T) {
	repo := &fakeWatchlistRepo{removeErr: ErrWatchlistNotFound}
	h := NewWatchlistHandler(repo)
	r := httptest.NewRequest(http.MethodDelete, "/v1/watchlist/iid-1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("instrument_id", "iid-1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	r = r.WithContext(middleware.WithUserID(r.Context(), "user-1"))

	w := httptest.NewRecorder()
	h.Remove(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("got %d want 404", w.Code)
	}
}
