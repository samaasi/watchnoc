CREATE TABLE iso27001_mappings (
    id                  BIGINT          PRIMARY KEY,
    org_id              BIGINT          NOT NULL REFERENCES orgs(id),
    deploy_event_id     BIGINT          NOT NULL REFERENCES deploy_events(id),
    evidence_report_id  BIGINT          REFERENCES evidence_reports(id),
    control_ref         VARCHAR(32)     NOT NULL,
    control_title       VARCHAR(255)    NOT NULL,
    evidence_summary    TEXT,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX idx_iso27001_org_control ON iso27001_mappings (org_id, control_ref) WHERE deleted_at IS NULL;
CREATE INDEX idx_iso27001_report      ON iso27001_mappings (evidence_report_id)   WHERE evidence_report_id IS NOT NULL;
