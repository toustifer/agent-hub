-- Migration 0007: Email invites + business join_policy

BEGIN;

CREATE TABLE IF NOT EXISTS hub.hub_invites (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(32) NOT NULL DEFAULT 'member',
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    invited_by BIGINT REFERENCES hub.hub_users(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_by BIGINT REFERENCES hub.hub_users(id) ON DELETE SET NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS hub_invites_pending_email
    ON hub.hub_invites (business_id, lower(email))
    WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS idx_hub_invites_business_status
    ON hub.hub_invites(business_id, status);

CREATE INDEX IF NOT EXISTS idx_hub_invites_token_hash
    ON hub.hub_invites(token_hash);

ALTER TABLE hub.hub_businesses
    ADD COLUMN IF NOT EXISTS join_policy VARCHAR(20) NOT NULL DEFAULT 'invite_only';

INSERT INTO hub.hub_migrations (version) VALUES ('0007_invites_and_join_policy') ON CONFLICT DO NOTHING;

COMMIT;
