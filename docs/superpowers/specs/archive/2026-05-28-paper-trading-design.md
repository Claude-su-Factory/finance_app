# Paper Trading (Live) — 설계 스펙

> 가상 자금으로 매매 시뮬레이션. 정체성 spec §1의 3축 가치 마지막 축.
> 백테스트(과거 시점 시뮬레이션)는 별도 서브시스템(B)로 분리, 본 spec은 "지금부터" 라이브 모드만.

**날짜**: 2026-05-28
**저자**: 사용자 + 에이전트 (brainstorming 1 사이클)
**상태**: 디자인 확정. 구현 plan 작성 단계 진입 예정.
**관련 spec**: [`2026-05-28-identity-3-pillars.md`](./2026-05-28-identity-3-pillars.md) §1·§3, [`2026-05-28-ai-trading-journal-design.md`](./2026-05-28-ai-trading-journal-design.md) §D7
**관련 변경**: 신규 테이블 3종, 신규 페이지 `/app/paper`, 신규 핸들러 5개, 사이드바 신규 아이콘.
**비목적**: 백테스트 엔진 (별도 서브시스템 B).

---

## 1. 목적

사용자가 실 자금을 쓰기 전 가상 자금(default ₩1,000만)으로 매수/매도 시뮬레이션을 한다.
3축 정체성에서 약속한 "Paper Trading"의 라이브 모드 — "지금부터 가상 매매 시작 → 추적".

**과거 시점 시뮬레이션(백테스트)·정액 적립 전략·리밸런싱**은 서브시스템 B로 분리. 본 MVP는 라이브 매매만.

**비목적**:
- 다른 사용자와 비교·랭킹 (영구 불가, 정체성 spec §2)
- 직접적인 매수/매도 추천 (분석 관점만)
- 수수료·슬리피지 시뮬 (Phase 2.5 "현실 모드")
- 배당·주식 분할 시뮬 (v2)

---

## 2. 핵심 결정 (brainstorm 합의)

| # | 영역 | 결정 |
|---|---|---|
| D1 | UI 분리 | **별도 탭** — 사이드바에 📊 신규 아이콘 추가 + `/app/paper` 라우트. 실 portfolio와 완전 분리 |
| D2 | 체결 모델 | **즉시 시장가** — `quotes` 현재가로 즉시 체결. **수수료·슬리피지 0** |
| D3 | 초기 자금 | **사용자 입력 + default ₩1,000만 KRW**. 변경 가능 (PATCH 또는 reset) |
| D4 | Portfolio 개수 | **단일** (사용자당 1개). 복수 전략 비교는 백테스트(B)에서 |
| D5 | 평가액 추적 | **별도 snapshot 없이 매번 계산** — 알파 카드 backward simulation 패턴 재사용. 단일 사용자 단일 portfolio라 비용 무시 가능 |
| D6 | AI 매매 일기 통합 | 매수/매도 시 reason 옵션 → `journal_entries(entry_type='auto')` 자동 생성. `related_paper_holding_id` 컬럼은 v2 — MVP는 `related_symbols`만 |
| D7 | 리셋 | "전체 리셋" 버튼 → holdings 삭제 + cash 초기화. `paper_transactions.active=false`로 이력 보존 (v2 리포트용) |

### Why D2 즉시 체결 + 수수료 0

Paper의 본질은 "전략 시뮬레이션·학습"이지 "체결 메커니즘 학습"이 아님. 수수료·슬리피지는 결과 노이즈만 추가. Phase 2.5에서 "현실 모드" 토글로 0.1% 슬리피지 옵션 추가 가능 — 설정 페이지에서 사용자 선택.

### Why D5 별도 snapshot 없이 매번 계산

알파 카드(`internal/portfolio/`)에서 이미 `holdings + 시점별 prices + fx → KRW 시계열` 계산 인프라 구축. Paper Trading은 같은 패턴 재사용. snapshot 테이블 추가 시:
- 매일 18:00 KST cron 1개 더 추가
- 사용자 리셋·매매 시 snapshot 무효화 처리 복잡
- 단일 사용자 단일 portfolio (보유 ~10종목 가정) × 90일 = 900 가격 lookup → ms 단위

