# Quotient W3 — 포트폴리오 · Watchlist · 홈 대시보드

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development` (권장) 또는 `superpowers:executing-plans`. 체크박스(`- [ ]`) 단위 추적.

**Goal:** W2a 마켓 마스터 + W2b 시세 파이프라인 위에 사용자 데이터(`holdings`, `watchlist`)를 얹는다. 포트폴리오 CRUD UI + 홈 대시보드(총자산·도넛·보유 상위·마켓 위젯·관심 종목) + 온보딩 wizard 3단계(holdings 1~3개 추가 스텝 복원). cron `JobUpdateIndexQuotes` polling 대상을 INDEX ∪ holdings ∪ watchlist union으로 확장(dedup·60s TTL 유지). W3 종료 시점: 사용자가 종목을 추가하면 1분 내 해당 종목 `quotes`가 적재되고 포트폴리오 테이블·홈 대시보드에 평가액·수익률·비중이 표시된다.

**Architecture:**
- 데이터: `holdings(user_id, instrument_id, quantity numeric(20,8), avg_cost numeric(20,4), opened_at, note)` UNIQUE (user_id, instrument_id). `watchlist(user_id, instrument_id, added_at)` PK (user_id, instrument_id). 둘 다 RLS `user_id = auth.uid()`. ON DELETE CASCADE로 instrument·user 삭제 전파.
- API: 슈퍼유저 풀 + `WHERE user_id = $1` 필터 패턴 유지(profiles와 동일, STATUS.md 결함은 W5 이후로 미룸 — 사용자 결정). 통화 환산은 Go 서비스 레이어에서 KRW 기준으로 일괄 처리(FX는 `quotes` 테이블의 `USD_KRW`/`EUR_KRW`/`JPY_KRW` 조회, 없으면 1.0 fallback + 경고 로그).
- cron: `JobUpdateIndexQuotes` → `JobUpdateMarketQuotes`로 확장. 폴링 대상 = INDEX(시장 시간 무관, 항상 fetch) ∪ holdings.instrument_id ∪ watchlist.instrument_id (dedup). KR/US 장중 판정은 instrument별 적용.
- UI: Next.js 16 client component. 도넛은 SVG `<path d="...">` arc 직접 계산(recharts 등 무거운 라이브러리 미도입). 총자산 카운트업은 `requestAnimationFrame` 250ms.
- 미니 스파크라인(spec §6) 및 마켓 탭(`/app/market`)은 W3 비범위 — Phase 1 후반·W5에서 별도 처리.

**Tech Stack:** Go 1.25 + chi v5 + pgx v5 + Next.js 16 + Tailwind v4 + shadcn (dialog·input·button 기존 컴포넌트 재사용).

**참고 스펙:** [`2026-05-22-quotient-mvp-design.md`](../specs/2026-05-22-quotient-mvp-design.md) §3(데이터 모델)·§6(정보 구조)·§10-2(quotes TTL 캐시)·§10-6(instruments unique). [`W2b plan`](2026-05-22-p1-w2b-pipeline-runtime.md) (cron 패턴·핸들러 패턴).

> **Next.js 16 주의 (UI Task)**: `apps/web/AGENTS.md` — 기존 Next.js 지식과 API 차이 있음. client component 작성 전 `apps/web/node_modules/next/dist/docs/`에서 관련 가이드 확인 필수. `use client` 디렉티브, `useEffect`/`useState`는 동일하지만 Server Action·Route Handler 시그니처 확인 필요.

---

## 외부 셋업 (Task 0)

W2a·W2b에서 완료. 추가 외부 키 없음. `.env` 변경 없음.

---

## File Structure (W3 생성·수정)

```
supabase/
└── migrations/
    └── 20260523000001_holdings_watchlist.sql   (신규)

apps/api/
├── cmd/server/main.go                          (수정) handler 와이어링 2개 추가
├── internal/
│   ├── models/
│   │   ├── holding.go                          (신규)
│   │   └── watchlist.go                        (신규)
│   ├── handlers/
│   │   ├── holdings.go                         (신규) GET/POST/PATCH/DELETE
│   │   ├── holdings_repo_pg.go                 (신규)
│   │   ├── holdings_test.go                    (신규)
│   │   ├── watchlist.go                        (신규) GET/POST/DELETE
│   │   ├── watchlist_repo_pg.go                (신규)
│   │   ├── watchlist_test.go                   (신규)
│   │   └── pricing.go                          (신규) FX 환산 helper
│   ├── schedule/
│   │   └── jobs_quotes.go                      (수정) union 폴링
│   └── router/router.go                        (수정) 라우트 7개 추가

apps/web/
├── lib/api/
│   ├── holdings.ts                             (신규)
│   ├── watchlist.ts                            (신규)
│   ├── instruments.ts                          (신규) search·select 래퍼
│   └── auth-fetch.ts                           (신규) 토큰 자동 첨부 fetch helper
├── components/
│   ├── portfolio/
│   │   ├── HoldingsTable.tsx                   (신규)
│   │   ├── AddHoldingDialog.tsx                (신규)
│   │   ├── EditHoldingDialog.tsx               (신규)
│   │   ├── InstrumentSearchInput.tsx           (신규) 디바운스 검색
│   │   └── DeleteConfirmDialog.tsx             (신규)
│   ├── home/
│   │   ├── TotalAssetCard.tsx                  (신규) 카운트업
│   │   ├── AllocationDonut.tsx                 (신규) SVG arc
│   │   ├── TopHoldingsCard.tsx                 (신규)
│   │   ├── MarketWidgetsCard.tsx               (신규)
│   │   ├── WatchlistMiniCard.tsx               (신규)
│   │   └── BriefingPlaceholderCard.tsx         (신규)
│   └── onboarding/
│       ├── Wizard.tsx                          (수정) 3단계
│       ├── HoldingsStep.tsx                    (신규)
│       └── StepIndicator.tsx                   (수정) total 3
└── app/app/
    ├── page.tsx                                (수정) 홈 대시보드 그리드
    └── portfolio/page.tsx                      (신규)
```

API 엔드포인트 (7개):
- `GET    /v1/holdings`              — 보유 자산 목록 (FX 환산 + 평가액·수익률·비중 계산)
- `POST   /v1/holdings`              — 추가 `{ instrument_id, quantity, avg_cost, opened_at?, note? }`
- `PATCH  /v1/holdings/{id}`         — 수정 (quantity·avg_cost·note만)
- `DELETE /v1/holdings/{id}`         — 삭제
- `GET    /v1/watchlist`             — 관심 종목 목록 (현재가 join)
- `POST   /v1/watchlist`             — 추가 `{ instrument_id }`
- `DELETE /v1/watchlist/{instrument_id}` — 삭제

---

## Task 1: holdings · watchlist 마이그레이션 + RLS

**Files:**
- Create: `supabase/migrations/20260523000001_holdings_watchlist.sql`

스펙 §3 인덱스 정의 정확 반영. ON DELETE CASCADE로 instrument·user 삭제 시 자동 정리. `touch_updated_at()` 함수는 W1 migration 0001에서 이미 정의됐으므로 재사용.

- [ ] **Step 1: 마이그레이션 파일 작성**

```sql
-- holdings: 사용자 보유 자산
create table public.holdings (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references auth.users(id) on delete cascade,
  instrument_id uuid not null references public.instruments(id) on delete cascade,
  quantity numeric(20, 8) not null check (quantity > 0),
  avg_cost numeric(20, 4) not null check (avg_cost >= 0),
  opened_at date,
  note text,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint holdings_user_instrument_unique unique (user_id, instrument_id)
);

create index holdings_user_created_idx on public.holdings (user_id, created_at desc);

create trigger holdings_touch
  before update on public.holdings
  for each row execute function public.touch_updated_at();

-- watchlist: 관심 종목
create table public.watchlist (
  user_id uuid not null references auth.users(id) on delete cascade,
  instrument_id uuid not null references public.instruments(id) on delete cascade,
  added_at timestamptz not null default now(),
  primary key (user_id, instrument_id)
);

create index watchlist_user_idx on public.watchlist (user_id, instrument_id);

-- RLS
alter table public.holdings  enable row level security;
alter table public.watchlist enable row level security;

create policy "holdings_select_own" on public.holdings
  for select using (auth.uid() = user_id);
create policy "holdings_insert_own" on public.holdings
  for insert with check (auth.uid() = user_id);
create policy "holdings_update_own" on public.holdings
  for update using (auth.uid() = user_id) with check (auth.uid() = user_id);
create policy "holdings_delete_own" on public.holdings
  for delete using (auth.uid() = user_id);

create policy "watchlist_select_own" on public.watchlist
  for select using (auth.uid() = user_id);
create policy "watchlist_insert_own" on public.watchlist
  for insert with check (auth.uid() = user_id);
create policy "watchlist_delete_own" on public.watchlist
  for delete using (auth.uid() = user_id);
```

- [ ] **Step 2: 마이그레이션 적용**

Run: `supabase db reset` (로컬)
Expected: `Applying migration 20260523000001_holdings_watchlist.sql...` 후 에러 없이 종료. `psql $DATABASE_URL -c "\dt public.*"`로 `holdings`, `watchlist` 존재 확인.

- [ ] **Step 3: RLS 정책 검증 쿼리**

Run:
```bash
psql "$DATABASE_URL" -c "select tablename, policyname from pg_policies where schemaname='public' and tablename in ('holdings','watchlist') order by tablename, policyname;"
```
Expected: holdings 4개(select·insert·update·delete) + watchlist 3개(select·insert·delete) = 총 7행.

- [ ] **Step 4: 커밋**

```bash
git add supabase/migrations/20260523000001_holdings_watchlist.sql
git commit -m "feat(db): holdings + watchlist 마이그레이션 + RLS 7개 정책"
```

---

## Task 1.5: middleware 테스트 helper 추가

**Files:**
- Modify: `apps/api/internal/middleware/auth.go`

`apps/api/internal/middleware/auth.go`에 현재 `UserID(ctx)`만 있고 `WithUserID(ctx, uid)` 함수는 없다. W3 신규 핸들러 테스트가 모두 이 helper를 사용하므로 먼저 추가한다. 기존 `profiles_test.go`가 `context.WithValue(req.Context(), middleware.UserIDKey, uid)` 직접 호출 패턴이라면 그대로 유지해도 무방하나, 명시적 helper가 가독성·일관성에 유리.

- [ ] **Step 1: helper 추가**

`auth.go` 파일 끝(`RawJWT` 함수 아래)에 추가:

```go
// WithUserID는 테스트·미들웨어 외 경로에서 사용자 ID를 ctx에 주입.
// 프로덕션 코드는 RequireAuth 미들웨어가 이미 주입하므로 호출하지 않는다.
func WithUserID(ctx context.Context, uid string) context.Context {
	return context.WithValue(ctx, UserIDKey, uid)
}
```

- [ ] **Step 2: 빌드 확인**

Run: `cd apps/api && go build ./internal/middleware/...`
Expected: 에러 없음.

- [ ] **Step 3: 커밋**

```bash
git add apps/api/internal/middleware/auth.go
git commit -m "feat(api): middleware.WithUserID 테스트 helper 추가"
```

---

## Task 2: Go 모델 정의

**Files:**
- Create: `apps/api/internal/models/holding.go`
- Create: `apps/api/internal/models/watchlist.go`

핸들러가 반환하는 enriched 구조까지 모델에 포함. `pgtype.Numeric` 대신 `float64`로 통일 — 금액 8자리 정밀도는 frontend·DB에서 보장, Go 중간 계산은 float64로 단순화(MVP 한정, 정밀 회계는 Phase 3 백로그).

- [ ] **Step 1: holding.go 작성**

```go
package models

import "time"

