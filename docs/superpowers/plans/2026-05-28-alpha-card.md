# 알파 카드 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 홈 1행 3번째에 "포트폴리오 vs KOSPI·S&P 500·한미 60/40" 누적 수익률 비교 카드를 추가한다. 1M/90D/1Y/All 기간 토글 + 시점별 환율 + backward simulation.

**Architecture:** 백엔드 신규 패키지 `internal/portfolio/alpha.go`가 holdings·prices·fx 시계열을 합쳐 누적 수익률 시리즈를 계산하고, 핸들러 `GET /v1/portfolio/alpha`로 노출. 프론트 `AlphaCard` 컴포넌트가 토글·텍스트·SVG 차트를 렌더. 사용자 데이터 read는 `db.AsUser` JWT 트랜잭션, 공개 가격·환율 read는 슈퍼유저 풀.

**Tech Stack:** Go 1.25 (chi v5 + pgx v5) · Next.js 16 · Tailwind v4 · Supabase Postgres.

**Spec:** [`docs/superpowers/specs/2026-05-28-alpha-card-design.md`](../specs/2026-05-28-alpha-card-design.md)

**사전 조건**:
- 지수(KOSPI·SPX) 5년 일봉이 `public.prices`에 적재돼 있어야 한다. 기존 `cmd/backfill`은 주식만 → **Task 0에서 backfill CLI를 지수까지 확장**.
- 그 외 마이그레이션·시드는 모두 적용된 상태(STATUS 기준).

---

## File Structure

신규:
- `apps/api/internal/portfolio/alpha.go` — AlphaService + 알파 계산 알고리즘
- `apps/api/internal/portfolio/alpha_test.go` — 6 unit 케이스
- `apps/api/internal/portfolio/alpha_integration_test.go` — integration build tag, 실 Supabase
- `apps/api/internal/handlers/portfolio_alpha.go` — HTTP handler
- `apps/api/internal/handlers/portfolio_alpha_test.go` — handler unit (fake service)
- `apps/web/components/home/AlphaCard.tsx` — 카드 컴포넌트
- `apps/web/components/home/AlphaCard.test.tsx` — vitest 렌더 테스트

수정:
- `apps/api/cmd/backfill/main.go` — 지수(KOSPI·KOSDAQ·SPX·NDX·DJI) 5년 백필 추가 (Task 0)
- `apps/api/internal/router/router.go` — 라우트 등록
- `apps/api/cmd/server/main.go` — handler 와이어링
- `apps/web/lib/api/portfolio.ts` — `getAlpha(period)` 함수 (신규 파일)
- `apps/web/components/home/HomeDashboard.tsx` — 그리드 재배치 (알파 1행 3번째 + 브리핑 마지막 행 와이드)
- `apps/web/components/home/AlphaCard.test.tsx` — vitest 4 케이스 (정상·account_too_young·no_holdings·토글)
- `docs/STATUS.md` / `docs/ROADMAP.md` — 완료 반영

---

## Task 0: 지수 5년 백필 (사전 조건)

**Files:**
- Modify: `apps/api/cmd/backfill/main.go`

**Context:** 기존 backfill CLI는 `exchange = 'KRX' and is_active = true` 같은 주식 조건으로 KIND·Yahoo에서 일봉을 적재한다. 지수(`KOSPI`, `SPX` 등 `asset_class='INDEX'`)는 별도 cron이 매일 1점만 추가 → MVP 출시 시점까지 누적 수십 일치만 존재. 알파 카드 90D/1Y가 동작하려면 지수도 5년 백필 필요.

지수 yahoo symbol 매핑:
- KOSPI → `^KS11`
- KOSDAQ → `^KQ11`
- SPX → `^GSPC`
- NDX → `^IXIC` (NASDAQ Composite — 실제 NDX는 `^NDX`지만 `instruments` 시드 확인 후 결정)
- DJI → `^DJI`

기존 `internal/sources/yahoo` 클라이언트가 `.KS`/`.KQ` 매핑은 처리하나 `^GSPC` 같은 인덱스 ticker는 별도 패스 필요. 확인 → 추가.

