package handlers

// TODO(security): 이 repo는 postgres 슈퍼유저 풀로 연결되어 RLS를 우회한다.
// 현재는 핸들러의 `WHERE id = uid` (JWT에서 추출한 uid) 필터로 격리하지만,
// 다중 행 쿼리가 추가되는 W3 이후 시점에 다음 중 하나로 전환 필요:
//   (a) Supabase PostgREST를 사용자 JWT로 호출
//   (b) SET LOCAL role = authenticated + request.jwt.claims 패턴
// 스펙 §10-1 (사용자 JWT 전파)와 정합 위해 W3 전 재검토.

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProfileNotFound = errors.New("profile not found")

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

func (r *PgProfileRepo) Update(ctx context.Context, uid string, patch map[string]any) (map[string]any, error) {
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
		// No whitelisted fields — fall back to Get to return current state
		return r.Get(ctx, uid)
	}
	args = append(args, uid)
	q := fmt.Sprintf(`update public.profiles set %s where id = $%d
		returning id, display_name, base_currency, ui_intensity,
		          onboarding_completed, daily_briefing_enabled, created_at, updated_at`,
		strings.Join(sets, ", "), i)

	row := r.pool.QueryRow(ctx, q, args...)
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
