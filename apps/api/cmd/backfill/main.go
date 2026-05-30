package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	"github.com/quotient/quotient/apps/api/internal/backfill"
	"github.com/quotient/quotient/apps/api/internal/config"
	"github.com/quotient/quotient/apps/api/internal/db"
)

func main() {
	years := flag.Int("years", 5, "백필 기간 (연 단위)")
	market := flag.String("market", "KOSPI", "KOSPI | KOSDAQ | NASDAQ | INDICES")
	limit := flag.Int("limit", 0, "최대 종목 수 (0=전체). 디버깅용.")
	flag.Parse()

	ctx := context.Background()
	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}
	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	switch *market {
	case "KOSPI", "KOSDAQ":
		err = backfill.RunKR(ctx, pool, *market, *years, *limit)
	case "NASDAQ":
		err = backfill.RunUS(ctx, pool, *years, *limit)
	case "INDICES":
		err = backfill.RunIndices(ctx, pool, *years, *limit)
	default:
		slog.Error("unknown market", "market", *market)
		os.Exit(1)
	}
	if err != nil {
		slog.Error("backfill", "market", *market, "err", err)
		os.Exit(1)
	}
	slog.Info("backfill done", "market", *market, "years", *years)
}
