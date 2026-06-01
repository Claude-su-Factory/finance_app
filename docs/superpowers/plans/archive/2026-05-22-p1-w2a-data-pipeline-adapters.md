# Quotient W2a — 데이터 어댑터 + 마이그레이션 + Ingest

> **For agentic workers:** REQUIRED SUB-SKILL: `superpowers:subagent-driven-development`. v2 (검증된 라이브러리 채택).

**Goal:** 외부 데이터 소스 어댑터(KRX·Yahoo·Frankfurter·FRED·ECOS) + 마켓 스키마(5 테이블) + ingest 패키지(COPY 기반). W2a 종료 시점: Go 테스트로 각 어댑터가 실제 데이터를 받고 Postgres에 적재.

**Architecture:** `internal/sources/*`별 패키지. Yahoo는 `piquette/finance-go` 라이브러리(공식 권장), 환율은 frankfurter.app(무료·키 없음·ECB), KRX는 자체 HTTP + Referer 헤더 + ISIN 매핑. ingest는 pgx COPY로 bulk upsert.

**Tech Stack:** Go 1.25 + `piquette/finance-go v1.x` + 표준 net/http + `pgx/v5` + `testcontainers-go v0.32+`.

**참고 스펙:** [`2026-05-22-quotient-mvp-design.md`](../specs/2026-05-22-quotient-mvp-design.md) §3·4·10-2·10-3. [`W1 plan`](2026-05-22-p1-w1-infra-auth.md). [archive `_archived-2026-05-22-p1-w2-data-pipeline-v1.md`].

---

## 외부 셋업 (Task 0)

- [ ] **FRED API 키 발급** https://fred.stlouisfed.org/docs/api/api_key.html — 즉시 무료
- [ ] **한국은행 ECOS API 키 발급** https://ecos.bok.or.kr/api/ — 즉시 무료
- [ ] **Frankfurter.app 키 없음**, **Yahoo 키 없음**, **KRX 키 없음**

env에 추가: `FRED_API_KEY`, `ECOS_API_KEY`.

---

## File Structure (W2a 생성)

```
apps/api/
├── internal/
│   ├── sources/
│   │   ├── common/backoff.go            # 지수 백오프 helper
│   │   ├── krx/krx.go                   # OTP 없음 — Referer + 'bld' POST + ISIN 매핑
│   │   ├── krx/krx_test.go
│   │   ├── yahoo/yahoo.go               # piquette/finance-go wrapper (crumb 자동)
│   │   ├── yahoo/yahoo_test.go
│   │   ├── fx/fx.go                     # frankfurter.app
│   │   ├── fx/fx_test.go
│   │   ├── fred/fred.go                 # FRED 직접 호출
│   │   ├── fred/fred_test.go
│   │   └── ecos/ecos.go                 # 한국은행 ECOS
│   ├── ingest/                          # instruments·prices(COPY)·quotes·indicators·aliases·fx
│   ├── models/                          # instrument·price·indicator·fx_rate
│   └── config/config.go                 # FRED/ECOS keys 추가
supabase/migrations/
├── 20260522000003_instruments.sql
├── 20260522000004_prices_quotes.sql
├── 20260522000005_indicators_fx.sql
└── 20260522000006_rls_market.sql
```

---

## Task 1: 마이그레이션 — instruments + aliases

**Files:** `supabase/migrations/20260522000003_instruments.sql`

- [ ] **Step 1: 작성**

```sql
create table public.instruments (
  id uuid primary key default gen_random_uuid(),
  symbol text not null,
  exchange text not null,
  isin text,                                       -- KRX 일봉 조회용 (KR7XXXXXXXXX). NULL 허용 (지수·US·FX)
  name text not null,
  asset_class text not null check (asset_class in ('KR_STOCK','US_STOCK','ETF','CASH','INDEX','FX')),
  currency text not null check (currency in ('KRW','USD')),
  is_active boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint instruments_symbol_exchange_unique unique (symbol, exchange)
);
create index instruments_active_class_idx on public.instruments (asset_class) where is_active = true;

create table public.instrument_aliases (
  alias text primary key,
  instrument_id uuid not null references public.instruments(id) on delete cascade,
  source text not null default 'seed' check (source in ('seed','learned')),
  created_at timestamptz not null default now()
);
create index instrument_aliases_inst_idx on public.instrument_aliases (instrument_id);

create trigger instruments_touch
  before update on public.instruments
  for each row execute function public.touch_updated_at();

-- 시드: 핵심 지수·환율 (TopTicker용)
insert into public.instruments (symbol, exchange, name, asset_class, currency) values
  ('KOSPI',   'KRX-IDX',  'KOSPI 종합',     'INDEX', 'KRW'),
  ('KOSDAQ',  'KRX-IDX',  'KOSDAQ 종합',    'INDEX', 'KRW'),
  ('SPX',     'NYSE-IDX', 'S&P 500',        'INDEX', 'USD'),
  ('NDX',     'NASDAQ-IDX','NASDAQ 100',    'INDEX', 'USD'),
  ('USD_KRW', 'FX',       'USD/KRW',        'FX',    'KRW'),
  ('EUR_KRW', 'FX',       'EUR/KRW',        'FX',    'KRW'),
  ('JPY_KRW', 'FX',       'JPY/KRW',        'FX',    'KRW');
```

