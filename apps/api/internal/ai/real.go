package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// RealClient는 anthropic-sdk-go 어댑터.
// SDK 버전: v1.45.0+. Sonnet 4.6 / Opus 4.7 / Haiku 4.5 지원.
// spec §5 모델 라우팅 + §10-7 SSE 끊김 처리 + prompt caching 의무.
type RealClient struct {
	client anthropic.Client
}

func NewRealClient(apiKey string) *RealClient {
	return &RealClient{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
	}
}

func (c *RealClient) StreamChat(ctx context.Context, req ChatRequest) (<-chan Event, error) {
	params, err := buildMessageParams(req)
	if err != nil {
		return nil, fmt.Errorf("build params: %w", err)
	}

	stream := c.client.Messages.NewStreaming(ctx, params)

	ch := make(chan Event, 32)
	go func() {
		defer close(ch)
		emit := func(e Event) {
			select {
			case ch <- e:
			case <-ctx.Done():
			}
		}

		// Tool use 누적 상태 — content_block_start(tool_use)에서 시작,
		// content_block_delta(input_json_delta)로 누적, content_block_stop에서 emit.
		type pendingTool struct {
			id, name string
			input    strings.Builder
		}
		var current *pendingTool

		var inputTokens, outputTokens int
		var finishReason string

		for stream.Next() {
			if ctx.Err() != nil {
				return
			}
			event := stream.Current()
			switch ev := event.AsAny().(type) {
			case anthropic.MessageStartEvent:
				if v := ev.Message.Usage.InputTokens; v > 0 {
					inputTokens = int(v)
				}
			case anthropic.ContentBlockStartEvent:
				if tb, ok := ev.ContentBlock.AsAny().(anthropic.ToolUseBlock); ok {
					current = &pendingTool{id: tb.ID, name: tb.Name}
				}
			case anthropic.ContentBlockDeltaEvent:
				switch delta := ev.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					emit(Event{Type: EventToken, Data: map[string]any{"text": delta.Text}})
				case anthropic.InputJSONDelta:
					if current != nil {
						current.input.WriteString(delta.PartialJSON)
					}
				}
			case anthropic.ContentBlockStopEvent:
				if current != nil {
					input := map[string]any{}
					raw := current.input.String()
					if raw != "" {
						_ = json.Unmarshal([]byte(raw), &input)
					}
					emit(Event{Type: EventToolCall, Data: map[string]any{
						"id":    current.id,
						"name":  current.name,
						"input": input,
					}})
					current = nil
				}
			case anthropic.MessageDeltaEvent:
				if v := ev.Usage.OutputTokens; v > 0 {
					outputTokens = int(v)
				}
				if sr := string(ev.Delta.StopReason); sr != "" {
					finishReason = sr
				}
			}
		}

		if err := stream.Err(); err != nil {
			emit(Event{Type: EventError, Data: map[string]any{"message": err.Error()}})
			return
		}

		emit(Event{Type: EventDone, Data: map[string]any{
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
			"finish_reason": finishReason,
		}})
	}()

	return ch, nil
}

// buildMessageParams converts our ChatRequest to anthropic.MessageNewParams.
// - System: prompt caching 적용 (spec §5 의무)
// - Messages: text + tool_use/tool_result content blocks
// - Tools: ToolSpec → ToolUnionParam
// - Thinking: adaptive (4.6/4.7 권장)
func buildMessageParams(req ChatRequest) (anthropic.MessageNewParams, error) {
	maxTokens := int64(req.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 4096
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		MaxTokens: maxTokens,
	}

	// System with prompt caching (spec §5)
	if req.System != "" {
		params.System = []anthropic.TextBlockParam{{
			Text:         req.System,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}}
	}

	// Tools — 마지막 도구에 cache_control로 tools+system 묶어서 캐시
	if len(req.Tools) > 0 {
		tools := make([]anthropic.ToolUnionParam, 0, len(req.Tools))
		for i, t := range req.Tools {
			tp := anthropic.ToolParam{
				Name:        t.Name,
				Description: anthropic.String(t.Description),
				InputSchema: toolInputSchemaFromSpec(t.InputSchema),
			}
			// 마지막 도구에 cache_control
			if i == len(req.Tools)-1 {
				tp.CacheControl = anthropic.NewCacheControlEphemeralParam()
			}
			tools = append(tools, anthropic.ToolUnionParam{OfTool: &tp})
		}
		params.Tools = tools
	}

	// Messages
	msgs, err := convertMessages(req.Messages)
	if err != nil {
		return params, err
	}
	params.Messages = msgs

	// Adaptive thinking (Sonnet 4.6 / Opus 4.7 권장 — 자동 활성)
	if supportsAdaptiveThinking(req.Model) {
		params.Thinking = anthropic.ThinkingConfigParamUnion{
			OfAdaptive: &anthropic.ThinkingConfigAdaptiveParam{},
		}
	}

	return params, nil
}

