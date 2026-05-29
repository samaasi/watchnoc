CREATE TABLE approval_policies (
    id                  BIGINT          PRIMARY KEY,
    org_id              BIGINT          NOT NULL REFERENCES orgs(id),
    version             INT             NOT NULL DEFAULT 1,
    is_active           BOOLEAN         NOT NULL DEFAULT FALSE,
    rules               JSONB           NOT NULL DEFAULT '[]',
    description         TEXT,
    created_by_user_id  BIGINT          REFERENCES users(id),
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    CONSTRAINT approval_policies_org_version_key UNIQUE (org_id, version)
);

CREATE UNIQUE INDEX idx_approval_policies_active
    ON approval_policies (org_id)
    WHERE is_active = TRUE AND deleted_at IS NULL;
