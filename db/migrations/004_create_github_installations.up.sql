CREATE TABLE github_installations (
    id                      BIGINT          PRIMARY KEY,
    org_id                  BIGINT          NOT NULL REFERENCES orgs(id),
    installation_id         BIGINT          NOT NULL,
    installer_github_login  VARCHAR(255),
    account_login           VARCHAR(255)    NOT NULL,
    account_type            VARCHAR(32)     NOT NULL,
    suspended_at            TIMESTAMPTZ,
    revoked_at              TIMESTAMPTZ,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ,
    CONSTRAINT github_installations_org_id_key          UNIQUE (org_id),
    CONSTRAINT github_installations_installation_id_key UNIQUE (installation_id)
);
