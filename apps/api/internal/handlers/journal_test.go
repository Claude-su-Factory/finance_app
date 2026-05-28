package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/quotient/quotient/apps/api/internal/ai"
	"github.com/quotient/quotient/apps/api/internal/ai/tools"
	"github.com/quotient/quotient/apps/api/internal/db"
	"github.com/quotient/quotient/apps/api/internal/middleware"
	"github.com/quotient/quotient/apps/api/internal/models"
)

type fakeJournalRepo struct {
	entries           []models.JournalEntry
	createReturn      *models.JournalEntry
	updateReturn      *models.JournalEntry
	updateErr         error
	deleteErr         error
	analyses          []models.AnalysisRun
	insertAnalysis    *models.AnalysisRun
	hasAutoMonthly    bool
	lastCreateContent string // T6에서 holdings reason → auto entry 검증용
}

func (f *fakeJournalRepo) ListEntries(_ context.Context, _ db.Executor, _ string, _ int, _ *time.Time) ([]models.JournalEntry, bool, error) {
	return f.entries, false, nil
}
func (f *fakeJournalRepo) CreateEntry(_ context.Context, _ db.Executor, uid, entryType string, b models.JournalEntryCreate, holdingID *string) (*models.JournalEntry, error) {
	f.lastCreateContent = b.Content
	if f.createReturn != nil {
		return f.createReturn, nil
	}
	e := models.JournalEntry{ID: "new-id", UserID: uid, EntryType: entryType, Content: b.Content, RelatedSymbols: b.RelatedSymbols}
	if entryType == "manual" && b.Action == nil {
		a := "observation"
		e.Action = &a
	} else {
		e.Action = b.Action
	}
	return &e, nil
}
func (f *fakeJournalRepo) UpdateEntry(_ context.Context, _ db.Executor, _ string, _ string, _ models.JournalEntryPatch) (*models.JournalEntry, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	return f.updateReturn, nil
}
func (f *fakeJournalRepo) DeleteEntry(_ context.Context, _ db.Executor, _ string, _ string) error {
	return f.deleteErr
}
func (f *fakeJournalRepo) CountEntriesSince(_ context.Context, _ db.Executor, _ string, _ time.Time) (int, error) {
	return len(f.entries), nil
}
func (f *fakeJournalRepo) ListEntriesSince(_ context.Context, _ db.Executor, _ string, _ time.Time) ([]models.JournalEntry, error) {
	return f.entries, nil
}
func (f *fakeJournalRepo) InsertAnalysis(_ context.Context, _ db.Executor, uid, runType, ps, pe string, cnt int, md, model string) (*models.AnalysisRun, error) {
	if f.insertAnalysis != nil {
		return f.insertAnalysis, nil
	}
	return &models.AnalysisRun{ID: "a-1", UserID: uid, RunType: runType, PeriodStart: ps, PeriodEnd: pe, EntriesCount: cnt, ContentMD: md, Model: model}, nil
}
func (f *fakeJournalRepo) ListAnalyses(_ context.Context, _ db.Executor, _ string, _ int) ([]models.AnalysisRun, error) {
	return f.analyses, nil
}
func (f *fakeJournalRepo) HasAutoMonthlyForPeriod(_ context.Context, _ db.Executor, _, _ string) (bool, error) {
	return f.hasAutoMonthly, nil
}

func reqJ(method, path, body, uid string) *http.Request {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if uid != "" {
		r = r.WithContext(middleware.WithUserID(r.Context(), uid))
	}
	return r
}

func TestCreateEntry_Manual_OK(t *testing.T) {
	repo := &fakeJournalRepo{}
	h := NewJournalHandler(repo, &fakeChatRepo{}, nil, &ai.MockClient{}, tools.NewRegistry())
	w := httptest.NewRecorder()
	h.Create(w, reqJ(http.MethodPost, "/v1/journal/entries",
		`{"content":"관찰 내용 테스트","related_symbols":["005930"]}`, "user-1"))
	if w.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	var got models.JournalEntry
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.EntryType != "manual" {
		t.Errorf("entry_type=%s, want manual", got.EntryType)
	}
	if got.Action == nil || *got.Action != "observation" {
		t.Errorf("action default not applied: %v", got.Action)
	}
}

func TestCreateEntry_ContentTooLong_422(t *testing.T) {
	h := NewJournalHandler(&fakeJournalRepo{}, &fakeChatRepo{}, nil, &ai.MockClient{}, tools.NewRegistry())
	longContent := make([]byte, 2001)
	for i := range longContent {
		longContent[i] = 'a'
	}
	body, _ := json.Marshal(map[string]any{"content": string(longContent)})
	w := httptest.NewRecorder()
	h.Create(w, reqJ(http.MethodPost, "/v1/journal/entries", string(body), "user-1"))
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status %d, want 422", w.Code)
	}
}

func TestPatchEntry_AutoType_422(t *testing.T) {
	repo := &fakeJournalRepo{updateErr: ErrJournalEntryNotMutable}
	h := NewJournalHandler(repo, &fakeChatRepo{}, nil, &ai.MockClient{}, tools.NewRegistry())
	r := reqJ(http.MethodPatch, "/v1/journal/entries/x", `{"content":"수정"}`, "user-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "x")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	r = r.WithContext(middleware.WithUserID(r.Context(), "user-1"))
	w := httptest.NewRecorder()
	h.Patch(w, r)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", w.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	errBlock := got["error"].(map[string]any)
	if errBlock["code"] != "CANNOT_MODIFY_AUTO" {
		t.Errorf("code=%v", errBlock["code"])
	}
}

func TestDeleteEntry_NotFound(t *testing.T) {
	repo := &fakeJournalRepo{deleteErr: ErrJournalEntryNotFound}
	h := NewJournalHandler(repo, &fakeChatRepo{}, nil, &ai.MockClient{}, tools.NewRegistry())
	r := reqJ(http.MethodDelete, "/v1/journal/entries/x", "", "user-1")
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "x")
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	r = r.WithContext(middleware.WithUserID(r.Context(), "user-1"))
	w := httptest.NewRecorder()
	h.Delete(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("status %d", w.Code)
	}
}

func TestAnalyze_ExceededQuota_429(t *testing.T) {
	chat := &fakeChatRepo{limitErr: ErrUsageExceeded}
	h := NewJournalHandler(&fakeJournalRepo{}, chat, nil, &ai.MockClient{}, tools.NewRegistry())
	w := httptest.NewRecorder()
	h.Analyze(w, reqJ(http.MethodPost, "/v1/journal/analyze", `{"period_days":90}`, "user-1"))
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status %d, want 429", w.Code)
	}
}

func TestList_Empty_OK(t *testing.T) {
	h := NewJournalHandler(&fakeJournalRepo{}, &fakeChatRepo{}, nil, &ai.MockClient{}, tools.NewRegistry())
	w := httptest.NewRecorder()
	h.List(w, reqJ(http.MethodGet, "/v1/journal/entries", "", "user-1"))
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var got map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["has_more"] != false {
		t.Errorf("has_more=%v", got["has_more"])
	}
}

func TestListAnalyses_Empty_OK(t *testing.T) {
	h := NewJournalHandler(&fakeJournalRepo{}, &fakeChatRepo{}, nil, &ai.MockClient{}, tools.NewRegistry())
	w := httptest.NewRecorder()
	h.ListAnalyses(w, reqJ(http.MethodGet, "/v1/journal/analyses", "", "user-1"))
	if w.Code != http.StatusOK {
		t.Errorf("status %d", w.Code)
	}
}
