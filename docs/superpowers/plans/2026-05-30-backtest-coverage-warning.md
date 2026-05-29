# 백테스트 커버리지 경고 결함 fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 백테스트 시작일이 벤치마크(KOSPI·S&P 500) 또는 USD 환율 데이터 부족으로 조정될 때 `coverage_warnings`에 사유를 채워 프론트 `CoverageNotice`가 렌더되게 하고, 동시에 모든 백테스트에서 스푸리어스하게 떠 있던 레그 경고를 제거한다.

**Architecture:** `backtest.go`의 `Run` 메서드 단일 파일 수정. 경고 기준선을 `reqStartStr`(요청 시작 = today−5Y, 거의 항상 과거)에서 `naturalStart := allDays[0]`(윈도우의 실제 첫 영업일)으로 교정한다. 레그 경고 조건을 고치고, 레그 루프 뒤에 KOSPI·S&P 500·USD/KRW 경고를 추가한다. 클램프(clampStart) 계산 로직 자체는 이미 올바르므로 건드리지 않는다 — 결함은 오직 경고 방출(emission)에만 있다.

**Tech Stack:** Go 1.x, 기존 `fakeBTDeps` 테스트 하네스. 프론트엔드 변경 없음 (`CoverageNotice.tsx`는 `w.symbol`로 키링 → 신규 심볼 "KOSPI"/"S&P 500"/"USD/KRW"은 레그 심볼과 충돌 없음).

---

## 배경: 두 개의 결함

`backtest.go`의 `Run`은 두 가지 별개 일을 한다:

1. **클램프 계산** (line 616–660): `clampStart`를 레그 종가 first, USD 레그 fx first, KOSPI bench first, SPX bench first의 최댓값으로 잡는다. **이 로직은 이미 올바르다.**
2. **경고 방출** (line 674–694): `coverage_warnings` 배열을 채운다. **여기에 두 결함이 있다.**

### 결함 A — 스푸리어스 레그 경고 (모든 백테스트에서 발생)

line 687 조건이 `ld.first > reqStartStr`다. `reqStartStr`은 `today.AddDate(-5,0,0)`(5년 전)이다. 어떤 종목이든 `firstAvailable(closes)`는 윈도우의 실제 첫 영업일 `allDays[0]` 이상이고, 이는 거의 항상 5년 전보다 나중이다. 따라서 **모든 레그가 매 백테스트마다 경고를 방출**한다 → 프론트 `CoverageNotice`가 상시 렌더.

### 결함 B — 벤치마크/fx 클램프 시 경고 누락

`clampStart`가 KOSPI·SPX·USD-fx의 firstAvailable에 의해 밀릴 때, 경고 루프는 레그만 순회하므로 `coverage_warnings`에 해당 사유가 없다. 클램프는 일어났는데(시작일이 조정됨) 프론트는 이유를 표시하지 못한다.

### 교정

기준선을 `naturalStart := allDays[0]`로 통일한다. 어떤 소스든 `first > naturalStart`일 때만 "이 소스가 윈도우를 단축시켰다"는 의미이므로 경고한다. 이 한 가지 변경이 결함 A(레그 조건 교정)와 결함 B(벤치/fx 경고 추가)를 모두 해결한다.

---

## File Structure

- **Modify:** `apps/api/internal/portfolio/backtest.go` — `Run` 메서드의 경고 방출 로직만. 5개 지점 (naturalStart 선언, hasUSDLeg 선언·세팅, 레그 조건 교정, 벤치/fx 경고 추가).
- **Test:** `apps/api/internal/portfolio/backtest_test.go` — 신규 테스트 3개 추가. 기존 테스트는 수정하지 않는다(green 유지 확인만).

테스트 명령은 모두 `apps/api` 디렉터리에서 실행:
```bash
cd /Users/yuhojin/Desktop/finance/apps/api
```

---

## Task 1: 스푸리어스 레그 경고 제거 (naturalStart 기준선 도입)

**Files:**
- Modify: `apps/api/internal/portfolio/backtest.go:672` (naturalStart 선언), `:687` (레그 조건)
- Test: `apps/api/internal/portfolio/backtest_test.go` (신규 `TestRun_FullCoverage_NoSpuriousWarning`)

- [ ] **Step 1: 실패하는 회귀 테스트 작성**

