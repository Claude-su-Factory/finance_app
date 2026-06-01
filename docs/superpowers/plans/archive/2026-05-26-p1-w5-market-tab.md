# Quotient W5 — 마켓 탭 · 차트 · Watchlist UI · AdSlot

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development` (권장) 또는 `superpowers:executing-plans`. 체크박스(`- [ ]`) 단위 추적.

**Goal:** 마켓 탭(`/app/market`) 신설 — KR/US 지수·환율·경제 지표 카드(라인 차트 포함) + 관심 종목 추가/제거 UI. 동시에 포트폴리오 보유 자산 테이블에 7일 미니 스파크라인을 추가하고, AdSense 활성화 전까지 비활성 상태로 표시할 `<AdSlot>` 추상화를 도입한다. W5 종료 시점: 사용자가 마켓 탭에서 6개 카드를 보고 종목을 watchlist에 추가/제거하며, 포트폴리오 테이블 각 행에 미니 스파크라인이 표시된다.

**Architecture:**
- **차트 라이브러리**: `recharts` (React 표준, SVG 기반, ~80KB gzipped). 라인·도넛·미니 스파크라인 모두 동일 라이브러리. dynamic import로 마켓·포트폴리오 페이지에서만 로드.
- **가격 history API**: `GET /v1/prices/history?symbol=&range=` (1w·1mo·6mo·1y·5y) + `GET /v1/indicators/history?code=` (경제 지표 시계열) + `GET /v1/fx/history?base=&quote=` (환율 시계열, `public.fx_rates`). 세 API 모두 RLS 우회 superuser pool — 마켓 데이터는 공개. 단 인증 사용자만 호출 가능 (RequireAuth 미들웨어 통과).
- **마켓 탭 페이지 구조**: `/app/market` 단일 페이지에 6개 카드 그리드 (KR 지수·US 지수·환율·경제 지표·관심 종목·AdSlot). 각 카드는 독립 client component — 자체 데이터 fetch + 차트 렌더.
- **Watchlist 추가/제거 UI**: 마켓 탭 관심 종목 카드 안에 `InstrumentSearchInput` 재사용(W3-T11) + watchlist API(W3-T9). 카드 내 검색 입력 + 추가 + 행별 × 버튼.
- **포트폴리오 스파크라인**: HoldingsTable의 각 행에 마지막 컬럼 직전에 sparkline 셀 추가. 7일 가격 데이터를 batch fetch (한 번에 모든 instrument_id의 7일 history) — 사용자별 holdings 수만큼 개별 호출 회피.
- **AdSlot**: `<AdSlot slot="market_top" />` 같은 의미 슬롯명을 받고, `process.env.NEXT_PUBLIC_ENABLE_ADS`가 `true`가 아니면 회색 placeholder "ADS_DISABLED" 표시. AdSense 활성 시 슬롯명으로 `data-ad-slot` 매핑하는 hook만 향후 추가.

**Tech Stack:** Go 1.25 + chi v5 + pgx v5 + Next.js 16 + Tailwind v4 + **recharts 2.x** (신규).

**참고 스펙:** [`2026-05-22-quotient-mvp-design.md`](../specs/2026-05-22-quotient-mvp-design.md) §4(데이터 수집·범위) · §6(정보 구조 — 마켓 탭 컴포넌트·포트폴리오 미니 차트) · §8(광고 슬롯 설계). [`W3 plan`](2026-05-23-p1-w3-portfolio-watchlist-home.md) (InstrumentSearchInput 재사용·포트폴리오 행 패턴).

> **Next.js 16 주의**: `apps/web/AGENTS.md` — recharts는 client component 전용. 마켓 탭의 카드들은 모두 `"use client"`. dynamic import로 SSR 단계에서 로드 회피 → 초기 페이지 번들 최소화.

---

## 외부 셋업 (Task 0)

- 추가 외부 키 없음. AdSense 가입은 Phase 2 (가입자 100명 + 일평균 PV 500 도달 시) 별도 작업.
- recharts npm 패키지 추가 (Task 2 step에 포함).

---

## File Structure (W5 생성·수정)

```
apps/api/
├── internal/
│   ├── handlers/
│   │   ├── history.go                        (신규) /v1/prices/history · /v1/indicators/history · /v1/fx/history
│   │   ├── history_repo_pg.go                (신규)
│   │   └── history_test.go                   (신규)
│   └── router/router.go                      (수정) 라우트 3개 추가

apps/web/
├── package.json                              (수정) recharts 추가
├── lib/api/
│   └── history.ts                            (신규) 3 종 history fetch
├── components/
│   ├── charts/
│   │   ├── LineChartCard.tsx                 (신규) 공통 라인 차트 카드
│   │   ├── Sparkline.tsx                     (신규) 미니 스파크라인 (mini 모드 전용)
│   │   └── chart-tokens.ts                   (신규) 색상·포맷 토큰
│   ├── market/
│   │   ├── KRIndicesCard.tsx                 (신규)
│   │   ├── USIndicesCard.tsx                 (신규)
│   │   ├── FxCard.tsx                        (신규)
│   │   ├── IndicatorsCard.tsx                (신규)
│   │   └── WatchlistEditorCard.tsx           (신규) 추가/제거 + 미니 차트
│   ├── ads/
│   │   └── AdSlot.tsx                        (신규)
│   └── portfolio/
│       └── HoldingsTable.tsx                 (수정) 행에 sparkline 셀 추가
└── app/app/
    └── market/
        └── page.tsx                          (신규) 마켓 탭 라우트
```

API 엔드포인트 (3개):
- `GET /v1/prices/history?symbol=<sym>&range=<r>` → `{ symbol, range, points: [{date, close}], count }`
- `GET /v1/prices/history?ids=<id1,id2,...>&range=7d` → batch 모드. `{ items: { [instrument_id]: [{date, close}] } }`. 포트폴리오 스파크라인용
- `GET /v1/indicators/history?code=<code>&days=<n>` → `{ code, points: [{observed_at, value}] }`
- `GET /v1/fx/history?base=<b>&quote=<q>&days=<n>` → `{ base, quote, points: [{observed_at, rate}] }`

range enum: `1w` | `1mo` | `6mo` | `1y` | `5y` (스펙 §5의 get_price_history와 정합).

---

## Task 1: 백엔드 history API (3 라우트 + repo + 테스트)

**Files:**
- Create: `apps/api/internal/handlers/history_repo_pg.go`
- Create: `apps/api/internal/handlers/history.go`
- Create: `apps/api/internal/handlers/history_test.go`
- Modify: `apps/api/internal/router/router.go` (시그니처 + 라우트 3개)
- Modify: `apps/api/cmd/server/main.go` (handler 와이어링)

### Step 1: history_repo_pg.go

```go
package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PricePoint struct {
	Date  string  `json:"date"`  // YYYY-MM-DD
	Close float64 `json:"close"`
}

type IndicatorPoint struct {
	ObservedAt string  `json:"observed_at"`
	Value      float64 `json:"value"`
}

type FxPoint struct {
	ObservedAt string  `json:"observed_at"`
	Rate       float64 `json:"rate"`
}

type HistoryRepo interface {
	PriceBySymbol(ctx context.Context, symbol, rng string) ([]PricePoint, error)
	PriceByIDsBatch(ctx context.Context, ids []string, rng string) (map[string][]PricePoint, error)
	Indicator(ctx context.Context, code string, days int) ([]IndicatorPoint, error)
	Fx(ctx context.Context, base, quote string, days int) ([]FxPoint, error)
}

type PgHistoryRepo struct {
	pool *pgxpool.Pool
}

func NewPgHistoryRepo(pool *pgxpool.Pool) *PgHistoryRepo {
	return &PgHistoryRepo{pool: pool}
}

