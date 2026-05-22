package kind

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"

	"github.com/quotient/quotient/apps/api/internal/models"
	"github.com/quotient/quotient/apps/api/internal/sources/common"
)

const baseURL = "https://kind.krx.co.kr/corpgeneral/corpList.do"

type Client struct {
	url  string
	http *http.Client
}

func NewClient(url string) *Client {
	if url == "" {
		url = baseURL
	}
	return &Client{url: url, http: &http.Client{Timeout: 30 * time.Second}}
}

// FetchInstruments returns KOSPI or KOSDAQ listings from KIND public HTML download.
// market: "KOSPI" or "KOSDAQ".
// KIND는 ISIN을 노출하지 않으므로 Instrument.ISIN = nil.
func (c *Client) FetchInstruments(ctx context.Context, market string) ([]models.Instrument, error) {
	mt := "stockMkt"
	if strings.EqualFold(market, "KOSDAQ") {
		mt = "kosdaqMkt"
	}
	u := fmt.Sprintf("%s?method=download&searchType=13&marketType=%s", c.url, mt)

	resp, err := common.DoWithBackoff(ctx, func() (*http.Response, error) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0 (Quotient/1.0)")
		return c.http.Do(req)
	})
	if err != nil {
		return nil, fmt.Errorf("kind fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kind HTTP %d", resp.StatusCode)
	}

	// EUC-KR → UTF-8 + HTML 파싱
	utf8r := transform.NewReader(resp.Body, korean.EUCKR.NewDecoder())
	doc, err := html.Parse(utf8r)
	if err != nil {
		return nil, fmt.Errorf("kind html parse: %w", err)
	}

	return parseKindTable(doc), nil
}

// parseKindTable walks <tr><td> rows. 0=name 1=market 2=symbol(6자리) 3=sector ...
func parseKindTable(doc *html.Node) []models.Instrument {
	var out []models.Instrument
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			tds := []string{}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && c.Data == "td" {
					tds = append(tds, strings.TrimSpace(textOf(c)))
				}
			}
			if len(tds) >= 3 {
				symbol := tds[2]
				// 종목코드 6자리 검증
				if len(symbol) == 6 && isAllDigits(symbol) {
					out = append(out, models.Instrument{
						Symbol:     symbol,
						Exchange:   "KRX",
						Name:       tds[0],
						AssetClass: models.AssetKRStock,
						Currency:   "KRW",
						IsActive:   true,
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out
}

func textOf(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	var b strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		b.WriteString(textOf(c))
	}
	return b.String()
}

func isAllDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
