// Package portfolio — 포트폴리오 분석 로직 (알파, 백테스트 등 공통 시계열 계산).
package portfolio

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/quotient/quotient/apps/api/internal/db"
)

// Period는 알파 카드 기간 토글 값.
type Period string

const (
	Period1M  Period = "1m"
	Period90D Period = "90d"
	Period1Y  Period = "1y"
	PeriodAll Period = "all"
)

func ParsePeriod(s string) (Period, error) {
	switch Period(s) {
	case Period1M, Period90D, Period1Y, PeriodAll:
		return Period(s), nil
	}
	return "", errors.New("invalid period")
}

// Days는 period의 일수 (All은 0 — 가입 시점 시작 의미).
func (p Period) Days() int {
	switch p {
	case Period1M:
		return 30
	case Period90D:
		return 90
	case Period1Y:
		return 365
	}
	return 0
}

// SeriesPoint는 (일자, 시작점 대비 누적 % 수익률).
type SeriesPoint struct {
	Date     string  `json:"date"`
	ValuePct float64 `json:"value_pct"`
}

// DataGap은 종목별 데이터 부족 정보 (UI 라벨용).
type DataGap struct {
	Symbol         string `json:"symbol"`
	FirstPriceDate string `json:"first_price_date"`
}

// AlphaResult는 핸들러가 JSON으로 반환할 최종 형태.
type AlphaResult struct {
	Period        Period       `json:"period"`
	DaysRequested int          `json:"days_requested"`
	DaysUsed      int          `json:"days_used"`
	Since         string       `json:"since"`
	FxMode        string       `json:"fx_mode"` // "spot"
	Model         string       `json:"model"`   // "current_holdings_backward_simulation"
	Portfolio     PortfolioRet `json:"portfolio"`
	Benchmarks    []Benchmark  `json:"benchmarks"`
}

type PortfolioRet struct {
	TotalReturnPct float64       `json:"total_return_pct"`
	Series         []SeriesPoint `json:"series"`
	DataGaps       []DataGap     `json:"data_gaps,omitempty"`
}

type Benchmark struct {
	Key            string  `json:"key"`
	Label          string  `json:"label"`
	TotalReturnPct float64 `json:"total_return_pct"`
	AlphaPP        float64 `json:"alpha_pp"`
	// omitempty 제거 — spec §4 약속 "kr_us_6040.series: null" 보장.
	// nil slice + 기본 직렬화 = JSON `null`. 다른 두 벤치마크는 채워진 슬라이스.
	Series []SeriesPoint `json:"series"`
}

// InsufficientDataError는 빈 상태 응답(422)을 위한 구분 가능 에러.
type InsufficientDataError struct {
	Reason      string // "account_too_young" | "no_holdings"
	MinDays     int
	CurrentDays int
}

func (e *InsufficientDataError) Error() string { return "insufficient data: " + e.Reason }

// HoldingRow는 alpha 계산용 holdings 행.
type HoldingRow struct {
	InstrumentID string
	Symbol       string
	Currency     string
	Quantity     float64
}

// PricePoint는 (date YYYY-MM-DD, close 종가).
type PricePoint struct {
	Date  string
	Close float64
}

// Deps는 AlphaService가 DB에서 가져와야 하는 모든 read.
// production 구현은 internal/portfolio/pg_deps.go (Task 3에서),
// test 구현은 alpha_test.go의 fakeAlphaDeps.
type Deps interface {
	UserCreatedAt(ctx context.Context, exec db.Executor, uid string) (time.Time, error)
	UserHoldings(ctx context.Context, exec db.Executor, uid string) ([]HoldingRow, error)
	TradingDays(ctx context.Context, pool db.Executor, since, until time.Time) ([]string, error)
	InstrumentClosesOnDates(ctx context.Context, pool db.Executor, instrumentID string, dates []string) (map[string]float64, error)
	FxRatesOnDates(ctx context.Context, pool db.Executor, currency string, dates []string) (map[string]float64, error)
	BenchmarkSeries(ctx context.Context, pool db.Executor, symbol string, dates []string) ([]PricePoint, error)
}

// Service는 알파 계산 진입점. 호출자가 exec(JWT 트랜잭션)와 pool(공개 read)을 분리해 주입.
type Service struct {
	deps Deps
	now  func() time.Time
}

