// Package observability — Sentry 초기화 + chi 미들웨어 wiring.
// DSN이 비어 있으면 no-op (개발 환경 안전).
package observability

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
)

// InitSentry는 SENTRY_DSN_API가 비어 있으면 no-op.
// 호출자는 종료 시 sentry.Flush(2s)로 잔여 이벤트 flush 필요.
func InitSentry(dsn, env, release string) error {
	if dsn == "" {
		slog.Info("sentry disabled (no DSN)")
		return nil
	}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      env,
		Release:          release,
		TracesSampleRate: 0.1, // 10% 트랜잭션 샘플
		// 개인정보 보호 — 기본값 false 명시.
		SendDefaultPII: false,
	})
	if err != nil {
		return err
	}
	slog.Info("sentry initialized", "env", env)
	return nil
}

// SentryMiddleware는 chi 호환 panic capture + 요청 컨텍스트 wrap.
// DSN 비어도 안전(sentry-go 내부 no-op).
func SentryMiddleware() func(http.Handler) http.Handler {
	h := sentryhttp.New(sentryhttp.Options{
		Repanic:         true,        // chi의 Recoverer가 다시 처리할 수 있도록
		WaitForDelivery: false,
		Timeout:         2 * time.Second,
	})
	return h.Handle
}

// Flush는 graceful shutdown 시 호출.
func Flush() {
	sentry.Flush(2 * time.Second)
}
