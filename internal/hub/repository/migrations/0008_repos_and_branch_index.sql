-- Migration 0008: repos, branch index, branch bindings, dag_state (with branch columns)
-- Fully idempotent for prod that already has hub_repos / hub_dag_state without branch columns.

BEGIN;

CREATE TABLE IF NOT EXISTS hub.hub_repos (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
    repo_url TEXT NOT NULL,
    default_branch VARCHAR(128) NOT NULL DEFAULT 'main',
    provider VARCHAR(32) NOT NULL DEFAULT 'generic', -- github|generic
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (business_id, repo_url)
);

ALTER TABLE hub.hub_repos ADD COLUMN IF NOT EXISTS provider VARCHAR(32) DEFAULT 'generic';
ALTER TABLE hub.hub_repos ADD COLUMN IF NOT EXISTS default_branch VARCHAR(128) DEFAULT 'main';
ALTER TABLE hub.hub_repos ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();
CREATE INDEX IF NOT EXISTS idx_hub_repos_business ON hub.hub_repos(business_id);

CREATE TABLE IF NOT EXISTS hub.hub_branches (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
    repo_id BIGINT NOT NULL REFERENCES hub.hub_repos(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    tip_sha VARCHAR(64) NOT NULL DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT false,
    pr_number INT,
    pr_url TEXT,
    pr_state VARCHAR(32),
    last_reporter TEXT,
    source VARCHAR(32) NOT NULL DEFAULT 'report', -- report|github_api|ls_remote
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (repo_id, name)
);

CREATE INDEX IF NOT EXISTS idx_hub_branches_business ON hub.hub_branches(business_id);
CREATE INDEX IF NOT EXISTS idx_hub_branches_name ON hub.hub_branches(business_id, name);

CREATE TABLE IF NOT EXISTS hub.hub_branch_bindings (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
    branch_name TEXT NOT NULL,
    bind_type VARCHAR(32) NOT NULL, -- dag|task|worker|user
    bind_id TEXT NOT NULL,
    worktree_host TEXT,
    head_sha VARCHAR(64),
    status VARCHAR(32) NOT NULL DEFAULT 'active', -- active|idle|stale
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (business_id, bind_type, bind_id)
);

CREATE INDEX IF NOT EXISTS idx_hub_branch_bindings_branch
    ON hub.hub_branch_bindings(business_id, branch_name);

CREATE TABLE IF NOT EXISTS hub.hub_dag_state (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
    task_id VARCHAR(128) NOT NULL,
    title TEXT,
    status VARCHAR(64),
    assigned_worker VARCHAR(128),
    depends_on TEXT[],
    output_files TEXT[],
    branch TEXT,
    head_sha VARCHAR(64),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (business_id, task_id)
);

ALTER TABLE hub.hub_dag_state ADD COLUMN IF NOT EXISTS branch TEXT;
ALTER TABLE hub.hub_dag_state ADD COLUMN IF NOT EXISTS head_sha VARCHAR(64);
ALTER TABLE hub.hub_dag_state ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT now();

INSERT INTO hub.hub_migrations (version) VALUES ('0008_repos_and_branch_index') ON CONFLICT DO NOTHING;

COMMIT;
