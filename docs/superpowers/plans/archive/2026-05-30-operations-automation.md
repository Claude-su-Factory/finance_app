# 운영 자동화 (A+B) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 신규 환경 셋업에서 수동 백필 CLI 실행과 `supabase db push`를 제거한다 — 부팅 시 비어 있는 지수·NASDAQ 시리즈를 비동기 자동 백필(A)하고, 배포 시 Fly `release_command`가 미적용 마이그레이션을 적용(B)한다.

**Architecture:** A는 `cmd/backfill`의 `run*` 로직을 `internal/backfill` 패키지로 추출하고 `SeedIfEmpty`(멱등 only-if-empty, per-series)를 추가해 server 부팅에서 fire-and-forget 고루틴으로 실행한다. B는 Supabase의 `supabase_migrations.schema_migrations` 이력 테이블을 공유하는 forward-only Go 마이그레이터(`internal/migrate` + `cmd/migrate`)를 만들고, Docker 빌드 컨텍스트를 레포 루트로 확장해 `supabase/migrations`를 이미지에 넣은 뒤 `release_command = "/app/migrate"`로 배포 전 적용한다.

**Tech Stack:** Go 1.25, pgx v5.9.2 (`pgxpool`), Fly.io (`release_command`), Docker (distroless), GitHub Actions, Sentry (sentry-go).

**Spec:** `docs/superpowers/specs/2026-05-30-operations-automation-design.md`

---

## File Structure

| 구분 | 경로 | 책임 |
|---|---|---|
| 신규 | `apps/api/internal/migrate/migrate.go` | `Migration` 타입 · `parseMigrationFilename` · `pendingMigrations` · `Load` · `ensureHistory` · `appliedVersions` · `applyOne` · `Run` |
| 신규 | `apps/api/internal/migrate/migrate_test.go` | `parseMigrationFilename` · `pendingMigrations` 순수 단위 테스트 + 실제 `supabase/migrations`에 대한 `Load` 테스트 |
| 신규 | `apps/api/cmd/migrate/main.go` | `--dir` 플래그 · `DATABASE_URL` 직접 읽기 엔트리 |
| 수정 | `apps/api/internal/observability/sentry.go` | `CaptureException(err)` nil-safe 헬퍼 (비-HTTP 경로용) |
| 신규 | `apps/api/internal/observability/sentry_test.go` | `CaptureException(nil)` no-panic 테스트 |
| 신규 | `apps/api/internal/backfill/backfill.go` | `RunKR`/`RunUS`/`RunIndices` (이동·export) · `SeedIfEmpty` · `hasBars` · `backfillSymbol` · `seedYahooSymbol` · `nasdaqSeed` |
| 수정 | `apps/api/cmd/backfill/main.go` | `internal/backfill` 호출 얇은 wrapper로 축소 |
| 수정 | `apps/api/cmd/server/main.go` | `schedule.Start` 직후 `SeedIfEmpty` 고루틴 디스패치 |
| 수정 | `apps/api/Dockerfile` | 빌드 컨텍스트 루트화 + `migrate` 빌드 + `supabase/migrations` COPY |
| 신규 | `.dockerignore` (레포 루트) | 빌드 컨텍스트 슬림화 |
| 수정 | `apps/api/fly.toml` | `[deploy] release_command = "/app/migrate"` |
| 수정 | `.github/workflows/deploy-api.yml` | 빌드 컨텍스트(working-directory 제거)·트리거 paths 조정 |
| 수정 | `docs/USER_ACTIONS.md` · `STATUS.md` · `ROADMAP.md` · `ARCHITECTURE.md` | 완료 반영 |

**구현 순서 근거:** B의 순수 함수(Task 1)가 가장 깨끗한 TDD 대상이라 먼저. B는 A와 독립이므로 1~3에서 마이그레이터를 완성한다. 이어 A 의존인 `observability.CaptureException`(Task 4) → 백필 추출(Task 5) → 시드 로직(Task 6) → server 와이어링(Task 7, 4·5·6 필요). 빌드 컨텍스트(Task 8)는 `cmd/migrate`(Task 3)가 존재해야 Dockerfile 빌드가 통과하므로 후반. 문서(Task 9) 마지막.

**핵심 correctness 노트 (B):** pgx v5는 `Exec`에 **바인딩 인자가 0개이면 자동으로 simple query protocol을 사용**한다(`conn.go:515-518`). 마이그레이션 파일은 다중 statement(`;` 구분)이므로 `tx.Exec(ctx, m.SQL)`는 **인자 없이** 호출해야 한다(→ simple protocol → 다중 statement 실행). 이력 기록 INSERT는 `$1/$2/$3` 인자가 있어 extended protocol·단일 statement다. 둘을 섞지 말 것.

---

## Task 1: `internal/migrate` 순수 함수 (parse·pending) + 단위 테스트

**Files:**
- Create: `apps/api/internal/migrate/migrate.go`
- Test: `apps/api/internal/migrate/migrate_test.go`

- [ ] **Step 1: 실패하는 테스트 작성**

Create `apps/api/internal/migrate/migrate_test.go`:

