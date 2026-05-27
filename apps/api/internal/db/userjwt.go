package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AsUser opens a transaction, sets role=authenticated and the JWT claims
// (LOCAL scope) so Supabase RLS policies that reference auth.uid() apply.
// 인자로 받은 ctx의 cancel 여부와 무관하게 rollback이 깨끗하게 끝나도록
// rollback은 별도 background ctx로 호출한다.
//
// set_config(_, _, true) → LOCAL — 트랜잭션 종료 시 자동 해제.
// SET LOCAL은 placeholder를 지원하지 않으므로 set_config 함수를 사용한다.
// Supabase auth.uid()는 claim.sub 폴백을 먼저 시도하므로 둘 다 set한다.
func AsUser(ctx context.Context, pool *pgxpool.Pool, userID string, fn func(Executor) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()

	claims, err := json.Marshal(map[string]any{
		"sub":  userID,
		"role": "authenticated",
	})
	if err != nil {
		return fmt.Errorf("marshal claims: %w", err)
	}

	if _, err := tx.Exec(ctx, `select set_config('role', 'authenticated', true)`); err != nil {
		return fmt.Errorf("set role: %w", err)
	}
	if _, err := tx.Exec(ctx, `select set_config('request.jwt.claims', $1, true)`, string(claims)); err != nil {
		return fmt.Errorf("set claims: %w", err)
	}
	if _, err := tx.Exec(ctx, `select set_config('request.jwt.claim.sub', $1, true)`, userID); err != nil {
		return fmt.Errorf("set claim.sub: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Compile-time assertions.
var _ Executor = (*pgxpool.Pool)(nil)
var _ Executor = (pgx.Tx)(nil)
