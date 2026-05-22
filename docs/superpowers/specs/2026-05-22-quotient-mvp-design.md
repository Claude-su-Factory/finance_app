---
프로젝트: Quotient
스펙 종류: MVP 디자인
작성일: 2026-05-22
최종 갱신: 2026-05-22
상태: 9개 섹션 결정 완료. 자체 검토 진행 중.
---

# Quotient MVP — 디자인 스펙

## 섹션 1. 정체성·카피 ✅

- **이름**: Quotient
- **한 줄 카피**: `Portfolio Intelligence Terminal.`
- **서브 카피**: "한국·미국 자산을 한 화면에. 자연어로 묻고, 즉시 분석을 받으세요."
- **금기**: 메인·서브 카피에 "AI" 단어 노출 금지. 대체어 = 분석가·인텔리전스·터미널·엔진.
- **규제 카피**: 푸터에 "투자 자문이 아닙니다. 모든 의사결정은 본인 책임입니다." 명시.

### 차별점

1. 블룸버그 미감 + 분석가 두뇌 (대중 핀테크와 시각·기능 모두 분리)
2. 한국·미국·환율·지표 통합 한 화면 (국내 서비스 글로벌 시각 약점 보강)
3. 자연어 질의 인터페이스 (TradingView·Yahoo 부재)
4. 개인 가격대 (Bloomberg 월 $2000 → Quotient 월 1만 원대)
5. 데이터 소유권 (export 자유, 마이데이터 의존 없음)

## 섹션 2. 정보 구조 ⚠️ 잠정 합의 (섹션 5 이후 재확인)

```
공개:  / · /pricing · /login
인증:  /app · /app/portfolio · /app/chat · /app/market · /app/settings
```

앱 셸: 상단 실시간 티커, 좌측 아이콘 사이드바(5탭), 하단 상태바(시계·API 상태·⌘K 힌트).

탭별 핵심:

- **홈**: 총 자산 카드 + 자산 분포 도넛 + 오늘의 브리핑 + 보유 종목 상위 5
- **포트폴리오**: 보유 자산 테이블 + 미니 차트
- **AI 채팅**: 세션 리스트 + 메시지 영역 + 추천 질문 칩
- **마켓**: 지수·환율·지표·관심종목
- **설정**: 계정·구독·데이터 export·화면 강도

전역: ⌘K 명령 팔레트, 단축키(1~5 탭 이동·`/` 검색·`c` 채팅).

## 섹션 3. 데이터 모델 ✅

Supabase Postgres + RLS. 총 12개 테이블.

### 사용자

- `profiles` (id PK = `auth.users.id`, display_name, base_currency, ui_intensity)
- `subscriptions` (user_id, plan, status, current_period_end, toss_billing_key)
- `payment_events` (id, user_id, raw_payload jsonb, processed_at) — 멱등·감사

### 마켓 마스터 (공개)

- `instruments` (id, symbol, exchange, name, asset_class, currency, is_active)
- `instrument_aliases` (instrument_id, alias) — 한글·영문·티커 매핑
- `prices` (instrument_id, date, OHLC, volume) — 일봉, PK (instrument_id, date)
- `quotes` (instrument_id PK, price, change_abs, change_pct, updated_at) — 직전 시세 캐시
- `economic_indicators` (code, name, value, observed_at)

`asset_class` 도메인: `KR_STOCK | US_STOCK | ETF | CASH` (MVP 한정).

### 사용자 데이터

- `holdings` (user_id, instrument_id, quantity, avg_cost, opened_at, note) UNIQUE (user_id, instrument_id). 현금은 `asset_class=CASH`로 통합.
- `watchlist` (user_id, instrument_id, added_at) PK (user_id, instrument_id)
- `ai_briefings` (user_id, date, content_md, model, created_at) PK (user_id, date) — 일일 브리핑 캐시

### 채팅

- `chat_sessions` (id, user_id, title, created_at, updated_at)
- `chat_messages` (id, session_id, role, content, tool_calls jsonb, input_tokens, output_tokens, model, created_at)

### RLS

| 그룹 | 정책 |
|---|---|
| `profiles`, `holdings`, `watchlist`, `chat_*`, `subscriptions`, `ai_briefings` | `user_id = auth.uid()` |
| `instruments`, `instrument_aliases`, `prices`, `quotes`, `economic_indicators` | 인증 사용자 읽기, service_role 쓰기 |

### 통화·시간 정책

- 저장: 금액 = `instrument.currency` 원본. 시간 = UTC.
- 표시: 사용자 `base_currency` (기본 KRW) 환산. 시간 = KST.

### 인덱스 (핵심)

```
holdings(user_id)
watchlist(user_id, instrument_id)
chat_messages(session_id, created_at)
prices(instrument_id, date DESC)
instruments(symbol, exchange)
instrument_aliases(alias)
```

### 외부 도구 위임 (테이블 미생성)

