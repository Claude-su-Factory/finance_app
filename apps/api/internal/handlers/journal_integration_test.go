//go:build integration

package handlers_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/handlers"
	"github.com/quotient/quotient/apps/api/internal/models"
)

func openJournalPool(t *testing.T) *pgxpool.Pool {
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

func seedJournalUser(t *testing.T, pool *pgxpool.Pool, uid string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		insert into auth.users (id, email, encrypted_password)
		values ($1::uuid, $1::text || '@journal.test', '')
		on conflict (id) do nothing
	`, uid)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}
}

func cleanupJournalUser(pool *pgxpool.Pool, uid string) {
	_, _ = pool.Exec(context.Background(), `delete from auth.users where id = $1`, uid)
}

func strPtrJ(s string) *string { return &s }

func TestJournal_E2E_CreateListDelete(t *testing.T) {
	pool := openJournalPool(t)
	uid := uuid.NewString()
	seedJournalUser(t, pool, uid)
	defer cleanupJournalUser(pool, uid)

	ctx := context.Background()
	repo := handlers.NewPgJournalRepo()

	// 사용자 JWT 트랜잭션 안에서 CRUD
	var createdID string
	err := db.AsUser(ctx, pool, uid, func(exec db.Executor) error {
		title := "통합 테스트"
		body := models.JournalEntryCreate{
			Action:         strPtrJ("observation"),
			RelatedSymbols: []string{"005930"},
			Title:          &title,
			Content:        "통합 테스트 entry",
		}
		e, err := repo.CreateEntry(ctx, exec, uid, "manual", body, nil)
		if err != nil {
			return err
		}
		createdID = e.ID
		return nil
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// list
	err = db.AsUser(ctx, pool, uid, func(exec db.Executor) error {
		entries, _, err := repo.ListEntries(ctx, exec, uid, 10, nil)
		if err != nil {
			return err
		}
		if len(entries) != 1 {
			t.Errorf("entries=%d, want 1", len(entries))
		}
		if entries[0].Content != "통합 테스트 entry" {
			t.Errorf("content=%q", entries[0].Content)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// delete
	err = db.AsUser(ctx, pool, uid, func(exec db.Executor) error {
		return repo.DeleteEntry(ctx, exec, uid, createdID)
	})
	if err != nil {
		t.Fatal(err)
	}

	// list — 0건
	err = db.AsUser(ctx, pool, uid, func(exec db.Executor) error {
		entries, _, err := repo.ListEntries(ctx, exec, uid, 10, nil)
		if err != nil {
			return err
		}
		if len(entries) != 0 {
			t.Errorf("entries after delete=%d, want 0", len(entries))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestJournal_RLS_Isolation(t *testing.T) {
	pool := openJournalPool(t)
	uA := uuid.NewString()
	uB := uuid.NewString()
	seedJournalUser(t, pool, uA)
	seedJournalUser(t, pool, uB)
	defer cleanupJournalUser(pool, uA)
	defer cleanupJournalUser(pool, uB)

	ctx := context.Background()
	repo := handlers.NewPgJournalRepo()

	// A로 entry 생성
	_ = db.AsUser(ctx, pool, uA, func(exec db.Executor) error {
		_, err := repo.CreateEntry(ctx, exec, uA, "manual",
			models.JournalEntryCreate{Action: strPtrJ("observation"), Content: "A의 entry"},
			nil)
		return err
	})

	// B가 list — A의 entry 안 보여야
	var bCount int
	_ = db.AsUser(ctx, pool, uB, func(exec db.Executor) error {
		entries, _, err := repo.ListEntries(ctx, exec, uB, 10, nil)
		bCount = len(entries)
		return err
	})
	if bCount != 0 {
		t.Errorf("RLS leak: user B sees %d entries of user A", bCount)
	}
}
