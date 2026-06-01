# Paper Trading 백테스트 (서브시스템 B) — 설계 스펙

> 과거 시점 시뮬레이션. "이렇게 투자했다면 시장 대비 어땠나"를 규칙(바스켓+전략)으로 자동 검증.
> 라이브 Paper Trading(지금부터 손으로 매매)과 역할 분리. 정체성 spec §1 3축의 "백테스트" 약속 완성.

**날짜**: 2026-05-29
**저자**: 사용자 + 에이전트 (brainstorming 1 사이클, visual companion)
**상태**: 디자인 확정. 구현 plan 작성 단계 진입 예정.
**관련 spec**: [`2026-05-28-identity-3-pillars.md`](./2026-05-28-identity-3-pillars.md) §1, [`2026-05-28-paper-trading-design.md`](./2026-05-28-paper-trading-design.md) (라이브 — 본 spec과 분리), [`2026-05-28-alpha-card-design.md`](./2026-05-28-alpha-card-design.md) (시계열 인프라 재사용처)
**관련 변경**: 신규 패키지 `internal/portfolio/backtest.go`, 신규 핸들러 1개, 신규 페이지 `/app/backtest`, `Deps`에 메서드 1개 추가, 사이드바 신규 아이콘. **신규 테이블 0개** (무상태).
**비목적**: 라이브 매매(서브시스템 = 별도 spec), 런 저장, 채팅 도구.

---

## 1. 목적

사용자가 바스켓(종목+목표 비중)과 전략(투입 방식·리밸런싱)을 선언하면, 과거 데이터로 시뮬레이션해 **평가액 곡선 + 벤치마크 비교 + 위험·수익 지표**를 산출한다.

답하려는 질문: *"지난 3년간 삼성전자 40% · 애플 30% · KODEX200 30%에 매월 50만원씩 적립하고 분기마다 리밸런싱했다면, KOSPI·S&P·한미 60/40 대비 어땠을까?"*

**라이브 Paper와의 경계**:
- 라이브 = "지금부터" 손으로 매수/매도, 상태 보존(테이블).
- 백테스트 = "과거를" 규칙으로 자동 재현, 무상태(입력만 있으면 언제든 재계산).

---

## 2. 핵심 결정 (brainstorm 합의)

| # | 영역 | 결정 |
|---|---|---|
| D1 | 입력 모델 | **선언적** — 바스켓 + 목표 비중 + 전략. 자유 매매 시퀀스(타임머신)는 라이브와 중복이라 채택 X |
| D2 | 전략 모델 | **2축 직교** — 투입 방식(일시불 / 월 적립) × 리밸런싱(없음 / 분기·반기·연). 특수 케이스 3개 대신 토글 2개 |
| D3 | 데이터 범위 | **공통 구간 클램프** — 바스켓 전 종목의 최초 데이터일 중 가장 늦은 날로 시작일 자동 조정 + 안내 |
| D4 | 출력 | 평가액 곡선(KRW) + 투입 원금 기준선 + 벤치마크 3종 + 지표(총수익률·CAGR·MDD·변동성·초과수익). **벤치마크 = 동일 현금흐름 투입** |
| D5 | 저장 | **무상태(on-demand)** — 테이블 없음. 라이브 D5(snapshot 없이 매번 계산) 철학 일관 |
| D6 | AI 통합 | **페이지 전용** — 채팅 `run_backtest` 도구는 v2 (텍스트 요약 + 딥링크 형태) |

### Why D1 선언적 입력

자유 매매 시퀀스는 "과거 시점에서 시작하는 라이브 Paper"에 불과 — 기능 중복 + 5년치 수동 클릭 부담. 선언적 입력은 자동 시뮬이라 "전략 검증" 본질에 부합하고, 정체성 §1의 "DCA·리밸런싱" 약속과 정확히 일치.

### Why D2 2축 직교

"딱 3개 프리셋"보다 `투입 × 리밸런싱` 직교 모델이 **사용자에겐 토글 2개로 단순**하고, **구현은 if-else 분기 대신 단일 시뮬 루프**(매 영업일 DCA 체크 + 리밸런싱 체크)라 더 깨끗하다. "월 적립 + 분기 리밸런싱" 같은 현실 조합도 자연 표현. Buy&Hold = 둘 다 off.

### Why D4 동일 현금흐름 벤치마크

