# RLS JWT 풀 전환 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 사용자 데이터 핸들러(profile/holdings/watchlist/chat/briefing)를 슈퍼유저 풀에서 사용자 JWT 컨텍스트로 전환하여 Supabase RLS가 자동 적용되도록 한다. 애플리케이션 단의 `WHERE user_id = $1` 필터는 이중 방어로 유지한다.

**Architecture:**
- pgxpool 단일 인스턴스를 그대로 사용하되, 사용자 요청 핸들러는 `db.AsUser(ctx, pool, userID, fn)` 헬퍼로 트랜잭션을 열어 `set_config('role', 'authenticated', true)` + `set_config('request.jwt.claims', '...', true)` + `set_config('request.jwt.claim.sub', userID, true)`를 LOCAL 적용한다.
- repo 메서드는 `*pgxpool.Pool` 대신 `db.Executor` 인터페이스(=`pgx.Tx`와 pool 모두 만족)를 받아 호출자가 트랜잭션·풀을 선택할 수 있게 한다.
- cron 잡, 도구 레지스트리, 공개 read(market/instruments/history), 브리핑 워커는 슈퍼유저 풀을 유지한다(시스템 작업·다중 사용자 fan-out).
- AI 도구 호출(portfolio/search/quote)은 단일 사용자 ctx지만 plan 범위 밖 — Task 8에서 후속 결정으로 명시.

**Tech Stack:** Go 1.25, pgx v5, chi v5, Supabase Postgres (RLS).

---

## File Structure

새 파일:
- `apps/api/internal/db/executor.go` — `Executor` 인터페이스(`Query`/`QueryRow`/`Exec`).
- `apps/api/internal/db/userjwt.go` — `AsUser(ctx, pool, userID, fn)` 헬퍼.
- `apps/api/internal/db/userjwt_integration_test.go` — testcontainers/로컬 Supabase로 SET/auth.uid() 호환 검증(integration build tag).
- `apps/api/internal/db/rls_integration_test.go` — 사용자 A의 행이 사용자 B 컨텍스트에서 보이지 않음 검증.

수정 파일:
- `apps/api/internal/handlers/profile_repo_pg.go` — `Get`/`Update` 시그니처 → `db.Executor`. pool 필드 제거.
- `apps/api/internal/handlers/profiles.go` — `txRunner` 패턴 도입.
- `apps/api/internal/handlers/holdings_repo_pg.go` / `holdings.go` — 동일.
- `apps/api/internal/handlers/watchlist_repo_pg.go` / `watchlist.go` — 동일.
- `apps/api/internal/handlers/chat_repo_pg.go` — 모든 메서드 시그니처 변경.
- `apps/api/internal/handlers/chat.go` — inline 영속화 4지점(user/assistant/tool/usage)을 `db.AsUser`로 wrap.
- `apps/api/internal/handlers/briefing.go` — `Today` 핸들러를 `db.AsUser`로 wrap.
- `apps/api/cmd/server/main.go` — handler 생성자 시그니처 정리.
- `apps/api/internal/handlers/profiles_test.go` / `holdings_test.go` / `watchlist_test.go` / `chat_test.go` — fake repo 시그니처 갱신.
- `docs/STATUS.md` — RLS 우회 결함 해제 + 변경 이력 추가.
- `docs/ARCHITECTURE.md` — "사용자 JWT 풀 패턴" 섹션 추가.
- 각 repo 상단 `TODO(security)` 코멘트 제거.

---

## Task 1: Executor 인터페이스 + AsUser 헬퍼

**Files:**
- Create: `apps/api/internal/db/executor.go`
- Create: `apps/api/internal/db/userjwt.go`
- Create: `apps/api/internal/db/userjwt_integration_test.go`

- [ ] **Step 1: Executor 인터페이스 작성**

`apps/api/internal/db/executor.go`:

```go
package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Executor는 *pgxpool.Pool과 pgx.Tx가 모두 만족하는 최소 인터페이스.
// repo는 이 타입을 받아 호출자가 트랜잭션·풀을 선택할 수 있게 한다.
// 사용자 데이터 repo는 handler에서 AsUser로 트랜잭션을 열어 전달,
// 공개 데이터 repo·cron은 pool을 직접 전달한다.
type Executor interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}
```

- [ ] **Step 2: AsUser 헬퍼 구현**

`apps/api/internal/db/userjwt.go`:

```go
package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AsUser opens a transaction, sets role=authenticated and the JWT claims
// (LOCAL scope) so Supabase RLS policies that reference auth.uid() apply.
// 인자로 받은 ctx의 cancel 여부와 무관하게 rollback이 깨끗하게 끝나도록
// rollback은 별도 background ctx로 호출한다.
//
// set_config(_, _, true) → LOCAL — 트랜잭션 종료 시 자동 해제.
// SET LOCAL은 placeholder를 지원하지 않으므로 set_config 함수를 사용한다.
// Supabase auth.uid()는 claim.sub 폴백을 먼저 시도하므로 둘 다 set한다.
func AsUser(ctx context.Context, pool *pgxpool.Pool, userID string, fn func(Executor) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	claims, err := json.Marshal(map[string]any{
		"sub":  userID,
		"role": "authenticated",
	})
	if err != nil {
		return fmt.Errorf("marshal claims: %w", err)
	}

	if _, err := tx.Exec(ctx, `select set_config('role', 'authenticated', true)`); err != nil {
		return fmt.Errorf("set role: %w", err)
	}
	if _, err := tx.Exec(ctx, `select set_config('request.jwt.claims', $1, true)`, string(claims)); err != nil {
		return fmt.Errorf("set claims: %w", err)
	}
	if _, err := tx.Exec(ctx, `select set_config('request.jwt.claim.sub', $1, true)`, userID); err != nil {
		return fmt.Errorf("set claim.sub: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
```

