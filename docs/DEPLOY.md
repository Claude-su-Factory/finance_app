# 배포 가이드

Quotient는 **Fly.io** (Go API + cron 워커) + **Vercel** (Next.js) + **Supabase** (Postgres + Auth) 3개 외부 서비스를 사용합니다.
모든 배포 설정은 코드에 포함되어 있고, 사용자는 계정 가입·토큰 발급·시크릿 등록만 하면 됩니다.

---

## 1. 사전 준비

### 계정 가입
- [Supabase](https://supabase.com) — Free 플랜으로 시작
- [Fly.io](https://fly.io) — 결제 카드 등록 필수($5/mo free credit 제공, 2024년 9월 이후 free hobby 플랜 폐지)
- [Vercel](https://vercel.com) — Hobby Free
- [Sentry](https://sentry.io) — Developer Free (월 5K 이벤트)
- [PostHog](https://posthog.com) — Free 1M 이벤트/월
- [Anthropic](https://console.anthropic.com) — 결제 카드 등록 필요(사용량 기반)
- [FRED](https://fred.stlouisfed.org/docs/api/api_key.html) — 무료 키
- [ECOS](https://ecos.bok.or.kr/api/) — 무료 키

### 로컬 CLI 설치
```bash
brew install supabase/tap/supabase  # 이미 사용 중
brew install flyctl                  # Fly.io CLI
npm i -g vercel                      # Vercel CLI
```

---

## 2. Supabase 프로젝트 셋업

1. Dashboard → "New Project" — region: `ap-northeast-2` (Seoul)
2. 마이그레이션 적용:
   ```bash
   supabase link --project-ref <project-ref>
   supabase db push   # supabase/migrations/ 8개 모두 적용
   ```
3. Auth → Providers에서 Google OAuth 활성화(client_id/secret 입력) — 필요 시
4. Auth → URL Configuration → Site URL · Redirect URLs에 Vercel 도메인 등록
5. Settings → API에서 다음 값 복사 보관:
   - `Project URL` → `NEXT_PUBLIC_SUPABASE_URL`
   - `anon public` 키 → `NEXT_PUBLIC_SUPABASE_ANON_KEY`
   - `service_role` 키 → `SUPABASE_SERVICE_ROLE_KEY` (서버 전용, 노출 금지)
   - `JWT Settings` → "Legacy JWT Secret" 활성화 후 secret 복사 → `SUPABASE_JWT_SECRET`
6. Settings → Database → Connection string 탭에서 **"Session pooler"**(포트 5432) 또는 **"Direct connection"** 복사 → `DATABASE_URL`
   - ⚠️ "Transaction pooler"(포트 6543)는 사용하지 말 것 — 트랜잭션마다 connection 회전 → 우리 코드의 `db.AsUser` LOCAL set + prepared statement 캐시와 비효율. Transaction pooler는 serverless 환경(Vercel functions 등)에서만 권장.
   - Fly처럼 long-running 서비스는 Session pooler 또는 Direct connection이 정합.

---

## 3. Fly.io API 배포

```bash
cd apps/api
flyctl auth login
flyctl apps create <your-app-name>  # 예: quotient-api-yhj
```

`fly.toml`의 `app = "quotient-api"` 부분을 본인 앱 이름으로 변경.

시크릿 등록(한 번만 — `DATABASE_URL`은 Session pooler 또는 direct connection 포트 5432 사용):
```bash
flyctl secrets set \
  DATABASE_URL="postgresql://postgres.<ref>:<pw>@aws-0-ap-northeast-2.pooler.supabase.com:5432/postgres" \
  SUPABASE_JWT_SECRET="<legacy-jwt-secret>" \
  ANTHROPIC_API_KEY="<sk-ant-...>" \
  FRED_API_KEY="<key>" \
  ECOS_API_KEY="<key>" \
  SENTRY_DSN_API="<https://...@sentry.io/...>" \
  CORS_ORIGIN="https://<your-vercel-domain>.vercel.app"
```

> `CORS_ORIGIN`은 콤마로 여러 origin 지정 가능. Vercel preview URL을 허용하려면 와일드카드 패턴 추가:
> `CORS_ORIGIN="https://quotient.example.com,https://quotient-*.vercel.app"`

배포:
```bash
flyctl deploy
flyctl logs    # 실시간 로그 stream (default)
flyctl open    # /healthz 200 확인
```

5년 백필(최초 1회) — Dockerfile에 `backfill` 바이너리 포함됨:
```bash
# 옵션 1: 운영 컨테이너 안에서 one-off 실행 (DB connection 재사용)
flyctl ssh console -C "/app/backfill"

# 옵션 2: 로컬에서 production DSN으로 실행
DATABASE_URL="<prod-dsn>" go run ./cmd/backfill
```

---

## 4. Vercel Next.js 배포

```bash
cd apps/web
vercel    # 첫 실행: 프로젝트 생성 prompt (team·project name·root dir)
          # 이후 실행: 기존 프로젝트로 deploy
```

> 이미 Vercel Dashboard에서 프로젝트를 만든 경우 `vercel link`로 연결.

Project Settings → Environment Variables에 추가(Production·Preview 양쪽):

| 키 | 값 |
|---|---|
| `NEXT_PUBLIC_SUPABASE_URL` | https://xxx.supabase.co |
| `NEXT_PUBLIC_SUPABASE_ANON_KEY` | (anon key) |
| `NEXT_PUBLIC_API_URL` | https://your-app.fly.dev |
| `NEXT_PUBLIC_SENTRY_DSN_WEB` | (Sentry web DSN) |
| `NEXT_PUBLIC_POSTHOG_KEY` | (PostHog 프로젝트 키) |
| `NEXT_PUBLIC_POSTHOG_HOST` | https://us.i.posthog.com |
| `NEXT_PUBLIC_ENV` | production |
| `NEXT_PUBLIC_ENABLE_ADS` | false |

production 배포:
```bash
vercel --prod
```

Supabase Dashboard로 돌아가서 Auth → URL Configuration에 배포된 Vercel URL 등록.

---

## 5. Sentry 셋업

1. Sentry → 새 프로젝트 2개 생성 (Platform: Go, Next.js)
2. 각 프로젝트 DSN을 위 환경변수에 입력
3. 기본 동작 — 클라이언트·서버 양쪽에서 에러를 잡지만 **stack trace는 minified 상태**로 표시됨.
4. 소스맵 업로드(stack trace 정규화) 원하면 next.config.ts에 `withSentryConfig` 적용 + Vercel env에 `SENTRY_AUTH_TOKEN` 추가:
   ```bash
   cd apps/web
   npx @sentry/wizard@latest -i nextjs
   ```
   wizard가 next.config 자동 wrap + auth token 발급 안내. MVP 단계에서는 생략 가능(stack trace는 source location만 minified).

---

## 6. PostHog 셋업

1. PostHog → 프로젝트 생성 (region: US 또는 EU)
2. Project Settings → Project API Key 복사 → `NEXT_PUBLIC_POSTHOG_KEY`
3. Dashboard에서 첫 `$pageview` 이벤트 확인 (Vercel 배포 후 사이트 방문)

---

## 7. GitHub Actions CI/CD

`.github/workflows/` 에 3개 workflow가 정의돼 있습니다:
- `ci.yml` — 모든 PR/push에서 Go·Next.js lint + test + build (secrets 불필요)
- `deploy-api.yml` — master push 시 `apps/api/**` 변경되면 Fly 자동 배포
- `deploy-web.yml` — Vercel git integration 사용 시 비활성(권장), CLI 직접 배포 원하면 `if: false` 제거

### GitHub Secrets 등록
Repository → Settings → Secrets and variables → Actions → New repository secret:

| 키 | 값 | 사용처 |
|---|---|---|
| `FLY_API_TOKEN` | `flyctl tokens create deploy --name "github-actions"` 출력 | deploy-api.yml |
| `VERCEL_TOKEN` | Vercel → Settings → Tokens → Create | deploy-web.yml(optional) |
| `VERCEL_ORG_ID` | Vercel → Team Settings → General | deploy-web.yml(optional) |
| `VERCEL_PROJECT_ID` | Vercel → Project Settings → General | deploy-web.yml(optional) |

> `flyctl tokens create deploy`는 **배포 전용 scoped token**으로 권장. 옛 `flyctl auth token`은 personal access token(전체 권한)이라 노출 시 위험이 크다. 토큰 노출 의심 시 `flyctl tokens revoke <id>`.

### Vercel git integration (권장)
Vercel → Project → Settings → Git → "Connect Git Repository"로 GitHub repo 연결.
이후 master push → Production 배포, PR → Preview 배포가 자동. `deploy-web.yml`은 그대로 두고 `if: false` 유지.

---

## 8. 운영 점검

- `flyctl status` — 머신 상태
- `flyctl logs --tail` — 실시간 로그
- `flyctl ssh console` — 컨테이너 진입
- Vercel Analytics → 페이지 응답 시간
- Sentry → 에러 수신 확인 (`/v1/throw-test` 같은 임시 endpoint로 검증 가능)
- PostHog → 이벤트 스트림 + Live events

---

## 비용 (월간 추정, 사용자 ~100명 기준)

| 서비스 | 무료 한도 | 초과 시 |
|---|---|---|
| Supabase Free | 500MB DB, 1GB egress | $25/mo Pro |
| Fly.io | shared-cpu-1x × 3 무료 | 머신당 ~$2/mo |
| Vercel Hobby | 100GB 대역폭 | $20/mo Pro |
| Sentry Developer | 5K errors/mo | $26/mo Team |
| PostHog Free | 1M events/mo | usage-based |
| Anthropic | 사용량 기반 | Sonnet 4.6 $3/$15 per 1M tok |

MVP 단계(가입자 100명)는 무료 한도 내 충분히 운영 가능.