`backtest_test.go`의 `TestRun_ClampIncludesBenchmarkFirstAvailable` 함수 바로 뒤(현재 line 489 닫는 `}` 다음)에 추가:

```go
func TestRun_FullCoverage_NoSpuriousWarning(t *testing.T) {
	days := genDays(t, "2024-01-01", 40)
	deps := krStockDeps(days, []string{"id1", "id2"})
	// 모든 소스(레그·KOSPI·SPX)가 윈도우 전체를 커버 → 경고 0건이어야 한다.
	svc := newBacktestServiceWithDeps(deps, mustParse(t, "2024-02-15"))
	res, err := svc.Run(context.Background(), nil, BacktestRequest{
		Period: "all", InitialCash: 1_000_000, Rebalance: "none",
		Basket: []BasketItem{{"id1", 50}, {"id2", 50}},
	})
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	if len(res.CoverageWarnings) != 0 {
		t.Errorf("full coverage인데 스푸리어스 경고 발생: %+v", res.CoverageWarnings)
	}
}
```

- [ ] **Step 2: 테스트 실패 확인**

Run: `go test ./internal/portfolio/ -run TestRun_FullCoverage_NoSpuriousWarning -v`
Expected: FAIL — 현재 코드는 `ld.first > reqStartStr`(reqStartStr="2019-02-15") 조건으로 id1·id2 모두 경고를 방출 → `len(CoverageWarnings)==2`, want 0.

- [ ] **Step 3: naturalStart 선언 추가**

`backtest.go` line 672 직전(현재 `stratLegs := make([]Leg, len(rows))` 줄 바로 위)에 한 줄 추가. 변경 전:

```go
	stratLegs := make([]Leg, len(rows))
	normalized := make([]NormalizedLeg, len(rows))
	var warnings []CoverageWarning
```

변경 후:

```go
	naturalStart := allDays[0] // 커버리지 경고 기준선 = 윈도우의 실제 첫 영업일(allDays는 line 605에서 비어있지 않음 보장)
	stratLegs := make([]Leg, len(rows))
	normalized := make([]NormalizedLeg, len(rows))
	var warnings []CoverageWarning
```

- [ ] **Step 4: 레그 경고 조건 교정**

`backtest.go` line 687. 변경 전:

```go
		if ld.first > reqStartStr {
```

변경 후:

```go
		if ld.first > naturalStart {
```

- [ ] **Step 5: 신규 테스트 통과 + 전체 회귀 확인**

Run: `go test ./internal/portfolio/ -v`
Expected: PASS — `TestRun_FullCoverage_NoSpuriousWarning` 통과. `TestRun_ClampsStartToCommonWindow`(id2 first="2024-01-11" > naturalStart="2024-01-01" → 경고 유지, assertion 통과), `TestRun_ClampIncludesBenchmarkFirstAvailable`(경고 미검증 → 영향 없음) 등 기존 테스트 전부 green.

- [ ] **Step 6: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance
git add apps/api/internal/portfolio/backtest.go apps/api/internal/portfolio/backtest_test.go
git commit -m "fix(api): 백테스트 스푸리어스 레그 커버리지 경고 제거 (기준선 reqStartStr→naturalStart)"
```

---

## Task 2: 벤치마크(KOSPI·S&P 500) 커버리지 경고 추가

**Files:**
- Modify: `apps/api/internal/portfolio/backtest.go:694` 직후 (레그 경고 루프 종료 지점)
- Test: `apps/api/internal/portfolio/backtest_test.go` (신규 `TestRun_BenchmarkClamp_EmitsWarning`)

- [ ] **Step 1: 실패하는 테스트 작성**

`backtest_test.go`의 `TestRun_FullCoverage_NoSpuriousWarning`(Task 1에서 추가) 뒤에 추가:

```go
func TestRun_BenchmarkClamp_EmitsWarning(t *testing.T) {
	days := genDays(t, "2024-01-01", 50)
	deps := krStockDeps(days, []string{"id1"})
	// KOSPI 벤치마크가 2024-01-16(days[15])부터만 존재 → 클램프 지배 + 경고 방출 기대.
	deps.bench["KOSPI"] = benchPts(days[15:], 2000)
	svc := newBacktestServiceWithDeps(deps, mustParse(t, "2024-03-01"))
	res, err := svc.Run(context.Background(), nil, BacktestRequest{
		Period: "all", InitialCash: 1_000_000, Rebalance: "none",
		Basket: []BasketItem{{"id1", 100}},
	})
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	found := false
	for _, w := range res.CoverageWarnings {
		if w.Symbol == "KOSPI" && w.FirstAvailable == "2024-01-16" {
			found = true
		}
	}
	if !found {
		t.Errorf("KOSPI 커버리지 경고 누락: %+v", res.CoverageWarnings)
	}
}
```

- [ ] **Step 2: 테스트 실패 확인**

Run: `go test ./internal/portfolio/ -run TestRun_BenchmarkClamp_EmitsWarning -v`
Expected: FAIL — Task 1 적용 후에도 벤치마크 경고 방출 코드가 없어 `CoverageWarnings`에 KOSPI 항목 없음(레그 id1 first="2024-01-01"=naturalStart → 레그 경고도 없음). `found==false`.

- [ ] **Step 3: 벤치마크 경고 추가**

`backtest.go` line 694(레그 경고 `for` 루프의 닫는 `}`) 직후, line 696 `kospiCloses := ...` 직전에 삽입. 변경 전:

```go
	}

	kospiCloses := restrictForwardFilled(benchCloseMap(kospiPts), allDays, clampedDays)