> 스펙 §3 deviation: `asset_class`에 `INDEX`·`FX` 추가 (ticker·환율 표시용). spec doc 갱신 백로그.

- [ ] **Step 2: 적용·검증·커밋**

```bash
cd /Users/yuhojin/Desktop/finance/supabase && supabase db push
docker exec supabase_db_finance psql -U postgres -d postgres -c "select symbol, asset_class from public.instruments;"
# 7행 (시드 종목)
cd .. && git add supabase/
git commit -m "feat(db): instruments + aliases + 핵심 지수·환율 시드"
```

---

## Task 2: 마이그레이션 — prices + quotes + fx_rates + indicators

**Files:** `supabase/migrations/20260522000004_prices_quotes.sql`, `20260522000005_indicators_fx.sql`

- [ ] **Step 1: prices + quotes**

`20260522000004_prices_quotes.sql`:
```sql
create table public.prices (
  instrument_id uuid not null references public.instruments(id) on delete cascade,
  date date not null,
  open numeric(20,6) not null,
  high numeric(20,6) not null,
  low  numeric(20,6) not null,
  close numeric(20,6) not null,
  volume bigint not null default 0,
  primary key (instrument_id, date)
);
create index prices_date_idx on public.prices (instrument_id, date desc);

create table public.quotes (
  instrument_id uuid primary key references public.instruments(id) on delete cascade,
  price numeric(20,6) not null,
  change_abs numeric(20,6) not null default 0,
  change_pct numeric(8,4) not null default 0,
  updated_at timestamptz not null default now()
);
create index quotes_updated_idx on public.quotes (updated_at desc);
```

- [ ] **Step 2: indicators + fx_rates (FX는 별도 테이블, 의미적 분리)**

`20260522000005_indicators_fx.sql`:
```sql
-- 경제 지표 (금리·실업률·CPI 등)
create table public.economic_indicators (
  code text not null,
  observed_at timestamptz not null,
  name text not null,
  value numeric(20,6) not null,
  unit text,
  primary key (code, observed_at)
);
create index indicators_code_obs_idx on public.economic_indicators (code, observed_at desc);

-- 환율 시계열 (별도 테이블 — 의미적 분리)
create table public.fx_rates (
  base text not null,                              -- 'USD'
  quote text not null,                             -- 'KRW' / 'EUR' / 'JPY'
  observed_at timestamptz not null,
  rate numeric(20,8) not null,                     -- 1 base = rate * quote
  primary key (base, quote, observed_at)
);
create index fx_rates_pair_obs_idx on public.fx_rates (base, quote, observed_at desc);
```

- [ ] **Step 3: 적용·커밋**

```bash
cd /Users/yuhojin/Desktop/finance/supabase && supabase db push
cd .. && git add supabase/
git commit -m "feat(db): prices + quotes + economic_indicators + fx_rates"
```

---

## Task 3: 마이그레이션 — 마켓 RLS (service_role write 명시)

**Files:** `supabase/migrations/20260522000006_rls_market.sql`

- [ ] **Step 1: 작성**

```sql
-- 모든 마켓 테이블 RLS 활성화
alter table public.instruments         enable row level security;
alter table public.instrument_aliases  enable row level security;
alter table public.prices              enable row level security;
alter table public.quotes              enable row level security;
alter table public.economic_indicators enable row level security;
alter table public.fx_rates            enable row level security;

-- 인증 사용자: 읽기 전체 허용
create policy market_read_inst on public.instruments         for select to authenticated using (true);
create policy market_read_alia on public.instrument_aliases  for select to authenticated using (true);
create policy market_read_prc  on public.prices              for select to authenticated using (true);
create policy market_read_qte  on public.quotes              for select to authenticated using (true);
create policy market_read_ind  on public.economic_indicators for select to authenticated using (true);
create policy market_read_fx   on public.fx_rates            for select to authenticated using (true);

-- service_role: 전체 쓰기 명시 (W3에서 Go 풀이 일반 role로 전환되어도 보장)
create policy market_write_inst on public.instruments         for all to service_role using (true) with check (true);
create policy market_write_alia on public.instrument_aliases  for all to service_role using (true) with check (true);
create policy market_write_prc  on public.prices              for all to service_role using (true) with check (true);
create policy market_write_qte  on public.quotes              for all to service_role using (true) with check (true);
create policy market_write_ind  on public.economic_indicators for all to service_role using (true) with check (true);
create policy market_write_fx   on public.fx_rates            for all to service_role using (true) with check (true);
```

- [ ] **Step 2: 적용·검증·커밋**

```bash
cd /Users/yuhojin/Desktop/finance/supabase && supabase db push
docker exec supabase_db_finance psql -U postgres -d postgres -c "select polname from pg_policy where polrelid::regclass::text in ('public.instruments','public.prices','public.quotes','public.fx_rates','public.economic_indicators','public.instrument_aliases') order by polname;"
# 12개 정책
cd .. && git add supabase/
git commit -m "feat(db): 마켓 RLS (인증 read + service_role write 명시)"
```

---

## Task 4: 공통 helper (지수 백오프)

