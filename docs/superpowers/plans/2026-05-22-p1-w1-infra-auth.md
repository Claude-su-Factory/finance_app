# Quotient W1 — 인프라·인증·온보딩 구현 계획

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:subagent-driven-development` (recommended) or `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Supabase·Go·Next.js 인프라 셋업 + 가입/로그인/이메일 인증/온보딩 wizard. W1 종료 시점에 사용자가 신규 가입 → 이메일 인증 → 온보딩 wizard 완주 → 빈 앱 셸(`/app`)에 진입 가능.

**Architecture:** 분리형 모노레포. `apps/api`(Go + Chi + sqlc) + `apps/web`(Next.js 15 App Router) + `supabase/`(SQL 마이그레이션). Supabase Auth가 JWT 발급, Next.js와 Go 양쪽이 검증. RLS로 DB 레벨 권한 강제. Sentry·PostHog 관측, GitHub Actions CI/CD, Fly·Vercel 배포.

**Tech Stack:** Go 1.23 + Chi v5 + slog + sqlc + jwx, Next.js 15 + TypeScript 5 + Tailwind 3 + shadcn/ui + Motion + Lucide, Supabase Postgres + Auth, Fly.io + Vercel, Sentry + PostHog, GitHub Actions.

**참고 스펙:** [`docs/superpowers/specs/2026-05-22-quotient-mvp-design.md`](../specs/2026-05-22-quotient-mvp-design.md) 섹션 3, 6, 7, 8.

---

## File Structure (W1 생성 파일)

```
finance/
├── apps/
│   ├── api/
│   │   ├── cmd/server/main.go              # Go 진입점
│   │   ├── internal/
│   │   │   ├── config/config.go            # 환경설정 (envconfig)
│   │   │   ├── db/db.go                    # pgx 연결 풀
│   │   │   ├── auth/jwt.go                 # Supabase JWKS JWT 검증
│   │   │   ├── auth/jwt_test.go
│   │   │   ├── middleware/auth.go          # 인증 미들웨어
│   │   │   ├── middleware/cors.go
│   │   │   ├── middleware/logger.go        # slog 요청 로깅
│   │   │   ├── handlers/health.go          # /healthz, /readyz
│   │   │   ├── handlers/profiles.go        # /v1/profiles GET·PATCH
│   │   │   ├── handlers/profiles_test.go
│   │   │   ├── models/profile.go
│   │   │   ├── obs/sentry.go               # Sentry 초기화
│   │   │   └── router/router.go
│   │   ├── go.mod
│   │   ├── go.sum
│   │   ├── Dockerfile
│   │   ├── fly.toml
│   │   └── .air.toml                       # 핫리로드
│   └── web/
│       ├── app/
│       │   ├── layout.tsx                  # 루트 레이아웃 (다크 기본)
│       │   ├── globals.css                 # Tailwind + 블룸버그 토큰
│       │   ├── page.tsx                    # 랜딩
│       │   ├── pricing/page.tsx            # 가격 (정적)
│       │   ├── login/page.tsx
│       │   ├── signup/page.tsx
│       │   ├── verify-email/page.tsx
│       │   ├── forgot-password/page.tsx
│       │   ├── reset-password/page.tsx
│       │   ├── privacy/page.tsx
│       │   ├── terms/page.tsx
│       │   ├── auth/callback/route.ts      # OAuth 콜백
│       │   ├── app/
│       │   │   ├── layout.tsx              # 인증 셸 (사이드바·티커·상태바)
│       │   │   ├── page.tsx                # 홈 placeholder
│       │   │   └── onboarding/page.tsx     # wizard (3단계)
│       │   └── error.tsx                   # 500 페이지
│       ├── components/
│       │   ├── ui/                         # shadcn (button, input, label, card, ...)
│       │   ├── shell/Sidebar.tsx
│       │   ├── shell/TopTicker.tsx         # placeholder (W2 후 동작)
│       │   ├── shell/StatusBar.tsx
│       │   ├── shell/AppShell.tsx
│       │   ├── auth/SignupForm.tsx
│       │   ├── auth/LoginForm.tsx
│       │   ├── auth/GoogleButton.tsx
│       │   ├── auth/AuthCard.tsx
│       │   ├── onboarding/Wizard.tsx
│       │   ├── onboarding/StepIndicator.tsx
│       │   ├── onboarding/CurrencyStep.tsx
│       │   └── onboarding/DemoOrStartStep.tsx
│       ├── lib/
│       │   ├── supabase/client.ts          # 브라우저 클라이언트
│       │   ├── supabase/server.ts          # SSR 클라이언트
│       │   ├── supabase/middleware.ts      # 세션 갱신
│       │   ├── api/client.ts               # Go API fetch wrapper
│       │   ├── utils/cn.ts
│       │   └── obs/posthog.tsx             # PostHog provider
│       ├── middleware.ts                   # Next.js 미들웨어 (라우트 가드)
│       ├── next.config.mjs
│       ├── tailwind.config.ts
│       ├── postcss.config.mjs
│       ├── tsconfig.json
│       ├── package.json
│       ├── components.json                 # shadcn 설정
│       ├── vitest.config.ts
│       └── playwright.config.ts            # (W6 사용, 자리만)
├── supabase/
│   ├── migrations/
│   │   ├── 20260522000001_profiles.sql
│   │   └── 20260522000002_rls_profiles.sql
│   ├── seed.sql                            # (비어있음, W2에서 사용)
│   └── config.toml
├── .github/
│   └── workflows/
│       ├── ci.yml                          # lint·test·build
│       └── deploy.yml                      # staging·prod 자동 배포
├── .gitignore
├── .env.example                            # 모든 환경변수 키만
├── README.md                               # 개발 가이드
└── Makefile                                # 자주 쓰는 명령 alias
```

---

## 외부 셋업 (사용자 직접 수행 — Task 0)

**이 작업들은 Task 시작 전에 사용자가 미리 완료해야 합니다.** 각 단계의 출력값을 `.env.local` (Next.js) 와 Fly secrets 에 보관.

