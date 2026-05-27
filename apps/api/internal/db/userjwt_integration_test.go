//go:build integration

package db_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/db"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
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

func TestAsUser_SetsRoleAndClaims(t *testing.T) {
	pool := newTestPool(t)
	const uid = "00000000-0000-0000-0000-000000000001"
	err := db.AsUser(context.Background(), pool, uid, func(exec db.Executor) error {
		var role string
		if err := exec.QueryRow(context.Background(), `select current_setting('role', true)`).Scan(&role); err != nil {
			return err
		}
		if role != "authenticated" {
			t.Fatalf("role = %q, want authenticated", role)
		}
		var sub string
		if err := exec.QueryRow(context.Background(), `select current_setting('request.jwt.claim.sub', true)`).Scan(&sub); err != nil {
			return err
		}
		if sub != uid {
			t.Fatalf("claim.sub = %q, want %q", sub, uid)
		}
		var authUID string
		if err := exec.QueryRow(context.Background(), `select auth.uid()::text`).Scan(&authUID); err != nil {
			return err
		}
		if authUID != uid {
			t.Fatalf("auth.uid() = %q, want %q", authUID, uid)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("AsUser: %v", err)
	}
}

func TestAsUser_RollbackOnError(t *testing.T) {
	pool := newTestPool(t)
	sentinel := errors.New("boom")
	err := db.AsUser(context.Background(), pool, "00000000-0000-0000-0000-000000000001", func(exec db.Executor) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("want sentinel, got %v", err)
	}
}
