package portfolio

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/quotient/quotient/apps/api/internal/db"
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

func TestSimulate_Rebalance_RestoresTargetWeights(t *testing.T) {
	// 분기 리밸런싱: t0=01-15, nextRebal=04-15. leg0 2배 상승 → 04-15에 50/50 복원.
	days := []string{"2021-01-15", "2021-04-15", "2021-05-15"}
	legs := []Leg{
		{Weight: 0.5, Closes: map[string]float64{"2021-01-15": 100, "2021-04-15": 200, "2021-05-15": 200}},
		{Weight: 0.5, Closes: map[string]float64{"2021-01-15": 100, "2021-04-15": 100, "2021-05-15": 50}},
	}
	out := simulate(days, legs, Plan{Initial: 1_000_000}, RebalanceQuarterly)

	// 04-15: 리밸런싱은 총 평가액 보존 (1.5M)
	approx(t, out.Equity[1].Value, 1_500_000, 1e-6, "equity on rebalance day")
	// 05-15: 리밸런싱된 주식수(leg0=3750, leg1=7500)로 평가 → 1,125,000.
	// (리밸런싱 없었다면 5000/5000 → 1,250,000)
	approx(t, out.Equity[2].Value, 1_125_000, 1e-6, "equity reflects restored weights")
}

func TestSimulate_DCAplusRebalance_OrderCorrect(t *testing.T) {
	// 04-15는 적립일(+1m 커서가 도달)이자 분기 리밸런싱일. 적립 먼저 → 리밸런싱은 적립 후 V 사용.
	days := []string{"2021-01-15", "2021-04-15"}
	legs := []Leg{
		{Weight: 0.5, Closes: map[string]float64{"2021-01-15": 100, "2021-04-15": 200}},
		{Weight: 0.5, Closes: map[string]float64{"2021-01-15": 100, "2021-04-15": 100}},
	}
	out := simulate(days, legs, Plan{Initial: 1_000_000, Monthly: 300_000}, RebalanceQuarterly)

	// 적립 후 V=1.8M (1.5M 평가 + 300k 투입). 리밸런싱이 이 1.8M을 배분 → Equity=1.8M.
	// 버그(캐시된 1.5M 사용) 시 300k 미배분 → 1.5M.
	approx(t, out.Equity[1].Value, 1_800_000, 1e-6, "contribution allocated before rebalance")
	approx(t, out.TotalContributed, 1_300_000, 1e-6, "totalContributed = 1M + 300k")
}

func TestXIRR_LumpSum_MatchesCAGR(t *testing.T) {
	// -1M 투입, 3년 후 +2M 회수 → (2)^(1/3)-1 ≈ 0.259921
	cfs := []Cashflow{{Amount: -1_000_000, Date: "2021-01-01"}, {Amount: 2_000_000, Date: "2024-01-01"}}
	r := xirr(cfs)
	if r == nil {
		t.Fatalf("xirr returned nil")
	}
	approx(t, *r, 0.259921, 1e-4, "lump-sum xirr == cube-root(2)-1")
}

func TestXIRR_DCA_KnownCashflows(t *testing.T) {
	// 두 번 투입 후 회수 — 양의 수익. 근에서 NPV≈0 + 양수 수렴 검증.
	cfs := []Cashflow{
		{Amount: -1000, Date: "2020-01-01"},
		{Amount: -1000, Date: "2021-01-01"},
		{Amount: 2100, Date: "2022-01-01"},
	}
	r := xirr(cfs)
	if r == nil {
		t.Fatalf("xirr returned nil")
	}
	if *r <= 0 {
		t.Errorf("expected positive return, got %v", *r)
	}
	approx(t, xirrNPV(cfs, *r), 0, 1e-3, "NPV at root ≈ 0")
}

func TestXIRR_NonConverging_ReturnsNull(t *testing.T) {
	// 부호가 모두 음수 → 해 없음 → nil.
	cfs := []Cashflow{{Amount: -1000, Date: "2020-01-01"}, {Amount: -1000, Date: "2021-01-01"}}
	if r := xirr(cfs); r != nil {
		t.Errorf("expected nil, got %v", *r)
	}
}

