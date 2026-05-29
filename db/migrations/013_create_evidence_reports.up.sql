CREATE TABLE evidence_reports (
    id                      BIGINT              PRIMARY KEY,
    org_id                  BIGINT              NOT NULL REFERENCES orgs(id),
    framework               VARCHAR(32)         NOT NULL,
    period_start            TIMESTAMPTZ         NOT NULL,
    period_end              TIMESTAMPTZ         NOT NULL,
    status                  VARCHAR(32)         NOT NULL DEFAULT 'pending',
    summary                 JSONB               NOT NULL DEFAULT '{}',
    s3_key                  VARCHAR(1024),
    generated_by_user_id    BIGINT              REFERENCES users(id),
    failure_reason          TEXT,
    expires_at              TIMESTAMPTZ,
    created_at              TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ
);

CREATE INDEX idx_evidence_reports_org        ON evidence_reports (org_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_evidence_reports_status     ON evidence_reports (status) WHERE status IN ('pending','generating');
CREATE INDEX idx_evidence_reports_expires_at ON evidence_reports (expires_at) WHERE expires_at IS NOT NULL;