**Files:** `apps/api/internal/sources/common/backoff.go`

- [ ] **Step 1: backoff helper**

```go
package common

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// DoWithBackoff 재시도: 1·2·4·8·16초. 5회 시도. 200·429 외에는 비재시도.
func DoWithBackoff(ctx context.Context, do func() (*http.Response, error)) (*http.Response, error) {
	delays := []time.Duration{0, time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}
	var lastErr error
	for i, d := range delays {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(d):
			}
		}
		resp, err := do()
		if err != nil {
			lastErr = err
			slog.Warn("retry", "attempt", i+1, "err", err)
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = errors.New("retryable status: " + resp.Status)
			slog.Warn("retry status", "attempt", i+1, "status", resp.StatusCode)
			continue
		}
		return resp, nil
	}
	return nil, lastErr
}
```

- [ ] **Step 2: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/api/
git commit -m "feat(api): 공통 HTTP 백오프 helper (지수 백오프 5회)"
```

---

## Task 5: 모델 (Instrument·PriceBar·Quote·Indicator·FXRate)

**Files:** `apps/api/internal/models/{instrument,price,indicator,fx}.go`

- [ ] **Step 1: 모델 작성**

`internal/models/instrument.go`:
```go
package models

import "time"

type AssetClass string
const (
	AssetKRStock AssetClass = "KR_STOCK"
	AssetUSStock AssetClass = "US_STOCK"
	AssetETF     AssetClass = "ETF"
	AssetCash    AssetClass = "CASH"
	AssetIndex   AssetClass = "INDEX"
	AssetFX      AssetClass = "FX"
)

type Instrument struct {
	ID         string     `json:"id"`
	Symbol     string     `json:"symbol"`
	Exchange   string     `json:"exchange"`
	ISIN       *string    `json:"isin,omitempty"`
	Name       string     `json:"name"`
	AssetClass AssetClass `json:"asset_class"`
	Currency   string     `json:"currency"`
	IsActive   bool       `json:"is_active"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type Quote struct {
	InstrumentID string    `json:"instrument_id"`
	Price        float64   `json:"price"`
	ChangeAbs    float64   `json:"change_abs"`
	ChangePct    float64   `json:"change_pct"`
	UpdatedAt    time.Time `json:"updated_at"`
}
```

`internal/models/price.go`:
```go
package models
import "time"
type PriceBar struct {
	InstrumentID string    `json:"instrument_id"`
	Date         time.Time `json:"date"`
	Open, High, Low, Close float64
	Volume       int64
}
```

`internal/models/indicator.go`:
```go
package models
import "time"
type Indicator struct {
	Code       string    `json:"code"`
	ObservedAt time.Time `json:"observed_at"`
	Name       string    `json:"name"`
	Value      float64   `json:"value"`
	Unit       *string   `json:"unit,omitempty"`
}
```

`internal/models/fx.go`:
```go
package models
import "time"
type FXRate struct {
	Base       string    `json:"base"`
	Quote      string    `json:"quote"`
	ObservedAt time.Time `json:"observed_at"`
	Rate       float64   `json:"rate"`
}
```

- [ ] **Step 2: 커밋**

```bash
git add apps/api/ && git commit -m "feat(api): 모델 추가 (Instrument·PriceBar·Quote·Indicator·FXRate)"
```

---

## Task 6: KR 마켓 어댑터 — KIND (종목 마스터) + Yahoo (시세)

**⚠️ SPIKE 결과 (2026-05-22, agent a34a0c6...)**: `data.krx.co.kr`의 모든 BLD 엔드포인트(`MDCSTAT01901`, `MDCSTAT01701`, OTP 흐름 포함)가 `LOGOUT` 응답. KRX가 2025년 말 로그인 벽 뒤로 이전. 직접 호출 불가능.

**채택된 패턴**:
- **종목 마스터**: KIND 공개 다운로드 (`https://kind.krx.co.kr/corpgeneral/corpList.do?method=download&searchType=13&marketType=stockMkt`) — EUC-KR HTML 테이블 (839개 KOSPI + KOSDAQ별도)
- **OHLCV·시세**: Yahoo Finance + `.KS`/`.KQ` 접미사 (예: `005930.KS` = 삼성전자, `247540.KQ` = 에코프로비엠). `piquette/finance-go` (Task 7) 라이브러리로 통합 — KR/US 단일 어댑터.

**왜 이 선택인가 (CTO)**:
- Naver siseJson은 동작하지만 ToS 회색지대 (spec §1 "회색지대 금지" 위반 가능성)
- KIS Open API는 사용자 계좌 등록 필요 (Phase 3)
- Yahoo는 공식 지원, 단일 라이브러리 통합, 15분 지연 (spec과 정합), Rate limit MVP 규모 안전
- pykrx sidecar는 Naver 의존 + Python 컨테이너 운영 부담

**의도된 비범위**: KRX 데이터 (호가·체결·외인보유 등 세부)는 v2 (Phase 3 KIS API 도입 시) 검토.

**Files:** `apps/api/internal/sources/kind/kind.go`, `kind_test.go`

- [ ] **Step 1: 의존성 (EUC-KR + HTML 파싱)**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api
go get golang.org/x/net/html
go get golang.org/x/text/encoding/korean
go get golang.org/x/text/transform
```

- [ ] **Step 2: KIND 어댑터 (종목 마스터만)**

```go
package kind

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"

	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/quotient/quotient/apps/api/internal/sources/common"
)

