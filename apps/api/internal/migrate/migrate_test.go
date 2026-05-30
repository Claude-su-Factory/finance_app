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

func TestLoadRealMigrations(t *testing.T) {
	// 실제 레포의 supabase/migrations 디렉터리(테스트 패키지 기준 상대경로).
	migs, err := Load("../../../../supabase/migrations")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(migs) != 10 {
		t.Fatalf("len = %d, want 10 (supabase/migrations 파일 수)", len(migs))
	}
	// 정렬·파싱 검증.
	if migs[0].Version != "20260522000001" {
		t.Errorf("first version = %q, want 20260522000001", migs[0].Version)
	}
	if migs[len(migs)-1].Version != "20260528000002" {
		t.Errorf("last version = %q, want 20260528000002", migs[len(migs)-1].Version)
	}
	// SQL 본문이 비어 있지 않아야 한다.
	for _, m := range migs {
		if m.SQL == "" {
			t.Errorf("%s: empty SQL", m.Name)
		}
	}
}
