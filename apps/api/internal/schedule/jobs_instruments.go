package schedule

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/quotient/quotient/apps/api/internal/ingest"
)

// JobUpdateInstruments refreshes the KR instrument master from KIND.
// 매일 06:00 KST 실행. KOSPI + KOSDAQ 순차 fetch + upsert + alias 시드.
func JobUpdateInstruments(ctx context.Context, d Deps) error {
	for _, market := range []string{"KOSPI", "KOSDAQ"} {
		items, err := d.KIND.FetchInstruments(ctx, market)
		if err != nil {
			return fmt.Errorf("kind %s: %w", market, err)
		}
		n, err := ingest.UpsertInstruments(ctx, d.Pool, items)
		if err != nil {
			return fmt.Errorf("upsert %s: %w", market, err)
		}
		slog.Info("instruments updated", "market", market, "count", n)
	}
	// spec §10-9: 시드 단계 alias 등록 — KR 전체 종목의 회사명·종목코드를 alias로.
	// 두 시장 처리 후 일괄 — UpsertInstruments가 id 반환 안 하므로 DB 재조회.
	seeded, err := seedKRAliases(ctx, d)
	if err != nil {
		slog.Warn("alias seed partial", "err", err)
	}
	slog.Info("aliases seeded", "count", seeded)
	// US 종목 마스터는 lazy: 사용자가 추가 시 Yahoo 메타로 등록 (W3)
	return nil
}

// seedKRAliases는 KRX 종목에 대해 회사명·종목코드를 alias로 시드 등록.
// (alias) PK이라 source='seed'로 ON CONFLICT update — 항상 최신 매핑 유지.
func seedKRAliases(ctx context.Context, d Deps) (int, error) {
	rows, err := d.Pool.Query(ctx, `
		select id::text, symbol, name from public.instruments
		where exchange = 'KRX' and is_active = true
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	var n int
	for rows.Next() {
		var id, symbol, name string
		if err := rows.Scan(&id, &symbol, &name); err != nil {
			return n, err
		}
		// 종목코드 alias (예: "005930" → 삼성전자 instrument)
		if err := ingest.SeedAlias(ctx, d.Pool, symbol, id); err == nil {
			n++
		}
		// 한글명 alias (예: "삼성전자" → 같은 instrument)
		if name != "" {
			if err := ingest.SeedAlias(ctx, d.Pool, name, id); err == nil {
				n++
			}
		}
	}
	return n, rows.Err()
}