```go
package migrate

import "testing"

func TestParseMigrationFilename(t *testing.T) {
	tests := []struct {
		name        string
		wantVersion string
		wantLabel   string
		wantOK      bool
	}{
		{"20260522000001_profiles.sql", "20260522000001", "profiles", true},
		{"20260528000002_paper_trading.sql", "20260528000002", "paper_trading", true},
		{"notes.txt", "", "", false},       // .sql 아님
		{"_leading.sql", "", "", false},    // version 비어 있음
		{"abc_def.sql", "", "", false},     // version 비숫자
		{"20260522000001.sql", "", "", false}, // '_' 없음
	}
	for _, tt := range tests {
		v, l, ok := parseMigrationFilename(tt.name)
		if v != tt.wantVersion || l != tt.wantLabel || ok != tt.wantOK {
			t.Errorf("parseMigrationFilename(%q) = (%q,%q,%v), want (%q,%q,%v)",
				tt.name, v, l, ok, tt.wantVersion, tt.wantLabel, tt.wantOK)
		}
	}
}

func TestPendingMigrations(t *testing.T) {
	available := []Migration{
		{Version: "003", Name: "c.sql"},
		{Version: "001", Name: "a.sql"},
		{Version: "002", Name: "b.sql"},
	}
	applied := map[string]bool{"001": true}
	got := pendingMigrations(available, applied)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Version != "002" || got[1].Version != "003" {
		t.Errorf("order = [%s,%s], want [002,003]", got[0].Version, got[1].Version)
	}
}

func TestPendingMigrationsAllApplied(t *testing.T) {
	available := []Migration{{Version: "001"}, {Version: "002"}}
	applied := map[string]bool{"001": true, "002": true}
	if got := pendingMigrations(available, applied); len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}
```

- [ ] **Step 2: 테스트 실패 확인**

Run: `cd apps/api && go test ./internal/migrate/ -run 'TestParse|TestPending' -v`
Expected: 컴파일 실패 — `undefined: Migration`, `undefined: parseMigrationFilename`, `undefined: pendingMigrations`

- [ ] **Step 3: 최소 구현 작성**

Create `apps/api/internal/migrate/migrate.go`:

```go
// Package migrate — Supabase가 쓰는 supabase_migrations.schema_migrations 이력 테이블을
// 공유하는 forward-only 마이그레이터. release_command(/app/migrate)로 배포 전 적용한다.
package migrate

import (
	"sort"
	"strings"
)

// Migration은 하나의 .sql 마이그레이션 파일을 나타낸다.
type Migration struct {
	Version string // 파일명 숫자 prefix (예: "20260522000001")
	Name    string // 전체 파일명 (예: "20260522000001_profiles.sql")
	SQL     string // 파일 내용
}

// parseMigrationFilename은 "20260522000001_profiles.sql"에서
// version="20260522000001", label="profiles"를 추출한다.
// 규칙: ".sql"로 끝나고, 첫 '_' 앞이 비어 있지 않은 순수 숫자여야 하며, label도 비어 있지 않아야 한다.
func parseMigrationFilename(name string) (version, label string, ok bool) {
	if !strings.HasSuffix(name, ".sql") {
		return "", "", false
	}
	base := strings.TrimSuffix(name, ".sql")
	idx := strings.Index(base, "_")
	if idx <= 0 {
		return "", "", false
	}
	version = base[:idx]
	label = base[idx+1:]
	if label == "" {
		return "", "", false
	}
	for _, r := range version {
		if r < '0' || r > '9' {
			return "", "", false
		}
	}
	return version, label, true
}

// pendingMigrations는 available 중 applied에 없는 마이그레이션을 version 오름차순으로 반환한다.
func pendingMigrations(available []Migration, applied map[string]bool) []Migration {
	var pending []Migration
	for _, m := range available {
		if !applied[m.Version] {
			pending = append(pending, m)
		}
	}
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Version < pending[j].Version
	})
	return pending
}
```

- [ ] **Step 4: 테스트 통과 확인**

Run: `cd apps/api && go test ./internal/migrate/ -run 'TestParse|TestPending' -v`
Expected: PASS (3 테스트)

- [ ] **Step 5: 커밋**

```bash
git add apps/api/internal/migrate/migrate.go apps/api/internal/migrate/migrate_test.go
git commit -m "feat(migrate): 마이그레이션 파일명 파싱·pending 선별 순수 함수"
```

---

## Task 2: `internal/migrate` Load + DB 함수 + Run

**Files:**
- Modify: `apps/api/internal/migrate/migrate.go`
- Test: `apps/api/internal/migrate/migrate_test.go:end` (Load 테스트 추가)

- [ ] **Step 1: 실패하는 Load 테스트 추가**

Append to `apps/api/internal/migrate/migrate_test.go`:

```go
func TestLoadRealMigrations(t *testing.T) {
	// 실제 레포의 supabase/migrations 디렉터리(테스트 패키지 기준 상대경로).
	migs, err := Load("../../../../supabase/migrations")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(migs) != 10 {
		t.Fatalf("len = %d, want 10 (supabase/migrations 파일 수)", len(migs))
	}
	// 정렬·파싱 검증.
	if migs[0].Version != "20260522000001" {
		t.Errorf("first version = %q, want 20260522000001", migs[0].Version)
	}
	if migs[len(migs)-1].Version != "20260528000002" {
		t.Errorf("last version = %q, want 20260528000002", migs[len(migs)-1].Version)
	}
	// SQL 본문이 비어 있지 않아야 한다.
	for _, m := range migs {
		if m.SQL == "" {
			t.Errorf("%s: empty SQL", m.Name)
		}
	}
}
```

- [ ] **Step 2: 테스트 실패 확인**

Run: `cd apps/api && go test ./internal/migrate/ -run TestLoadRealMigrations -v`
Expected: 컴파일 실패 — `undefined: Load`

- [ ] **Step 3: Load + DB 함수 + Run 구현**

Append to `apps/api/internal/migrate/migrate.go` (imports를 아래 블록으로 교체하고 함수 추가):

먼저 import 블록을 교체:

```go
import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)
```

그 다음 파일 끝에 함수 추가:

```go
// Load는 dir의 *.sql 마이그레이션을 읽어 version 오름차순으로 반환한다.
// 파일명이 parseMigrationFilename 규칙에 맞지 않으면 조용히 건너뛴다.
func Load(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}
	var migs []Migration
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		version, _, ok := parseMigrationFilename(e.Name())
		if !ok {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		migs = append(migs, Migration{Version: version, Name: e.Name(), SQL: string(b)})
	}
	sort.Slice(migs, func(i, j int) bool { return migs[i].Version < migs[j].Version })
	return migs, nil
}

// ensureHistory는 이력 스키마·테이블을 보장한다(멱등). 각 문장은 인자 없는 단일 statement.
func ensureHistory(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `create schema if not exists supabase_migrations`); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	if _, err := pool.Exec(ctx, `create table if not exists supabase_migrations.schema_migrations (
		version text primary key,
		statements text[],
		name text
	)`); err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	return nil
}

// appliedVersions는 이미 적용된 version 집합을 반환한다.
func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `select version from supabase_migrations.schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

// applyOne은 마이그레이션 하나를 단일 트랜잭션으로 적용하고 이력에 기록한다.
// 주의: m.SQL은 다중 statement일 수 있으므로 인자 없이 Exec한다
// (pgx는 인자 0개일 때 simple query protocol을 써서 다중 statement를 허용한다).
// 이력 INSERT는 $1/$2/$3 인자가 있어 extended protocol·단일 statement다.
func applyOne(ctx context.Context, pool *pgxpool.Pool, m Migration) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, m.SQL); err != nil {
		return fmt.Errorf("exec %s: %w", m.Name, err)
	}
	if _, err := tx.Exec(ctx,
		`insert into supabase_migrations.schema_migrations (version, name, statements)
		 values ($1, $2, $3) on conflict (version) do nothing`,
		m.Version, m.Name, []string{m.SQL}); err != nil {
		return fmt.Errorf("record %s: %w", m.Name, err)
	}
	return tx.Commit(ctx)
}

// Run은 dir의 미적용 마이그레이션을 version 순으로 적용한다(forward-only, 멱등).
// 어느 하나라도 실패하면 즉시 에러를 반환한다 → release_command 비0 exit → 배포 중단(fail-closed).
func Run(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if err := ensureHistory(ctx, pool); err != nil {
		return fmt.Errorf("ensure history: %w", err)
	}
	available, err := Load(dir)
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return fmt.Errorf("applied versions: %w", err)
	}
	pending := pendingMigrations(available, applied)
	if len(pending) == 0 {
		slog.Info("migrate: no pending migrations", "total", len(available))
		return nil
	}
	for _, m := range pending {
		if err := applyOne(ctx, pool, m); err != nil {
			return fmt.Errorf("apply %s: %w", m.Name, err)
		}
		slog.Info("migrate: applied", "version", m.Version, "name", m.Name)
	}
	slog.Info("migrate: done", "applied", len(pending), "total", len(available))
	return nil
}
```

`strings` import는 Task 1에서 이미 쓰이므로 유지된다. `sort`도 유지. 새로 추가된 `context`/`fmt`/`log/slog`/`os`/`path/filepath`/`pgxpool`이 import 블록에 모두 있는지 확인.

- [ ] **Step 4: 테스트·빌드·vet 통과 확인**

Run: `cd apps/api && go test ./internal/migrate/ -v && go vet ./internal/migrate/`
Expected: PASS (4 테스트: Parse, Pending x2, LoadRealMigrations), vet 무경고

- [ ] **Step 5: 커밋**

```bash
git add apps/api/internal/migrate/migrate.go apps/api/internal/migrate/migrate_test.go
git commit -m "feat(migrate): Load·ensureHistory·applyOne·Run (이력 테이블 공유, 트랜잭션 적용)"
```

---

## Task 3: `cmd/migrate` 엔트리

**Files:**
- Create: `apps/api/cmd/migrate/main.go`

- [ ] **Step 1: 엔트리 작성**

Create `apps/api/cmd/migrate/main.go`:

```go
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/migrate"
)

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

	pool, err := db.New(ctx, dsn)
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := migrate.Run(ctx, pool, *dir); err != nil {
		slog.Error("migrate failed", "err", err)
		os.Exit(1) // release_command 실패 → Fly 배포 중단(fail-closed)
	}
}
```

- [ ] **Step 2: 빌드·vet 통과 확인**

Run: `cd apps/api && go build ./cmd/migrate && go vet ./cmd/migrate`
Expected: 빌드 성공, vet 무경고

- [ ] **Step 3: (선택) 로컬 멱등 통합 검증 — DATABASE_URL 있을 때만**

로컬에 개발 DB가 있으면 멱등성을 직접 확인한다. 없으면 건너뛰고 첫 배포의 `release_command`가 통합 검증 역할을 한다(멱등이라 재실행 안전).

Run: `cd apps/api && go run ./cmd/migrate --dir=../../supabase/migrations`
Expected: 첫 실행 시 `migrate: applied ...` 로그 후 `migrate: done`. 즉시 재실행 시 `migrate: no pending migrations`.

- [ ] **Step 4: 커밋**

```bash
git add apps/api/cmd/migrate/main.go
git commit -m "feat(migrate): cmd/migrate 엔트리 (DATABASE_URL 직접 읽기, --dir 플래그)"
```

---

## Task 4: `observability.CaptureException` nil-safe 헬퍼

**Files:**
- Modify: `apps/api/internal/observability/sentry.go`
- Test: `apps/api/internal/observability/sentry_test.go`

- [ ] **Step 1: 실패하는 테스트 작성**

Create `apps/api/internal/observability/sentry_test.go`:

```go
package observability

import "testing"

