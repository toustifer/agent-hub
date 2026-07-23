-- Migration 0011: work log attribution (who / role / worker template)
BEGIN;

ALTER TABLE hub.hub_events
  ADD COLUMN IF NOT EXISTS actor_user_id BIGINT REFERENCES hub.hub_users(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS actor_email TEXT,
  ADD COLUMN IF NOT EXISTS actor_role VARCHAR(32),
  ADD COLUMN IF NOT EXISTS actor_worker_id VARCHAR(128);

CREATE INDEX IF NOT EXISTS idx_hub_events_actor_user
  ON hub.hub_events(business_id, actor_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_hub_events_role
  ON hub.hub_events(business_id, actor_role, created_at DESC);

INSERT INTO hub.hub_migrations (version) VALUES ('0011_event_actors') ON CONFLICT DO NOTHING;

COMMIT;
