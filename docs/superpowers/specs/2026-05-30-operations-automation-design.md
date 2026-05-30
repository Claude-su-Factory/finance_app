# 운영 자동화 설계 (A+B 통합)

> 목표: 사용자 액션을 "계정·키 1회 셋업"으로 압축한다. 수동 백필 CLI 실행 의무와 `supabase db push` 수동 단계를 자동화로 대체한다.

## 배경

현재 신규 환경을 띄우려면 사용자가 직접 다음을 실행해야 한다 (`docs/USER_ACTIONS.md`):

- 🔴 시급 #1 — `go run ./cmd/backfill --market=INDICES --years=5` (알파 카드 전제)
- 🔴 시급 #2 — `go run ./cmd/backfill --market=NASDAQ --years=5` (백테스트 전제)
- 🟡 배포 — `flyctl ssh console -C "/app/backfill --market=INDICES --years=5"` (production 지수 백필)
- 🟡 배포 — `supabase db push` (스키마 마이그레이션 적용)

이 네 가지가 "코드 외 운영 명령"으로 남아 있어 셋업을 무겁게 만든다. 본 설계는 이를 두 자동화로 제거한다.

- **A. 부팅 시 지수 자동 백필** — server 부팅에서 비어 있는 시리즈를 비동기로 채운다.
- **B. Fly `release_command` 마이그레이션** — 배포 시 Go 마이그레이터가 미적용 마이그레이션을 적용한다.

A와 B는 한 스펙에 담되 **독립 구현 가능**하게 분리한다. 동시에 자연스러운 선후 의존이 있다: B가 instrument 행(지수·FX)을 시드하는 마이그레이션을 적용 → A가 그 instrument의 5년 봉을 백필. `release_command`는 앱 부팅 전에 실행되므로 순서가 보장된다.

## 비범위 (Out of Scope)

- FX 시계열 백필 — `runIndices`는 `asset_class='INDEX'`만 다룬다(기존 동작). FX(`asset_class='FX'`) 히스토리는 본 자동화가 건드리지 않는다. 부팅 백필은 기존 수동 `--market=INDICES` 명령과 **정확히 동일한 범위**를 재현한다. (함의: 알파 카드가 시점별 환율을 FX 시계열에 의존한다면 이는 본 자동화로 해결되지 않는 **선재 이슈**다 — 수동 명령도 동일했음. plan에서 알파 카드의 FX 소스를 확인하고, 별개 이슈면 STATUS "알려진 결함"에 등재한다.)
- KR 전종목(KOSPI/KOSDAQ 개별 종목) 부팅 시드 — 🔴 시급 항목이 아니므로 제외. CLI `--market=KOSPI|KOSDAQ`로 수동 유지.
- Supabase 마이그레이션 생성·작성 워크플로우 변경 — 파일은 계속 `supabase/migrations/`에 사람이 작성한다. 본 설계는 "적용"만 자동화한다.
- 마이그레이션 롤백(down) — 본 마이그레이터는 forward-only.

---

## A. 부팅 시 지수 자동 백필

### A-1. 로직 추출 (`cmd/backfill` → `internal/backfill`)

현재 `runKR`/`runUS`/`runIndices`는 `cmd/backfill/main.go`의 `package main`에 있어 server에서 호출 불가하다. 이 함수들은 `(ctx, *pgxpool.Pool, ...)` 시그니처로 `yahoo.NewClient()`·`kind.NewClient("")`를 내부에서 생성한다(외부 deps 주입 없음). 따라서 추출이 단순하다.

신규 패키지 `apps/api/internal/backfill`:

```go
package backfill

// nasdaqSeed (cmd/backfill에서 이동)
var nasdaqSeed = []struct{ Symbol, Name string }{ /* 30종목 */ }

// CLI·부팅 공용 시장별 전체 백필 (기존 run* 로직 그대로 이동)
func RunKR(ctx context.Context, pool *pgxpool.Pool, market string, years, limit int) error
func RunUS(ctx context.Context, pool *pgxpool.Pool, years, limit int) error
func RunIndices(ctx context.Context, pool *pgxpool.Pool, years, limit int) error

// 부팅 전용: 비어 있는 시리즈만 채움 (멱등 only-if-empty, per-series)
func SeedIfEmpty(ctx context.Context, pool *pgxpool.Pool, years int) error

// 시리즈 단위 봉 존재 여부
func hasBars(ctx context.Context, pool *pgxpool.Pool, instrumentID string) (bool, error)
```