// CaptureException(nil)은 Sentry 미초기화 상태에서도 panic 없이 무시되어야 한다.
func TestCaptureExceptionNilSafe(t *testing.T) {
	CaptureException(nil) // nil → 즉시 반환, no panic
}
```

- [ ] **Step 2: 테스트 실패 확인**

Run: `cd apps/api && go test ./internal/observability/ -run TestCaptureExceptionNilSafe -v`
Expected: 컴파일 실패 — `undefined: CaptureException`

- [ ] **Step 3: 헬퍼 추가**

Edit `apps/api/internal/observability/sentry.go` — `Flush` 함수 뒤에 추가:

```go
// CaptureException은 비-HTTP 경로(부팅 백필·cron 등)에서 에러를 Sentry로 보고한다.
// InitSentry가 초기화한 global hub를 사용한다. DSN 미설정 시 sentry-go 내부 no-op. nil-safe.
func CaptureException(err error) {
	if err == nil {
		return
	}
	sentry.CaptureException(err)
}
```

`sentry` import는 이미 존재(`github.com/getsentry/sentry-go`).

- [ ] **Step 4: 테스트 통과 확인**

Run: `cd apps/api && go test ./internal/observability/ -v && go vet ./internal/observability/`
Expected: PASS, vet 무경고

- [ ] **Step 5: 커밋**

```bash
git add apps/api/internal/observability/sentry.go apps/api/internal/observability/sentry_test.go
git commit -m "feat(observability): 비-HTTP 경로용 CaptureException nil-safe 헬퍼"
```

---

## Task 5: `internal/backfill` 추출 (run* 이동·export) + `cmd/backfill` 축소

이 Task는 동작 변경 없는 이동(refactor)이다. `cmd/backfill/main.go`의 `runKR`/`runUS`/`runIndices`/`nasdaqSeed`를 `internal/backfill`로 옮기고 export한다. CLI 동작은 동일하게 유지된다.

**Files:**
- Create: `apps/api/internal/backfill/backfill.go`
- Modify: `apps/api/cmd/backfill/main.go`

- [ ] **Step 1: `internal/backfill/backfill.go` 작성 (run* 이동·export)**

Create `apps/api/internal/backfill/backfill.go`:

```go
// Package backfill — KIND/Yahoo에서 가격 히스토리를 채우는 시장별 백필 로직.
// cmd/backfill(CLI)과 cmd/server(부팅 시드)가 공유한다.
package backfill

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/ingest"
	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/quotient/quotient/apps/api/internal/schedule"
	"github.com/quotient/quotient/apps/api/internal/sources/kind"
	"github.com/quotient/quotient/apps/api/internal/sources/yahoo"
)

// nasdaqSeed는 backfill 시드 30종목. Phase 2에서 S&P 100 전체로 확장.
var nasdaqSeed = []struct{ Symbol, Name string }{
	{"AAPL", "Apple Inc."}, {"MSFT", "Microsoft"}, {"GOOGL", "Alphabet Class A"},
	{"AMZN", "Amazon"}, {"NVDA", "NVIDIA"}, {"META", "Meta Platforms"},
	{"TSLA", "Tesla"}, {"AVGO", "Broadcom"}, {"NFLX", "Netflix"},
	{"AMD", "Advanced Micro Devices"}, {"INTC", "Intel"}, {"ORCL", "Oracle"},
	{"CRM", "Salesforce"}, {"ADBE", "Adobe"}, {"QCOM", "Qualcomm"},
	{"TXN", "Texas Instruments"}, {"COST", "Costco"}, {"PEP", "PepsiCo"},
	{"CSCO", "Cisco"}, {"TMUS", "T-Mobile US"}, {"INTU", "Intuit"},
	{"AMAT", "Applied Materials"}, {"BKNG", "Booking Holdings"},
	{"ISRG", "Intuitive Surgical"}, {"REGN", "Regeneron"},
	{"VRTX", "Vertex Pharmaceuticals"}, {"LRCX", "Lam Research"},
	{"PANW", "Palo Alto Networks"}, {"ADP", "Automatic Data Processing"},
	{"GILD", "Gilead Sciences"},
}

// RunKR은 KIND에서 종목 마스터를 받아 KOSPI/KOSDAQ 전 종목의 years년 일봉을 백필한다.
func RunKR(ctx context.Context, pool *pgxpool.Pool, market string, years, limit int) error {
	kc := kind.NewClient("")
	yc := yahoo.NewClient()

	items, err := kc.FetchInstruments(ctx, market)
	if err != nil {
		return fmt.Errorf("kind: %w", err)
	}
	slog.Info("instruments fetched", "market", market, "count", len(items))
	if _, err := ingest.UpsertInstruments(ctx, pool, items); err != nil {
		return fmt.Errorf("upsert instruments: %w", err)
	}

	rows, err := pool.Query(ctx,
		`select id::text, symbol from public.instruments where exchange = 'KRX' and is_active = true order by symbol`)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	type s struct{ id, code string }
	var syms []s
	for rows.Next() {
		var x s
		if err := rows.Scan(&x.id, &x.code); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		syms = append(syms, x)
	}
	if limit > 0 && len(syms) > limit {
		syms = syms[:limit]
	}

	end := time.Now().UTC()
	start := end.AddDate(-years, 0, 0)
	for i, sym := range syms {
		ysym := yahoo.SymbolKR(sym.code, market)
		n, err := backfillSymbol(ctx, pool, yc, sym.id, ysym, start, end)
		if err != nil {
			slog.Warn("kr skip", "symbol", ysym, "err", err)
			continue
		}
		slog.Info("backfilled", "i", i+1, "total", len(syms), "symbol", ysym, "rows", n)
		time.Sleep(200 * time.Millisecond) // Yahoo rate limit 보호
	}
	return nil
}

