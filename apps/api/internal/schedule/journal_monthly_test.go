package schedule

import "testing"

func TestUserMinuteSlot_Deterministic(t *testing.T) {
	uid := "00000000-0000-0000-0000-000000000001"
	a := userMinuteSlot(uid)
	b := userMinuteSlot(uid)
	if a != b {
		t.Errorf("slot non-deterministic: %d vs %d", a, b)
	}
	if a < 0 || a >= 60 {
		t.Errorf("slot out of range: %d", a)
	}
}
