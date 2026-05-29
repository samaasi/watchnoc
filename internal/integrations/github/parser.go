// internal/integrations/github/parser.go

package github

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ParsedDeployEvent is the internal representation of a deploy-relevant
// GitHub event. All three event types (push, deployment, deployment_status)
// are normalised to this shape.
//
// This type is defined in this package and consumed by the deploy domain
// via the DeployProcessor interface. No GitHub SDK types escape this package.
type ParsedDeployEvent struct {
	// DeduplicationKey is used to prevent duplicate DeployEvent records.
	// For deployment events: "deployment:{github_deployment_id}"
	// For push events: "push:{commit_sha}:{repo_full_name}"
	DeduplicationKey string

	EventType  GitHubEventType
	DeliveryID string

	// Repository
	RepoOwner    string
	RepoName     string
	RepoFullName string // "owner/name"

	// Commit
	CommitSHA     string
	CommitMessage string
	Branch        string
	Environment   string // "production", "staging", etc.

	// Author — PII, tagged in deploy/model.go
	AuthorLogin string
	AuthorEmail string

	// Timing
	TriggeredAt time.Time

	// Diff stats — populated by API enrichment, not from webhook payload
	FilesChanged int
	Additions    int
	Deletions    int

	// GitHub-specific IDs for cross-referencing
	GitHubDeploymentID *int64 // set for deployment events, null for push events
	GitHubRunID        *int64 // Actions run ID if available

	// Outcome — set by deployment_status events
	DeploymentState string // "success" | "failure" | "error" | "pending" | "in_progress"
	CommittedAt     *time.Time
	CompletedAt     *time.Time

	// Workflow info
	WorkflowName string
	DeployLogURL string

	// API Enriched Fields
	LinkedPRs        []LinkedPR
	AffectedServices []string

	// Raw payload — stored in deploy_events.raw_payload for reprocessing
	RawPayload json.RawMessage
}

type GitHubEventType string

const (
	EventTypePush             GitHubEventType = "push"
	EventTypeDeployment       GitHubEventType = "deployment"
	EventTypeDeploymentStatus GitHubEventType = "deployment_status"
	EventTypeWorkflowRun      GitHubEventType = "workflow_run" // Gate 2
	EventTypeCheckSuite       GitHubEventType = "check_suite"  // Gate 2
)

// ParsePushEvent translates a GitHub push event payload.
//
// Push events fire on every git push, not just deploys. We filter to
// production-relevant pushes in the deploy service — the parser does no
// filtering. The parser's job is translation, not business logic.
//
// NOTE: Relying purely on push events to infer deployments can be brittle
// in modern CI/CD (e.g., a push triggers a 20-min test suite that might fail).
// To handle GitHub Actions more accurately, we strongly recommend users
// add the `environment: production` tag to their deploy jobs (which triggers
// reliable `deployment` events). In Gate 2, we will expand this to ingest
// `workflow_run` events for deeper CI/CD accuracy.
func ParsePushEvent(body []byte) (*ParsedDeployEvent, error) {
	// GitHub push event payload shape
	var payload struct {
		Ref        string `json:"ref"`    // "refs/heads/main"
		After      string `json:"after"`  // commit SHA after push
		Before     string `json:"before"` // commit SHA before push
		Forced     bool   `json:"forced"`
		Repository struct {
			Name          string `json:"name"`
			FullName      string `json:"full_name"`
			DefaultBranch string `json:"default_branch"`
			Owner         struct {
				Login string `json:"login"`
				Email string `json:"email"`
			} `json:"owner"`
		} `json:"repository"`
		Pusher struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"pusher"`
		HeadCommit *struct {
			ID        string    `json:"id"`
			Message   string    `json:"message"`
			Timestamp time.Time `json:"timestamp"`
			Author    struct {
				Name     string `json:"name"`
				Email    string `json:"email"`
				Username string `json:"username"`
			} `json:"author"`
			Added    []string `json:"added"`
			Removed  []string `json:"removed"`
			Modified []string `json:"modified"`
		} `json:"head_commit"`
		Commits []struct {
			ID     string `json:"id"`
			Author struct {
				Username string `json:"username"`
				Email    string `json:"email"`
			} `json:"author"`
		} `json:"commits"`
		Installation *struct {
			ID int64 `json:"id"`
		} `json:"installation"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal push event: %w", err)
	}

	if payload.HeadCommit == nil {
		// Happens on branch deletion pushes — no commit to process
		return nil, ErrNoCommitData
	}

	// Extract branch name from ref: "refs/heads/main" → "main"
	branch := strings.TrimPrefix(payload.Ref, "refs/heads/")
	branch = strings.TrimPrefix(branch, "refs/tags/")

	// Determine environment from branch name.
	// Default heuristic — overridden by org-specific config in Gate 2.
	environment := inferEnvironment(branch, payload.Repository.DefaultBranch)

	// Count changed files across all file change lists
	changedFiles := payload.HeadCommit.Added
	changedFiles = append(changedFiles, payload.HeadCommit.Removed...)
	changedFiles = append(changedFiles, payload.HeadCommit.Modified...)

	authorLogin := payload.HeadCommit.Author.Username
	if authorLogin == "" {
		authorLogin = payload.Pusher.Name
	}

	return &ParsedDeployEvent{
		DeduplicationKey: fmt.Sprintf("push:%s:%s", payload.HeadCommit.ID, payload.Repository.FullName),
		EventType:        EventTypePush,
		RepoOwner:        payload.Repository.Owner.Login,
		RepoName:         payload.Repository.Name,
		RepoFullName:     payload.Repository.FullName,
		CommitSHA:        payload.HeadCommit.ID,
		CommitMessage:    firstLine(payload.HeadCommit.Message),
		Branch:           branch,
		Environment:      environment,
		AuthorLogin:      authorLogin,
		AuthorEmail:      payload.HeadCommit.Author.Email,
		CommittedAt:      &payload.HeadCommit.Timestamp,
		TriggeredAt:      payload.HeadCommit.Timestamp,
		RawPayload:       body,
	}, nil
}

