package ai

import (
	"context"
	"strings"
	"testing"
)

func TestBuildMessages_UnderWindow(t *testing.T) {
	msgs := []Message{
		{Role: RoleUser, Content: "a"},
		{Role: RoleAssistant, Content: "b"},
	}
	got := BuildMessages(context.Background(), msgs, nil)
	if len(got) != 2 {
		t.Errorf("got %d, want 2 (unchanged)", len(got))
	}
}

func TestBuildMessages_OverWindow_NilSummarizer(t *testing.T) {
	var msgs []Message
	for range 25 {
		msgs = append(msgs, Message{Role: RoleUser, Content: "msg"})
	}
	got := BuildMessages(context.Background(), msgs, nil)
	if len(got) != RecentWindow+1 {
		t.Errorf("got %d, want %d", len(got), RecentWindow+1)
	}
	if got[0].Role != RoleUser {
		t.Errorf("placeholder must be user role")
	}
	if !strings.Contains(got[0].Content, "생략") {
		t.Errorf("placeholder should mention 생략, got: %q", got[0].Content)
	}
}

func TestBuildMessages_OverWindow_WithMockSummarizer(t *testing.T) {
	var msgs []Message
	for i := range 25 {
		role := RoleUser
		if i%2 == 1 {
			role = RoleAssistant
		}
		msgs = append(msgs, Message{Role: role, Content: "메시지 본문"})
	}
	mock := &MockClient{} // TokenDelay 0
	got := BuildMessages(context.Background(), msgs, mock)
	if len(got) != RecentWindow+1 {
		t.Errorf("got %d, want %d", len(got), RecentWindow+1)
	}
	// 요약 메시지에 "요약" 접두 포함
	if !strings.Contains(got[0].Content, "요약") {
		t.Errorf("summary should mention 요약, got: %q", got[0].Content)
	}
}