- PostHog — 활동 추적
- Sentry — 에러 추적
- Resend — 이메일 발송

### 미결 / 의도된 비범위

- `transactions`, `cash_transactions` — Phase 2 (CSV 업로드 시 도입)
- 세금 계산 — v2 이후 별도 프로젝트로 분리

## 섹션 4. 데이터 수집 파이프라인 ✅

### 데이터 소스

| 데이터 | 소스 | 인증 | 비용 |
|---|---|---|---|
| KR 종목 마스터·일봉 | KRX 정보데이터시스템 공식 다운로드 | 무 | 무료 |
| KR 시세 (장중) | KRX 공개 데이터 **15분 지연** | 무 | 무료 |
| US 종목·시세 | Yahoo Finance (`piquette/finance-go`) **15분 지연** | 무 | 무료 |
| 환율 | `exchangerate.host` | 무 | 무료 |
| 경제 지표 | FRED API + 한국은행 ECOS API | 키 무료 등록 | 무료 |
| KIS Open API | Phase 3까지 보류 | — | — |

회색지대 스크래핑(네이버 금융 등) 금지. UI에 "시세 지연 15분" 명시.

### 갱신 주기

| 작업 | 빈도 / 시각 (KST) |
|---|---|
| 종목 마스터 (KR·US) | 일 1회, 06:00 |
| KR 일봉 | 일 1회, 16:30 (장 마감 후) |
| US 일봉 | 일 1회, 06:00 (US 장 마감 후 한국시각) |
| 실시간 시세 `quotes` | 1분, 장중만, **사용자 보유·관심 종목 한정** |
| 환율 | 5분, 24/7 |
| 경제 지표 | 일 1회, 07:00 |
| 일일 AI 브리핑 | 일 1회, 07:00, 사용자별 |

장중 정의:
- KR: 평일 09:00–15:30 KST (공휴일 제외)
- US: 평일 23:30–06:00 KST (서머타임 보정)

### 프로세스 구조 (MVP)

**단일 Go 프로세스**: API 서버 + 워커 goroutine 동거. `robfig/cron` 사용.

```
finance/
├── apps/
│   ├── api/        # Go API + 워커 cron (단일 바이너리)
│   └── web/        # Next.js
└── internal/
    ├── sources/    # krx · yahoo · fred · ecos · exrate
    ├── ingest/     # 정규화·검증·DB 적재
    └── schedule/   # cron 정의·실행
```

Phase 2에서 워커 분리 검토 (사용자 증가 시 또는 워커 부하 분리 필요 시).

### 백필 전략

- 초기 배포: 과거 5년치 일봉 (KR·US 주요 종목 + 사용자 보유 종목)
- 사용자 새 종목 추가 시 → 즉시 5년 백필 잡 트리거
- 미적재 종목 첫 조회 시 lazy 백필

### 실패 처리

| 실패 종류 | 처리 |
|---|---|
| 외부 API 일회성 실패 | 지수 백오프 5회 (1·2·4·8·16초) |
| 데이터 누락 | DB 마지막 값 유지 + Sentry 경고 |
| 가격 ±50% 이상 변동 | 의심 데이터 마킹 + 알림 + 통과 (자동 차단 X) |
| 3회 연속 잡 실패 | 해당 잡 일시 정지 + Resend 이메일 알림 |

### 비용 가드레일 (CFO)

- Phase 1 데이터 인프라 비용 = $0 (모두 무료 티어)
- 외부 호출 수 메트릭 = Fly metrics + Sentry
- 종량 비용 API(Alpha Vantage 등) 도입 시 한도 설정 필수

### 의도된 비범위

- 실시간(Tick) 시세 — Phase 3 KIS Open API 도입 시 사용자 본인 키
- 옵션·선물·파생 데이터 — 향후 검토
- 코인 — v2 이후

## 섹션 5. AI 채팅 흐름 ✅

> **2026-05-22 갱신**: 수익화 경로 1 결정에 따라 Free/Pro 차등 제거. MVP는 모두 무료. 사업자 등록 후 Phase 2에 Pro·결제 활성화 예정.

### 모델 라우팅

| 사용처 | 모델 | 사유 |
|---|---|---|
| 기본 채팅·일일 브리핑 | `claude-sonnet-4-6` | 가성비 최적 |
| "심층 분석" 요청 (Pro 전용) | `claude-opus-4-7` | 명시적 선택 |
| 시스템 내부 (요약·라우팅) | `claude-haiku-4-5` | 비용 최소 |

**Prompt caching 의무**: 시스템 프롬프트 + 도구 스키마 + 사용자 포트폴리오 컨텍스트에 cache control 적용. 캐시 hit rate 메트릭 노출.

### Tool Use (MVP 도구 9개)