// ParseDeploymentEvent translates a GitHub Deployments API event.
// These are more structured than push events — they have an explicit environment
// field and a numeric deployment ID used for status tracking.
func ParseDeploymentEvent(body []byte) (*ParsedDeployEvent, error) {
	var payload struct {
		Action     string `json:"action"` // "created"
		Deployment struct {
			ID          int64  `json:"id"`
			SHA         string `json:"sha"`
			Ref         string `json:"ref"`
			Task        string `json:"task"`        // "deploy"
			Environment string `json:"environment"` // explicit — much better than push
			Description string `json:"description"`
			Creator     struct {
				Login string `json:"login"`
				Email string `json:"email"`
			} `json:"creator"`
			CreatedAt time.Time `json:"created_at"`
			Payload   any       `json:"payload"` // arbitrary deploy metadata
		} `json:"deployment"`
		Repository struct {
			Name     string `json:"name"`
			FullName string `json:"full_name"`
			Owner    struct {
				Login string `json:"login"`
			} `json:"owner"`
		} `json:"repository"`
		Sender struct {
			Login string `json:"login"`
		} `json:"sender"`
		Installation *struct {
			ID int64 `json:"id"`
		} `json:"installation"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal deployment event: %w", err)
	}

	deployID := payload.Deployment.ID

	return &ParsedDeployEvent{
		// Deployment events have a stable GitHub ID — ideal deduplication key
		DeduplicationKey:   fmt.Sprintf("deployment:%d", deployID),
		EventType:          EventTypeDeployment,
		RepoOwner:          payload.Repository.Owner.Login,
		RepoName:           payload.Repository.Name,
		RepoFullName:       payload.Repository.FullName,
		CommitSHA:          payload.Deployment.SHA,
		Branch:             payload.Deployment.Ref,
		Environment:        payload.Deployment.Environment, // authoritative — no inference needed
		AuthorLogin:        payload.Deployment.Creator.Login,
		AuthorEmail:        payload.Deployment.Creator.Email,
		TriggeredAt:        payload.Deployment.CreatedAt,
		GitHubDeploymentID: &deployID,
		RawPayload:         body,
	}, nil
}

// ParseDeploymentStatusEvent translates a deployment_status event.
// These arrive after the deployment completes (or fails) and carry the outcome.
// They update an existing DeployEvent record — they do not create a new one.
func ParseDeploymentStatusEvent(body []byte) (*ParsedDeployStatusEvent, error) {
	var payload struct {
		Action           string `json:"action"` // always "created" for status events
		DeploymentStatus struct {
			ID          int64     `json:"id"`
			State       string    `json:"state"` // "success"|"failure"|"error"|"pending"|"in_progress"
			Description string    `json:"description"`
			LogURL      string    `json:"log_url"`
			CreatedAt   time.Time `json:"created_at"`
		} `json:"deployment_status"`
		Deployment struct {
			ID          int64  `json:"id"`
			SHA         string `json:"sha"`
			Environment string `json:"environment"`
		} `json:"deployment"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
		Installation *struct {
			ID int64 `json:"id"`
		} `json:"installation"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal deployment_status event: %w", err)
	}

	return &ParsedDeployStatusEvent{
		GitHubDeploymentID: payload.Deployment.ID,
		State:              payload.DeploymentStatus.State,
		Description:        payload.DeploymentStatus.Description,
		LogURL:             payload.DeploymentStatus.LogURL,
		OccurredAt:         payload.DeploymentStatus.CreatedAt,
		RepoFullName:       payload.Repository.FullName,
		RawPayload:         body,
	}, nil
}

// ParsedDeployStatusEvent is the outcome update for an existing deployment.
type ParsedDeployStatusEvent struct {
	GitHubDeploymentID int64
	State              string // "success" | "failure" | "error" | "pending" | "in_progress"
	Description        string
	LogURL             string
	OccurredAt         time.Time
	RepoFullName       string
	RawPayload         json.RawMessage
}

// ParsePREvent extracts pull request metadata for risk scoring support.
func ParsePREvent(body []byte) (*ParsedPREvent, error) {
	var payload struct {
		Action      string `json:"action"` // "opened"|"closed"|"merged"|"synchronize"
		Number      int    `json:"number"`
		PullRequest struct {
			ID     int64  `json:"id"`
			Number int    `json:"number"`
			State  string `json:"state"`
			Merged bool   `json:"merged"`
			Head   struct {
				SHA   string `json:"sha"`
				Ref   string `json:"ref"`
				Label string `json:"label"`
			} `json:"head"`
			Base struct {
				Ref string `json:"ref"`
			} `json:"base"`
			User struct {
				Login string `json:"login"`
			} `json:"user"`
			MergedAt  *time.Time `json:"merged_at"`
			CreatedAt time.Time  `json:"created_at"`
			Title     string     `json:"title"`
			Body      string     `json:"body"`
		} `json:"pull_request"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
		Installation *struct {
			ID int64 `json:"id"`
		} `json:"installation"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal pull_request event: %w", err)
	}

	return &ParsedPREvent{
		RepoFullName: payload.Repository.FullName,
		Number:       payload.PullRequest.Number,
		HeadSHA:      payload.PullRequest.Head.SHA,
		HeadBranch:   payload.PullRequest.Head.Ref,
		BaseBranch:   payload.PullRequest.Base.Ref,
		AuthorLogin:  payload.PullRequest.User.Login,
		State:        payload.PullRequest.State,
		Merged:       payload.PullRequest.Merged,
		MergedAt:     payload.PullRequest.MergedAt,
		Title:        payload.PullRequest.Title,
		Action:       payload.Action,
	}, nil
}

type ParsedPREvent struct {
	RepoFullName string
	Number       int
	HeadSHA      string
	HeadBranch   string
	BaseBranch   string
	AuthorLogin  string
	State        string
	Merged       bool
	MergedAt     *time.Time
	Title        string
	Action       string
}

// ParseWorkflowRunEvent translates a GitHub Actions workflow_run event payload
// into a ParsedDeployEvent. This is the Gate 2 path for customers whose CI/CD
// pipelines do not use the GitHub Deployments API.
//
// CRITICAL: workflow_run fires for every conclusion state. This parser returns
// ErrWorkflowNotSuccess when conclusion != "success" — the caller must treat
// this as a signal to discard the event, not as a processing error.
//
// A second gate exists beyond conclusion: workflow name matching. This parser
// extracts the workflow name but does NOT enforce the name allow-list — that
// is the responsibility of the deploy service, which has access to per-org
// configuration. The parser's job is translation only.
//
// Fields extracted and why:
//
//	conclusion   — "success" is the only valid deploy signal. Any other value
//	               (failure, cancelled, skipped, timed_out) must be discarded.
//
//	head_sha     — The commit SHA the workflow ran against. This is the canonical
//	               link back to a PR via the enrichment client's
//	               ListPullRequestsWithCommit call — same as push events.
//
//	head_branch  — Used for environment inference (inferEnvironment) when the
//	               workflow does not declare an environment explicitly.
//
//	name         — The workflow's display name from the YAML `name:` field.
//	               Used by the deploy service to match against the org's
//	               configured allow-list of deploy workflow names.
//
//	id (run ID)  — Stored as GitHubRunID in ParsedDeployEvent. Used to
//	               construct the workflow run URL for the audit trail and
//	               to deduplicate re-deliveries.
//
//	event        — The event that triggered the workflow: "push",
//	               "pull_request", "workflow_dispatch", "schedule", etc.
//	               Stored for observability — does not affect processing.
//
//	created_at   — Workflow run creation timestamp, used as TriggeredAt on
//	               the DeployEvent when no deployment event is available.
func ParseWorkflowRunEvent(body []byte) (*ParsedDeployEvent, error) {
	var payload struct {
		Action      string `json:"action"` // always "completed" for finished runs
		WorkflowRun struct {
			ID         int64     `json:"id"`
			Name       string    `json:"name"`        // workflow display name, e.g. "Deploy Production"
			HeadBranch string    `json:"head_branch"` // branch the workflow ran on
			HeadSHA    string    `json:"head_sha"`    // commit SHA — key for PR linkage
			Event      string    `json:"event"`       // triggering event: "push", "workflow_dispatch", etc.
			Status     string    `json:"status"`      // "completed" by the time this webhook fires
			Conclusion string    `json:"conclusion"`  // "success"|"failure"|"cancelled"|"skipped"|"timed_out"|"action_required"
			HTMLURL    string    `json:"html_url"`    // link to the run in the GitHub UI — stored in audit log
			CreatedAt  time.Time `json:"created_at"`
			UpdatedAt  time.Time `json:"updated_at"`
			Actor      struct {
				Login string `json:"login"`
			} `json:"actor"` // the user or bot that triggered the run
			HeadCommit *struct {
				ID        string    `json:"id"` // same as HeadSHA — use HeadSHA instead
				Message   string    `json:"message"`
				Timestamp time.Time `json:"timestamp"`
				Author    struct {
					Name  string `json:"name"`
					Email string `json:"email"`
				} `json:"author"`
			} `json:"head_commit"`
		} `json:"workflow_run"`
		Repository struct {
			Name     string `json:"name"`
			FullName string `json:"full_name"`
			Owner    struct {
				Login string `json:"login"`
			} `json:"owner"`
			DefaultBranch string `json:"default_branch"`
		} `json:"repository"`
		Sender struct {
			Login string `json:"login"`
		} `json:"sender"`
		Installation *struct {
			ID int64 `json:"id"`
		} `json:"installation"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal workflow_run event: %w", err)
	}

	wr := payload.WorkflowRun

	// PRIMARY GATE: discard every conclusion except "success".
	// "failure", "cancelled", "skipped", "timed_out", and "action_required"
	// all mean the deploy did not happen. This check must remain here — do
	// not move it to the worker, as it keeps non-deploy events out of the queue.
	if wr.Conclusion != "success" {
		return nil, ErrWorkflowNotSuccess
	}

	if wr.HeadSHA == "" {
		return nil, fmt.Errorf("workflow_run event missing head_sha for run %d", wr.ID)
	}

	// Infer environment from branch name.
	// Workflow runs have no explicit environment field unless the job declares one,
	// which emits a deployment event instead (the preferred path). Here we fall back
	// to the same heuristic used for push events.
	environment := inferEnvironment(wr.HeadBranch, payload.Repository.DefaultBranch)

	authorLogin := wr.Actor.Login
	var authorEmail string
	var commitMessage string
	var committedAt *time.Time

	if wr.HeadCommit != nil {
		authorEmail = wr.HeadCommit.Author.Email
		commitMessage = firstLine(wr.HeadCommit.Message)
		committedAt = &wr.HeadCommit.Timestamp
	}

	// TriggeredAt is when the CI pipeline started
	triggeredAt := wr.CreatedAt

	runID := wr.ID

	return &ParsedDeployEvent{
		// Deduplication key uses the run ID — stable across webhook retries
		DeduplicationKey: fmt.Sprintf("workflow_run:%d", wr.ID),
		EventType:        EventTypeWorkflowRun,
		RepoOwner:        payload.Repository.Owner.Login,
		RepoName:         payload.Repository.Name,
		RepoFullName:     payload.Repository.FullName,
		CommitSHA:        wr.HeadSHA,
		CommitMessage:    commitMessage,
		Branch:           wr.HeadBranch,
		Environment:      environment,
		AuthorLogin:      authorLogin,
		AuthorEmail:      authorEmail,
		CommittedAt:      committedAt,
		TriggeredAt:      triggeredAt,
		GitHubRunID:      &runID,
		WorkflowName:     wr.Name,
		DeployLogURL:     wr.HTMLURL,
		CompletedAt:      &wr.UpdatedAt,
		RawPayload:       body,
	}, nil
}

