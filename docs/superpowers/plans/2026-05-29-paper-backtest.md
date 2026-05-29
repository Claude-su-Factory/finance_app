# Paper Trading 백테스트 (서브시스템 B) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 선언적 바스켓(종목+비중) + 2축 전략(투입 방식·리밸런싱)을 과거 데이터로 시뮬레이션해 평가액 곡선·벤치마크 3종 비교·위험/수익 지표를 산출하는 무상태 백테스트 엔진과 `/app/backtest` 페이지를 구현한다.

**Architecture:** 순수 함수 `simulate()`(NAV/유닛 펀드 회계, DB 무관) + `metrics()`/`xirr()`를 `internal/portfolio/backtest.go`에 두고, `Service.Run`이 DB 해석(가격·환율·메타 조회)·클램프·벤치마크 4회 호출을 오케스트레이션한다. 전략과 벤치마크(KOSPI·SPX·60/40)를 동일 현금흐름·동일 리밸런싱 설정으로 같은 `simulate()`에 4번 통과시켜 apples-to-apples 비교. 핸들러는 사용자 데이터 미접근(슈퍼유저 풀 읽기, RLS 불필요)이며 인증 게이트만. 프런트는 `/app/backtest` 폼→결과 페이지.

**Tech Stack:** Go 1.25 (chi v5 + pgx v5), Next.js 16 (App Router) + React + recharts, Supabase Postgres. 테스트: Go `testing` + `//go:build integration`, web `vitest` + `@testing-library/react`.

---

## File Structure

**Backend (`apps/api`):**
- Create `internal/portfolio/backtest.go` — 신규 패키지 파일(같은 `portfolio` 패키지). 타입(`Leg`, `Plan`, `Rebalance`, `Cashflow`, `ValuePoint`, `SimOutput`, `SeriesMetrics`, `InstrumentMeta`, `BacktestDeps`, `ValidationError`), 순수 함수(`simulate`, `metrics`, `maxDrawdown`, `annualizedVol`, `xirr`, `closeAt`, `legFx`, `portValue`, `addMonthsClamped`, `rebalanceMonths`, `restrictForwardFilled`), `BacktestService.Run`.
- Create `internal/portfolio/backtest_test.go` — 엔진 단위 + 서비스/클램프 테스트. **`package portfolio`(white-box)** — 비공개 `simulate`/`metrics`/`xirr` 직접 호출 + `BacktestDeps` 모킹. (기존 `alpha_test.go`는 `portfolio_test`라 다른 패키지지만 같은 디렉터리 공존 가능.)
- Create `internal/portfolio/backtest_integration_test.go` — `//go:build integration`, 시드 가격으로 E2E.
- Modify `internal/portfolio/pg_deps.go` — `PgDeps.InstrumentsMeta` 메서드 추가(딱 1개).
- Create `internal/handlers/backtest.go` — `BacktestHandler`, `BacktestRunner` 인터페이스, 구조적 검증.
- Create `internal/handlers/backtest_test.go` — 핸들러 테스트(fake runner).
- Modify `internal/router/router.go` — `backtestHandler` 파라미터 + `POST /v1/backtest/run` 라우트.
- Modify `cmd/server/main.go` — 서비스·핸들러 wiring.

**Frontend (`apps/web`):**
- Create `lib/api/backtest.ts` — 요청/응답 타입 + `runBacktest()` (union 반환: 결과 | 422 에러).
- Create `components/backtest/BacktestPage.tsx` — `"use client"` 오케스트레이터.
- Create `components/backtest/BacktestForm.tsx` — 폼(period·cash·basket·rebalance) 컨테이너.
- Create `components/backtest/PeriodPicker.tsx`, `CashInputs.tsx`, `BasketBuilder.tsx`, `RebalanceSelect.tsx`.
- Create `components/backtest/BacktestResults.tsx` + `MetricCards.tsx`, `BacktestEquityChart.tsx`, `CompareTable.tsx`, `CoverageNotice.tsx`.
- Create `components/backtest/BacktestPage.test.tsx` — vitest.
- Create `app/app/backtest/page.tsx` — thin wrapper.
- Modify `components/shell/Sidebar.tsx` — 백테스트 아이콘 1개 추가.

**문서 (완료 시):** `docs/STATUS.md`, `docs/ROADMAP.md`, `docs/ARCHITECTURE.md`, `docs/USER_ACTIONS.md`.

---

## Task 순서 개요

T1 엔진 타입 + `simulate()` 일시불 Buy&Hold → T2 DCA 적립 → T3 리밸런싱(+같은 날 순서) → T4 `xirr()` → T5 `metrics()`(CAGR는 `xirr()` 호출하므로 xirr 먼저) → T6 `BacktestDeps`+`InstrumentMeta`+`PgDeps.InstrumentsMeta` → T7 `BacktestService.Run`(클램프·벤치마크×4) → T8 핸들러 → T9 router+main wiring → T10 통합 테스트 → **(프런트)** T11 `lib/api/backtest.ts` → T12 입력 폼 컴포넌트(PeriodPicker·CashInputs·RebalanceSelect·BasketBuilder·BacktestForm) → T13 결과 컴포넌트(MetricCards·BacktestEquityChart·CompareTable·CoverageNotice·BacktestResults) → T14 페이지+라우트+Sidebar → T15 프런트 vitest → T16 문서 4종.

각 Go 태스크(T1~T10)는 TDD(실패 테스트 → 최소 구현 → 통과 → 커밋). 프런트(T11~T14)는 컴포넌트 작성 후 T15에서 일괄 테스트.

---

### Task 1: 엔진 타입 + 헬퍼 + 일시불 `simulate()`

순수 함수의 뼈대. 타입 정의 + 좌표 헬퍼(`closeAt`/`legFx`/`portValue`) + 일시불(Buy&Hold) 시뮬. 적립·리밸런싱은 T2·T3에서 추가. **단위 테스트는 비공개 함수 접근을 위해 `package portfolio`(white-box)** — 기존 `alpha_test.go`(`package portfolio_test`)와 다른 패키지로 같은 디렉터리에 공존 가능.

**Files:**
- Create: `apps/api/internal/portfolio/backtest.go`
- Test: `apps/api/internal/portfolio/backtest_test.go`

- [ ] **Step 1: 실패 테스트 작성**

`apps/api/internal/portfolio/backtest_test.go`:

```go
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
```

- [ ] **Step 2: 테스트 실패 확인**

Run: `cd apps/api && go test ./internal/portfolio/ -run TestSimulate_ -v`
Expected: FAIL — `undefined: simulate`, `undefined: Leg` 등 컴파일 에러.

- [ ] **Step 3: 최소 구현 작성**

`apps/api/internal/portfolio/backtest.go`:

```go
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
```

> `rb` 파라미터는 T1에서 미사용(Go는 미사용 함수 파라미터 허용). T3에서 리밸런싱 분기로 사용.

- [ ] **Step 4: 테스트 통과 확인**

Run: `cd apps/api && go test ./internal/portfolio/ -run TestSimulate_ -v`
Expected: PASS (4 tests).

- [ ] **Step 5: 커밋**

```bash
git add apps/api/internal/portfolio/backtest.go apps/api/internal/portfolio/backtest_test.go
git commit -m "feat(api): 백테스트 엔진 타입 + 일시불 simulate()"
```

### Task 2: 월 적립(DCA) — NAV 불변 유닛 발행 + 월말 안전 커서

`simulate`에 적립 로직 추가. 적립일에 현재 NAV로 유닛을 발행해 NAV가 점프하지 않음(§5-3 증명). 발생일은 t0+1개월 커서, 월말(1/31)은 대상 월 말일로 클램프. `time` import 추가.

**Files:**
- Modify: `apps/api/internal/portfolio/backtest.go` (`simulate` 교체 + 헬퍼 2개 + import)
- Test: `apps/api/internal/portfolio/backtest_test.go` (테스트 2개 추가)

- [ ] **Step 1: 실패 테스트 작성** — `backtest_test.go`에 추가 (import에 `"time"` 추가):

```go
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
```

- [ ] **Step 2: 테스트 실패 확인**

Run: `cd apps/api && go test ./internal/portfolio/ -run TestSimulate_DCA -v`
Expected: FAIL — `TotalContributed` = Initial만(적립 미반영), NAV 점프 또는 적립 0회.

- [ ] **Step 3: 구현 — `backtest.go` 수정**

import 블록 추가:
```go
import "time"
```

`simulate` 아래(또는 위)에 헬퍼 2개 추가:
```go
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
```

`simulate`를 아래로 교체(적립 커서 init + 루프 내 적립 블록 추가):
```go
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

	// 적립 커서 — t0 당일은 적립 없음, 첫 적립은 t0+1개월(또는 직후 첫 영업일).
	contribCount := 0
	nextContrib := ""
	if plan.Monthly > 0 {
		contribCount = 1
		nextContrib = addMonthsClamped(t0, 1)
	}

	for idx, d := range tradingDays {
		// 적립: 현재 NAV로 유닛 발행 → NAV 불변. 발생일 1회만(단일 advance).
		// 전제: tradingDays는 일 단위 공통 축이므로 연속 영업일이 여러 앵커를 건너뛰지 않음(다월 공백 없음).
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
```

- [ ] **Step 4: 테스트 통과 확인**

Run: `cd apps/api && go test ./internal/portfolio/ -run TestSimulate -v`
Expected: PASS (T1 4개 + T2 2개 = 6개).

- [ ] **Step 5: 커밋**

```bash
git add apps/api/internal/portfolio/backtest.go apps/api/internal/portfolio/backtest_test.go
git commit -m "feat(api): 백테스트 DCA 적립 (NAV 불변 유닛 발행)"
```

### Task 3: 리밸런싱 + 같은 날 적립→리밸런싱 순서

`simulate`에 리밸런싱 추가. 발생일에 목표 비중 복원(신규 현금 없음, V·NAV 불변). 같은 날 적립과 겹치면 **적립 먼저**, 리밸런싱은 적립 반영 후 V를 다시 계산(하루치 V 캐시 재사용 금지 — 적립현금 미배분 버그 방지, §5-3 I3).

**Files:**
- Modify: `apps/api/internal/portfolio/backtest.go` (`rebalanceMonths` 추가 + `simulate` 교체)
- Test: `apps/api/internal/portfolio/backtest_test.go` (테스트 2개 추가)

- [ ] **Step 1: 실패 테스트 작성** — `backtest_test.go`에 추가:

```go
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
```

- [ ] **Step 2: 테스트 실패 확인**

