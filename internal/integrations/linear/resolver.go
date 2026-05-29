
package linear

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

var LinearTicketRegex = regexp.MustCompile(`(?i)\b([A-Z]+-\d+)\b`)

// IssueResolver extracts Linear issue keys and resolves them to full issue details.
type IssueResolver struct {
	clientFactory *ClientFactory
}

// NewIssueResolver creates a new IssueResolver.
func NewIssueResolver(clientFactory *ClientFactory) *IssueResolver {
	return &IssueResolver{
		clientFactory: clientFactory,
	}
}

// ExtractLinearTickets extracts Linear issue keys from PR title, branch name, and commit message
// in order of priority.
func ExtractLinearTickets(prTitle, branchName, commitMessage string) []string {
	if match := LinearTicketRegex.FindString(prTitle); match != "" {
		return []string{strings.ToUpper(match)}
	}
	if match := LinearTicketRegex.FindString(branchName); match != "" {
		return []string{strings.ToUpper(match)}
	}
	if match := LinearTicketRegex.FindString(commitMessage); match != "" {
		return []string{strings.ToUpper(match)}
	}
	return nil
}

// ExtractKeyFromURL parses a Linear issue URL and returns the issue key.
func (r *IssueResolver) ExtractKeyFromURL(rawInput string) (string, error) {
	rawInput = strings.TrimSpace(strings.TrimRight(rawInput, "/"))
	if LinearTicketRegex.MatchString(rawInput) && !strings.Contains(rawInput, "/") {
		return strings.ToUpper(rawInput), nil
	}
	parts := strings.Split(rawInput, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid linear url")
	}
	key := parts[len(parts)-1]
	if !LinearTicketRegex.MatchString(key) {
		return "", fmt.Errorf("invalid linear issue key")
	}
	return strings.ToUpper(key), nil
}

// ResolveIssue fetches full issue details for a given key from the Linear API.
func (r *IssueResolver) ResolveIssue(ctx context.Context, orgID uint64, key string) (*LinearIssue, error) {
	client, err := r.clientFactory.ForOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get linear client: %w", err)
	}
	issue, err := client.GetIssue(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("resolve linear issue: %w", err)
	}
	return issue, nil
}
