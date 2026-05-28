CREATE TABLE slack_installations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES orgs(id) ON DELETE CASCADE,
    team_id VARCHAR(255) NOT NULL,
    team_name VARCHAR(255),
    bot_user_id VARCHAR(255) NOT NULL,
    bot_token TEXT NOT NULL,
    app_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, team_id)
);