```
get_portfolio()                          # 사용자 보유 자산 전체
get_holding_detail(instrument_id)        # 특정 보유 상세 + 수익률
get_quote(symbol | instrument_id)        # 현재 시세 (15분 지연 표기)
get_price_history(symbol, range)         # 5y | 1y | 6mo | 1mo | 1w
get_market_overview()                    # 지수·환율·지표 스냅샷
search_instrument(query)                 # 한글·영문·티커 별칭 매핑
get_watchlist()                          # 관심 종목
get_economic_indicator(code)             # 특정 지표 값·추이
calc_portfolio_metrics()                 # 변동성·샤프·집중도·통화 분산
```

`get_news`는 v2 이후. 백엔드가 `auth.uid()`로 RLS 강제.

### 컨텍스트 관리

1. 최근 20 메시지 원본 유지
2. 그 이전 메시지는 Haiku로 요약본 1개로 대체
3. 매 호출 = 시스템 프롬프트 + 도구 + 사용자 컨텍스트 요약 + 최근 메시지 + 신규 질문

세션당 누적 토큰 한도 도달 시 새 세션 권유.

### 사용량 정책 (MVP — 모두 무료)

| 항목 | 한도 |
|---|---|
| 채팅 횟수 | 월 30회 |
| 토큰 (입력/출력) | 50K / 10K |
| 심층 분석 (Opus) | 월 1회 (체험) |
| 일일 비용 cap (사용자별) | $0.30/day |
| 시스템 전체 cap | $10/day |

Phase 2 (사업자 등록 시) Pro ₩14,900 도입 검토. 그때까지 결제·구독 UI는 모두 비활성.

### 일일 자동 브리핑

- 매일 07:00 KST, 사용자별 실행
- 입력: 보유 자산 + 어제 시세 변화 + 주요 지수·환율 변화 + 관심 종목 변화
- 출력: 3~5문장 마크다운 → `ai_briefings` 저장
- 홈 탭 카드에 표시
- 설정에서 끄기 가능

### 스트리밍 + UX

- 백엔드 SSE. Next.js fetch reader로 수신
- 청크 단위 타이핑 효과 (Motion)
- 도구 호출 중 "데이터 조회 중…" 인디케이터

### 규제·안전 가드레일

시스템 프롬프트 명시:
- 직접 매수/매도 권유 금지. "분석 관점에서 ~를 살펴볼 수 있습니다" 형태
- 모든 수치는 도구 호출 결과 근거. 추측 금지
- 응답 끝에 `(데이터 기준: YYYY-MM-DD HH:MM KST, 시세 지연 15분)` 자동 부착
- 세금·법무 자문 요청 → "전문가에게" 권유

### 에러 처리

| 실패 | 처리 |
|---|---|
| Claude API 실패 | 친절한 에러 + 재시도 버튼 |
| 도구 호출 실패 | "데이터 조회 문제" + Sentry 보고 |
| 스트리밍 중단 | 부분 응답 저장 + 다음 접속 시 이어쓰기 옵션 |

### 의도된 비범위

- 음성·이미지 멀티모달 — v2 이후
- 사용자 자체 프롬프트 커스터마이징 — v2 이후
- 종목 자동 매수/매도 신호 — **영구 금지** (규제)

## 섹션 6. 정보 구조 락인 ✅

### 경로 맵

| 경로 | 페이지 | 인증 | 핵심 |
|---|---|---|---|
| `/` | 랜딩 | × | Hero · 기능 카드 · 가격 · CTA |
| `/pricing` | 가격 | × | Free vs Pro 비교 |
| `/login` | 로그인 | × | 이메일+비밀번호 · Google OAuth |
| `/signup` | 가입 | × | 이메일+비밀번호 |
| `/forgot-password` | 비밀번호 재설정 | × | 이메일 토큰 |
| `/privacy`, `/terms` | 법적 | × | 정적 |
| `/app` | 홈 | ✓ | 총자산·도넛·브리핑·상위5·마켓위젯·관심종목 |
| `/app/portfolio` | 포트폴리오 | ✓ | 보유테이블·CRUD·미니차트 |
| `/app/chat` | 채팅 | ✓ | 세션·메시지·도구·심층토글·사용량 |
| `/app/chat/[sessionId]` | 채팅 세션 | ✓ | 세션 상세 |
| `/app/market` | 마켓 | ✓ | 지수·환율·지표·워치리스트 |
| `/app/settings` | 설정 | ✓ | 계정·구독·데이터·알림·화면·사용량·AI |

### 탭 핵심 컴포넌트

**홈**: 그리드 카드. (1) 총 자산 카운트업 (2) 자산 분포 도넛 (asset_class·통화·종목 토글) (3) 오늘 브리핑 (4) 보유 상위 5 (5) 마켓 위젯 (KOSPI/S&P/USD-KRW) (6) 관심 종목 미니.

**포트폴리오**: 보유 자산 테이블 (종목·수량·평단가·현재가·평가액·수익률·비중·통화), 정렬·필터·검색, CRUD 모달, CSV 업로드 자리(Phase 2 배지), 선택 종목 우측 sliding panel.

