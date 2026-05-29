package portfolio

import (
	"context"
	"math"
	"time"

	"github.com/quotient/quotient/apps/api/internal/db"
)

// Rebalance는 리밸런싱 주기 (없음·분기·반기·연).
type Rebalance int

const (
	RebalanceNone Rebalance = iota
	RebalanceQuarterly
	RebalanceSemiannual
	RebalanceAnnual
)

// Leg는 바스켓 한 종목 (정규화 비중 + 시점별 종가/환율). 벤치마크도 leg로 표현.
type Leg struct {
	Weight  float64            // 정규화된 목표 비중 (Σ=1.0)
	Closes  map[string]float64 // date("2006-01-02") → 종가 (매매 통화 기준)
	FxToKRW map[string]float64 // date → 환율 (KRW leg은 nil/빈 맵 → 1.0)
}

// Plan은 현금 투입 규칙. Monthly==0이면 일시불(lump).
type Plan struct {
	Initial float64 // 초기 자금 (KRW)
	Monthly float64 // 월 적립금 (KRW)
}

// Cashflow는 XIRR 입력 (음수=투입 유출, 양수=최종 평가액 유입).
type Cashflow struct {
	Amount float64
	Date   string
}

// ValuePoint는 (일자, KRW 절대값). 알파의 SeriesPoint(value_pct)와 달리 절대 평가액.
type ValuePoint struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// SimOutput은 한 바스켓(전략 또는 벤치마크) 시뮬 결과.
type SimOutput struct {
	Equity           []ValuePoint // 일자별 평가액 (KRW) — 사용자 표시
	NAV              []ValuePoint // 일자별 NAV (시작 1.0) — 위험·수익 지표용
	Contributed      []ValuePoint // 일자별 누적 투입 원금 (KRW) — 기준선
	TotalContributed float64
	FinalEquity      float64
	Cashflows        []Cashflow
}

// BacktestDeps는 백테스트가 쓰는 데이터 접근만 노출하는 분리 인터페이스.
// 알파의 공유 Deps를 확장하지 않는다(EquityDeps 선례). PgDeps가 이를 만족.
type BacktestDeps interface {
	TradingDays(ctx context.Context, pool db.Executor, since, until time.Time) ([]string, error)
	InstrumentClosesOnDates(ctx context.Context, pool db.Executor, instrumentID string, dates []string) (map[string]float64, error)
	FxRatesOnDates(ctx context.Context, pool db.Executor, currency string, dates []string) (map[string]float64, error)
	BenchmarkSeries(ctx context.Context, pool db.Executor, symbol string, dates []string) ([]PricePoint, error)
	InstrumentsMeta(ctx context.Context, pool db.Executor, ids []string) (map[string]InstrumentMeta, error)
}

// InstrumentMeta — 바스켓 종목의 통화·자산군 (fx 적용 + INDEX/FX/CASH 가드용).
type InstrumentMeta struct {
	Symbol     string
	Name       string
	Currency   string // "KRW" | "USD"
	AssetClass string // "KR_STOCK" | "US_STOCK" | "INDEX" | "FX" | "CASH" ...
}

// closeAt — idx 일자 종가 없으면 직전 영업일로 후퇴, 없으면 "__before", 그래도 없으면 0.
func closeAt(m map[string]float64, dates []string, idx int) float64 {
	for i := idx; i >= 0; i-- {
		if v, ok := m[dates[i]]; ok && v > 0 {
			return v
		}
	}
	if v, ok := m["__before"]; ok && v > 0 {
		return v
	}
	return 0
}

// legFx — leg 환율 전진 채움. KRW leg(FxToKRW 비어있음)은 1.0. lookupFxForward 재사용.
func legFx(leg Leg, dates []string, idx int) float64 {
	if len(leg.FxToKRW) == 0 {
		return 1.0
	}
	return lookupFxForward(leg.FxToKRW, dates, idx)
}

// portValue — idx 일자 전체 leg 평가액 합 (KRW).
func portValue(legs []Leg, shares []float64, dates []string, idx int) float64 {
	var v float64
	for i, leg := range legs {
		px := closeAt(leg.Closes, dates, idx)
		fx := legFx(leg, dates, idx)
		v += shares[i] * px * fx
	}
	return v
}

