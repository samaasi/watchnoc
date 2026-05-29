package dora

import (
	"context"
)

type Service interface {
	CalculateMetrics(ctx context.Context, orgID uint64) (*DORASnapshot, error)
}

type service struct {}

func NewService() Service {
	return &service{}
}

func (s *service) CalculateMetrics(ctx context.Context, orgID uint64) (*DORASnapshot, error) {
	// Scaffold logic
	return nil, nil
}
