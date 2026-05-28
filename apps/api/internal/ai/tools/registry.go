package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/quotient/quotient/apps/api/internal/ai"
	"github.com/quotient/quotient/apps/api/internal/db"
)

// Tool은 단일 도구 함수.
// 모든 도구는 db.Executor를 받아 트랜잭션·풀 양쪽에서 동작.
// RequiresUserContext가 true면 ExecuteAndSerialize가 db.AsUser로 wrap하여
// Supabase RLS가 자동 적용되도록 한다. false면 슈퍼유저 풀 직접 사용(공개 데이터).
type Tool interface {
	Spec() ai.ToolSpec
	Run(ctx context.Context, exec db.Executor, userID string, input map[string]any) (any, error)
	RequiresUserContext() bool
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
// 사용자 데이터 도구는 db.AsUser로 트랜잭션 wrap → RLS 자동 적용.
// pool == nil이면(테스트) wrap 우회 — fake 도구가 exec 무시.
// 실패 시 error 메시지를 result로 반환 (Claude가 retry 결정).
func ExecuteAndSerialize(ctx context.Context, r *Registry, pool *pgxpool.Pool, name, userID string, input map[string]any) string {
	t, ok := r.Get(name)
	if !ok {
		return fmt.Sprintf(`{"error":"unknown tool: %s"}`, name)
	}

	var out any
	var err error
	if t.RequiresUserContext() && pool != nil {
		err = db.AsUser(ctx, pool, userID, func(exec db.Executor) error {
			o, e := t.Run(ctx, exec, userID, input)
			if e != nil {
				return e
			}
			out = o
			return nil
		})
	} else {
		// 공개 데이터 도구 또는 test passthrough.
		var exec db.Executor
		if pool != nil {
			exec = pool
		}
		out, err = t.Run(ctx, exec, userID, input)
	}

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
	Pool   *pgxpool.Pool
	Client ai.Client // analyze_journal 같은 LLM 호출 도구가 사용. 다른 도구는 무시.
}