`Executor` 인터페이스가 `*pgxpool.Pool`·`pgx.Tx` 양쪽을 정말 만족하는지 컴파일 타임 검증을 같은 파일 하단에:

```go
// Compile-time assertions.
var _ Executor = (*pgxpool.Pool)(nil)
var _ Executor = (pgx.Tx)(nil)
```

(import에 `pgx` 추가)

- [ ] **Step 3: integration 테스트 작성**

`apps/api/internal/db/userjwt_integration_test.go`:

```go
//go:build integration

package db_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/db"
)

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestAsUser_SetsRoleAndClaims(t *testing.T) {
	pool := newTestPool(t)
	const uid = "00000000-0000-0000-0000-000000000001"
	err := db.AsUser(context.Background(), pool, uid, func(exec db.Executor) error {
		var role string
		if err := exec.QueryRow(context.Background(), `select current_setting('role', true)`).Scan(&role); err != nil {
			return err
		}
		if role != "authenticated" {
			t.Fatalf("role = %q, want authenticated", role)
		}
		var sub string
		if err := exec.QueryRow(context.Background(), `select current_setting('request.jwt.claim.sub', true)`).Scan(&sub); err != nil {
			return err
		}
		if sub != uid {
			t.Fatalf("claim.sub = %q, want %q", sub, uid)
		}
		// Supabase auth.uid()가 실제로 같은 값을 반환하는지 종단 검증
		var authUID string
		if err := exec.QueryRow(context.Background(), `select auth.uid()::text`).Scan(&authUID); err != nil {
			return err
		}
		if authUID != uid {
			t.Fatalf("auth.uid() = %q, want %q", authUID, uid)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("AsUser: %v", err)
	}
}

func TestAsUser_RollbackOnError(t *testing.T) {
	pool := newTestPool(t)
	sentinel := errors.New("boom")
	err := db.AsUser(context.Background(), pool, "00000000-0000-0000-0000-000000000001", func(exec db.Executor) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("want sentinel, got %v", err)
	}
}
```

- [ ] **Step 4: 빌드 + integration 테스트 실행**

```bash
cd apps/api && go build ./...
cd apps/api && TEST_DATABASE_URL=$(supabase status -o json | jq -r .DB_URL) go test -tags integration ./internal/db -run TestAsUser -v
```

Expected: 컴파일 OK, 2개 테스트 PASS(또는 TEST_DATABASE_URL 미설정 시 SKIP).

- [ ] **Step 5: 커밋**

```bash
git add apps/api/internal/db/executor.go apps/api/internal/db/userjwt.go apps/api/internal/db/userjwt_integration_test.go
git commit -m "feat(db): Executor 인터페이스 + AsUser JWT 트랜잭션 헬퍼"
```

> 단독 commit 후 `db.Executor`·`db.AsUser` 사용처 없음 → `go vet ./...`은 unused 경고 없음(인터페이스·exported 함수는 unused 검사 대상 외). 빌드는 깨끗.

---

## Task 2: profile repo·handler 전환

**Files:**
- Modify: `apps/api/internal/handlers/profile_repo_pg.go`
- Modify: `apps/api/internal/handlers/profiles.go`
- Modify: `apps/api/internal/handlers/profiles_test.go`
- Modify: `apps/api/cmd/server/main.go`

- [ ] **Step 1: PgProfileRepo 시그니처 변경 + TODO 코멘트 제거**

`apps/api/internal/handlers/profile_repo_pg.go` 전체 교체(상단 `TODO(security)` 블록 함께 제거):

```go
package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/quotient/quotient/apps/api/internal/db"
)

var ErrProfileNotFound = errors.New("profile not found")

// PgProfileRepo는 stateless. 호출자가 db.Executor(트랜잭션 또는 풀)를 주입.
type PgProfileRepo struct{}

func NewPgProfileRepo() *PgProfileRepo { return &PgProfileRepo{} }

func (r *PgProfileRepo) Get(ctx context.Context, exec db.Executor, uid string) (map[string]any, error) {
	row := exec.QueryRow(ctx, `
		select id, display_name, base_currency, ui_intensity,
		       onboarding_completed, daily_briefing_enabled,
		       created_at, updated_at
		from public.profiles where id = $1`, uid)
	var id, baseCurrency, uiIntensity string
	var displayName *string
	var onboarding, daily bool
	var created, updated any
	if err := row.Scan(&id, &displayName, &baseCurrency, &uiIntensity, &onboarding, &daily, &created, &updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
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

func (r *PgProfileRepo) Update(ctx context.Context, exec db.Executor, uid string, patch map[string]any) (map[string]any, error) {
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
		return r.Get(ctx, exec, uid)
	}
	args = append(args, uid)
	q := fmt.Sprintf(`update public.profiles set %s where id = $%d
		returning id, display_name, base_currency, ui_intensity,
		          onboarding_completed, daily_briefing_enabled, created_at, updated_at`,
		strings.Join(sets, ", "), i)

	row := exec.QueryRow(ctx, q, args...)
	var id, baseCurrency, uiIntensity string
	var displayName *string
	var onboarding, daily bool
	var created, updated any
	if err := row.Scan(&id, &displayName, &baseCurrency, &uiIntensity, &onboarding, &daily, &created, &updated); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
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
```

> **WHERE user_id/id 필터 유지 결정**: RLS가 동일 행을 한 번 더 평가하므로 redundant이지만, fail-safe 이중 방어로 유지. 404 vs 403 구분이 필요해지면 별도 PR에서 RLS only로 전환.

- [ ] **Step 2: ProfileRepo 인터페이스 + txRunner 패턴**

`apps/api/internal/handlers/profiles.go` 상단:

```go
import (
	// 기존 import 유지
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
)

type ProfileRepo interface {
	Get(ctx context.Context, exec db.Executor, uid string) (map[string]any, error)
	Update(ctx context.Context, exec db.Executor, uid string, patch map[string]any) (map[string]any, error)
}

// txRunner는 핸들러가 사용하는 트랜잭션 wrap 추상화.
// production: db.AsUser(pool, uid, fn). test: passthrough(nil exec).
type txRunner func(ctx context.Context, fn func(db.Executor) error) error

type ProfileHandler struct {
	repo ProfileRepo
	run  txRunner
}

// NewProfileHandler는 production용. pool == nil이면 test passthrough.
// 테스트에서는 NewProfileHandler(repo, nil)로 호출하여 fake repo가 exec를 무시.
func NewProfileHandler(repo ProfileRepo, pool *pgxpool.Pool) *ProfileHandler {
	h := &ProfileHandler{repo: repo}
	if pool == nil {
		h.run = func(ctx context.Context, fn func(db.Executor) error) error { return fn(nil) }
		return h
	}
	h.run = func(ctx context.Context, fn func(db.Executor) error) error {
		uid := middleware.UserID(ctx)
		return db.AsUser(ctx, pool, uid, fn)
	}
	return h
}
```

핸들러 메서드 변환 예:

```go
func (h *ProfileHandler) Get(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	var out map[string]any
	err := h.run(r.Context(), func(exec db.Executor) error {
		p, err := h.repo.Get(r.Context(), exec, uid)
		if err != nil {
			return err
		}
		out = p
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrProfileNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "no profile")
			return
		}
		slog.Error("profile get failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "get failed")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
```

Update도 같은 패턴(트랜잭션 안에서 patch 처리 후 반환값 캡처).

- [ ] **Step 3: profiles_test.go fake repo 시그니처 갱신**

`fakeProfileRepo`(또는 그에 해당하는 타입)의 모든 메서드에 `exec db.Executor` 인자를 첫 번째 ctx 직후에 추가. 본문에서는 `_ = exec`로 무시. 핸들러 생성은 기존대로 `NewProfileHandler(fake, nil)` — passthrough 분기 자동 선택.

- [ ] **Step 4: main.go 주입 수정**

```go
profileRepo := handlers.NewPgProfileRepo()
profileHandler := handlers.NewProfileHandler(profileRepo, pool)
```

- [ ] **Step 5: 빌드·테스트**

```bash
cd apps/api && go build ./... && go test ./internal/handlers -run Profile -v
```

Expected: PASS.

- [ ] **Step 6: 커밋**

```bash
git add apps/api/internal/handlers/profile_repo_pg.go apps/api/internal/handlers/profiles.go apps/api/internal/handlers/profiles_test.go apps/api/cmd/server/main.go
git commit -m "refactor(api): profile handler를 사용자 JWT 풀(AsUser)로 전환"
```

---

## Task 3: holdings repo·handler 전환

**Files:**
- Modify: `apps/api/internal/handlers/holdings_repo_pg.go`
- Modify: `apps/api/internal/handlers/holdings.go`
- Modify: `apps/api/internal/handlers/holdings_test.go`

- [ ] **Step 1: HoldingRepo 인터페이스 + 구현 시그니처 변경**

```go
type HoldingRepo interface {
	List(ctx context.Context, exec db.Executor, userID string) ([]models.HoldingEnriched, error)
	Create(ctx context.Context, exec db.Executor, userID, instrumentID string, qty, avgCost float64, openedAt *string, note *string) (*models.Holding, error)
	Update(ctx context.Context, exec db.Executor, userID, id string, patch map[string]any) (*models.Holding, error)
	Delete(ctx context.Context, exec db.Executor, userID, id string) error
}

type PgHoldingRepo struct{}
func NewPgHoldingRepo() *PgHoldingRepo { return &PgHoldingRepo{} }
```

모든 메서드 본문에서 `r.pool` → `exec`. `WHERE user_id = $N` 조건 유지(이중 방어).

- [ ] **Step 2: HoldingHandler txRunner 패턴**

`holdings.go`에 Task 2와 동일한 `txRunner`·생성자 가드 패턴 적용. `NewHoldingHandler(repo, pool)` 시그니처 유지(이미 그 형태) → `pool == nil`일 때 passthrough.

핸들러 메서드는 `h.run(r.Context(), func(exec db.Executor) error { ... })`로 wrap.

- [ ] **Step 3: holdings_test.go fake repo 시그니처 갱신**

`fakeHoldingRepo`의 모든 메서드에 `exec db.Executor` 추가, 본문에서는 `_ = exec`. 테스트의 `NewHoldingHandler(fake, nil)` 호출은 그대로 동작(가드 분기).

- [ ] **Step 4: main.go 주입 수정**

```go
holdingRepo := handlers.NewPgHoldingRepo()
holdingHandler := handlers.NewHoldingHandler(holdingRepo, pool)
```

- [ ] **Step 5: 빌드·테스트**

```bash
cd apps/api && go build ./... && go test ./internal/handlers -run Holding -v
```

- [ ] **Step 6: 커밋**

```bash
git add apps/api/internal/handlers/holdings_repo_pg.go apps/api/internal/handlers/holdings.go apps/api/internal/handlers/holdings_test.go apps/api/cmd/server/main.go
git commit -m "refactor(api): holdings handler를 사용자 JWT 풀로 전환"
```

---

## Task 4: watchlist repo·handler 전환

**Files:**
- Modify: `apps/api/internal/handlers/watchlist_repo_pg.go`
- Modify: `apps/api/internal/handlers/watchlist.go`
- Modify: `apps/api/internal/handlers/watchlist_test.go`

- [ ] **Step 1: WatchlistRepo 시그니처 변경**

