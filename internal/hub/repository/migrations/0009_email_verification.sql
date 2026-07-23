-- Migration 0009: email verification for registration
-- Grandfathers existing users as verified. New users start unverified.

BEGIN;

ALTER TABLE hub.hub_users
    ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;

-- Existing accounts are trusted (already using the system)
UPDATE hub.hub_users
SET email_verified_at = now()
WHERE email_verified_at IS NULL;

CREATE TABLE IF NOT EXISTS hub.hub_email_verifications (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES hub.hub_users(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_hub_email_verifications_user
    ON hub.hub_email_verifications(user_id)
    WHERE used_at IS NULL;

INSERT INTO hub.hub_migrations (version) VALUES ('0009_email_verification') ON CONFLICT DO NOTHING;

COMMIT;