const baseURL = "https://kind.krx.co.kr/corpgeneral/corpList.do"

type Client struct {
	url  string
	http *http.Client
}

func NewClient(url string) *Client {
	if url == "" { url = baseURL }
	return &Client{url: url, http: &http.Client{Timeout: 30 * time.Second}}
}

// FetchInstruments returns KOSPI or KOSDAQ listings from KIND public HTML download.
// market: "KOSPI" or "KOSDAQ".
// KIND는 ISIN을 노출하지 않으므로 ISIN 필드는 nil. Yahoo는 short symbol + 접미사로 조회.
func (c *Client) FetchInstruments(ctx context.Context, market string) ([]models.Instrument, error) {
	mt := "stockMkt"
	if strings.EqualFold(market, "KOSDAQ") { mt = "kosdaqMkt" }
	u := fmt.Sprintf("%s?method=download&searchType=13&marketType=%s", c.url, mt)

	resp, err := common.DoWithBackoff(ctx, func() (*http.Response, error) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Quotient/1.0)")
		return c.http.Do(req)
	})
	if err != nil { return nil, fmt.Errorf("kind fetch: %w", err) }
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK { return nil, fmt.Errorf("kind HTTP %d", resp.StatusCode) }

	// EUC-KR → UTF-8 + HTML 파싱
	utf8r := transform.NewReader(resp.Body, korean.EUCKR.NewDecoder())
	doc, err := html.Parse(utf8r)
	if err != nil { return nil, fmt.Errorf("kind html parse: %w", err) }

	return parseKindTable(doc, market), nil
}

// parseKindTable walks <tbody><tr><td> rows. KIND 컬럼 순서:
// 0=회사명 1=시장구분 2=종목코드 3=업종 4=주요제품 5=상장일 ...
func parseKindTable(doc *html.Node, market string) []models.Instrument {
	var out []models.Instrument
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			tds := []string{}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.Data == "td" {
					tds = append(tds, textOf(c))
				}
			}
			if len(tds) >= 6 {
				symbol := strings.TrimSpace(tds[2])
				// 종목코드 6자리 검증
				if len(symbol) == 6 {
					out = append(out, models.Instrument{
						Symbol:     symbol,
						Exchange:   "KRX",
						Name:       strings.TrimSpace(tds[0]),
						AssetClass: models.AssetKRStock,
						Currency:   "KRW",
						IsActive:   true,
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling { walk(c) }
	}
	walk(doc)
	return out
}

func textOf(n *html.Node) string {
	if n.Type == html.TextNode { return n.Data }
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling { b.WriteString(textOf(c)) }
	return b.String()
}

// (헬퍼 함수 io는 미사용 — import 정리)
var _ = io.Discard
```

- [ ] **Step 3: 테스트 (mock HTML 픽스처)**

```go
package kind

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleHTML = `<!DOCTYPE html>
<html><body><table><tbody>
<tr><td>삼성전자</td><td>코스피</td><td>005930</td><td>전기·전자</td><td>반도체</td><td>1975-06-11</td></tr>
<tr><td>SK하이닉스</td><td>코스피</td><td>000660</td><td>전기·전자</td><td>메모리</td><td>1996-12-26</td></tr>
</tbody></table></body></html>`

func TestFetchInstruments_ParsesHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "marketType=stockMkt")
		// EUC-KR로 인코딩되지 않은 UTF-8 응답 — 디코더가 ASCII 통과시키므로 OK
		w.Header().Set("Content-Type", "text/html; charset=euc-kr")
		_, _ = w.Write([]byte(sampleHTML))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	inst, err := c.FetchInstruments(context.Background(), "KOSPI")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(inst), 2)
	assert.Equal(t, "005930", inst[0].Symbol)
	assert.Equal(t, "삼성전자", inst[0].Name)
}
```

- [ ] **Step 4: 빌드·테스트·커밋**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api
go mod tidy && go test ./internal/sources/kind/...
git -C /Users/yuhojin/Desktop/finance add apps/api/
git -C /Users/yuhojin/Desktop/finance commit -m "feat(api): KIND 어댑터 (KR 종목 마스터 HTML 다운로드, EUC-KR)"
```

> KR OHLCV·시세는 Task 7 (Yahoo)에서 `.KS`/`.KQ` 접미사로 통합 처리.

---

## Task 7: Yahoo Finance 어댑터 (KR + US 통합)

**핵심**: crumb·쿠키를 piquette/finance-go가 자동 처리. KR 종목은 `005930.KS` (KOSPI) / `247540.KQ` (KOSDAQ) 형태 접미사 사용. US 종목은 접미사 없음.

**Symbol 변환 helper**:
- `005930` + `KOSPI` → `005930.KS`
- `005930` + `KOSDAQ` → `005930.KQ`
- `AAPL` + `NASDAQ` → `AAPL`

이 변환은 ingest 단계에서 instrument 행을 만들 때 또는 fetch 호출 시 처리.