Run: `cd apps/api && go test ./internal/portfolio/ -run TestSimulate_Rebalance -v && go test ./internal/portfolio/ -run TestSimulate_DCAplusRebalance -v`
Expected: FAIL — 리밸런싱 미구현(`rb` 미사용) → Equity[2]=1,250,000, Equity[1]=1.5M.

- [ ] **Step 3: 구현 — `backtest.go` 수정**

`rebalanceMonths` 헬퍼 추가:
```go
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
```

`simulate`를 아래로 교체(리밸런싱 커서 init + 루프 내 리밸런싱 블록 추가):
```go
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
```

- [ ] **Step 4: 테스트 통과 확인**

Run: `cd apps/api && go test ./internal/portfolio/ -run TestSimulate -v`
Expected: PASS (T1·T2·T3 합 8개).

- [ ] **Step 5: 커밋**

```bash
git add apps/api/internal/portfolio/backtest.go apps/api/internal/portfolio/backtest_test.go
git commit -m "feat(api): 백테스트 리밸런싱 (적립→리밸런싱 순서)"
```

### Task 4: `xirr()` — 머니가중 CAGR (Newton + 이분법 폴백 + null)

CAGR은 XIRR(현금흐름 NPV=0 연이율). Newton 반복 → 실패 시 이분법 폴백 → 그래도 실패 시 `nil`. `metrics()`(T5)가 사용하므로 먼저 구현. `math` import 추가.

**Files:**
- Modify: `apps/api/internal/portfolio/backtest.go` (`math` import + xirr 함수군)
- Test: `apps/api/internal/portfolio/backtest_test.go`

- [ ] **Step 1: 실패 테스트 작성** — `backtest_test.go`에 추가:

```go
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
```

- [ ] **Step 2: 테스트 실패 확인**

Run: `cd apps/api && go test ./internal/portfolio/ -run TestXIRR -v`
Expected: FAIL — `undefined: xirr`, `undefined: xirrNPV`.

- [ ] **Step 3: 구현 — `backtest.go` 수정**

import를 블록으로 교체(`math` 추가):
```go
import (
	"math"
	"time"
)
```

xirr 함수군 추가:
```go
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
```

> `flo`/`fhi`는 폴백 진입 직후 부호 검사에서 읽히므로 미사용 에러 없음. 루프 내 재대입은 Go에서 경고 대상 아님.

- [ ] **Step 4: 테스트 통과 확인**

Run: `cd apps/api && go test ./internal/portfolio/ -run TestXIRR -v`
Expected: PASS (3개).

- [ ] **Step 5: 커밋**

```bash
git add apps/api/internal/portfolio/backtest.go apps/api/internal/portfolio/backtest_test.go
git commit -m "feat(api): 백테스트 XIRR (Newton + 이분법 폴백)"
```

### Task 5: `metrics()` — 총수익률·CAGR·MDD·변동성·TWR

`SimOutput`에서 한 시계열 지표 산출. MDD·변동성은 **NAV 기준**(적립 왜곡 제거), CAGR은 `xirr(Cashflows)`. 초과수익은 per-series 아님 → T7 `Service.Run`에서 전략 TWR − 벤치마크 TWR로 계산.

> **스펙 대비 의도적 편차**: §7·§12가 `metrics(out SimOutput, days []string)`로 적었으나, `out.NAV`가 이미 일별 간격을 담아 `days`가 불필요 → 시그니처를 `metrics(out SimOutput)`로 단순화(YAGNI). 변동성 연환산은 √252 고정.

**Files:**
- Modify: `apps/api/internal/portfolio/backtest.go`
- Test: `apps/api/internal/portfolio/backtest_test.go`

- [ ] **Step 1: 실패 테스트 작성** — `backtest_test.go`에 추가:

```go
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
```

- [ ] **Step 2: 테스트 실패 확인**

Run: `cd apps/api && go test ./internal/portfolio/ -run TestMetrics -v`
Expected: FAIL — `undefined: metrics`, `undefined: SeriesMetrics`.

- [ ] **Step 3: 구현 — `backtest.go` 수정**

```go
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
```

- [ ] **Step 4: 테스트 통과 확인**

Run: `cd apps/api && go test ./internal/portfolio/ -v`
Expected: PASS (엔진 단위 전체 — T1~T5).

- [ ] **Step 5: 커밋**

```bash
git add apps/api/internal/portfolio/backtest.go apps/api/internal/portfolio/backtest_test.go
git commit -m "feat(api): 백테스트 metrics (MDD·변동성 NAV 기준, CAGR=XIRR)"
```

---

### Task 6: `BacktestDeps` 인터페이스 + `InstrumentMeta` + `PgDeps.InstrumentsMeta`

데이터 의존성 정의. 알파의 공유 `Deps`를 확장하지 않고 **분리된 `BacktestDeps` 인터페이스**(기존 `EquityDeps` 선례)를 새로 둔다 — 백테스트가 실제로 쓰는 5개 메서드만 노출. `PgDeps`는 기존 4개 메서드(`TradingDays`/`InstrumentClosesOnDates`/`FxRatesOnDates`/`BenchmarkSeries`)에 `InstrumentsMeta` 하나만 추가하면 `BacktestDeps`를 만족한다.

> **스펙과의 의도적 편차**: spec §3-2는 `InstrumentsMeta(... pool *pgxpool.Pool ...)`로 적었으나, 기존 `Deps`/`EquityDeps`의 모든 메서드가 `db.Executor`를 받으므로 테스트 가능성을 위해 `db.Executor`로 통일한다. (alpha.go·pg_deps.go 전체가 이 컨벤션.)

> **SQL 캐스팅**: spec은 `where id = any($1)`로 적었으나 `id`는 uuid 컬럼이고 `[]string`을 바인딩하므로 `any($1::uuid[])` 캐스트가 필요하다 (기존 `InstrumentClosesOnDates`의 `$1::uuid`, `FxRatesOnDates`의 `$2::date[]` 선례와 동일).

**Files:**
- Modify: `apps/api/internal/portfolio/backtest.go` (인터페이스·구조체 추가)
- Modify: `apps/api/internal/portfolio/pg_deps.go` (`InstrumentsMeta` 구현 추가)
- Test: `apps/api/internal/portfolio/backtest_test.go` (기존 white-box 파일에 추가)

- [ ] **Step 1: 실패 테스트 작성**

`apps/api/internal/portfolio/backtest_test.go` 맨 위 import에 `"context"` 추가(이미 있으면 생략) 후, 아래 테스트 추가:

```go
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
```

- [ ] **Step 2: 테스트 실패 확인**

Run: `cd apps/api && go test ./internal/portfolio/ -run TestPgDepsSatisfiesBacktestDeps`
Expected: 컴파일 실패 — `undefined: BacktestDeps` / `PgDeps has no field or method InstrumentsMeta`.

- [ ] **Step 3: 인터페이스·구조체 추가 (`backtest.go`)**

`apps/api/internal/portfolio/backtest.go` 상단(타입 정의 구역, `package portfolio` import 블록에 `"context"`·`"time"`이 이미 포함되어 있어야 함 — T2에서 `time` 추가됨, `context`는 여기서 추가)에 다음을 추가:

```go
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
```

`backtest.go` import 블록이 다음을 포함하도록 보장(T1~T5에서 `math`·`time` 추가됨):

```go
import (
	"context"
	"math"
	"time"

	"github.com/quotient/quotient/apps/api/internal/db"
)
```

- [ ] **Step 4: `PgDeps.InstrumentsMeta` 구현 (`pg_deps.go`)**

`apps/api/internal/portfolio/pg_deps.go`의 `BenchmarkSeries` 메서드 다음(파일 끝)에 추가:

```go
// InstrumentsMeta — 바스켓 종목 메타(symbol·name·currency·asset_class) 일괄 조회.
func (PgDeps) InstrumentsMeta(ctx context.Context, pool db.Executor, ids []string) (map[string]InstrumentMeta, error) {
	if len(ids) == 0 {
		return map[string]InstrumentMeta{}, nil
	}
	rows, err := pool.Query(ctx, `
		select id::text, symbol, name, currency, asset_class
		from public.instruments where id = any($1::uuid[])
	`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]InstrumentMeta{}
	for rows.Next() {
		var id string
		var m InstrumentMeta
		if err := rows.Scan(&id, &m.Symbol, &m.Name, &m.Currency, &m.AssetClass); err != nil {
			return nil, err
		}
		out[id] = m
	}
	return out, rows.Err()
}
```

- [ ] **Step 5: 테스트 통과 확인**

Run: `cd apps/api && go test ./internal/portfolio/ -run TestPgDepsSatisfiesBacktestDeps`
Expected: PASS.

- [ ] **Step 6: 커밋**

```bash
git add apps/api/internal/portfolio/backtest.go apps/api/internal/portfolio/pg_deps.go apps/api/internal/portfolio/backtest_test.go
git commit -m "feat(api): 백테스트 BacktestDeps 인터페이스 + InstrumentsMeta 조회"
```

---

### Task 7: `BacktestService.Run` — 바스켓·벤치마크 해석 + 클램프 + 4× simulate

오케스트레이터. 종목 메타 조회·자산군 가드·비중 정규화 → 시작일 클램프(전략 leg ∪ {KOSPI,SPX} firstAvailable의 max) → leg 구성(전진 채움 densify) → 전략+벤치마크 3종을 동일 `plan`·동일 `rb`로 `simulate()` 4회 → metrics 4회 → 초과수익(전략.TWR − 60/40.TWR) → 응답 조립. `simulate()`/`metrics()`/`xirr()`는 T1~T5에서 완성됨 — 여기선 DB 해석과 조립만.

**Files:**
- Modify: `apps/api/internal/portfolio/backtest.go` (요청/응답 타입 + `ValidationError` + `restrictForwardFilled` + `parseRebalance`/`periodStart` + `BacktestService` + `Run`)
- Modify: `apps/api/internal/portfolio/pg_deps.go` (`NewBacktestService` 생성자)
- Test: `apps/api/internal/portfolio/backtest_test.go` (서비스 테스트 + fake deps 추가)

- [ ] **Step 1: 실패 테스트 작성**

`backtest_test.go`의 import 블록을 다음으로 교체(엔진 테스트의 `math`·`testing`에 서비스 테스트용 `context`·`errors`·`time` + fake deps가 쓰는 `db` 추가):

```go
import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/quotient/quotient/apps/api/internal/db"
)
```

같은 파일 끝에 fake deps + 헬퍼 + 서비스 테스트를 추가:

