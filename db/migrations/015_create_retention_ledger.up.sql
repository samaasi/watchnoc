CREATE TABLE retention_ledger (
    id                      BIGINT          PRIMARY KEY,
    org_id                  BIGINT          NOT NULL REFERENCES orgs(id),
    table_name              VARCHAR(64)     NOT NULL,
    records_purged          INT             NOT NULL,
    oldest_purged_at        TIMESTAMPTZ     NOT NULL,
    retention_days_applied  INT             NOT NULL,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_retention_ledger_org ON retention_ledger (org_id, created_at DESC);