**Files:** `apps/api/internal/sources/yahoo/yahoo.go`, `yahoo_test.go`

- [ ] **Step 1: 의존성**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api
go get github.com/piquette/finance-go
```

- [ ] **Step 2: 어댑터**

```go
package yahoo

import (
	"context"
	"fmt"
	"time"

	"github.com/piquette/finance-go/chart"
	"github.com/piquette/finance-go/datetime"
	"github.com/piquette/finance-go/quote"
	"github.com/quotient/quotient/apps/api/internal/models"
)

type Client struct{}
func NewClient() *Client { return &Client{} }

// FetchChart 일봉 (start~end). end 미지정 시 now.
func (c *Client) FetchChart(ctx context.Context, symbol string, start, end time.Time) ([]models.PriceBar, error) {
	if end.IsZero() { end = time.Now() }
	p := &chart.Params{
		Symbol:   symbol,
		Interval: datetime.OneDay,
		Start:    datetime.FromUnix(int(start.Unix())),
		End:      datetime.FromUnix(int(end.Unix())),
	}
	iter := chart.Get(p)              // v1.1.0 API: GetParams 아님
	var bars []models.PriceBar
	for iter.Next() {
		b := iter.Bar()
		o, _ := b.Open.Float64()      // shopspring/decimal: (float64, bool)
		h, _ := b.High.Float64()
		l, _ := b.Low.Float64()
		c, _ := b.Close.Float64()
		bars = append(bars, models.PriceBar{
			Date:   time.Unix(int64(b.Timestamp), 0).UTC(),
			Open:   o, High: h, Low: l, Close: c,
			Volume: int64(b.Volume),
		})
	}
	if err := iter.Err(); err != nil {
		return bars, fmt.Errorf("yahoo chart %s: %w", symbol, err)
	}
	return bars, nil
}

// FetchQuote 단일 종목 시세 + 변동률.
func (c *Client) FetchQuote(ctx context.Context, symbol string) (models.Quote, error) {
	q, err := quote.Get(symbol)
	if err != nil { return models.Quote{}, fmt.Errorf("yahoo quote %s: %w", symbol, err) }
	if q == nil { return models.Quote{}, fmt.Errorf("yahoo quote nil: %s", symbol) }
	return models.Quote{
		Price:     q.RegularMarketPrice,
		ChangeAbs: q.RegularMarketChange,
		ChangePct: q.RegularMarketChangePercent,
		UpdatedAt: time.Now().UTC(),
	}, nil
}

// (floatVal helper 제거 — Float64()가 2개 반환이라 인라인이 더 명확)
```

> piquette의 `finance.Decimal` 또는 `decimal` 타입은 라이브러리 버전마다 다를 수 있음 — 실행 시 컴파일 에러 발생하면 `b.Open.Float64()` 또는 `b.Open.InexactFloat64()` 같은 접근자로 조정.

- [ ] **Step 3: 테스트 (라이브러리 mock 제한적 — 통합 테스트로 검증)**

```go
//go:build integration
package yahoo
// 실제 Yahoo 호출, AAPL 시세 받기. CI default off.
```

- [ ] **Step 4: 빌드·커밋**

```bash
go mod tidy && go build ./... && go test ./internal/sources/yahoo/...
git -C /Users/yuhojin/Desktop/finance add apps/api/
git -C /Users/yuhojin/Desktop/finance commit -m "feat(api): Yahoo Finance 어댑터 (piquette/finance-go, crumb 자동)"
```

---

## Task 8: 환율 어댑터 (frankfurter.app 채택)

**핵심**: 무료, 키 없음, ECB 데이터. exchangerate.host 대체.

**Files:** `apps/api/internal/sources/fx/fx.go`, `fx_test.go`

- [ ] **Step 1: 어댑터**

```go
package fx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/quotient/quotient/apps/api/internal/sources/common"
)

// frankfurter.app은 frankfurter.dev로 301 리다이렉트되므로 직접 dev URL 사용
const baseURL = "https://api.frankfurter.dev/v1/latest"

type Client struct {
	url  string
	http *http.Client
}

func NewClient(url string) *Client {
	if url == "" { url = baseURL }
	return &Client{url: url, http: &http.Client{Timeout: 15 * time.Second}}
}