```go
// --- 서비스 레이어 테스트 (BacktestDeps 모킹) ---

type fakeBTDeps struct {
	tradingDays []string                     // 정렬된 영업일 축
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
		Period: "all", InitialCash: 1_000_000, Rebalance: "none",
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
```

- [ ] **Step 2: 테스트 실패 확인**

Run: `cd apps/api && go test ./internal/portfolio/ -run TestRun_ -v`
Expected: FAIL — `undefined: newBacktestServiceWithDeps`, `undefined: BacktestRequest` 등 컴파일 에러.

- [ ] **Step 3: 최소 구현 작성 (`backtest.go`)**

`backtest.go`에 요청/응답 타입 + `ValidationError` + 파싱/전진채움 헬퍼 + `BacktestService` + `Run`을 추가(파일 끝). import 블록에 이미 `context`/`math`/`time`/`db`가 있어야 함(T1~T6).

```go
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
```

`pg_deps.go`에 production 생성자 추가(파일 끝):

```go
// NewBacktestService는 production용 BacktestService. Pg 구현 주입.
func NewBacktestService() *BacktestService {
	return &BacktestService{deps: &PgDeps{}, now: time.Now}
}
```

- [ ] **Step 4: 테스트 통과 확인**

Run: `cd apps/api && go test ./internal/portfolio/ -v`
Expected: PASS (엔진 T1~T5 + 서비스 T7 전체).

- [ ] **Step 5: 커밋**

```bash
git add apps/api/internal/portfolio/backtest.go apps/api/internal/portfolio/pg_deps.go apps/api/internal/portfolio/backtest_test.go
git commit -m "feat(api): 백테스트 Service.Run (클램프·벤치마크 4종·초과수익)"
```

---

### Task 8: `BacktestHandler` — 인증 + 구조적 검증 + 에러 매핑

HTTP 경계. 인증 게이트 → JSON 디코드 → **구조적 검증**(바스켓 크기·금액·비중 범위·빈 id → 422 VALIDATION) → `svc.Run` → 에러 매핑(`ValidationError` → 422 code별, `InsufficientDataError` → 422 INSUFFICIENT_DATA). **txRunner/AsUser 없음** — 공개 데이터만 읽으므로 `h.pool`을 `svc.Run`에 직접 전달(인증만). 테스트는 fake `BacktestRunner` + `pool=nil`.

> 검증 분리: 구조적(요청 형태) = 핸들러. 의미적(종목 존재·자산군·공통 구간) = 서비스. spec §9 표 그대로.

**Files:**
- Create: `apps/api/internal/handlers/backtest.go`
- Test: `apps/api/internal/handlers/backtest_test.go`

- [ ] **Step 1: 실패 테스트 작성**

`apps/api/internal/handlers/backtest_test.go`:

```go
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

type fakeBacktestRunner struct {
	res *portfolio.BacktestResult
	err error
}

func (f *fakeBacktestRunner) Run(_ context.Context, _ db.Executor, _ portfolio.BacktestRequest) (*portfolio.BacktestResult, error) {
	return f.res, f.err
}

func validBacktestBody() portfolio.BacktestRequest {
	return portfolio.BacktestRequest{
		Period: "3Y", InitialCash: 10_000_000, Monthly: 0, Rebalance: "none",
		Basket: []portfolio.BasketItem{{InstrumentID: "id1", Weight: 100}},
	}
}

func reqBacktest(t *testing.T, body portfolio.BacktestRequest, uid string) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		t.Fatalf("encode: %v", err)
	}
	r := httptest.NewRequest(http.MethodPost, "/v1/backtest/run", &buf)
	if uid != "" {
		r = r.WithContext(middleware.WithUserID(r.Context(), uid))
	}
	return r
}

func TestBacktestHandler_OK(t *testing.T) {
	svc := &fakeBacktestRunner{res: &portfolio.BacktestResult{ClampedStart: "2023-05-30", End: "2026-05-29"}}
	h := NewBacktestHandler(svc, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, validBacktestBody(), "user-1"))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["clamped_start"] != "2023-05-30" {
		t.Errorf("clamped_start=%v", got["clamped_start"])
	}
}

func TestBacktestHandler_NoAuth(t *testing.T) {
	h := NewBacktestHandler(&fakeBacktestRunner{}, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, validBacktestBody(), ""))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d, want 401", w.Code)
	}
}

func TestBacktestHandler_EmptyBasket(t *testing.T) {
	b := validBacktestBody()
	b.Basket = nil
	h := NewBacktestHandler(&fakeBacktestRunner{}, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, b, "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d, want 422", w.Code)
	}
}

func TestBacktestHandler_TooManyBasket(t *testing.T) {
	b := validBacktestBody()
	b.Basket = nil
	for i := 0; i < 11; i++ {
		b.Basket = append(b.Basket, portfolio.BasketItem{InstrumentID: "id", Weight: 1})
	}
	h := NewBacktestHandler(&fakeBacktestRunner{}, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, b, "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d, want 422", w.Code)
	}
}

func TestBacktestHandler_BadInitialCash(t *testing.T) {
	b := validBacktestBody()
	b.InitialCash = 0
	h := NewBacktestHandler(&fakeBacktestRunner{}, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, b, "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d, want 422", w.Code)
	}
}

func TestBacktestHandler_NegativeMonthly(t *testing.T) {
	b := validBacktestBody()
	b.Monthly = -1
	h := NewBacktestHandler(&fakeBacktestRunner{}, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, b, "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d, want 422", w.Code)
	}
}

func TestBacktestHandler_ZeroWeight(t *testing.T) {
	b := validBacktestBody()
	b.Basket = []portfolio.BasketItem{{InstrumentID: "id1", Weight: 0}}
	h := NewBacktestHandler(&fakeBacktestRunner{}, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, b, "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d, want 422", w.Code)
	}
}

func TestBacktestHandler_AssetNotSupported(t *testing.T) {
	svc := &fakeBacktestRunner{err: &portfolio.ValidationError{Code: "ASSET_NOT_SUPPORTED", Message: "지수·환율은 백테스트 불가"}}
	h := NewBacktestHandler(svc, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, validBacktestBody(), "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", w.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	errBlock := got["error"].(map[string]any)
	if errBlock["code"] != "ASSET_NOT_SUPPORTED" {
		t.Errorf("code=%v", errBlock["code"])
	}
}

func TestBacktestHandler_InsufficientData(t *testing.T) {
	svc := &fakeBacktestRunner{err: &portfolio.InsufficientDataError{Reason: "backtest_window_too_short", MinDays: 30, CurrentDays: 12}}
	h := NewBacktestHandler(svc, nil)
	w := httptest.NewRecorder()
	h.Run(w, reqBacktest(t, validBacktestBody(), "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", w.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	errBlock := got["error"].(map[string]any)
	if errBlock["code"] != "INSUFFICIENT_DATA" || errBlock["current_days"].(float64) != 12 {
		t.Errorf("error=%+v", errBlock)
	}
}

func TestBacktestHandler_BadJSON(t *testing.T) {
	h := NewBacktestHandler(&fakeBacktestRunner{}, nil)
	r := httptest.NewRequest(http.MethodPost, "/v1/backtest/run", strings.NewReader("{bad"))
	r = r.WithContext(middleware.WithUserID(r.Context(), "user-1"))
	w := httptest.NewRecorder()
	h.Run(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400", w.Code)
	}
}
```

- [ ] **Step 2: 테스트 실패 확인**

Run: `cd apps/api && go test ./internal/handlers/ -run TestBacktestHandler_ -v`
Expected: FAIL — `undefined: NewBacktestHandler`.

- [ ] **Step 3: 최소 구현 작성**

`apps/api/internal/handlers/backtest.go`:

```go
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

// BacktestRunner는 portfolio.BacktestService의 인터페이스 (테스트 fake용).
type BacktestRunner interface {
	Run(ctx context.Context, pool db.Executor, req portfolio.BacktestRequest) (*portfolio.BacktestResult, error)
}

type BacktestHandler struct {
	svc  BacktestRunner
	pool *pgxpool.Pool
}

// NewBacktestHandler — pool은 슈퍼유저 공개 read용. 백테스트는 사용자 데이터 미접근 → txRunner 불필요.
func NewBacktestHandler(svc BacktestRunner, pool *pgxpool.Pool) *BacktestHandler {
	return &BacktestHandler{svc: svc, pool: pool}
}

// POST /v1/backtest/run
func (h *BacktestHandler) Run(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	var req portfolio.BacktestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if msg := validateBacktestReq(req); msg != "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", msg)
		return
	}

	result, err := h.svc.Run(r.Context(), h.pool, req)
	if err != nil {
		var ve *portfolio.ValidationError
		if errors.As(err, &ve) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error": map[string]any{"code": ve.Code, "message": ve.Message},
			})
			return
		}
		var ie *portfolio.InsufficientDataError
		if errors.As(err, &ie) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error": map[string]any{
					"code":         "INSUFFICIENT_DATA",
					"reason":       ie.Reason,
					"message":      "기간이 너무 짧습니다 (최소 30영업일)",
					"min_days":     ie.MinDays,
					"current_days": ie.CurrentDays,
				},
			})
			return
		}
		slog.Error("backtest run failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "backtest failed")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// validateBacktestReq — 구조적 검증만(요청 형태). 빈 문자열 반환 = 통과.
func validateBacktestReq(req portfolio.BacktestRequest) string {
	if len(req.Basket) == 0 || len(req.Basket) > 10 {
		return "종목을 1~10개 선택하세요"
	}
	if req.InitialCash <= 0 {
		return "초기 자금은 0보다 커야 합니다"
	}
	if req.Monthly < 0 {
		return "월 적립금은 0 이상이어야 합니다"
	}
	for _, b := range req.Basket {
		if b.InstrumentID == "" {
			return "종목을 찾을 수 없습니다"
		}
		if b.Weight <= 0 {
			return "비중은 0보다 커야 합니다"
		}
	}
	return ""
}
```

- [ ] **Step 4: 테스트 통과 확인**

Run: `cd apps/api && go test ./internal/handlers/ -run TestBacktestHandler_ -v`
Expected: PASS (10개 케이스).

- [ ] **Step 5: 커밋**

```bash
git add apps/api/internal/handlers/backtest.go apps/api/internal/handlers/backtest_test.go
git commit -m "feat(api): 백테스트 핸들러 (인증·구조 검증·422 매핑)"
```

---

### Task 9: Router + main.go wiring

