package portfolio_test

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

type fakeEquityDeps struct {
	tradingDays []string
	prices      map[string]map[string]float64
	fxRates     map[string]map[string]float64
}

func (f *fakeEquityDeps) TradingDays(_ context.Context, _ db.Executor, _, _ time.Time) ([]string, error) {
	return f.tradingDays, nil
}
func (f *fakeEquityDeps) InstrumentClosesOnDates(_ context.Context, _ db.Executor, iid string, _ []string) (map[string]float64, error) {
	return f.prices[iid], nil
}
func (f *fakeEquityDeps) FxRatesOnDates(_ context.Context, _ db.Executor, cur string, _ []string) (map[string]float64, error) {
	return f.fxRates[cur], nil
}

// compile-time guard
var _ db.Executor = (*pgxpoolStub)(nil)

type pgxpoolStub struct{}

func (pgxpoolStub) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) { return nil, nil }
func (pgxpoolStub) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row        { return nil }
func (pgxpoolStub) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func TestComputeEquity_NoTransactions_StartsFromInitial(t *testing.T) {
	deps := &fakeEquityDeps{
		tradingDays: []string{"2026-02-27", "2026-05-28"},
	}
	account := &models.PaperAccount{
		InitialCash: 10000000,
		CashBalance: 10000000,
		CreatedAt:   mustTimeP("2026-01-01T00:00:00Z"),
	}
	c := portfolio.NewEquityComputer(deps)
	out, err := c.Compute(context.Background(), nil, account, nil, nil, 90)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("len=%d, want 2", len(out))
	}
	for _, p := range out {
		if p.EquityKRW != 10000000 {
			t.Errorf("%s equity=%.0f, want 10000000", p.Date, p.EquityKRW)
		}
	}
}

func TestComputeEquity_AfterBuy_ReflectsPriceChanges(t *testing.T) {
	deps := &fakeEquityDeps{
		tradingDays: []string{"2026-05-27", "2026-05-28"},
		prices: map[string]map[string]float64{
			"inst-1": {"2026-05-27": 60000, "2026-05-28": 66000},
		},
		fxRates: map[string]map[string]float64{},
	}
	account := &models.PaperAccount{
		InitialCash: 10000000,
		CashBalance: 9400000,
		CreatedAt:   mustTimeP("2026-01-01T00:00:00Z"),
	}
	transactions := []models.PaperTransaction{
		{
			InstrumentID: "inst-1", Action: "buy", Quantity: 10,
			Price: 60000, Currency: "KRW", FxToKRW: 1.0, TotalKRW: 600000,
			CreatedAt: mustTimeP("2026-05-27T10:00:00Z"),
			Active:    true,
		},
	}
	c := portfolio.NewEquityComputer(deps)
	out, err := c.Compute(context.Background(), nil, account, transactions, nil, 90)
	if err != nil {
		t.Fatal(err)
	}
	want1 := 10000000.0
	want2 := 10060000.0
	if absP(out[0].EquityKRW-want1) > 0.01 {
		t.Errorf("day0=%.0f, want %.0f", out[0].EquityKRW, want1)
	}
	if absP(out[1].EquityKRW-want2) > 0.01 {
		t.Errorf("day1=%.0f, want %.0f", out[1].EquityKRW, want2)
	}
}

func TestComputeEquity_AfterSell_ReflectsCashIncrease(t *testing.T) {
	deps := &fakeEquityDeps{
		tradingDays: []string{"2026-05-27", "2026-05-28"},
		prices: map[string]map[string]float64{
			"inst-1": {"2026-05-27": 60000, "2026-05-28": 66000},
		},
	}
	account := &models.PaperAccount{
		InitialCash: 10000000,
		CashBalance: 10060000,
		CreatedAt:   mustTimeP("2026-01-01T00:00:00Z"),
	}
	transactions := []models.PaperTransaction{
		{InstrumentID: "inst-1", Action: "buy", Quantity: 10, Price: 60000, Currency: "KRW",
			FxToKRW: 1, TotalKRW: 600000, CreatedAt: mustTimeP("2026-05-27T10:00:00Z"), Active: true},
		{InstrumentID: "inst-1", Action: "sell", Quantity: 10, Price: 66000, Currency: "KRW",
			FxToKRW: 1, TotalKRW: 660000, CreatedAt: mustTimeP("2026-05-28T10:00:00Z"), Active: true},
	}
	c := portfolio.NewEquityComputer(deps)
	out, err := c.Compute(context.Background(), nil, account, transactions, nil, 90)
	if err != nil {
		t.Fatal(err)
	}
	want := 10060000.0
	if absP(out[1].EquityKRW-want) > 0.01 {
		t.Errorf("day1=%.0f, want %.0f", out[1].EquityKRW, want)
	}
}

func TestComputeEquity_SinceClampedToCreatedAt(t *testing.T) {
	deps := &fakeEquityDeps{
		tradingDays: []string{"2026-05-25", "2026-05-26", "2026-05-28"},
	}
	account := &models.PaperAccount{
		InitialCash: 10000000,
		CashBalance: 10000000,
		CreatedAt:   mustTimeP("2026-05-25T00:00:00Z"),
	}
	c := portfolio.NewEquityComputer(deps)
	out, err := c.Compute(context.Background(), nil, account, nil, nil, 90)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 3 {
		t.Errorf("len=%d, want 3", len(out))
	}
}

func mustTimeP(s string) time.Time { t, _ := time.Parse(time.RFC3339, s); return t }
func absP(f float64) float64       { if f < 0 { return -f }; return f }
