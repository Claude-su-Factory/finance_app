package fred

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchObservations(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "series_id=DFF")
		assert.Contains(t, r.URL.RawQuery, "api_key=test-key")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"observations": []map[string]any{
				{"date": "2025-12-01", "value": "5.25"},
				{"date": "2025-12-02", "value": "."},  // missing → 필터링되어야
				{"date": "2025-12-03", "value": "5.50"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	obs, err := c.FetchObservations(context.Background(), "DFF")
	require.NoError(t, err)
	require.Equal(t, 2, len(obs), "missing 값 필터링")
	assert.Equal(t, 5.25, obs[0].Value)
	assert.Equal(t, "DFF", obs[0].Code)
}
