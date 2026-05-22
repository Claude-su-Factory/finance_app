package yahoo

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSymbolKR(t *testing.T) {
	assert.Equal(t, "005930.KS", SymbolKR("005930", "KOSPI"))
	assert.Equal(t, "247540.KQ", SymbolKR("247540", "KOSDAQ"))
	assert.Equal(t, "AAPL", SymbolKR("AAPL", "NASDAQ"))
}
