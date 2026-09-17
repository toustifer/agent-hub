-- Migration 0017: 给 hub_playbooks 添加 (business_id, category, title) 唯一索引
-- 支持 sync_handler.go 与 playbook_service.go 中的 ON CONFLICT (business_id, category, title)
CREATE UNIQUE INDEX IF NOT EXISTS uq_hub_playbooks_business_category_title
    ON hub.hub_playbooks (business_id, category, title);
