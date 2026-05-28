package handlers

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/models"
)

var ErrPaperAccountNotFound = errors.New("paper account not found")
var ErrPaperHoldingNotFound = errors.New("paper holding not found")
var ErrInsufficientCash = errors.New("insufficient cash balance")

type PaperRepo interface {
	// account
	GetOrCreateAccount(ctx context.Context, exec db.Executor, userID string) (*models.PaperAccount, error)
	// ApplyCashDelta: atomic UPDATE. delta>0=입금, delta<0=출금. 잔고 부족 시 ErrInsufficientCash.
	ApplyCashDelta(ctx context.Context, exec db.Executor, userID string, delta float64) (float64, error)
	ResetAccount(ctx context.Context, exec db.Executor, userID string, initialCash float64) (*models.PaperAccount, error)

	// holdings
	ListHoldings(ctx context.Context, exec db.Executor, userID string) ([]models.PaperHolding, error)
	GetHolding(ctx context.Context, exec db.Executor, userID, instrumentID string) (*models.PaperHolding, error)
	UpsertHolding(ctx context.Context, exec db.Executor, userID, instrumentID string, quantity, avgCost float64) error
	DeleteHolding(ctx context.Context, exec db.Executor, userID, instrumentID string) error
	DeleteAllHoldings(ctx context.Context, exec db.Executor, userID string) error

	// transactions
	InsertTransaction(ctx context.Context, exec db.Executor, t *models.PaperTransaction) error
	ListTransactions(ctx context.Context, exec db.Executor, userID string, limit int, before *time.Time) ([]models.PaperTransaction, bool, error)
	ListActiveTransactionsSince(ctx context.Context, exec db.Executor, userID string, since time.Time) ([]models.PaperTransaction, error)
	InactivateAllTransactions(ctx context.Context, exec db.Executor, userID string) error
}

type PgPaperRepo struct{}

func NewPgPaperRepo() *PgPaperRepo { return &PgPaperRepo{} }

// --- account ---

func (PgPaperRepo) GetOrCreateAccount(ctx context.Context, exec db.Executor, userID string) (*models.PaperAccount, error) {
	a := &models.PaperAccount{}
	// 단일 round-trip + 동시 두 요청 race-safe.
	// ON CONFLICT 시 의미 없는 자기 자신 UPDATE로 RETURNING 보장.
	row := exec.QueryRow(ctx, `
		insert into public.paper_account (user_id) values ($1)
		on conflict (user_id) do update set updated_at = public.paper_account.updated_at
		returning user_id::text, initial_cash::float8, cash_balance::float8, base_currency, created_at, updated_at
	`, userID)
	if err := row.Scan(&a.UserID, &a.InitialCash, &a.CashBalance, &a.BaseCurrency, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	return a, nil
}

// ApplyCashDelta는 atomic UPDATE로 잔고를 갱신. delta > 0이면 입금(매도), < 0이면 출금(매수).
// 출금 시 잔고 부족하면 RowsAffected=0 → ErrInsufficientCash 반환 — race-safe.
func (PgPaperRepo) ApplyCashDelta(ctx context.Context, exec db.Executor, userID string, delta float64) (float64, error) {
	var newBalance float64
	q := `
		update public.paper_account
		set cash_balance = cash_balance + $2
		where user_id = $1 and cash_balance + $2 >= 0
		returning cash_balance::float8
	`
	err := exec.QueryRow(ctx, q, userID, delta).Scan(&newBalance)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrInsufficientCash
		}
		return 0, err
	}
	return newBalance, nil
}