- [ ] **Step 1: yahoo 클라이언트의 instrument → yahoo symbol 매핑 확인**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && grep -n "yahooSymbol\|YahooSymbol" internal/sources/yahoo/*.go internal/schedule/yahoo_symbols.go 2>&1 | head -20
```

매핑 함수가 INDEX asset_class를 처리하지 않으면 Step 2에서 확장. 처리하면 Step 3로 직행.

- [ ] **Step 2: yahoo 심볼 매핑에 INDEX 분기 추가 (필요 시)**

`apps/api/internal/schedule/yahoo_symbols.go`(또는 동등 위치)의 매핑 함수에 추가:

```go
// IndexYahooSymbol returns Yahoo Finance ticker for index instruments.
func IndexYahooSymbol(symbol string) string {
    switch symbol {
    case "KOSPI":
        return "^KS11"
    case "KOSDAQ":
        return "^KQ11"
    case "SPX":
        return "^GSPC"
    case "NDX":
        return "^IXIC"
    case "DJI":
        return "^DJI"
    }
    return ""
}
```

(파일 구조와 함수명은 실제 grep 결과에 맞춰 조정. INDEX용 매핑이 이미 있으면 시드만 검증.)

- [ ] **Step 3: backfill CLI에 indices 분기 추가**

`apps/api/cmd/backfill/main.go`를 읽고 기존 stocks 백필 로직을 그대로 따라 indices 백필 분기를 추가. 기존 코드가:

```go
// (예시 — 실제 구조는 파일 읽고 따라가야 함)
case "KOSPI":
    fetchKindThenYahoo(...)
case "KOSDAQ":
    fetchKindThenYahoo(...)
case "NASDAQ":
    fetchNasdaqYahoo(...)
```

같은 switch라면 새 case 추가:

```go
case "indices":
    // INDEX asset_class 5종목 (KOSPI·KOSDAQ·SPX·NDX·DJI) 5년치 Yahoo 일봉
    err := backfillIndices(ctx, pool, yahooClient, 5)
    if err != nil {
        log.Fatal(err)
    }
```

`backfillIndices` 함수:

```go
func backfillIndices(ctx context.Context, pool *pgxpool.Pool, y *yahoo.Client, years int) error {
    rows, err := pool.Query(ctx, `
        select id::text, symbol from public.instruments
        where asset_class = 'INDEX' and is_active = true
        order by symbol
    `)
    if err != nil { return err }
    defer rows.Close()
    type idx struct{ ID, Symbol string }
    var list []idx
    for rows.Next() {
        var x idx
        if err := rows.Scan(&x.ID, &x.Symbol); err != nil { return err }
        list = append(list, x)
    }
    for _, x := range list {
        ysym := IndexYahooSymbol(x.Symbol)
        if ysym == "" {
            slog.Warn("no yahoo symbol for index", "symbol", x.Symbol)
            continue
        }
        bars, err := y.FetchDaily(ctx, ysym, years*365)
        if err != nil {
            slog.Warn("backfill index failed", "symbol", x.Symbol, "err", err)
            continue
        }
        // 기존 stocks 백필이 사용하는 prices INSERT 헬퍼 재사용
        if err := upsertPrices(ctx, pool, x.ID, bars); err != nil {
            return err
        }
        slog.Info("backfilled index", "symbol", x.Symbol, "bars", len(bars))
    }
    return nil
}
```

(`upsertPrices`·`yahoo.Client.FetchDaily`·시그니처는 기존 코드 따라 정확히 조정.)

- [ ] **Step 4: 실행 — 지수 5년 백필**

로컬 Supabase 상태에서:
```bash
cd /Users/yuhojin/Desktop/finance/apps/api && set -a; source .env; set +a; go run ./cmd/backfill indices
```

Expected: 5종목 각 ~1,250 bars 적재. 검증:
```bash
psql "$DATABASE_URL" -c "select i.symbol, count(*) from prices p join instruments i on i.id = p.instrument_id where i.asset_class='INDEX' group by i.symbol order by 1;"
```
→ 각 행 count가 1,000+ (영업일 기준).

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/api/cmd/backfill apps/api/internal/schedule && git commit -m "feat(backfill): 지수(KOSPI/KOSDAQ/SPX/NDX/DJI) 5년 일봉 적재 — 알파 카드 사전 조건"
```

> 운영 배포 시점에서도 prod DB에 1회 실행 필요. `flyctl ssh console -C "/app/backfill indices"` 또는 로컬에서 prod DSN으로.

---

## Task 1: 백엔드 — Portfolio 패키지 골격 + 타입 정의

**Files:**
- Create: `apps/api/internal/portfolio/alpha.go`
- Create: `apps/api/internal/portfolio/alpha_test.go`

- [ ] **Step 1: 타입·인터페이스만 먼저 정의 (실패 테스트 작성용)**

`apps/api/internal/portfolio/alpha.go`:

```go
// Package portfolio — 포트폴리오 분석 로직 (알파, 백테스트 등 공통 시계열 계산).
package portfolio

import (
	"context"
	"errors"
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
	Symbol          string `json:"symbol"`
	FirstPriceDate  string `json:"first_price_date"`
}

// AlphaResult는 핸들러가 JSON으로 반환할 최종 형태.
type AlphaResult struct {
	Period          Period       `json:"period"`
	DaysRequested   int          `json:"days_requested"`
	DaysUsed        int          `json:"days_used"`
	Since           string       `json:"since"`
	FxMode          string       `json:"fx_mode"` // "spot"
	Model           string       `json:"model"`   // "current_holdings_backward_simulation"
	Portfolio       PortfolioRet `json:"portfolio"`
	Benchmarks      []Benchmark  `json:"benchmarks"`
}

type PortfolioRet struct {
	TotalReturnPct float64       `json:"total_return_pct"`
	Series         []SeriesPoint `json:"series"`
	DataGaps       []DataGap     `json:"data_gaps,omitempty"`
}

type Benchmark struct {
	Key            string        `json:"key"`
	Label          string        `json:"label"`
	TotalReturnPct float64       `json:"total_return_pct"`
	AlphaPP        float64       `json:"alpha_pp"`
	// omitempty 제거 — spec §4 약속 "kr_us_6040.series: null" 보장.
	// nil slice + 기본 직렬화 = JSON `null`. 다른 두 벤치마크는 채워진 슬라이스.
	Series []SeriesPoint `json:"series"`
}

// ErrInsufficientData는 빈 상태 응답(422)을 위한 구분 가능 에러.
type InsufficientDataError struct {
	Reason       string // "account_too_young" | "no_holdings"
	MinDays      int
	CurrentDays  int
}

func (e *InsufficientDataError) Error() string { return "insufficient data: " + e.Reason }

// Service·NewService 정의는 Task 3에서. 본 Task에서는 타입만.
```

> Task 1 단독 commit 시점에는 `Service`·`Compute` 정의가 없다. Service는 Task 3의 `pg_deps.go`에서 production용으로 한 번만 정의되어 중복 선언 회피.

- [ ] **Step 2: 실패 테스트 작성**

`apps/api/internal/portfolio/alpha_test.go`:

```go
package portfolio_test

import (
	"testing"

	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

func TestParsePeriod(t *testing.T) {
	cases := map[string]bool{
		"1m": true, "90d": true, "1y": true, "all": true,
		"":     false,
		"30d":  false,
		"2y":   false,
		"FOO":  false,
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
```

- [ ] **Step 3: 빌드 + 테스트 실행**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./internal/portfolio/... && go test ./internal/portfolio/ -v
```

Expected: `TestParsePeriod` + `TestPeriodDays` 모두 PASS.

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/api/internal/portfolio/ && git commit -m "feat(portfolio): alpha 패키지 골격 — Period·SeriesPoint·AlphaResult 타입"
```

---

## Task 2: AlphaService — 시계열 계산 코어

**Files:**
- Modify: `apps/api/internal/portfolio/alpha.go` — `Service` 정의 + `Compute` 본문 + private helpers
- Modify: `apps/api/internal/portfolio/alpha_test.go` — fake repos + 핵심 계산 테스트

> **빌드 메모**: 이 Task의 Step 1·2는 컴파일 안 됨 (fakeAlphaDeps가 Deps 인터페이스를 Step 3까지 완성). 빌드 check는 Step 4에서 처음 수행. 중간 commit 금지.

- [ ] **Step 1: 실패 테스트 먼저 — 기본 2종목 계산**

`apps/api/internal/portfolio/alpha_test.go` 끝에 추가:

```go
import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

// fakeExec는 db.Executor stub. 호출자가 expected SQL과 응답을 미리 등록.
type fakeExec struct {
	// 구현은 Step 3에서. 일단 컴파일용 placeholder.
}

func (f *fakeExec) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)        { return nil, errors.New("nyi") }
func (f *fakeExec) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row               { return nil }
func (f *fakeExec) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error){ return pgconn.CommandTag{}, errors.New("nyi") }

// fakeAlphaDeps는 모든 DB 호출을 흉내내는 가짜. private helper로 주입.
type fakeAlphaDeps struct {
	createdAt    time.Time
	holdings     []portfolio.HoldingRow
	prices       map[string]map[string]float64 // instrument_id → date(YYYY-MM-DD) → close
	fxRates      map[string]map[string]float64 // currency → date → rate (to KRW)
	kospiSeries  []portfolio.PricePoint         // date + close
	sp500Series  []portfolio.PricePoint
	tradingDays  []string                       // YYYY-MM-DD, 순서대로
}

func TestCompute_BasicTwoHoldings(t *testing.T) {
	// 90D, KRW 종목 1 + USD 종목 1, 단순 +10%·+20% 시나리오
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
			"KRW": {"2026-02-27": 1, "2026-05-28": 1}, // KRW는 환율 1 고정
		},
		kospiSeries: []portfolio.PricePoint{
			{Date: "2026-02-27", Close: 2500}, {Date: "2026-05-28", Close: 2580},
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

func abs(f float64) float64 { if f < 0 { return -f }; return f }
func mustTime(s string) time.Time { t, _ := time.Parse(time.RFC3339, s); return t }
```

> **메모**: `HoldingRow`, `PricePoint`, `NewServiceWithDeps`, `Deps` 인터페이스는 Step 2에서 정의한다. 위 테스트는 이를 사용한다.

- [ ] **Step 2: alpha.go에 Deps 인터페이스 + 본문 구현**

`apps/api/internal/portfolio/alpha.go` 끝부분(`Compute` 함수 자리)에:

```go
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

// NewServiceWithDeps는 테스트용. production은 NewService.
func NewServiceWithDeps(deps Deps, fixedNow time.Time) *Service {
	return &Service{deps: deps, now: func() time.Time { return fixedNow }}
}

type Service struct {
	deps Deps
	now  func() time.Time
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

	// TradingDays는 KOSPI ∪ SPX 합집합 (시점 매칭이 다른 영업일 캘린더에서도 보장).
	// 종목 가격·fx 결손은 forward-fill로 처리.
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
		return nil, &InsufficientDataError{Reason: "no_holdings"} // 모든 종목 데이터 부족
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
	kr40 := composite6040(kospi, sp500) // 60% KOSPI + 40% S&P, 시점별 가중 평균

	// data_gaps slice (안정 순서를 위해 symbol 정렬)
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
// trading days 첫 일자에도 fx가 없으면 m["__before"]로 저장된 history fx 사용 (FxRatesOnDates가 since-7d 확장 fetch한 결과).
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
```

import 블록에 `"sort"` 추가.

기존 `Compute` 빈 구현·`NewService()` 함수는 제거. `NewService()`는 production용 별도 파일(Task 3)에서 만든다.

- [ ] **Step 3: fakeAlphaDeps를 Deps 구현체로 완성**

`apps/api/internal/portfolio/alpha_test.go`의 `fakeAlphaDeps`를 다음으로 교체:

```go
type fakeAlphaDeps struct {
	createdAt    time.Time
	holdings     []portfolio.HoldingRow
	prices       map[string]map[string]float64
	fxRates      map[string]map[string]float64
	tradingDays  []string
	benchmarks   map[string][]portfolio.PricePoint // symbol → series
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
```

import에 `"github.com/quotient/quotient/apps/api/internal/db"` 추가.

`TestCompute_BasicTwoHoldings` 본문 끝의 `sp500SeriesL` 오타를 `benchmarks` 맵 형태로 수정:

```go
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
```

- [ ] **Step 4: 빌드 + 테스트 실행 — basic 케이스 PASS**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./internal/portfolio/... && go test ./internal/portfolio/ -run TestCompute_BasicTwoHoldings -v
```

Expected: PASS.

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/api/internal/portfolio/ && git commit -m "feat(portfolio): AlphaService.Compute 코어 — Deps 인터페이스 + 시계열 합산 + 60/40 합성"
```

---

## Task 3: AlphaService 추가 케이스 + Pg Deps 구현

**Files:**
- Modify: `apps/api/internal/portfolio/alpha_test.go` — 5 추가 케이스
- Create: `apps/api/internal/portfolio/pg_deps.go` — production Deps 구현

- [ ] **Step 1: 5 추가 케이스 작성**

`alpha_test.go` 끝에:

```go
func TestCompute_AccountTooYoung(t *testing.T) {
	deps := &fakeAlphaDeps{
		createdAt: mustTime("2026-05-25T00:00:00Z"), // today - 3일
	}
	svc := portfolio.NewServiceWithDeps(deps, mustTime("2026-05-28T15:00:00+09:00"))
	_, err := svc.Compute(context.Background(), nil, nil, "user-1", portfolio.Period90D)
	var ie *portfolio.InsufficientDataError
	if !errors.As(err, &ie) || ie.Reason != "account_too_young" {
		t.Fatalf("got %v, want account_too_young", err)
	}
	if ie.CurrentDays != 3 || ie.MinDays != 7 {
		t.Errorf("current=%d min=%d", ie.CurrentDays, ie.MinDays)
	}
}

func TestCompute_NoHoldings(t *testing.T) {
	deps := &fakeAlphaDeps{
		createdAt: mustTime("2024-01-01T00:00:00Z"),
		holdings:  nil,
	}
	svc := portfolio.NewServiceWithDeps(deps, mustTime("2026-05-28T15:00:00+09:00"))
	_, err := svc.Compute(context.Background(), nil, nil, "user-1", portfolio.Period90D)
	var ie *portfolio.InsufficientDataError
	if !errors.As(err, &ie) || ie.Reason != "no_holdings" {
		t.Fatalf("got %v, want no_holdings", err)
	}
}

func TestCompute_AccountYoungerThanPeriod(t *testing.T) {
	// 가입 30일, period 90d → days_used=30
	deps := &fakeAlphaDeps{
		createdAt: mustTime("2026-04-28T00:00:00Z"),
		holdings: []portfolio.HoldingRow{
			{InstrumentID: "kr-1", Symbol: "005930", Currency: "KRW", Quantity: 10},
		},
		tradingDays: []string{"2026-04-28", "2026-05-28"},
		prices: map[string]map[string]float64{
			"kr-1": {"2026-04-28": 60000, "2026-05-28": 66000},
		},
		benchmarks: map[string][]portfolio.PricePoint{
			"KOSPI": {{Date: "2026-04-28", Close: 2500}, {Date: "2026-05-28", Close: 2580}},
			"SPX":   {{Date: "2026-04-28", Close: 5500}, {Date: "2026-05-28", Close: 5850}},
		},
	}
	svc := portfolio.NewServiceWithDeps(deps, mustTime("2026-05-28T15:00:00+09:00"))
	res, err := svc.Compute(context.Background(), nil, nil, "user-1", portfolio.Period90D)
	if err != nil {
		t.Fatal(err)
	}
	if res.DaysUsed != 30 {
		t.Errorf("days_used=%d, want 30", res.DaysUsed)
	}
	if res.DaysRequested != 90 {
		t.Errorf("days_requested=%d, want 90", res.DaysRequested)
	}
}

func TestCompute_NewListingGap(t *testing.T) {
	// us-1은 since 이후 첫 가격 → data_gaps에 등록
	deps := &fakeAlphaDeps{
		createdAt: mustTime("2024-01-01T00:00:00Z"),
		holdings: []portfolio.HoldingRow{
			{InstrumentID: "kr-1", Symbol: "005930", Currency: "KRW", Quantity: 10},
			{InstrumentID: "us-1", Symbol: "NEWCO", Currency: "USD", Quantity: 5},
		},
		tradingDays: []string{"2026-02-27", "2026-03-15", "2026-05-28"},
		prices: map[string]map[string]float64{
			"kr-1": {"2026-02-27": 60000, "2026-03-15": 62000, "2026-05-28": 66000},
			"us-1": {"2026-03-15": 100, "2026-05-28": 120}, // 2026-02-27 누락
		},
		fxRates: map[string]map[string]float64{
			"USD": {"2026-02-27": 1300, "2026-03-15": 1320, "2026-05-28": 1378},
		},
		benchmarks: map[string][]portfolio.PricePoint{
			"KOSPI": {{Date: "2026-02-27", Close: 2500}, {Date: "2026-03-15", Close: 2520}, {Date: "2026-05-28", Close: 2580}},
			"SPX":   {{Date: "2026-02-27", Close: 5500}, {Date: "2026-03-15", Close: 5700}, {Date: "2026-05-28", Close: 5850}},
		},
	}
	svc := portfolio.NewServiceWithDeps(deps, mustTime("2026-05-28T15:00:00+09:00"))
	res, err := svc.Compute(context.Background(), nil, nil, "user-1", portfolio.Period90D)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Portfolio.DataGaps) != 1 || res.Portfolio.DataGaps[0].Symbol != "NEWCO" {
		t.Errorf("data_gaps=%+v, want [NEWCO]", res.Portfolio.DataGaps)
	}
	if res.Portfolio.DataGaps[0].FirstPriceDate != "2026-03-15" {
		t.Errorf("first_price_date=%s, want 2026-03-15", res.Portfolio.DataGaps[0].FirstPriceDate)
	}
}

func TestCompute_FxForwardFill(t *testing.T) {
	// USD fx의 중간 일자 누락 → 직전 값 사용
	deps := &fakeAlphaDeps{
		createdAt: mustTime("2024-01-01T00:00:00Z"),
		holdings: []portfolio.HoldingRow{
			{InstrumentID: "us-1", Symbol: "AAPL", Currency: "USD", Quantity: 5},
		},
		tradingDays: []string{"2026-02-27", "2026-02-28", "2026-05-28"},
		prices: map[string]map[string]float64{
			"us-1": {"2026-02-27": 100, "2026-02-28": 101, "2026-05-28": 120},
		},
		fxRates: map[string]map[string]float64{
			"USD": {"2026-02-27": 1300, "2026-05-28": 1378}, // 2026-02-28 누락
		},
		benchmarks: map[string][]portfolio.PricePoint{
			"KOSPI": {{Date: "2026-02-27", Close: 2500}, {Date: "2026-02-28", Close: 2510}, {Date: "2026-05-28", Close: 2580}},
			"SPX":   {{Date: "2026-02-27", Close: 5500}, {Date: "2026-02-28", Close: 5505}, {Date: "2026-05-28", Close: 5850}},
		},
	}
	svc := portfolio.NewServiceWithDeps(deps, mustTime("2026-05-28T15:00:00+09:00"))
	res, err := svc.Compute(context.Background(), nil, nil, "user-1", portfolio.Period90D)
	if err != nil {
		t.Fatal(err)
	}
	// 2026-02-28 값: 5 * 101 * 1300(forward fill from 02-27) = 656,500
	want0228 := 656500.0
	want0227 := 5.0 * 100 * 1300 // 650,000
	wantPct := (want0228 - want0227) / want0227 * 100
	if abs(res.Portfolio.Series[1].ValuePct-wantPct) > 0.01 {
		t.Errorf("series[1]=%.4f, want %.4f", res.Portfolio.Series[1].ValuePct, wantPct)
	}
}

func TestCompute_6040Calculation(t *testing.T) {
	// KOSPI +10%, SPX +20% → 60/40 = 0.6*10 + 0.4*20 = 14%
	deps := &fakeAlphaDeps{
		createdAt: mustTime("2024-01-01T00:00:00Z"),
		holdings: []portfolio.HoldingRow{
			{InstrumentID: "kr-1", Symbol: "005930", Currency: "KRW", Quantity: 1},
		},
		tradingDays: []string{"2026-02-27", "2026-05-28"},
		prices: map[string]map[string]float64{
			"kr-1": {"2026-02-27": 100, "2026-05-28": 100}, // 포트 +0%
		},
		benchmarks: map[string][]portfolio.PricePoint{
			"KOSPI": {{Date: "2026-02-27", Close: 1000}, {Date: "2026-05-28", Close: 1100}},
			"SPX":   {{Date: "2026-02-27", Close: 1000}, {Date: "2026-05-28", Close: 1200}},
		},
	}
	svc := portfolio.NewServiceWithDeps(deps, mustTime("2026-05-28T15:00:00+09:00"))
	res, err := svc.Compute(context.Background(), nil, nil, "user-1", portfolio.Period90D)
	if err != nil {
		t.Fatal(err)
	}
	// 60/40 total = 14
	if abs(res.Benchmarks[2].TotalReturnPct-14.0) > 0.01 {
		t.Errorf("kr_us_6040 total = %.4f, want 14", res.Benchmarks[2].TotalReturnPct)
	}
	// alpha vs 60/40 = 0 - 14 = -14
	if abs(res.Benchmarks[2].AlphaPP-(-14)) > 0.01 {
		t.Errorf("kr_us_6040 alpha = %.4f, want -14", res.Benchmarks[2].AlphaPP)
	}
}
```

- [ ] **Step 2: production PgDeps 작성**

`apps/api/internal/portfolio/pg_deps.go`:

```go
package portfolio

import (
	"context"
	"strings"
	"time"

	"github.com/quotient/quotient/apps/api/internal/db"
)

// Service는 알파 계산 진입점. 호출자가 exec(JWT 트랜잭션)와 pool(공개 read)을 분리해 주입.
type Service struct {
	deps Deps
	now  func() time.Time
}

func NewService() *Service {
	return &Service{deps: &PgDeps{}, now: time.Now}
}

// PgDeps는 Service의 production 구현. Pure SQL.
// 모든 일자 비교는 `::date` 직접 비교 — 인덱스 활용 + timezone 안전.
type PgDeps struct{}

func (PgDeps) UserCreatedAt(ctx context.Context, exec db.Executor, uid string) (time.Time, error) {
	var t time.Time
	err := exec.QueryRow(ctx, `select created_at from public.profiles where id = $1`, uid).Scan(&t)
	return t, err
}

func (PgDeps) UserHoldings(ctx context.Context, exec db.Executor, uid string) ([]HoldingRow, error) {
	rows, err := exec.Query(ctx, `
		select h.instrument_id::text, i.symbol, i.currency, h.quantity::float8
		from public.holdings h
		join public.instruments i on i.id = h.instrument_id
		where h.user_id = $1
	`, uid)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []HoldingRow
	for rows.Next() {
		var r HoldingRow
		if err := rows.Scan(&r.InstrumentID, &r.Symbol, &r.Currency, &r.Quantity); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// TradingDays는 KOSPI ∪ SPX prices distinct date 합집합.
// KR·US 영업일 캘린더 차이를 모두 포괄 → 종목 가격·fx는 forward-fill로 채움.
func (PgDeps) TradingDays(ctx context.Context, pool db.Executor, since, until time.Time) ([]string, error) {
	rows, err := pool.Query(ctx, `
		select distinct to_char(p.date, 'YYYY-MM-DD')
		from public.prices p
		join public.instruments i on i.id = p.instrument_id
		where i.symbol in ('KOSPI', 'SPX')
		  and p.date >= $1::date and p.date <= $2::date
		order by 1
	`, since, until)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// InstrumentClosesOnDates — p.date를 직접 date[]와 비교 (인덱스 활용).
// dates는 "YYYY-MM-DD" 문자열이지만 `$2::date[]` 캐스트로 PG가 자동 변환.
func (PgDeps) InstrumentClosesOnDates(ctx context.Context, pool db.Executor, iid string, dates []string) (map[string]float64, error) {
	if len(dates) == 0 {
		return map[string]float64{}, nil
	}
	rows, err := pool.Query(ctx, `
		select to_char(p.date, 'YYYY-MM-DD'), p.close::float8
		from public.prices p
		where p.instrument_id = $1::uuid and p.date = any($2::date[])
	`, iid, dates)
	if err != nil { return nil, err }
	defer rows.Close()
	out := map[string]float64{}
	for rows.Next() {
		var d string
		var v float64
		if err := rows.Scan(&d, &v); err != nil {
			return nil, err
		}
		out[d] = v
	}
	return out, rows.Err()
}

// FxRatesOnDates — observed_at::date 비교 (timezone 안전).
// 첫 일자 fx 누락 가드: range 시작 7일 전부터 가장 최근 1개 추가 조회 → 결과 맵에 "__before" 키로 저장.
// alpha.go의 lookupFxForward가 idx=0에서도 fx 못 찾으면 "__before" 사용.
func (PgDeps) FxRatesOnDates(ctx context.Context, pool db.Executor, currency string, dates []string) (map[string]float64, error) {
	if len(dates) == 0 || currency == "KRW" {
		return map[string]float64{}, nil
	}
	currency = strings.ToUpper(currency)
	rows, err := pool.Query(ctx, `
		select to_char(observed_at::date, 'YYYY-MM-DD'), rate::float8
		from public.fx_rates
		where base = $1 and quote = 'KRW' and observed_at::date = any($2::date[])
	`, currency, dates)
	if err != nil { return nil, err }
	defer rows.Close()
	out := map[string]float64{}
	for rows.Next() {
		var d string
		var v float64
		if err := rows.Scan(&d, &v); err != nil {
			return nil, err
		}
		out[d] = v
	}
	if err := rows.Err(); err != nil { return nil, err }

	// before fallback — range 첫 일자 직전 가장 최근 fx 1점
	firstDate := dates[0]
	var beforeRate float64
	err = pool.QueryRow(ctx, `
		select rate::float8
		from public.fx_rates
		where base = $1 and quote = 'KRW' and observed_at::date < $2::date
		order by observed_at desc limit 1
	`, currency, firstDate).Scan(&beforeRate)
	if err == nil && beforeRate > 0 {
		out["__before"] = beforeRate
	}
	// err가 sql.ErrNoRows인 경우 (5년 백필 시 fx_rates는 frankfurter 일별, 충분 — 단, 신규 currency는 없을 수 있음)
	return out, nil
}

func (PgDeps) BenchmarkSeries(ctx context.Context, pool db.Executor, symbol string, dates []string) ([]PricePoint, error) {
	if len(dates) == 0 {
		return nil, nil
	}
	rows, err := pool.Query(ctx, `
		select to_char(p.date, 'YYYY-MM-DD'), p.close::float8
		from public.prices p
		join public.instruments i on i.id = p.instrument_id
		where i.symbol = $1 and p.date = any($2::date[])
		order by p.date
	`, symbol, dates)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []PricePoint
	for rows.Next() {
		var p PricePoint
		if err := rows.Scan(&p.Date, &p.Close); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
```

- [ ] **Step 3: 빌드 + 전체 unit 테스트**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./... && go test ./internal/portfolio/ -v
```

Expected: 6 케이스 모두 PASS (TestParsePeriod·TestPeriodDays·TestCompute_BasicTwoHoldings + 신규 5건).

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/api/internal/portfolio/ && git commit -m "feat(portfolio): PgDeps 구현 + 5 unit 케이스 (young account·no holdings·new listing·fx fill·60/40)"
```

---

## Task 4: HTTP 핸들러 + 라우트 + main.go

**Files:**
- Create: `apps/api/internal/handlers/portfolio_alpha.go`
- Create: `apps/api/internal/handlers/portfolio_alpha_test.go`
- Modify: `apps/api/internal/router/router.go`
- Modify: `apps/api/cmd/server/main.go`

- [ ] **Step 1: 핸들러 작성**

`apps/api/internal/handlers/portfolio_alpha.go`:

```go
package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

// AlphaComputer는 portfolio.Service의 인터페이스 (테스트 fake용).
type AlphaComputer interface {
	Compute(ctx context.Context, exec db.Executor, pool db.Executor, uid string, period portfolio.Period) (*portfolio.AlphaResult, error)
}

type AlphaHandler struct {
	svc  AlphaComputer
	pool *pgxpool.Pool
	run  txRunner
}

func NewAlphaHandler(svc AlphaComputer, pool *pgxpool.Pool) *AlphaHandler {
	h := &AlphaHandler{svc: svc, pool: pool}
	if pool == nil {
		h.run = func(ctx context.Context, fn func(db.Executor) error) error { return fn(nil) }
		return h
	}
	h.run = func(ctx context.Context, fn func(db.Executor) error) error {
		uid := middleware.UserID(ctx)
		return db.AsUser(ctx, pool, uid, fn)
	}
	return h
}

func (h *AlphaHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	period, err := portfolio.ParsePeriod(r.URL.Query().Get("period"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "period must be 1m|90d|1y|all")
		return
	}

	var result *portfolio.AlphaResult
	err = h.run(r.Context(), func(exec db.Executor) error {
		res, e := h.svc.Compute(r.Context(), exec, h.pool, uid, period)
		if e != nil {
			return e
		}
		result = res
		return nil
	})
	if err != nil {
		var ie *portfolio.InsufficientDataError
		if errors.As(err, &ie) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error": map[string]any{
					"code":         "INSUFFICIENT_DATA",
					"reason":       ie.Reason,
					"message":      insufficientMessage(ie),
					"min_days":     ie.MinDays,
					"current_days": ie.CurrentDays,
				},
			})
			return
		}
		slog.Error("alpha compute failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "compute failed")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func insufficientMessage(e *portfolio.InsufficientDataError) string {
	switch e.Reason {
	case "account_too_young":
		return "7일 이상 보유 후 표시됩니다"
	case "no_holdings":
		return "보유 자산 추가 후 표시됩니다"
	}
	return "데이터 부족"
}
```

> `no_holdings` 케이스에서는 `min_days`·`current_days` 둘 다 0이 응답된다. 약간 어색하지만 frontend의 union 타입(`AlphaInsufficient`)이 `reason`으로 분기하므로 무해. 필요 시 v2에서 reason별 응답 shape 분리.

- [ ] **Step 2: 핸들러 unit 테스트**

`apps/api/internal/handlers/portfolio_alpha_test.go`:

```go
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

type fakeAlpha struct {
	res *portfolio.AlphaResult
	err error
}

func (f *fakeAlpha) Compute(_ context.Context, _ db.Executor, _ db.Executor, _ string, _ portfolio.Period) (*portfolio.AlphaResult, error) {
	return f.res, f.err
}

func reqAlpha(period, uid string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/v1/portfolio/alpha?period="+period, nil)
	if uid != "" {
		r = r.WithContext(middleware.WithUserID(r.Context(), uid))
	}
	return r
}

func TestAlphaHandler_OK(t *testing.T) {
	svc := &fakeAlpha{res: &portfolio.AlphaResult{Period: portfolio.Period90D, DaysUsed: 90}}
	h := NewAlphaHandler(svc, nil)
	w := httptest.NewRecorder()
	h.Get(w, reqAlpha("90d", "user-1"))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["period"] != "90d" {
		t.Errorf("period=%v", got["period"])
	}
}

func TestAlphaHandler_BadPeriod(t *testing.T) {
	h := NewAlphaHandler(&fakeAlpha{}, nil)
	w := httptest.NewRecorder()
	h.Get(w, reqAlpha("FOO", "user-1"))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status=%d, want 400", w.Code)
	}
}

func TestAlphaHandler_NoAuth(t *testing.T) {
	h := NewAlphaHandler(&fakeAlpha{}, nil)
	w := httptest.NewRecorder()
	h.Get(w, reqAlpha("90d", ""))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status=%d, want 401", w.Code)
	}
}

func TestAlphaHandler_AccountTooYoung(t *testing.T) {
	svc := &fakeAlpha{err: &portfolio.InsufficientDataError{Reason: "account_too_young", MinDays: 7, CurrentDays: 3}}
	h := NewAlphaHandler(svc, nil)
	w := httptest.NewRecorder()
	h.Get(w, reqAlpha("90d", "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d", w.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	errBlock := got["error"].(map[string]any)
	if errBlock["reason"] != "account_too_young" || errBlock["current_days"].(float64) != 3 {
		t.Errorf("error=%+v", errBlock)
	}
}

func TestAlphaHandler_NoHoldings(t *testing.T) {
	svc := &fakeAlpha{err: &portfolio.InsufficientDataError{Reason: "no_holdings"}}
	h := NewAlphaHandler(svc, nil)
	w := httptest.NewRecorder()
	h.Get(w, reqAlpha("1y", "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status=%d", w.Code)
	}
}
```

- [ ] **Step 3: 라우트 등록**

`apps/api/internal/router/router.go`의 `func New(...)` 시그니처 끝부분에 `alphaHandler *handlers.AlphaHandler` 추가 + `r.Get("/v1/profile", ...)` 줄 다음 묶음에 다음 줄 추가:

```go
r.Get("/v1/portfolio/alpha", alphaHandler.Get)
```

- [ ] **Step 4: main.go 와이어링**

`apps/api/cmd/server/main.go`의 import에 portfolio 추가:

```go
"github.com/quotient/quotient/apps/api/internal/portfolio"
```

handler 생성 블록(profileHandler 근처)에:

```go
alphaSvc := portfolio.NewService()
alphaHandler := handlers.NewAlphaHandler(alphaSvc, pool)
```

`router.New(...)` 호출 인자 마지막에 `alphaHandler` 추가.

- [ ] **Step 5: 빌드 + 테스트**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./... && go test ./internal/handlers -run Alpha -v
```

Expected: 5 handler 케이스 PASS.

- [ ] **Step 6: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/api/internal/handlers/portfolio_alpha.go apps/api/internal/handlers/portfolio_alpha_test.go apps/api/internal/router/router.go apps/api/cmd/server/main.go && git commit -m "feat(api): GET /v1/portfolio/alpha 핸들러 + 라우트 + 와이어링"
```

---

## Task 5: 통합 테스트 (실 Postgres)

**Files:**
- Create: `apps/api/internal/portfolio/alpha_integration_test.go`

- [ ] **Step 1: integration 테스트 작성**

`apps/api/internal/portfolio/alpha_integration_test.go`:

```go
//go:build integration

package portfolio_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

func openPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestPgDeps_E2E(t *testing.T) {
	pool := openPool(t)
	uid := uuid.NewString()
	ctx := context.Background()

	// seed user (90일 전 가입)
	pastCreatedAt := time.Now().Add(-200 * 24 * time.Hour)
	_, err := pool.Exec(ctx, `
		insert into auth.users (id, email, encrypted_password, created_at)
		values ($1::uuid, $1::text || '@alpha.test', '', $2)
		on conflict (id) do nothing
	`, uid, pastCreatedAt)
	if err != nil {
		t.Fatalf("seed auth.users: %v", err)
	}
	defer pool.Exec(ctx, `delete from auth.users where id = $1`, uid)

	// 사용자 created_at도 200일 전으로 강제
	_, _ = pool.Exec(ctx, `update public.profiles set created_at = $1 where id = $2`, pastCreatedAt, uid)

	// holdings — KOSPI에 있는 instrument 1개 사용
	var instID string
	err = pool.QueryRow(ctx, `select id::text from public.instruments where symbol = '005930' limit 1`).Scan(&instID)
	if err != nil {
		t.Skip("instrument 005930 not seeded")
	}
	_, err = pool.Exec(ctx, `
		insert into public.holdings (user_id, instrument_id, quantity, avg_cost)
		values ($1, $2::uuid, 10, 60000)
	`, uid, instID)
	if err != nil {
		t.Fatalf("seed holding: %v", err)
	}

	svc := portfolio.NewService()
	res, err := svc.Compute(ctx, pool, pool, uid, portfolio.Period90D)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Period != portfolio.Period90D {
		t.Errorf("period=%s", res.Period)
	}
	if len(res.Portfolio.Series) < 2 {
		t.Errorf("series too short: %d", len(res.Portfolio.Series))
	}
	if len(res.Benchmarks) != 3 {
		t.Errorf("benchmarks count=%d, want 3", len(res.Benchmarks))
	}
	// 알파 계산이 NaN이나 Inf로 끝나지 않는지
	for _, b := range res.Benchmarks {
		if b.AlphaPP != b.AlphaPP { // NaN check
			t.Errorf("benchmark %s alpha NaN", b.Key)
		}
	}
}

// compile-time: ensure exec types match
var _ db.Executor = (*pgxpool.Pool)(nil)
```

- [ ] **Step 2: 실행**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && TEST_DATABASE_URL="postgresql://postgres:postgres@127.0.0.1:54322/postgres" go test -tags integration ./internal/portfolio/ -run TestPgDeps_E2E -v
```

Expected: PASS (또는 instrument 005930 미시드 시 SKIP).

- [ ] **Step 3: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/api/internal/portfolio/alpha_integration_test.go && git commit -m "test(portfolio): alpha integration test — 실 Supabase + KOSPI 보유 시나리오"
```

---

## Task 6: 프론트엔드 API 클라이언트

**Files:**
- Create: `apps/web/lib/api/portfolio.ts`

- [ ] **Step 1: API 클라이언트 함수 작성**

`apps/web/lib/api/portfolio.ts`:

```ts
import { authFetch } from "./auth-fetch";

export type AlphaPeriod = "1m" | "90d" | "1y" | "all";

export type AlphaSeriesPoint = { date: string; value_pct: number };

export type AlphaBenchmark = {
  key: "kospi" | "sp500" | "kr_us_6040";
  label: string;
  total_return_pct: number;
  alpha_pp: number;
  series?: AlphaSeriesPoint[];
};

export type AlphaResult = {
  period: AlphaPeriod;
  days_requested: number;
  days_used: number;
  since: string;
  fx_mode: "spot";
  model: "current_holdings_backward_simulation";
  portfolio: {
    total_return_pct: number;
    series: AlphaSeriesPoint[];
    data_gaps?: { symbol: string; first_price_date: string }[];
  };
  benchmarks: AlphaBenchmark[];
};

export type AlphaInsufficient = {
  error: {
    code: "INSUFFICIENT_DATA";
    reason: "account_too_young" | "no_holdings";
    message: string;
    min_days: number;
    current_days: number;
  };
};

// HTTP 422 응답을 throw로 처리하면 사용 측이 try/catch + status 분기 부담.
// 정상 결과 또는 부족 상태를 union으로 반환.
export async function getAlpha(period: AlphaPeriod): Promise<AlphaResult | AlphaInsufficient> {
  const res = await authFetch(`/v1/portfolio/alpha?period=${period}`);
  if (res.status === 422) {
    return (await res.json()) as AlphaInsufficient;
  }
  if (!res.ok) {
    throw new Error(`alpha fetch failed: ${res.status}`);
  }
  return (await res.json()) as AlphaResult;
}

export function isInsufficient(r: AlphaResult | AlphaInsufficient): r is AlphaInsufficient {
  return (r as AlphaInsufficient).error !== undefined;
}
```

- [ ] **Step 2: 타입 검증**

```bash
cd /Users/yuhojin/Desktop/finance/apps/web && npx tsc --noEmit
```

Expected: EXIT 0.

- [ ] **Step 3: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/web/lib/api/portfolio.ts && git commit -m "feat(web): getAlpha API 클라이언트 + 타입"
```

---

## Task 7: AlphaCard 컴포넌트

**Files:**
- Create: `apps/web/components/home/AlphaCard.tsx`
- Modify: `apps/web/components/home/HomeDashboard.tsx`

- [ ] **Step 1: AlphaCard 작성**

`apps/web/components/home/AlphaCard.tsx`:

```tsx
"use client";

import { useEffect, useState } from "react";
import { getAlpha, isInsufficient, type AlphaPeriod, type AlphaResult, type AlphaInsufficient } from "@/lib/api/portfolio";

const PERIODS: AlphaPeriod[] = ["1m", "90d", "1y", "all"];
const LABELS: Record<AlphaPeriod, string> = { "1m": "1M", "90d": "90D", "1y": "1Y", "all": "All" };

export function AlphaCard() {
  const [period, setPeriod] = useState<AlphaPeriod>("90d");
  const [data, setData] = useState<AlphaResult | AlphaInsufficient | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setLoading(true);
    getAlpha(period)
      .then((r) => setData(r))
      .catch(() => setData(null))
      .finally(() => setLoading(false));
  }, [period]);

  return (
    <div className="border border-line bg-bg-subtle p-5">
      <Header period={period} onChange={setPeriod} />
      <Body data={data} loading={loading} />
    </div>
  );
}

function Header({ period, onChange }: { period: AlphaPeriod; onChange: (p: AlphaPeriod) => void }) {
  return (
    <div className="flex items-center justify-between mb-3">
      <div className="font-mono text-[10px] text-fg-muted tracking-widest">ALPHA</div>
      <div className="flex gap-1">
        {PERIODS.map((p) => (
          <button
            key={p}
            onClick={() => onChange(p)}
            className={`font-mono text-[10px] px-2 py-0.5 border ${
              period === p ? "border-bb-accent text-bb-accent" : "border-line text-fg-muted hover:text-fg"
            }`}
          >
            {LABELS[p]}
          </button>
        ))}
      </div>
    </div>
  );
}

function Body({ data, loading }: { data: AlphaResult | AlphaInsufficient | null; loading: boolean }) {
  if (data === null && !loading) {
    return <div className="font-mono text-xs text-fg-muted">로드 실패</div>;
  }
  if (data === null) {
    return <div className="font-mono text-xs text-fg-muted">로딩…</div>;
  }
  if (isInsufficient(data)) {
    return <Empty err={data.error} />;
  }
  return <Filled data={data} dim={loading} />;
}

function Empty({ err }: { err: AlphaInsufficient["error"] }) {
  return (
    <div className="space-y-2 py-2">
      <p className="text-sm">{err.message}</p>
      {err.reason === "account_too_young" && (
        <p className="font-mono text-[10px] text-fg-muted">
          가입 {err.current_days}일째 — {err.min_days - err.current_days}일 후부터 비교 가능
        </p>
      )}
      {err.reason === "no_holdings" && (
        <a href="/app/portfolio" className="font-mono text-[10px] text-bb-accent hover:text-bb-warn">
          포트폴리오로 이동 →
        </a>
      )}
    </div>
  );
}

function Filled({ data, dim }: { data: AlphaResult; dim: boolean }) {
  const fmt = (v: number) => `${v >= 0 ? "+" : ""}${v.toFixed(2)}%p`;
  const sign = (v: number) => (v >= 0 ? "text-bb-up" : "text-bb-down");
  return (
    <div className={`space-y-2 transition-opacity ${dim ? "opacity-60" : "opacity-100"}`}>
      {data.benchmarks.map((b) => (
        <div key={b.key} className="flex justify-between font-mono text-xs">
          <span className="text-fg-muted">vs {b.label}</span>
          <span className={sign(b.alpha_pp)}>{fmt(b.alpha_pp)}</span>
        </div>
      ))}
      <Chart data={data} />
      <div className="font-mono text-[10px] text-fg-muted pt-1">
        {data.days_used < data.days_requested && data.days_requested > 0 && (
          <>가입 {data.days_used}일 · </>
        )}
        환율 변동 포함 · 현재 보유 기준 시뮬레이션
        {data.portfolio.data_gaps && data.portfolio.data_gaps.length > 0 && (
          <> · {data.portfolio.data_gaps.length}개 종목 데이터 부족</>
        )}
      </div>
    </div>
  );
}

function Chart({ data }: { data: AlphaResult }) {
  const lines = [
    { series: data.portfolio.series, color: "#FFD500" },
    { series: data.benchmarks[0].series ?? [], color: "#00FFFF" },
    { series: data.benchmarks[1].series ?? [], color: "#FF9900" },
  ];
  const all = lines.flatMap((l) => l.series.map((p) => p.value_pct));
  if (all.length === 0) return null;
  const min = Math.min(...all, 0);
  const max = Math.max(...all, 0);
  const range = max - min || 1;
  const width = 240;
  const height = 36;
  return (
    <svg viewBox={`0 0 ${width} ${height}`} className="w-full h-9 mt-2">
      {lines.map((l, idx) => (
        <polyline
          key={idx}
          fill="none"
          stroke={l.color}
          strokeWidth="1.2"
          points={l.series
            .map((p, i) => {
              const x = (i / Math.max(l.series.length - 1, 1)) * width;
              const y = height - ((p.value_pct - min) / range) * height;
              return `${x.toFixed(1)},${y.toFixed(1)}`;
            })
            .join(" ")}
        />
      ))}
    </svg>
  );
}
```

- [ ] **Step 2: HomeDashboard에 추가 + 그리드 재배치**

`apps/web/components/home/HomeDashboard.tsx`:

```tsx
"use client";

import { useEffect, useState } from "react";
import { listHoldings, type Holding } from "@/lib/api/holdings";
import { TotalAssetCard } from "./TotalAssetCard";
import { AllocationDonut } from "./AllocationDonut";
import { TopHoldingsCard } from "./TopHoldingsCard";
import { MarketWidgetsCard } from "./MarketWidgetsCard";
import { WatchlistMiniCard } from "./WatchlistMiniCard";
import { BriefingCard } from "./BriefingCard";
import { AlphaCard } from "./AlphaCard";
import { Skeleton } from "@/components/ui/skeleton";

export function HomeDashboard() {
  const [holdings, setHoldings] = useState<Holding[] | null>(null);

  useEffect(() => {
    listHoldings().then(setHoldings).catch(() => setHoldings([]));
  }, []);

  if (holdings === null) {
    return (
      <div className="p-6 md:p-8 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {Array.from({ length: 7 }).map((_, i) => (
          <Skeleton key={i} className="h-32" />
        ))}
      </div>
    );
  }

  return (
    <div className="p-6 md:p-8 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {/* 1행: 총자산 · 도넛 · 알파 */}
      <TotalAssetCard holdings={holdings} />
      <AllocationDonut holdings={holdings} />
      <AlphaCard />
      {/* 2행: 상위5 · 마켓 · 관심종목 */}
      <TopHoldingsCard holdings={holdings} />
      <MarketWidgetsCard />
      <WatchlistMiniCard />
      {/* 3행: 브리핑 (가로 와이드) */}
      <div className="lg:col-span-3">
        <BriefingCard />
      </div>
    </div>
  );
}
```

- [ ] **Step 3: 타입 검증**

```bash
cd /Users/yuhojin/Desktop/finance/apps/web && npx tsc --noEmit
```

Expected: EXIT 0.

- [ ] **Step 4: AlphaCard vitest 작성 (spec §9 4 케이스)**

`apps/web/components/home/AlphaCard.test.tsx`:

```tsx
import { render, screen, act, waitFor } from "@testing-library/react";
import { fireEvent } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { AlphaCard } from "./AlphaCard";
import * as api from "@/lib/api/portfolio";

