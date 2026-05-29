package portfolio

import (
	"math"
	"testing"
	"time"
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

// monthlyAnchors — start 기준 0..months 개월 앵커 날짜 (day≤28이면 오버플로 없음 → addMonthsClamped와 동일).
func monthlyAnchors(t *testing.T, start string, months int) []string {
	t.Helper()
	base, err := time.Parse("2006-01-02", start)
	if err != nil {
		t.Fatalf("bad start: %v", err)
	}
	out := make([]string, 0, months+1)
	for k := 0; k <= months; k++ {
		out = append(out, base.AddDate(0, k, 0).Format("2006-01-02"))
	}
	return out
}

func TestSimulate_DCA_ContributionsMintUnits_NAVUnchanged(t *testing.T) {
	days := monthlyAnchors(t, "2021-01-15", 12) // 13일, 매월 앵커
	closes := map[string]float64{}
	for _, d := range days {
		closes[d] = 100 // 가격 불변
	}
	legs := []Leg{{Weight: 1.0, Closes: closes}}
	out := simulate(days, legs, Plan{Initial: 1_000_000, Monthly: 500_000}, RebalanceNone)

	// 가격 불변 → 적립으로 유닛만 늘고 NAV는 항상 1.0 (적립일 점프 없음)
	for i, p := range out.NAV {
		approx(t, p.Value, 1.0, 1e-9, "nav["+days[i]+"]")
	}
}

func TestSimulate_DCA_ContributionCount(t *testing.T) {
	days := monthlyAnchors(t, "2021-01-15", 36) // 37일 = t0 + 36 앵커
	closes := map[string]float64{}
	for _, d := range days {
		closes[d] = 100
	}
	legs := []Leg{{Weight: 1.0, Closes: closes}}
	out := simulate(days, legs, Plan{Initial: 1_000_000, Monthly: 500_000}, RebalanceNone)

	// 첫 적립 t0+1개월(t0 당일 없음) → 정확히 36회
	approx(t, out.TotalContributed, 1_000_000+36*500_000, 1e-6, "totalContributed")
	neg := 0
	for _, cf := range out.Cashflows {
		if cf.Amount < 0 {
			neg++
		}
	}
	if neg != 37 { // 초기 1 + 적립 36
		t.Errorf("negative cashflows=%d, want 37", neg)
	}
}