func (PgPaperRepo) ResetAccount(ctx context.Context, exec db.Executor, userID string, initialCash float64) (*models.PaperAccount, error) {
	a := &models.PaperAccount{}
	row := exec.QueryRow(ctx, `
		update public.paper_account
		set initial_cash = $2, cash_balance = $2
		where user_id = $1
		returning user_id::text, initial_cash::float8, cash_balance::float8, base_currency, created_at, updated_at
	`, userID, initialCash)
	if err := row.Scan(&a.UserID, &a.InitialCash, &a.CashBalance, &a.BaseCurrency, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	return a, nil
}

// --- holdings ---

const paperHoldingsBaseSelect = `
select h.id::text, h.user_id::text, h.instrument_id::text,
       i.symbol, i.name, i.currency,
       h.quantity::float8, h.avg_cost::float8,
       h.created_at, h.updated_at
from public.paper_holdings h
join public.instruments i on i.id = h.instrument_id
`

func (PgPaperRepo) ListHoldings(ctx context.Context, exec db.Executor, userID string) ([]models.PaperHolding, error) {
	rows, err := exec.Query(ctx, paperHoldingsBaseSelect+` where h.user_id = $1 order by h.updated_at desc`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.PaperHolding
	for rows.Next() {
		var h models.PaperHolding
		if err := rows.Scan(&h.ID, &h.UserID, &h.InstrumentID, &h.Symbol, &h.Name, &h.Currency, &h.Quantity, &h.AvgCost, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (PgPaperRepo) GetHolding(ctx context.Context, exec db.Executor, userID, instrumentID string) (*models.PaperHolding, error) {
	row := exec.QueryRow(ctx, paperHoldingsBaseSelect+` where h.user_id = $1 and h.instrument_id = $2::uuid`, userID, instrumentID)
	var h models.PaperHolding
	err := row.Scan(&h.ID, &h.UserID, &h.InstrumentID, &h.Symbol, &h.Name, &h.Currency, &h.Quantity, &h.AvgCost, &h.CreatedAt, &h.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrPaperHoldingNotFound
		}
		return nil, err
	}
	return &h, nil
}

// UpsertHolding — quantity, avg_cost 모두 새 값으로 SET (호출자가 가중 평균 계산해 전달).
func (PgPaperRepo) UpsertHolding(ctx context.Context, exec db.Executor, userID, instrumentID string, quantity, avgCost float64) error {
	_, err := exec.Exec(ctx, `
		insert into public.paper_holdings (user_id, instrument_id, quantity, avg_cost)
		values ($1, $2::uuid, $3, $4)
		on conflict (user_id, instrument_id) do update
		set quantity = excluded.quantity, avg_cost = excluded.avg_cost
	`, userID, instrumentID, quantity, avgCost)
	return err
}

func (PgPaperRepo) DeleteHolding(ctx context.Context, exec db.Executor, userID, instrumentID string) error {
	ct, err := exec.Exec(ctx, `delete from public.paper_holdings where user_id = $1 and instrument_id = $2::uuid`, userID, instrumentID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrPaperHoldingNotFound
	}
	return nil
}

func (PgPaperRepo) DeleteAllHoldings(ctx context.Context, exec db.Executor, userID string) error {
	_, err := exec.Exec(ctx, `delete from public.paper_holdings where user_id = $1`, userID)
	return err
}

// --- transactions ---

func (PgPaperRepo) InsertTransaction(ctx context.Context, exec db.Executor, t *models.PaperTransaction) error {
	row := exec.QueryRow(ctx, `
		insert into public.paper_transactions
		  (user_id, instrument_id, action, quantity, price, currency, fx_to_krw, total_krw)
		values ($1, $2::uuid, $3, $4, $5, $6, $7, $8)
		returning id::text, created_at
	`, t.UserID, t.InstrumentID, t.Action, t.Quantity, t.Price, t.Currency, t.FxToKRW, t.TotalKRW)
	return row.Scan(&t.ID, &t.CreatedAt)
}

const paperTxBaseSelect = `
select t.id::text, t.user_id::text, t.instrument_id::text, i.symbol,
       t.action, t.quantity::float8, t.price::float8, t.currency,
       t.fx_to_krw::float8, t.total_krw::float8, t.active, t.created_at
from public.paper_transactions t
join public.instruments i on i.id = t.instrument_id
`

func (PgPaperRepo) ListTransactions(ctx context.Context, exec db.Executor, userID string, limit int, before *time.Time) ([]models.PaperTransaction, bool, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := paperTxBaseSelect + ` where t.user_id = $1 and t.active = true`
	args := []any{userID}
	if before != nil {
		q += ` and t.created_at < $2`
		args = append(args, *before)
		q += ` order by t.created_at desc, t.id desc limit $3`
		args = append(args, limit+1)
	} else {
		q += ` order by t.created_at desc, t.id desc limit $2`
		args = append(args, limit+1)
	}
	rows, err := exec.Query(ctx, q, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	var out []models.PaperTransaction
	for rows.Next() {
		var t models.PaperTransaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.InstrumentID, &t.Symbol, &t.Action, &t.Quantity, &t.Price, &t.Currency, &t.FxToKRW, &t.TotalKRW, &t.Active, &t.CreatedAt); err != nil {
			return nil, false, err
		}
		out = append(out, t)
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

func (PgPaperRepo) ListActiveTransactionsSince(ctx context.Context, exec db.Executor, userID string, since time.Time) ([]models.PaperTransaction, error) {
	rows, err := exec.Query(ctx, paperTxBaseSelect+`
		where t.user_id = $1 and t.active = true and t.created_at >= $2
		order by t.created_at, t.id
	`, userID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.PaperTransaction
	for rows.Next() {
		var t models.PaperTransaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.InstrumentID, &t.Symbol, &t.Action, &t.Quantity, &t.Price, &t.Currency, &t.FxToKRW, &t.TotalKRW, &t.Active, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (PgPaperRepo) InactivateAllTransactions(ctx context.Context, exec db.Executor, userID string) error {
	_, err := exec.Exec(ctx, `update public.paper_transactions set active = false where user_id = $1 and active = true`, userID)
	return err
}
