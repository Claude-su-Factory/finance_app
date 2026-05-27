# 배포 & 로컬 셋업 가이드

이 문서 하나로 **로컬 개발**과 **프로덕션 배포** 모두 가능합니다.

구성:
- [Part A. 로컬에서 띄우기](#part-a-로컬에서-띄우기) — 처음 클론한 뒤 모든 기능을 로컬에서 검증
- [Part B. 프로덕션 배포](#part-b-프로덕션-배포) — Supabase + Fly + Vercel
- [Part C. CI/CD (GitHub Actions)](#part-c-cicd-github-actions)
- [Part D. 트러블슈팅](#part-d-트러블슈팅)
- [Part E. 비용 & 운영](#part-e-비용--운영)

---

# Part A. 로컬에서 띄우기

목표: 가입 → 온보딩 → 포트폴리오 추가 → AI 채팅까지 전부 동작.
소요: 처음 한 번 셋업 ~15분, 이후 부팅 ~30초.

## A-0. 도구 설치 (한 번만)

```bash
# Homebrew (macOS) 기준
brew install supabase/tap/supabase   # Supabase CLI
brew install go@1.25                  # pgx/v5 v5.9.2 요구
brew install node@20                  # Next.js 16
# (선택) Docker Desktop — Supabase CLI가 내부적으로 사용
```

확인:
```bash
supabase --version    # >= 2.0
go version            # go1.25.x
node --version        # v20.x
```

## A-1. 저장소 클론 + 의존성

```bash
git clone https://github.com/<your-account>/finance.git
cd finance

# Go API 의존성
cd apps/api
go mod download

# Next.js 의존성
cd ../web
npm install
cd ../..
```

## A-2. Supabase 로컬 부팅

```bash
supabase start
```

처음 실행 시 Docker 이미지 pull 때문에 5분 정도 걸립니다. 부팅 완료 후 출력에서 다음을 확인:

```
API URL:        http://127.0.0.1:54321
GraphQL URL:    http://127.0.0.1:54321/graphql/v1
DB URL:         postgresql://postgres:postgres@127.0.0.1:54322/postgres
Studio URL:     http://127.0.0.1:54323
Inbucket URL:   http://127.0.0.1:54324
JWT secret:     super-secret-jwt-token-with-at-least-32-characters-long
anon key:       eyJhbGc...
service_role:   eyJhbGc...
```

`supabase/migrations/`의 8개 마이그레이션이 자동 적용됩니다.

이후 키들이 필요할 때마다 다시 보려면:
```bash
supabase status                       # 사람이 읽는 형식
supabase status -o json | jq .        # 스크립트용
```

## A-3. 환경변수 파일 작성

### Next.js — `apps/web/.env.local`

```bash
cp apps/web/.env.example apps/web/.env.local
```

`apps/web/.env.local`을 열어서 `supabase status` 출력의 **anon key**를 채웁니다:

```
NEXT_PUBLIC_SUPABASE_URL=http://127.0.0.1:54321
NEXT_PUBLIC_SUPABASE_ANON_KEY=eyJhbGc...    # supabase status의 "anon key" 그대로 복붙
NEXT_PUBLIC_API_URL=http://localhost:8080
```

### Go API — `apps/api/.env`

`apps/api/.env`를 새로 만들고 다음 내용:

```bash
API_PORT=8080
API_ENV=development
DATABASE_URL=postgresql://postgres:postgres@127.0.0.1:54322/postgres
SUPABASE_JWT_SECRET=super-secret-jwt-token-with-at-least-32-characters-long
CORS_ORIGIN=http://localhost:3000

# 옵션 — 비어 있으면 해당 기능만 비활성. 다른 기능은 정상 동작.
ANTHROPIC_API_KEY=                # 빈 값이면 AI 채팅이 Mock으로 동작 (실제 응답 X, UI는 OK)
FRED_API_KEY=                     # 빈 값이면 미국 경제 지표 cron skip
ECOS_API_KEY=                     # 빈 값이면 한국 경제 지표 cron skip
SENTRY_DSN_API=                   # 빈 값이면 Sentry no-op
```

> Go API는 `.env` 파일을 자동으로 읽지 않습니다 — 부팅 시 `set -a; source .env; set +a` 패턴으로 export.
> A-4 명령에 이미 포함돼 있습니다.

## A-4. Go API 서버 부팅

별도 터미널에서:

```bash
cd apps/api
set -a; source .env; set +a
go run ./cmd/server
```

성공 로그:
```
{"time":"...", "level":"INFO", "msg":"sentry disabled (no DSN)"}
{"time":"...", "level":"INFO", "msg":"cron started", "jobs":7, "tz":"Asia/Seoul"}
{"time":"...", "level":"INFO", "msg":"API listening", "addr":":8080", "env":"development"}
```

확인:
```bash
curl http://localhost:8080/healthz   # → {"ok":true}
curl http://localhost:8080/readyz    # → {"db":"ok"}
```

## A-5. Next.js 부팅

또 별도 터미널에서:

```bash
cd apps/web
npm run dev
```

성공 로그: `▲ Next.js 16.x — Local: http://localhost:3000`

브라우저: http://localhost:3000

## A-6. 첫 가입 → 검증

1. `/auth/signup` → 이메일 + 비밀번호 + 약관 동의 → 가입
2. 이메일 인증 메일은 **Inbucket**에서 확인: http://127.0.0.1:54324 → 받은 메일 → verification link 클릭
3. 온보딩 wizard 3단계 진행 (기본 통화 → 브리핑 on/off → 보유 자산 추가)
4. `/app/home` 진입 — 사이드바·티커·6 카드 렌더링 확인
5. `/app/portfolio` → 보유 자산 추가 모달 → 종목 검색("삼성전자") → 저장
6. `/app/chat` → "내 포트폴리오 분석해줘" → SSE 스트리밍 응답 (Mock 또는 실 API)

## A-7. (선택) 5년 백필 데이터 적재

처음 가입 시 KOSPI·SPX 등 지표만 시드 데이터로 있고, 5년 prices/quotes는 비어 있습니다. 채팅의 `get_price_history` 도구가 빈 결과를 반환하면 백필 실행:

```bash
cd apps/api
set -a; source .env; set +a
go run ./cmd/backfill
```

소요 ~10분. KOSPI 30종목 + NASDAQ 30종목 + 환율 등 5년치 일봉이 들어갑니다.

## A-8. 로컬 통합 테스트 (선택)

```bash
cd apps/api
set -a; source .env; set +a
TEST_DATABASE_URL="$DATABASE_URL" go test -tags integration ./... -v
```

profile/handler·RLS 격리·`AsUser` 헬퍼 통합 테스트 9건이 PASS해야 정상.

## A-9. 종료

각 터미널 `Ctrl+C` 후:
```bash
supabase stop   # Docker 컨테이너 정지 (데이터는 유지)
# 데이터 초기화: supabase stop --no-backup
```

---

# Part B. 프로덕션 배포

목표: Supabase Cloud + Fly.io + Vercel에서 동일 기능 동작.
사전 가입(결제 카드 등록):
- [Supabase](https://supabase.com) — Free
- [Fly.io](https://fly.io) — 결제 카드 필수, $5/mo free credit
- [Vercel](https://vercel.com) — Hobby Free
- [Sentry](https://sentry.io) — Developer Free 5K events/mo (옵션)
- [PostHog](https://posthog.com) — Free 1M events/mo (옵션)
- [Anthropic](https://console.anthropic.com) — 결제 카드 필수
- [FRED](https://fred.stlouisfed.org/docs/api/api_key.html) — 무료 (한국 경제만 쓰면 skip)
- [ECOS](https://ecos.bok.or.kr/api/) — 무료 (미국 경제만 쓰면 skip)

CLI 설치:
```bash
brew install flyctl
npm i -g vercel
```

## B-1. Supabase 프로덕션 프로젝트

1. Dashboard → **New Project**
   - Name: `quotient-prod`
   - Region: `ap-northeast-2` (Seoul)
   - DB password 적어두기
2. 대시보드의 Project ref(`xxxxxxx`) 복사 보관

3. 로컬에서 마이그레이션 push:
```bash
supabase link --project-ref <xxxxxxx>
supabase db push    # supabase/migrations/ 8개 모두 적용
```

성공 출력: `Applying migration 20260522000001_profiles.sql...` × 8

4. Settings → API에서 다음 값 복사 보관:
   - **Project URL** → `NEXT_PUBLIC_SUPABASE_URL`
   - **anon public** 키 → `NEXT_PUBLIC_SUPABASE_ANON_KEY`
   - **service_role** 키 → `SUPABASE_SERVICE_ROLE_KEY` (서버 전용, 절대 노출 금지)
   - **JWT Settings** → "Legacy JWT Secret" 활성화 → `SUPABASE_JWT_SECRET`

5. Settings → Database → Connection string 탭:
   - **"Session pooler"**(포트 5432) 또는 **"Direct connection"** 복사 → `DATABASE_URL`
   - ⚠️ **"Transaction pooler"(포트 6543) 사용 금지** — pgxpool + 우리 `db.AsUser` LOCAL set과 비효율. Session pooler가 정합.

## B-2. (선택) Google OAuth 활성화

이메일 가입만 쓰려면 skip. Google 로그인 활성화하려면:
1. [Google Cloud Console](https://console.cloud.google.com) → APIs & Services → Credentials → **Create OAuth 2.0 Client**
   - Application type: Web
   - Authorized redirect URI: `https://<project-ref>.supabase.co/auth/v1/callback`
2. Client ID + Secret 복사
3. Supabase Dashboard → Authentication → Providers → Google → 활성화 + Client ID/Secret 입력

## B-3. Fly.io API 배포

```bash
cd apps/api
flyctl auth login    # 브라우저 로그인
```

앱 생성 (앱 이름은 글로벌 unique — 본인 식별자 포함):
```bash
flyctl apps create quotient-api-<your-id>   # 예: quotient-api-yhj
```

`apps/api/fly.toml`의 첫 줄을 본인 앱 이름으로 변경:
```toml
app = "quotient-api-yhj"   # ← 위에서 만든 이름
```

> CORS_ORIGIN은 Vercel 도메인이 확정된 후(B-4) 채워야 하므로 일단 placeholder.

시크릿 등록 (한 번만 — `DATABASE_URL`은 Session pooler 포트 5432):
```bash
flyctl secrets set \
  DATABASE_URL="postgresql://postgres.<ref>:<db-pw>@aws-0-ap-northeast-2.pooler.supabase.com:5432/postgres" \
  SUPABASE_JWT_SECRET="<legacy-jwt-secret>" \
  ANTHROPIC_API_KEY="<sk-ant-...>" \
  FRED_API_KEY="<key>" \
  ECOS_API_KEY="<key>" \
  SENTRY_DSN_API="<https://...@sentry.io/...>" \
  CORS_ORIGIN="http://localhost:3000"
```

> 옵션 키(`ANTHROPIC_API_KEY`, `FRED_API_KEY`, `ECOS_API_KEY`, `SENTRY_DSN_API`)는 빈 값으로 두면 해당 기능만 비활성.

배포:
```bash
flyctl deploy
```

성공 출력 끝부분:
```
✓ Machine ... started
Visit your newly deployed app at https://quotient-api-<your-id>.fly.dev/
```

검증:
```bash
flyctl logs                                              # 실시간 로그
curl https://quotient-api-<your-id>.fly.dev/healthz      # → {"ok":true}
curl https://quotient-api-<your-id>.fly.dev/readyz       # → {"db":"ok"}
```

## B-4. Vercel Next.js 배포

```bash
cd apps/web
vercel    # 첫 실행: 프로젝트 생성 prompt
```

Prompt 진행:
- Set up and deploy? → **Y**
- Which scope? → 본인 계정/팀
- Link to existing project? → **N**
- Project name → `quotient-web` (또는 원하는 이름)
- In which directory is your code located? → `./` (현재 디렉터리가 apps/web이라 OK)
- 자동 감지된 Next.js 설정 → **Y**

> Vercel Dashboard에서 import로 생성하는 경우 **Root Directory를 `apps/web`으로 명시 지정**해야 빌드 성공 (monorepo).

Project Settings → Environment Variables에 다음을 **Production + Preview** 양쪽에 추가:

| 키 | 값 |
|---|---|
| `NEXT_PUBLIC_SUPABASE_URL` | `https://<ref>.supabase.co` |
| `NEXT_PUBLIC_SUPABASE_ANON_KEY` | (Supabase anon key) |
| `NEXT_PUBLIC_API_URL` | `https://quotient-api-<your-id>.fly.dev` |
| `NEXT_PUBLIC_SENTRY_DSN_WEB` | (Sentry web DSN — Part B-5 후 추가) |
| `NEXT_PUBLIC_POSTHOG_KEY` | (PostHog key — Part B-6 후 추가) |
| `NEXT_PUBLIC_POSTHOG_HOST` | `https://us.i.posthog.com` |
| `NEXT_PUBLIC_ENV` | `production` |
| `NEXT_PUBLIC_ENABLE_ADS` | `false` |

Production 배포:
```bash
vercel --prod
```

배포된 URL 확인: `https://quotient-web-<random>.vercel.app` 또는 본인이 설정한 도메인.

## B-5. Fly CORS 갱신 (B-3 ↔ B-4 연결)

이제 Vercel 도메인이 확정됐으니 Fly에 등록:

```bash
cd apps/api
flyctl secrets set CORS_ORIGIN="https://quotient-web-<your-id>.vercel.app"
# secret 변경 시 Fly가 자동으로 머신 재시작.
```

> Vercel preview URL도 허용하려면 와일드카드 패턴 추가:
> ```bash
> flyctl secrets set CORS_ORIGIN="https://quotient.example.com,https://quotient-web-*-myteam.vercel.app"
> ```
> 우리 CORS 미들웨어가 콤마 다중 origin + 와일드카드(`*` 한 개) 지원.

## B-6. Supabase Auth Site URL 등록

Supabase Dashboard → Authentication → URL Configuration:
- **Site URL**: `https://quotient-web-<your-id>.vercel.app`
- **Redirect URLs** (추가): `https://quotient-web-<your-id>.vercel.app/**`

이 단계를 빼먹으면 OAuth/이메일 인증 후 redirect가 깨집니다.

## B-7. Sentry 셋업 (옵션)

1. Sentry → 새 프로젝트 2개 생성
   - Platform: **Go** → 이름 `quotient-api`
   - Platform: **Next.js** → 이름 `quotient-web`
2. 각 프로젝트의 DSN 복사
3. `quotient-api`의 DSN을 Fly에 등록:
   ```bash
   flyctl secrets set SENTRY_DSN_API="https://...@o....ingest.sentry.io/..."
   ```
4. `quotient-web`의 DSN을 Vercel env에 `NEXT_PUBLIC_SENTRY_DSN_WEB`로 추가 → Redeploy
5. 기본 동작 — 에러는 모두 잡히지만 **클라이언트 stack trace는 minified 상태**.
   소스맵 업로드(stack trace 정규화) 원하면:
   ```bash
   cd apps/web
   npx @sentry/wizard@latest -i nextjs
   ```
   wizard가 next.config.ts wrap + auth token 발급 안내. MVP에서는 건너뛰어도 무방.

## B-8. PostHog 셋업 (옵션)

1. PostHog → 프로젝트 생성 (region: **US** 또는 **EU**)
2. Project Settings → **Project API Key** 복사
3. Vercel env에 `NEXT_PUBLIC_POSTHOG_KEY` 추가 → Redeploy
4. 사이트 접속 후 PostHog Live events에 `$pageview` 도착 확인

## B-9. 5년 백필 (운영 데이터)

운영 컨테이너 안에서 일회성 실행 (Dockerfile에 `backfill` 바이너리 포함됨):

```bash
flyctl ssh console -C "/app/backfill"
```

또는 로컬에서 production DSN으로:
```bash
DATABASE_URL="<prod-session-pooler-dsn>" go run ./cmd/backfill
```

소요 ~10분, Yahoo·KIND·frankfurter API 호출 ~수만 건.

## B-10. 운영 스모크 검증

`docs/E2E_SMOKE.md` 9단계 체크리스트를 production 환경에서 수동 통과.

검증 통과 후 `docs/STATUS.md`의 "최근 변경 이력"에 한 줄 추가:
```markdown
- YYYY-MM-DD Production 스모크 PASS — 첫 배포
```

---

# Part C. CI/CD (GitHub Actions)

`.github/workflows/` 3개 파일이 이미 정의돼 있습니다:

- **`ci.yml`** — 모든 PR·push에서 Go·Next.js lint + test + build (secrets 불필요)
- **`deploy-api.yml`** — master push 시 `apps/api/**` 변경되면 Fly 자동 배포
- **`deploy-web.yml`** — Vercel git integration 사용 시 비활성(권장), CLI 배포 원하면 `if: false` 제거

## C-1. GitHub Secrets 등록

Repository → Settings → Secrets and variables → Actions → **New repository secret**:

| 키 | 발급 명령 / 위치 | 사용처 |
|---|---|---|
| `FLY_API_TOKEN` | `flyctl tokens create deploy --name "github-actions"` | deploy-api.yml |
| `VERCEL_TOKEN` | Vercel → Settings → Tokens → Create | deploy-web.yml(optional) |
| `VERCEL_ORG_ID` | Vercel → Team Settings → General | deploy-web.yml(optional) |
| `VERCEL_PROJECT_ID` | Vercel → Project Settings → General | deploy-web.yml(optional) |

> `flyctl tokens create deploy`는 **배포 전용 scoped token**. 옛 `flyctl auth token`은 personal access token(전권한)이라 노출 위험. 토큰 노출 시 즉시 `flyctl tokens revoke <id>`.

## C-2. Vercel git integration (권장)

Vercel → Project → Settings → Git → **Connect Git Repository**로 GitHub repo 연결.

이후:
- master push → Production 배포 자동
- PR → Preview 배포 자동 (PR URL에 코멘트로 자동 게시)

`deploy-web.yml`은 `if: false` 그대로 유지. CLI 워크플로우는 git integration이 동작 안 할 때만 비상용.

---

# Part D. 트러블슈팅

## D-1. 로컬

**`supabase start` 실패 — "port 54322 already in use"**
- 이전 Supabase 인스턴스가 살아 있음. `supabase stop` 후 재시도.
- 다른 Postgres가 점유 중이면: `lsof -i :54322` → 해당 프로세스 종료.

**`supabase status`에 JWT secret이 안 보임**
- CLI v2.x부터 일부 환경에서 가려질 수 있음. 로컬 기본값 사용:
  `SUPABASE_JWT_SECRET="super-secret-jwt-token-with-at-least-32-characters-long"`

**Go API `go run` 시 `config load failed: SUPABASE_JWT_SECRET required`**
- `.env`가 export 안 됨. `set -a; source .env; set +a; go run ./cmd/server` 패턴 그대로 실행.

**Next.js dev에서 `Invalid API URL`**
- `.env.local`의 `NEXT_PUBLIC_API_URL` 확인. 끝에 `/` 붙이지 말 것.
- 변경 후 `npm run dev` 재시작 필요(Next.js는 .env.local 변경을 hot-reload 안 함).

**이메일 인증 메일이 안 옴**
- 로컬은 **Inbucket** 사용: http://127.0.0.1:54324
- "Authentication" tab에서 가입 시 사용한 이메일 → verification link 클릭.

**Chat에서 "복잡한 질문이라 충분히 답하지 못했습니다" 에러만**
- `ANTHROPIC_API_KEY`가 비어 있어 MockClient 사용 중 — 정상.
- 실 응답 원하면 키 발급 후 `.env`에 추가, API 서버 재부팅.

## D-2. Supabase 운영

**`supabase db push`가 "migration already exists" 에러**
- 이미 일부 적용된 상태. `supabase migration repair --status applied <version>` 또는 `supabase db reset --linked` (⚠️ 데이터 손실).

**Connection refused — `aws-0-ap-northeast-2.pooler.supabase.com:5432`**
- region이 다른 경우 host가 다름. Dashboard → Settings → Database → Connection string에서 정확한 host 복사.

**`pq: password authentication failed for user "postgres"`**
- DB password에 특수문자가 있으면 URL encode 필요. `@` → `%40`, `#` → `%23` 등.

## D-3. Fly.io

**`flyctl apps create`에서 "name not available"**
- 글로벌 unique라 이미 사용 중. 다른 이름 시도. `fly.toml`의 `app =` 줄도 함께 수정.

**Deploy 후 머신이 계속 restart**
- `flyctl logs`에서 첫 에러 확인.
- 가장 흔한 원인 3가지:
  1. `DATABASE_URL` 잘못됨 — Session pooler(5432) 확인
  2. `SUPABASE_JWT_SECRET` 누락 → 401 직후 health check fail
  3. Supabase 프로젝트가 paused (Free tier 7일 미사용) — Dashboard에서 unpause

**`SSL connection required` 에러**
- DSN 끝에 `?sslmode=require` 추가:
  `postgresql://...:5432/postgres?sslmode=require`

**`flyctl ssh console`이 timeout**
- 머신이 stopped 상태. `flyctl machine list`로 확인 후 `flyctl machine start <id>`.

## D-4. Vercel

**Build fails — "Module not found: Can't resolve '@/...'"**
- **Root Directory** 설정이 빠짐. Project Settings → General → Root Directory → `apps/web` 지정.

**Build fails — `Type error: Cannot find module 'posthog-js'`**
- `apps/web/package.json`에 dependency가 있는데 install 안 됨. Vercel Dashboard → Settings → Build & Development → Install Command 확인 (`npm install` 또는 자동).

**Production은 동작하는데 Preview에서 CORS error**
- Fly의 `CORS_ORIGIN`이 Preview URL을 허용 안 함. 와일드카드 패턴 추가:
  ```bash
  flyctl secrets set CORS_ORIGIN="https://your-prod-domain.com,https://quotient-web-*-team.vercel.app"
  ```

**OAuth callback이 `localhost`로 redirect**
- Supabase → Authentication → URL Configuration에서 **Site URL**과 **Redirect URLs**가 Vercel 도메인으로 설정됐는지 확인. 둘 다 등록 필수.

## D-5. AI / Anthropic

**`401 unauthorized` from Anthropic**
- API 키 형식 확인 (`sk-ant-api03-...`).
- 결제 카드 등록 안 됨 — Console → Billing.
- 키 발급 직후 ~수 분 propagation 지연 가능.

**Chat 응답이 갑자기 멈추고 disconnect**
- SSE keepalive timeout. nginx/proxy 뒤에 있으면 `X-Accel-Buffering: no` 헤더 + `proxy_read_timeout` 늘리기. Fly는 기본 OK.

## D-6. RLS / DB

**`new row violates row-level security policy`**
- 사용자 컨텍스트(`db.AsUser`) 없이 INSERT 시도. 코드 trace 필요.
- 디버깅: `set_config('request.jwt.claim.sub', '<uid>', true)` 후 INSERT 재시도 — 성공하면 wrap 누락.

**`current setting "request.jwt.claims" is not set`**
- 트랜잭션 밖에서 read를 호출. `db.AsUser`로 wrap 필요. Public 도구는 `RequiresUserContext()` false인지 확인.

---

# Part E. 비용 & 운영

## E-1. 월간 추정 비용 (가입자 ~100명)

| 서비스 | 무료 한도 | 초과 시 |
|---|---|---|
| Supabase Free | 500MB DB, 1GB egress, 50K MAU | $25/mo Pro |
| Fly.io | 결제카드 + $5/mo free credit | shared-cpu-1x 머신 ~$2/mo |
| Vercel Hobby | 100GB 대역폭 | $20/mo Pro |
| Sentry Developer | 5K errors/mo | $26/mo Team |
| PostHog Free | 1M events/mo | usage-based |
| Anthropic | 사용량 기반 | Sonnet 4.6 $3/$15 per 1M tok |

MVP(가입자 100명, 일평균 PV 500)는 무료 한도 + Anthropic 사용량 수 달러로 운영 가능.

## E-2. 일상 운영 체크

```bash
# Fly
flyctl status                    # 머신 상태
flyctl logs                      # 실시간 로그 (streaming default)
flyctl ssh console               # 컨테이너 진입
flyctl secrets list              # 등록된 시크릿 이름 (값은 표시 안 됨)

# Supabase
supabase db diff --linked        # 로컬 vs 운영 스키마 diff
```

대시보드:
- Fly: `https://fly.io/apps/<app-name>`
- Vercel: `https://vercel.com/<team>/<project>`
- Supabase: `https://supabase.com/dashboard/project/<ref>`
- Sentry: 본인 org → Issues
- PostHog: Live events + Insights

## E-3. 비상시 롤백

**Fly**:
```bash
flyctl releases                  # 배포 이력
flyctl releases rollback <ver>   # 특정 release로 복원
```

**Vercel**: Dashboard → Deployments → 이전 배포 → ⋯ → **Promote to Production**

**Supabase**: 마이그레이션 revert는 위험 — 백업 복원이 안전. Dashboard → Database → Backups.

## E-4. 시크릿 노출 시

- Fly: `flyctl secrets unset <KEY>` 후 새 값으로 `flyctl secrets set` (자동 재배포)
- Vercel: Dashboard → Settings → Environment Variables → 해당 키 제거 후 새 값
- GitHub Secrets: Repository → Settings → Secrets → 해당 키 제거 후 새 값
- Fly token: `flyctl tokens list` → `flyctl tokens revoke <id>`
- Supabase: Settings → API → Reset keys (anon은 reset 시 모든 클라이언트 재배포 필요)

---

추가 막힘이나 환경별 변수 발견 시 본 문서 Part D에 케이스 추가.