- [ ] **A. Supabase 프로젝트 생성** (Free 티어)
  - [supabase.com](https://supabase.com) → New project
  - **W1에는 `quotient-staging` 1개만 생성**. `quotient-prod`는 W6 출시 직전 생성 (초기에 prod 자격증명이 로컬에 있으면 사고 위험)
  - 리전 `Northeast Asia (Seoul)`
  - **JWT 설정 확인**: Project Settings → API → JWT Settings. 2024-10 이후 신규 프로젝트는 비대칭 키(ES256/JWKS) 기본이며 HS256 공유 secret이 없음. plan은 HS256 가정이므로 **"Legacy JWT Secret" 활성화** 또는 W2에서 JWKS 기반 검증으로 마이그레이션 (백로그 추가). HS256으로 갈 경우 dashboard에서 secret 값을 복사
  - staging 프로젝트의 다음 값을 `.env.local`·Fly secrets에 보관:
    - `NEXT_PUBLIC_SUPABASE_URL`
    - `NEXT_PUBLIC_SUPABASE_ANON_KEY`
    - `SUPABASE_SERVICE_ROLE_KEY` (Go API 전용, 클라이언트 노출 금지)
    - `SUPABASE_JWT_SECRET` (Legacy 활성화 후 복사)

- [ ] **B. Fly.io 가입 + CLI 설치**
  - `brew install flyctl`
  - `fly auth signup`
  - `fly apps create quotient-api-prod --org personal`
  - `fly apps create quotient-api-staging --org personal`

- [ ] **C. Vercel 가입 + CLI 설치**
  - `npm install -g vercel`
  - `vercel login`
  - 프로젝트는 첫 배포 시 자동 생성

- [ ] **D. GitHub 저장소 생성**
  - `gh repo create quotient --private --source=. --remote=origin`
  - (혹은 dashboard에서 생성 후 remote 추가)

- [ ] **E. Sentry 가입 + 프로젝트 2개**
  - [sentry.io](https://sentry.io) → 새 프로젝트
  - 프로젝트 1: `quotient-web` (Next.js)
  - 프로젝트 2: `quotient-api` (Go)
  - 각 프로젝트 DSN을 환경변수에 보관:
    - `NEXT_PUBLIC_SENTRY_DSN_WEB`
    - `SENTRY_DSN_API`

- [ ] **F. PostHog 가입**
  - [posthog.com](https://posthog.com/) → 새 프로젝트
  - `NEXT_PUBLIC_POSTHOG_KEY`, `NEXT_PUBLIC_POSTHOG_HOST`

- [ ] **G. Google OAuth 클라이언트**
  - Google Cloud Console → OAuth 동의 화면 → OAuth 2.0 클라이언트 ID
  - Authorized redirect URI: `https://<프로젝트>.supabase.co/auth/v1/callback`
  - Client ID·Secret을 Supabase Dashboard → Authentication → Providers → Google 에 입력

- [ ] **H. 도메인 등록 (선택, W6 전까지 가능)**
  - 추천: `quotient.kr` 또는 `getquotient.com`

위 단계가 완료되면 모든 secrets를 `.env.local` (개발) + Fly secrets (`fly secrets set KEY=VALUE -a quotient-api-prod`) + Vercel env (대시보드)에 입력.

---

## Task 1: 모노레포 구조 + 기본 설정

**Files:**
- Create: `.gitignore`
- Create: `.env.example`
- Create: `README.md`
- Create: `Makefile`
- Create: `apps/api/`, `apps/web/`, `supabase/` 빈 디렉토리

- [ ] **Step 1: 디렉토리 구조 생성**

```bash
mkdir -p apps/api/cmd/server apps/api/internal/{config,db,auth,middleware,handlers,models,obs,router}
mkdir -p apps/web/{app,components,lib}
mkdir -p supabase/migrations
mkdir -p .github/workflows
```

- [ ] **Step 2: `.gitignore` 작성**

Create `.gitignore`:
```
# Node
node_modules/
.next/
out/
.vercel/

# Go
apps/api/server
apps/api/tmp/
*.exe

# Env
.env
.env.local
.env*.local

# IDE
.vscode/
.idea/
.DS_Store

# Testing
coverage/
*.out

# Supabase
supabase/.branches/
supabase/.temp/
```

- [ ] **Step 3: `.env.example` 작성**

Create `.env.example`:
```
# Supabase
NEXT_PUBLIC_SUPABASE_URL=https://xxx.supabase.co
NEXT_PUBLIC_SUPABASE_ANON_KEY=
SUPABASE_SERVICE_ROLE_KEY=
SUPABASE_JWT_SECRET=

# Go API
API_PORT=8080
API_ENV=development
DATABASE_URL=postgresql://postgres:postgres@localhost:54322/postgres
CORS_ORIGIN=http://localhost:3000

# 관측
NEXT_PUBLIC_SENTRY_DSN_WEB=
SENTRY_DSN_API=
NEXT_PUBLIC_POSTHOG_KEY=
NEXT_PUBLIC_POSTHOG_HOST=https://us.i.posthog.com

# 기능 플래그
PAYMENTS_ENABLED=false
ENABLE_ADS=false
```

- [ ] **Step 4: `Makefile` 작성**

Create `Makefile`:
```makefile
.PHONY: dev api web db-up db-down db-reset migrate test lint

dev:
	@echo "Run 'make api' and 'make web' in separate terminals"

api:
	cd apps/api && air

web:
	cd apps/web && npm run dev

db-up:
	cd supabase && supabase start

db-down:
	cd supabase && supabase stop

db-reset:
	cd supabase && supabase db reset

migrate:
	cd supabase && supabase db push

test:
	cd apps/api && go test ./...
	cd apps/web && npm test

lint:
	cd apps/api && golangci-lint run
	cd apps/web && npm run lint
```

- [ ] **Step 5: `README.md` 작성 (개발 환경 가이드)**

Create `README.md`:
```markdown
# Quotient — Portfolio Intelligence Terminal

개인 운영 금융 SaaS. 한국·미국 자산 통합 분석 + 자연어 분석가 인터페이스.

[설계 문서 →](docs/superpowers/specs/2026-05-22-quotient-mvp-design.md)

## 로컬 개발

### 사전 요구
- Go 1.23+
- Node 20+
- Supabase CLI: `brew install supabase/tap/supabase`
- Air (Go 핫리로드): `go install github.com/air-verse/air@latest`

### 셋업
1. `.env.example`을 `.env.local`로 복사하고 값 채우기
2. `make db-up` — 로컬 Supabase 띄우기
3. `make migrate` — 마이그레이션 실행
4. 두 터미널: `make api`, `make web`
5. http://localhost:3000 접속

### 문서
- [STATUS](docs/STATUS.md), [ROADMAP](docs/ROADMAP.md), [ARCHITECTURE](docs/ARCHITECTURE.md), [AGENTS](docs/AGENTS.md)
```

- [ ] **Step 6: git 초기화 + 첫 커밋**

```bash
git init
git add -A
git commit -m "chore: 모노레포 구조 + 기본 설정 추가"
```

---

## Task 2: Supabase 로컬 + 첫 마이그레이션 (profiles)

**Files:**
- Create: `supabase/config.toml`
- Create: `supabase/migrations/20260522000001_profiles.sql`

- [ ] **Step 1: Supabase 로컬 초기화**

```bash
cd supabase
supabase init
# config.toml 생성됨
```

`config.toml`을 편집해 포트 고정 (Fly 등 다른 서비스와 충돌 방지):
```toml
[api]
port = 54321

[db]
port = 54322

[studio]
port = 54323
```

- [ ] **Step 2: `profiles` 마이그레이션 작성**

Create `supabase/migrations/20260522000001_profiles.sql`:
```sql
-- profiles 테이블: auth.users와 1:1
create table public.profiles (
  id uuid primary key references auth.users(id) on delete cascade,
  display_name text,
  base_currency text not null default 'KRW' check (base_currency in ('KRW', 'USD')),
  ui_intensity text not null default 'standard' check (ui_intensity in ('vivid', 'standard', 'subtle')),
  onboarding_completed boolean not null default false,
  daily_briefing_enabled boolean not null default true,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

-- auth.users insert 시 profiles row 자동 생성
create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
begin
  insert into public.profiles (id, display_name)
  values (new.id, coalesce(new.raw_user_meta_data->>'name', split_part(new.email, '@', 1)));
  return new;
end;
$$;

create trigger on_auth_user_created
  after insert on auth.users
  for each row execute function public.handle_new_user();

-- updated_at 자동 갱신
create or replace function public.touch_updated_at()
returns trigger language plpgsql as $$
begin
  new.updated_at = now();
  return new;
end $$;

create trigger profiles_touch
  before update on public.profiles
  for each row execute function public.touch_updated_at();
```

- [ ] **Step 3: 마이그레이션 적용·확인**

```bash
cd supabase
supabase start
supabase db push
```

Expected: `profiles` 테이블 생성 확인 `supabase db diff` 시 변경 없음.

검증:
```bash
psql "postgresql://postgres:postgres@localhost:54322/postgres" -c "\d public.profiles"
```
Expected: 7개 컬럼 + 트리거 2개 확인.

- [ ] **Step 4: 커밋**

```bash
git add supabase/
git commit -m "feat(db): profiles 테이블 + 사용자 가입 트리거"
```

---

## Task 3: profiles RLS 정책

**Files:**
- Create: `supabase/migrations/20260522000002_rls_profiles.sql`

- [ ] **Step 1: RLS 마이그레이션 작성**

Create `supabase/migrations/20260522000002_rls_profiles.sql`:
```sql
-- RLS 활성화
alter table public.profiles enable row level security;

-- 본인 row 조회
create policy "profiles_select_own"
  on public.profiles for select
  using (auth.uid() = id);

-- 본인 row 갱신
create policy "profiles_update_own"
  on public.profiles for update
  using (auth.uid() = id)
  with check (auth.uid() = id);

-- INSERT는 트리거가 처리하므로 정책 없음 (service_role만 가능)
-- DELETE도 정책 없음 (계정 삭제는 별도 Admin API 흐름)
```

- [ ] **Step 2: 적용**

```bash
cd supabase
supabase db push
```

- [ ] **Step 3: 정책 동작 검증 (psql)**

```bash
psql "postgresql://postgres:postgres@localhost:54322/postgres" <<EOF
select polname, polcmd from pg_policy where polrelid = 'public.profiles'::regclass;
EOF
```
Expected: `profiles_select_own SELECT`, `profiles_update_own UPDATE` 출력.

- [ ] **Step 4: 커밋**

```bash
git add supabase/
git commit -m "feat(db): profiles RLS 정책 (본인 read·update만)"
```

---

## Task 4: Go 백엔드 스캐폴딩 + healthz

**Files:**
- Create: `apps/api/go.mod`
- Create: `apps/api/cmd/server/main.go`
- Create: `apps/api/internal/config/config.go`
- Create: `apps/api/internal/router/router.go`
- Create: `apps/api/internal/handlers/health.go`
- Create: `apps/api/internal/handlers/health_test.go`
- Create: `apps/api/.air.toml`

- [ ] **Step 1: Go 모듈 초기화 + 의존성**

```bash
cd apps/api
go mod init github.com/<user>/quotient/apps/api
go get github.com/go-chi/chi/v5
go get github.com/caarlos0/env/v10
go get github.com/stretchr/testify
```

- [ ] **Step 2: `config.go` 작성**

Create `apps/api/internal/config/config.go`:
```go
package config

import "github.com/caarlos0/env/v10"

type Config struct {
	Port             int    `env:"API_PORT" envDefault:"8080"`
	Env              string `env:"API_ENV" envDefault:"development"`
	DatabaseURL      string `env:"DATABASE_URL,required"`
	SupabaseJWTSecret string `env:"SUPABASE_JWT_SECRET,required"`
	CORSOrigin       string `env:"CORS_ORIGIN" envDefault:"http://localhost:3000"`
	SentryDSN        string `env:"SENTRY_DSN_API"`
}

func Load() (*Config, error) {
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
```

- [ ] **Step 3: 헬스 핸들러 실패 테스트 작성**

Create `apps/api/internal/handlers/health_test.go`:
```go
package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthz_OK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	Healthz(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"status":"ok"}`, rec.Body.String())
}
```

- [ ] **Step 4: 테스트 실패 확인**

```bash
cd apps/api
go test ./internal/handlers/...
```
Expected: FAIL — `Healthz` undefined.

- [ ] **Step 5: 헬스 핸들러 구현**

Create `apps/api/internal/handlers/health.go`:
```go
package handlers

import (
	"encoding/json"
	"net/http"
)

func Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
```

- [ ] **Step 6: 테스트 통과 확인**

```bash
go test ./internal/handlers/...
```
Expected: PASS.

- [ ] **Step 7: 라우터 작성**

Create `apps/api/internal/router/router.go`:
```go
package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/<user>/quotient/apps/api/internal/handlers"
)

func New() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/healthz", handlers.Healthz)
	return r
}
```

- [ ] **Step 8: main 작성**

Create `apps/api/cmd/server/main.go`:
```go
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/<user>/quotient/apps/api/internal/config"
	"github.com/<user>/quotient/apps/api/internal/router"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load failed", "err", err)
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           router.New(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		logger.Info("API listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen failed", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown failed", "err", err)
	}
}
```

- [ ] **Step 9: 로컬 실행 확인**

```bash
cd apps/api
DATABASE_URL=postgresql://postgres:postgres@localhost:54322/postgres \
SUPABASE_JWT_SECRET=test-secret \
go run ./cmd/server/main.go
```

별도 터미널:
```bash
curl http://localhost:8080/healthz
```
Expected: `{"status":"ok"}`

- [ ] **Step 10: air 설정 (핫리로드)**

```bash
go install github.com/air-verse/air@latest
```

Create `apps/api/.air.toml`:
```toml
[build]
cmd = "go build -o ./tmp/server ./cmd/server"
bin = "./tmp/server"
include_ext = ["go"]
exclude_dir = ["tmp", "vendor"]
```

- [ ] **Step 11: 커밋**

```bash
cd ../..
git add apps/api/
git commit -m "feat(api): Go 백엔드 스캐폴딩 + /healthz"
```

---

## Task 5: JWT 검증 미들웨어

**Files:**
- Create: `apps/api/internal/auth/jwt.go`
- Create: `apps/api/internal/auth/jwt_test.go`
- Create: `apps/api/internal/middleware/auth.go`
- Modify: `apps/api/internal/router/router.go`

- [ ] **Step 1: jwx 의존성 추가**

```bash
cd apps/api
go get github.com/lestrrat-go/jwx/v2
```

- [ ] **Step 2: JWT 검증 실패 테스트**

Create `apps/api/internal/auth/jwt_test.go`:
```go
package auth

import (
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "super-secret-key"

func makeToken(t *testing.T, sub string, exp time.Time) string {
	t.Helper()
	tok, err := jwt.NewBuilder().
		Subject(sub).
		Expiration(exp).
		Issuer("supabase").
		Build()
	require.NoError(t, err)
	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.HS256, []byte(testSecret)))
	require.NoError(t, err)
	return string(signed)
}

func TestVerify_ValidToken(t *testing.T) {
	v := NewVerifier(testSecret)
	token := makeToken(t, "user-123", time.Now().Add(time.Hour))

	uid, err := v.UserIDFromToken(token)

	assert.NoError(t, err)
	assert.Equal(t, "user-123", uid)
}

func TestVerify_ExpiredToken(t *testing.T) {
	v := NewVerifier(testSecret)
	token := makeToken(t, "user-123", time.Now().Add(-time.Hour))

	_, err := v.UserIDFromToken(token)

	assert.Error(t, err)
}

func TestVerify_WrongSecret(t *testing.T) {
	v := NewVerifier("different-secret")
	token := makeToken(t, "user-123", time.Now().Add(time.Hour))

	_, err := v.UserIDFromToken(token)

	assert.Error(t, err)
}
```

- [ ] **Step 3: 실패 확인 후 구현**

```bash
go test ./internal/auth/...
```
Expected: FAIL.

Create `apps/api/internal/auth/jwt.go`:
```go
package auth

import (
	"errors"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwt"
)

type Verifier struct {
	secret []byte
}

func NewVerifier(secret string) *Verifier {
	return &Verifier{secret: []byte(secret)}
}

func (v *Verifier) UserIDFromToken(token string) (string, error) {
	parsed, err := jwt.Parse([]byte(token),
		jwt.WithKey(jwa.HS256, v.secret),
		jwt.WithValidate(true),
	)
	if err != nil {
		return "", err
	}
	sub := parsed.Subject()
	if sub == "" {
		return "", errors.New("token has no subject")
	}
	return sub, nil
}
```

- [ ] **Step 4: 테스트 통과**

```bash
go test ./internal/auth/...
```
Expected: PASS (3개 테스트).

- [ ] **Step 5: 인증 미들웨어**

Create `apps/api/internal/middleware/auth.go`:
```go
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/<user>/quotient/apps/api/internal/auth"
)

