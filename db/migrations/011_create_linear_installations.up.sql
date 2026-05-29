
CREATE TABLE linear_installations (
    id                      BIGINT          PRIMARY KEY,
    org_id                  BIGINT          NOT NULL REFERENCES orgs(id),
    linear_workspace        VARCHAR(255),
    access_token            VARCHAR(512)    NOT NULL,
    webhook_secret          VARCHAR(255),
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ,
    CONSTRAINT linear_installations_org_id_key UNIQUE (org_id)
);
