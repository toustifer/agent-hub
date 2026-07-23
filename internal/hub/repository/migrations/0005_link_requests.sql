-- Migration 0005: 设备链接请求表 — Link Requests
-- 新设备/用户需经现有成员审批才能加入业务项目

BEGIN;

CREATE TABLE hub.hub_link_requests (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL,
    device_info TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    reviewed_by BIGINT,
    reviewed_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX hub_link_requests_pending
    ON hub.hub_link_requests (business_id, user_id)
    WHERE status = 'pending';

CREATE INDEX idx_hub_link_requests_business ON hub.hub_link_requests(business_id, status);

INSERT INTO hub.hub_migrations (version) VALUES ('0005_link_requests') ON CONFLICT DO NOTHING;

COMMIT;