내 전략이 매월 ₩50만 DCA면, KOSPI 벤치마크도 **같은 시점에 같은 금액을 KOSPI에 투입**한 것으로 계산해야 apples-to-apples. 단순히 "KOSPI 지수 시작~끝 수익률"과 비교하면 적립 타이밍 효과가 누락돼 불공정. → §5 통합 엔진에서 전략·벤치마크를 **같은 `simulate()`로 4번 호출**.

### Why D5 무상태

백테스트는 입력(바스켓+전략+기간)만 있으면 결정적으로 재현된다. 저장 가치가 낮고, `backtest_runs` 테이블 + 목록/삭제/이름짓기 UI는 스코프만 키운다. 계산은 배치 조회(§5)로 sub-second. "런 저장·비교"는 수요 확인 후 v2.

### Why D6 페이지 전용

백테스트 결과(차트+지표 표)는 SSE 채팅 안 렌더링이 어색하고 페이지가 본질적 산출물. 채팅 도구는 엔진 위에 얹는 추가 레이어 → v2에서 `run_backtest`(요약 + 파라미터 채워진 페이지 딥링크)로 통합하는 게 더 깔끔.

---

## 3. 데이터 / 의존

### 3-1. 신규 테이블 — 없음

백테스트는 사용자 데이터를 **읽지도 쓰지도 않는다**. 바스켓은 요청 본문으로 받고, 가격·환율·지수·종목 메타는 모두 공개 데이터(슈퍼유저 풀). → `db.AsUser`(RLS 트랜잭션) 불필요. 라이브 Paper보다 단순.

### 3-2. `Deps` 인터페이스 확장 (1개 메서드 추가)

알파 카드의 `internal/portfolio/Deps`를 재사용하되, 바스켓 종목의 **통화·자산군**을 알아야 한다(fx 적용 + INDEX/FX/CASH 가드).

```go
// internal/portfolio/alpha.go — Deps에 추가
InstrumentsMeta(ctx context.Context, pool *pgxpool.Pool, ids []string) (map[string]InstrumentMeta, error)

type InstrumentMeta struct {
    Symbol     string
    Name       string
    Currency   string // "KRW" | "USD"
    AssetClass string // "KR_STOCK" | "US_STOCK" | "INDEX" | "FX" | "CASH" ...
}
```

`pg_deps.go`에 구현 추가:
```sql
select id::text, symbol, name, currency, asset_class
from public.instruments where id = any($1)
```

기존 재사용 메서드: `TradingDays`, `InstrumentClosesOnDates`, `FxRatesOnDates`, `BenchmarkSeries`. 기존 헬퍼 재사용: `lookupFxForward`(전진 채움 환율), `firstAvailable`, `SeriesPoint`, `Period`.

### 3-3. 데이터 범위 제약 (하드)

- **최대 lookback ~5년** — 백필 CLI 기본 윈도우(`cmd/backfill -years 5`). 기간 옵션: `1Y · 3Y · 5Y · 전체`.
- **US 커버리지 = 30개 NASDAQ 종목** (`cmd/backfill` `nasdaqSeed`). KR은 KIND 전체 마스터. Phase 2에서 US 확장.
- 신규 상장 종목은 5년 미만 → §6 클램프로 처리.
- **백테스트는 백필 데이터 의존** → 프로덕션에서 backfill CLI 선행 실행 필요 (`docs/USER_ACTIONS.md` 등재 대상).

---

## 4. Architecture

