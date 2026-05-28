CREATE TABLE approvals (
    id                      BIGINT          PRIMARY KEY,
    org_id                  BIGINT          NOT NULL REFERENCES orgs(id),
    deploy_event_id         BIGINT          NOT NULL REFERENCES deploy_events(id),
    status                  VARCHAR(32)     NOT NULL DEFAULT 'pending',
    requested_at            TIMESTAMPTZ     NOT NULL,
    expires_at              TIMESTAMPTZ,
    approver_user_id        BIGINT          REFERENCES users(id),
    approver_github_login   VARCHAR(255),
    decided_at              TIMESTAMPTZ,
    channel                 VARCHAR(32),
    comment                 TEXT,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ,
    CONSTRAINT approvals_deploy_event_key UNIQUE (deploy_event_id, status) INCLUDE (id) WHERE status IN ('pending', 'granted', 'rejected')
);

CREATE INDEX idx_approvals_org_id ON approvals (org_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_approvals_deploy_event ON approvals (deploy_event_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_approvals_status ON approvals (status) WHERE deleted_at IS NULL;
