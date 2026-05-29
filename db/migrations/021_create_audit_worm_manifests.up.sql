CREATE TABLE audit_worm_manifests (
    id                  BIGINT          PRIMARY KEY,
    org_id              BIGINT          NOT NULL REFERENCES orgs(id),
    batch_start_id      BIGINT          NOT NULL,
    batch_end_id        BIGINT          NOT NULL,
    record_count        INT             NOT NULL,
    s3_bucket           VARCHAR(255)    NOT NULL,
    s3_key              VARCHAR(1024)   NOT NULL,
    manifest_hash       VARCHAR(64)     NOT NULL,
    retain_until        TIMESTAMPTZ     NOT NULL,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT audit_worm_manifests_hash_key UNIQUE (manifest_hash)
);

CREATE INDEX idx_worm_manifests_org ON audit_worm_manifests (org_id, created_at DESC);