```go
type WatchlistRepo interface {
	List(ctx context.Context, exec db.Executor, userID string) ([]models.WatchlistItem, error)
	Add(ctx context.Context, exec db.Executor, userID, instrumentID string) error
	Remove(ctx context.Context, exec db.Executor, userID, instrumentID string) error
}

type PgWatchlistRepo struct{}
func NewPgWatchlistRepo() *PgWatchlistRepo { return &PgWatchlistRepo{} }
```

모든 메서드 본문 `r.pool` → `exec`.

- [ ] **Step 2: WatchlistHandler txRunner 패턴**

Task 2와 동일. `NewWatchlistHandler(repo, pool)`의 pool nil 가드.

- [ ] **Step 3: watchlist_test.go fake repo 시그니처 갱신**

- [ ] **Step 4: main.go 주입 수정**

```go
watchlistRepo := handlers.NewPgWatchlistRepo()
watchlistHandler := handlers.NewWatchlistHandler(watchlistRepo, pool)
```

- [ ] **Step 5: 빌드·테스트**

```bash
cd apps/api && go build ./... && go test ./internal/handlers -run Watchlist -v
```

- [ ] **Step 6: 커밋**

```bash
git add apps/api/internal/handlers/watchlist_repo_pg.go apps/api/internal/handlers/watchlist.go apps/api/internal/handlers/watchlist_test.go apps/api/cmd/server/main.go
git commit -m "refactor(api): watchlist handler를 사용자 JWT 풀로 전환"
```

---

## Task 5: briefing handler 전환

**Files:**
- Modify: `apps/api/internal/handlers/briefing.go`

> `BriefingHandler.Today`는 사용자 read 엔드포인트. `ai_briefings` 테이블엔 RLS 정책(`ai_briefings_select_own`)이 정의돼 있으므로 JWT 풀로 전환해야 일관.
> 브리핑 *작성*(cron `JobBriefingDispatcher`)은 다중 사용자 fan-out → 슈퍼유저 풀 유지.

- [ ] **Step 1: BriefingHandler 시그니처 + 핸들러 변경**

```go
type BriefingHandler struct {
	pool *pgxpool.Pool
}

func NewBriefingHandler(pool *pgxpool.Pool) *BriefingHandler {
	return &BriefingHandler{pool: pool}
}

func (h *BriefingHandler) Today(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	loc, _ := time.LoadLocation("Asia/Seoul")
	today := time.Now().In(loc).Format("2006-01-02")
	var b *models.AIBriefing
	err := db.AsUser(r.Context(), h.pool, uid, func(exec db.Executor) error {
		got, err := loadBriefing(r.Context(), exec, uid, today)
		if err != nil {
			return err
		}
		b = got
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrBriefingNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "no briefing for today")
			return
		}
		slog.Error("briefing load failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// loadBriefing은 exec를 받아 트랜잭션·풀 양쪽에서 호출 가능.
// cron worker도 동일 함수를 슈퍼유저 풀로 호출할 수 있다.
func loadBriefing(ctx context.Context, exec db.Executor, uid, date string) (*models.AIBriefing, error) {
	var b models.AIBriefing
	row := exec.QueryRow(ctx, `
		select user_id::text, date::text, content_md, model, created_at
		from public.ai_briefings where user_id = $1 and date = $2
	`, uid, date)
	if err := row.Scan(&b.UserID, &b.Date, &b.ContentMD, &b.Model, &b.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBriefingNotFound
		}
		return nil, err
	}
	return &b, nil
}
```

- [ ] **Step 2: 빌드**

```bash
cd apps/api && go build ./...
```

Expected: OK (loadBriefing 호출자가 handler 외에 없으면 그대로 통과).

- [ ] **Step 3: 커밋**

```bash
git add apps/api/internal/handlers/briefing.go
git commit -m "refactor(api): briefing.Today를 사용자 JWT 풀로 전환"
```

---

## Task 6: chat repo 시그니처 + 동기 핸들러 전환

**Files:**
- Modify: `apps/api/internal/handlers/chat_repo_pg.go`
- Modify: `apps/api/internal/handlers/chat.go` (read endpoints만)
- Modify: `apps/api/internal/handlers/chat_test.go`
- Modify: `apps/api/cmd/server/main.go`

- [ ] **Step 1: ChatRepo 인터페이스 + 구현 시그니처 변경**

```go
type ChatRepo interface {
	CreateSession(ctx context.Context, exec db.Executor, userID, title string) (*models.ChatSession, error)
	ListSessions(ctx context.Context, exec db.Executor, userID string) ([]models.ChatSession, error)
	DeleteSession(ctx context.Context, exec db.Executor, userID, sessionID string) error
	ListMessages(ctx context.Context, exec db.Executor, userID, sessionID string) ([]models.ChatMessage, error)
	AppendMessage(ctx context.Context, exec db.Executor, sessionID string, m models.ChatMessage) (*models.ChatMessage, error)
	MarkFinished(ctx context.Context, exec db.Executor, messageID string) error  // 기존 인터페이스 보존(미사용)
	UnfinishedInSession(ctx context.Context, exec db.Executor, userID, sessionID string) (*models.ChatMessage, error)

	GetUsage(ctx context.Context, exec db.Executor, userID string) (*models.ChatUsageMonthly, error)
	IncrementUsage(ctx context.Context, exec db.Executor, userID string, inTok, outTok int, isOpus bool) error
	CheckLimits(ctx context.Context, exec db.Executor, userID string, isOpus bool) error
}

type PgChatRepo struct{}
func NewPgChatRepo() *PgChatRepo { return &PgChatRepo{} }
```

모든 메서드 본문 `r.pool` → `exec`. `CheckLimits`는 `GetUsage(ctx, exec, userID)` 호출.
`AppendMessage`의 두 번째 Exec(`update chat_sessions set updated_at`)도 `exec`로 동일 트랜잭션 안에서 실행.

