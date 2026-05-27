package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/quotient/quotient/apps/api/internal/ai"
	"github.com/quotient/quotient/apps/api/internal/ai/tools"
	"github.com/quotient/quotient/apps/api/internal/auth"
	"github.com/quotient/quotient/apps/api/internal/config"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/handlers"
	"github.com/quotient/quotient/apps/api/internal/router"
	"github.com/quotient/quotient/apps/api/internal/schedule"
	"github.com/quotient/quotient/apps/api/internal/sources/ecos"
	"github.com/quotient/quotient/apps/api/internal/sources/fred"
	"github.com/quotient/quotient/apps/api/internal/sources/fx"
	"github.com/quotient/quotient/apps/api/internal/sources/kind"
	"github.com/quotient/quotient/apps/api/internal/sources/yahoo"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load failed", "err", err)
		os.Exit(1)
	}

	// cron 잡 ctx — 잡 내 외부 호출 cancel 전파 가능하도록 별도 cancel 보유
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	verifier := auth.NewVerifier(cfg.SupabaseJWTSecret)
	profileRepo := handlers.NewPgProfileRepo()
	profileHandler := handlers.NewProfileHandler(profileRepo, pool)
	marketRepo := handlers.NewPgMarketRepo(pool)
	marketHandler := handlers.NewMarketHandler(marketRepo)
	instrumentRepo := handlers.NewPgInstrumentRepo(pool)
	instrumentHandler := handlers.NewInstrumentHandler(instrumentRepo)
	holdingRepo := handlers.NewPgHoldingRepo()
	holdingHandler := handlers.NewHoldingHandler(holdingRepo, pool)
	watchlistRepo := handlers.NewPgWatchlistRepo()
	watchlistHandler := handlers.NewWatchlistHandler(watchlistRepo, pool)

	// AI client (env에서 키 가져오기, 빈 값이면 Mock)
	aiClient := ai.New(cfg.AnthropicAPIKey)

	// Tool registry
	toolRegistry := tools.NewRegistry()
	toolDeps := &tools.Deps{Pool: pool}
	tools.RegisterPortfolio(toolRegistry, toolDeps)
	tools.RegisterQuote(toolRegistry, toolDeps)
	tools.RegisterSearch(toolRegistry, toolDeps)

	// Chat·Briefing handler
	chatRepo := handlers.NewPgChatRepo()
	chatHandler := handlers.NewChatHandler(chatRepo, pool, aiClient, toolRegistry)
	briefingHandler := handlers.NewBriefingHandler(pool)

	historyRepo := handlers.NewPgHistoryRepo(pool)
	historyHandler := handlers.NewHistoryHandler(historyRepo)

	readyz := handlers.ReadyzHandler(pool)

	// cron 워커 시작
	cronWorker := schedule.Start(ctx, schedule.Deps{
		Pool:  pool,
		KIND:  kind.NewClient(""),
		Yahoo: yahoo.NewClient(),
		FX:    fx.NewClient(""),
		FRED:  fred.NewClient("", cfg.FREDAPIKey),
		ECOS:  ecos.NewClient("", cfg.ECOSAPIKey),
	}, aiClient, toolRegistry)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler: router.New(
			verifier, cfg.CORSOrigin,
			profileHandler, marketHandler, instrumentHandler,
			holdingHandler, watchlistHandler,
			chatHandler, briefingHandler,
			historyHandler,
			readyz,
		),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.Info("API listening", "addr", srv.Addr, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen failed", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	logger.Info("shutting down")

	// 1) HTTP 서버 graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown failed", "err", err)
	}

	// 2) cron 워커 정지
	//   - Stop()은 새 잡 추가 차단 + 진행 중 잡 완료 대기를 위한 ctx 반환
	//   - cancel() 호출로 ctx-aware 잡이 외부 API 호출을 즉시 cancel하도록
	cronCtx := cronWorker.Stop()
	cancel()
	select {
	case <-cronCtx.Done():
		logger.Info("cron stopped cleanly")
	case <-time.After(30 * time.Second):
		logger.Warn("cron stop timeout — proceeding")
	}
}