func TestMetrics_MDD_OnNAV_NotEquity(t *testing.T) {
	// NAV는 1.0→0.8→1.1 (−20% 낙폭), Equity는 적립으로 단조 증가.
	// MDD가 Equity 기준이면 0%(단조증가), NAV 기준이면 −20%.
	out := SimOutput{
		Equity: []ValuePoint{{"d0", 1_000_000}, {"d1", 1_500_000}, {"d2", 2_500_000}},
		NAV:    []ValuePoint{{"d0", 1.0}, {"d1", 0.8}, {"d2", 1.1}},
		Cashflows: []Cashflow{
			{Amount: -1_000_000, Date: "2021-01-01"}, {Amount: -1_000_000, Date: "2021-06-01"},
			{Amount: 2_500_000, Date: "2022-01-01"},
		},
		TotalContributed: 2_000_000, FinalEquity: 2_500_000,
	}
	m := metrics(out)
	approx(t, m.MDDPct, -20, 1e-6, "MDD on NAV")
}

func TestMetrics_Volatility_Annualized(t *testing.T) {
	// 일간 수익률 +10%, −10% → 표본표준편차 √0.02, 연환산 ×√252.
	out := SimOutput{NAV: []ValuePoint{{"d0", 100}, {"d1", 110}, {"d2", 99}}}
	m := metrics(out)
	approx(t, m.VolatilityPct, 224.50, 0.1, "annualized vol")
}

func TestMetrics_LumpSum_TotalReturnAndCAGR(t *testing.T) {
	out := SimOutput{
		NAV:              []ValuePoint{{"d0", 1.0}, {"d1", 2.0}},
		Cashflows:        []Cashflow{{Amount: -1_000_000, Date: "2021-01-01"}, {Amount: 2_000_000, Date: "2024-01-01"}},
		TotalContributed: 1_000_000, FinalEquity: 2_000_000,
	}
	m := metrics(out)
	approx(t, m.TotalReturnPct, 100, 1e-6, "total return")
	approx(t, m.TWRPct, 100, 1e-6, "twr = (NAV_end-1)*100")
	if m.CAGRPct == nil {
		t.Fatalf("cagr nil")
	}
	approx(t, *m.CAGRPct, 25.9921, 1e-2, "cagr pct ~ 26")
}

func TestPgDepsSatisfiesBacktestDeps(t *testing.T) {
	// 컴파일 타임 인터페이스 만족 검사 — PgDeps가 BacktestDeps를 구현하는가.
	var _ BacktestDeps = PgDeps{}

	// InstrumentsMeta는 ids 비었을 때 Query 전에 단락 → DB 없이 검증 가능.
	got, err := PgDeps{}.InstrumentsMeta(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("InstrumentsMeta(empty) err: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty map, got %d entries", len(got))
	}
}

// --- 서비스 레이어 테스트 (BacktestDeps 모킹) ---

type fakeBTDeps struct {
	tradingDays []string                      // 정렬된 영업일 축
	closes      map[string]map[string]float64 // iid → date → close
	fx          map[string]map[string]float64 // currency → date → rate
	bench       map[string][]PricePoint       // symbol(KOSPI/SPX) → 정렬된 종가
	metas       map[string]InstrumentMeta     // iid → meta
}

func (f fakeBTDeps) TradingDays(_ context.Context, _ db.Executor, since, until time.Time) ([]string, error) {
	s, u := since.Format("2006-01-02"), until.Format("2006-01-02")
	var out []string
	for _, d := range f.tradingDays {
		if d >= s && d <= u {
			out = append(out, d)
		}
	}
	return out, nil
}

func (f fakeBTDeps) InstrumentClosesOnDates(_ context.Context, _ db.Executor, iid string, dates []string) (map[string]float64, error) {
	src := f.closes[iid]
	out := map[string]float64{}
	for _, d := range dates {
		if v, ok := src[d]; ok {
			out[d] = v
		}
	}
	return out, nil
}

