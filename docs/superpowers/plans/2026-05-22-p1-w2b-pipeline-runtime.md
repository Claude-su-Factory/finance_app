# Quotient W2b — Cron 워커 + 마켓 API + 백필 CLI

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development` (권장) 또는 `superpowers:executing-plans`. 체크박스(`- [ ]`) 단위 추적.

**Goal:** W2a에서 만든 어댑터·ingest·마이그레이션을 실제 런타임으로 연결한다. cron 워커(7 잡) + 마켓 API(`/v1/market/ticker`, `/v1/instruments/search`·`/select`) + TopTicker 실데이터 + 5년 백필 CLI. W2b 종료 시점: 로컬에서 `make api` 실행 후 1분 내 `quotes`·`fx_rates`에 데이터가 적재되고, 브라우저에서 TopTicker가 placeholder가 아닌 실데이터를 표시.

**Architecture:** 단일 Go 바이너리에 API 서버 + `robfig/cron/v3` 워커 동거(spec §4). KR `quotes`도 Yahoo `.KS`/`.KQ`로 통합(W2a spike: KRX 직접 호출 불가). 지수 quotes는 Yahoo `^KS11`/`^GSPC` 매핑, FX는 frankfurter.dev `fx_rates`. `quotes.updated_at` 60초 TTL 캐시(§10-2). holdings/watchlist는 W3에서 추가되므로 W2b의 polling 대상은 INDEX·시드 종목만 — 쿼리 구조는 W3 union 확장에 호환.

**Tech Stack:** Go 1.25 + `robfig/cron/v3` + `chi v5` + 기존 W2a 어댑터(KIND·Yahoo·fx·FRED·ECOS) + Next.js 16 + Tailwind v4.

**참고 스펙:** [`2026-05-22-quotient-mvp-design.md`](../specs/2026-05-22-quotient-mvp-design.md) §4·§10-2·§10-3·§10-8·§10-9. [`W2a plan`](2026-05-22-p1-w2a-data-pipeline-adapters.md) (어댑터·ingest·시드).

> **Next.js 16 주의 (Task 12)**: `apps/web/AGENTS.md`에 명시 — 기존 Next.js 지식과 API 차이 있음. TopTicker 작업 전 `node_modules/next/dist/docs/`에서 client component 가이드 확인 필수.

---

## 외부 셋업 (Task 0)

W2a에서 이미 완료. `.env`에 다음 키 존재 여부만 재확인:

```
FRED_API_KEY=<발급 완료>
ECOS_API_KEY=<발급 완료>
DATABASE_URL=postgresql://postgres:postgres@127.0.0.1:54322/postgres
SUPABASE_JWT_SECRET=<Supabase Dashboard에서 Legacy 활성>
```

---

## File Structure (W2b 생성·수정)

```
apps/api/
├── cmd/
│   ├── server/main.go                       (수정) schedule.Start 호출
│   └── backfill/main.go                     (신규) 5년 백필 CLI
├── internal/
│   ├── schedule/
│   │   ├── market_hours.go                  (신규) 장중 판단
│   │   ├── market_hours_test.go
│   │   ├── yahoo_symbols.go                 (신규) 지수→Yahoo 심볼 매핑
│   │   ├── yahoo_symbols_test.go
│   │   ├── cron.go                          (신규) Deps + Start/Stop
│   │   ├── jobs_instruments.go              (신규)
│   │   ├── jobs_prices.go                   (신규) KR + US
│   │   ├── jobs_quotes.go                   (신규) INDEX 폴링 (W3에서 holdings union 확장)
│   │   ├── jobs_fx.go                       (신규)
│   │   ├── jobs_indicators.go               (신규)
│   │   └── jobs_integration_test.go         (신규, build tag integration)
│   ├── handlers/
│   │   ├── market.go                        (신규) /v1/market/ticker
│   │   ├── market_repo_pg.go                (신규)
│   │   ├── market_test.go
│   │   ├── instruments.go                   (신규) search + select
│   │   ├── instruments_repo_pg.go           (신규)
│   │   └── instruments_test.go
│   └── router/router.go                     (수정) 라우트 3개 추가
apps/web/
├── lib/api/market.ts                        (신규)
└── components/shell/TopTicker.tsx           (수정) 실데이터
```

신규 마이그레이션 없음. 시드 데이터 보강도 없음 — W2a 시드(KOSPI·KOSDAQ·SPX·NDX·USD_KRW·EUR_KRW·JPY_KRW 7개)로 시작.

---

## Task 1: market_hours helper (장중 판단)

**Files:**
- Create: `apps/api/internal/schedule/market_hours.go`
- Create: `apps/api/internal/schedule/market_hours_test.go`

분리 이유: cron 잡은 testcontainers 비용이 크지만, 시간 판정은 pure function이라 표 기반 테이블 테스트로 검증 가능.

- [ ] **Step 1: 테스트 작성 (TDD)**

`apps/api/internal/schedule/market_hours_test.go`:
```go
package schedule

import (
	"testing"
	"time"
)

func TestIsKRMarketOpen(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Seoul")
	cases := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"평일 장 시작", time.Date(2025, 12, 2, 9, 0, 0, 0, loc), true},
		{"평일 장 마감 직전", time.Date(2025, 12, 2, 15, 30, 0, 0, loc), true},
		{"평일 장 마감 직후", time.Date(2025, 12, 2, 15, 31, 0, 0, loc), false},
		{"평일 장 시작 전", time.Date(2025, 12, 2, 8, 59, 0, 0, loc), false},
		{"토요일", time.Date(2025, 11, 29, 10, 0, 0, 0, loc), false},
		{"일요일", time.Date(2025, 11, 30, 10, 0, 0, 0, loc), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsKRMarketOpen(c.t); got != c.want {
				t.Errorf("IsKRMarketOpen(%v) = %v, want %v", c.t, got, c.want)
			}
		})
	}
}

func TestIsUSMarketOpen(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Seoul")
	cases := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"평일 KST 23:30 (US 개장)", time.Date(2025, 12, 2, 23, 30, 0, 0, loc), true},
		{"평일 KST 23:29", time.Date(2025, 12, 2, 23, 29, 0, 0, loc), false},
		{"평일 익일 KST 05:00", time.Date(2025, 12, 3, 5, 0, 0, 0, loc), true},
		{"평일 익일 KST 06:00 (마감)", time.Date(2025, 12, 3, 6, 0, 0, 0, loc), true},
		{"평일 익일 KST 06:01", time.Date(2025, 12, 3, 6, 1, 0, 0, loc), false},
		{"토요일 KST 02:00 (NY 금요일 장중 가능하나 보수적 OFF)", time.Date(2025, 11, 29, 2, 0, 0, 0, loc), false},
		{"일요일 KST 02:00", time.Date(2025, 11, 30, 2, 0, 0, 0, loc), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsUSMarketOpen(c.t); got != c.want {
				t.Errorf("IsUSMarketOpen(%v) = %v, want %v", c.t, got, c.want)
			}
		})
	}
}

func TestSeoulLoc(t *testing.T) {
	loc := SeoulLoc()
	if loc.String() != "Asia/Seoul" {
		t.Errorf("SeoulLoc = %s, want Asia/Seoul", loc.String())
	}
}
```

- [ ] **Step 2: 테스트 실패 확인**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api
go test ./internal/schedule/...
```
Expected: `package schedule: no Go files` 또는 `undefined: IsKRMarketOpen`

- [ ] **Step 3: 구현**

`apps/api/internal/schedule/market_hours.go`:
```go
package schedule

import "time"

// SeoulLoc returns the Asia/Seoul location, falling back to UTC if loadlocation fails.
func SeoulLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		return time.UTC
	}
	return loc
}

// IsKRMarketOpen reports whether KRX is in regular trading hours at t (KST).
// 평일 09:00–15:30 KST. 공휴일 미고려 (MVP).
func IsKRMarketOpen(t time.Time) bool {
	t = t.In(SeoulLoc())
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	mins := t.Hour()*60 + t.Minute()
	return mins >= 9*60 && mins <= 15*60+30
}

// IsUSMarketOpen reports whether NYSE/NASDAQ is in regular trading hours at t (KST).
// US 정규장(09:30~16:00 EST) ≈ KST 23:30~익일 06:00. 서머타임 보정 미반영 (MVP).
// 토·일 KST는 보수적으로 false (NY 시간 금요일 장 후반 일부 포함될 수 있으나 무시).
func IsUSMarketOpen(t time.Time) bool {
	t = t.In(SeoulLoc())
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	mins := t.Hour()*60 + t.Minute()
	return mins >= 23*60+30 || mins <= 6*60
}
```

- [ ] **Step 4: 테스트 통과 + 커밋**

```bash
go test ./internal/schedule/...
# PASS: TestIsKRMarketOpen, TestIsUSMarketOpen, TestSeoulLoc
cd /Users/yuhojin/Desktop/finance
git add apps/api/internal/schedule/market_hours.go apps/api/internal/schedule/market_hours_test.go
git commit -m "feat(api): market_hours helper (KR·US 장중 판정)"
```

---

## Task 2: 지수→Yahoo 심볼 매핑 helper

**Files:**
- Create: `apps/api/internal/schedule/yahoo_symbols.go`
- Create: `apps/api/internal/schedule/yahoo_symbols_test.go`

W2a에 `yahoo.SymbolKR`(KR 종목용 `.KS`/`.KQ`)이 있다. 지수·환율은 별도 매핑이 필요하므로 schedule 패키지에 둔다(데이터 흐름과 결합).

- [ ] **Step 1: 테스트**

```go
package schedule

import "testing"

func TestIndexYahooSymbol(t *testing.T) {
	cases := []struct {
		symbol, exchange, want string
	}{
		{"KOSPI", "KRX-IDX", "^KS11"},
		{"KOSDAQ", "KRX-IDX", "^KQ11"},
		{"SPX", "NYSE-IDX", "^GSPC"},
		{"NDX", "NASDAQ-IDX", "^NDX"},
		{"UNKNOWN", "X", ""},
	}
	for _, c := range cases {
		got := IndexYahooSymbol(c.symbol, c.exchange)
		if got != c.want {
			t.Errorf("IndexYahooSymbol(%q, %q) = %q, want %q", c.symbol, c.exchange, got, c.want)
		}
	}
}
```

- [ ] **Step 2: 구현**