```
Frontend (apps/web)
┌─────────────────────────────────────────────────────────┐
│ /app/backtest 페이지 (apps/web/app/app/backtest/page.tsx)│
│  - BacktestForm                                          │
│     · PeriodPicker (1Y·3Y·5Y·전체)                       │
│     · CashInputs (초기 자금 · 투입 방식 · 월 적립금)     │
│     · BasketBuilder (InstrumentSearchInput 재사용 + 비중)│
│     · RebalanceSelect (없음·분기·반기·연)                │
│  - BacktestResults                                       │
│     · MetricCards · EquityChart(다중선) · CompareTable   │
│     · CoverageNotice (클램프·커버리지 경고)              │
│ Sidebar 신규 아이콘 (lucide History, label "백테스트")  │
└─────────────────────────────────────────────────────────┘
                            │ POST /v1/backtest/run
                            ▼
Backend (apps/api)
┌─────────────────────────────────────────────────────────┐
│ handlers/backtest.go                                     │
│  - RunBacktest: 인증 → 검증 → 해석 → 엔진 → 응답         │
│    (사용자 데이터 미접근 → 슈퍼유저 풀 읽기만)          │
│ internal/portfolio/backtest.go (신규)                    │
│  - Service.Run: 바스켓·벤치마크 해석 + 클램프            │
│  - simulate(): 순수 함수 — NAV/유닛 시뮬 (DB 무관)       │
│  - metrics(): 총수익률·CAGR(XIRR)·MDD·변동성             │
│  - xirr(): Newton 반복                                   │
│ internal/portfolio/pg_deps.go (확장)                     │
│  - InstrumentsMeta 추가                                  │
└─────────────────────────────────────────────────────────┘
                            │ 슈퍼유저 풀 (공개 데이터)
                            ▼
Supabase Postgres
 - public.prices (일봉 종가), public.fx_rates (환율)
 - public.instruments (종목 메타), 지수 종가 (prices)
```

### Why 순수 `simulate()` 분리

DB 해석(가격·환율·메타 조회)과 시뮬 로직을 분리하면, `simulate()`를 **합성 시계열로 단위 테스트** 가능(DB·Supabase 불필요). 알파/paper_equity가 `Deps` 모킹으로 테스트하는 패턴과 동일하지만, 여기선 한 단계 더 — simulate는 가격 맵만 받는 순수 함수.

---

## 5. 시뮬레이션 엔진 (핵심)

### 5-1. 통합 모델 — "모든 것은 바스켓"

전략과 벤치마크를 **하나의 `simulate()`**로 처리한다. 벤치마크 = 사전정의 바스켓:

| 벤치마크 | 바스켓 |
|---|---|
| KOSPI | `{KOSPI 100%}` |
| S&P 500 | `{SPX 100%}` |
| 한미 60/40 | `{KOSPI 60%, SPX 40%}` |

전략 1회 + 벤치마크 3회 = `simulate()` 4회 호출. **모두 동일 현금흐름(초기 + 월 적립)·동일 리밸런싱 설정**. 단일 종목 벤치마크(KOSPI·SPX)는 리밸런싱이 no-op, 60/40은 전략과 같은 주기로 리밸런싱.

> 지수(KOSPI·SPX)는 `asset_class='INDEX'`라 사용자 바스켓에선 거부(§9)되지만, 벤치마크는 시스템 정의이므로 내부적으로 사용. 모순 아님.

**벤치마크 leg 단위·통화 계약**: `BenchmarkSeries`가 반환하는 **원시 지수 레벨**을 leg의 `Closes`로 쓴다. KOSPI leg는 `FxToKRW=1.0`(원화 지수), SPX leg는 `FxToKRW=USD/KRW`(달러 지수→원화). 각 leg는 §5-3에서 자기 t0 종가로 정규화되어 "지수 포인트 vs 주가" 절대 스케일은 상쇄되고 fx만 통화 일관성을 위해 적용된다. (BenchmarkSeries가 이미 KRW 환산본을 반환하면 SPX도 `FxToKRW=1.0` — 이중 환산 금지.)

### 5-2. 순수 함수 시그니처

```go
type Leg struct {
    Weight   float64            // 정규화된 목표 비중 (Σ=1.0)
    Closes   map[string]float64 // date("2006-01-02") → 종가 (매매 통화 기준)
    FxToKRW  map[string]float64 // date → 환율 (KRW=1.0). 전진 채움은 lookupFxForward
}

type Plan struct {
    Initial float64 // 초기 자금 (KRW)
    Monthly float64 // 월 적립금 (KRW). 0이면 일시불(lump)
}

type Rebalance int // None, Quarterly, Semiannual, Annual

func simulate(tradingDays []string, legs []Leg, plan Plan, rb Rebalance) SimOutput

type SimOutput struct {
    Equity           []SeriesPoint // 일자별 평가액 (KRW) — 사용자 표시
    NAV              []SeriesPoint // 일자별 NAV (시작 1.0) — 위험·수익 지표용
    Contributed      []SeriesPoint // 일자별 누적 투입 원금 (KRW) — 기준선
    TotalContributed float64
    FinalEquity      float64
    Cashflows        []Cashflow    // XIRR 입력: (음수 투입, date) ... (양수 최종, lastDate)
}

type Cashflow struct {
    Amount float64 // 음수=투입(유출), 양수=최종 평가액(유입)
    Date   string  // "2006-01-02"
}
```