vi.mock("@/lib/api/portfolio", async () => {
  const actual = await vi.importActual<typeof api>("@/lib/api/portfolio");
  return { ...actual, getAlpha: vi.fn() };
});

const mockedGet = vi.mocked(api.getAlpha);

describe("AlphaCard", () => {
  beforeEach(() => mockedGet.mockReset());

  it("renders 3 benchmark rows on success", async () => {
    mockedGet.mockResolvedValueOnce({
      period: "90d", days_requested: 90, days_used: 90, since: "2026-02-27",
      fx_mode: "spot", model: "current_holdings_backward_simulation",
      portfolio: { total_return_pct: 18.94, series: [
        { date: "2026-02-27", value_pct: 0 }, { date: "2026-05-28", value_pct: 18.94 },
      ] },
      benchmarks: [
        { key: "kospi", label: "KOSPI", total_return_pct: 3.2, alpha_pp: 15.74,
          series: [{ date: "2026-02-27", value_pct: 0 }, { date: "2026-05-28", value_pct: 3.2 }] },
        { key: "sp500", label: "S&P 500", total_return_pct: 6.36, alpha_pp: 12.58,
          series: [{ date: "2026-02-27", value_pct: 0 }, { date: "2026-05-28", value_pct: 6.36 }] },
        { key: "kr_us_6040", label: "한미 60/40", total_return_pct: 4.46, alpha_pp: 14.48,
          series: null as never },
      ],
    });
    render(<AlphaCard />);
    await waitFor(() => expect(screen.getByText(/vs KOSPI/)).toBeInTheDocument());
    expect(screen.getByText(/\+15.74%p/)).toBeInTheDocument();
    expect(screen.getByText(/vs S&P 500/)).toBeInTheDocument();
    expect(screen.getByText(/vs 한미 60\/40/)).toBeInTheDocument();
  });

  it("shows account_too_young empty state", async () => {
    mockedGet.mockResolvedValueOnce({
      error: { code: "INSUFFICIENT_DATA", reason: "account_too_young",
        message: "7일 이상 보유 후 표시됩니다", min_days: 7, current_days: 3 },
    });
    render(<AlphaCard />);
    await waitFor(() => expect(screen.getByText(/7일 이상 보유/)).toBeInTheDocument());
    expect(screen.getByText(/가입 3일째/)).toBeInTheDocument();
  });

  it("shows no_holdings empty state with portfolio link", async () => {
    mockedGet.mockResolvedValueOnce({
      error: { code: "INSUFFICIENT_DATA", reason: "no_holdings",
        message: "보유 자산 추가 후 표시됩니다", min_days: 0, current_days: 0 },
    });
    render(<AlphaCard />);
    await waitFor(() => expect(screen.getByText(/보유 자산 추가/)).toBeInTheDocument());
    expect(screen.getByRole("link", { name: /포트폴리오로 이동/ })).toHaveAttribute("href", "/app/portfolio");
  });

  it("re-fetches when period chip clicked", async () => {
    mockedGet.mockResolvedValue({
      period: "90d", days_requested: 90, days_used: 90, since: "2026-02-27",
      fx_mode: "spot", model: "current_holdings_backward_simulation",
      portfolio: { total_return_pct: 0, series: [] },
      benchmarks: [
        { key: "kospi", label: "KOSPI", total_return_pct: 0, alpha_pp: 0, series: [] },
        { key: "sp500", label: "S&P 500", total_return_pct: 0, alpha_pp: 0, series: [] },
        { key: "kr_us_6040", label: "한미 60/40", total_return_pct: 0, alpha_pp: 0, series: null as never },
      ],
    });
    render(<AlphaCard />);
    await waitFor(() => expect(mockedGet).toHaveBeenCalledWith("90d"));
    fireEvent.click(screen.getByText("1Y"));
    await waitFor(() => expect(mockedGet).toHaveBeenCalledWith("1y"));
  });
});
```

```bash
cd /Users/yuhojin/Desktop/finance/apps/web && npm test -- --run AlphaCard
```

Expected: 4 케이스 PASS.

- [ ] **Step 5: 수동 검증 (로컬 dev server)**

```bash
cd /Users/yuhojin/Desktop/finance/apps/web && npm run dev
```

브라우저로 http://localhost:3000/app 접속 → 1행 3번째에 ALPHA 카드 표시 확인:
- 빈 상태(no_holdings) 또는 정상 값
- 토글 클릭 → 다른 기간 응답
- 차트 3 라인 렌더링

- [ ] **Step 6: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/web/components/home/AlphaCard.tsx apps/web/components/home/AlphaCard.test.tsx apps/web/components/home/HomeDashboard.tsx && git commit -m "feat(web): AlphaCard 컴포넌트 + vitest 4건 + HomeDashboard 그리드 재배치"
```

