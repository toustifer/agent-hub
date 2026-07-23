-- Migration 0013: shareable worker templates (not runtime processes)
BEGIN;

CREATE TABLE IF NOT EXISTS hub.hub_worker_templates (
  id BIGSERIAL PRIMARY KEY,
  business_id BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
  worker_id VARCHAR(128) NOT NULL,
  display_name VARCHAR(256),
  description TEXT,
  skills TEXT[] NOT NULL DEFAULT '{}',
  handbook JSONB NOT NULL DEFAULT '{}',
  prompt_summary TEXT,
  published_by_user_id BIGINT REFERENCES hub.hub_users(id) ON DELETE SET NULL,
  published_by_email TEXT,
  visibility VARCHAR(16) NOT NULL DEFAULT 'team',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (business_id, worker_id)
);

CREATE INDEX IF NOT EXISTS idx_hub_worker_templates_business
  ON hub.hub_worker_templates(business_id, updated_at DESC);

INSERT INTO hub.hub_migrations (version) VALUES ('0013_worker_templates') ON CONFLICT DO NOTHING;

COMMIT;
