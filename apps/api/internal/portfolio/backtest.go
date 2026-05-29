package portfolio

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

// simulate — 순수 NAV/유닛 펀드 회계 시뮬 (DB 무관). T1: 일시불만. 적립·리밸런싱은 T2·T3.
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

	// t0 초기 매수 — 각 leg에 Initial*weight 배분
	for i, leg := range legs {
		alloc := plan.Initial * leg.Weight
		px := closeAt(leg.Closes, tradingDays, 0)
		fx := legFx(leg, tradingDays, 0)
		if px > 0 && fx > 0 {
			shares[i] = alloc / (px * fx)
		}
	}
	fundUnits := plan.Initial   // NAV(t0) = V/units = 1.0
	totalContributed := plan.Initial
	cashflows := []Cashflow{{Amount: -plan.Initial, Date: t0}}

	for idx, d := range tradingDays {
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
