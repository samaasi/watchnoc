CREATE TABLE gitlab_installations (
    id                        BIGINT          PRIMARY KEY,
    org_id                    BIGINT          NOT NULL REFERENCES orgs(id),
    gitlab_group_id           BIGINT          NOT NULL,
    gitlab_group_path         VARCHAR(255)    NOT NULL,
    access_token_encrypted    TEXT            NOT NULL,
    refresh_token_encrypted   TEXT,
    token_expires_at          TIMESTAMPTZ,
    webhook_secret_encrypted  TEXT            NOT NULL,
    revoked_at                TIMESTAMPTZ,
    created_at                TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at                TIMESTAMPTZ,
    CONSTRAINT gitlab_installations_org_id_key UNIQUE (org_id)
);

CREATE INDEX idx_gitlab_installations_token_expires ON gitlab_installations (token_expires_at) WHERE token_expires_at IS NOT NULL;
