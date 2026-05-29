//go:build integration

package trello_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebhookHandler_CardMoveToDeployed tests that a card-moved-to-Deployed
// webhook triggers an enrichment job.
func TestWebhookHandler_CardMoveToDeployed(t *testing.T) {
	var enrichmentJobs []TrelloEnrichmentPayload
	mockQueue := &mockJobQueue{
		onEnqueue: func(p TrelloEnrichmentPayload) {
			enrichmentJobs = append(enrichmentJobs, p)
		},
	}

	handler := newTestWebhookHandler(mockQueue)
	srv := httptest.NewServer(http.HandlerFunc(handler.Handle))
	defer srv.Close()

	body, err := os.ReadFile("testdata/card_update_event.json")
	require.NoError(t, err)

	// POST the webhook payload
	resp, err := http.Post(srv.URL+"/webhooks/trello/testtoken",
		"application/json", strings.NewReader(string(body)))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Give the goroutine time to run
	time.Sleep(50 * time.Millisecond)

	// A card moved to "Deployed" should enqueue an enrichment job
	assert.Len(t, enrichmentJobs, 1)
	assert.Equal(t, "AbCd1234", enrichmentJobs[0].CardShortID)
}
