package observability

import "testing"

// CaptureException(nil)은 Sentry 미초기화 상태에서도 panic 없이 무시되어야 한다.
func TestCaptureExceptionNilSafe(t *testing.T) {
	CaptureException(nil) // nil → 즉시 반환, no panic
}
