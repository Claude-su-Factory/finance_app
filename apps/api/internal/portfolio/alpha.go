// Package portfolio — 포트폴리오 분석 로직 (알파, 백테스트 등 공통 시계열 계산).
package portfolio

import "errors"

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

// Service·NewService 정의는 Task 3에서. 본 Task에서는 타입만.
