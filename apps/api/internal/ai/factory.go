package ai

import "log/slog"

// New는 ANTHROPIC_API_KEY가 비어있으면 MockClient, 아니면 RealClient 반환.
func New(apiKey string) Client {
	if apiKey == "" {
		slog.Info("ai: ANTHROPIC_API_KEY empty — using MockClient")
		return NewMockClient()
	}
	slog.Info("ai: using RealClient")
	return NewRealClient(apiKey)
}
