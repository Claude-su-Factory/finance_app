package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/quotient/quotient/apps/api/internal/ai"
	"github.com/quotient/quotient/apps/api/internal/ai/tools"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
)

type fakeChatRepo struct {
	sessions  []models.ChatSession
	messages  []models.ChatMessage
	usage     models.ChatUsageMonthly
	limitErr  error
	createErr error
	appendCnt int
	incCalls  int
}

func (f *fakeChatRepo) CreateSession(ctx context.Context, exec db.Executor, userID, title string) (*models.ChatSession, error) {
	_ = exec
	if f.createErr != nil {
		return nil, f.createErr
	}
	s := models.ChatSession{ID: "sess-1", UserID: userID, Title: title, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	f.sessions = append(f.sessions, s)
	return &s, nil
}
func (f *fakeChatRepo) ListSessions(ctx context.Context, exec db.Executor, userID string) ([]models.ChatSession, error) {
	_ = exec
	return f.sessions, nil
}
func (f *fakeChatRepo) DeleteSession(ctx context.Context, exec db.Executor, userID, id string) error {
	_ = exec
	return nil
}
func (f *fakeChatRepo) ListMessages(ctx context.Context, exec db.Executor, userID, sid string) ([]models.ChatMessage, error) {
	_ = exec
	return f.messages, nil
}
func (f *fakeChatRepo) AppendMessage(ctx context.Context, exec db.Executor, sid string, m models.ChatMessage) (*models.ChatMessage, error) {
	_ = exec
	f.appendCnt++
	m.ID = fmt.Sprintf("msg-%d", f.appendCnt)
	m.SessionID = sid
	m.CreatedAt = time.Now()
	f.messages = append(f.messages, m)
	return &m, nil
}
func (f *fakeChatRepo) MarkFinished(ctx context.Context, exec db.Executor, id string) error {
	_ = exec
	return nil
}
func (f *fakeChatRepo) UnfinishedInSession(ctx context.Context, exec db.Executor, uid, sid string) (*models.ChatMessage, error) {
	_ = exec
	return nil, nil
}
func (f *fakeChatRepo) GetUsage(ctx context.Context, exec db.Executor, uid string) (*models.ChatUsageMonthly, error) {
	_ = exec
	return &f.usage, nil
}
func (f *fakeChatRepo) IncrementUsage(ctx context.Context, exec db.Executor, uid string, in, out int, opus bool) error {
	_ = exec
	f.incCalls++
	return nil
}
func (f *fakeChatRepo) CheckLimits(ctx context.Context, exec db.Executor, uid string, opus bool) error {
	_ = exec
	return f.limitErr
}

func chatReq(t *testing.T, body string, uid string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/v1/chat", bytes.NewBufferString(body))
	if uid != "" {
		r = r.WithContext(middleware.WithUserID(r.Context(), uid))
	}
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestStreamChat_TextOnly(t *testing.T) {
	repo := &fakeChatRepo{}
	reg := tools.NewRegistry()
	mock := &ai.MockClient{}
	h := NewChatHandler(repo, nil, mock, reg)

	w := httptest.NewRecorder()
	h.StreamChat(w, chatReq(t, `{"message":"안녕"}`, "user-1"))

	if w.Code != http.StatusOK {
		t.Fatalf("status %d, body=%s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "event: token") {
		t.Errorf("expected token events, got: %s", body)
	}
	if !strings.Contains(body, "event: done") {
		t.Errorf("expected done event")
	}
	if !strings.Contains(body, `"session_id":"sess-1"`) {
		t.Errorf("done must include session_id")
	}
	if repo.incCalls != 1 {
		t.Errorf("expected 1 usage increment, got %d", repo.incCalls)
	}
	if repo.appendCnt < 2 {
		t.Errorf("expected user + assistant messages, got %d", repo.appendCnt)
	}
}

// collectTokenText는 SSE token 이벤트의 text를 이어 붙인다 — 프런트가 받는 최종 문자열.
// (개별 token은 공백 단위로 쪼개지므로 raw body 부분문자열 검사로는 footer를 못 잡는다.)
func collectTokenText(t *testing.T, body string) string {
	t.Helper()
	var sb strings.Builder
	for line := range strings.SplitSeq(body, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var ev struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &ev); err != nil {
			continue
		}
		sb.WriteString(ev.Text)
	}
	return sb.String()
}

// 개념 질문(도구 미사용)에는 데이터 footer를 붙이지 않는다 — 교육자 역할.
func TestStreamChat_ConceptNoFooter(t *testing.T) {
	repo := &fakeChatRepo{}
	h := NewChatHandler(repo, nil, &ai.MockClient{}, tools.NewRegistry())

	w := httptest.NewRecorder()
	h.StreamChat(w, chatReq(t, `{"message":"PER이 뭐야?"}`, "user-1"))

	if w.Code != http.StatusOK {
		t.Fatalf("status %d, body=%s", w.Code, w.Body.String())
	}
	if text := collectTokenText(t, w.Body.String()); strings.Contains(text, "데이터 기준") {
		t.Errorf("개념 질문에 데이터 footer가 붙음: %s", text)
	}
}

// 시세·보유 데이터를 쓴 답변에는 footer를 부착한다 (spec §5).
func TestStreamChat_DataAnswerHasFooter(t *testing.T) {
	repo := &fakeChatRepo{}
	h := NewChatHandler(repo, nil, &ai.MockClient{}, tools.NewRegistry())

	w := httptest.NewRecorder()
	h.StreamChat(w, chatReq(t, `{"message":"내 포트폴리오 분석해줘"}`, "user-1"))

	if w.Code != http.StatusOK {
		t.Fatalf("status %d, body=%s", w.Code, w.Body.String())
	}
	if text := collectTokenText(t, w.Body.String()); !strings.Contains(text, "데이터 기준") {
		t.Errorf("데이터 답변에 footer가 없음: %s", text)
	}
}

func TestStreamChat_LimitExceeded(t *testing.T) {
	repo := &fakeChatRepo{limitErr: ErrUsageExceeded}
	h := NewChatHandler(repo, nil, &ai.MockClient{}, tools.NewRegistry())
	w := httptest.NewRecorder()
	h.StreamChat(w, chatReq(t, `{"message":"hi"}`, "user-1"))
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("got %d, want 429", w.Code)
	}
}

func TestStreamChat_NoAuth(t *testing.T) {
	repo := &fakeChatRepo{}
	h := NewChatHandler(repo, nil, &ai.MockClient{}, tools.NewRegistry())
	w := httptest.NewRecorder()
	h.StreamChat(w, chatReq(t, `{"message":"hi"}`, ""))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", w.Code)
	}
}

func TestListSessions_Empty(t *testing.T) {
	repo := &fakeChatRepo{}
	h := NewChatHandler(repo, nil, &ai.MockClient{}, tools.NewRegistry())
	r := httptest.NewRequest(http.MethodGet, "/v1/chat/sessions", nil)
	r = r.WithContext(middleware.WithUserID(r.Context(), "user-1"))
	w := httptest.NewRecorder()
	h.ListSessions(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("got %d", w.Code)
	}
	var got []models.ChatSession
	_ = json.NewDecoder(w.Body).Decode(&got)
	if len(got) != 0 {
		t.Errorf("got %d sessions, want 0", len(got))
	}
}