`apps/api/internal/schedule/yahoo_symbols.go`:
```go
package schedule

// IndexYahooSymbol maps Quotient의 internal index symbol+exchange를 Yahoo Finance 심볼로 변환한다.
// 미지 매핑은 "" 반환 → 호출자가 skip.
func IndexYahooSymbol(symbol, exchange string) string {
	switch {
	case symbol == "KOSPI" && exchange == "KRX-IDX":
		return "^KS11"
	case symbol == "KOSDAQ" && exchange == "KRX-IDX":
		return "^KQ11"
	case symbol == "SPX" && exchange == "NYSE-IDX":
		return "^GSPC"
	case symbol == "NDX" && exchange == "NASDAQ-IDX":
		return "^NDX"
	default:
		return ""
	}
}
```

- [ ] **Step 3: 테스트 통과 + 커밋**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api
go test ./internal/schedule/...
cd /Users/yuhojin/Desktop/finance
git add apps/api/internal/schedule/yahoo_symbols.go apps/api/internal/schedule/yahoo_symbols_test.go
git commit -m "feat(api): 지수→Yahoo 심볼 매핑 helper"
```

---

## Task 3: cron 스켈레톤 (Deps + Start + Stop)

**Files:**
- Create: `apps/api/internal/schedule/cron.go`

먼저 의존성 추가 + 빈 잡으로 동작 확인. 잡은 Task 4~8에서 채운다.

- [ ] **Step 1: 의존성**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api
go get github.com/robfig/cron/v3
```

- [ ] **Step 2: cron.go 스켈레톤**

`apps/api/internal/schedule/cron.go`:
```go
package schedule

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/sources/ecos"
	"github.com/quotient/quotient/apps/api/internal/sources/fred"
	"github.com/quotient/quotient/apps/api/internal/sources/fx"
	"github.com/quotient/quotient/apps/api/internal/sources/kind"
	"github.com/quotient/quotient/apps/api/internal/sources/yahoo"
	"github.com/robfig/cron/v3"
)

// Deps groups dependencies passed into job functions. 명시적 주입으로 테스트 용이.
type Deps struct {
	Pool  *pgxpool.Pool
	KIND  *kind.Client
	Yahoo *yahoo.Client
	FX    *fx.Client
	FRED  *fred.Client
	ECOS  *ecos.Client
}

// Start registers cron schedules and returns the running cron instance.
// 호출자가 Stop()으로 graceful shutdown 책임.
func Start(ctx context.Context, d Deps) *cron.Cron {
	// SkipIfStillRunning: 같은 잡이 직전 tick에서 진행 중이면 새 실행 skip.
	// instruments 잡(KOSPI+KOSDAQ 풀 fetch)이 1분을 넘기더라도 중복 실행 방지.
	c := cron.New(
		cron.WithLocation(SeoulLoc()),
		cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)),
	)

	// 종목 마스터 — 매일 06:00 KST (spec §4)
	mustAdd(c, "0 6 * * *", "instruments", func() {
		if err := JobUpdateInstruments(ctx, d); err != nil {
			slog.Error("cron instruments failed", "err", err)
		}
	})
	// KR 일봉 — 매일 16:30 KST (장 마감 후)
	mustAdd(c, "30 16 * * *", "kr_prices", func() {
		if err := JobUpdateKRPrices(ctx, d); err != nil {
			slog.Error("cron kr_prices failed", "err", err)
		}
	})
	// US 일봉 — 매일 06:00 KST (US 장 마감 후)
	mustAdd(c, "0 6 * * *", "us_prices", func() {
		if err := JobUpdateUSPrices(ctx, d); err != nil {
			slog.Error("cron us_prices failed", "err", err)
		}
	})
	// quotes (INDEX 폴링) — 매분, 장중만
	mustAdd(c, "* * * * *", "quotes", func() {
		if err := JobUpdateIndexQuotes(ctx, d); err != nil {
			slog.Error("cron quotes failed", "err", err)
		}
	})
	// FX — 5분, 24/7
	mustAdd(c, "*/5 * * * *", "fx", func() {
		if err := JobUpdateFXRates(ctx, d); err != nil {
			slog.Error("cron fx failed", "err", err)
		}
	})
	// 경제 지표 — 매일 07:00 KST
	mustAdd(c, "0 7 * * *", "indicators", func() {
		if err := JobUpdateIndicators(ctx, d); err != nil {
			slog.Error("cron indicators failed", "err", err)
		}
	})

	c.Start()
	slog.Info("cron started", "jobs", 6, "tz", "Asia/Seoul")
	return c
}

func mustAdd(c *cron.Cron, spec, name string, fn func()) {
	if _, err := c.AddFunc(spec, fn); err != nil {
		slog.Error("cron AddFunc failed", "name", name, "spec", spec, "err", err)
	}
}
```

> Task 4~8에서 `JobUpdate*` 함수들을 같은 패키지(`internal/schedule/jobs_*.go`)에 추가한다. 이 시점에는 컴파일이 실패한다 (의도). 다음 Task부터 채운다.

- [ ] **Step 3: 임시 stub로 컴파일 통과 (Task 4~8에서 진짜 구현으로 교체)**

빈 잡 함수 6개를 임시 stub으로 만들어 컴파일이 통과하게 한다. 각 Task에서 차례로 교체.

`apps/api/internal/schedule/jobs_stub.go` (임시 — Task 4에서 첫 잡 구현 시 해당 stub 제거):
```go
package schedule

import "context"

func JobUpdateInstruments(ctx context.Context, d Deps) error { return nil }
func JobUpdateKRPrices(ctx context.Context, d Deps) error    { return nil }
func JobUpdateUSPrices(ctx context.Context, d Deps) error    { return nil }
func JobUpdateIndexQuotes(ctx context.Context, d Deps) error { return nil }
func JobUpdateFXRates(ctx context.Context, d Deps) error     { return nil }
func JobUpdateIndicators(ctx context.Context, d Deps) error  { return nil }
```

- [ ] **Step 4: 빌드 통과 + 커밋**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api
go mod tidy
go build ./...
go test ./internal/schedule/...
cd /Users/yuhojin/Desktop/finance
git add apps/api/
git commit -m "feat(api): schedule 패키지 cron 스켈레톤 (robfig/cron, 6 잡)"
```

---

## Task 4: JobUpdateInstruments (KIND HTML 다운로드 + 시드 alias 자동 등록)

**Files:**
- Create: `apps/api/internal/schedule/jobs_instruments.go`
- Modify: `apps/api/internal/schedule/jobs_stub.go` (해당 stub 삭제)

KOSPI + KOSDAQ 종목 마스터 → KIND fetch → `ingest.UpsertInstruments`. spec §10-9 시드 단계 alias 자동 등록을 같이 수행 — 한글 회사명·종목코드를 alias로 학습하여 "삼성전자"·"005930" 검색이 즉시 매칭되도록.

- [ ] **Step 1: 구현**

`apps/api/internal/schedule/jobs_instruments.go`:
```go
package schedule

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/quotient/quotient/apps/api/internal/ingest"
)

// JobUpdateInstruments refreshes the KR instrument master from KIND.
// 매일 06:00 KST 실행. KOSPI + KOSDAQ 순차 fetch + upsert + alias 시드.
func JobUpdateInstruments(ctx context.Context, d Deps) error {
	for _, market := range []string{"KOSPI", "KOSDAQ"} {
		items, err := d.KIND.FetchInstruments(ctx, market)
		if err != nil {
			return fmt.Errorf("kind %s: %w", market, err)
		}
		n, err := ingest.UpsertInstruments(ctx, d.Pool, items)
		if err != nil {
			return fmt.Errorf("upsert %s: %w", market, err)
		}
		slog.Info("instruments updated", "market", market, "count", n)

	}
	// spec §10-9: 시드 단계 alias 등록 — KR 전체 종목의 회사명·종목코드를 alias로.
	// 두 시장 처리 후 일괄 — UpsertInstruments가 id 반환 안 하므로 DB 재조회.
	seeded, err := seedKRAliases(ctx, d)
	if err != nil {
		slog.Warn("alias seed partial", "err", err)
	}
	slog.Info("aliases seeded", "count", seeded)
	// US 종목 마스터는 lazy: 사용자가 추가 시 Yahoo 메타로 등록 (W3)
	return nil
}