---

## Task 8: 문서 갱신 + 최종 정합성

**Files:**
- Modify: `docs/STATUS.md`
- Modify: `docs/ROADMAP.md`

- [ ] **Step 1: STATUS 변경 이력 추가**

`docs/STATUS.md` "최근 변경 이력" 맨 위에:

```markdown
- 2026-05-28 알파 카드 출시 — 홈 1행 3번째에 "포트폴리오 vs KOSPI · S&P 500 · 한미 60/40" 비교. 기간 토글 1M/90D/1Y/All, 시점별 환율, backward simulation. 빈 상태 처리(가입 < 7일 + 보유 자산 0). `internal/portfolio/` 신규 패키지 + `GET /v1/portfolio/alpha` 핸들러 + 5 unit + 1 integration test. 정체성 spec §2 약속 이행.
```

"마지막 업데이트" 날짜를 `2026-05-28`로 갱신.

- [ ] **Step 2: ROADMAP에서 알파 카드 항목 제거 + Phase 2 다음 항목 정리**

`docs/ROADMAP.md` Phase 2 "핵심 차별화" 표에서 알파 카드 행 삭제 (이미 출시됨).
"현재 추천 다음 작업" 절에 "AI 매매 일기" 또는 "Paper Portfolio" 중 다음 우선순위 추가 (spec §3 우선순위 1·2번).

