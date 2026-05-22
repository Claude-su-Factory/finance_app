package schedule

import (
	"testing"
	"time"
)

func TestIsKRMarketOpen(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Seoul")
	cases := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"평일 장 시작", time.Date(2025, 12, 2, 9, 0, 0, 0, loc), true},
		{"평일 장 마감 직전", time.Date(2025, 12, 2, 15, 30, 0, 0, loc), true},
		{"평일 장 마감 직후", time.Date(2025, 12, 2, 15, 31, 0, 0, loc), false},
		{"평일 장 시작 전", time.Date(2025, 12, 2, 8, 59, 0, 0, loc), false},
		{"토요일", time.Date(2025, 11, 29, 10, 0, 0, 0, loc), false},
		{"일요일", time.Date(2025, 11, 30, 10, 0, 0, 0, loc), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsKRMarketOpen(c.t); got != c.want {
				t.Errorf("IsKRMarketOpen(%v) = %v, want %v", c.t, got, c.want)
			}
		})
	}
}

func TestIsUSMarketOpen(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Seoul")
	cases := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"평일 KST 23:30 (US 개장)", time.Date(2025, 12, 2, 23, 30, 0, 0, loc), true},
		{"평일 KST 23:29", time.Date(2025, 12, 2, 23, 29, 0, 0, loc), false},
		{"평일 익일 KST 05:00", time.Date(2025, 12, 3, 5, 0, 0, 0, loc), true},
		{"평일 익일 KST 06:00 (마감)", time.Date(2025, 12, 3, 6, 0, 0, 0, loc), true},
		{"평일 익일 KST 06:01", time.Date(2025, 12, 3, 6, 1, 0, 0, loc), false},
		{"토요일 KST 02:00 (NY 금요일 장중 가능하나 보수적 OFF)", time.Date(2025, 11, 29, 2, 0, 0, 0, loc), false},
		{"일요일 KST 02:00", time.Date(2025, 11, 30, 2, 0, 0, 0, loc), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsUSMarketOpen(c.t); got != c.want {
				t.Errorf("IsUSMarketOpen(%v) = %v, want %v", c.t, got, c.want)
			}
		})
	}
}

func TestSeoulLoc(t *testing.T) {
	loc := SeoulLoc()
	if loc.String() != "Asia/Seoul" {
		t.Errorf("SeoulLoc = %s, want Asia/Seoul", loc.String())
	}
}