// seedKRAliases는 KRX 종목에 대해 회사명·종목코드를 alias로 시드 등록.
// (alias) PK이라 source='seed'로 ON CONFLICT update — 항상 최신 매핑 유지.
func seedKRAliases(ctx context.Context, d Deps) (int, error) {
	rows, err := d.Pool.Query(ctx, `
		select id::text, symbol, name from public.instruments
		where exchange = 'KRX' and is_active = true
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var n int
	for rows.Next() {
		var id, symbol, name string
		if err := rows.Scan(&id, &symbol, &name); err != nil {
			return n, err
		}
		// 종목코드 alias (예: "005930" → 삼성전자 instrument)
		if err := ingest.SeedAlias(ctx, d.Pool, symbol, id); err == nil {
			n++
		}
		// 한글명 alias (예: "삼성전자" → 같은 instrument)
		if name != "" {
			if err := ingest.SeedAlias(ctx, d.Pool, name, id); err == nil {
				n++
			}
		}
	}
	return n, rows.Err()
}
```

- [ ] **Step 2: stub에서 해당 함수 제거**

`apps/api/internal/schedule/jobs_stub.go`에서 `JobUpdateInstruments` 행만 삭제:
```go
package schedule

import "context"

func JobUpdateKRPrices(ctx context.Context, d Deps) error    { return nil }
func JobUpdateUSPrices(ctx context.Context, d Deps) error    { return nil }
func JobUpdateIndexQuotes(ctx context.Context, d Deps) error { return nil }
func JobUpdateFXRates(ctx context.Context, d Deps) error     { return nil }
func JobUpdateIndicators(ctx context.Context, d Deps) error  { return nil }
```

- [ ] **Step 3: 빌드 + 커밋**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./... && go test ./...
cd /Users/yuhojin/Desktop/finance
git add apps/api/internal/schedule/
git commit -m "feat(api): JobUpdateInstruments (KIND KOSPI+KOSDAQ 종목 마스터)"
```

---

## Task 5: JobUpdateKRPrices / JobUpdateUSPrices (Yahoo 통합)

**Files:**
- Create: `apps/api/internal/schedule/jobs_prices.go`
- Modify: `apps/api/internal/schedule/jobs_stub.go`

KR·US 일봉을 모두 Yahoo `.KS`/`.KQ` 또는 plain symbol로 fetch. 어제~오늘 범위만(증분).

- [ ] **Step 1: 구현**

`apps/api/internal/schedule/jobs_prices.go`:
```go
package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/quotient/quotient/apps/api/internal/ingest"
	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/quotient/quotient/apps/api/internal/sources/yahoo"
)

// JobUpdateKRPrices fetches yesterday's daily bar for all active KRX instruments via Yahoo.
// 매일 16:30 KST 실행.
func JobUpdateKRPrices(ctx context.Context, d Deps) error {
	return updateDailyByExchange(ctx, d, "KRX")
}

// JobUpdateUSPrices fetches yesterday's daily bar for all active US instruments via Yahoo.
// 매일 06:00 KST 실행.
func JobUpdateUSPrices(ctx context.Context, d Deps) error {
	return updateDailyByExchange(ctx, d, "NASDAQ", "NYSE")
}

func updateDailyByExchange(ctx context.Context, d Deps, exchanges ...string) error {
	type sym struct{ id, code, exchange string }
	var syms []sym

	// pgx의 IN ($1) 바인딩은 array 필요 → ANY($1::text[]).
	// id::text 명시 캐스트 — pgx v5 기본 type map의 uuid → string 미지원 회피.
	rows, err := d.Pool.Query(ctx,
		`select id::text, symbol, exchange from public.instruments
		 where exchange = ANY($1::text[]) and is_active = true`,
		exchanges,
	)
	if err != nil {
		return fmt.Errorf("query instruments: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var s sym
		if err := rows.Scan(&s.id, &s.code, &s.exchange); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		syms = append(syms, s)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}

	// 증분 갱신: 어제~오늘 1일 (Yahoo는 end exclusive — 하루 buffer)
	end := time.Now().UTC()
	start := end.AddDate(0, 0, -2)

	var total int64
	for _, s := range syms {
		ysym := yahooSymbolForExchange(s.code, s.exchange)
		bars, err := d.Yahoo.FetchChart(ctx, ysym, start, end)
		if err != nil {
			slog.Warn("yahoo skip", "symbol", ysym, "err", err)
			continue
		}
		if len(bars) == 0 {
			continue
		}
		for i := range bars {
			bars[i].InstrumentID = s.id
		}
		n, err := ingest.UpsertPrices(ctx, d.Pool, bars)
		if err != nil {
			slog.Warn("upsert skip", "symbol", ysym, "err", err)
			continue
		}
		total += n
		time.Sleep(50 * time.Millisecond) // rate limit (spec §10-2)
	}
	slog.Info("prices updated", "exchanges", exchanges, "instruments", len(syms), "rows", total)
	_ = models.PriceBar{} // import 유지 안전망
	return nil
}

// yahooSymbolForExchange는 W2a의 yahoo.SymbolKR을 활용. KRX → .KS/.KQ는
// 정확한 KOSPI/KOSDAQ 시장 구분 필요. 현재 instruments.exchange는 'KRX' 단일이라
// 시장 정보 부재 → KOSPI(.KS) 기본 + 실패 시 .KQ 재시도.
func yahooSymbolForExchange(symbol, exchange string) string {
	switch exchange {
	case "KRX":
		return yahoo.SymbolKR(symbol, "KOSPI") // .KS suffix
	default:
		return symbol // NASDAQ, NYSE는 plain
	}
}
```

> **알려진 한계 (의도)**: W2a `kind` 어댑터는 모든 KR 종목의 `exchange`를 `"KRX"`로 적재 (KOSPI/KOSDAQ 시장 구분 없음). Yahoo는 `.KS`/`.KQ` 구분 필요. MVP는 `.KS` 기본 사용 — KOSDAQ 종목은 실패하지만 cron이 매일 재시도 (잡 누적 실패 ≠ 데이터 누락). **Task 13 backfill CLI에서는 market 인자로 명시적 구분**. W3에서 instruments 테이블에 `market` 컬럼 추가 검토 (백로그).

- [ ] **Step 2: stub 정리**

`apps/api/internal/schedule/jobs_stub.go`:
```go
package schedule

import "context"

func JobUpdateIndexQuotes(ctx context.Context, d Deps) error { return nil }
func JobUpdateFXRates(ctx context.Context, d Deps) error     { return nil }
func JobUpdateIndicators(ctx context.Context, d Deps) error  { return nil }
```

- [ ] **Step 3: 빌드 + 커밋**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./... && go test ./...
cd /Users/yuhojin/Desktop/finance
git add apps/api/internal/schedule/
git commit -m "feat(api): JobUpdate{KR,US}Prices (Yahoo 통합, 어제~오늘 증분)"
```

---

## Task 6: JobUpdateIndexQuotes (60s TTL 캐시 + Yahoo 지수)

**Files:**
- Create: `apps/api/internal/schedule/jobs_quotes.go`
- Modify: `apps/api/internal/schedule/jobs_stub.go`

매분 폴링하되, `quotes.updated_at`이 60초 미만이면 skip(§10-2). W2b에서는 INDEX(asset_class)만 폴링 — W3에서 holdings/watchlist union으로 확장 예정.

장중 판정: KOSPI/KOSDAQ는 KR 장중에만, SPX/NDX는 US 장중에만 폴링.

- [ ] **Step 1: 구현**

`apps/api/internal/schedule/jobs_quotes.go`:
```go
package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/quotient/quotient/apps/api/internal/ingest"
	"github.com/quotient/quotient/apps/api/internal/models"
)

const quotesCacheTTL = 60 * time.Second

// JobUpdateIndexQuotes refreshes quotes for indices (KOSPI, KOSDAQ, SPX, NDX) every minute during market hours.
// 시세 TTL 60초 (spec §10-2) — updated_at이 60초 미만이면 skip.
// W3에서 holdings/watchlist union으로 확장 (현재는 INDEX 전용).
func JobUpdateIndexQuotes(ctx context.Context, d Deps) error {
	type row struct {
		id, symbol, exchange string
		updatedAt            *time.Time // nullable: 신규 종목은 quotes 행 없음
	}

	// 1) 폴링 대상 + 마지막 적재 시각 (id::text — pgx v5 uuid scan 안전)
	rs, err := d.Pool.Query(ctx, `
		select i.id::text, i.symbol, i.exchange, q.updated_at
		from public.instruments i
		left join public.quotes q on q.instrument_id = i.id
		where i.is_active = true and i.asset_class = 'INDEX'
	`)
	if err != nil {
		return fmt.Errorf("query indices: %w", err)
	}
	defer rs.Close()

	var rows []row
	for rs.Next() {
		var r row
		if err := rs.Scan(&r.id, &r.symbol, &r.exchange, &r.updatedAt); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		rows = append(rows, r)
	}
	if err := rs.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}

	now := time.Now()
	quotes := make([]models.Quote, 0, len(rows))
	for _, r := range rows {
		// TTL 캐시 (spec §10-2)
		if r.updatedAt != nil && now.Sub(*r.updatedAt) < quotesCacheTTL {
			continue
		}

		// 장중 판정 (KOSPI/KOSDAQ → KR, SPX/NDX → US)
		if isKRIndex(r.symbol) && !IsKRMarketOpen(now) {
			continue
		}
		if isUSIndex(r.symbol) && !IsUSMarketOpen(now) {
			continue
		}

		ysym := IndexYahooSymbol(r.symbol, r.exchange)
		if ysym == "" {
			continue
		}
		q, err := d.Yahoo.FetchQuote(ctx, ysym)
		if err != nil {
			slog.Warn("yahoo quote skip", "symbol", ysym, "err", err)
			continue
		}
		q.InstrumentID = r.id
		quotes = append(quotes, q)
		time.Sleep(50 * time.Millisecond)
	}

	if len(quotes) == 0 {
		return nil
	}
	n, err := ingest.UpsertQuotes(ctx, d.Pool, quotes)
	if err != nil {
		return fmt.Errorf("upsert quotes: %w", err)
	}
	slog.Info("index quotes updated", "count", n)
	return nil
}

func isKRIndex(symbol string) bool {
	return symbol == "KOSPI" || symbol == "KOSDAQ"
}
func isUSIndex(symbol string) bool {
	return strings.HasPrefix(symbol, "SP") || symbol == "NDX"
}
```

- [ ] **Step 2: stub 정리**

`apps/api/internal/schedule/jobs_stub.go`:
```go
package schedule

import "context"

func JobUpdateFXRates(ctx context.Context, d Deps) error    { return nil }
func JobUpdateIndicators(ctx context.Context, d Deps) error { return nil }
```

- [ ] **Step 3: 빌드 + 커밋**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./... && go test ./...
cd /Users/yuhojin/Desktop/finance
git add apps/api/internal/schedule/
git commit -m "feat(api): JobUpdateIndexQuotes (지수 분 단위 폴링, 60s TTL, 장중 한정)"
```

---

## Task 7: JobUpdateFXRates (frankfurter.dev, 24/7)

**Files:**
- Create: `apps/api/internal/schedule/jobs_fx.go`
- Modify: `apps/api/internal/schedule/jobs_stub.go`

5분마다 USD→{KRW, EUR, JPY} 환율 → `fx_rates` 적재. 또한 같은 데이터에서 USD_KRW·EUR_KRW·JPY_KRW의 quotes 행도 갱신(TopTicker가 quotes에서 읽도록 단일 경로).

설계: `fx_rates`는 시계열(과거 보존, observed_at는 frankfurter의 date 필드=자정 UTC → 일별 1행/통화쌍 PK). `quotes`는 현재값만. TopTicker는 quotes 단일 SELECT — 핸들러 단순화.

**change_pct 정의**: "어제 또는 직전 거래일 종가 대비 현재값". frankfurter는 ECB 영업일 기준 일별 갱신이라 토요일·일요일·공휴일에는 어제 행이 없을 수 있음 → "오늘 미만 최신 행" fallback으로 휴일 정상 동작. **첫 배포 직후 fx_rates에 행이 1개뿐인 경우 change_pct=0은 의도** (다음 영업일 데이터 적재 후 정상화).

- [ ] **Step 1: 구현**

`apps/api/internal/schedule/jobs_fx.go`:
```go
package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/quotient/quotient/apps/api/internal/ingest"
	"github.com/quotient/quotient/apps/api/internal/models"
)

// JobUpdateFXRates는 frankfurter.dev에서 USD 기준 KRW/EUR/JPY 환율을 받아
// fx_rates(시계열) + quotes(현재값) 양쪽에 적재한다.
// 5분 cron, 24/7.
func JobUpdateFXRates(ctx context.Context, d Deps) error {
	rates, err := d.FX.FetchRates(ctx, "USD", []string{"KRW", "EUR", "JPY"})
	if err != nil {
		return fmt.Errorf("frankfurter: %w", err)
	}

	// 1) fx_rates 적재 (시계열)
	nFX, err := ingest.UpsertFXRates(ctx, d.Pool, rates)
	if err != nil {
		return fmt.Errorf("upsert fx_rates: %w", err)
	}

	// 2) quotes 갱신 (USD_KRW 등 instrument에 매핑)
	rateMap := map[string]float64{}
	for _, r := range rates {
		rateMap[fmt.Sprintf("%s_%s", r.Base, r.Quote)] = r.Rate
	}

	type instRow struct{ id, symbol string }
	rs, err := d.Pool.Query(ctx, `
		select id, symbol from public.instruments
		where asset_class = 'FX' and is_active = true
	`)
	if err != nil {
		return fmt.Errorf("query fx instruments: %w", err)
	}
	defer rs.Close()

	var quotes []models.Quote
	prevRates := previousRates(ctx, d, time.Now().UTC())
	for rs.Next() {
		var r instRow
		if err := rs.Scan(&r.id, &r.symbol); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		v, ok := rateMap[r.symbol]
		if !ok {
			continue
		}
		var changeAbs, changePct float64
		if prev, ok := prevRates[r.symbol]; ok && prev > 0 {
			changeAbs = v - prev
			changePct = (changeAbs / prev) * 100.0
		}
		quotes = append(quotes, models.Quote{
			InstrumentID: r.id,
			Price:        v,
			ChangeAbs:    changeAbs,
			ChangePct:    changePct,
		})
	}
	if err := rs.Err(); err != nil {
		return fmt.Errorf("rows: %w", err)
	}

	nQ, err := ingest.UpsertQuotes(ctx, d.Pool, quotes)
	if err != nil {
		return fmt.Errorf("upsert fx quotes: %w", err)
	}

	slog.Info("fx updated", "fx_rates", nFX, "quotes", nQ)
	return nil
}

// previousRates는 오늘 미만 최신 영업일의 환율을 USD_XXX 키로 반환.
// 휴일·주말로 어제 데이터 부재 시 그 이전 가장 최근 영업일을 자동 선택.
// 첫 배포로 fx_rates에 오늘 행만 있으면 빈 맵 반환 → 호출자가 change_pct=0 처리 (의도).
func previousRates(ctx context.Context, d Deps, today time.Time) map[string]float64 {
	out := map[string]float64{}
	rs, err := d.Pool.Query(ctx, `
		select distinct on (base, quote) base, quote, rate
		from public.fx_rates
		where observed_at::date < $1::date
		order by base, quote, observed_at desc
	`, today.Format("2006-01-02"))
	if err != nil {
		return out
	}
	defer rs.Close()
	for rs.Next() {
		var base, q string
		var r float64
		if err := rs.Scan(&base, &q, &r); err == nil {
			out[fmt.Sprintf("%s_%s", base, q)] = r
		}
	}
	return out
}
```

- [ ] **Step 2: stub 정리**

`apps/api/internal/schedule/jobs_stub.go`:
```go
package schedule

import "context"

func JobUpdateIndicators(ctx context.Context, d Deps) error { return nil }
```

- [ ] **Step 3: 빌드 + 커밋**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./... && go test ./...
cd /Users/yuhojin/Desktop/finance
git add apps/api/internal/schedule/
git commit -m "feat(api): JobUpdateFXRates (frankfurter USD→KRW/EUR/JPY, fx_rates+quotes 동시)"
```

---

## Task 8: JobUpdateIndicators (FRED + ECOS)

**Files:**
- Create: `apps/api/internal/schedule/jobs_indicators.go`
- Delete: `apps/api/internal/schedule/jobs_stub.go`

매일 07:00 KST. FRED의 `DFF` (Fed funds), `DGS10` (10년 국채), ECOS의 `722Y001` (한국 기준금리).

- [ ] **Step 1: 구현**

`apps/api/internal/schedule/jobs_indicators.go`:
```go
package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/quotient/quotient/apps/api/internal/ingest"
)

// JobUpdateIndicators fetches FRED + ECOS economic indicators daily at 07:00 KST.
// 각 시리즈 실패는 다른 시리즈에 영향 안 줌 (부분 실패 허용 — spec §4 "부분 적재").
func JobUpdateIndicators(ctx context.Context, d Deps) error {
	var anyOK bool

	// FRED 시리즈: Fed funds rate + 10y treasury
	for _, code := range []string{"DFF", "DGS10"} {
		obs, err := d.FRED.FetchObservations(ctx, code)
		if err != nil {
			slog.Warn("fred fetch skip", "code", code, "err", err)
			continue
		}
		// FRED는 name·unit 미제공 → ingest에서 code만 사용 (UI에서 표시명 정의)
		n, err := ingest.UpsertIndicators(ctx, d.Pool, obs)
		if err != nil {
			slog.Warn("fred upsert skip", "code", code, "err", err)
			continue
		}
		slog.Info("fred indicator", "code", code, "rows", n)
		anyOK = true
	}

	// ECOS: 한국 기준금리 — 최근 5년치 (Cycle "M" 월별)
	end := time.Now().Format("200601")
	start := time.Now().AddDate(-5, 0, 0).Format("200601")
	obs, err := d.ECOS.FetchSeries(ctx, "722Y001", "M", start, end)
	if err != nil {
		slog.Warn("ecos skip", "err", err)
	} else {
		n, err := ingest.UpsertIndicators(ctx, d.Pool, obs)
		if err != nil {
			slog.Warn("ecos upsert skip", "err", err)
		} else {
			slog.Info("ecos indicator", "code", "722Y001", "rows", n)
			anyOK = true
		}
	}

	if !anyOK {
		return fmt.Errorf("all indicators failed")
	}
	return nil
}
```

- [ ] **Step 2: stub 삭제**

```bash
rm /Users/yuhojin/Desktop/finance/apps/api/internal/schedule/jobs_stub.go
```

- [ ] **Step 3: 빌드 + 커밋**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./... && go test ./...
cd /Users/yuhojin/Desktop/finance
git add -A apps/api/internal/schedule/
git commit -m "feat(api): JobUpdateIndicators (FRED DFF/DGS10 + ECOS 722Y001 한국 기준금리)"
```

---

## Task 9: 마켓 핸들러 — `/v1/market/ticker`

> **Note**: 원 plan v1의 "Task 9 main.go 통합"은 Task 11과 합쳐 단일 커밋 task로 재배치. Task 9·10은 handlers 단독 구현(빌드 가능), Task 11에서 router + main.go 묶음 커밋. subagent-driven-development의 atomic commit 원칙 준수.

**Files:**
- Create: `apps/api/internal/handlers/market.go`
- Create: `apps/api/internal/handlers/market_repo_pg.go`
- Create: `apps/api/internal/handlers/market_test.go`

repo 인터페이스 → fake로 핸들러 단위 테스트. profile_repo_pg.go 패턴 답습.

- [ ] **Step 1: 핸들러 작성**

`apps/api/internal/handlers/market.go`:
```go
package handlers

import (
	"context"
	"log/slog"
	"net/http"
)

// TickerItem은 헤더 티커용 한 행.
type TickerItem struct {
	Symbol    string  `json:"symbol"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
	ChangePct float64 `json:"change_pct"`
}

type MarketRepo interface {
	TickerSeed(ctx context.Context) ([]TickerItem, error)
}

type MarketHandler struct {
	repo MarketRepo
}

func NewMarketHandler(repo MarketRepo) *MarketHandler {
	return &MarketHandler{repo: repo}
}

// GET /v1/market/ticker → 시드 지수·환율 (KOSPI, SPX, USD_KRW)
func (h *MarketHandler) Ticker(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.TickerSeed(r.Context())
	if err != nil {
		slog.Error("ticker fetch failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "ticker fetch failed")
		return
	}
	if items == nil {
		items = []TickerItem{} // null이 아니라 []로 직렬화
	}
	writeJSON(w, http.StatusOK, items)
}
```

- [ ] **Step 2: pg repo**

`apps/api/internal/handlers/market_repo_pg.go`:
```go
package handlers

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgMarketRepo struct {
	pool *pgxpool.Pool
}

func NewPgMarketRepo(pool *pgxpool.Pool) *PgMarketRepo {
	return &PgMarketRepo{pool: pool}
}

// TickerSeed returns the seed indices + USD/KRW for the header ticker.
// quotes 행이 없는 종목은 price=0, change_pct=0으로 반환 (UI가 "—"로 처리).
func (r *PgMarketRepo) TickerSeed(ctx context.Context) ([]TickerItem, error) {
	rows, err := r.pool.Query(ctx, `
		select i.symbol, i.name,
		       coalesce(q.price, 0)::float8 as price,
		       coalesce(q.change_pct, 0)::float8 as change_pct
		from public.instruments i
		left join public.quotes q on q.instrument_id = i.id
		where i.symbol in ('KOSPI', 'SPX', 'USD_KRW')
		order by case i.symbol
		  when 'KOSPI' then 1
		  when 'SPX'   then 2
		  when 'USD_KRW' then 3
		end
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TickerItem
	for rows.Next() {
		var it TickerItem
		if err := rows.Scan(&it.Symbol, &it.Name, &it.Price, &it.ChangePct); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
```

- [ ] **Step 3: 단위 테스트 (fake repo)**

`apps/api/internal/handlers/market_test.go`:
```go
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeMarketRepo struct {
	items []TickerItem
	err   error
}

func (f *fakeMarketRepo) TickerSeed(ctx context.Context) ([]TickerItem, error) {
	return f.items, f.err
}

func TestMarketHandler_Ticker_OK(t *testing.T) {
	repo := &fakeMarketRepo{items: []TickerItem{
		{Symbol: "KOSPI", Name: "KOSPI 종합", Price: 2700.5, ChangePct: 0.42},
		{Symbol: "SPX", Name: "S&P 500", Price: 4800, ChangePct: -0.1},
	}}
	h := NewMarketHandler(repo)
	w := httptest.NewRecorder()
	h.Ticker(w, httptest.NewRequest(http.MethodGet, "/v1/market/ticker", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var got []TickerItem
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got) != 2 || got[0].Symbol != "KOSPI" {
		t.Errorf("got %+v", got)
	}
}

func TestMarketHandler_Ticker_NilToEmptyArray(t *testing.T) {
	h := NewMarketHandler(&fakeMarketRepo{items: nil})
	w := httptest.NewRecorder()
	h.Ticker(w, httptest.NewRequest(http.MethodGet, "/v1/market/ticker", nil))
	if got := w.Body.String(); got != "[]\n" {
		t.Errorf("nil items should serialize as [], got %q", got)
	}
}

func TestMarketHandler_Ticker_Error(t *testing.T) {
	h := NewMarketHandler(&fakeMarketRepo{err: errors.New("db down")})
	w := httptest.NewRecorder()
	h.Ticker(w, httptest.NewRequest(http.MethodGet, "/v1/market/ticker", nil))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", w.Code)
	}
}
```

- [ ] **Step 4: 테스트 통과 (router 미수정 상태)**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api
# router/main 미수정으로 빌드 실패는 의도. handlers 패키지만 단독 테스트.
go test ./internal/handlers/... 2>&1 | tail -20
# 예상: market_test 3개 PASS, 다른 테스트도 통과 (router/main 미터치라 영향 없음)
```

- [ ] **Step 5: stage (커밋은 Task 11)**

```bash
cd /Users/yuhojin/Desktop/finance
git add apps/api/internal/handlers/market.go apps/api/internal/handlers/market_repo_pg.go apps/api/internal/handlers/market_test.go
```

---

## Task 10: 종목 검색 핸들러 — `/v1/instruments/search` + `/select`

**Files:**
- Create: `apps/api/internal/handlers/instruments.go`
- Create: `apps/api/internal/handlers/instruments_repo_pg.go`
- Create: `apps/api/internal/handlers/instruments_test.go`

spec §10-9: alias 학습. 사용자가 검색 결과 선택 시 검색어 → instrument_id를 `instrument_aliases (source='learned')` 적재.

- [ ] **Step 1: 핸들러**

`apps/api/internal/handlers/instruments.go`:
```go
package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
)

type SearchResult struct {
	ID       string `json:"id"`
	Symbol   string `json:"symbol"`
	Exchange string `json:"exchange"`
	Name     string `json:"name"`
}

type InstrumentRepo interface {
	SearchByAlias(ctx context.Context, query string) ([]SearchResult, error)
	SearchByText(ctx context.Context, query string) ([]SearchResult, error)
	LearnAlias(ctx context.Context, alias, instrumentID string) error
}

type InstrumentHandler struct {
	repo InstrumentRepo
}

func NewInstrumentHandler(repo InstrumentRepo) *InstrumentHandler {
	return &InstrumentHandler{repo: repo}
}

// GET /v1/instruments/search?q=
func (h *InstrumentHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, http.StatusOK, []SearchResult{})
		return
	}

	// 1차: alias 정확 매칭
	results, err := h.repo.SearchByAlias(r.Context(), q)
	if err != nil {
		slog.Error("alias search failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "search failed")
		return
	}

	// 2차: name/symbol ILIKE (alias 매칭 없을 때만)
	if len(results) == 0 {
		results, err = h.repo.SearchByText(r.Context(), q)
		if err != nil {
			slog.Error("text search failed", "err", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "search failed")
			return
		}
	}

	if results == nil {
		results = []SearchResult{}
	}
	writeJSON(w, http.StatusOK, results)
}

// POST /v1/instruments/select {query, instrument_id} → alias 학습
func (h *InstrumentHandler) Select(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var body struct {
		Query        string `json:"query"`
		InstrumentID string `json:"instrument_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if strings.TrimSpace(body.Query) == "" || strings.TrimSpace(body.InstrumentID) == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "query and instrument_id required")
		return
	}
	if err := h.repo.LearnAlias(r.Context(), body.Query, body.InstrumentID); err != nil {
		slog.Error("learn alias failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "learn failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"learned": true})
}
```

- [ ] **Step 2: pg repo**

`apps/api/internal/handlers/instruments_repo_pg.go`:
```go
package handlers

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/ingest"
)

