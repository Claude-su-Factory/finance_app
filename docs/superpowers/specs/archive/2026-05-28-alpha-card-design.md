# 알파 카드 (Alpha Card) — 설계 스펙

> 홈 대시보드에 "내 포트폴리오 vs 외부 지수" 비교 카드를 추가한다.
> 정체성 3축 spec의 차별화 카드 중 하나(`2026-05-28-identity-3-pillars.md` §2).

**날짜**: 2026-05-28
**저자**: 사용자 + 에이전트 (brainstorming 1 사이클)
**상태**: 디자인 확정. 구현 plan 작성 단계 진입 예정.
**관련 spec**: [`2026-05-28-identity-3-pillars.md`](./2026-05-28-identity-3-pillars.md)
**관련 코드**: 신규 핸들러·컴포넌트 (구현 시 plan에서 명시).

---

## 1. 목적

사용자가 "내가 잘 굴리고 있나?"라는 가장 빈도 높은 질문에 홈 첫 화면에서 즉답을 제공한다.
3개 외부 벤치마크(KOSPI · S&P 500 · 한미 60/40) 대비 본인 포트폴리오의 누적 수익률 차이(알파)를 표시.

**비목적**:
- 다른 사용자와의 비교·랭킹 (정체성 spec §2에서 영구 불가 결정)
- 종목 추천 (분석 관점만)
- 실시간 갱신 (일봉 데이터 기반)

---

## 2. 핵심 결정 (8건, brainstorm 합의)

| # | 영역 | 결정 |
|---|---|---|
| D1 | 비교 대상 | KOSPI · S&P 500 · 한미 60/40 (3개) stacked |
| D2 | 기간 토글 | 1M / **90D**(default) / 1Y / All — 카드 우측 상단 chip |
| D3 | 60/40 정의 | 60% KOSPI + 40% S&P 500 ("한미 60/40" 라벨, 표준 60-40 아님 명시) |
| D4 | 환율 처리 | 시점별 환율 (USD 종목 환차익 포함) → 카드에 "환율 변동 포함" 라벨 |
| D5 | 빈 상태 | 가입 시점부터 단축 계산 + "가입 N일" 라벨. 7일 미만은 "7일 이상 필요" |
| D6 | 위치 | 홈 대시보드 1행 3번째 (총자산·자산분포 옆) |
| D7 | 차트 | 누적 수익률 3 라인 (포트·KOSPI·S&P). 60/40은 텍스트만(라인 추가 시 빽빽) |
| D8 | 계산 모델 | 현재 보유 종목·수량을 N일 전부터 보유했다고 가정 (backward simulation, `transactions` 테이블 부재) |

### Why D8 backward simulation인가

`transactions`·`paper_transactions` 테이블이 없는 MVP 단계에서 TWR/MWR은 구현 불가. 다음 한계 명시:
- 90일 이내 새로 산 종목 → 90일 전엔 없었지만 "있었다고 가정"
- 90일 이내 판 종목 → holdings에서 빠져서 자동 무시
- 즉 "현재 보유 구성의 시점별 수익률 시뮬레이션"이지 "실제 거래 기반 수익률"은 아님
- 카드 하단에 "* 현재 보유 기준 시뮬레이션" 라벨 명시
- transactions 테이블 도입 시 (Phase 2) 정확한 TWR로 자연 전환

### Why D7에서 60/40만 차트 제외인가

3 라인까지는 monospace 카드에서 식별 가능, 4 라인부터는 색·기울기 구분 어려움. 60/40은 KOSPI·S&P의 가중 평균이라 시각적으로 두 라인 사이에 위치 → 별도 라인 추가 가치 낮음. 텍스트(절대 %p)로 충분.

---

## 3. Architecture

