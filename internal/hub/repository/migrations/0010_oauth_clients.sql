-- Migration 0010: OAuth dynamic client registration (RFC 7591) for remote MCP
BEGIN;

CREATE TABLE IF NOT EXISTS hub.hub_oauth_clients (
    client_id          VARCHAR(128) PRIMARY KEY,
    client_name        TEXT,
    redirect_uris      TEXT[] NOT NULL DEFAULT '{}',
    grant_types        TEXT[] NOT NULL DEFAULT '{authorization_code,urn:ietf:params:oauth:grant-type:device_code}',
    response_types     TEXT[] NOT NULL DEFAULT '{code}',
    token_endpoint_auth_method VARCHAR(64) NOT NULL DEFAULT 'none',
    client_secret_hash VARCHAR(128),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO hub.hub_migrations (version) VALUES ('0010_oauth_clients') ON CONFLICT DO NOTHING;

COMMIT;