`internal/backfill`은 `IndexYahooSymbol` 때문에 `internal/schedule`을 임포트한다. `internal/schedule`은 `internal/backfill`을 임포트하지 않으므로 순환 없음(plan에서 재확인).

`cmd/backfill/main.go`는 얇은 wrapper로 축소:

```go
func main() {
    // flag 파싱 (years/market/limit) + config.Load + db.New
    switch *market {
    case "KOSPI", "KOSDAQ": err = backfill.RunKR(ctx, pool, *market, *years, *limit)
    case "NASDAQ":          err = backfill.RunUS(ctx, pool, *years, *limit)
    case "INDICES":         err = backfill.RunIndices(ctx, pool, *years, *limit)
    }
    // err != nil → slog.Error + os.Exit(1)
}
```

CLI 동작은 변하지 않는다(전체 재백필 유지). 부팅의 only-if-empty 동작은 `SeedIfEmpty`에만 존재한다.

### A-2. 트리거 — 멱등 only-if-empty (per-series)

`SeedIfEmpty`는:

1. **NASDAQ instrument 시드 보장** — `ingest.UpsertInstruments(nasdaqSeed)` (멱등 upsert). INDEX instrument는 마이그레이션 `20260522000003`이 이미 시드하므로 별도 시드 불필요.
2. 대상 instrument 조회 — `asset_class='INDEX'` + `exchange='NASDAQ'` 활성 종목.
3. 각 instrument에 대해 `hasBars`로 봉 존재 확인 → **0행인 시리즈만** Yahoo fetch + `ingest.UpsertPrices`. 이미 있으면 스킵.

per-series 단위라 부분 실패(예: SPX는 채웠고 AAPL은 실패)도 다음 부팅에서 실패분만 재시도된다(cross-boot self-heal). 시리즈 루프는 `RunUS`/`RunIndices`의 per-symbol 본문을 공유 헬퍼로 추출해 재사용한다(200ms Yahoo rate-limit sleep 포함).

```go
func backfillSymbol(ctx context.Context, pool *pgxpool.Pool, yc *yahoo.Client,
    instrumentID, yahooSymbol string, start, end time.Time) (int, error)
```

### A-3. 실행 모델 — 비동기·실패 무시

`cmd/server/main.go`에서 `schedule.Start(...)` 직후 고루틴 디스패치:

```go
go func() {
    if err := backfill.SeedIfEmpty(ctx, pool, 5); err != nil {
        logger.Error("boot backfill failed (continuing)", "err", err)
        observability.CaptureException(err) // 신규 헬퍼 (아래 참조)
    }
}()
```

`observability.CaptureException`는 **현재 존재하지 않는다**. 코드베이스의 Sentry는 HTTP 미들웨어(`SentryMiddleware`)로만 연결돼 있어 비-HTTP 경로(부팅 백필·cron)에서 에러를 보고할 수단이 없다. 따라서 `internal/observability/sentry.go`에 얇은 헬퍼를 추가한다(`InitSentry`가 초기화한 global hub 사용):

```go
// CaptureException은 비-HTTP 경로(부팅 백필 등)에서 에러를 Sentry로 보고한다. nil-safe.
func CaptureException(err error) {
    if err == nil {
        return
    }
    sentry.CaptureException(err)
}
```

- `ListenAndServe`·readiness를 **절대 차단하지 않는다**. 부팅 백필은 순수 백그라운드.
- 실패 시 로그 + Sentry capture, 서버 크래시 없음("실패 무시").
- 기존 server lifecycle `ctx`(line 47, shutdown 시 `cancel()`)를 그대로 전달 → graceful shutdown 시 진행 중 백필도 취소.
- min_machines_running=1 + auto_stop/start 환경에서 cold start마다 `SeedIfEmpty`가 호출되지만, 시드 완료 후엔 count 쿼리 몇 번의 cheap no-op.

### A-4. 범위

- INDICES — 마이그레이션이 시드한 INDEX instrument(현재 KOSPI·KOSDAQ·SPX·NDX 4종)의 5년 일봉.
- NASDAQ — `nasdaqSeed` 30종목의 5년 일봉.

---

## B. Fly `release_command` 마이그레이션