```
┌──────────────────────────────────────────────────────┐
│  Frontend                                            │
│  components/home/AlphaCard.tsx                       │
│   - 기간 토글 chip (1M/90D/1Y/All)                   │
│   - 텍스트 3줄 (vs KOSPI·S&P·60/40)                  │
│   - SVG polyline 3 라인 (포트·KOSPI·S&P 누적 수익률) │
│   - 라벨: "환율 변동 포함" + "현재 보유 기준 시뮬레이션" │
│  lib/api/portfolio.ts: getAlpha(period)              │
└──────────────────────────────────────────────────────┘
                       │
                       │ GET /v1/portfolio/alpha?period=90d
                       ▼
┌──────────────────────────────────────────────────────┐
│  Backend (apps/api)                                  │
│  handlers/portfolio_alpha.go                         │
│   - 인증 + 기간 파싱 + AlphaService 호출             │
│  internal/portfolio/alpha.go (신규 패키지)            │
│   - AlphaService.Compute(ctx, exec, uid, period)     │
│   - 단계: holdings → 시점별 prices/fx → 시계열 합산  │
│           → 벤치마크 시계열 → 알파 계산              │
│  PricesRepo.OnDateRange / FxRepo.SeriesRange         │
│  (기존 history API 인프라 재사용)                    │
└──────────────────────────────────────────────────────┘
                       │
                       │ db.AsUser 트랜잭션 (사용자 holdings 조회)
                       │ + 슈퍼유저 풀 (공개 prices/fx 조회)
                       ▼
┌──────────────────────────────────────────────────────┐
│  Supabase Postgres                                   │
│   - public.holdings (사용자 보유)                    │
│   - public.prices (5년 일봉)                          │
│   - public.fx_rates (시점별 환율)                     │
│   - public.profiles.created_at (가입일 — 빈 상태 판정)│
└──────────────────────────────────────────────────────┘
```

### Why 신규 패키지 `internal/portfolio/`

- 알파 계산 로직이 단순 핸들러 1개를 넘어선다 (시점별 가격·환율 JOIN + 시계열 합산 + 정규화)
- Phase 2 백테스트·AI 매매 일기에서도 동일 시계열 계산 재사용 예정 → 공통 인프라
- 신규 디렉토리: `apps/api/internal/portfolio/{alpha.go, alpha_test.go}`

---

## 4. API 명세

### 요청

```http
GET /v1/portfolio/alpha?period=90d
Authorization: Bearer <jwt>
```

`period` 값: `1m` · `90d` · `1y` · `all`. 잘못된 값 → 400.

### 응답 (200)

```json
{
  "period": "90d",
  "days_requested": 90,
  "days_used": 90,
  "since": "2026-02-27",
  "fx_mode": "spot",
  "model": "current_holdings_backward_simulation",
  "portfolio": {
    "total_return_pct": 11.84,
    "series": [
      { "date": "2026-02-27", "value_pct": 0.0 },
      { "date": "2026-02-28", "value_pct": 0.42 },
      // ... 90일치 일봉 (영업일만, ~63개)
      { "date": "2026-05-28", "value_pct": 11.84 }
    ],
    "data_gaps": [
      { "symbol": "신규상장종목", "first_price_date": "2026-04-10" }
    ]
  },
  "benchmarks": [
    {
      "key": "kospi",
      "label": "KOSPI",
      "total_return_pct": 3.42,
      "alpha_pp": 8.42,
      "series": [{ "date": "2026-02-27", "value_pct": 0.0 }, /* ... */]
    },
    {
      "key": "sp500",
      "label": "S&P 500",
      "total_return_pct": 13.94,
      "alpha_pp": -2.10,
      "series": [/* ... */]
    },
    {
      "key": "kr_us_6040",
      "label": "한미 60/40",
      "total_return_pct": 7.31,
      "alpha_pp": 4.53,
      "series": null
    }
  ]
}
```

### 응답 (422 — 빈 상태)

```json
{
  "error": {
    "code": "INSUFFICIENT_DATA",
    "reason": "account_too_young",   // "account_too_young" | "no_holdings"
    "message": "7일 이상 보유 후 표시됩니다",
    "min_days": 7,
    "current_days": 3
  }
}
```

### Why series에 알파 자체를 포함시키지 않는가

알파(포트 - 벤치마크)는 클라이언트에서 series 차감으로 계산 가능. 응답 크기 ↓ + 토글 시 차트 재계산 없음.

---

## 5. 계산 알고리즘 (의사 코드)

