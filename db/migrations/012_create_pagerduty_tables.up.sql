
CREATE TABLE pagerduty_installations (
    id                      BIGINT          PRIMARY KEY,
    org_id                  BIGINT          NOT NULL REFERENCES orgs(id),
    webhook_secret          VARCHAR(255)    NOT NULL,
    account_id              VARCHAR(64),
    revoked_at              TIMESTAMPTZ,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ,
    CONSTRAINT pagerduty_installations_org_id_key UNIQUE (org_id)
);

CREATE TABLE pagerduty_service_maps (
    id                      BIGINT          PRIMARY KEY,
    org_id                  BIGINT          NOT NULL REFERENCES orgs(id),
    pagerduty_service       VARCHAR(255)    NOT NULL,
    deployguard_service     VARCHAR(255)    NOT NULL,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ
);
