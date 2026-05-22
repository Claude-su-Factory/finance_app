package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/quotient/quotient/apps/api/internal/auth"
)

type ctxKey string

const UserIDKey ctxKey = "user_id"
// RawJWTKey: AI tool use에서 Supabase 호출 시 사용자 JWT를 전파하기 위해 보존 (스펙 §10-1).
// 로깅·외부 출력 시 절대 노출 금지.
const RawJWTKey ctxKey = "raw_jwt"

func writeAuthError(w http.ResponseWriter, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":{"code":"` + code + `","message":"` + msg + `"}}`))
}

func RequireAuth(v *auth.Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if len(header) < 7 || !strings.EqualFold(header[:7], "bearer ") {
				writeAuthError(w, "UNAUTHENTICATED", "missing bearer token")
				return
			}
			// RFC 7235: scheme 대소문자 무관
			token := header[7:]
			uid, err := v.UserIDFromToken(token)
			if err != nil {
				writeAuthError(w, "UNAUTHENTICATED", "invalid token")
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, uid)
			ctx = context.WithValue(ctx, RawJWTKey, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func UserID(ctx context.Context) string {
	if v, ok := ctx.Value(UserIDKey).(string); ok {
		return v
	}
	return ""
}

func RawJWT(ctx context.Context) string {
	if v, ok := ctx.Value(RawJWTKey).(string); ok {
		return v
	}
	return ""
}
