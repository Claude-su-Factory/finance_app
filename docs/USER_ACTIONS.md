# 사용자가 직접 해야 하는 작업

코드 외 — 외부 계정 가입·키 발급·운영 시점 명령 등 **에이전트가 대신 못 하는 것만** 모음.
완료 시 항목을 `~~취소선~~` 처리하고 하단 "완료" 절로 이동.

> 갱신 규칙: 새 작업 완료 직후 에이전트가 본 문서에 액션을 추가/이동한다.
> CLAUDE.md 빠른 네비게이션에 등재되어 있어 매 세션 시작 시 로드된다.

---

## 🔴 시급 — 다음 기능 동작에 필요

- [ ] **지수 5년 백필 (KOSPI·KOSDAQ·SPX·NDX·DJI)** — *알파 카드 동작 전제*
  - 로컬: `cd apps/api && set -a; source .env; set +a; go run ./cmd/backfill --market=INDICES --years=5`
  - 운영: `flyctl ssh console -C "/app/backfill --market=INDICES --years=5"`
  - 1회 실행. ~5분 소요. 미실행 시 알파 카드는 "데이터 부족"으로 빈 상태.

- [ ] **백테스트 대상 종목 가격 백필** — *백테스트 동작 전제 (spec §3-3)*
  - KOSPI·KOSDAQ 전종목 + 지수는 W2b 백필 CLI로 적재됨. **NASDAQ은 시드 30종목만** 보유 → 그 외 미국 종목을 바스켓에 넣으면 클램프되거나 "데이터 부족"으로 거부될 수 있음.
  - 미국 종목 백필: `cd apps/api && set -a; source .env; set +a; go run ./cmd/backfill --market=NASDAQ --years=5` (대상 확장은 백필 CLI의 NASDAQ 시드 목록 편집)
  - 운영: `flyctl ssh console -C "/app/backfill --market=NASDAQ --years=5"`
  - 지수 백필(위 항목)이 선행돼야 벤치마크 3종(KOSPI·S&P·60/40)이 그려진다.

---

## 🟡 운영 배포 시 — Phase 1 출시 전 마지막 단계

상세 절차는 [`docs/DEPLOY.md`](DEPLOY.md) 참조.

### 외부 계정 가입 (결제 카드 필요)

- [ ] **Fly.io 계정** — `flyctl auth login` ($5/mo free credit, 결제 카드 등록 필수)
- [ ] **Supabase Cloud 프로젝트** — region `ap-northeast-2`, `supabase link --project-ref <ref>` + `supabase db push`
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
- [ ] **production 지수 백필** — Fly machine 안에서 `flyctl ssh console -C "/app/backfill --market=INDICES --years=5"`

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

(아직 없음)
