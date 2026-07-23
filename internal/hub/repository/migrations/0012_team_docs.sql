-- Migration 0012: team-visible docs (timely shared company documents)
BEGIN;

CREATE TABLE IF NOT EXISTS hub.hub_team_docs (
  id BIGSERIAL PRIMARY KEY,
  business_id BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
  doc_key VARCHAR(128) NOT NULL,
  title VARCHAR(256) NOT NULL,
  content TEXT NOT NULL DEFAULT '',
  category VARCHAR(64) NOT NULL DEFAULT 'general',
  visibility VARCHAR(16) NOT NULL DEFAULT 'team',
  created_by_user_id BIGINT REFERENCES hub.hub_users(id) ON DELETE SET NULL,
  updated_by_user_id BIGINT REFERENCES hub.hub_users(id) ON DELETE SET NULL,
  created_by_email TEXT,
  updated_by_email TEXT,
  source VARCHAR(32) DEFAULT 'api',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (business_id, doc_key)
);

CREATE INDEX IF NOT EXISTS idx_hub_team_docs_business_updated
  ON hub.hub_team_docs(business_id, updated_at DESC);

INSERT INTO hub.hub_migrations (version) VALUES ('0012_team_docs') ON CONFLICT DO NOTHING;

COMMIT;
