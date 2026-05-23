package handlers

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/models"
)

var ErrWatchlistConflict = errors.New("watchlist item already exists")
var ErrWatchlistNotFound = errors.New("watchlist item not found")

type WatchlistRepo interface {
	List(ctx context.Context, userID string) ([]models.WatchlistItem, error)
	Add(ctx context.Context, userID, instrumentID string) error
	Remove(ctx context.Context, userID, instrumentID string) error
}

type PgWatchlistRepo struct {
	pool *pgxpool.Pool
}

func NewPgWatchlistRepo(pool *pgxpool.Pool) *PgWatchlistRepo {
	return &PgWatchlistRepo{pool: pool}
}

func (r *PgWatchlistRepo) List(ctx context.Context, userID string) ([]models.WatchlistItem, error) {
	rows, err := r.pool.Query(ctx, `
		select
		  w.instrument_id::text, w.added_at,
		  i.symbol, i.exchange, i.name, i.asset_class, i.currency,
		  coalesce(q.price, 0)::float8, coalesce(q.change_pct, 0)::float8
		from public.watchlist w
		join public.instruments i on i.id = w.instrument_id
		left join public.quotes q on q.instrument_id = w.instrument_id
		where w.user_id = $1
		order by w.added_at desc
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.WatchlistItem
	for rows.Next() {
		var x models.WatchlistItem
		if err := rows.Scan(&x.InstrumentID, &x.AddedAt, &x.Symbol, &x.Exchange, &x.Name, &x.AssetClass, &x.Currency, &x.Price, &x.ChangePct); err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

func (r *PgWatchlistRepo) Add(ctx context.Context, userID, instrumentID string) error {
	_, err := r.pool.Exec(ctx, `
		insert into public.watchlist (user_id, instrument_id)
		values ($1, $2)
	`, userID, instrumentID)
	if err != nil && strings.Contains(err.Error(), "23505") {
		return ErrWatchlistConflict
	}
	return err
}

func (r *PgWatchlistRepo) Remove(ctx context.Context, userID, instrumentID string) error {
	ct, err := r.pool.Exec(ctx, `delete from public.watchlist where user_id = $1 and instrument_id = $2`, userID, instrumentID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrWatchlistNotFound
	}
	return nil
}