### 5-3. NAV(유닛) 방식 — 적립 왜곡 제거 (정확성 핵심)

적립이 있으면 평가액이 오르는 게 일부는 "돈을 더 넣어서"다. 단순 평가액 % 변화로 MDD·변동성·TWR을 재면 **왜곡**된다. → 펀드 유닛 회계로 분리:

상태: `shares map[legIdx]float64`(분수 주식), `fundUnits float64`.

**시작일 `t0` (클램프된 시작)** — 전제 `Initial > 0` (§9에서 강제; 0이면 NAV(t0)=0/0=NaN):
```
각 leg i: alloc_i = Initial * w_i
          shares_i = alloc_i / (Closes_i[t0] * Fx_i[t0])
V(t0) = Initial            // 정의상
fundUnits = Initial        // NAV(t0) = V/fundUnits = 1.0
TotalContributed = Initial          // ★ 초기 자금도 투입 원금에 포함
Cashflows = [ (-Initial, t0) ]      // ★ XIRR 최초 유출 (없으면 총수익률·CAGR 전부 틀림)
```

**매 영업일 `t` (t0 이후)**:
```
V(t)   = Σ shares_i * Closes_i[t] * Fx_i[t]      // 결측 종가는 전진 채움
NAV(t) = V(t) / fundUnits
```

**발생일 판정 (적립·리밸런싱 공통, 커서 방식)**: `nextContrib`·`nextRebal` 커서를 t0의 +1개월/+1주기로 초기화한다(t0 당일은 적립·리밸런싱 모두 없음). 영업일 루프에서 `t >= nextContrib`이면 적립 후 커서 +1개월, `t >= nextRebal`(rb≠None)이면 리밸런싱 후 커서 +1주기. 월말(1/31→2/28 등)은 달력 월 연산으로 클램프, 주말·휴일은 "직후 첫 영업일"에 1회만 실행(누락·중복 없음). 3년·월적립 = 적립 36회(t0+1m … t0+36m).

**월 경계 (Monthly>0, t0 기준 매월 기념일 당일 또는 직후 첫 영업일)** — 적립:
```
C = Monthly
newUnits = C / NAV(t);  fundUnits += newUnits      // 현재 NAV로 유닛 발행 → NAV 불변
각 leg i: shares_i += (C * w_i) / (Closes_i[t] * Fx_i[t])   // 목표 비중으로 매수
TotalContributed += C;  Cashflows += (-C, t)
```
> 증명: 적립 전 `V0, U0, NAV0=V0/U0`. 적립 후 `V1=V0+C`, `U1=U0+C/NAV0=U0(V0+C)/V0`, `NAV1=V1/U1=V0/U0=NAV0`. ✓ 적립이 NAV를 점프시키지 않음.

**리밸런싱 경계 (rb≠None, t0 기준 주기 기념일 당일/직후 첫 영업일, t0 자체 제외)** — 재조정(신규 현금 X):
```
각 leg i: shares_i = (V(t) * w_i) / (Closes_i[t] * Fx_i[t])   // 목표 비중 복원
// V·fundUnits·NAV 불변
```
> 같은 날 적립+리밸런싱이 겹치면 **적립 먼저, 리밸런싱 나중**. 리밸런싱의 `V(t)`는 **적립 반영 후 평가액(직전 V + C)**을 다시 계산해 쓴다(적립분까지 목표 비중으로 배분). 하루치 V를 캐시해 재사용하면 적립 현금이 미배분되는 버그.

**종료일 `tN`**:
```
FinalEquity = V(tN);  Cashflows += (+FinalEquity, tN)
```

### 5-4. 분수 주식 + 현금 드래그

분수 주식 허용(잔여 현금 노이즈 제거). 초기·적립 자금은 **당일 전액 목표 비중 매수** → 현금 잔고 ≈ 0. 라이브의 "수수료·슬리피지 0"과 같은 이상화. (현실 모드는 비범위.)

### 5-5. 영업일·전진 채움