// RunUS는 nasdaqSeed 30종목을 시드한 뒤 years년 일봉을 백필한다.
func RunUS(ctx context.Context, pool *pgxpool.Pool, years, limit int) error {
	yc := yahoo.NewClient()

	insts := make([]models.Instrument, 0, len(nasdaqSeed))
	for _, s := range nasdaqSeed {
		insts = append(insts, models.Instrument{
			Symbol: s.Symbol, Exchange: "NASDAQ", Name: s.Name,
			AssetClass: models.AssetUSStock, Currency: "USD", IsActive: true,
		})
	}
	if _, err := ingest.UpsertInstruments(ctx, pool, insts); err != nil {
		return fmt.Errorf("upsert seed: %w", err)
	}

	rows, err := pool.Query(ctx,
		`select id::text, symbol from public.instruments where exchange = 'NASDAQ' and is_active = true order by symbol`)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	type s struct{ id, code string }
	var syms []s
	for rows.Next() {
		var x s
		if err := rows.Scan(&x.id, &x.code); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		syms = append(syms, x)
	}
	if limit > 0 && len(syms) > limit {
		syms = syms[:limit]
	}

	end := time.Now().UTC()
	start := end.AddDate(-years, 0, 0)
	for i, sym := range syms {
		n, err := backfillSymbol(ctx, pool, yc, sym.id, sym.code, start, end)
		if err != nil {
			slog.Warn("us skip", "symbol", sym.code, "err", err)
			continue
		}
		slog.Info("backfilled", "i", i+1, "total", len(syms), "symbol", sym.code, "rows", n)
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

// RunIndices는 asset_class='INDEX' 활성 종목의 years년 일봉을 Yahoo에서 백필한다.
// 알파 카드(90D/1Y)가 의존한다.
func RunIndices(ctx context.Context, pool *pgxpool.Pool, years, limit int) error {
	yc := yahoo.NewClient()

	rows, err := pool.Query(ctx,
		`select id::text, symbol, exchange from public.instruments
		 where asset_class = 'INDEX' and is_active = true
		 order by symbol`)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	type idx struct{ id, symbol, exchange string }
	var list []idx
	for rows.Next() {
		var x idx
		if err := rows.Scan(&x.id, &x.symbol, &x.exchange); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		list = append(list, x)
	}
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}

	end := time.Now().UTC()
	start := end.AddDate(-years, 0, 0)
	for i, x := range list {
		ysym := schedule.IndexYahooSymbol(x.symbol, x.exchange)
		if ysym == "" {
			slog.Warn("no yahoo symbol for index", "symbol", x.symbol, "exchange", x.exchange)
			continue
		}
		n, err := backfillSymbol(ctx, pool, yc, x.id, ysym, start, end)
		if err != nil {
			slog.Warn("indices skip", "symbol", ysym, "err", err)
			continue
		}
		slog.Info("backfilled", "i", i+1, "total", len(list), "symbol", ysym, "rows", n)
		time.Sleep(200 * time.Millisecond) // Yahoo rate limit 보호
	}
	return nil
}

// backfillSymbol은 한 instrument의 [start,end] 일봉을 fetch·upsert한다. 호출자가 rate-limit sleep 담당.
func backfillSymbol(ctx context.Context, pool *pgxpool.Pool, yc *yahoo.Client,
	instrumentID, yahooSymbol string, start, end time.Time) (int64, error) {
	bars, err := yc.FetchChart(ctx, yahooSymbol, start, end)
	if err != nil {
		return 0, err
	}
	if len(bars) == 0 {
		return 0, nil
	}
	for j := range bars {
		bars[j].InstrumentID = instrumentID
	}
	return ingest.UpsertPrices(ctx, pool, bars)
}
```

> 참고: 기존 `runKR`/`runUS`/`runIndices`의 per-symbol fetch→upsert 본문을 `backfillSymbol`로 추출했다. 로그 메시지는 `"yahoo skip"`+`"upsert skip"` 2종에서 시장별 단일 `"... skip"`으로 합쳐졌으나(에러 시 continue) 동작은 동일하다.

- [ ] **Step 2: `cmd/backfill/main.go`를 얇은 wrapper로 축소**

Replace entire `apps/api/cmd/backfill/main.go` with:

```go
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/quotient/quotient/apps/api/internal/backfill"
	"github.com/quotient/quotient/apps/api/internal/config"
	"github.com/quotient/quotient/apps/api/internal/db"
)

