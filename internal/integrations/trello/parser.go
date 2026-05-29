package trello

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// cardShortIDPattern matches Trello card short IDs in URLs, PR titles, and branch names.
// Trello short IDs are 8 alphanumeric characters.
// Matches: https://trello.com/c/ABC12345 or branch feature/ABC12345-my-feature
var cardShortIDPattern = regexp.MustCompile(`(?i)trello\.com/c/([a-zA-Z0-9]{8})`)

// ParsedCardUpdateEvent is the internal representation of a Trello updateCard action.
type ParsedCardUpdateEvent struct {
	CardID      string
	CardShortID string
	CardName    string
	BoardID     string
	BoardName   string

	// List change — indicates card was moved between columns
	ListChangedFrom string
	ListChangedTo   string

	// Field changes
	TitleChanged       bool
	NewTitle           string
	DescriptionChanged bool
	NewDescription     string
	DueDateChanged     bool
	NewDueDate         *time.Time

	// Who performed the action
	ActorMemberID string
	ActorUsername string
	ActorFullName string

	OccurredAt time.Time
	RawPayload json.RawMessage
}

// ParsedCardDeleteEvent is the internal representation of a Trello deleteCard action.
type ParsedCardDeleteEvent struct {
	CardID      string
	CardShortID string
	BoardID     string
	OccurredAt  time.Time
}

// ParsedCardDetail is the full card state fetched from the REST API.
// Used for enrichment — not from webhooks.
type ParsedCardDetail struct {
	CardID      string
	ShortID     string
	ShortURL    string
	Name        string
	Description string
	BoardID     string
	BoardName   string
	ListID      string
	ListName    string
	DueDate     *time.Time
	DueComplete bool
	Labels      []CardLabel
	Members     []CardMember
	URL         string
}

type CardLabel struct {
	ID    string
	Name  string
	Color string
}

type CardMember struct {
	ID       string
	Username string
	FullName string
}

