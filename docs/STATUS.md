# Quotient — 구현 상태

마지막 업데이트: 2026-05-23

## 현재 Phase

**Phase 1 — W1·W2a·W2b·W3 완료. W4 (AI 채팅) 작성 대기.**

## 진행 중

- [ ] W1-T13 Sentry + PostHog (외부 DSN 필요)
- [ ] W1-T14 Fly + Vercel 배포 (외부 계정 필요)
- [ ] W1-T15 GitHub Actions CI/CD (외부 토큰 필요)
- [ ] W1-T16 통합 동작 검증 (W2b·W3+ 후 풀 E2E — W2b 런타임 검증 11항 plan §Task 14 참조)

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

## 알려진 결함 / 백로그

- **Go API**: `pgx` 풀이 postgres 슈퍼유저로 연결 — RLS 우회. 현재는 핸들러 `WHERE id = uid` 필터로 격리. W3 이전에 사용자 JWT 기반 쿼리로 전환 결정 필요 (spec §10-1, profile_repo_pg.go TODO)
- **Next.js 미들웨어**: profile fetch가 모든 `/app/*` 요청마다 발생 (N+1). 사용자 100명까지는 무시 가능. 트래픽 증가 시 JWT custom claim 캐싱 검토
- **Pretendard 폰트**: pretendard CSS만 import, next/font/local로 최적화 미적용
- **Profile handler 통합 테스트**: fake repo만 있고 실제 Postgres 통합 테스트는 W2 testcontainers-go 도입 후
- **stack 버전 변경**: Next.js 16.2.6 + Tailwind v4 (스펙은 15 + v3) — 최신 GA 수용, 스펙 문서 업데이트 필요
- **Go 1.25 강제**: pgx/v5 v5.9.2가 Go 1.25 요구. Task 14 Dockerfile · Task 15 CI 모두 `golang:1.25-alpine` / `go-version: "1.25"` 사용 필요
- **Supabase Auth JWT secret**: CLI v2.98이 legacy 키 노출 안 함. 사용자가 dashboard에서 "Legacy JWT Secret" 활성화 필요. JWKS 마이그레이션 백로그
- **KOSDAQ 종목 cron 일봉 누락**: `JobUpdateKRPrices`가 `.KS` 기본만 시도. KOSDAQ 종목은 backfill CLI(`-market KOSDAQ`)로 별도 백필 후 cron이 일별 갱신 못 함. W3에서 `instruments.market` 컬럼 추가 검토
- **US 장중 NY Friday 후반 세션 누락**: `IsUSMarketOpen`이 토요일 일괄 false. KST 토요일 새벽 NY Friday 정규장(quotes 분 단위 폴링) skip. 일봉(prices)은 06:00 cron이 별도 처리 → 데이터 손실 없음
- **US 장중 DST 미반영**: KST 23:30~06:00 고정. 미국 일광절약시간 기간 30분 어긋남
- **fx_rates change_pct 첫날 0**: frankfurter 일별 갱신. 첫 배포로 fx_rates에 오늘 행만 있으면 change_pct=0 (다음 영업일 정상화)
- **FX 환율 EUR/JPY 미적재**: `JobUpdateFXRates`가 rateMap key `USD_EUR`/`USD_JPY` 생성하나 instrument symbol은 `EUR_KRW`/`JPY_KRW`라 매칭 실패. 결과: EUR/JPY 자산 보유 시 KRW 환산이 fallback 1.0 → 평가액 왜곡. MVP는 KR/US 자산 가정으로 W3 비범위 처리, W5 마켓 탭 작업 시 정리 예정 (`pricing.go` Warn 로그로 표면화)
- **watchlist 추가 UI 부재**: 백엔드 API + 홈 미니카드 조회만 W3에 포함. 종목 추가/제거 UI는 W5 마켓 탭에서 제공 예정
- **포트폴리오 미니 스파크라인 미구현**: 스펙 §6 보유 테이블의 종목별 7일 가격 sparkline은 Phase 1 후반(W5)로 미룸. recharts 도입 + prices 7일 조회 API 동시 작업
- **포트폴리오 우측 sliding panel 미구현**: 스펙 §6 선택 행 상세 패널은 위와 동일 시점

## 최근 변경 이력

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
