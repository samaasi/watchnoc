package evidence

import (
	"context"
)

type Service interface {
	GenerateReport(ctx context.Context, deployID uint64) (*EvidenceReport, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GenerateReport(ctx context.Context, deployID uint64) (*EvidenceReport, error) {
	// Scaffold logic
	return nil, nil
}
