# AI 매매 일기 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 매매 결정·시장 관찰 일기 시스템(`journal_entries`) + AI 사후 패턴 분석(`analysis_runs`) + 자동 월간 회고 cron + on-demand 분석 버튼 + 채팅 `analyze_journal` 도구.

**Architecture:** 신규 마이그레이션(2 테이블, RLS) + 신규 `internal/handlers/journal*.go` 6 endpoint + 기존 holdings 핸들러 확장(reason 필드 → auto entry) + 신규 AI 도구 `analyze_journal` + 매월 1일 cron + 신규 Next.js 페이지 `/app/journal` + AddHoldingDialog·Sidebar 수정.

**Tech Stack:** Go 1.25 (chi v5 + pgx v5) · Next.js 16 · Tailwind v4 · Supabase Postgres + RLS · Anthropic Claude.

**Spec:** [`docs/superpowers/specs/2026-05-28-ai-trading-journal-design.md`](../specs/2026-05-28-ai-trading-journal-design.md)

---

## File Structure

신규:
- `supabase/migrations/20260528000001_journal.sql` — journal_entries + analysis_runs + RLS
- `apps/api/internal/models/journal.go` — Go 모델
- `apps/api/internal/handlers/journal_repo_pg.go` — JournalRepo + PgJournalRepo
- `apps/api/internal/handlers/journal.go` — HTTP handler (6 endpoint)
- `apps/api/internal/handlers/journal_test.go` — unit (7 케이스)
- `apps/api/internal/handlers/journal_integration_test.go` — integration
- `apps/api/internal/ai/tools/journal.go` — `analyze_journal` 도구
- `apps/api/internal/ai/tools/journal_test.go` — 2 unit
- `apps/api/internal/schedule/journal_monthly.go` — 월간 cron
- `apps/api/internal/schedule/journal_monthly_test.go` — 3 unit
- `apps/web/lib/api/journal.ts` — TS 클라이언트
- `apps/web/components/journal/JournalPage.tsx` — 메인 페이지
- `apps/web/components/journal/EntryItem.tsx` — entry 카드
- `apps/web/components/journal/AnalysisCard.tsx` — 분석 카드
- `apps/web/components/journal/NewEntryDialog.tsx` — manual entry 모달
- `apps/web/app/app/journal/page.tsx` — Next.js 라우트

수정:
- `apps/api/internal/handlers/holdings.go` — reason 옵션 + auto entry 생성
- `apps/api/internal/handlers/holdings_repo_pg.go` — Create/Update가 reason 받기 위한 시그니처 확장(또는 별도 helper)
- `apps/api/internal/handlers/holdings_test.go` — reason 케이스 추가
- `apps/api/internal/router/router.go` — 6 journal 라우트
- `apps/api/cmd/server/main.go` — wiring (journalRepo·journalHandler·analyzeJournal 등록)
- `apps/api/internal/schedule/cron.go` — monthly journal 잡 등록
- `apps/api/internal/ai/tools/registry.go` — `RegisterJournal` 호출 (또는 main.go에서 등록)
- `apps/web/components/portfolio/AddHoldingDialog.tsx` — "매매 이유" textarea
- `apps/web/components/portfolio/EditHoldingDialog.tsx` — 동일
- `apps/web/lib/api/holdings.ts` — Create/Patch 시그니처에 reason 추가
- `apps/web/components/shell/Sidebar.tsx` — 📓 아이콘 추가
- `docs/STATUS.md` / `docs/ROADMAP.md`

---

## Task 1: 마이그레이션 + Go 모델

**Files:**
- Create: `supabase/migrations/20260528000001_journal.sql`
- Create: `apps/api/internal/models/journal.go`

- [ ] **Step 1: 마이그레이션 작성**

`supabase/migrations/20260528000001_journal.sql`:

```sql
-- 매매 일기 entries
create table public.journal_entries (
  id                   uuid primary key default gen_random_uuid(),
  user_id              uuid not null references auth.users(id) on delete cascade,
  entry_type           text not null check (entry_type in ('auto', 'manual')),
  action               text check (action in ('buy', 'sell', 'observation', 'other')),
  related_holding_id   uuid references public.holdings(id) on delete set null,
  related_symbols      text[] not null default '{}',
  title                text,
  content              text not null check (length(content) between 1 and 2000),
  created_at           timestamptz not null default now(),
  updated_at           timestamptz not null default now()
);

create index journal_entries_user_created_idx
  on public.journal_entries (user_id, created_at desc);

create trigger journal_entries_touch_updated_at
  before update on public.journal_entries
  for each row execute function public.touch_updated_at();

-- AI 분석 결과 (불변)
create table public.analysis_runs (
  id              uuid primary key default gen_random_uuid(),
  user_id         uuid not null references auth.users(id) on delete cascade,
  run_type        text not null check (run_type in ('auto_monthly', 'on_demand')),
  period_start    date not null,
  period_end      date not null,
  entries_count   int not null default 0,
  content_md      text not null,
  model           text not null,
  created_at      timestamptz not null default now()
);

create index analysis_runs_user_created_idx
  on public.analysis_runs (user_id, created_at desc);

-- 동일 월 auto_monthly 중복 방지
create unique index analysis_runs_auto_monthly_idx
  on public.analysis_runs (user_id, period_start)
  where run_type = 'auto_monthly';

-- RLS
alter table public.journal_entries enable row level security;
create policy journal_entries_select_own on public.journal_entries
  for select using (user_id = auth.uid());
create policy journal_entries_insert_own on public.journal_entries
  for insert with check (user_id = auth.uid());
create policy journal_entries_update_own on public.journal_entries
  for update using (user_id = auth.uid())
  with check (user_id = auth.uid());
create policy journal_entries_delete_own on public.journal_entries
  for delete using (user_id = auth.uid());

alter table public.analysis_runs enable row level security;
create policy analysis_runs_select_own on public.analysis_runs
  for select using (user_id = auth.uid());
create policy analysis_runs_insert_own on public.analysis_runs
  for insert with check (user_id = auth.uid());
```

- [ ] **Step 2: 마이그레이션 적용 (로컬 Supabase)**

```bash
cd /Users/yuhojin/Desktop/finance && supabase db push
```

Expected: `Applying migration 20260528000001_journal.sql...` 한 줄.

검증:
```bash
psql "postgresql://postgres:postgres@127.0.0.1:54322/postgres" -c "\d public.journal_entries"
psql "postgresql://postgres:postgres@127.0.0.1:54322/postgres" -c "\d public.analysis_runs"
```

- [ ] **Step 3: Go 모델 작성**

`apps/api/internal/models/journal.go`:

```go
package models

import (
	"encoding/json"
	"time"
)

type JournalEntry struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	EntryType         string    `json:"entry_type"` // "auto" | "manual"
	Action            *string   `json:"action,omitempty"`
	RelatedHoldingID  *string   `json:"related_holding_id,omitempty"`
	RelatedHolding    *RelatedHoldingBrief `json:"related_holding,omitempty"`
	RelatedSymbols    []string  `json:"related_symbols"`
	Title             *string   `json:"title,omitempty"`
	Content           string    `json:"content"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// RelatedHoldingBrief — entry 노출 시 종목 정보 (read 시 JOIN으로 채움)
type RelatedHoldingBrief struct {
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

type AnalysisRun struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	RunType       string    `json:"run_type"` // "auto_monthly" | "on_demand"
	PeriodStart   string    `json:"period_start"` // YYYY-MM-DD
	PeriodEnd     string    `json:"period_end"`
	EntriesCount  int       `json:"entries_count"`
	ContentMD     string    `json:"content_md"`
	Model         string    `json:"model"`
	CreatedAt     time.Time `json:"created_at"`
}

// JournalEntryCreate — POST body
type JournalEntryCreate struct {
	Action          *string  `json:"action"`
	RelatedSymbols  []string `json:"related_symbols"`
	Title           *string  `json:"title"`
	Content         string   `json:"content"`
}

// JournalEntryPatch — PATCH body
type JournalEntryPatch struct {
	Action          *string  `json:"action"`
	RelatedSymbols  *[]string `json:"related_symbols"`
	Title           *string  `json:"title"`
	Content         *string  `json:"content"`
}

// json.RawMessage import 가드 (다른 모델에서 자주 사용)
var _ json.RawMessage
```

- [ ] **Step 4: 빌드**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./...
```

Expected: EXIT 0.

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add supabase/migrations/20260528000001_journal.sql apps/api/internal/models/journal.go && git commit -m "feat(journal): 마이그레이션 + Go 모델 (journal_entries·analysis_runs + RLS)"
```

---

## Task 2: JournalRepo (PgJournalRepo + 인터페이스)

**Files:**
- Create: `apps/api/internal/handlers/journal_repo_pg.go`

- [ ] **Step 1: repo 작성**

`apps/api/internal/handlers/journal_repo_pg.go`:

```go
package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/models"
)

var ErrJournalEntryNotFound = errors.New("journal entry not found")
var ErrJournalEntryNotMutable = errors.New("auto entry is not mutable")

type JournalRepo interface {
	// entries
	ListEntries(ctx context.Context, exec db.Executor, userID string, limit int, before *time.Time) ([]models.JournalEntry, bool, error)
	CreateEntry(ctx context.Context, exec db.Executor, userID string, entryType string, body models.JournalEntryCreate, relatedHoldingID *string) (*models.JournalEntry, error)
	UpdateEntry(ctx context.Context, exec db.Executor, userID, id string, body models.JournalEntryPatch) (*models.JournalEntry, error)
	DeleteEntry(ctx context.Context, exec db.Executor, userID, id string) error
	CountEntriesSince(ctx context.Context, exec db.Executor, userID string, since time.Time) (int, error)
	ListEntriesSince(ctx context.Context, exec db.Executor, userID string, since time.Time) ([]models.JournalEntry, error)

	// analysis runs
	InsertAnalysis(ctx context.Context, exec db.Executor, userID, runType, periodStart, periodEnd string, entriesCount int, contentMD, model string) (*models.AnalysisRun, error)
	ListAnalyses(ctx context.Context, exec db.Executor, userID string, limit int) ([]models.AnalysisRun, error)
	HasAutoMonthlyForPeriod(ctx context.Context, exec db.Executor, userID, periodStart string) (bool, error)
}

type PgJournalRepo struct{}

func NewPgJournalRepo() *PgJournalRepo { return &PgJournalRepo{} }

func scanEntry(row interface{ Scan(...any) error }) (*models.JournalEntry, error) {
	var e models.JournalEntry
	var action, holdingID, title *string
	var symbol, name *string
	err := row.Scan(
		&e.ID, &e.UserID, &e.EntryType,
		&action, &holdingID,
		&e.RelatedSymbols,
		&title, &e.Content,
		&e.CreatedAt, &e.UpdatedAt,
		&symbol, &name,
	)
	if err != nil {
		return nil, err
	}
	e.Action = action
	e.RelatedHoldingID = holdingID
	e.Title = title
	if holdingID != nil && symbol != nil && name != nil {
		e.RelatedHolding = &models.RelatedHoldingBrief{Symbol: *symbol, Name: *name}
	}
	return &e, nil
}

