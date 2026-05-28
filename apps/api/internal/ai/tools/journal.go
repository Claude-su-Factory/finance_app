package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/quotient/quotient/apps/api/internal/ai"
	"github.com/quotient/quotient/apps/api/internal/db"
)

type analyzeJournal struct{ *Deps }

func (t *analyzeJournal) Spec() ai.ToolSpec {
	return ai.ToolSpec{
		Name:        "analyze_journal",
		Description: "사용자의 매매 일기 entries + 보유 자산 변화를 종합하여 매매 패턴·습관을 분석. 직접 매수/매도 권유 금지, 회고 관점만.",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"period_days": map[string]any{
					"type": "integer", "default": 90, "minimum": 7, "maximum": 365,
				},
			},
		},
	}
}

func (t *analyzeJournal) RequiresUserContext() bool { return true }

// journalEntryRow는 prompt 빌드에 사용하는 패키지 레벨 named type.
// Run 내부의 local type과 helper의 anonymous struct 사이 타입 불일치를 회피한다.
type journalEntryRow struct {
	EntryType string
	Action    *string
	Symbols   []string
	Content   string
	CreatedAt time.Time
	Symbol    string
}

func (t *analyzeJournal) Run(ctx context.Context, exec db.Executor, userID string, input map[string]any) (any, error) {
	days := 90
	if v, ok := input["period_days"].(float64); ok {
		days = int(v)
	}
	if days < 7 || days > 365 {
		days = 90
	}
	since := time.Now().AddDate(0, 0, -days)

	rows, err := exec.Query(ctx, `
		select je.entry_type, je.action, je.related_symbols, je.content, je.created_at,
		       coalesce(i.symbol, '') as symbol
		from public.journal_entries je
		left join public.holdings h on h.id = je.related_holding_id
		left join public.instruments i on i.id = h.instrument_id
		where je.user_id = $1 and je.created_at >= $2
		order by je.created_at
	`, userID, since)
	if err != nil {
		return map[string]any{"error": err.Error()}, nil
	}
	defer rows.Close()
	var entries []journalEntryRow
	for rows.Next() {
		var e journalEntryRow
		if err := rows.Scan(&e.EntryType, &e.Action, &e.Symbols, &e.Content, &e.CreatedAt, &e.Symbol); err != nil {
			return map[string]any{"error": err.Error()}, nil
		}
		entries = append(entries, e)
	}
	if len(entries) < 3 {
		return map[string]any{"error": "분석할 일기가 3개 이상 필요합니다"}, nil
	}

	// 보유 자산 — 에러 시 nil slice로 진행 (briefing 패턴과 동일)
	holdings, _ := loadHoldingsForJournal(ctx, exec, userID)

	system := buildJournalAnalysisPrompt(entries, holdings)

	req := ai.ChatRequest{
		Model:  ai.ModelSonnet,
		System: system,
		Messages: []ai.Message{
			{Role: ai.RoleUser, Content: fmt.Sprintf("최근 %d일 매매 일기를 분석해 3~5 bullet 한국어 마크다운으로 회고를 작성해주세요.", days)},
		},
		MaxTokens: 1024,
	}
	ch, err := t.Deps.Client.StreamChat(ctx, req)
	if err != nil {
		return map[string]any{"error": "AI 호출 실패: " + err.Error()}, nil
	}
	var content string
	for ev := range ch {
		if ev.Type == ai.EventToken {
			text, _ := ev.Data["text"].(string)
			content += text
		}
	}
	if content == "" {
		content = "분석 결과를 생성하지 못했습니다."
	}

	return map[string]any{
		"entries_count": len(entries),
		"content_md":    content,
		"model":         string(ai.ModelSonnet),
	}, nil
}

type holdingBrief struct {
	Symbol   string
	Quantity float64
	AvgCost  float64
}

func loadHoldingsForJournal(ctx context.Context, exec db.Executor, userID string) ([]holdingBrief, error) {
	rows, err := exec.Query(ctx, `
		select i.symbol, h.quantity::float8, h.avg_cost::float8
		from public.holdings h
		join public.instruments i on i.id = h.instrument_id
		where h.user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []holdingBrief
	for rows.Next() {
		var h holdingBrief
		if err := rows.Scan(&h.Symbol, &h.Quantity, &h.AvgCost); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, nil
}

func buildJournalAnalysisPrompt(entries []journalEntryRow, holdings []holdingBrief) string {
	base := `당신은 Quotient의 매매 일기 분석가입니다.

규칙:
- 직접 매수/매도 권유 금지. "회고 관점에서 ~를 살펴볼 수 있습니다" 표현.
- 추측 금지. 사용자가 직접 쓴 텍스트만 근거.
- 마크다운 3~5 bullet. 한국어. 각 bullet 100자 이내.
- 사용자 비난·평가 표현 금지. 중립·관찰 중심.

분석 관점:
- 매매 이유에 반복되는 키워드·논리
- 감정적 매매 가능성 신호 (변동성 직후 매도 등)
- 매매 일관성 vs 모순`

	var b strings.Builder
	b.WriteString(base)
	b.WriteString("\n\n[보유 자산 현황]\n")
	for _, h := range holdings {
		fmt.Fprintf(&b, "- %s: %.2f주 @ 평단 %.0f\n", h.Symbol, h.Quantity, h.AvgCost)
	}
	b.WriteString("\n[매매 일기 시계열]\n")
	for _, e := range entries {
		action := "관찰"
		if e.Action != nil {
			action = *e.Action
		}
		symbols := strings.Join(e.Symbols, ",")
		if symbols == "" && e.Symbol != "" {
			symbols = e.Symbol
		}
		extra := ""
		if symbols != "" {
			extra = " · " + symbols
		}
		fmt.Fprintf(&b, "- %s (%s · %s%s): %s\n",
			e.CreatedAt.Format("2006-01-02"), e.EntryType, action, extra,
			truncateRunes(e.Content, 200))
	}
	return b.String()
}

func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// RegisterJournal — main.go에서 호출
func RegisterJournal(r *Registry, d *Deps) {
	r.Register(&analyzeJournal{Deps: d})
}