// Holding은 보유 자산 한 건의 raw 데이터.
type Holding struct {
	ID           string     `json:"id"`
	InstrumentID string     `json:"instrument_id"`
	Quantity     float64    `json:"quantity"`
	AvgCost      float64    `json:"avg_cost"`
	OpenedAt     *time.Time `json:"opened_at,omitempty"`
	Note         *string    `json:"note,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// HoldingEnriched는 instrument·quote join + KRW 환산 평가액·수익률·비중.
type HoldingEnriched struct {
	Holding
	Symbol        string  `json:"symbol"`
	Exchange      string  `json:"exchange"`
	Name          string  `json:"name"`
	AssetClass    string  `json:"asset_class"`
	Currency      string  `json:"currency"`        // 원본 통화
	CurrentPrice  float64 `json:"current_price"`   // quotes.price (원본 통화), 없으면 0
	MarketValue   float64 `json:"market_value"`    // qty * price (원본 통화)
	MarketValueKRW float64 `json:"market_value_krw"` // KRW 환산
	CostBasisKRW  float64 `json:"cost_basis_krw"`  // qty * avg_cost in KRW
	PnLKRW        float64 `json:"pnl_krw"`         // market_value_krw - cost_basis_krw
	PnLPct        float64 `json:"pnl_pct"`         // (market_value - cost_basis) / cost_basis * 100, 원본 통화 기준
	WeightPct     float64 `json:"weight_pct"`      // 전체 KRW 평가 합계 대비 비중
}
```

- [ ] **Step 2: watchlist.go 작성**

```go
package models

import "time"

type WatchlistItem struct {
	InstrumentID string    `json:"instrument_id"`
	AddedAt      time.Time `json:"added_at"`
	Symbol       string    `json:"symbol"`
	Exchange     string    `json:"exchange"`
	Name         string    `json:"name"`
	AssetClass   string    `json:"asset_class"`
	Currency     string    `json:"currency"`
	Price        float64   `json:"price"`      // quotes.price, 없으면 0
	ChangePct    float64   `json:"change_pct"`
}
```

- [ ] **Step 3: 컴파일 확인**

Run: `cd apps/api && go build ./internal/models/...`
Expected: 에러 없음.

- [ ] **Step 4: 커밋**

```bash
git add apps/api/internal/models/holding.go apps/api/internal/models/watchlist.go
git commit -m "feat(api): holdings·watchlist 모델 (enriched 포함)"
```

---

## Task 3: FX 환산 helper

**Files:**
- Create: `apps/api/internal/handlers/pricing.go`

`quotes` 테이블에서 `USD_KRW` 심볼 조회 → float64. 캐시는 단일 요청 scope로 충분(요청당 1회 조회). 환율 행이 없거나 `price=0`이면 fallback 1.0 + 경고 로그(USD 자산 가치가 USD 단위로 표시될 위험, MVP 트레이드오프).

> **알려진 한계 (W3에서 의식적 비범위)**: 현재 `JobUpdateFXRates` (`apps/api/internal/schedule/jobs_fx.go`)는 quotes에 `USD_KRW` 1행만 적재한다(rateMap key가 `USD_KRW`/`USD_EUR`/`USD_JPY`인데 instrument symbol은 `USD_KRW`/`EUR_KRW`/`JPY_KRW`라 EUR_KRW·JPY_KRW는 매칭 안 됨). **결과: W3 시점에 KRW 외 환산은 USD만 정확하고 EUR·JPY 자산은 fallback 1.0로 왜곡됨**. MVP 사용자는 거의 KR/US 자산만 보유한다는 가정으로 W3 비범위 처리. EUR_KRW/JPY_KRW 적재는 W2b 별도 fix-up 또는 W5 마켓 탭 작업 시 정리 (별도 백로그 항목 추가). 위 Task 3의 fallback 경고 로그가 운영 모니터링에서 이를 표면화한다.

- [ ] **Step 1: pricing.go 작성**

```go
package handlers

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

// FetchFXRates는 현재 quotes에서 통화별 KRW 환율을 반환.
// 키: 원본 통화(예: "USD"). 값: KRW 환산 배수(예: 1380.5).
// KRW는 항상 1.0. quotes 미존재 통화는 1.0 fallback + 경고.
func FetchFXRates(ctx context.Context, pool *pgxpool.Pool) (map[string]float64, error) {
	// FX instruments의 symbol은 "USD_KRW", "EUR_KRW", "JPY_KRW" 형식.
	// "<CURRENCY>_KRW" 패턴에서 첫 토큰을 키로 추출.
	rows, err := pool.Query(ctx, `
		select i.symbol, coalesce(q.price, 0)::float8
		from public.instruments i
		left join public.quotes q on q.instrument_id = i.id
		where i.asset_class = 'FX' and i.symbol like '%\_KRW' escape '\'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rates := map[string]float64{"KRW": 1.0}
	for rows.Next() {
		var sym string
		var price float64
		if err := rows.Scan(&sym, &price); err != nil {
			return nil, err
		}
		// "USD_KRW" → "USD"
		base := sym
		for i, c := range sym {
			if c == '_' {
				base = sym[:i]
				break
			}
		}
		if price <= 0 {
			slog.Warn("FX rate missing, fallback 1.0", "symbol", sym)
			rates[base] = 1.0
			continue
		}
		rates[base] = price
	}
	return rates, rows.Err()
}

// ToKRW는 amount를 원본 통화에서 KRW로 환산.
// 알 수 없는 통화는 1.0(KRW 가정) + 경고 로그.
func ToKRW(amount float64, currency string, rates map[string]float64) float64 {
	r, ok := rates[currency]
	if !ok {
		slog.Warn("unknown currency, treating as KRW", "currency", currency)
		return amount
	}
	return amount * r
}
```

- [ ] **Step 2: 단위 테스트**

`apps/api/internal/handlers/pricing_test.go`:
```go
package handlers

import "testing"

func TestToKRW(t *testing.T) {
	rates := map[string]float64{"KRW": 1.0, "USD": 1380.0, "JPY": 9.2}
	cases := []struct {
		amount   float64
		currency string
		want     float64
	}{
		{1000, "KRW", 1000},
		{10, "USD", 13800},
		{1000, "JPY", 9200},
		{100, "EUR", 100}, // 미정의 → 1.0 fallback (경고)
	}
	for _, c := range cases {
		got := ToKRW(c.amount, c.currency, rates)
		if got != c.want {
			t.Errorf("ToKRW(%v,%v) = %v, want %v", c.amount, c.currency, got, c.want)
		}
	}
}
```

Run: `cd apps/api && go test ./internal/handlers/ -run TestToKRW -v`
Expected: PASS (4 케이스).

- [ ] **Step 3: 커밋**

```bash
git add apps/api/internal/handlers/pricing.go apps/api/internal/handlers/pricing_test.go
git commit -m "feat(api): FX 환산 helper (FetchFXRates + ToKRW)"
```

---

## Task 4: holdings repo (Postgres)

**Files:**
- Create: `apps/api/internal/handlers/holdings_repo_pg.go`

repo는 raw CRUD만 담당. enrichment(quote join + FX 환산 + 비중 계산)는 handler 레이어에서 별도 호출. 분리 이유: 테스트 시 환산 로직과 SQL을 따로 검증 가능.

- [ ] **Step 1: repo 인터페이스·구현 작성**

```go
package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/models"
)

var ErrHoldingNotFound = errors.New("holding not found")
var ErrHoldingConflict = errors.New("holding already exists for this instrument")

type HoldingRepo interface {
	List(ctx context.Context, userID string) ([]models.HoldingEnriched, error)
	Create(ctx context.Context, userID, instrumentID string, qty, avgCost float64, openedAt *string, note *string) (*models.Holding, error)
	Update(ctx context.Context, userID, id string, patch map[string]any) (*models.Holding, error)
	Delete(ctx context.Context, userID, id string) error
}

type PgHoldingRepo struct {
	pool *pgxpool.Pool
}

func NewPgHoldingRepo(pool *pgxpool.Pool) *PgHoldingRepo {
	return &PgHoldingRepo{pool: pool}
}

// List는 holdings + instruments + 최신 quotes를 join하여 enrichment 전 상태로 반환.
// 가격·환율 환산·비중 계산은 handler에서 처리.
func (r *PgHoldingRepo) List(ctx context.Context, userID string) ([]models.HoldingEnriched, error) {
	rows, err := r.pool.Query(ctx, `
		select
		  h.id::text, h.instrument_id::text, h.quantity::float8, h.avg_cost::float8,
		  h.opened_at, h.note, h.created_at, h.updated_at,
		  i.symbol, i.exchange, i.name, i.asset_class, i.currency,
		  coalesce(q.price, 0)::float8
		from public.holdings h
		join public.instruments i on i.id = h.instrument_id
		left join public.quotes q on q.instrument_id = h.instrument_id
		where h.user_id = $1
		order by h.created_at desc
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.HoldingEnriched
	for rows.Next() {
		var h models.HoldingEnriched
		if err := rows.Scan(
			&h.ID, &h.InstrumentID, &h.Quantity, &h.AvgCost,
			&h.OpenedAt, &h.Note, &h.CreatedAt, &h.UpdatedAt,
			&h.Symbol, &h.Exchange, &h.Name, &h.AssetClass, &h.Currency,
			&h.CurrentPrice,
		); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *PgHoldingRepo) Create(ctx context.Context, userID, instrumentID string, qty, avgCost float64, openedAt *string, note *string) (*models.Holding, error) {
	row := r.pool.QueryRow(ctx, `
		insert into public.holdings (user_id, instrument_id, quantity, avg_cost, opened_at, note)
		values ($1, $2, $3, $4, $5, $6)
		returning id::text, instrument_id::text, quantity::float8, avg_cost::float8, opened_at, note, created_at, updated_at
	`, userID, instrumentID, qty, avgCost, openedAt, note)
	var h models.Holding
	if err := row.Scan(&h.ID, &h.InstrumentID, &h.Quantity, &h.AvgCost, &h.OpenedAt, &h.Note, &h.CreatedAt, &h.UpdatedAt); err != nil {
		// pgx unique violation: SQLSTATE 23505
		if strings.Contains(err.Error(), "23505") {
			return nil, ErrHoldingConflict
		}
		return nil, err
	}
	return &h, nil
}

func (r *PgHoldingRepo) Update(ctx context.Context, userID, id string, patch map[string]any) (*models.Holding, error) {
	sets := []string{}
	args := []any{}
	i := 1
	for k, v := range patch {
		switch k {
		case "quantity", "avg_cost", "note", "opened_at":
			sets = append(sets, fmt.Sprintf("%s = $%d", k, i))
			args = append(args, v)
			i++
		}
	}
	if len(sets) == 0 {
		// 갱신 없음 — 현재 행 반환 (List에서 1건만 추출하기보다 직접 select)
		row := r.pool.QueryRow(ctx, `
			select id::text, instrument_id::text, quantity::float8, avg_cost::float8, opened_at, note, created_at, updated_at
			from public.holdings where id = $1 and user_id = $2
		`, id, userID)
		var h models.Holding
		if err := row.Scan(&h.ID, &h.InstrumentID, &h.Quantity, &h.AvgCost, &h.OpenedAt, &h.Note, &h.CreatedAt, &h.UpdatedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrHoldingNotFound
			}
			return nil, err
		}
		return &h, nil
	}
	args = append(args, id, userID)
	q := fmt.Sprintf(`
		update public.holdings set %s
		where id = $%d and user_id = $%d
		returning id::text, instrument_id::text, quantity::float8, avg_cost::float8, opened_at, note, created_at, updated_at
	`, strings.Join(sets, ", "), i, i+1)

	row := r.pool.QueryRow(ctx, q, args...)
	var h models.Holding
	if err := row.Scan(&h.ID, &h.InstrumentID, &h.Quantity, &h.AvgCost, &h.OpenedAt, &h.Note, &h.CreatedAt, &h.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrHoldingNotFound
		}
		return nil, err
	}
	return &h, nil
}

func (r *PgHoldingRepo) Delete(ctx context.Context, userID, id string) error {
	ct, err := r.pool.Exec(ctx, `delete from public.holdings where id = $1 and user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrHoldingNotFound
	}
	return nil
}
```

- [ ] **Step 2: 컴파일 확인**

Run: `cd apps/api && go build ./internal/handlers/...`
Expected: 에러 없음.

- [ ] **Step 3: 커밋**

```bash
git add apps/api/internal/handlers/holdings_repo_pg.go
git commit -m "feat(api): holdings Postgres repo (CRUD + enrichment join)"
```

---

## Task 5: holdings handler + 라우트·핸들러 통합 테스트

**Files:**
- Create: `apps/api/internal/handlers/holdings.go`
- Create: `apps/api/internal/handlers/holdings_test.go`

핸들러는: ① 인증 추출 ② JSON 검증 ③ repo 호출 ④ List 응답 시 FX 환산·비중 계산.

- [ ] **Step 1: handler 작성**

```go
package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
)

type HoldingHandler struct {
	repo HoldingRepo
	pool *pgxpool.Pool // FX 환율 조회용
}

func NewHoldingHandler(repo HoldingRepo, pool *pgxpool.Pool) *HoldingHandler {
	return &HoldingHandler{repo: repo, pool: pool}
}

