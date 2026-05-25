package schedule

import "testing"

func TestUserMinuteSlot_Stable(t *testing.T) {
	uid := "00000000-0000-0000-0000-000000000001"
	a := userMinuteSlot(uid)
	b := userMinuteSlot(uid)
	if a != b {
		t.Errorf("unstable hash: got %d then %d for same uid", a, b)
	}
}

func TestUserMinuteSlot_Range(t *testing.T) {
	for _, uid := range []string{
		"00000000-0000-0000-0000-000000000001",
		"abc",
		"xyz",
		"super-long-user-identifier-1234567890",
	} {
		s := userMinuteSlot(uid)
		if s < 0 || s >= 60 {
			t.Errorf("uid %s slot %d out of [0, 60)", uid, s)
		}
	}
}

func TestUserMinuteSlot_Distribution(t *testing.T) {
	counts := make(map[int]int)
	for i := range 1000 {
		counts[userMinuteSlot(string(rune(i)))]++
	}
	for slot, n := range counts {
		if n > 50 {
			t.Errorf("slot %d has %d users (poor distribution)", slot, n)
		}
	}
}
