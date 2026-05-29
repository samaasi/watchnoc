CREATE TABLE sso_configs (
    id                  BIGINT          PRIMARY KEY,
    org_id              BIGINT          NOT NULL REFERENCES orgs(id),
    protocol            VARCHAR(16)     NOT NULL,
    is_enforced         BOOLEAN         NOT NULL DEFAULT FALSE,
    config              JSONB           NOT NULL,
    domain              VARCHAR(255)    NOT NULL,
    setup_by_user_id    BIGINT          REFERENCES users(id),
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    CONSTRAINT sso_configs_org_id_key UNIQUE (org_id)
);

CREATE INDEX idx_sso_configs_domain ON sso_configs (domain) WHERE deleted_at IS NULL;