// toolInputSchemaFromSpec converts our spec map to SDK ToolInputSchemaParam.
// Our InputSchema follows JSON Schema: { type, properties, required? }.
func toolInputSchemaFromSpec(schema map[string]any) anthropic.ToolInputSchemaParam {
	out := anthropic.ToolInputSchemaParam{}
	if props, ok := schema["properties"].(map[string]any); ok {
		out.Properties = props
	}
	if req, ok := schema["required"].([]string); ok {
		out.Required = req
	} else if rawReq, ok := schema["required"].([]any); ok {
		// JSON unmarshal 시 []any로 올 수 있음
		req := make([]string, 0, len(rawReq))
		for _, r := range rawReq {
			if s, ok := r.(string); ok {
				req = append(req, s)
			}
		}
		out.Required = req
	}
	return out
}

// convertMessages converts our ai.Message slice to SDK MessageParam slice.
// 우리 Message는 ToolCalls(json.RawMessage)에 tool_use/tool_result 블록 배열을 저장.
// SDK는 content blocks의 union (text / tool_use / tool_result)를 받음.
func convertMessages(in []Message) ([]anthropic.MessageParam, error) {
	out := make([]anthropic.MessageParam, 0, len(in))
	for _, m := range in {
		blocks, err := messageToContentBlocks(m)
		if err != nil {
			return nil, fmt.Errorf("convert message: %w", err)
		}
		if len(blocks) == 0 {
			continue
		}
		switch m.Role {
		case RoleUser, RoleTool:
			// RoleTool은 user role + tool_result content로 표현
			out = append(out, anthropic.NewUserMessage(blocks...))
		case RoleAssistant:
			out = append(out, anthropic.NewAssistantMessage(blocks...))
		default:
			return nil, fmt.Errorf("unknown role: %s", m.Role)
		}
	}
	return out, nil
}

// messageToContentBlocks builds SDK content blocks from our ai.Message.
// Content 텍스트와 ToolCalls(blocks JSON 배열) 두 소스를 머지.
func messageToContentBlocks(m Message) ([]anthropic.ContentBlockParamUnion, error) {
	var out []anthropic.ContentBlockParamUnion

	// 1) 텍스트 본문
	if m.Content != "" {
		out = append(out, anthropic.NewTextBlock(m.Content))
	}

	// 2) ToolCalls JSON (tool_use 또는 tool_result 블록 배열)
	if len(m.ToolCalls) > 0 {
		var blocks []map[string]any
		if err := json.Unmarshal(m.ToolCalls, &blocks); err != nil {
			return nil, fmt.Errorf("unmarshal tool_calls: %w", err)
		}
		for _, b := range blocks {
			bt, _ := b["type"].(string)
			switch bt {
			case "tool_use":
				id, _ := b["id"].(string)
				name, _ := b["name"].(string)
				input, _ := b["input"].(map[string]any)
				inputJSON, _ := json.Marshal(input)
				out = append(out, anthropic.ContentBlockParamUnion{
					OfToolUse: &anthropic.ToolUseBlockParam{
						ID:    id,
						Name:  name,
						Input: json.RawMessage(inputJSON),
					},
				})
			case "tool_result":
				toolUseID, _ := b["tool_use_id"].(string)
				content, _ := b["content"].(string)
				out = append(out, anthropic.NewToolResultBlock(toolUseID, content, false))
			}
		}
	}

	return out, nil
}

// supportsAdaptiveThinking returns true for models that support adaptive thinking.
// Sonnet 4.6, Opus 4.7 — claude-api 스킬 기준. Haiku 4.5는 미지원.
func supportsAdaptiveThinking(model string) bool {
	return strings.Contains(model, "sonnet-4-6") ||
		strings.Contains(model, "opus-4-7") ||
		strings.Contains(model, "opus-4-6")
}
