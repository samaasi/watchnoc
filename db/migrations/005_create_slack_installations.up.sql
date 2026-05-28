CREATE TABLE slack_installations (
    id                    BIGINT          PRIMARY KEY,
    org_id                BIGINT          NOT NULL REFERENCES orgs(id),
    team_id               VARCHAR(32)     NOT NULL,
    team_name             VARCHAR(255),
    bot_user_id           VARCHAR(32),
    bot_token_encrypted   TEXT            NOT NULL,
    default_channel_id    VARCHAR(32),
    revoked_at            TIMESTAMPTZ,
    created_at            TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at            TIMESTAMPTZ,
    CONSTRAINT slack_installations_org_id_key UNIQUE (org_id)
);