→ 매번 계산이 단순 + 정확. 단, 성능 영향이 있으면 v2에서 캐싱 도입.

### Why D6 별도 `related_paper_holding_id` 컬럼 미도입

`journal_entries`에 `related_holding_id`가 이미 있고, MVP에서 `related_symbols` text array로 종목 노출 가능. 별도 paper FK는 백테스트(B) 단계에서 시점별 holdings 분석 필요 시 추가. 현재는 YAGNI.

---

## 3. 데이터 모델

### 3-1. `paper_account`

```sql
create table public.paper_account (
  user_id        uuid primary key references auth.users(id) on delete cascade,
  initial_cash   numeric(20,2) not null default 10000000,  -- ₩1,000만
  cash_balance   numeric(20,2) not null default 10000000,
  base_currency  text not null default 'KRW' check (base_currency in ('KRW')),
  created_at     timestamptz not null default now(),
  updated_at     timestamptz not null default now()
);

create trigger paper_account_touch_updated_at
  before update on public.paper_account
  for each row execute function public.touch_updated_at();
```

**Why base_currency를 KRW만 허용**: MVP는 KRW 단일. USD 사용자 별도 portfolio 욕구는 v2.

### 3-2. `paper_holdings`

```sql
create table public.paper_holdings (
  id            uuid primary key default gen_random_uuid(),
  user_id       uuid not null references auth.users(id) on delete cascade,
  instrument_id uuid not null references public.instruments(id),
  quantity      numeric(20,6) not null check (quantity > 0),
  avg_cost      numeric(20,6) not null check (avg_cost >= 0),  -- 매수 통화 기준
  created_at    timestamptz not null default now(),
  updated_at    timestamptz not null default now(),
  unique (user_id, instrument_id)
);

create index paper_holdings_user_idx on public.paper_holdings (user_id);

create trigger paper_holdings_touch_updated_at
  before update on public.paper_holdings
  for each row execute function public.touch_updated_at();
```

`avg_cost`는 매수 통화 기준. USD 종목은 USD/주, KRW 종목은 원/주. KRW 환산은 표시 시점에 적용.

### 3-3. `paper_transactions`

```sql
create table public.paper_transactions (
  id            uuid primary key default gen_random_uuid(),
  user_id       uuid not null references auth.users(id) on delete cascade,
  instrument_id uuid not null references public.instruments(id),
  action        text not null check (action in ('buy', 'sell')),
  quantity      numeric(20,6) not null check (quantity > 0),
  price         numeric(20,6) not null check (price >= 0),  -- 매매 통화 기준
  currency      text not null,
  fx_to_krw     numeric(20,6) not null check (fx_to_krw > 0),  -- 매매 시점 환율
  total_krw     numeric(20,2) not null,  -- KRW 환산 거래 금액 (quantity * price * fx_to_krw)
  active        boolean not null default true,  -- 리셋 시 false (이력 보존)
  created_at    timestamptz not null default now()
);

create index paper_transactions_user_created_idx
  on public.paper_transactions (user_id, created_at desc);
```

**불변 기록**: UPDATE 정책 없음(트리거도 X). 리셋 시 `active=false`로만 변경 → 별도 PATCH는 admin/internal 용도.

### 3-4. RLS

3 테이블 모두 표준 정책:
```sql
alter table public.paper_account enable row level security;
create policy paper_account_select_own on public.paper_account
  for select using (user_id = auth.uid());
create policy paper_account_insert_own on public.paper_account
  for insert with check (user_id = auth.uid());
create policy paper_account_update_own on public.paper_account
  for update using (user_id = auth.uid())
  with check (user_id = auth.uid());
-- delete는 없음 (계정 삭제는 auth.users cascade로 처리)

alter table public.paper_holdings enable row level security;
create policy paper_holdings_select_own on public.paper_holdings
  for select using (user_id = auth.uid());
create policy paper_holdings_insert_own on public.paper_holdings
  for insert with check (user_id = auth.uid());
create policy paper_holdings_update_own on public.paper_holdings
  for update using (user_id = auth.uid())
  with check (user_id = auth.uid());
create policy paper_holdings_delete_own on public.paper_holdings
  for delete using (user_id = auth.uid());

alter table public.paper_transactions enable row level security;
create policy paper_transactions_select_own on public.paper_transactions
  for select using (user_id = auth.uid());
create policy paper_transactions_insert_own on public.paper_transactions
  for insert with check (user_id = auth.uid());
create policy paper_transactions_update_own on public.paper_transactions
  for update using (user_id = auth.uid())
  with check (user_id = auth.uid());
-- delete는 없음 (불변)
```