// ErrWorkflowNotSuccess is returned by ParseWorkflowRunEvent when the workflow
// concluded with any state other than "success". The caller should discard
// the event — it does not represent a completed deployment.
var ErrWorkflowNotSuccess = errors.New("github: workflow_run conclusion is not success — not a deploy signal")

// inferEnvironment maps a branch name to an environment label.
// Used only for push events where no explicit environment is available.
// Deployment events carry the environment directly and do not use this.
func inferEnvironment(branch, defaultBranch string) string {
	branch = strings.ToLower(branch)

	// Exact matches first
	switch branch {
	case "main", "master", defaultBranch:
		return "production"
	case "staging", "stage":
		return "staging"
	case "develop", "development", "dev":
		return "dev"
	}

	// Prefix matches
	if strings.HasPrefix(branch, "release/") || strings.HasPrefix(branch, "hotfix/") {
		return "production"
	}
	if strings.HasPrefix(branch, "staging/") {
		return "staging"
	}

	// Tag-based releases
	if strings.HasPrefix(branch, "v") || strings.HasPrefix(branch, "refs/tags/v") {
		return "production"
	}

	return "unknown"
}

// firstLine returns the first line of a multi-line string.
// Used to truncate commit messages that may be very long.
func firstLine(s string) string {
	if idx := strings.Index(s, "\n"); idx != -1 {
		return s[:idx]
	}
	return s
}

