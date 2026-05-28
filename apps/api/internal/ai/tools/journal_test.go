package tools

import (
	"testing"

	"github.com/quotient/quotient/apps/api/internal/ai"
)

func TestAnalyzeJournal_Spec(t *testing.T) {
	tool := &analyzeJournal{Deps: &Deps{}}
	spec := tool.Spec()
	if spec.Name != "analyze_journal" {
		t.Errorf("name=%s", spec.Name)
	}
	if !tool.RequiresUserContext() {
		t.Error("RequiresUserContext must be true")
	}
}

func TestAnalyzeJournal_RegisterPicksUpInRegistry(t *testing.T) {
	r := NewRegistry()
	RegisterJournal(r, &Deps{Pool: nil, Client: &ai.MockClient{}})
	if _, ok := r.Get("analyze_journal"); !ok {
		t.Fatal("analyze_journal not registered")
	}
}
