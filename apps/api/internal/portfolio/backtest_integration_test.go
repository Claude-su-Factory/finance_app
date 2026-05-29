//go:build integration

package portfolio_test

import (
	"context"
	"errors"
	"testing"

	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

func TestBacktest_E2E_SingleLeg(t *testing.T) {
	pool := openPool(t)
	ctx := context.Background()

	var instID string
	if err := pool.QueryRow(ctx, `select id::text from public.instruments where symbol = '005930' limit 1`).Scan(&instID); err != nil {
		t.Skip("instrument 005930 not seeded")
	}

	svc := portfolio.NewBacktestService()
	res, err := svc.Run(ctx, pool, portfolio.BacktestRequest{
		Period: "1Y", InitialCash: 10_000_000, Monthly: 0, Rebalance: "none",
		Basket: []portfolio.BasketItem{{InstrumentID: instID, Weight: 100}},
	})
	if err != nil {
		var ie *portfolio.InsufficientDataError
		if errors.As(err, &ie) {
			t.Skipf("백필 부족 — Task 0(backfill CLI) 선행 필요: %v", ie)
		}
		t.Fatalf("Run: %v", err)
	}
	if len(res.EquitySeries) < 30 {
		t.Errorf("equity series too short: %d", len(res.EquitySeries))
	}
	if len(res.NormalizedBasket) != 1 || res.NormalizedBasket[0].Symbol != "005930" {
		t.Errorf("normalized basket=%+v", res.NormalizedBasket)
	}
	if w := res.NormalizedBasket[0].Weight; w < 0.999 || w > 1.001 {
		t.Errorf("single-leg weight=%v, want 1.0", w)
	}
	if len(res.Benchmarks.Kospi.EquitySeries) < 30 || len(res.Benchmarks.Spx.EquitySeries) < 30 {
		t.Errorf("benchmark series too short (kospi=%d spx=%d)",
			len(res.Benchmarks.Kospi.EquitySeries), len(res.Benchmarks.Spx.EquitySeries))
	}
	if res.Metrics.FinalEquity <= 0 {
		t.Errorf("final equity=%v", res.Metrics.FinalEquity)
	}
	// MDD는 ≤ 0 (낙폭), 변동성은 ≥ 0 — NaN/Inf 아님
	if res.Metrics.MddPct > 0 || res.Metrics.MddPct != res.Metrics.MddPct {
		t.Errorf("mdd=%v invalid", res.Metrics.MddPct)
	}
}

func TestBacktest_E2E_DCAQuarterly(t *testing.T) {
	pool := openPool(t)
	ctx := context.Background()

	var instID string
	if err := pool.QueryRow(ctx, `select id::text from public.instruments where symbol = '005930' limit 1`).Scan(&instID); err != nil {
		t.Skip("instrument 005930 not seeded")
	}
	svc := portfolio.NewBacktestService()
	res, err := svc.Run(ctx, pool, portfolio.BacktestRequest{
		Period: "3Y", InitialCash: 1_000_000, Monthly: 100_000, Rebalance: "quarterly",
		Basket: []portfolio.BasketItem{{InstrumentID: instID, Weight: 100}},
	})
	if err != nil {
		var ie *portfolio.InsufficientDataError
		if errors.As(err, &ie) {
			t.Skipf("백필 부족: %v", ie)
		}
		t.Fatalf("Run: %v", err)
	}
	// 적립이 있으므로 누적 투입 > 초기 자금
	if res.Metrics.TotalContributed <= 1_000_000 {
		t.Errorf("total_contributed=%v, want > 1,000,000 (적립 반영)", res.Metrics.TotalContributed)
	}
	if len(res.ContributedSeries) != len(res.EquitySeries) {
		t.Errorf("contributed/equity length mismatch: %d vs %d", len(res.ContributedSeries), len(res.EquitySeries))
	}
}

func TestBacktest_E2E_RejectsIndex(t *testing.T) {
	pool := openPool(t)
	ctx := context.Background()

	var idxID string
	if err := pool.QueryRow(ctx, `select id::text from public.instruments where asset_class = 'INDEX' limit 1`).Scan(&idxID); err != nil {
		t.Skip("no INDEX instrument seeded")
	}
	svc := portfolio.NewBacktestService()
	_, err := svc.Run(ctx, pool, portfolio.BacktestRequest{
		Period: "1Y", InitialCash: 10_000_000, Rebalance: "none",
		Basket: []portfolio.BasketItem{{InstrumentID: idxID, Weight: 100}},
	})
	var ve *portfolio.ValidationError
	if !errors.As(err, &ve) || ve.Code != "ASSET_NOT_SUPPORTED" {
		t.Errorf("want ASSET_NOT_SUPPORTED, got %v", err)
	}
}
