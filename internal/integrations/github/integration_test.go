// internal/integrations/github/integration_test.go

//go:build integration

package github_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "os"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

// TestWebhookHandler_FullFlow tests the complete webhook → parse → queue flow
// using an httptest server. Does not require a real GitHub App.
func TestWebhookHandler_FullFlow(t *testing.T) {
    const secret = "integration-test-secret-32chars!"

    // Capture queued jobs
    var queuedPayloads [][]byte
    mockQueue := &mockJobQueue{
        onEnqueue: func(payload []byte) {
            queuedPayloads = append(queuedPayloads, payload)
        },
    }

    handler := newTestWebhookHandler(secret, mockQueue)
    srv := httptest.NewServer(http.HandlerFunc(handler.Handle))
    defer srv.Close()

    body, err := os.ReadFile("testdata/push_event.json")
    require.NoError(t, err)

    req := buildSignedWebhookRequest(t, srv.URL+"/webhooks/github",
        body, secret, "push", "test-delivery-001")

    resp, err := http.DefaultClient.Do(req)
    require.NoError(t, err)
    defer resp.Body.Close()

    // Webhook handler always returns 202 immediately
    assert.Equal(t, http.StatusAccepted, resp.StatusCode)

    // Give the goroutine time to process
    // In real tests use a channel or WaitGroup for determinism
    time.Sleep(100 * time.Millisecond)

    assert.Len(t, queuedPayloads, 1)
}