func (f fakeBTDeps) FxRatesOnDates(_ context.Context, _ db.Executor, currency string, dates []string) (map[string]float64, error) {
	if currency == "KRW" {
		return map[string]float64{}, nil
	}
	src := f.fx[currency]
	out := map[string]float64{}
	for _, d := range dates {
		if v, ok := src[d]; ok {
			out[d] = v
		}
	}
	return out, nil
}

func (f fakeBTDeps) BenchmarkSeries(_ context.Context, _ db.Executor, symbol string, dates []string) ([]PricePoint, error) {
	set := map[string]bool{}
	for _, d := range dates {
		set[d] = true
	}
	var out []PricePoint
	for _, p := range f.bench[symbol] {
		if set[p.Date] {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f fakeBTDeps) InstrumentsMeta(_ context.Context, _ db.Executor, ids []string) (map[string]InstrumentMeta, error) {
	out := map[string]InstrumentMeta{}
	for _, id := range ids {
		if m, ok := f.metas[id]; ok {
			out[id] = m
		}
	}
	return out, nil
}

func genDays(t *testing.T, start string, n int) []string {
	t.Helper()
	base, err := time.Parse("2006-01-02", start)
	if err != nil {
		t.Fatalf("genDays parse: %v", err)
	}
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = base.AddDate(0, 0, i).Format("2006-01-02")
	}
	return out
}

func constMap(days []string, v float64) map[string]float64 {
	m := map[string]float64{}
	for _, d := range days {
		m[d] = v
	}
	return m
}

func benchPts(days []string, level float64) []PricePoint {
	out := make([]PricePoint, len(days))
	for i, d := range days {
		out[i] = PricePoint{Date: d, Close: level}
	}
	return out
}

// 공통 벤치마크/메타 시드 — KR 단일 종목 전략용.
func krStockDeps(days []string, ids []string) fakeBTDeps {
	closes := map[string]map[string]float64{}
	metas := map[string]InstrumentMeta{}
	for i, id := range ids {
		closes[id] = constMap(days, 100)
		metas[id] = InstrumentMeta{Symbol: "00" + string(rune('A'+i)), Name: "종목" + string(rune('A'+i)), Currency: "KRW", AssetClass: "KR_STOCK"}
	}
	return fakeBTDeps{
		tradingDays: days,
		closes:      closes,
		fx:          map[string]map[string]float64{"USD": constMap(days, 1300)},
		bench:       map[string][]PricePoint{"KOSPI": benchPts(days, 2000), "SPX": benchPts(days, 4000)},
		metas:       metas,
	}
}

func TestRun_NormalizesWeights(t *testing.T) {
	days := genDays(t, "2024-01-01", 40)
	deps := krStockDeps(days, []string{"id1", "id2", "id3"})
	svc := newBacktestServiceWithDeps(deps, mustParse(t, "2024-02-15"))
	res, err := svc.Run(context.Background(), nil, BacktestRequest{
		Period: "all", InitialCash: 1_000_000, Monthly: 0, Rebalance: "none",
		Basket: []BasketItem{{"id1", 40}, {"id2", 30}, {"id3", 30}},
	})
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	want := []float64{0.4, 0.3, 0.3}
	for i, nl := range res.NormalizedBasket {
		approx(t, nl.Weight, want[i], 1e-9, "normalized weight")
	}
	approx(t, res.EquitySeries[0].Value, 1_000_000, 1e-6, "equity[0] flat")
	approx(t, res.Metrics.FinalEquity, 1_000_000, 1e-6, "final flat")
}

func TestRun_InsufficientData_422(t *testing.T) {
	days := genDays(t, "2024-01-01", 10)
	deps := krStockDeps(days, []string{"id1"})
	svc := newBacktestServiceWithDeps(deps, mustParse(t, "2024-02-15"))
	_, err := svc.Run(context.Background(), nil, BacktestRequest{
		Period: "all", InitialCash: 1_000_000, Rebalance: "none",
		Basket: []BasketItem{{"id1", 100}},
	})
	var ie *InsufficientDataError
	if !errors.As(err, &ie) {
		t.Fatalf("want InsufficientDataError, got %v", err)
	}
	if ie.MinDays != 30 || ie.CurrentDays != 10 {
		t.Errorf("ie=%+v, want Min=30 Current=10", ie)
	}
}

func TestRun_ClampsStartToCommonWindow(t *testing.T) {
	days := genDays(t, "2024-01-01", 45)
	deps := krStockDeps(days, []string{"id1", "id2"})
	// id2는 11번째 날(2024-01-11)부터만 데이터 존재 → 클램프 지배.
	deps.closes["id2"] = map[string]float64{}
	for _, d := range days[10:] {
		deps.closes["id2"][d] = 100
	}
	svc := newBacktestServiceWithDeps(deps, mustParse(t, "2024-02-20"))
	res, err := svc.Run(context.Background(), nil, BacktestRequest{
		Period: "all", InitialCash: 1_000_000, Rebalance: "none",
		Basket: []BasketItem{{"id1", 50}, {"id2", 50}},
	})
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	if res.ClampedStart != "2024-01-11" {
		t.Errorf("clampedStart=%s, want 2024-01-11", res.ClampedStart)
	}
	found := false
	for _, w := range res.CoverageWarnings {
		if w.FirstAvailable == "2024-01-11" {
			found = true
		}
	}
	if !found {
		t.Errorf("coverage_warnings missing clamp source: %+v", res.CoverageWarnings)
	}
}

func TestRun_SPXBenchmark_AppliesUsdKrwFx(t *testing.T) {
	days := genDays(t, "2024-01-01", 35)
	deps := krStockDeps(days, []string{"id1"})
	// USD fx: 첫날 1300, 마지막날 1430 (+10%). 중간은 전진 채움.
	fx := constMap(days, 1300)
	fx[days[len(days)-1]] = 1430
	deps.fx["USD"] = fx
	svc := newBacktestServiceWithDeps(deps, mustParse(t, "2024-02-15"))
	res, err := svc.Run(context.Background(), nil, BacktestRequest{
		Period: "all", InitialCash: 1_000_000, Monthly: 0, Rebalance: "none",
		Basket: []BasketItem{{"id1", 100}},
	})
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	spx := res.Benchmarks.Spx.EquitySeries
	approx(t, spx[len(spx)-1].Value, 1_100_000, 1.0, "SPX 벤치마크 fx +10% 반영")
	kospi := res.Benchmarks.Kospi.EquitySeries
	approx(t, kospi[len(kospi)-1].Value, 1_000_000, 1.0, "KOSPI 벤치마크 fx=1.0 평탄")
}

func TestRun_BenchmarksUseSameCashflow(t *testing.T) {
	days := genDays(t, "2024-01-01", 35)
	deps := krStockDeps(days, []string{"id1"})
	svc := newBacktestServiceWithDeps(deps, mustParse(t, "2024-02-15"))
	res, err := svc.Run(context.Background(), nil, BacktestRequest{
		Period: "all", InitialCash: 1_000_000, Monthly: 500_000, Rebalance: "none",
		Basket: []BasketItem{{"id1", 100}},
	})
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	// 전 종목 가격 불변 → 평가액 == 누적 투입. 동일 plan → 전략·벤치마크 최종 동일.
	approx(t, res.Metrics.TotalContributed, 1_500_000, 1e-6, "초기 1M + 적립 1회 500k")
	approx(t, res.Metrics.FinalEquity, 1_500_000, 1e-6, "flat → final == contributed")
	kospi := res.Benchmarks.Kospi.EquitySeries
	spx := res.Benchmarks.Spx.EquitySeries
	approx(t, kospi[len(kospi)-1].Value, 1_500_000, 1.0, "KOSPI 동일 현금흐름")
	approx(t, spx[len(spx)-1].Value, 1_500_000, 1.0, "SPX 동일 현금흐름")
}

func mustParse(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("mustParse %s: %v", s, err)
	}
	return tm
}