```

변경 후:

```go
	}

	if kf := benchFirst(kospiPts); kf > naturalStart {
		warnings = append(warnings, CoverageWarning{
			Symbol:         "KOSPI",
			FirstAvailable: kf,
			Message:        "비교 지수(KOSPI) 데이터가 " + kf + "부터 존재해 시작일을 조정했습니다",
		})
	}
	if sf := benchFirst(spxPts); sf > naturalStart {
		warnings = append(warnings, CoverageWarning{
			Symbol:         "S&P 500",
			FirstAvailable: sf,
			Message:        "비교 지수(S&P 500) 데이터가 " + sf + "부터 존재해 시작일을 조정했습니다",
		})
	}

	kospiCloses := restrictForwardFilled(benchCloseMap(kospiPts), allDays, clampedDays)
```

> `benchFirst`은 빈 시리즈에서 ""을 반환하고 `"" > naturalStart`은 false이므로 별도 빈 검사 불필요.

- [ ] **Step 4: 테스트 통과 + 회귀 확인**

Run: `go test ./internal/portfolio/ -v`
Expected: PASS — `TestRun_BenchmarkClamp_EmitsWarning` 통과. `TestRun_FullCoverage_NoSpuriousWarning`도 여전히 green(KOSPI·SPX benchFirst="2024-01-01"=naturalStart → 경고 없음).

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance
git add apps/api/internal/portfolio/backtest.go apps/api/internal/portfolio/backtest_test.go
git commit -m "feat(api): 백테스트 벤치마크(KOSPI·S&P 500) 커버리지 경고 추가"
```

---

## Task 3: USD/KRW 환율 커버리지 경고 추가

**Files:**
- Modify: `apps/api/internal/portfolio/backtest.go:616` (hasUSDLeg 선언), `:635` (세팅), 벤치 경고 블록 직후 (fx 경고)
- Test: `apps/api/internal/portfolio/backtest_test.go` (신규 `TestRun_UsdFxClamp_EmitsWarning`)

- [ ] **Step 1: 실패하는 테스트 작성**

`backtest_test.go`의 `TestRun_BenchmarkClamp_EmitsWarning`(Task 2에서 추가) 뒤에 추가:

```go
func TestRun_UsdFxClamp_EmitsWarning(t *testing.T) {
	days := genDays(t, "2024-01-01", 50)
	deps := krStockDeps(days, []string{"id1"})
	// id1을 USD 종목으로 전환.
	deps.metas["id1"] = InstrumentMeta{Symbol: "00A", Name: "종목A", Currency: "USD", AssetClass: "US_STOCK"}
	// USD 환율이 2024-01-16(days[15])부터만 존재 → fx firstAvailable이 클램프 지배 + 경고 기대.
	usdFx := map[string]float64{}
	for _, d := range days[15:] {
		usdFx[d] = 1300
	}
	deps.fx["USD"] = usdFx
	svc := newBacktestServiceWithDeps(deps, mustParse(t, "2024-03-01"))
	res, err := svc.Run(context.Background(), nil, BacktestRequest{
		Period: "all", InitialCash: 1_000_000, Rebalance: "none",
		Basket: []BasketItem{{"id1", 100}},
	})
	if err != nil {
		t.Fatalf("Run err: %v", err)
	}
	found := false
	for _, w := range res.CoverageWarnings {
		if w.Symbol == "USD/KRW" && w.FirstAvailable == "2024-01-16" {
			found = true
		}
	}
	if !found {
		t.Errorf("USD/KRW 커버리지 경고 누락: %+v", res.CoverageWarnings)
	}
}
```

