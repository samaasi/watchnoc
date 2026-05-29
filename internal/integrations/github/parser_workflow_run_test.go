// internal/integrations/github/parser_workflow_run_test.go

package github_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	github "github.com/samaasi/watchnoc/internal/integrations/github"
)

func TestParseWorkflowRunEvent_Success(t *testing.T) {
	body, err := os.ReadFile("testdata/workflow_run_event.json")
	require.NoError(t, err)

	event, err := github.ParseWorkflowRunEvent(body)
	require.NoError(t, err)
	require.NotNil(t, event)

	assert.Equal(t, "acme-corp", event.RepoOwner)
	assert.Equal(t, "payment-service", event.RepoName)
	assert.Equal(t, "acme-corp/payment-service", event.RepoFullName)
	assert.Equal(t, "a3f9d2c1b4e5f6789012345678901234567890ab", event.CommitSHA)
	assert.Equal(t, "feat(payments): add retry logic for failed transactions", event.CommitMessage)
	assert.Equal(t, "main", event.Branch)
	assert.Equal(t, "production", event.Environment) // inferred from main branch
	assert.Equal(t, "alice-chen", event.AuthorLogin)
	assert.Equal(t, "alice@acme.com", event.AuthorEmail)
	assert.Equal(t, "workflow_run:9876543210", event.DeduplicationKey)
	assert.Equal(t, github.EventTypeWorkflowRun, event.EventType)

	require.NotNil(t, event.GitHubRunID)
	assert.Equal(t, int64(9876543210), *event.GitHubRunID)

	expectedTime, _ := time.Parse(time.RFC3339, "2026-05-27T14:32:00Z")
	assert.Equal(t, expectedTime, event.TriggeredAt)
}

func TestParseWorkflowRunEvent_NonSuccessConclusions(t *testing.T) {
	// Every non-success conclusion must return ErrWorkflowNotSuccess.
	// This is the most important correctness property of this parser —
	// a failed pipeline must NEVER create a DeployEvent.
	nonSuccessConclusions := []string{
		"failure",
		"cancelled",
		"skipped",
		"timed_out",
		"action_required",
		"neutral",
		"stale",
	}

	for _, conclusion := range nonSuccessConclusions {
		t.Run("conclusion="+conclusion, func(t *testing.T) {
			body := buildWorkflowRunPayload(conclusion, "main", "abc123sha")
			_, err := github.ParseWorkflowRunEvent(body)
			assert.ErrorIs(t, err, github.ErrWorkflowNotSuccess,
				"conclusion %q should return ErrWorkflowNotSuccess", conclusion)
		})
	}
}

func TestParseWorkflowRunEvent_EnvironmentInference(t *testing.T) {
	// workflow_run has no explicit environment field — environment is inferred
	// from branch name using the same logic as push events.
	tests := []struct {
		branch      string
		expectedEnv string
	}{
		{"main", "production"},
		{"master", "production"},
		{"release/v2.0.0", "production"},
		{"hotfix/pay-bug", "production"},
		{"staging", "staging"},
		{"develop", "dev"},
		{"feature/new-thing", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			body := buildWorkflowRunPayload("success", tt.branch, "deadbeef1234")
			event, err := github.ParseWorkflowRunEvent(body)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedEnv, event.Environment)
		})
	}
}

func TestParseWorkflowRunEvent_DeduplicationKey(t *testing.T) {
	// Two deliveries of the same workflow run must produce the same dedup key
	// so the worker can discard the second one.
	body := buildWorkflowRunPayload("success", "main", "deadbeef1234")

	event1, err := github.ParseWorkflowRunEvent(body)
	require.NoError(t, err)

	event2, err := github.ParseWorkflowRunEvent(body)
	require.NoError(t, err)

	assert.Equal(t, event1.DeduplicationKey, event2.DeduplicationKey)
	assert.Contains(t, event1.DeduplicationKey, "workflow_run:")
}

func TestParseWorkflowRunEvent_MissingHeadSHA(t *testing.T) {
	body := []byte(`{
        "action": "completed",
        "workflow_run": {
            "id": 111,
            "name": "Deploy Production",
            "head_branch": "main",
            "head_sha": "",
            "conclusion": "success",
            "created_at": "2026-05-27T14:30:00Z",
            "updated_at": "2026-05-27T14:35:00Z",
            "actor": {"login": "alice"}
        },
        "repository": {
            "name": "repo", "full_name": "owner/repo",
            "default_branch": "main",
            "owner": {"login": "owner"}
        }
    }`)

	_, err := github.ParseWorkflowRunEvent(body)
	assert.Error(t, err)
	assert.NotErrorIs(t, err, github.ErrWorkflowNotSuccess) // different error class
}

func TestParseWorkflowRunEvent_FallbackTriggeredAt(t *testing.T) {
	// When head_commit is absent, TriggeredAt falls back to the run's created_at.
	body := []byte(`{
        "action": "completed",
        "workflow_run": {
            "id": 222,
            "name": "Deploy Production",
            "head_branch": "main",
            "head_sha": "deadbeef12345678",
            "conclusion": "success",
            "created_at": "2026-05-27T10:00:00Z",
            "updated_at": "2026-05-27T10:05:00Z",
            "actor": {"login": "bot-deployer"},
            "head_commit": null
        },
        "repository": {
            "name": "api", "full_name": "acme/api",
            "default_branch": "main",
            "owner": {"login": "acme"}
        }
    }`)

	event, err := github.ParseWorkflowRunEvent(body)
	require.NoError(t, err)

	expectedTime, _ := time.Parse(time.RFC3339, "2026-05-27T10:00:00Z")
	assert.Equal(t, expectedTime, event.TriggeredAt,
		"TriggeredAt should fall back to created_at when head_commit is nil")
}

// buildWorkflowRunPayload constructs a minimal workflow_run webhook payload
// for use in table-driven tests. Avoids loading testdata for parametric cases.
func buildWorkflowRunPayload(conclusion, branch, sha string) []byte {
	return []byte(fmt.Sprintf(`{
        "action": "completed",
        "workflow_run": {
            "id": 9876543210,
            "name": "Deploy Production",
            "head_branch": %q,
            "head_sha": %q,
            "event": "push",
            "status": "completed",
            "conclusion": %q,
            "html_url": "https://github.com/acme-corp/payment-service/actions/runs/9876543210",
            "created_at": "2026-05-27T14:30:00Z",
            "updated_at": "2026-05-27T14:35:00Z",
            "actor": {"login": "alice-chen"},
            "head_commit": {
                "id": %q,
                "message": "feat: test commit",
                "timestamp": "2026-05-27T14:32:00Z",
                "author": {"name": "Alice Chen", "email": "alice@acme.com"}
            }
        },
        "repository": {
            "name": "payment-service",
            "full_name": "acme-corp/payment-service",
            "default_branch": "main",
            "owner": {"login": "acme-corp"}
        },
        "sender": {"login": "alice-chen"},
        "installation": {"id": 12345678}
    }`, branch, sha, conclusion, sha))
}

// BenchmarkParseWorkflowRunEvent ensures the parser stays performant
// under high delivery volume — each pipeline completion triggers one event.
func BenchmarkParseWorkflowRunEvent(b *testing.B) {
	body, _ := os.ReadFile("testdata/workflow_run_event.json")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = github.ParseWorkflowRunEvent(body)
	}
}