type PgInstrumentRepo struct {
	pool *pgxpool.Pool
}

func NewPgInstrumentRepo(pool *pgxpool.Pool) *PgInstrumentRepo {
	return &PgInstrumentRepo{pool: pool}
}

func (r *PgInstrumentRepo) SearchByAlias(ctx context.Context, query string) ([]SearchResult, error) {
	rows, err := r.pool.Query(ctx, `
		select i.id::text, i.symbol, i.exchange, i.name
		from public.instrument_aliases a
		join public.instruments i on i.id = a.instrument_id
		where a.alias = $1 and i.is_active = true
		limit 10
	`, strings.ToLower(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanResults(rows)
}

func (r *PgInstrumentRepo) SearchByText(ctx context.Context, query string) ([]SearchResult, error) {
	pat := "%" + strings.ToLower(query) + "%"
	rows, err := r.pool.Query(ctx, `
		select id::text, symbol, exchange, name from public.instruments
		where (lower(name) like $1 or lower(symbol) like $1) and is_active = true
		order by length(symbol) asc
		limit 10
	`, pat)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanResults(rows)
}

func (r *PgInstrumentRepo) LearnAlias(ctx context.Context, alias, instrumentID string) error {
	return ingest.LearnAlias(ctx, r.pool, alias, instrumentID)
}

func scanResults(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]SearchResult, error) {
	var out []SearchResult
	for rows.Next() {
		var s SearchResult
		if err := rows.Scan(&s.ID, &s.Symbol, &s.Exchange, &s.Name); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
```

- [ ] **Step 3: 테스트 (fake repo)**

`apps/api/internal/handlers/instruments_test.go`:
```go
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeInstrRepo struct {
	aliasResults []SearchResult
	textResults  []SearchResult
	learnErr     error
	lastLearned  struct{ alias, id string }
}

func (f *fakeInstrRepo) SearchByAlias(ctx context.Context, q string) ([]SearchResult, error) {
	return f.aliasResults, nil
}
func (f *fakeInstrRepo) SearchByText(ctx context.Context, q string) ([]SearchResult, error) {
	return f.textResults, nil
}
func (f *fakeInstrRepo) LearnAlias(ctx context.Context, alias, id string) error {
	f.lastLearned.alias = alias
	f.lastLearned.id = id
	return f.learnErr
}

func TestSearch_EmptyQuery(t *testing.T) {
	h := NewInstrumentHandler(&fakeInstrRepo{})
	w := httptest.NewRecorder()
	h.Search(w, httptest.NewRequest(http.MethodGet, "/v1/instruments/search?q=", nil))
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Errorf("empty q should return [], got %d %q", w.Code, w.Body.String())
	}
}

func TestSearch_AliasFirst(t *testing.T) {
	repo := &fakeInstrRepo{
		aliasResults: []SearchResult{{ID: "1", Symbol: "005930", Exchange: "KRX", Name: "삼성전자"}},
		textResults:  []SearchResult{{ID: "2", Symbol: "X", Exchange: "X", Name: "X"}}, // 호출되면 안 됨
	}
	h := NewInstrumentHandler(repo)
	w := httptest.NewRecorder()
	h.Search(w, httptest.NewRequest(http.MethodGet, "/v1/instruments/search?q=samsung", nil))
	var got []SearchResult
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if len(got) != 1 || got[0].Symbol != "005930" {
		t.Errorf("alias should win, got %+v", got)
	}
}

func TestSearch_FallbackText(t *testing.T) {
	repo := &fakeInstrRepo{
		aliasResults: nil,
		textResults:  []SearchResult{{ID: "2", Symbol: "AAPL", Exchange: "NASDAQ", Name: "Apple"}},
	}
	h := NewInstrumentHandler(repo)
	w := httptest.NewRecorder()
	h.Search(w, httptest.NewRequest(http.MethodGet, "/v1/instruments/search?q=apple", nil))
	var got []SearchResult
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if len(got) != 1 || got[0].Symbol != "AAPL" {
		t.Errorf("text fallback failed, got %+v", got)
	}
}

func TestSelect_LearnsAlias(t *testing.T) {
	repo := &fakeInstrRepo{}
	h := NewInstrumentHandler(repo)
	body, _ := json.Marshal(map[string]string{"query": "삼전", "instrument_id": "uuid-1"})
	w := httptest.NewRecorder()
	h.Select(w, httptest.NewRequest(http.MethodPost, "/v1/instruments/select", bytes.NewReader(body)))
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	if repo.lastLearned.alias != "삼전" || repo.lastLearned.id != "uuid-1" {
		t.Errorf("learn args wrong: %+v", repo.lastLearned)
	}
}

func TestSelect_BadJSON(t *testing.T) {
	h := NewInstrumentHandler(&fakeInstrRepo{})
	w := httptest.NewRecorder()
	h.Select(w, httptest.NewRequest(http.MethodPost, "/v1/instruments/select", strings.NewReader("not json")))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400", w.Code)
	}
}

func TestSelect_MissingFields(t *testing.T) {
	h := NewInstrumentHandler(&fakeInstrRepo{})
	body, _ := json.Marshal(map[string]string{"query": "", "instrument_id": ""})
	w := httptest.NewRecorder()
	h.Select(w, httptest.NewRequest(http.MethodPost, "/v1/instruments/select", bytes.NewReader(body)))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400", w.Code)
	}
}

