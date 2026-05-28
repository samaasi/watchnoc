CREATE TABLE users (
    id              BIGINT          PRIMARY KEY,
    external_id     VARCHAR(255)    NOT NULL,
    email           VARCHAR(255)    NOT NULL,
    display_name    VARCHAR(255),
    avatar_url      VARCHAR(1024),
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    CONSTRAINT users_external_id_key UNIQUE (external_id)
);

CREATE INDEX idx_users_email      ON users (email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_deleted_at ON users (deleted_at);