라우트 등록 + 의존성 조립. 새 unit 없음 — `go build` + 기존 전체 테스트로 검증(wiring 태스크). `router.New`에 `backtestHandler` 파라미터를 `paperHandler` 다음에 추가하고 인증 그룹에 `POST /v1/backtest/run`을 등록한다.

**Files:**
- Modify: `apps/api/internal/router/router.go`
- Modify: `apps/api/cmd/server/main.go`

- [ ] **Step 1: `router.go` — 파라미터 + 라우트 추가**

`New(...)` 시그니처의 `paperHandler *handlers.PaperHandler,` 다음 줄에 추가:

```go
	paperHandler *handlers.PaperHandler,
	backtestHandler *handlers.BacktestHandler,
) *chi.Mux {
```

인증 그룹 내부, `r.Post("/v1/paper/reset", paperHandler.Reset)` 다음 줄에 추가:

```go
		r.Post("/v1/paper/reset", paperHandler.Reset)

		r.Post("/v1/backtest/run", backtestHandler.Run)
```

- [ ] **Step 2: `main.go` — 서비스·핸들러 조립 + router.New 인자 추가**

`paperHandler := handlers.NewPaperHandler(...)` 다음(line 95 뒤)에 추가:

```go
	paperHandler := handlers.NewPaperHandler(paperRepo, journalRepo, equityComputer, pool)

	backtestSvc := portfolio.NewBacktestService()
	backtestHandler := handlers.NewBacktestHandler(backtestSvc, pool)
```

`router.New(...)` 호출의 `paperHandler,` 다음 줄에 추가:

```go
			paperHandler,
			backtestHandler,
		),
```

- [ ] **Step 3: 빌드 + 전체 테스트 확인**

Run: `cd apps/api && go build ./... && go test ./...`
Expected: 빌드 성공 + 모든 패키지 PASS (integration 빌드태그 제외 — 기본 `go test`는 `//go:build integration` 파일을 건너뜀).

- [ ] **Step 4: 커밋**

```bash
git add apps/api/internal/router/router.go apps/api/cmd/server/main.go
git commit -m "feat(api): 백테스트 라우트 등록 (POST /v1/backtest/run) + wiring"
```

---

### Task 10: 통합 테스트 (`//go:build integration`)

실제 Postgres에 시드된 가격으로 E2E. 백테스트는 무상태(사용자 데이터 미접근)라 user 시드 불필요 — 공개 `instruments`/`prices`/`fx_rates`만 읽는다. `openPool`은 같은 `portfolio_test` 패키지의 `alpha_integration_test.go`에 정의돼 있으므로 **재사용**(재정의 금지). 백필이 안 된 환경에선 `Skip`.

**Files:**
- Create: `apps/api/internal/portfolio/backtest_integration_test.go`

- [ ] **Step 1: 통합 테스트 작성**

`apps/api/internal/portfolio/backtest_integration_test.go`:

```go
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
```

- [ ] **Step 2: 통합 테스트 실행 (DB 있을 때)**

Run: `cd apps/api && TEST_DATABASE_URL=$TEST_DATABASE_URL go test -tags integration ./internal/portfolio/ -run TestBacktest_E2E -v`
Expected: PASS (백필 데이터 있으면) 또는 SKIP (`TEST_DATABASE_URL` 미설정 / 005930 미시드 / 백필 부족).

- [ ] **Step 3: 기본 빌드 회귀 확인**

Run: `cd apps/api && go build ./... && go vet ./internal/portfolio/`
Expected: 성공 (integration 태그 없는 기본 빌드에 영향 없음).

- [ ] **Step 4: 커밋**

```bash
git add apps/api/internal/portfolio/backtest_integration_test.go
git commit -m "test(api): 백테스트 통합 테스트 (E2E 단일·DCA·INDEX 거부)"
```

---

### Task 11: 프론트 API 클라이언트 (`lib/api/backtest.ts`)

§8 응답 JSON을 그대로 미러링한 TypeScript 타입 + `runBacktest()`(정상/422 union 반환) + `isBacktestError` 가드. `lib/api/portfolio.ts`의 union 패턴을 그대로 따른다(422를 throw 대신 union으로 반환 → 사용 측 try/catch 부담 제거). 백엔드 Go 타입(T7)의 json 태그와 1:1 일치해야 한다 — 벤치마크 `metrics`는 `twr_pct` 포함(`SeriesMetrics`), 최상위 `metrics`는 `excess_vs_6040_pct`·`total_contributed`·`final_equity` 포함하고 `twr_pct` 미포함(`StrategyMetrics`). `cagr_pct`는 둘 다 `number | null`.

**Files:**
- Create: `apps/web/lib/api/backtest.ts`

- [ ] **Step 1: 타입 + 클라이언트 작성**

`apps/web/lib/api/backtest.ts`:

```typescript
import { authFetch } from "./auth-fetch";

export type BacktestPeriod = "1Y" | "3Y" | "5Y" | "all";
export type RebalanceFreq = "none" | "quarterly" | "semiannual" | "annual";

export type BasketInput = { instrument_id: string; weight: number };

export type BacktestReq = {
  period: BacktestPeriod;
  initial_cash: number;
  monthly_contribution: number;
  basket: BasketInput[];
  rebalance: RebalanceFreq;
};

export type ValuePoint = { date: string; value: number };

export type NormalizedLeg = {
  instrument_id: string;
  symbol: string;
  name: string;
  weight: number; // 정규화된 비중 0..1
};

// 벤치마크 metrics (Go SeriesMetrics) — twr_pct 포함
export type SeriesMetrics = {
  total_return_pct: number;
  cagr_pct: number | null;
  mdd_pct: number;
  volatility_pct: number;
  twr_pct: number;
};

export type BenchmarkResult = {
  equity_series: ValuePoint[];
  metrics: SeriesMetrics;
};

// 전략 metrics (Go StrategyMetrics) — excess·투입·최종 포함, twr 제외
export type StrategyMetrics = {
  total_return_pct: number;
  cagr_pct: number | null;
  mdd_pct: number;
  volatility_pct: number;
  excess_vs_6040_pct: number;
  total_contributed: number;
  final_equity: number;
};

export type CoverageWarning = {
  symbol: string;
  first_available: string;
  message: string;
};

export type BacktestResult = {
  clamped_start: string;
  end: string;
  normalized_basket: NormalizedLeg[];
  equity_series: ValuePoint[];
  contributed_series: ValuePoint[];
  benchmarks: {
    kospi: BenchmarkResult;
    spx: BenchmarkResult;
    sixty_forty: BenchmarkResult;
  };
  metrics: StrategyMetrics;
  coverage_warnings: CoverageWarning[];
};

export type BacktestErrorBody = {
  error: {
    code: "VALIDATION" | "ASSET_NOT_SUPPORTED" | "INSUFFICIENT_DATA";
    message: string;
    min_days?: number;
    current_days?: number;
  };
};

// 422(검증·자산미지원·데이터부족)는 throw 대신 union 반환 — 알파 카드(getAlpha)와 동일 패턴.
export async function runBacktest(
  req: BacktestReq,
): Promise<BacktestResult | BacktestErrorBody> {
  const res = await authFetch("/v1/backtest/run", {
    method: "POST",
    body: JSON.stringify(req),
  });
  if (res.status === 422 || res.status === 400) {
    return (await res.json()) as BacktestErrorBody;
  }
  if (!res.ok) {
    throw new Error(`backtest failed: ${res.status}`);
  }
  return (await res.json()) as BacktestResult;
}

export function isBacktestError(
  r: BacktestResult | BacktestErrorBody,
): r is BacktestErrorBody {
  return (r as BacktestErrorBody).error !== undefined;
}
```

- [ ] **Step 2: 타입체크 통과 확인**

Run: `cd apps/web && npx tsc --noEmit`
Expected: 성공 (신규 파일이 컴파일됨; 아직 사용처 없음).

- [ ] **Step 3: 커밋**

```bash
git add apps/web/lib/api/backtest.ts
git commit -m "feat(web): 백테스트 API 클라이언트 (타입 + runBacktest union)"
```

---

### Task 12: 입력 폼 컴포넌트 (`PeriodPicker`·`CashInputs`·`RebalanceSelect`·`BasketBuilder`·`BacktestForm`)

§10 입력 폼. 하위 4개는 제어 컴포넌트(value+onChange), `BacktestForm`이 폼 상태를 소유하고 `BacktestReq`를 만들어 `onRun`을 호출한다. 종목 검색은 기존 `InstrumentSearchInput` 재사용(INDEX·FX·CASH는 이미 비활성). 비중은 서버에서 정규화하므로 합계 100% 강제는 하지 않고 표시만 한다. 최대 10종목·중복 차단·각 비중 > 0·초기 자금 > 0일 때만 실행 활성. 프라이머리 버튼 스타일은 기존 `bg-bb-accent text-bg ... disabled:opacity-50`(LoginForm/SignupForm) 패턴을 따른다.

**Files:**
- Create: `apps/web/components/backtest/PeriodPicker.tsx`
- Create: `apps/web/components/backtest/CashInputs.tsx`
- Create: `apps/web/components/backtest/RebalanceSelect.tsx`
- Create: `apps/web/components/backtest/BasketBuilder.tsx`
- Create: `apps/web/components/backtest/BacktestForm.tsx`

- [ ] **Step 1: `PeriodPicker` 작성**

`apps/web/components/backtest/PeriodPicker.tsx`:

```tsx
"use client";
import { clsx } from "clsx";
import type { BacktestPeriod } from "@/lib/api/backtest";

const OPTS: { value: BacktestPeriod; label: string }[] = [
  { value: "1Y", label: "1Y" },
  { value: "3Y", label: "3Y" },
  { value: "5Y", label: "5Y" },
  { value: "all", label: "전체" },
];

export function PeriodPicker({
  value,
  onChange,
}: {
  value: BacktestPeriod;
  onChange: (p: BacktestPeriod) => void;
}) {
  return (
    <div className="flex gap-1.5">
      {OPTS.map((o) => (
        <button
          key={o.value}
          type="button"
          onClick={() => onChange(o.value)}
          className={clsx(
            "px-3 py-1 text-xs font-mono border transition-colors",
            value === o.value
              ? "border-bb-accent text-bb-accent"
              : "border-line text-fg-muted hover:text-fg",
          )}
        >
          {o.label}
        </button>
      ))}
    </div>
  );
}
```

- [ ] **Step 2: `CashInputs` 작성** (`ContributionMode` 타입의 소유자)

`apps/web/components/backtest/CashInputs.tsx`:

```tsx
"use client";

export type ContributionMode = "lump" | "dca";

export function CashInputs({
  initialCash,
  mode,
  monthly,
  onInitialCash,
  onMode,
  onMonthly,
}: {
  initialCash: number;
  mode: ContributionMode;
  monthly: number;
  onInitialCash: (v: number) => void;
  onMode: (m: ContributionMode) => void;
  onMonthly: (v: number) => void;
}) {
  return (
    <div className="flex gap-3">
      <label className="flex-1">
        <div className="text-xs text-fg-muted mb-1">초기 자금 (₩)</div>
        <input
          type="number"
          min={0}
          value={initialCash}
          onChange={(e) => onInitialCash(Number(e.target.value))}
          className="bg-bg-deep border border-line px-3 py-1.5 text-sm font-mono w-full tabular-nums"
        />
      </label>
      <label className="flex-1">
        <div className="text-xs text-fg-muted mb-1">투입 방식</div>
        <select
          value={mode}
          onChange={(e) => onMode(e.target.value as ContributionMode)}
          className="bg-bg-deep border border-line px-3 py-1.5 text-sm font-mono w-full"
        >
          <option value="lump">일시불</option>
          <option value="dca">월 적립</option>
        </select>
      </label>
      <label className="flex-1">
        <div className="text-xs text-fg-muted mb-1">월 적립금 (₩)</div>
        <input
          type="number"
          min={0}
          value={monthly}
          disabled={mode === "lump"}
          onChange={(e) => onMonthly(Number(e.target.value))}
          className="bg-bg-deep border border-line px-3 py-1.5 text-sm font-mono w-full tabular-nums disabled:opacity-40"
        />
      </label>
    </div>
  );
}
```

- [ ] **Step 3: `RebalanceSelect` 작성**

`apps/web/components/backtest/RebalanceSelect.tsx`:

```tsx
"use client";
import type { RebalanceFreq } from "@/lib/api/backtest";

const OPTS: { value: RebalanceFreq; label: string }[] = [
  { value: "none", label: "없음" },
  { value: "quarterly", label: "분기" },
  { value: "semiannual", label: "반기" },
  { value: "annual", label: "연" },
];

export function RebalanceSelect({
  value,
  onChange,
}: {
  value: RebalanceFreq;
  onChange: (r: RebalanceFreq) => void;
}) {
  return (
    <label className="flex-1">
      <div className="text-xs text-fg-muted mb-1">리밸런싱</div>
      <select
        value={value}
        onChange={(e) => onChange(e.target.value as RebalanceFreq)}
        className="bg-bg-deep border border-line px-3 py-1.5 text-sm font-mono w-full"
      >
        {OPTS.map((o) => (
          <option key={o.value} value={o.value}>
            {o.label}
          </option>
        ))}
      </select>
    </label>
  );
}
```

- [ ] **Step 4: `BasketBuilder` 작성**

`apps/web/components/backtest/BasketBuilder.tsx`:

```tsx
"use client";
import { InstrumentSearchInput } from "@/components/portfolio/InstrumentSearchInput";
import type { InstrumentResult } from "@/lib/api/instruments";
import { X } from "lucide-react";

export type BasketRow = { inst: InstrumentResult; weight: number };

export const MAX_LEGS = 10;

export function BasketBuilder({
  rows,
  onChange,
}: {
  rows: BasketRow[];
  onChange: (rows: BasketRow[]) => void;
}) {
  const sum = rows.reduce((a, r) => a + (r.weight || 0), 0);

  function add(inst: InstrumentResult) {
    if (rows.length >= MAX_LEGS) return;
    if (rows.some((r) => r.inst.id === inst.id)) return; // 중복 차단
    onChange([...rows, { inst, weight: 0 }]);
  }
  function setWeight(id: string, w: number) {
    onChange(rows.map((r) => (r.inst.id === id ? { ...r, weight: w } : r)));
  }
  function remove(id: string) {
    onChange(rows.filter((r) => r.inst.id !== id));
  }

  return (
    <div>
      <div className="text-xs text-fg-muted mb-1">
        바스켓 (목표 비중 · 실행 시 자동 정규화)
      </div>
      <div className="border border-line divide-y divide-line/50">
        {rows.length === 0 && (
          <div className="px-3 py-2 text-xs text-fg-muted font-mono">
            종목을 추가하세요 (최대 {MAX_LEGS}개)
          </div>
        )}
        {rows.map((r) => (
          <div key={r.inst.id} className="flex items-center gap-2 px-3 py-2">
            <div className="flex-1 font-mono text-sm">
              {r.inst.symbol}
              <span className="text-fg-muted text-xs ml-2">{r.inst.name}</span>
            </div>
            <input
              type="number"
              min={0}
              value={r.weight}
              onChange={(e) => setWeight(r.inst.id, Number(e.target.value))}
              className="w-16 bg-bg-deep border border-line px-2 py-1 text-sm font-mono text-right tabular-nums"
              aria-label={`${r.inst.symbol} 비중`}
            />
            <span className="text-xs text-fg-muted">%</span>
            <button
              type="button"
              onClick={() => remove(r.inst.id)}
              className="text-fg-muted hover:text-bb-down"
              aria-label={`${r.inst.symbol} 삭제`}
            >
              <X size={14} />
            </button>
          </div>
        ))}
      </div>
      {rows.length < MAX_LEGS && (
        <div className="mt-2">
          <InstrumentSearchInput onSelect={add} placeholder="＋ 종목 추가 (검색)" />
        </div>
      )}
      <div className="text-xs text-fg-muted mt-1 text-right tabular-nums">
        합계 {sum}%
      </div>
    </div>
  );
}
```

- [ ] **Step 5: `BacktestForm` 작성 (폼 상태 소유 + `onRun`)**

`apps/web/components/backtest/BacktestForm.tsx`:

```tsx
"use client";
import { useState } from "react";
import { PeriodPicker } from "./PeriodPicker";
import { CashInputs, type ContributionMode } from "./CashInputs";
import { RebalanceSelect } from "./RebalanceSelect";
import { BasketBuilder, type BasketRow } from "./BasketBuilder";
import type {
  BacktestPeriod,
  BacktestReq,
  RebalanceFreq,
} from "@/lib/api/backtest";

export function BacktestForm({
  onRun,
  running,
}: {
  onRun: (req: BacktestReq) => void;
  running: boolean;
}) {
  const [period, setPeriod] = useState<BacktestPeriod>("3Y");
  const [initialCash, setInitialCash] = useState(10_000_000);
  const [mode, setMode] = useState<ContributionMode>("lump");
  const [monthly, setMonthly] = useState(500_000);
  const [rebalance, setRebalance] = useState<RebalanceFreq>("quarterly");
  const [rows, setRows] = useState<BasketRow[]>([]);

  const canRun =
    rows.length > 0 &&
    rows.every((r) => r.weight > 0) &&
    initialCash > 0 &&
    !running;

  function submit() {
    if (!canRun) return;
    onRun({
      period,
      initial_cash: initialCash,
      monthly_contribution: mode === "dca" ? monthly : 0,
      rebalance,
      basket: rows.map((r) => ({ instrument_id: r.inst.id, weight: r.weight })),
    });
  }

  return (
    <div className="border border-line p-4 space-y-4">
      <div>
        <div className="text-xs text-fg-muted mb-1">기간 (종료일 = 오늘)</div>
        <PeriodPicker value={period} onChange={setPeriod} />
      </div>
      <CashInputs
        initialCash={initialCash}
        mode={mode}
        monthly={monthly}
        onInitialCash={setInitialCash}
        onMode={setMode}
        onMonthly={setMonthly}
      />
      <BasketBuilder rows={rows} onChange={setRows} />
      <div className="flex gap-3 items-end">
        <RebalanceSelect value={rebalance} onChange={setRebalance} />
        <button
          type="button"
          onClick={submit}
          disabled={!canRun}
          className="flex-1 bg-bb-accent text-bg font-mono text-sm py-2 disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {running ? "백테스트 중…" : "백테스트 실행"}
        </button>
      </div>
    </div>
  );
}
```

- [ ] **Step 6: 타입체크 통과 확인**

Run: `cd apps/web && npx tsc --noEmit`
Expected: 성공 (폼 컴포넌트 컴파일; 아직 페이지에서 미사용이나 export는 유효).

- [ ] **Step 7: 커밋**

```bash
git add apps/web/components/backtest/PeriodPicker.tsx apps/web/components/backtest/CashInputs.tsx apps/web/components/backtest/RebalanceSelect.tsx apps/web/components/backtest/BasketBuilder.tsx apps/web/components/backtest/BacktestForm.tsx
git commit -m "feat(web): 백테스트 입력 폼 컴포넌트 (기간·자금·바스켓·리밸런싱)"
```

---

### Task 13: 결과 컴포넌트 (`MetricCards`·`BacktestEquityChart`·`CompareTable`·`CoverageNotice`·`BacktestResults`)

§10 결과 화면. `BacktestEquityChart`는 5선 멀티라인(전략 굵게 + KOSPI·S&P·60/40 + 투입 원금 점선, KRW)이라 단선 `LineChartCard`를 재사용하지 못하고 동일한 recharts dynamic 패턴으로 새로 작성한다. 색은 `chart-tokens`의 `CHART_COLORS`를 쓰되 4번째 라인용 보라색만 로컬 상수로 추가. `metrics`(전략)는 `cagr_pct` null 가능 → "—" 표기. `CompareTable`은 전략+벤치마크 3종을 행으로, 총수익·CAGR·MDD를 열로. `CoverageNotice`는 경고 0건이면 렌더 안 함.

**Files:**
- Create: `apps/web/components/backtest/MetricCards.tsx`
- Create: `apps/web/components/backtest/BacktestEquityChart.tsx`
- Create: `apps/web/components/backtest/CompareTable.tsx`
- Create: `apps/web/components/backtest/CoverageNotice.tsx`
- Create: `apps/web/components/backtest/BacktestResults.tsx`

- [ ] **Step 1: `MetricCards` 작성**

`apps/web/components/backtest/MetricCards.tsx`:

```tsx
"use client";
import type { StrategyMetrics } from "@/lib/api/backtest";

function pct(v: number | null, signed = false): string {
  if (v === null || Number.isNaN(v)) return "—";
  const s = v.toFixed(2);
  return signed && v >= 0 ? `+${s}%` : `${s}%`;
}

function won(v: number): string {
  return `₩${Math.round(v).toLocaleString()}`;
}

export function MetricCards({ m }: { m: StrategyMetrics }) {
  const cards: { label: string; value: string; tone?: "up" | "down" }[] = [
    { label: "총수익률", value: pct(m.total_return_pct, true), tone: m.total_return_pct >= 0 ? "up" : "down" },
    { label: "CAGR", value: pct(m.cagr_pct, true), tone: (m.cagr_pct ?? 0) >= 0 ? "up" : "down" },
    { label: "MDD", value: pct(m.mdd_pct), tone: "down" },
    { label: "변동성", value: pct(m.volatility_pct) },
    { label: "초과수익 vs 60/40", value: pct(m.excess_vs_6040_pct, true), tone: m.excess_vs_6040_pct >= 0 ? "up" : "down" },
  ];
  return (
    <div className="grid grid-cols-2 sm:grid-cols-5 gap-2">
      {cards.map((c) => (
        <div key={c.label} className="border border-line p-3">
          <div className="text-[10px] text-fg-muted font-mono">{c.label}</div>
          <div
            className={`font-mono text-base tabular-nums ${
              c.tone === "up" ? "text-bb-up" : c.tone === "down" ? "text-bb-down" : ""
            }`}
          >
            {c.value}
          </div>
        </div>
      ))}
      <div className="col-span-2 sm:col-span-5 flex justify-between text-xs text-fg-muted font-mono pt-1">
        <span>누적 투입 {won(m.total_contributed)}</span>
        <span>최종 평가액 {won(m.final_equity)}</span>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: `BacktestEquityChart` 작성 (5선 멀티라인)**

`apps/web/components/backtest/BacktestEquityChart.tsx`:

```tsx
"use client";
import dynamic from "next/dynamic";
import { CHART_COLORS } from "@/components/charts/chart-tokens";
import type { BacktestResult } from "@/lib/api/backtest";

/* eslint-disable @typescript-eslint/no-explicit-any */
const ResponsiveContainer = dynamic<any>(() => import("recharts").then((m) => m.ResponsiveContainer), { ssr: false });
const LineChart = dynamic<any>(() => import("recharts").then((m) => m.LineChart), { ssr: false });
const Line = dynamic<any>(() => import("recharts").then((m) => m.Line), { ssr: false });
const XAxis = dynamic<any>(() => import("recharts").then((m) => m.XAxis), { ssr: false });
const YAxis = dynamic<any>(() => import("recharts").then((m) => m.YAxis), { ssr: false });
const Tooltip = dynamic<any>(() => import("recharts").then((m) => m.Tooltip), { ssr: false });
const Legend = dynamic<any>(() => import("recharts").then((m) => m.Legend), { ssr: false });
/* eslint-enable @typescript-eslint/no-explicit-any */

const SIXTY_FORTY_COLOR = "#A78BFA"; // 보라 — 토큰 외 4번째 라인 구분용

type ChartRow = {
  x: string;
  strategy?: number;
  kospi?: number;
  spx?: number;
  sixtyForty?: number;
  contributed?: number;
};

function buildRows(res: BacktestResult): ChartRow[] {
  const byDate = new Map<string, ChartRow>();
  const ensure = (d: string): ChartRow => {
    let row = byDate.get(d);
    if (!row) {
      row = { x: d };
      byDate.set(d, row);
    }
    return row;
  };
  for (const p of res.equity_series) ensure(p.date).strategy = p.value;
  for (const p of res.benchmarks.kospi.equity_series) ensure(p.date).kospi = p.value;
  for (const p of res.benchmarks.spx.equity_series) ensure(p.date).spx = p.value;
  for (const p of res.benchmarks.sixty_forty.equity_series) ensure(p.date).sixtyForty = p.value;
  for (const p of res.contributed_series) ensure(p.date).contributed = p.value;
  return [...byDate.values()].sort((a, b) => a.x.localeCompare(b.x));
}