- [ ] **Step 3: 빌드·테스트 최종 통과 확인**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./... && go test ./...
cd /Users/yuhojin/Desktop/finance/apps/web && npx tsc --noEmit
```

Expected: 모두 EXIT 0.

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add docs/STATUS.md docs/ROADMAP.md && git commit -m "docs: 알파 카드 출시 완료 반영 (STATUS·ROADMAP)"
```

---

## Self-Review

### 1. Spec coverage

| Spec 섹션 | Task |
|---|---|
| §2 D1 다중 stacked | Task 7 Filled — 3 benchmark map |
| §2 D2 기간 토글 | Task 7 Header — 4 chip |
| §2 D3 한미 60/40 | Task 3 composite6040 + Task 2 unit |
| §2 D4 시점별 환율 | Task 2 lookupFxForward + Task 3 fx test |
| §2 D5 빈 상태 | Task 2 InsufficientDataError + Task 7 Empty |
| §2 D6 1행 3번째 | Task 7 HomeDashboard 재배치 |
| §2 D7 3 라인 차트 | Task 7 Chart |
| §2 D8 backward simulation | Task 2 model 라벨 + Task 7 라벨 |
| §4 API 명세 | Task 4 handler |
| §5 알고리즘 | Task 2 + Task 3 |
| §6 UI 명세 | Task 7 |
| §8 에러 처리 표 | Task 4 + Task 7 |
| §9 테스트 | Task 2·3 (6 unit) + Task 5 (1 integration) + Task 7 수동 |

