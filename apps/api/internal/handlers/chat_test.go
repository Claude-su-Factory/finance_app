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

func (f *fakeChatRepo) CreateSession(ctx context.Context, userID, title string) (*models.ChatSession, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	s := models.ChatSession{ID: "sess-1", UserID: userID, Title: title, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	f.sessions = append(f.sessions, s)
	return &s, nil
}
func (f *fakeChatRepo) ListSessions(ctx context.Context, userID string) ([]models.ChatSession, error) {
	return f.sessions, nil
}
func (f *fakeChatRepo) DeleteSession(ctx context.Context, userID, id string) error { return nil }
func (f *fakeChatRepo) ListMessages(ctx context.Context, userID, sid string) ([]models.ChatMessage, error) {
	return f.messages, nil
}
func (f *fakeChatRepo) AppendMessage(ctx context.Context, sid string, m models.ChatMessage) (*models.ChatMessage, error) {
	f.appendCnt++
	m.ID = fmt.Sprintf("msg-%d", f.appendCnt)
	m.SessionID = sid
	m.CreatedAt = time.Now()
	f.messages = append(f.messages, m)
	return &m, nil
}
func (f *fakeChatRepo) MarkFinished(ctx context.Context, id string) error { return nil }
func (f *fakeChatRepo) UnfinishedInSession(ctx context.Context, uid, sid string) (*models.ChatMessage, error) {
	return nil, nil
}
func (f *fakeChatRepo) GetUsage(ctx context.Context, uid string) (*models.ChatUsageMonthly, error) {
	return &f.usage, nil
}
func (f *fakeChatRepo) IncrementUsage(ctx context.Context, uid string, in, out int, opus bool) error {
	f.incCalls++
	return nil
}
func (f *fakeChatRepo) CheckLimits(ctx context.Context, uid string, opus bool) error {
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
	h := NewChatHandler(repo, mock, reg)

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

func TestStreamChat_LimitExceeded(t *testing.T) {
	repo := &fakeChatRepo{limitErr: ErrUsageExceeded}
	h := NewChatHandler(repo, &ai.MockClient{}, tools.NewRegistry())
	w := httptest.NewRecorder()
	h.StreamChat(w, chatReq(t, `{"message":"hi"}`, "user-1"))
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("got %d, want 429", w.Code)
	}
}

func TestStreamChat_NoAuth(t *testing.T) {
	repo := &fakeChatRepo{}
	h := NewChatHandler(repo, &ai.MockClient{}, tools.NewRegistry())
	w := httptest.NewRecorder()
	h.StreamChat(w, chatReq(t, `{"message":"hi"}`, ""))
	if w.Code != http.StatusUnauthorized {
		t.Errorf("got %d, want 401", w.Code)
	}
}

func TestListSessions_Empty(t *testing.T) {
	repo := &fakeChatRepo{}
	h := NewChatHandler(repo, &ai.MockClient{}, tools.NewRegistry())
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