`tradingDays` = `Deps.TradingDays(clampedStart, today)` — **KOSPI 거래일 캘린더를 전 종목 공통 축**으로 쓴다(알파와 동일). all-US 바스켓도 이 축에서 표본되며, leg별 종가/환율 결측 시 직전 값 전진 채움(`lookupFxForward` 환율, 종가는 동일 패턴 헬퍼). 시작일 클램프로 모든 leg에 종가 존재 보장. USD leg는 추가로 `clampedStart`에 환율이 존재해야 한다 — `fx_rates`는 백필이 가격 윈도우(≥5년) 이상으로 적재하므로 보장되고, 혹시 USD leg의 fx firstAvailable이 더 늦으면 그 값도 clampedStart 클램프에 포함한다(가격과 동일 처리).

---

## 6. 시작일 클램프 + 커버리지 경고

```
요청: period(1Y/3Y/5Y/전체) → requestedStart = today - period (전체=today-5y)
firstAvailable(최초 종가일) 조회 대상 = 전략 leg ∪ 벤치마크 구성 {KOSPI, SPX}
clampedStart = max(requestedStart, max over (전략 legs ∪ {KOSPI, SPX}) firstAvailable)
```

- `clampedStart > requestedStart`이면 응답 `coverage_warnings`에 사유별 경고:
  `{ symbol, first_available, message: "데이터가 2022-03-15부터 존재해 시작일을 조정했습니다" }`
- 벤치마크(KOSPI·SPX)의 `firstAvailable`도 위 max에 포함 → 전략·벤치마크가 **동일 clampedStart·동일 영업일 축**을 공유(공통 윈도우 보장, apples-to-apples). 보통 지수는 5년 풀 존재라 전략 leg가 max를 지배.
- **clampedStart ~ today 영업일 < 30** → 422 `INSUFFICIENT_DATA` (알파 `InsufficientDataError{Reason, MinDays, CurrentDays}` 재사용).

---

## 7. 지표 계산 (`metrics()`)

`SimOutput`에서 계산. 전략·벤치마크 모두 동일 함수.

| 지표 | 정의 | 비고 |
|---|---|---|
| 총수익률 | `(FinalEquity − TotalContributed) / TotalContributed` | 투입 대비, 직관적 |
| CAGR | **XIRR** (머니가중) — `Cashflows`의 NPV=0 되는 연이율 | 일시불·적립 모두 통일 처리. 일시불은 `(V/Initial)^(1/년)−1`로 자연 수렴 |
| MDD | `NAV` 시계열의 최대 고점-저점 낙폭 | 적립 왜곡 제거(§5-3) |
| 변동성 | `NAV` 일간 수익률 표준편차 × √252 | 연환산 |
| 초과수익 | 전략 TWR − 벤치마크 TWR, `TWR = NAV(tN) − 1` | 60/40 기준 메인 카드. 표엔 3종 모두. **누적·시간가중**(머니가중 총수익률과 측정축 다름 → UI에 "누적·시간가중" 명기) |

`func metrics(out SimOutput, days []string) SeriesMetrics` — 한 시계열(전략 또는 개별 벤치마크)의 {총수익률, CAGR, MDD, 변동성, TWR}를 산출. **초과수익은 per-series가 아니라** `Service.Run`에서 `전략.TWR − 벤치마크.TWR`로 계산해 전략 metrics에만 싣는다(§8 응답: 벤치마크 metrics엔 초과수익 없음).

### 7-1. XIRR (Newton 반복)

```
f(r) = Σ_k CF_k / (1+r)^(yearfrac_k)
yearfrac_k = (date_k − date_0) / 365.0
f'(r) = Σ_k CF_k · (−yearfrac_k) / (1+r)^(yearfrac_k + 1)
Newton: r ← r − f(r)/f'(r), 초기값 0.1, 최대 100회 또는 |f|<1e-6
실패(미수렴·부호 단일)시: 이분법 폴백 [-0.99, 10.0]. 그래도 실패 → null(응답에 cagr=null, UI "—")
```

> XIRR은 비표준 현금흐름에서 미수렴/다중해 가능 → 폴백 + null 처리 명시. 단위 테스트로 알려진 케이스 검증.

---

## 8. API 명세

### `POST /v1/backtest/run`

복잡한 바스켓 배열 본문이라 POST. 무상태(저장 없음). 인증 필수.

