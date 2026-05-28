CREATE TABLE audit_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    event_type VARCHAR(255) NOT NULL,
    event_data JSONB NOT NULL,
    record_hash VARCHAR(255) NOT NULL,
    previous_hash VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_records_org_id ON audit_records(org_id);
CREATE INDEX idx_audit_records_created_at ON audit_records(created_at DESC);
