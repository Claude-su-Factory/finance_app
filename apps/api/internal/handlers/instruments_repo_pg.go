package handlers

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/ingest"
)

type PgInstrumentRepo struct {
	pool *pgxpool.Pool
}

func NewPgInstrumentRepo(pool *pgxpool.Pool) *PgInstrumentRepo {
	return &PgInstrumentRepo{pool: pool}
}

func (r *PgInstrumentRepo) SearchByAlias(ctx context.Context, query string) ([]SearchResult, error) {
	rows, err := r.pool.Query(ctx, `
		select i.id::text, i.symbol, i.exchange, i.name
		from public.instrument_aliases a
		join public.instruments i on i.id = a.instrument_id
		where a.alias = $1 and i.is_active = true
		limit 10
	`, strings.ToLower(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanResults(rows)
}

func (r *PgInstrumentRepo) SearchByText(ctx context.Context, query string) ([]SearchResult, error) {
	pat := "%" + strings.ToLower(query) + "%"
	rows, err := r.pool.Query(ctx, `
		select id::text, symbol, exchange, name from public.instruments
		where (lower(name) like $1 or lower(symbol) like $1) and is_active = true
		order by length(symbol) asc
		limit 10
	`, pat)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanResults(rows)
}

func (r *PgInstrumentRepo) LearnAlias(ctx context.Context, alias, instrumentID string) error {
	return ingest.LearnAlias(ctx, r.pool, alias, instrumentID)
}

func scanResults(rows interface {
	Next() bool
	Scan(...any) error
	Err() error
}) ([]SearchResult, error) {
	var out []SearchResult
	for rows.Next() {
		var s SearchResult
		if err := rows.Scan(&s.ID, &s.Symbol, &s.Exchange, &s.Name); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