// rangeToInterval maps the public range enum to a Postgres interval string.
// Unknown values return "" — caller treats as 400.
func rangeToInterval(rng string) string {
	switch rng {
	case "1w":
		return "7 days"
	case "1mo":
		return "30 days"
	case "6mo":
		return "180 days"
	case "1y":
		return "365 days"
	case "5y":
		return "1825 days"
	}
	return ""
}

func (r *PgHistoryRepo) PriceBySymbol(ctx context.Context, symbol, rng string) ([]PricePoint, error) {
	interval := rangeToInterval(rng)
	if interval == "" {
		return nil, fmt.Errorf("invalid range")
	}
	// p.date는 DATE 타입이므로 명시적 DATE cast로 비교 (timezone 영향 회피)
	rows, err := r.pool.Query(ctx, `
		select to_char(p.date, 'YYYY-MM-DD'), p.close::float8
		from public.prices p
		join public.instruments i on i.id = p.instrument_id
		where i.symbol = $1 and i.is_active = true
		  and p.date >= (current_date - $2::interval)::date
		order by p.date
	`, symbol, interval)
	if err != nil {
		return nil, err
	}
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

// PriceByIDsBatch returns last `rng` days of close prices for each instrument_id.
// Designed for portfolio sparklines — single SQL round trip for N instruments.
func (r *PgHistoryRepo) PriceByIDsBatch(ctx context.Context, ids []string, rng string) (map[string][]PricePoint, error) {
	interval := rangeToInterval(rng)
	if interval == "" {
		return nil, fmt.Errorf("invalid range")
	}
	if len(ids) == 0 {
		return map[string][]PricePoint{}, nil
	}
	// UUID 직접 비교로 prices_date_idx(instrument_id, date desc) 인덱스 활용
	rows, err := r.pool.Query(ctx, `
		select p.instrument_id::text, to_char(p.date, 'YYYY-MM-DD'), p.close::float8
		from public.prices p
		where p.instrument_id = any($1::uuid[])
		  and p.date >= (current_date - $2::interval)::date
		order by p.instrument_id, p.date
	`, ids, interval)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]PricePoint{}
	for rows.Next() {
		var iid string
		var p PricePoint
		if err := rows.Scan(&iid, &p.Date, &p.Close); err != nil {
			return nil, err
		}
		out[iid] = append(out[iid], p)
	}
	return out, rows.Err()
}

