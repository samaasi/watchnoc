CREATE TABLE deploy_events (
    id                      BIGINT          PRIMARY KEY,
    org_id                  BIGINT          NOT NULL REFERENCES orgs(id),
    source                  VARCHAR(32)     NOT NULL DEFAULT 'github',
    repo_owner              VARCHAR(255)    NOT NULL,
    repo_name               VARCHAR(255)    NOT NULL,
    commit_sha              VARCHAR(40)     NOT NULL,
    commit_message          TEXT,
    branch                  VARCHAR(255),
    environment             VARCHAR(64)     NOT NULL,
    author_login            VARCHAR(255)    NOT NULL,
    author_email            VARCHAR(255),
    committed_at            TIMESTAMPTZ,
    triggered_at            TIMESTAMPTZ     NOT NULL,
    completed_at            TIMESTAMPTZ,
    duration_seconds        INT,
    risk_score              INT             NOT NULL DEFAULT 0
                                            CHECK (risk_score >= 0 AND risk_score <= 100),
    risk_level              VARCHAR(16)     NOT NULL DEFAULT 'low',
    risk_factors            JSONB           NOT NULL DEFAULT '[]',
    status                  VARCHAR(32)     NOT NULL DEFAULT 'pending',
    deployment_outcome      VARCHAR(32)     NOT NULL DEFAULT 'unknown',
    ci_status               VARCHAR(32)     NOT NULL DEFAULT 'unknown',
    ci_passed_at            TIMESTAMPTZ,
    health_status           VARCHAR(32)     NOT NULL DEFAULT 'unknown',
    health_checked_at       TIMESTAMPTZ,
    files_changed           INT             DEFAULT 0,
    additions               INT             DEFAULT 0,
    deletions               INT             DEFAULT 0,
    affected_services       JSONB           NOT NULL DEFAULT '[]',
    github_deployment_id    BIGINT,
    deploy_log_url          VARCHAR(1024),
    workflow_name           VARCHAR(255),
    deploy_job_name         VARCHAR(255),
    deploy_job_conclusion   VARCHAR(32),
    promotion_chain_id      VARCHAR(64),
    
    ci_job_duration_seconds INT,
    pr_review_duration_seconds INT,
    incident_provider       VARCHAR(32),
    linked_incident_id      VARCHAR(255),
    linked_incident_url     VARCHAR(1024),
    
    voided_at               TIMESTAMPTZ,
    void_reason             VARCHAR(255),
    voided_by_user          BIGINT          REFERENCES users(id),
    reverts_deploy_event_id BIGINT          REFERENCES deploy_events(id),
    raw_payload             JSONB           NOT NULL,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    CONSTRAINT deploy_events_org_id_fkey FOREIGN KEY (org_id) REFERENCES orgs(id)
);

CREATE INDEX idx_deploy_events_org_time
    ON deploy_events (org_id, triggered_at DESC)
    WHERE voided_at IS NULL;

CREATE INDEX idx_deploy_events_org_risk
    ON deploy_events (org_id, risk_level)
    WHERE voided_at IS NULL;

CREATE INDEX idx_deploy_events_org_status
    ON deploy_events (org_id, status)
    WHERE voided_at IS NULL;

CREATE UNIQUE INDEX idx_deploy_events_github_id
    ON deploy_events (org_id, github_deployment_id)
    WHERE github_deployment_id IS NOT NULL AND voided_at IS NULL;

CREATE INDEX idx_deploy_events_org_env_time
    ON deploy_events (org_id, environment, triggered_at DESC)
    WHERE voided_at IS NULL;

CREATE INDEX idx_deploy_events_promotion_chain
    ON deploy_events (promotion_chain_id)
    WHERE promotion_chain_id IS NOT NULL;
    
CREATE TABLE linked_tickets (
    id              BIGINT          PRIMARY KEY,
    deploy_event_id BIGINT          NOT NULL REFERENCES deploy_events(id),
    provider        VARCHAR(32)     NOT NULL,
    ticket_id       VARCHAR(255)    NOT NULL,
    ticket_url      VARCHAR(1024),
    status          VARCHAR(64),
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_linked_tickets_deploy_event ON linked_tickets (deploy_event_id);
