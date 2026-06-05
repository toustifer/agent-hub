// Package repository 包含数据访问层。
package repository

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// Migrator 负责按顺序执行 SQL migration。
//
// 工作原理：
//  1. 读 embed.FS 里的所有 .sql 文件，按文件名排序
//  2. 对每个文件，从 SQL 注释里提取 version（首行 "Migration XXXX: ..."）
//  3. 查 hub.hub_migrations 看是否已应用
//  4. 未应用则执行，记录到 hub.hub_migrations
type Migrator struct {
	pool *pgxpool.Pool
}

func NewMigrator(pool *pgxpool.Pool) *Migrator {
	return &Migrator{pool: pool}
}

// Run 跑所有未应用的 migration。
func (m *Migrator) Run(ctx context.Context) error {
	// 确保 hub_migrations 表存在（0001 自己会建，但跑 migrator 之前也要能查）
	if _, err := m.pool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS hub`); err != nil {
		return fmt.Errorf("create hub schema: %w", err)
	}
	if _, err := m.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS hub.hub_migrations (
		    version VARCHAR(64) PRIMARY KEY,
		    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("create hub_migrations table: %w", err)
	}

	// 列出所有 migration 文件
	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	// 查已应用
	applied, err := m.appliedVersions(ctx)
	if err != nil {
		return fmt.Errorf("query applied migrations: %w", err)
	}

	// 按顺序跑
	for _, name := range files {
		version := extractVersion(name)
		if _, ok := applied[version]; ok {
			continue
		}

		sqlBytes, err := fs.ReadFile(migrationFS, "migrations/"+name)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}

		fmt.Printf("[migrator] applying %s ...\n", name)
		if err := m.runOne(ctx, string(sqlBytes), version); err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		fmt.Printf("[migrator] ✓ %s applied\n", name)
	}

	return nil
}

func (m *Migrator) appliedVersions(ctx context.Context) (map[string]bool, error) {
	rows, err := m.pool.Query(ctx, `SELECT version FROM hub.hub_migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]bool)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, err
		}
		result[v] = true
	}
	return result, rows.Err()
}

func (m *Migrator) runOne(ctx context.Context, sql string, version string) error {
	// 用 pgx 跑多语句 SQL：split by ;
	// 简单起见，整个 SQL 包在一个 conn.RunQuery 里
	conn, err := m.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()

	// pgx.Batch 或直接 Exec
	// 多语句需 SimpleProtocol=false（默认）
	_, err = conn.Exec(ctx, sql)
	if err != nil {
		return err
	}

	// 文件内的 INSERT INTO hub_migrations 会自己跑
	// 兜底：如果 SQL 末尾没 INSERT，再补一次
	_, err = m.pool.Exec(ctx,
		`INSERT INTO hub.hub_migrations (version) VALUES ($1) ON CONFLICT DO NOTHING`,
		version)
	return err
}

func extractVersion(filename string) string {
	// 0001_init_schema.sql → 0001_init_schema
	return strings.TrimSuffix(filename, ".sql")
}

// 确保 pgx 被引用（避免 import 错误）
var _ = pgx.ErrNoRows
