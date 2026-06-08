-- Migration 0004: 社区市场表 — Community Workers + Reviews
-- 支持跨项目 Worker 发布、浏览、评分

BEGIN;

-- 1. Community Workers 发布表
CREATE TABLE hub.hub_community_workers (
    id BIGSERIAL PRIMARY KEY,
    publisher_user_id BIGINT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    domain VARCHAR(32) NOT NULL,
    scope TEXT,
    handbook JSONB,
    playbooks JSONB,
    tags TEXT[],
    install_count INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'published',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_hub_community_workers_domain ON hub.hub_community_workers(domain);
CREATE INDEX idx_hub_community_workers_install_count ON hub.hub_community_workers(install_count);
CREATE INDEX idx_hub_community_workers_publisher ON hub.hub_community_workers(publisher_user_id);

-- 2. Community Reviews 评分表
CREATE TABLE hub.hub_community_reviews (
    id BIGSERIAL PRIMARY KEY,
    worker_id BIGINT NOT NULL REFERENCES hub.hub_community_workers(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    rating INT NOT NULL,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_hub_community_reviews_worker ON hub.hub_community_reviews(worker_id);

-- 3. 记录迁移
INSERT INTO hub.hub_migrations (version) VALUES ('0004_community_workers') ON CONFLICT DO NOTHING;

COMMIT;
