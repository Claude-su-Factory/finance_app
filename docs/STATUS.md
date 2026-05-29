# Quotient — 구현 상태

마지막 업데이트: 2026-05-29

## 현재 Phase

**Phase 1 — W1·W2a·W2b·W3·W4·W5 완료. Phase 1 핵심 종료 (외부 셋업·일부 후반 작업 남음).**

## 진행 중 (외부 계정 셋업만 남음 — 코드 부분 완료)

- [ ] W1-T13 Sentry + PostHog: 코드 통합 ✅ — 사용자가 계정 가입 + DSN/Key 환경변수 주입 필요 (`docs/DEPLOY.md` §5·§6)
- [ ] W1-T14 Fly + Vercel 배포: 설정 파일 ✅ — 사용자가 Fly·Vercel·Supabase 프로젝트 생성 + 시크릿 등록 + 첫 배포 명령 (`docs/DEPLOY.md` §3·§4)
- [ ] W1-T15 GitHub Actions CI/CD: workflow 3개 ✅ — 사용자가 GitHub secrets에 토큰 등록 (`docs/DEPLOY.md` §7)
- [ ] W1-T16 E2E 검증: 체크리스트 ✅ — 배포 후 사용자가 시나리오 수동 검증 (`docs/E2E_SMOKE.md`)

## Phase 0 스펙 섹션 (완료, 참고용)

`docs/superpowers/specs/2026-05-22-quotient-mvp-design.md` 10개 섹션 모두 확정 + 검토 이력 포함. 신규 작업 시 본 spec 우선 참조.

- ✅ 섹션 1 정체성·카피 / 섹션 2 정보 구조(잠정) / 섹션 3 데이터 모델 / 섹션 4 데이터 수집 / 섹션 5 AI 채팅 / 섹션 6 정보 구조 락인 / 섹션 7 인증·결제·구독 / 섹션 8 에러·관측·보안 / 섹션 9 테스트·배포·MVP 일정 / 섹션 10 보강 명세 + 검토 이력

## 완료

