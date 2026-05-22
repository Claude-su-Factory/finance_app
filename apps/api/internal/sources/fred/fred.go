package fred

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/quotient/quotient/apps/api/internal/sources/common"
)

type Client struct {
	url, key string
	http     *http.Client
}

func NewClient(baseURL, key string) *Client {
	if baseURL == "" { baseURL = "https://api.stlouisfed.org/fred/series/observations" }
	return &Client{url: baseURL, key: key, http: &http.Client{Timeout: 20 * time.Second}}
}

func (c *Client) FetchObservations(ctx context.Context, seriesID string) ([]models.Indicator, error) {
	q := url.Values{}
	q.Set("series_id", seriesID)
	q.Set("api_key", c.key)
	q.Set("file_type", "json")

	resp, err := common.DoWithBackoff(ctx, func() (*http.Response, error) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"?"+q.Encode(), nil)
		return c.http.Do(req)
	})
	if err != nil { return nil, fmt.Errorf("fred: %w", err) }
	defer resp.Body.Close()

	var p struct {
		Observations []struct {
			Date  string `json:"date"`
			Value string `json:"value"`
		} `json:"observations"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, fmt.Errorf("fred parse: %w", err)
	}
	out := make([]models.Indicator, 0, len(p.Observations))
	for _, o := range p.Observations {
		if o.Value == "." { continue } // FRED missing-value
		date, err := time.Parse("2006-01-02", o.Date)
		if err != nil { continue }
		val, err := strconv.ParseFloat(o.Value, 64)
		if err != nil { continue }
		out = append(out, models.Indicator{Code: seriesID, ObservedAt: date, Value: val})
	}
	return out, nil
}