**Request**:
```json
{
  "period": "3Y",
  "initial_cash": 10000000,
  "monthly_contribution": 500000,
  "basket": [
    { "instrument_id": "uuid-삼성", "weight": 40 },
    { "instrument_id": "uuid-aapl", "weight": 30 },
    { "instrument_id": "uuid-kodex200", "weight": 30 }
  ],
  "rebalance": "quarterly"
}
```
- `period` ∈ `1Y·3Y·5Y·all`. `monthly_contribution=0` → 일시불. `rebalance` ∈ `none·quarterly·semiannual·annual`.
- `weight`는 양수 임의값 → 서버에서 합=1.0 정규화.

**Response (200)**:
```json
{
  "clamped_start": "2023-05-30",
  "end": "2026-05-29",
  "normalized_basket": [
    { "instrument_id": "uuid-삼성", "symbol": "005930", "name": "삼성전자", "weight": 0.4 }
  ],
  "equity_series":      [ { "date": "2023-05-30", "value": 10000000 }, ... ],
  "contributed_series": [ { "date": "2023-05-30", "value": 10000000 }, ... ],
  "benchmarks": {
    "kospi":      { "equity_series": [...], "metrics": {...} },
    "spx":        { "equity_series": [...], "metrics": {...} },
    "sixty_forty":{ "equity_series": [...], "metrics": {...} }
  },
  "metrics": {
    "total_return_pct": 42.3,
    "cagr_pct": 8.1,
    "mdd_pct": -18.4,
    "volatility_pct": 14.2,
    "excess_vs_6040_pct": 6.5,
    "total_contributed": 28000000,
    "final_equity": 39844000
  },
  "coverage_warnings": [
    { "symbol": "AAPL", "first_available": "2023-05-30", "message": "데이터가 2023-05-30부터 존재해 시작일을 조정했습니다" }
  ]
}
```

> `equity_series`·`contributed_series`는 KRW. 차트는 전략 곡선 + 벤치마크 3선 + 투입 원금 점선. `cagr_pct`는 XIRR 실패 시 `null`.

---

## 9. 에러 처리

| 상황 | 응답 | UI |
|---|---|---|
| 미인증 | 401 | 라우터 차단 |
| 바스켓 비었음 / >10종목 | 422 `VALIDATION` | "종목을 1~10개 선택하세요" |
| 비중 합 ≤ 0 또는 음수 비중 | 422 `VALIDATION` | "비중은 0보다 커야 합니다" |
| INDEX·FX·CASH 종목 포함 | 422 `ASSET_NOT_SUPPORTED` | "지수·환율은 백테스트 불가" |
| instrument_id 미존재 | 422 `VALIDATION` | "종목을 찾을 수 없습니다" |
| 클램프 후 공통 구간 < 30영업일 | 422 `INSUFFICIENT_DATA` (Min·Current 포함) | "기간이 너무 짧습니다 (최소 30영업일)" |
| initial_cash ≤ 0 | 422 `VALIDATION` | "초기 자금은 0보다 커야 합니다" |
| monthly_contribution < 0 | 422 `VALIDATION` | "월 적립금은 0 이상이어야 합니다" |
| 데이터 짧아 시작일 조정 | 200 + `coverage_warnings` | 결과 상단 노란 배너 |
| XIRR 미수렴 | 200 + `cagr_pct=null` | CAGR "—" 표시 |

---

## 10. UI 명세

### 10-1. Sidebar

```
🏠 홈 / 💼 포트폴리오 / 💬 채팅 / 📊 마켓 / 📓 매매 일기 / 📈 Paper / ⏮ 백테스트 / ⚙️ 설정
```
lucide `History`(또는 `Rewind`) 아이콘, label "백테스트". Paper(📈 LineChart) 바로 아래. (`apps/web/components/shell/Sidebar.tsx`)

### 10-2. 입력 폼 (좌) → 결과 (우/하)

- **PeriodPicker**: 세그먼트 `1Y·3Y·5Y·전체` (종료=오늘).
- **CashInputs**: 초기 자금 + 투입 방식(일시불/월 적립) + 월 적립금(월 적립 선택 시만).
- **BasketBuilder**: 행마다 `InstrumentSearchInput`(재사용, INDEX/FX/CASH 비활성) + 비중(%) + 삭제. "＋ 종목 추가". 합계 표시(자동 정규화 안내). 최대 10행.
- **RebalanceSelect**: `없음·분기·반기·연`.
- **[백테스트 실행]** → POST → 로딩 → 결과.

### 10-3. 결과