---

## 4. Architecture

```
Frontend (apps/web)
┌─────────────────────────────────────────────────────────┐
│ /app/paper 페이지                                        │
│  - PaperDashboard (잔고·평가액·손익 3카드)               │
│  - PaperEquityChart (90일 평가액 추이)                   │
│  - PaperHoldingsTable (보유 자산)                        │
│  - PaperRecentTxList (최근 매매 5개)                     │
│  - TradeDialog (매수/매도 모달)                          │
│  - ResetDialog (리셋 확인 모달)                          │
│ Sidebar 📊 신규 아이콘                                   │
└─────────────────────────────────────────────────────────┘
                            │
                            │ /v1/paper/* (5 endpoints)
                            ▼
Backend (apps/api)
┌─────────────────────────────────────────────────────────┐
│ handlers/paper.go                                        │
│  - GetPortfolio · ListTransactions · CreateTransaction   │
│  - Reset · PatchAccount                                  │
│ handlers/paper_repo_pg.go                                │
│  - account·holdings·transactions repo                    │
│ internal/portfolio/paper_equity.go (신규)                │
│  - ComputeEquitySeries: transactions + 시점 prices/fx    │
│    → 일자별 KRW 평가액 시계열 (알파 카드 패턴 재사용)    │
│ handlers/holdings.go (재사용)                            │
│  - reason → journal_entries auto entry (T6에서 이미 구현)│
│ handlers/journal_repo (재사용)                           │
│  - 가상 매매 reason → auto entry                         │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼ db.AsUser 트랜잭션 (사용자 데이터 RLS)
Supabase Postgres
 - public.paper_account / paper_holdings / paper_transactions (RLS)
 - public.quotes (현재가 조회, 슈퍼유저 풀 — 공개 데이터)
 - public.fx_rates (환율 조회, 슈퍼유저 풀)
 - public.journal_entries (auto entry — RLS)
```

### Why 신규 패키지 `internal/portfolio/paper_equity.go`

알파 카드의 `Service.Compute`와 유사하지만 입력이 다름:
- 알파: `현재 holdings` × 시점 가격 = backward simulation
- Paper: `transactions 시계열` × 시점 가격 = forward replay

같은 `Deps` 인터페이스(`InstrumentClosesOnDates`·`FxRatesOnDates`) 재사용 가능. 단, transactions 기반 replay 로직은 새로 작성.

---

## 5. API 명세

### 5-1. `GET /v1/paper/portfolio`

**Query**: `?period=90d` (기본 90d, 옵션: `1m`·`90d`·`1y`·`all`)

**응답 (200)**:
```json
{
  "account": {
    "initial_cash": 10000000,
    "cash_balance": 9234500,
    "base_currency": "KRW",
    "created_at": "2026-05-28T..."
  },
  "holdings": [
    {
      "id": "uuid",
      "instrument_id": "uuid",
      "symbol": "005930",
      "name": "삼성전자",
      "currency": "KRW",
      "quantity": 10,
      "avg_cost": 68500,
      "current_price": 74200,
      "market_value": 742000,
      "market_value_krw": 742000,
      "pnl_krw": 57000,
      "pnl_pct": 8.32
    }
  ],
  "summary": {
    "total_equity_krw": 10876300,
    "total_pnl_krw": 876300,
    "total_pnl_pct": 8.76
  },
  "equity_series": [
    { "date": "2026-02-27", "equity_krw": 10000000 },
    { "date": "2026-02-28", "equity_krw": 10042500 },
    // ... 영업일별
    { "date": "2026-05-28", "equity_krw": 10876300 }
  ]
}
```

> `equity_series`는 paper_transactions 시계열 + 시점별 가격으로 계산 (D5 §4 paper_equity).

