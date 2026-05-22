package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PgProfileRepo struct {
	pool *pgxpool.Pool
}

func NewPgProfileRepo(pool *pgxpool.Pool) *PgProfileRepo {
	return &PgProfileRepo{pool: pool}
}

func (r *PgProfileRepo) Get(ctx context.Context, uid string) (map[string]any, error) {
	row := r.pool.QueryRow(ctx, `
		select id, display_name, base_currency, ui_intensity,
		       onboarding_completed, daily_briefing_enabled,
		       created_at, updated_at
		from public.profiles where id = $1`, uid)
	var id, baseCurrency, uiIntensity string
	var displayName *string
	var onboarding, daily bool
	var created, updated any
	if err := row.Scan(&id, &displayName, &baseCurrency, &uiIntensity, &onboarding, &daily, &created, &updated); err != nil {
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

func (r *PgProfileRepo) Update(ctx context.Context, uid string, patch map[string]any) error {
	if len(patch) == 0 {
		return nil
	}
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
		return nil
	}
	args = append(args, uid)
	q := fmt.Sprintf(`update public.profiles set %s where id = $%d`, strings.Join(sets, ", "), i)
	_, err := r.pool.Exec(ctx, q, args...)
	return err
}
