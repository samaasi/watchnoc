package trello

import "context"

// InstallationRepository defines the interface for Trello installation persistence.
type InstallationRepository interface {
	Create(ctx context.Context, install *Installation) error
	Update(ctx context.Context, install *Installation) error
	FindByOrgID(ctx context.Context, orgID uint64) (*Installation, error)
	FindWebhookByBoardID(ctx context.Context, orgID uint64, boardID string) (*WebhookRecord, error)
	ListWebhooks(ctx context.Context, orgID uint64) ([]*WebhookRecord, error)
	SaveWebhook(ctx context.Context, record *WebhookRecord) error
	DeleteWebhook(ctx context.Context, record *WebhookRecord) error
	ListActive(ctx context.Context) ([]*Installation, error)
	MarkTokenExpired(ctx context.Context, orgID uint64) error
	UpdateWebhookID(ctx context.Context, oldID, newID string) error
}

// DeliveryRepository defines the interface for Trello webhook delivery persistence.
type DeliveryRepository interface {
	Exists(ctx context.Context, deliveryID string) (bool, error)
	Record(ctx context.Context, deliveryID, eventType string) error
	UpdateResult(ctx context.Context, deliveryID, result string) error
}

// LinkedTicketRepository is a placeholder interface for the ticket domain repository.
type LinkedTicketRepository interface {
	ListStaleCards(ctx context.Context, orgID uint64, days, hours int) ([]string, error)
	UpdateCardMetadata(ctx context.Context, cardShortID string, detail *ParsedCardDetail) error
}
