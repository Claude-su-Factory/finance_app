package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/models"
)

var ErrHoldingNotFound = errors.New("holding not found")
var ErrHoldingConflict = errors.New("holding already exists for this instrument")

type HoldingRepo interface {
	List(ctx context.Context, userID string) ([]models.HoldingEnriched, error)
	Create(ctx context.Context, userID, instrumentID string, qty, avgCost float64, openedAt *string, note *string) (*models.Holding, error)
	Update(ctx context.Context, userID, id string, patch map[string]any) (*models.Holding, error)
	Delete(ctx context.Context, userID, id string) error
}

type PgHoldingRepo struct {
	pool *pgxpool.Pool
}

func NewPgHoldingRepo(pool *pgxpool.Pool) *PgHoldingRepo {
	return &PgHoldingRepo{pool: pool}
}

// List는 holdings + instruments + 최신 quotes를 join하여 enrichment 전 상태로 반환.
// 가격·환율 환산·비중 계산은 handler에서 처리.
func (r *PgHoldingRepo) List(ctx context.Context, userID string) ([]models.HoldingEnriched, error) {
	rows, err := r.pool.Query(ctx, `
		select
		  h.id::text, h.instrument_id::text, h.quantity::float8, h.avg_cost::float8,
		  h.opened_at, h.note, h.created_at, h.updated_at,
		  i.symbol, i.exchange, i.name, i.asset_class, i.currency,
		  coalesce(q.price, 0)::float8
		from public.holdings h
		join public.instruments i on i.id = h.instrument_id
		left join public.quotes q on q.instrument_id = h.instrument_id
		where h.user_id = $1
		order by h.created_at desc
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.HoldingEnriched
	for rows.Next() {
		var h models.HoldingEnriched
		if err := rows.Scan(
			&h.ID, &h.InstrumentID, &h.Quantity, &h.AvgCost,
			&h.OpenedAt, &h.Note, &h.CreatedAt, &h.UpdatedAt,
			&h.Symbol, &h.Exchange, &h.Name, &h.AssetClass, &h.Currency,
			&h.CurrentPrice,
		); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *PgHoldingRepo) Create(ctx context.Context, userID, instrumentID string, qty, avgCost float64, openedAt *string, note *string) (*models.Holding, error) {
	row := r.pool.QueryRow(ctx, `
		insert into public.holdings (user_id, instrument_id, quantity, avg_cost, opened_at, note)
		values ($1, $2, $3, $4, $5, $6)
		returning id::text, instrument_id::text, quantity::float8, avg_cost::float8, opened_at, note, created_at, updated_at
	`, userID, instrumentID, qty, avgCost, openedAt, note)
	var h models.Holding
	if err := row.Scan(&h.ID, &h.InstrumentID, &h.Quantity, &h.AvgCost, &h.OpenedAt, &h.Note, &h.CreatedAt, &h.UpdatedAt); err != nil {
		// pgx unique violation: SQLSTATE 23505
		if strings.Contains(err.Error(), "23505") {
			return nil, ErrHoldingConflict
		}
		return nil, err
	}
	return &h, nil
}

func (r *PgHoldingRepo) Update(ctx context.Context, userID, id string, patch map[string]any) (*models.Holding, error) {
	sets := []string{}
	args := []any{}
	i := 1
	for k, v := range patch {
		switch k {
		case "quantity", "avg_cost", "note", "opened_at":
			sets = append(sets, fmt.Sprintf("%s = $%d", k, i))
			args = append(args, v)
			i++
		}
	}
	if len(sets) == 0 {
		// 갱신 없음 — 현재 행 반환
		row := r.pool.QueryRow(ctx, `
			select id::text, instrument_id::text, quantity::float8, avg_cost::float8, opened_at, note, created_at, updated_at
			from public.holdings where id = $1 and user_id = $2
		`, id, userID)
		var h models.Holding
		if err := row.Scan(&h.ID, &h.InstrumentID, &h.Quantity, &h.AvgCost, &h.OpenedAt, &h.Note, &h.CreatedAt, &h.UpdatedAt); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrHoldingNotFound
			}
			return nil, err
		}
		return &h, nil
	}
	args = append(args, id, userID)
	q := fmt.Sprintf(`
		update public.holdings set %s
		where id = $%d and user_id = $%d
		returning id::text, instrument_id::text, quantity::float8, avg_cost::float8, opened_at, note, created_at, updated_at
	`, strings.Join(sets, ", "), i, i+1)

	row := r.pool.QueryRow(ctx, q, args...)
	var h models.Holding
	if err := row.Scan(&h.ID, &h.InstrumentID, &h.Quantity, &h.AvgCost, &h.OpenedAt, &h.Note, &h.CreatedAt, &h.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrHoldingNotFound
		}
		return nil, err
	}
	return &h, nil
}

func (r *PgHoldingRepo) Delete(ctx context.Context, userID, id string) error {
	ct, err := r.pool.Exec(ctx, `delete from public.holdings where id = $1 and user_id = $2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrHoldingNotFound
	}
	return nil
}
