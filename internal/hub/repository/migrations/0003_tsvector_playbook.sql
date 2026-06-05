-- Migration 0003: 给 hub_playbooks 加 tsvector 全文搜索
-- 包含 title + content + tags，权重：title=A, content=B, tags=B
-- 中文用 'simple' 配置（不依赖任何词典），跨语言场景更稳

BEGIN;

-- 1. 创建 trigger function：自动维护 tsv 列
CREATE OR REPLACE FUNCTION hub.hub_playbooks_tsv_update()
RETURNS TRIGGER AS $$
BEGIN
    NEW.tsv :=
        setweight(to_tsvector('simple', COALESCE(NEW.title, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(NEW.content, '')), 'B') ||
        setweight(to_tsvector('simple', COALESCE(array_to_string(NEW.tags, ' '), '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 2. 创建 trigger（INSERT / UPDATE 时触发）
DROP TRIGGER IF EXISTS hub_playbooks_tsv_trigger ON hub.hub_playbooks;
CREATE TRIGGER hub_playbooks_tsv_trigger
    BEFORE INSERT OR UPDATE OF title, content, tags
    ON hub.hub_playbooks
    FOR EACH ROW
    EXECUTE FUNCTION hub.hub_playbooks_tsv_update();

-- 3. 给已有数据回填 tsv（如果有旧数据）
UPDATE hub.hub_playbooks
SET tsv =
    setweight(to_tsvector('simple', COALESCE(title, '')), 'A') ||
    setweight(to_tsvector('simple', COALESCE(content, '')), 'B') ||
    setweight(to_tsvector('simple', COALESCE(array_to_string(tags, ' '), '')), 'B')
WHERE tsv IS NULL;

-- 4. GIN 索引（用于 tsvector 全文搜索加速）
CREATE INDEX IF NOT EXISTS hub_playbooks_tsv_idx
    ON hub.hub_playbooks USING GIN (to_tsvector('simple', tsv));

-- 记录
INSERT INTO hub.hub_migrations (version) VALUES ('0003_tsvector_playbook') ON CONFLICT DO NOTHING;

COMMIT;
