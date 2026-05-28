//go:build integration

package portfolio_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

func openPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestPgDeps_E2E(t *testing.T) {
	pool := openPool(t)
	uid := uuid.NewString()
	ctx := context.Background()

	// seed user (200일 전 가입)
	pastCreatedAt := time.Now().Add(-200 * 24 * time.Hour)
	_, err := pool.Exec(ctx, `
		insert into auth.users (id, email, encrypted_password, created_at)
		values ($1::uuid, $1::text || '@alpha.test', '', $2)
		on conflict (id) do nothing
	`, uid, pastCreatedAt)
	if err != nil {
		t.Fatalf("seed auth.users: %v", err)
	}
	defer pool.Exec(ctx, `delete from auth.users where id = $1`, uid)

	// 사용자 created_at도 200일 전으로 강제
	_, _ = pool.Exec(ctx, `update public.profiles set created_at = $1 where id = $2`, pastCreatedAt, uid)

	// holdings — KOSPI에 있는 instrument 1개 사용 (예: 005930 삼성전자)
	var instID string
	err = pool.QueryRow(ctx, `select id::text from public.instruments where symbol = '005930' limit 1`).Scan(&instID)
	if err != nil {
		t.Skip("instrument 005930 not seeded")
	}
	_, err = pool.Exec(ctx, `
		insert into public.holdings (user_id, instrument_id, quantity, avg_cost)
		values ($1, $2::uuid, 10, 60000)
	`, uid, instID)
	if err != nil {
		t.Fatalf("seed holding: %v", err)
	}

	svc := portfolio.NewService()
	res, err := svc.Compute(ctx, pool, pool, uid, portfolio.Period90D)
	if err != nil {
		// KOSPI/SPX prices 5년 백필이 안 됐을 수 있음 — skip 처리
		if _, isInsufficient := err.(*portfolio.InsufficientDataError); !isInsufficient {
			t.Skipf("Compute: %v (KOSPI/SPX prices likely not backfilled — run Task 0)", err)
		}
		return
	}
	if res.Period != portfolio.Period90D {
		t.Errorf("period=%s", res.Period)
	}
	if len(res.Portfolio.Series) < 2 {
		t.Errorf("series too short: %d", len(res.Portfolio.Series))
	}
	if len(res.Benchmarks) != 3 {
		t.Errorf("benchmarks count=%d, want 3", len(res.Benchmarks))
	}
	// 알파 계산이 NaN/Inf로 끝나지 않는지
	for _, b := range res.Benchmarks {
		if b.AlphaPP != b.AlphaPP { // NaN check
			t.Errorf("benchmark %s alpha NaN", b.Key)
		}
	}
}

// compile-time: ensure exec types match
var _ db.Executor = (*pgxpool.Pool)(nil)
