package policy

import (
	"context"
)

// Evaluator evaluates approval policies against deployments.
type Evaluator struct{}

func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

func (e *Evaluator) Evaluate(ctx context.Context, deployID uint64, policyID uint64) (bool, error) {
	// Scaffold logic
	return true, nil
}