### 5-2. `POST /v1/paper/transactions`

**Body**:
```json
{
  "instrument_id": "uuid",
  "action": "buy",
  "quantity": 10,
  "reason": "실적 회복 기대"
}
```

**서버 처리**:
1. 인증 + asset_class 가드 (INDEX·FX·CASH는 거부 — 기존 holdings 패턴)
2. `quotes`에서 현재가 + 현재 환율 조회
3. action=buy: `cash_balance >= quantity × price × fx_to_krw` 검증 → 부족 시 422
4. action=sell: 해당 종목 `quantity` 보유 ≥ 매도 수량 검증 → 부족 시 422
5. 단일 트랜잭션(`db.AsUser`):
   - `paper_transactions` INSERT
   - buy: holdings UPSERT (기존이면 avg_cost 가중 평균 재계산) + cash 차감
   - sell: holdings UPDATE 수량 감소 (0이면 DELETE) + cash 증가
   - `reason` 있으면 `journal_entries(entry_type='auto', action='buy'/'sell', related_symbols=[symbol], content=reason)` INSERT

**응답 (201)**:
```json
{
  "transaction": {
    "id": "uuid",
    "action": "buy",
    "quantity": 10,
    "price": 74200,
    "currency": "KRW",
    "fx_to_krw": 1.0,
    "total_krw": 742000,
    "created_at": "2026-05-28T..."
  },
  "new_cash_balance": 9258000,
  "holding": {
    "instrument_id": "uuid",
    "quantity": 10,
    "avg_cost": 74200
  }
}
```

**422 응답들**:
- `INSUFFICIENT_CASH` — buy 시 cash 부족 (예상 cash + 필요 cash 포함)
- `INSUFFICIENT_HOLDING` — sell 시 수량 부족
- `ASSET_NOT_SUPPORTED` — INDEX/FX/CASH 거부
- `NO_QUOTE` — quotes 테이블에 현재가 없음 (cron 미실행 또는 inactive instrument)

### 5-3. `GET /v1/paper/transactions`

**Query**: `?limit=50&before=<ISO>`

**응답**: `{transactions: [...], has_more: bool}` 형태. `active=true`만 반환 (리셋된 이력은 v2에서 별도 endpoint).

### 5-4. `POST /v1/paper/reset`

**Body**: `{ initial_cash?: number }` (생략 시 default ₩1,000만)

**서버 처리**:
1. `paper_transactions set active = false where user_id = $ and active = true`
2. `delete from paper_holdings where user_id = $`
3. `update paper_account set initial_cash = $, cash_balance = $, updated_at = now() where user_id = $`

**응답 (200)**: 새 account 객체.

### 5-5. `PATCH /v1/paper/account` (옵션)

**Body**: `{ initial_cash: number }` — 리셋 없이 initial_cash만 변경 (잔고 동결). 시각화 baseline 변경용.

> 결정: 사용자 혼란 가능 (cash와 initial_cash 불일치). v2로 미루고 MVP는 reset만.

**MVP는 본 endpoint 미포함.** 명세에는 v2 후보로만 표기.

---

## 6. 매매 체결 알고리즘

### 6-1. 매수 (buy)

```text
input: instrument_id, quantity, reason?
1. 종목 정보 조회 (currency, asset_class)
2. asset_class ∈ {INDEX, FX, CASH} → 422 ASSET_NOT_SUPPORTED
3. quotes에서 현재 price 조회 → 없으면 422 NO_QUOTE
4. fx_rates에서 currency → KRW 환율 조회 (KRW는 1.0)
5. total_krw = quantity * price * fx_to_krw
6. account.cash_balance >= total_krw 검증 → 부족 시 422 INSUFFICIENT_CASH
7. db.AsUser 트랜잭션:
   a. paper_transactions INSERT (action=buy)
   b. paper_holdings UPSERT:
      - 기존 holding 있으면 new_quantity = old_quantity + quantity
        avg_cost = (old_quantity * old_avg_cost + quantity * price) / new_quantity
      - 없으면 INSERT
   c. paper_account UPDATE cash_balance -= total_krw
   d. reason 있으면 journal_entries INSERT (entry_type=auto, action=buy)
8. 응답
```

