package portfolio

import (
	"math"
	"testing"
)

func approx(t *testing.T, got, want, eps float64, msg string) {
	t.Helper()
	if math.Abs(got-want) > eps {
		t.Errorf("%s: got %v, want %v (eps %v)", msg, got, want, eps)
	}
}

func TestSimulate_LumpBuyHold_EquityTracksPrice(t *testing.T) {
	days := []string{"2024-01-02", "2024-06-03"}
	legs := []Leg{{Weight: 1.0, Closes: map[string]float64{"2024-01-02": 100, "2024-06-03": 200}}}
	out := simulate(days, legs, Plan{Initial: 1_000_000}, RebalanceNone)

	approx(t, out.Equity[0].Value, 1_000_000, 1e-6, "equity[0]")
	approx(t, out.Equity[1].Value, 2_000_000, 1e-6, "equity[1]")
	approx(t, out.NAV[0].Value, 1.0, 1e-9, "nav[0]")
	approx(t, out.NAV[1].Value, 2.0, 1e-9, "nav[1]")
	approx(t, out.FinalEquity, 2_000_000, 1e-6, "finalEquity")
}

func TestSimulate_T0_RecordsInitialContribAndCashflow(t *testing.T) {
	days := []string{"2024-01-02", "2024-06-03"}
	legs := []Leg{{Weight: 1.0, Closes: map[string]float64{"2024-01-02": 100, "2024-06-03": 110}}}
	out := simulate(days, legs, Plan{Initial: 1_000_000}, RebalanceNone)

	approx(t, out.TotalContributed, 1_000_000, 1e-6, "totalContributed==Initial")
	if len(out.Cashflows) < 1 {
		t.Fatalf("no cashflows")
	}
	if out.Cashflows[0].Amount != -1_000_000 || out.Cashflows[0].Date != "2024-01-02" {
		t.Errorf("cashflows[0]=%+v, want {-1000000, 2024-01-02}", out.Cashflows[0])
	}
	last := out.Cashflows[len(out.Cashflows)-1]
	if last.Date != "2024-06-03" || last.Amount <= 0 {
		t.Errorf("final cashflow=%+v, want positive on last day", last)
	}
}

func TestSimulate_ForwardFillMissingClose(t *testing.T) {
	days := []string{"2024-01-02", "2024-01-03", "2024-01-04"}
	// 2024-01-03 종가 결측 → 직전(100) 전진 채움
	legs := []Leg{{Weight: 1.0, Closes: map[string]float64{"2024-01-02": 100, "2024-01-04": 100}}}
	out := simulate(days, legs, Plan{Initial: 1_000_000}, RebalanceNone)
	approx(t, out.Equity[1].Value, out.Equity[0].Value, 1e-6, "missing day forward-filled")
}

func TestSimulate_TwoLegFx_USDLegConvertedKRW(t *testing.T) {
	days := []string{"2024-01-02", "2024-06-03"}
	legs := []Leg{
		{Weight: 0.5, Closes: map[string]float64{"2024-01-02": 100, "2024-06-03": 100}},
		{Weight: 0.5, Closes: map[string]float64{"2024-01-02": 10, "2024-06-03": 10},
			FxToKRW: map[string]float64{"2024-01-02": 1300, "2024-06-03": 1300}},
	}
	out := simulate(days, legs, Plan{Initial: 1_000_000}, RebalanceNone)
	// 가격·환율 불변 → 평가액 불변
	approx(t, out.Equity[0].Value, 1_000_000, 1e-6, "equity[0] two-leg")
	approx(t, out.Equity[1].Value, 1_000_000, 1e-6, "equity[1] two-leg")
}
