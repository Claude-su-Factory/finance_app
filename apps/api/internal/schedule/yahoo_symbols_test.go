package schedule

import "testing"

func TestIndexYahooSymbol(t *testing.T) {
	cases := []struct {
		symbol, exchange, want string
	}{
		{"KOSPI", "KRX-IDX", "^KS11"},
		{"KOSDAQ", "KRX-IDX", "^KQ11"},
		{"SPX", "NYSE-IDX", "^GSPC"},
		{"NDX", "NASDAQ-IDX", "^NDX"},
		{"UNKNOWN", "X", ""},
	}
	for _, c := range cases {
		got := IndexYahooSymbol(c.symbol, c.exchange)
		if got != c.want {
			t.Errorf("IndexYahooSymbol(%q, %q) = %q, want %q", c.symbol, c.exchange, got, c.want)
		}
	}
}

func TestStockYahooSymbol(t *testing.T) {
	cases := []struct {
		sym, ex, want string
	}{
		{"005930", "KOSPI", "005930.KS"},
		{"035720", "KOSDAQ", "035720.KQ"},
		{"AAPL", "NASDAQ", "AAPL"},
		{"AAPL", "UNKNOWN", ""},
	}
	for _, c := range cases {
		if got := StockYahooSymbol(c.sym, c.ex); got != c.want {
			t.Errorf("StockYahooSymbol(%q,%q) = %q want %q", c.sym, c.ex, got, c.want)
		}
	}
}
