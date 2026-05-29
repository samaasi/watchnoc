
package gitlab

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ParsedDeployEvent is the internal representation of a deploy-relevant
// GitLab event.
type ParsedDeployEvent struct {
	DeduplicationKey string
	EventType        GitLabEventType
	EventUUID        string // GitLab's X-Gitlab-Event-Uuid
	RepoPath         string // "group/repo"
	CommitSHA        string
	CommitMessage    string
	Branch           string
	Environment      string
	AuthorLogin      string
	AuthorEmail      string
	TriggeredAt      time.Time
	RawPayload       json.RawMessage
}

type GitLabEventType string

const (
	EventTypePush       GitLabEventType = "push"
	EventTypeDeployment GitLabEventType = "deployment"
)

// ParsePushEvent translates a GitLab push event payload.
func ParsePushEvent(body []byte) (*ParsedDeployEvent, error) {
	var payload struct {
		ObjectKind string `json:"object_kind"`
		EventUUID  string `json:"event_uuid"`
		Ref        string `json:"ref"`
		After      string `json:"after"`
		Project    struct {
			PathWithNamespace string `json:"path_with_namespace"`
		} `json:"project"`
		Commits []struct {
			ID        string    `json:"id"`
			Message   string    `json:"message"`
			Timestamp time.Time `json:"timestamp"`
			Author    struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"author"`
		} `json:"commits"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal push event: %w", err)
	}

	if len(payload.Commits) == 0 || payload.After == "0000000000000000000000000000000000000000" {
		return nil, ErrNoCommitData
	}

	headCommit := payload.Commits[len(payload.Commits)-1]

	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")
	environment := inferEnvironment(branch)

	return &ParsedDeployEvent{
		DeduplicationKey: fmt.Sprintf("push:%s:%s", headCommit.ID, payload.Project.PathWithNamespace),
		EventType:        EventTypePush,
		EventUUID:        payload.EventUUID,
		RepoPath:         payload.Project.PathWithNamespace,
		CommitSHA:        headCommit.ID,
		CommitMessage:    firstLine(headCommit.Message),
		Branch:           branch,
		Environment:      environment,
		AuthorLogin:      headCommit.Author.Name,
		AuthorEmail:      headCommit.Author.Email,
		TriggeredAt:      headCommit.Timestamp,
		RawPayload:       body,
	}, nil
}

// inferEnvironment maps a branch name to an environment label.
func inferEnvironment(branch string) string {
	branch = strings.ToLower(branch)

	switch branch {
	case "main", "master":
		return "production"
	case "staging", "stage":
		return "staging"
	case "develop", "development", "dev":
		return "dev"
	}

	if strings.HasPrefix(branch, "release/") || strings.HasPrefix(branch, "hotfix/") {
		return "production"
	}
	if strings.HasPrefix(branch, "staging/") {
		return "staging"
	}
	if strings.HasPrefix(branch, "v") || strings.HasPrefix(branch, "refs/tags/v") {
		return "production"
	}

	return "unknown"
}

// firstLine returns the first line of a multi-line string.
func firstLine(s string) string {
	if idx := strings.Index(s, "\n"); idx != -1 {
		return s[:idx]
	}
	return s
}