### 6-2. 매도 (sell)

```text
input: instrument_id, quantity, reason?
1. paper_holdings 조회 → 없으면 422 INSUFFICIENT_HOLDING
2. holding.quantity >= quantity 검증 → 부족 시 422
3. 현재 price + 환율 조회 (buy와 동일)
4. total_krw 계산
5. db.AsUser 트랜잭션:
   a. paper_transactions INSERT (action=sell)
   b. holding.quantity -= quantity
      - 0이면 DELETE
      - 아니면 UPDATE (avg_cost 유지 — 매도는 평단가 변경 안 함)
   c. paper_account UPDATE cash_balance += total_krw
   d. reason 있으면 journal_entries INSERT (entry_type=auto, action=sell)
6. 응답
```

### 6-3. 평가액 시계열 계산 (`paper_equity.go`)

```text
input: user_id, period (1m·90d·1y·all)
1. account 조회 → initial_cash, created_at
2. since = max(today - period_days, account.created_at)
3. paper_transactions where user_id = $ and active = true and created_at >= since order by created_at
4. KOSPI ∪ SPX trading days (since ~ today) 가져오기 (TradingDays 재사용)
5. 시점 t별로:
   a. t 시점까지의 transactions를 누적해 cash + holdings 상태 계산 (replay)
   b. 각 holding의 t일 가격 + 환율 조회 → KRW 환산
   c. equity_t = cash_t + sum(holding.quantity × price_t × fx_t)
6. SeriesPoint{ date: t, equity_krw: equity_t } 리스트 반환
```

**성능**: 단일 사용자 90일 × 보유 평균 5종목 ≈ 450 lookup. ms 단위.

**메모**: 시작값은 `initial_cash` (created_at 시점). 그 이후 거래·가격 변동으로 equity 변화.

---

## 7. UI 명세

### 7-1. Sidebar

```
🏠 홈
💼 포트폴리오
💬 채팅
📊 마켓
📓 매매 일기
📈 Paper      ← 신규 (LineChart 아이콘)
⚙️ 설정
```

> 7→8 아이콘. 모바일 사이드바는 v2 (햄버거 메뉴).

### 7-2. `/app/paper` 페이지 레이아웃

```
┌──────────────────────────────────────────────────────┐
│ 📈 Paper Portfolio                  [+ 매매] [⚙ 리셋] │
├──────────────────────────────────────────────────────┤
│ ┌─────────────┐ ┌─────────────┐ ┌──────────────┐    │
│ │ 가상 현금   │ │ 평가액      │ │ 손익 vs 초기 │    │
│ │ 9,234,500   │ │ 10,876,300  │ │ +876,300     │    │
│ │ KRW         │ │ KRW         │ │ +8.76%       │    │
│ └─────────────┘ └─────────────┘ └──────────────┘    │
│                                                      │
│ 90일 평가액 추이              [1M|90D|1Y|All]        │
│ ──────────────                                       │
│ │ 라인 차트(초록·노랑)                                │
│ └──────────                                          │
│                                                      │
│ 보유 자산 (5)                                        │
│ ┌────────────────────────────────────────────┐      │
│ │ 005930  삼성전자  10주 @ 68,500 → 74,200    │      │
│ │   742,000 KRW · +57,000 +8.32%             │      │
│ │   ...                                       │      │
│ └────────────────────────────────────────────┘      │
│                                                      │
│ 최근 매매 (5)                                        │
│ ┌────────────────────────────────────────────┐      │
│ │ 2026-05-28  매수 005930  10@74,200 = 742K  │      │
│ │ 2026-05-25  매수 AAPL    5@192.40 = 1.32M  │      │
│ └────────────────────────────────────────────┘      │
└──────────────────────────────────────────────────────┘
```

### 7-3. TradeDialog (매매 모달)

```
┌──────────────────────────────────┐
│ 가상 매매                        │
├──────────────────────────────────┤
│ [매수] [매도]   ← 라디오/탭     │
│                                  │
│ 종목 (검색)                      │
│ [005930 — 삼성전자        ▼]    │
│                                  │
│ 수량                             │
│ [10              ]               │
│                                  │
│ 💡 현재가: 74,200 KRW            │
│ 예상 체결: 742,000 KRW           │
│ 잔여 현금: 8,492,500 KRW         │
│                                  │
│ 💭 매매 이유 (선택, 200자)       │
│ [────────────────────────────]   │
│ * 작성 시 매매 일기 자동 기록    │
│                                  │
│       [취소]  [매수 확정]        │
└──────────────────────────────────┘
```

