package ai

import "testing"

func TestBuildMessages_UnderWindow(t *testing.T) {
	msgs := []Message{
		{Role: RoleUser, Content: "a"},
		{Role: RoleAssistant, Content: "b"},
	}
	got := BuildMessages(msgs)
	if len(got) != 2 {
		t.Errorf("got %d, want 2 (unchanged)", len(got))
	}
}

func TestBuildMessages_OverWindow(t *testing.T) {
	var msgs []Message
	for range 25 {
		msgs = append(msgs, Message{Role: RoleUser, Content: "msg"})
	}
	got := BuildMessages(msgs)
	if len(got) != RecentWindow+1 {
		t.Errorf("got %d, want %d", len(got), RecentWindow+1)
	}
	if got[0].Role != RoleUser {
		t.Errorf("placeholder must be user role")
	}
}
