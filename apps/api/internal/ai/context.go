package ai

import (
	"context"
	"fmt"
	"strings"
)

const RecentWindow = 20

// BuildMessages는 전체 메시지 시퀀스를 최근 20개 + 이전 요약 1개로 정리.
// summarizer가 nil이면 placeholder 텍스트만 (Haiku 호출 없이 fallback).
// summarizer가 있으면 Haiku로 압축한 요약을 첫 user 메시지로 삽입.
func BuildMessages(ctx context.Context, all []Message, summarizer Client) []Message {
	if len(all) <= RecentWindow {
		return all
	}
	droppedN := len(all) - RecentWindow
	dropped := all[:droppedN]
	recent := all[droppedN:]

	var content string
	if summarizer != nil {
		content = summarizeWithHaiku(ctx, summarizer, dropped)
	}
	if content == "" {
		content = fmt.Sprintf("(이전 대화 %d개 — 컨텍스트 절약을 위해 생략됨)", droppedN)
	}

	out := make([]Message, 0, RecentWindow+1)
	out = append(out, Message{Role: RoleUser, Content: content})
	out = append(out, recent...)
	return out
}

// summarizeWithHaiku는 dropped 메시지들을 Haiku로 1턴 요약. 실패·빈 응답이면 빈 문자열.
// MVP는 단순 텍스트 직렬화 → 1턴 호출. tool_use/tool_result 블록은 직렬화에서 생략.
func summarizeWithHaiku(ctx context.Context, client Client, dropped []Message) string {
	if len(dropped) == 0 {
		return ""
	}
	var transcript strings.Builder
	for _, m := range dropped {
		if m.Content == "" {
			continue // tool 호출만 있는 메시지는 요약에서 생략
		}
		transcript.WriteString(string(m.Role))
		transcript.WriteString(": ")
		transcript.WriteString(m.Content)
		transcript.WriteString("\n\n")
	}
	if transcript.Len() == 0 {
		return ""
	}

	req := ChatRequest{
		Model:     ModelHaiku,
		System:    "다음 대화 기록을 한국어 3~5문장으로 요약하라. 핵심 주제·결정·미해결 질문만 포함하라. 도구 호출 세부는 생략하라.",
		Messages:  []Message{{Role: RoleUser, Content: transcript.String()}},
		MaxTokens: 512,
	}
	ch, err := client.StreamChat(ctx, req)
	if err != nil {
		return ""
	}
	var sb strings.Builder
	for ev := range ch {
		if ev.Type == EventToken {
			if text, ok := ev.Data["text"].(string); ok {
				sb.WriteString(text)
			}
		}
	}
	if sb.Len() == 0 {
		return ""
	}
	return fmt.Sprintf("(이전 대화 %d개 요약)\n%s", len(dropped), sb.String())
}
