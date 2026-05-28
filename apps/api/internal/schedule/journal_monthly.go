package schedule

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/quotient/quotient/apps/api/internal/ai"
	"github.com/quotient/quotient/apps/api/internal/ai/tools"
)

// JobMonthlyJournalDispatcher — 매월 1일 07:00~07:59 KST 사용자 hash 분단위 분산.
// 직전 1달 entries 있는 사용자만 → analyze_journal 도구 호출 → analysis_runs INSERT.
func JobMonthlyJournalDispatcher(ctx context.Context, d Deps, client ai.Client, registry *tools.Registry) error {
	now := time.Now().In(SeoulLoc())
	if now.Day() != 1 || now.Hour() != 7 {
		return nil
	}
	currentMinute := now.Minute()

	// 윈도우: lastMonthStart(KST) <= created_at < thisMonthStart(KST)
	lastMonthStartT := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, SeoulLoc())
	thisMonthStartT := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, SeoulLoc())
	rows, err := d.Pool.Query(ctx, `
		select distinct user_id::text from public.journal_entries
		where created_at >= $1 and created_at < $2
	`, lastMonthStartT, thisMonthStartT)
	if err != nil {
		return fmt.Errorf("query active users: %w", err)
	}
	defer rows.Close()

	var users []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	thisMonthStart := thisMonthStartT.Format("2006-01-02")
	lastMonthStart := lastMonthStartT.Format("2006-01-02")

	processed := 0
	for _, uid := range users {
		if userMinuteSlot(uid) != currentMinute {
			continue
		}
		// 중복 방지 — 이번 달에 이미 생성
		var exists bool
		_ = d.Pool.QueryRow(ctx, `
			select exists(select 1 from public.analysis_runs
			              where user_id = $1::uuid and run_type='auto_monthly' and period_start = $2::date)
		`, uid, lastMonthStart).Scan(&exists)
		if exists {
			continue
		}

		// 도구 호출 (사용자 JWT 트랜잭션 안에서 — ExecuteAndSerialize가 RequiresUserContext=true 처리)
		result := tools.ExecuteAndSerialize(ctx, registry, d.Pool, "analyze_journal", uid, map[string]any{
			"period_days": 30,
		})
		var toolOut struct {
			EntriesCount int    `json:"entries_count"`
			ContentMD    string `json:"content_md"`
			Model        string `json:"model"`
			Error        string `json:"error,omitempty"`
		}
		_ = json.Unmarshal([]byte(result), &toolOut)
		if toolOut.Error != "" {
			slog.Warn("monthly journal analyze skipped", "user", uid, "reason", toolOut.Error)
			continue
		}

		// 슈퍼유저 풀로 INSERT — cron은 fan-out이라 user JWT 없음.
		// ON CONFLICT DO NOTHING — partial unique index가 race 차단.
		_, err := d.Pool.Exec(ctx, `
			insert into public.analysis_runs
			  (user_id, run_type, period_start, period_end, entries_count, content_md, model)
			values ($1::uuid, 'auto_monthly', $2::date, $3::date, $4, $5, $6)
			on conflict do nothing
		`, uid, lastMonthStart, thisMonthStart, toolOut.EntriesCount, toolOut.ContentMD, toolOut.Model)
		if err != nil {
			slog.Warn("monthly journal insert failed", "user", uid, "err", err)
			continue
		}
		processed++
	}
	slog.Info("monthly journal dispatched", "minute", currentMinute, "processed", processed)
	// client 인자는 briefing dispatcher와 시그니처 일관 위해 받음.
	// analyze_journal 도구는 tools.Deps.Client를 직접 사용하므로 본 함수에선 미사용.
	_ = client
	return nil
}