**채팅**: 좌측 세션 리스트 + 중앙 메시지 + 하단 입력+칩. 우상단 사용량 (`Free: 7/10`), "심층 분석(Opus)" 토글 (Pro만, Free는 자물쇠+업그레이드 유도).

**마켓**: KR 지수 · US 지수 · 환율 · 경제 지표 · 관심 종목. 모든 카드는 라인 차트 + 현재값.

**설정**: 좌측 탭 메뉴 — 계정·구독·데이터·알림·화면·사용량·AI(Pro 전용). 화면 색 강도 (형광/표준/은은), 다크/라이트 (기본 다크).

### 가입 온보딩 (Hard 결정)

1. `/signup` → 이메일·비밀번호
2. Supabase Auth 이메일 인증
3. 첫 로그인 시 wizard (3단계, skip 가능):
   - 기본 통화 (KRW/USD)
   - 첫 보유 자산 1~3개 추가
   - 또는 "데모 데이터로 둘러보기"
4. `/app` 진입

### 전역 컴포넌트

- 상단 실시간 티커 + "지연 15분" 배지
- 하단 상태바 (시계 KST · API 점 · `⌘K` 힌트 · 시세 지연 배지)
- 명령 팔레트 `⌘K`: 종목 검색·즉시 추가, 탭 이동, "AI에게 묻기", 설정 진입
- 단축키: `1~5` 탭, `/` 검색, `c` 채팅, `g h` 홈 (vim-like)
- 시세 지연 배지 전역 적용 (모든 가격 표시 페이지 헤더)

### Long-running 작업 진행 표시 정책

| 작업 | UI |
|---|---|
| 페이지 전환·라우팅 | 상단 1px 가로 라인 (nprogress 스타일) |
| 데이터 로딩 | Skeleton (shadcn) |
| AI 응답 스트리밍 | 깜빡이는 캐럿 + 토큰 카운터 |
| AI 도구 체인 | 단계 리스트 (`▸ tool_name()  DONE  0.12s`) |
| 종목 백필 | 결정적 progress bar + `INGESTING X — n/N` |
| CSV 업로드 (Phase 2) | 결정적 progress bar (행 단위) |
| 데이터 export | 결정적 (테이블별 단계) |
| 결제·계정 삭제 | 무결정 + 단계 표시 모달 |
| 온보딩 위저드 | 상단 step (1/3 · 2/3 · 3/3) |
| 백그라운드 결과 | Toast (`sonner`) 우하단, 5초 자동 (실패 10초) |

#### 디자인 가이드

- 두께 1~2px, 두꺼운 바·둥근 끝 금지
- 색: 진행 형광 시안 `#00FFFF` / 노랑 `#FFD500`. 완료 형광 초록 `#00FF7F`. 실패 형광 빨강 `#FF3344`
- 타이포: 진행률·작업명은 monospace (JetBrains Mono). `1247/1825 (68.3%)` 절대+상대 동시 표기
- 모션: 부드러운 transition만, 출렁임·confetti 금지

### 의도된 비범위

- 다국어 (영어) — Phase 2 검토
- 모바일 네이티브 — 반응형만, 앱 없음
- 인앱 알림 센터 — Phase 2
- 화려한 축하 애니메이션 — 영구 금지 (톤 충돌)

## 섹션 7. 인증·결제·구독 흐름 ✅

> **2026-05-22 갱신**: 수익화 경로 1 결정에 따라 결제·구독 부분은 MVP에서 **비활성** (스키마·env flag만 유지). 사업자 등록 시 활성화. 인증·온보딩은 그대로 유효.

### 인증

Supabase Auth 위임.

| 방법 | MVP |
|---|---|
| 이메일 + 비밀번호 | ✅ |
| Google OAuth | ✅ |
| Apple / GitHub / Kakao | v2 이후 |

- 이메일 인증 의무 (미인증은 `/app` 차단)
- 비밀번호 최소 8자 (Supabase 기본)
- 세션: 액세스 1h + 리프레시 7일 (SDK 자동 갱신)
- 가입 시 trigger로 `profiles` row 자동 생성

### 결제 — Toss Payments + 빌링키

**최초 구독**:
1. `/pricing` → Pro 선택 → 로그인 필요 시 redirect
2. `/app/settings/subscription` 결제 모달
3. Toss 결제 위젯 → 카드 등록 → 빌링키 발급
4. webhook → `subscriptions.toss_billing_key` 저장
5. 즉시 첫 결제 → `status = active`, `current_period_end = +30일`

**정기 결제 cron (매일 02:00 KST)**:
- 만료 임박 구독 조회 → 빌링키 자동 결제
- 성공: `current_period_end += 30일`, 영수증 이메일
- 실패: `past_due`, 3일 후 재시도 (최대 3회) → 모두 실패 시 `canceled` → 만료 시 Free

