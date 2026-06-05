-- Migration 0002: 加 hub_locks 的 partial unique index
-- 这是锁并发安全的关键：同一个 resource_key 在任何时刻只能有一个活跃锁
-- partial 条件：released_at IS NULL AND expires_at > now()

BEGIN;

-- 删除冗余的普通索引（如果存在）
DROP INDEX IF EXISTS hub.idx_hub_locks_resource;

-- 加 partial unique index
-- 注意：现在所有活跃锁的 resource_key 必须唯一
-- 已过期的锁（expires_at < now()）不受影响，所以 acquire 之前要先清理过期锁
CREATE UNIQUE INDEX hub_locks_active_resource
    ON hub.hub_locks (resource_key)
    WHERE released_at IS NULL AND expires_at > now();

-- 记录
INSERT INTO hub.hub_migrations (version) VALUES ('0002_partial_unique_locks') ON CONFLICT DO NOTHING;

COMMIT;
