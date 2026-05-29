package trello_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	trello "github.com/samaasi/watchnoc/internal/integrations/trello"
)

func TestParseCardUpdateEvent_ListMove(t *testing.T) {
	body, err := os.ReadFile("testdata/card_update_event.json")
	require.NoError(t, err)

	event, err := trello.ParseCardUpdateEvent(body)
	require.NoError(t, err)

	assert.Equal(t, "AbCd1234", event.CardShortID)
	assert.Equal(t, "ENG-1234: Add retry logic for payment failures", event.CardName)
	assert.Equal(t, "65e1a2b3c4d5e6f7890abcde", event.BoardID)
	assert.Equal(t, "Payment Service Sprint 12", event.BoardName)
	assert.Equal(t, "In Progress", event.ListChangedFrom)
	assert.Equal(t, "Deployed", event.ListChangedTo)
	assert.Equal(t, "alice_chen", event.ActorUsername)
	assert.False(t, event.TitleChanged)
	assert.False(t, event.DescriptionChanged)

	expectedTime, _ := time.Parse(time.RFC3339, "2026-05-27T14:32:00Z")
	assert.Equal(t, expectedTime, event.OccurredAt)
}

func TestExtractTrelloCardShortID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "full url in commit message",
			input:    "feat: add retry logic - https://trello.com/c/AbCd1234/my-card-title",
			expected: "AbCd1234",
		},
		{
			name:     "short url without slug",
			input:    "closes https://trello.com/c/AbCd1234",
			expected: "AbCd1234",
		},
		{
			name:     "url in brackets",
			input:    "fix(payment): timeout [trello: https://trello.com/c/AbCd1234]",
			expected: "AbCd1234",
		},
		{
			name:     "no trello url",
			input:    "fix: minor style changes",
			expected: "",
		},
		{
			name:     "jira url — not trello",
			input:    "fix: ENG-1234 https://acme.atlassian.net/browse/ENG-1234",
			expected: "",
		},
		{
			name:     "case insensitive",
			input:    "feat: thing HTTPS://TRELLO.COM/C/AbCd1234",
			expected: "AbCd1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trello.ExtractTrelloCardShortID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractTrelloCardShortIDFromURL(t *testing.T) {
	t.Run("valid card url with slug", func(t *testing.T) {
		id, err := trello.ExtractTrelloCardShortIDFromURL(
			"https://trello.com/c/AbCd1234/42-my-card-title")
		require.NoError(t, err)
		assert.Equal(t, "AbCd1234", id)
	})

	t.Run("valid card url without slug", func(t *testing.T) {
		id, err := trello.ExtractTrelloCardShortIDFromURL(
			"https://trello.com/c/AbCd1234")
		require.NoError(t, err)
		assert.Equal(t, "AbCd1234", id)
	})

	t.Run("non-trello url", func(t *testing.T) {
		_, err := trello.ExtractTrelloCardShortIDFromURL(
			"https://github.com/acme/repo/pull/42")
		assert.Error(t, err)
	})

	t.Run("trello board url — not a card", func(t *testing.T) {
		_, err := trello.ExtractTrelloCardShortIDFromURL(
			"https://trello.com/b/AbCd1234/board-name")
		assert.Error(t, err)
	})
}

func TestIsDeployedListName(t *testing.T) {
	deployed := []string{
		"Deployed", "DEPLOYED", "Done", "Production", "Live",
		"Released", "Shipped", "Complete", "Completed",
		"In Production", "deployed to prod",
	}
	notDeployed := []string{
		"To Do", "In Progress", "Review", "Blocked", "Backlog",
		"Ready for Deploy", "QA",
	}

	for _, name := range deployed {
		t.Run(name, func(t *testing.T) {
			assert.True(t, trello.IsDeployedListNameExported(name))
		})
	}

	for _, name := range notDeployed {
		t.Run(name, func(t *testing.T) {
			assert.False(t, trello.IsDeployedListNameExported(name))
		})
	}
}

func TestWebhookHandler_HeadRequest(t *testing.T) {
	// Trello sends HEAD to validate the webhook URL before registration.
	// Must return 200 with no body.
	handler := trello.NewTestWebhookHandler()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodHead, "/webhooks/trello/testtoken", nil)

	handler.Handle(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, 0, w.Body.Len())
}
