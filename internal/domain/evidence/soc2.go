package evidence

import (
	"context"
)

// SOC2Generator generates SOC2 compliant evidence reports.
type SOC2Generator struct{}

func NewSOC2Generator() *SOC2Generator {
	return &SOC2Generator{}
}

func (g *SOC2Generator) Generate(ctx context.Context, deployID uint64) (*EvidenceReport, error) {
	// Scaffold logic
	return nil, nil
}
