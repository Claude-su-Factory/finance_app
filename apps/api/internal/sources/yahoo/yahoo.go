package yahoo

import (
	"context"
	"fmt"
	"time"

	"github.com/piquette/finance-go/chart"
	"github.com/piquette/finance-go/datetime"
	"github.com/piquette/finance-go/quote"
	"github.com/quotient/quotient/apps/api/internal/models"
)

type Client struct{}

func NewClient() *Client { return &Client{} }

// FetchChart retrieves daily OHLCV bars for the given symbol over [start, end).
// If end is zero, it defaults to now.
func (c *Client) FetchChart(ctx context.Context, symbol string, start, end time.Time) ([]models.PriceBar, error) {
	if end.IsZero() {
		end = time.Now()
	}
	p := &chart.Params{
		Symbol:   symbol,
		Interval: datetime.OneDay,
		Start:    datetime.FromUnix(int(start.Unix())),
		End:      datetime.FromUnix(int(end.Unix())),
	}
	iter := chart.Get(p)
	var bars []models.PriceBar
	for iter.Next() {
		b := iter.Bar()
		// shopspring/decimal: Float64() returns (float64, bool)
		o, _ := b.Open.Float64()
		h, _ := b.High.Float64()
		l, _ := b.Low.Float64()
		cl, _ := b.Close.Float64()
		bars = append(bars, models.PriceBar{
			Date:   time.Unix(int64(b.Timestamp), 0).UTC(),
			Open:   o,
			High:   h,
			Low:    l,
			Close:  cl,
			Volume: int64(b.Volume),
		})
	}
	if err := iter.Err(); err != nil {
		return bars, fmt.Errorf("yahoo chart %s: %w", symbol, err)
	}
	return bars, nil
}

// FetchQuote retrieves a real-time quote for the given symbol.
func (c *Client) FetchQuote(ctx context.Context, symbol string) (models.Quote, error) {
	q, err := quote.Get(symbol)
	if err != nil {
		return models.Quote{}, fmt.Errorf("yahoo quote %s: %w", symbol, err)
	}
	if q == nil {
		return models.Quote{}, fmt.Errorf("yahoo quote nil: %s", symbol)
	}
	return models.Quote{
		Price:     q.RegularMarketPrice,
		ChangeAbs: q.RegularMarketChange,
		ChangePct: q.RegularMarketChangePercent,
		UpdatedAt: time.Now().UTC(),
	}, nil
}

// SymbolKR returns the Yahoo Finance-formatted symbol for Korean stocks.
// KOSPI symbols get the ".KS" suffix; KOSDAQ symbols get ".KQ".
// For US symbols (or unknown markets), the symbol is returned unchanged.
func SymbolKR(symbol, market string) string {
	switch market {
	case "KOSPI":
		return symbol + ".KS"
	case "KOSDAQ":
		return symbol + ".KQ"
	default:
		return symbol
	}
}
