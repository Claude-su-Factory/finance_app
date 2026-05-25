package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/ai"
)

// Tool은 단일 도구 함수.
// userID는 RLS 우회 superuser pool 패턴에서 WHERE user_id 필터에 사용.
// 미래 JWT 기반 풀 전환 시 시그니처는 동일 유지(의미만 jwt→userID 변환).
type Tool interface {
	Spec() ai.ToolSpec
	Run(ctx context.Context, userID string, input map[string]any) (any, error)
}

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Spec().Name] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// Specs는 Claude API에 전달할 전체 스펙 슬라이스 (이름순 정렬 — prompt cache 안정성).
func (r *Registry) Specs() []ai.ToolSpec {
	names := make([]string, 0, len(r.tools))
	for n := range r.tools {
		names = append(names, n)
	}
	// 이름순 정렬
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j-1] > names[j]; j-- {
			names[j-1], names[j] = names[j], names[j-1]
		}
	}
	out := make([]ai.ToolSpec, 0, len(names))
	for _, n := range names {
		out = append(out, r.tools[n].Spec())
	}
	return out
}

// ExecuteAndSerialize는 도구 실행 + 결과 JSON 직렬화 (tool_result content).
// 실패 시 error 메시지를 result로 반환 (Claude가 retry 결정).
func ExecuteAndSerialize(ctx context.Context, r *Registry, name, userID string, input map[string]any) string {
	t, ok := r.Get(name)
	if !ok {
		return fmt.Sprintf(`{"error":"unknown tool: %s"}`, name)
	}
	out, err := t.Run(ctx, userID, input)
	if err != nil {
		b, _ := json.Marshal(err.Error())
		return fmt.Sprintf(`{"error":%s}`, b)
	}
	b, err := json.Marshal(out)
	if err != nil {
		return fmt.Sprintf(`{"error":"serialize failed: %s"}`, err.Error())
	}
	return string(b)
}

// Deps는 모든 도구가 공유하는 의존성.
type Deps struct {
	Pool *pgxpool.Pool
}
