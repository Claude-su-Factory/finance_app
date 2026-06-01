# 사용자가 직접 해야 하는 작업

코드 외 — 외부 계정 가입·키 발급·운영 시점 명령 등 **에이전트가 대신 못 하는 것만** 모음.
완료 시 항목을 `~~취소선~~` 처리하고 하단 "완료" 절로 이동.

> 갱신 규칙: 새 작업 완료 직후 에이전트가 본 문서에 액션을 추가/이동한다.
> CLAUDE.md 빠른 네비게이션에 등재되어 있어 매 세션 시작 시 로드된다.

---

## 🔴 시급 — 다음 기능 동작에 필요

*(이 절의 모든 백필·마이그레이션 항목은 자동화 완료. 부팅 시 `SeedIfEmpty`가 KOSPI·KOSDAQ·SPX·NDX 지수 + NASDAQ 30 시드를 자동 백필하며, 스키마 마이그레이션은 Fly `release_command`(`/app/migrate`)가 자동 적용한다. 수동 실행 불필요.)*

> ⚠️ **마이그레이션 SQL 작성 주의**: `release_command`는 비0 exit 시 해당 배포 전체를 중단(fail-closed)한다. 마이그레이션 SQL은 트랜잭션-안전이어야 한다 — standalone `BEGIN;/COMMIT;` 포함 금지, `CREATE INDEX CONCURRENTLY` 금지. 이러한 구문이 필요하면 마이그레이터에 no-tx 경로를 먼저 추가해야 한다.

---

## 🟡 운영 배포 시 — Phase 1 출시 전 마지막 단계

상세 절차는 [`docs/DEPLOY.md`](DEPLOY.md) 참조.

### 외부 계정 가입 (결제 카드 필요)

- [ ] **Fly.io 계정** — `flyctl auth login` ($5/mo free credit, 결제 카드 등록 필수)
- [ ] **Supabase Cloud 프로젝트** — region `ap-northeast-2`, `supabase link --project-ref <ref>` (스키마 마이그레이션은 Fly `release_command`가 자동 적용 — 수동 `supabase db push` 불필요)
- [ ] **Vercel 계정** — GitHub repo 연결 (`apps/web` Root Directory 지정 필수)
- [ ] **Anthropic API 키** — `sk-ant-...` 발급 (결제 카드 필요)
- [ ] **Sentry 프로젝트 2개** (Go + Next.js) — DSN 발급 (옵션, Developer Free)
- [ ] **PostHog 프로젝트** — Project API Key (옵션, Free 1M events/mo)
- [ ] **FRED API key** — https://fred.stlouisfed.org (무료)
- [ ] **ECOS API key** — https://ecos.bok.or.kr (한국은행, 무료)

### 시크릿 등록

- [ ] **Fly 시크릿**:
  ```bash
  flyctl secrets set \
    DATABASE_URL="postgresql://postgres.<ref>:<pw>@aws-0-ap-northeast-2.pooler.supabase.com:5432/postgres" \
    SUPABASE_JWT_SECRET="<legacy-jwt-secret>" \
    ANTHROPIC_API_KEY="<sk-ant-...>" \
    FRED_API_KEY="<key>" \
    ECOS_API_KEY="<key>" \
    SENTRY_DSN_API="<dsn>" \
    CORS_ORIGIN="https://<vercel-domain>.vercel.app"
  ```
  ⚠️ DATABASE_URL은 **Session pooler 포트 5432** (Transaction pooler 6543 ❌)

- [ ] **Vercel 환경변수** (Production + Preview):
  - `NEXT_PUBLIC_SUPABASE_URL`, `NEXT_PUBLIC_SUPABASE_ANON_KEY`
  - `NEXT_PUBLIC_API_URL` (Fly 도메인)
  - `NEXT_PUBLIC_SENTRY_DSN_WEB`, `NEXT_PUBLIC_POSTHOG_KEY`, `NEXT_PUBLIC_POSTHOG_HOST`
  - `NEXT_PUBLIC_ENV=production`, `NEXT_PUBLIC_ENABLE_ADS=false`

- [ ] **GitHub Secrets** (Repository Settings → Secrets):
  - `FLY_API_TOKEN` (`flyctl tokens create deploy --name "github-actions"` 출력)
  - (옵션) `VERCEL_TOKEN`, `VERCEL_ORG_ID`, `VERCEL_PROJECT_ID`

### Supabase Auth 설정

- [ ] **Site URL + Redirect URLs** — Dashboard → Authentication → URL Configuration에 Vercel 도메인 등록
- [ ] **(옵션) Google OAuth 활성화** — Google Cloud Console에서 OAuth 2.0 Client 생성 후 Supabase에 client_id/secret 입력

### 배포 후 검증

- [ ] **E2E 스모크 시나리오 9단계** — [`docs/E2E_SMOKE.md`](E2E_SMOKE.md) 통과

> 참고: `/app/backfill` CLI는 여전히 존재하며 `flyctl ssh console -C "/app/backfill --market=INDICES --years=5"` 형태로 수동 전체 재백필이 가능하다. 단, 첫 배포 시 필수 단계가 아니다 — 부팅 시 `SeedIfEmpty`가 자동 처리한다.

---

## 🔵 개발 도구 (선택)

- [선택] UI 미리보기 검수: `cd apps/web && npm run preview` → http://localhost:3000/preview. 외부 계정·키 불필요(가짜 데이터). 3000 포트만 비어 있으면 됨.

---

## 🟢 가입자 100명 도달 시 — Phase 2 활성

- [ ] **AdSense 계정 가입 + 마켓 페이지 하단 슬롯 발급**
  - Vercel env: `NEXT_PUBLIC_ENABLE_ADS=true`, `NEXT_PUBLIC_ADSENSE_CLIENT=ca-pub-...`, `NEXT_PUBLIC_ADSENSE_SLOT_MARKET_BOTTOM=<slot-id>`

- [ ] **Toss 후원 닉네임 등록** (옵션, 즉시 가능)
  - https://toss.me 에서 닉네임 등록 후
  - Vercel env: `NEXT_PUBLIC_TOSS_DONATION_URL=https://toss.me/<nickname>` → 사이드바 footer에 ♡ 아이콘 노출

---

## 🔵 Phase 3 — 사업자 등록 후

- [ ] **사업자 등록 + 통신판매업 신고**
- [ ] **Toss Payments 가맹 계약** — 빌링키 정기결제 활성
- [ ] **Fly secret**: `PAYMENTS_ENABLED=true`
- [ ] **(선택) 증권사 affiliate 제휴** — 단순 광고 형태로만 ("추천" 표현 금지)

---

## ✅ 완료

(완료 항목 이동 — 가장 최근이 위)

- ~~**지수 5년 백필 (KOSPI·KOSDAQ·SPX·NDX)**~~ — 부팅 시 `SeedIfEmpty` 자동 처리로 대체(2026-05-30)
- ~~**백테스트 대상 종목 가격 백필 (NASDAQ 시드)**~~ — 부팅 시 `SeedIfEmpty` 자동 처리로 대체(2026-05-30)
- ~~**production 지수 백필** (`flyctl ssh console -C "/app/backfill ..."`)~~ — 부팅 자동화로 필수 단계 해제(2026-05-30)
- ~~**`supabase db push` (스키마 적용)**~~ — Fly `release_command` (`/app/migrate`) 자동 적용으로 대체(2026-05-30)
