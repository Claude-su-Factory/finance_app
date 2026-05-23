# Quotient — Portfolio Intelligence Terminal

개인 운영 금융 SaaS. 한국·미국 자산 통합 분석 + 자연어 분석가 인터페이스.
블룸버그 터미널의 정보 밀도·미감을 개인 가격대로.

> 개발자·파워유저 타겟. 마이데이터·자금 보관 없음 — 데이터·분석만 제공.

## 기술 스택

- **백엔드**: Go 1.25 + chi v5 + pgx v5 + `robfig/cron/v3` (API 서버 + 데이터 워커 단일 바이너리)
- **프론트**: Next.js 16 + Tailwind v4 + shadcn/base-ui + TypeScript
- **DB·인증**: Supabase Postgres + RLS + Supabase Auth (Google OAuth)
- **AI**: Anthropic Claude (Sonnet 4.6 기본 + Opus 4.7 심층 + Haiku 4.5 내부)
- **데이터 소스**: KIND(KRX), Yahoo Finance, frankfurter.dev, FRED, 한국은행 ECOS

## 디렉토리 구조

```
finance/
├── apps/
│   ├── api/           Go 백엔드 (API + cron 워커 + 백필 CLI)
│   │   ├── cmd/       server, backfill 진입점
│   │   └── internal/  handlers, schedule, ingest, sources, models, middleware
│   └── web/           Next.js 프론트
│       ├── app/       App Router 페이지 (랜딩·인증·앱 쉘·포트폴리오)
│       ├── components/  shell, portfolio, home, onboarding, ui
│       └── lib/       api 클라이언트 + supabase SSR
├── supabase/
│   ├── migrations/    Postgres 마이그레이션 + RLS 정책
│   └── config.toml    로컬 Supabase 설정
└── docs/
    ├── STATUS.md          현재 구현 상태
    ├── ROADMAP.md         다음 작업
    ├── ARCHITECTURE.md    핵심 설계 결정 (Why 포함)
    ├── AGENTS.md          에이전트 디스패치 규칙
    └── superpowers/
        ├── specs/         기능별 상세 설계
        └── plans/         기능별 구현 계획
```

## 로컬 개발

### 사전 요구

| 도구 | 버전 | 설치 |
|---|---|---|
| Go | 1.25+ (pgx/v5 v5.9.2 요구) | `brew install go` |
| Node | 20+ | `brew install node@20` |
| Supabase CLI | latest | `brew install supabase/tap/supabase` |
| Air (Go 핫리로드) | latest | `go install github.com/air-verse/air@latest` |
| Docker Desktop | — | 로컬 Supabase Postgres 컨테이너 구동용 |

### 환경 변수 설정

루트와 `apps/web/`에 각각 env 파일이 필요합니다 (둘 다 `.gitignore`에 등록됨).

```bash
# 루트: 백엔드 + 공통
cp .env.example .env

# 프론트: 클라이언트 사이드 변수
cp apps/web/.env.example apps/web/.env.local
```

채워야 할 값:

- **`.env`** (루트)
  - `DATABASE_URL` — 로컬은 `postgresql://postgres:postgres@127.0.0.1:54322/postgres`
  - `SUPABASE_JWT_SECRET` — `supabase status` 출력 또는 Supabase Dashboard → Auth → Settings → Legacy JWT Secret
  - `FRED_API_KEY` — https://fred.stlouisfed.org/docs/api/api_key.html (무료)
  - `ECOS_API_KEY` — https://ecos.bok.or.kr/api/ (한국은행, 무료 등록)
- **`apps/web/.env.local`**
  - `NEXT_PUBLIC_SUPABASE_URL` — 로컬은 `http://127.0.0.1:54321`
  - `NEXT_PUBLIC_SUPABASE_ANON_KEY` — `supabase status` 출력의 `anon key`

> Supabase CLI v2.98 이상은 Legacy JWT Secret를 기본 숨김 처리합니다. Dashboard에서 "Legacy JWT Secret" 활성화 필요.

### 셋업·실행

```bash
make db-up      # 로컬 Supabase (Docker 컨테이너) 띄우기
make migrate    # 마이그레이션 적용
make api        # 터미널 1 — Go API + cron 워커
make web        # 터미널 2 — Next.js 15
```

접속: http://localhost:3000

기타 명령:

```bash
make db-reset   # DB 초기화 + 마이그레이션 재실행 (스키마 변경 후)
make db-down    # Supabase 중지
make test       # Go + Web 테스트 일괄
make lint       # golangci-lint + Next.js lint
```

### 5년 가격 백필 (선택)

cron이 갱신하기 시작한 시점부터 누적되므로, 초기에 과거 데이터가 필요하면:

```bash
cd apps/api && go run ./cmd/backfill -market KOSPI -years 5
cd apps/api && go run ./cmd/backfill -market KOSDAQ -years 5
cd apps/api && go run ./cmd/backfill -market NASDAQ -years 5
```

## 문서

| 문서 | 내용 |
|---|---|
| [STATUS](docs/STATUS.md) | 어디까지 구현됐는가 + 알려진 결함 |
| [ROADMAP](docs/ROADMAP.md) | 다음 작업 + Phase별 우선순위 |
| [ARCHITECTURE](docs/ARCHITECTURE.md) | 핵심 설계 결정 (Why 중심) |
| [AGENTS](docs/AGENTS.md) | 에이전트 팀 구성 |
| [MVP 설계 스펙](docs/superpowers/specs/2026-05-22-quotient-mvp-design.md) | 데이터 모델·UI·운영 전체 명세 |

## 라이선스

이 저장소는 학습·포트폴리오 용도로 공개됩니다. 운영 배포 인스턴스의 데이터·사용자 정보와는 별개입니다.