func TestSelect_RepoError(t *testing.T) {
	h := NewInstrumentHandler(&fakeInstrRepo{learnErr: errors.New("db")})
	body, _ := json.Marshal(map[string]string{"query": "x", "instrument_id": "y"})
	w := httptest.NewRecorder()
	h.Select(w, httptest.NewRequest(http.MethodPost, "/v1/instruments/select", bytes.NewReader(body)))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status %d, want 500", w.Code)
	}
}
```

- [ ] **Step 4: 테스트 통과 + stage**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api
go test ./internal/handlers/... -run 'TestSearch|TestSelect' -v
cd /Users/yuhojin/Desktop/finance
git add apps/api/internal/handlers/instruments.go apps/api/internal/handlers/instruments_repo_pg.go apps/api/internal/handlers/instruments_test.go
```

---

## Task 11: 라우터 + main.go 통합 커밋

**Files:**
- Modify: `apps/api/internal/router/router.go`
- Modify: `apps/api/cmd/server/main.go`

Task 9·10에서 만든 market·instrument 핸들러를 라우터에 연결 + main.go에서 인스턴스화·cron 워커 시작·graceful shutdown 처리. 단일 task·단일 커밋으로 atomic하게 처리 (subagent-driven-development의 원자성 원칙).

- [ ] **Step 1: router.go 갱신**

