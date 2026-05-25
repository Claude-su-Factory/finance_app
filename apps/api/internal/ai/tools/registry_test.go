package tools

import (
	"context"
	"testing"

	"github.com/quotient/quotient/apps/api/internal/ai"
)

type fakeTool struct {
	name string
	out  any
	err  error
}

func (f *fakeTool) Spec() ai.ToolSpec {
	return ai.ToolSpec{Name: f.name, Description: "test", InputSchema: map[string]any{"type": "object"}}
}
func (f *fakeTool) Run(ctx context.Context, userID string, input map[string]any) (any, error) {
	return f.out, f.err
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeTool{name: "foo", out: map[string]any{"ok": true}})
	got, ok := r.Get("foo")
	if !ok || got == nil {
		t.Fatal("expected foo registered")
	}
	if _, ok := r.Get("nope"); ok {
		t.Error("expected nope absent")
	}
}

func TestRegistry_Specs(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeTool{name: "a"})
	r.Register(&fakeTool{name: "b"})
	specs := r.Specs()
	if len(specs) != 2 {
		t.Errorf("got %d specs, want 2", len(specs))
	}
	// 정렬 검증
	if specs[0].Name > specs[1].Name {
		t.Errorf("specs not sorted: %v", specs)
	}
}

func TestExecuteAndSerialize_Success(t *testing.T) {
	r := NewRegistry()
	r.Register(&fakeTool{name: "foo", out: map[string]any{"value": 42}})
	s := ExecuteAndSerialize(context.Background(), r, "foo", "user-1", nil)
	if s != `{"value":42}` {
		t.Errorf("got %s", s)
	}
}

func TestExecuteAndSerialize_UnknownTool(t *testing.T) {
	r := NewRegistry()
	s := ExecuteAndSerialize(context.Background(), r, "missing", "user-1", nil)
	if s != `{"error":"unknown tool: missing"}` {
		t.Errorf("got %s", s)
	}
}
