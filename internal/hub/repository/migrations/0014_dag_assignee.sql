-- Migration 0014: task board assignee (GitHub Issue-style owner email)
BEGIN;

ALTER TABLE hub.hub_dag_state
  ADD COLUMN IF NOT EXISTS assignee_user_id BIGINT REFERENCES hub.hub_users(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS assignee_email TEXT;

CREATE INDEX IF NOT EXISTS idx_hub_dag_assignee
  ON hub.hub_dag_state(business_id, assignee_user_id)
  WHERE assignee_user_id IS NOT NULL;

INSERT INTO hub.hub_migrations (version) VALUES ('0014_dag_assignee') ON CONFLICT DO NOTHING;

COMMIT;