- [ ] **Step 2: 테스트 실패 확인**

Run: `go test ./internal/portfolio/ -run TestRun_UsdFxClamp_EmitsWarning -v`
Expected: FAIL — USD fx 경고 방출 코드 없음. (레그 id1 closes first="2024-01-01"=naturalStart → 레그 경고 없음; KOSPI·SPX full → 벤치 경고 없음.) `found==false`.

- [ ] **Step 3: hasUSDLeg 선언 추가**

`backtest.go` line 616~617. 변경 전:

```go
	clampStart := reqStartStr // 문자열(YYYY-MM-DD) 사전순 비교 = 시간순 max
	rows := make([]legData, 0, len(req.Basket))
```

변경 후:

```go
	clampStart := reqStartStr // 문자열(YYYY-MM-DD) 사전순 비교 = 시간순 max
	hasUSDLeg := false        // USD 환율 경고 게이트: 비KRW 레그가 있을 때만 fx가 클램프에 기여
	rows := make([]legData, 0, len(req.Basket))
```

- [ ] **Step 4: hasUSDLeg 세팅 추가**

`backtest.go` line 635~639의 비KRW 분기. 변경 전:

```go
		if m.Currency != "KRW" {
			if ff := firstAvailable(fx); ff != "" && ff > clampStart {
				clampStart = ff // USD leg fx firstAvailable도 포함 (§5-5)
			}
		}
```

변경 후:

```go
		if m.Currency != "KRW" {
			hasUSDLeg = true
			if ff := firstAvailable(fx); ff != "" && ff > clampStart {
				clampStart = ff // USD leg fx firstAvailable도 포함 (§5-5)
			}
		}
```

- [ ] **Step 5: USD/KRW 경고 추가**

Task 2에서 삽입한 S&P 500 경고 블록 닫는 `}` 직후, `kospiCloses := ...` 직전에 삽입. 변경 전(Task 2 적용 상태):

```go
	if sf := benchFirst(spxPts); sf > naturalStart {
		warnings = append(warnings, CoverageWarning{
			Symbol:         "S&P 500",
			FirstAvailable: sf,
			Message:        "비교 지수(S&P 500) 데이터가 " + sf + "부터 존재해 시작일을 조정했습니다",
		})
	}

	kospiCloses := restrictForwardFilled(benchCloseMap(kospiPts), allDays, clampedDays)
```

변경 후:

```go
	if sf := benchFirst(spxPts); sf > naturalStart {
		warnings = append(warnings, CoverageWarning{
			Symbol:         "S&P 500",
			FirstAvailable: sf,
			Message:        "비교 지수(S&P 500) 데이터가 " + sf + "부터 존재해 시작일을 조정했습니다",
		})
	}
	if hasUSDLeg {
		if ff := firstAvailable(usdFx); ff != "" && ff > naturalStart {
			warnings = append(warnings, CoverageWarning{
				Symbol:         "USD/KRW",
				FirstAvailable: ff,
				Message:        "환율(USD/KRW) 데이터가 " + ff + "부터 존재해 시작일을 조정했습니다",
			})
		}
	}

	kospiCloses := restrictForwardFilled(benchCloseMap(kospiPts), allDays, clampedDays)
```

> `usdFx`는 line 651에서 이미 조회됨(SPX 벤치마크용). 비KRW 레그의 통화는 KR/US 유니버스에서 항상 USD이며 같은 "USD" 시리즈를 공유하므로 `usdFx` 단일 경고로 충분(중복 없음).

- [ ] **Step 6: 테스트 통과 + 전체 회귀 확인**

Run: `go test ./internal/portfolio/ -v`
Expected: PASS — `TestRun_UsdFxClamp_EmitsWarning` 통과 + 기존 전체 green. 특히 `TestRun_SPXBenchmark_AppliesUsdKrwFx`·`TestRun_BenchmarksUseSameCashflow`(KRW 레그만 → hasUSDLeg=false → USD 경고 없음) 영향 없음.

