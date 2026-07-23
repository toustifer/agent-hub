-- Migration 0006: Auth foundation tables (users, memberships, api_keys, device_codes)
-- Code already used these tables; this makes clean installs reproducible.
-- IF NOT EXISTS / ADD COLUMN IF NOT EXISTS for prod compatibility.
-- NOTE: prod may already have ad-hoc auth tables with different column shapes.

BEGIN;

CREATE TABLE IF NOT EXISTS hub.hub_users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS hub.hub_memberships (
    user_id BIGINT NOT NULL REFERENCES hub.hub_users(id) ON DELETE CASCADE,
    business_id BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
    role VARCHAR(32) NOT NULL DEFAULT 'member',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, business_id)
);
CREATE INDEX IF NOT EXISTS idx_hub_memberships_business ON hub.hub_memberships(business_id);

CREATE TABLE IF NOT EXISTS hub.hub_api_keys (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
    key_hash VARCHAR(64) NOT NULL,
    label TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ,
    UNIQUE (business_id, key_hash)
);
-- Prod may have older hub_api_keys without revoked_at / label
ALTER TABLE hub.hub_api_keys ADD COLUMN IF NOT EXISTS revoked_at TIMESTAMPTZ;
ALTER TABLE hub.hub_api_keys ADD COLUMN IF NOT EXISTS label TEXT;
ALTER TABLE hub.hub_api_keys ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT now();
CREATE INDEX IF NOT EXISTS idx_hub_api_keys_business ON hub.hub_api_keys(business_id) WHERE revoked_at IS NULL;

CREATE TABLE IF NOT EXISTS hub.hub_device_codes (
    code VARCHAR(16) PRIMARY KEY,
    token TEXT,
    user_id BIGINT REFERENCES hub.hub_users(id) ON DELETE SET NULL,
    confirmed BOOLEAN NOT NULL DEFAULT false,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Prod may have older device_codes without user_id column
ALTER TABLE hub.hub_device_codes ADD COLUMN IF NOT EXISTS user_id BIGINT;
ALTER TABLE hub.hub_device_codes ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT now();
ALTER TABLE hub.hub_device_codes ADD COLUMN IF NOT EXISTS confirmed BOOLEAN DEFAULT false;
ALTER TABLE hub.hub_device_codes ADD COLUMN IF NOT EXISTS token TEXT;

INSERT INTO hub.hub_migrations (version) VALUES ('0006_auth_foundation') ON CONFLICT DO NOTHING;

COMMIT;
