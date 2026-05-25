package ai

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// MockClient는 실 API 없이 SSE/tool routing 흐름을 시뮬레이션.
// 응답 시나리오는 마지막 user 메시지 내용 기준.
type MockClient struct {
	TokenDelay time.Duration // 테스트에서 0으로 설정
}

func NewMockClient() *MockClient {
	return &MockClient{TokenDelay: 30 * time.Millisecond}
}

func (m *MockClient) StreamChat(ctx context.Context, req ChatRequest) (<-chan Event, error) {
	ch := make(chan Event, 32)
	go func() {
		defer close(ch)
		// 마지막 user 메시지 추출 (tool_result가 아닌 일반 user 메시지)
		var lastUserMsg string
		for i := len(req.Messages) - 1; i >= 0; i-- {
			if req.Messages[i].Role == RoleUser && len(req.Messages[i].ToolCalls) == 0 {
				lastUserMsg = req.Messages[i].Content
				break
			}
		}
		// 도구 결과가 마지막이면 (2턴째) → 텍스트 응답
		toolResultLast := false
		if len(req.Messages) > 0 {
			last := req.Messages[len(req.Messages)-1]
			if last.Role == RoleUser && len(last.ToolCalls) > 0 {
				toolResultLast = true
			}
		}

		if !toolResultLast {
			// 1턴: 키워드 → 도구 시나리오 매핑
			scenarios := []struct {
				keywords []string
				tool     string
				input    map[string]any
			}{
				{[]string{"포트폴리오", "보유"}, "get_portfolio", map[string]any{}},
				{[]string{"환율", "시장", "지수"}, "get_market_overview", map[string]any{}},
				{[]string{"관심"}, "get_watchlist", map[string]any{}},
				{[]string{"검색", "찾아"}, "search_instrument", map[string]any{"query": "삼성"}},
				{[]string{"시세", "가격"}, "get_quote", map[string]any{"symbol": "KOSPI"}},
				{[]string{"차트", "히스토리", "추이"}, "get_price_history", map[string]any{"symbol": "KOSPI", "range": "1mo"}},
				{[]string{"금리", "지표"}, "get_economic_indicator", map[string]any{"code": "DFF"}},
				{[]string{"집중도", "분산", "분석"}, "calc_portfolio_metrics", map[string]any{}},
			}
			for _, s := range scenarios {
				for _, kw := range s.keywords {
					if strings.Contains(lastUserMsg, kw) {
						m.emitToolCall(ctx, ch, "call_1", s.tool, s.input)
						return
					}
				}
			}
		}

		// 2턴 또는 도구 무관 응답: 텍스트 스트리밍
		text := m.composeReply(lastUserMsg, req.Messages)
		m.streamText(ctx, ch, text)
		m.emit(ctx, ch, Event{Type: EventDone, Data: map[string]any{
			"input_tokens":  estimateTokens(req),
			"output_tokens": len(text) / 4,
			"finish_reason": "end_turn",
		}})
	}()
	return ch, nil
}

func (m *MockClient) emit(ctx context.Context, ch chan<- Event, e Event) {
	select {
	case ch <- e:
	case <-ctx.Done():
	}
}

func (m *MockClient) emitToolCall(ctx context.Context, ch chan<- Event, id, name string, input map[string]any) {
	m.emit(ctx, ch, Event{Type: EventToolCall, Data: map[string]any{
		"id": id, "name": name, "input": input,
	}})
	m.emit(ctx, ch, Event{Type: EventDone, Data: map[string]any{
		"input_tokens":  100,
		"output_tokens": 20,
		"finish_reason": "tool_use",
	}})
}

func (m *MockClient) streamText(ctx context.Context, ch chan<- Event, text string) {
	for _, tok := range tokenize(text) {
		m.emit(ctx, ch, Event{Type: EventToken, Data: map[string]any{"text": tok}})
		if m.TokenDelay > 0 {
			select {
			case <-time.After(m.TokenDelay):
			case <-ctx.Done():
				return
			}
		}
	}
}

func (m *MockClient) composeReply(userMsg string, _ []Message) string {
	stamp := time.Now().Format("2006-01-02 15:04")
	return fmt.Sprintf(
		"질문 \"%s\"에 대한 분석입니다.\n\n"+
			"보유 자산과 최근 시세를 기준으로 보면, 변동성이 큰 종목과 안정적 종목의 비중을 점검할 시점입니다. "+
			"분산 효과를 위해 자산군별 비중을 재확인해보실 수 있습니다.\n\n"+
			"(데이터 기준: %s KST, 시세 지연 15분)",
		truncate(userMsg, 60), stamp,
	)
}

func tokenize(s string) []string {
	parts := strings.SplitAfter(s, " ")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

func estimateTokens(req ChatRequest) int {
	total := len(req.System) / 4
	for _, m := range req.Messages {
		total += len(m.Content) / 4
	}
	return total
}