### B-1. 마이그레이터 (`internal/migrate`)

Supabase가 사용하는 동일 이력 테이블 `supabase_migrations.schema_migrations`를 공유해 divergence를 만들지 않는다.

```go
package migrate

type Migration struct {
    Version string // 파일명 숫자 prefix (예: "20260522000001")
    Name    string // 전체 파일명 (예: "20260522000001_profiles.sql")
    SQL     string // 파일 내용
}

// 순수 함수 (DB 불필요) — 단위 테스트 핵심
func parseMigrationFilename(name string) (version, label string, ok bool)
func pendingMigrations(available []Migration, applied map[string]bool) []Migration

// fs 읽기
func Load(dir string) ([]Migration, error) // *.sql 읽고 파일명 정렬

// DB
func ensureHistory(ctx, pool) error          // schema + table CREATE IF NOT EXISTS
func appliedVersions(ctx, pool) (map[string]bool, error)
func applyOne(ctx, pool, m Migration) error  // 트랜잭션: SQL exec + version INSERT

// 오케스트레이션
func Run(ctx context.Context, pool *pgxpool.Pool, dir string) error
```

`Run` 흐름:

1. `ensureHistory` — 스키마·테이블 보장:
   ```sql
   create schema if not exists supabase_migrations;
   create table if not exists supabase_migrations.schema_migrations (
     version text primary key,
     statements text[],
     name text
   );
   ```
2. `Load(dir)` → `appliedVersions` → `pendingMigrations` (정렬·미적용만).
3. 각 pending에 대해 `applyOne` — **마이그레이션당 트랜잭션**:
   ```
   BEGIN
     <파일 SQL 실행>
     INSERT INTO supabase_migrations.schema_migrations (version, name, statements)
       VALUES ($1, $2, $3) ON CONFLICT (version) DO NOTHING   -- statements = ARRAY[<파일 전체 SQL>]
   COMMIT
   ```
   실패 시 ROLLBACK + 에러 반환 → `Run`이 비0 exit → **배포 중단(fail-closed)**.
4. 모두 적용 완료 시 slog로 요약 로그.

버전 기반 스킵이므로 멱등이다. 동일 이력 테이블을 쓰므로 이후 `supabase db push`·Studio가 적용 상태를 정확히 인식한다.

제약·결정:

- **tx-안전성** — 마이그레이션당 단일 트랜잭션 래핑은 마이그레이션이 tx-안전(standalone `BEGIN;/COMMIT;` 없음, `CREATE INDEX CONCURRENTLY` 없음)일 때만 옳다. 현재 10개 마이그레이션은 모두 tx-안전임을 확인했다(파일 내 `begin/end`는 PL/pgSQL 함수 본문 `as $$ begin … end; $$`로 트랜잭션 제어가 아님). **향후 CONCURRENTLY 등 tx 불가 statement를 쓰는 마이그레이션이 생기면 no-tx 적용 경로가 필요**하다 — 이 제약을 `supabase/migrations` 작성 규칙으로 문서화한다.
- **`statements` 컬럼** — Supabase는 statement 단위로 분해해 저장하지만, 본 마이그레이터는 파일 전체 SQL을 1-element `text[]`(`ARRAY[<파일 SQL>]`)로 기록한다. `version` 기반 스킵에는 영향 없고 `statements`는 정보용이다. SQL을 정확히 statement 단위로 쪼개는 것(문자열·함수 본문 내 세미콜론 처리)은 불필요한 복잡도이므로 채택하지 않는다.

### B-2. 엔트리 (`cmd/migrate`)

```go
func main() {
    dir := flag.String("dir", "/app/migrations", "마이그레이션 .sql 디렉터리")
    flag.Parse()
    ctx := context.Background()
    // config.Load는 SUPABASE_JWT_SECRET 등 마이그레이션과 무관한 secret까지 required로 요구한다.
    // 마이그레이터는 DATABASE_URL만 필요 → 직접 읽어 decouple (release_command·로컬 실행 모두 단순화).
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        slog.Error("DATABASE_URL required")
        os.Exit(1)
    }
    pool, err := db.New(ctx, dsn) // Session pooler 5432 (DDL·tx 지원)
    if err != nil {
        slog.Error("db connect failed", "err", err)
        os.Exit(1)
    }
    defer pool.Close()
    if err := migrate.Run(ctx, pool, *dir); err != nil {
        slog.Error("migrate failed", "err", err)
        os.Exit(1) // release_command 실패 → Fly 배포 중단
    }
}
```

