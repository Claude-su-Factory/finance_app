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
