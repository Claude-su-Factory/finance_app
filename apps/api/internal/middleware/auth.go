package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/quotient/quotient/apps/api/internal/auth"
)

type ctxKey string

const UserIDKey ctxKey = "user_id"
const RawJWTKey ctxKey = "raw_jwt"

func RequireAuth(v *auth.Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if len(header) < 7 || !strings.EqualFold(header[:7], "bearer ") {
				http.Error(w, `{"error":{"code":"UNAUTHENTICATED","message":"missing bearer token"}}`, http.StatusUnauthorized)
				return
			}
			// RFC 7235: scheme 대소문자 무관
			token := header[7:]
			uid, err := v.UserIDFromToken(token)
			if err != nil {
				http.Error(w, `{"error":{"code":"UNAUTHENTICATED","message":"invalid token"}}`, http.StatusUnauthorized)
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
