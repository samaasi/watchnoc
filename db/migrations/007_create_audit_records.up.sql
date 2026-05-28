CREATE TABLE audit_records (
    id              BIGINT          PRIMARY KEY,
    org_id          BIGINT          NOT NULL REFERENCES orgs(id),
    event_type      VARCHAR(64)     NOT NULL,
    actor_user_id   BIGINT          REFERENCES users(id),
    actor_service   VARCHAR(64),
    resource_type   VARCHAR(64)     NOT NULL,
    resource_id     VARCHAR(64)     NOT NULL,
    payload         JSONB           NOT NULL,
    record_hash     VARCHAR(64)     NOT NULL,
    previous_hash   VARCHAR(64),
    occurred_at     TIMESTAMPTZ     NOT NULL,
    trace_id        VARCHAR(64),
    ip_address      VARCHAR(45),
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT audit_records_hash_key UNIQUE (record_hash)
);

CREATE INDEX idx_audit_records_org_time
    ON audit_records (org_id, occurred_at DESC);

CREATE INDEX idx_audit_records_org_event
    ON audit_records (org_id, event_type, occurred_at DESC);

CREATE INDEX idx_audit_records_resource
    ON audit_records (resource_type, resource_id);

CREATE INDEX idx_audit_records_org_created
    ON audit_records (org_id, created_at DESC);

CREATE INDEX idx_audit_records_prev_hash
    ON audit_records (previous_hash)
    WHERE previous_hash IS NOT NULL;