type ctxKey string

const UserIDKey ctxKey = "user_id"
const RawJWTKey ctxKey = "raw_jwt"

func RequireAuth(v *auth.Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if len(header) < 7 || !strings.EqualFold(header[:7], "bearer ") {
				http.Error(w, `{"error":{"code":"UNAUTHENTICATED","message":"missing bearer token"}}`, http.StatusUnauthorized)
				return
			}
			// RFC 7235: scheme 대소문자 무관
			token := header[7:]
			uid, err := v.UserIDFromToken(token)
			if err != nil {
				http.Error(w, `{"error":{"code":"UNAUTHENTICATED","message":"invalid token"}}`, http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, uid)
			ctx = context.WithValue(ctx, RawJWTKey, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

func RawJWT(ctx context.Context) string {
	if v, ok := ctx.Value(RawJWTKey).(string); ok {
		return v
	}
	return ""
}
```

- [ ] **Step 6: 라우터에 인증 그룹 추가**

Replace `apps/api/internal/router/router.go`:
```go
package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/<user>/quotient/apps/api/internal/auth"
	"github.com/<user>/quotient/apps/api/internal/handlers"
	"github.com/<user>/quotient/apps/api/internal/middleware"
)

func New(verifier *auth.Verifier) *chi.Mux {
	r := chi.NewRouter()

	r.Get("/healthz", handlers.Healthz)
	r.Get("/readyz", handlers.Readyz)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(verifier))
		// /v1 라우트는 다음 task에서 추가
	})

	return r
}
```

- [ ] **Step 7: `Readyz` 핸들러 추가**

Add to `apps/api/internal/handlers/health.go`:
```go
func Readyz(w http.ResponseWriter, r *http.Request) {
	// 추후 DB ping 추가 (Task 6)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}
```

- [ ] **Step 8: main에 verifier 주입**

Update `apps/api/cmd/server/main.go`의 `srv.Handler` 부분:
```go
verifier := auth.NewVerifier(cfg.SupabaseJWTSecret)
srv := &http.Server{
	Addr:              fmt.Sprintf(":%d", cfg.Port),
	Handler:           router.New(verifier),
	ReadHeaderTimeout: 10 * time.Second,
}
```
import `"github.com/<user>/quotient/apps/api/internal/auth"`.

- [ ] **Step 9: 테스트·빌드 확인**

```bash
go test ./...
go build ./cmd/server
```
Expected: PASS, 빌드 성공.

- [ ] **Step 10: CORS 미들웨어 추가**

```bash
go get github.com/go-chi/cors
```

Create `apps/api/internal/middleware/cors.go`:
```go
package middleware

import (
	"net/http"

	"github.com/go-chi/cors"
)

func CORS(origin string) func(http.Handler) http.Handler {
	return cors.Handler(cors.Options{
		AllowedOrigins:   []string{origin},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}
```

Update `apps/api/internal/router/router.go` 시그니처:
```go
func New(verifier *auth.Verifier, corsOrigin string) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.CORS(corsOrigin))
	r.Get("/healthz", handlers.Healthz)
	r.Get("/readyz", handlers.Readyz)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(verifier))
		// /v1 라우트는 Task 6
	})

	return r
}
```

Update `main.go`에서 `router.New(verifier, cfg.CORSOrigin)`.

- [ ] **Step 11: 커밋**

```bash
git add apps/api/
git commit -m "feat(api): Supabase JWT 검증 미들웨어 + CORS"
```

---

## Task 6: profiles GET·PATCH 엔드포인트

**Files:**
- Create: `apps/api/internal/db/db.go`
- Create: `apps/api/internal/models/profile.go`
- Create: `apps/api/internal/handlers/profiles.go`
- Create: `apps/api/internal/handlers/profiles_test.go`
- Modify: `apps/api/internal/router/router.go`
- Modify: `apps/api/cmd/server/main.go`

- [ ] **Step 1: pgx 의존성**

```bash
cd apps/api
go get github.com/jackc/pgx/v5/pgxpool
```

- [ ] **Step 2: DB 연결 풀**

Create `apps/api/internal/db/db.go`:
```go
package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func New(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, dsn)
}
```

- [ ] **Step 3: 도메인 모델**

Create `apps/api/internal/models/profile.go`:
```go
package models

import "time"