func main() {
	years := flag.Int("years", 5, "백필 기간 (연 단위)")
	market := flag.String("market", "KOSPI", "KOSPI | KOSDAQ | NASDAQ | INDICES")
	limit := flag.Int("limit", 0, "최대 종목 수 (0=전체). 디버깅용.")
	flag.Parse()

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	switch *market {
	case "KOSPI", "KOSDAQ":
		err = backfill.RunKR(ctx, pool, *market, *years, *limit)
	case "NASDAQ":
		err = backfill.RunUS(ctx, pool, *years, *limit)
	case "INDICES":
		err = backfill.RunIndices(ctx, pool, *years, *limit)
	default:
		slog.Error("unknown market", "market", *market)
		os.Exit(1)
	}
	if err != nil {
		slog.Error("backfill", "market", *market, "err", err)
		os.Exit(1)
	}
	slog.Info("backfill done", "market", *market, "years", *years)
}
```

- [ ] **Step 3: 빌드·vet 통과 확인 (전체)**

Run: `cd apps/api && go build ./... && go vet ./...`
Expected: 전체 빌드 성공, vet 무경고. (`cmd/backfill`이 더 이상 `ingest`/`models`/`schedule`/`kind`/`yahoo`를 직접 import하지 않아 unused import 에러가 없어야 한다.)

- [ ] **Step 4: 커밋**

```bash
git add apps/api/internal/backfill/backfill.go apps/api/cmd/backfill/main.go
git commit -m "refactor(backfill): run* 로직을 internal/backfill로 추출·export (CLI 동작 보존)"
```

---

## Task 6: `SeedIfEmpty` (멱등 only-if-empty, per-series)

**Files:**
- Modify: `apps/api/internal/backfill/backfill.go`

- [ ] **Step 1: `SeedIfEmpty`·`hasBars`·`seedYahooSymbol` 추가**

Append to `apps/api/internal/backfill/backfill.go`:

```go
// SeedIfEmpty는 부팅 시 비어 있는 INDEX·NASDAQ 시리즈만 채운다(멱등, per-series).
// NASDAQ instrument를 멱등 시드하고(INDEX는 마이그레이션이 시드), 각 대상의 봉 존재 여부를
// 확인해 0행인 시리즈만 백필한다. 부분 실패는 다음 부팅에서 실패분만 재시도된다(cross-boot self-heal).
// 하나라도 실패하면 집계 에러를 반환한다(호출자가 Sentry 보고용).
func SeedIfEmpty(ctx context.Context, pool *pgxpool.Pool, years int) error {
	// 1) NASDAQ instrument 시드 보장 (멱등 upsert).
	insts := make([]models.Instrument, 0, len(nasdaqSeed))
	for _, s := range nasdaqSeed {
		insts = append(insts, models.Instrument{
			Symbol: s.Symbol, Exchange: "NASDAQ", Name: s.Name,
			AssetClass: models.AssetUSStock, Currency: "USD", IsActive: true,
		})
	}
	if _, err := ingest.UpsertInstruments(ctx, pool, insts); err != nil {
		return fmt.Errorf("seed nasdaq instruments: %w", err)
	}

	// 2) 대상 조회: INDEX 전체 + NASDAQ 종목.
	rows, err := pool.Query(ctx,
		`select id::text, symbol, exchange, asset_class from public.instruments
		 where (asset_class = 'INDEX' or exchange = 'NASDAQ') and is_active = true
		 order by symbol`)
	if err != nil {
		return fmt.Errorf("query targets: %w", err)
	}
	defer rows.Close()
	type target struct{ id, symbol, exchange, assetClass string }
	var targets []target
	for rows.Next() {
		var t target
		if err := rows.Scan(&t.id, &t.symbol, &t.exchange, &t.assetClass); err != nil {
			return fmt.Errorf("scan target: %w", err)
		}
		targets = append(targets, t)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate targets: %w", err)
	}

	yc := yahoo.NewClient()
	end := time.Now().UTC()
	start := end.AddDate(-years, 0, 0)
	var seeded, failures int
	for _, t := range targets {
		has, err := hasBars(ctx, pool, t.id)
		if err != nil {
			slog.Warn("seed: hasBars failed", "symbol", t.symbol, "err", err)
			failures++
			continue
		}
		if has {
			continue // 이미 채워짐 — skip
		}
		ysym := seedYahooSymbol(t.assetClass, t.symbol, t.exchange)
		if ysym == "" {
			slog.Warn("seed: no yahoo symbol", "symbol", t.symbol, "exchange", t.exchange)
			continue
		}
		n, err := backfillSymbol(ctx, pool, yc, t.id, ysym, start, end)
		if err != nil {
			slog.Warn("seed: backfill failed", "symbol", ysym, "err", err)
			failures++
			continue
		}
		slog.Info("seed: backfilled", "symbol", ysym, "rows", n)
		seeded++
		time.Sleep(200 * time.Millisecond) // Yahoo rate limit 보호
	}
	slog.Info("seed: done", "seeded", seeded, "failures", failures, "targets", len(targets))
	if failures > 0 {
		return fmt.Errorf("seed: %d series failed (will retry next boot)", failures)
	}
	return nil
}

// hasBars는 instrument에 봉이 1개라도 있으면 true.
func hasBars(ctx context.Context, pool *pgxpool.Pool, instrumentID string) (bool, error) {
	var exists bool
	err := pool.QueryRow(ctx,
		`select exists(select 1 from public.prices where instrument_id = $1)`,
		instrumentID).Scan(&exists)
	return exists, err
}

// seedYahooSymbol은 INDEX면 IndexYahooSymbol, NASDAQ 종목이면 심볼 그대로(RunUS와 동일).
func seedYahooSymbol(assetClass, symbol, exchange string) string {
	if assetClass == string(models.AssetIndex) {
		return schedule.IndexYahooSymbol(symbol, exchange)
	}
	return symbol
}
```

- [ ] **Step 2: 빌드·vet 통과 확인**

Run: `cd apps/api && go build ./internal/backfill/ && go vet ./internal/backfill/`
Expected: 빌드 성공, vet 무경고

- [ ] **Step 3: (선택) 로컬 only-if-empty 동작 검증 — DATABASE_URL 있을 때만**

로컬 DB가 있으면 CLI로 INDICES를 1종목 백필한 뒤, `SeedIfEmpty`가 채워진 시리즈를 skip하고 빈 것만 채우는지 로그로 확인한다. (`SeedIfEmpty` 자체는 부팅 경로에서만 호출되므로, 이 단계는 Task 7 와이어링 후 server 부팅 로그로 검증해도 된다.) DB가 없으면 건너뛴다.

- [ ] **Step 4: 커밋**

```bash
git add apps/api/internal/backfill/backfill.go
git commit -m "feat(backfill): SeedIfEmpty — 부팅 시 빈 시리즈만 채우는 멱등 per-series 백필"
```

---

## Task 7: server 부팅 시 `SeedIfEmpty` 고루틴 디스패치

**Files:**
- Modify: `apps/api/cmd/server/main.go`

- [ ] **Step 1: import 추가**

Edit `apps/api/cmd/server/main.go` — import 블록에서 `backfill`을 `auth` 다음·`config` 앞에 추가 (alphabetical 순서 유지):

```go
	"github.com/quotient/quotient/apps/api/internal/auth"
	"github.com/quotient/quotient/apps/api/internal/backfill"
	"github.com/quotient/quotient/apps/api/internal/config"