- **MetricCards**: 총수익률 · CAGR · MDD · 변동성 · 초과수익(vs 60/40).
- **EquityChart**: 전략(굵은 선) + KOSPI·S&P·60/40 + 투입 원금(점선). KRW. 기존 차트 컴포넌트 패턴 재사용.
- **CompareTable**: 행=내전략·KOSPI·S&P·60/40, 열=총수익·CAGR·MDD.
- **CoverageNotice**: `coverage_warnings` 있으면 상단 배너.

### 10-4. 빈/초기 상태

페이지 진입 시 폼만. 실행 전 결과 영역은 "바스켓과 전략을 설정하고 실행하세요" 플레이스홀더.

---

## 11. 보안

- 백테스트는 **사용자 데이터 미접근** — `db.AsUser`/RLS 불필요. 인증 게이트(로그인 사용자)만 둔다.
- 가격·환율·종목·지수는 공개 데이터 → 슈퍼유저 풀 읽기. (마켓 엔드포인트와 동일 모델)
- 입력 검증: instrument_id UUID 형식·존재, 비중·금액 범위, 바스켓 크기 → 모두 422. SQL은 파라미터 바인딩(`= any($1)`).

---

## 12. 테스트

### 12-1. 엔진 단위 (`internal/portfolio/backtest_test.go`) — 순수, DB 무관

- `TestSimulate_LumpBuyHold_EquityTracksPrice` — 일시불·리밸X, 가격 2배 → 평가액 2배, NAV 2.0
- `TestSimulate_DCA_ContributionsMintUnits_NAVUnchanged` — 적립일에 NAV 불연속 없음(§5-3 증명 검증)
- `TestSimulate_Rebalance_RestoresTargetWeights` — 드리프트 후 리밸런싱일에 목표 비중 복원
- `TestSimulate_DCAplusRebalance_OrderCorrect` — 같은 날 적립→리밸런싱 순서
- `TestSimulate_TwoLegFx_USDLegConvertedKRW` — USD leg에 환율 적용
- `TestSimulate_ForwardFillMissingClose` — 결측 종가 전진 채움
- `TestMetrics_MDD_OnNAV_NotEquity` — 적립 있는 케이스에서 MDD가 평가액이 아닌 NAV 기준
- `TestMetrics_Volatility_Annualized`
- `TestXIRR_LumpSum_MatchesCAGR` — 일시불 XIRR == `(V/Initial)^(1/년)−1`
- `TestXIRR_DCA_KnownCashflows` — 알려진 현금흐름의 XIRR 수렴값
- `TestXIRR_NonConverging_ReturnsNull` — 폴백·null
- `TestSimulate_T0_RecordsInitialContribAndCashflow` — t0에 `TotalContributed=Initial`, `Cashflows[0]=(-Initial, t0)` (Critical 회귀 방지)
- `TestSimulate_DCA_ContributionCount` — 3년 월적립 = 36회, 첫 적립 t0+1개월(t0 당일 없음)

### 12-2. 서비스/클램프 (`backtest_test.go`, `Deps` 모킹)

- `TestRun_ClampsStartToCommonWindow` — leg별 firstAvailable 다를 때 max로 클램프 + 경고
- `TestRun_InsufficientData_422` — 공통 구간 < 30영업일
- `TestRun_NormalizesWeights` — 40/30/30 → 0.4/0.3/0.3
- `TestRun_BenchmarksUseSameCashflow` — 벤치마크가 동일 Plan으로 계산됨
- `TestRun_SPXBenchmark_AppliesUsdKrwFx` — SPX 벤치마크에 fx 적용(이중·누락 없음)
- `TestRun_ClampIncludesBenchmarkFirstAvailable` — 벤치마크 firstAvailable도 공통 윈도우에 반영

### 12-3. 핸들러 (`handlers/backtest_test.go`)

- `TestRunBacktest_Happy_Shape` — 200 + 응답 키 존재
- `TestRunBacktest_EmptyBasket_422` / `TooMany_422` / `BadWeight_422`
- `TestRunBacktest_IndexInBasket_422_AssetNotSupported`
- `TestRunBacktest_NoAuth_401`

### 12-4. 통합 (`backtest_integration_test.go`)

- `TestBacktest_E2E_SeededPrices` — 실 Supabase + 시드 종목/가격(`seedTestInstrument` 패턴 확장: 가격 시계열도 시드) → POST → 평가액·지표 sanity
- (RLS 격리 테스트 불필요 — 사용자 데이터 미접근)