**해지·환불**:
- 해지 → `status = canceled`, `current_period_end` 유지 (기간 끝까지 Pro)
- **환불**: 결제 후 7일 이내 전액 환불 (Toss 취소 API). 이후 무환불, 다음 월부터 미과금

### 구독 상태 머신

| 상태 | 의미 | 권한 |
|---|---|---|
| `free` | 무료 | Free 한도 |
| `active` | 정상 구독 | Pro |
| `past_due` | 결제 실패 재시도 (최대 3일) | Pro 유지 |
| `canceled` | 해지 (기간 미만료) | Pro 유지 (만료까지) |
| `expired` | 만료 후 | Free |

`trial` 상태는 MVP 제외.

### Webhook 보안

- 엔드포인트: `POST /webhooks/toss` (Go API)
- Toss 시크릿 HMAC 서명 검증 + IP 화이트리스트
- **멱등성**: `payment_events.id` dedup (이중 결제 방지 Critical)
- 처리 실패 시 5xx → Toss 자동 재시도

### 가격 정책

| 항목 | 정책 |
|---|---|
| Pro 월 | ₩14,900 (**부가세 포함**) |
| 공급가액 / VAT | ₩13,546 / ₩1,354 |
| 영수증 | 결제 성공 시 자동 이메일 (Resend) |
| 세금계산서 | 사업자 요청 시 수동 발행 — Phase 2 자동화 |
| 연간 플랜·쿠폰·학생할인 | v2 이후 |

### 데이터 보호

- 카드 정보는 Toss 보관, Quotient는 빌링키만
- 결제수단 변경 = 빌링키 재발급, 기존 폐기
- `payment_events` **7년 보관** (한국 세법)

### 결제 코드 비활성 전략 (MVP)

CTO 결정: 결제 코드는 작성하되 feature flag로 비활성. 사업자 등록 후 환경변수만 바꿔 활성화.

```
PAYMENTS_ENABLED=false   # MVP 기본
PAYMENTS_ENABLED=true    # 사업자 등록 후
```

**MVP 포함**:
- `subscriptions`, `payment_events` 테이블 (스키마만, 빈 상태)
- 결제 UI는 라우트 자체 차단 (`/app/settings/subscription` "곧 도입")

**MVP 제외 (Phase 2로 이관)**:
- Toss 위젯 통합
- Webhook 핸들러
- 정기 결제 cron
- 영수증 이메일

→ 사업자 등록 후 2~3일 작업으로 활성화 가능.

### 의도된 비범위

- 연간 플랜, 쿠폰, 팀 플랜, 가상계좌·계좌이체, 세금계산서 자동화

## 섹션 8. 에러·관측·보안 ✅

### 에러 처리 (3 레이어)

**클라이언트 (Next.js)**
- React Error Boundary → 친절 메시지 + "다시 시도"
- 자동 Sentry 보고
- 404·500 전용 페이지 (블룸버그 풍)

**서버 (Go API)**
- 표준 에러 응답: `{ "error": { "code", "message", "details? } }` JSON
- HTTP 상태 표준 (400/401/403/404/409/422/429/500/503)
- 모든 에러 Sentry + 구조화 로그 (slog JSON)
- 민감 정보 자동 마스킹

**워커 (cron)**
- 누적 3회 실패 → Resend 이메일 + Sentry

### 관측 (Observability)

| 도구 | 용도 | 무료 한도 |
|---|---|---|
| Sentry | 에러 추적 (3 레이어) | 5K event/월 |
| PostHog | 활동·funnel·페이지뷰 | 1M event/월 |
| Fly metrics | CPU·메모리·요청 | 기본 |
| Supabase logs | DB 쿼리 | 기본 |
| Resend | 운영자 알림 이메일 | 3K/월 |

**커스텀 메트릭** (Go):
- Claude API 호출 수·토큰·예상 비용 (user별)
- 외부 소스 호출 성공률
- 캐시 hit rate
- 사용자별 API 호출 빈도

**구조화 로그**: `slog` JSON → Fly logs. 요청 ID 부여. 민감 정보 마스킹.

**운영자 알림 채널**: 결제 실패, 워커 누적 실패, 비용 cap 도달, Sentry critical.

### 보안

**인증·인가**
- 모든 API 인증 필수 (예외: webhook, `/healthz`)
- JWT 검증 = Go 백엔드 + RLS 양쪽 (이중 방어)
- 쿠키 `HttpOnly`, `Secure`, `SameSite=Strict`

**인풋 검증**
- Go: `go-playground/validator` 모든 핸들러
- Next.js: `zod` 폼 검증
- 화이트리스트 원칙

**Secrets**
- Fly·Vercel·Supabase env 분리
- `.env.example` 만 커밋, `.env` gitignore
- 회전: Claude API 6개월, Supabase service_role 12개월
- 노출 시 즉시 회전