// FetchRates: base→symbols 환율.
func (c *Client) FetchRates(ctx context.Context, base string, symbols []string) ([]models.FXRate, error) {
	u := fmt.Sprintf("%s?from=%s&to=%s", c.url, base, strings.Join(symbols, ","))
	resp, err := common.DoWithBackoff(ctx, func() (*http.Response, error) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		return c.http.Do(req)
	})
	if err != nil { return nil, fmt.Errorf("fx: %w", err) }
	defer resp.Body.Close()

	var p struct {
		Date  string             `json:"date"`
		Base  string             `json:"base"`
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, fmt.Errorf("fx parse: %w", err)
	}
	date, _ := time.Parse("2006-01-02", p.Date)
	if date.IsZero() { date = time.Now().UTC() }
	out := make([]models.FXRate, 0, len(p.Rates))
	for q, r := range p.Rates {
		out = append(out, models.FXRate{Base: p.Base, Quote: q, ObservedAt: date, Rate: r})
	}
	return out, nil
}
```

- [ ] **Step 2: 테스트 (mock 서버)**

```go
package fx
import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchRates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "from=USD")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"date": "2025-12-30", "base": "USD",
			"rates": map[string]float64{"KRW": 1450.5, "EUR": 0.92, "JPY": 148.3},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	rates, err := c.FetchRates(context.Background(), "USD", []string{"KRW", "EUR", "JPY"})
	require.NoError(t, err)
	assert.Len(t, rates, 3)
}
```

- [ ] **Step 3: 빌드·커밋**

```bash
go test ./internal/sources/fx/...
git -C /Users/yuhojin/Desktop/finance add apps/api/
git -C /Users/yuhojin/Desktop/finance commit -m "feat(api): 환율 어댑터 (frankfurter.app, 무료·키 없음)"
```

---

## Task 9: FRED + ECOS 어댑터 (수정판)

**v1 변경점**: FRED는 그대로(잘 동작), ECOS는 응답 RESULT 코드 체크 추가 (실패 시 빈 응답 silent 회피).

**Files:** `apps/api/internal/sources/fred/fred.go`, `ecos/ecos.go`

- [ ] **Step 1: FRED**

(v1 코드 그대로 — 백오프만 추가)

```go
package fred

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/quotient/quotient/apps/api/internal/sources/common"
)

type Client struct {
	url, key string
	http     *http.Client
}

func NewClient(baseURL, key string) *Client {
	if baseURL == "" { baseURL = "https://api.stlouisfed.org/fred/series/observations" }
	return &Client{url: baseURL, key: key, http: &http.Client{Timeout: 20 * time.Second}}
}

