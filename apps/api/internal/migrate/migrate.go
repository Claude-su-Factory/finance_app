// Package migrate — Supabase가 쓰는 supabase_migrations.schema_migrations 이력 테이블을
// 공유하는 forward-only 마이그레이터. release_command(/app/migrate)로 배포 전 적용한다.
package migrate

import (
	"sort"
	"strings"
)

// Migration은 하나의 .sql 마이그레이션 파일을 나타낸다.
type Migration struct {
	Version string // 파일명 숫자 prefix (예: "20260522000001")
	Name    string // 전체 파일명 (예: "20260522000001_profiles.sql")
	SQL     string // 파일 내용
}

// parseMigrationFilename은 "20260522000001_profiles.sql"에서
// version="20260522000001", label="profiles"를 추출한다.
// 규칙: ".sql"로 끝나고, 첫 '_' 앞이 비어 있지 않은 순수 숫자여야 하며, label도 비어 있지 않아야 한다.
func parseMigrationFilename(name string) (version, label string, ok bool) {
	if !strings.HasSuffix(name, ".sql") {
		return "", "", false
	}
	base := strings.TrimSuffix(name, ".sql")
	idx := strings.Index(base, "_")
	if idx <= 0 {
		return "", "", false
	}
	version = base[:idx]
	label = base[idx+1:]
	if label == "" {
		return "", "", false
	}
	for _, r := range version {
		if r < '0' || r > '9' {
			return "", "", false
		}
	}
	return version, label, true
}

// pendingMigrations는 available 중 applied에 없는 마이그레이션을 version 오름차순으로 반환한다.
func pendingMigrations(available []Migration, applied map[string]bool) []Migration {
	var pending []Migration
	for _, m := range available {
		if !applied[m.Version] {
			pending = append(pending, m)
		}
	}
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Version < pending[j].Version
	})
	return pending
}
