package policy

import (
	"context"
)

type Service interface {
	EvaluatePolicy(ctx context.Context, deployID uint64) (bool, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) EvaluatePolicy(ctx context.Context, deployID uint64) (bool, error) {
	// Scaffold logic
	return true, nil
}