- [ ] **Step 2: ChatHandler 구조체 + 동기 read 핸들러 전환**

```go
type ChatHandler struct {
	repo     ChatRepo
	pool     *pgxpool.Pool
	client   ai.Client
	registry *tools.Registry
	run      txRunner
}

func NewChatHandler(repo ChatRepo, pool *pgxpool.Pool, client ai.Client, registry *tools.Registry) *ChatHandler {
	h := &ChatHandler{repo: repo, pool: pool, client: client, registry: registry}
	if pool == nil {
		h.run = func(ctx context.Context, fn func(db.Executor) error) error { return fn(nil) }
		return h
	}
	h.run = func(ctx context.Context, fn func(db.Executor) error) error {
		uid := middleware.UserID(ctx)
		return db.AsUser(ctx, pool, uid, fn)
	}
	return h
}
```

다음 핸들러를 `h.run`으로 wrap:
- `ListSessions` (GET /v1/chat/sessions)
- `DeleteSession` (DELETE /v1/chat/sessions/:id)
- `ListMessages` (GET /v1/chat/sessions/:id/messages)
- `GetUsage` (GET /v1/chat/usage)
- `UnfinishedInSession` (있다면)
- `CreateSession` 엔드포인트(있다면)

> **StreamChat는 Task 7에서 별도 처리.** SSE 루프 안 inline persist는 ctx 분리 + tx 분리가 필요.

- [ ] **Step 3: chat_test.go fake repo 시그니처 갱신**

`fakeChatRepo`의 모든 메서드에 `exec db.Executor` 인자 추가, 본문은 `_ = exec`. `NewChatHandler(fake, nil, ...)` 호출은 그대로 동작.

- [ ] **Step 4: main.go 주입 수정**

```go
chatRepo := handlers.NewPgChatRepo()
chatHandler := handlers.NewChatHandler(chatRepo, pool, aiClient, toolRegistry)
```

- [ ] **Step 5: 빌드·테스트**

```bash
cd apps/api && go build ./... && go test ./internal/handlers -run Chat -v
```

Expected: PASS. `TestStreamChat_*`는 inline persist 경로가 fake repo로 가므로 동작.

- [ ] **Step 6: 커밋**

```bash
git add apps/api/internal/handlers/chat_repo_pg.go apps/api/internal/handlers/chat.go apps/api/internal/handlers/chat_test.go apps/api/cmd/server/main.go
git commit -m "refactor(api): chat repo + 동기 read 핸들러를 JWT 풀로 전환"
```

---

## Task 7: chat StreamChat inline 영속화 전환

**Files:**
- Modify: `apps/api/internal/handlers/chat.go` (StreamChat 본문)

> **현재 코드 사실 확인** (`chat.go:80-277`):
> - line 81: `ListMessages` (sync, r.Context()) — Task 6에서 이미 wrap.
> - line 93: 사용자 메시지 `AppendMessage` (sync, r.Context()).
> - line 215-225: assistant turn `AppendMessage` (inline, `context.Background()` w/ 5s timeout).
> - line 245-252: tool result `AppendMessage` (inline, `context.Background()` w/ 5s timeout).
> - line 265-269: `IncrementUsage` (inline, `context.Background()` w/ 5s timeout).
> - **별도 goroutine 없음**. assistant·tool·usage 영속화는 SSE 루프 안 inline.
> - **MarkFinished는 호출되지 않음**. AppendMessage가 `FinishedAt=ptrTime(time.Now())`를 직접 INSERT.

각 inline persist를 `db.AsUser`로 wrap. ctx는 그대로 `context.Background()` 기반(사용자 disconnect 후에도 영속화 보장). 사용자 메시지 영속화는 r.Context() 유지(SSE 시작 전이라 cancel risk 낮음).

- [ ] **Step 1: 사용자 메시지 영속화 wrap (line 93)**

기존:
```go
_, err = h.repo.AppendMessage(r.Context(), body.SessionID, models.ChatMessage{...})
```

신규:
```go
err = h.run(r.Context(), func(exec db.Executor) error {
	_, err := h.repo.AppendMessage(r.Context(), exec, body.SessionID, models.ChatMessage{
		Role: "user", Content: body.Message,
		FinishedAt: ptrTime(time.Now()),
	})
	return err
})
```

- [ ] **Step 2: assistant turn 영속화 wrap (line 215-230)**

기존:
```go
persistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
saved, err := h.repo.AppendMessage(persistCtx, body.SessionID, models.ChatMessage{...})
cancel()
if err != nil {
	slog.Error("append assistant turn failed", "err", err)
} else if saved != nil {
	lastSavedID = saved.ID
}
```

신규(사용자 disconnect 후에도 영속화 보장 → `context.Background()` 기반 + `db.AsUser`에 uid 전달):
```go
persistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
var saved *models.ChatMessage
err := db.AsUser(persistCtx, h.pool, uid, func(exec db.Executor) error {
	s, perr := h.repo.AppendMessage(persistCtx, exec, body.SessionID, models.ChatMessage{
		Role:         "assistant",
		Content:      turnText,
		ToolCalls:    blocksJSON,
		InputTokens:  totalInput,
		OutputTokens: totalOutput,
		Model:        &model,
		FinishedAt:   ptrTime(time.Now()),
	})
	if perr != nil {
		return perr
	}
	saved = s
	return nil
})
cancel()
if err != nil {
	slog.Error("append assistant turn failed", "err", err)
} else if saved != nil {
	lastSavedID = saved.ID
}
```