type Profile struct {
	ID                    string    `json:"id"`
	DisplayName           *string   `json:"display_name"`
	BaseCurrency          string    `json:"base_currency"`
	UIIntensity           string    `json:"ui_intensity"`
	OnboardingCompleted   bool      `json:"onboarding_completed"`
	DailyBriefingEnabled  bool      `json:"daily_briefing_enabled"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type ProfilePatch struct {
	DisplayName          *string `json:"display_name,omitempty"`
	BaseCurrency         *string `json:"base_currency,omitempty" validate:"omitempty,oneof=KRW USD"`
	UIIntensity          *string `json:"ui_intensity,omitempty" validate:"omitempty,oneof=vivid standard subtle"`
	OnboardingCompleted  *bool   `json:"onboarding_completed,omitempty"`
	DailyBriefingEnabled *bool   `json:"daily_briefing_enabled,omitempty"`
}
```

- [ ] **Step 4: profiles 핸들러 테스트 (integration — testcontainers는 W2, 지금은 mock pool)**

Create `apps/api/internal/handlers/profiles_test.go`:
```go
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/<user>/quotient/apps/api/internal/middleware"
	"github.com/stretchr/testify/assert"
)

// 임시 fake repo — Task 7에서 sqlc로 교체
type fakeRepo struct {
	getResp map[string]any
}

func (r *fakeRepo) Get(ctx context.Context, uid string) (map[string]any, error) {
	return r.getResp, nil
}

func (r *fakeRepo) Update(ctx context.Context, uid string, patch map[string]any) error {
	return nil
}

func TestGetProfile_ReturnsCurrent(t *testing.T) {
	repo := &fakeRepo{getResp: map[string]any{
		"id":            "user-1",
		"display_name":  "Hojin",
		"base_currency": "KRW",
	}}
	h := NewProfileHandler(repo)
	req := httptest.NewRequest(http.MethodGet, "/v1/profile", nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-1")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Get(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	assert.Equal(t, "user-1", got["id"])
}

func TestPatchProfile_AcceptsValidBody(t *testing.T) {
	repo := &fakeRepo{getResp: map[string]any{"id": "user-1"}}
	h := NewProfileHandler(repo)
	body, _ := json.Marshal(map[string]any{"base_currency": "USD"})
	req := httptest.NewRequest(http.MethodPatch, "/v1/profile", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-1")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Patch(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestPatchProfile_RejectsInvalidCurrency(t *testing.T) {
	repo := &fakeRepo{}
	h := NewProfileHandler(repo)
	body, _ := json.Marshal(map[string]any{"base_currency": "EUR"})
	req := httptest.NewRequest(http.MethodPatch, "/v1/profile", bytes.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-1")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.Patch(rec, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}
```

- [ ] **Step 5: 실패 확인**

```bash
go test ./internal/handlers/...
```
Expected: FAIL — `NewProfileHandler` undefined.

- [ ] **Step 6: 핸들러 구현 (fake repo 인터페이스로)**

Create `apps/api/internal/handlers/profiles.go`:
```go
package handlers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/<user>/quotient/apps/api/internal/middleware"
)

type ProfileRepo interface {
	Get(ctx context.Context, userID string) (map[string]any, error)
	Update(ctx context.Context, userID string, patch map[string]any) error
}

type ProfileHandler struct {
	repo ProfileRepo
}

func NewProfileHandler(repo ProfileRepo) *ProfileHandler {
	return &ProfileHandler{repo: repo}
}

func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	p, err := h.repo.Get(r.Context(), uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *ProfileHandler) Patch(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	var patch map[string]any
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if v, ok := patch["base_currency"].(string); ok && v != "KRW" && v != "USD" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "base_currency must be KRW or USD")
		return
	}
	if v, ok := patch["ui_intensity"].(string); ok && v != "vivid" && v != "standard" && v != "subtle" {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "ui_intensity invalid")
		return
	}
	if err := h.repo.Update(r.Context(), uid, patch); err != nil {
		writeError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	p, _ := h.repo.Get(r.Context(), uid)
	writeJSON(w, http.StatusOK, p)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": msg},
	})
}
```

- [ ] **Step 7: 실제 Postgres repo (pgxpool 사용)**

Create `apps/api/internal/handlers/profile_repo_pg.go`:
```go
package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgProfileRepo struct {
	pool *pgxpool.Pool
}

func NewPgProfileRepo(pool *pgxpool.Pool) *PgProfileRepo {
	return &PgProfileRepo{pool: pool}
}

func (r *PgProfileRepo) Get(ctx context.Context, uid string) (map[string]any, error) {
	row := r.pool.QueryRow(ctx, `
		select id, display_name, base_currency, ui_intensity,
		       onboarding_completed, daily_briefing_enabled,
		       created_at, updated_at
		from public.profiles where id = $1`, uid)
	var id, baseCurrency, uiIntensity string
	var displayName *string
	var onboarding, daily bool
	var created, updated any
	if err := row.Scan(&id, &displayName, &baseCurrency, &uiIntensity, &onboarding, &daily, &created, &updated); err != nil {
		return nil, err
	}
	return map[string]any{
		"id":                     id,
		"display_name":           displayName,
		"base_currency":          baseCurrency,
		"ui_intensity":           uiIntensity,
		"onboarding_completed":   onboarding,
		"daily_briefing_enabled": daily,
		"created_at":             created,
		"updated_at":             updated,
	}, nil
}

func (r *PgProfileRepo) Update(ctx context.Context, uid string, patch map[string]any) error {
	if len(patch) == 0 {
		return nil
	}
	sets := []string{}
	args := []any{}
	i := 1
	for k, v := range patch {
		switch k {
		case "display_name", "base_currency", "ui_intensity", "onboarding_completed", "daily_briefing_enabled":
			sets = append(sets, fmt.Sprintf("%s = $%d", k, i))
			args = append(args, v)
			i++
		}
	}
	if len(sets) == 0 {
		return nil
	}
	args = append(args, uid)
	q := fmt.Sprintf(`update public.profiles set %s where id = $%d`, strings.Join(sets, ", "), i)
	_, err := r.pool.Exec(ctx, q, args...)
	return err
}
```

- [ ] **Step 8: 라우터·main 연결**

Replace `apps/api/internal/router/router.go`:
```go
package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/<user>/quotient/apps/api/internal/auth"
	"github.com/<user>/quotient/apps/api/internal/handlers"
	"github.com/<user>/quotient/apps/api/internal/middleware"
)

func New(verifier *auth.Verifier, profileHandler *handlers.ProfileHandler) *chi.Mux {
	r := chi.NewRouter()
	r.Get("/healthz", handlers.Healthz)
	r.Get("/readyz", handlers.Readyz)

	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(verifier))
		r.Get("/v1/profile", profileHandler.Get)
		r.Patch("/v1/profile", profileHandler.Patch)
	})

	return r
}
```

Update `apps/api/cmd/server/main.go` (DB 풀 + handler 생성):
```go
ctx := context.Background()
pool, err := db.New(ctx, cfg.DatabaseURL)
if err != nil {
	logger.Error("db connect failed", "err", err); os.Exit(1)
}
defer pool.Close()

verifier := auth.NewVerifier(cfg.SupabaseJWTSecret)
profileRepo := handlers.NewPgProfileRepo(pool)
profileHandler := handlers.NewProfileHandler(profileRepo)

srv := &http.Server{
	Addr:              fmt.Sprintf(":%d", cfg.Port),
	Handler:           router.New(verifier, profileHandler),
	ReadHeaderTimeout: 10 * time.Second,
}
```

imports에 `"github.com/<user>/quotient/apps/api/internal/db"`, `"github.com/<user>/quotient/apps/api/internal/handlers"` 추가.

- [ ] **Step 9: 모든 테스트 통과**

```bash
go test ./...
go build ./cmd/server
```
Expected: PASS.

- [ ] **Step 10: 통합 동작 확인 (로컬)**

```bash
make db-up
DATABASE_URL=postgresql://postgres:postgres@localhost:54322/postgres \
SUPABASE_JWT_SECRET=<로컬 supabase JWT secret — supabase status로 확인> \
go run ./cmd/server/main.go
```

별도 터미널에서 (인증 없이 401 확인):
```bash
curl -i http://localhost:8080/v1/profile
```
Expected: 401 + `{"error":...}`.

- [ ] **Step 11: 커밋**

```bash
git add apps/api/
git commit -m "feat(api): GET·PATCH /v1/profile + Postgres repo"
```

---

## Task 7: Next.js 스캐폴딩 + Tailwind + shadcn/ui

**Files:**
- Create: `apps/web/` 전체 Next.js 프로젝트
- Create: `apps/web/tailwind.config.ts` (블룸버그 토큰)
- Create: `apps/web/app/globals.css`
- Create: `apps/web/app/layout.tsx`
- Create: `apps/web/app/page.tsx`
- Create: 다수 shadcn UI 컴포넌트

- [ ] **Step 1: Next.js 프로젝트 생성**

```bash
cd apps
npx create-next-app@latest web \
  --typescript --tailwind --eslint --app --no-src-dir \
  --import-alias "@/*" --use-npm
cd web
```

- [ ] **Step 2: 핵심 의존성 추가**

```bash
# 코어
npm install lucide-react clsx tailwind-merge class-variance-authority
npm install motion@^11
npm install @supabase/supabase-js @supabase/ssr
npm install @tanstack/react-query zustand zod react-hook-form @hookform/resolvers

# Tailwind plugin (shadcn 의존)
npm install -D tailwindcss-animate

# 폰트
npm install pretendard

# 테스트
npm install -D vitest @vitejs/plugin-react @testing-library/react @testing-library/jest-dom jsdom
```

- [ ] **Step 3: shadcn/ui 초기화**

```bash
npx shadcn@latest init
```
프롬프트 응답:
- Style: `Default`
- Base color: `Slate` (다크 톤 기본)
- CSS variables: `Yes`

```bash
npx shadcn@latest add button input label card dialog dropdown-menu skeleton sonner toggle separator
```

- [ ] **Step 4: Tailwind 설정에 블룸버그 토큰 추가 (shadcn 토큰 보존)**

**중요**: shadcn init이 생성한 `tailwind.config.ts`를 **전체 교체하지 말 것**. shadcn은 `--background`·`--foreground`·`--border` 등 CSS 변수에 의존한다. 우리 블룸버그 톤은 `bb.*` 와 의미적 별칭(`bg`·`fg`·`line`)으로 **추가**한다.

shadcn이 생성한 파일의 `theme.extend.colors` 안에 다음을 추가:
```ts
// theme.extend.colors 내부에 추가
bg: {
  DEFAULT: "#0A0A0A",
  subtle:  "#111111",
  card:    "#141414",
},
fg: {
  DEFAULT: "#E5E5E5",
  muted:   "#737373",
  subtle:  "#525252",
},
line: "#262626",
bb: {
  up:     "#00FF7F",  // 상승·완료
  down:   "#FF3344",  // 하락·실패
  accent: "#FFD500",  // 강조
  info:   "#00FFFF",  // 진행·정보
  warn:   "#FF9900",  // 경고
},
```

그리고 `theme.extend.fontFamily` 추가/확장:
```ts
fontFamily: {
  sans: ["var(--font-pretendard)", "Inter", "system-ui", "sans-serif"],
  mono: ["JetBrains Mono", "Menlo", "monospace"],
},
```

shadcn의 기본 `background`·`foreground`·`border`·`primary` 등은 **유지** (shadcn 컴포넌트가 의존).

- [ ] **Step 5: globals.css 블룸버그 톤 적용**

Replace `apps/web/app/globals.css`:
```css
@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
  html { color-scheme: dark; }
  body {
    @apply bg-bg text-fg font-sans antialiased;
    font-feature-settings: "tnum" 1, "cv01" 1;
  }
  ::selection { @apply bg-bb-accent text-bg; }
}

@layer utilities {
  .num   { @apply font-mono tabular-nums; }
  .up    { @apply text-bb-up; }
  .down  { @apply text-bb-down; }
  .accent{ @apply text-bb-accent; }
}
```

- [ ] **Step 6: 루트 레이아웃 + 랜딩**

Replace `apps/web/app/layout.tsx`:
```tsx
import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Quotient — Portfolio Intelligence Terminal",
  description: "한국·미국 자산을 한 화면에. 자연어로 묻고, 즉시 분석을 받으세요.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ko" className="dark">
      <body>{children}</body>
    </html>
  );
}
```

Replace `apps/web/app/page.tsx`:
```tsx
import Link from "next/link";

export default function LandingPage() {
  return (
    <main className="min-h-screen flex flex-col">
      <section className="flex-1 flex flex-col items-center justify-center gap-8 px-6">
        <h1 className="font-mono text-5xl md:text-7xl tracking-tight text-center">
          Portfolio Intelligence Terminal
        </h1>
        <p className="text-fg-muted text-center max-w-xl">
          한국·미국 자산을 한 화면에. 자연어로 묻고, 즉시 분석을 받으세요.
        </p>
        <div className="flex gap-3">
          <Link href="/signup" className="px-6 py-2 bg-bb-accent text-bg font-mono">
            가입하기
          </Link>
          <Link href="/login" className="px-6 py-2 border border-line font-mono">
            로그인
          </Link>
        </div>
      </section>
      <footer className="border-t border-line py-4 px-6 text-fg-muted text-xs">
        투자 자문이 아닙니다. 모든 의사결정은 본인 책임입니다.
      </footer>
    </main>
  );
}
```

- [ ] **Step 7: 정적 페이지 (pricing, privacy, terms)**

Create `apps/web/app/pricing/page.tsx`:
```tsx
export default function Pricing() {
  return (
    <main className="min-h-screen p-12">
      <h1 className="font-mono text-3xl mb-6">PRICING</h1>
      <div className="border border-line p-6">
        <h2 className="font-mono text-xl">Free</h2>
        <p className="text-fg-muted mt-2">현재 모든 기능 무료 (베타).</p>
        <ul className="mt-4 space-y-1 text-fg">
          <li>· 포트폴리오 관리</li>
          <li>· 분석가 채팅 (월 30회)</li>
          <li>· 일일 브리핑</li>
          <li>· 한국·미국 시세 (15분 지연)</li>
        </ul>
      </div>
    </main>
  );
}
```

> 카피 규칙: "AI" 단어는 main copy·UI 텍스트에 금지 (스펙 §1). "분석가"·"인텔리전스"·"엔진" 사용.

Create `apps/web/app/privacy/page.tsx` and `apps/web/app/terms/page.tsx` (각각 minimal 정적 마크다운):
```tsx
// privacy
export default function Privacy() {
  return (
    <main className="min-h-screen p-12 max-w-3xl mx-auto prose prose-invert">
      <h1>개인정보 처리방침</h1>
      <p>최종 갱신: 2026-05-22</p>
      <h2>수집 항목</h2>
      <ul><li>이메일, 비밀번호 (해시), 이름</li><li>보유 자산 정보</li><li>AI 채팅 기록</li></ul>
      <h2>처리 위탁</h2>
      <ul>
        <li>Supabase (인증·DB 호스팅)</li>
        <li>Anthropic (AI 분석)</li>
        <li>Resend (이메일 발송)</li>
        <li>Sentry (에러 추적)</li>
        <li>PostHog (활동 분석)</li>
      </ul>
      <h2>보유 기간</h2>
      <p>회원 탈퇴 시 즉시 파기. 결제 기록은 한국 세법에 따라 7년 보관 (익명화).</p>
    </main>
  );
}
```

`terms/page.tsx`도 유사한 골격으로.

- [ ] **Step 8: 빌드 확인**

```bash
npm run build
```
Expected: 빌드 성공, 5개 정적 페이지 prerender.

- [ ] **Step 9: dev 서버 확인**

```bash
npm run dev
```
Expected: http://localhost:3000 접속 시 블룸버그 풍 랜딩 페이지.

- [ ] **Step 10: 커밋**

```bash
cd ../..
git add apps/web/
git commit -m "feat(web): Next.js 스캐폴딩 + Tailwind 블룸버그 토큰 + shadcn/ui + 랜딩"
```

---

## Task 8: Supabase 클라이언트 + SSR

**Files:**
- Create: `apps/web/lib/supabase/client.ts`
- Create: `apps/web/lib/supabase/server.ts`
- Create: `apps/web/lib/supabase/middleware.ts`
- Create: `apps/web/middleware.ts`

- [ ] **Step 1: 브라우저 클라이언트**

Create `apps/web/lib/supabase/client.ts`:
```ts
import { createBrowserClient } from "@supabase/ssr";

export function createSupabaseBrowser() {
  return createBrowserClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!
  );
}
```

- [ ] **Step 2: SSR 클라이언트**

Create `apps/web/lib/supabase/server.ts`:
```ts
import { createServerClient } from "@supabase/ssr";
import { cookies } from "next/headers";

export async function createSupabaseServer() {
  const cookieStore = await cookies();
  return createServerClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
    {
      cookies: {
        getAll: () => cookieStore.getAll(),
        setAll: (list) => {
          try {
            list.forEach(({ name, value, options }) => cookieStore.set(name, value, options));
          } catch {
            // SSR에서 cookies write 시도 (RSC)에는 noop
          }
        },
      },
    }
  );
}
```

- [ ] **Step 3: 미들웨어 (세션 갱신 + 라우트 가드)**

Create `apps/web/lib/supabase/middleware.ts`:
```ts
import { createServerClient } from "@supabase/ssr";
import { NextResponse, type NextRequest } from "next/server";

export async function updateSession(request: NextRequest) {
  let response = NextResponse.next({ request });

  const supabase = createServerClient(
    process.env.NEXT_PUBLIC_SUPABASE_URL!,
    process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY!,
    {
      cookies: {
        getAll: () => request.cookies.getAll(),
        setAll: (list) => {
          list.forEach(({ name, value }) => request.cookies.set(name, value));
          response = NextResponse.next({ request });
          list.forEach(({ name, value, options }) =>
            response.cookies.set(name, value, options)
          );
        },
      },
    }
  );

  const { data: { user } } = await supabase.auth.getUser();

  // /app/* 는 인증 필수
  if (!user && request.nextUrl.pathname.startsWith("/app")) {
    return NextResponse.redirect(new URL("/login", request.url));
  }
  // 로그인 사용자가 /login·/signup 접근 시 /app으로
  if (user && ["/login", "/signup"].includes(request.nextUrl.pathname)) {
    return NextResponse.redirect(new URL("/app", request.url));
  }

  // 온보딩 미완료 사용자가 /app/* 접근 시 /app/onboarding으로 (단, /app/onboarding 자체는 통과)
  if (user && request.nextUrl.pathname.startsWith("/app") && request.nextUrl.pathname !== "/app/onboarding") {
    const { data: profile } = await supabase
      .from("profiles")
      .select("onboarding_completed")
      .eq("id", user.id)
      .single();
    if (profile && !profile.onboarding_completed) {
      return NextResponse.redirect(new URL("/app/onboarding", request.url));
    }
  }

  return response;
}
```

Create `apps/web/middleware.ts`:
```ts
import { updateSession } from "@/lib/supabase/middleware";
import type { NextRequest } from "next/server";

export async function middleware(request: NextRequest) {
  return await updateSession(request);
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico|.*\\.(?:svg|png|jpg|jpeg|gif|webp)$).*)"],
};
```

- [ ] **Step 4: 빌드·환경변수 동작 확인**

```bash
npm run build
```
Expected: 미들웨어 컴파일 성공.

- [ ] **Step 5: 커밋**

```bash
cd ../..
git add apps/web/
git commit -m "feat(web): Supabase SSR 클라이언트 + 세션 미들웨어 + 라우트 가드"
```

---

## Task 9: 가입·로그인·이메일 인증 페이지

**Files:**
- Create: `apps/web/components/auth/AuthCard.tsx`
- Create: `apps/web/components/auth/SignupForm.tsx`
- Create: `apps/web/components/auth/LoginForm.tsx`
- Create: `apps/web/components/auth/GoogleButton.tsx`
- Create: `apps/web/app/signup/page.tsx`
- Create: `apps/web/app/login/page.tsx`
- Create: `apps/web/app/verify-email/page.tsx`
- Create: `apps/web/app/auth/callback/route.ts`

- [ ] **Step 1: AuthCard (공통 레이아웃)**

Create `apps/web/components/auth/AuthCard.tsx`:
```tsx
export function AuthCard({ title, subtitle, children }: {
  title: string;
  subtitle?: string;
  children: React.ReactNode;
}) {
  return (
    <main className="min-h-screen flex items-center justify-center px-6">
      <div className="w-full max-w-sm border border-line bg-bg-card p-8 space-y-6">
        <header>
          <h1 className="font-mono text-xl">{title}</h1>
          {subtitle && <p className="text-fg-muted text-sm mt-1">{subtitle}</p>}
        </header>
        {children}
      </div>
    </main>
  );
}
```

- [ ] **Step 2: SignupForm**

Create `apps/web/components/auth/SignupForm.tsx`:
```tsx
"use client";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { createSupabaseBrowser } from "@/lib/supabase/client";

// PIPA: 약관·개인정보 처리방침 분리 동의 + 14세 이상 확인
const schema = z.object({
  email: z.string().email("올바른 이메일을 입력해주세요"),
  password: z.string().min(8, "비밀번호는 최소 8자입니다"),
  agree_terms: z.literal(true, { message: "서비스 약관에 동의해주세요" }),
  agree_privacy: z.literal(true, { message: "개인정보 처리방침에 동의해주세요" }),
  age_14: z.literal(true, { message: "만 14세 이상이어야 가입할 수 있습니다" }),
});
type FormData = z.infer<typeof schema>;

export function SignupForm() {
  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<FormData>({
    resolver: zodResolver(schema),
  });
  const [serverError, setServerError] = useState<string | null>(null);
  const router = useRouter();

  async function onSubmit(data: FormData) {
    setServerError(null);
    const supabase = createSupabaseBrowser();
    const { error } = await supabase.auth.signUp({
      email: data.email,
      password: data.password,
      options: {
        emailRedirectTo: `${window.location.origin}/auth/callback`,
      },
    });
    if (error) { setServerError(error.message); return; }
    router.push(`/verify-email?email=${encodeURIComponent(data.email)}`);
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div>
        <label className="text-xs text-fg-muted">이메일</label>
        <input
          {...register("email")}
          type="email"
          className="w-full bg-bg border border-line px-3 py-2 mt-1 font-mono text-sm"
          autoComplete="email"
        />
        {errors.email && <p className="text-bb-down text-xs mt-1">{errors.email.message}</p>}
      </div>
      <div>
        <label className="text-xs text-fg-muted">비밀번호 (8자 이상)</label>
        <input
          {...register("password")}
          type="password"
          className="w-full bg-bg border border-line px-3 py-2 mt-1 font-mono text-sm"
          autoComplete="new-password"
        />
        {errors.password && <p className="text-bb-down text-xs mt-1">{errors.password.message}</p>}
      </div>
      <div className="space-y-2">
        <label className="flex items-start gap-2 text-xs text-fg-muted">
          <input type="checkbox" {...register("agree_terms")} className="mt-1" />
          <span><a href="/terms" className="underline">서비스 약관</a>에 동의합니다.</span>
        </label>
        {errors.agree_terms && <p className="text-bb-down text-xs">{errors.agree_terms.message}</p>}

        <label className="flex items-start gap-2 text-xs text-fg-muted">
          <input type="checkbox" {...register("agree_privacy")} className="mt-1" />
          <span><a href="/privacy" className="underline">개인정보 처리방침</a>에 동의합니다.</span>
        </label>
        {errors.agree_privacy && <p className="text-bb-down text-xs">{errors.agree_privacy.message}</p>}

        <label className="flex items-start gap-2 text-xs text-fg-muted">
          <input type="checkbox" {...register("age_14")} className="mt-1" />
          <span>만 14세 이상입니다.</span>
        </label>
        {errors.age_14 && <p className="text-bb-down text-xs">{errors.age_14.message}</p>}
      </div>
      {serverError && <p className="text-bb-down text-xs">{serverError}</p>}
      <button type="submit" disabled={isSubmitting} className="w-full bg-bb-accent text-bg font-mono py-2 disabled:opacity-50">
        {isSubmitting ? "가입 처리 중…" : "가입하기"}
      </button>
    </form>
  );
}
```

- [ ] **Step 3: LoginForm**

Create `apps/web/components/auth/LoginForm.tsx`:
```tsx
"use client";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { createSupabaseBrowser } from "@/lib/supabase/client";

const schema = z.object({
  email: z.string().email(),
  password: z.string().min(1),
});
type FormData = z.infer<typeof schema>;

export function LoginForm() {
  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<FormData>({ resolver: zodResolver(schema) });
  const [serverError, setServerError] = useState<string | null>(null);
  const router = useRouter();

  async function onSubmit(data: FormData) {
    setServerError(null);
    const supabase = createSupabaseBrowser();
    const { error } = await supabase.auth.signInWithPassword({ email: data.email, password: data.password });
    if (error) { setServerError("이메일 또는 비밀번호가 올바르지 않습니다."); return; }
    router.push("/app");
    router.refresh();
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div>
        <label className="text-xs text-fg-muted">이메일</label>
        <input {...register("email")} type="email" className="w-full bg-bg border border-line px-3 py-2 mt-1 font-mono text-sm" />
      </div>
      <div>
        <label className="text-xs text-fg-muted">비밀번호</label>
        <input {...register("password")} type="password" className="w-full bg-bg border border-line px-3 py-2 mt-1 font-mono text-sm" />
      </div>
      <a href="/forgot-password" className="text-xs text-fg-muted underline">비밀번호를 잊으셨나요?</a>
      {serverError && <p className="text-bb-down text-xs">{serverError}</p>}
      <button disabled={isSubmitting} className="w-full bg-bb-accent text-bg font-mono py-2 disabled:opacity-50">
        {isSubmitting ? "로그인 중…" : "로그인"}
      </button>
    </form>
  );
}
```

- [ ] **Step 4: GoogleButton**

Create `apps/web/components/auth/GoogleButton.tsx`:
```tsx
"use client";
import { createSupabaseBrowser } from "@/lib/supabase/client";

export function GoogleButton() {
  async function onClick() {
    const supabase = createSupabaseBrowser();
    await supabase.auth.signInWithOAuth({
      provider: "google",
      options: { redirectTo: `${window.location.origin}/auth/callback` },
    });
  }
  return (
    <button onClick={onClick} className="w-full border border-line py-2 font-mono text-sm">
      Google로 계속하기
    </button>
  );
}
```

- [ ] **Step 5: 페이지 연결**

Create `apps/web/app/signup/page.tsx`:
```tsx
import { AuthCard } from "@/components/auth/AuthCard";
import { SignupForm } from "@/components/auth/SignupForm";
import { GoogleButton } from "@/components/auth/GoogleButton";

export default function SignupPage() {
  return (
    <AuthCard title="가입" subtitle="Portfolio Intelligence Terminal">
      <SignupForm />
      <div className="text-center text-fg-muted text-xs">또는</div>
      <GoogleButton />
      <p className="text-center text-xs text-fg-muted">
        이미 계정이 있으세요? <a href="/login" className="underline">로그인</a>
      </p>
    </AuthCard>
  );
}
```

Create `apps/web/app/login/page.tsx`:
```tsx
import { AuthCard } from "@/components/auth/AuthCard";
import { LoginForm } from "@/components/auth/LoginForm";
import { GoogleButton } from "@/components/auth/GoogleButton";

export default function LoginPage() {
  return (
    <AuthCard title="로그인">
      <LoginForm />
      <div className="text-center text-fg-muted text-xs">또는</div>
      <GoogleButton />
      <p className="text-center text-xs text-fg-muted">
        계정이 없으세요? <a href="/signup" className="underline">가입</a>
      </p>
    </AuthCard>
  );
}
```

- [ ] **Step 6: 이메일 인증 안내 페이지**

Create `apps/web/app/verify-email/page.tsx`:
```tsx
// Next.js 15: searchParams는 Promise — async 컴포넌트 + await 필수
export default async function VerifyEmailPage({
  searchParams,
}: {
  searchParams: Promise<{ email?: string }>;
}) {
  const { email } = await searchParams;
  return (
    <main className="min-h-screen flex items-center justify-center px-6">
      <div className="max-w-md text-center space-y-4">
        <h1 className="font-mono text-2xl">이메일을 확인해주세요</h1>
        <p className="text-fg-muted">
          {email ? <span className="font-mono">{email}</span> : "가입한 이메일"} 로 인증 메일을 보냈습니다.
        </p>
        <p className="text-fg-muted text-sm">
          메일 내 링크를 클릭하면 가입이 완료됩니다.
        </p>
      </div>
    </main>
  );
}
```

- [ ] **Step 7: OAuth 콜백 라우트**

Create `apps/web/app/auth/callback/route.ts`:
```ts
import { createSupabaseServer } from "@/lib/supabase/server";
import { NextResponse } from "next/server";

export async function GET(request: Request) {
  const { searchParams, origin } = new URL(request.url);
  const code = searchParams.get("code");
  if (code) {
    const supabase = await createSupabaseServer();
    await supabase.auth.exchangeCodeForSession(code);
  }
  return NextResponse.redirect(`${origin}/app`);
}
```

- [ ] **Step 8: 동작 확인 (로컬)**

```bash
make db-up
cd apps/web && npm run dev
```
- http://localhost:3000/signup 에서 신규 가입
- 로컬 Supabase inbucket에서 인증 메일 확인: http://localhost:54324
- 링크 클릭 → /app 진입 (다음 task에서 셸 구현, 우선 404 또는 빈 페이지여도 OK)

- [ ] **Step 9: 커밋**

```bash
cd ../..
git add apps/web/
git commit -m "feat(web): 가입·로그인·이메일 인증·Google OAuth 페이지"
```

---

## Task 10: 비밀번호 재설정 페이지

**Files:**
- Create: `apps/web/app/forgot-password/page.tsx`
- Create: `apps/web/app/reset-password/page.tsx`

- [ ] **Step 1: forgot-password 페이지**

Create `apps/web/app/forgot-password/page.tsx`:
```tsx
"use client";
import { useState } from "react";
import { AuthCard } from "@/components/auth/AuthCard";
import { createSupabaseBrowser } from "@/lib/supabase/client";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [sent, setSent] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    const supabase = createSupabaseBrowser();
    const { error } = await supabase.auth.resetPasswordForEmail(email, {
      redirectTo: `${window.location.origin}/reset-password`,
    });
    if (error) setErr(error.message);
    else setSent(true);
  }

  if (sent) {
    return (
      <AuthCard title="이메일 발송 완료">
        <p className="text-fg-muted text-sm"><span className="font-mono">{email}</span>로 비밀번호 재설정 링크를 보냈습니다.</p>
      </AuthCard>
    );
  }

  return (
    <AuthCard title="비밀번호 재설정" subtitle="가입한 이메일을 입력해주세요">
      <form onSubmit={onSubmit} className="space-y-4">
        <input
          type="email" value={email} onChange={(e) => setEmail(e.target.value)} required
          className="w-full bg-bg border border-line px-3 py-2 font-mono text-sm"
        />
        {err && <p className="text-bb-down text-xs">{err}</p>}
        <button className="w-full bg-bb-accent text-bg font-mono py-2">재설정 링크 받기</button>
      </form>
    </AuthCard>
  );
}
```

- [ ] **Step 2: reset-password 페이지**

Create `apps/web/app/reset-password/page.tsx`:
```tsx
"use client";
import { useState } from "react";
import { AuthCard } from "@/components/auth/AuthCard";
import { createSupabaseBrowser } from "@/lib/supabase/client";
import { useRouter } from "next/navigation";

export default function ResetPasswordPage() {
  const [pw, setPw] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const router = useRouter();

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setErr(null);
    if (pw.length < 8) { setErr("최소 8자"); return; }
    const supabase = createSupabaseBrowser();
    const { error } = await supabase.auth.updateUser({ password: pw });
    if (error) { setErr(error.message); return; }
    router.push("/login");
  }

  return (
    <AuthCard title="새 비밀번호 설정">
      <form onSubmit={onSubmit} className="space-y-4">
        <input
          type="password" value={pw} onChange={(e) => setPw(e.target.value)}
          className="w-full bg-bg border border-line px-3 py-2 font-mono text-sm"
          placeholder="8자 이상"
        />
        {err && <p className="text-bb-down text-xs">{err}</p>}
        <button className="w-full bg-bb-accent text-bg font-mono py-2">변경하기</button>
      </form>
    </AuthCard>
  );
}
```

- [ ] **Step 3: 동작 확인**

`/forgot-password` 입력 → 로컬 inbucket 메일 확인 → 링크 클릭 → `/reset-password` → 변경 → 로그인.

- [ ] **Step 4: 커밋**

```bash
git add apps/web/app/forgot-password apps/web/app/reset-password
git commit -m "feat(web): 비밀번호 재설정 흐름"
```

---

## Task 11: 앱 셸 (사이드바·티커·상태바)

**Files:**
- Create: `apps/web/components/shell/Sidebar.tsx`
- Create: `apps/web/components/shell/TopTicker.tsx`
- Create: `apps/web/components/shell/StatusBar.tsx`
- Create: `apps/web/components/shell/AppShell.tsx`
- Create: `apps/web/app/app/layout.tsx`
- Create: `apps/web/app/app/page.tsx`
- Create: `apps/web/app/error.tsx`

- [ ] **Step 1: Sidebar**

Create `apps/web/components/shell/Sidebar.tsx`:
```tsx
"use client";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Home, Wallet, MessageSquare, BarChart3, Settings } from "lucide-react";
import { clsx } from "clsx";

const items = [
  { href: "/app", icon: Home, label: "홈" },
  { href: "/app/portfolio", icon: Wallet, label: "포트폴리오" },
  { href: "/app/chat", icon: MessageSquare, label: "채팅" },
  { href: "/app/market", icon: BarChart3, label: "마켓" },
  { href: "/app/settings", icon: Settings, label: "설정" },
];

export function Sidebar() {
  const path = usePathname();
  return (
    <aside className="w-14 border-r border-line bg-bg flex flex-col items-center py-3 gap-1">
      <div className="font-mono text-bb-accent text-[10px] mb-3">Q</div>
      {items.map(({ href, icon: Icon, label }) => {
        const active = path === href;
        return (
          <Link key={href} href={href}
            className={clsx(
              "w-10 h-10 flex items-center justify-center transition-colors",
              active ? "text-bb-accent" : "text-fg-muted hover:text-fg"
            )}
            title={label}
          >
            <Icon size={18} strokeWidth={1.5} />
          </Link>
        );
      })}
    </aside>
  );
}
```

- [ ] **Step 2: TopTicker (placeholder)**

Create `apps/web/components/shell/TopTicker.tsx`:
```tsx
// W2 후 실데이터로 교체
export function TopTicker() {
  const items = [
    { sym: "KOSPI",   val: "—", chg: null },
    { sym: "S&P 500", val: "—", chg: null },
    { sym: "USD/KRW", val: "—", chg: null },
  ];
  return (
    <header className="h-9 border-b border-line bg-bg flex items-center px-4 gap-6 text-xs">
      <span className="font-mono text-bb-accent">QUOTIENT</span>
      {items.map((it) => (
        <span key={it.sym} className="font-mono text-fg-muted">
          {it.sym} <span className="text-fg">{it.val}</span>
        </span>
      ))}
      <span className="ml-auto font-mono text-fg-muted text-[10px]">시세 지연 15분</span>
    </header>
  );
}
```

- [ ] **Step 3: StatusBar**

Create `apps/web/components/shell/StatusBar.tsx`:
```tsx
"use client";
import { useEffect, useState } from "react";

export function StatusBar() {
  const [now, setNow] = useState(new Date());
  useEffect(() => {
    const id = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(id);
  }, []);
  const time = now.toLocaleTimeString("ko-KR", { hour12: false, timeZone: "Asia/Seoul" });
  return (
    <footer className="h-6 border-t border-line bg-bg flex items-center px-4 gap-4 text-[10px] font-mono text-fg-muted">
      <span>↑ {time} KST</span>
      <span className="text-bb-up">● API</span>
      <span className="ml-auto">⌘K 명령 팔레트</span>
    </footer>
  );
}
```

- [ ] **Step 4: AppShell**

Create `apps/web/components/shell/AppShell.tsx`:
```tsx
import { Sidebar } from "./Sidebar";
import { TopTicker } from "./TopTicker";
import { StatusBar } from "./StatusBar";

export function AppShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen flex flex-col">
      <TopTicker />
      <div className="flex flex-1">
        <Sidebar />
        <main className="flex-1 overflow-auto">{children}</main>
      </div>
      <StatusBar />
    </div>
  );
}
```

- [ ] **Step 5: /app 레이아웃 + 홈 placeholder + 온보딩 가드**

Create `apps/web/app/app/layout.tsx`:
```tsx
// 인증·온보딩 redirect는 middleware.ts가 담당 (deep link 우회 방지)
// 여기서는 셸만 렌더. user 확인은 미들웨어가 이미 수행했으므로 여기서 다시 안 함.
import { AppShell } from "@/components/shell/AppShell";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  return <AppShell>{children}</AppShell>;
}
```

> **결정 (C9 보강)**: 사용자 메타데이터 (`profile`) 수정은 RLS 신뢰 모델 안에서 Supabase JS로 직접 수행. 결제·민감 작업만 Go API 강제. 이중 방어는 RLS + JWT 검증의 자연스러운 결과.

Create `apps/web/app/app/page.tsx`:
```tsx
// 인증·온보딩 redirect는 미들웨어가 처리하므로 여기서는 단순 렌더만
import { createSupabaseServer } from "@/lib/supabase/server";

export default async function HomePage() {
  const supabase = await createSupabaseServer();
  const { data: { user } } = await supabase.auth.getUser();

  const { data: profile } = await supabase
    .from("profiles")
    .select("display_name")
    .eq("id", user!.id)
    .single();

  return (
    <div className="p-8">
      <h1 className="font-mono text-2xl mb-2">홈</h1>
      <p className="text-fg-muted">환영합니다{profile?.display_name ? `, ${profile.display_name}` : ""}. 대시보드는 W3에서 구현 예정.</p>
    </div>
  );
}
```

- [ ] **Step 6: 500 에러 페이지**

Create `apps/web/app/error.tsx`:
```tsx
"use client";
import { useEffect } from "react";
import * as Sentry from "@sentry/nextjs";

export default function Error({ error, reset }: { error: Error & { digest?: string }; reset: () => void }) {
  useEffect(() => { Sentry.captureException(error); }, [error]);

  // 사용자에게는 내부 정보 노출 금지 — generic 메시지만
  return (
    <main className="min-h-screen flex items-center justify-center px-6">
      <div className="max-w-md text-center space-y-4">
        <h1 className="font-mono text-3xl text-bb-down">500</h1>
        <p className="text-fg-muted text-sm">예상치 못한 오류가 발생했습니다. 잠시 후 다시 시도해주세요.</p>
        {error.digest && <p className="text-fg-subtle text-[10px] font-mono">{error.digest}</p>}
        <button onClick={reset} className="border border-line px-4 py-2 font-mono text-sm">
          다시 시도
        </button>
      </div>
    </main>
  );
}
```

- [ ] **Step 7: 동작 확인**

로그인 후 `/app` 접근 → 셸 렌더링 확인. 미인증 시 `/login` redirect.

- [ ] **Step 8: 커밋**

```bash
git add apps/web/
git commit -m "feat(web): 앱 셸 (사이드바·티커·상태바) + 인증 가드"
```

---

## Task 12: 온보딩 wizard

**Files:**
- Create: `apps/web/components/onboarding/StepIndicator.tsx`
- Create: `apps/web/components/onboarding/CurrencyStep.tsx`
- Create: `apps/web/components/onboarding/DemoOrStartStep.tsx`
- Create: `apps/web/components/onboarding/Wizard.tsx`
- Create: `apps/web/app/app/onboarding/page.tsx`

- [ ] **Step 1: StepIndicator (1/3 · 2/3 · 3/3)**

Create `apps/web/components/onboarding/StepIndicator.tsx`:
```tsx
export function StepIndicator({ current, total }: { current: number; total: number }) {
  return (
    <div className="flex gap-1">
      {Array.from({ length: total }).map((_, i) => (
        <div
          key={i}
          className={`h-[2px] flex-1 ${i < current ? "bg-bb-accent" : "bg-line"}`}
        />
      ))}
    </div>
  );
}
```

- [ ] **Step 2: CurrencyStep**

Create `apps/web/components/onboarding/CurrencyStep.tsx`:
```tsx
"use client";
export function CurrencyStep({
  value, onChange, onNext,
}: { value: "KRW" | "USD"; onChange: (v: "KRW" | "USD") => void; onNext: () => void }) {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="font-mono text-2xl">기본 통화</h2>
        <p className="text-fg-muted text-sm mt-1">자산 평가액·수익률 표시 통화입니다. 추후 설정에서 변경 가능합니다.</p>
      </div>
      <div className="grid grid-cols-2 gap-3">
        {(["KRW", "USD"] as const).map((c) => (
          <button key={c}
            onClick={() => onChange(c)}
            className={`border p-6 font-mono text-2xl ${value === c ? "border-bb-accent text-bb-accent" : "border-line text-fg"}`}
          >
            {c}
          </button>
        ))}
      </div>
      <button onClick={onNext} className="w-full bg-bb-accent text-bg font-mono py-2">다음</button>
    </div>
  );
}
```

- [ ] **Step 3: DemoOrStartStep**

Create `apps/web/components/onboarding/DemoOrStartStep.tsx`:
```tsx
"use client";
export function DemoOrStartStep({
  onDemo, onStart, loading,
}: { onDemo: () => void; onStart: () => void; loading: boolean }) {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="font-mono text-2xl">시작 방법</h2>
        <p className="text-fg-muted text-sm mt-1">데모 데이터로 둘러보거나, 빈 포트폴리오로 시작합니다.</p>
      </div>
      <div className="space-y-3">
        <button onClick={onDemo} disabled={loading} className="w-full border border-line p-4 text-left disabled:opacity-50">
          <div className="font-mono">데모 포트폴리오로 시작</div>
          <div className="text-fg-muted text-xs mt-1">샘플 종목 5개가 자동 입력됩니다 (W3에서 활성).</div>
        </button>
        <button onClick={onStart} disabled={loading} className="w-full border border-line p-4 text-left disabled:opacity-50">
          <div className="font-mono">빈 상태로 시작</div>
          <div className="text-fg-muted text-xs mt-1">직접 자산을 추가합니다.</div>
        </button>
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Wizard (상태 + Supabase 호출)**

Create `apps/web/components/onboarding/Wizard.tsx`:
```tsx
"use client";
import { useState } from "react";
import { useRouter } from "next/navigation";
import { createSupabaseBrowser } from "@/lib/supabase/client";
import { StepIndicator } from "./StepIndicator";
import { CurrencyStep } from "./CurrencyStep";
import { DemoOrStartStep } from "./DemoOrStartStep";

export function Wizard() {
  const [step, setStep] = useState(1);
  const [currency, setCurrency] = useState<"KRW" | "USD">("KRW");
  const [loading, setLoading] = useState(false);
  const router = useRouter();

  const [err, setErr] = useState<string | null>(null);

  async function complete(demo: boolean) {
    setLoading(true);
    setErr(null);
    const supabase = createSupabaseBrowser();
    const { data: { user } } = await supabase.auth.getUser();
    if (!user) { router.push("/login"); return; }

    const { error } = await supabase.from("profiles").update({
      base_currency: currency,
      onboarding_completed: true,
    }).eq("id", user.id);

    if (error) {
      setErr("프로필 저장에 실패했습니다. 잠시 후 다시 시도해주세요.");
      setLoading(false);
      return;
    }

    // demo seeding은 W3에서 holdings 구현 후 추가. 현재는 flag만 통과.
    void demo;

    router.push("/app");
    router.refresh();
  }

  return (
    <main className="min-h-screen flex flex-col">
      <div className="border-b border-line p-4 font-mono text-xs text-fg-muted">
        ONBOARDING — {step}/2
      </div>
      <StepIndicator current={step} total={2} />
      <div className="flex-1 flex items-center justify-center px-6">
        <div className="w-full max-w-md">
          {step === 1 && (
            <CurrencyStep
              value={currency}
              onChange={setCurrency}
              onNext={() => setStep(2)}
            />
          )}
          {step === 2 && (
            <DemoOrStartStep
              onDemo={() => complete(true)}
              onStart={() => complete(false)}
              loading={loading}
            />
          )}
          {err && <p className="text-bb-down text-xs mt-4 font-mono">{err}</p>}
        </div>
      </div>
    </main>
  );
}
```

> **스펙 deviation (M1)**: 스펙 §6은 3단계(통화·첫 자산 추가·데모 옵션). W1은 holdings API 미구현이므로 2단계로 단축 (통화·데모/시작). W3에 첫 자산 추가 단계를 추가하여 3단계로 복원 예정.

- [ ] **Step 5: 페이지 연결**

Create `apps/web/app/app/onboarding/page.tsx`:
```tsx
import { redirect } from "next/navigation";
import { createSupabaseServer } from "@/lib/supabase/server";
import { Wizard } from "@/components/onboarding/Wizard";

export default async function OnboardingPage() {
  const supabase = await createSupabaseServer();
  const { data: { user } } = await supabase.auth.getUser();
  if (!user) redirect("/login");

  const { data: profile } = await supabase
    .from("profiles")
    .select("onboarding_completed")
    .eq("id", user.id)
    .single();

  if (profile?.onboarding_completed) redirect("/app");

  return <Wizard />;
}
```

- [ ] **Step 6: 흐름 확인**

신규 가입 → 인증 → `/app` 진입 시 → `/app/onboarding` redirect → 2단계 완료 → `/app` 진입.

- [ ] **Step 7: 커밋**

```bash
git add apps/web/
git commit -m "feat(web): 온보딩 wizard 2단계 (통화·데모/시작)"
```

---

## Task 13: Sentry + PostHog 통합

**Files:**
- Create: `apps/web/lib/obs/posthog.tsx`
- Create: `apps/web/sentry.client.config.ts`
- Create: `apps/web/sentry.server.config.ts`
- Create: `apps/web/sentry.edge.config.ts`
- Create: `apps/web/instrumentation.ts`
- Create: `apps/api/internal/obs/sentry.go`
- Modify: `apps/web/app/layout.tsx`
- Modify: `apps/api/cmd/server/main.go`

- [ ] **Step 1: Next.js Sentry SDK 수동 설치 (wizard 대화형 → subagent 환경 비호환)**

```bash
cd apps/web
npm install @sentry/nextjs
```

Create `apps/web/sentry.client.config.ts`:
```ts
import * as Sentry from "@sentry/nextjs";
Sentry.init({
  dsn: process.env.NEXT_PUBLIC_SENTRY_DSN_WEB,
  tracesSampleRate: 0.1,
  replaysSessionSampleRate: 0,
  replaysOnErrorSampleRate: 1.0,
});
```

Create `apps/web/sentry.server.config.ts` and `apps/web/sentry.edge.config.ts` (동일 골격, edge config은 replays 제외).

Create `apps/web/instrumentation.ts`:
```ts
export async function register() {
  if (process.env.NEXT_RUNTIME === "nodejs") {
    await import("./sentry.server.config");
  }
  if (process.env.NEXT_RUNTIME === "edge") {
    await import("./sentry.edge.config");
  }
}
```

Update `apps/web/next.config.mjs` to wrap with `withSentryConfig`:
```js
import { withSentryConfig } from "@sentry/nextjs";
const nextConfig = {};
export default withSentryConfig(nextConfig, {
  silent: true,
  org: process.env.SENTRY_ORG,
  project: process.env.SENTRY_PROJECT,
});
```

- [ ] **Step 2: PostHog provider**

```bash
npm install posthog-js
```

Create `apps/web/lib/obs/posthog.tsx`:
```tsx
"use client";
import posthog from "posthog-js";
import { PostHogProvider } from "posthog-js/react";
import { useEffect, useState } from "react";

export function PHProvider({ children }: { children: React.ReactNode }) {
  const [ready, setReady] = useState(false);

  useEffect(() => {
    const key = process.env.NEXT_PUBLIC_POSTHOG_KEY;
    if (!key) { setReady(true); return; }
    posthog.init(key, {
      api_host: process.env.NEXT_PUBLIC_POSTHOG_HOST || "https://us.i.posthog.com",
      capture_pageview: true,
      persistence: "localStorage",
    });
    setReady(true);
  }, []);

  if (!ready) return <>{children}</>;
  return <PostHogProvider client={posthog}>{children}</PostHogProvider>;
}
```

> **C7 보강**: PostHog init은 모듈 최상단이 아닌 `useEffect` 내부에서만 (SSR/multi-import 안전).

Update `apps/web/app/layout.tsx`:
```tsx
import { PHProvider } from "@/lib/obs/posthog";
// ...
export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ko" className="dark">
      <body>
        <PHProvider>{children}</PHProvider>
      </body>
    </html>
  );
}
```

- [ ] **Step 3: Go Sentry SDK**

```bash
cd ../api
go get github.com/getsentry/sentry-go
```

Create `apps/api/internal/obs/sentry.go`:
```go
package obs

