package portfolio_test

import (
	"testing"

	"github.com/quotient/quotient/apps/api/internal/portfolio"
)

func TestParsePeriod(t *testing.T) {
	cases := map[string]bool{
		"1m": true, "90d": true, "1y": true, "all": true,
		"":    false,
		"30d": false,
		"2y":  false,
		"FOO": false,
	}
	for in, ok := range cases {
		_, err := portfolio.ParsePeriod(in)
		if (err == nil) != ok {
			t.Errorf("ParsePeriod(%q): want ok=%v, got err=%v", in, ok, err)
		}
	}
}

func TestPeriodDays(t *testing.T) {
	cases := map[portfolio.Period]int{
		portfolio.Period1M:  30,
		portfolio.Period90D: 90,
		portfolio.Period1Y:  365,
		portfolio.PeriodAll: 0,
	}
	for p, want := range cases {
		if got := p.Days(); got != want {
			t.Errorf("%s.Days(): got %d, want %d", p, got, want)
		}
	}
}
