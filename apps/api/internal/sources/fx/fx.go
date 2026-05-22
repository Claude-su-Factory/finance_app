package fx

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/quotient/quotient/apps/api/internal/sources/common"
)

const baseURL = "https://api.frankfurter.dev/v1/latest"

type Client struct {
	url  string
	http *http.Client
}

func NewClient(url string) *Client {
	if url == "" {
		url = baseURL
	}
	return &Client{url: url, http: &http.Client{Timeout: 15 * time.Second}}
}

// FetchRates fetches exchange rates from frankfurter.dev.
// Returns a list of FXRate records for the given base currency and target symbols.
// Response format: { date, base, rates }
func (c *Client) FetchRates(ctx context.Context, base string, symbols []string) ([]models.FXRate, error) {
	u := fmt.Sprintf("%s?base=%s&symbols=%s", c.url, base, strings.Join(symbols, ","))

	resp, err := common.DoWithBackoff(ctx, func() (*http.Response, error) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		req.Header.Set("User-Agent", "Quotient/1.0")
		return c.http.Do(req)
	})
	if err != nil {
		return nil, fmt.Errorf("fx: %w", err)
	}
	defer resp.Body.Close()

	var p struct {
		Date  string             `json:"date"`
		Base  string             `json:"base"`
		Rates map[string]float64 `json:"rates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, fmt.Errorf("fx parse: %w", err)
	}

	date, err := time.Parse("2006-01-02", p.Date)
	if err != nil {
		date = time.Now().UTC()
	}

	out := make([]models.FXRate, 0, len(p.Rates))
	for q, r := range p.Rates {
		out = append(out, models.FXRate{Base: p.Base, Quote: q, ObservedAt: date, Rate: r})
	}
	return out, nil
}
