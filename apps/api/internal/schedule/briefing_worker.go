package schedule

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"time"

	"github.com/quotient/quotient/apps/api/internal/ai"
	"github.com/quotient/quotient/apps/api/internal/ai/tools"
)

// JobBriefingDispatcher는 매 분 동작. 현재 분에 해당하는 사용자만 처리.
// 07:00 ~ 07:59 사이에서 user hash 분단위 분산.
// registry가 nil이 아니면 brief 작성 전 사용자 portfolio/market/watchlist를
// 미리 fetch하여 system prompt에 주입 (spec §10-8).
func JobBriefingDispatcher(ctx context.Context, d Deps, client ai.Client, registry *tools.Registry) error {
	now := time.Now().In(SeoulLoc())
	if now.Hour() != 7 {
		return nil
	}
	currentMinute := now.Minute()

	// profiles.id가 auth.users.id의 FK이므로 직접 조회
	rows, err := d.Pool.Query(ctx, `
		select id::text from public.profiles
		where daily_briefing_enabled = true and onboarding_completed = true
	`)
	if err != nil {
		return fmt.Errorf("query active users: %w", err)
	}
	defer rows.Close()

	type user struct{ id string }
	var users []user
	for rows.Next() {
		var u user
		if err := rows.Scan(&u.id); err != nil {
			return err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	today := now.Format("2006-01-02")
	processed := 0
	for _, u := range users {
		if userMinuteSlot(u.id) != currentMinute {
			continue
		}
		var exists bool
		_ = d.Pool.QueryRow(ctx, `select exists(select 1 from public.ai_briefings where user_id = $1 and date = $2)`, u.id, today).Scan(&exists)
		if exists {
			continue
		}
		content := generateBriefing(ctx, client, registry, u.id)
		_, err := d.Pool.Exec(ctx, `
			insert into public.ai_briefings (user_id, date, content_md, model)
			values ($1, $2, $3, $4)
			on conflict (user_id, date) do nothing
		`, u.id, today, content, ai.ModelSonnet)
		if err != nil {
			slog.Warn("briefing insert failed", "user", u.id, "err", err)
			continue
		}
		processed++
	}
	slog.Info("briefings dispatched", "minute", currentMinute, "processed", processed)
	return nil
}

// userMinuteSlot는 user id의 sha256 → 0~59.
func userMinuteSlot(uid string) int {
	sum := sha256.Sum256([]byte(uid))
	return int(sum[0]) % 60
}

// generateBriefing은 사용자 데이터(portfolio/market/watchlist)를 미리 fetch하여
// system prompt에 주입한 뒤 ai.Client를 1턴 호출.
// registry가 nil이면 데이터 주입 없이 일반 안내 브리핑 (fallback).
// spec §10-8: "보유 자산 + 어제 시세 변화 + 주요 지수·환율 변화 + 관심 종목 변화" 입력.
func generateBriefing(ctx context.Context, client ai.Client, registry *tools.Registry, uid string) string {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	system := buildBriefingSystem(ctx, registry, uid)

	req := ai.ChatRequest{
		Model:  ai.ModelSonnet,
		System: system,
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: "오늘의 브리핑을 한국어 3~5문장으로 작성해주세요. 보유 자산·시장·관심 종목의 변화에 초점을 맞추세요."},
		},
		MaxTokens: 1024,
	}
	ch, err := client.StreamChat(ctx, req)
	if err != nil {
		return "## 오늘의 브리핑\n\n시장 데이터를 분석 중입니다. 잠시 후 다시 확인해주세요.\n\n*(MVP: 실 API 키가 없으면 Mock 메시지가 표시됩니다.)*"
	}
	var content string
	for ev := range ch {
		if ev.Type == ai.EventToken {
			text, _ := ev.Data["text"].(string)
			content += text
		}
	}
	if content == "" {
		content = "## 오늘의 브리핑\n\n오늘은 보유 자산을 점검하기 좋은 날입니다."
	}
	return content
}

// buildBriefingSystem composes the system prompt with prefetched user data.
// registry nil → fallback system without data injection.
func buildBriefingSystem(ctx context.Context, registry *tools.Registry, uid string) string {
	base := `당신은 Quotient의 일일 브리핑 작성자입니다. 한국어 마크다운 3~5문장으로 작성하세요.

규칙:
- 직접 매수/매도 권유 금지. "분석 관점에서 ~를 점검해볼 수 있습니다" 형태로.
- 사용자의 보유 자산·관심 종목·시장 변화에 초점.
- 추측 금지. 제공된 데이터만 근거로.`

	if registry == nil {
		return base
	}

	portfolio := tools.ExecuteAndSerialize(ctx, registry, "get_portfolio", uid, map[string]any{})
	market := tools.ExecuteAndSerialize(ctx, registry, "get_market_overview", uid, map[string]any{})
	watchlist := tools.ExecuteAndSerialize(ctx, registry, "get_watchlist", uid, map[string]any{})

	return fmt.Sprintf(`%s

오늘 데이터 컨텍스트 (JSON):

[보유 자산]
%s

[시장 스냅샷]
%s

[관심 종목]
%s`, base, portfolio, market, watchlist)
}
