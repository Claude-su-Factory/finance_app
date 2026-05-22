package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakePinger struct{ err error }

func (p fakePinger) Ping(ctx context.Context) error { return p.err }

func TestHealthz_OK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	Healthz(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
}

func TestReadyz_OK(t *testing.T) {
	h := ReadyzHandler(fakePinger{})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	h(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"status":"ready"}`, rec.Body.String())
}

func TestReadyz_DBDown(t *testing.T) {
	h := ReadyzHandler(fakePinger{err: errors.New("db down")})
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	h(rec, req)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}
