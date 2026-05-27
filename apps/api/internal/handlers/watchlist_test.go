package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
)

type fakeWatchlistRepo struct {
	addErr    error
	removeErr error
}

func (f *fakeWatchlistRepo) List(ctx context.Context, exec db.Executor, userID string) ([]models.WatchlistItem, error) {
	_ = exec
	return []models.WatchlistItem{{InstrumentID: "iid-1", Symbol: "KOSPI"}}, nil
}
func (f *fakeWatchlistRepo) Add(ctx context.Context, exec db.Executor, userID, instrumentID string) error {
	_ = exec
	return f.addErr
}
func (f *fakeWatchlistRepo) Remove(ctx context.Context, exec db.Executor, userID, instrumentID string) error {
	_ = exec
	return f.removeErr
}

// TestWatchlistAdd_ValidationBeforeAssetClassGuard: asset_class 가드는 DB 호출 필요라
// fake repo + nil pool로는 도달 불가. 여기선 가드 이전 검증 단계만 테스트.
// holdings_test.go(W3-T5)와 동일 패턴.
func TestWatchlistAdd_ValidationBeforeAssetClassGuard(t *testing.T) {
	repo := &fakeWatchlistRepo{}
	h := NewWatchlistHandler(repo, nil)

	cases := []struct {
		name string
		body string
		want int
	}{
		{"missing instrument_id", `{}`, http.StatusUnprocessableEntity},
		{"no auth", `{"instrument_id":"abc"}`, http.StatusUnauthorized},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var r *http.Request
			if c.name == "no auth" {
				r = httptest.NewRequest(http.MethodPost, "/v1/watchlist", strings.NewReader(c.body))
			} else {
				r = httptest.NewRequest(http.MethodPost, "/v1/watchlist", strings.NewReader(c.body))
				r = r.WithContext(middleware.WithUserID(r.Context(), "user-1"))
			}
			r.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			h.Add(w, r)
			if w.Code != c.want {
				t.Errorf("got %d want %d, body=%s", w.Code, c.want, w.Body.String())
			}
		})
	}
}

func TestWatchlistRemove_NotFound(t *testing.T) {
	repo := &fakeWatchlistRepo{removeErr: ErrWatchlistNotFound}
	h := NewWatchlistHandler(repo, nil)
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
