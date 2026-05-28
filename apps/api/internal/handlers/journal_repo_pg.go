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
