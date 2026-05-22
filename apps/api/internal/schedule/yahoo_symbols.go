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
	default:
		return ""
	}
}