- ✅ Phase 0 (브레인스토밍·스펙·plan 작성·subagent 검토)
- ✅ W1-T1 모노레포 구조 + 기본 설정 (`6f35a7c`, `fda393a`)
- ✅ W1-T2 Supabase 로컬 + profiles 마이그레이션 (`2268f2a`)
- ✅ W1-T3 profiles RLS 정책 (`352b3fd`)
- ✅ W1-T4 Go 백엔드 스캐폴딩 + healthz/readyz (`f70d754`, `8700245`)
- ✅ W1-T5 JWT 검증 미들웨어 + CORS (`58bf1e8`, `bc7b483`)
- ✅ W1-T6 profiles GET·PATCH 엔드포인트 (`92063c2`, `b3ea3d7`)
- ✅ W1-T7 Next.js 스캐폴딩 + Tailwind v4 블룸버그 토큰 + shadcn v4 + 랜딩/법적 페이지 (`193cb0f`, `0939b4a`)
- ✅ W1-T8 Supabase 클라이언트 + SSR + proxy 미들웨어 (`a7ab721`)
- ✅ W1-T9 가입·로그인·이메일 인증·Google OAuth (PIPA 분리 동의 + 14세 차단) (`2d55518`, `ebde451`)
- ✅ W1-T10 비밀번호 재설정 (`c0459f0`)
- ✅ W1-T11 앱 셸 (사이드바·티커 placeholder·상태바) + 500 페이지 (`b6b93e7`)
- ✅ W1-T12 온보딩 wizard 2단계 (`f23e86b`)
- ✅ W2a-T1 instruments + aliases + 핵심 시드 (KOSPI·SPX·USD_KRW 등 7개) (`fd89564`)
- ✅ W2a-T2 prices + quotes 마이그레이션 (`4fedf5f`)
- ✅ W2a-T3 economic_indicators + fx_rates + 마켓 RLS (12 정책) (`8a44d4e`)
- ✅ W2a-T4·T5 backoff helper + 5 모델 (`a4410ed`)
- ✅ W2a-T6 KIND 어댑터 (KR 종목 마스터, EUC-KR HTML) (`5bd8c8a`)
- ✅ W2a-T7 Yahoo 어댑터 (piquette/finance-go, KR.KS/.KQ + US) (`cf4a9b3`)
- ✅ W2a-T8 FX 어댑터 (frankfurter.dev) (`5653e85`)
- ✅ W2a-T9 FRED + ECOS 어댑터 + 백오프 (`d2ff081`)
- ✅ W2a-T10 ingest 패키지 (Batch + COPY) + testcontainers 통합 테스트 3/3 (`2ac5ca4`)
- ✅ W2a-T11 config FRED/ECOS 키 (`2a343fe`)
- ✅ W2b-T1 market_hours helper (KR·US 장중) (`15a368e`)
- ✅ W2b-T2 yahoo_symbols helper (`4a5d82d`)
- ✅ W2b-T3 cron 스켈레톤 (robfig/cron, 6 잡 SkipIfStillRunning) (`c2342a6`)
- ✅ W2b-T4 JobUpdateInstruments + 시드 alias (KIND KOSPI+KOSDAQ) (`a5a59f3`)
- ✅ W2b-T5 JobUpdate{KR,US}Prices (Yahoo .KS 통합) (`c0e5686`, `945f2f8`)
- ✅ W2b-T6 JobUpdateIndexQuotes (60s TTL, 장중) (`af9ea09`)
- ✅ W2b-T7 JobUpdateFXRates (frankfurter, fx_rates+quotes 동시) (`8b67100`)
- ✅ W2b-T8 JobUpdateIndicators (FRED DFF/DGS10 + ECOS 722Y001) (`c336fd0`)
- ✅ W2b-T9·10·11 마켓 API (/v1/market/ticker, /v1/instruments/search·select) + cron 워커 main.go 통합 (`a6f75e0`)
- ✅ W2b-T12 TopTicker 실데이터 + visibility skip (`951690c`)
- ✅ W2b-T13 5년 백필 CLI (cmd/backfill) (`d2c24e3`)
- ✅ W3-T1 holdings + watchlist 마이그레이션 + RLS 7개 (`3dd1c97`)
- ✅ W3-T1.5 middleware.WithUserID helper (`d52473b`)
- ✅ W3-T2 holdings·watchlist 모델 (enriched 포함) (`17a61fa`)
- ✅ W3-T3 FX 환산 helper (FetchFXRates + ToKRW) (`206ca94`)
- ✅ W3-T4 holdings Postgres repo (`f0eae1e`)
- ✅ W3-T5 holdings CRUD 핸들러 (검증 + asset_class 가드 + enrichment) (`5555aa6`)
- ✅ W3-T6 watchlist 추가·삭제·조회 핸들러 (`2374614`)
- ✅ W3-T7 holdings·watchlist 라우트 7개 등록 (`baa5d61`)
- ✅ W3-T8 quotes 폴링 INDEX∪holdings∪watchlist 확장 + JobUpdateMarketQuotes rename (`d4698d4`)
- ✅ W3-T9 web API 클라이언트(holdings·watchlist·instruments) + authFetch + 백엔드 search 응답 확장 (`993731c`, `6397a4d`)
- ✅ W3-T10 포트폴리오 페이지 + 보유 테이블 (`a8ff34d`)
- ✅ W3-T11 보유 자산 추가 모달 + 디바운스 종목 검색 (`7716edf`)
- ✅ W3-T12 보유 자산 수정·삭제 모달 + 행 액션 (`6ca1f0b`)
- ✅ W3-T13 홈 — 총자산 카운트업 + 자산 분포 도넛 (`683303b`)
- ✅ W3-T14 홈 6카드 (상위5·마켓·관심종목·브리핑 placeholder) (`3fd3289`)
- ✅ W3-T15 온보딩 wizard 3단계 (holdings 추가 + 세션 가드 + toast) (`61ea24a`)
- ✅ W4-T1 chat_*·ai_briefings 마이그+RLS 8 정책 (`3291bcc`)
- ✅ W4-T2 ai.Client 인터페이스 + Event + chat·briefing 모델 (`f62bdc2`)
- ✅ W4-T3 MockClient 8 시나리오 키워드 매핑 + 4 테스트 (`41c5178`)
- ✅ W4-T4 RealClient stub + factory (env 자동 토글, anthropic-sdk-go) (`bd1f060`)
- ✅ W4-T5 Tool Registry + ExecuteAndSerialize (정렬·error JSON) (`91c1fb7`)
- ✅ W4-T6 9개 AI 도구 구현 (portfolio·quote·search) (`a5c4302`)
- ✅ W4-T7 컨텍스트 관리 (최근 20 + placeholder) (`50cff9a`)
- ✅ W4-T8 chat repo (세션·메시지·사용량 UPSERT + 한도 체크) (`68fa5f9`)
- ✅ W4-T9 SSE 핸들러 + tool routing (turn별 메시지 묶음·max depth 8·session_id in done) (`56576aa`)
- ✅ W4-T10 일일 브리핑 cron (사용자별 hash 분단위 분산 07:00~07:59) (`601bbfb`)
- ✅ W4-T11 briefing 핸들러 + 라우트 6개 + AI client 와이어링 (`1fee649`)
- ✅ W4-T12 chat·sessions·usage·briefing API 클라이언트 + SSE async generator (`5611439`)
- ✅ W4-T13 채팅 페이지 골격 + 세션 리스트 + 라우팅 (`bb2d7f3`)
- ✅ W4-T14 메시지·스트리밍·도구 인디케이터·입력·사용량 (`d54eac2`)
- ✅ W4-T15 BriefingCard 실 API 연결 (`c4de6d2`)
- ✅ W4-T16 SSE 끊김 처리 + 이어서 받기 (`a78c691`)
- ✅ W5-T1 history API (prices·indicators·fx) + 라우트 3개 + 7 테스트 (`a2a80cd`)
- ✅ W5-T1.5 cron KR·US prices에 `*-IDX` exchange 포함 (인덱스 일봉 적재) (`5ca1b56`)
- ✅ W5-T2 recharts 도입 + LineChartCard·Sparkline 공통 컴포넌트 (`8b39c74`)
- ✅ W5-T3 history API 클라이언트 (`e4fe042`)
- ✅ W5-T4 AdSlot 추상화 + NEXT_PUBLIC_ENABLE_ADS env (`d2fca16`)
- ✅ W5-T5 마켓 탭 페이지 + KR/US 지수·환율·경제 지표 카드 (`0e07f14`)
- ✅ W5-T6 watchlist editor + backend asset_class 가드 (`19eea04`)
- ✅ W5-T7 포트폴리오 행 7일 스파크라인 (batch fetch) (`8c01b38`)
- ✅ Paper Trading (라이브) — 가상 자금 매매 시뮬레이션. `/app/paper` 페이지, 매매·리셋 모달, 평가액 시계열 차트, 매매 일기 통합.
- ✅ AI 채팅 교육자 역할 — 개념 질문(PER·분산투자 등)에 도구 없이 친절 설명. 데이터 footer는 시세·보유 데이터 사용 답변에만 부착(`usedAnyTool` 게이팅). 정체성 spec §3 교육자 역할.

