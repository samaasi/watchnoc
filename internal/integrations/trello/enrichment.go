package trello

import (
    "context"
    "fmt"
    "log/slog"
)

// RegisterBoardWebhook registers a Trello webhook for a board.
// Called when a user links a card from a board we are not yet watching.
// Idempotent — if a webhook already exists for this board, returns the existing ID.
func (c *EnrichmentClient) RegisterBoardWebhook(
    ctx context.Context,
    install *Installation,
    boardID string,
) (string, error) {
    // Check if we already have a webhook for this board
    existing, err := c.installRepo.FindWebhookByBoardID(ctx, install.OrgID, boardID)
    if err == nil && existing != nil {
        return existing.TrelloWebhookID, nil
    }

    trelloWebhookID, err := c.client.WithToken(install.AccessToken).CreateWebhook(ctx, CreateWebhookRequest{
        // CallbackURL includes a secret path token — see Section 16.
        CallbackURL: fmt.Sprintf("%s/webhooks/trello/%s",
            c.webhookCallbackBase,
            install.WebhookPathToken,
        ),
        IDModel:     boardID,
        Description: fmt.Sprintf("DeployGuard board watcher — org %d", install.OrgID),
        Active:      true,
    })
    if err != nil {
        return "", fmt.Errorf("register board webhook %s: %w", boardID, err)
    }

    if err := c.installRepo.SaveWebhook(ctx, &WebhookRecord{
        OrgID:           install.OrgID,
        TrelloWebhookID: trelloWebhookID,
        BoardID:         boardID,
    }); err != nil {
        // Webhook registered in Trello but not saved locally — log, do not fail.
        // The reconciler will catch and re-register if needed.
        slog.Error("trello: webhook registered but not saved locally",
            "trello_webhook_id", trelloWebhookID,
            "board_id",          boardID,
        )
    }

    slog.Info("trello: registered board webhook",
        "org_id",            install.OrgID,
        "board_id",          boardID,
        "trello_webhook_id", trelloWebhookID,
    )

    return trelloWebhookID, nil
}

// EnrichmentClient fetches card details from the Trello REST API.
// Used for:
//   1. Validating and enriching a card URL pasted by a deployer.
//   2. Updating cached card metadata when a webhook signals a card changed.
//   3. Backfilling card data during reconciliation.
type EnrichmentClient struct {
    client              *Client
    installRepo         InstallationRepository
    webhookCallbackBase string
}

func NewEnrichmentClient(
    client *Client,
    installRepo InstallationRepository,
    webhookCallbackBase string,
) *EnrichmentClient {
    return &EnrichmentClient{
        client:              client,
        installRepo:         installRepo,
        webhookCallbackBase: webhookCallbackBase,
    }
}

// GetCard fetches full card details for a given short ID.
// Used when a deployer pastes a Trello card URL — we validate the card exists
// and retrieve its metadata for display and audit logging.
//
// Enrichment failures are non-fatal: a linked ticket with partial data
// is better than no ticket. The card URL is always stored even if enrichment fails.
func (c *EnrichmentClient) GetCard(
    ctx context.Context,
    install *Installation,
    cardShortID string,
) (*ParsedCardDetail, error) {
    // Trello card lookup endpoint accepts either the full card ID or the short link.
    // We always use the short link (shortId) — it is stable and user-visible.
    raw, err := c.client.WithToken(install.AccessToken).GetCard(ctx, cardShortID)
    if err != nil {
        return nil, fmt.Errorf("get trello card %s: %w", cardShortID, err)
    }

    detail := &ParsedCardDetail{
        CardID:      raw.ID,
        ShortID:     raw.ShortLink,
        ShortURL:    raw.ShortURL,
        Name:        raw.Name,
        Description: raw.Desc,
        BoardID:     raw.IDBoard,
        ListID:      raw.IDList,
        URL:         raw.URL,
    }

    // Fetch board name for display
    if raw.IDBoard != "" {
        board, err := c.client.WithToken(install.AccessToken).GetBoard(ctx, raw.IDBoard)
        if err != nil {
            slog.Warn("trello: failed to fetch board name — continuing without it",
                "board_id", raw.IDBoard,
                "error",    err,
            )
        } else {
            detail.BoardName = board.Name
        }
    }

    // Fetch list name for deployment stage context
    if raw.IDList != "" {
        list, err := c.client.WithToken(install.AccessToken).GetList(ctx, raw.IDList)
        if err != nil {
            slog.Warn("trello: failed to fetch list name — continuing without it",
                "list_id", raw.IDList,
                "error",   err,
            )
        } else {
            detail.ListName = list.Name
        }
    }

    // Map due date
    detail.DueDate     = raw.Due
    detail.DueComplete = raw.DueComplete

    // Map labels
    detail.Labels = make([]CardLabel, 0, len(raw.Labels))
    for _, l := range raw.Labels {
        detail.Labels = append(detail.Labels, CardLabel{
            ID:    l.ID,
            Name:  l.Name,
            Color: l.Color,
        })
    }

    // Map members
    detail.Members = make([]CardMember, 0, len(raw.Members))
    for _, m := range raw.Members {
        detail.Members = append(detail.Members, CardMember{
            ID:       m.ID,
            Username: m.Username,
            FullName: m.FullName,
        })
    }

    return detail, nil
}

// GetCardAndEnsureWebhook fetches a card and ensures a board-level webhook exists
// so DeployGuard receives future updates to that card's board.
// Called when a new card is linked to a deploy event.
func (c *EnrichmentClient) GetCardAndEnsureWebhook(
    ctx context.Context,
    install *Installation,
    cardShortID string,
) (*ParsedCardDetail, error) {
    detail, err := c.GetCard(ctx, install, cardShortID)
    if err != nil {
        return nil, err
    }

    // Register a board webhook if we do not already have one for this board.
    // Non-fatal if registration fails — the card is still linked without realtime updates.
    if _, err := c.RegisterBoardWebhook(ctx, install, detail.BoardID); err != nil {
        slog.Warn("trello: failed to register board webhook after card link",
            "board_id",    detail.BoardID,
            "card_short_id", cardShortID,
            "error",       err,
        )
    }

    return detail, nil
}

// GetMe fetches the authenticated member's profile.
// Used during OAuth to confirm the token works and record who connected.
func (c *EnrichmentClient) GetMe(ctx context.Context, token string) (*MemberInfo, error) {
    return c.client.WithToken(token).GetMe(ctx)
}

// ValidateCardURL checks whether a URL is a valid, accessible Trello card.
// Returns the card detail if valid, or an error explaining why it is not.
// Used for immediate feedback when a deployer manually pastes a card URL.
func (c *EnrichmentClient) ValidateCardURL(
    ctx context.Context,
    install *Installation,
    rawURL string,
) (*ParsedCardDetail, error) {
    shortID, err := ExtractTrelloCardShortIDFromURL(rawURL)
    if err != nil {
        return nil, fmt.Errorf("invalid trello card url: %w", err)
    }

    detail, err := c.GetCard(ctx, install, shortID)
    if err != nil {
        return nil, fmt.Errorf("card not found or not accessible: %w", err)
    }

    return detail, nil
}
