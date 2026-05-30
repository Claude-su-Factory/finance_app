package migrate

import "testing"

func TestParseMigrationFilename(t *testing.T) {
	tests := []struct {
		name        string
		wantVersion string
		wantLabel   string
		wantOK      bool
	}{
		{"20260522000001_profiles.sql", "20260522000001", "profiles", true},
		{"20260528000002_paper_trading.sql", "20260528000002", "paper_trading", true},
		{"notes.txt", "", "", false},       // .sql 아님
		{"_leading.sql", "", "", false},    // version 비어 있음
		{"abc_def.sql", "", "", false},     // version 비숫자
		{"20260522000001.sql", "", "", false}, // '_' 없음
	}
	for _, tt := range tests {
		v, l, ok := parseMigrationFilename(tt.name)
		if v != tt.wantVersion || l != tt.wantLabel || ok != tt.wantOK {
			t.Errorf("parseMigrationFilename(%q) = (%q,%q,%v), want (%q,%q,%v)",
				tt.name, v, l, ok, tt.wantVersion, tt.wantLabel, tt.wantOK)
		}
	}
}

func TestPendingMigrations(t *testing.T) {
	available := []Migration{
		{Version: "003", Name: "c.sql"},
		{Version: "001", Name: "a.sql"},
		{Version: "002", Name: "b.sql"},
	}
	applied := map[string]bool{"001": true}
	got := pendingMigrations(available, applied)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Version != "002" || got[1].Version != "003" {
		t.Errorf("order = [%s,%s], want [002,003]", got[0].Version, got[1].Version)
	}
}

func TestPendingMigrationsAllApplied(t *testing.T) {
	available := []Migration{{Version: "001"}, {Version: "002"}}
	applied := map[string]bool{"001": true, "002": true}
	if got := pendingMigrations(available, applied); len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}
