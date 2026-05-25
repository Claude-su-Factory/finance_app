package ai

import "fmt"

const RecentWindow = 20

// BuildMessages는 전체 메시지 시퀀스를 최근 20개 + 이전 placeholder로 정리.
// placeholder는 첫 user 메시지로 삽입 (system 별도).
func BuildMessages(all []Message) []Message {
	if len(all) <= RecentWindow {
		return all
	}
	dropped := len(all) - RecentWindow
	placeholder := Message{
		Role:    RoleUser,
		Content: fmt.Sprintf("(이전 대화 %d개 — 컨텍스트 절약을 위해 생략됨)", dropped),
	}
	out := make([]Message, 0, RecentWindow+1)
	out = append(out, placeholder)
	out = append(out, all[dropped:]...)
	return out
}
