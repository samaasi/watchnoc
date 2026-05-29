CREATE TABLE dora_snapshots (
    id                          BIGINT          PRIMARY KEY,
    org_id                      BIGINT          NOT NULL REFERENCES orgs(id),
    period_start                TIMESTAMPTZ     NOT NULL,
    period_end                  TIMESTAMPTZ     NOT NULL,
    deployment_frequency        FLOAT           NOT NULL DEFAULT 0,
    deployment_frequency_tier   VARCHAR(16)     NOT NULL,
    lead_time_hours             FLOAT           NOT NULL DEFAULT 0,
    lead_time_tier              VARCHAR(16)     NOT NULL,
    mttr_hours                  FLOAT           NOT NULL DEFAULT 0,
    mttr_tier                   VARCHAR(16)     NOT NULL,
    change_failure_rate         FLOAT           NOT NULL DEFAULT 0,
    change_failure_tier         VARCHAR(16)     NOT NULL,
    total_deploys               INT             NOT NULL DEFAULT 0,
    total_incidents             INT             NOT NULL DEFAULT 0,
    created_at                  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_dora_snapshots_org ON dora_snapshots (org_id, period_end DESC);