- [ ] **Step 7: 패키지 빌드·vet 확인**

Run: `go build ./... && go vet ./internal/portfolio/`
Expected: 출력 없음(성공).

- [ ] **Step 8: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance
git add apps/api/internal/portfolio/backtest.go apps/api/internal/portfolio/backtest_test.go
git commit -m "feat(api): 백테스트 USD/KRW 환율 커버리지 경고 추가"
```

---

## Task 4: 문서 갱신 (STATUS / ROADMAP)

**Files:**
- Modify: `docs/STATUS.md` (알려진 결함 항목 → 해결 처리 + 최근 변경 이력 + 마지막 업데이트)
- Modify: `docs/ROADMAP.md` (해당 결함 항목 제거 또는 완료 처리)

- [ ] **Step 1: STATUS.md 갱신**

`docs/STATUS.md`의 "알려진 결함" 섹션에서 백테스트 커버리지 경고 누락 항목(현재 line 125 근방)을 찾아 ✅ 해결로 표기하거나 제거한다. "최근 변경 이력" 맨 위에 한 줄 추가:

```markdown
- 백테스트 커버리지 경고 결함 fix — 벤치마크(KOSPI·S&P 500)·USD/KRW 클램프 시 경고 방출 + 스푸리어스 레그 경고 제거 (기준선 naturalStart)
```

"마지막 업데이트"를 `2026-05-30`으로 갱신한다.

- [ ] **Step 2: ROADMAP.md 갱신**

`docs/ROADMAP.md`에서 백테스트 커버리지 경고 결함 항목을 제거하고, "현재 추천 다음 작업"이 이 항목을 가리키고 있었다면 다음 항목(미들웨어 N+1 최적화)으로 재설정한다.

- [ ] **Step 3: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance
git add docs/STATUS.md docs/ROADMAP.md
git commit -m "docs: 백테스트 커버리지 경고 결함 fix 반영 (STATUS·ROADMAP)"
```

---

## Self-Review (작성자용 체크)

**1. 스펙 커버리지:** 결함 A(스푸리어스 레그) → Task 1. 결함 B(벤치/fx 경고 누락) → Task 2(벤치)·Task 3(fx). 문서 의무 → Task 4. ✅

**2. Placeholder 스캔:** 모든 스텝에 실제 코드·정확한 라인·실행 명령·기대 출력 포함. TBD/TODO 없음. ✅

**3. 타입 일관성:**
- `CoverageWarning{Symbol, FirstAvailable, Message string}` (backtest.go:455) — 모든 신규 경고가 동일 필드 사용. ✅
- `benchFirst([]PricePoint) string` (backtest.go:508) — `kospiPts`·`spxPts`는 `[]PricePoint`. ✅
- `firstAvailable(map[string]float64) string` (alpha.go) — `usdFx`는 `map[string]float64` (line 651 `FxRatesOnDates`). ✅
- `allDays[0]`은 line 605의 빈 검사로 보장. `naturalStart` string. ✅
- 테스트 헬퍼 `genDays`·`krStockDeps`·`benchPts`·`mustParse`·`newBacktestServiceWithDeps` 시그니처 모두 기존 사용처와 일치. ✅

**4. red→green 검증:**
- Task 1 테스트: 현재 `ld.first > reqStartStr`로 id1·id2 둘 다 경고 → len==2 → FAIL. 수정 후 naturalStart 기준 → len==0 → PASS. ✅
- Task 2 테스트: Task 1만으론 벤치 경고 코드 없음 → FAIL. Task 2 후 KOSPI 경고 → PASS. ✅
- Task 3 테스트: Task 2까지론 fx 경고 코드 없음 → FAIL. Task 3 후 USD/KRW 경고 → PASS. ✅
- 윈도우 길이: Task 2·3 모두 50일, days[15:]=35일 ≥ minBacktestDays(30) → InsufficientDataError 미발생. ✅

**5. 프론트엔드 영향 없음 확인:** `CoverageNotice.tsx`는 `warnings.map(key={w.symbol})`. 신규 심볼 "KOSPI"/"S&P 500"/"USD/KRW"은 레그 심볼("00A" 등)과 충돌 없음 → 변경 불필요. ✅