로컬: `go run ./cmd/migrate --dir=../../supabase/migrations` (`DATABASE_URL`만 env에 있으면 됨).

### B-3. 빌드 컨텍스트 — 레포 루트로 확장

`supabase/migrations`(레포 루트)를 이미지에 넣으려면 빌드 컨텍스트가 레포 루트여야 한다(현재는 `apps/api`). 변경:

**Dockerfile** (`apps/api/Dockerfile`, 경로를 레포 루트 기준으로 재작성):

```dockerfile
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download
COPY apps/api/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server   ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/backfill ./cmd/backfill
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate  ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/server   /app/server
COPY --from=build /out/backfill /app/backfill
COPY --from=build /out/migrate  /app/migrate
COPY supabase/migrations /app/migrations
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
```

**루트 `.dockerignore` 신규** — 원격 빌드 컨텍스트를 가볍게 유지:

```
.git
apps/web
**/node_modules
**/.next
**/*_test.go
**/testdata
**/*.md
.vscode
.idea
```

(기존 `apps/api/.dockerignore`는 빌드 컨텍스트가 루트로 바뀌면 적용되지 않으므로 루트 것으로 대체. 단 `*_test.go`·`testdata` 제외 규칙은 보존.)

**`apps/api/fly.toml`** — release_command 추가:

```toml
[deploy]
  release_command = "/app/migrate"
```

**`.github/workflows/deploy-api.yml`** — 빌드 컨텍스트·트리거 조정:

```yaml
on:
  push:
    branches: [master, main]
    paths:
      - "apps/api/**"
      - "supabase/migrations/**"   # 추가: 마이그레이션 변경도 release 트리거
      - ".github/workflows/deploy-api.yml"
jobs:
  deploy:
    steps:
      - uses: actions/checkout@v4
      - uses: superfly/flyctl-actions/setup-flyctl@master
      - name: deploy
        run: flyctl deploy --remote-only --config apps/api/fly.toml --dockerfile apps/api/Dockerfile
        # working-directory 제거 → 레포 루트가 빌드 컨텍스트
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
```

로컬 배포도 동일하게 레포 루트에서 `flyctl deploy --config apps/api/fly.toml --dockerfile apps/api/Dockerfile`.

---

## 파일 구조

| 구분 | 경로 | 책임 |
|---|---|---|
| 신규 | `apps/api/internal/backfill/backfill.go` | RunKR/RunUS/RunIndices + SeedIfEmpty + hasBars + backfillSymbol + nasdaqSeed |
| 신규 | `apps/api/internal/migrate/migrate.go` | Load·parse·pending·ensureHistory·applyOne·Run |
| 신규 | `apps/api/internal/migrate/migrate_test.go` | parseMigrationFilename·pendingMigrations 순수 단위 테스트 |
| 신규 | `apps/api/cmd/migrate/main.go` | --dir·DATABASE_URL 엔트리 |
| 수정 | `apps/api/cmd/backfill/main.go` | internal/backfill 호출로 축소 (run* 본문 이동) |
| 수정 | `apps/api/cmd/server/main.go` | schedule.Start 직후 SeedIfEmpty 고루틴 |
| 수정 | `apps/api/internal/observability/sentry.go` | `CaptureException(err)` 헬퍼 신규 (비-HTTP 경로용) |
| 수정 | `apps/api/Dockerfile` | 빌드 컨텍스트 루트화 + migrate 빌드 + migrations COPY |
| 수정 | `apps/api/fly.toml` | `[deploy] release_command` |
| 신규 | `.dockerignore` (루트) | 빌드 컨텍스트 슬림화 |
| 수정 | `.github/workflows/deploy-api.yml` | 빌드 컨텍스트·트리거 paths |
| 수정 | `docs/USER_ACTIONS.md` | 🔴 시급 2건 + 🟡 지수 백필 + `supabase db push` → 자동화로 이동 |
| 수정 | `docs/STATUS.md`·`ROADMAP.md`·`ARCHITECTURE.md` | 완료 반영 |

---

## 에러 핸들링

