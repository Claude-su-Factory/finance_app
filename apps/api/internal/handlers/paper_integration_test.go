//go:build integration

package handlers_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/handlers"
	"github.com/quotient/quotient/apps/api/internal/models"
)

func openPaperPool(t *testing.T) *pgxpool.Pool {
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

func seedPaperUser(t *testing.T, pool *pgxpool.Pool, uid string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		insert into auth.users (id, email, encrypted_password)
		values ($1::uuid, $1::text || '@paper.test', '')
		on conflict (id) do nothing
	`, uid)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
}

func cleanupPaperUser(pool *pgxpool.Pool, uid string) {
	_, _ = pool.Exec(context.Background(), `delete from auth.users where id = $1`, uid)
}

// seedTestInstrument는 FK 충족용 테스트 종목을 멱등 삽입한다. 백필 CLI 실행 여부와
// 무관하게 통합 테스트가 동작하도록 — 실제 시세 데이터는 필요 없다(테스트가 가격을 직접 전달).
func seedTestInstrument(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(), `
		insert into public.instruments (symbol, exchange, name, asset_class, currency)
		values ('TST', 'TEST', 'Paper Test Instrument', 'KR_STOCK', 'KRW')
		on conflict (symbol, exchange) do update set name = excluded.name
		returning id::text
	`).Scan(&id)
	if err != nil {
		t.Fatalf("seed instrument: %v", err)
	}
	return id
}

func TestPaper_E2E_AccountAutoCreate(t *testing.T) {
	pool := openPaperPool(t)
	uid := uuid.NewString()
	seedPaperUser(t, pool, uid)
	defer cleanupPaperUser(pool, uid)

	repo := handlers.NewPgPaperRepo()
	ctx := context.Background()

	err := db.AsUser(ctx, pool, uid, func(exec db.Executor) error {
		a, err := repo.GetOrCreateAccount(ctx, exec, uid)
		if err != nil {
			return err
		}
		if a.InitialCash != 10000000 || a.CashBalance != 10000000 {
			t.Errorf("initial=%.0f balance=%.0f", a.InitialCash, a.CashBalance)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPaper_E2E_BuyTransaction(t *testing.T) {
	pool := openPaperPool(t)
	uid := uuid.NewString()
	seedPaperUser(t, pool, uid)
	defer cleanupPaperUser(pool, uid)

	instID := seedTestInstrument(t, pool)

	repo := handlers.NewPgPaperRepo()
	ctx := context.Background()

	err := db.AsUser(ctx, pool, uid, func(exec db.Executor) error {
		if _, err := repo.GetOrCreateAccount(ctx, exec, uid); err != nil {
			return err
		}
		tx := &models.PaperTransaction{
			UserID: uid, InstrumentID: instID, Action: "buy",
			Quantity: 10, Price: 70000, Currency: "KRW",
			FxToKRW: 1.0, TotalKRW: 700000, Active: true,
		}
		if err := repo.InsertTransaction(ctx, exec, tx); err != nil {
			return err
		}
		if err := repo.UpsertHolding(ctx, exec, uid, instID, 10, 70000); err != nil {
			return err
		}
		newBal, err := repo.ApplyCashDelta(ctx, exec, uid, -700000)
		if err != nil {
			return err
		}
		if newBal != 9300000 {
			t.Errorf("newBal=%.0f, want 9300000", newBal)
		}

		holdings, err := repo.ListHoldings(ctx, exec, uid)
		if err != nil {
			return err
		}
		if len(holdings) != 1 || holdings[0].Quantity != 10 {
			t.Errorf("holdings=%+v", holdings)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestPaper_RLS_Isolation(t *testing.T) {
	pool := openPaperPool(t)
	uA := uuid.NewString()
	uB := uuid.NewString()
	seedPaperUser(t, pool, uA)
	seedPaperUser(t, pool, uB)
	defer cleanupPaperUser(pool, uA)
	defer cleanupPaperUser(pool, uB)

	instID := seedTestInstrument(t, pool)
	repo := handlers.NewPgPaperRepo()
	ctx := context.Background()

	_ = db.AsUser(ctx, pool, uA, func(exec db.Executor) error {
		_, _ = repo.GetOrCreateAccount(ctx, exec, uA)
		_ = repo.UpsertHolding(ctx, exec, uA, instID, 1, 100)
		return nil
	})

	var bCount int
	_ = db.AsUser(ctx, pool, uB, func(exec db.Executor) error {
		hs, err := repo.ListHoldings(ctx, exec, uB)
		if err != nil {
			return err
		}
		bCount = len(hs)
		return nil
	})
	if bCount != 0 {
		t.Errorf("RLS leak: user B sees %d holdings of A", bCount)
	}
}

func TestPaper_E2E_InsufficientCash(t *testing.T) {
	pool := openPaperPool(t)
	uid := uuid.NewString()
	seedPaperUser(t, pool, uid)
	defer cleanupPaperUser(pool, uid)

	repo := handlers.NewPgPaperRepo()
	ctx := context.Background()

	err := db.AsUser(ctx, pool, uid, func(exec db.Executor) error {
		if _, err := repo.GetOrCreateAccount(ctx, exec, uid); err != nil {
			return err
		}
		_, err := repo.ApplyCashDelta(ctx, exec, uid, -99999999)
		if !errors.Is(err, handlers.ErrInsufficientCash) {
			t.Errorf("err=%v, want ErrInsufficientCash", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
