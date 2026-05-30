package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/migrate"
)

func main() {
	dir := flag.String("dir", "/app/migrations", "마이그레이션 .sql 디렉터리")
	flag.Parse()

	ctx := context.Background()

	// config.Load는 SUPABASE_JWT_SECRET 등 마이그레이션과 무관한 secret까지 required로 요구한다.
	// 마이그레이터는 DATABASE_URL만 필요 → 직접 읽어 decouple (release_command·로컬 실행 모두 단순화).
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		slog.Error("DATABASE_URL required")
		os.Exit(1)
	}

	pool, err := db.New(ctx, dsn)
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := migrate.Run(ctx, pool, *dir); err != nil {
		slog.Error("migrate failed", "err", err)
		os.Exit(1) // release_command 실패 → Fly 배포 중단(fail-closed)
	}
}