// ParseCheckSuiteEvent translates a GitHub check_suite event payload.
// It tracks the overall CI status for a specific commit.
func ParseCheckSuiteEvent(body []byte) (*ParsedCheckSuiteEvent, error) {
	var payload struct {
		Action     string `json:"action"`
		CheckSuite struct {
			HeadSHA    string `json:"head_sha"`
			Status     string `json:"status"`     // "queued", "in_progress", "completed"
			Conclusion string `json:"conclusion"` // "success", "failure", "neutral", "cancelled", "skipped", "timed_out", "action_required"
		} `json:"check_suite"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal check_suite event: %w", err)
	}

	return &ParsedCheckSuiteEvent{
		RepoFullName: payload.Repository.FullName,
		CommitSHA:    payload.CheckSuite.HeadSHA,
		Status:       payload.CheckSuite.Status,
		Conclusion:   payload.CheckSuite.Conclusion,
		RawPayload:   body,
	}, nil
}

// ParsedCheckSuiteEvent holds the outcome of a CI run for a commit.
type ParsedCheckSuiteEvent struct {
	RepoFullName string
	CommitSHA    string
	Status       string
	Conclusion   string
	RawPayload   json.RawMessage
}

// DetectRevert checks if a commit message indicates it's a git revert.
// If it is, it returns the original commit message being reverted.
// E.g. "Revert \"Original message\"" -> "Original message"
func DetectRevert(commitMessage string) (bool, string) {
	if strings.HasPrefix(commitMessage, "Revert \"") && strings.HasSuffix(commitMessage, "\"") {
		original := strings.TrimSuffix(strings.TrimPrefix(commitMessage, "Revert \""), "\"")
		return true, original
	}
	return false, ""
}

// InferEnvironmentExported is an exported version of inferEnvironment for testing
func InferEnvironmentExported(branch, defaultBranch string) string {
	return inferEnvironment(branch, defaultBranch)
}

// DetectAffectedServicesExported is an exported version of detectAffectedServices for testing
func DetectAffectedServicesExported(paths []string) []string {
	return detectAffectedServices(paths)
}

// Aliases for tests (lowercase, so test package can access)
//var inferEnvironmentExported = InferEnvironmentExported
//var detectAffectedServicesExported = DetectAffectedServicesExported
