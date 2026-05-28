CREATE TABLE approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    deploy_event_id UUID NOT NULL REFERENCES deploy_events(id) ON DELETE CASCADE,
    requester_id UUID NOT NULL REFERENCES users(id),
    approver_id UUID REFERENCES users(id),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    channel VARCHAR(50) NOT NULL DEFAULT 'web_app',
    comment TEXT,
    expires_at TIMESTAMP,
    approved_at TIMESTAMP,
    rejected_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_approvals_deploy_event_id ON approvals(deploy_event_id);
CREATE INDEX idx_approvals_status ON approvals(status);
