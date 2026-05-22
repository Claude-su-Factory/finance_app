package schedule

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/quotient/quotient/apps/api/internal/ingest"
)

// JobUpdateIndicators fetches FRED + ECOS economic indicators daily at 07:00 KST.
// 각 시리즈 실패는 다른 시리즈에 영향 안 줌 (부분 실패 허용 — spec §4 "부분 적재").
func JobUpdateIndicators(ctx context.Context, d Deps) error {
	var anyOK bool

	// FRED 시리즈: Fed funds rate + 10y treasury
	for _, code := range []string{"DFF", "DGS10"} {
		obs, err := d.FRED.FetchObservations(ctx, code)
		if err != nil {
			slog.Warn("fred fetch skip", "code", code, "err", err)
			continue
		}
		// FRED는 name·unit 미제공 → ingest에서 code만 사용 (UI에서 표시명 정의)
		n, err := ingest.UpsertIndicators(ctx, d.Pool, obs)
		if err != nil {
			slog.Warn("fred upsert skip", "code", code, "err", err)
			continue
		}
		slog.Info("fred indicator", "code", code, "rows", n)
		anyOK = true
	}

	// ECOS: 한국 기준금리 — 최근 5년치 (Cycle "M" 월별)
	end := time.Now().Format("200601")
	start := time.Now().AddDate(-5, 0, 0).Format("200601")
	obs, err := d.ECOS.FetchSeries(ctx, "722Y001", "M", start, end)
	if err != nil {
		slog.Warn("ecos skip", "err", err)
	} else {
		n, err := ingest.UpsertIndicators(ctx, d.Pool, obs)
		if err != nil {
			slog.Warn("ecos upsert skip", "err", err)
		} else {
			slog.Info("ecos indicator", "code", "722Y001", "rows", n)
			anyOK = true
		}
	}

	if !anyOK {
		return fmt.Errorf("all indicators failed")
	}
	return nil
}