import (
	"log/slog"
	"time"

	"github.com/getsentry/sentry-go"
)

func InitSentry(dsn, env string) error {
	if dsn == "" {
		slog.Info("sentry disabled (no DSN)")
		return nil
	}
	return sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      env,
		EnableTracing:    true,
		TracesSampleRate: 0.1,
	})
}

func Flush() {
	sentry.Flush(2 * time.Second)
}
```

Update `apps/api/cmd/server/main.go` (main 시작 부분):
```go
if err := obs.InitSentry(cfg.SentryDSN, cfg.Env); err != nil {
	logger.Error("sentry init failed", "err", err)
}
defer obs.Flush()
```
import `"github.com/<user>/quotient/apps/api/internal/obs"`.

- [ ] **Step 4: 동작 확인**

Next.js:
```tsx
// 임시로 app/page.tsx에 <button onClick={() => { throw new Error("test") }}>Test Sentry</button> 추가 후 클릭
// Sentry 대시보드에서 이벤트 수신 확인 후 제거
```

Go:
```bash
# 임시로 healthz에 sentry.CaptureMessage("test") 추가 후 curl, 확인 후 제거
```

- [ ] **Step 5: 커밋**

```bash
cd ../..
git add apps/web/ apps/api/
git commit -m "feat(obs): Sentry + PostHog 통합"
```

---

## Task 14: Fly.io + Vercel 배포 설정

**Files:**
- Create: `apps/api/Dockerfile`
- Create: `apps/api/fly.toml`
- Create: `apps/web/vercel.json` (선택)

- [ ] **Step 1: Dockerfile (multi-stage)**

Create `apps/api/Dockerfile`:
```dockerfile
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Seoul
COPY --from=build /out/server /server
EXPOSE 8080
CMD ["/server"]
```

- [ ] **Step 2: fly.toml**

Create `apps/api/fly.toml`:
```toml
app = "quotient-api-staging"
primary_region = "nrt"

