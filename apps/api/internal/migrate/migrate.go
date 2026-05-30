// Package migrate — Supabase가 쓰는 supabase_migrations.schema_migrations 이력 테이블을
// 공유하는 forward-only 마이그레이터. release_command(/app/migrate)로 배포 전 적용한다.
package migrate

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
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

// Load는 dir의 *.sql 마이그레이션을 읽어 version 오름차순으로 반환한다.
// 파일명이 parseMigrationFilename 규칙에 맞지 않으면 조용히 건너뛴다.
func Load(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}
	var migs []Migration
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		version, _, ok := parseMigrationFilename(e.Name())
		if !ok {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}
		migs = append(migs, Migration{Version: version, Name: e.Name(), SQL: string(b)})
	}
	sort.Slice(migs, func(i, j int) bool { return migs[i].Version < migs[j].Version })
	return migs, nil
}

// ensureHistory는 이력 스키마·테이블을 보장한다(멱등). 각 문장은 인자 없는 단일 statement.
func ensureHistory(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, `create schema if not exists supabase_migrations`); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	if _, err := pool.Exec(ctx, `create table if not exists supabase_migrations.schema_migrations (
		version text primary key,
		statements text[],
		name text
	)`); err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	return nil
}

// appliedVersions는 이미 적용된 version 집합을 반환한다.
func appliedVersions(ctx context.Context, pool *pgxpool.Pool) (map[string]bool, error) {
	rows, err := pool.Query(ctx, `select version from supabase_migrations.schema_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	applied := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		applied[v] = true
	}
	return applied, rows.Err()
}

// applyOne은 마이그레이션 하나를 단일 트랜잭션으로 적용하고 이력에 기록한다.
// 주의: m.SQL은 다중 statement일 수 있으므로 인자 없이 Exec한다
// (pgx는 인자 0개일 때 simple query protocol을 써서 다중 statement를 허용한다).
// 이력 INSERT는 $1/$2/$3 인자가 있어 extended protocol·단일 statement다.
func applyOne(ctx context.Context, pool *pgxpool.Pool, m Migration) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, m.SQL); err != nil {
		return fmt.Errorf("exec %s: %w", m.Name, err)
	}
	if _, err := tx.Exec(ctx,
		`insert into supabase_migrations.schema_migrations (version, name, statements)
		 values ($1, $2, $3) on conflict (version) do nothing`,
		m.Version, m.Name, []string{m.SQL}); err != nil {
		return fmt.Errorf("record %s: %w", m.Name, err)
	}
	return tx.Commit(ctx)
}

// Run은 dir의 미적용 마이그레이션을 version 순으로 적용한다(forward-only, 멱등).
// 어느 하나라도 실패하면 즉시 에러를 반환한다 → release_command 비0 exit → 배포 중단(fail-closed).
func Run(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if err := ensureHistory(ctx, pool); err != nil {
		return fmt.Errorf("ensure history: %w", err)
	}
	available, err := Load(dir)
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return fmt.Errorf("applied versions: %w", err)
	}
	pending := pendingMigrations(available, applied)
	if len(pending) == 0 {
		slog.Info("migrate: no pending migrations", "total", len(available))
		return nil
	}
	for _, m := range pending {
		if err := applyOne(ctx, pool, m); err != nil {
			return fmt.Errorf("apply %s: %w", m.Name, err)
		}
		slog.Info("migrate: applied", "version", m.Version, "name", m.Name)
	}
	slog.Info("migrate: done", "applied", len(pending), "total", len(available))
	return nil
}
