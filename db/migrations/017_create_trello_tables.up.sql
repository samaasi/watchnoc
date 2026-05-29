CREATE TABLE trello_installations (
    id                      BIGINT          PRIMARY KEY,
    org_id                  BIGINT          NOT NULL REFERENCES orgs(id),
    trello_member_id        VARCHAR(32)     NOT NULL,
    trello_member_name      VARCHAR(255),
    trello_username         VARCHAR(255),
    access_token            VARCHAR(512)    NOT NULL,
    webhook_path_token      VARCHAR(64)     NOT NULL,
    revoked_at              TIMESTAMPTZ,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ,
    CONSTRAINT trello_installations_org_id_key UNIQUE (org_id),
    CONSTRAINT trello_installations_webhook_path_token_key UNIQUE (webhook_path_token)
);

CREATE TABLE trello_webhooks (
    id                      BIGINT          PRIMARY KEY,
    org_id                  BIGINT          NOT NULL REFERENCES orgs(id),
    trello_webhook_id       VARCHAR(32)     NOT NULL,
    board_id                VARCHAR(32)     NOT NULL,
    board_name              VARCHAR(255),
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ,
    CONSTRAINT trello_webhooks_trello_webhook_id_key UNIQUE (trello_webhook_id)
);

CREATE INDEX idx_trello_webhooks_org ON trello_webhooks (org_id);