[build]

[env]
  API_PORT = "8080"
  API_ENV = "staging"

[http_service]
  internal_port = 8080
  force_https = true
  auto_stop_machines = true
  auto_start_machines = true
  min_machines_running = 0
  processes = ["app"]

  [[http_service.checks]]
    grace_period = "10s"
    interval = "30s"
    method = "GET"
    timeout = "5s"
    path = "/healthz"

[[vm]]
  memory = "256mb"
  cpu_kind = "shared"
  cpus = 1
```

- [ ] **Step 3: Fly secrets 등록**

```bash
cd apps/api
fly secrets set \
  DATABASE_URL="<staging supabase pooler URL>" \
  SUPABASE_JWT_SECRET="<staging>" \
  SENTRY_DSN_API="<api dsn>" \
  CORS_ORIGIN="https://quotient-web.vercel.app" \
  -a quotient-api-staging
```

- [ ] **Step 4: 첫 배포**

```bash
fly deploy -a quotient-api-staging --remote-only
curl https://quotient-api-staging.fly.dev/healthz
```
Expected: `{"status":"ok"}`

- [ ] **Step 5: Vercel 프로젝트 연결 + env 등록 후 배포** (env 누락 시 빌드 실패)

```bash
cd ../web
vercel link
# 프로젝트 이름: quotient-web
```

**먼저** Vercel Dashboard → Project → Settings → Environment Variables 에서 다음을 모두 추가 (Preview·Production 둘 다):
- `NEXT_PUBLIC_SUPABASE_URL`
- `NEXT_PUBLIC_SUPABASE_ANON_KEY`
- `NEXT_PUBLIC_POSTHOG_KEY`
- `NEXT_PUBLIC_POSTHOG_HOST`
- `NEXT_PUBLIC_SENTRY_DSN_WEB`
- `SENTRY_ORG`, `SENTRY_PROJECT`, `SENTRY_AUTH_TOKEN` (sourcemap 업로드용)
- `PAYMENTS_ENABLED=false`
- `ENABLE_ADS=false`

**그 다음** 배포:
```bash
vercel deploy --prod=false  # preview 먼저 확인
vercel deploy --prod
```

- [ ] **Step 6: 커밋**

```bash
cd ../..
git add apps/api/Dockerfile apps/api/fly.toml
git commit -m "chore(deploy): Fly.io Dockerfile + fly.toml + Vercel 연결"
```

---

## Task 15: GitHub Actions CI/CD

**Files:**
- Create: `.github/workflows/ci.yml`
- Create: `.github/workflows/deploy.yml`

- [ ] **Step 1: CI 워크플로우**

Create `.github/workflows/ci.yml`:
```yaml
name: CI

