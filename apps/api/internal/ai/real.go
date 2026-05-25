package ai

import (
	"context"
	"errors"
)

// RealClient는 anthropic-sdk-go 어댑터.
// MVP에서는 stub — 키 발급 후 실 구현 채움. claude-api 스킬 참고.
type RealClient struct {
	apiKey string
	// TODO(W4 후속): sdk client 보유 + tool 스키마 변환 캐시
}

func NewRealClient(apiKey string) *RealClient {
	return &RealClient{apiKey: apiKey}
}

func (c *RealClient) StreamChat(ctx context.Context, req ChatRequest) (<-chan Event, error) {
	// claude-api 스킬로 anthropic-sdk-go의 Messages.NewStreaming API 확인 후 실 구현 작성.
	// 핵심 단계:
	//   1) sdk.New(option.WithAPIKey(c.apiKey))
	//   2) tools := convertTools(req.Tools)
	//   3) msgs := convertMessages(req.Messages)
	//   4) stream := client.Messages.NewStreaming(ctx, MessageNewParams{Model, System, Messages, Tools, MaxTokens})
	//   5) for stream.Next(): 이벤트 타입 분기 → Event 채널 emit
	//   6) tool_use 발견 시 EventToolCall, content_block_delta → EventToken, message_stop → EventDone
	//   7) ctx.Done() 감시
	ch := make(chan Event, 1)
	ch <- Event{Type: EventError, Data: map[string]any{
		"message": "RealClient not yet implemented — set ANTHROPIC_API_KEY empty to use MockClient",
	}}
	close(ch)
	return ch, errors.New("RealClient stub: anthropic-sdk-go integration pending")
}
