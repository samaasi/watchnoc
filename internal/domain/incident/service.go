package incident

import (
	"context"
)

type Service interface {
	CorrelateIncident(ctx context.Context, extIncidentID string) error
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) CorrelateIncident(ctx context.Context, extIncidentID string) error {
	// Scaffold logic
	return nil
}
