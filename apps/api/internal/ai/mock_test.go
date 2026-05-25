package ai

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestMockClient_TextOnly(t *testing.T) {
	m := &MockClient{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ch, err := m.StreamChat(ctx, ChatRequest{
		Model:    ModelSonnet,
		Messages: []Message{{Role: RoleUser, Content: "안녕"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var tokens int
	var doneSeen bool
	for ev := range ch {
		if ev.Type == EventToken {
			tokens++
		}
		if ev.Type == EventDone {
			doneSeen = true
		}
	}
	if tokens == 0 {
		t.Error("expected at least 1 token event")
	}
	if !doneSeen {
		t.Error("expected EventDone")
	}
}

func TestMockClient_ToolCall_Portfolio(t *testing.T) {
	m := &MockClient{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ch, err := m.StreamChat(ctx, ChatRequest{
		Model:    ModelSonnet,
		Messages: []Message{{Role: RoleUser, Content: "내 포트폴리오 어때?"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var toolName string
	for ev := range ch {
		if ev.Type == EventToolCall {
			toolName, _ = ev.Data["name"].(string)
		}
	}
	if toolName != "get_portfolio" {
		t.Errorf("got tool %q want get_portfolio", toolName)
	}
}

func TestMockClient_ToolResultThenText(t *testing.T) {
	m := &MockClient{}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ch, err := m.StreamChat(ctx, ChatRequest{
		Model: ModelSonnet,
		Messages: []Message{
			{Role: RoleUser, Content: "내 포트폴리오 어때?"},
			{Role: RoleAssistant, ToolCalls: []byte(`[{"type":"tool_use","id":"call_1","name":"get_portfolio"}]`)},
			{Role: RoleUser, ToolCalls: []byte(`[{"type":"tool_result","tool_use_id":"call_1","content":"{...}"}]`)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var text strings.Builder
	for ev := range ch {
		if ev.Type == EventToken {
			text.WriteString(ev.Data["text"].(string))
		}
	}
	if text.Len() == 0 {
		t.Error("expected text response after tool result")
	}
	if !strings.Contains(text.String(), "데이터 기준") {
		t.Errorf("expected disclaimer in response, got: %s", text.String())
	}
}

func TestMockClient_CtxCancel(t *testing.T) {
	m := &MockClient{TokenDelay: 100 * time.Millisecond}
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := m.StreamChat(ctx, ChatRequest{
		Model:    ModelSonnet,
		Messages: []Message{{Role: RoleUser, Content: "긴 응답"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	<-ch
	cancel()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		default:
		}
	}
	t.Error("channel did not close after context cancel")
}
