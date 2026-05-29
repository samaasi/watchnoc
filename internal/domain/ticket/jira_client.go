package ticket

import (
	"context"
)

type JiraClient struct{}

func NewJiraClient() *JiraClient {
	return &JiraClient{}
}

func (c *JiraClient) FetchTicketInfo(ctx context.Context, key string) (*LinkedTicket, error) {
	// Scaffold for Jira API interaction
	return nil, nil
}