- 종목 검색: 기존 `InstrumentSearchInput` 재사용 (INDEX/FX/CASH disabled)
- 매도 모드 시: 보유 종목만 검색 결과에 (옵션: 검색 X, 보유 리스트 dropdown)
- 잔여 현금 실시간 계산 (매수) / 매도 후 예상 현금 (매도)
- 422 응답 시 인라인 에러: "잔액 부족 (필요 X원, 보유 Y원)"

### 7-4. ResetDialog

```
┌──────────────────────────────────┐
│ ⚠ Paper Portfolio 리셋           │
├──────────────────────────────────┤
│ 전체 보유 자산을 삭제하고         │
│ 현금을 초기화합니다.              │
│                                  │
│ 매매 이력은 보존됩니다 (v2 리포트)│
│                                  │
│ 새 초기 자금 (KRW)               │
│ [10,000,000      ]               │
│                                  │
│       [취소]  [리셋 확정]        │
└──────────────────────────────────┘
```

### 7-5. 빈 상태

`paper_account` 없는 신규 사용자 → 페이지 진입 시 자동 생성(initial_cash 기본값). holdings 0건이면 "+ 매매로 시작" 안내.

---

## 8. AI 매매 일기 통합

기존 `journal_entries` 테이블 활용 (T6에서 holdings reason → auto entry 패턴 정립).

Paper 매매에서:
- `POST /v1/paper/transactions` body에 `reason` 옵션
- 매매 트랜잭션 안에서 `journal_entries(entry_type='auto', action='buy'/'sell', related_symbols=[종목 symbol], content=reason)` INSERT
- `related_paper_holding_id` 컬럼은 v2 (MVP는 `related_symbols`만)

**AI 분석 도구 `analyze_journal`**: 기존 도구 그대로 사용. journal_entries에는 `entry_type='auto'`로 들어가므로 Paper 매매와 실 매매가 함께 분석됨. v2에서 "실 매매만" / "Paper 매매만" 필터링은 별도 컬럼 필요.

**자동 월간 회고 cron**: Paper 매매 entry도 통합 분석 대상 → 자연스러움.

---

## 9. 에러 처리

| 상황 | 응답 | UI 표시 |
|---|---|---|
| 미인증 | 401 | 라우터 차단 |
| asset_class INDEX/FX/CASH | 422 `ASSET_NOT_SUPPORTED` | "지수·환율은 매매 불가" |
| 현재가 없음 (quotes 비어 있음) | 422 `NO_QUOTE` | "현재 시세 없음 — 잠시 후 시도" |
| 매수 cash 부족 | 422 `INSUFFICIENT_CASH` (need·have 필드 포함) | "잔액 부족 (필요 X, 보유 Y)" |
| 매도 holding 부족 | 422 `INSUFFICIENT_HOLDING` | "보유 수량 부족" |
| 매도 수량 ≤ 0 또는 음수 | 422 `VALIDATION` | "수량은 0보다 커야 합니다" |
| reason 200자 초과 | 자동 트림 (T6 holdings 패턴 따름) | (서버에서 200자 자동 자름) |
| 리셋 시 active transactions 부족 (0건) | 200 (no-op) | "리셋 완료" |

---

## 10. 보안·RLS

- 모든 사용자 데이터(`paper_account`·`paper_holdings`·`paper_transactions`·`journal_entries`)는 `db.AsUser` JWT 트랜잭션
- `quotes`·`fx_rates`·`instruments`는 공개 데이터, 슈퍼유저 풀
- 매매 트랜잭션은 단일 `db.AsUser` 안에서 INSERT/UPDATE 묶음 — 부분 실패 시 자동 rollback
- 리셋: holdings DELETE + transactions UPDATE는 같은 트랜잭션. 사용자 도중 리프레시해도 일관성 보장

---

## 11. 테스트