export function BacktestEquityChart({ result }: { result: BacktestResult }) {
  const rows = buildRows(result);
  return (
    <div className="border border-line p-4">
      <div className="font-mono text-sm mb-2">평가액 추이 (₩)</div>
      <div style={{ width: "100%", height: 320 }}>
        <ResponsiveContainer>
          <LineChart data={rows} margin={{ top: 5, right: 12, bottom: 0, left: 0 }}>
            <XAxis dataKey="x" tick={{ fontSize: 10, fill: CHART_COLORS.muted }} minTickGap={40} />
            <YAxis
              tick={{ fontSize: 10, fill: CHART_COLORS.muted }}
              tickFormatter={(v: number) => `${Math.round(v / 1_000_000)}M`}
              width={40}
            />
            <Tooltip
              contentStyle={{ background: "#0a0a0a", border: `1px solid ${CHART_COLORS.line}`, fontSize: 11, fontFamily: "monospace" }}
              labelStyle={{ color: CHART_COLORS.muted }}
              formatter={(v: unknown) => (typeof v === "number" ? `₩${v.toLocaleString()}` : String(v))}
            />
            <Legend wrapperStyle={{ fontSize: 11, fontFamily: "monospace" }} />
            <Line type="monotone" dataKey="strategy" name="내 전략" stroke={CHART_COLORS.accent} strokeWidth={2} dot={false} isAnimationActive={false} />
            <Line type="monotone" dataKey="kospi" name="KOSPI" stroke={CHART_COLORS.up} strokeWidth={1} dot={false} isAnimationActive={false} />
            <Line type="monotone" dataKey="spx" name="S&P 500" stroke={CHART_COLORS.warn} strokeWidth={1} dot={false} isAnimationActive={false} />
            <Line type="monotone" dataKey="sixtyForty" name="한미 60/40" stroke={SIXTY_FORTY_COLOR} strokeWidth={1} dot={false} isAnimationActive={false} />
            <Line type="monotone" dataKey="contributed" name="투입 원금" stroke={CHART_COLORS.muted} strokeWidth={1} strokeDasharray="4 3" dot={false} isAnimationActive={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
```

- [ ] **Step 3: `CompareTable` 작성**

`apps/web/components/backtest/CompareTable.tsx`:

```tsx
"use client";
import type { BacktestResult } from "@/lib/api/backtest";

function pct(v: number | null): string {
  if (v === null || Number.isNaN(v)) return "—";
  return `${v.toFixed(2)}%`;
}

export function CompareTable({ result }: { result: BacktestResult }) {
  const rows = [
    { label: "내 전략", tr: result.metrics.total_return_pct, cagr: result.metrics.cagr_pct, mdd: result.metrics.mdd_pct, bold: true },
    { label: "KOSPI", tr: result.benchmarks.kospi.metrics.total_return_pct, cagr: result.benchmarks.kospi.metrics.cagr_pct, mdd: result.benchmarks.kospi.metrics.mdd_pct },
    { label: "S&P 500", tr: result.benchmarks.spx.metrics.total_return_pct, cagr: result.benchmarks.spx.metrics.cagr_pct, mdd: result.benchmarks.spx.metrics.mdd_pct },
    { label: "한미 60/40", tr: result.benchmarks.sixty_forty.metrics.total_return_pct, cagr: result.benchmarks.sixty_forty.metrics.cagr_pct, mdd: result.benchmarks.sixty_forty.metrics.mdd_pct },
  ];
  return (
    <div className="border border-line">
      <table className="w-full text-sm font-mono">
        <thead>
          <tr className="text-fg-muted text-xs border-b border-line">
            <th className="text-left px-3 py-2 font-normal">전략·벤치마크</th>
            <th className="text-right px-3 py-2 font-normal">총수익률</th>
            <th className="text-right px-3 py-2 font-normal">CAGR</th>
            <th className="text-right px-3 py-2 font-normal">MDD</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.label} className={`border-b border-line/40 last:border-b-0 ${r.bold ? "text-bb-accent" : ""}`}>
              <td className="text-left px-3 py-2">{r.label}</td>
              <td className="text-right px-3 py-2 tabular-nums">{pct(r.tr)}</td>
              <td className="text-right px-3 py-2 tabular-nums">{pct(r.cagr)}</td>
              <td className="text-right px-3 py-2 tabular-nums">{pct(r.mdd)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
```

- [ ] **Step 4: `CoverageNotice` 작성**

`apps/web/components/backtest/CoverageNotice.tsx`:

```tsx
"use client";
import type { CoverageWarning } from "@/lib/api/backtest";

export function CoverageNotice({
  warnings,
  clampedStart,
}: {
  warnings: CoverageWarning[];
  clampedStart: string;
}) {
  if (warnings.length === 0) return null;
  return (
    <div className="border border-bb-warn/40 bg-bb-warn/5 p-3 text-xs font-mono space-y-1">
      <div className="text-bb-warn">
        일부 종목의 데이터가 짧아 시작일이 {clampedStart}로 조정되었습니다.
      </div>
      {warnings.map((w) => (
        <div key={w.symbol} className="text-fg-muted">
          · {w.symbol}: {w.message} (최초 {w.first_available})
        </div>
      ))}
    </div>
  );
}
```

- [ ] **Step 5: `BacktestResults` 작성 (조합)**

`apps/web/components/backtest/BacktestResults.tsx`:

```tsx
"use client";
import type { BacktestResult } from "@/lib/api/backtest";
import { MetricCards } from "./MetricCards";
import { BacktestEquityChart } from "./BacktestEquityChart";
import { CompareTable } from "./CompareTable";
import { CoverageNotice } from "./CoverageNotice";

export function BacktestResults({ result }: { result: BacktestResult }) {
  return (
    <div className="space-y-3">
      <CoverageNotice warnings={result.coverage_warnings} clampedStart={result.clamped_start} />
      <MetricCards m={result.metrics} />
      <BacktestEquityChart result={result} />
      <CompareTable result={result} />
    </div>
  );
}
```

- [ ] **Step 6: 타입체크 통과 확인**

Run: `cd apps/web && npx tsc --noEmit`
Expected: 성공.

- [ ] **Step 7: 커밋**

```bash
git add apps/web/components/backtest/MetricCards.tsx apps/web/components/backtest/BacktestEquityChart.tsx apps/web/components/backtest/CompareTable.tsx apps/web/components/backtest/CoverageNotice.tsx apps/web/components/backtest/BacktestResults.tsx
git commit -m "feat(web): 백테스트 결과 컴포넌트 (지표·멀티라인 차트·비교표·커버리지)"
```

---

### Task 14: 페이지 오케스트레이터 + 라우트 + 사이드바 (`BacktestPage`·`/app/backtest`·`Sidebar`)

폼 + 결과를 묶는 클라이언트 페이지. 실행 상태/결과/에러를 보유하고 `runBacktest` union을 분기(`isBacktestError` → 메시지, 정상 → 결과). 빈 상태는 §10 카피("바스켓과 전략을 설정하고 실행하세요"). 라우트 래퍼는 기존 `app/app/paper/page.tsx`와 동일하게 얇게. 사이드바는 Paper 다음·설정 앞에 lucide `History` 아이콘 + "백테스트" 추가.

**Files:**
- Create: `apps/web/components/backtest/BacktestPage.tsx`
- Create: `apps/web/app/app/backtest/page.tsx`
- Modify: `apps/web/components/shell/Sidebar.tsx`

- [ ] **Step 1: `BacktestPage` 작성**

`apps/web/components/backtest/BacktestPage.tsx`:

```tsx
"use client";
import { useState } from "react";
import { BacktestForm } from "./BacktestForm";
import { BacktestResults } from "./BacktestResults";
import {
  runBacktest,
  isBacktestError,
  type BacktestReq,
  type BacktestResult,
} from "@/lib/api/backtest";

export function BacktestPage() {
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<BacktestResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function handleRun(req: BacktestReq) {
    setRunning(true);
    setError(null);
    try {
      const res = await runBacktest(req);
      if (isBacktestError(res)) {
        setError(res.error.message);
        setResult(null);
      } else {
        setResult(res);
      }
    } catch {
      setError("백테스트 실행 중 오류가 발생했습니다.");
      setResult(null);
    } finally {
      setRunning(false);
    }
  }

  return (
    <div className="p-6 max-w-4xl mx-auto space-y-4">
      <div>
        <h1 className="font-mono text-lg text-bb-accent">백테스트</h1>
        <p className="text-xs text-fg-muted font-mono mt-1">
          과거 데이터로 바스켓 + 적립·리밸런싱 전략을 시뮬레이션하고 KOSPI·S&P·한미 60/40과 비교합니다. (최근 5년·종료일 오늘)
        </p>
      </div>
      <BacktestForm onRun={handleRun} running={running} />
      {error && (
        <div className="border border-bb-down/40 bg-bb-down/5 p-3 text-xs font-mono text-bb-down">
          {error}
        </div>
      )}
      {result ? (
        <BacktestResults result={result} />
      ) : (
        !error && (
          <div className="border border-line border-dashed p-8 text-center text-sm text-fg-muted font-mono">
            바스켓과 전략을 설정하고 실행하세요.
          </div>
        )
      )}
    </div>
  );
}
```

- [ ] **Step 2: 라우트 래퍼 작성**

`apps/web/app/app/backtest/page.tsx`:

```tsx
import { BacktestPage } from "@/components/backtest/BacktestPage";

export default function Page() {
  return <BacktestPage />;
}
```

- [ ] **Step 3: 사이드바 import에 `History` 추가**

`apps/web/components/shell/Sidebar.tsx` — lucide import 한 줄 교체:

```tsx
import { Home, Wallet, MessageSquare, BarChart3, BookOpen, LineChart, History, Settings, Heart } from "lucide-react";
```

(기존: `import { Home, Wallet, MessageSquare, BarChart3, BookOpen, LineChart, Settings, Heart } from "lucide-react";`)

- [ ] **Step 4: 사이드바 `items`에 백테스트 추가 (Paper 다음·설정 앞)**

`apps/web/components/shell/Sidebar.tsx` — `items` 배열에서 Paper와 설정 사이에 한 줄 삽입:

```tsx
  { href: "/app/paper", icon: LineChart, label: "Paper" },
  { href: "/app/backtest", icon: History, label: "백테스트" },
  { href: "/app/settings", icon: Settings, label: "설정" },
```

- [ ] **Step 5: 타입체크 + 빌드 확인**

Run: `cd apps/web && npx tsc --noEmit && npx next build`
Expected: 성공 (`/app/backtest` 라우트가 빌드 출력에 포함).

- [ ] **Step 6: 커밋**

```bash
git add apps/web/components/backtest/BacktestPage.tsx apps/web/app/app/backtest/page.tsx apps/web/components/shell/Sidebar.tsx
git commit -m "feat(web): 백테스트 페이지 + 라우트 + 사이드바 탭"
```

---

### Task 15: 프론트 컴포넌트 테스트 (`BacktestPage.test.tsx`)

`@/lib/api/backtest`(runBacktest)와 `@/lib/api/instruments`(searchInstruments·selectInstrument)를 mock. `AlphaCard.test.tsx` 패턴(module mock + waitFor) 그대로. 폼이 유효해지려면 종목 1개·비중 > 0 필요하므로, 검색→선택→비중 입력→실행 흐름을 헬퍼로 구동한다. `isBacktestError`는 `importActual`로 실제 구현 유지(BacktestPage가 호출).

**Files:**
- Create: `apps/web/components/backtest/BacktestPage.test.tsx`

- [ ] **Step 1: 테스트 작성**

`apps/web/components/backtest/BacktestPage.test.tsx`:

```tsx
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { BacktestPage } from "./BacktestPage";
import * as btApi from "@/lib/api/backtest";
import * as instApi from "@/lib/api/instruments";

vi.mock("@/lib/api/backtest", async () => {
  const actual = await vi.importActual<typeof btApi>("@/lib/api/backtest");
  return { ...actual, runBacktest: vi.fn() };
});
vi.mock("@/lib/api/instruments", () => ({
  searchInstruments: vi.fn(),
  selectInstrument: vi.fn(),
}));

const mockedRun = vi.mocked(btApi.runBacktest);
const mockedSearch = vi.mocked(instApi.searchInstruments);

const FAKE_INST: instApi.InstrumentResult = {
  id: "11111111-1111-1111-1111-111111111111",
  symbol: "005930",
  exchange: "KRX",
  name: "삼성전자",
  currency: "KRW",
  asset_class: "KR_STOCK",
};

function fakeResult(): btApi.BacktestResult {
  const pts = (a: number, b: number) => [
    { date: "2023-05-29", value: a },
    { date: "2026-05-29", value: b },
  ];
  return {
    clamped_start: "2023-05-29",
    end: "2026-05-29",
    normalized_basket: [
      { instrument_id: FAKE_INST.id, symbol: "005930", name: "삼성전자", weight: 1 },
    ],
    equity_series: pts(10_000_000, 12_000_000),
    contributed_series: pts(10_000_000, 10_000_000),
    benchmarks: {
      kospi: { equity_series: pts(10_000_000, 10_500_000), metrics: { total_return_pct: 5, cagr_pct: 1.64, mdd_pct: -8, volatility_pct: 12, twr_pct: 5 } },
      spx: { equity_series: pts(10_000_000, 11_000_000), metrics: { total_return_pct: 10, cagr_pct: 3.2, mdd_pct: -6, volatility_pct: 14, twr_pct: 10 } },
      sixty_forty: { equity_series: pts(10_000_000, 10_700_000), metrics: { total_return_pct: 7, cagr_pct: 2.3, mdd_pct: -5, volatility_pct: 9, twr_pct: 7 } },
    },
    metrics: {
      total_return_pct: 20,
      cagr_pct: 6.27,
      mdd_pct: -10,
      volatility_pct: 15,
      excess_vs_6040_pct: 13,
      total_contributed: 10_000_000,
      final_equity: 12_000_000,
    },
    coverage_warnings: [],
  };
}

async function addLegAndRun() {
  fireEvent.change(screen.getByPlaceholderText("＋ 종목 추가 (검색)"), {
    target: { value: "삼성" },
  });
  const pick = await screen.findByText("삼성전자", {}, { timeout: 2000 });
  fireEvent.click(pick.closest("button")!);
  fireEvent.change(await screen.findByLabelText("005930 비중"), {
    target: { value: "100" },
  });
  fireEvent.click(screen.getByText("백테스트 실행"));
}

describe("BacktestPage", () => {
  beforeEach(() => {
    mockedRun.mockReset();
    mockedSearch.mockReset();
    mockedSearch.mockResolvedValue([FAKE_INST]);
  });

  it("shows empty state before running", () => {
    render(<BacktestPage />);
    expect(
      screen.getByText(/바스켓과 전략을 설정하고 실행하세요/),
    ).toBeInTheDocument();
  });

  it("renders metrics and compare table on success", async () => {
    mockedRun.mockResolvedValueOnce(fakeResult());
    render(<BacktestPage />);
    await addLegAndRun();
    await waitFor(() =>
      expect(screen.getByText("초과수익 vs 60/40")).toBeInTheDocument(),
    );
    expect(screen.getByText("+20.00%")).toBeInTheDocument(); // 전략 총수익률 카드(부호 표기)
    expect(screen.getByText("내 전략")).toBeInTheDocument();
    expect(screen.getByText("KOSPI")).toBeInTheDocument();
    expect(mockedRun).toHaveBeenCalledWith(
      expect.objectContaining({
        period: "3Y",
        initial_cash: 10_000_000,
        basket: [{ instrument_id: FAKE_INST.id, weight: 100 }],
      }),
    );
  });

  it("shows error message when API returns 422", async () => {
    mockedRun.mockResolvedValueOnce({
      error: { code: "INSUFFICIENT_DATA", message: "데이터가 부족합니다", min_days: 30, current_days: 12 },
    });
    render(<BacktestPage />);
    await addLegAndRun();
    await waitFor(() =>
      expect(screen.getByText("데이터가 부족합니다")).toBeInTheDocument(),
    );
  });
});
```

- [ ] **Step 2: 테스트 실행**

Run: `cd apps/web && npx vitest run components/backtest/BacktestPage.test.tsx`
Expected: PASS (3 tests).

- [ ] **Step 3: 커밋**

```bash
git add apps/web/components/backtest/BacktestPage.test.tsx
git commit -m "test(web): 백테스트 페이지 테스트 (빈 상태·성공·422 에러)"
```

---

### Task 16: 문서 갱신 (STATUS·ROADMAP·ARCHITECTURE·USER_ACTIONS)

CLAUDE.md "문서 업데이트 규칙 (MANDATORY)" 이행. 기능 완료 → 4개 문서 갱신 없이는 작업 미완료로 간주. T1~T15가 docs를 건드리지 않으므로 아래 앵커는 안정적이다. USER_ACTIONS에는 백필 선행 조건(spec §3-3: NASDAQ 시드 30종목만 보유)을 반드시 등재.

**Files:**
- Modify: `docs/STATUS.md`
- Modify: `docs/ROADMAP.md`
- Modify: `docs/ARCHITECTURE.md`
- Modify: `docs/USER_ACTIONS.md`

- [ ] **Step 1: STATUS.md — 완료 목록에 백테스트 추가**

`docs/STATUS.md`의 `- ✅ AI 채팅 교육자 역할 …` 줄 바로 **다음**에 한 줄 삽입:

```markdown
- ✅ Paper Trading 백테스트 (서브시스템 B) — 과거 시점 시뮬레이션. `/app/backtest`, 선언적 바스켓(최대 10·자동 정규화) + 2축 전략(일시불/월 적립 × 없음·분기·반기·연), NAV/유닛 적립중립 수익률, 단일 `simulate()`로 KOSPI·S&P·한미 60/40 동시 비교(초과수익 vs 60/40), XIRR·MDD·변동성, 5년 클램프 + 커버리지 경고. 무상태(신규 테이블 0). `internal/portfolio/backtest.go` + `POST /v1/backtest/run`.
```

- [ ] **Step 2: STATUS.md — 최근 변경 이력 최상단 + 마지막 업데이트**

`## 최근 변경 이력` 헤더 아래 첫 항목(`- 2026-05-29 AI 채팅 교육자 역할 추가 …`) **앞**에 삽입:

```markdown
- 2026-05-29 백테스트(서브시스템 B) 출시 — `/app/backtest` 신규 탭(사이드바 History 아이콘). 선언적 바스켓(최대 10종목·실행 시 자동 정규화) + 2축 전략(투입: 일시불/월 적립 × 리밸런싱: 없음·분기·반기·연) 과거 시뮬레이션. NAV/유닛 펀드 회계로 적립 중립 수익률(TWR) + 단일 `simulate()`로 전략·KOSPI·S&P·한미 60/40 4종을 동일 캐시플로우·리밸런싱으로 실행 → 초과수익(vs 60/40). XIRR 머니가중 CAGR(Newton+이분법, 실패 시 null) + MDD + 변동성. 5년·종료일 오늘 클램프(레그별 최초 가용일 + USD 레그 fx 반영), <30일 INSUFFICIENT_DATA, 데이터 짧은 종목 커버리지 경고. 무상태(신규 테이블 0, 공개 가격·지수·환율만 슈퍼유저 풀 read — `db.AsUser`/RLS 불필요, 인증 게이트만). `internal/portfolio/backtest.go`(알파와 패키지 공존) + `POST /v1/backtest/run` + 5선 멀티라인 평가액 차트 + 비교표. 정체성 spec §1 3축의 '백테스트' 약속 완성. 라이브 Paper Trading과 별개 서브시스템.
```

`마지막 업데이트:` 줄이 오늘 날짜(2026-05-29)인지 확인하고 다르면 갱신.

- [ ] **Step 3: STATUS.md 변경 확인 + 커밋 (4개 문서 묶어 마지막에)**

본 Step에서는 커밋하지 않는다 — Step 8에서 4개 문서를 한 커밋으로 묶는다.

- [ ] **Step 4: ROADMAP.md — "현재 추천 다음 작업" 재설정**

다음 블록(기존):

```markdown
알파 카드 · AI 매매 일기 · Paper Trading (라이브) · AI 교육자 역할 출시 완료. 3축 정체성 핵심 모두 충족. 다음:

1. **Paper Trading 백테스트(서브시스템 B)** — 과거 시점 시뮬레이션, 정액 적립·리밸런싱 전략. spec §1 3축의 "백테스트" 약속 완성. brainstorm→spec→plan 풀 사이클 필요
2. **운영 자동화** — 부팅 시 지수 자동 백필(비동기·실패 무시) + `release_command` 마이그레이션. 사용자 액션을 "계정·키 1회 셋업"으로 압축

(CSV import는 드롭(2026-05-29). 사유: 유일 가치가 "실 자산 초기 입력 절감"뿐인데 증권사별 포맷 파싱 유지보수 비용 과다. 실 자산 입력은 수동 UX + Phase 3 KIS Open API 자동 동기화로 대체.)
(AI 교육자 역할 완료(2026-05-29): 개념 질문 친절 답변 + footer 게이팅. STATUS 참조.)
```

를 다음으로 교체:

```markdown
알파 카드 · AI 매매 일기 · Paper Trading (라이브) · 백테스트(서브시스템 B) · AI 교육자 역할 출시 완료. 3축 정체성 + 백테스트 약속 모두 충족. 다음:

1. **운영 자동화** — 부팅 시 지수 자동 백필(비동기·실패 무시) + `release_command` 마이그레이션. 사용자 액션을 "계정·키 1회 셋업"으로 압축

(CSV import는 드롭(2026-05-29). 사유: 유일 가치가 "실 자산 초기 입력 절감"뿐인데 증권사별 포맷 파싱 유지보수 비용 과다. 실 자산 입력은 수동 UX + Phase 3 KIS Open API 자동 동기화로 대체.)
(AI 교육자 역할 완료(2026-05-29): 개념 질문 친절 답변 + footer 게이팅. STATUS 참조.)
(백테스트 완료(2026-05-29): NAV/유닛 통합 바스켓 시뮬레이터 + KOSPI·S&P·60/40 비교. STATUS 참조.)
```

- [ ] **Step 5: ROADMAP.md — Phase 2 차별화 표 행 갱신**

다음 행(기존):

```markdown
| **Phase 2 핵심 차별화** | **Paper Trading 백테스트(서브시스템 B)** | 과거 시점 시뮬레이션, 정액 적립·리밸런싱 전략. Paper Trading (라이브) 출시 완료(2026-05-28). 백테스트는 별도 서브시스템 |
```

를 다음으로 교체(완료 처리):

```markdown
| ~~Phase 2 핵심 차별화~~ | ~~Paper Trading 백테스트(서브시스템 B)~~ | **완료(2026-05-29)** — NAV/유닛 통합 바스켓 시뮬레이터(전략 + KOSPI·S&P·60/40 동일 캐시플로우) + 초과수익. STATUS 참조 |
```

- [ ] **Step 6: ARCHITECTURE.md — 핵심 설계 결정 §11 추가**

`### 10. 차트 라이브러리 — recharts (W5)` 섹션의 Trade-off 줄 다음, 파일 끝 `---\n업데이트 규칙:` **앞**에 삽입:

```markdown
### 11. 백테스트 엔진 — 통합 바스켓 시뮬레이터 (서브시스템 B)

**Why**: 전략과 벤치마크(KOSPI·S&P·한미 60/40)를 같은 코드로 돌려야 초과수익 비교가 공정하다. 적립식(DCA)에서 단순 누적수익률은 투입 시점에 오염되므로 NAV/유닛 펀드 회계로 "수익률"과 "투입"을 분리한다.

**How**:
- "모든 것은 바스켓" — 순수 `simulate(days, legs, plan, rebalance)` 하나로 전략 + 벤치마크 3종을 동일 캐시플로우·리밸런싱으로 4회 실행. 벤치마크는 지수=비중 100% 바스켓(60/40은 KOSPI 60·SPX 40).
- NAV/유닛: 투입 시 현재 NAV로 유닛 발행 → NAV는 투입에 불변. TWR = NAV(tN)−1, 머니가중 CAGR은 XIRR(Newton + 이분법 폴백, 실패 시 null).
- 통화 정합: 각 레그(KOSPI·SPX)는 자기 t0에 정규화 → 지수·주가 스케일 상쇄. SPX 레그만 시점별 USD/KRW 적용(이중 환산 방지).
- 5년·종료일 오늘 클램프: 시작일 = max(요청, 전략·벤치마크 레그별 최초 가용일, USD 레그 fx 최초 가용일). <30일이면 INSUFFICIENT_DATA. 데이터 짧은 전략 레그는 커버리지 경고.
- 무상태: 신규 테이블 0. 사용자 데이터 미접근 → `db.AsUser`/RLS 불필요(인증 게이트만, 공개 가격·지수·환율을 슈퍼유저 풀로 read). `internal/portfolio/` 패키지에서 알파 카드와 공존.

**Trade-off**: 전진 채움(forward-fill)으로 희소 가격을 클램프 축에 조밀화 — 결측 구간은 직전 종가 유지(룩어헤드 없음). 파라미터 최적화·전략 자동 추천은 규제·과적합 우려로 영구 제외(spec §13).

```

- [ ] **Step 7: USER_ACTIONS.md — 백필 선행 조건 등재**

`## 🔴 시급` 절의 기존 지수 백필 항목(`- [ ] **지수 5년 백필 …**` 블록) **다음**에 삽입:

```markdown
- [ ] **백테스트 대상 종목 가격 백필** — *백테스트 동작 전제 (spec §3-3)*
  - KOSPI·KOSDAQ 전종목 + 지수는 W2b 백필 CLI로 적재됨. **NASDAQ은 시드 30종목만** 보유 → 그 외 미국 종목을 바스켓에 넣으면 클램프되거나 "데이터 부족"으로 거부될 수 있음.
  - 미국 종목 백필: `cd apps/api && set -a; source .env; set +a; go run ./cmd/backfill --market=NASDAQ --years=5` (대상 확장은 백필 CLI의 NASDAQ 시드 목록 편집)
  - 운영: `flyctl ssh console -C "/app/backfill --market=NASDAQ --years=5"`
  - 지수 백필(위 항목)이 선행돼야 벤치마크 3종(KOSPI·S&P·60/40)이 그려진다.
```

- [ ] **Step 8: 4개 문서 묶어 커밋**

```bash
git add docs/STATUS.md docs/ROADMAP.md docs/ARCHITECTURE.md docs/USER_ACTIONS.md
git commit -m "docs: 백테스트(서브시스템 B) 완료 반영 (STATUS·ROADMAP·ARCHITECTURE §11·USER_ACTIONS 백필 선행조건)"
```

---

## 완료 기준 (Definition of Done)

- [ ] T1~T10: `cd apps/api && go test ./...` 그린 (integration 태그 제외 기본 빌드/유닛 전부 PASS).
- [ ] T1~T10: `go build ./...` 성공, `POST /v1/backtest/run` 라우트 등록됨.
- [ ] T11~T15: `cd apps/web && npx tsc --noEmit` + `npx vitest run components/backtest/` PASS, `npx next build`에 `/app/backtest` 포함.
- [ ] 사이드바에 "백테스트" 탭 노출(Paper 다음·설정 앞).
- [ ] T16: STATUS·ROADMAP·ARCHITECTURE·USER_ACTIONS 4개 문서 갱신 커밋.
- [ ] 구현 완료 후 `superpowers:requesting-code-review`로 전체 리뷰 (CLAUDE.md MANDATORY 3단계).