**Rate Limiting**
- 사용자별 채팅 분당 30회 (Postgres count 또는 Redis)
- 글로벌: Cloudflare 또는 Fly 기본
- Webhook: Toss IP 화이트리스트

**CORS**
- 프로덕션: 결정된 도메인만, 와일드카드 금지

**SQL Injection / XSS**
- SQL: `sqlc` + 파라미터 바인딩 강제
- XSS: React escape + CSP 헤더
- 마크다운: `react-markdown` + `rehype-sanitize` (**Critical** — Claude 응답이 사용자 화면에 직접 렌더링)

**백업**
- Supabase 일일 자동 백업 (Free 7일)
- 사용자 데이터 export (자기 데이터 다운로드 권리)

**규제 준수 (한국 PIPA)**
- 가입 시 약관·개인정보 처리방침 분리 동의 (단일 동의 금지)
- 처리 위탁 고지: Supabase·Toss·Anthropic·Resend·PostHog·Sentry
- 14세 미만 차단
- 보유 기간: 탈퇴 시 즉시 파기 (`payment_events`는 7년 — 세법)
- GDPR·CCPA: MVP는 한국만, 글로벌 진출 시 별도

### 헬스체크·SLO

- `GET /healthz`: DB + 외부 의존 ping
- `GET /readyz`: 트래픽 수용 준비
- **MVP SLO**: 가용성 99%, p95 < 500ms (Claude 제외), Claude p95 < 5s

### 비용 가드레일 (CFO)

| 항목 | 무료 한도 | 예상 |
|---|---|---|
| Sentry | 5K event/월 | < 1K |
| PostHog | 1M event/월 | < 100K |
| Resend | 3K email/월 | < 500 |
| Fly | 1머신 hobby | $0~5 |
| Supabase | 500MB DB · 2GB 대역 | < 100MB |
| Vercel | 100GB 대역 | < 10GB |

MVP 관측·보안 비용 = $0. Claude API 종량만.

### 의도된 비범위

- 2FA — Phase 2
- 강력 DDoS (Cloudflare WAF) — 트래픽 임계 도달 시
- SOC2 / ISO27001 — 사업 확장 시
- 침투 테스트 — 출시 후 3개월

## 섹션 9. 테스트·배포·MVP 일정 ✅

### 테스트 전략

**Go 백엔드 (TDD)**
- Unit: 도메인 로직 (계산·검증·정책) 80%+
- Integration: `testcontainers-go` Postgres
- Handler: HTTP 엔드포인트 60%+
- 외부 의존성 mock (Claude·Toss·KRX·Yahoo)

**Next.js 프론트엔드**
- 컴포넌트: Vitest + Testing Library, 핵심 70%+
- E2E: Playwright, 5개 핵심 시나리오
- 시각 회귀: MVP 비범위

**E2E 핵심 시나리오 (5개)**
1. 신규 가입 → 이메일 인증 → 온보딩 → 첫 자산 추가 → 홈
2. 로그인 → 채팅 → 도구 호출 응답 수신
3. 포트폴리오 CRUD
4. 일일 브리핑 생성 확인
5. 데이터 export

### 환경

| 환경 | Supabase | Fly | Vercel |
|---|---|---|---|
| Local | docker | `air` watch | `npm run dev` |
| Staging | 별도 프로젝트 | 별도 앱 | preview |
| Production | 별도 프로젝트 | 메인 앱 | main |

### CI/CD (GitHub Actions)

- **PR**: lint + test + build
- **develop**: staging 자동
- **main**: production 자동 (DB 마이그레이션은 수동 승인 step)
- main 직접 push 금지

**롤백**: Fly instant rollback, Vercel deployment promote. DB는 down 마이그레이션 의무 작성.

### MVP 일정 — **6주** (수익화 경로 1로 단축)

| 주 | 마일스톤 | Tier 1 |
|---|---|---|
| W1 | 인프라 + Auth + 온보딩 | Supabase 스키마/RLS, Fly·Vercel 셋업, Next.js 스캐폴딩, 가입/로그인/이메일 인증/온보딩 wizard |
| W2 | 데이터 워커 + 시세 | KRX·Yahoo·exchangerate·FRED·ECOS 수집, `instruments/prices/quotes/economic_indicators` 적재, 5년 백필 |
| W3 | 포트폴리오 + 홈 | `holdings` CRUD, 홈 대시보드 (총자산·도넛·상위5·마켓위젯), 시세 표시 + 지연 배지 |
| W4 | AI 채팅 (도구 + 컨텍스트) | Claude tool use 9개 도구, SSE 스트리밍, 컨텍스트 요약 |
| W5 | 마켓 탭 + 설정 + 광고 슬롯 + 일일 브리핑 | 마켓 시각화, 설정 페이지, `<AdSlot>` 추상화 (비활성 상태), 07:00 브리핑 cron |
| W6 | E2E + 명령 팔레트 + 베타 + 출시 | E2E 5개, ⌘K, 베타 5~10명, 출시 |

