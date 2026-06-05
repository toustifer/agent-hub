-- Migration 0001: 初始化 hub schema 和 5 张核心表
-- 注意：所有表都在 hub schema 下，不污染 public
-- 运行顺序：必须先 0001，再 0002，再 0003

BEGIN;

-- 1. 创建 schema
CREATE SCHEMA IF NOT EXISTS hub;

-- 2. 创建迁移记录表（hub 自己管理）
CREATE TABLE IF NOT EXISTS hub.hub_migrations (
    version VARCHAR(32) PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 3. 业务注册表
CREATE TABLE hub.hub_businesses (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    repo_url VARCHAR(512),
    owner_user_id BIGINT,
    description VARCHAR(1024),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_hub_businesses_status ON hub.hub_businesses(status);

-- 4. Worker 心跳表
CREATE TABLE hub.hub_workers (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
    worker_id VARCHAR(128) NOT NULL,
    version VARCHAR(32),
    last_heartbeat_at TIMESTAMPTZ,
    status VARCHAR(20) NOT NULL DEFAULT 'offline',
    host VARCHAR(128),
    pid INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(business_id, worker_id)
);
CREATE INDEX idx_hub_workers_status ON hub.hub_workers(status);
CREATE INDEX idx_hub_workers_last_heartbeat ON hub.hub_workers(last_heartbeat_at);

-- 5. 分布式锁表
-- partial unique index 在 0002 migration 加
CREATE TABLE hub.hub_locks (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
    resource_key VARCHAR(256) NOT NULL,
    holder_token VARCHAR(64) NOT NULL,
    holder_worker_id VARCHAR(128) NOT NULL,
    acquired_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    heartbeat_at TIMESTAMPTZ NOT NULL,
    released_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_hub_locks_business ON hub.hub_locks(business_id);
CREATE INDEX idx_hub_locks_expires ON hub.hub_locks(expires_at) WHERE released_at IS NULL;

-- 6. Playbook 知识库表
-- tsvector + GIN 在 0003 migration 加
CREATE TABLE hub.hub_playbooks (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT REFERENCES hub.hub_businesses(id) ON DELETE SET NULL,  -- NULL = 跨业务
    category VARCHAR(32) NOT NULL,
    title VARCHAR(256) NOT NULL,
    content TEXT NOT NULL,
    tags TEXT[],
    tsv VARCHAR(1024),  -- 由 trigger 维护的 tsvector 字符串
    created_by_worker_id VARCHAR(128),
    deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_hub_playbooks_category ON hub.hub_playbooks(category);
CREATE INDEX idx_hub_playbooks_business ON hub.hub_playbooks(business_id);
CREATE INDEX idx_hub_playbooks_tags ON hub.hub_playbooks USING GIN(tags);

-- 7. 审计日志表（append-only）
CREATE TABLE hub.hub_events (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
    actor VARCHAR(128) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_hub_events_type ON hub.hub_events(event_type);
CREATE INDEX idx_hub_events_created ON hub.hub_events(created_at DESC);
CREATE INDEX idx_hub_events_business_created ON hub.hub_events(business_id, created_at DESC);

-- 8. 记录本 migration
INSERT INTO hub.hub_migrations (version) VALUES ('0001_init_schema') ON CONFLICT DO NOTHING;

COMMIT;