```go
package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/quotient/quotient/apps/api/internal/auth"
	"github.com/quotient/quotient/apps/api/internal/handlers"
	"github.com/quotient/quotient/apps/api/internal/middleware"
)

func New(
	verifier *auth.Verifier,
	corsOrigin string,
	profileHandler *handlers.ProfileHandler,
	marketHandler *handlers.MarketHandler,
	instrumentHandler *handlers.InstrumentHandler,
	readyz http.HandlerFunc,
) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.CORS(corsOrigin))
	r.Get("/healthz", handlers.Healthz)
	r.Get("/readyz", readyz)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(verifier))
		r.Get("/v1/profile", profileHandler.Get)
		r.Patch("/v1/profile", profileHandler.Patch)
		r.Get("/v1/market/ticker", marketHandler.Ticker)
		r.Get("/v1/instruments/search", instrumentHandler.Search)
		r.Post("/v1/instruments/select", instrumentHandler.Select)
	})

	return r
}
```

- [ ] **Step 2: main.go 갱신**

`apps/api/cmd/server/main.go` 전체 교체:
```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/quotient/quotient/apps/api/internal/auth"
	"github.com/quotient/quotient/apps/api/internal/config"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/handlers"
	"github.com/quotient/quotient/apps/api/internal/router"
	"github.com/quotient/quotient/apps/api/internal/schedule"
	"github.com/quotient/quotient/apps/api/internal/sources/ecos"
	"github.com/quotient/quotient/apps/api/internal/sources/fred"
	"github.com/quotient/quotient/apps/api/internal/sources/fx"
	"github.com/quotient/quotient/apps/api/internal/sources/kind"
	"github.com/quotient/quotient/apps/api/internal/sources/yahoo"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load failed", "err", err)
		os.Exit(1)
	}

	// cron 잡 ctx — 잡 내 외부 호출 cancel 전파 가능하도록 별도 cancel 보유
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	verifier := auth.NewVerifier(cfg.SupabaseJWTSecret)
	profileRepo := handlers.NewPgProfileRepo(pool)
	profileHandler := handlers.NewProfileHandler(profileRepo)
	marketRepo := handlers.NewPgMarketRepo(pool)
	marketHandler := handlers.NewMarketHandler(marketRepo)
	instrumentRepo := handlers.NewPgInstrumentRepo(pool)
	instrumentHandler := handlers.NewInstrumentHandler(instrumentRepo)
	readyz := handlers.ReadyzHandler(pool)

	// cron 워커 시작
	cronWorker := schedule.Start(ctx, schedule.Deps{
		Pool:  pool,
		KIND:  kind.NewClient(""),
		Yahoo: yahoo.NewClient(),
		FX:    fx.NewClient(""),
		FRED:  fred.NewClient("", cfg.FREDAPIKey),
		ECOS:  ecos.NewClient("", cfg.ECOSAPIKey),
	})

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           router.New(verifier, cfg.CORSOrigin, profileHandler, marketHandler, instrumentHandler, readyz),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.Info("API listening", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen failed", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger.Info("shutting down")

	// 1) HTTP 서버 graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown failed", "err", err)
	}

	// 2) cron 워커 정지
	//   - Stop()은 새 잡 추가 차단 + 진행 중 잡 완료 대기를 위한 ctx 반환
	//   - cancel() 호출로 ctx-aware 잡이 외부 API 호출을 즉시 cancel하도록
	cronCtx := cronWorker.Stop()
	cancel()
	select {
	case <-cronCtx.Done():
		logger.Info("cron stopped cleanly")
	case <-time.After(30 * time.Second):
		logger.Warn("cron stop timeout — proceeding")
	}
}
```

- [ ] **Step 3: 전체 빌드 + 테스트**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api
go mod tidy
go build ./...
go test ./...
# 예상: 전체 PASS. integration 테스트는 build tag로 분리되어 기본 실행 안 됨.
```

- [ ] **Step 4: 통합 커밋 (Task 9·10·11 묶음)**

```bash
cd /Users/yuhojin/Desktop/finance
git add apps/api/internal/handlers/market.go apps/api/internal/handlers/market_repo_pg.go apps/api/internal/handlers/market_test.go
git add apps/api/internal/handlers/instruments.go apps/api/internal/handlers/instruments_repo_pg.go apps/api/internal/handlers/instruments_test.go
git add apps/api/internal/router/router.go apps/api/cmd/server/main.go
git status
git commit -m "feat(api): cron 워커 + 마켓 API (/v1/market/ticker, /v1/instruments/search·select) 통합"
```

---

## Task 12: TopTicker 실데이터 연결 (Next.js 16)

**Files:**
- Create: `apps/web/lib/api/market.ts`
- Modify: `apps/web/components/shell/TopTicker.tsx`
- Modify: `apps/web/.env.example` (있다면) — `NEXT_PUBLIC_API_URL`

> **Next.js 16 사전 확인**: `apps/web/AGENTS.md` 지시대로 client component 가이드 확인. 본 task는 기본 client component(`"use client"` + hooks) 패턴을 사용 — Next.js 16에서도 유효하나, fetch 옵션·env 노출 규칙이 바뀌었을 수 있으니 시작 전 가이드를 훑는다.

- [ ] **Step 0: Next.js 16 client component·data fetching 가이드 확인**

```bash
cd /Users/yuhojin/Desktop/finance/apps/web
ls node_modules/next/dist/docs/ 2>/dev/null | head
# 가이드 파일이 있으면 아래에서 핵심 항목 확인 (없으면 npm next docs 또는 release notes)
find node_modules/next/dist/docs -name "*client*" -o -name "*fetch*" 2>/dev/null | head
```
다음 항목 확인 후 진행:
- `"use client"` directive: 유효한지, deprecation 있는지
- `useEffect` + `fetch` + `cache: "no-store"`: Next.js 16에서 권장되는 패턴인지
- `process.env.NEXT_PUBLIC_*` 클라이언트 노출: 그대로인지

가이드와 본 Task의 구현이 충돌하면 main agent에 보고 후 재계획.

- [ ] **Step 1: API 클라이언트 helper**

`apps/web/lib/api/market.ts`:
```ts
export type Ticker = {
  symbol: string;
  name: string;
  price: number;
  change_pct: number;
};

const API_BASE =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export async function fetchTicker(accessToken: string): Promise<Ticker[]> {
  const res = await fetch(`${API_BASE}/v1/market/ticker`, {
    headers: { Authorization: `Bearer ${accessToken}` },
    cache: "no-store",
  });
  if (!res.ok) return [];
  return res.json() as Promise<Ticker[]>;
}
```

- [ ] **Step 2: TopTicker 갱신**

`apps/web/components/shell/TopTicker.tsx`:
```tsx
"use client";

import { useEffect, useState } from "react";
import { createSupabaseBrowser } from "@/lib/supabase/client";
import { fetchTicker, type Ticker } from "@/lib/api/market";

const SEED_DISPLAY = [
  { symbol: "KOSPI", label: "KOSPI" },
  { symbol: "SPX", label: "S&P 500" },
  { symbol: "USD_KRW", label: "USD/KRW" },
];

function formatPrice(symbol: string, price: number): string {
  if (price === 0) return "—";
  if (symbol === "USD_KRW") return price.toFixed(2);
  return price.toLocaleString("ko-KR", {
    maximumFractionDigits: 2,
    minimumFractionDigits: 2,
  });
}

function changeClass(pct: number): string {
  if (pct > 0) return "text-bb-up";
  if (pct < 0) return "text-bb-down";
  return "text-fg";
}

export function TopTicker() {
  const [items, setItems] = useState<Ticker[]>([]);

  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setInterval> | undefined;

    async function load() {
      // 탭이 백그라운드면 호출 skip (배터리·API quota 절약)
      if (typeof document !== "undefined" && document.visibilityState === "hidden") return;
      const supabase = createSupabaseBrowser();
      const {
        data: { session },
      } = await supabase.auth.getSession();
      if (!session) return;
      const data = await fetchTicker(session.access_token);
      if (!cancelled) setItems(data);
    }

    void load();
    timer = setInterval(() => void load(), 60_000);
    return () => {
      cancelled = true;
      if (timer) clearInterval(timer);
    };
  }, []);

  const merged = SEED_DISPLAY.map((seed) => {
    const found = items.find((it) => it.symbol === seed.symbol);
    return {
      label: seed.label,
      price: found?.price ?? 0,
      changePct: found?.change_pct ?? 0,
      symbol: seed.symbol,
    };
  });

  return (
    <header className="h-9 border-b border-line bg-bg flex items-center px-4 gap-6 text-xs">
      <span className="font-mono text-bb-accent">QUOTIENT</span>
      {merged.map((it) => (
        <span key={it.symbol} className="font-mono text-fg-muted">
          {it.label}{" "}
          <span className={changeClass(it.changePct)}>
            {formatPrice(it.symbol, it.price)}
          </span>
          {it.price > 0 && (
            <span className={`${changeClass(it.changePct)} ml-1`}>
              ({it.changePct >= 0 ? "+" : ""}
              {it.changePct.toFixed(2)}%)
            </span>
          )}
        </span>
      ))}
      <span className="ml-auto font-mono text-fg-muted text-[10px]">
        시세 지연 15분
      </span>
    </header>
  );
}
```

- [ ] **Step 3: env 추가**

`apps/web/.env.example`에 추가 (이미 있으면 skip):
```
NEXT_PUBLIC_API_URL=http://localhost:8080
```

`apps/web/.env.local`에도 동일 행 추가 (개인 로컬, gitignore됨):
```bash
echo "NEXT_PUBLIC_API_URL=http://localhost:8080" >> /Users/yuhojin/Desktop/finance/apps/web/.env.local
```

- [ ] **Step 4: 빌드 + 커밋**

```bash
cd /Users/yuhojin/Desktop/finance/apps/web
npm run build
# Next.js 16 빌드 통과 확인
cd /Users/yuhojin/Desktop/finance
git add apps/web/lib/api/market.ts apps/web/components/shell/TopTicker.tsx apps/web/.env.example
git commit -m "feat(web): TopTicker 실데이터 연결 (/v1/market/ticker, 60초 폴링)"
```

---

## Task 13: 5년 백필 CLI (`cmd/backfill`)

**Files:**
- Create: `apps/api/cmd/backfill/main.go`

KOSPI/KOSDAQ는 KIND로 종목 마스터 채우고 Yahoo로 5년 일봉. NASDAQ은 시드 리스트(`AAPL` 등 30개)로 Yahoo만.

- [ ] **Step 1: CLI 골격**

`apps/api/cmd/backfill/main.go`:
```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/config"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/ingest"
	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/quotient/quotient/apps/api/internal/sources/kind"
	"github.com/quotient/quotient/apps/api/internal/sources/yahoo"
)

