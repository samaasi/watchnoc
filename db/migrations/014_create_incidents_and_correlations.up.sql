CREATE TABLE incidents (
    id              BIGINT          PRIMARY KEY,
    org_id          BIGINT          NOT NULL REFERENCES orgs(id),
    external_id     VARCHAR(64)     NOT NULL,
    source          VARCHAR(32)     NOT NULL DEFAULT 'pagerduty',
    title           VARCHAR(512)    NOT NULL,
    severity        VARCHAR(32),
    fired_at        TIMESTAMPTZ     NOT NULL,
    resolved_at     TIMESTAMPTZ,
    raw_payload     JSONB,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    CONSTRAINT incidents_org_external_key UNIQUE (org_id, external_id)
);

CREATE INDEX idx_incidents_org_fired   ON incidents (org_id, fired_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_incidents_resolved_at ON incidents (resolved_at) WHERE resolved_at IS NULL;

CREATE TABLE incident_deploy_correlations (
    id                      BIGINT          PRIMARY KEY,
    incident_id             BIGINT          NOT NULL REFERENCES incidents(id),
    deploy_event_id         BIGINT          NOT NULL REFERENCES deploy_events(id),
    time_delta_seconds      INT             NOT NULL,
    correlation_confidence  VARCHAR(16)     NOT NULL DEFAULT 'low',
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ,
    CONSTRAINT incident_deploy_correlations_key UNIQUE (incident_id, deploy_event_id)
);

CREATE INDEX idx_incident_deploy_corr_deploy ON incident_deploy_correlations (deploy_event_id);
