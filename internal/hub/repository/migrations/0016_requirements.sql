-- Migration 0016: requirements and comments for non-technical collaboration
BEGIN;

CREATE TABLE IF NOT EXISTS hub.hub_requirements (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL DEFAULT 'draft',
    -- status values: draft, submitted, planning, in_progress, in_review, accepted, rejected, cancelled
    created_by_user_id BIGINT REFERENCES hub.hub_users(id) ON DELETE SET NULL,
    created_by_email TEXT,
    submitted_at TIMESTAMPTZ,
    accepted_at TIMESTAMPTZ,
    accepted_by_user_id BIGINT REFERENCES hub.hub_users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (business_id, id)
);

CREATE INDEX IF NOT EXISTS idx_hub_requirements_business
    ON hub.hub_requirements(business_id);
CREATE INDEX IF NOT EXISTS idx_hub_requirements_status
    ON hub.hub_requirements(business_id, status);
CREATE UNIQUE INDEX IF NOT EXISTS uq_hub_requirements_business_id_id
    ON hub.hub_requirements(business_id, id);

CREATE TABLE IF NOT EXISTS hub.hub_requirement_links (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
    requirement_id BIGINT NOT NULL,
    task_id VARCHAR(128) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (business_id, requirement_id, task_id),
    FOREIGN KEY (business_id, requirement_id) REFERENCES hub.hub_requirements(business_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_hub_req_links_requirement
    ON hub.hub_requirement_links(business_id, requirement_id);
CREATE INDEX IF NOT EXISTS idx_hub_req_links_task
    ON hub.hub_requirement_links(business_id, task_id);

CREATE TABLE IF NOT EXISTS hub.hub_requirement_comments (
    id BIGSERIAL PRIMARY KEY,
    business_id BIGINT NOT NULL REFERENCES hub.hub_businesses(id) ON DELETE CASCADE,
    requirement_id BIGINT NOT NULL,
    author_user_id BIGINT REFERENCES hub.hub_users(id) ON DELETE SET NULL,
    author_email TEXT,
    body TEXT NOT NULL,
    decision VARCHAR(32),
    -- decision values: null (regular comment), accept, reject, changes_requested
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (business_id, requirement_id) REFERENCES hub.hub_requirements(business_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_hub_req_comments
    ON hub.hub_requirement_comments(business_id, requirement_id);

INSERT INTO hub.hub_migrations (version) VALUES ('0016_requirements') ON CONFLICT DO NOTHING;

COMMIT;
