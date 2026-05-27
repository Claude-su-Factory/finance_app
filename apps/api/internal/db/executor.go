package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Executor는 *pgxpool.Pool과 pgx.Tx가 모두 만족하는 최소 인터페이스.
// repo는 이 타입을 받아 호출자가 트랜잭션·풀을 선택할 수 있게 한다.
// 사용자 데이터 repo는 handler에서 AsUser로 트랜잭션을 열어 전달,
// 공개 데이터 repo·cron은 pool을 직접 전달한다.
type Executor interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}
