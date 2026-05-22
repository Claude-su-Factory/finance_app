package handlers

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgMarketRepo struct {
	pool *pgxpool.Pool
}

func NewPgMarketRepo(pool *pgxpool.Pool) *PgMarketRepo {
	return &PgMarketRepo{pool: pool}
}

// TickerSeed returns the seed indices + USD/KRW for the header ticker.
// quotes 행이 없는 종목은 price=0, change_pct=0으로 반환 (UI가 "—"로 처리).
func (r *PgMarketRepo) TickerSeed(ctx context.Context) ([]TickerItem, error) {
	rows, err := r.pool.Query(ctx, `
		select i.symbol, i.name,
		       coalesce(q.price, 0)::float8 as price,
		       coalesce(q.change_pct, 0)::float8 as change_pct
		from public.instruments i
		left join public.quotes q on q.instrument_id = i.id
		where i.symbol in ('KOSPI', 'SPX', 'USD_KRW')
		order by case i.symbol
		  when 'KOSPI' then 1
		  when 'SPX'   then 2
		  when 'USD_KRW' then 3
		end
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []TickerItem
	for rows.Next() {
		var it TickerItem
		if err := rows.Scan(&it.Symbol, &it.Name, &it.Price, &it.ChangePct); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
