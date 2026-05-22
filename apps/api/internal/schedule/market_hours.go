package schedule

import "time"

// SeoulLoc returns the Asia/Seoul location, falling back to UTC if loadlocation fails.
func SeoulLoc() *time.Location {
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		return time.UTC
	}
	return loc
}

// IsKRMarketOpen reports whether KRX is in regular trading hours at t (KST).
// 평일 09:00–15:30 KST. 공휴일 미고려 (MVP).
func IsKRMarketOpen(t time.Time) bool {
	t = t.In(SeoulLoc())
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	mins := t.Hour()*60 + t.Minute()
	return mins >= 9*60 && mins <= 15*60+30
}

// IsUSMarketOpen reports whether NYSE/NASDAQ is in regular trading hours at t (KST).
// US 정규장(09:30~16:00 EST) ≈ KST 23:30~익일 06:00. 서머타임 보정 미반영 (MVP).
// 토·일 KST는 보수적으로 false (NY 시간 금요일 장 후반 일부 포함될 수 있으나 무시).
func IsUSMarketOpen(t time.Time) bool {
	t = t.In(SeoulLoc())
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	mins := t.Hour()*60 + t.Minute()
	return mins >= 23*60+30 || mins <= 6*60
}