// ParseCardUpdateEvent translates a Trello updateCard webhook payload.
func ParseCardUpdateEvent(body []byte) (*ParsedCardUpdateEvent, error) {
	var payload struct {
		Action struct {
			ID   string    `json:"id"`
			Type string    `json:"type"`
			Date time.Time `json:"date"`
			Data struct {
				Card struct {
					ID      string `json:"id"`
					ShortID int    `json:"idShort"`
					Name    string `json:"name"`
				} `json:"card"`
				Board struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"board"`
				ListBefore *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"listBefore"`
				ListAfter *struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"listAfter"`
				Old *struct {
					Name string     `json:"name"`
					Desc string     `json:"desc"`
					Due  *time.Time `json:"due"`
				} `json:"old"`
			} `json:"data"`
			MemberCreator struct {
				ID       string `json:"id"`
				Username string `json:"username"`
				FullName string `json:"fullName"`
			} `json:"memberCreator"`
		} `json:"action"`
		Model struct {
			ID        string     `json:"id"`
			Name      string     `json:"name"`
			ShortID   int        `json:"idShort"`
			ShortLink string     `json:"shortLink"`
			Desc      string     `json:"desc"`
			Due       *time.Time `json:"due"`
		} `json:"model"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal updateCard event: %w", err)
	}

	event := &ParsedCardUpdateEvent{
		CardID:        payload.Action.Data.Card.ID,
		CardShortID:   payload.Model.ShortLink,
		CardName:      payload.Action.Data.Card.Name,
		BoardID:       payload.Action.Data.Board.ID,
		BoardName:     payload.Action.Data.Board.Name,
		ActorMemberID: payload.Action.MemberCreator.ID,
		ActorUsername: payload.Action.MemberCreator.Username,
		ActorFullName: payload.Action.MemberCreator.FullName,
		OccurredAt:    payload.Action.Date,
		RawPayload:    body,
	}

	// Detect list change (card moved between columns)
	if payload.Action.Data.ListBefore != nil && payload.Action.Data.ListAfter != nil {
		event.ListChangedFrom = payload.Action.Data.ListBefore.Name
		event.ListChangedTo = payload.Action.Data.ListAfter.Name
	}

	// Detect field changes from the "old" object (Trello includes prior values)
	if payload.Action.Data.Old != nil {
		if payload.Action.Data.Old.Name != "" {
			event.TitleChanged = true
			event.NewTitle = payload.Action.Data.Card.Name
		}
		if payload.Action.Data.Old.Desc != "" {
			event.DescriptionChanged = true
			event.NewDescription = payload.Model.Desc
		}
		if payload.Action.Data.Old.Due != nil {
			event.DueDateChanged = true
			event.NewDueDate = payload.Model.Due
		}
	}

	return event, nil
}

// ParseCardDeleteEvent translates a Trello deleteCard webhook payload.
func ParseCardDeleteEvent(body []byte) (*ParsedCardDeleteEvent, error) {
	var payload struct {
		Action struct {
			Date time.Time `json:"date"`
			Data struct {
				Card struct {
					ID        string `json:"id"`
					ShortLink string `json:"shortLink"`
				} `json:"card"`
				Board struct {
					ID string `json:"id"`
				} `json:"board"`
			} `json:"data"`
		} `json:"action"`
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("unmarshal deleteCard event: %w", err)
	}

	return &ParsedCardDeleteEvent{
		CardID:      payload.Action.Data.Card.ID,
		CardShortID: payload.Action.Data.Card.ShortLink,
		BoardID:     payload.Action.Data.Board.ID,
		OccurredAt:  payload.Action.Date,
	}, nil
}

// ExtractTrelloCardShortID parses a Trello card short ID from a commit message
// or arbitrary text. Returns the first match found, or "" if none.
//
// Commit message examples we detect:
//
//	"feat: add retry logic [trello: https://trello.com/c/ABC12345]"
//	"fix: payment timeout - https://trello.com/c/ABC12345/card-title"
//	"closes trello.com/c/ABC12345"
func ExtractTrelloCardShortID(text string) string {
	matches := cardShortIDPattern.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// ExtractTrelloCardShortIDFromURL extracts the short ID from a fully-formed
// Trello card URL. Used when a user pastes a URL into the manual link field.
func ExtractTrelloCardShortIDFromURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse trello url: %w", err)
	}

	if !strings.Contains(u.Host, "trello.com") {
		return "", fmt.Errorf("not a trello url: %s", rawURL)
	}

	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	// Path format: /c/{shortID} or /c/{shortID}/{slug}
	if len(parts) < 2 || parts[0] != "c" {
		return "", fmt.Errorf("unrecognised trello card url format: %s", rawURL)
	}

	shortID := parts[1]
	if len(shortID) != 8 {
		return "", fmt.Errorf("invalid trello card short id length: %s", shortID)
	}

	return shortID, nil
}

// isDeployedListName returns true if a list name indicates a deployment state.
// Teams use various naming conventions — we match the most common patterns.
// In Gate 2, this becomes configurable per org.
func isDeployedListName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	deployedNames := []string{
		"deployed", "done", "production", "live", "released",
		"shipped", "complete", "completed", "in production",
	}
	for _, dn := range deployedNames {
		if name == dn || strings.Contains(name, dn) {
			return true
		}
	}
	return false
}

// IsDeployedListNameExported is an exported version of isDeployedListName for testing
func IsDeployedListNameExported(name string) bool {
	return isDeployedListName(name)
}

// internal/integrations/trello/parser.go

// TrelloDeduplicationKey creates a stable key for a Trello action event.
// Trello action IDs are ULIDs and are stable — use the action ID as the key.
// Format: "trello:{actionID}"
func TrelloDeduplicationKey(actionID string) string {
	return fmt.Sprintf("trello:%s", actionID)
}