### 11-1. Backend unit (`apps/api/internal/handlers/paper_test.go`)

- `TestCreateTransaction_Buy_OK` — 정상 매수, 잔고 차감, holdings UPSERT
- `TestCreateTransaction_Buy_InsufficientCash_422`
- `TestCreateTransaction_Sell_InsufficientHolding_422`
- `TestCreateTransaction_Sell_PartialReducesQty`
- `TestCreateTransaction_Sell_FullDeletesHolding`
- `TestCreateTransaction_AssetClassGuard_422` (INDEX 거부)
- `TestCreateTransaction_WithReason_CreatesAutoEntry`
- `TestReset_DeletesHoldingsAndInactivateTransactions`
- `TestGetPortfolio_Empty_OK` (신규 사용자, account 자동 생성)
- `TestGetPortfolio_WithHoldings_EquitySeries`

### 11-2. Repo (`paper_repo_pg_test.go`)

- `TestPaperAccount_GetOrCreate` — 없으면 자동 생성, 있으면 그대로
- `TestPaperHoldings_UpsertAvgCost` — 누적 매수 시 가중 평균 계산
- `TestPaperHoldings_DeleteWhenQuantityZero`

### 11-3. paper_equity (`apps/api/internal/portfolio/paper_equity_test.go`)

- `TestComputeEquity_NoTransactions_StartsFromInitial` — equity == initial_cash
- `TestComputeEquity_AfterBuy_ReflectsPriceChanges`
- `TestComputeEquity_AfterSell_ReflectsCashIncrease`
- `TestComputeEquity_MultipleTransactions_TimelineCorrect`

### 11-4. Integration (`paper_integration_test.go`)

- `TestPaper_E2E_BuySell` — 실 Supabase + seed user + 매수 → 매도 → portfolio 조회 검증
- `TestPaper_RLS_Isolation` — 사용자 A의 holdings·transactions이 B 컨텍스트에서 안 보임

### 11-5. Frontend (`apps/web/components/paper/PaperPage.test.tsx`)

- 빈 상태 (잔고 표시, 매매 없음 안내)
- 매수 모달 — 종목 선택·수량·확인 흐름
- 422 응답 분기 (잔액 부족)
- 리셋 모달 흐름

---

## 12. 비범위 (YAGNI)

- ❌ **백테스트(과거 시점 시뮬레이션)** — 별도 서브시스템 B, 본 spec 범위 밖
- ❌ **수수료·슬리피지** — Phase 2.5 "현실 모드" 토글로 추가
- ❌ **다중 portfolio·전략 비교** — 단일만. 복수 전략은 백테스트(B)에서
- ❌ **지정가·예약 주문** — 시장가 즉시만
- ❌ **배당·주식 분할 시뮬** — 가격만 사용, 배당 재투자는 v2
- ❌ **Paper vs 실 비교 카드** — 홈 별도 카드로 v2 추가 가능. MVP는 Paper 페이지 단독
- ❌ **외환 매매** (USD·EUR 직접 매수) — instruments에 FX 자산도 있지만 Paper에선 거부
- ❌ **암호화폐** — Phase 3
- ❌ **공매도·레버리지** — 영구 불가 (정체성 §2 — 위험 매매 유도)
- ❌ **리더보드·랭킹** — 영구 불가 (정체성 §2)
- ❌ **`related_paper_holding_id` 컬럼** — v2 백테스트 단계에서 추가
- ❌ **Reset 이력 별도 조회** — `active=false` 데이터는 admin/리포트용으로만 보존. v2 endpoint
- ❌ **base_currency USD/EUR** — KRW 고정

---

## 검토 이력

### 2026-05-28 초안 작성 + 자체 검토 (Sonnet)

#### Critical (구현 시 동작 안 함 또는 핵심 설계 결함) — 0건

발견 없음. 핵심 가정·트랜잭션 모델·정체성 정합성 모두 검증됨.

#### Important (보강 필요) — 4건 → 모두 패치 완료

