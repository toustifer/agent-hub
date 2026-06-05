// Package repository 包装 ent 客户端和 migrator。
package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/stifer/agent-hub/ent"
)

// NewClient 创建连接到 PostgreSQL 的 ent 客户端。
//
// 配置要求：
//   - search_path 包含 hub（让 ent 查询的表都在 hub schema 下）
//   - 跑 NewMigrator().Run(ctx) 先建表
func NewClient(ctx context.Context, dsn string) (*ent.Client, *pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("parse dsn: %w", err)
	}

	// 强制 search_path 为 hub
	if cfg.ConnConfig.RuntimeParams == nil {
		cfg.ConnConfig.RuntimeParams = map[string]string{}
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = "hub,public"

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create pool: %w", err)
	}

	// 跑 migration
	m := NewMigrator(pool)
	if err := m.Run(ctx); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("run migrations: %w", err)
	}

	// ent 驱动用 pgx5
	drv := ent.Driver(pool)
	client := ent.NewClient(drv)

	return client, pool, nil
}
