package handlers

import (
	"context"
	"errors"
	"strings"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/models"
)

var ErrWatchlistConflict = errors.New("watchlist item already exists")
var ErrWatchlistNotFound = errors.New("watchlist item not found")
var ErrInstrumentRefMissing = errors.New("referenced instrument not found")

type WatchlistRepo interface {
	List(ctx context.Context, exec db.Executor, userID string) ([]models.WatchlistItem, error)
	Add(ctx context.Context, exec db.Executor, userID, instrumentID string) error
	Remove(ctx context.Context, exec db.Executor, userID, instrumentID string) error
}

type PgWatchlistRepo struct{}

func NewPgWatchlistRepo() *PgWatchlistRepo { return &PgWatchlistRepo{} }

func (r *PgWatchlistRepo) List(ctx context.Context, exec db.Executor, userID string) ([]models.WatchlistItem, error) {
	rows, err := exec.Query(ctx, `
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

func (r *PgWatchlistRepo) Add(ctx context.Context, exec db.Executor, userID, instrumentID string) error {
	_, err := exec.Exec(ctx, `
		insert into public.watchlist (user_id, instrument_id)
		values ($1, $2)
	`, userID, instrumentID)
	if err != nil {
		// PK 위반 → 이미 추가됨
		if strings.Contains(err.Error(), "23505") {
			return ErrWatchlistConflict
		}
		// FK 위반 → 존재하지 않는 instrument_id (잘못된 입력)
		if strings.Contains(err.Error(), "23503") {
			return ErrInstrumentRefMissing
		}
	}
	return err
}

func (r *PgWatchlistRepo) Remove(ctx context.Context, exec db.Executor, userID, instrumentID string) error {
	ct, err := exec.Exec(ctx, `delete from public.watchlist where user_id = $1 and instrument_id = $2`, userID, instrumentID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrWatchlistNotFound
	}
	return nil
}