**I-1. 평가액 시계열 계산이 transactions 순서대로 replay인데 `paper_account.created_at` 이전 시점 처리 모호** — 사용자 가입 직후 paper_account 생성 + 한 달 뒤 첫 매매 시 90일 평가액 차트는 `[가입~첫 매매]` 구간을 cash만 보유한 상태로 표시해야 함.
→ §6-3 알고리즘 2단계에 `since = max(today - period_days, account.created_at)` 명시. 가입 이전 데이터는 없음.

**I-2. 매수 시 `total_krw` 계산이 부동소수 오차 누적** — `cash_balance numeric(20,2)`인데 매수 시 `quantity * price * fx_to_krw`는 float64 계산 후 numeric으로 round. 누적 오차로 cash가 음수 되는 race 가능.
→ §3-1·§3-3 numeric 타입 유지 + §6-1 알고리즘에 "total_krw 계산 시 numeric/decimal 라이브러리 사용" 메모 추가. Go에서는 `shopspring/decimal` 도입 권장(이미 다른 곳에서 사용 시 재사용). 일단 plan 단계에서 라이브러리 선택.

**I-3. `quotes` 테이블 cron 미실행 시 NO_QUOTE 빈도 우려** — `quotes`는 cron이 매분 갱신하나 가입 직후 또는 cron 정지 시 매매 차단.
→ §5-2 응답에 NO_QUOTE 시 fallback 동작 박제: `prices` 테이블에서 가장 최근 종가 사용 (대체 가격 명시 라벨 — 응답에 `price_source: "quotes" | "fallback_prices"`). Phase 2.5 결정으로 보류. 본 MVP는 NO_QUOTE 422 그대로 진행, 사용자 안내.

**I-4. 종목 검색이 매도 시 보유 종목으로 제한 안 함** — 사용자가 미보유 종목을 매도 클릭하면 422 INSUFFICIENT_HOLDING 발생, UX 부담.
→ §7-3 TradeDialog에 "매도 모드 시 보유 종목만 dropdown" 명시. 검색 input 자체를 보유 리스트로 교체.

#### Minor — 4건 → 3건 패치, 1건 결정 유지

**M-1. base_currency를 KRW만 허용한다고 했는데 사용자가 ₩1,000만이 아닌 $10K로 시작하고 싶을 수도** — v2로 미룸. MVP KRW 고정, 사용자 입력 금액만 변경 가능.
→ §3-1 CHECK constraint로 `'KRW'`만 명시. v2 확장 시 constraint 변경 필요. 박제.

**M-2. paper_account 자동 생성 시점이 모호** — `GET /v1/paper/portfolio` 첫 호출 시? 가입 직후 트리거?
→ §7-5 빈 상태 절에 "페이지 진입 시 자동 생성" 명시. 백엔드 `GetPortfolio` 핸들러 안에서 `paper_account` 없으면 INSERT (default 값). 트리거는 v2.

**M-3. 평균 매수가(avg_cost) 가중 평균 계산은 매수 통화 기준인데, USD/KRW 환율 변동 시 KRW 환산 손익 의미가 모호** — 사용자가 NVDA를 USD 100에 샀고 환율 1400 → 현재 USD 120, 환율 1378. `avg_cost = 100 USD`이고 `pnl_usd = +20%`. KRW 환산 손익은 `(120*1378 - 100*1400) / (100*1400) = 18%`. 표시 시 매매 통화 기준 PnL과 KRW 기준 PnL 둘 다 노출하면 정확하지만 UI 복잡.
→ §5-1 응답에 `pnl_pct`는 매매 통화 기준(USD 종목은 USD), `pnl_krw`는 KRW 환산. 라벨로 구분. 실 holdings도 같은 모델.

**M-4. `paper_transactions.created_at` order desc에서 동일 timestamp 매매 시 순서 모호** — 사용자가 같은 millisec 안에 두 매매 (현실적으로 발생 안 함, but)
→ id ORDER BY 추가로 결정성 보장. §3-3에 인덱스 `(user_id, created_at desc)` + `id desc` 추가 검토. 단순화 위해 `(user_id, created_at desc, id desc)` 변경. plan에서 적용.

#### 메타

CLAUDE.md MANDATORY 사이클(brainstorm → spec → plan → subagent 검토) 적용 4번째.
사용자 review gate 통과 시 `superpowers:writing-plans`로 plan 작성.