> **테스트 호환 메모**: `h.pool == nil`인 테스트 경로에서는 `db.AsUser(nil pool ...)`가 panic. → StreamChat 안에서는 `h.run` 클로저를 사용하지 않고 `db.AsUser`를 직접 부르기 때문에 panic 위험. 해결: StreamChat 본문도 `h.run`을 사용하되, **`h.run`이 `context.Background()` 기반 ctx도 받을 수 있게 만든다**. txRunner 시그니처는 이미 `ctx context.Context`를 받으므로 그대로 사용:
>
> ```go
> err := h.run(persistCtx, func(exec db.Executor) error { ... })
> ```
>
> `h.run` 안에서 `middleware.UserID(ctx)`를 호출하는데 `persistCtx`는 `context.Background()` 기반이라 UserID가 빈 문자열. 따라서 chat handler용 `runAs(uid, ctx, fn)`을 별도 도입:
>
> ```go
> runAs func(ctx context.Context, uid string, fn func(db.Executor) error) error
> ```
>
> 생성자에서:
> ```go
> if pool == nil {
> 	h.runAs = func(ctx context.Context, uid string, fn func(db.Executor) error) error { return fn(nil) }
> } else {
> 	h.runAs = func(ctx context.Context, uid string, fn func(db.Executor) error) error {
> 		return db.AsUser(ctx, pool, uid, fn)
> 	}
> }
> ```
>
> StreamChat 안 모든 wrap은 `h.runAs(persistCtx, uid, fn)` 형태. Task 6에서 도입한 `h.run`은 read 핸들러용(ctx에서 uid 추출). **Task 6 마지막 commit 전에 ChatHandler 구조체에 `runAs`를 함께 추가**해두는 게 더 일관 — Task 6 Step 2 본문을 그렇게 수정.

> 정정: 위 메모를 Task 6 Step 2에 반영하라 — `ChatHandler`는 read용 `run`과 inline용 `runAs` 두 필드를 모두 갖는다.

- [ ] **Step 3: tool result 영속화 wrap (line 245-252)**

```go
toolPersistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
_ = h.runAs(toolPersistCtx, uid, func(exec db.Executor) error {
	_, perr := h.repo.AppendMessage(toolPersistCtx, exec, body.SessionID, models.ChatMessage{
		Role:       "tool",
		ToolCalls:  resultsJSON,
		FinishedAt: ptrTime(time.Now()),
	})
	return perr
})
cancel()
```

- [ ] **Step 4: usage 영속화 wrap (line 265-269)**

```go
usageCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
if err := h.runAs(usageCtx, uid, func(exec db.Executor) error {
	return h.repo.IncrementUsage(usageCtx, exec, uid, totalInput, totalOutput, body.UseOpus)
}); err != nil {
	slog.Error("usage increment failed", "err", err)
}
cancel()
```

- [ ] **Step 5: 빌드·테스트**

```bash
cd apps/api && go build ./... && go test ./internal/handlers -run Chat -v
```

Expected: PASS. `TestStreamChat_TextOnly` 등 fake repo 테스트는 `pool == nil` passthrough로 정상.

- [ ] **Step 6: 수동 SSE 통합 검증**

`supabase start` + `cd apps/api && go run ./cmd/server` 실행 후 별도 터미널에서 dev JWT로:

```bash
TOKEN=$(curl -sX POST "$SUPABASE_URL/auth/v1/token?grant_type=password" \
  -H "apikey: $SUPABASE_ANON_KEY" \
  -d '{"email":"dev@quotient.local","password":"<dev-pw>"}' | jq -r .access_token)
curl -N -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"session_id":"<existing>","message":"테스트"}' \
  http://localhost:8080/v1/chat/stream
```

Expected: SSE 토큰 스트림 정상, 종료 후 `psql` 확인:

```sql
select role, length(content), finished_at is not null from chat_messages
where session_id = '<...>' order by created_at desc limit 4;
```

→ user/assistant 행 모두 존재, `finished_at` 모두 NOT NULL.

- [ ] **Step 7: 커밋**

```bash
git add apps/api/internal/handlers/chat.go
git commit -m "refactor(api): chat StreamChat inline 영속화를 사용자 JWT 트랜잭션으로 전환"
```

---

## Task 8: RLS 격리 통합 테스트

**Files:**
- Create: `apps/api/internal/db/rls_integration_test.go`

- [ ] **Step 1: 통합 테스트 작성**

`apps/api/internal/db/rls_integration_test.go`:

```go
//go:build integration

package db_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/db"
)

// 본 테스트는 로컬 Supabase가 떠 있어야 한다 (`supabase start`).
// 모든 마이그레이션이 적용된 상태에서 RLS가 사용자 격리하는지 검증.
//
// 사전 조건:
// - auth.users에 직접 INSERT 시 on_auth_user_created 트리거가
//   public.profiles 행을 자동 생성한다(handle_new_user, SECURITY DEFINER).
//   따라서 테스트는 auth.users만 seed하고 profiles는 트리거에 맡긴다.
// - holdings/watchlist/chat_*는 user_id에 on delete cascade가 걸려 있어
//   auth.users DELETE 한 번으로 모두 정리된다(cleanup 명시 호출은 over-explicit).
func TestRLS_HoldingsIsolation(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	userA := uuid.NewString()
	userB := uuid.NewString()
	mustSeedUser(t, pool, userA)
	mustSeedUser(t, pool, userB)
	defer cleanupUser(pool, userA)
	defer cleanupUser(pool, userB)

	var instID string
	if err := pool.QueryRow(context.Background(), `select id::text from instruments limit 1`).Scan(&instID); err != nil {
		t.Skip("no instrument seeded")
	}

	if err := db.AsUser(context.Background(), pool, userA, func(exec db.Executor) error {
		_, err := exec.Exec(context.Background(), `
			insert into holdings (user_id, instrument_id, quantity, avg_cost)
			values ($1, $2, 10, 50000)
		`, userA, instID)
		return err
	}); err != nil {
		t.Fatalf("user A insert: %v", err)
	}

	var bCount int
	if err := db.AsUser(context.Background(), pool, userB, func(exec db.Executor) error {
		return exec.QueryRow(context.Background(), `select count(*) from holdings`).Scan(&bCount)
	}); err != nil {
		t.Fatalf("user B select: %v", err)
	}
	if bCount != 0 {
		t.Fatalf("RLS leak: user B sees %d holdings of user A", bCount)
	}

	var aCount int
	if err := db.AsUser(context.Background(), pool, userA, func(exec db.Executor) error {
		return exec.QueryRow(context.Background(), `select count(*) from holdings`).Scan(&aCount)
	}); err != nil {
		t.Fatalf("user A select: %v", err)
	}
	if aCount != 1 {
		t.Fatalf("user A sees %d holdings, want 1", aCount)
	}
}

// TestRLS_WatchlistIsolation / TestRLS_ChatSessionsIsolation도 동일 패턴.
// (생략 — 같은 셋업/검증, 테이블만 watchlist / chat_sessions로 교체)

// mustSeedUser는 슈퍼유저 풀로 auth.users 1행을 만든다.
// handle_new_user 트리거가 public.profiles 행을 자동 생성하므로 별도 INSERT 불필요.
func mustSeedUser(t *testing.T, pool *pgxpool.Pool, uid string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		insert into auth.users (id, email, encrypted_password)
		values ($1, $1 || '@test.local', '')
		on conflict (id) do nothing
	`, uid)
	if err != nil {
		t.Fatalf("seed auth.users: %v", err)
	}
}