### 12-5. 프런트 (`apps/web/components/backtest/*.test.tsx`)

- 폼 입력 → 실행 → 결과 렌더
- 커버리지 경고 배너 표시
- 422(검증) 인라인 에러
- 비중 자동 정규화 표시

---

## 13. 비범위 (YAGNI)

- ❌ **런 저장·재조회·비교** — 무상태. 테이블 0. v2
- ❌ **채팅 `run_backtest` 도구** — v2 (요약 + 페이지 딥링크)
- ❌ **AI 매매 일기 통합** — 백테스트는 매매가 아닌 분석. 저장도 안 함
- ❌ **커스텀 날짜 범위** — 트레일링 프리셋(1Y·3Y·5Y·전체)만. 임의 구간은 v2
- ❌ **주/분기 적립** — 월 적립만
- ❌ **임계값 기반 리밸런싱** — 달력 기반(분기·반기·연)만
- ❌ **샤프·소르티노·회전율** — v2
- ❌ **세금·수수료·슬리피지·배당 재투자** — 이상화 모드만 (라이브와 일관)
- ❌ **여러 전략 동시 오버레이** — 1런=1전략. 비교는 재실행
- ❌ **공매도·레버리지·비중 음수** — 영구 불가 (정체성 §2)
- ❌ **파라미터 최적화/스윕·전략 자동 추천** — 투자권유 영역, 정체성 §2 위반
- ❌ **US 30종목 초과** — 백필 확장(Phase 2)에 의존

---

## 검토 이력

### 2026-05-29 초안 작성 + 자체 검토 (Opus)

작성 직후 MANDATORY 자체 검토 1사이클. **Critical 1 / Important 6 / Minor 7** 발견 → 전부 인라인 패치.

**Critical (명세대로면 결과가 틀림)**
- C1. §5-3 `t0` 블록에 `TotalContributed = Initial`과 `Cashflows = [(-Initial, t0)]` 누락. → 총수익률은 초기자금을 순이익으로 오인하고, XIRR은 최초 유출이 없어 CAGR이 전부 틀림(§8 예시 `total_contributed`=28M이 이미 이를 전제). t0 블록에 두 줄 추가.

**Important (실행은 되나 잘못된 값·누락 위험)**
- I1. `Initial=0`이면 NAV(t0)=0/0=NaN으로 엔진 붕괴 → t0 전제 `Initial>0`(§9 강제) 명시.
- I2. 적립·리밸런싱 발생일 판정 알고리즘(커서 방식) + "첫 월적립은 t0+1개월, t0 당일 제외, 3년=36회" 명시.
- I3. 같은 날 적립+리밸런싱 시 리밸런싱 `V(t)`는 적립 반영 후(직전 V+C) 재계산임을 명시(하루치 V 캐시 재사용 시 적립현금 미배분 버그).
- I4. `clampedStart`를 전략 leg ∪ 벤치마크 leg(KOSPI·SPX) 공통 max로 명시(동일 윈도우·영업일 축 공유).
- I5. USD leg는 `clampedStart`에 fx 존재 필요 → fx_rates가 가격 윈도우 이상 커버하는 불변식 + 미충족 시 fx firstAvailable도 클램프 포함.
- I6. 벤치마크 leg의 단위·통화 계약 명시(SPX=지수레벨×USD/KRW, KOSPI=레벨×1.0; BenchmarkSeries 원시 레벨 가정, 이중 환산 금지).

**Minor (명확성·예시·검증 보강)**
- M1. `Cashflow` 타입 정의 추가. M2. `metrics()` 시그니처 + 초과수익은 Service.Run에서 TWR 차로 계산 명시. M3. XIRR `f'(r)` 도함수 추가. M4. `monthly_contribution < 0` 검증(422) 추가. M5. 트레이딩데이 축=KOSPI 거래일 공통 축으로 표현 정리(all-US 바스켓 포함). M6. 초과수익=누적·시간가중(머니가중 총수익률과 측정축 다름) UI 라벨 명기. M7. 테스트 4종 추가(t0 cashflow/contributed, 적립 횟수, SPX fx, 벤치마크 클램프).

자체 검토 후 상태: 구현 plan 작성 단계 진입 준비 완료.