// nasdaqSeed는 backfill 시드 30종목. Phase 2에서 S&P 100 전체로 확장.
var nasdaqSeed = []struct{ Symbol, Name string }{
	{"AAPL", "Apple Inc."}, {"MSFT", "Microsoft"}, {"GOOGL", "Alphabet Class A"},
	{"AMZN", "Amazon"}, {"NVDA", "NVIDIA"}, {"META", "Meta Platforms"},
	{"TSLA", "Tesla"}, {"AVGO", "Broadcom"}, {"NFLX", "Netflix"},
	{"AMD", "Advanced Micro Devices"}, {"INTC", "Intel"}, {"ORCL", "Oracle"},
	{"CRM", "Salesforce"}, {"ADBE", "Adobe"}, {"QCOM", "Qualcomm"},
	{"TXN", "Texas Instruments"}, {"COST", "Costco"}, {"PEP", "PepsiCo"},
	{"CSCO", "Cisco"}, {"TMUS", "T-Mobile US"}, {"INTU", "Intuit"},
	{"AMAT", "Applied Materials"}, {"BKNG", "Booking Holdings"},
	{"ISRG", "Intuitive Surgical"}, {"REGN", "Regeneron"},
	{"VRTX", "Vertex Pharmaceuticals"}, {"LRCX", "Lam Research"},
	{"PANW", "Palo Alto Networks"}, {"ADP", "Automatic Data Processing"},
	{"GILD", "Gilead Sciences"},
}

func main() {
	years := flag.Int("years", 5, "백필 기간 (연 단위)")
	market := flag.String("market", "KOSPI", "KOSPI | KOSDAQ | NASDAQ")
	limit := flag.Int("limit", 0, "최대 종목 수 (0=전체). 디버깅용.")
	flag.Parse()

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	switch *market {
	case "KOSPI", "KOSDAQ":
		if err := runKR(ctx, pool, *market, *years, *limit); err != nil {
			slog.Error("kr backfill", "err", err)
			os.Exit(1)
		}
	case "NASDAQ":
		if err := runUS(ctx, pool, *years, *limit); err != nil {
			slog.Error("us backfill", "err", err)
			os.Exit(1)
		}
	default:
		slog.Error("unknown market", "market", *market)
		os.Exit(1)
	}
	slog.Info("backfill done", "market", *market, "years", *years)
}

func runKR(ctx context.Context, pool *pgxpool.Pool, market string, years, limit int) error {
	kc := kind.NewClient("")
	yc := yahoo.NewClient()

	// 1) 종목 마스터
	items, err := kc.FetchInstruments(ctx, market)
	if err != nil {
		return fmt.Errorf("kind: %w", err)
	}
	slog.Info("instruments fetched", "market", market, "count", len(items))
	if _, err := ingest.UpsertInstruments(ctx, pool, items); err != nil {
		return fmt.Errorf("upsert instruments: %w", err)
	}

	// 2) DB에서 id+symbol 조회 (지수·FX 제외, KRX exchange)
	rows, err := pool.Query(ctx,
		`select id::text, symbol from public.instruments where exchange = 'KRX' and is_active = true order by symbol`)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	type s struct{ id, code string }
	var syms []s
	for rows.Next() {
		var x s
		if err := rows.Scan(&x.id, &x.code); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		syms = append(syms, x)
	}
	if limit > 0 && len(syms) > limit {
		syms = syms[:limit]
	}

	end := time.Now().UTC()
	start := end.AddDate(-years, 0, 0)
	for i, sym := range syms {
		ysym := yahoo.SymbolKR(sym.code, market) // KOSPI→.KS, KOSDAQ→.KQ
		bars, err := yc.FetchChart(ctx, ysym, start, end)
		if err != nil {
			slog.Warn("yahoo skip", "symbol", ysym, "err", err)
			continue
		}
		if len(bars) == 0 {
			continue
		}
		for j := range bars {
			bars[j].InstrumentID = sym.id
		}
		n, err := ingest.UpsertPrices(ctx, pool, bars)
		if err != nil {
			slog.Warn("upsert skip", "symbol", ysym, "err", err)
			continue
		}
		slog.Info("backfilled", "i", i+1, "total", len(syms), "symbol", ysym, "rows", n)
		time.Sleep(200 * time.Millisecond) // Yahoo rate limit 보호
	}
	return nil
}

