CREATE TABLE org_members (
    id          BIGINT          PRIMARY KEY,
    org_id      BIGINT          NOT NULL REFERENCES orgs(id),
    user_id     BIGINT          NOT NULL REFERENCES users(id),
    role        VARCHAR(32)     NOT NULL DEFAULT 'engineer',
    invited_by  BIGINT          REFERENCES users(id),
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    CONSTRAINT org_members_org_user_key UNIQUE (org_id, user_id)
);

CREATE INDEX idx_org_members_org_id    ON org_members (org_id)  WHERE deleted_at IS NULL;
CREATE INDEX idx_org_members_user_id   ON org_members (user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_org_members_deleted_at ON org_members (deleted_at);