// GET /v1/holdings
func (h *HoldingHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	items, err := h.repo.List(r.Context(), uid)
	if err != nil {
		slog.Error("holdings list failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}

	// FX 환산 + 평가액·수익률 계산
	rates, err := FetchFXRates(r.Context(), h.pool)
	if err != nil {
		slog.Error("fx rates fetch failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "fx load failed")
		return
	}
	enriched := enrichHoldings(items, rates)

	if enriched == nil {
		enriched = []models.HoldingEnriched{}
	}
	writeJSON(w, http.StatusOK, enriched)
}

// POST /v1/holdings
func (h *HoldingHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var body struct {
		InstrumentID string  `json:"instrument_id"`
		Quantity     float64 `json:"quantity"`
		AvgCost      float64 `json:"avg_cost"`
		OpenedAt     *string `json:"opened_at,omitempty"` // "YYYY-MM-DD"
		Note         *string `json:"note,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if body.InstrumentID == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "instrument_id required")
		return
	}
	if body.Quantity <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "quantity must be > 0")
		return
	}
	if body.AvgCost < 0 {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "avg_cost must be >= 0")
		return
	}
	// asset_class 가드: INDEX·FX·CASH instrument는 holdings 대상이 아님 (W3 MVP).
	// 사용자 검색 UI는 KR_STOCK·US_STOCK·ETF만 노출하지만 직접 ID 입력 우회 차단.
	var assetClass string
	if err := h.pool.QueryRow(r.Context(), `select asset_class from public.instruments where id = $1`, body.InstrumentID).Scan(&assetClass); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "instrument not found")
		return
	}
	if assetClass == "INDEX" || assetClass == "FX" || assetClass == "CASH" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "asset_class not supported for holdings: "+assetClass)
		return
	}
	out, err := h.repo.Create(r.Context(), uid, body.InstrumentID, body.Quantity, body.AvgCost, body.OpenedAt, body.Note)
	if err != nil {
		if errors.Is(err, ErrHoldingConflict) {
			writeError(w, http.StatusConflict, "CONFLICT", "holding already exists for this instrument; use PATCH to update")
			return
		}
		slog.Error("holding create failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "create failed")
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// PATCH /v1/holdings/{id}
func (h *HoldingHandler) Patch(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "id required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var patch map[string]any
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if v, ok := patch["quantity"].(float64); ok && v <= 0 {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "quantity must be > 0")
		return
	}
	if v, ok := patch["avg_cost"].(float64); ok && v < 0 {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "avg_cost must be >= 0")
		return
	}
	out, err := h.repo.Update(r.Context(), uid, id, patch)
	if err != nil {
		if errors.Is(err, ErrHoldingNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "holding not found")
			return
		}
		slog.Error("holding update failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "update failed")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// DELETE /v1/holdings/{id}
func (h *HoldingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "id required")
		return
	}
	if err := h.repo.Delete(r.Context(), uid, id); err != nil {
		if errors.Is(err, ErrHoldingNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "holding not found")
			return
		}
		slog.Error("holding delete failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "delete failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// enrichHoldings는 raw List 결과에 KRW 환산·수익률·비중을 채워 반환.
func enrichHoldings(items []models.HoldingEnriched, rates map[string]float64) []models.HoldingEnriched {
	totalKRW := 0.0
	for i := range items {
		mv := items[i].Quantity * items[i].CurrentPrice
		cb := items[i].Quantity * items[i].AvgCost
		items[i].MarketValue = mv
		items[i].MarketValueKRW = ToKRW(mv, items[i].Currency, rates)
		items[i].CostBasisKRW = ToKRW(cb, items[i].Currency, rates)
		if cb > 0 && items[i].CurrentPrice > 0 {
			items[i].PnLPct = (mv - cb) / cb * 100.0
		}
		items[i].PnLKRW = items[i].MarketValueKRW - items[i].CostBasisKRW
		totalKRW += items[i].MarketValueKRW
	}
	if totalKRW > 0 {
		for i := range items {
			items[i].WeightPct = items[i].MarketValueKRW / totalKRW * 100.0
		}
	}
	return items
}
```

- [ ] **Step 2: enrichment 단위 테스트 (fake repo)**

`apps/api/internal/handlers/holdings_test.go`:
```go
package handlers

import (
	"testing"

	"github.com/quotient/quotient/apps/api/internal/models"
)

func TestEnrichHoldings(t *testing.T) {
	rates := map[string]float64{"KRW": 1.0, "USD": 1380.0}
	items := []models.HoldingEnriched{
		{
			Holding:      models.Holding{Quantity: 10, AvgCost: 70000},
			Currency:     "KRW",
			CurrentPrice: 80000,
		},
		{
			Holding:      models.Holding{Quantity: 5, AvgCost: 150},
			Currency:     "USD",
			CurrentPrice: 180,
		},
	}
	got := enrichHoldings(items, rates)

	// holding 0: KR, mv=800000, cb=700000, pnl_pct ≈ 14.2857
	if got[0].MarketValueKRW != 800000 {
		t.Errorf("KR market_value_krw: got %v want 800000", got[0].MarketValueKRW)
	}
	if got[0].PnLPct < 14.28 || got[0].PnLPct > 14.29 {
		t.Errorf("KR pnl_pct: got %v want ~14.2857", got[0].PnLPct)
	}

	// holding 1: USD mv=900, krw=900*1380=1242000
	if got[1].MarketValueKRW != 1242000 {
		t.Errorf("US market_value_krw: got %v want 1242000", got[1].MarketValueKRW)
	}

	// 비중: total = 800000 + 1242000 = 2042000
	total := 800000.0 + 1242000.0
	wantW0 := 800000 / total * 100
	if got[0].WeightPct < wantW0-0.01 || got[0].WeightPct > wantW0+0.01 {
		t.Errorf("KR weight: got %v want ~%v", got[0].WeightPct, wantW0)
	}
}
```

Run: `cd apps/api && go test ./internal/handlers/ -run TestEnrichHoldings -v`
Expected: PASS.

- [ ] **Step 3: HTTP 핸들러 통합 테스트 (in-memory fake repo)**

`apps/api/internal/handlers/holdings_test.go` 같은 파일에 추가:
```go
import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/quotient/quotient/apps/api/internal/middleware"
)

type fakeHoldingRepo struct {
	createErr error
	deleteErr error
	created   *models.Holding
}

func (f *fakeHoldingRepo) List(ctx context.Context, userID string) ([]models.HoldingEnriched, error) {
	return []models.HoldingEnriched{}, nil
}
func (f *fakeHoldingRepo) Create(ctx context.Context, userID, instrumentID string, qty, avgCost float64, openedAt *string, note *string) (*models.Holding, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	h := &models.Holding{ID: "fake-id", InstrumentID: instrumentID, Quantity: qty, AvgCost: avgCost}
	f.created = h
	return h, nil
}
func (f *fakeHoldingRepo) Update(ctx context.Context, userID, id string, patch map[string]any) (*models.Holding, error) {
	return &models.Holding{ID: id}, nil
}
func (f *fakeHoldingRepo) Delete(ctx context.Context, userID, id string) error {
	return f.deleteErr
}

func reqWithUID(method, target, body, uid string) *http.Request {
	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r = r.WithContext(middleware.WithUserID(r.Context(), uid))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestHoldingCreate_Validation(t *testing.T) {
	repo := &fakeHoldingRepo{}
	h := NewHoldingHandler(repo, nil)

	cases := []struct {
		name string
		body string
		want int
	}{
		{"missing instrument", `{"quantity":1,"avg_cost":100}`, http.StatusUnprocessableEntity},
		{"zero quantity", `{"instrument_id":"x","quantity":0,"avg_cost":100}`, http.StatusUnprocessableEntity},
		{"negative avg_cost", `{"instrument_id":"x","quantity":1,"avg_cost":-1}`, http.StatusUnprocessableEntity},
		{"no auth", `{"instrument_id":"x","quantity":1,"avg_cost":100}`, http.StatusUnauthorized},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var r *http.Request
			if c.name == "no auth" {
				r = httptest.NewRequest(http.MethodPost, "/v1/holdings", strings.NewReader(c.body))
			} else {
				r = reqWithUID(http.MethodPost, "/v1/holdings", c.body, "user-1")
			}
			w := httptest.NewRecorder()
			h.Create(w, r)
			if w.Code != c.want {
				t.Errorf("got %d want %d, body=%s", w.Code, c.want, w.Body.String())
			}
		})
	}
}

func TestHoldingCreate_OK(t *testing.T) {
	repo := &fakeHoldingRepo{}
	h := NewHoldingHandler(repo, nil)
	body := `{"instrument_id":"abc","quantity":10,"avg_cost":70000}`
	r := reqWithUID(http.MethodPost, "/v1/holdings", body, "user-1")
	w := httptest.NewRecorder()
	h.Create(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("got %d, body=%s", w.Code, w.Body.String())
	}
	var got models.Holding
	if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.InstrumentID != "abc" || got.Quantity != 10 {
		t.Errorf("unexpected response: %+v", got)
	}
}

func TestHoldingCreate_Conflict(t *testing.T) {
	repo := &fakeHoldingRepo{createErr: ErrHoldingConflict}
	h := NewHoldingHandler(repo, nil)
	body := `{"instrument_id":"abc","quantity":10,"avg_cost":70000}`
	r := reqWithUID(http.MethodPost, "/v1/holdings", body, "user-1")
	w := httptest.NewRecorder()
	h.Create(w, r)
	if w.Code != http.StatusConflict {
		t.Errorf("got %d want 409", w.Code)
	}
}

func TestHoldingDelete_NotFound(t *testing.T) {
	repo := &fakeHoldingRepo{deleteErr: ErrHoldingNotFound}
	h := NewHoldingHandler(repo, nil)
	r := reqWithUID(http.MethodDelete, "/v1/holdings/abc", "", "user-1")
	// chi URL param 주입
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	r = r.WithContext(middleware.WithUserID(r.Context(), "user-1"))

	w := httptest.NewRecorder()
	h.Delete(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("got %d want 404", w.Code)
	}
}

// suppress unused
var _ = bytes.NewReader
```

> **참고**: `middleware.WithUserID`가 기존 코드에 없을 수 있다. 있으면 그대로 사용, 없으면 `apps/api/internal/middleware/auth.go`에서 user context key를 확인하고 테스트용 helper를 추가하라(테스트 setup에 한정). 기존 `profiles_test.go`가 어떻게 uid를 주입하는지 패턴 확인.

- [ ] **Step 4: 테스트 실행**

Run: `cd apps/api && go test ./internal/handlers/ -v`
Expected: 모든 holdings 테스트 PASS + 기존 profiles/instruments/market 테스트 회귀 없음.

- [ ] **Step 5: 커밋**

```bash
git add apps/api/internal/handlers/holdings.go apps/api/internal/handlers/holdings_test.go
git commit -m "feat(api): holdings CRUD 핸들러 (검증 + enrichment + 테스트)"
```

---

## Task 6: watchlist repo + handler + 테스트

**Files:**
- Create: `apps/api/internal/handlers/watchlist_repo_pg.go`
- Create: `apps/api/internal/handlers/watchlist.go`
- Create: `apps/api/internal/handlers/watchlist_test.go`

watchlist는 단순. holdings 패턴을 축소 적용.

- [ ] **Step 1: repo 작성**

```go
package handlers

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/models"
)

var ErrWatchlistConflict = errors.New("watchlist item already exists")
var ErrWatchlistNotFound = errors.New("watchlist item not found")

type WatchlistRepo interface {
	List(ctx context.Context, userID string) ([]models.WatchlistItem, error)
	Add(ctx context.Context, userID, instrumentID string) error
	Remove(ctx context.Context, userID, instrumentID string) error
}

type PgWatchlistRepo struct {
	pool *pgxpool.Pool
}

func NewPgWatchlistRepo(pool *pgxpool.Pool) *PgWatchlistRepo {
	return &PgWatchlistRepo{pool: pool}
}

func (r *PgWatchlistRepo) List(ctx context.Context, userID string) ([]models.WatchlistItem, error) {
	rows, err := r.pool.Query(ctx, `
		select
		  w.instrument_id::text, w.added_at,
		  i.symbol, i.exchange, i.name, i.asset_class, i.currency,
		  coalesce(q.price, 0)::float8, coalesce(q.change_pct, 0)::float8
		from public.watchlist w
		join public.instruments i on i.id = w.instrument_id
		left join public.quotes q on q.instrument_id = w.instrument_id
		where w.user_id = $1
		order by w.added_at desc
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.WatchlistItem
	for rows.Next() {
		var x models.WatchlistItem
		if err := rows.Scan(&x.InstrumentID, &x.AddedAt, &x.Symbol, &x.Exchange, &x.Name, &x.AssetClass, &x.Currency, &x.Price, &x.ChangePct); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *PgWatchlistRepo) Add(ctx context.Context, userID, instrumentID string) error {
	_, err := r.pool.Exec(ctx, `
		insert into public.watchlist (user_id, instrument_id)
		values ($1, $2)
	`, userID, instrumentID)
	if err != nil && strings.Contains(err.Error(), "23505") {
		return ErrWatchlistConflict
	}
	return err
}

func (r *PgWatchlistRepo) Remove(ctx context.Context, userID, instrumentID string) error {
	ct, err := r.pool.Exec(ctx, `delete from public.watchlist where user_id = $1 and instrument_id = $2`, userID, instrumentID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrWatchlistNotFound
	}
	return nil
}
```

- [ ] **Step 2: handler 작성**

```go
package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
)

type WatchlistHandler struct {
	repo WatchlistRepo
}

func NewWatchlistHandler(repo WatchlistRepo) *WatchlistHandler {
	return &WatchlistHandler{repo: repo}
}

func (h *WatchlistHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	items, err := h.repo.List(r.Context(), uid)
	if err != nil {
		slog.Error("watchlist list failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if items == nil {
		items = []models.WatchlistItem{}
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *WatchlistHandler) Add(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var body struct {
		InstrumentID string `json:"instrument_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if body.InstrumentID == "" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "instrument_id required")
		return
	}
	if err := h.repo.Add(r.Context(), uid, body.InstrumentID); err != nil {
		if errors.Is(err, ErrWatchlistConflict) {
			writeError(w, http.StatusConflict, "CONFLICT", "already in watchlist")
			return
		}
		slog.Error("watchlist add failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "add failed")
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *WatchlistHandler) Remove(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	iid := chi.URLParam(r, "instrument_id")
	if iid == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "instrument_id required")
		return
	}
	if err := h.repo.Remove(r.Context(), uid, iid); err != nil {
		if errors.Is(err, ErrWatchlistNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "not in watchlist")
			return
		}
		slog.Error("watchlist remove failed", "user", uid, "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "remove failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 3: 테스트 작성**

`apps/api/internal/handlers/watchlist_test.go`:
```go
package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
)

type fakeWatchlistRepo struct {
	addErr    error
	removeErr error
}

func (f *fakeWatchlistRepo) List(ctx context.Context, userID string) ([]models.WatchlistItem, error) {
	return []models.WatchlistItem{{InstrumentID: "iid-1", Symbol: "KOSPI"}}, nil
}
func (f *fakeWatchlistRepo) Add(ctx context.Context, userID, instrumentID string) error {
	return f.addErr
}
func (f *fakeWatchlistRepo) Remove(ctx context.Context, userID, instrumentID string) error {
	return f.removeErr
}

func TestWatchlistAdd_Conflict(t *testing.T) {
	repo := &fakeWatchlistRepo{addErr: ErrWatchlistConflict}
	h := NewWatchlistHandler(repo)
	body := `{"instrument_id":"abc"}`
	r := httptest.NewRequest(http.MethodPost, "/v1/watchlist", strings.NewReader(body))
	r = r.WithContext(middleware.WithUserID(r.Context(), "user-1"))
	r.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	h.Add(w, r)
	if w.Code != http.StatusConflict {
		t.Errorf("got %d want 409", w.Code)
	}
}

func TestWatchlistRemove_NotFound(t *testing.T) {
	repo := &fakeWatchlistRepo{removeErr: ErrWatchlistNotFound}
	h := NewWatchlistHandler(repo)
	r := httptest.NewRequest(http.MethodDelete, "/v1/watchlist/iid-1", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("instrument_id", "iid-1")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	r = r.WithContext(middleware.WithUserID(r.Context(), "user-1"))

	w := httptest.NewRecorder()
	h.Remove(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("got %d want 404", w.Code)
	}
}
```

Run: `cd apps/api && go test ./internal/handlers/ -run Watchlist -v`
Expected: PASS.

- [ ] **Step 4: 커밋**

```bash
git add apps/api/internal/handlers/watchlist_repo_pg.go apps/api/internal/handlers/watchlist.go apps/api/internal/handlers/watchlist_test.go
git commit -m "feat(api): watchlist 추가·삭제·조회 핸들러 + 테스트"
```

---

## Task 7: 라우터 통합 + main.go 와이어링

**Files:**
- Modify: `apps/api/internal/router/router.go`
- Modify: `apps/api/cmd/server/main.go`

- [ ] **Step 1: router.go에 라우트 7개 추가**

기존 `router.New` 시그니처를 확장:

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
	holdingHandler *handlers.HoldingHandler,
	watchlistHandler *handlers.WatchlistHandler,
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

		r.Get("/v1/holdings", holdingHandler.List)
		r.Post("/v1/holdings", holdingHandler.Create)
		r.Patch("/v1/holdings/{id}", holdingHandler.Patch)
		r.Delete("/v1/holdings/{id}", holdingHandler.Delete)

		r.Get("/v1/watchlist", watchlistHandler.List)
		r.Post("/v1/watchlist", watchlistHandler.Add)
		r.Delete("/v1/watchlist/{instrument_id}", watchlistHandler.Remove)
	})

	return r
}
```

- [ ] **Step 2: main.go 와이어링 수정**

`apps/api/cmd/server/main.go`의 의존성 와이어링 섹션 (현재 `instrumentHandler := handlers.NewInstrumentHandler(...)` 줄 다음, `readyz := ...` 줄 이전):

```go
// 추가 (기존 instrumentRepo/Handler 줄 바로 아래):
holdingRepo := handlers.NewPgHoldingRepo(pool)
holdingHandler := handlers.NewHoldingHandler(holdingRepo, pool) // pool은 FX 환산용으로 두 번째 인자
watchlistRepo := handlers.NewPgWatchlistRepo(pool)
watchlistHandler := handlers.NewWatchlistHandler(watchlistRepo)
```

`srv := &http.Server{...}` 블록의 `Handler:` 필드를 다음으로 교체:

```go
Handler: router.New(
    verifier, cfg.CORSOrigin,
    profileHandler, marketHandler, instrumentHandler,
    holdingHandler, watchlistHandler,
    readyz,
),
```

> **주의**: `pool` 변수는 같은 함수에서 `pool, err := db.New(ctx, cfg.DatabaseURL)`로 이미 정의됨 (`main.go:41`). `NewHoldingHandler`의 두 번째 인자에 그대로 재사용.

- [ ] **Step 3: 빌드 + 회귀 테스트**

Run: `cd apps/api && go build ./... && go test ./...`
Expected: 빌드 성공 + 전체 테스트 PASS.

- [ ] **Step 4: 커밋**

```bash
git add apps/api/internal/router/router.go apps/api/cmd/server/main.go
git commit -m "feat(api): holdings·watchlist 라우트 7개 등록"
```

---

## Task 8: cron quotes job — holdings ∪ watchlist union 확장

**Files:**
- Modify: `apps/api/internal/schedule/jobs_quotes.go`

폴링 대상을 확장. INDEX는 그대로(KR/US 장중만), holdings·watchlist는 instrument의 asset_class·exchange를 기준으로 장중 판정. dedup은 SQL `UNION`이 자동 처리.

- [ ] **Step 1: jobs_quotes.go의 SQL 확장**

기존 `JobUpdateIndexQuotes`의 `select i.id::text, i.symbol, i.exchange, q.updated_at from instruments...` 쿼리를 다음으로 교체:

```go
// 1) 폴링 대상 = INDEX ∪ holdings ∪ watchlist (dedup via UNION).
//    asset_class도 함께 가져와 장중 판정에 활용.
rs, err := d.Pool.Query(ctx, `
	with targets as (
		select id from public.instruments where is_active = true and asset_class = 'INDEX'
		union
		select instrument_id as id from public.holdings
		union
		select instrument_id as id from public.watchlist
	)
	select i.id::text, i.symbol, i.exchange, i.asset_class, q.updated_at
	from targets t
	join public.instruments i on i.id = t.id
	left join public.quotes q on q.instrument_id = i.id
	where i.is_active = true
`)
```

- [ ] **Step 2: row struct + 장중 판정 로직 확장**

```go
type row struct {
	id, symbol, exchange, assetClass string
	updatedAt                        *time.Time
}
// ...
for rs.Next() {
	var r row
	if err := rs.Scan(&r.id, &r.symbol, &r.exchange, &r.assetClass, &r.updatedAt); err != nil {
		return fmt.Errorf("scan: %w", err)
	}
	rows = append(rows, r)
}
```

장중 판정 분기 확장:

```go
for _, r := range rows {
	if r.updatedAt != nil && now.Sub(*r.updatedAt) < quotesCacheTTL {
		continue
	}

	// 장중 판정:
	//   INDEX는 symbol로 분기 (기존)
	//   KR_STOCK/ETF(KRX 상장)는 KR 장중
	//   US_STOCK은 US 장중
	//   CASH/FX는 항상 fetch (FX는 별도 job이 처리하지만 quotes에도 동기화)
	switch r.assetClass {
	case "INDEX":
		if isKRIndex(r.symbol) && !IsKRMarketOpen(now) { continue }
		if isUSIndex(r.symbol) && !IsUSMarketOpen(now) { continue }
	case "KR_STOCK", "ETF":
		// ETF는 KR/US 모두 가능. exchange 기반 판정.
		if isKRExchange(r.exchange) && !IsKRMarketOpen(now) { continue }
		if isUSExchange(r.exchange) && !IsUSMarketOpen(now) { continue }
	case "US_STOCK":
		if !IsUSMarketOpen(now) { continue }
	case "CASH":
		continue // CASH는 시세 없음
	case "FX":
		// FX 별도 잡이 5분 주기 — quotes 폴링에서는 skip
		continue
	}

	ysym := IndexYahooSymbol(r.symbol, r.exchange)
	if ysym == "" {
		// INDEX 매핑이 아니면 일반 종목 Yahoo 심볼 사용
		ysym = StockYahooSymbol(r.symbol, r.exchange)
	}
	if ysym == "" {
		continue
	}
	// ... 기존 d.Yahoo.FetchQuote 호출 유지
}
```

> **참고**: `isKRExchange`, `isUSExchange`, `StockYahooSymbol`이 없다면 `yahoo_symbols.go`에 추가하라. `yahoo_symbols.go`의 기존 함수 시그니처를 먼저 확인하고, 다음 패턴으로 단순 분기:
> - `isKRExchange`: `exchange in {"KOSPI", "KOSDAQ"}` → true
> - `isUSExchange`: `exchange in {"NYSE", "NASDAQ", "AMEX"}` → true
> - `StockYahooSymbol`: KR이면 `symbol + ".KS"` 또는 `.KQ`, US면 `symbol` 그대로

- [ ] **Step 3: yahoo_symbols.go 보강 (필요 시)**

`apps/api/internal/schedule/yahoo_symbols.go`를 읽어 위 helper들이 있는지 확인. 없다면 추가:

```go
// StockYahooSymbol은 일반 종목의 Yahoo 심볼을 반환.
// KR: "005930" + "KOSPI" → "005930.KS", "KOSDAQ" → ".KQ"
// US: 그대로
func StockYahooSymbol(symbol, exchange string) string {
	switch exchange {
	case "KOSPI":
		return symbol + ".KS"
	case "KOSDAQ":
		return symbol + ".KQ"
	case "NYSE", "NASDAQ", "AMEX":
		return symbol
	}
	return ""
}

func isKRExchange(ex string) bool {
	return ex == "KOSPI" || ex == "KOSDAQ"
}

func isUSExchange(ex string) bool {
	return ex == "NYSE" || ex == "NASDAQ" || ex == "AMEX"
}
```

테스트 추가 (`yahoo_symbols_test.go`에):
```go
func TestStockYahooSymbol(t *testing.T) {
	cases := []struct {
		sym, ex, want string
	}{
		{"005930", "KOSPI", "005930.KS"},
		{"035720", "KOSDAQ", "035720.KQ"},
		{"AAPL", "NASDAQ", "AAPL"},
		{"AAPL", "UNKNOWN", ""},
	}
	for _, c := range cases {
		if got := StockYahooSymbol(c.sym, c.ex); got != c.want {
			t.Errorf("StockYahooSymbol(%q,%q) = %q want %q", c.sym, c.ex, got, c.want)
		}
	}
}
```

- [ ] **Step 4: 함수명·로그·주석 갱신**

함수가 INDEX만 처리하지 않으므로 의미 보존을 위해 rename. `apps/api/internal/schedule/jobs_quotes.go`:

```go
// JobUpdateMarketQuotes refreshes quotes for INDEX ∪ holdings ∪ watchlist (dedup).
// 시세 TTL 60초 (spec §10-2). KR/US 장중 판정은 asset_class·exchange 기반.
func JobUpdateMarketQuotes(ctx context.Context, d Deps) error {
    // ... 기존 본문
    slog.Info("market quotes updated", "count", n)
    return nil
}
```

그리고 `apps/api/internal/schedule/cron.go`의 호출도 함께 갱신:

```go
mustAdd(c, "* * * * *", "quotes", func() {
    if err := JobUpdateMarketQuotes(ctx, d); err != nil {
        slog.Error("cron quotes failed", "err", err)
    }
})
```

- [ ] **Step 5: 통합 회귀 테스트 + 빌드**

Run: `cd apps/api && go build ./... && go test ./internal/schedule/...`
Expected: 빌드 성공 + `StockYahooSymbol` 새 테스트 PASS + 기존 `market_hours`, `yahoo_symbols` 테스트 회귀 없음.

- [ ] **Step 6: 커밋**

```bash
git add apps/api/internal/schedule/jobs_quotes.go apps/api/internal/schedule/yahoo_symbols.go apps/api/internal/schedule/yahoo_symbols_test.go
git commit -m "feat(api): quotes 폴링 대상 holdings·watchlist union 확장"
```

---

## Task 9: web — API 클라이언트 + auth-fetch helper

**Files:**
- Create: `apps/web/lib/api/auth-fetch.ts`
- Create: `apps/web/lib/api/holdings.ts`
- Create: `apps/web/lib/api/watchlist.ts`
- Create: `apps/web/lib/api/instruments.ts`

기존 `lib/api/market.ts`는 `fetchTicker(accessToken)` 패턴으로 매번 토큰을 인자로 받는다. W3에서 호출이 늘어나므로 `authFetch` helper로 통합 — 호출자는 토큰을 신경 쓰지 않는다.

- [ ] **Step 1: auth-fetch.ts 작성**

```ts
import { createSupabaseBrowser } from "@/lib/supabase/client";

const API_BASE =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

async function getToken(): Promise<string | null> {
  const supabase = createSupabaseBrowser();
  const { data: { session } } = await supabase.auth.getSession();
  return session?.access_token ?? null;
}

export async function authFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const token = await getToken();
  const headers = new Headers(init.headers);
  if (token) headers.set("Authorization", `Bearer ${token}`);
  if (init.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  return fetch(`${API_BASE}${path}`, { ...init, headers, cache: "no-store" });
}

export type ApiError = { code: string; message: string };

export async function readError(res: Response): Promise<ApiError> {
  try {
    const body = await res.json();
    if (body?.error) return body.error;
    return { code: `HTTP_${res.status}`, message: res.statusText };
  } catch {
    return { code: `HTTP_${res.status}`, message: res.statusText };
  }
}
```

- [ ] **Step 2: holdings.ts 작성**

```ts
import { authFetch, readError } from "./auth-fetch";

export type Holding = {
  id: string;
  instrument_id: string;
  quantity: number;
  avg_cost: number;
  opened_at?: string | null;
  note?: string | null;
  created_at: string;
  updated_at: string;
  symbol: string;
  exchange: string;
  name: string;
  asset_class: string;
  currency: string;
  current_price: number;
  market_value: number;
  market_value_krw: number;
  cost_basis_krw: number;
  pnl_krw: number;
  pnl_pct: number;
  weight_pct: number;
};

export async function listHoldings(): Promise<Holding[]> {
  const res = await authFetch("/v1/holdings");
  if (!res.ok) throw await readError(res);
  return res.json();
}

export async function createHolding(input: {
  instrument_id: string;
  quantity: number;
  avg_cost: number;
  opened_at?: string;
  note?: string;
}): Promise<Holding> {
  const res = await authFetch("/v1/holdings", {
    method: "POST",
    body: JSON.stringify(input),
  });
  if (!res.ok) throw await readError(res);
  return res.json();
}

export async function updateHolding(
  id: string,
  patch: Partial<Pick<Holding, "quantity" | "avg_cost" | "note" | "opened_at">>,
): Promise<Holding> {
  const res = await authFetch(`/v1/holdings/${id}`, {
    method: "PATCH",
    body: JSON.stringify(patch),
  });
  if (!res.ok) throw await readError(res);
  return res.json();
}

export async function deleteHolding(id: string): Promise<void> {
  const res = await authFetch(`/v1/holdings/${id}`, { method: "DELETE" });
  if (!res.ok && res.status !== 204) throw await readError(res);
}
```

- [ ] **Step 3: watchlist.ts 작성**

```ts
import { authFetch, readError } from "./auth-fetch";

export type WatchlistItem = {
  instrument_id: string;
  added_at: string;
  symbol: string;
  exchange: string;
  name: string;
  asset_class: string;
  currency: string;
  price: number;
  change_pct: number;
};

export async function listWatchlist(): Promise<WatchlistItem[]> {
  const res = await authFetch("/v1/watchlist");
  if (!res.ok) throw await readError(res);
  return res.json();
}

export async function addWatchlist(instrument_id: string): Promise<void> {
  const res = await authFetch("/v1/watchlist", {
    method: "POST",
    body: JSON.stringify({ instrument_id }),
  });
  if (!res.ok && res.status !== 201) throw await readError(res);
}

export async function removeWatchlist(instrument_id: string): Promise<void> {
  const res = await authFetch(`/v1/watchlist/${instrument_id}`, { method: "DELETE" });
  if (!res.ok && res.status !== 204) throw await readError(res);
}
```

- [ ] **Step 4: instruments.ts 작성 (검색 래퍼)**

> **선행 작업**: 백엔드 `apps/api/internal/handlers/instruments.go`의 `SearchResult` 타입과 `instruments_repo_pg.go`의 SELECT 절에 `currency`와 `asset_class` 컬럼을 추가하라. `enrichHoldings`에서 currency 분기는 이미 처리되지만 프론트에서 "평단가 (KRW/USD)" 라벨 분기와 CASH·INDEX·FX 필터링 (Critical 2와 정합)에 필요. 백엔드 변경 SQL 한 줄:
> ```go
> // SearchByAlias·SearchByText 양쪽 SELECT에:
> select i.id::text, i.symbol, i.exchange, i.name, i.currency, i.asset_class
> ```
> 그리고 `SearchResult` struct에 `Currency string \`json:"currency"\`` + `AssetClass string \`json:"asset_class"\`` 추가, Scan에도 두 변수 추가.

```ts
import { authFetch, readError } from "./auth-fetch";

export type InstrumentResult = {
  id: string;
  symbol: string;
  exchange: string;
  name: string;
  currency: string;     // "KRW" | "USD"
  asset_class: string;  // "KR_STOCK" | "US_STOCK" | "ETF" | "INDEX" | "FX" | "CASH"
};

export async function searchInstruments(query: string): Promise<InstrumentResult[]> {
  if (!query.trim()) return [];
  const res = await authFetch(`/v1/instruments/search?q=${encodeURIComponent(query)}`);
  if (!res.ok) throw await readError(res);
  return res.json();
}

export async function selectInstrument(query: string, instrument_id: string): Promise<void> {
  await authFetch("/v1/instruments/select", {
    method: "POST",
    body: JSON.stringify({ query, instrument_id }),
  });
  // 학습 실패는 silent — 검색 결과 사용에 영향 없음
}
```

- [ ] **Step 5: 빌드 확인**

Run: `cd apps/web && npx tsc --noEmit`
Expected: 타입 에러 없음.

- [ ] **Step 6: 커밋**

```bash
git add apps/web/lib/api/auth-fetch.ts apps/web/lib/api/holdings.ts apps/web/lib/api/watchlist.ts apps/web/lib/api/instruments.ts
git commit -m "feat(web): holdings·watchlist·instruments API 클라이언트 + authFetch helper"
```

---

## Task 10: 포트폴리오 페이지 — 보유 테이블 (조회·표시)

**Files:**
- Create: `apps/web/app/app/portfolio/page.tsx`
- Create: `apps/web/components/portfolio/HoldingsTable.tsx`

스펙 §6: 종목·수량·평단가·현재가·평가액·수익률·비중·통화 컬럼. 정렬·검색은 클라이언트 측 처리(MVP 사용자 수 기준 100건 이하 가정). CRUD 모달은 다음 Task에서.

- [ ] **Step 1: page.tsx (서버 컴포넌트, 인증 가드 + client 컴포넌트 호스팅)**

```tsx
import { HoldingsTable } from "@/components/portfolio/HoldingsTable";

export default function PortfolioPage() {
  return (
    <div className="p-6 md:p-8">
      <header className="flex items-baseline justify-between mb-6">
        <div>
          <h1 className="font-mono text-2xl">포트폴리오</h1>
          <p className="text-fg-muted text-sm mt-1">보유 자산. 시세 지연 15분.</p>
        </div>
      </header>
      <HoldingsTable />
    </div>
  );
}
```

- [ ] **Step 2: HoldingsTable.tsx 작성**

```tsx
"use client";

import { useEffect, useState } from "react";
import { listHoldings, type Holding } from "@/lib/api/holdings";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";

type SortKey = "weight_pct" | "market_value_krw" | "pnl_pct" | "symbol";

export function HoldingsTable() {
  const [holdings, setHoldings] = useState<Holding[] | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [query, setQuery] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>("weight_pct");

  async function load() {
    try {
      const data = await listHoldings();
      setHoldings(data);
      setErr(null);
    } catch (e: any) {
      setErr(e?.message ?? "로드 실패");
      setHoldings([]);
    }
  }

  useEffect(() => {
    load();
  }, []);

  if (holdings === null) {
    return (
      <div className="space-y-2">
        {[1, 2, 3].map((i) => <Skeleton key={i} className="h-10 w-full" />)}
      </div>
    );
  }

  const filtered = holdings.filter((h) =>
    !query || h.symbol.toLowerCase().includes(query.toLowerCase()) || h.name.toLowerCase().includes(query.toLowerCase())
  );
  const sorted = [...filtered].sort((a, b) => {
    if (sortKey === "symbol") return a.symbol.localeCompare(b.symbol);
    return (b[sortKey] as number) - (a[sortKey] as number);
  });

  if (holdings.length === 0) {
    return (
      <div className="border border-line p-12 text-center text-fg-muted">
        <p className="font-mono">보유 자산이 없습니다.</p>
        <p className="text-xs mt-2">우상단 [+ 추가] 버튼으로 첫 종목을 등록하세요.</p>
        {/* 추가 버튼은 다음 Task에서 통합 */}
      </div>
    );
  }

  return (
    <div>
      <div className="flex items-center gap-2 mb-4">
        <input
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="종목·코드 검색"
          className="bg-bg-deep border border-line px-3 py-1.5 text-sm font-mono w-64"
        />
        <select
          value={sortKey}
          onChange={(e) => setSortKey(e.target.value as SortKey)}
          className="bg-bg-deep border border-line px-2 py-1.5 text-sm font-mono"
        >
          <option value="weight_pct">비중 ↓</option>
          <option value="market_value_krw">평가액 ↓</option>
          <option value="pnl_pct">수익률 ↓</option>
          <option value="symbol">종목 A→Z</option>
        </select>
        {err && <span className="text-bb-down text-xs ml-auto">{err}</span>}
      </div>

      <div className="border border-line overflow-x-auto">
        <table className="w-full text-sm font-mono">
          <thead className="border-b border-line bg-bg-deep text-fg-muted text-xs">
            <tr>
              <th className="text-left px-3 py-2">종목</th>
              <th className="text-right px-3 py-2">수량</th>
              <th className="text-right px-3 py-2">평단가</th>
              <th className="text-right px-3 py-2">현재가</th>
              <th className="text-right px-3 py-2">평가액 (KRW)</th>
              <th className="text-right px-3 py-2">손익 (KRW)</th>
              <th className="text-right px-3 py-2">수익률</th>
              <th className="text-right px-3 py-2">비중</th>
              <th className="px-3 py-2"></th>
            </tr>
          </thead>
          <tbody>
            {sorted.map((h) => (
              <tr key={h.id} className="border-b border-line/50 hover:bg-bg-deep/50">
                <td className="px-3 py-2">
                  <div>{h.symbol}</div>
                  <div className="text-xs text-fg-muted">{h.name}</div>
                </td>
                <td className="text-right px-3 py-2">{h.quantity}</td>
                <td className="text-right px-3 py-2">{h.avg_cost.toLocaleString()}</td>
                <td className="text-right px-3 py-2">{h.current_price > 0 ? h.current_price.toLocaleString() : "—"}</td>
                <td className="text-right px-3 py-2">{Math.round(h.market_value_krw).toLocaleString()}</td>
                <td className={`text-right px-3 py-2 ${h.pnl_krw >= 0 ? "text-bb-up" : "text-bb-down"}`}>
                  {Math.round(h.pnl_krw).toLocaleString()}
                </td>
                <td className={`text-right px-3 py-2 ${h.pnl_pct >= 0 ? "text-bb-up" : "text-bb-down"}`}>
                  {h.pnl_pct.toFixed(2)}%
                </td>
                <td className="text-right px-3 py-2">{h.weight_pct.toFixed(1)}%</td>
                <td className="px-3 py-2 text-right">
                  {/* 수정/삭제 액션은 다음 Task에서 추가 */}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
```

> **주의**: `text-bb-up`, `text-bb-down`, `border-line`, `bg-bg-deep` 등은 W1 Tailwind v4 토큰. 기존 `TopTicker.tsx`·`AppShell.tsx`에서 사용되는 클래스명 확인 후 동일 이름 사용.

- [ ] **Step 3: 사이드바 메뉴 확인**

`apps/web/components/shell/Sidebar.tsx`를 읽어 `/app/portfolio` 링크가 이미 있는지 확인. 없으면 추가(W1에서 placeholder로 있을 가능성 높음). 있다면 변경 없음.

- [ ] **Step 4: 로컬 브라우저 확인**

Run:
```bash
cd apps/web && npm run dev
```
Expected: `http://localhost:3000/app/portfolio` 접속 → 빈 상태 메시지 표시(holdings 없음 + 인증된 상태). 빌드 에러 없음.

- [ ] **Step 5: 커밋**

```bash
git add apps/web/app/app/portfolio/page.tsx apps/web/components/portfolio/HoldingsTable.tsx apps/web/components/shell/Sidebar.tsx
git commit -m "feat(web): 포트폴리오 페이지 + 보유 테이블 (조회·정렬·검색)"
```

---

## Task 11: 포트폴리오 — 종목 검색 입력 + 추가 모달

**Files:**
- Create: `apps/web/components/portfolio/InstrumentSearchInput.tsx`
- Create: `apps/web/components/portfolio/AddHoldingDialog.tsx`
- Modify: `apps/web/components/portfolio/HoldingsTable.tsx` (추가 버튼·모달 호스팅)

`InstrumentSearchInput`은 watchlist Task에서도 재사용. 300ms 디바운스 검색 + 클릭 선택 → `/v1/instruments/select` 학습 호출.

- [ ] **Step 1: InstrumentSearchInput.tsx 작성**

```tsx
"use client";

import { useEffect, useState } from "react";
import { searchInstruments, selectInstrument, type InstrumentResult } from "@/lib/api/instruments";

export function InstrumentSearchInput({
  onSelect,
  placeholder = "종목 검색 (예: 삼성전자, AAPL)",
}: {
  onSelect: (inst: InstrumentResult) => void;
  placeholder?: string;
}) {
  const [query, setQuery] = useState("");
  const [results, setResults] = useState<InstrumentResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);

  useEffect(() => {
    if (!query.trim()) {
      setResults([]);
      return;
    }
    const t = setTimeout(async () => {
      setLoading(true);
      try {
        const r = await searchInstruments(query);
        setResults(r);
        setOpen(true);
      } catch {
        setResults([]);
      } finally {
        setLoading(false);
      }
    }, 300);
    return () => clearTimeout(t);
  }, [query]);

  function handlePick(inst: InstrumentResult) {
    // holdings 대상이 아닌 자산군은 선택 차단 (Critical 2 — 백엔드 422와 정합)
    if (inst.asset_class === "INDEX" || inst.asset_class === "FX" || inst.asset_class === "CASH") {
      // 부모가 선택을 막을 수 있도록 onSelect를 호출하지 않고 사용자 경고만
      return;
    }
    void selectInstrument(query, inst.id); // 학습은 fire-and-forget
    setOpen(false);
    setQuery(`${inst.symbol} — ${inst.name}`);
    onSelect(inst);
  }

  return (
    <div className="relative">
      <input
        value={query}
        onChange={(e) => { setQuery(e.target.value); setOpen(true); }}
        placeholder={placeholder}
        className="bg-bg-deep border border-line px-3 py-1.5 text-sm font-mono w-full"
        onFocus={() => results.length && setOpen(true)}
      />
      {open && (results.length > 0 || loading) && (
        <div className="absolute z-10 mt-1 w-full border border-line bg-bg-deep max-h-64 overflow-auto">
          {loading && <div className="px-3 py-2 text-xs text-fg-muted">검색 중…</div>}
          {results.map((r) => (
            <button
              key={r.id}
              type="button"
              onClick={() => handlePick(r)}
              className="w-full text-left px-3 py-2 hover:bg-line/30 font-mono text-sm border-b border-line/50 last:border-b-0"
            >
              <span>{r.symbol}</span>
              <span className="text-fg-muted text-xs ml-2">{r.exchange}</span>
              <div className="text-xs text-fg-muted">{r.name}</div>
            </button>
          ))}
          {!loading && results.length === 0 && query && (
            <div className="px-3 py-2 text-xs text-fg-muted">결과 없음</div>
          )}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 2: AddHoldingDialog.tsx 작성**

```tsx
"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { InstrumentSearchInput } from "./InstrumentSearchInput";
import { createHolding } from "@/lib/api/holdings";
import type { InstrumentResult } from "@/lib/api/instruments";

export function AddHoldingDialog({
  open, onOpenChange, onAdded,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onAdded: () => void;
}) {
  const [inst, setInst] = useState<InstrumentResult | null>(null);
  const [quantity, setQuantity] = useState("");
  const [avgCost, setAvgCost] = useState("");
  const [openedAt, setOpenedAt] = useState("");
  const [note, setNote] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  function reset() {
    setInst(null); setQuantity(""); setAvgCost(""); setOpenedAt(""); setNote(""); setErr(null);
  }

  async function submit() {
    if (!inst) { setErr("종목을 선택해주세요"); return; }
    const q = parseFloat(quantity);
    const c = parseFloat(avgCost);
    if (!(q > 0)) { setErr("수량은 0보다 커야 합니다"); return; }
    if (!(c >= 0)) { setErr("평단가는 0 이상이어야 합니다"); return; }
    setSubmitting(true);
    setErr(null);
    try {
      await createHolding({
        instrument_id: inst.id,
        quantity: q,
        avg_cost: c,
        opened_at: openedAt || undefined,
        note: note || undefined,
      });
      onAdded();
      reset();
      onOpenChange(false);
    } catch (e: any) {
      if (e?.code === "CONFLICT") {
        setErr("이미 등록된 종목입니다. 수정으로 진행해주세요.");
      } else {
        setErr(e?.message ?? "추가 실패");
      }
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) reset(); onOpenChange(v); }}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="font-mono">보유 자산 추가</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div>
            <Label className="text-xs font-mono">종목</Label>
            <InstrumentSearchInput onSelect={setInst} />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <Label className="text-xs font-mono">수량</Label>
              <Input value={quantity} onChange={(e) => setQuantity(e.target.value)} type="number" step="any" />
            </div>
            <div>
              <Label className="text-xs font-mono">평단가 ({inst?.currency ?? "통화"})</Label>
              <Input value={avgCost} onChange={(e) => setAvgCost(e.target.value)} type="number" step="any" />
            </div>
          </div>
          <div>
            <Label className="text-xs font-mono">매수일 (선택)</Label>
            <Input value={openedAt} onChange={(e) => setOpenedAt(e.target.value)} type="date" />
          </div>
          <div>
            <Label className="text-xs font-mono">메모 (선택)</Label>
            <Input value={note} onChange={(e) => setNote(e.target.value)} maxLength={200} />
          </div>
          {err && <p className="text-bb-down text-xs font-mono">{err}</p>}
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>취소</Button>
          <Button onClick={submit} disabled={submitting}>{submitting ? "추가 중…" : "추가"}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
```

- [ ] **Step 3: HoldingsTable.tsx에 추가 버튼·모달 통합**

테이블 컴포넌트 상단 헤더에 버튼을 호스팅하고 `AddHoldingDialog`를 띄운다. 빈 상태에도 버튼 노출.

`HoldingsTable.tsx` 수정 — useState 추가 + 헤더 변경:
```tsx
import { AddHoldingDialog } from "./AddHoldingDialog";
// ... 컴포넌트 내부:
const [addOpen, setAddOpen] = useState(false);

// 빈 상태 div 안의 placeholder 주석을 다음으로 교체:
<Button className="mt-4" onClick={() => setAddOpen(true)}>+ 첫 종목 추가</Button>

// 검색·정렬 컨트롤 줄에 우측 버튼 추가 (ml-auto err 옆 또는 별도):
<Button onClick={() => setAddOpen(true)} className="ml-auto">+ 추가</Button>

// 컴포넌트 return 맨 끝에:
<AddHoldingDialog open={addOpen} onOpenChange={setAddOpen} onAdded={load} />
```

> **주의**: 빈 상태와 정상 상태 둘 다 추가 버튼이 동작해야 하므로 두 분기 모두에 `<AddHoldingDialog ... />` 마운트가 도달하는지 확인. 빈 상태 return을 별도로 두면 모달이 unmount된다 — return 구조를 fragment로 감싸 한 번에 렌더하거나 두 분기 모두에 다이얼로그를 포함한다. 권장: 빈 상태도 컴포넌트 본체 내부에서 조건부 렌더로 처리.

- [ ] **Step 4: 브라우저 검증**

Run: `cd apps/web && npm run dev`
Expected: `/app/portfolio` → [+ 추가] 클릭 → 모달 열림 → "삼성전자" 검색 → 결과 클릭 → 수량 10, 평단가 70000 입력 → 추가 → 모달 닫힘 → 테이블에 행 출현. (instrument 시드에 005930 없으면 KOSPI 종목 마스터 백필 cron 1회 또는 수동 instruments seed 필요)

- [ ] **Step 5: 커밋**

```bash
git add apps/web/components/portfolio/
git commit -m "feat(web): 보유 자산 추가 모달 + 디바운스 종목 검색"
```

---

## Task 12: 포트폴리오 — 수정·삭제

**Files:**
- Create: `apps/web/components/portfolio/EditHoldingDialog.tsx`
- Create: `apps/web/components/portfolio/DeleteConfirmDialog.tsx`
- Modify: `apps/web/components/portfolio/HoldingsTable.tsx` (행 액션 메뉴)

수정은 quantity/avg_cost/note만(스펙 §3 — instrument_id 변경은 삭제 후 재추가). 삭제는 확인 다이얼로그 1회.

- [ ] **Step 1: EditHoldingDialog.tsx 작성**

```tsx
"use client";

import { useState, useEffect } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { updateHolding, type Holding } from "@/lib/api/holdings";

export function EditHoldingDialog({
  holding, open, onOpenChange, onSaved,
}: {
  holding: Holding | null;
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onSaved: () => void;
}) {
  const [quantity, setQuantity] = useState("");
  const [avgCost, setAvgCost] = useState("");
  const [note, setNote] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    if (holding) {
      setQuantity(String(holding.quantity));
      setAvgCost(String(holding.avg_cost));
      setNote(holding.note ?? "");
    }
  }, [holding]);

  if (!holding) return null;

  async function submit() {
    const q = parseFloat(quantity);
    const c = parseFloat(avgCost);
    if (!(q > 0)) { setErr("수량은 0보다 커야 합니다"); return; }
    if (!(c >= 0)) { setErr("평단가는 0 이상이어야 합니다"); return; }
    setSubmitting(true);
    try {
      await updateHolding(holding!.id, { quantity: q, avg_cost: c, note: note || null });
      onSaved();
      onOpenChange(false);
    } catch (e: any) {
      setErr(e?.message ?? "수정 실패");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="font-mono">{holding.symbol} 수정</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <div className="grid grid-cols-2 gap-3">
            <div>
              <Label className="text-xs font-mono">수량</Label>
              <Input value={quantity} onChange={(e) => setQuantity(e.target.value)} type="number" step="any" />
            </div>
            <div>
              <Label className="text-xs font-mono">평단가</Label>
              <Input value={avgCost} onChange={(e) => setAvgCost(e.target.value)} type="number" step="any" />
            </div>
          </div>
          <div>
            <Label className="text-xs font-mono">메모</Label>
            <Input value={note} onChange={(e) => setNote(e.target.value)} maxLength={200} />
          </div>
          {err && <p className="text-bb-down text-xs font-mono">{err}</p>}
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>취소</Button>
          <Button onClick={submit} disabled={submitting}>{submitting ? "저장 중…" : "저장"}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
```

- [ ] **Step 2: DeleteConfirmDialog.tsx 작성**

```tsx
"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { deleteHolding, type Holding } from "@/lib/api/holdings";

export function DeleteConfirmDialog({
  holding, open, onOpenChange, onDeleted,
}: {
  holding: Holding | null;
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onDeleted: () => void;
}) {
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  if (!holding) return null;

  async function submit() {
    setSubmitting(true);
    try {
      await deleteHolding(holding!.id);
      onDeleted();
      onOpenChange(false);
    } catch (e: any) {
      setErr(e?.message ?? "삭제 실패");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="font-mono">{holding.symbol} 삭제</DialogTitle>
        </DialogHeader>
        <p className="text-sm font-mono">이 보유 자산을 삭제합니다. 되돌릴 수 없습니다.</p>
        {err && <p className="text-bb-down text-xs font-mono mt-2">{err}</p>}
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>취소</Button>
          <Button variant="destructive" onClick={submit} disabled={submitting}>{submitting ? "삭제 중…" : "삭제"}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
```

> **참고**: `<Button variant="destructive">`이 기존 shadcn Button에 있는지 확인. 없으면 `variant="ghost"` + `className="text-bb-down"` 조합으로 대체.

- [ ] **Step 3: HoldingsTable.tsx에 행 액션 통합**

각 `<tr>`의 마지막 td를 다음으로:
```tsx
<td className="px-3 py-2 text-right">
  <button onClick={() => { setEditTarget(h); setEditOpen(true); }} className="text-xs text-fg-muted hover:text-fg mr-2">수정</button>
  <button onClick={() => { setDeleteTarget(h); setDeleteOpen(true); }} className="text-xs text-fg-muted hover:text-bb-down">삭제</button>
</td>
```

컴포넌트 상태 추가:
```tsx
const [editOpen, setEditOpen] = useState(false);
const [editTarget, setEditTarget] = useState<Holding | null>(null);
const [deleteOpen, setDeleteOpen] = useState(false);
const [deleteTarget, setDeleteTarget] = useState<Holding | null>(null);
```

return 끝에 두 모달 마운트:
```tsx
<EditHoldingDialog holding={editTarget} open={editOpen} onOpenChange={setEditOpen} onSaved={load} />
<DeleteConfirmDialog holding={deleteTarget} open={deleteOpen} onOpenChange={setDeleteOpen} onDeleted={load} />
```

- [ ] **Step 4: 브라우저 검증**

Run: dev 서버. `/app/portfolio`에서 행 [수정] → 수량 변경 → 저장 → 테이블 갱신. [삭제] → 확인 → 행 사라짐.

- [ ] **Step 5: 커밋**

```bash
git add apps/web/components/portfolio/EditHoldingDialog.tsx apps/web/components/portfolio/DeleteConfirmDialog.tsx apps/web/components/portfolio/HoldingsTable.tsx
git commit -m "feat(web): 보유 자산 수정·삭제 모달 + 행 액션"
```

---

## Task 13: 홈 대시보드 — 총자산 카운트업 + 자산 분포 도넛

**Files:**
- Create: `apps/web/components/home/TotalAssetCard.tsx`
- Create: `apps/web/components/home/AllocationDonut.tsx`
- Modify: `apps/web/app/app/page.tsx`

총자산은 holdings의 `market_value_krw` 합. 도넛은 asset_class 토글(MVP는 토글 없이 asset_class 단일). SVG `<circle>`의 `strokeDasharray`로 arc 표현.

- [ ] **Step 1: TotalAssetCard.tsx**

```tsx
"use client";

import { useEffect, useState } from "react";
import type { Holding } from "@/lib/api/holdings";

export function TotalAssetCard({ holdings }: { holdings: Holding[] }) {
  const total = holdings.reduce((s, h) => s + h.market_value_krw, 0);
  const pnl = holdings.reduce((s, h) => s + h.pnl_krw, 0);
  const cost = holdings.reduce((s, h) => s + h.cost_basis_krw, 0);
  const pnlPct = cost > 0 ? (pnl / cost) * 100 : 0;

  const [displayed, setDisplayed] = useState(0);
  useEffect(() => {
    let raf = 0;
    const start = performance.now();
    const dur = 600;
    const from = displayed;
    function tick(now: number) {
      const t = Math.min(1, (now - start) / dur);
      // ease-out cubic
      const eased = 1 - Math.pow(1 - t, 3);
      setDisplayed(from + (total - from) * eased);
      if (t < 1) raf = requestAnimationFrame(tick);
    }
    raf = requestAnimationFrame(tick);
    return () => cancelAnimationFrame(raf);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [total]);

  return (
    <div className="border border-line p-4">
      <div className="text-xs text-fg-muted font-mono mb-1">총 자산 (KRW)</div>
      <div className="font-mono text-3xl tabular-nums">
        ₩{Math.round(displayed).toLocaleString()}
      </div>
      <div className={`text-sm font-mono mt-1 ${pnl >= 0 ? "text-bb-up" : "text-bb-down"}`}>
        {pnl >= 0 ? "+" : ""}{Math.round(pnl).toLocaleString()} ({pnlPct.toFixed(2)}%)
      </div>
    </div>
  );
}
```

- [ ] **Step 2: AllocationDonut.tsx**

SVG 도넛: 각 슬라이스는 `<circle>` + `stroke-dasharray` 누적 offset.

```tsx
"use client";

import type { Holding } from "@/lib/api/holdings";

const PALETTE = ["#00FFFF", "#FFD500", "#00FF7F", "#FF3344", "#A78BFA", "#FF9F1C"];

export function AllocationDonut({ holdings }: { holdings: Holding[] }) {
  // asset_class 기준 집계
  const byClass = new Map<string, number>();
  for (const h of holdings) {
    byClass.set(h.asset_class, (byClass.get(h.asset_class) ?? 0) + h.market_value_krw);
  }
  const total = Array.from(byClass.values()).reduce((s, v) => s + v, 0);

  if (total === 0) {
    return (
      <div className="border border-line p-4">
        <div className="text-xs text-fg-muted font-mono mb-1">자산 분포</div>
        <div className="text-fg-muted text-sm font-mono">데이터 없음</div>
      </div>
    );
  }

  const slices = Array.from(byClass.entries()).map(([k, v], i) => ({
    key: k,
    value: v,
    pct: (v / total) * 100,
    color: PALETTE[i % PALETTE.length],
  }));

  const R = 50;
  const C = 2 * Math.PI * R;
  let offset = 0;

  return (
    <div className="border border-line p-4">
      <div className="text-xs text-fg-muted font-mono mb-3">자산 분포 (자산군)</div>
      <div className="flex items-center gap-4">
        <svg viewBox="0 0 120 120" className="w-32 h-32 -rotate-90">
          <circle cx="60" cy="60" r={R} fill="none" stroke="#1a1a1a" strokeWidth="14" />
          {slices.map((s) => {
            const dash = (s.pct / 100) * C;
            const seg = (
              <circle
                key={s.key}
                cx="60" cy="60" r={R}
                fill="none"
                stroke={s.color}
                strokeWidth="14"
                strokeDasharray={`${dash} ${C - dash}`}
                strokeDashoffset={-offset}
              />
            );
            offset += dash;
            return seg;
          })}
        </svg>
        <ul className="flex-1 space-y-1 font-mono text-xs">
          {slices.map((s) => (
            <li key={s.key} className="flex items-center gap-2">
              <span className="inline-block w-2 h-2" style={{ background: s.color }} />
              <span className="flex-1">{s.key}</span>
              <span className="tabular-nums">{s.pct.toFixed(1)}%</span>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
```

- [ ] **Step 3: page.tsx 변경 (홈 그리드의 첫 두 카드만 활성)**

```tsx
import { HomeDashboard } from "@/components/home/HomeDashboard";

export default function HomePage() {
  return <HomeDashboard />;
}
```

새 client 컴포넌트 `apps/web/components/home/HomeDashboard.tsx`:
```tsx
"use client";

import { useEffect, useState } from "react";
import { listHoldings, type Holding } from "@/lib/api/holdings";
import { TotalAssetCard } from "./TotalAssetCard";
import { AllocationDonut } from "./AllocationDonut";
import { Skeleton } from "@/components/ui/skeleton";

export function HomeDashboard() {
  const [holdings, setHoldings] = useState<Holding[] | null>(null);

  useEffect(() => {
    listHoldings().then(setHoldings).catch(() => setHoldings([]));
  }, []);

  if (holdings === null) {
    return (
      <div className="p-6 md:p-8 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {[1, 2, 3].map((i) => <Skeleton key={i} className="h-32" />)}
      </div>
    );
  }

  return (
    <div className="p-6 md:p-8 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <TotalAssetCard holdings={holdings} />
      <AllocationDonut holdings={holdings} />
      {/* 상위 5·마켓 위젯·관심종목·브리핑은 다음 Task에서 */}
    </div>
  );
}
```

- [ ] **Step 4: 브라우저 확인**

Run: dev. `/app` → 총자산 카드 + 도넛 표시. holdings 있으면 카운트업 애니메이션 동작.

- [ ] **Step 5: 커밋**

```bash
git add apps/web/components/home/TotalAssetCard.tsx apps/web/components/home/AllocationDonut.tsx apps/web/components/home/HomeDashboard.tsx apps/web/app/app/page.tsx
git commit -m "feat(web): 홈 — 총자산 카운트업 + 자산 분포 도넛"
```

---

## Task 14: 홈 대시보드 — 상위 5 + 마켓 위젯 + 관심 종목 + 브리핑 placeholder

**Files:**
- Create: `apps/web/components/home/TopHoldingsCard.tsx`
- Create: `apps/web/components/home/MarketWidgetsCard.tsx`
- Create: `apps/web/components/home/WatchlistMiniCard.tsx`
- Create: `apps/web/components/home/BriefingPlaceholderCard.tsx`
- Modify: `apps/web/components/home/HomeDashboard.tsx`

- [ ] **Step 1: TopHoldingsCard.tsx**

```tsx
"use client";
import type { Holding } from "@/lib/api/holdings";

export function TopHoldingsCard({ holdings }: { holdings: Holding[] }) {
  const top = [...holdings].sort((a, b) => b.market_value_krw - a.market_value_krw).slice(0, 5);
  return (
    <div className="border border-line p-4">
      <div className="text-xs text-fg-muted font-mono mb-3">보유 상위 5</div>
      {top.length === 0 ? (
        <div className="text-fg-muted text-sm font-mono">보유 자산 없음</div>
      ) : (
        <ul className="space-y-1 font-mono text-sm">
          {top.map((h) => (
            <li key={h.id} className="flex items-baseline gap-2">
              <span className="flex-1 truncate">{h.symbol}</span>
              <span className="tabular-nums text-xs text-fg-muted">{h.weight_pct.toFixed(1)}%</span>
              <span className={`tabular-nums text-xs ${h.pnl_pct >= 0 ? "text-bb-up" : "text-bb-down"}`}>
                {h.pnl_pct >= 0 ? "+" : ""}{h.pnl_pct.toFixed(2)}%
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
```

- [ ] **Step 2: MarketWidgetsCard.tsx**

기존 `/v1/market/ticker` 재사용. authFetch로 호출.

```tsx
"use client";
import { useEffect, useState } from "react";
import { authFetch } from "@/lib/api/auth-fetch";

type Ticker = { symbol: string; name: string; price: number; change_pct: number };

export function MarketWidgetsCard() {
  const [items, setItems] = useState<Ticker[]>([]);
  useEffect(() => {
    authFetch("/v1/market/ticker").then((r) => r.ok ? r.json() : []).then(setItems).catch(() => {});
  }, []);
  return (
    <div className="border border-line p-4">
      <div className="text-xs text-fg-muted font-mono mb-3">마켓</div>
      <ul className="space-y-1 font-mono text-sm">
        {items.map((t) => (
          <li key={t.symbol} className="flex items-baseline gap-2">
            <span className="flex-1">{t.name}</span>
            <span className="tabular-nums">{t.price > 0 ? t.price.toLocaleString() : "—"}</span>
            <span className={`tabular-nums text-xs ${t.change_pct >= 0 ? "text-bb-up" : "text-bb-down"}`}>
              {t.change_pct >= 0 ? "+" : ""}{t.change_pct.toFixed(2)}%
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}
```

- [ ] **Step 3: WatchlistMiniCard.tsx**

```tsx
"use client";
import { useEffect, useState } from "react";
import { listWatchlist, type WatchlistItem } from "@/lib/api/watchlist";

export function WatchlistMiniCard() {
  const [items, setItems] = useState<WatchlistItem[] | null>(null);
  useEffect(() => { listWatchlist().then(setItems).catch(() => setItems([])); }, []);
  if (items === null) return <div className="border border-line p-4 font-mono text-xs text-fg-muted">로드 중…</div>;
  return (
    <div className="border border-line p-4">
      <div className="text-xs text-fg-muted font-mono mb-3">관심 종목</div>
      {items.length === 0 ? (
        <div className="text-fg-muted text-sm font-mono">아직 없음. 마켓 탭 (W5)에서 추가 예정</div>
      ) : (
        <ul className="space-y-1 font-mono text-sm">
          {items.slice(0, 5).map((w) => (
            <li key={w.instrument_id} className="flex items-baseline gap-2">
              <span className="flex-1 truncate">{w.symbol}</span>
              <span className="tabular-nums">{w.price > 0 ? w.price.toLocaleString() : "—"}</span>
              <span className={`tabular-nums text-xs ${w.change_pct >= 0 ? "text-bb-up" : "text-bb-down"}`}>
                {w.change_pct >= 0 ? "+" : ""}{w.change_pct.toFixed(2)}%
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
```

- [ ] **Step 4: BriefingPlaceholderCard.tsx**

```tsx
export function BriefingPlaceholderCard() {
  return (
    <div className="border border-line p-4">
      <div className="text-xs text-fg-muted font-mono mb-3">오늘 브리핑</div>
      <div className="text-fg-muted text-sm font-mono">
        AI 일일 브리핑은 W4에서 제공됩니다.
      </div>
    </div>
  );
}
```

- [ ] **Step 5: HomeDashboard.tsx 통합**

```tsx
"use client";

import { useEffect, useState } from "react";
import { listHoldings, type Holding } from "@/lib/api/holdings";
import { TotalAssetCard } from "./TotalAssetCard";
import { AllocationDonut } from "./AllocationDonut";
import { TopHoldingsCard } from "./TopHoldingsCard";
import { MarketWidgetsCard } from "./MarketWidgetsCard";
import { WatchlistMiniCard } from "./WatchlistMiniCard";
import { BriefingPlaceholderCard } from "./BriefingPlaceholderCard";
import { Skeleton } from "@/components/ui/skeleton";

export function HomeDashboard() {
  const [holdings, setHoldings] = useState<Holding[] | null>(null);
  useEffect(() => { listHoldings().then(setHoldings).catch(() => setHoldings([])); }, []);

  if (holdings === null) {
    return (
      <div className="p-6 md:p-8 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-32" />)}
      </div>
    );
  }

  return (
    <div className="p-6 md:p-8 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <TotalAssetCard holdings={holdings} />
      <AllocationDonut holdings={holdings} />
      <BriefingPlaceholderCard />
      <TopHoldingsCard holdings={holdings} />
      <MarketWidgetsCard />
      <WatchlistMiniCard />
    </div>
  );
}
```

- [ ] **Step 6: 브라우저 확인**

Run: dev. `/app` → 6개 카드 그리드 표시. 빈 상태/실데이터 둘 다 자연스럽게.

- [ ] **Step 7: 커밋**

```bash
git add apps/web/components/home/
git commit -m "feat(web): 홈 6카드 (상위5·마켓·관심종목·브리핑 placeholder)"
```

---

## Task 15: 온보딩 wizard 3단계 (holdings 1~3개 추가 스텝 복원)

**Files:**
- Create: `apps/web/components/onboarding/HoldingsStep.tsx`
- Modify: `apps/web/components/onboarding/Wizard.tsx`
- Modify: `apps/web/components/onboarding/StepIndicator.tsx` (total 3 지원 확인)

스펙 §6: 1) 통화 → 2) 첫 보유 자산 1~3개 → 3) 데모/시작. skip 가능.

- [ ] **Step 1: HoldingsStep.tsx**

`InstrumentSearchInput` 재사용. 최대 3개 누적 + skip 옵션. 추가는 client에서 임시 배열에 쌓고 wizard 완료 시 일괄 호출(실패 시 부분 적재 허용).

```tsx
"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { InstrumentSearchInput } from "@/components/portfolio/InstrumentSearchInput";
import type { InstrumentResult } from "@/lib/api/instruments";

export type DraftHolding = {
  instrument: InstrumentResult;
  quantity: number;
  avg_cost: number;
};

export function HoldingsStep({
  value, onChange, onNext, onSkip,
}: {
  value: DraftHolding[];
  onChange: (v: DraftHolding[]) => void;
  onNext: () => void;
  onSkip: () => void;
}) {
  const [inst, setInst] = useState<InstrumentResult | null>(null);
  const [quantity, setQuantity] = useState("");
  const [avgCost, setAvgCost] = useState("");
  const [err, setErr] = useState<string | null>(null);

  function addOne() {
    if (!inst) { setErr("종목을 선택해주세요"); return; }
    const q = parseFloat(quantity);
    const c = parseFloat(avgCost);
    if (!(q > 0)) { setErr("수량은 0보다 커야 합니다"); return; }
    if (!(c >= 0)) { setErr("평단가는 0 이상이어야 합니다"); return; }
    if (value.find((d) => d.instrument.id === inst.id)) { setErr("이미 추가된 종목입니다"); return; }
    onChange([...value, { instrument: inst, quantity: q, avg_cost: c }]);
    setInst(null); setQuantity(""); setAvgCost(""); setErr(null);
  }

  return (
    <div className="space-y-4">
      <h2 className="font-mono text-lg">첫 보유 자산을 1~3개 추가하세요</h2>
      <p className="text-fg-muted text-xs font-mono">건너뛰면 빈 포트폴리오로 시작합니다. 나중에 언제든 추가할 수 있습니다.</p>

      {value.length > 0 && (
        <ul className="space-y-1 font-mono text-sm border border-line p-2">
          {value.map((d, i) => (
            <li key={i} className="flex gap-2">
              <span className="flex-1">{d.instrument.symbol} — {d.instrument.name}</span>
              <span className="tabular-nums text-xs text-fg-muted">{d.quantity} @ {d.avg_cost.toLocaleString()}</span>
              <button onClick={() => onChange(value.filter((_, j) => j !== i))} className="text-xs text-bb-down">×</button>
            </li>
          ))}
        </ul>
      )}

      {value.length < 3 && (
        <div className="space-y-3 border border-line p-3">
          <div>
            <Label className="text-xs font-mono">종목</Label>
            <InstrumentSearchInput onSelect={setInst} />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <Label className="text-xs font-mono">수량</Label>
              <Input value={quantity} onChange={(e) => setQuantity(e.target.value)} type="number" step="any" />
            </div>
            <div>
              <Label className="text-xs font-mono">평단가</Label>
              <Input value={avgCost} onChange={(e) => setAvgCost(e.target.value)} type="number" step="any" />
            </div>
          </div>
          {err && <p className="text-bb-down text-xs font-mono">{err}</p>}
          <Button variant="ghost" onClick={addOne}>+ 추가 ({value.length}/3)</Button>
        </div>
      )}

      <div className="flex gap-2 justify-end">
        <Button variant="ghost" onClick={onSkip}>건너뛰기</Button>
        <Button onClick={onNext}>다음 →</Button>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Wizard.tsx 3단계로 확장**

```tsx
"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { createSupabaseBrowser } from "@/lib/supabase/client";
import { createHolding } from "@/lib/api/holdings";
import { StepIndicator } from "./StepIndicator";
import { CurrencyStep } from "./CurrencyStep";
import { HoldingsStep, type DraftHolding } from "./HoldingsStep";
import { DemoOrStartStep } from "./DemoOrStartStep";

export function Wizard() {
  const [step, setStep] = useState(1);
  const [currency, setCurrency] = useState<"KRW" | "USD">("KRW");
  const [drafts, setDrafts] = useState<DraftHolding[]>([]);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const router = useRouter();

  async function complete(demo: boolean) {
    setLoading(true);
    setErr(null);
    const supabase = createSupabaseBrowser();
    // 세션 존재 + access_token 보장 — drafts 적재 시 authFetch가 토큰 없으면 401
    const { data: { session } } = await supabase.auth.getSession();
    if (!session?.user || !session.access_token) {
      router.push("/login");
      return;
    }
    const user = session.user;

    const { error } = await supabase
      .from("profiles")
      .update({ base_currency: currency, onboarding_completed: true })
      .eq("id", user.id);

    if (error) {
      setErr("프로필 저장에 실패했습니다. 잠시 후 다시 시도해주세요.");
      setLoading(false);
      return;
    }

    // 사용자가 추가한 drafts를 일괄 적재 (부분 실패 허용)
    const failed: string[] = [];
    for (const d of drafts) {
      try {
        await createHolding({
          instrument_id: d.instrument.id,
          quantity: d.quantity,
          avg_cost: d.avg_cost,
        });
      } catch (e) {
        console.warn("draft holding failed", d.instrument.symbol, e);
        failed.push(d.instrument.symbol);
      }
    }
    if (failed.length > 0) {
      // sonner toast로 부분 실패 안내 (silent fail 금지)
      const { toast } = await import("sonner");
      toast.error(`다음 종목 추가에 실패했습니다: ${failed.join(", ")}. 포트폴리오에서 재시도해주세요.`);
    }

    void demo; // demo seeding은 Phase 2 (실제 시드 데이터 미정)

    router.push("/app");
    router.refresh();
  }

  return (
    <main className="min-h-screen flex flex-col">
      <div className="border-b border-line p-4 font-mono text-xs text-fg-muted">
        ONBOARDING — {step}/3
      </div>
      <StepIndicator current={step} total={3} />
      <div className="flex-1 flex items-center justify-center px-6">
        <div className="w-full max-w-lg">
          {step === 1 && (
            <CurrencyStep value={currency} onChange={setCurrency} onNext={() => setStep(2)} />
          )}
          {step === 2 && (
            <HoldingsStep
              value={drafts}
              onChange={setDrafts}
              onNext={() => setStep(3)}
              onSkip={() => { setDrafts([]); setStep(3); }}
            />
          )}
          {step === 3 && (
            <DemoOrStartStep
              onDemo={() => complete(true)}
              onStart={() => complete(false)}
              loading={loading}
            />
          )}
          {err && <p className="text-bb-down text-xs mt-4 font-mono">{err}</p>}
        </div>
      </div>
    </main>
  );
}
```

- [ ] **Step 3: StepIndicator total 3 동작 확인**

`apps/web/components/onboarding/StepIndicator.tsx`를 읽어 `total` prop이 3을 받아도 정상 렌더되는지 확인. 일반적으로 dot 개수만 바뀌므로 변경 불필요할 가능성 높음. 깨지면 단순 fix.

- [ ] **Step 4: 신규 가입 흐름 검증**

Run: dev. 새 계정 가입 → 이메일 인증 → 위저드 1/3 통화 선택 → 2/3 삼성전자 10주 70000원 추가 → 3/3 시작 → `/app` 진입 → 홈 대시보드에 보유 상위 5에 삼성전자 표시.

- [ ] **Step 5: 커밋**

```bash
git add apps/web/components/onboarding/
git commit -m "feat(web): 온보딩 wizard 3단계 (holdings 1~3개 추가 스텝 복원)"
```

---

## Task 16: 통합 동작 검증 + 문서 갱신

**Files:**
- Modify: `docs/STATUS.md`
- Modify: `docs/ROADMAP.md`
- (선택) `docs/ARCHITECTURE.md` — holdings·watchlist 데이터 흐름이 아키텍처 변경에 해당하지 않으므로 갱신 없음

W3 종료 검증 11항. 한 항목이라도 실패하면 fix → 재검증 → 커밋.

- [ ] **Step 1: 백엔드 통합 검증**

서버 기동:
```bash
cd apps/api && go run ./cmd/server
```

별도 터미널에서 (JWT는 supabase studio 또는 dev tools에서 추출):
```bash
TOKEN="<access_token>"
# 1. 빈 holdings
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8080/v1/holdings
# 기대: []

# 2. 추가
curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"instrument_id":"<삼성전자 id>","quantity":10,"avg_cost":70000}' \
  http://localhost:8080/v1/holdings
# 기대: 201 + Holding JSON

# 3. 중복 추가
curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"instrument_id":"<삼성전자 id>","quantity":1,"avg_cost":1}' \
  http://localhost:8080/v1/holdings
# 기대: 409 CONFLICT

# 4. List (enrichment)
curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8080/v1/holdings | jq .
# 기대: market_value_krw, pnl_pct, weight_pct 계산된 값

# 5. PATCH
curl -s -X PATCH -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"quantity":15}' http://localhost:8080/v1/holdings/<id>
# 기대: 200 + quantity=15

# 6. DELETE
curl -s -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8080/v1/holdings/<id>
# 기대: 204

# 7. watchlist 추가/조회/삭제
curl -s -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"instrument_id":"<id>"}' http://localhost:8080/v1/watchlist
# 기대: 201

curl -s -H "Authorization: Bearer $TOKEN" http://localhost:8080/v1/watchlist | jq .
# 기대: 1행 + price·change_pct

curl -s -X DELETE -H "Authorization: Bearer $TOKEN" http://localhost:8080/v1/watchlist/<id>
# 기대: 204
```

- [ ] **Step 2: cron polling 확장 검증**

서버 로그 관찰. holdings에 종목을 추가한 직후, 1~2분 내 다음 로그 확인:
```
{"msg":"market quotes updated","count":N}
```
N >= holdings 개수가 되어야 함(장중 가정). DB에서 직접 확인:
```bash
psql "$DATABASE_URL" -c "select i.symbol, q.price, q.updated_at from public.quotes q join public.instruments i on i.id = q.instrument_id order by q.updated_at desc limit 10;"
```

- [ ] **Step 3: UI 흐름 검증**

브라우저:
1. `/app/portfolio` → [+ 추가] → 삼성전자 검색·선택 → 수량 10 평단가 70000 → 추가
2. 행 표시. [수정] → 수량 15 → 저장. 갱신됨
3. [삭제] → 확인 → 행 사라짐
4. 다시 종목 1개 추가 후 `/app` 진입
5. 홈 6개 카드 정상 표시 (총자산 카운트업, 도넛 슬라이스, 상위5, 마켓 위젯, 관심종목, 브리핑 placeholder)
6. 신규 계정 가입 → 온보딩 3단계 정상 완료

- [ ] **Step 4: 회귀 테스트 풀 실행**

Run: `cd apps/api && go test ./... && cd ../web && npx tsc --noEmit`
또한 vitest 설정(`apps/web/vitest.config.ts` 존재)이 살아 있으면: `cd apps/web && npx vitest run`
Expected: Go 전체 PASS, tsc 에러 0, vitest 기존 테스트 회귀 없음.

- [ ] **Step 5: STATUS.md 갱신**

`docs/STATUS.md`에서:
1. "현재 Phase" → `W1·W2a·W2b·W3 완료. W4 (AI 채팅) 작성 대기.`
2. "진행 중"에서 W3 관련 항목 제거
3. "완료" 섹션 맨 아래에 추가:
   ```
   - ✅ W3-T1 holdings + watchlist 마이그레이션 + RLS 7개 (`<sha>`)
   - ✅ W3-T2~7 holdings·watchlist CRUD API + FX 환산 + 라우트 통합 (`<sha>`)
   - ✅ W3-T8 cron quotes polling INDEX ∪ holdings ∪ watchlist union 확장 (`<sha>`)
   - ✅ W3-T9~12 web API 클라이언트 + 포트폴리오 페이지(테이블·CRUD 모달) (`<sha>`)
   - ✅ W3-T13~14 홈 대시보드 6카드 (총자산·도넛·상위5·마켓·관심종목·브리핑placeholder) (`<sha>`)
   - ✅ W3-T15 온보딩 wizard 3단계 (holdings 추가 스텝 복원) (`<sha>`)
   ```
4. "알려진 결함"에서 "온보딩 단계 수: 스펙 §6은 3단계, W1 구현은 2단계" 항목 제거
5. "최근 변경 이력" 맨 위에 한 줄 추가:
   ```
   - 2026-05-23 W3 전체 완료. holdings·watchlist 마이그+CRUD API, cron polling union 확장, 포트폴리오 페이지(테이블+CRUD 모달), 홈 대시보드 6카드, 온보딩 3단계 복원.
   ```
6. "마지막 업데이트: 2026-05-23"

- [ ] **Step 6: ROADMAP.md 갱신**

`docs/ROADMAP.md`에서:
1. "현재 추천 다음 작업" → `W4 plan 작성 — AI 채팅(Claude tool use, 스트리밍, 사용량 추적). chat_sessions·chat_messages·chat_usage_monthly 마이그 + Claude API tool routing(§10-1 JWT 전파).`
2. Phase 1 표에서 W3 관련 행 3개 (`포트폴리오 CRUD UI + API`, `홈 대시보드`, `JobUpdateIndexQuotes polling 대상 holdings/watchlist union 확장`) 제거
3. 표 첫 행이 `AI 채팅 (Claude tool use, 스트리밍) | W4`가 됨

- [ ] **Step 7: 최종 커밋**

```bash
git add docs/STATUS.md docs/ROADMAP.md
git commit -m "docs(w3): W3 완료 반영 (STATUS·ROADMAP 갱신, 결함 1건 해제)"
```

---

## 후속 작업 (W3 비범위, 백로그)

- **포트폴리오 미니 스파크라인(spec §6)**: Phase 1 후반/W5에서 recharts 도입 + prices 7일 조회 API 추가
- **포트폴리오 선택 행 우측 sliding panel(spec §6)**: 위와 동일 시점
- **마켓 탭 `/app/market`**: W5에서 KR/US 지수·환율·경제 지표·관심 종목 카드 5종 + 라인 차트. **watchlist 추가/제거 UI는 이 탭에서 제공** (W3은 조회만, API는 동작)
- **FX 환율 EUR_KRW·JPY_KRW quotes 적재**: W2b 결함. W5 마켓 탭 작업 시 `jobs_fx.go`의 rateMap key 보정 또는 instrument symbol 정규화 중 택일
- **CSV 업로드 자리 placeholder 배지**: W5 또는 광고 슬롯과 함께
- **자산 분포 도넛 토글(asset_class·통화·종목)**: 사용자 피드백 확보 후 Phase 2
- **API authFetch 토큰 자동 refresh**: Supabase JS v2가 자동 갱신하나, 명시적 401 → refresh → 재시도 1회 패턴은 W4 AI 채팅 SSE 호출과 함께 설계 (Important 10)

---

## 검토 이력

### 2026-05-23 — 1차 subagent 검토 (general-purpose)

| 우선순위 | 항목 | 패치 위치 |
|---|---|---|
| Critical | `middleware.WithUserID` 부재 → 테스트 컴파일 실패 | Task 1.5 신설 (helper 추가) |
| Critical | CASH/INDEX/FX instrument 차단 누락 → 평가액 왜곡 | Task 5 handler + Task 11 search input |
| Critical | FX 환율 매핑(EUR_KRW·JPY_KRW) W2b 결함 의존 | Task 3 알려진 한계 명시 + 후속 작업 등재 |
| Critical | `enrichHoldings`의 unused `context.Background()` | Task 5 코드 블록 정정 + import 제거 |
| Critical | wizard 세션 토큰 race + silent fail | Task 15 `getSession` 가드 + sonner toast |
| Critical | main.go 와이어링 코드 블록 단편적 | Task 7 Step 2 코드 완전화 |
| Important | `JobUpdateIndexQuotes` 함수명이 동작과 불일치 | Task 8 → `JobUpdateMarketQuotes` rename + cron.go 동시 갱신 |
| Important | `InstrumentResult`에 `currency`·`asset_class` 누락 | Task 9 Step 4 백엔드 응답·프론트 타입 확장 |
| Important | `holdings_user_idx`만으로 sort 비효율 | Task 1 → `(user_id, created_at desc)` 복합 인덱스 |
| Important | Task 16 `npm test` 명령 미정의 | vitest 명령으로 교체 |
| Minor | watchlist UI 추가 경로 부재 (W5 비범위 미명시) | WatchlistMiniCard 카피 + 후속 작업 명시 |

Important 잔여(외래키 23503 매핑, EditDialog non-null assertion 등) 및 Minor 다수는 구현 시점에 자연 해결 가능 — plan 본문 추가 패치 보류.

## Self-Review 체크리스트

- [ ] 스펙 §3 holdings·watchlist 스키마 모두 반영 (UNIQUE, PK, 인덱스)
- [ ] 스펙 §6 포트폴리오 컬럼 9종 모두 반영 (종목·수량·평단가·현재가·평가액·수익률·비중·통화·액션)
- [ ] 스펙 §6 홈 6카드(총자산·도넛·브리핑·상위5·마켓·관심종목) 모두 반영 (브리핑은 placeholder)
- [ ] 스펙 §10-2 quotes TTL 60s + union dedup 유지
- [ ] 스펙 §10-6 instruments unique 활용 (holdings 충돌 처리)
- [ ] 스펙 §10-9 instrument_aliases 학습: AddHolding 시 selectInstrument 호출
- [ ] RLS 7개 정책 정의됨 (holdings 4 + watchlist 3)
- [ ] FX 환산 KRW base + fallback 1.0 + 경고 로그
- [ ] 온보딩 3단계 (skip 가능)
- [ ] 모든 핸들러 인증 가드 + 검증 + 에러 매핑
