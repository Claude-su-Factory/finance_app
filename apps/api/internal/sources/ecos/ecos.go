package ecos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	if baseURL == "" { baseURL = "https://ecos.bok.or.kr/api/StatisticSearch" }
	return &Client{url: baseURL, key: key, http: &http.Client{Timeout: 20 * time.Second}}
}

// FetchSeries: cycle = "D"|"M"|"A". start/end format depends on cycle.
func (c *Client) FetchSeries(ctx context.Context, statCode, cycle, start, end string) ([]models.Indicator, error) {
	u := fmt.Sprintf("%s/%s/json/kr/1/1000/%s/%s/%s/%s", c.url, c.key, statCode, cycle, start, end)

	resp, err := common.DoWithBackoff(ctx, func() (*http.Response, error) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		req.Header.Set("User-Agent", "Quotient/1.0")
		return c.http.Do(req)
	})
	if err != nil { return nil, fmt.Errorf("ecos: %w", err) }
	defer resp.Body.Close()

	var raw map[string]json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("ecos parse: %w", err)
	}
	// 실패 응답 (RESULT 키 존재) 명시 처리
	if r, ok := raw["RESULT"]; ok {
		var res struct {
			Code    string `json:"CODE"`
			Message string `json:"MESSAGE"`
		}
		_ = json.Unmarshal(r, &res)
		return nil, fmt.Errorf("ecos error %s: %s", res.Code, res.Message)
	}
	stat, ok := raw["StatisticSearch"]
	if !ok { return nil, fmt.Errorf("ecos: missing StatisticSearch in response") }
	var s struct {
		Row []struct {
			StatCode  string `json:"STAT_CODE"`
			StatName  string `json:"STAT_NAME"`
			Time      string `json:"TIME"`
			DataValue string `json:"DATA_VALUE"`
			UnitName  string `json:"UNIT_NAME"`
		} `json:"row"`
	}
	if err := json.Unmarshal(stat, &s); err != nil {
		return nil, fmt.Errorf("ecos parse row: %w", err)
	}

	layout := timeLayoutFor(cycle)
	out := make([]models.Indicator, 0, len(s.Row))
	for _, r := range s.Row {
		date, err := time.Parse(layout, r.Time)
		if err != nil { continue }
		val, err := strconv.ParseFloat(r.DataValue, 64)
		if err != nil { continue }
		unit := r.UnitName
		out = append(out, models.Indicator{
			Code: statCode, ObservedAt: date, Name: r.StatName,
			Value: val, Unit: &unit,
		})
	}
	return out, nil
}

func timeLayoutFor(cycle string) string {
	switch cycle {
	case "D": return "20060102"
	case "A": return "2006"
	default:  return "200601"  // M
	}
}
