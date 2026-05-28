package schedule

// IndexYahooSymbol maps Quotient의 internal index symbol+exchange를 Yahoo Finance 심볼로 변환한다.
// 미지 매핑은 "" 반환 → 호출자가 skip.
func IndexYahooSymbol(symbol, exchange string) string {
	switch {
	case symbol == "KOSPI" && exchange == "KRX-IDX":
		return "^KS11"
	case symbol == "KOSDAQ" && exchange == "KRX-IDX":
		return "^KQ11"
	case symbol == "SPX" && exchange == "NYSE-IDX":
		return "^GSPC"
	case symbol == "NDX" && exchange == "NASDAQ-IDX":
		return "^NDX"
	case symbol == "DJI" && exchange == "NYSE-IDX":
		return "^DJI"
	default:
		return ""
	}
}

// StockYahooSymbol은 일반 종목(KR_STOCK/US_STOCK/ETF)의 Yahoo 심볼을 반환.
// KR: "005930" + "KOSPI" → "005930.KS", "KOSDAQ" → ".KQ"
// US: 그대로
// 미지 exchange는 "" 반환 → 호출자가 skip.
func StockYahooSymbol(symbol, exchange string) string {
	switch exchange {
	case "KOSPI":
		return symbol + ".KS"
	case "KOSDAQ":
		return symbol + ".KQ"
	case "NYSE", "NASDAQ", "AMEX":
		return symbol
	}
	return ""
}

func isKRExchange(ex string) bool {
	return ex == "KOSPI" || ex == "KOSDAQ"
}

func isUSExchange(ex string) bool {
	return ex == "NYSE" || ex == "NASDAQ" || ex == "AMEX"
}