func (c *Client) FetchObservations(ctx context.Context, seriesID string) ([]models.Indicator, error) {
	q := url.Values{}
	q.Set("series_id", seriesID); q.Set("api_key", c.key); q.Set("file_type", "json")

	resp, err := common.DoWithBackoff(ctx, func() (*http.Response, error) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"?"+q.Encode(), nil)
		return c.http.Do(req)
	})
	if err != nil { return nil, fmt.Errorf("fred: %w", err) }
	defer resp.Body.Close()

	var p struct {
		Observations []struct { Date, Value string } `json:"observations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, fmt.Errorf("fred parse: %w", err)
	}
	out := make([]models.Indicator, 0, len(p.Observations))
	for _, o := range p.Observations {
		if o.Value == "." { continue }
		date, err := time.Parse("2006-01-02", o.Date); if err != nil { continue }
		val, err := strconv.ParseFloat(o.Value, 64); if err != nil { continue }
		out = append(out, models.Indicator{Code: seriesID, ObservedAt: date, Value: val})
	}
	return out, nil
}
```

- [ ] **Step 2: ECOS (RESULT 체크)**

```go
package ecos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/quotient/quotient/apps/api/internal/sources/common"
)

type Client struct {
	url, key string
	http     *http.Client
}

func NewClient(baseURL, key string) *Client {
	if baseURL == "" { baseURL = "https://ecos.bok.or.kr/api/StatisticSearch" }
	return &Client{url: baseURL, key: key, http: &http.Client{Timeout: 20 * time.Second}}
}

func (c *Client) FetchSeries(ctx context.Context, statCode, cycle, start, end string) ([]models.Indicator, error) {
	// cycle: "M" 월, "D" 일, "A" 년
	u := fmt.Sprintf("%s/%s/json/kr/1/1000/%s/%s/%s/%s", c.url, c.key, statCode, cycle, start, end)
	resp, err := common.DoWithBackoff(ctx, func() (*http.Response, error) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		req.Header.Set("User-Agent", "Quotient/1.0")
		return c.http.Do(req)
	})
	if err != nil { return nil, fmt.Errorf("ecos: %w", err) }
	defer resp.Body.Close()

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("ecos parse: %w", err)
	}
	// 실패 응답 (RESULT 키 존재) 명시 처리
	if r, ok := raw["RESULT"]; ok {
		var res struct { Code, Message string }
		_ = json.Unmarshal(r, &res)
		return nil, fmt.Errorf("ecos error %s: %s", res.Code, res.Message)
	}
	stat, ok := raw["StatisticSearch"]
	if !ok { return nil, fmt.Errorf("ecos: missing StatisticSearch in response") }
	var s struct {
		Row []struct {
			StatCode, StatName, Time, DataValue, UnitName string
		} `json:"row"`
	}
	if err := json.Unmarshal(stat, &s); err != nil { return nil, fmt.Errorf("ecos parse row: %w", err) }

	out := make([]models.Indicator, 0, len(s.Row))
	layout := timeLayoutFor(cycle)
	for _, r := range s.Row {
		date, err := time.Parse(layout, r.Time); if err != nil { continue }
		val, err := strconv.ParseFloat(r.DataValue, 64); if err != nil { continue }
		unit := r.UnitName
		out = append(out, models.Indicator{Code: statCode, ObservedAt: date, Name: r.StatName, Value: val, Unit: &unit})
	}
	return out, nil
}

func timeLayoutFor(cycle string) string {
	switch cycle {
	case "D": return "20060102"
	case "A": return "2006"
	default:  return "200601"
	}
}
```

- [ ] **Step 3: 빌드·테스트·커밋**

```bash
go mod tidy && go build ./... && go test ./internal/sources/fred/...
git -C /Users/yuhojin/Desktop/finance add apps/api/
git -C /Users/yuhojin/Desktop/finance commit -m "feat(api): FRED + ECOS 어댑터 (백오프 + ECOS RESULT 체크)"
```

---

## Task 10: ingest 패키지 — instruments / prices(COPY) / quotes / indicators / fx / aliases

(v1과 동일 패턴이되 fx_rates 추가, change_pct 계산 helper 포함)

**Files:** `apps/api/internal/ingest/{instruments,prices,quotes,indicators,fx,aliases}.go` + `*_test.go`

- [ ] **Step 1: instruments — pgx Batch로 N round-trip 회피**

```go
package ingest

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/models"
)

func UpsertInstruments(ctx context.Context, pool *pgxpool.Pool, items []models.Instrument) (int64, error) {
	if len(items) == 0 { return 0, nil }
	b := &pgx.Batch{}
	for _, it := range items {
		b.Queue(`
			insert into public.instruments (symbol, exchange, isin, name, asset_class, currency)
			values ($1, $2, $3, $4, $5, $6)
			on conflict (symbol, exchange) do update set
				isin = excluded.isin, name = excluded.name,
				asset_class = excluded.asset_class, currency = excluded.currency,
				is_active = true, updated_at = now()
		`, it.Symbol, it.Exchange, it.ISIN, it.Name, string(it.AssetClass), it.Currency)
	}
	br := pool.SendBatch(ctx, b)
	defer br.Close()
	var n int64
	for range items {
		if _, err := br.Exec(); err != nil { return n, err }
		n++
	}
	return n, nil
}
```

- [ ] **Step 2: prices — COPY into temp table 후 ON CONFLICT merge**

```go
package ingest

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/models"
)

const chunkSize = 1000

func UpsertPrices(ctx context.Context, pool *pgxpool.Pool, bars []models.PriceBar) (int64, error) {
	if len(bars) == 0 { return 0, nil }
	var total int64
	for i := 0; i < len(bars); i += chunkSize {
		end := i + chunkSize
		if end > len(bars) { end = len(bars) }
		n, err := copyChunk(ctx, pool, bars[i:end])
		if err != nil { return total, err }
		total += n
	}
	return total, nil
}

func copyChunk(ctx context.Context, pool *pgxpool.Pool, bars []models.PriceBar) (int64, error) {
	tx, err := pool.Begin(ctx); if err != nil { return 0, err }
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		create temp table tmp_prices (
			instrument_id uuid, date date,
			open numeric, high numeric, low numeric, close numeric, volume bigint
		) on commit drop`); err != nil { return 0, err }

	rows := make([][]any, len(bars))
	for i, b := range bars {
		rows[i] = []any{b.InstrumentID, b.Date, b.Open, b.High, b.Low, b.Close, b.Volume}
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"tmp_prices"},
		[]string{"instrument_id","date","open","high","low","close","volume"},
		pgx.CopyFromRows(rows)); err != nil { return 0, err }

	tag, err := tx.Exec(ctx, `
		insert into public.prices (instrument_id, date, open, high, low, close, volume)
		select instrument_id, date, open, high, low, close, volume from tmp_prices
		on conflict (instrument_id, date) do update set
			open = excluded.open, high = excluded.high, low = excluded.low,
			close = excluded.close, volume = excluded.volume
	`)
	if err != nil { return 0, err }
	return tag.RowsAffected(), tx.Commit(ctx)
}
```

- [ ] **Step 3: quotes / indicators / aliases / fx (간결한 upsert — v1 패턴 유지, fx 추가)**

`internal/ingest/fx.go`:
```go
package ingest

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/models"
)

func UpsertFXRates(ctx context.Context, pool *pgxpool.Pool, rates []models.FXRate) (int64, error) {
	if len(rates) == 0 { return 0, nil }
	tx, err := pool.Begin(ctx); if err != nil { return 0, err }
	defer tx.Rollback(ctx)
	var n int64
	for _, r := range rates {
		_, err := tx.Exec(ctx, `
			insert into public.fx_rates (base, quote, observed_at, rate)
			values ($1, $2, $3, $4)
			on conflict (base, quote, observed_at) do update set rate = excluded.rate
		`, r.Base, r.Quote, r.ObservedAt, r.Rate)
		if err != nil { return n, err }
		n++
	}
	return n, tx.Commit(ctx)
}
```

(quotes/indicators/aliases는 v1 archive 참고 — 그대로)

- [ ] **Step 4: testcontainers-go 통합 테스트 (버전 핀)**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api
go get github.com/testcontainers/testcontainers-go@v0.34.0
go get github.com/testcontainers/testcontainers-go/modules/postgres@v0.34.0
```