### 광고 슬롯 설계

| 슬롯 ID | 위치 | 형태 |
|---|---|---|
| `footer-sponsor` | 푸터 | 텍스트 1줄 "Sponsored by ~" |
| `sidebar-bottom` | 좌측 사이드바 최하단 | 미니멀 카드 |
| `market-sidebar` | 마켓 탭 우측 (옵션) | 300×250 다크 톤 |

**원칙**: 화려한 디스플레이 광고 금지. 텍스트·미니멀만. 다크·monospace 통일.

**구현**: `<AdSlot id fallback>`. `ENABLE_ADS=true` env → AdSense 렌더. false (MVP 기본) → 자체 메시지 (친구 초대·베타 피드백 등). PostHog로 클릭률 추적.

**AdSense 가입 조건**: 가입자 100명 + 일평균 PV 500 도달 시.

### 출시 전 체크리스트

**기능**
- [ ] 모든 Tier 1 완료
- [ ] E2E 5개 통과
- [ ] 베타 3명이 가입→자산→채팅→일일 브리핑 수신 완주

**규제 (CEO·Security)**
- [ ] 약관·개인정보 처리방침 작성·게시
- [ ] 도메인 등록·SSL
- [ ] 14세 미만 차단 동의 항목
- [ ] 사업자 등록 — **MVP 출시에는 불필요** (무료 운영). 결제 활성화 시 필요

**관측**
- [ ] Sentry·PostHog 연동 확인
- [ ] 운영자 알림 이메일 동작
- [ ] `/healthz`·`/readyz` 동작

**보안**
- [ ] Secrets 회전 절차 문서화
- [ ] 데이터 export 동작
- [ ] 계정 삭제 동작
- [ ] CSP 헤더 설정
- [ ] HTTPS 강제

**비용**
- [ ] 비용 가드레일 동작 (사용자별·시스템 cap)
- [ ] 무료 티어 모니터링 알림

### Phase 1 → Phase 2 전환 조건

다음 중 하나 충족 시:
- 가입자 100명 이상
- 일평균 PV 500 이상
- MVP 출시 후 3개월 경과

Phase 2 작업: CSV 업로드 + `transactions` 도입 + 자동 알림 + AdSense 가입 + (수익화 의사 결정 시) 사업자 등록 + Pro 결제 활성화.

### 의도된 비범위 (MVP)

- 모바일 네이티브, 다국어, 시각 회귀, 마케팅 자동화

## 10. 보강 명세 (자체 검토 반영, 2026-05-22)

자체 검토 결과 식별된 Critical·Important 이슈 보강. 충돌 시 본 섹션이 우선한다.

### 10-1. AI Tool Use 컨텍스트 전달 [Critical, 섹션 5 보강]

Claude 도구 호출 시 백엔드가 사용자 JWT를 보관·전파:

1. `POST /chat`이 JWT 검증 후 핸들러 컨텍스트에 사용자 JWT 보관
2. Claude API 응답에서 `tool_use` 블록 추출
3. Go 백엔드가 도구 실행 시 **사용자 JWT로 Supabase 쿼리** → RLS 자동 권한 적용
4. 결과를 `tool_result`로 Claude API에 재전송
5. JWT 만료 시 refresh token 갱신, 실패 시 재로그인 요청

### 10-2. 외부 API Rate Limit·캐싱 전략 [Critical, 섹션 4 보강]

`quotes` 갱신:
- **batch 호출**: 다중 심볼 1요청 활용 (Yahoo 등)
- **사용자 보유·관심 종목 union 후 dedup**
- **캐시 TTL**: `quotes.updated_at` 기준 60초 미만이면 재호출 안 함
- **점진 백오프**: 429 응답 시 다음 폴링 주기 2배 (최대 5분)

기본 한도 가정: Yahoo ≤2000/min(비공식, batch로 50/min 유지), FRED ≤120/min, ECOS ≤10000/day. 호출 수 메트릭은 Sentry custom metric.

### 10-3. 5년 백필 적재 전략 [Critical, 섹션 4 보강]

- **chunked insert**: 100일 단위 transaction
- **`COPY FROM`**: KRX·Yahoo CSV → PostgreSQL COPY로 일괄 적재 (수십 배 빠름)
- **async job**: 가입·종목 추가는 즉시 응답 + 백그라운드 백필. 진행도 표시
- **부분 적재 허용**: 일부 누락은 다음 일일 cron이 메움
- 용량 추정: 100종목 × 1300일 = ~6.5MB. Supabase Free 500MB 한도 안전

### 10-4. 토큰 사용량 추적 효율화 [Critical, 섹션 3·5 보강]

월별 사용량 캐시 테이블:

```sql
chat_usage_monthly (
  user_id, year_month,
  chat_count, input_tokens, output_tokens, opus_count,
  PRIMARY KEY (user_id, year_month)
)
```

