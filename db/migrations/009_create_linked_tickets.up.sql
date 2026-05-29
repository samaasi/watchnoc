CREATE TABLE linked_tickets (
    id                  BIGINT          PRIMARY KEY,
    org_id              BIGINT          NOT NULL REFERENCES orgs(id),
    deploy_event_id     BIGINT          NOT NULL REFERENCES deploy_events(id),
    ticket_url          VARCHAR(1024)   NOT NULL,
    ticket_key          VARCHAR(64),
    ticket_source       VARCHAR(32)     NOT NULL DEFAULT 'manual',
    ticket_metadata     JSONB           NOT NULL DEFAULT '{}',
    linked_by_user_id   BIGINT          REFERENCES users(id),
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE INDEX idx_linked_tickets_deploy ON linked_tickets (deploy_event_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_linked_tickets_key    ON linked_tickets (org_id, ticket_key) WHERE ticket_key IS NOT NULL AND deleted_at IS NULL;