```text
Compute(ctx, exec, uid, period):
  # 1. 기간 파싱
  days = parse_period(period)              # "90d" → 90
  since = today - days
  
  # 2. 가입일 판정
  created_at = profiles.select(uid).created_at
  account_days = today - created_at
  if account_days < 7:
    return ERR_INSUFFICIENT_DATA(account_too_young, current=account_days)
  if account_days < days:
    since = created_at                     # 단축 계산
    days_used = account_days
  
  # 3. 현재 보유 종목·수량
  holdings = holdings_repo.list(exec, uid)
  if holdings.empty:
    return ERR_INSUFFICIENT_DATA(no_holdings)
  
  # 4. 영업일 시계열 (since ~ today)
  trading_days = prices_repo.distinct_dates(KOSPI, since, today)
  
  # 5. 종목별 시계열 (CLOSE × 환율 × 수량 → KRW)
  port_series = []
  data_gaps = []
  for d in trading_days:
    total = 0
    for h in holdings:
      price = prices_repo.close(exec=pool, h.instrument_id, d)
      if price is None:
        if h.first_available_after(since):  # 신규 상장
          data_gaps.append(h.symbol, h.first_price_date)
          continue                          # 이 일자에 이 종목만 제외
      fx = fx_repo.rate(exec=pool, h.currency, "KRW", d)
      total += h.quantity * price * fx
    port_series.append((d, total))
  
  # 6. 누적 수익률 % (시작점 = 0%)
  start_value = port_series[0].value
  port_pct = [(d, (v - start_value) / start_value * 100) for d, v in port_series]
  
  # 7. 벤치마크 시계열
  kospi = prices_repo.close_series(KOSPI, since, today) → 누적 %
  sp500 = prices_repo.close_series(SP500, since, today) → 누적 %
  kr_us_6040 = 0.6 * kospi + 0.4 * sp500  # 시점별 가중 평균 누적 %
  
  # 8. 알파 = 포트 - 벤치마크 (last value)
  alpha_kospi = port_pct[-1] - kospi[-1]
  ...
  
  return {
    portfolio: { total_return_pct, series, data_gaps },
    benchmarks: [{ key:"kospi", series, alpha_pp }, ...]
  }
```

### 환율 처리 (D4)

각 일자 `d`마다 `fx_rates.rate(currency, "KRW", d)`. fx 누락 일자 → 직전 유효 환율 forward-fill (frankfurter는 영업일만 발급, 주말·공휴일 채움). KRW 종목은 환율 1.0 고정.

### 한미 60/40 (D3)

`0.6 * kospi_return_pct(d) + 0.4 * sp500_return_pct(d)`. 시작점 정규화 후의 누적 % 값에 가중. 분리한 두 자산을 매일 리밸런싱하는 가정과 등가.

### 데이터 부족 처리

- 종목의 `since` 시점 가격 없음(신규 상장): 해당 종목을 `data_gaps`에 추가하고 그 일자에 제외. 첫 가격 보유일부터 합류 → 시계열 시작 시점에 그 종목 비중만 비는 효과.
- 종목의 환율 누락: forward-fill 후에도 없으면 그 일자 통째 skip.
- 벤치마크(KOSPI/SP500) 자체 누락: 5년 백필이 끝났으면 발생 안 함. 발생 시 500 에러.

---

## 6. UI 명세

### 카드 레이아웃 (홈 1행 3번째)

```
┌──────────────────────────────────────┐
│ ALPHA       1M  [90D]  1Y  All       │  ← 헤더 + 토글 chip
├──────────────────────────────────────┤
│ vs KOSPI            +8.42%p          │  ← 텍스트 3줄
│ vs S&P 500          -2.10%p          │
│ vs 한미 60/40       +4.53%p          │
│                                      │
│  ╱╲    ╱──     포트                  │  ← SVG 3 라인
│ ╱  ╲╱╲╱        KOSPI                 │
│ ╱─────         S&P 500               │
│                                      │
│ 환율 변동 포함 · 현재 보유 기준 시뮬레이션 │  ← 라벨
└──────────────────────────────────────┘
```

### 시각 토큰

