package ecos

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchSeries_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"StatisticSearch": map[string]any{
				"row": []map[string]any{
					{"STAT_CODE": "722Y001", "STAT_NAME": "기준금리", "TIME": "202501", "DATA_VALUE": "3.25", "UNIT_NAME": "%"},
					{"STAT_CODE": "722Y001", "STAT_NAME": "기준금리", "TIME": "202502", "DATA_VALUE": "3.00", "UNIT_NAME": "%"},
				},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	obs, err := c.FetchSeries(context.Background(), "722Y001", "M", "202501", "202502")
	require.NoError(t, err)
	require.Equal(t, 2, len(obs))
	assert.Equal(t, 3.25, obs[0].Value)
	assert.Equal(t, "%", *obs[0].Unit)
}

func TestFetchSeries_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"RESULT": map[string]string{"CODE": "INFO-200", "MESSAGE": "해당하는 자료가 없습니다."},
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "test-key")
	_, err := c.FetchSeries(context.Background(), "BAD", "M", "202501", "202501")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "INFO-200")
}
