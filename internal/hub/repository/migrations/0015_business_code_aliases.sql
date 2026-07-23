-- Migration 0015: business code aliases (old slug → new short code, 90-day grace)
BEGIN;

CREATE TABLE IF NOT EXISTS hub.hub_business_code_aliases (
    old_code     VARCHAR(64) PRIMARY KEY,
    business_id  BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at   TIMESTAMPTZ NOT NULL DEFAULT (now() + interval '90 days')
);

CREATE INDEX IF NOT EXISTS idx_hub_business_code_aliases_biz
    ON hub.hub_business_code_aliases(business_id);

CREATE INDEX IF NOT EXISTS idx_hub_business_code_aliases_exp
    ON hub.hub_business_code_aliases(expires_at);

INSERT INTO hub.hub_migrations (version) VALUES ('0015_business_code_aliases') ON CONFLICT DO NOTHING;

COMMIT;