// NewServiceWithDeps는 테스트용. production은 Task 3의 NewService.
func NewServiceWithDeps(deps Deps, fixedNow time.Time) *Service {
	return &Service{deps: deps, now: func() time.Time { return fixedNow }}
}

const MinAccountDays = 7

func (s *Service) Compute(ctx context.Context, exec db.Executor, pool db.Executor, uid string, period Period) (*AlphaResult, error) {
	now := s.now()
	today := now.Truncate(24 * time.Hour)
	requested := period.Days() // 0 for "all"

	createdAt, err := s.deps.UserCreatedAt(ctx, exec, uid)
	if err != nil {
		return nil, err
	}
	accountDays := int(today.Sub(createdAt.Truncate(24 * time.Hour)).Hours() / 24)
	if accountDays < MinAccountDays {
		return nil, &InsufficientDataError{Reason: "account_too_young", MinDays: MinAccountDays, CurrentDays: accountDays}
	}

	daysUsed := requested
	if requested == 0 || accountDays < requested {
		daysUsed = accountDays
	}
	since := today.AddDate(0, 0, -daysUsed)
	if since.Before(createdAt.Truncate(24 * time.Hour)) {
		since = createdAt.Truncate(24 * time.Hour)
	}

	holdings, err := s.deps.UserHoldings(ctx, exec, uid)
	if err != nil {
		return nil, err
	}
	if len(holdings) == 0 {
		return nil, &InsufficientDataError{Reason: "no_holdings"}
	}

	tradingDays, err := s.deps.TradingDays(ctx, pool, since, today)
	if err != nil {
		return nil, err
	}
	if len(tradingDays) < 2 {
		return nil, &InsufficientDataError{Reason: "account_too_young", MinDays: MinAccountDays, CurrentDays: accountDays}
	}

	// 종목별 가격·환율 일괄 조회
	priceByInst := map[string]map[string]float64{}
	for _, h := range holdings {
		m, err := s.deps.InstrumentClosesOnDates(ctx, pool, h.InstrumentID, tradingDays)
		if err != nil {
			return nil, err
		}
		priceByInst[h.InstrumentID] = m
	}
	fxByCur := map[string]map[string]float64{"KRW": {}}
	for _, h := range holdings {
		if h.Currency == "KRW" || fxByCur[h.Currency] != nil {
			continue
		}
		m, err := s.deps.FxRatesOnDates(ctx, pool, h.Currency, tradingDays)
		if err != nil {
			return nil, err
		}
		fxByCur[h.Currency] = m
	}

	// 일자별 포트 가치(KRW) 합산 + data_gaps 수집
	portValues := make([]float64, len(tradingDays))
	gapBySymbol := map[string]string{}
	for di, d := range tradingDays {
		var total float64
		for _, h := range holdings {
			price, ok := priceByInst[h.InstrumentID][d]
			if !ok || price == 0 {
				if _, seen := gapBySymbol[h.Symbol]; !seen {
					if first := firstAvailable(priceByInst[h.InstrumentID]); first != "" {
						gapBySymbol[h.Symbol] = first
					}
				}
				continue
			}
			fx := 1.0
			if h.Currency != "KRW" {
				fx = lookupFxForward(fxByCur[h.Currency], tradingDays, di)
			}
			total += h.Quantity * price * fx
		}
		portValues[di] = total
	}

	if portValues[0] == 0 {
		return nil, &InsufficientDataError{Reason: "no_holdings"}
	}

	portSeries := normalizeToPct(tradingDays, portValues)
	portTotal := portSeries[len(portSeries)-1].ValuePct

	// 벤치마크 시리즈
	kospi, err := computeBenchmarkSeries(ctx, s.deps, pool, "KOSPI", tradingDays)
	if err != nil {
		return nil, err
	}
	sp500, err := computeBenchmarkSeries(ctx, s.deps, pool, "SPX", tradingDays)
	if err != nil {
		return nil, err
	}
	kr40 := composite6040(kospi, sp500)

	var gaps []DataGap
	for sym, first := range gapBySymbol {
		gaps = append(gaps, DataGap{Symbol: sym, FirstPriceDate: first})
	}
	sortGaps(gaps)

	return &AlphaResult{
		Period:        period,
		DaysRequested: requested,
		DaysUsed:      daysUsed,
		Since:         since.Format("2006-01-02"),
		FxMode:        "spot",
		Model:         "current_holdings_backward_simulation",
		Portfolio: PortfolioRet{
			TotalReturnPct: portTotal,
			Series:         portSeries,
			DataGaps:       gaps,
		},
		Benchmarks: []Benchmark{
			{Key: "kospi", Label: "KOSPI", Series: kospi, TotalReturnPct: lastPct(kospi), AlphaPP: portTotal - lastPct(kospi)},
			{Key: "sp500", Label: "S&P 500", Series: sp500, TotalReturnPct: lastPct(sp500), AlphaPP: portTotal - lastPct(sp500)},
			{Key: "kr_us_6040", Label: "한미 60/40", Series: nil, TotalReturnPct: lastPct(kr40), AlphaPP: portTotal - lastPct(kr40)},
		},
	}, nil
}