on:
  pull_request:
  push:
    branches: [main, develop]

jobs:
  api:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_PASSWORD: postgres
        ports: ["5432:5432"]
        options: --health-cmd pg_isready --health-interval 10s --health-timeout 5s --health-retries 5
    defaults:
      run:
        working-directory: apps/api
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.23" }
      - run: go mod download
      - run: go build ./...
      - run: go test ./...
        env:
          DATABASE_URL: postgresql://postgres:postgres@localhost:5432/postgres
          SUPABASE_JWT_SECRET: test-secret

  web:
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: apps/web
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: "20", cache: "npm", cache-dependency-path: apps/web/package-lock.json }
      - run: npm ci
      - run: npm run lint
      - run: npm run build
        env:
          NEXT_PUBLIC_SUPABASE_URL: http://localhost:54321
          NEXT_PUBLIC_SUPABASE_ANON_KEY: dummy
```

- [ ] **Step 2: 배포 워크플로우**

Create `.github/workflows/deploy.yml`:
```yaml
name: Deploy

on:
  push:
    branches: [main, develop]

jobs:
  api:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: superfly/flyctl-actions/setup-flyctl@master
      - run: |
          APP=${{ github.ref == 'refs/heads/main' && 'quotient-api-prod' || 'quotient-api-staging' }}
          flyctl deploy --remote-only -a $APP
        working-directory: apps/api
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}

  web:
    runs-on: ubuntu-latest
    # Vercel은 git 통합으로 자동 배포되므로 별도 step 불필요.
    # 통합이 없는 경우 vercel CLI step 추가.
    steps:
      - run: echo "Vercel auto-deploy via git integration."
