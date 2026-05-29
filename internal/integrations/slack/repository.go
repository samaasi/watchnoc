package slack

import "context"

// InstallationRepository defines the interface for Slack installation persistence.
type InstallationRepository interface {
	Create(ctx context.Context, install *Installation) error
	Update(ctx context.Context, install *Installation) error
	FindByOrgID(ctx context.Context, orgID uint64) (*Installation, error)
	FindByTeamID(ctx context.Context, teamID string) (*Installation, error)
	ListActive(ctx context.Context) ([]*Installation, error)
	MarkRevoked(ctx context.Context, orgID uint64) error
}

// NewInstallationRepository creates a new InstallationRepository (stub).
func NewInstallationRepository(db interface{}) InstallationRepository {
	// TODO: implement real repository
	return &stubInstallationRepository{}
}

type stubInstallationRepository struct{}

func (s *stubInstallationRepository) Create(ctx context.Context, install *Installation) error {
	return nil
}
func (s *stubInstallationRepository) Update(ctx context.Context, install *Installation) error {
	return nil
}
func (s *stubInstallationRepository) FindByOrgID(ctx context.Context, orgID uint64) (*Installation, error) {
	return nil, nil
}
func (s *stubInstallationRepository) FindByTeamID(ctx context.Context, teamID string) (*Installation, error) {
	return nil, nil
}
func (s *stubInstallationRepository) ListActive(ctx context.Context) ([]*Installation, error) {
	return nil, nil
}
func (s *stubInstallationRepository) MarkRevoked(ctx context.Context, orgID uint64) error {
	return nil
}