// --- helpers ---

func firstAvailable(m map[string]float64) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys[0]
}

// lookupFxForward — 현재 idx 일자 fx 없으면 직전 일자로 후퇴.
// trading days 첫 일자에도 fx가 없으면 m["__before"]로 저장된 history fx 사용 (FxRatesOnDates가 since-1d 확장 fetch한 결과).
func lookupFxForward(m map[string]float64, dates []string, idx int) float64 {
	for i := idx; i >= 0; i-- {
		if v, ok := m[dates[i]]; ok && v > 0 {
			return v
		}
	}
	if v, ok := m["__before"]; ok && v > 0 {
		return v
	}
	return 1.0
}

func normalizeToPct(dates []string, values []float64) []SeriesPoint {
	out := make([]SeriesPoint, 0, len(dates))
	start := values[0]
	for i, d := range dates {
		pct := 0.0
		if start != 0 {
			pct = (values[i] - start) / start * 100
		}
		out = append(out, SeriesPoint{Date: d, ValuePct: pct})
	}
	return out
}

func computeBenchmarkSeries(ctx context.Context, deps Deps, pool db.Executor, symbol string, dates []string) ([]SeriesPoint, error) {
	pts, err := deps.BenchmarkSeries(ctx, pool, symbol, dates)
	if err != nil {
		return nil, err
	}
	if len(pts) < 2 {
		return nil, errors.New("benchmark series too short: " + symbol)
	}
	values := make([]float64, len(pts))
	dts := make([]string, len(pts))
	for i, p := range pts {
		dts[i] = p.Date
		values[i] = p.Close
	}
	return normalizeToPct(dts, values), nil
}

// composite6040: 0.6*KOSPI + 0.4*SPX (이미 누적 % 정규화).
// 두 시리즈는 KR·US 영업일 캘린더가 달라 길이가 다를 수 있음 → union dates + forward-fill.
func composite6040(kospi, sp500 []SeriesPoint) []SeriesPoint {
	kospiM := map[string]float64{}
	for _, p := range kospi {
		kospiM[p.Date] = p.ValuePct
	}
	spM := map[string]float64{}
	for _, p := range sp500 {
		spM[p.Date] = p.ValuePct
	}
	dateSet := map[string]struct{}{}
	for d := range kospiM {
		dateSet[d] = struct{}{}
	}
	for d := range spM {
		dateSet[d] = struct{}{}
	}
	dates := make([]string, 0, len(dateSet))
	for d := range dateSet {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	out := make([]SeriesPoint, 0, len(dates))
	lastK, lastS := 0.0, 0.0
	for _, d := range dates {
		if v, ok := kospiM[d]; ok {
			lastK = v
		}
		if v, ok := spM[d]; ok {
			lastS = v
		}
		out = append(out, SeriesPoint{Date: d, ValuePct: 0.6*lastK + 0.4*lastS})
	}
	return out
}

func lastPct(s []SeriesPoint) float64 {
	if len(s) == 0 {
		return 0
	}
	return s[len(s)-1].ValuePct
}

func sortGaps(g []DataGap) {
	sort.Slice(g, func(i, j int) bool { return g[i].Symbol < g[j].Symbol })
}