func TestRun_ClampIncludesBenchmarkFirstAvailable(t *testing.T) {
	days := genDays(t, "2024-01-01", 50)
	deps := krStockDeps(days, []string{"id1"})
	// 전략 leg(id1)은 2024-01-01부터 존재하나, KOSPI 벤치마크는 2024-01-16부터만 존재
	// → 벤치마크 firstAvailable이 클램프를 지배해야 한다(§5-5).
	deps.bench["KOSPI"] = benchPts(days[15:], 2000)
	svc := newBacktestServiceWithDeps(deps, mustParse(t, "2024-03-01"))
	res, err := svc.Run(context.Background(), nil, BacktestRequest{
		Period: "all", InitialCash: 1_000_000, Rebalance: "none",
		Basket: []BasketItem{{"id1", 100}},
	})
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	if res.ClampedStart != "2024-01-16" {
		t.Errorf("clampedStart=%s, want 2024-01-16 (KOSPI firstAvailable)", res.ClampedStart)
	}
}

func TestRun_FullCoverage_NoSpuriousWarning(t *testing.T) {
	days := genDays(t, "2024-01-01", 40)
	deps := krStockDeps(days, []string{"id1", "id2"})
	// 모든 소스(레그·KOSPI·SPX)가 윈도우 전체를 커버 → 경고 0건이어야 한다.
	svc := newBacktestServiceWithDeps(deps, mustParse(t, "2024-02-15"))
	res, err := svc.Run(context.Background(), nil, BacktestRequest{
		Period: "all", InitialCash: 1_000_000, Rebalance: "none",
		Basket: []BasketItem{{"id1", 50}, {"id2", 50}},
	})
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	if len(res.CoverageWarnings) != 0 {
		t.Errorf("full coverage인데 스푸리어스 경고 발생: %+v", res.CoverageWarnings)
	}
}