## 알려진 결함 / 백로그

- ~~**Go API**: `pgx` 풀이 postgres 슈퍼유저로 연결 — RLS 우회~~ — **2026-05-27 해결**: 사용자 데이터 핸들러(profile/holdings/watchlist/chat/briefing)를 `db.AsUser` JWT 트랜잭션으로 전환. `SET LOCAL role = authenticated` + `request.jwt.claims/.sub` LOCAL 적용으로 Supabase RLS 자동 트리거. cron·도구 레지스트리·공개 read·브리핑 작성은 슈퍼유저 풀 유지(다중 사용자 fan-out). 격리 회귀 가드: `apps/api/internal/db/rls_integration_test.go` 3 케이스 (holdings·watchlist·chat).
- ~~**AI 도구 호출 JWT 미전파**~~ — **2026-05-27 해결**: Tool 인터페이스에 `RequiresUserContext()` + `db.Executor` 인자 추가. ExecuteAndSerialize가 사용자 데이터 도구 4개만 `db.AsUser` 트랜잭션 wrap. spec §10-1 완전 정합.
- ~~**Profile handler 통합 테스트 미구현**~~ — **2026-05-27 해결**: `profile_integration_test.go` 4건(자동 생성 확인 + PATCH 영속화 + 잘못된 currency 거부 + RLS 격리).
- **Next.js 미들웨어**: profile fetch가 모든 `/app/*` 요청마다 발생 (N+1). 사용자 100명까지는 무시 가능. 트래픽 증가 시 JWT custom claim 캐싱 검토
- ~~**Pretendard 폰트 next/font 최적화**~~ — **2026-05-27 해결**: pretendard 패키지의 woff2를 `next/font/local`로 self-host. CSS @import 제거 + preload + display: swap 자동 적용.
- **Profile handler 통합 테스트**: fake repo만 있고 실제 Postgres 통합 테스트는 W2 testcontainers-go 도입 후
- **stack 버전 변경**: Next.js 16.2.6 + Tailwind v4 (스펙은 15 + v3) — 최신 GA 수용, 스펙 문서 업데이트 필요
- **Go 1.25 강제**: pgx/v5 v5.9.2가 Go 1.25 요구. Task 14 Dockerfile · Task 15 CI 모두 `golang:1.25-alpine` / `go-version: "1.25"` 사용 필요
- **Supabase Auth JWT secret**: CLI v2.98이 legacy 키 노출 안 함. 사용자가 dashboard에서 "Legacy JWT Secret" 활성화 필요. JWKS 마이그레이션 백로그
- ~~**KOSDAQ 종목 cron 일봉 누락**~~ — **2026-05-27 해결**: KIND 어댑터가 `Exchange: "KRX"` 하드코딩하던 것을 시장별로 `"KOSPI"`/`"KOSDAQ"` 적재로 변경. W5 cron 매칭(`KOSPI`/`KOSDAQ` 포함) + `StockYahooSymbol` (.KS/.KQ 분기)와 정합 → KOSDAQ 종목 일봉 정상 갱신. 기존 KRX 적재분은 alias seed 쿼리에서 KRX 호환 유지로 backward compat.
- ~~**US 장중 NY Friday 후반 세션 누락**~~ — **2026-05-27 해결**: `IsUSMarketOpen`을 NY 타임존 기반으로 변경. KST 토요일 새벽이 NY 금요일 장중이면 자동 true.
- ~~**US 장중 DST 미반영**~~ — **2026-05-27 해결**: `time.LoadLocation("America/New_York")` 사용 → DST 자동 적용 (EST/EDT 전환).
- **fx_rates change_pct 첫날 0**: frankfurter 일별 갱신. 첫 배포로 fx_rates에 오늘 행만 있으면 change_pct=0 (다음 영업일 정상화)
- ~~**포트폴리오 우측 sliding panel 미구현**~~ — **2026-05-27 해결**: HoldingDetailPanel — 행 클릭 시 우측 슬라이드, 30일 차트 + 보유 상세 + 수정/삭제 액션. ESC·backdrop 닫기, 선택 행 하이라이트.
- **AdSense 미가입**: AdSlot은 `NEXT_PUBLIC_ENABLE_ADS=false` 기본 → placeholder만. 가입자 100명·일평균 PV 500 도달 시 Phase 2에서 활성
- ~~**AI RealClient 미구현**~~ — **2026-05-27 해결**: anthropic-sdk-go v1.45 실 구현 완료. `ANTHROPIC_API_KEY` 설정 시 자동 활성, adaptive thinking(4.6/4.7)·prompt caching·tool streaming·SSE 끊김 ctx 감시 모두 포함
- ~~**AI 컨텍스트 요약 미구현**~~ — **2026-05-27 해결**: `BuildMessages(ctx, all, summarizer)`로 시그니처 확장. summarizer 주입 시 Haiku로 dropped 메시지 1턴 요약, nil이면 기존 placeholder fallback. chat handler가 `h.client` 그대로 주입 (Mock/Real 무관).
- ~~**일일 브리핑 도구 호출 없음**~~ — **2026-05-27 해결**: `briefing_worker`가 brief 작성 전 `get_portfolio` + `get_market_overview` + `get_watchlist` 도구를 직접 실행하여 결과 JSON을 system prompt에 주입. spec §10-8 충족. registry nil이면 fallback (데이터 없이 일반 안내).
- **사용량 토큰 turn-by-turn 누적 단순화**: 마지막 turn row에 누적 합계를 저장. turn별 분리 metrics는 v2
- ~~**disclaimer 강제 부착 system prompt 의존**~~ — **2026-05-27 해결**: chat handler가 마지막 turn(도구 호출 없음) 영속화 직전에 turnText 끝에 `(데이터 기준: ... KST, 시세 지연 15분)` 강제 부착 + SSE token으로 emit. 이미 포함 시 중복 방지. **2026-05-29 보강**: 교육자 역할 도입으로 footer 강제 부착을 `usedAnyTool`로 게이팅 — 시세·보유 데이터를 한 번도 안 쓴 개념 답변에는 부착 안 함(붙일 데이터가 없으므로).