```

- [ ] **Step 2: `schedule.Start(...)` 직후 고루틴 추가**

Edit `apps/api/cmd/server/main.go` — `cronWorker := schedule.Start(...)` 블록(끝나는 `}, aiClient, toolRegistry)` 줄) 바로 다음에 추가:

```go

	// 부팅 시 비어 있는 지수·NASDAQ 시리즈 자동 백필 (비동기·실패 무시).
	// readiness·ListenAndServe를 절대 차단하지 않는다. 실패 시 로그+Sentry, 크래시 없음.
	// lifecycle ctx를 전달 → graceful shutdown 시 진행 중 백필도 취소된다.
	go func() {
		if err := backfill.SeedIfEmpty(ctx, pool, 5); err != nil {
			logger.Error("boot backfill failed (continuing)", "err", err)
			observability.CaptureException(err)
		}
	}()
```

- [ ] **Step 3: 빌드·vet 통과 확인**

Run: `cd apps/api && go build ./... && go vet ./...`
Expected: 전체 빌드 성공, vet 무경고

- [ ] **Step 4: (선택) 부팅 로그 스모크 — DATABASE_URL 있을 때만**

로컬에서 server를 띄워 부팅 로그에 `seed: done`이 찍히고 HTTP가 정상 listen하는지 확인한다(백필이 부팅을 차단하지 않음을 확인).

Run: `cd apps/api && go run ./cmd/server` (별도 터미널에서 `curl localhost:8080/healthz`)
Expected: `API listening` 로그 즉시 출력 → 백그라운드에서 `seed: ...` 로그. healthz는 백필과 무관하게 즉시 200.

- [ ] **Step 5: 커밋**

```bash
git add apps/api/cmd/server/main.go
git commit -m "feat(server): 부팅 시 SeedIfEmpty 비동기 디스패치 (실패 무시, readiness 비차단)"
```

---

## Task 8: 빌드 컨텍스트 레포 루트화 + `release_command`

**Files:**
- Modify: `apps/api/Dockerfile`
- Create: `.dockerignore` (레포 루트)
- Modify: `apps/api/fly.toml`
- Modify: `.github/workflows/deploy-api.yml`

- [ ] **Step 1: Dockerfile 재작성 (경로를 레포 루트 기준으로)**

Replace entire `apps/api/Dockerfile` with:

```dockerfile
# Build stage — Go 1.25 (pgx v5 요구). 빌드 컨텍스트 = 레포 루트.
FROM golang:1.25-alpine AS build

WORKDIR /src
# 캐시 효율: go.mod·sum만 먼저 복사
COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

COPY apps/api/ ./

# 정적 바이너리 3개 (CGO 비활성).
# - server:   HTTP API + cron 워커 (+ 부팅 시 SeedIfEmpty)
# - backfill: 일회성 CLI (수동 전체 재백필) — `flyctl ssh console -C "/app/backfill"`
# - migrate:  release_command (배포 전 마이그레이션 적용)
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server   ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/backfill ./cmd/backfill
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate  ./cmd/migrate

# Runtime stage — distroless (rootless, minimal)
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build /out/server   /app/server
COPY --from=build /out/backfill /app/backfill
COPY --from=build /out/migrate  /app/migrate
# 마이그레이션 SQL을 이미지에 포함 (레포 루트 컨텍스트라 접근 가능). migrate가 --dir=/app/migrations로 읽음.
COPY supabase/migrations /app/migrations

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]
```

- [ ] **Step 2: 레포 루트 `.dockerignore` 신규 작성**

Create `.dockerignore` (레포 루트):

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

> 빌드 컨텍스트가 루트로 바뀌면 기존 `apps/api/.dockerignore`는 적용되지 않는다. `*_test.go`·`testdata` 제외 규칙은 이 루트 파일이 승계한다.

- [ ] **Step 3: `fly.toml`에 release_command 추가**

Edit `apps/api/fly.toml` — `[build]` 블록 바로 다음(또는 `[env]` 앞)에 추가:

```toml
[deploy]
  release_command = "/app/migrate"
