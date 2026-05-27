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

// nyLoc returns the America/New_York location (DST 자동 적용), falling back to UTC.
func nyLoc() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
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

// IsUSMarketOpen reports whether NYSE/NASDAQ is in regular trading hours at t.
// NY 타임존 기준 평일 09:30~16:00. DST 자동 적용 (EST/EDT).
// KST 토요일 새벽 = NY 금요일 후반 → 자동 true (Friday 세션 정상 포함).
func IsUSMarketOpen(t time.Time) bool {
	t = t.In(nyLoc())
	if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
		return false
	}
	mins := t.Hour()*60 + t.Minute()
	return mins >= 9*60+30 && mins <= 16*60
}
