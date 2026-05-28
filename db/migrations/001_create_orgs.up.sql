CREATE TABLE orgs (
    id              BIGINT          PRIMARY KEY,
    name            VARCHAR(255)    NOT NULL,
    slug            VARCHAR(48)     NOT NULL,
    plan            VARCHAR(32)     NOT NULL DEFAULT 'starter',
    retention_days  INT             NOT NULL DEFAULT 90,
    trial_ends_at   TIMESTAMPTZ,
    billing_email   VARCHAR(255),
    settings        JSONB           NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    CONSTRAINT orgs_slug_key UNIQUE (slug)
);

CREATE INDEX idx_orgs_slug        ON orgs (slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_orgs_plan        ON orgs (plan) WHERE deleted_at IS NULL;
CREATE INDEX idx_orgs_deleted_at  ON orgs (deleted_at);
