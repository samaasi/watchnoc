package github

import "context"

// InstallationRepository defines the interface for GitHub installation persistence.
type InstallationRepository interface {
	Create(ctx context.Context, install *GitHubInstallation) error
	Update(ctx context.Context, install *GitHubInstallation) error
	FindByOrgID(ctx context.Context, orgID uint64) (*GitHubInstallation, error)
	FindByRepoFullName(ctx context.Context, repoFullName string) (*GitHubInstallation, error)
	ListActive(ctx context.Context) ([]*GitHubInstallation, error)
	MarkRevoked(ctx context.Context, installationID int64) error
}

// DeliveryRepository defines the interface for GitHub webhook delivery persistence.
type DeliveryRepository interface {
	Exists(ctx context.Context, deliveryID string) (bool, error)
	Record(ctx context.Context, deliveryID, eventType string) error
	UpdateResult(ctx context.Context, deliveryID, result string, durationMs int64) error
}