func lastDayOfMonth(y int, m time.Month) int {
	return time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// addMonthsClamped — t0(YYYY-MM-DD) + k개월. 월말 오버플로(1/31+1mo=3/3) 방지: 일자를 대상 월 말일로 클램프.
// 누적 커서 대신 t0에서 매번 새로 계산 → 드리프트 없음.
func addMonthsClamped(t0 string, k int) string {
	t, err := time.Parse("2006-01-02", t0)
	if err != nil {
		return t0
	}
	total := int(t.Month()) - 1 + k
	y := t.Year() + total/12
	mi := total % 12
	if mi < 0 {
		mi += 12
		y--
	}
	m := time.Month(mi + 1)
	day := t.Day()
	if last := lastDayOfMonth(y, m); day > last {
		day = last
	}
	return time.Date(y, m, day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
}

// rebalanceMonths — 리밸런싱 주기를 개월 수로 변환. RebalanceNone이면 0.
func rebalanceMonths(rb Rebalance) int {
	switch rb {
	case RebalanceQuarterly:
		return 3
	case RebalanceSemiannual:
		return 6
	case RebalanceAnnual:
		return 12
	}
	return 0
}

// simulate — 순수 NAV/유닛 펀드 회계 시뮬 (DB 무관). T1: 일시불. T2: DCA 적립. T3: 리밸런싱.
func simulate(tradingDays []string, legs []Leg, plan Plan, rb Rebalance) SimOutput {
	n := len(tradingDays)
	out := SimOutput{
		Equity:      make([]ValuePoint, 0, n),
		NAV:         make([]ValuePoint, 0, n),
		Contributed: make([]ValuePoint, 0, n),
	}
	if n == 0 || plan.Initial <= 0 {
		return out
	}
	shares := make([]float64, len(legs))
	t0 := tradingDays[0]

	for i, leg := range legs {
		alloc := plan.Initial * leg.Weight
		px := closeAt(leg.Closes, tradingDays, 0)
		fx := legFx(leg, tradingDays, 0)
		if px > 0 && fx > 0 {
			shares[i] = alloc / (px * fx)
		}
	}
	fundUnits := plan.Initial
	totalContributed := plan.Initial
	cashflows := []Cashflow{{Amount: -plan.Initial, Date: t0}}

	contribCount := 0
	nextContrib := ""
	if plan.Monthly > 0 {
		contribCount = 1
		nextContrib = addMonthsClamped(t0, 1)
	}
	rbMonths := rebalanceMonths(rb)
	rebalCount := 0
	nextRebal := ""
	if rbMonths > 0 {
		rebalCount = 1
		nextRebal = addMonthsClamped(t0, rbMonths)
	}

	for idx, d := range tradingDays {
		// 1) 적립 먼저 — 현재 NAV로 유닛 발행 → NAV 불변.
		if idx > 0 && nextContrib != "" && d >= nextContrib {
			nav := portValue(legs, shares, tradingDays, idx) / fundUnits
			if nav > 0 {
				fundUnits += plan.Monthly / nav
			}
			for i, leg := range legs {
				px := closeAt(leg.Closes, tradingDays, idx)
				fx := legFx(leg, tradingDays, idx)
				if px > 0 && fx > 0 {
					shares[i] += (plan.Monthly * leg.Weight) / (px * fx)
				}
			}
			totalContributed += plan.Monthly
			cashflows = append(cashflows, Cashflow{Amount: -plan.Monthly, Date: d})
			contribCount++
			nextContrib = addMonthsClamped(t0, contribCount)
		}

		// 2) 리밸런싱 나중 — 적립 반영 후 V 재계산(캐시 금지). 목표 비중 복원, V·NAV 불변.
		if idx > 0 && nextRebal != "" && d >= nextRebal {
			v := portValue(legs, shares, tradingDays, idx)
			for i, leg := range legs {
				px := closeAt(leg.Closes, tradingDays, idx)
				fx := legFx(leg, tradingDays, idx)
				if px > 0 && fx > 0 {
					shares[i] = (v * leg.Weight) / (px * fx)
				}
			}
			rebalCount++
			nextRebal = addMonthsClamped(t0, rbMonths*rebalCount)
		}

		v := portValue(legs, shares, tradingDays, idx)
		nav := v / fundUnits
		out.Equity = append(out.Equity, ValuePoint{Date: d, Value: v})
		out.NAV = append(out.NAV, ValuePoint{Date: d, Value: nav})
		out.Contributed = append(out.Contributed, ValuePoint{Date: d, Value: totalContributed})
	}

	out.TotalContributed = totalContributed
	out.FinalEquity = out.Equity[n-1].Value
	cashflows = append(cashflows, Cashflow{Amount: out.FinalEquity, Date: tradingDays[n-1]})
	out.Cashflows = cashflows
	return out
}

// SeriesMetrics는 한 시계열(전략 또는 개별 벤치마크) 지표. excess는 Service.Run에서 별도 계산.
type SeriesMetrics struct {
	TotalReturnPct float64  `json:"total_return_pct"`
	CAGRPct        *float64 `json:"cagr_pct"` // XIRR 실패 시 null
	MDDPct         float64  `json:"mdd_pct"`
	VolatilityPct  float64  `json:"volatility_pct"`
	TWRPct         float64  `json:"twr_pct"`
}

// maxDrawdown — NAV 시계열 최대 고점-저점 낙폭 (음수 %).
func maxDrawdown(nav []ValuePoint) float64 {
	if len(nav) == 0 {
		return 0
	}
	peak := nav[0].Value
	maxDD := 0.0
	for _, p := range nav {
		if p.Value > peak {
			peak = p.Value
		}
		if peak > 0 {
			if dd := (p.Value - peak) / peak; dd < maxDD {
				maxDD = dd
			}
		}
	}
	return maxDD * 100
}

// annualizedVol — NAV 일간 수익률 표본표준편차 × √252 (%).
// nav<3이면 0 반환은 방어용일 뿐: 클램프 하한(minBacktestDays=30)이 실 실행 시 항상 ≥30 NAV 포인트를 보장 → 변동성은 항상 계산됨.
func annualizedVol(nav []ValuePoint) float64 {
	if len(nav) < 3 {
		return 0
	}
	rets := make([]float64, 0, len(nav)-1)
	for i := 1; i < len(nav); i++ {
		if prev := nav[i-1].Value; prev > 0 {
			rets = append(rets, (nav[i].Value-prev)/prev)
		}
	}
	if len(rets) < 2 {
		return 0
	}
	var mean float64
	for _, r := range rets {
		mean += r
	}
	mean /= float64(len(rets))
	var ss float64
	for _, r := range rets {
		ss += (r - mean) * (r - mean)
	}
	variance := ss / float64(len(rets)-1)
	return math.Sqrt(variance) * math.Sqrt(252) * 100
}

// metrics — 한 시계열의 {총수익률, CAGR(XIRR), MDD, 변동성, TWR}.
func metrics(out SimOutput) SeriesMetrics {
	m := SeriesMetrics{
		MDDPct:        maxDrawdown(out.NAV),
		VolatilityPct: annualizedVol(out.NAV),
	}
	if out.TotalContributed > 0 {
		m.TotalReturnPct = (out.FinalEquity - out.TotalContributed) / out.TotalContributed * 100
	}
	if len(out.NAV) > 0 {
		m.TWRPct = (out.NAV[len(out.NAV)-1].Value - 1.0) * 100
	}
	if r := xirr(out.Cashflows); r != nil {
		pct := *r * 100
		m.CAGRPct = &pct
	}
	return m
}

func yearFrac(d0, dk string) float64 {
	a, _ := time.Parse("2006-01-02", d0)
	b, _ := time.Parse("2006-01-02", dk)
	return b.Sub(a).Hours() / 24.0 / 365.0
}

func xirrNPV(cfs []Cashflow, r float64) float64 {
	if len(cfs) == 0 {
		return 0
	}
	d0 := cfs[0].Date
	var s float64
	for _, cf := range cfs {
		s += cf.Amount / math.Pow(1+r, yearFrac(d0, cf.Date))
	}
	return s
}

func xirrDeriv(cfs []Cashflow, r float64) float64 {
	if len(cfs) == 0 {
		return 0
	}
	d0 := cfs[0].Date
	var s float64
	for _, cf := range cfs {
		yf := yearFrac(d0, cf.Date)
		s += cf.Amount * (-yf) / math.Pow(1+r, yf+1)
	}
	return s
}

// xirr — 머니가중 연이율. Newton(초기 0.1, 100회, |f|<1e-6) → 이분법 폴백 [-0.99,10] → nil.
func xirr(cfs []Cashflow) *float64 {
	if len(cfs) < 2 {
		return nil
	}
	hasNeg, hasPos := false, false
	for _, cf := range cfs {
		if cf.Amount < 0 {
			hasNeg = true
		} else if cf.Amount > 0 {
			hasPos = true
		}
	}
	if !hasNeg || !hasPos {
		return nil
	}

	r := 0.1
	for i := 0; i < 100; i++ {
		f := xirrNPV(cfs, r)
		if math.Abs(f) < 1e-6 {
			return &r
		}
		d := xirrDeriv(cfs, r)
		if d == 0 || math.IsNaN(d) || math.IsInf(d, 0) {
			break
		}
		next := r - f/d
		if next <= -0.999999 {
			next = (-0.999999 + r) / 2
		}
		if math.IsNaN(next) || math.IsInf(next, 0) {
			break
		}
		r = next
	}

	// 이분법 폴백
	lo, hi := -0.99, 10.0
	flo := xirrNPV(cfs, lo)
	fhi := xirrNPV(cfs, hi)
	if flo == 0 {
		return &lo
	}
	if fhi == 0 {
		return &hi
	}
	if (flo < 0) == (fhi < 0) {
		return nil
	}
	for i := 0; i < 200; i++ {
		mid := (lo + hi) / 2
		fmid := xirrNPV(cfs, mid)
		if math.Abs(fmid) < 1e-6 {
			return &mid
		}
		if (fmid < 0) == (flo < 0) {
			lo, flo = mid, fmid
		} else {
			hi, fhi = mid, fmid
		}
	}
	mid := (lo + hi) / 2
	return &mid
}

const minBacktestDays = 30

// --- 요청/응답 타입 ---

type BacktestRequest struct {
	Period      string       `json:"period"`
	InitialCash float64      `json:"initial_cash"`
	Monthly     float64      `json:"monthly_contribution"`
	Basket      []BasketItem `json:"basket"`
	Rebalance   string       `json:"rebalance"`
}

type BasketItem struct {
	InstrumentID string  `json:"instrument_id"`
	Weight       float64 `json:"weight"`
}

type NormalizedLeg struct {
	InstrumentID string  `json:"instrument_id"`
	Symbol       string  `json:"symbol"`
	Name         string  `json:"name"`
	Weight       float64 `json:"weight"`
}

type BenchmarkResult struct {
	EquitySeries []ValuePoint  `json:"equity_series"`
	Metrics      SeriesMetrics `json:"metrics"`
}

type BenchmarkSet struct {
	Kospi      BenchmarkResult `json:"kospi"`
	Spx        BenchmarkResult `json:"spx"`
	SixtyForty BenchmarkResult `json:"sixty_forty"`
}

// StrategyMetrics — 전략 metrics(§8). 벤치마크의 SeriesMetrics와 달리 excess·투입·최종 포함, twr 제외.
type StrategyMetrics struct {
	TotalReturnPct   float64  `json:"total_return_pct"`
	CagrPct          *float64 `json:"cagr_pct"`
	MddPct           float64  `json:"mdd_pct"`
	VolatilityPct    float64  `json:"volatility_pct"`
	ExcessVs6040Pct  float64  `json:"excess_vs_6040_pct"`
	TotalContributed float64  `json:"total_contributed"`
	FinalEquity      float64  `json:"final_equity"`
}

type CoverageWarning struct {
	Symbol         string `json:"symbol"`
	FirstAvailable string `json:"first_available"`
	Message        string `json:"message"`
}

type BacktestResult struct {
	ClampedStart      string            `json:"clamped_start"`
	End               string            `json:"end"`
	NormalizedBasket  []NormalizedLeg   `json:"normalized_basket"`
	EquitySeries      []ValuePoint      `json:"equity_series"`
	ContributedSeries []ValuePoint      `json:"contributed_series"`
	Benchmarks        BenchmarkSet      `json:"benchmarks"`
	Metrics           StrategyMetrics   `json:"metrics"`
	CoverageWarnings  []CoverageWarning `json:"coverage_warnings"`
}

// ValidationError — 422 구분 가능 에러. Code ∈ {VALIDATION, ASSET_NOT_SUPPORTED}.
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string { return e.Code + ": " + e.Message }

// --- 파싱/전진채움 헬퍼 ---

func parseRebalance(s string) (Rebalance, error) {
	switch s {
	case "", "none":
		return RebalanceNone, nil
	case "quarterly":
		return RebalanceQuarterly, nil
	case "semiannual":
		return RebalanceSemiannual, nil
	case "annual":
		return RebalanceAnnual, nil
	}
	return RebalanceNone, &ValidationError{Code: "VALIDATION", Message: "리밸런싱 값이 올바르지 않습니다"}
}

func periodStart(period string, today time.Time) (time.Time, error) {
	switch period {
	case "1Y":
		return today.AddDate(-1, 0, 0), nil
	case "3Y":
		return today.AddDate(-3, 0, 0), nil
	case "5Y", "all":
		return today.AddDate(-5, 0, 0), nil
	}
	return time.Time{}, &ValidationError{Code: "VALIDATION", Message: "기간 값이 올바르지 않습니다"}
}

func benchFirst(pts []PricePoint) string {
	if len(pts) == 0 {
		return ""
	}
	return pts[0].Date // BenchmarkSeries는 order by date (오름차순)
}

func benchCloseMap(pts []PricePoint) map[string]float64 {
	m := make(map[string]float64, len(pts))
	for _, p := range pts {
		m[p.Date] = p.Close
	}
	return m
}

// restrictForwardFilled — 희소 맵을 clampedDays 축으로 조밀화(전진 채움).
// allDays(⊇clampedDays)를 순회하며 직전 유효값을 추적 → clampedDays 첫날(t0)도 클램프 이전 데이터로 시드.
func restrictForwardFilled(sparse map[string]float64, allDays, clampedDays []string) map[string]float64 {
	clampSet := make(map[string]bool, len(clampedDays))
	for _, d := range clampedDays {
		clampSet[d] = true
	}
	out := make(map[string]float64, len(clampedDays))
	last := 0.0
	have := false
	for _, d := range allDays {
		if v, ok := sparse[d]; ok && v > 0 {
			last = v
			have = true
		}
		if clampSet[d] && have {
			out[d] = last
		}
	}
	return out
}

// --- 서비스 ---

type BacktestService struct {
	deps BacktestDeps
	now  func() time.Time
}

func newBacktestServiceWithDeps(deps BacktestDeps, fixedNow time.Time) *BacktestService {
	return &BacktestService{deps: deps, now: func() time.Time { return fixedNow }}
}

// Run — 바스켓·벤치마크 해석 + 클램프 + 4× simulate + 지표. pool은 슈퍼유저 공개 read(인증만, RLS X).
func (s *BacktestService) Run(ctx context.Context, pool db.Executor, req BacktestRequest) (*BacktestResult, error) {
	rb, err := parseRebalance(req.Rebalance)
	if err != nil {
		return nil, err
	}
	today := s.now()
	requestedStart, err := periodStart(req.Period, today)
	if err != nil {
		return nil, err
	}
	reqStartStr := requestedStart.Format("2006-01-02")

	// 메타 조회 + 자산군 가드 + 비중 합
	ids := make([]string, len(req.Basket))
	for i, b := range req.Basket {
		ids[i] = b.InstrumentID
	}
	metas, err := s.deps.InstrumentsMeta(ctx, pool, ids)
	if err != nil {
		return nil, err
	}
	var sumW float64
	for _, b := range req.Basket {
		m, ok := metas[b.InstrumentID]
		if !ok {
			return nil, &ValidationError{Code: "VALIDATION", Message: "종목을 찾을 수 없습니다"}
		}
		switch m.AssetClass {
		case "INDEX", "FX", "CASH":
			return nil, &ValidationError{Code: "ASSET_NOT_SUPPORTED", Message: "지수·환율은 백테스트 불가"}
		}
		sumW += b.Weight
	}
	if sumW <= 0 {
		return nil, &ValidationError{Code: "VALIDATION", Message: "비중은 0보다 커야 합니다"}
	}

	allDays, err := s.deps.TradingDays(ctx, pool, requestedStart, today)
	if err != nil {
		return nil, err
	}
	if len(allDays) == 0 {
		return nil, &InsufficientDataError{Reason: "no_trading_days", MinDays: minBacktestDays, CurrentDays: 0}
	}

	type legData struct {
		item   BasketItem
		meta   InstrumentMeta
		closes map[string]float64
		fx     map[string]float64
		first  string
	}
	clampStart := reqStartStr // 문자열(YYYY-MM-DD) 사전순 비교 = 시간순 max
	rows := make([]legData, 0, len(req.Basket))
	for _, b := range req.Basket {
		m := metas[b.InstrumentID]
		closes, err := s.deps.InstrumentClosesOnDates(ctx, pool, b.InstrumentID, allDays)
		if err != nil {
			return nil, err
		}
		fx, err := s.deps.FxRatesOnDates(ctx, pool, m.Currency, allDays)
		if err != nil {
			return nil, err
		}
		first := firstAvailable(closes)
		if first == "" {
			return nil, &InsufficientDataError{Reason: "no_prices", MinDays: minBacktestDays, CurrentDays: 0}
		}
		if first > clampStart {
			clampStart = first
		}
		if m.Currency != "KRW" {
			if ff := firstAvailable(fx); ff != "" && ff > clampStart {
				clampStart = ff // USD leg fx firstAvailable도 포함 (§5-5)
			}
		}
		rows = append(rows, legData{item: b, meta: m, closes: closes, fx: fx, first: first})
	}

	kospiPts, err := s.deps.BenchmarkSeries(ctx, pool, "KOSPI", allDays)
	if err != nil {
		return nil, err
	}
	spxPts, err := s.deps.BenchmarkSeries(ctx, pool, "SPX", allDays)
	if err != nil {
		return nil, err
	}
	usdFx, err := s.deps.FxRatesOnDates(ctx, pool, "USD", allDays)
	if err != nil {
		return nil, err
	}
	if kf := benchFirst(kospiPts); kf != "" && kf > clampStart {
		clampStart = kf
	}
	if sf := benchFirst(spxPts); sf != "" && sf > clampStart {
		clampStart = sf
	}

	clampedDays := make([]string, 0, len(allDays))
	for _, d := range allDays {
		if d >= clampStart {
			clampedDays = append(clampedDays, d)
		}
	}
	if len(clampedDays) < minBacktestDays {
		return nil, &InsufficientDataError{Reason: "backtest_window_too_short", MinDays: minBacktestDays, CurrentDays: len(clampedDays)}
	}

	stratLegs := make([]Leg, len(rows))
	normalized := make([]NormalizedLeg, len(rows))
	var warnings []CoverageWarning
	for i, ld := range rows {
		w := ld.item.Weight / sumW
		var fxMap map[string]float64
		if ld.meta.Currency != "KRW" {
			fxMap = restrictForwardFilled(ld.fx, allDays, clampedDays)
		}
		stratLegs[i] = Leg{
			Weight:  w,
			Closes:  restrictForwardFilled(ld.closes, allDays, clampedDays),
			FxToKRW: fxMap,
		}
		normalized[i] = NormalizedLeg{InstrumentID: ld.item.InstrumentID, Symbol: ld.meta.Symbol, Name: ld.meta.Name, Weight: w}
		if ld.first > reqStartStr {
			warnings = append(warnings, CoverageWarning{
				Symbol:         ld.meta.Symbol,
				FirstAvailable: ld.first,
				Message:        "데이터가 " + ld.first + "부터 존재해 시작일을 조정했습니다",
			})
		}
	}

	kospiCloses := restrictForwardFilled(benchCloseMap(kospiPts), allDays, clampedDays)
	spxCloses := restrictForwardFilled(benchCloseMap(spxPts), allDays, clampedDays)
	spxFx := restrictForwardFilled(usdFx, allDays, clampedDays)
	kospiLeg := Leg{Weight: 1.0, Closes: kospiCloses}
	spxLeg := Leg{Weight: 1.0, Closes: spxCloses, FxToKRW: spxFx}
	leg6040 := []Leg{{Weight: 0.6, Closes: kospiCloses}, {Weight: 0.4, Closes: spxCloses, FxToKRW: spxFx}}

	plan := Plan{Initial: req.InitialCash, Monthly: req.Monthly}
	strat := simulate(clampedDays, stratLegs, plan, rb)
	kospi := simulate(clampedDays, []Leg{kospiLeg}, plan, rb)
	spx := simulate(clampedDays, []Leg{spxLeg}, plan, rb)
	s6040 := simulate(clampedDays, leg6040, plan, rb)

	sm := metrics(strat)
	m6040 := metrics(s6040)

	return &BacktestResult{
		ClampedStart:      clampStart,
		End:               today.Format("2006-01-02"),
		NormalizedBasket:  normalized,
		EquitySeries:      strat.Equity,
		ContributedSeries: strat.Contributed,
		Benchmarks: BenchmarkSet{
			Kospi:      BenchmarkResult{EquitySeries: kospi.Equity, Metrics: metrics(kospi)},
			Spx:        BenchmarkResult{EquitySeries: spx.Equity, Metrics: metrics(spx)},
			SixtyForty: BenchmarkResult{EquitySeries: s6040.Equity, Metrics: m6040},
		},
		Metrics: StrategyMetrics{
			TotalReturnPct:   sm.TotalReturnPct,
			CagrPct:          sm.CAGRPct,
			MddPct:           sm.MDDPct,
			VolatilityPct:    sm.VolatilityPct,
			ExcessVs6040Pct:  sm.TWRPct - m6040.TWRPct,
			TotalContributed: strat.TotalContributed,
			FinalEquity:      strat.FinalEquity,
		},
		CoverageWarnings: warnings,
	}, nil
}