func TestRun_BenchmarkClamp_EmitsWarning(t *testing.T) {
	days := genDays(t, "2024-01-01", 50)
	deps := krStockDeps(days, []string{"id1"})
	// KOSPI 벤치마크가 2024-01-16(days[15])부터만 존재 → 클램프 지배 + 경고 방출 기대.
	deps.bench["KOSPI"] = benchPts(days[15:], 2000)
	svc := newBacktestServiceWithDeps(deps, mustParse(t, "2024-03-01"))
	res, err := svc.Run(context.Background(), nil, BacktestRequest{
		Period: "all", InitialCash: 1_000_000, Rebalance: "none",
		Basket: []BasketItem{{"id1", 100}},
	})
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	found := false
	for _, w := range res.CoverageWarnings {
		if w.Symbol == "KOSPI" && w.FirstAvailable == "2024-01-16" {
			found = true
		}
	}
	if !found {
		t.Errorf("KOSPI 커버리지 경고 누락: %+v", res.CoverageWarnings)
	}
}

func TestRestrictForwardFilled_SeedsFromBeforeSentinel(t *testing.T) {
	// "__before"(윈도우 직전 폴백)만 있고 clampStart 당일 실데이터가 없을 때,
	// densify 결과가 폴백값으로 시드되어야 한다(엔진 lookupFxForward 계약과 일치 → 1.0 오평가 방지).
	allDays := []string{"2024-01-01", "2024-01-02", "2024-01-03"}
	clamped := []string{"2024-01-01", "2024-01-02", "2024-01-03"}
	sparse := map[string]float64{"__before": 1300, "2024-01-03": 1430}
	out := restrictForwardFilled(sparse, allDays, clamped)
	approx(t, out["2024-01-01"], 1300, 1e-9, "t0 seeded from __before")
	approx(t, out["2024-01-02"], 1300, 1e-9, "carry forward __before")
	approx(t, out["2024-01-03"], 1430, 1e-9, "real value overrides __before")
}
