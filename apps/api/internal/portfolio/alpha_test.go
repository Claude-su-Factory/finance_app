package portfolio_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

func TestParsePeriod(t *testing.T) {
	cases := map[string]bool{
		"1m": true, "90d": true, "1y": true, "all": true,
		"":    false,
		"30d": false,
		"2y":  false,
		"FOO": false,
	}
	for in, ok := range cases {
		_, err := portfolio.ParsePeriod(in)
		if (err == nil) != ok {
			t.Errorf("ParsePeriod(%q): want ok=%v, got err=%v", in, ok, err)
		}
	}
}

func TestPeriodDays(t *testing.T) {
	cases := map[portfolio.Period]int{
		portfolio.Period1M:  30,
		portfolio.Period90D: 90,
		portfolio.Period1Y:  365,
		portfolio.PeriodAll: 0,
	}
	for p, want := range cases {
		if got := p.Days(); got != want {
			t.Errorf("%s.Days(): got %d, want %d", p, got, want)
		}
	}
}

// fakeAlphaDeps는 모든 DB 호출을 흉내내는 가짜.
type fakeAlphaDeps struct {
	createdAt   time.Time
	holdings    []portfolio.HoldingRow
	prices      map[string]map[string]float64
	fxRates     map[string]map[string]float64
	tradingDays []string
	benchmarks  map[string][]portfolio.PricePoint
}

func (f *fakeAlphaDeps) UserCreatedAt(_ context.Context, _ db.Executor, _ string) (time.Time, error) {
	return f.createdAt, nil
}
func (f *fakeAlphaDeps) UserHoldings(_ context.Context, _ db.Executor, _ string) ([]portfolio.HoldingRow, error) {
	return f.holdings, nil
}
func (f *fakeAlphaDeps) TradingDays(_ context.Context, _ db.Executor, _, _ time.Time) ([]string, error) {
	return f.tradingDays, nil
}
func (f *fakeAlphaDeps) InstrumentClosesOnDates(_ context.Context, _ db.Executor, iid string, _ []string) (map[string]float64, error) {
	return f.prices[iid], nil
}
func (f *fakeAlphaDeps) FxRatesOnDates(_ context.Context, _ db.Executor, cur string, _ []string) (map[string]float64, error) {
	return f.fxRates[cur], nil
}
func (f *fakeAlphaDeps) BenchmarkSeries(_ context.Context, _ db.Executor, symbol string, _ []string) ([]portfolio.PricePoint, error) {
	return f.benchmarks[symbol], nil
}

func TestCompute_BasicTwoHoldings(t *testing.T) {
	deps := &fakeAlphaDeps{
		createdAt: mustTime("2024-01-01T00:00:00Z"),
		holdings: []portfolio.HoldingRow{
			{InstrumentID: "kr-1", Symbol: "005930", Currency: "KRW", Quantity: 10},
			{InstrumentID: "us-1", Symbol: "AAPL", Currency: "USD", Quantity: 5},
		},
		tradingDays: []string{"2026-02-27", "2026-05-28"},
		prices: map[string]map[string]float64{
			"kr-1": {"2026-02-27": 60000, "2026-05-28": 66000},
			"us-1": {"2026-02-27": 100, "2026-05-28": 120},
		},
		fxRates: map[string]map[string]float64{
			"USD": {"2026-02-27": 1300, "2026-05-28": 1378},
		},
		benchmarks: map[string][]portfolio.PricePoint{
			"KOSPI": {{Date: "2026-02-27", Close: 2500}, {Date: "2026-05-28", Close: 2580}},
			"SPX":   {{Date: "2026-02-27", Close: 5500}, {Date: "2026-05-28", Close: 5850}},
		},
	}
	svc := portfolio.NewServiceWithDeps(deps, mustTime("2026-05-28T15:00:00+09:00"))
	res, err := svc.Compute(context.Background(), nil, nil, "user-1", portfolio.Period90D)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	// 포트 시작값: 10*60000*1 + 5*100*1300 = 600,000 + 650,000 = 1,250,000
	// 포트 종료값: 10*66000*1 + 5*120*1378 = 660,000 + 826,800 = 1,486,800
	// 수익률: (1486800 - 1250000) / 1250000 * 100 = 18.944%
	wantPort := 18.944
	if abs(res.Portfolio.TotalReturnPct-wantPort) > 0.01 {
		t.Errorf("portfolio total = %.4f, want %.4f", res.Portfolio.TotalReturnPct, wantPort)
	}

	// KOSPI: (2580 - 2500) / 2500 * 100 = 3.2%
	// alpha = 18.944 - 3.2 = 15.744
	wantAlphaK := 15.744
	if got := res.Benchmarks[0].AlphaPP; abs(got-wantAlphaK) > 0.01 {
		t.Errorf("kospi alpha = %.4f, want %.4f", got, wantAlphaK)
	}
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
func mustTime(s string) time.Time { t, _ := time.Parse(time.RFC3339, s); return t }

// 컴파일 가드 — pgx/pgconn import unused 방지 (이후 다른 테스트에서 쓰일 수 있음)
var _ = pgx.ErrNoRows
var _ = pgconn.CommandTag{}
var _ = errors.New
