CREATE TABLE jira_installations (
    id                      BIGINT          PRIMARY KEY,
    org_id                  BIGINT          NOT NULL REFERENCES orgs(id),
    cloud_id                VARCHAR(255),
    cloud_name              VARCHAR(255),
    cloud_url               VARCHAR(1024),
    scope                   VARCHAR(512),
    access_token_encrypted  TEXT,
    refresh_token_encrypted TEXT,
    token_expires_at        TIMESTAMPTZ,
    revoked_at              TIMESTAMPTZ,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ,
    CONSTRAINT jira_installations_org_id_key UNIQUE (org_id)
);

CREATE INDEX idx_jira_installations_token_expires ON jira_installations (token_expires_at) WHERE token_expires_at IS NOT NULL;