| 영역 | 실패 시 동작 | 근거 |
|---|---|---|
| A 부팅 백필 | 로그 + Sentry, 서버 정상 기동 유지. per-series라 다음 부팅에서 실패분 재시도 | ROADMAP "비동기·실패 무시" — 백필은 부가 데이터, 가용성 우선 |
| B 마이그레이션 | ROLLBACK + 비0 exit → Fly 배포 중단 | 스키마 정합성은 가용성보다 우선 — fail-closed가 안전 |

B의 fail-closed는 운영상 함의가 있다: 깨진 마이그레이션은 **모든 후속 배포를 막는다**. 이는 의도된 안전장치이며 `docs/DEPLOY.md`·`USER_ACTIONS.md`에 명시한다.

---

## 테스트 전략

- **`internal/migrate` (순수 핵심)** — `parseMigrationFilename`(숫자 prefix 추출, 비정상 파일명 거부), `pendingMigrations`(applied 제외·버전 정렬)를 DB 없이 단위 테스트. 실제 `supabase/migrations` 10개 파일명으로 정렬·파싱 검증.
- **DB 접촉부(`applyOne`·`ensureHistory`)** — 레포에 Go DB 테스트 하네스(testcontainers 등)가 없으므로 순수 로직 우선. `release_command` 자체가 첫 배포의 통합 검증 역할(멱등이라 재실행 안전).
- **`internal/backfill`** — `SeedIfEmpty`의 emptiness 판정·스킵은 외부 fetch(KIND/Yahoo)와 DB에 묶여 있어 기존처럼 수동 통합 검증. 추출 자체는 CLI 동작 보존(빌드·`go vet` 통과)으로 회귀 방지.
- **빌드** — `go build ./...`·`go vet ./...` 통과. Docker 빌드는 plan 단계에서 로컬 `docker build`로 컨텍스트 루트화 검증.

---

## 검토 이력

### 2026-05-30 — 작성 직후 자체 검토 (Critical/Important/Minor)

코드베이스 검증을 동반한 자체 검토. 패치 완료.

**Critical** — 없음 (설계 단계에서 선제 해소).
- 빌드 컨텍스트가 `apps/api`라 Dockerfile이 레포 루트 `supabase/migrations`에 닿지 못하는 문제 → 빌드 컨텍스트 레포 루트 확장(B-3)으로 해소.
- 마이그레이션 tx 래핑 안전성 → 현재 10개 전부 tx-안전 확인(함수 본문 `begin/end`, CONCURRENTLY 없음). B-1에 제약 명시.
- B(instrument 시드)→A(봉 백필) 선후 의존 → `release_command`가 부팅 전 실행되어 순서 보장 확인.

**Important** — 패치 완료.
1. `observability.CaptureException` 부재 — 코드베이스에 비-HTTP Sentry capture 경로가 전무(미들웨어만 연결). → `sentry.go`에 nil-safe 헬퍼 신규(A-3·파일 구조에 반영).
2. `cmd/migrate`의 `config.Load` 커플링 — `config.Load`는 `SUPABASE_JWT_SECRET`까지 `required`라 마이그레이션과 무관한 secret에 묶임. → `os.Getenv("DATABASE_URL")` 직접 읽기로 decouple(B-2 패치). release_command·로컬 실행 모두 단순화.
3. 알파 카드 FX 시계열 — 부팅 백필(INDICES)이 FX를 안 채움. 선재 이슈(수동 명령도 동일). → 비범위에 함의 명시 + plan에서 알파 카드 FX 소스 확인 의무.

**Minor** — 문서/주석으로 처리.
1. DJI 불일치 — USER_ACTIONS는 "KOSPI·KOSDAQ·SPX·NDX·DJI"라 하나 마이그레이션 시드는 4종(DJI 없음). 부팅 백필은 시드된 INDEX만 커버 → USER_ACTIONS 문구를 실제(4종)에 맞춰 정정(DJI 추가는 비범위).
2. `statements[]` 저장 granularity — 파일 전체를 1-element로 기록(B-1 명시). 정보용, 스킵 로직 무관.
3. tx-불가 마이그레이션 제약 — 향후 CONCURRENTLY 등 도입 시 no-tx 경로 필요. `supabase/migrations` 작성 규칙으로 문서화(B-1).
4. `SeedIfEmpty`가 cold start마다 실행 — 시드 후 count 쿼리 몇 번의 cheap no-op(A-3 명시).