// cleanupUser는 t와 무관하게 호출 가능(defer 안전성). auth.users 삭제 → cascade.
func cleanupUser(pool *pgxpool.Pool, uid string) {
	_, _ = pool.Exec(context.Background(), `delete from auth.users where id = $1`, uid)
}
```

- [ ] **Step 2: watchlist + chat_sessions 격리 케이스 추가**

같은 파일에 `TestRLS_WatchlistIsolation`, `TestRLS_ChatSessionsIsolation` 함수 추가. 패턴 동일(테이블만 교체).

- [ ] **Step 3: 실행**

```bash
cd apps/api && TEST_DATABASE_URL=$(supabase status -o json | jq -r .DB_URL) go test -tags integration ./internal/db -v
```

Expected: 3+α 케이스 모두 PASS.

- [ ] **Step 4: 커밋**

```bash
git add apps/api/internal/db/rls_integration_test.go
git commit -m "test(db): RLS 사용자 격리 통합 테스트 (holdings·watchlist·chat)"
```

---

## Task 9: 문서 갱신 + 후속 결정 명시

**Files:**
- Modify: `docs/STATUS.md`
- Modify: `docs/ARCHITECTURE.md`
- Modify: `docs/ROADMAP.md` (해당 시)

- [ ] **Step 1: STATUS.md 갱신**

"알려진 결함" 섹션에서 "RLS 우회"·"사용자 JWT 미전파" 항목 제거(또는 ✅로 이동).
"최근 변경 이력" 맨 위 한 줄:

```markdown
- 2026-05-27: 사용자 데이터 핸들러(profile/holdings/watchlist/chat/briefing)를 `db.AsUser` JWT 트랜잭션으로 전환 — Supabase RLS 자동 적용 + 애플리케이션 `WHERE user_id` 이중 방어 유지
```

"마지막 업데이트" 날짜 갱신.

- [ ] **Step 2: ARCHITECTURE.md "핵심 설계 결정"에 섹션 추가**

```markdown
### 사용자 JWT 풀 패턴 (db.AsUser)

**결정**: 사용자 데이터 핸들러는 단일 슈퍼유저 풀 위에서 트랜잭션을 열어
`SET LOCAL role = authenticated` + `request.jwt.claims` + `request.jwt.claim.sub`를 LOCAL 적용한다.
cron 잡·도구 레지스트리·공개 read(market/instruments/history)·브리핑 작성 워커는 슈퍼유저 풀 그대로 사용.

**Why**:
- 풀을 분리하지 않아 connection 소비를 1배수로 유지(Supabase Free 60 connection 한도 압박 회피).
- `set_config(_, _, true)`로 LOCAL 적용 — 트랜잭션 종료 시 자동 해제, leak 불가.
- 애플리케이션 `WHERE user_id = $1` 필터는 fail-safe 이중 방어로 유지.
- cron/brief 작성은 다중 사용자 fan-out이라 단일 사용자 JWT를 가질 수 없음 → 슈퍼유저 풀이 자연스러움.

**How**:
- `internal/db/executor.go`: `Executor` 인터페이스(=`*pgxpool.Pool` ∪ `pgx.Tx`).
- `internal/db/userjwt.go`: `AsUser(ctx, pool, userID, fn)` 헬퍼. `claims` JSON + `claim.sub` 폴백 둘 다 set.
- repo는 stateless, 메서드가 `db.Executor` 받음.
- handler는 `txRunner`/`runAs` 클로저로 wrap. test에서는 `pool == nil` passthrough로 fake repo 호환.
- 보호 대상 테이블: `profiles`, `holdings`, `watchlist`, `chat_sessions`, `chat_messages`, `chat_usage_monthly`, `ai_briefings`.
- RLS 격리 회귀 가드: `internal/db/rls_integration_test.go` (integration build tag).

**비용**:
- 요청당 BEGIN + set_config*3 + COMMIT = 5 round-trip 추가. 로컬 PG에서는 무시 가능, US east → 한국 RTT 200ms 가정 시 1s 추가 가능. Vercel→Fly→Supabase 모두 같은 region에 배치 권장.

