package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeInstrRepo struct {
	aliasResults []SearchResult
	textResults  []SearchResult
	learnErr     error
	lastLearned  struct{ alias, id string }
}

func (f *fakeInstrRepo) SearchByAlias(ctx context.Context, q string) ([]SearchResult, error) {
	return f.aliasResults, nil
}
func (f *fakeInstrRepo) SearchByText(ctx context.Context, q string) ([]SearchResult, error) {
	return f.textResults, nil
}
func (f *fakeInstrRepo) LearnAlias(ctx context.Context, alias, id string) error {
	f.lastLearned.alias = alias
	f.lastLearned.id = id
	return f.learnErr
}

func TestSearch_EmptyQuery(t *testing.T) {
	h := NewInstrumentHandler(&fakeInstrRepo{})
	w := httptest.NewRecorder()
	h.Search(w, httptest.NewRequest(http.MethodGet, "/v1/instruments/search?q=", nil))
	if w.Code != 200 || strings.TrimSpace(w.Body.String()) != "[]" {
		t.Errorf("empty q should return [], got %d %q", w.Code, w.Body.String())
	}
}

func TestSearch_AliasFirst(t *testing.T) {
	repo := &fakeInstrRepo{
		aliasResults: []SearchResult{{ID: "1", Symbol: "005930", Exchange: "KRX", Name: "삼성전자"}},
		textResults:  []SearchResult{{ID: "2", Symbol: "X", Exchange: "X", Name: "X"}}, // 호출되면 안 됨
	}
	h := NewInstrumentHandler(repo)
	w := httptest.NewRecorder()
	h.Search(w, httptest.NewRequest(http.MethodGet, "/v1/instruments/search?q=samsung", nil))
	var got []SearchResult
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if len(got) != 1 || got[0].Symbol != "005930" {
		t.Errorf("alias should win, got %+v", got)
	}
}

func TestSearch_FallbackText(t *testing.T) {
	repo := &fakeInstrRepo{
		aliasResults: nil,
		textResults:  []SearchResult{{ID: "2", Symbol: "AAPL", Exchange: "NASDAQ", Name: "Apple"}},
	}
	h := NewInstrumentHandler(repo)
	w := httptest.NewRecorder()
	h.Search(w, httptest.NewRequest(http.MethodGet, "/v1/instruments/search?q=apple", nil))
	var got []SearchResult
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if len(got) != 1 || got[0].Symbol != "AAPL" {
		t.Errorf("text fallback failed, got %+v", got)
	}
}

func TestSelect_LearnsAlias(t *testing.T) {
	repo := &fakeInstrRepo{}
	h := NewInstrumentHandler(repo)
	body, _ := json.Marshal(map[string]string{"query": "삼전", "instrument_id": "uuid-1"})
	w := httptest.NewRecorder()
	h.Select(w, httptest.NewRequest(http.MethodPost, "/v1/instruments/select", bytes.NewReader(body)))
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	if repo.lastLearned.alias != "삼전" || repo.lastLearned.id != "uuid-1" {
		t.Errorf("learn args wrong: %+v", repo.lastLearned)
	}
}

func TestSelect_BadJSON(t *testing.T) {
	h := NewInstrumentHandler(&fakeInstrRepo{})
	w := httptest.NewRecorder()
	h.Select(w, httptest.NewRequest(http.MethodPost, "/v1/instruments/select", strings.NewReader("not json")))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400", w.Code)
	}
}

func TestSelect_MissingFields(t *testing.T) {
	h := NewInstrumentHandler(&fakeInstrRepo{})
	body, _ := json.Marshal(map[string]string{"query": "", "instrument_id": ""})
	w := httptest.NewRecorder()
	h.Select(w, httptest.NewRequest(http.MethodPost, "/v1/instruments/select", bytes.NewReader(body)))
	if w.Code != http.StatusBadRequest {
		t.Errorf("status %d, want 400", w.Code)
	}
}

func TestSelect_RepoError(t *testing.T) {
	h := NewInstrumentHandler(&fakeInstrRepo{learnErr: errors.New("db")})
	body, _ := json.Marshal(map[string]string{"query": "x", "instrument_id": "y"})
	w := httptest.NewRecorder()
	h.Select(w, httptest.NewRequest(http.MethodPost, "/v1/instruments/select", bytes.NewReader(body)))
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status %d, want 500", w.Code)
	}
}
