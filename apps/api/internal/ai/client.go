package ai

import "context"

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role      Role   `json:"role"`
	Content   string `json:"content"`
	ToolCalls []byte `json:"tool_calls,omitempty"`
}

type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

type EventType string

const (
	EventToken      EventType = "token"
	EventToolCall   EventType = "tool_call"
	EventToolResult EventType = "tool_result"
	EventDone       EventType = "done"
	EventError      EventType = "error"
)

type Event struct {
	Type EventType      `json:"type"`
	Data map[string]any `json:"data"`
}

type ChatRequest struct {
	Model       string
	System      string
	Messages    []Message
	Tools       []ToolSpec
	MaxTokens   int
	Temperature float64
}

type Client interface {
	StreamChat(ctx context.Context, req ChatRequest) (<-chan Event, error)
}