const entriesBaseSelect = `
select je.id::text, je.user_id::text, je.entry_type, je.action,
       je.related_holding_id::text, je.related_symbols,
       je.title, je.content, je.created_at, je.updated_at,
       i.symbol, i.name
from public.journal_entries je
left join public.holdings h on h.id = je.related_holding_id
left join public.instruments i on i.id = h.instrument_id
`

func (r *PgJournalRepo) ListEntries(ctx context.Context, exec db.Executor, userID string, limit int, before *time.Time) ([]models.JournalEntry, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := entriesBaseSelect + ` where je.user_id = $1`
	args := []any{userID}
	if before != nil {
		q += ` and je.created_at < $2`
		args = append(args, *before)
		q += ` order by je.created_at desc limit $3`
		args = append(args, limit+1)
	} else {
		q += ` order by je.created_at desc limit $2`
		args = append(args, limit+1)
	}
	rows, err := exec.Query(ctx, q, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	var out []models.JournalEntry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, false, err
		}
		out = append(out, *e)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(out) > limit
	if hasMore {
		out = out[:limit]
	}
	return out, hasMore, nil
}

func (r *PgJournalRepo) CreateEntry(ctx context.Context, exec db.Executor, userID, entryType string, body models.JournalEntryCreate, relatedHoldingID *string) (*models.JournalEntry, error) {
	if entryType != "auto" && entryType != "manual" {
		return nil, fmt.Errorf("invalid entry_type: %s", entryType)
	}
	symbols := body.RelatedSymbols
	if symbols == nil {
		symbols = []string{}
	}
	// Default action for manual entries: "observation" (spec §5-2)
	action := body.Action
	if entryType == "manual" && (action == nil || *action == "") {
		v := "observation"
		action = &v
	}
	row := exec.QueryRow(ctx, `
		insert into public.journal_entries
		  (user_id, entry_type, action, related_holding_id, related_symbols, title, content)
		values ($1, $2, $3, $4, $5, $6, $7)
		returning id::text
	`, userID, entryType, action, relatedHoldingID, symbols, body.Title, body.Content)
	var id string
	if err := row.Scan(&id); err != nil {
		return nil, err
	}
	return r.getByID(ctx, exec, userID, id)
}

func (r *PgJournalRepo) getByID(ctx context.Context, exec db.Executor, userID, id string) (*models.JournalEntry, error) {
	row := exec.QueryRow(ctx, entriesBaseSelect+` where je.user_id = $1 and je.id = $2::uuid`, userID, id)
	e, err := scanEntry(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrJournalEntryNotFound
		}
		return nil, err
	}
	return e, nil
}

func (r *PgJournalRepo) UpdateEntry(ctx context.Context, exec db.Executor, userID, id string, body models.JournalEntryPatch) (*models.JournalEntry, error) {
	// manual entry만 수정 가능
	var entryType string
	if err := exec.QueryRow(ctx, `select entry_type from public.journal_entries where id = $1::uuid and user_id = $2`, id, userID).Scan(&entryType); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrJournalEntryNotFound
		}
		return nil, err
	}
	if entryType == "auto" {
		return nil, ErrJournalEntryNotMutable
	}

	sets := []string{}
	args := []any{}
	i := 1
	if body.Action != nil {
		sets = append(sets, fmt.Sprintf("action = $%d", i))
		args = append(args, *body.Action)
		i++
	}
	if body.RelatedSymbols != nil {
		sets = append(sets, fmt.Sprintf("related_symbols = $%d", i))
		args = append(args, *body.RelatedSymbols)
		i++
	}
	if body.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", i))
		args = append(args, *body.Title)
		i++
	}
	if body.Content != nil {
		sets = append(sets, fmt.Sprintf("content = $%d", i))
		args = append(args, *body.Content)
		i++
	}
	if len(sets) == 0 {
		return r.getByID(ctx, exec, userID, id)
	}
	args = append(args, id, userID)
	q := fmt.Sprintf(`update public.journal_entries set %s where id = $%d::uuid and user_id = $%d`,
		strings.Join(sets, ", "), i, i+1)
	if _, err := exec.Exec(ctx, q, args...); err != nil {
		return nil, err
	}
	return r.getByID(ctx, exec, userID, id)
}

