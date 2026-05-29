CREATE TABLE service_metadata (
    id              BIGINT          PRIMARY KEY,
    org_id          BIGINT          NOT NULL REFERENCES orgs(id),
    service_name    VARCHAR(255)    NOT NULL,
    hipaa_in_scope  BOOLEAN         NOT NULL DEFAULT false,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    CONSTRAINT uq_org_service UNIQUE (org_id, service_name)
);

CREATE INDEX idx_service_metadata_deleted_at ON service_metadata (deleted_at);