## 최근 변경 이력

- 2026-05-29 AI 채팅 교육자 역할 추가 — 분석가 페르소나에 "학습 도우미" 결합. 개념 질문(PER·분산투자·ETF 등)은 도구 호출 없이 일반 지식으로 친절히 설명하고 데이터 footer를 생략. 핸들러의 footer 강제 부착을 `usedAnyTool` 게이팅으로 변경 — 시세·보유 데이터를 실제 사용한 답변에만 `(데이터 기준 …)` 부착. mock도 도구 결과 여부로 footer 분기. 신규 테스트 2(개념 무footer·데이터 footer) + SSE 토큰 재조립 헬퍼. 정체성 spec §3 교육자 역할 이행. 통합 테스트 자체 종목 시드로 백필 의존 제거(`seedTestInstrument`). CSV import 드롭(유지보수 비용 대비 가치 부족).
- 2026-05-28 Paper Trading (라이브) 출시 — 사이드바 📈 신규 탭 + `/app/paper` 별도 페이지. 가상 자금(default ₩1,000만) + 즉시 시장가 체결 + 가중 평균 avg_cost + 매매 이유 → journal_entries auto entry 자동. 리셋 기능(holdings 삭제 + cash 초기화 + transactions active=false 보존). 신규 테이블 3 + RLS 10 정책 + 4 HTTP endpoint(/v1/paper/*) + 12 unit + 4 integration. 평가액 시계열은 transactions replay + 시점별 가격(알파 카드 패턴 재사용). 정체성 spec §1 3축 마지막 축 이행. 백테스트(서브시스템 B)는 별도.
- 2026-05-28 AI 매매 일기 출시 — Holdings CRUD 통합(reason → auto entry) + `/app/journal` 별도 페이지(manual entry 자유 작성). 자동 월간 회고 cron(매월 1일 07:00 KST 사용자 hash 분단위 분산) + on-demand 분석 버튼(채팅 한도 차감) + 채팅 `analyze_journal` 도구. 신규 테이블 2(`journal_entries`·`analysis_runs`) + RLS 6 정책 + 6 HTTP endpoint(/v1/journal/*) + 7 unit + 2 integration. 사이드바 📓 아이콘 추가. 정체성 spec §3 최우선 차별화 카드 이행.
- 2026-05-28 알파 카드 출시 — 홈 1행 3번째에 "포트폴리오 vs KOSPI · S&P 500 · 한미 60/40" 비교. 기간 토글 1M/90D/1Y/All, 시점별 환율, backward simulation. 빈 상태(가입 < 7일 + 보유 자산 0) 처리. `internal/portfolio/` 신규 패키지 + `GET /v1/portfolio/alpha` 핸들러 + 9 unit + 1 integration test. 지수 백필 CLI 확장(Task 0). 정체성 spec §2 약속 이행.
- 2026-05-27 정체성 3축 정립 — "실 자산 분석 + Paper Trading + AI 학습". 랜딩 페이지 완전 재작성: 라이브 ticker 띠 + Hero(3축 가치) + Dashboard SVG preview + AI Chat preview(도구 호출 인디케이터) + 4 기능 그리드 + Paper Trading teaser(Phase 2) + 신뢰 카드(안 하는 것 4) + FAQ + 하단 CTA + 확장 footer. 자산 추가 모달·온보딩에 "본인 입력·검증 없음·분석 도구" 안내 박스. 전체 사용자 수익률 랭킹은 영구 불가(자기 신고 데이터 + 투자권유 영역)로 ROADMAP 명시, 후킹은 알파 카드(외부 지수 비교)로 대체.
- 2026-05-27 수익화 1단계 코드 — AdSense 실 통합(`<ins>` + `adsbygoogle.push`, layout 스크립트 lazy load) + Toss 개인 후원 사이드바 footer 아이콘. 환경변수 미설정 시 둘 다 비활성 → MVP 무료 출시 정합. 노출 정책: 마켓 페이지 하단만(`market_bottom` 슬롯). 사용자가 AdSense 가입 후 `NEXT_PUBLIC_ADSENSE_CLIENT` + `NEXT_PUBLIC_ADSENSE_SLOT_MARKET_BOTTOM` + `NEXT_PUBLIC_TOSS_DONATION_URL` 주입하면 활성.
- 2026-05-27 W1-T13~T16 코드 부분 완료 (외부 계정 셋업은 사용자 액션 남음). Sentry(go·next) + PostHog(next) SDK 통합 — DSN/Key 미설정 시 no-op. Fly Dockerfile + fly.toml, Vercel vercel.json, GitHub Actions(ci.yml + deploy-api.yml + deploy-web.yml). 가이드 `docs/DEPLOY.md` + 스모크 체크리스트 `docs/E2E_SMOKE.md`.
- 2026-05-27 AI 도구 호출 JWT 전파. Tool 인터페이스에 `RequiresUserContext()` 추가, 모든 9개 도구가 `db.Executor` 받음. `ExecuteAndSerialize`가 사용자 데이터 도구 4개(portfolio/holdingDetail/calcMetrics/watchlist)만 `db.AsUser` 트랜잭션 wrap → 공개 도구 5개(quote/priceHistory/marketOverview/search/economicIndicator)는 슈퍼유저 풀 그대로. chat tool routing + briefing 워커 호출 wiring 포함. spec §10-1 완전 정합.
- 2026-05-27 Profile handler 통합 테스트 4건 추가 — 실 Postgres + 트리거 자동 생성 + PATCH→GET 영속화 + RLS 격리 종단 검증.
- 2026-05-27 사용자 데이터 핸들러(profile/holdings/watchlist/chat/briefing)를 `db.AsUser` 사용자 JWT 트랜잭션으로 전환. `Executor` 인터페이스 + `set_config('role','authenticated',true)` + `request.jwt.claims/.sub` LOCAL 적용 → Supabase RLS 자동 트리거. 애플리케이션 `WHERE user_id` 필터는 fail-safe 이중 방어로 유지. testcontainers 없이 로컬 Supabase 기반 RLS 격리 통합 테스트 3 케이스 모두 PASS. cron·도구·공개 read는 슈퍼유저 풀 유지.
- 2026-05-27 STATUS 결함 5건 일괄 fix: KIND 시장별 exchange 적재(KOSDAQ 일봉 정상화), Haiku 컨텍스트 요약, 일일 브리핑 도구 데이터 주입(spec §10-8), Pretendard `next/font/local` 최적화.
- 2026-05-27 Tier 1 묶음 fix. (1) chat handler post-process로 disclaimer 강제 부착 — RealClient 응답에도 spec §5 가드레일 보장. (2) `IsUSMarketOpen`을 NY 타임존(`America/New_York`) 기반으로 재구현 — DST 자동 적용 + NY Friday 후반 세션 자동 포함. 테스트에 EDT 케이스 추가.
- 2026-05-27 포트폴리오 sliding panel. 행 클릭 시 우측에서 슬라이드인하는 상세 패널 — 헤더(심볼·이름·거래소·통화) + 현재가/손익 + 30일 LineChartCard + 보유 상세(8필드) + 메모 + 수정/삭제 액션. ESC + backdrop 닫기, 행 클릭 ↔ 수정/삭제 버튼은 stopPropagation으로 분리, 선택 행은 bb-accent 하이라이트.
- 2026-05-27 명령 팔레트 ⌘K + vim-like 단축키. cmdk 도입 + `CommandPalette`(종목 검색·AI 묻기·탭 이동) + `useKeyboardShortcuts`(⌘K/`/`/1~5/`g h|p|c|m|s`). 입력 필드 안에서는 chord/숫자 단축키 무시.
- 2026-05-27 AI RealClient 실 구현. `anthropic-sdk-go` v1.45 어댑터 — Messages.NewStreaming + adaptive thinking(Sonnet 4.6·Opus 4.6/4.7) + prompt caching(system+tools) + tool_use 누적/emit + ctx.Done 감시. 사용자가 ANTHROPIC_API_KEY 설정 시 즉시 실 채팅 활성.
- 2026-05-26 W5 전체 완료. 마켓 탭(`/app/market`) — KR/US 지수·환율·경제 지표 4종 카드 + 관심 종목 editor + AdSlot. recharts 도입(LineChartCard·Sparkline). history API 3 라우트(prices·indicators·fx) + 인덱스 일봉 cron 확장(`*-IDX` exchange 포함). 포트폴리오 행 7일 스파크라인(batch fetch). watchlist backend asset_class 가드. Phase 1 핵심 완료.
- 2026-05-25 W4 전체 완료. AI 채팅 — Mock·Stub 클라이언트 + 9개 도구 + SSE 스트리밍(tool routing turn별 메시지 묶음, max depth 8, session_id in done) + 사용량 추적(월 30회·50K/10K·Opus 1회) + 일일 브리핑 cron(사용자 hash 분단위 분산) + 채팅 UI(세션 리스트·메시지·도구 인디케이터·입력·사용량 배지) + 끊김 처리(이어서 받기). dev에서 API 키 없이 Mock으로 전 흐름 검증 가능.
- 2026-05-23 W3 후속 FX fix. `jobs_fx.go`에 EUR_KRW/JPY_KRW derived 계산 추가 (USD_KRW / USD_EUR, USD_KRW / USD_JPY). prev rates에도 동일 적용으로 change_pct 정상화. holdings KRW 환산 정확도 회복.
- 2026-05-23 W3 전체 완료. holdings·watchlist 마이그+CRUD API + asset_class 가드 + FX 환산 + cron polling union 확장(JobUpdateMarketQuotes rename) + 포트폴리오 페이지(CRUD 모달) + 홈 대시보드 6카드 + 온보딩 3단계 복원(세션 가드+toast).
- 2026-05-22 W2b 전체 완료. cron 워커 6 잡(robfig/cron + SkipIfStillRunning) + 마켓 API 3 라우트 + TopTicker 실데이터 + 5년 백필 CLI. 시드 alias 자동 등록(§10-9) 포함. 알려진 한계: KOSDAQ .KS fallback, NY Friday session 누락, DST 미반영.
- 2026-05-22 W2a 전체 (T1~T11) 완료. 4 마이그레이션 + 5 어댑터(KIND·Yahoo·FX·FRED·ECOS) + 백오프 + 6 모델 + ingest(Batch+COPY) + testcontainers. KRX 직접 호출 불가 확인 후 KIND+Yahoo 단일화.
- 2026-05-22 W1-T11·T12 완료. 앱 셸 + 온보딩 wizard 2단계.
- 2026-05-22 W1-T8·T9·T10 완료. Supabase SSR/proxy 미들웨어 + 가입/로그인/OAuth/비밀번호 재설정 (PIPA 준수).
- 2026-05-22 W1-T7 완료. Next.js 16 + Tailwind v4 + shadcn v4 (스펙 15/v3에서 최신 GA로 이전).
- 2026-05-22 W1-T4·T5·T6 완료. Go API healthz/readyz + JWT/CORS + profiles 엔드포인트.
- 2026-05-22 W1-T2·T3 완료. Supabase 로컬 + profiles 테이블·RLS.
- 2026-05-22 W1-T1 완료. 모노레포 구조 + 기본 설정.
- 2026-05-22 W1 plan 작성 완료 + subagent 검토 (general-purpose) 1차 사이클. 9 Critical + 12 Important + 6 Minor 패치 적용.
- 2026-05-22 1차 자체 검토 완료. Critical 6건 + Important 4건 식별·패치 (섹션 10). 사용자 검토 단계로 이행.
- 2026-05-22 섹션 9 (테스트·배포·MVP 일정) 확정. 일정 6주, 모두 무료 출시, 광고 슬롯 추상화, 결제 비활성.
- 2026-05-22 수익화 경로 1 선택: MVP 완전 무료 + 광고 슬롯 토글 + 결제는 사업자 등록 시점 활성화.
- 2026-05-22 섹션 8 (에러·관측·보안) 확정. Sentry + PostHog + Fly logs + Resend, 3 레이어 에러 처리, RLS 이중 방어, rehype-sanitize 의무, PIPA 준수.
- 2026-05-22 섹션 7 (인증·결제·구독) 확정. Supabase Auth + Google OAuth, Toss 빌링키 정기결제, 7일 이내 환불, 부가세 포함 ₩14,900, webhook 멱등성 의무.
- 2026-05-22 섹션 6 (정보 구조 락인) 확정. 경로 맵 12개, 온보딩 wizard, 전역 컴포넌트, Long-running 작업 progress UI 정책(블룸버그 풍 1px 라인 + monospace 진행률 + sonner toast).
- 2026-05-22 섹션 5 (AI 채팅 흐름) 확정. 모델 라우팅(Sonnet 4.6 기본 + Opus 4.7 심층 + Haiku 4.5 내부), tool use 9개, Freemium 10회/월·Pro ₩14,900, 일일 브리핑 07:00 KST.
- 2026-05-22 섹션 4 (데이터 수집 파이프라인) 확정. KRX·Yahoo·FRED·ECOS·exchangerate.host. 시세 15분 지연. KIS는 Phase 3.
- 2026-05-22 하네스 골격 생성 + 브레인스토밍 섹션 1·2·3 결정 사항 스펙 파일에 이관.
- 2026-05-22 프로젝트 시작. MVP 스코프·아키텍처·데이터 모델 확정.

---
업데이트 규칙: 기능 완료 시 즉시 갱신. 알려진 결함은 발견 즉시 등재.