- 카드: `border border-line bg-bg-subtle p-5` (다른 홈 카드와 동일)
- 토글 chip: 선택 = `text-bb-accent border-bb-accent`, 비선택 = `text-fg-muted border-line`
- 양수 알파: `text-bb-up` (#00FF7F)
- 음수 알파: `text-bb-down` (#FF3344)
- 차트 라인: 포트 = `#FFD500`, KOSPI = `#00FFFF`, S&P = `#FF9900` (3색 구분 용이)
- 라벨: `font-mono text-[10px] text-fg-muted`

### 토글 동작

- 클라이언트 상태(`useState`)에 현재 period 보관 → 변경 시 `getAlpha(period)` 재호출
- 로딩 중 — 차트만 회색 처리 + 텍스트는 직전 값 유지(깜빡임 방지)
- 응답 도착 → 차트 transition (CSS opacity 200ms)

### 빈 상태 (422)

```
┌──────────────────────────────────────┐
│ ALPHA                                │
├──────────────────────────────────────┤
│ 7일 이상 보유 후 표시됩니다          │
│                                      │
│ 가입 3일째 — 4일 후부터 비교 가능    │
└──────────────────────────────────────┘
```

`no_holdings`인 경우: "보유 자산 추가 후 표시됩니다" + 포트폴리오 페이지 링크.

---

## 7. 성능·캐싱

- 매 토글마다 API 호출. 90일 = 영업일 ~63개 × 5종목 가정 = 315 점. 응답 ~5KB.
- 백엔드 쿼리: holdings 1회 + prices range 1회 + fx range 1회 = JOIN 1번 또는 3 쿼리.
- 캐싱: MVP에서는 없음. 사용자별 데이터라 캐시 무효화 복잡. 일봉이라 분당 갱신 불필요 → Phase 2에 redis 또는 in-memory TTL 1시간 검토.

---

## 8. 에러 처리

| 상황 | 응답 | 카드 표시 |
|---|---|---|
| 사용자 미인증 | 401 | (라우터가 차단, 카드 안 마운트) |
| `period` 잘못된 값 | 400 | "잘못된 기간" 에러 + 토글 직전 값 유지 |
| 가입 < 7일 | 422 `account_too_young` | "7일 이상 보유 후 표시" + D-day |
| 보유 자산 0 | 422 `no_holdings` | "보유 자산 추가 후 표시" + 링크 |
| 벤치마크 데이터 누락 | 500 | "일시적 오류 — 잠시 후 다시" |
| 종목별 가격 누락 | 200 + `data_gaps` 채움 | 정상 표시 + "*N개 종목 데이터 부족" 작은 라벨 |

---

## 9. 테스트

### Backend unit (`alpha_test.go`)

| 케이스 | 입력 | 기대 |
|---|---|---|
| `Compute_BasicTwoHoldings` | KRW 종목 1 + USD 종목 1, mock prices/fx | 알려진 알파 값 일치 |
| `Compute_AccountTooYoung` | created_at = today - 3일 | `ERR_INSUFFICIENT_DATA(account_too_young)` |
| `Compute_NoHoldings` | holdings 0개 | `ERR_INSUFFICIENT_DATA(no_holdings)` |
| `Compute_AccountYoungerThanPeriod` | gain 30일, period=90d | `days_used=30` + 단축 계산 |
| `Compute_NewListing` | 종목 첫 가격이 since 이후 | `data_gaps`에 등록 + 그 일자 제외 |
| `Compute_FxForwardFill` | fx 1일 결손 | 직전 값 forward-fill 사용 |

### Backend integration (`alpha_integration_test.go`)

- 로컬 Supabase + 실 마이그레이션 + 시드 KOSPI/SPX + mock 사용자 + holdings 2종목 → 종단 응답 검증
- RLS 격리 확인: 사용자 A의 holdings로 계산한 결과를 B 컨텍스트에서 호출 시 422 또는 다른 결과

### Frontend (`AlphaCard.test.tsx`)

- 정상 응답 → 3줄 텍스트 + 3 라인 차트 렌더
- 422 `account_too_young` → 빈 상태 메시지 + D-day
- 422 `no_holdings` → 빈 상태 + 링크
- 토글 클릭 → `getAlpha` 재호출 + 로딩 중 직전 값 유지

---

## 10. 비범위 (YAGNI)

- ❌ TWR/MWR 정확 계산 — Phase 2 transactions 테이블 도입 후
- ❌ Sharpe·Sortino 비율 — 변동성·무위험 수익률 추가 필요. v2
- ❌ 종목별 알파 분해 — 어느 종목이 알파에 기여했나. v2
- ❌ 사용자 정의 벤치마크 — KOSPI/S&P/60-40 고정. v2
- ❌ 알파 자체 시계열 차트(2 라인 차이) — 누적 수익률 3 라인이 더 직관적
- ❌ 실시간 갱신 — 일봉 기반이라 1일 1회면 충분
- ❌ AI 분석 통합 — "알파가 왜 +8%인가" 같은 질문은 AI 채팅에서. 카드는 숫자만

---

## 검토 이력

### 2026-05-28 초안 작성 + 자체 검토

#### Critical (구현 시 동작 안 함 또는 핵심 가정 오류) → 1건 → 패치 완료

**C-1. 한미 60/40 합산 방식 모호** — 초안에 `0.6 * kospi_return + 0.4 * sp500_return`라고만 적었으나, 두 가지 해석 가능:
- (a) **시점별 가중 평균 누적 %** — 매일 60/40 비중 재조정 (continuous rebalancing). 산식 정확.
- (b) **시작 시점 자금 분배 + buy-and-hold** — 시작 시 60/40 분배 후 무리밸런싱. 시점별 비중은 시장 변동에 따라 drift.
- 두 방식의 결과 차이는 변동성 큰 기간에 1~2%p까지 벌어진다. 명시 안 하면 구현자가 임의 선택.

→ §5 "한미 60/40" 절을 (a) **시점별 가중 평균(continuous rebalancing)** 명시로 수정. 단순 + 명확 + KOSPI/S&P 두 시계열이 이미 누적 % 정규화된 상태에서 가중 평균이라 계산 비용 0. 라벨에는 "* 일별 리밸런싱 가정" 추가 권장 — Phase 2 사용자 review에서 결정.

#### Important (보강 필요) → 3건 → 모두 패치 완료

**I-1. data_gaps 처리 시 비중 왜곡** — 신규 상장 종목을 그 일자에 제외하면 그 일자 포트 가치가 그 종목 가치만큼 빠짐 → "다음 일자에 갑자기 합류"하면 누적 수익률 시계열에 점프 발생.
→ §5에 "첫 가격 보유일부터 합류 → 시계열 시작 시점에 그 종목 비중만 비는 효과" 명시. 단순 누락 처리 + UI 라벨로 사용자에게 알림. 정확한 시점별 비중 재계산은 transactions 테이블 도입 후 Phase 2.

**I-2. days_used 표시 일관성** — 응답에 `days_used`와 `since` 둘 다 있어 클라이언트가 어느 것을 표시할지 모호.
→ §4 응답 예시에 `days_requested`, `days_used`, `since` 셋 다 두고, 빈 상태가 아닌 정상 경로에서 `days_used < days_requested`이면 UI가 라벨 "가입 N일" 표시. 명확.

**I-3. 토글 로딩 중 깜빡임** — 토글 클릭마다 fetch → 로딩 spinner → 새 값 표시. 정보 밀도 톤과 안 맞음.
→ §6 "토글 동작"에 "직전 값 유지 + 차트만 회색 처리, 응답 도착 시 200ms transition" 명시. 빈 카드 보여주는 깜빡임 방지.

#### Minor (명확성) → 2건 → 모두 패치 완료

**M-1. 60/40 series가 null인 이유 미설명** — 응답 예시에 `kr_us_6040.series: null`을 두었지만 왜 null인지 unclear.
→ §2 "Why D7" 절을 추가 — 4 라인이 빽빽해서 60/40은 텍스트만, 차트는 KOSPI·S&P 2 라인만.

**M-2. 빈 상태 422 vs 200 + flag** — 일반적으로 빈 상태는 200 응답에 빈 데이터로 표현하지만 본 spec은 422 사용. 이유 미설명.
→ §4에 빈 상태는 의미적으로 "분석 불가능"이라 422가 적합 명시(404 not found도 아니고, 200 + 빈 series는 카드 렌더링 후 빈 차트 그리는 비용 발생). 카드는 422 응답 시 별도 분기로 빈 상태 UI 렌더.

#### 메타

본 spec은 절차 강화 후 첫 brainstorming → spec 사이클. CLAUDE.md MANDATORY 흐름(brainstorm → spec → plan → 구현) 준수.
사용자 review gate 거친 후 `superpowers:writing-plans` 호출 예정. Plan은 작성 후 subagent 자체 검토 의무.