func runUS(ctx context.Context, pool *pgxpool.Pool, years, limit int) error {
	yc := yahoo.NewClient()

	// 1) instruments에 시드 upsert
	insts := make([]models.Instrument, 0, len(nasdaqSeed))
	for _, s := range nasdaqSeed {
		insts = append(insts, models.Instrument{
			Symbol: s.Symbol, Exchange: "NASDAQ", Name: s.Name,
			AssetClass: models.AssetUSStock, Currency: "USD", IsActive: true,
		})
	}
	if _, err := ingest.UpsertInstruments(ctx, pool, insts); err != nil {
		return fmt.Errorf("upsert seed: %w", err)
	}

	// 2) DB에서 id 조회
	rows, err := pool.Query(ctx,
		`select id::text, symbol from public.instruments where exchange = 'NASDAQ' and is_active = true order by symbol`)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	type s struct{ id, code string }
	var syms []s
	for rows.Next() {
		var x s
		if err := rows.Scan(&x.id, &x.code); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		syms = append(syms, x)
	}
	if limit > 0 && len(syms) > limit {
		syms = syms[:limit]
	}

	end := time.Now().UTC()
	start := end.AddDate(-years, 0, 0)
	for i, sym := range syms {
		bars, err := yc.FetchChart(ctx, sym.code, start, end)
		if err != nil {
			slog.Warn("yahoo skip", "symbol", sym.code, "err", err)
			continue
		}
		if len(bars) == 0 {
			continue
		}
		for j := range bars {
			bars[j].InstrumentID = sym.id
		}
		n, err := ingest.UpsertPrices(ctx, pool, bars)
		if err != nil {
			slog.Warn("upsert skip", "symbol", sym.code, "err", err)
			continue
		}
		slog.Info("backfilled", "i", i+1, "total", len(syms), "symbol", sym.code, "rows", n)
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}
```

- [ ] **Step 2: 빌드**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api
go build -o /tmp/backfill ./cmd/backfill
```
Expected: 빌드 성공, `/tmp/backfill` 생성.

- [ ] **Step 3: 소규모 실행 테스트 (NASDAQ 30종목, 1년)**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api
export $(cat ../../.env | grep -v '^#' | xargs)
go run ./cmd/backfill -market NASDAQ -years 1 -limit 5 2>&1 | tail -30
docker exec supabase_db_finance psql -U postgres -d postgres -c "select count(*) from public.prices;"
```
Expected: `count > 0` (대략 5종목 × 250영업일 = ~1250행).

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance
git add apps/api/cmd/backfill/
git commit -m "feat(api): 5년 백필 CLI (KOSPI/KOSDAQ via KIND+Yahoo, NASDAQ 30종목 시드)"
```

---

## Task 14: 통합 검증 + 문서 갱신

**Files:**
- Modify: `docs/STATUS.md`
- Modify: `docs/ROADMAP.md`

cron 워커 + API + UI가 실제로 작동하는지 단위·통합 검증.

- [ ] **검증 1: 전체 빌드·테스트 통과**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api
go build ./...
go test ./...
go test -tags integration ./internal/ingest/...   # W2a 통합 테스트
```
Expected: 모두 PASS. (schedule 잡 자체의 integration 테스트는 향후 작성.)

- [ ] **검증 2: API 서버 + cron 워커 기동**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api
export $(cat ../../.env | grep -v '^#' | xargs)
go run ./cmd/server 2>&1 | tee /tmp/api.log &
sleep 3
curl -s http://localhost:8080/healthz
# {"status":"ok"}
curl -s http://localhost:8080/readyz
# {"status":"ready"}
```
Expected: 로그에 `cron started jobs=6 tz=Asia/Seoul`. ready/healthz 통과.

- [ ] **검증 3: FX 잡 5분 내 적재**

```bash
# 첫 cron tick 대기 (현재 분 + 1)
sleep 65
docker exec supabase_db_finance psql -U postgres -d postgres -c "select base, quote, rate from public.fx_rates order by observed_at desc limit 5;"
docker exec supabase_db_finance psql -U postgres -d postgres -c "select i.symbol, q.price, q.change_pct from public.quotes q join public.instruments i on i.id = q.instrument_id where i.asset_class = 'FX';"
```
Expected: `fx_rates`에 USD→KRW/EUR/JPY 행 + `quotes`에 USD_KRW 등 행.

- [ ] **검증 4: 지수 quotes 잡 (장중 한정)**

장중(KST 09:00~15:30 또는 23:30~06:00)에 실행 시:
```bash
sleep 65
docker exec supabase_db_finance psql -U postgres -d postgres -c "select i.symbol, q.price, q.updated_at from public.quotes q join public.instruments i on i.id = q.instrument_id where i.asset_class = 'INDEX';"
```
Expected: KR 장중이면 KOSPI/KOSDAQ, US 장중이면 SPX/NDX 행. 장 마감 시간이면 행 없음 (정상).

- [ ] **검증 5: 테스트 user 생성 + access_token 획득**

W1 가입 페이지가 있으면 브라우저로:
1. http://localhost:3000/signup 에서 `test@quotient.app` / `TestPass123!` 가입
2. 이메일 인증 (Supabase Studio Inbox: `http://localhost:54323` → Authentication → Users → test@quotient.app → "Send email reset" 또는 직접 confirmed_at 수동 update)
3. http://localhost:3000/login 후 DevTools → Application → Local Storage → `sb-<project-ref>-auth-token` 키의 `access_token` 복사

또는 CLI로 직접 토큰 발급:
```bash
curl -s -X POST "http://localhost:54321/auth/v1/token?grant_type=password" \
  -H "apikey: $SUPABASE_ANON_KEY" \
  -H "Content-Type: application/json" \
  -d '{"email":"test@quotient.app","password":"TestPass123!"}' | jq -r .access_token
```
(SUPABASE_ANON_KEY는 `apps/web/.env.local` 또는 `supabase status` 출력에서 확인)

```bash
TOKEN="<paste-access-token>"
```

- [ ] **검증 6: `/v1/market/ticker` 응답 (인증 필요)**

```bash
curl -s http://localhost:8080/v1/market/ticker -H "Authorization: Bearer $TOKEN" | jq .
```
Expected:
```json
[
  {"symbol":"KOSPI","name":"KOSPI 종합","price":<num or 0>,"change_pct":<num>},
  {"symbol":"SPX","name":"S&P 500","price":<num or 0>,"change_pct":<num>},
  {"symbol":"USD_KRW","name":"USD/KRW","price":<num>,"change_pct":<num>}
]
```

- [ ] **검증 7: `/v1/instruments/search` 응답**

```bash
# backfill 실행 후 (Task 13 완료 가정)
curl -s "http://localhost:8080/v1/instruments/search?q=apple" -H "Authorization: Bearer $TOKEN" | jq .
curl -s "http://localhost:8080/v1/instruments/search?q=AAPL" -H "Authorization: Bearer $TOKEN" | jq .
```
Expected: AAPL 행 반환.

- [ ] **검증 8: alias 학습 (`/v1/instruments/select`)**

```bash
APPLE_ID=$(curl -s "http://localhost:8080/v1/instruments/search?q=apple" -H "Authorization: Bearer $TOKEN" | jq -r '.[0].id')
curl -s -X POST "http://localhost:8080/v1/instruments/select" \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d "{\"query\":\"애플\",\"instrument_id\":\"$APPLE_ID\"}"
# {"learned":true}
curl -s "http://localhost:8080/v1/instruments/search?q=애플" -H "Authorization: Bearer $TOKEN" | jq .
```
Expected: "애플"로 검색 시 AAPL 반환 (alias 학습 동작).

- [ ] **검증 9: TopTicker UI 실데이터**

```bash
cd /Users/yuhojin/Desktop/finance/apps/web
npm run dev &
sleep 5
# 브라우저로 http://localhost:3000 접속 → 로그인 → /app 진입
```
Expected: 상단 티커에 "—" 대신 실수치(예: USD/KRW 1,450.20 (+0.12%)). 60초마다 갱신.

- [ ] **검증 10: graceful shutdown**

```bash
kill -TERM <api-pid>
# 로그에 "shutting down" → "cron stopped cleanly" 또는 "cron stop timeout" 둘 중 하나
```
Expected: 잡 진행 중이면 완료 후 종료, 30초 초과 시 timeout 메시지 + 정상 종료.

- [ ] **검증 11: 백필 (이미 Task 13 Step 3에서 검증)**

5종목 1년 백필 후 `prices` 테이블에 ~1250행 적재 확인 완료.

- [ ] **STATUS.md 갱신**

`docs/STATUS.md`:
- "현재 Phase"를 `Phase 1 — W1·W2a·W2b 완료. W3 작성 대기.`로 수정
- "진행 중"에서 `W2b plan 작성` 행 제거
- "완료" 섹션에 W2b-T1~T15 행 추가 (각각 commit hash 포함):
  ```
  - ✅ W2b-T1 market_hours helper (`<hash>`)
  - ✅ W2b-T2 yahoo_symbols helper (`<hash>`)
  - ... (T15까지)
  ```
- "최근 변경 이력" 맨 위에 한 줄:
  ```
  - 2026-05-22 W2b 전체 완료. cron 워커 6 잡 + 마켓 API + TopTicker 실데이터 + 5년 백필 CLI.
  ```
- "마지막 업데이트" → 2026-05-22

- [ ] **ROADMAP.md 갱신**

`docs/ROADMAP.md`:
- "현재 추천 다음 작업"을 W3(포트폴리오 holdings + watchlist + CRUD UI)로 갱신
- Phase 1 표에서 완료된 행(`마켓 데이터 수집 워커`, `종목 마스터 + 시세 캐시`) 제거

- [ ] **최종 커밋**

```bash
cd /Users/yuhojin/Desktop/finance
git add docs/STATUS.md docs/ROADMAP.md
git commit -m "docs: W2b 완료 반영 (STATUS·ROADMAP 갱신)"
```

---

## 자체 검토 (W2b)

### 스펙 커버리지

| Spec | W2b 반영 |
|---|---|
| §4 갱신 주기 (마스터·prices·quotes·FX·indicators) | ✅ Task 3 cron 6 잡 |
| §4 장중 정의 (KR/US) | ✅ Task 1 market_hours + Task 6 적용 |
| §4 단일 Go 프로세스 + robfig/cron | ✅ Task 3·9 |
| §4 부분 적재 허용 | ✅ Task 4·5·8 (개별 종목 실패 skip) |
| §10-2 quotes 60초 TTL | ✅ Task 6 (`quotesCacheTTL`) |
| §10-2 점진 백오프 429 | ⚠️ 의도된 비범위 — W2a `common.DoWithBackoff`가 5xx·429에 backoff 적용. polling 주기 자체의 2배 확대는 W3 (관측 수치 확보 후) |
| §10-3 5년 백필 | ✅ Task 13 (KOSPI·KOSDAQ·NASDAQ) |
| §10-3 COPY 적재 | ✅ W2a `ingest.UpsertPrices`가 chunked COPY |
| §10-8 일일 브리핑 분산 | ⚠️ 의도된 비범위 — W4 (AI 채팅 작업) |
| §10-9 alias 학습 | ✅ Task 10 `/v1/instruments/select` |
| §10-9 시드 단계 alias (한글명·티커 자동 등록) | ✅ Task 4 `seedKRAliases` |

### Placeholder 스캔

- 모든 Task에 실제 파일 경로 + 코드 블록 + 명령어 명시.
- Task 13 NASDAQ 시드는 명시적 30종목 리스트 (placeholder 아님).
- Task 14 검증의 `<paste-access-token>`은 절차 설명용 — Step 5에서 분명히 표시.

### 타입 일관성

- `models.Instrument`, `models.PriceBar`, `models.Quote`, `models.Indicator`, `models.FXRate` — W2a 정의 그대로 참조.
- `schedule.Deps`의 필드 5개(`KIND`/`Yahoo`/`FX`/`FRED`/`ECOS`) — main.go가 동일 시그니처로 주입.
- `handlers.TickerItem`, `handlers.SearchResult` — repo 인터페이스가 동일 타입 반환.

### 의도된 비범위 (W3+ 이관)

- holdings + watchlist union polling (현재 INDEX만)
- US 종목 마스터 갱신 잡 (현재 lazy, backfill에서 시드만)
- 잡 누적 실패 시 Resend 알림 (W5 관측)
- `quotes` polling 주기의 동적 백오프 (W3+)
- 일일 브리핑 분산 워커 (W4)
- `instruments.market` 컬럼 (KOSPI/KOSDAQ 명시 구분 — 현재 .KS 기본 fallback)
- 검색 텍스트 fuzzy 인덱스 (pg_trgm) — spec §10 Minor 비범위

### 알려진 한계

1. **schedule jobs의 통합 테스트 미작성**: testcontainers로 가능하나 외부 API 호출 mock 비용이 큼. unit-level handler 테스트로 대체. 잡 실제 동작은 Task 14 수동 검증으로 확인.
2. **KR 종목 KOSPI/KOSDAQ 구분 부재**: KIND 어댑터가 `exchange='KRX'` 단일 적재. cron `JobUpdateKRPrices`는 `.KS` 기본 → KOSDAQ 종목 실패. backfill CLI는 `-market` 인자로 명시 구분 (정확). W3에서 instruments에 `market` 컬럼 추가 검토.
3. **DST 미반영**: US 장중 판정이 KST 23:30~06:00 고정. 서머타임 기간(미국 일광절약시간) 30분 어긋남. MVP 허용.
4. **NY Friday 후반 세션 누락**: `IsUSMarketOpen`이 토요일을 일괄 false로 처리. 실제로는 NY 금요일 정규장 후반(KST 토요일 새벽 ~06:00)도 false가 되어 quotes 잡이 해당 시간대 폴링 skip. 일봉(prices)은 06:00 cron이 별도 처리하므로 데이터 손실 없음. W3에서 holdings polling 도입 시 재검토.

## 검토 이력

- 2026-05-22 v1 작성. W2a 종료 후 W2b 분리 — cron + API + 백필. KRX 직접 호출 불가 결정 반영 (KR quotes 모두 Yahoo).
- 2026-05-22 v1 subagent 자체 검토 (general-purpose) → NEEDS PATCHES (Critical 3 + Important 8 + Minor 6).
- 2026-05-22 v2 패치 적용:
  - **Critical 1** [Task 5·6 SQL]: `select id::text` 명시 캐스트로 pgx v5 uuid→string scan 안전.
  - **Critical 2** [Task 3 cron.go]: `cron.WithChain(cron.SkipIfStillRunning(...))` 추가 — 잡 중복 실행 방지.
  - **Critical 3** [Task 7 previousRates]: 휴일 fallback(`distinct on (base,quote) ... order by observed_at desc`)으로 토·일·공휴일 정상 동작. 첫 배포 시 change_pct=0이 의도임을 plan에 명시.
  - **Important 1** [Task 11 main.go]: `cronWorker.Stop()` 직후 `cancel()` 호출 — 진행 중 잡의 외부 API ctx cancel 전파.
  - **Important 6** [Task 9~12 재구성]: 원 Task 9(main.go) 삭제. Task 9·10 = handlers, Task 11 = router + main.go 단일 task·단일 커밋. atomic 원칙 준수. 후속 Task 11→10, 12→11, 13→12, 14→13, 15→14 재번호.
  - **§10-9 시드 alias 미등록 갭** [Task 4]: `seedKRAliases` helper 추가 — KIND 종목 마스터 적재 후 종목코드·한글명을 alias로 자동 등록. "삼성전자" 검색 즉시 매칭.
  - **Important 7** [Task 12 (TopTicker)]: Next.js 16 가이드 확인 Step 0 명시. `document.visibilityState === 'hidden'` 시 polling skip.
  - **Important 8** [Task 14 검증]: 테스트 user 생성 절차(검증 5) 추가. curl로 access_token 직접 발급 옵션 포함.
- 다음: 사용자 보고 → 승인 → subagent-driven-development로 실행.