```

- [ ] **Step 3: Repo secrets 등록**

GitHub Dashboard → Settings → Secrets → Actions:
- `FLY_API_TOKEN` = `fly auth token` 출력값

- [ ] **Step 4: 푸시 + CI 동작 확인**

```bash
git add .github/
git commit -m "ci: GitHub Actions CI + Fly 배포 워크플로우"
git push -u origin main
```

GitHub Actions 탭에서 CI 통과 + Fly 배포 성공 확인.

- [ ] **Step 5: 추가 커밋 (필요 시)**

CI 실패하면 수정 후 재푸시.

---

## Task 16: 통합 동작 검증 (수동 E2E)

다음을 직접 수행하여 W1 완성을 검증:

- [ ] **검증 1**: 신규 가입 (`/signup`) → 이메일 인증 메일 수신 → 링크 클릭 → `/app/onboarding` 진입
- [ ] **검증 2**: 온보딩 wizard 2단계 완주 → `/app` 진입 → 셸 렌더링 (사이드바·티커 placeholder·상태바·"홈" placeholder)
- [ ] **검증 3**: 로그아웃 → `/login` → 같은 계정 로그인 → 곧바로 `/app` (재인증·재온보딩 없음)
- [ ] **검증 4**: 비밀번호 재설정 (`/forgot-password`) → 메일 → `/reset-password` → 변경 → 로그인 성공
- [ ] **검증 5**: Google OAuth 로그인 → `/app/onboarding` → 완료 → `/app`
- [ ] **검증 6**: Go API `/healthz`·`/v1/profile` (Authorization 헤더 포함) 호출 → 200
- [ ] **검증 7**: PostHog 대시보드에 페이지뷰 이벤트 수신 확인
- [ ] **검증 8**: Sentry에 임시 에러 발송 확인 (검증 후 제거)

검증 8개 모두 통과 시 W1 종료.

- [ ] **최종 커밋 / STATUS·ROADMAP 갱신**

```bash
# docs/STATUS.md에 W1 완료 항목 ✅ 처리, 변경 이력 한 줄 추가
# docs/ROADMAP.md에 W1 작업 제거, "현재 추천 다음 작업" → W2
git add docs/
git commit -m "docs: W1 완료 — STATUS·ROADMAP 갱신"
```

---

## 자체 검토 + Subagent 검토 (2026-05-22)

### 인라인 self-review
**스펙 커버리지 (W1)**: Supabase 스키마·RLS (Task 2·3), 백엔드 스캐폴딩 (Task 4-6), 가입·로그인·OAuth (Task 9), 비밀번호 재설정 (Task 10), 앱 셸 (Task 11), 온보딩 (Task 12), Sentry+PostHog (Task 13), 배포 (Task 14), CI/CD (Task 15), 법적 페이지 (Task 7). 전체 커버.

**Placeholder 없음.** `<user>` 만 GitHub 사용자명으로 치환 필요 (명시).

### Subagent 검토 (general-purpose agent — 2026-05-22)

**Verdict: READY WITH PATCHES** — 9 Critical + 12 Important + 6 Minor 식별. 본 plan에 모두 반영 완료.

| 우선순위 | 항목 | 처리 |
|---|---|---|
| C1 | Next.js CLI 플래그 무효 | Task 7 Step 1 패치 |
| C2 | `tailwindcss-animate` 미설치 + 설정 전체 교체 | Task 7 Step 2·4 패치 (merge 방식) |
| C3 | Next.js 15 `searchParams` Promise | Task 9 Step 6 패치 (async + await) |
| C4 | 온보딩 가드 deep link 우회 | Task 11 Step 5 패치 (middleware로 이동) |
| C5 | layout·page에서 `getUser`·`profiles` 중복 호출 | layout 단순화 + middleware 일원화로 해소 |
| C6 | Wizard 업데이트 결과 무시 | Task 12 Step 4 패치 (`error` 처리) |
| C7 | PostHog 모듈 최상단 init | Task 13 Step 2 패치 (useEffect 기반) |
| C8 | HS256 vs Supabase 신규 비대칭 키 | Task 0-A에 Legacy JWT Secret 활성화 명시 + JWKS 마이그레이션 백로그 |
| C9 | Wizard Supabase 직접 vs Go API 일관성 | Task 11 결정 명시 (메타데이터 직접, 민감 작업 Go) |
| I1 | Bearer 대소문자 | Task 5 Step 5 패치 (`strings.EqualFold`) |
| I2 | `cosmtrek/air` 아카이브 | 전역 치환 `air-verse/air` |
| I3 | `motion` 버전 핀 | Task 7 Step 2 `motion@^11` |
| I4 | Sentry wizard 대화형 | Task 13 Step 1 수동 설치로 변경 |
| I5 | CORS 미들웨어 task 없음 | Task 5 Step 10 추가 |
| I6 | Profile 핸들러 테스트가 middleware 우회 | W2 testcontainers 도입 시 보강 (백로그) |
| I7 | npm workspaces 없이 monorepo | MVP 단순화, 각 앱 독립 설치 — 의도된 결정 |
| I8 | OAuth 콜백 origin (Vercel 프록시) | Supabase Site URL 정확 설정으로 우회 (Task 0-G에 추가 필요 — 백로그) |
| I9 | PIPA 분리 동의 + 14세 차단 | Task 9 Step 2 패치 (별도 체크박스 3개) |
| I10 | `react-markdown` + `rehype-sanitize` | W4 plan에서 도입 (백로그) |
| I11 | Vercel env 미입력 후 배포 | Task 14 Step 5 패치 (env 먼저, 배포 나중) |
| I12 | prod Supabase 프로젝트 W1 생성 | Task 0-A 패치 (staging만, prod는 W6) |
| M1 | Wizard 2단계 vs 스펙 3단계 | Task 12 deviation 명시 |
| M2 | `/pricing`에 "AI 채팅" 카피 | Task 7 Step 7 패치 ("분석가 채팅") |
| M3 | `bg-bg` 별칭 가독성 | skip (관용 표현, 추가 패치 불요) |
| M4 | `error.tsx` 원본 메시지 노출 | Task 11 Step 6 패치 (generic 메시지) |
| M5 | Pretendard 미로딩 | Task 7 Step 2 `pretendard` 패키지 추가 (사용은 `var(--font-pretendard)` 설정으로 root layout에서 next/font 통해 로드 — 별도 미세 작업) |
| M6 | Day-by-day 일정표 | 백로그 (실행 시 작성자가 배분) |

### 백로그 (Phase 2 또는 후속 plan에서 처리)

- JWKS 기반 JWT 검증 마이그레이션 (Supabase 비대칭 키 표준 정착 시)
- Profile handler 통합 테스트 (testcontainers-go 도입 후)
- OAuth 콜백 forwarded host 처리 정밀화
- `react-markdown` + `rehype-sanitize` (W4 plan)
- next/font 통합으로 Pretendard 로딩 최적화
- W2~W6 plan에서 동일 검토 사이클 반복

## 다음 단계

이 plan에 대해 사용자 directive에 따라 **subagent 자체 검토**를 요청합니다 (`general-purpose` agent에 위임).

이후 사용자 검토 → 승인 → 실행 옵션 선택:

1. **Subagent-Driven (추천)** — task별 fresh subagent 디스패치, task 사이 검토. `superpowers:subagent-driven-development` 사용.
2. **Inline Execution** — 현재 세션에서 batch + checkpoint. `superpowers:executing-plans` 사용.

W1 완료 후 → W2 plan 작성 → 동일 사이클.
