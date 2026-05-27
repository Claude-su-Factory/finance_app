package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/stretchr/testify/assert"
)

type fakeRepo struct {
	getResp map[string]any
	updErr  error
}

func (r *fakeRepo) Get(ctx context.Context, exec db.Executor, uid string) (map[string]any, error) {
	_ = exec
	return r.getResp, nil
}

func (r *fakeRepo) Update(ctx context.Context, exec db.Executor, uid string, patch map[string]any) (map[string]any, error) {
	_ = exec
	if r.updErr != nil {
		return nil, r.updErr
	}
	return r.getResp, nil
}

func reqWithUser(method, path string, body []byte, uid string) *http.Request {
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, uid)
	return req.WithContext(ctx)
}

func TestGetProfile_ReturnsCurrent(t *testing.T) {
	repo := &fakeRepo{getResp: map[string]any{
		"id":            "user-1",
		"display_name":  "Hojin",
		"base_currency": "KRW",
	}}
	h := NewProfileHandler(repo, nil)
	req := reqWithUser(http.MethodGet, "/v1/profile", nil, "user-1")
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	assert.Equal(t, "user-1", got["id"])
}

func TestGetProfile_NoUser_Unauthorized(t *testing.T) {
	h := NewProfileHandler(&fakeRepo{}, nil)
	req := httptest.NewRequest(http.MethodGet, "/v1/profile", nil)
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestPatchProfile_AcceptsValidBody(t *testing.T) {
	repo := &fakeRepo{getResp: map[string]any{"id": "user-1"}}
	h := NewProfileHandler(repo, nil)
	body, _ := json.Marshal(map[string]any{"base_currency": "USD"})
	req := reqWithUser(http.MethodPatch, "/v1/profile", body, "user-1")
	rec := httptest.NewRecorder()

	h.Patch(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestPatchProfile_RejectsInvalidCurrency(t *testing.T) {
	h := NewProfileHandler(&fakeRepo{}, nil)
	body, _ := json.Marshal(map[string]any{"base_currency": "EUR"})
	req := reqWithUser(http.MethodPatch, "/v1/profile", body, "user-1")
	rec := httptest.NewRecorder()

	h.Patch(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestPatchProfile_RejectsInvalidUIIntensity(t *testing.T) {
	h := NewProfileHandler(&fakeRepo{}, nil)
	body, _ := json.Marshal(map[string]any{"ui_intensity": "bright"})
	req := reqWithUser(http.MethodPatch, "/v1/profile", body, "user-1")
	rec := httptest.NewRecorder()

	h.Patch(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}