### 2. Placeholder scan

- 모든 코드 블록은 실 실행 가능 (TODO/TBD 없음)
- 명령은 정확한 절대 경로 사용
- expected output 명시

### 3. Type consistency

- `portfolio.Period` 타입을 Task 1 정의 → 2·3·4·6에서 일관 사용
- `AlphaResult` JSON 필드명을 spec §4와 정확히 일치
- frontend `AlphaPeriod` 타입과 backend `Period` 값 동일
- `AlphaComputer` 인터페이스(handler) ↔ `Service.Compute` 시그니처 일치
- `db.Executor`가 양쪽 exec·pool 모두 받는 점 명시

---

Plan complete and saved to `docs/superpowers/plans/2026-05-28-alpha-card.md`.

---

## 검토 이력

### 2026-05-28 subagent 자체 검토 (general-purpose, CLAUDE.md MANDATORY)

#### Critical (반드시 수정) — 4건 → 모두 패치 완료

- **C-1. Benchmark.Series `omitempty` 태그가 spec §4 `"series": null` 약속 위반.** → 태그를 `json:"series"`로 변경. nil slice는 기본 직렬화로 `null` 출력.
- **C-2. Task 2 단독 commit 시 빌드 깨짐.** Step 1·2에서 fakeAlphaDeps가 Deps 인터페이스 미완성 + `sp500SeriesL` 오타. → Task 2에 "Step 1·2는 컴파일 안 됨, Step 4에서 첫 빌드 check" 메모 + 오타 수정 (`benchmarks` map 형태).
- **C-3. KOSPI/SPX 5년 백필 부재.** 기존 `cmd/backfill`은 주식만 → 지수 시계열이 거의 비어 있어 알파 카드가 동작 안 함. → **Task 0 신설**: backfill CLI에 indices 분기 + yahoo symbol 매핑 + 운영 시점 실행 안내.
- **C-4. `composite6040` 일자 매칭 결함.** KOSPI 기준 일자에서만 SPX 매칭 → KR·US 영업일 캘린더 차이로 SPX 누락 일자에 60/40 잘림. → TradingDays를 KOSPI ∪ SPX 합집합으로 확장 + `composite6040`을 union dates + forward-fill로 재작성.