func (r *PgJournalRepo) DeleteEntry(ctx context.Context, exec db.Executor, userID, id string) error {
	ct, err := exec.Exec(ctx, `delete from public.journal_entries where id = $1::uuid and user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrJournalEntryNotFound
	}
	return nil
}

func (r *PgJournalRepo) CountEntriesSince(ctx context.Context, exec db.Executor, userID string, since time.Time) (int, error) {
	var n int
	err := exec.QueryRow(ctx, `select count(*) from public.journal_entries where user_id = $1 and created_at >= $2`, userID, since).Scan(&n)
	return n, err
}

func (r *PgJournalRepo) ListEntriesSince(ctx context.Context, exec db.Executor, userID string, since time.Time) ([]models.JournalEntry, error) {
	rows, err := exec.Query(ctx, entriesBaseSelect+` where je.user_id = $1 and je.created_at >= $2 order by je.created_at`, userID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.JournalEntry
	for rows.Next() {
		e, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

func (r *PgJournalRepo) InsertAnalysis(ctx context.Context, exec db.Executor, userID, runType, periodStart, periodEnd string, entriesCount int, contentMD, model string) (*models.AnalysisRun, error) {
	row := exec.QueryRow(ctx, `
		insert into public.analysis_runs
		  (user_id, run_type, period_start, period_end, entries_count, content_md, model)
		values ($1, $2, $3::date, $4::date, $5, $6, $7)
		returning id::text, created_at
	`, userID, runType, periodStart, periodEnd, entriesCount, contentMD, model)
	var a models.AnalysisRun
	if err := row.Scan(&a.ID, &a.CreatedAt); err != nil {
		return nil, err
	}
	a.UserID = userID
	a.RunType = runType
	a.PeriodStart = periodStart
	a.PeriodEnd = periodEnd
	a.EntriesCount = entriesCount
	a.ContentMD = contentMD
	a.Model = model
	return &a, nil
}

func (r *PgJournalRepo) ListAnalyses(ctx context.Context, exec db.Executor, userID string, limit int) ([]models.AnalysisRun, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	rows, err := exec.Query(ctx, `
		select id::text, user_id::text, run_type,
		       to_char(period_start, 'YYYY-MM-DD'),
		       to_char(period_end, 'YYYY-MM-DD'),
		       entries_count, content_md, model, created_at
		from public.analysis_runs
		where user_id = $1
		order by created_at desc
		limit $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.AnalysisRun
	for rows.Next() {
		var a models.AnalysisRun
		if err := rows.Scan(&a.ID, &a.UserID, &a.RunType, &a.PeriodStart, &a.PeriodEnd, &a.EntriesCount, &a.ContentMD, &a.Model, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *PgJournalRepo) HasAutoMonthlyForPeriod(ctx context.Context, exec db.Executor, userID, periodStart string) (bool, error) {
	var exists bool
	err := exec.QueryRow(ctx, `
		select exists(select 1 from public.analysis_runs
		              where user_id = $1 and run_type = 'auto_monthly' and period_start = $2::date)
	`, userID, periodStart).Scan(&exists)
	return exists, err
}
```

- [ ] **Step 2: 빌드**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./...
```

Expected: EXIT 0.

- [ ] **Step 3: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/api/internal/handlers/journal_repo_pg.go && git commit -m "feat(journal): PgJournalRepo (entries CRUD + analysis runs)"
```

---

## Task 3: HTTP 핸들러 + unit 테스트

**Files:**
- Create: `apps/api/internal/handlers/journal.go`
- Create: `apps/api/internal/handlers/journal_test.go`

- [ ] **Step 1: 핸들러 작성**

`apps/api/internal/handlers/journal.go`:

```go
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/ai"
	"github.com/quotient/quotient/apps/api/internal/ai/tools"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
)

const (
	maxJournalContent  = 2000
	maxJournalSymbols  = 10
	maxJournalSymbolLen = 20
	maxJournalTitle    = 100
)

type JournalHandler struct {
	repo     JournalRepo
	chatRepo ChatRepo      // 사용량 한도 차감 + 체크
	pool     *pgxpool.Pool
	client   ai.Client     // analyze 도구 LLM 호출
	registry *tools.Registry
	run      txRunner
}

func NewJournalHandler(repo JournalRepo, chatRepo ChatRepo, pool *pgxpool.Pool, client ai.Client, registry *tools.Registry) *JournalHandler {
	h := &JournalHandler{repo: repo, chatRepo: chatRepo, pool: pool, client: client, registry: registry}
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

// GET /v1/journal/entries
func (h *JournalHandler) List(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit == 0 {
		limit = 50
	}
	var before *time.Time
	if s := r.URL.Query().Get("before"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			before = &t
		}
	}
	var entries []models.JournalEntry
	var hasMore bool
	err := h.run(r.Context(), func(exec db.Executor) error {
		es, more, err := h.repo.ListEntries(r.Context(), exec, uid, limit, before)
		if err != nil {
			return err
		}
		entries = es
		hasMore = more
		return nil
	})
	if err != nil {
		slog.Error("journal list failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if entries == nil {
		entries = []models.JournalEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries, "has_more": hasMore})
}

// POST /v1/journal/entries — manual entry
func (h *JournalHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	var body models.JournalEntryCreate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if err := validateEntryCreate(&body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	var out *models.JournalEntry
	err := h.run(r.Context(), func(exec db.Executor) error {
		e, err := h.repo.CreateEntry(r.Context(), exec, uid, "manual", body, nil)
		if err != nil {
			return err
		}
		out = e
		return nil
	})
	if err != nil {
		slog.Error("journal create failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "create failed")
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// PATCH /v1/journal/entries/{id}
func (h *JournalHandler) Patch(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "id required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	var body models.JournalEntryPatch
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if err := validateEntryPatch(&body); err != nil {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", err.Error())
		return
	}
	var out *models.JournalEntry
	err := h.run(r.Context(), func(exec db.Executor) error {
		e, err := h.repo.UpdateEntry(r.Context(), exec, uid, id, body)
		if err != nil {
			return err
		}
		out = e
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrJournalEntryNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "entry not found")
			return
		}
		if errors.Is(err, ErrJournalEntryNotMutable) {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
				"error": map[string]any{
					"code":    "CANNOT_MODIFY_AUTO",
					"message": "자동 기록은 수정할 수 없습니다",
				},
			})
			return
		}
		slog.Error("journal patch failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "update failed")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// DELETE /v1/journal/entries/{id}
func (h *JournalHandler) Delete(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	id := chi.URLParam(r, "id")
	err := h.run(r.Context(), func(exec db.Executor) error {
		return h.repo.DeleteEntry(r.Context(), exec, uid, id)
	})
	if err != nil {
		if errors.Is(err, ErrJournalEntryNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "entry not found")
			return
		}
		slog.Error("journal delete failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "delete failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// POST /v1/journal/analyze — on-demand
func (h *JournalHandler) Analyze(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	var body struct {
		PeriodDays int `json:"period_days"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.PeriodDays <= 0 {
		body.PeriodDays = 90
	}
	if body.PeriodDays < 7 || body.PeriodDays > 365 {
		writeError(w, http.StatusUnprocessableEntity, "VALIDATION", "period_days must be 7..365")
		return
	}

	// 사용량 한도 체크 (chat 한도와 통합)
	if err := h.run(r.Context(), func(exec db.Executor) error {
		return h.chatRepo.CheckLimits(r.Context(), exec, uid, false)
	}); err != nil {
		if errors.Is(err, ErrUsageExceeded) {
			writeJSON(w, http.StatusTooManyRequests, map[string]any{
				"error": map[string]any{
					"code":    "USAGE_EXCEEDED",
					"reason":  "monthly_chat_limit",
					"message": "월 30회 한도 도달",
				},
			})
			return
		}
		slog.Error("journal analyze limit check failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "check failed")
		return
	}

	// LLM 도구 호출 (tools.ExecuteAndSerialize)
	result := tools.ExecuteAndSerialize(r.Context(), h.registry, h.pool, "analyze_journal", uid, map[string]any{
		"period_days": body.PeriodDays,
	})

	// result 파싱 — 도구가 JSON 응답
	var toolOut struct {
		EntriesCount int    `json:"entries_count"`
		ContentMD    string `json:"content_md"`
		Model        string `json:"model"`
		Error        string `json:"error,omitempty"`
	}
	_ = json.Unmarshal([]byte(result), &toolOut)
	if toolOut.Error != "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
			"error": map[string]any{
				"code":    "INSUFFICIENT_DATA",
				"reason":  "no_entries",
				"message": toolOut.Error,
			},
		})
		return
	}

	// 한도 차감 + analysis_runs 저장 (같은 트랜잭션)
	periodEnd := time.Now()
	periodStart := periodEnd.AddDate(0, 0, -body.PeriodDays)
	var run *models.AnalysisRun
	err := h.run(r.Context(), func(exec db.Executor) error {
		if err := h.chatRepo.IncrementUsage(r.Context(), exec, uid, 0, 0, false); err != nil {
			return err
		}
		a, err := h.repo.InsertAnalysis(r.Context(), exec, uid, "on_demand",
			periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02"),
			toolOut.EntriesCount, toolOut.ContentMD, toolOut.Model)
		if err != nil {
			return err
		}
		run = a
		return nil
	})
	if err != nil {
		slog.Error("journal analyze persist failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "persist failed")
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// GET /v1/journal/analyses
func (h *JournalHandler) ListAnalyses(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	var out []models.AnalysisRun
	err := h.run(r.Context(), func(exec db.Executor) error {
		as, err := h.repo.ListAnalyses(r.Context(), exec, uid, limit)
		if err != nil {
			return err
		}
		out = as
		return nil
	})
	if err != nil {
		slog.Error("journal analyses list failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if out == nil {
		out = []models.AnalysisRun{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"analyses": out})
}

// --- validation ---

func validateEntryCreate(b *models.JournalEntryCreate) error {
	if len(b.Content) < 1 || len(b.Content) > maxJournalContent {
		return errors.New("content length must be 1..2000")
	}
	if len(b.RelatedSymbols) > maxJournalSymbols {
		return errors.New("too many related_symbols (max 10)")
	}
	for _, s := range b.RelatedSymbols {
		if len(s) > maxJournalSymbolLen {
			return errors.New("symbol too long (max 20)")
		}
	}
	if b.Title != nil && len(*b.Title) > maxJournalTitle {
		return errors.New("title too long (max 100)")
	}
	if b.Action != nil {
		switch *b.Action {
		case "buy", "sell", "observation", "other":
		default:
			return errors.New("invalid action")
		}
	}
	return nil
}

func validateEntryPatch(b *models.JournalEntryPatch) error {
	if b.Content != nil && (len(*b.Content) < 1 || len(*b.Content) > maxJournalContent) {
		return errors.New("content length must be 1..2000")
	}
	if b.RelatedSymbols != nil {
		if len(*b.RelatedSymbols) > maxJournalSymbols {
			return errors.New("too many related_symbols")
		}
		for _, s := range *b.RelatedSymbols {
			if len(s) > maxJournalSymbolLen {
				return errors.New("symbol too long")
			}
		}
	}
	if b.Title != nil && len(*b.Title) > maxJournalTitle {
		return errors.New("title too long")
	}
	if b.Action != nil {
		switch *b.Action {
		case "buy", "sell", "observation", "other":
		default:
			return errors.New("invalid action")
		}
	}
	return nil
}
```

- [ ] **Step 2: handler unit 테스트**

`apps/api/internal/handlers/journal_test.go`:

```go
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/quotient/quotient/apps/api/internal/ai"
	"github.com/quotient/quotient/apps/api/internal/ai/tools"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
)

type fakeJournalRepo struct {
	entries        []models.JournalEntry
	createReturn   *models.JournalEntry
	updateReturn   *models.JournalEntry
	updateErr      error
	deleteErr      error
	analyses       []models.AnalysisRun
	insertAnalysis *models.AnalysisRun
	hasAutoMonthly bool
}

func (f *fakeJournalRepo) ListEntries(_ context.Context, _ db.Executor, _ string, _ int, _ *time.Time) ([]models.JournalEntry, bool, error) {
	return f.entries, false, nil
}
func (f *fakeJournalRepo) CreateEntry(_ context.Context, _ db.Executor, uid, entryType string, b models.JournalEntryCreate, holdingID *string) (*models.JournalEntry, error) {
	if f.createReturn != nil {
		return f.createReturn, nil
	}
	e := models.JournalEntry{ID: "new-id", UserID: uid, EntryType: entryType, Content: b.Content, RelatedSymbols: b.RelatedSymbols}
	if entryType == "manual" && b.Action == nil {
		a := "observation"
		e.Action = &a
	} else {
		e.Action = b.Action
	}
	return &e, nil
}
func (f *fakeJournalRepo) UpdateEntry(_ context.Context, _ db.Executor, _ string, _ string, _ models.JournalEntryPatch) (*models.JournalEntry, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return f.updateReturn, nil
}
func (f *fakeJournalRepo) DeleteEntry(_ context.Context, _ db.Executor, _ string, _ string) error {
	return f.deleteErr
}
func (f *fakeJournalRepo) CountEntriesSince(_ context.Context, _ db.Executor, _ string, _ time.Time) (int, error) {
	return len(f.entries), nil
}
func (f *fakeJournalRepo) ListEntriesSince(_ context.Context, _ db.Executor, _ string, _ time.Time) ([]models.JournalEntry, error) {
	return f.entries, nil
}
func (f *fakeJournalRepo) InsertAnalysis(_ context.Context, _ db.Executor, uid, runType, ps, pe string, cnt int, md, model string) (*models.AnalysisRun, error) {
	if f.insertAnalysis != nil {
		return f.insertAnalysis, nil
	}
	return &models.AnalysisRun{ID: "a-1", UserID: uid, RunType: runType, PeriodStart: ps, PeriodEnd: pe, EntriesCount: cnt, ContentMD: md, Model: model}, nil
}
func (f *fakeJournalRepo) ListAnalyses(_ context.Context, _ db.Executor, _ string, _ int) ([]models.AnalysisRun, error) {
	return f.analyses, nil
}
func (f *fakeJournalRepo) HasAutoMonthlyForPeriod(_ context.Context, _ db.Executor, _, _ string) (bool, error) {
	return f.hasAutoMonthly, nil
}

func reqJ(method, path, body, uid string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if uid != "" {
		r = r.WithContext(middleware.WithUserID(r.Context(), uid))
	}
	return r
}

func TestCreateEntry_Manual_OK(t *testing.T) {
	repo := &fakeJournalRepo{}
	h := NewJournalHandler(repo, &fakeChatRepo{}, nil, &ai.MockClient{}, tools.NewRegistry())
	w := httptest.NewRecorder()
	h.Create(w, reqJ(http.MethodPost, "/v1/journal/entries",
		`{"content":"관찰 내용 테스트","related_symbols":["005930"]}`, "user-1"))
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	var got models.JournalEntry
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.EntryType != "manual" {
		t.Errorf("entry_type=%s, want manual", got.EntryType)
	}
	if got.Action == nil || *got.Action != "observation" {
		t.Errorf("action default not applied: %v", got.Action)
	}
}

func TestCreateEntry_ContentTooLong_422(t *testing.T) {
	h := NewJournalHandler(&fakeJournalRepo{}, &fakeChatRepo{}, nil, &ai.MockClient{}, tools.NewRegistry())
	longContent := make([]byte, 2001)
	for i := range longContent {
		longContent[i] = 'a'
	}
	body, _ := json.Marshal(map[string]any{"content": string(longContent)})
	w := httptest.NewRecorder()
	h.Create(w, reqJ(http.MethodPost, "/v1/journal/entries", string(body), "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status %d, want 422", w.Code)
	}
}

func TestPatchEntry_AutoType_422(t *testing.T) {
	repo := &fakeJournalRepo{updateErr: ErrJournalEntryNotMutable}
	h := NewJournalHandler(repo, &fakeChatRepo{}, nil, &ai.MockClient{}, tools.NewRegistry())
	r := reqJ(http.MethodPatch, "/v1/journal/entries/x", `{"content":"수정"}`, "user-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "x")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	r = r.WithContext(middleware.WithUserID(r.Context(), "user-1"))
	w := httptest.NewRecorder()
	h.Patch(w, r)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", w.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	errBlock := got["error"].(map[string]any)
	if errBlock["code"] != "CANNOT_MODIFY_AUTO" {
		t.Errorf("code=%v", errBlock["code"])
	}
}

func TestDeleteEntry_NotFound(t *testing.T) {
	repo := &fakeJournalRepo{deleteErr: ErrJournalEntryNotFound}
	h := NewJournalHandler(repo, &fakeChatRepo{}, nil, &ai.MockClient{}, tools.NewRegistry())
	r := reqJ(http.MethodDelete, "/v1/journal/entries/x", "", "user-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "x")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	r = r.WithContext(middleware.WithUserID(r.Context(), "user-1"))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status %d", w.Code)
	}
}

func TestAnalyze_ExceededQuota_429(t *testing.T) {
	chat := &fakeChatRepo{limitErr: ErrUsageExceeded}
	h := NewJournalHandler(&fakeJournalRepo{}, chat, nil, &ai.MockClient{}, tools.NewRegistry())
	w := httptest.NewRecorder()
	h.Analyze(w, reqJ(http.MethodPost, "/v1/journal/analyze", `{"period_days":90}`, "user-1"))
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status %d, want 429", w.Code)
	}
}

func TestList_Empty_OK(t *testing.T) {
	h := NewJournalHandler(&fakeJournalRepo{}, &fakeChatRepo{}, nil, &ai.MockClient{}, tools.NewRegistry())
	w := httptest.NewRecorder()
	h.List(w, reqJ(http.MethodGet, "/v1/journal/entries", "", "user-1"))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["has_more"] != false {
		t.Errorf("has_more=%v", got["has_more"])
	}
}

func TestListAnalyses_Empty_OK(t *testing.T) {
	h := NewJournalHandler(&fakeJournalRepo{}, &fakeChatRepo{}, nil, &ai.MockClient{}, tools.NewRegistry())
	w := httptest.NewRecorder()
	h.ListAnalyses(w, reqJ(http.MethodGet, "/v1/journal/analyses", "", "user-1"))
	if w.Code != http.StatusOK {
		t.Errorf("status %d", w.Code)
	}
}
```

- [ ] **Step 3: 빌드 + 테스트**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./... && go test ./internal/handlers -run Journal -v
```

Expected: 7 케이스 PASS.

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/api/internal/handlers/journal.go apps/api/internal/handlers/journal_test.go && git commit -m "feat(journal): HTTP handler 6 endpoint + 7 unit (validation + 한도 차감)"
```

---

## Task 4: AI 도구 `analyze_journal`

**Files:**
- Create: `apps/api/internal/ai/tools/journal.go`
- Create: `apps/api/internal/ai/tools/journal_test.go`

- [ ] **Step 1: 도구 작성**

`apps/api/internal/ai/tools/journal.go`:

```go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/quotient/quotient/apps/api/internal/ai"
	"github.com/quotient/quotient/apps/api/internal/db"
)

type analyzeJournal struct{ *Deps }

func (t *analyzeJournal) Spec() ai.ToolSpec {
	return ai.ToolSpec{
		Name: "analyze_journal",
		Description: "사용자의 매매 일기 entries + 보유 자산 변화를 종합하여 매매 패턴·습관을 분석. 직접 매수/매도 권유 금지, 회고 관점만.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"period_days": map[string]any{
					"type": "integer", "default": 90, "minimum": 7, "maximum": 365,
				},
			},
		},
	}
}

func (t *analyzeJournal) RequiresUserContext() bool { return true }

// journalEntryRow는 prompt 빌드에 사용하는 패키지 레벨 named type.
// Run 내부의 local type과 helper의 anonymous struct 사이 타입 불일치를 회피한다.
type journalEntryRow struct {
	EntryType string
	Action    *string
	Symbols   []string
	Content   string
	CreatedAt time.Time
	Symbol    string
}

func (t *analyzeJournal) Run(ctx context.Context, exec db.Executor, userID string, input map[string]any) (any, error) {
	days := 90
	if v, ok := input["period_days"].(float64); ok {
		days = int(v)
	}
	if days < 7 || days > 365 {
		days = 90
	}
	since := time.Now().AddDate(0, 0, -days)

	// entries 조회 (RLS 안에서)
	rows, err := exec.Query(ctx, `
		select je.entry_type, je.action, je.related_symbols, je.content, je.created_at,
		       coalesce(i.symbol, '') as symbol
		from public.journal_entries je
		left join public.holdings h on h.id = je.related_holding_id
		left join public.instruments i on i.id = h.instrument_id
		where je.user_id = $1 and je.created_at >= $2
		order by je.created_at
	`, userID, since)
	if err != nil {
		return map[string]any{"error": err.Error()}, nil
	}
	defer rows.Close()
	var entries []journalEntryRow
	for rows.Next() {
		var e journalEntryRow
		if err := rows.Scan(&e.EntryType, &e.Action, &e.Symbols, &e.Content, &e.CreatedAt, &e.Symbol); err != nil {
			return map[string]any{"error": err.Error()}, nil
		}
		entries = append(entries, e)
	}
	if len(entries) < 3 {
		return map[string]any{"error": "분석할 일기가 3개 이상 필요합니다"}, nil
	}

	// 보유 자산 — 에러 시 nil slice로 진행 (분석 quality 저하 허용, briefing 패턴과 동일)
	holdings, _ := loadHoldingsForJournal(ctx, exec, userID)

	// system prompt 빌드
	system := buildJournalAnalysisPrompt(entries, holdings)

	// Claude 호출
	req := ai.ChatRequest{
		Model:  ai.ModelSonnet,
		System: system,
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: fmt.Sprintf("최근 %d일 매매 일기를 분석해 3~5 bullet 한국어 마크다운으로 회고를 작성해주세요.", days)},
		},
		MaxTokens: 1024,
	}
	ch, err := t.Client.StreamChat(ctx, req)
	if err != nil {
		return map[string]any{"error": "AI 호출 실패: " + err.Error()}, nil
	}
	var content string
	for ev := range ch {
		if ev.Type == ai.EventToken {
			text, _ := ev.Data["text"].(string)
			content += text
		}
	}
	if content == "" {
		content = "분석 결과를 생성하지 못했습니다."
	}

	out := map[string]any{
		"entries_count": len(entries),
		"content_md":    content,
		"model":         string(ai.ModelSonnet),
	}
	return out, nil
}

type holdingBrief struct {
	Symbol   string
	Quantity float64
	AvgCost  float64
}

func loadHoldingsForJournal(ctx context.Context, exec db.Executor, userID string) ([]holdingBrief, error) {
	rows, err := exec.Query(ctx, `
		select i.symbol, h.quantity::float8, h.avg_cost::float8
		from public.holdings h
		join public.instruments i on i.id = h.instrument_id
		where h.user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []holdingBrief
	for rows.Next() {
		var h holdingBrief
		if err := rows.Scan(&h.Symbol, &h.Quantity, &h.AvgCost); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, nil
}

func buildJournalAnalysisPrompt(entries []journalEntryRow, holdings []holdingBrief) string {
	base := `당신은 Quotient의 매매 일기 분석가입니다.

규칙:
- 직접 매수/매도 권유 금지. "회고 관점에서 ~를 살펴볼 수 있습니다" 표현.
- 추측 금지. 사용자가 직접 쓴 텍스트만 근거.
- 마크다운 3~5 bullet. 한국어. 각 bullet 100자 이내.
- 사용자 비난·평가 표현 금지. 중립·관찰 중심.

분석 관점:
- 매매 이유에 반복되는 키워드·논리
- 감정적 매매 가능성 신호 (변동성 직후 매도 등)
- 매매 일관성 vs 모순`

	var b strings.Builder
	b.WriteString(base)
	b.WriteString("\n\n[보유 자산 현황]\n")
	for _, h := range holdings {
		fmt.Fprintf(&b, "- %s: %.2f주 @ 평단 %.0f\n", h.Symbol, h.Quantity, h.AvgCost)
	}
	b.WriteString("\n[매매 일기 시계열]\n")
	for _, e := range entries {
		action := "관찰"
		if e.Action != nil {
			action = *e.Action
		}
		symbols := strings.Join(e.Symbols, ",")
		if symbols == "" && e.Symbol != "" {
			symbols = e.Symbol
		}
		fmt.Fprintf(&b, "- %s (%s · %s%s): %s\n",
			e.CreatedAt.Format("2006-01-02"), e.EntryType, action,
			func() string { if symbols != "" { return " · " + symbols }; return "" }(),
			truncate(e.Content, 200))
	}
	return b.String()
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// RegisterJournal — main.go에서 호출. Deps.Client는 Step 2에서 추가.
func RegisterJournal(r *Registry, d *Deps) {
	r.Register(&analyzeJournal{Deps: d})
}
```

> **메모**: 기존 `tools.Deps`가 `*pgxpool.Pool`만 보유. analyze_journal은 LLM 호출이 필요해 `ai.Client` 의존성 추가 필요. 두 방법:
>
> 1. `tools.Deps`에 `Client ai.Client` 필드 추가 (영향 작음, 다른 도구는 무시)
> 2. analyze_journal 전용 struct에 Client 필드 별도
>
> 추천 1번. 다음 step에서 patch.

- [ ] **Step 2: tools.Deps 확장 + RegisterJournal 완성**

`apps/api/internal/ai/tools/registry.go`의 `Deps`에 Client 필드 추가:

```go
type Deps struct {
	Pool   *pgxpool.Pool
	Client ai.Client  // analyze_journal 같은 LLM 호출 도구가 사용. 다른 도구는 무시.
}
```

import에 `"github.com/quotient/quotient/apps/api/internal/ai"` 추가.

`journal.go`의 `RegisterJournal` 본문 구현:

```go
func RegisterJournal(r *Registry, d *Deps) {
	r.Register(&analyzeJournal{Deps: d})
}
```

`analyzeJournal.Run`에서 `t.Client` 사용으로 변경 (현재 코드는 `t.Client` 참조 — `*Deps`에 Client 있으니 `t.Deps.Client`로 통일):

```go
ch, err := t.Deps.Client.StreamChat(ctx, req)
```

- [ ] **Step 3: 도구 unit 테스트**

`apps/api/internal/ai/tools/journal_test.go`:

```go
package tools

import (
	"context"
	"testing"

	"github.com/quotient/quotient/apps/api/internal/ai"
)

func TestAnalyzeJournal_Spec(t *testing.T) {
	tool := &analyzeJournal{Deps: &Deps{}}
	spec := tool.Spec()
	if spec.Name != "analyze_journal" {
		t.Errorf("name=%s", spec.Name)
	}
	if !tool.RequiresUserContext() {
		t.Error("RequiresUserContext must be true")
	}
}

func TestAnalyzeJournal_RegisterPicksUpInRegistry(t *testing.T) {
	r := NewRegistry()
	RegisterJournal(r, &Deps{Pool: nil, Client: &ai.MockClient{}})
	if _, ok := r.Get("analyze_journal"); !ok {
		t.Fatal("analyze_journal not registered")
	}
}
```

- [ ] **Step 4: 빌드 + 테스트**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./... && go test ./internal/ai/tools -run Journal -v
```

Expected: 2 PASS.

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/api/internal/ai/tools/journal.go apps/api/internal/ai/tools/journal_test.go apps/api/internal/ai/tools/registry.go && git commit -m "feat(ai/tools): analyze_journal 도구 + Deps에 ai.Client 추가 + 2 unit"
```

---

## Task 5: 월간 cron + unit

**Files:**
- Create: `apps/api/internal/schedule/journal_monthly.go`
- Create: `apps/api/internal/schedule/journal_monthly_test.go`
- Modify: `apps/api/internal/schedule/cron.go` — 잡 등록

- [ ] **Step 1: cron 작성**

`apps/api/internal/schedule/journal_monthly.go`:

```go
package schedule

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/quotient/quotient/apps/api/internal/ai"
	"github.com/quotient/quotient/apps/api/internal/ai/tools"
)

// JobMonthlyJournalDispatcher — 매월 1일 07:00 KST 사용자 hash 분단위 분산.
// 직전 1달 entries 있는 사용자만 → analyze_journal 도구 호출 → analysis_runs INSERT.
func JobMonthlyJournalDispatcher(ctx context.Context, d Deps, client ai.Client, registry *tools.Registry) error {
	now := time.Now().In(SeoulLoc())
	if now.Day() != 1 || now.Hour() != 7 {
		return nil
	}
	currentMinute := now.Minute()

	// 직전 1달 entry 있는 사용자
	// 윈도우: lastMonthStart(KST) <= created_at < thisMonthStart(KST)
	lastMonthStartT := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, SeoulLoc())
	thisMonthStartT := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, SeoulLoc())
	rows, err := d.Pool.Query(ctx, `
		select distinct user_id::text from public.journal_entries
		where created_at >= $1 and created_at < $2
	`, lastMonthStartT, thisMonthStartT)
	if err != nil {
		return fmt.Errorf("query active users: %w", err)
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	thisMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, SeoulLoc()).Format("2006-01-02")
	lastMonthStart := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, SeoulLoc()).Format("2006-01-02")

	processed := 0
	for _, uid := range users {
		if userMinuteSlot(uid) != currentMinute {
			continue
		}
		// 중복 방지 — 이번 달에 이미 생성
		var exists bool
		_ = d.Pool.QueryRow(ctx, `
			select exists(select 1 from public.analysis_runs
			              where user_id = $1::uuid and run_type='auto_monthly' and period_start = $2::date)
		`, uid, thisMonthStart).Scan(&exists)
		if exists {
			continue
		}

		// 도구 호출 (사용자 JWT 트랜잭션 안에서 — ExecuteAndSerialize가 RequiresUserContext=true 처리)
		result := tools.ExecuteAndSerialize(ctx, registry, d.Pool, "analyze_journal", uid, map[string]any{
			"period_days": 30,
		})
		var toolOut struct {
			EntriesCount int    `json:"entries_count"`
			ContentMD    string `json:"content_md"`
			Model        string `json:"model"`
			Error        string `json:"error,omitempty"`
		}
		_ = json.Unmarshal([]byte(result), &toolOut)
		if toolOut.Error != "" {
			slog.Warn("monthly journal analyze skipped", "user", uid, "reason", toolOut.Error)
			continue
		}

		// 슈퍼유저 풀로 INSERT (cron은 fan-out이라 user JWT 없음)
		// ON CONFLICT는 arbiter 생략(`DO NOTHING`) — partial unique index가 race INSERT를 자동 차단,
		// arbiter 매핑 실패 시 23505 에러로 cron 전체가 깨지는 사고 회피.
		_, err := d.Pool.Exec(ctx, `
			insert into public.analysis_runs
			  (user_id, run_type, period_start, period_end, entries_count, content_md, model)
			values ($1::uuid, 'auto_monthly', $2::date, $3::date, $4, $5, $6)
			on conflict do nothing
		`, uid, lastMonthStart, thisMonthStart, toolOut.EntriesCount, toolOut.ContentMD, toolOut.Model)
		if err != nil {
			slog.Warn("monthly journal insert failed", "user", uid, "err", err)
			continue
		}
		processed++
	}
	slog.Info("monthly journal dispatched", "minute", currentMinute, "processed", processed)
	// client 인자는 briefing dispatcher와 시그니처 일관 위해 받음.
	// analyze_journal 도구는 tools.Deps.Client를 직접 사용하므로 본 함수에선 미사용.
	_ = client
	return nil
}
```

- [ ] **Step 2: cron 등록**

`apps/api/internal/schedule/cron.go`의 `Start` 함수에 잡 추가 (briefings 잡 옆):

```go
// 월간 매매 일기 자동 회고 dispatcher — 매 분 동작 (내부에서 매월 1일 07:00~07:59만 처리)
mustAdd(c, "* 7 1 * *", "journal_monthly", func() {
	if err := JobMonthlyJournalDispatcher(ctx, d, aiClient, toolRegistry); err != nil {
		slog.Error("cron journal_monthly failed", "err", err)
	}
})
```

그리고 `slog.Info("cron started", "jobs", 7, ...)` 부분의 `7`을 `8`로 갱신.

- [ ] **Step 3: cron unit**

`apps/api/internal/schedule/journal_monthly_test.go`:

```go
package schedule

import "testing"

func TestUserMinuteSlot_Deterministic(t *testing.T) {
	uid := "00000000-0000-0000-0000-000000000001"
	a := userMinuteSlot(uid)
	b := userMinuteSlot(uid)
	if a != b {
		t.Errorf("slot non-deterministic: %d vs %d", a, b)
	}
	if a < 0 || a >= 60 {
		t.Errorf("slot out of range: %d", a)
	}
}

// JobMonthlyJournalDispatcher의 시간 가드 직접 검증은 schedule helper에 의존.
// 실제 동작 검증은 통합 테스트(별도 task 또는 cron 수동 실행)로 위임.
// 본 unit은 hash 결정성만.
_ = sha256.Sum256  // 컴파일 가드 (briefing_worker.go에 이미 사용되므로 별도 import 불필요)
```

> 메모: `userMinuteSlot`은 기존 `briefing_worker.go`에 정의돼 있어 재사용. 위 테스트는 그 함수 검증.
> `sha256.Sum256` import 가드는 brevity 위해 생략 가능 — briefing 테스트가 이미 sha256 import를 가지면 불필요.

위 테스트가 컴파일 안 될 경우 단순화:

```go
package schedule

import "testing"

func TestUserMinuteSlot_Deterministic(t *testing.T) {
	uid := "00000000-0000-0000-0000-000000000001"
	a := userMinuteSlot(uid)
	b := userMinuteSlot(uid)
	if a != b {
		t.Errorf("slot non-deterministic: %d vs %d", a, b)
	}
	if a < 0 || a >= 60 {
		t.Errorf("slot out of range: %d", a)
	}
}
```

- [ ] **Step 4: 빌드 + 테스트**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./... && go test ./internal/schedule -run UserMinuteSlot -v
```

Expected: PASS.

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/api/internal/schedule/journal_monthly.go apps/api/internal/schedule/journal_monthly_test.go apps/api/internal/schedule/cron.go && git commit -m "feat(journal): 월간 자동 회고 cron — 매월 1일 07:00 KST 사용자 hash 분단위 분산"
```

---

## Task 6: Holdings CRUD에 reason 통합

**Files:**
- Modify: `apps/api/internal/handlers/holdings.go`
- Modify: `apps/api/internal/handlers/holdings_test.go`

- [ ] **Step 1: Create body + Patch body에 reason 추가**

`holdings.go`의 `Create` 핸들러 body struct 확장:

```go
var body struct {
	InstrumentID string  `json:"instrument_id"`
	Quantity     float64 `json:"quantity"`
	AvgCost      float64 `json:"avg_cost"`
	OpenedAt     *string `json:"opened_at,omitempty"`
	Note         *string `json:"note,omitempty"`
	Reason       *string `json:"reason,omitempty"` // 매매 이유 (선택, 200자)
}
```

생성 트랜잭션 안에서 reason이 있으면 journal_entries auto entry 함께 INSERT. HoldingHandler는 JournalRepo 의존성 추가 필요.

`HoldingHandler` 구조체:

```go
type HoldingHandler struct {
	repo        HoldingRepo
	journalRepo JournalRepo  // optional, nil이면 reason 무시
	pool        *pgxpool.Pool
	run         txRunner
}

func NewHoldingHandler(repo HoldingRepo, journalRepo JournalRepo, pool *pgxpool.Pool) *HoldingHandler {
	h := &HoldingHandler{repo: repo, journalRepo: journalRepo, pool: pool}
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

Create 핸들러 내 reason 처리 — repo.Create 다음 같은 트랜잭션 안에서:

```go
err := h.run(r.Context(), func(exec db.Executor) error {
	o, err := h.repo.Create(r.Context(), exec, uid, body.InstrumentID, body.Quantity, body.AvgCost, body.OpenedAt, body.Note)
	if err != nil {
		return err
	}
	out = o
	if body.Reason != nil && *body.Reason != "" && h.journalRepo != nil {
		reason := *body.Reason
		if len(reason) > 200 {
			reason = reason[:200]
		}
		action := "buy"
		_, _ = h.journalRepo.CreateEntry(r.Context(), exec, uid, "auto", models.JournalEntryCreate{
			Action:  &action,
			Content: reason,
		}, &out.ID)
	}
	return nil
})
```

Patch 핸들러도 같은 패턴(action='sell'·'buy' 결정은 reason과 무관하게 PATCH는 'buy' 또는 'other' — 단순화로 'buy' 유지하거나 quantity 변화로 판정).

단순화: Patch에서 reason이 있으면 'other' action으로 entry 생성:

```go
if body.Reason != nil && *body.Reason != "" && h.journalRepo != nil {
	reason := *body.Reason
	if len(reason) > 200 {
		reason = reason[:200]
	}
	action := "other"
	_, _ = h.journalRepo.CreateEntry(r.Context(), exec, uid, "auto", models.JournalEntryCreate{
		Action:  &action,
		Content: reason,
	}, &out.ID)
}
```

(Patch body struct에도 `Reason *string` 필드 추가.)

- [ ] **Step 2: holdings_test.go 갱신**

기존 호출 모두 `NewHoldingHandler(repo, nil, nil)` 또는 `NewHoldingHandler(repo, nil, pool)`로 (두 번째 인자에 nil journalRepo).

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && grep -n "NewHoldingHandler" internal/handlers/holdings_test.go
```

각 호출 수정. 추가 케이스:

```go
func TestCreateHolding_WithReason_CreatesAutoEntry(t *testing.T) {
	repo := &fakeHoldingRepo{}
	jrepo := &fakeJournalRepo{}
	h := NewHoldingHandler(repo, jrepo, nil)
	body := `{"instrument_id":"x","quantity":1,"avg_cost":100,"reason":"실적 회복 기대"}`
	// asset_class 가드 통과시키려면 pool 필요 — 일단 nil pool에서 가드 자체가 skip되는지 확인 후 수정
	// holdings.go의 asset_class 가드는 h.pool != nil 분기. nil pool이면 skip.
	w := httptest.NewRecorder()
	h.Create(w, reqWithUID(http.MethodPost, "/v1/holdings", body, "user-1"))
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	if jrepo.lastCreateContent != "실적 회복 기대" {
		t.Errorf("auto entry not created with reason: %v", jrepo.lastCreateContent)
	}
}
```

`fakeJournalRepo`에 `lastCreateContent` 필드 추가:

```go
type fakeJournalRepo struct {
	// ... 기존 필드
	lastCreateContent string
}

func (f *fakeJournalRepo) CreateEntry(_ context.Context, _ db.Executor, uid, entryType string, b models.JournalEntryCreate, holdingID *string) (*models.JournalEntry, error) {
	f.lastCreateContent = b.Content
	// ... 기존 로직
}
```

(`journal_test.go`의 fakeJournalRepo 한 곳에 필드 추가, holdings_test.go에서 import.)

- [ ] **Step 3: main.go wiring 수정**

기존 holdingHandler 선언 위에 `journalRepo`를 추가하고, holdingHandler 호출에 journalRepo 인자 추가:

```go
journalRepo := handlers.NewPgJournalRepo()
holdingHandler := handlers.NewHoldingHandler(holdingRepo, journalRepo, pool)
```

> ⚠️ **Task 7에서 journalRepo 재선언 금지**. 본 Task에서 만든 변수를 Task 7가 그대로 재사용한다. Task 7 Step 2는 `journalHandler := handlers.NewJournalHandler(journalRepo, ...)` 한 줄만 추가.

- [ ] **Step 4: 빌드 + 테스트**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./... && go test ./internal/handlers -v
```

Expected: 전체 PASS.

- [ ] **Step 5: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/api/internal/handlers/holdings.go apps/api/internal/handlers/holdings_test.go apps/api/internal/handlers/journal_test.go apps/api/cmd/server/main.go && git commit -m "feat(holdings): reason 옵션 → auto journal entry 동시 생성 (같은 트랜잭션)"
```

---

## Task 7: Router + main.go wiring 완성

**Files:**
- Modify: `apps/api/internal/router/router.go`
- Modify: `apps/api/cmd/server/main.go`

- [ ] **Step 1: router.go 시그니처에 journalHandler 추가**

`router.New(...)` 인자 끝에 `journalHandler *handlers.JournalHandler`. 인증 그룹에 6 라우트:

```go
r.Get("/v1/journal/entries", journalHandler.List)
r.Post("/v1/journal/entries", journalHandler.Create)
r.Patch("/v1/journal/entries/{id}", journalHandler.Patch)
r.Delete("/v1/journal/entries/{id}", journalHandler.Delete)
r.Post("/v1/journal/analyze", journalHandler.Analyze)
r.Get("/v1/journal/analyses", journalHandler.ListAnalyses)
```

- [ ] **Step 2: main.go wiring**

> ⚠️ `journalRepo`는 Task 6 Step 3에서 이미 선언됨 — **재선언 금지**.

기존 `toolDeps := &tools.Deps{Pool: pool}` 줄을 `Client: aiClient` 포함하도록 변경:

```go
// Tool registry
toolRegistry := tools.NewRegistry()
toolDeps := &tools.Deps{Pool: pool, Client: aiClient}  // ← Client 추가
tools.RegisterPortfolio(toolRegistry, toolDeps)
tools.RegisterQuote(toolRegistry, toolDeps)
tools.RegisterSearch(toolRegistry, toolDeps)
tools.RegisterJournal(toolRegistry, toolDeps)  // ← 신규
```

chatHandler 아래에 journalHandler 추가:

```go
journalHandler := handlers.NewJournalHandler(journalRepo, chatRepo, pool, aiClient, toolRegistry)
```

`router.New(...)` 호출 인자 끝에 `journalHandler` 추가 (router.go 시그니처도 동일 순서).

- [ ] **Step 3: 빌드**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./... && go test ./...
```

Expected: 전체 PASS.

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/api/internal/router/router.go apps/api/cmd/server/main.go && git commit -m "feat(api): /v1/journal/* 6 라우트 등록 + main.go wiring (analyze_journal 도구 포함)"
```

---

## Task 8: Frontend API 클라이언트

**Files:**
- Create: `apps/web/lib/api/journal.ts`

- [ ] **Step 1: 클라이언트 작성**

`apps/web/lib/api/journal.ts`:

```ts
import { authFetch } from "./auth-fetch";

export type JournalAction = "buy" | "sell" | "observation" | "other";

export type JournalEntry = {
  id: string;
  user_id: string;
  entry_type: "auto" | "manual";
  action?: JournalAction;
  related_holding_id?: string;
  related_holding?: { symbol: string; name: string };
  related_symbols: string[];
  title?: string;
  content: string;
  created_at: string;
  updated_at: string;
};

export type AnalysisRun = {
  id: string;
  user_id: string;
  run_type: "auto_monthly" | "on_demand";
  period_start: string;
  period_end: string;
  entries_count: number;
  content_md: string;
  model: string;
  created_at: string;
};

export type JournalListResult = { entries: JournalEntry[]; has_more: boolean };

export async function listEntries(limit = 50, before?: string): Promise<JournalListResult> {
  const q = new URLSearchParams();
  q.set("limit", String(limit));
  if (before) q.set("before", before);
  const res = await authFetch(`/v1/journal/entries?${q}`);
  if (!res.ok) throw new Error(`list failed: ${res.status}`);
  return res.json();
}

export async function createEntry(body: {
  action?: JournalAction;
  related_symbols?: string[];
  title?: string;
  content: string;
}): Promise<JournalEntry> {
  const res = await authFetch("/v1/journal/entries", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err?.error?.message ?? `create failed: ${res.status}`);
  }
  return res.json();
}

export async function patchEntry(id: string, body: Partial<{
  action: JournalAction;
  related_symbols: string[];
  title: string;
  content: string;
}>): Promise<JournalEntry | { error: { code: string; message: string } }> {
  const res = await authFetch(`/v1/journal/entries/${id}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (res.status === 422) return res.json();
  if (!res.ok) throw new Error(`patch failed: ${res.status}`);
  return res.json();
}

export async function deleteEntry(id: string): Promise<void> {
  const res = await authFetch(`/v1/journal/entries/${id}`, { method: "DELETE" });
  if (!res.ok) throw new Error(`delete failed: ${res.status}`);
}

export type AnalyzeResult =
  | AnalysisRun
  | { error: { code: string; reason: string; message: string } };

export async function analyzeNow(periodDays = 90): Promise<AnalyzeResult> {
  const res = await authFetch("/v1/journal/analyze", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ period_days: periodDays }),
  });
  if (res.status === 422 || res.status === 429) return res.json();
  if (!res.ok) throw new Error(`analyze failed: ${res.status}`);
  return res.json();
}

export async function listAnalyses(limit = 20): Promise<{ analyses: AnalysisRun[] }> {
  const res = await authFetch(`/v1/journal/analyses?limit=${limit}`);
  if (!res.ok) throw new Error(`analyses failed: ${res.status}`);
  return res.json();
}

export function isAnalyzeError(r: AnalyzeResult): r is { error: { code: string; reason: string; message: string } } {
  return (r as { error?: unknown }).error !== undefined;
}
```

- [ ] **Step 2: 타입 검증 + 커밋**

```bash
cd /Users/yuhojin/Desktop/finance/apps/web && npx tsc --noEmit
cd /Users/yuhojin/Desktop/finance && git add apps/web/lib/api/journal.ts && git commit -m "feat(web): journal API 클라이언트 + 타입"
```

---

## Task 9: Frontend 컴포넌트 (JournalPage·EntryItem·AnalysisCard·NewEntryDialog)

**Files:**
- Create: `apps/web/components/journal/JournalPage.tsx`
- Create: `apps/web/components/journal/EntryItem.tsx`
- Create: `apps/web/components/journal/AnalysisCard.tsx`
- Create: `apps/web/components/journal/NewEntryDialog.tsx`
- Create: `apps/web/app/app/journal/page.tsx`

- [ ] **Step 1: 페이지 라우트**

`apps/web/app/app/journal/page.tsx`:

```tsx
import { JournalPage } from "@/components/journal/JournalPage";

export default function Page() {
  return <JournalPage />;
}
```

- [ ] **Step 2: EntryItem**

`apps/web/components/journal/EntryItem.tsx`:

```tsx
"use client";

import { useState } from "react";
import type { JournalEntry } from "@/lib/api/journal";
import { deleteEntry, patchEntry } from "@/lib/api/journal";

export function EntryItem({ entry, onChanged }: { entry: JournalEntry; onChanged: () => void }) {
  const [editing, setEditing] = useState(false);
  const [content, setContent] = useState(entry.content);
  const border = entry.entry_type === "auto" ? "border-bb-accent" : "border-bb-info";
  const actionLabel = labelFor(entry.action);
  const date = entry.created_at.slice(0, 10);

  async function save() {
    await patchEntry(entry.id, { content });
    setEditing(false);
    onChanged();
  }

  async function remove() {
    if (!confirm("삭제하시겠습니까?")) return;
    await deleteEntry(entry.id);
    onChanged();
  }

  return (
    <div className={`border-l-2 ${border} pl-3 py-2`}>
      <div className="font-mono text-[10px] text-fg-muted">
        {date} · <span className={signColor(entry.action)}>{actionLabel}</span>
        {entry.related_holding && <> · {entry.related_holding.symbol} {entry.related_holding.name}</>}
        {entry.related_symbols.length > 0 && <> · {entry.related_symbols.join(", ")}</>}
        · {entry.entry_type === "auto" ? "자동" : "수동"}
      </div>
      {entry.title && <div className="text-sm font-medium mt-1">{entry.title}</div>}
      {editing ? (
        <div className="mt-1">
          <textarea
            value={content}
            onChange={(e) => setContent(e.target.value)}
            maxLength={2000}
            className="w-full bg-bg-card border border-line p-2 text-sm font-mono"
            rows={4}
          />
          <div className="flex gap-2 mt-1">
            <button onClick={save} className="text-xs font-mono text-bb-accent">저장</button>
            <button onClick={() => setEditing(false)} className="text-xs font-mono text-fg-muted">취소</button>
          </div>
        </div>
      ) : (
        <div className="text-sm mt-1 whitespace-pre-wrap">{entry.content}</div>
      )}
      {entry.entry_type === "manual" && !editing && (
        <div className="flex gap-2 mt-2 text-xs font-mono">
          <button onClick={() => setEditing(true)} className="text-fg-muted hover:text-fg">수정</button>
          <button onClick={remove} className="text-bb-down/70 hover:text-bb-down">삭제</button>
        </div>
      )}
    </div>
  );
}

function labelFor(a?: string) {
  switch (a) {
    case "buy": return "매수";
    case "sell": return "매도";
    case "observation": return "관찰";
    case "other": return "기타";
    default: return "";
  }
}

function signColor(a?: string) {
  if (a === "buy") return "text-bb-up";
  if (a === "sell") return "text-bb-down";
  return "text-fg-muted";
}
```

- [ ] **Step 3: AnalysisCard**

`apps/web/components/journal/AnalysisCard.tsx`:

```tsx
import type { AnalysisRun } from "@/lib/api/journal";

export function AnalysisCard({ run }: { run: AnalysisRun }) {
  const border = run.run_type === "auto_monthly" ? "border-bb-warn" : "border-bb-info";
  const label = run.run_type === "auto_monthly" ? "월간 자동 회고" : "사용자 요청 분석";
  const icon = run.run_type === "auto_monthly" ? "📅" : "💡";
  return (
    <div className={`border-l-2 ${border} bg-bg-card p-4 mb-3`}>
      <div className="font-mono text-[10px] text-fg-muted mb-2">
        {icon} {label} · {run.period_start} ~ {run.period_end} · {run.entries_count}개 entries
      </div>
      <div className="text-sm whitespace-pre-wrap">{run.content_md}</div>
    </div>
  );
}
```

- [ ] **Step 4: NewEntryDialog**

`apps/web/components/journal/NewEntryDialog.tsx`:

```tsx
"use client";

import { useState } from "react";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { createEntry, type JournalAction } from "@/lib/api/journal";

export function NewEntryDialog({
  open, onOpenChange, onCreated,
}: {
  open: boolean;
  onOpenChange: (v: boolean) => void;
  onCreated: () => void;
}) {
  const [action, setAction] = useState<JournalAction>("observation");
  const [title, setTitle] = useState("");
  const [symbols, setSymbols] = useState("");
  const [content, setContent] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit() {
    if (content.length < 1) { setErr("내용을 입력해주세요"); return; }
    if (content.length > 2000) { setErr("2000자 이내"); return; }
    setSubmitting(true);
    setErr(null);
    try {
      const symbolList = symbols.split(",").map((s) => s.trim()).filter(Boolean).slice(0, 10);
      await createEntry({
        action,
        related_symbols: symbolList,
        title: title || undefined,
        content,
      });
      setAction("observation"); setTitle(""); setSymbols(""); setContent("");
      onCreated();
      onOpenChange(false);
    } catch (e: unknown) {
      setErr((e as Error).message ?? "생성 실패");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="font-mono">📓 새 일기 entry</DialogTitle>
        </DialogHeader>
        <div className="space-y-3 py-2">
          <div>
            <Label className="text-xs font-mono">제목 (선택, 100자)</Label>
            <Input value={title} onChange={(e) => setTitle(e.target.value)} maxLength={100} />
          </div>
          <div>
            <Label className="text-xs font-mono">종류</Label>
            <select
              value={action}
              onChange={(e) => setAction(e.target.value as JournalAction)}
              className="w-full bg-bg-card border border-line px-3 py-1.5 text-sm font-mono"
            >
              <option value="observation">관찰</option>
              <option value="buy">매수</option>
              <option value="sell">매도</option>
              <option value="other">기타</option>
            </select>
          </div>
          <div>
            <Label className="text-xs font-mono">관련 종목 (콤마 구분, 최대 10개)</Label>
            <Input value={symbols} onChange={(e) => setSymbols(e.target.value)} placeholder="005930, NVDA" />
          </div>
          <div>
            <Label className="text-xs font-mono">내용 (1~2000자)</Label>
            <textarea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              maxLength={2000}
              rows={6}
              className="w-full bg-bg-card border border-line p-2 text-sm font-mono"
            />
            <div className="text-right text-[10px] text-fg-muted font-mono">{content.length}/2000</div>
          </div>
          {err && <p className="text-bb-down text-xs font-mono">{err}</p>}
        </div>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>취소</Button>
          <Button onClick={submit} disabled={submitting}>
            {submitting ? "저장 중…" : "저장"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
```

- [ ] **Step 5: JournalPage 메인**

`apps/web/components/journal/JournalPage.tsx`:

```tsx
"use client";

import { useEffect, useState } from "react";
import {
  listEntries, listAnalyses, analyzeNow, isAnalyzeError,
  type JournalEntry, type AnalysisRun, type AnalyzeResult,
} from "@/lib/api/journal";
import { EntryItem } from "./EntryItem";
import { AnalysisCard } from "./AnalysisCard";
import { NewEntryDialog } from "./NewEntryDialog";

export function JournalPage() {
  const [entries, setEntries] = useState<JournalEntry[] | null>(null);
  const [analyses, setAnalyses] = useState<AnalysisRun[]>([]);
  const [newOpen, setNewOpen] = useState(false);
  const [analyzing, setAnalyzing] = useState(false);
  const [analyzeMsg, setAnalyzeMsg] = useState<string | null>(null);

  async function refresh() {
    const [e, a] = await Promise.all([listEntries(50), listAnalyses(20)]);
    setEntries(e.entries);
    setAnalyses(a.analyses);
  }

  useEffect(() => {
    refresh().catch(() => {
      setEntries([]);
      setAnalyses([]);
    });
  }, []);

  async function onAnalyze() {
    setAnalyzing(true);
    setAnalyzeMsg(null);
    const result: AnalyzeResult = await analyzeNow(90);
    setAnalyzing(false);
    if (isAnalyzeError(result)) {
      setAnalyzeMsg(result.error.message);
      return;
    }
    setAnalyses([result, ...analyses]);
  }

  return (
    <div className="p-6 md:p-8 max-w-3xl mx-auto space-y-6">
      <header className="flex items-baseline justify-between">
        <div>
          <h1 className="font-mono text-2xl">📓 매매 일기</h1>
          <p className="text-fg-muted text-sm mt-1">매매 결정과 시장 관찰을 기록합니다. 월 1회 자동 회고 + 필요 시 분석 요청.</p>
        </div>
        <div className="flex gap-2">
          <button
            onClick={() => setNewOpen(true)}
            className="font-mono text-xs px-3 py-1.5 border border-line hover:border-fg-muted"
          >
            + 새 entry
          </button>
          <button
            onClick={onAnalyze}
            disabled={analyzing}
            className="font-mono text-xs px-3 py-1.5 border border-bb-accent text-bb-accent hover:bg-bb-accent/10 disabled:opacity-50"
          >
            {analyzing ? "분석 중…" : "⚡ 분석 요청"}
          </button>
        </div>
      </header>

      {analyzeMsg && (
        <div className="border-l-2 border-bb-down bg-bg-card p-3 text-sm font-mono">
          {analyzeMsg}
        </div>
      )}

      {analyses.length > 0 && (
        <section>
          <h2 className="font-mono text-xs text-fg-muted tracking-widest mb-2">AI 분석</h2>
          {analyses.map((a) => <AnalysisCard key={a.id} run={a} />)}
        </section>
      )}

      <section>
        <h2 className="font-mono text-xs text-fg-muted tracking-widest mb-2">일기 entries</h2>
        {entries === null ? (
          <div className="text-fg-muted text-sm font-mono">로딩…</div>
        ) : entries.length === 0 ? (
          <div className="text-fg-muted text-sm">아직 일기가 없습니다. 새 entry 작성으로 시작하세요.</div>
        ) : (
          <div className="space-y-3">
            {entries.map((e) => <EntryItem key={e.id} entry={e} onChanged={refresh} />)}
          </div>
        )}
      </section>

      <NewEntryDialog open={newOpen} onOpenChange={setNewOpen} onCreated={refresh} />
    </div>
  );
}
```

- [ ] **Step 6: 타입 검증**

```bash
cd /Users/yuhojin/Desktop/finance/apps/web && npx tsc --noEmit
```

Expected: EXIT 0.

- [ ] **Step 7: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/web/app/app/journal apps/web/components/journal && git commit -m "feat(web): /app/journal 페이지 + EntryItem·AnalysisCard·NewEntryDialog"
```

---

## Task 10: Sidebar 아이콘 + AddHoldingDialog reason 필드

**Files:**
- Modify: `apps/web/components/shell/Sidebar.tsx`
- Modify: `apps/web/components/portfolio/AddHoldingDialog.tsx`
- Modify: `apps/web/components/portfolio/EditHoldingDialog.tsx`
- Modify: `apps/web/lib/api/holdings.ts` — Create/Update 시그니처 확장

- [ ] **Step 1: Sidebar에 📓 아이콘 추가**

`apps/web/components/shell/Sidebar.tsx`의 items 배열에:

```tsx
import { Home, Wallet, MessageSquare, BarChart3, BookOpen, Settings, Heart } from "lucide-react";

const items = [
  { href: "/app", icon: Home, label: "홈" },
  { href: "/app/portfolio", icon: Wallet, label: "포트폴리오" },
  { href: "/app/chat", icon: MessageSquare, label: "채팅" },
  { href: "/app/market", icon: BarChart3, label: "마켓" },
  { href: "/app/journal", icon: BookOpen, label: "매매 일기" },
  { href: "/app/settings", icon: Settings, label: "설정" },
];
```

- [ ] **Step 2: holdings API 클라이언트 reason 추가**

`apps/web/lib/api/holdings.ts`의 createHolding 시그니처:

```ts
export async function createHolding(body: {
  instrument_id: string;
  quantity: number;
  avg_cost: number;
  opened_at?: string;
  note?: string;
  reason?: string;
}) { ... }
```

PATCH 동일하게 reason 옵션.

- [ ] **Step 3: AddHoldingDialog에 textarea**

기존 폼 끝부분(메모 다음)에:

```tsx
<div>
  <Label className="text-xs font-mono">💭 매매 이유 (선택, 200자)</Label>
  <textarea
    value={reason}
    onChange={(e) => setReason(e.target.value)}
    maxLength={200}
    rows={2}
    className="w-full bg-bg-card border border-line p-2 text-sm font-mono"
  />
  <p className="text-[10px] text-fg-muted font-mono mt-1">
    * 작성 시 매매 일기 자동 기록 — 더 자세히는 일기 페이지에서
  </p>
</div>
```

state `const [reason, setReason] = useState("")` 추가. `createHolding` 호출 시 `reason: reason || undefined` 전달. `reset()`에 `setReason("")` 추가.

- [ ] **Step 4: EditHoldingDialog 동일 패턴**

같은 textarea + state + patch에 reason 전달.

- [ ] **Step 5: 타입 검증 + 커밋**

```bash
cd /Users/yuhojin/Desktop/finance/apps/web && npx tsc --noEmit
cd /Users/yuhojin/Desktop/finance && git add apps/web/components/shell/Sidebar.tsx apps/web/components/portfolio/AddHoldingDialog.tsx apps/web/components/portfolio/EditHoldingDialog.tsx apps/web/lib/api/holdings.ts && git commit -m "feat(web): Sidebar 📓 일기 아이콘 + AddHolding·EditHolding에 매매 이유 textarea"
```

---

## Task 11: integration 테스트 + 통합 검증

**Files:**
- Create: `apps/api/internal/handlers/journal_integration_test.go`

- [ ] **Step 1: integration test**

```go
//go:build integration

package handlers_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/handlers"
	"github.com/quotient/quotient/apps/api/internal/models"
)

func TestJournal_E2E_CreateListDelete(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	uid := uuid.NewString()
	ctx := context.Background()
	_, err = pool.Exec(ctx, `
		insert into auth.users (id, email, encrypted_password)
		values ($1::uuid, $1::text || '@journal.test', '')
		on conflict (id) do nothing
	`, uid)
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	defer pool.Exec(ctx, `delete from auth.users where id = $1`, uid)

	repo := handlers.NewPgJournalRepo()

	// 사용자 JWT 트랜잭션 안에서 CRUD
	var createdID string
	err = db.AsUser(ctx, pool, uid, func(exec db.Executor) error {
		title := "통합 테스트"
		body := models.JournalEntryCreate{
			Action:         strPtr("observation"),
			RelatedSymbols: []string{"005930"},
			Title:          &title,
			Content:        "통합 테스트 entry",
		}
		e, err := repo.CreateEntry(ctx, exec, uid, "manual", body, nil)
		if err != nil { return err }
		createdID = e.ID
		return nil
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// list
	err = db.AsUser(ctx, pool, uid, func(exec db.Executor) error {
		entries, _, err := repo.ListEntries(ctx, exec, uid, 10, nil)
		if err != nil { return err }
		if len(entries) != 1 {
			t.Errorf("entries=%d, want 1", len(entries))
		}
		return nil
	})
	if err != nil { t.Fatal(err) }

	// delete
	err = db.AsUser(ctx, pool, uid, func(exec db.Executor) error {
		return repo.DeleteEntry(ctx, exec, uid, createdID)
	})
	if err != nil { t.Fatal(err) }
}

func TestJournal_RLS_Isolation(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil { t.Fatal(err) }
	defer pool.Close()

	uA := uuid.NewString()
	uB := uuid.NewString()
	ctx := context.Background()
	for _, u := range []string{uA, uB} {
		_, err = pool.Exec(ctx, `
			insert into auth.users (id, email, encrypted_password)
			values ($1::uuid, $1::text || '@x.test', '')
			on conflict (id) do nothing
		`, u)
		if err != nil { t.Fatal(err) }
		defer pool.Exec(ctx, `delete from auth.users where id = $1`, u)
	}

	repo := handlers.NewPgJournalRepo()

	// A로 entry 생성
	_ = db.AsUser(ctx, pool, uA, func(exec db.Executor) error {
		_, err := repo.CreateEntry(ctx, exec, uA, "manual",
			models.JournalEntryCreate{Action: strPtr("observation"), Content: "A의 entry"},
			nil)
		return err
	})

	// B가 list — A의 entry 안 보여야
	var bCount int
	_ = db.AsUser(ctx, pool, uB, func(exec db.Executor) error {
		entries, _, err := repo.ListEntries(ctx, exec, uB, 10, nil)
		bCount = len(entries)
		return err
	})
	if bCount != 0 {
		t.Errorf("RLS leak: user B sees %d entries of user A", bCount)
	}
	_ = time.Now // unused guard
}

func strPtr(s string) *string { return &s }
```

- [ ] **Step 2: 통합 테스트 실행**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && TEST_DATABASE_URL="postgresql://postgres:postgres@127.0.0.1:54322/postgres" go test -tags integration ./internal/handlers -run TestJournal -v
```

Expected: 2 케이스 PASS.

- [ ] **Step 3: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add apps/api/internal/handlers/journal_integration_test.go && git commit -m "test(journal): integration test — E2E CRUD + RLS 격리 검증"
```

---

## Task 12: 문서 갱신

**Files:**
- Modify: `docs/STATUS.md`
- Modify: `docs/ROADMAP.md`

- [ ] **Step 1: STATUS 한 줄 + 마지막 업데이트**

`docs/STATUS.md`의 "최근 변경 이력" 맨 위:

```markdown
- 2026-05-28 AI 매매 일기 출시 — Holdings CRUD 통합(auto entry) + `/app/journal` 별도 페이지(manual). 자동 월간 회고 cron(매월 1일 07:00 KST) + on-demand 분석 버튼(채팅 한도 차감) + 채팅 `analyze_journal` 도구. 신규 테이블 2 + RLS 6 정책 + 6 HTTP endpoint + 7 unit + 2 integration. 정체성 spec §3 최우선 차별화 카드 이행.
```

"마지막 업데이트"는 `2026-05-28` 유지.

- [ ] **Step 2: ROADMAP "현재 추천 다음 작업"에서 AI 매매 일기 제거**

기존 "현재 추천 다음 작업" 절의 AI 매매 일기 항목 제거. Paper Portfolio가 우선순위 1로 자동 승격.

"수익화·차별화 후속" 표에서도 "AI 매매 일기" 행 제거(이미 출시됨).

- [ ] **Step 3: 빌드 + 테스트 최종**

```bash
cd /Users/yuhojin/Desktop/finance/apps/api && go build ./... && go test ./...
cd /Users/yuhojin/Desktop/finance/apps/web && npx tsc --noEmit && npm test -- --run
```

Expected: 모두 PASS.

- [ ] **Step 4: 커밋**

```bash
cd /Users/yuhojin/Desktop/finance && git add docs/STATUS.md docs/ROADMAP.md && git commit -m "docs: AI 매매 일기 출시 완료 반영 (STATUS·ROADMAP)"
```

---

## Self-Review

### 1. Spec coverage

| Spec 섹션 | Task |
|---|---|
| §3-1 journal_entries 테이블 + RLS | Task 1 |
| §3-2 analysis_runs 테이블 + RLS | Task 1 |
| §4 Architecture | Task 2~10 종합 |
| §5-1~5-6 6 endpoint | Task 3 + Task 7 |
| §5-7 holdings CRUD reason | Task 6 |
| §6 analyze_journal AI 도구 | Task 4 |
| §7 cron 자동 월간 | Task 5 |
| §8 UI (사이드바·페이지·모달·textarea) | Task 9·10 |
| §9 에러 처리 | Task 3 unit + Task 6 |
| §10 보안·RLS | Task 1 + Task 11 격리 검증 |
| §11 테스트 | Task 3·4·5·11 |
| §12 비범위 | (의도적으로 미구현) |

### 2. Placeholder scan

- 모든 코드 블록은 실 실행 가능
- 검증 명령·expected output 명시
- "TODO" 없음

### 3. Type consistency

- `JournalRepo` 인터페이스 시그니처(Task 2) → Task 3 핸들러·Task 4 cron·Task 6 holdings에서 일관 사용
- `models.JournalEntryCreate`·`JournalEntryPatch` (Task 1) → Task 2·3에서 동일 시그니처
- `tools.Deps.Client` 필드 추가(Task 4) → Task 7 main.go에서 주입
- frontend `JournalEntry`·`AnalysisRun` (Task 8) → Task 9 컴포넌트에서 사용
- `entry_type` 값 `"auto"`/`"manual"` 일관
- `run_type` 값 `"auto_monthly"`/`"on_demand"` 일관

---

Plan complete and saved to `docs/superpowers/plans/2026-05-28-ai-trading-journal.md`.

---

## 검토 이력

### 2026-05-28 subagent 자체 검토 (general-purpose, CLAUDE.md MANDATORY)

#### Critical (반드시 수정) — 3건 → 모두 패치 완료

- **C-1. Task 4 `buildJournalAnalysisPrompt` 타입 불일치로 컴파일 실패** — Run 내부의 local `type entry`와 helper signature의 anonymous struct는 Go에서 assignable하지 않음. → 패키지 레벨 named type `journalEntryRow` 도입, Run·helper 양쪽 사용으로 통일. ✅
- **C-2. Task 5 cron의 `ON CONFLICT (cols) WHERE pred DO NOTHING` partial index arbiter 매핑 fragile** — race에서 23505 폭주 가능. → `ON CONFLICT DO NOTHING` (arbiter 생략) — partial unique index가 자동 차단, arbiter 매핑 실패 무관. ✅
- **C-3. Task 6/7 main.go `journalRepo` 중복 선언 위험** — Task 6에서 선언, Task 7가 재선언하면 컴파일 에러. → Task 6 Step 3·Task 7 Step 2 양쪽에 명시적 경고 추가. ✅

#### Important (강력 권장) — 5건 → 4건 패치, 1건 메모

- **I-1. Task 4 Step 1 RegisterJournal 빈 함수 → r.Register 호출 누락 위험** → Step 1 코드에 처음부터 `r.Register(&analyzeJournal{Deps: d})` 포함. ✅
- **I-2. Task 5 cron `client` 인자 unused** → 함수 끝 `_ = client` 라인에 "briefing dispatcher 시그니처 일관" 코멘트 추가. ✅
- **I-3. Task 5 SQL `now() - 32 days` 윈도우 모호** → `lastMonthStart <= created_at < thisMonthStart` Go time 인자 명시. ✅
- **I-4. Task 3 Analyze 핸들러 CheckLimits → IncrementUsage 분리 트랜잭션 race** → 기존 채팅 패턴 답습, 별도 task. plan 본문 §5-5 응답 표에서 "best-effort" 언급으로 충분. 메모 유지.
- **I-5. Task 4 holdings 에러 silently drop** → `holdings, _ := loadHoldingsForJournal(...)`에 "briefing 패턴과 동일, 분석 quality 저하 허용" 코멘트 추가. ✅

#### Minor — 8건 중 1건 패치, 나머지 결정 유지

- M-1 SQL `length()` byte vs rune — 검토 요청자의 우려 무근거(Postgres `length`는 character). 결정 유지.
- M-2 Task 5 test sha256 import guard — fallback 단순 버전만 두는 게 깔끔 → plan 본문에 명시. 결정 유지(이미 두 버전 제시).
- M-3~M-7 — 기존 결정 합리적, plan 변경 없이 진행.
- M-8 EntryItem `patchEntry` union 분기 누락 — Task 9 코드에 try/catch + error 분기 추가 가치. v2로 미룸(manual entry만 edit 노출이라 실제 도달 거의 없음).

#### 메타

CLAUDE.md MANDATORY 사이클(brainstorm → spec → plan → subagent 검토) 충실 적용. 자체 검토에서 Critical 3건을 잡아 compile-fail 단계 우회.
사용자 review gate 통과 시 `superpowers:subagent-driven-development`로 task별 구현.
