package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/quotient/quotient/apps/api/internal/ai"
	"github.com/quotient/quotient/apps/api/internal/ai/tools"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
)

const MaxToolDepth = 8 // spec §10-10

const systemPrompt = `당신은 Quotient의 분석가입니다. 사용자의 보유 자산·관심 종목·시장 데이터를 도구로 조회하여 분석 관점을 제공합니다.

규칙:
- 직접 매수/매도 권유 금지. "분석 관점에서 ~를 살펴볼 수 있습니다" 형태로 표현.
- 모든 수치는 도구 호출 결과를 근거로. 추측 금지.
- 응답 끝에 "(데이터 기준: YYYY-MM-DD HH:MM KST, 시세 지연 15분)" 자동 부착.
- 세금·법무 자문 요청 → "전문가에게 문의하세요" 안내.`

type ChatHandler struct {
	repo     ChatRepo
	client   ai.Client
	registry *tools.Registry
}

func NewChatHandler(repo ChatRepo, client ai.Client, registry *tools.Registry) *ChatHandler {
	return &ChatHandler{repo: repo, client: client, registry: registry}
}

// POST /v1/chat — SSE 스트리밍
func (h *ChatHandler) StreamChat(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		http.Error(w, "unauthenticated", http.StatusUnauthorized)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	var body struct {
		SessionID string `json:"session_id,omitempty"`
		Message   string `json:"message"`
		UseOpus   bool   `json:"use_opus,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if body.Message == "" {
		http.Error(w, "message required", http.StatusUnprocessableEntity)
		return
	}

	// 한도 체크
	if err := h.repo.CheckLimits(r.Context(), uid, body.UseOpus); err != nil {
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return
	}

	// 세션 보장
	if body.SessionID == "" {
		title := truncateRunes(body.Message, 50)
		s, err := h.repo.CreateSession(r.Context(), uid, title)
		if err != nil {
			slog.Error("create session failed", "err", err)
			http.Error(w, "create session failed", http.StatusInternalServerError)
			return
		}
		body.SessionID = s.ID
	}

	// 기존 메시지 로드
	history, err := h.repo.ListMessages(r.Context(), uid, body.SessionID)
	if err != nil {
		if errors.Is(err, ErrChatSessionNotFound) {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		slog.Error("list messages failed", "err", err)
		http.Error(w, "load failed", http.StatusInternalServerError)
		return
	}

	// 사용자 메시지 영속화
	_, err = h.repo.AppendMessage(r.Context(), body.SessionID, models.ChatMessage{
		Role: "user", Content: body.Message,
		FinishedAt: ptrTime(time.Now()),
	})
	if err != nil {
		slog.Error("append user message failed", "err", err)
		http.Error(w, "save failed", http.StatusInternalServerError)
		return
	}

	// SSE 헤더
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// ai 메시지 시퀀스 구성
	aiMsgs := historyToAI(history)
	aiMsgs = append(aiMsgs, ai.Message{Role: ai.RoleUser, Content: body.Message})

	model := ai.ModelSonnet
	if body.UseOpus {
		model = ai.ModelOpus
	}

	totalInput, totalOutput := 0, 0
	var lastSavedID string

	// Tool routing loop
	for depth := range MaxToolDepth {
		windowed := ai.BuildMessages(aiMsgs)
		req := ai.ChatRequest{
			Model:     model,
			System:    systemPrompt,
			Messages:  windowed,
			Tools:     h.registry.Specs(),
			MaxTokens: 4096,
		}
		ch, err := h.client.StreamChat(r.Context(), req)
		if err != nil {
			writeSSE(w, flusher, "error", map[string]any{"message": err.Error()})
			return
		}

		var turnText string
		var turnToolUses []map[string]any
		var turnToolResults []map[string]any
		finishReason := ""
		for ev := range ch {
			if r.Context().Err() != nil {
				goto persist
			}
			switch ev.Type {
			case ai.EventToken:
				text, _ := ev.Data["text"].(string)
				turnText += text
				writeSSE(w, flusher, "token", ev.Data)
			case ai.EventToolCall:
				writeSSE(w, flusher, "tool_call", ev.Data)
				name, _ := ev.Data["name"].(string)
				input, _ := ev.Data["input"].(map[string]any)
				id, _ := ev.Data["id"].(string)
				turnToolUses = append(turnToolUses, map[string]any{
					"type": "tool_use", "id": id, "name": name, "input": input,
				})
				// 즉시 실행
				result := tools.ExecuteAndSerialize(r.Context(), h.registry, name, uid, input)
				writeSSE(w, flusher, "tool_result", map[string]any{
					"id": id, "name": name, "result": result,
				})
				turnToolResults = append(turnToolResults, map[string]any{
					"type": "tool_result", "tool_use_id": id, "content": result,
				})
			case ai.EventDone:
				if v, ok := ev.Data["input_tokens"].(int); ok {
					totalInput += v
				}
				if v, ok := ev.Data["output_tokens"].(int); ok {
					totalOutput += v
				}
				if v, ok := ev.Data["finish_reason"].(string); ok {
					finishReason = v
				}
			case ai.EventError:
				writeSSE(w, flusher, "error", ev.Data)
				return
			}
		}

		// turn 종료 — assistant 메시지 1개 영속화
		if turnText != "" || len(turnToolUses) > 0 {
			var blocks []map[string]any
			if turnText != "" {
				blocks = append(blocks, map[string]any{"type": "text", "text": turnText})
			}
			for _, tu := range turnToolUses {
				blocks = append(blocks, tu)
			}
			blocksJSON, _ := json.Marshal(blocks)

			aiMsgs = append(aiMsgs, ai.Message{
				Role:      ai.RoleAssistant,
				Content:   turnText,
				ToolCalls: blocksJSON,
			})

			persistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			saved, err := h.repo.AppendMessage(persistCtx, body.SessionID, models.ChatMessage{
				Role:         "assistant",
				Content:      turnText,
				ToolCalls:    blocksJSON,
				InputTokens:  totalInput,
				OutputTokens: totalOutput,
				Model:        &model,
				FinishedAt:   ptrTime(time.Now()),
			})
			cancel()
			if err != nil {
				slog.Error("append assistant turn failed", "err", err)
			} else if saved != nil {
				lastSavedID = saved.ID
			}
		}

		// 도구 호출 없으면 응답 완료
		if len(turnToolUses) == 0 {
			break
		}

		// tool_result를 다음 user 메시지 1개로 묶어 추가
		resultsJSON, _ := json.Marshal(turnToolResults)
		aiMsgs = append(aiMsgs, ai.Message{
			Role:      ai.RoleUser,
			ToolCalls: resultsJSON,
		})

		// tool 결과도 DB row로 영속화 (감사·디버깅용 — role='tool')
		toolPersistCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _ = h.repo.AppendMessage(toolPersistCtx, body.SessionID, models.ChatMessage{
			Role:       "tool",
			ToolCalls:  resultsJSON,
			FinishedAt: ptrTime(time.Now()),
		})
		cancel()

		if depth == MaxToolDepth-1 {
			writeSSE(w, flusher, "error", map[string]any{
				"message": "복잡한 질문이라 충분히 답하지 못했습니다. 더 좁혀서 다시 물어주세요.",
			})
			break
		}
		_ = finishReason // 미사용 (loop break 결정은 turnToolUses len이 더 정확)
	}

persist:
	// 사용량 갱신
	usageCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := h.repo.IncrementUsage(usageCtx, uid, totalInput, totalOutput, body.UseOpus); err != nil {
		slog.Error("usage increment failed", "err", err)
	}
	cancel()

	writeSSE(w, flusher, "done", map[string]any{
		"session_id":    body.SessionID,
		"message_id":    lastSavedID,
		"input_tokens":  totalInput,
		"output_tokens": totalOutput,
	})
}

// GET /v1/chat/sessions
func (h *ChatHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	sessions, err := h.repo.ListSessions(r.Context(), uid)
	if err != nil {
		slog.Error("list sessions failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if sessions == nil {
		sessions = []models.ChatSession{}
	}
	writeJSON(w, http.StatusOK, sessions)
}

// DELETE /v1/chat/sessions/{id}
func (h *ChatHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.repo.DeleteSession(r.Context(), uid, id); err != nil {
		if errors.Is(err, ErrChatSessionNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "session not found")
			return
		}
		slog.Error("delete session failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "delete failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// GET /v1/chat/sessions/{id}/messages
func (h *ChatHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	id := chi.URLParam(r, "id")
	msgs, err := h.repo.ListMessages(r.Context(), uid, id)
	if err != nil {
		if errors.Is(err, ErrChatSessionNotFound) {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "session not found")
			return
		}
		slog.Error("list messages failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if msgs == nil {
		msgs = []models.ChatMessage{}
	}
	writeJSON(w, http.StatusOK, msgs)
}

// GET /v1/chat/sessions/{id}/unfinished
func (h *ChatHandler) GetUnfinished(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	id := chi.URLParam(r, "id")
	m, err := h.repo.UnfinishedInSession(r.Context(), uid, id)
	if err != nil {
		slog.Error("unfinished load failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	if m == nil {
		writeJSON(w, http.StatusOK, map[string]any{"unfinished": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"unfinished": m})
}

// GET /v1/chat/usage
func (h *ChatHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	uid := middleware.UserID(r.Context())
	if uid == "" {
		writeError(w, http.StatusUnauthorized, "UNAUTHENTICATED", "no user")
		return
	}
	u, err := h.repo.GetUsage(r.Context(), uid)
	if err != nil {
		slog.Error("get usage failed", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "load failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"usage": u,
		"limits": map[string]int{
			"chat":          LimitChatPerMonth,
			"input_tokens":  LimitInputTokens,
			"output_tokens": LimitOutputTokens,
			"opus":          LimitOpusPerMonth,
		},
	})
}

func writeSSE(w http.ResponseWriter, f http.Flusher, event string, data map[string]any) {
	b, _ := json.Marshal(data)
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
	f.Flush()
}

func historyToAI(history []models.ChatMessage) []ai.Message {
	out := make([]ai.Message, 0, len(history))
	for _, m := range history {
		out = append(out, ai.Message{
			Role:      ai.Role(m.Role),
			Content:   m.Content,
			ToolCalls: []byte(m.ToolCalls),
		})
	}
	return out
}

func ptrTime(t time.Time) *time.Time { return &t }

// truncateRunes는 유니코드 룬 기준으로 문자열을 n개로 자른다.
// ai/mock.go의 truncate와 동일 로직이나 패키지가 달라 별도 정의.
func truncateRunes(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
