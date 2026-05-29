package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// Standard Jira issue key pattern: PROJECT_KEY-NUMBER
// Project keys are 2-10 uppercase letters/numbers, starting with a letter.
// Issue numbers are one or more digits.
var defaultTicketKeyPattern = regexp.MustCompile(`\b([A-Z][A-Z0-9]{1,9}-\d+)\b`)

// IssueResolver extracts Jira issue keys from text and resolves them to
// full issue details via the Jira REST API.
type IssueResolver struct {
	clientFactory *ClientFactory
	keyPattern    *regexp.Regexp
}

// NewIssueResolver creates a new IssueResolver.
func NewIssueResolver(clientFactory *ClientFactory) *IssueResolver {
	return &IssueResolver{
		clientFactory: clientFactory,
		keyPattern:    defaultTicketKeyPattern,
	}
}

// ExtractKeys scans a string for Jira issue key patterns and returns all
// unique matches. Input is typically a commit message, branch name, or PR title.
//
// Examples:
//
//	"fix: payment retry ENG-1234"            → ["ENG-1234"]
//	"feat/ENG-456-add-retry-logic"           → ["ENG-456"]
//	"closes ENG-789, relates to ENG-100"     → ["ENG-789", "ENG-100"]
//	"chore: update dependencies"             → []
func (r *IssueResolver) ExtractKeys(text string) []string {
	matches := r.keyPattern.FindAllString(text, -1)

	// Deduplicate while preserving order
	seen := make(map[string]struct{})
	unique := make([]string, 0, len(matches))
	for _, m := range matches {
		if _, exists := seen[m]; !exists {
			seen[m] = struct{}{}
			unique = append(unique, m)
		}
	}

	return unique
}

// ExtractKeyFromURL parses a Jira issue URL and returns the issue key.
// Supports URLs in the format: https://{site}.atlassian.net/browse/{KEY}
//
// Used for Gate 1 manual paste linkage.
func (r *IssueResolver) ExtractKeyFromURL(rawURL string) (string, error) {
	// Normalize: trim whitespace and trailing slashes
	rawURL = strings.TrimSpace(strings.TrimRight(rawURL, "/"))

	// Accept both full URLs and bare keys
	if defaultTicketKeyPattern.MatchString(rawURL) && !strings.Contains(rawURL, "/") {
		// Bare key: "ENG-1234"
		return strings.ToUpper(rawURL), nil
	}

	// Full URL: extract the last path segment
	parts := strings.Split(rawURL, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid Jira URL: %s", rawURL)
	}

	key := parts[len(parts)-1]
	if !defaultTicketKeyPattern.MatchString(key) {
		return "", fmt.Errorf("could not extract a valid Jira issue key from: %s", rawURL)
	}

	return strings.ToUpper(key), nil
}

// ResolveIssue fetches full issue details for a given key from the Jira REST API.
// Uses the Jira Cloud REST API v3 with a targeted field list to minimize payload size.
func (r *IssueResolver) ResolveIssue(ctx context.Context, orgID uint64, key string) (*JiraIssue, error) {
	client, err := r.clientFactory.ForOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get jira client for org %d: %w", orgID, err)
	}

	// Request only the fields DeployGuard needs — reduces payload size
	// and avoids transmitting large custom field data we don't use.
	fields := strings.Join([]string{
		"summary",
		"status",
		"assignee",
		"reporter",
		"issuetype",
		"priority",
		"created",
		"updated",
		"resolutiondate",
		"labels",
		"comment",           // for approval comment evidence
		"customfield_10010", // Sprint field — common custom field
	}, ",")

	path := fmt.Sprintf("/rest/api/3/issue/%s?fields=%s", key, fields)

	resp, err := client.Do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("resolve jira issue %s: %w", key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrIssueNotFound
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := json.Marshal(resp.Body)
		return nil, fmt.Errorf("jira issue %s: unexpected status %d: %s", key, resp.StatusCode, string(body))
	}

	var issue JiraIssueResponse
	if err := json.NewDecoder(resp.Body).Decode(&issue); err != nil {
		return nil, fmt.Errorf("decode jira issue %s: %w", key, err)
	}

	return normaliseIssue(&issue), nil
}

// SearchByCommitRef searches Jira for issues that reference a commit SHA or
// branch name using Jira's development panel integration. This is a Gate 2
// feature — Gate 1 uses key extraction from commit messages only.
//
// Uses JQL (Jira Query Language) to find issues with matching development links.
func (r *IssueResolver) SearchByCommitRef(ctx context.Context, orgID uint64, commitSHA string) ([]*JiraIssue, error) {
	client, err := r.clientFactory.ForOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// JQL: find issues with a development link matching the commit SHA (short form)
	shortSHA := commitSHA
	if len(shortSHA) > 8 {
		shortSHA = shortSHA[:8]
	}

	jql := fmt.Sprintf(
		`issue in linkedIssues() AND text ~ "%s" ORDER BY updated DESC`,
		shortSHA,
	)

	return r.search(ctx, client, jql, 5)
}

// search executes a JQL query and returns the matching issues.
func (r *IssueResolver) search(ctx context.Context, client *Client, jql string, maxResults int) ([]*JiraIssue, error) {
	payload := map[string]any{
		"jql":        jql,
		"maxResults": maxResults,
		"fields":     []string{"summary", "status", "assignee", "issuetype", "priority"},
	}

	data, _ := json.Marshal(payload)
	resp, err := client.Do(ctx, http.MethodPost, "/rest/api/3/issue/search", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("jira search: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Total  int                 `json:"total"`
		Issues []JiraIssueResponse `json:"issues"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode jira search: %w", err)
	}

	issues := make([]*JiraIssue, 0, len(result.Issues))
	for i := range result.Issues {
		issues = append(issues, normaliseIssue(&result.Issues[i]))
	}

	return issues, nil
}

// normaliseIssue converts a raw Jira API response to DeployGuard's internal type.
// This is the boundary — no Jira API response shapes escape this file.
func normaliseIssue(raw *JiraIssueResponse) *JiraIssue {
	issue := &JiraIssue{
		Key:            raw.Key,
		ID:             raw.ID,
		Summary:        raw.Fields.Summary,
		IssueType:      raw.Fields.Issuetype.Name,
		Priority:       raw.Fields.Priority.Name,
		Status:         raw.Fields.Status.Name,
		StatusCategory: raw.Fields.Status.StatusCategory.Name,
		CreatedAt:      raw.Fields.Created,
		UpdatedAt:      raw.Fields.Updated,
		URL:            fmt.Sprintf("https://PLACEHOLDER.atlassian.net/browse/%s", raw.Key),
	}

	if raw.Fields.Assignee != nil {
		issue.AssigneeLogin = raw.Fields.Assignee.AccountID
		issue.AssigneeDisplayName = raw.Fields.Assignee.DisplayName
	}

	if raw.Fields.Reporter != nil {
		issue.ReporterLogin = raw.Fields.Reporter.AccountID
		issue.ReporterDisplayName = raw.Fields.Reporter.DisplayName
	}

	return issue
}