#### Important (강력 권장) — 5건 → 모두 패치 완료

- **I-1. `lookupFxForward`가 trading days 첫 일자 fx 없으면 1.0 fallback** → USD를 1KRW로 환산하는 치명적 결함. → `FxRatesOnDates`에 "since-1d 이전 가장 최근 fx 1점" 추가 fetch + `__before` 키로 저장. `lookupFxForward`가 idx=0 실패 시 `__before` 사용.
- **I-2. `to_char(date, 'YYYY-MM-DD') = any($::text[])`가 `prices_date_idx` 인덱스 미사용** → `p.date = any($::date[])`로 변경 (PG가 string → date 자동 캐스트). 인덱스 활용.
- **I-3. `to_char(observed_at::timestamptz)`가 세션 timezone 의존** → `observed_at::date = any(...)`로 통일.
- **I-4. `NewService` 중복 정의** — Task 1과 Task 3에서 두 번 정의 → redeclared 컴파일 오류. → Task 1에서 `Service`·`NewService` 정의 제거, Task 3의 `pg_deps.go`에서만 정의.
- **I-5. AlphaCard.test.tsx 누락 (spec §9 4건 약속)** → Task 7에 vitest Step 추가 (정상·account_too_young·no_holdings·토글 4 케이스).

#### Minor — 5건 중 3건 패치, 2건은 결정 그대로