```

- [ ] **Step 4: `deploy-api.yml` — 빌드 컨텍스트·트리거 조정**

Edit `.github/workflows/deploy-api.yml`:

`paths`에 `supabase/migrations/**` 추가:

```yaml
    paths:
      - "apps/api/**"
      - "supabase/migrations/**"
      - ".github/workflows/deploy-api.yml"
```

`deploy` step에서 `working-directory` 제거하고 `--config`·`--dockerfile` 명시:

```yaml
      - name: deploy
        run: flyctl deploy --remote-only --config apps/api/fly.toml --dockerfile apps/api/Dockerfile
        env:
          FLY_API_TOKEN: ${{ secrets.FLY_API_TOKEN }}
```

- [ ] **Step 5: 레포 루트에서 docker build 검증**

Run: `docker build -f apps/api/Dockerfile -t quotient-api-test .` (레포 루트에서)
Expected: 3개 바이너리 빌드 + `COPY supabase/migrations /app/migrations` 성공. 빌드 완료.
이미지에 migrations가 들어갔는지 확인: `docker run --rm --entrypoint /app/migrate quotient-api-test --help 2>&1 | head -1` (flag 사용법 출력) 및 `docker run --rm --entrypoint ls quotient-api-test /app/migrations | head` (10개 .sql 나열).

> Docker 데몬이 없으면 이 단계는 CI/배포 시점 검증으로 미룬다 — 그 경우 Step 6 커밋 메시지에 "docker build 미검증(로컬 데몬 없음)"을 명시하고 사용자에게 보고한다.

- [ ] **Step 6: 커밋**

```bash
git add apps/api/Dockerfile .dockerignore apps/api/fly.toml .github/workflows/deploy-api.yml
git commit -m "build(api): 빌드 컨텍스트 레포 루트화 + migrate 바이너리 + release_command 마이그레이션"
```

---

## Task 9: 문서 업데이트 (완료 반영)

CLAUDE.md "문서 업데이트 규칙(MANDATORY)"에 따라 STATUS/ROADMAP/ARCHITECTURE/USER_ACTIONS를 갱신한다.

**Files:**
- Modify: `docs/USER_ACTIONS.md`
- Modify: `docs/STATUS.md`
- Modify: `docs/ROADMAP.md`
- Modify: `docs/ARCHITECTURE.md`

- [ ] **Step 1: `docs/USER_ACTIONS.md` — 자동화로 이동**

- 🔴 시급 "지수 5년 백필" + "백테스트 대상 종목 가격 백필" → **삭제 또는 "자동화됨" 섹션으로 이동**. 사유: 부팅 시 `SeedIfEmpty`가 자동 수행.
- 🟡 "production 지수 백필"(`flyctl ssh console -C "/app/backfill --market=INDICES"`) → 삭제. 부팅 자동화로 대체.
- 🟡 `supabase db push` → 삭제 또는 "배포 시 release_command 자동 적용"으로 대체.
- DJI 표기 정정: "KOSPI·KOSDAQ·SPX·NDX·DJI" → 실제 시드 4종 "KOSPI·KOSDAQ·SPX·NDX" (DJI는 마이그레이션 미시드).
- **신규 운영 주의 추가**: "깨진 마이그레이션은 release_command 비0 exit로 **모든 후속 배포를 막는다**(fail-closed). 마이그레이션 SQL은 tx-안전(standalone `BEGIN;/COMMIT;`·`CREATE INDEX CONCURRENTLY` 금지)이어야 한다."

- [ ] **Step 2: `docs/STATUS.md` — 완료 반영**

- "운영 자동화"(A 부팅 백필 + B release_command 마이그레이션)를 완료(✅) 항목으로 추가.
- "최근 변경 이력" 맨 위에 한 줄: `2026-05-30 — 운영 자동화: 부팅 시 지수·NASDAQ 자동 백필(SeedIfEmpty, 비동기·멱등) + Fly release_command Go 마이그레이터(이력 테이블 공유)`.
- "마지막 업데이트" 날짜 갱신.
- (알파 카드 FX 시계열) — 비범위에서 언급한 선재 이슈: 알파 카드가 시점별 환율을 FX 시계열에 의존하는지 plan 구현 중 확인하고, 자동 백필로 안 채워지는 결함이면 "알려진 결함"에 등재. (확인 결과를 이 Step에서 반영하거나, 무관하면 등재 생략.)

- [ ] **Step 3: `docs/ROADMAP.md` — 다음 작업 재설정**

- "현재 추천 다음 작업"의 "1. 운영 자동화" 제거.
- 완료 메모를 괄호 항목으로 한 줄 추가(다른 완료 항목 형식과 동일).
- 남은 "현재 추천 다음 작업"을 재설정(외부 계정·키 발급 사용자 액션 또는 Phase 2 항목 중 선정).

- [ ] **Step 4: `docs/ARCHITECTURE.md` — 핵심 설계 결정 추가**

"핵심 설계 결정"에 2건 추가(Why/How 필수):
- **부팅 시 멱등 자동 백필(A)** — Why: 수동 CLI 의무 제거, 가용성 우선. How: `internal/backfill.SeedIfEmpty`를 server 부팅 고루틴에서 fire-and-forget, per-series only-if-empty(cross-boot self-heal), 실패는 로그+Sentry(`observability.CaptureException`)·크래시 없음.
- **release_command Go 마이그레이터(B)** — Why: `supabase db push` 수동 단계 제거 + 스키마 정합성 fail-closed. How: `supabase_migrations.schema_migrations` 이력 공유, 마이그레이션당 트랜잭션, 비0 exit가 Fly 배포 중단. 빌드 컨텍스트 레포 루트화로 `supabase/migrations`를 이미지에 포함. 제약: 마이그레이션은 tx-안전이어야 함(CONCURRENTLY 도입 시 no-tx 경로 필요).

- [ ] **Step 5: 커밋**

```bash
git add docs/USER_ACTIONS.md docs/STATUS.md docs/ROADMAP.md docs/ARCHITECTURE.md
git commit -m "docs: 운영 자동화 완료 반영 (STATUS·ROADMAP·ARCHITECTURE·USER_ACTIONS)"
```

---

## 완료 후 (controller 책임)

모든 Task 완료 후:
1. 전체 코드 리뷰 디스패치 (`superpowers:requesting-code-review`) — BASE_SHA = 이 작업 시작 전 HEAD, HEAD_SHA = 마지막 커밋.
2. (DB·Docker 접촉부) 첫 배포 시점의 통합 검증을 사용자에게 안내 — release_command 멱등이라 재실행 안전.
3. `superpowers:finishing-a-development-branch`로 마무리.

## 테스트 커버리지 요약 (정직한 한계 명시)

| 영역 | 검증 수단 |
|---|---|
| `migrate.parseMigrationFilename`·`pendingMigrations` | 순수 단위 테스트 (Task 1) |
| `migrate.Load` | 실제 `supabase/migrations` 10개 파일 통합 테스트 (Task 2) |
| `migrate.ensureHistory`·`applyOne`·`Run` (DB) | 단위 테스트 없음(레포에 Go DB 하네스 부재) — 로컬 `go run ./cmd/migrate`(선택) + 첫 배포 release_command가 통합 검증. 멱등이라 안전 |
| `observability.CaptureException` | nil-safe no-panic 단위 테스트 (Task 4) |
| `backfill.RunKR/RunUS/RunIndices` 추출 | 동작 보존 — `go build`·`go vet`·CLI 동작 불변 (Task 5) |
| `backfill.SeedIfEmpty`·`hasBars` (DB·외부 fetch) | 단위 테스트 없음 — 빌드·vet + (선택) 로컬 부팅 로그 스모크 (Task 6·7) |
| Dockerfile 빌드 컨텍스트 루트화 | 레포 루트 `docker build` (Task 8) — 데몬 없으면 CI/배포 검증 |