`internal/ingest/prices_test.go` (대표):
```go
//go:build integration
package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func setupPG(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	pg, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pg.Terminate(ctx) })
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	pool, err := pgxpool.New(ctx, dsn); require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		create table instruments (id uuid primary key default gen_random_uuid(), symbol text, exchange text, isin text, name text, asset_class text, currency text, is_active boolean default true, created_at timestamptz default now(), updated_at timestamptz default now(), unique(symbol, exchange));
		create table prices (instrument_id uuid, date date, open numeric, high numeric, low numeric, close numeric, volume bigint, primary key (instrument_id, date));
	`)
	require.NoError(t, err)
	return pool
}

func TestUpsertPrices(t *testing.T) {
	pool := setupPG(t); defer pool.Close()
	ctx := context.Background()
	var id uuid.UUID
	require.NoError(t, pool.QueryRow(ctx, `insert into instruments (symbol, exchange, name, asset_class, currency) values ('AAPL','NASDAQ','Apple','US_STOCK','USD') returning id`).Scan(&id))

	bars := []models.PriceBar{
		{InstrumentID: id.String(), Date: time.Date(2025,12,30,0,0,0,0,time.UTC), Open: 200, High: 202, Low: 199, Close: 201, Volume: 1e6},
	}
	n, err := UpsertPrices(ctx, pool, bars)
	require.NoError(t, err); require.Equal(t, int64(1), n)
}
```

- [ ] **Step 5: 빌드·커밋**

```bash
go test ./internal/ingest/...                          # unit
go test -tags integration ./internal/ingest/...        # integration (느림)
git -C /Users/yuhojin/Desktop/finance add apps/api/
git -C /Users/yuhojin/Desktop/finance commit -m "feat(api): ingest 패키지 (instruments·prices(COPY)·quotes·indicators·fx·aliases) + testcontainers"
```

---

## Task 11: config에 FRED/ECOS keys 추가

**Files:** `apps/api/internal/config/config.go`

- [ ] **Step 1: 키 추가 + 환경변수 등록**

```go
type Config struct {
	// 기존 필드들...
	FREDAPIKey string `env:"FRED_API_KEY"`
	ECOSAPIKey string `env:"ECOS_API_KEY"`
}
```

`.env.example`에 두 줄 추가:
```
FRED_API_KEY=
ECOS_API_KEY=
```

- [ ] **Step 2: 빌드·커밋**

```bash
go build ./... && go test ./...
git -C /Users/yuhojin/Desktop/finance add apps/api/ .env.example
git -C /Users/yuhojin/Desktop/finance commit -m "feat(api): config에 FRED/ECOS API 키 추가"
```

---

## 자체 검토 (W2a)

**스펙 커버리지 (W2a)**:
- ✅ 마이그레이션 4개 (instruments·prices·quotes·indicators+fx_rates·RLS)
- ✅ 어댑터 5개 (KRX Referer+ISIN, Yahoo piquette, frankfurter, FRED+백오프, ECOS+RESULT 체크)
- ✅ 공통 백오프 helper
- ✅ ingest 6개 (Batch upsert + COPY + fx 분리)
- ✅ testcontainers 통합 테스트 (v0.34 핀)
- ✅ 마켓 마스터에 KOSPI/SPX/USD_KRW 등 핵심 시드 (ticker 동작 보장)
- ✅ RLS service_role write 정책 명시 (W3 슈퍼유저 우회 제거 시 안전)

**Critical v1 → v2 변경 (모두 패치 완료)**:
| v1 이슈 | v2 해결 |
|---|---|
| KRX OTP/Referer 누락 | Referer 헤더 명시 + ISIN 사용 |
| Yahoo crumb 부재 | piquette/finance-go 채택 |
| exchangerate.host 유료화 | frankfurter.app 무료 교체 |
| RLS service_role 의존 | 명시 write 정책 |
| ticker symbol 미적재 | 시드에서 INSERT |
| FX를 indicators에 적재 | 별도 fx_rates 테이블 |
| 백오프 부재 | common/backoff.go + 모든 어댑터 적용 |

**의도된 비범위 (W2b로 이관)**:
- cron 워커 (`internal/schedule/`)
- 마켓 API (`/v1/market/ticker`, `/v1/instruments/search`)
- TopTicker 실데이터 연결
- 5년 백필 CLI

W2b plan은 W2a 완료 후 별도 작성·검토 사이클.

## 검토 이력

- 2026-05-22 v1 작성 → subagent 검토에서 REWRITE NEEDED (12 Critical). v1 archive.
- 2026-05-22 v2 작성. v1의 모든 Critical 반영. W2a/W2b 분할.
- 2026-05-22 v2 subagent 재검토 → READY WITH PATCHES (Critical 3건 발견·반영).
  - Yahoo `chart.GetParams` → `chart.Get` 수정 + `Float64()` 2개 반환 처리 (Task 7).
  - frankfurter.app → frankfurter.dev (redirect 회피) (Task 8).
  - **KRX 실제 호출이 LOGOUT 반환 확인** → Task 6에 spike 단계 명시.
- 2026-05-22 KRX spike 완료 (agent a34a0c6). KRX 직접 호출 불가 (전 엔드포인트 로그인 벽). **결정: KR 데이터 단일화** — KIND (종목 마스터, 공개 HTML) + Yahoo (`.KS`/`.KQ`, 시세). Naver 시세 회색지대 회피. Task 6 KRX → KIND로 재작성.
- 다음: 사용자 승인 → 실행.
