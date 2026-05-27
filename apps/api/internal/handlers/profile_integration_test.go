//go:build integration

package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/handlers"
	"github.com/quotient/quotient/apps/api/internal/middleware"
)

// 로컬 Supabase 위에서 profile 핸들러의 GET/PATCH를 종단 검증.
// 사전 조건:
// - `supabase start` 실행 + 마이그레이션 적용된 상태
// - TEST_DATABASE_URL env 설정 (예: postgresql://postgres:postgres@127.0.0.1:54322/postgres)
// auth.users INSERT → handle_new_user 트리거가 public.profiles 자동 생성.

func openPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func seedUser(t *testing.T, pool *pgxpool.Pool, uid string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		insert into auth.users (id, email, encrypted_password)
		values ($1::uuid, $1::text || '@test.local', '')
		on conflict (id) do nothing
	`, uid)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func cleanupUser(pool *pgxpool.Pool, uid string) {
	_, _ = pool.Exec(context.Background(), `delete from auth.users where id = $1`, uid)
}

func TestProfileGet_Integration_ReturnsAutoCreatedRow(t *testing.T) {
	pool := openPool(t)
	uid := uuid.NewString()
	seedUser(t, pool, uid)
	defer cleanupUser(pool, uid)

	h := handlers.NewProfileHandler(handlers.NewPgProfileRepo(), pool)

	req := httptest.NewRequest(http.MethodGet, "/v1/profile", nil)
	req = req.WithContext(middleware.WithUserID(req.Context(), uid))
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d, body=%s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["id"] != uid {
		t.Errorf("id = %v, want %s", got["id"], uid)
	}
	if got["base_currency"] != "KRW" {
		t.Errorf("base_currency = %v, want KRW (default)", got["base_currency"])
	}
}

func TestProfilePatch_Integration_UpdatesAndPersists(t *testing.T) {
	pool := openPool(t)
	uid := uuid.NewString()
	seedUser(t, pool, uid)
	defer cleanupUser(pool, uid)

	h := handlers.NewProfileHandler(handlers.NewPgProfileRepo(), pool)

	patch, _ := json.Marshal(map[string]any{
		"display_name":           "테스트유저",
		"base_currency":          "USD",
		"ui_intensity":           "vivid",
		"onboarding_completed":   true,
		"daily_briefing_enabled": false,
	})
	req := httptest.NewRequest(http.MethodPatch, "/v1/profile", bytes.NewReader(patch))
	req = req.WithContext(middleware.WithUserID(req.Context(), uid))
	rec := httptest.NewRecorder()
	h.Patch(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("PATCH status %d, body=%s", rec.Code, rec.Body.String())
	}

	// 후속 GET이 갱신된 값을 반환하는지
	req2 := httptest.NewRequest(http.MethodGet, "/v1/profile", nil)
	req2 = req2.WithContext(middleware.WithUserID(req2.Context(), uid))
	rec2 := httptest.NewRecorder()
	h.Get(rec2, req2)

	var got map[string]any
	if err := json.Unmarshal(rec2.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got["display_name"] != "테스트유저" {
		t.Errorf("display_name = %v", got["display_name"])
	}
	if got["base_currency"] != "USD" {
		t.Errorf("base_currency = %v", got["base_currency"])
	}
	if got["ui_intensity"] != "vivid" {
		t.Errorf("ui_intensity = %v", got["ui_intensity"])
	}
	if got["onboarding_completed"] != true {
		t.Errorf("onboarding_completed = %v", got["onboarding_completed"])
	}
	if got["daily_briefing_enabled"] != false {
		t.Errorf("daily_briefing_enabled = %v", got["daily_briefing_enabled"])
	}
}

func TestProfilePatch_Integration_RejectsInvalidCurrency(t *testing.T) {
	pool := openPool(t)
	uid := uuid.NewString()
	seedUser(t, pool, uid)
	defer cleanupUser(pool, uid)

	h := handlers.NewProfileHandler(handlers.NewPgProfileRepo(), pool)

	patch, _ := json.Marshal(map[string]any{"base_currency": "EUR"})
	req := httptest.NewRequest(http.MethodPatch, "/v1/profile", bytes.NewReader(patch))
	req = req.WithContext(middleware.WithUserID(req.Context(), uid))
	rec := httptest.NewRecorder()
	h.Patch(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status %d, want 422", rec.Code)
	}
}

func TestProfileGet_Integration_RLSIsolation(t *testing.T) {
	// 사용자 A의 핸들러로 사용자 B의 ctx를 보내면 — handler는 uid 기반으로 RLS 트랜잭션을 열고
	// 사용자 B의 profile만 보여야 한다(섞이지 않음).
	pool := openPool(t)
	userA := uuid.NewString()
	userB := uuid.NewString()
	seedUser(t, pool, userA)
	seedUser(t, pool, userB)
	defer cleanupUser(pool, userA)
	defer cleanupUser(pool, userB)

	h := handlers.NewProfileHandler(handlers.NewPgProfileRepo(), pool)

	// 사용자 A로 PATCH
	patch, _ := json.Marshal(map[string]any{"display_name": "A user"})
	reqA := httptest.NewRequest(http.MethodPatch, "/v1/profile", bytes.NewReader(patch))
	reqA = reqA.WithContext(middleware.WithUserID(reqA.Context(), userA))
	recA := httptest.NewRecorder()
	h.Patch(recA, reqA)
	if recA.Code != http.StatusOK {
		t.Fatalf("user A patch: %d, body=%s", recA.Code, recA.Body.String())
	}

	// 사용자 B의 ctx로 GET — A의 display_name이 보이면 안 됨
	reqB := httptest.NewRequest(http.MethodGet, "/v1/profile", nil)
	reqB = reqB.WithContext(middleware.WithUserID(reqB.Context(), userB))
	recB := httptest.NewRecorder()
	h.Get(recB, reqB)
	if recB.Code != http.StatusOK {
		t.Fatalf("user B get: %d", recB.Code)
	}
	var gotB map[string]any
	_ = json.Unmarshal(recB.Body.Bytes(), &gotB)
	if gotB["display_name"] == "A user" {
		t.Errorf("RLS leak: user B saw user A's display_name")
	}
	if gotB["id"] != userB {
		t.Errorf("user B got id %v, want %s", gotB["id"], userB)
	}
}
