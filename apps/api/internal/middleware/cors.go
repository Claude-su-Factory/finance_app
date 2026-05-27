package middleware

import (
	"net/http"
	"strings"

	"github.com/go-chi/cors"
)

// CORS는 콤마 구분 origin 목록을 받는다.
// production: "https://quotient.example.com,https://quotient-staging.vercel.app"
// Vercel preview처럼 host가 매번 다른 경우 와일드카드 prefix 사용:
//   "https://quotient.example.com,https://quotient-*-myteam.vercel.app"
// 와일드카드가 하나라도 있으면 AllowOriginFunc로 매칭, 없으면 정적 AllowedOrigins.
func CORS(originList string) func(http.Handler) http.Handler {
	origins := splitOrigins(originList)
	var exact, wildcards []string
	for _, o := range origins {
		if strings.Contains(o, "*") {
			wildcards = append(wildcards, o)
		} else {
			exact = append(exact, o)
		}
	}

	opts := cors.Options{
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}
	if len(wildcards) > 0 {
		opts.AllowOriginFunc = func(_ *http.Request, origin string) bool {
			for _, o := range exact {
				if origin == o {
					return true
				}
			}
			for _, pat := range wildcards {
				if matchWildcard(pat, origin) {
					return true
				}
			}
			return false
		}
	} else {
		opts.AllowedOrigins = exact
	}
	return cors.Handler(opts)
}

func splitOrigins(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// matchWildcard는 단일 `*` 한 개 패턴 매칭 (예: `https://quotient-*.vercel.app`).
func matchWildcard(pat, origin string) bool {
	i := strings.Index(pat, "*")
	if i < 0 {
		return pat == origin
	}
	prefix, suffix := pat[:i], pat[i+1:]
	return strings.HasPrefix(origin, prefix) &&
		strings.HasSuffix(origin, suffix) &&
		len(origin) >= len(prefix)+len(suffix)
}
