package schedule

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"time"

	"github.com/quotient/quotient/apps/api/internal/ai"
)

// JobBriefingDispatcher는 매 분 동작. 현재 분에 해당하는 사용자만 처리.
// 07:00 ~ 07:59 사이에서 user hash 분단위 분산.
func JobBriefingDispatcher(ctx context.Context, d Deps, client ai.Client) error {
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
		content := generateBriefing(ctx, client, u.id, d)
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

// generateBriefing은 ai.Client를 1턴 호출. Mock에서는 placeholder, real에서는 실제 분석.
// MVP는 단순화 — 도구 호출 없이 system+user 1턴.
func generateBriefing(ctx context.Context, client ai.Client, uid string, _ Deps) string {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req := ai.ChatRequest{
		Model:  ai.ModelSonnet,
		System: `당신은 Quotient의 일일 브리핑 작성자입니다. 사용자의 보유 자산과 어제 시세 변화를 토대로 3~5문장의 마크다운 브리핑을 작성하세요. 직접 매수/매도 권유 금지.`,
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: fmt.Sprintf("user_id=%s의 오늘 시장 브리핑을 작성해주세요.", uid)},
		},
		MaxTokens: 1024,
	}
	ch, err := client.StreamChat(ctx, req)
	if err != nil {
		return "## 오늘의 브리핑\n\n시장 데이터를 분석 중입니다. 잠시 후 다시 확인해주세요.\n\n*(W4 MVP: 실 API 키가 없으면 Mock 메시지가 표시됩니다.)*"
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
