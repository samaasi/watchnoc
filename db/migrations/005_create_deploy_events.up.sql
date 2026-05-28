CREATE TABLE deploy_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    service_name VARCHAR(255) NOT NULL,
    version VARCHAR(255),
    deployer VARCHAR(255),
    commit_sha VARCHAR(255),
    branch VARCHAR(255),
    environment VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    risk_score INT NOT NULL DEFAULT 0,
    risk_level VARCHAR(50) NOT NULL DEFAULT 'low',
    risk_factors JSONB NOT NULL DEFAULT '[]',
    triggered_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deploy_events_org_id ON deploy_events(org_id);
CREATE INDEX idx_deploy_events_triggered_at ON deploy_events(triggered_at DESC);
