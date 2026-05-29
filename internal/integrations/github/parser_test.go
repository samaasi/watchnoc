package github_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	github "github.com/samaasi/watchnoc/internal/integrations/github"
)

func TestParsePushEvent(t *testing.T) {
	body, err := os.ReadFile("testdata/push_event.json")
	require.NoError(t, err)

	event, err := github.ParsePushEvent(body)
	require.NoError(t, err)

	assert.Equal(t, "acme-corp", event.RepoOwner)
	assert.Equal(t, "payment-service", event.RepoName)
	assert.Equal(t, "acme-corp/payment-service", event.RepoFullName)
	assert.Equal(t, "a3f9d2c1b4e5f6789012345678901234567890ab", event.CommitSHA)
	assert.Equal(t, "feat(payments): add retry logic for failed transactions", event.CommitMessage)
	assert.Equal(t, "main", event.Branch)
	assert.Equal(t, "production", event.Environment) // inferred from main branch
	assert.Equal(t, "alice-chen", event.AuthorLogin)
	assert.Equal(t, "alice@acme.com", event.AuthorEmail)
	assert.Equal(t, "push:a3f9d2c1b4e5f6789012345678901234567890ab:acme-corp/payment-service",
		event.DeduplicationKey)

	expectedTime, _ := time.Parse(time.RFC3339, "2026-05-27T14:32:00Z")
	assert.Equal(t, expectedTime, event.TriggeredAt)
}

func TestParsePushEvent_BranchDeletion(t *testing.T) {
	// Branch deletion pushes have a zero "after" SHA and no head commit
	body := []byte(`{
        "ref": "refs/heads/feature/old-branch",
        "after": "0000000000000000000000000000000000000000",
        "head_commit": null,
        "repository": {"name": "repo", "full_name": "owner/repo", "owner": {"login": "owner"}},
        "pusher": {"name": "alice"}
    }`)

	_, err := github.ParsePushEvent(body)
	assert.ErrorIs(t, err, github.ErrNoCommitData)
}

func TestInferEnvironment(t *testing.T) {
	tests := []struct {
		branch        string
		defaultBranch string
		expected      string
	}{
		{"main", "main", "production"},
		{"master", "main", "production"},
		{"main", "main", "production"},
		{"release/v2.1.0", "main", "production"},
		{"hotfix/critical-bug", "main", "production"},
		{"staging", "main", "staging"},
		{"develop", "main", "dev"},
		{"feature/new-thing", "main", "unknown"},
		{"v1.2.3", "main", "production"},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			// inferEnvironment is unexported — test via InferEnvironmentExported
			result := github.InferEnvironmentExported(tt.branch, tt.defaultBranch)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateSignature(t *testing.T) {
	const secret = "test-webhook-secret-32-chars-long"
	handler := github.NewTestWebhookHandler(secret)

	body := []byte(`{"test": "payload"}`)

	t.Run("valid signature", func(t *testing.T) {
		sig := github.ComputeTestSignature(secret, body)
		assert.True(t, handler.ValidateSignatureExported(body, "sha256="+sig))
	})

	t.Run("invalid signature", func(t *testing.T) {
		assert.False(t, handler.ValidateSignatureExported(body, "sha256=invalidsignature"))
	})

	t.Run("empty signature", func(t *testing.T) {
		assert.False(t, handler.ValidateSignatureExported(body, ""))
	})

	t.Run("missing sha256 prefix", func(t *testing.T) {
		sig := github.ComputeTestSignature(secret, body)
		assert.False(t, handler.ValidateSignatureExported(body, sig))
	})

	t.Run("wrong secret", func(t *testing.T) {
		sig := github.ComputeTestSignature("different-secret", body)
		assert.False(t, handler.ValidateSignatureExported(body, "sha256="+sig))
	})
}

func TestDetectAffectedServices(t *testing.T) {
	paths := []string{
		"services/payment/handler.go",
		"services/payment/retry.go",
		"services/auth/middleware.go",
		"pkg/shared/utils.go",
		"Dockerfile",
		"go.mod",
	}

	services := github.DetectAffectedServicesExported(paths)

	assert.Contains(t, services, "payment")
	assert.Contains(t, services, "auth")
	assert.Contains(t, services, "pkg")
	assert.Contains(t, services, "root")
	assert.Len(t, services, 4) // payment, auth, pkg, root — no duplicates
}

// Benchmark for high-throughput webhook processing
func BenchmarkParsePushEvent(b *testing.B) {
	body, _ := os.ReadFile("testdata/push_event.json")
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = github.ParsePushEvent(body)
	}
}

func BenchmarkValidateSignature(b *testing.B) {
	const secret = "test-webhook-secret-32-chars-long"
	handler := github.NewTestWebhookHandler(secret)
	body := []byte(`{"test": "payload of reasonable size for a webhook event"}`)
	sig := "sha256=" + github.ComputeTestSignature(secret, body)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		handler.ValidateSignatureExported(body, sig)
	}
}