- `chat_messages` 삽입 시 동일 트랜잭션에서 `chat_usage_monthly` UPSERT (`+= delta`)
- 한도 체크 = O(1) 단일 행 조회
- RLS는 `user_id = auth.uid()` 강제

총 테이블 수: 12 → **13개**.

### 10-5. 사용자 탈퇴 시 익명화 [Critical, 섹션 8 보강]

PIPA 즉시 파기와 세법 7년 보관 충돌 해소:

1. `profiles.email = 'deleted-{uuid}@quotient.app'`, `display_name = '탈퇴 사용자'`
2. `auth.users` row 삭제 (Supabase Admin API)
3. `holdings`, `watchlist`, `chat_*`, `ai_briefings`, `chat_usage_monthly` 즉시 삭제
4. `payment_events`는 **익명화 유지** (`user_id`만 보존, payload PII 마스킹). 7년 경과 행 자동 삭제 cron 분기
5. 탈퇴 확인 이메일 발송

탈퇴는 비동기 큐 처리. UI는 즉시 완료 표시. 실패 시 운영자 알림.

### 10-6. instruments unique key [Critical, 섹션 3 보강]

```sql
ALTER TABLE instruments
  ADD CONSTRAINT instruments_symbol_exchange_unique UNIQUE (symbol, exchange);
```

upsert는 항상 `ON CONFLICT (symbol, exchange) DO UPDATE`. 동일 종목의 다중 상장 시 거래소별 별개 행.

### 10-7. SSE 연결 끊김 처리 [Important, 섹션 5 보강]

- 핸들러에서 `r.Context().Done()` 감시
- 끊김 시 Claude API 호출 cancel (context propagation), 부분 응답을 `chat_messages`에 저장 (`finished_at IS NULL` 마킹)
- 클라이언트 재접속 시 미완료 메시지 감지 → "이전 응답이 중단되었습니다. 이어서 받으시겠습니까?" 버튼

### 10-8. 일일 브리핑 분산 [Important, 섹션 4·5 보강]

- 07:00 정각 일괄 호출 금지 (Claude rate limit 위배)
- **사용자별 hash → 07:00~08:00 무작위 분산** (1시간 윈도우)
- Go 내부 워커 큐 (`pg_cron` 미사용, 단순화)
- 실패 시 30분 후 1회 재시도. 재실패 시 그날 브리핑 생략 (사용자 알림 없음)

### 10-9. instrument_aliases 채우기 [Important, 섹션 3·4 보강]

- **시드**: 종목 마스터 적재 시 한글명·영문명·티커 자동 alias 등록
- **학습**: 사용자가 `search_instrument`에서 미매칭 검색 후 결과 선택 → 검색어를 alias로 학습 (예: "삼전" → 005930)
- **검증**: 학습 alias는 운영자 정기 검토 (남용·오매핑 방지)

### 10-10. AI Tool 호출 max depth [Important, 섹션 5 보강]

- 단일 사용자 메시지 응답 사이클 내 도구 호출 **최대 8회**
- 초과 시 강제 종료 + "복잡한 질문이라 충분히 답하지 못했습니다. 더 좁혀서 다시 물어주세요"
- 메트릭: tool call count per turn → Sentry custom

### 검토에서 제외 (Minor — 구현 단계 결정)

- 디자인 토큰 (색상·폰트·간격 변수화)
- 가격 표시 포맷 (천 단위 콤마·소수점 자릿수·통화별 차이)
- 시간 표시 표준 ("16:30 KST", "Today", "어제" 등)
- fuzzy 검색 인덱스 (pg_trgm 등) 선택

## 검토 이력

### 2026-05-22 — 1차 자체 검토

| 우선순위 | 항목 | 패치 위치 |
|---|---|---|
| Critical | AI tool use 컨텍스트 전달 미정의 | §10-1 |
| Critical | 외부 API rate limit·캐싱 전략 부재 | §10-2 |
| Critical | 5년 백필 적재 전략 부재 | §10-3 |
| Critical | 토큰 사용량 추적 비효율 (SUM 쿼리) | §10-4 (+ 신규 테이블) |
| Critical | 사용자 탈퇴 시 익명화 흐름 부재 | §10-5 |
| Critical | `instruments` unique key 미정의 | §10-6 |
| Important | SSE 연결 끊김 처리 부재 | §10-7 |
| Important | 일일 브리핑 동시 호출 분산 부재 | §10-8 |
| Important | `instrument_aliases` 채우기 전략 부재 | §10-9 |
| Important | AI tool 호출 max depth 부재 | §10-10 |

Minor 4건은 구현 단계에서 결정. 다음 단계: 사용자 검토 → `superpowers:writing-plans`로 구현 계획 작성 (subagent 자체 검토 의무).

---
업데이트 규칙: 각 섹션 결정 시 본 문서 즉시 갱신. 결정 변경 시 사유·날짜를 함께 명시한다.
