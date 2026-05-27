//go:build integration

package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/db"
)

// 본 테스트는 로컬 Supabase가 떠 있어야 한다 (`supabase start`).
// 모든 마이그레이션이 적용된 상태에서 RLS가 사용자 격리하는지 검증.
//
// 사전 조건:
// - auth.users에 직접 INSERT 시 on_auth_user_created 트리거가
//   public.profiles 행을 자동 생성한다(handle_new_user, SECURITY DEFINER).
//   따라서 테스트는 auth.users만 seed하고 profiles는 트리거에 맡긴다.
// - holdings/watchlist/chat_*는 user_id에 on delete cascade가 걸려 있어
//   auth.users DELETE 한 번으로 모두 정리된다.
func TestRLS_HoldingsIsolation(t *testing.T) {
	pool := openTestPool(t)
	defer pool.Close()

	userA := uuid.NewString()
	userB := uuid.NewString()
	mustSeedUser(t, pool, userA)
	mustSeedUser(t, pool, userB)
	defer cleanupUser(pool, userA)
	defer cleanupUser(pool, userB)

	var instID string
	if err := pool.QueryRow(context.Background(), `select id::text from public.instruments limit 1`).Scan(&instID); err != nil {
		t.Skip("no instrument seeded")
	}

	if err := db.AsUser(context.Background(), pool, userA, func(exec db.Executor) error {
		_, err := exec.Exec(context.Background(), `
			insert into public.holdings (user_id, instrument_id, quantity, avg_cost)
			values ($1, $2, 10, 50000)
		`, userA, instID)
		return err
	}); err != nil {
		t.Fatalf("user A insert: %v", err)
	}

	var bCount int
	if err := db.AsUser(context.Background(), pool, userB, func(exec db.Executor) error {
		return exec.QueryRow(context.Background(), `select count(*) from public.holdings`).Scan(&bCount)
	}); err != nil {
		t.Fatalf("user B select: %v", err)
	}
	if bCount != 0 {
		t.Fatalf("RLS leak: user B sees %d holdings of user A", bCount)
	}

	var aCount int
	if err := db.AsUser(context.Background(), pool, userA, func(exec db.Executor) error {
		return exec.QueryRow(context.Background(), `select count(*) from public.holdings`).Scan(&aCount)
	}); err != nil {
		t.Fatalf("user A select: %v", err)
	}
	if aCount != 1 {
		t.Fatalf("user A sees %d holdings, want 1", aCount)
	}
}

func TestRLS_WatchlistIsolation(t *testing.T) {
	pool := openTestPool(t)
	defer pool.Close()

	userA := uuid.NewString()
	userB := uuid.NewString()
	mustSeedUser(t, pool, userA)
	mustSeedUser(t, pool, userB)
	defer cleanupUser(pool, userA)
	defer cleanupUser(pool, userB)

	var instID string
	if err := pool.QueryRow(context.Background(), `select id::text from public.instruments limit 1`).Scan(&instID); err != nil {
		t.Skip("no instrument seeded")
	}

	if err := db.AsUser(context.Background(), pool, userA, func(exec db.Executor) error {
		_, err := exec.Exec(context.Background(), `
			insert into public.watchlist (user_id, instrument_id) values ($1, $2)
		`, userA, instID)
		return err
	}); err != nil {
		t.Fatalf("user A insert: %v", err)
	}

	var bCount int
	if err := db.AsUser(context.Background(), pool, userB, func(exec db.Executor) error {
		return exec.QueryRow(context.Background(), `select count(*) from public.watchlist`).Scan(&bCount)
	}); err != nil {
		t.Fatalf("user B select: %v", err)
	}
	if bCount != 0 {
		t.Fatalf("RLS leak: user B sees %d watchlist of user A", bCount)
	}
}

func TestRLS_ChatSessionsIsolation(t *testing.T) {
	pool := openTestPool(t)
	defer pool.Close()

	userA := uuid.NewString()
	userB := uuid.NewString()
	mustSeedUser(t, pool, userA)
	mustSeedUser(t, pool, userB)
	defer cleanupUser(pool, userA)
	defer cleanupUser(pool, userB)

	if err := db.AsUser(context.Background(), pool, userA, func(exec db.Executor) error {
		_, err := exec.Exec(context.Background(), `
			insert into public.chat_sessions (user_id, title) values ($1, 'A session')
		`, userA)
		return err
	}); err != nil {
		t.Fatalf("user A insert: %v", err)
	}

	var bCount int
	if err := db.AsUser(context.Background(), pool, userB, func(exec db.Executor) error {
		return exec.QueryRow(context.Background(), `select count(*) from public.chat_sessions`).Scan(&bCount)
	}); err != nil {
		t.Fatalf("user B select: %v", err)
	}
	if bCount != 0 {
		t.Fatalf("RLS leak: user B sees %d chat_sessions of user A", bCount)
	}
}

func openTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	return pool
}

// mustSeedUser는 슈퍼유저 풀로 auth.users 1행을 만든다.
// handle_new_user 트리거가 public.profiles 행을 자동 생성하므로 별도 INSERT 불필요.
func mustSeedUser(t *testing.T, pool *pgxpool.Pool, uid string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		insert into auth.users (id, email, encrypted_password)
		values ($1::uuid, $1::text || '@test.local', '')
		on conflict (id) do nothing
	`, uid)
	if err != nil {
		t.Fatalf("seed auth.users: %v", err)
	}
}

// cleanupUser: auth.users 삭제 → FK on delete cascade로 holdings/watchlist/chat_* 모두 정리.
func cleanupUser(pool *pgxpool.Pool, uid string) {
	_, _ = pool.Exec(context.Background(), `delete from auth.users where id = $1`, uid)
}
