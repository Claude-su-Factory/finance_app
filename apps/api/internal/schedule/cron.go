package schedule

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/sources/ecos"
	"github.com/quotient/quotient/apps/api/internal/sources/fred"
	"github.com/quotient/quotient/apps/api/internal/sources/fx"
	"github.com/quotient/quotient/apps/api/internal/sources/kind"
	"github.com/quotient/quotient/apps/api/internal/sources/yahoo"
	"github.com/robfig/cron/v3"
)

// Deps groups dependencies passed into job functions. 명시적 주입으로 테스트 용이.
type Deps struct {
	Pool  *pgxpool.Pool
	KIND  *kind.Client
	Yahoo *yahoo.Client
	FX    *fx.Client
	FRED  *fred.Client
	ECOS  *ecos.Client
}

// Start registers cron schedules and returns the running cron instance.
// 호출자가 Stop()으로 graceful shutdown 책임.
func Start(ctx context.Context, d Deps) *cron.Cron {
	// SkipIfStillRunning: 같은 잡이 직전 tick에서 진행 중이면 새 실행 skip.
	// instruments 잡(KOSPI+KOSDAQ 풀 fetch)이 1분을 넘기더라도 중복 실행 방지.
	c := cron.New(
		cron.WithLocation(SeoulLoc()),
		cron.WithChain(cron.SkipIfStillRunning(cron.DefaultLogger)),
	)

	// 종목 마스터 — 매일 06:00 KST (spec §4)
	mustAdd(c, "0 6 * * *", "instruments", func() {
		if err := JobUpdateInstruments(ctx, d); err != nil {
			slog.Error("cron instruments failed", "err", err)
		}
	})
	// KR 일봉 — 매일 16:30 KST (장 마감 후)
	mustAdd(c, "30 16 * * *", "kr_prices", func() {
		if err := JobUpdateKRPrices(ctx, d); err != nil {
			slog.Error("cron kr_prices failed", "err", err)
		}
	})
	// US 일봉 — 매일 06:00 KST (US 장 마감 후)
	mustAdd(c, "0 6 * * *", "us_prices", func() {
		if err := JobUpdateUSPrices(ctx, d); err != nil {
			slog.Error("cron us_prices failed", "err", err)
		}
	})
	// quotes (INDEX∪holdings∪watchlist 폴링) — 매분, 장중만
	mustAdd(c, "* * * * *", "quotes", func() {
		if err := JobUpdateMarketQuotes(ctx, d); err != nil {
			slog.Error("cron quotes failed", "err", err)
		}
	})
	// FX — 5분, 24/7
	mustAdd(c, "*/5 * * * *", "fx", func() {
		if err := JobUpdateFXRates(ctx, d); err != nil {
			slog.Error("cron fx failed", "err", err)
		}
	})
	// 경제 지표 — 매일 07:00 KST
	mustAdd(c, "0 7 * * *", "indicators", func() {
		if err := JobUpdateIndicators(ctx, d); err != nil {
			slog.Error("cron indicators failed", "err", err)
		}
	})

	c.Start()
	slog.Info("cron started", "jobs", 6, "tz", "Asia/Seoul")
	return c
}

func mustAdd(c *cron.Cron, spec, name string, fn func()) {
	if _, err := c.AddFunc(spec, fn); err != nil {
		slog.Error("cron AddFunc failed", "name", name, "spec", spec, "err", err)
	}
}
