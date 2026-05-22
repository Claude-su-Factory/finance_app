package common

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

// DoWithBackoff retries with exponential delays: 0, 1, 2, 4, 8, 16s. Up to 5 retries.
// Retries on transport error, 429, or 5xx. Non-retryable status returns immediately.
func DoWithBackoff(ctx context.Context, do func() (*http.Response, error)) (*http.Response, error) {
	delays := []time.Duration{0, time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}
	var lastErr error
	for i, d := range delays {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(d):
			}
		}
		resp, err := do()
		if err != nil {
			lastErr = err
			slog.Warn("backoff retry on error", "attempt", i+1, "err", err)
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			resp.Body.Close()
			lastErr = errors.New("retryable status: " + resp.Status)
			slog.Warn("backoff retry on status", "attempt", i+1, "status", resp.StatusCode)
			continue
		}
		return resp, nil
	}
	return nil, lastErr
}