- **M-1. `no_holdings` 응답의 `min_days/current_days = 0` 노출** → Plan에 "v2에서 reason별 union 분리" 메모만 추가 (현 union 타입으로 무해).
- **M-2. Chart 색 hex hardcode** → 결정 유지. Tailwind v4 svg 안에서 CSS 변수 사용은 prerender 시 fallback 처리 복잡. spec §6 토큰 명시했고 일관성 위해 hex 직접 사용이 더 안정.
- **M-3. 좁은 카드 토글 줄바꿈** → 결정 유지 (lg 기준 ~340px, 안전 마진 확보).
- **M-4. profiles UPDATE의 touch_updated_at 트리거 부작용** → Task 5 integration test에 부작용 명시 코멘트 추가 가치 있으나, 무해라 패치 보류.
- **M-5. router.go 시그니처와 main.go 호출자 인자 순서 일관성** → Task 4 Step 3·4에 "alphaHandler는 readyz 다음 (마지막) 위치, router.New 시그니처도 같은 순서" 한 줄로 명시 보강.

#### 메타

본 plan은 CLAUDE.md MANDATORY 절차(brainstorm → spec → plan → subagent 자체 검토)를 강화 후 첫 사이클로 정합 적용.
패치 후 사용자 review gate 통과 시 `superpowers:subagent-driven-development`로 task별 구현 진행.