**범위 외**:
- AI 도구(portfolio·search·quote)의 사용자 데이터 read는 여전히 슈퍼유저 풀 + handler 단 `user_id` 필터 사용. spec §10-1 완전 정합을 위해서는 도구 시그니처에 `exec db.Executor` 인자 추가 + chat handler tool routing 안에서 `db.AsUser` wrap이 필요. 별도 PR로 분리.
```

- [ ] **Step 3: ROADMAP.md 갱신 (해당 시)**

"현재 추천 다음 작업"에서 RLS JWT 전환 항목 제거. 후속 작업으로 "AI 도구 호출 경로 JWT 전파" 추가.

- [ ] **Step 4: 잔여 TODO 코멘트 제거 확인**

```bash
cd /Users/yuhojin/Desktop/finance && grep -r "TODO(security)" apps/api/internal/handlers/
```

남은 코멘트가 있다면 같은 commit에서 제거.

- [ ] **Step 5: 빌드·전체 테스트**

```bash
cd apps/api && go build ./... && go test ./...
```

Expected: 컴파일 + unit 테스트 통과(integration은 별도 tag).

- [ ] **Step 6: 커밋**

```bash
git add docs/STATUS.md docs/ARCHITECTURE.md docs/ROADMAP.md
git commit -m "docs: RLS JWT 풀 전환 완료 반영 (STATUS·ARCHITECTURE·ROADMAP)"
```

---

## Self-Review

**1. Spec coverage**:
- ✅ SET LOCAL ROLE + claims + claim.sub 폴백 — Task 1.
- ✅ 사용자 데이터 핸들러 전환 — Task 2~6.
- ✅ briefing.Today 포함 — Task 5.
- ✅ chat inline persist 정확한 라인(line 93/215/247/265) 매핑 — Task 7.
- ✅ cron/도구/공개 read는 슈퍼유저 풀 유지 — 파일 목록에서 명시적으로 제외, ARCHITECTURE Why 섹션에 기술.
- ✅ AI 도구 호출 JWT 전파는 plan 범위 밖 — Task 9 후속 결정으로 명시.
- ✅ RLS 격리 통합 테스트 — Task 8.
- ✅ 문서·TODO 코멘트 정리 — Task 9.

**2. Placeholder scan**: 없음. 모든 코드 블록은 실행 가능, 명령어 expected output 명시.

**3. Type consistency**:
- `Executor` 인터페이스 시그니처(Task 1) → 모든 repo 메서드(Task 2~6) 일관 사용.
- `txRunner`(read용) vs `runAs`(inline persist용) 두 클로저를 ChatHandler가 함께 가지는 것으로 통일 — Task 6 Step 2 본문 메모에 반영.
- `pool == nil` passthrough 가드를 profile/holdings/watchlist/chat 모두 동일 패턴 적용.

**4. 빌드 단위 분할**:
- Task 1 단독 commit: `Executor`/`AsUser` exported → unused 경고 없음.
- Task 2~6 commit 사이에 시그니처가 바뀌지만 각 commit이 main.go까지 같이 수정 → 빌드 깨지지 않음.
- Task 7 commit 후에야 chat StreamChat가 최종 형태. Task 6과 Task 7 사이 빌드는 동기 read만 전환된 중간 상태로 정상 동작.
- Task 8 integration test는 Task 1~7 완료 후 의미 — `//go:build integration`로 일반 빌드와 분리되어 영향 없음.

**리스크 노트**:
- assistant turn 영속화 트랜잭션 commit 직전까지 SELECT에서 안 보임. 현재 UI는 SSE 토큰만 받고 별도 fetch 없음 → race 없음. 향후 변경 시 분리 고려(메모만).
- 각 inline persist마다 BEGIN/COMMIT → SSE 응답 지연(commit fsync 대기). 한 turn당 1번이라 사용자 인식 가능한 지연은 아니지만, Supabase region 거리에 비례. ARCHITECTURE 비용 절에 명시.
- `set_config('request.jwt.claim.sub', ..., true)` 폴백 추가 — 현재 Supabase `auth.uid()` 구현이 jsonb 캐스트만으로 충분하지만, upstream 정의가 단순화되어 `claim.sub`만 보는 변형으로 회귀해도 안전.

---

## 검토 이력

- **2026-05-27 초안 작성**: Sonnet — 8 task 구성.
- **2026-05-27 자체 검토(general-purpose subagent)**:
  - Critical 4건:
    1. Task 6이 가상 코드(별도 goroutine, MarkFinished) 가정 → 실제 inline persist 구조로 재작성. 4개 호출 지점 라인 번호 명시. ✅ Task 6+7로 분리·재작성.
    2. fake repo + `nil` pool로 핸들러 생성하는 기존 테스트가 panic 위험 → `pool == nil` passthrough 가드 도입. ✅ 모든 핸들러 생성자에 반영.
    3. `mustSeedUser`가 profiles INSERT를 중복으로 함(트리거가 자동 생성) → seed에서 profiles 줄 제거. ✅ Task 8 본문 반영.
    4. briefing.Today + AI 도구 누락 → briefing은 Task 5로 추가, AI 도구는 명시적 범위 외(Task 9 ARCHITECTURE 범위 외 절). ✅ 반영.
  - Important 6건: `claim.sub` 폴백 추가(✅), `defer rollback` ctx 분리(✅), Task 1을 integration test로 분리(✅), inline persist ctx=`context.Background()` 유지 명시(✅), WHERE user_id 유지 메모(✅), task 분할 단독 commit 안전성 노트(✅).
  - Minor 5건: STATUS.md 위치 명시(✅ Task 9 본문), SSE 토큰 발급 실제 명령(✅), set_config 일관 유지(결정 보존), 운영 RTT 비용(✅ ARCHITECTURE), cleanup Skip 분기(✅ Task 8).

Plan complete and saved to `docs/superpowers/plans/2026-05-27-rls-jwt-pool.md`.