func (r *PgHistoryRepo) Indicator(ctx context.Context, code string, days int) ([]IndicatorPoint, error) {
	if days <= 0 || days > 3650 {
		days = 90
	}
	// observed_at은 timestamptz. now() - 일 단위 interval 명시 (timezone 안전)
	rows, err := r.pool.Query(ctx, `
		select to_char(observed_at, 'YYYY-MM-DD'), value::float8
		from public.economic_indicators
		where code = $1
		  and observed_at >= now() - ($2::int * interval '1 day')
		order by observed_at
	`, code, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []IndicatorPoint
	for rows.Next() {
		var p IndicatorPoint
		if err := rows.Scan(&p.ObservedAt, &p.Value); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PgHistoryRepo) Fx(ctx context.Context, base, quote string, days int) ([]FxPoint, error) {
	if days <= 0 || days > 3650 {
		days = 30
	}
	base = strings.ToUpper(base)
	quote = strings.ToUpper(quote)
	rows, err := r.pool.Query(ctx, `
		select to_char(observed_at, 'YYYY-MM-DD'), rate::float8
		from public.fx_rates
		where base = $1 and quote = $2
		  and observed_at >= now() - ($3::int * interval '1 day')
		order by observed_at
	`, base, quote, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FxPoint
	for rows.Next() {
		var p FxPoint
		if err := rows.Scan(&p.ObservedAt, &p.Rate); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

```

### Step 2: history.go (핸들러 3개)

```go
package handlers

import (
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

type HistoryHandler struct {
	repo HistoryRepo
}

func NewHistoryHandler(repo HistoryRepo) *HistoryHandler {
	return &HistoryHandler{repo: repo}
}

// GET /v1/prices/history?symbol=<sym>&range=<r>
// GET /v1/prices/history?ids=<id1,id2>&range=<r>  (batch)
func (h *HistoryHandler) Prices(w http.ResponseWriter, r *http.Request) {
	rng := r.URL.Query().Get("range")
	if rng == "" {
		rng = "1mo"
	}
	if rangeToInterval(rng) == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "invalid range")
		return
	}

	// batch mode
	if idsStr := r.URL.Query().Get("ids"); idsStr != "" {
		ids := splitCSV(idsStr)
		if len(ids) > 100 {
			writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "too many ids (max 100)")
			return
		}
		items, err := h.repo.PriceByIDsBatch(r.Context(), ids, rng)
		if err != nil {
			slog.Error("price batch failed", "err", err)
			writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"items": items, "range": rng})
		return
	}

	sym := r.URL.Query().Get("symbol")
	if sym == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "symbol or ids required")
		return
	}
	points, err := h.repo.PriceBySymbol(r.Context(), sym, rng)
	if err != nil {
		slog.Error("price history failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if points == nil {
		points = []PricePoint{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"symbol": sym, "range": rng, "points": points, "count": len(points)})
}

// GET /v1/indicators/history?code=<code>&days=<n>
func (h *HistoryHandler) Indicators(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "code required")
		return
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	points, err := h.repo.Indicator(r.Context(), code, days)
	if err != nil {
		slog.Error("indicator history failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if points == nil {
		points = []IndicatorPoint{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"code": code, "points": points, "count": len(points)})
}

// GET /v1/fx/history?base=<b>&quote=<q>&days=<n>
func (h *HistoryHandler) Fx(w http.ResponseWriter, r *http.Request) {
	base := r.URL.Query().Get("base")
	quote := r.URL.Query().Get("quote")
	if base == "" || quote == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "base and quote required")
		return
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	points, err := h.repo.Fx(r.Context(), base, quote, days)
	if err != nil {
		slog.Error("fx history failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if points == nil {
		points = []FxPoint{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"base": base, "quote": quote, "points": points, "count": len(points)})
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
```

### Step 3: history_test.go (검증 단위 테스트 — repo는 통합 테스트로 별도)

```go
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quotient/quotient/apps/api/internal/middleware"
)

type fakeHistoryRepo struct {
	prices      []PricePoint
	batch       map[string][]PricePoint
	indicators  []IndicatorPoint
	fxPoints    []FxPoint
	err         error
}

func (f *fakeHistoryRepo) PriceBySymbol(ctx context.Context, symbol, rng string) ([]PricePoint, error) {
	return f.prices, f.err
}
func (f *fakeHistoryRepo) PriceByIDsBatch(ctx context.Context, ids []string, rng string) (map[string][]PricePoint, error) {
	return f.batch, f.err
}
func (f *fakeHistoryRepo) Indicator(ctx context.Context, code string, days int) ([]IndicatorPoint, error) {
	return f.indicators, f.err
}
func (f *fakeHistoryRepo) Fx(ctx context.Context, base, quote string, days int) ([]FxPoint, error) {
	return f.fxPoints, f.err
}

func historyReq(method, target string) *http.Request {
	r := httptest.NewRequest(method, target, nil)
	r = r.WithContext(middleware.WithUserID(r.Context(), "user-1"))
	return r
}

func TestPrices_InvalidRange(t *testing.T) {
	h := NewHistoryHandler(&fakeHistoryRepo{})
	w := httptest.NewRecorder()
	h.Prices(w, historyReq(http.MethodGet, "/v1/prices/history?symbol=KOSPI&range=invalid"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("got %d want 422", w.Code)
	}
}

func TestPrices_MissingSymbol(t *testing.T) {
	h := NewHistoryHandler(&fakeHistoryRepo{})
	w := httptest.NewRecorder()
	h.Prices(w, historyReq(http.MethodGet, "/v1/prices/history"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("got %d want 422", w.Code)
	}
}

func TestPrices_SingleSymbol(t *testing.T) {
	repo := &fakeHistoryRepo{prices: []PricePoint{{Date: "2026-05-20", Close: 100}}}
	h := NewHistoryHandler(repo)
	w := httptest.NewRecorder()
	h.Prices(w, historyReq(http.MethodGet, "/v1/prices/history?symbol=KOSPI&range=1w"))
	if w.Code != http.StatusOK {
		t.Fatalf("got %d, body=%s", w.Code, w.Body.String())
	}
	var got struct {
		Symbol string       `json:"symbol"`
		Points []PricePoint `json:"points"`
		Count  int          `json:"count"`
	}
	_ = json.NewDecoder(w.Body).Decode(&got)
	if got.Symbol != "KOSPI" || got.Count != 1 {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestPrices_BatchMode(t *testing.T) {
	repo := &fakeHistoryRepo{batch: map[string][]PricePoint{
		"id-1": {{Date: "2026-05-20", Close: 100}},
		"id-2": {{Date: "2026-05-20", Close: 200}},
	}}
	h := NewHistoryHandler(repo)
	w := httptest.NewRecorder()
	h.Prices(w, historyReq(http.MethodGet, "/v1/prices/history?ids=id-1,id-2&range=1w"))
	if w.Code != http.StatusOK {
		t.Fatalf("got %d", w.Code)
	}
	var got struct {
		Items map[string][]PricePoint `json:"items"`
	}
	_ = json.NewDecoder(w.Body).Decode(&got)
	if len(got.Items) != 2 {
		t.Errorf("got %d items, want 2", len(got.Items))
	}
}

func TestPrices_BatchTooMany(t *testing.T) {
	h := NewHistoryHandler(&fakeHistoryRepo{})
	ids := ""
	for i := 0; i < 101; i++ {
		if i > 0 {
			ids += ","
		}
		ids += "id"
	}
	w := httptest.NewRecorder()
	h.Prices(w, historyReq(http.MethodGet, "/v1/prices/history?ids="+ids+"&range=1w"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("got %d want 422", w.Code)
	}
}

func TestIndicators_MissingCode(t *testing.T) {
	h := NewHistoryHandler(&fakeHistoryRepo{})
	w := httptest.NewRecorder()
	h.Indicators(w, historyReq(http.MethodGet, "/v1/indicators/history"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("got %d want 422", w.Code)
	}
}

func TestFx_MissingPair(t *testing.T) {
	h := NewHistoryHandler(&fakeHistoryRepo{})
	w := httptest.NewRecorder()
	h.Fx(w, historyReq(http.MethodGet, "/v1/fx/history?base=USD"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("got %d want 422", w.Code)
	}
}
```

### Step 4: router.go에 추가

`router.New(...)` 시그니처의 마지막 핸들러(`briefingHandler`)와 `readyz` 사이에 `historyHandler *handlers.HistoryHandler`를 삽입. 명시적 코드 블록:

```go
func New(
	verifier *auth.Verifier,
	corsOrigin string,
	profileHandler *handlers.ProfileHandler,
	marketHandler *handlers.MarketHandler,
	instrumentHandler *handlers.InstrumentHandler,
	holdingHandler *handlers.HoldingHandler,
	watchlistHandler *handlers.WatchlistHandler,
	chatHandler *handlers.ChatHandler,
	briefingHandler *handlers.BriefingHandler,
	historyHandler *handlers.HistoryHandler, // 신규
	readyz http.HandlerFunc,
) *chi.Mux {
	// ...기존 셋업 그대로...
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(verifier))
		// ...기존 라우트 유지...
		r.Get("/v1/briefings/today", briefingHandler.Today)
		// 신규 (briefing 라우트 다음에 추가)
		r.Get("/v1/prices/history", historyHandler.Prices)
		r.Get("/v1/indicators/history", historyHandler.Indicators)
		r.Get("/v1/fx/history", historyHandler.Fx)
	})
	return r
}
```

### Step 5: main.go 와이어링

기존 `briefingHandler` 정의 다음에 추가:
```go
historyRepo := handlers.NewPgHistoryRepo(pool)
historyHandler := handlers.NewHistoryHandler(historyRepo)
```

`router.New` 호출 마지막에 `historyHandler` 추가 (`briefingHandler`와 `readyz` 사이):
```go
Handler: router.New(
    verifier, cfg.CORSOrigin,
    profileHandler, marketHandler, instrumentHandler,
    holdingHandler, watchlistHandler,
    chatHandler, briefingHandler,
    historyHandler,
    readyz,
),
```

### Step 6: 빌드·테스트

Run: `cd apps/api && go build ./... && go test ./internal/handlers/ -run "Prices|Indicators|Fx" -v`
Expected: 7 PASS.

Run: `cd apps/api && go test ./...`
Expected: 회귀 없음.

### Step 7: 커밋

```bash
git add apps/api/internal/handlers/history_repo_pg.go apps/api/internal/handlers/history.go apps/api/internal/handlers/history_test.go apps/api/internal/router/router.go apps/api/cmd/server/main.go
git commit -m "feat(api): history API (prices·indicators·fx) + 라우트 3개 + 7 테스트"
```

---

## Task 1.5: 인덱스 일봉 데이터 가용성 확보 (cron 확장)

**Files:**
- Modify: `apps/api/internal/schedule/jobs_prices.go`
- Modify: `apps/api/internal/schedule/yahoo_symbols.go` (필요 시 helper 추가)

**문제**: 현재 `JobUpdateKRPrices`는 exchange가 `KOSPI`/`KOSDAQ`인 종목(KIND 적재 종목 마스터)만 처리. `JobUpdateUSPrices`는 `NYSE`/`NASDAQ`/`AMEX`만. 그러나 W2a 시드의 KOSPI/KOSDAQ/SPX/NDX 인덱스는 exchange가 `KRX-IDX`/`NYSE-IDX`/`NASDAQ-IDX`로 매칭에서 제외 → **prices 테이블에 인덱스 일봉이 영구 비어있음**. W5 마켓 탭의 KR/US 지수 카드가 항상 "데이터 없음" 표시.

**해결**: `JobUpdateKRPrices`·`JobUpdateUSPrices`의 SQL where절 + Yahoo 심볼 매핑에 `*-IDX` 케이스 추가. cron이 매일 일봉을 Yahoo `^KS11`/`^KQ11`/`^GSPC`/`^NDX`에서 fetch하여 인덱스에도 적재.

### Step 1: jobs_prices.go의 KR/US prices 함수 확장

기존 `JobUpdateKRPrices`/`JobUpdateUSPrices`를 읽어 다음 변경:

1. 종목 조회 where절에 IDX exchange 포함:
   - KR: `where exchange in ('KOSPI', 'KOSDAQ', 'KRX-IDX')`
   - US: `where exchange in ('NYSE', 'NASDAQ', 'AMEX', 'NYSE-IDX', 'NASDAQ-IDX')`
2. Yahoo 심볼 매핑은 기존 `StockYahooSymbol`/`IndexYahooSymbol` 둘 다 시도:
   ```go
   ysym := IndexYahooSymbol(symbol, exchange)
   if ysym == "" {
       ysym = StockYahooSymbol(symbol, exchange)
   }
   if ysym == "" { continue }
   ```

> 실 구현 위치는 `jobs_prices.go`의 종목 페치 루프 안 — 함수 전체 재작성보다 SQL where 한 줄과 심볼 분기 한 블록만 수정.

### Step 2: 빌드·테스트

Run: `cd apps/api && go build ./internal/schedule/... && go test ./internal/schedule/...`
Expected: 빌드 + 기존 테스트 PASS.

### Step 3: 수동 백필 (검증용, 별도 PR로 자동화 권장)

dev 환경에서 cron이 한 번 돌면 다음 영업일에 인덱스 일봉이 적재된다. 즉시 검증하려면 한 번 직접 호출:

```bash
# psql 또는 supabase studio에서 yahoo 호출 결과를 prices에 1개 행만 임시 삽입하여 차트 fallback 확인
docker exec supabase_db_finance psql -U postgres -d postgres -c \
  "insert into public.prices (instrument_id, date, open, high, low, close, volume) \
   select id, current_date - 1, 2500, 2520, 2480, 2510, 0 from public.instruments where symbol = 'KOSPI' and exchange = 'KRX-IDX' \
   on conflict do nothing;"
```

이 한 행은 SmokeTest용. 다음 cron 실행 시 자동 적재 시작.

### Step 4: 커밋

```bash
git add apps/api/internal/schedule/jobs_prices.go apps/api/internal/schedule/yahoo_symbols.go
git commit -m "feat(api): cron KR·US prices에 *-IDX exchange 포함 (인덱스 일봉 적재)"
```

## Self-Review 항목
- KR/US prices SQL exchange 매칭에 IDX 추가
- IndexYahooSymbol fallback 분기
- 기존 종목 일봉 회귀 없음

---

## Task 2: recharts 추가 + 공통 차트 컴포넌트

**Files:**
- Modify: `apps/web/package.json` (recharts 추가)
- Create: `apps/web/components/charts/chart-tokens.ts`
- Create: `apps/web/components/charts/LineChartCard.tsx`
- Create: `apps/web/components/charts/Sparkline.tsx`

### Step 1: recharts 설치

Run: `cd apps/web && npm install recharts@^2.15.0`
Expected: package.json + package-lock.json에 추가.

### Step 2: chart-tokens.ts

```ts
// 차트 색상 토큰 — Tailwind 토큰과 정합.
export const CHART_COLORS = {
  up: "#00FF7F",     // bb-up
  down: "#FF3344",   // bb-down
  accent: "#00FFFF", // bb-accent (cyan)
  warn: "#FFD500",
  muted: "#666666",
  line: "#1a1a1a",
} as const;

// 단순 양음 결정 — 첫·마지막 비교.
export function trendColor(points: { value: number }[]): string {
  if (points.length < 2) return CHART_COLORS.muted;
  return points[points.length - 1].value >= points[0].value
    ? CHART_COLORS.up
    : CHART_COLORS.down;
}
```

### Step 3: LineChartCard.tsx (마켓 탭 카드 공통 라인 차트)

```tsx
"use client";

import dynamic from "next/dynamic";
import { CHART_COLORS, trendColor } from "./chart-tokens";

// recharts를 dynamic import — 페이지 초기 번들에서 제외 (SSR 비활성).
const ResponsiveContainer = dynamic(
  () => import("recharts").then((m) => m.ResponsiveContainer),
  { ssr: false },
);
const LineChart = dynamic(() => import("recharts").then((m) => m.LineChart), { ssr: false });
const Line = dynamic(() => import("recharts").then((m) => m.Line), { ssr: false });
const XAxis = dynamic(() => import("recharts").then((m) => m.XAxis), { ssr: false });
const YAxis = dynamic(() => import("recharts").then((m) => m.YAxis), { ssr: false });
const Tooltip = dynamic(() => import("recharts").then((m) => m.Tooltip), { ssr: false });

export type ChartPoint = { x: string; value: number };

export function LineChartCard({
  title,
  subtitle,
  current,
  changePct,
  points,
  height = 160,
  unit,
}: {
  title: string;
  subtitle?: string;
  current?: number;
  changePct?: number;
  points: ChartPoint[];
  height?: number;
  unit?: string;
}) {
  const color = trendColor(points);
  const positive = (changePct ?? 0) >= 0;
  return (
    <div className="border border-line p-4">
      <div className="flex items-baseline justify-between mb-2">
        <div>
          <div className="font-mono text-sm">{title}</div>
          {subtitle && <div className="text-xs text-fg-muted font-mono">{subtitle}</div>}
        </div>
        {current !== undefined && (
          <div className="text-right">
            <div className="font-mono text-base tabular-nums">
              {current.toLocaleString()}
              {unit ? ` ${unit}` : ""}
            </div>
            {changePct !== undefined && (
              <div className={`text-xs font-mono tabular-nums ${positive ? "text-bb-up" : "text-bb-down"}`}>
                {positive ? "+" : ""}{changePct.toFixed(2)}%
              </div>
            )}
          </div>
        )}
      </div>
      <div style={{ width: "100%", height }}>
        {points.length === 0 ? (
          <div className="h-full flex items-center justify-center text-xs text-fg-muted font-mono">
            데이터 없음
          </div>
        ) : (
          <ResponsiveContainer>
            <LineChart data={points} margin={{ top: 5, right: 8, bottom: 0, left: 0 }}>
              <XAxis dataKey="x" tick={{ fontSize: 10, fill: CHART_COLORS.muted }} hide />
              <YAxis domain={["auto", "auto"]} tick={{ fontSize: 10, fill: CHART_COLORS.muted }} hide />
              <Tooltip
                contentStyle={{ background: "#0a0a0a", border: `1px solid ${CHART_COLORS.line}`, fontSize: 11, fontFamily: "monospace" }}
                labelStyle={{ color: CHART_COLORS.muted }}
                itemStyle={{ color }}
                formatter={(v: number) => v.toLocaleString()}
              />
              <Line type="monotone" dataKey="value" stroke={color} strokeWidth={1.5} dot={false} isAnimationActive={false} />
            </LineChart>
          </ResponsiveContainer>
        )}
      </div>
    </div>
  );
}
```

### Step 4: Sparkline.tsx (포트폴리오 행 미니)

```tsx
"use client";

import dynamic from "next/dynamic";
import { trendColor } from "./chart-tokens";

const ResponsiveContainer = dynamic(
  () => import("recharts").then((m) => m.ResponsiveContainer),
  { ssr: false },
);
const LineChart = dynamic(() => import("recharts").then((m) => m.LineChart), { ssr: false });
const Line = dynamic(() => import("recharts").then((m) => m.Line), { ssr: false });

export function Sparkline({
  points,
  width = 80,
  height = 24,
}: {
  points: { value: number }[];
  width?: number;
  height?: number;
}) {
  if (points.length < 2) {
    return <span className="inline-block text-fg-muted text-xs" style={{ width, height }}>—</span>;
  }
  const color = trendColor(points);
  return (
    <span style={{ display: "inline-block", width, height }}>
      <ResponsiveContainer>
        <LineChart data={points} margin={{ top: 2, right: 2, bottom: 2, left: 2 }}>
          <Line type="monotone" dataKey="value" stroke={color} strokeWidth={1} dot={false} isAnimationActive={false} />
        </LineChart>
      </ResponsiveContainer>
    </span>
  );
}
```

### Step 5: tsc + 빌드

Run: `cd apps/web && npx tsc --noEmit`
Expected: 에러 없음.

### Step 6: 커밋

```bash
git add apps/web/package.json apps/web/package-lock.json apps/web/components/charts/
git commit -m "feat(web): recharts 도입 + LineChartCard·Sparkline 공통 컴포넌트"
```

## Context

- recharts는 `"use client"` + dynamic import 패턴 필수 (Next.js 16 SSR에서 window 의존 회피).
- `bb-up`/`bb-down`/`bb-accent`는 W1 Tailwind 토큰 — 차트는 hex 직접 사용 (recharts가 Tailwind class 인식 못 함).

## Self-Review 항목
- recharts dynamic import (ssr: false)
- trendColor: 빈 데이터/단일 포인트 fallback
- LineChartCard 빈 상태 "데이터 없음"
- Sparkline 2 미만 시 "—"
- tsc PASS

---

## Task 3: lib/api/history.ts (3 종 fetch + batch)

**Files:**
- Create: `apps/web/lib/api/history.ts`

### Step 1: history.ts

```ts
import { authFetch, readError } from "./auth-fetch";

export type PricePoint = { date: string; close: number };
export type IndicatorPoint = { observed_at: string; value: number };
export type FxPoint = { observed_at: string; rate: number };

export type Range = "1w" | "1mo" | "6mo" | "1y" | "5y";

export async function fetchPriceHistory(symbol: string, range: Range): Promise<PricePoint[]> {
  const res = await authFetch(`/v1/prices/history?symbol=${encodeURIComponent(symbol)}&range=${range}`);
  if (!res.ok) throw await readError(res);
  const data = await res.json();
  return data.points ?? [];
}

export async function fetchPriceHistoryBatch(
  ids: string[],
  range: Range = "1w",
): Promise<Record<string, PricePoint[]>> {
  if (ids.length === 0) return {};
  const res = await authFetch(`/v1/prices/history?ids=${ids.join(",")}&range=${range}`);
  if (!res.ok) throw await readError(res);
  const data = await res.json();
  return data.items ?? {};
}

export async function fetchIndicatorHistory(code: string, days = 90): Promise<IndicatorPoint[]> {
  const res = await authFetch(`/v1/indicators/history?code=${encodeURIComponent(code)}&days=${days}`);
  if (!res.ok) throw await readError(res);
  const data = await res.json();
  return data.points ?? [];
}

export async function fetchFxHistory(base: string, quote: string, days = 30): Promise<FxPoint[]> {
  const res = await authFetch(`/v1/fx/history?base=${base}&quote=${quote}&days=${days}`);
  if (!res.ok) throw await readError(res);
  const data = await res.json();
  return data.points ?? [];
}
```

### Step 2: tsc + 커밋

Run: `cd apps/web && npx tsc --noEmit`
Expected: 에러 없음.

```bash
git add apps/web/lib/api/history.ts
git commit -m "feat(web): history API 클라이언트 (prices·batch·indicators·fx)"
```

---

## Task 4: AdSlot 컴포넌트

**Files:**
- Create: `apps/web/components/ads/AdSlot.tsx`

스펙 §8: `ENABLE_ADS=false`면 자체 메시지. AdSense 활성 시 슬롯명으로 매핑 후속 작업.

### Step 1: AdSlot.tsx

```tsx
"use client";

const ADS_ENABLED = process.env.NEXT_PUBLIC_ENABLE_ADS === "true";

// AdSlot은 의미 슬롯명을 받고, 비활성 상태면 자체 placeholder를 표시.
// Phase 2에서 AdSense 가입 후 slot → data-ad-slot 매핑 추가.
export function AdSlot({
  slot,
  height = 90,
  label,
}: {
  slot: string;
  height?: number;
  label?: string;
}) {
  if (!ADS_ENABLED) {
    return (
      <div
        className="border border-dashed border-line/50 flex items-center justify-center text-fg-muted/60 font-mono text-xs"
        style={{ height }}
        data-ad-slot={slot}
      >
        ADS_DISABLED · {label ?? slot}
      </div>
    );
  }
  // ADS_ENABLED true일 때의 실 광고 렌더는 Phase 2에서 추가.
  return (
    <div className="border border-dashed border-line/50" style={{ height }} data-ad-slot={slot} />
  );
}
```

### Step 2: `.env.example`에 client용 변수 추가

기존 `.env.example`의 "기능 플래그" 섹션을 수정:

```
# 기능 플래그
PAYMENTS_ENABLED=false
ENABLE_ADS=false                  # 서버 측 (cron·문서 토글)
NEXT_PUBLIC_ENABLE_ADS=false      # 클라이언트 측 (AdSlot — Next.js inline)
```

> Next.js는 `NEXT_PUBLIC_` 프리픽스 없는 변수를 client bundle에 inline하지 않음 — AdSlot 활성화 시 반드시 NEXT_PUBLIC_ 키 사용.

### Step 3: tsc + 커밋

Run: `cd apps/web && npx tsc --noEmit`
Expected: 에러 없음.

```bash
git add apps/web/components/ads/AdSlot.tsx .env.example
git commit -m "feat(web): AdSlot 추상화 + NEXT_PUBLIC_ENABLE_ADS env 추가"
```

## Context

- `process.env.NEXT_PUBLIC_*`는 build time inline — runtime 변경 안 됨. dev 재시작 필요.
- 기존 `.env.example`에는 server용 `ENABLE_ADS=false`만 있었음 → client 측 토글이 동작 안 함. W5에서 정합 보정.

---

## Task 5: 마켓 탭 페이지 + 4 데이터 카드

**Files:**
- Create: `apps/web/app/app/market/page.tsx`
- Create: `apps/web/components/market/KRIndicesCard.tsx`
- Create: `apps/web/components/market/USIndicesCard.tsx`
- Create: `apps/web/components/market/FxCard.tsx`
- Create: `apps/web/components/market/IndicatorsCard.tsx`

각 카드는 자체 fetch + `LineChartCard` 렌더.

### Step 1: page.tsx

```tsx
import { KRIndicesCard } from "@/components/market/KRIndicesCard";
import { USIndicesCard } from "@/components/market/USIndicesCard";
import { FxCard } from "@/components/market/FxCard";
import { IndicatorsCard } from "@/components/market/IndicatorsCard";
import { WatchlistEditorCard } from "@/components/market/WatchlistEditorCard";
import { AdSlot } from "@/components/ads/AdSlot";

export default function MarketPage() {
  return (
    <div className="p-6 md:p-8 space-y-4">
      <header className="flex items-baseline justify-between mb-2">
        <div>
          <h1 className="font-mono text-2xl">마켓</h1>
          <p className="text-fg-muted text-sm mt-1">지수·환율·지표. 시세 지연 15분.</p>
        </div>
      </header>

      <AdSlot slot="market_top" height={72} label="market_top" />

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <KRIndicesCard />
        <USIndicesCard />
        <FxCard pair={["USD", "KRW"]} title="USD/KRW" />
        <FxCard pair={["EUR", "KRW"]} title="EUR/KRW" />
        <FxCard pair={["JPY", "KRW"]} title="JPY/KRW" />
        <IndicatorsCard code="DFF" title="Fed Funds Rate" unit="%" />
        <IndicatorsCard code="DGS10" title="US 10Y Treasury" unit="%" />
        <IndicatorsCard code="722Y001" title="BOK 기준금리" unit="%" />
        <WatchlistEditorCard />
      </div>
    </div>
  );
}
```

### Step 2: KRIndicesCard.tsx

```tsx
"use client";

import { useEffect, useState } from "react";
import { LineChartCard, type ChartPoint } from "@/components/charts/LineChartCard";
import { fetchPriceHistory } from "@/lib/api/history";
import { fetchTicker, type Ticker } from "@/lib/api/market";

export function KRIndicesCard() {
  const [kospi, setKospi] = useState<ChartPoint[]>([]);
  const [kosdaq, setKosdaq] = useState<ChartPoint[]>([]);
  const [tickers, setTickers] = useState<Record<string, Ticker>>({});

  useEffect(() => {
    (async () => {
      const [k1, k2, ts] = await Promise.all([
        fetchPriceHistory("KOSPI", "1mo").catch(() => []),
        fetchPriceHistory("KOSDAQ", "1mo").catch(() => []),
        fetchTicker().catch(() => [] as Ticker[]),
      ]);
      setKospi(k1.map((p) => ({ x: p.date, value: p.close })));
      setKosdaq(k2.map((p) => ({ x: p.date, value: p.close })));
      const m: Record<string, Ticker> = {};
      for (const t of ts) m[t.symbol] = t;
      setTickers(m);
    })();
  }, []);

  return (
    <div className="grid grid-cols-1 gap-4">
      <LineChartCard
        title="KOSPI"
        subtitle="KRX 종합지수 · 1mo"
        current={tickers.KOSPI?.price}
        changePct={tickers.KOSPI?.change_pct}
        points={kospi}
      />
      <LineChartCard
        title="KOSDAQ"
        subtitle="코스닥 종합 · 1mo"
        current={tickers.KOSDAQ?.price}
        changePct={tickers.KOSDAQ?.change_pct}
        points={kosdaq}
      />
    </div>
  );
}
```

`fetchTicker`는 W3 후속 fix에서 인자 없는 버전(`fetchTicker()`)으로 통합됨.

### Step 3: USIndicesCard.tsx

```tsx
"use client";

import { useEffect, useState } from "react";
import { LineChartCard, type ChartPoint } from "@/components/charts/LineChartCard";
import { fetchPriceHistory } from "@/lib/api/history";
import { fetchTicker, type Ticker } from "@/lib/api/market";

export function USIndicesCard() {
  const [spx, setSpx] = useState<ChartPoint[]>([]);
  const [ndx, setNdx] = useState<ChartPoint[]>([]);
  const [tickers, setTickers] = useState<Record<string, Ticker>>({});

  useEffect(() => {
    (async () => {
      const [a, b, ts] = await Promise.all([
        fetchPriceHistory("SPX", "1mo").catch(() => []),
        fetchPriceHistory("NDX", "1mo").catch(() => []),
        fetchTicker().catch(() => [] as Ticker[]),
      ]);
      setSpx(a.map((p) => ({ x: p.date, value: p.close })));
      setNdx(b.map((p) => ({ x: p.date, value: p.close })));
      const m: Record<string, Ticker> = {};
      for (const t of ts) m[t.symbol] = t;
      setTickers(m);
    })();
  }, []);

  return (
    <div className="grid grid-cols-1 gap-4">
      <LineChartCard
        title="S&P 500"
        subtitle="NYSE · 1mo"
        current={tickers.SPX?.price}
        changePct={tickers.SPX?.change_pct}
        points={spx}
      />
      <LineChartCard
        title="NASDAQ 100"
        subtitle="NDX · 1mo"
        current={tickers.NDX?.price}
        changePct={tickers.NDX?.change_pct}
        points={ndx}
      />
    </div>
  );
}
```

### Step 4: FxCard.tsx

```tsx
"use client";

import { useEffect, useState } from "react";
import { LineChartCard, type ChartPoint } from "@/components/charts/LineChartCard";
import { fetchFxHistory } from "@/lib/api/history";

export function FxCard({ pair, title }: { pair: [string, string]; title: string }) {
  const [points, setPoints] = useState<ChartPoint[]>([]);
  const [current, setCurrent] = useState<number | undefined>();
  const [changePct, setChangePct] = useState<number | undefined>();

  useEffect(() => {
    (async () => {
      const data = await fetchFxHistory(pair[0], pair[1], 30).catch(() => []);
      const mapped = data.map((p) => ({ x: p.observed_at, value: p.rate }));
      setPoints(mapped);
      if (mapped.length > 0) {
        const last = mapped[mapped.length - 1].value;
        const first = mapped[0].value;
        setCurrent(last);
        if (first > 0) setChangePct(((last - first) / first) * 100);
      }
    })();
  }, [pair]);

  return (
    <LineChartCard
      title={title}
      subtitle={`${pair[0]}→${pair[1]} · 30d`}
      current={current}
      changePct={changePct}
      points={points}
    />
  );
}
```

### Step 5: IndicatorsCard.tsx

```tsx
"use client";

import { useEffect, useState } from "react";
import { LineChartCard, type ChartPoint } from "@/components/charts/LineChartCard";
import { fetchIndicatorHistory } from "@/lib/api/history";

export function IndicatorsCard({
  code,
  title,
  unit,
}: {
  code: string;
  title: string;
  unit?: string;
}) {
  const [points, setPoints] = useState<ChartPoint[]>([]);
  const [current, setCurrent] = useState<number | undefined>();
  const [changePct, setChangePct] = useState<number | undefined>();

  useEffect(() => {
    (async () => {
      const data = await fetchIndicatorHistory(code, 90).catch(() => []);
      const mapped = data.map((p) => ({ x: p.observed_at, value: p.value }));
      setPoints(mapped);
      if (mapped.length > 0) {
        const last = mapped[mapped.length - 1].value;
        const first = mapped[0].value;
        setCurrent(last);
        if (first > 0) setChangePct(((last - first) / first) * 100);
      }
    })();
  }, [code]);

  return (
    <LineChartCard
      title={title}
      subtitle={`${code} · 90d`}
      current={current}
      changePct={changePct}
      points={points}
      unit={unit}
    />
  );
}
```

### Step 6: tsc + 빌드

Run: `cd apps/web && npx tsc --noEmit`
Expected: 에러 없음.

### Step 7: 커밋

```bash
git add apps/web/app/app/market/page.tsx apps/web/components/market/
git commit -m "feat(web): 마켓 탭 페이지 + KR/US 지수·환율·경제 지표 카드"
```

## Self-Review 항목
- 4 카드 모두 dynamic recharts import 통과
- 빈 데이터 fallback "데이터 없음"
- ticker price + change_pct 표시
- changePct 계산 (마지막/첫 비교) — fx·indicators에 적용

---

## Task 6: WatchlistEditorCard + 백엔드 asset_class 가드

**Files:**
- Modify: `apps/api/internal/handlers/watchlist.go` — Add 핸들러에 asset_class 가드 추가
- Create: `apps/web/components/market/WatchlistEditorCard.tsx`
- Modify: `apps/web/components/home/WatchlistMiniCard.tsx` — 빈 상태 카피 정정 ("마켓 탭에서 추가")

InstrumentSearchInput 재사용 + addWatchlist + removeWatchlist (모두 W3에 존재). holdings의 asset_class 가드 패턴(W3-T5)을 watchlist에도 동일 적용 — frontend·backend 이중 방어.

### Step 0: watchlist.go에 asset_class 가드 추가

기존 `Add` 핸들러의 `body.InstrumentID == ""` 검증 다음, `h.repo.Add(...)` 호출 전에 삽입:

```go
// asset_class 가드: INDEX·FX·CASH는 watchlist 대상이 아님 (W5).
// holdings.go(W3-T5)와 동일 패턴. backend·frontend 이중 방어.
var assetClass string
if err := h.pool.QueryRow(r.Context(), `select asset_class from public.instruments where id = $1`, body.InstrumentID).Scan(&assetClass); err != nil {
    writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "instrument not found")
    return
}
if assetClass == "INDEX" || assetClass == "FX" || assetClass == "CASH" {
    writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "asset_class not supported for watchlist: "+assetClass)
    return
}
```

> WatchlistHandler 구조체에 `pool *pgxpool.Pool` 필드가 없으므로 추가 필요. `NewWatchlistHandler(repo WatchlistRepo, pool *pgxpool.Pool) *WatchlistHandler` 시그니처로 확장. main.go의 `handlers.NewWatchlistHandler(watchlistRepo)` 호출도 `handlers.NewWatchlistHandler(watchlistRepo, pool)`로 갱신.

### Step 1: WatchlistEditorCard.tsx

```tsx
"use client";

import { useEffect, useState } from "react";
import { InstrumentSearchInput } from "@/components/portfolio/InstrumentSearchInput";
import { listWatchlist, addWatchlist, removeWatchlist, type WatchlistItem } from "@/lib/api/watchlist";
import type { InstrumentResult } from "@/lib/api/instruments";
import { Sparkline } from "@/components/charts/Sparkline";
import { fetchPriceHistoryBatch } from "@/lib/api/history";

export function WatchlistEditorCard() {
  const [items, setItems] = useState<WatchlistItem[]>([]);
  const [sparks, setSparks] = useState<Record<string, { value: number }[]>>({});
  const [err, setErr] = useState<string | null>(null);

  async function load() {
    try {
      const data = await listWatchlist();
      setItems(data);
      const ids = data.map((w) => w.instrument_id);
      if (ids.length > 0) {
        const batch = await fetchPriceHistoryBatch(ids, "1w").catch(() => ({}));
        const sp: Record<string, { value: number }[]> = {};
        for (const [iid, points] of Object.entries(batch)) {
          sp[iid] = points.map((p) => ({ value: p.close }));
        }
        setSparks(sp);
      }
    } catch (e: unknown) {
      setErr((e as { message?: string })?.message ?? "로드 실패");
    }
  }

  useEffect(() => { load(); }, []);

  async function handleAdd(inst: InstrumentResult) {
    if (inst.asset_class !== "KR_STOCK" && inst.asset_class !== "US_STOCK" && inst.asset_class !== "ETF") {
      setErr("관심 종목은 주식·ETF만 추가할 수 있습니다.");
      return;
    }
    try {
      await addWatchlist(inst.id);
      await load();
      setErr(null);
    } catch (e: unknown) {
      const code = (e as { code?: string })?.code;
      setErr(code === "CONFLICT" ? "이미 추가됨" : "추가 실패");
    }
  }

  async function handleRemove(iid: string) {
    try {
      await removeWatchlist(iid);
      setItems((prev) => prev.filter((x) => x.instrument_id !== iid));
    } catch {
      setErr("삭제 실패");
    }
  }

  return (
    <div className="border border-line p-4 md:col-span-2 lg:col-span-3">
      <div className="font-mono text-sm mb-3">관심 종목</div>
      <div className="mb-3">
        <InstrumentSearchInput onSelect={handleAdd} placeholder="종목 검색하여 추가" />
      </div>
      {err && <p className="text-bb-down text-xs font-mono mb-2">{err}</p>}
      {items.length === 0 ? (
        <p className="text-fg-muted text-xs font-mono">관심 종목이 없습니다. 위에서 검색하여 추가하세요.</p>
      ) : (
        <ul className="divide-y divide-line/50">
          {items.map((w) => (
            <li key={w.instrument_id} className="flex items-center gap-3 py-2 font-mono text-sm">
              <div className="flex-1 min-w-0">
                <div className="truncate">{w.symbol}</div>
                <div className="text-xs text-fg-muted truncate">{w.name}</div>
              </div>
              <Sparkline points={sparks[w.instrument_id] ?? []} width={80} height={24} />
              <div className="text-right tabular-nums w-24">
                {w.price > 0 ? w.price.toLocaleString() : "—"}
              </div>
              <div className={`text-xs tabular-nums w-16 text-right ${w.change_pct >= 0 ? "text-bb-up" : "text-bb-down"}`}>
                {w.change_pct >= 0 ? "+" : ""}{w.change_pct.toFixed(2)}%
              </div>
              <button
                onClick={() => handleRemove(w.instrument_id)}
                className="text-xs text-fg-muted hover:text-bb-down px-2"
                title="삭제"
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
```

### Step 2: WatchlistMiniCard 카피 정정

`apps/web/components/home/WatchlistMiniCard.tsx`에서:
```tsx
<div className="text-fg-muted text-sm font-mono">아직 없음. 마켓 탭 (W5)에서 추가 예정</div>
```
→
```tsx
<div className="text-fg-muted text-sm font-mono">아직 없음. 마켓 탭에서 추가하세요.</div>
```

### Step 3: tsc + 커밋

Run: `cd apps/web && npx tsc --noEmit`
Expected: 에러 없음.

```bash
git add apps/web/components/market/WatchlistEditorCard.tsx apps/web/components/home/WatchlistMiniCard.tsx
git commit -m "feat(web): 마켓 탭 watchlist editor (추가·제거·미니 스파크라인) + 홈 카피 정정"
```

---

## Task 7: 포트폴리오 행 스파크라인

**Files:**
- Modify: `apps/web/components/portfolio/HoldingsTable.tsx`

각 행의 마지막 액션 셀 직전에 sparkline 셀 추가. 마운트 시 `fetchPriceHistoryBatch(ids, "1w")`로 일괄 로드.

### Step 1: HoldingsTable 수정

기존 HoldingsTable에 다음을 추가:

1. import 추가:
```tsx
import { Sparkline } from "@/components/charts/Sparkline";
import { fetchPriceHistoryBatch } from "@/lib/api/history";
```

2. state 추가:
```tsx
const [sparks, setSparks] = useState<Record<string, { value: number }[]>>({});
```

3. `load()` 함수 끝에 batch fetch:
```tsx
async function load() {
  try {
    const data = await listHoldings();
    setHoldings(data);
    setErr(null);
    // 스파크라인 batch (7일)
    const ids = data.map((h) => h.instrument_id);
    if (ids.length > 0) {
      const batch = await fetchPriceHistoryBatch(ids, "1w").catch(() => ({}));
      const sp: Record<string, { value: number }[]> = {};
      for (const [iid, points] of Object.entries(batch)) {
        sp[iid] = points.map((p) => ({ value: p.close }));
      }
      setSparks(sp);
    }
  } catch (e: unknown) {
    setErr((e as { message?: string })?.message ?? "로드 실패");
    setHoldings([]);
  }
}
```

4. 테이블 헤더 — 마지막 빈 컬럼 앞에 새 `<th>` 추가:
```tsx
<th className="text-right px-3 py-2">7일</th>
<th className="px-3 py-2"></th>
```

5. 각 `<tr>` body — 비중 셀 다음에 sparkline 셀 추가:
```tsx
<td className="text-right px-3 py-2">{h.weight_pct.toFixed(1)}%</td>
<td className="px-3 py-2">
  <Sparkline points={sparks[h.instrument_id] ?? []} width={70} height={20} />
</td>
<td className="px-3 py-2 text-right">
  {/* 기존 수정/삭제 버튼 */}
</td>
```

### Step 2: tsc + 커밋

Run: `cd apps/web && npx tsc --noEmit`
Expected: 에러 없음.

```bash
git add apps/web/components/portfolio/HoldingsTable.tsx
git commit -m "feat(web): 포트폴리오 행 7일 스파크라인 (batch fetch)"
```

## Self-Review 항목
- batch fetch 1회만 (각 행별 호출 X)
- 빈 sparkline은 "—"로 fallback
- table column 개수 변경 (header + body 행 균일)

---

## Task 8: 통합 검증 + 문서 갱신

**Files:**
- Modify: `docs/STATUS.md`
- Modify: `docs/ROADMAP.md`
- (선택) `docs/ARCHITECTURE.md` — recharts 도입 결정만 짧게

자동 검증:
- `cd apps/api && go build ./... && go test ./...`
- `cd apps/web && npx tsc --noEmit && npm run build`

런타임 검증 (사용자 환경, dev 서버 띄운 상태):
- `/app/market` 접속 → 9 카드 + AdSlot 로드 (ADS_DISABLED placeholder)
- 마켓 탭에서 종목 검색 → watchlist 추가 → 행 출현 + sparkline
- 행 × 버튼 → 즉시 제거
- `/app/portfolio` → 각 행 끝에 7일 sparkline 표시
- 홈 탭 watchlist mini card도 동기화

### Step 1: 자동 검증

```bash
cd apps/api && go build ./... && go test ./...
cd apps/web && npx tsc --noEmit && npm run build
```

Expected: 모두 PASS.

### Step 2: STATUS.md 갱신

`docs/STATUS.md`에서:

1. 날짜 갱신: `마지막 업데이트: 2026-05-26`
2. "## 현재 Phase" → `**Phase 1 — W1·W2a·W2b·W3·W4·W5 완료. Phase 1 핵심 종료 (외부 셋업·일부 후반 작업 남음).**`
3. "완료" 섹션에 W5-T1~T8 추가 (commit SHA는 git log로 확인 — 8 entry).
4. "## 알려진 결함"에서 다음 줄 제거:
   - `- **watchlist 추가 UI 부재**: ...`
   - `- **포트폴리오 미니 스파크라인 미구현**: ...`
5. "## 알려진 결함"에 W5 결함 추가 (구체):
   ```
   - **AdSense 미가입**: AdSlot은 ENABLE_ADS=false 기본 → placeholder만 표시. 사용자 100명·일평균 PV 500 달성 시 Phase 2에서 활성
   - **포트폴리오 sliding panel 미구현**: 스펙 §6 선택 행 상세 패널은 Phase 1 후반·v2로 미룸
   ```
6. "최근 변경 이력" 맨 위에 추가:
   ```
   - 2026-05-26 W5 전체 완료. 마켓 탭 (KR/US 지수·환율·경제지표·관심종목) + recharts 도입 + 포트폴리오 7일 스파크라인(batch fetch) + AdSlot 추상화. history API 3 라우트(prices·indicators·fx) 추가.
   ```

### Step 3: ROADMAP.md 갱신

`docs/ROADMAP.md`에서:

1. "현재 추천 다음 작업" → 다음으로 교체:
   ```
   Phase 1 외부 셋업·후반 작업:
   - W1-T13~T16: Sentry/PostHog DSN + Fly/Vercel 배포 + GitHub Actions + 풀 E2E
   - AI RealClient 실 구현 (anthropic-sdk-go, claude-api 스킬)
   - 명령 팔레트 (⌘K) + 키보드 단축키
   - 포트폴리오 sliding panel
   ```
2. Phase 1 표에서 W5 관련 행 2개 제거:
   - 마켓 탭
   - AdSlot
3. 표의 남은 항목 우선순위 재정렬.

### Step 4: ARCHITECTURE.md에 차트 라이브러리 결정 추가 (필수)

`docs/ARCHITECTURE.md`의 "핵심 설계 결정" 끝에 추가:
```markdown
### 10. 차트 라이브러리 — recharts

**Why**: React 표준 + SVG + TypeScript. 단일 라이브러리로 라인·도넛·스파크라인 모두 처리. 도입 비용 낮고 커뮤니티 견고.

**How**: 마켓 탭 LineChartCard + 포트폴리오·watchlist Sparkline 공통 사용. dynamic import로 사용 페이지에서만 로드 (~80KB gzipped).

**Trade-off**: lightweight-charts 대비 캔들·볼린저 등 금융 차트 전용 기능 부재 → v2 이후 트레이딩뷰 스타일 필요 시 별도 페이지로 분리 검토.
```

### Step 5: 커밋

```bash
git add docs/STATUS.md docs/ROADMAP.md docs/ARCHITECTURE.md
git commit -m "docs(w5): W5 완료 반영 (STATUS·ROADMAP·ARCHITECTURE recharts 결정 추가)"
```

---

## 후속 작업 (W5 비범위, 백로그)

- **AI RealClient 실 구현**: anthropic-sdk-go의 Messages.NewStreaming + tool 변환 + prompt caching. claude-api 스킬 활용
- **AdSense 가입 + ENABLE_ADS=true**: 가입자 100명·일평균 PV 500 도달 시 Phase 2
- **포트폴리오 sliding panel**: 선택 행 우측 상세 패널 (스펙 §6)
- **명령 팔레트 ⌘K**: 종목 검색·탭 이동·"AI에게 묻기"·설정 진입
- **키보드 단축키 풀세트**: `1~5` 탭, `/` 검색, `c` 채팅, `g h` 홈 등 (vim-like, 스펙 §6)
- **차트 줌·범위 선택 UI**: 마켓 탭 카드별 1w/1mo/6mo/1y/5y 토글
- **AI 일일 브리핑 도구 호출 통합**: 현재 1턴 호출 → 보유 자산·시세 변화를 도구로 자동 주입 (스펙 §10-8 강화)

---

## 검토 이력

### 2026-05-26 — 1차 subagent 검토 (general-purpose)

| 우선순위 | 항목 | 패치 위치 |
|---|---|---|
| Critical | SQL `date - interval` 패턴 timezone 1일 어긋남 | Task 1 — `(current_date - $2::interval)::date` 명시 + indicator/fx는 `now() - ($3::int * interval '1 day')` |
| Critical | `PriceByIDsBatch`의 `instrument_id::text` cast로 인덱스 미사용 | Task 1 — `any($1::uuid[])` UUID 직접 비교로 prices_date_idx 활용 |
| Critical | router.New 시그니처 변경 코드 블록 부재 | Task 1 Step 4 — 전체 시그니처 + 라우트 위치 명시 |
| Critical | `NEXT_PUBLIC_ENABLE_ADS` env 누락 (client 측 미주입) | Task 4 Step 2 — `.env.example`에 양쪽 키 추가 |
| Critical | KRIndicesCard 코드 블록의 `void fetchPriceHistoryBatch` 자기모순 | Task 5 — import에서 제거, void 라인 삭제로 정정 |
| Critical | 인덱스 일봉 데이터 부재 — 마켓 탭 시연 불가 | Task 1.5 신설 — cron `JobUpdateKRPrices`/`JobUpdateUSPrices`에 `*-IDX` exchange 포함 |
| Important | `time` import 미사용 placeholder | Task 1 — import 삭제 + var 제거 |
| Important | watchlist asset_class frontend 가드만 — backend 우회 가능 | Task 6 Step 0 — `Add` 핸들러에 가드 추가, WatchlistHandler 시그니처에 pool 주입 |
| Important | ARCHITECTURE.md 차트 라이브러리 결정 "선택" → 필수 | Task 8 Step 4 |
| Minor | `recharts dynamic import` loading state 부재, MarketPage dedupe | 후속 backlog (UX 폴리시) |

Important 잔여(days fallback 명시, holdings.instrument_id 타입 확인, exhaustive-deps 경고 등) 및 Minor 다수는 구현 시점 정리.

### 2026-05-26 — 사용자 결정 사항 반영

- **차트 라이브러리**: recharts 단일 도입.
- **포트폴리오 미니 스파크라인**: W5에 포함 (Task 7).

---

## Self-Review 체크리스트

- [ ] 스펙 §6 마켓 탭 5 카드(KR/US 지수·환율·지표·관심종목) — Task 5·6
- [ ] 스펙 §6 포트폴리오 미니 차트 — Task 7
- [ ] 스펙 §8 광고 슬롯 추상화 — Task 4
- [ ] 차트 라이브러리 recharts 단일 도입 — Task 2
- [ ] history API 3종 + batch 모드 — Task 1
- [ ] 모든 핸들러 인증 가드 (chi auth group)
- [ ] 빈 데이터 fallback "데이터 없음"/"—"
- [ ] watchlist 추가 시 asset_class 가드 (KR_STOCK·US_STOCK·ETF만)
- [ ] dynamic recharts import (Next.js 16 SSR 비활성)
