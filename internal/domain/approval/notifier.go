package approval

import (
	"context"
)

// Notifier dispatches notifications to Slack (or other channels) for pending approvals.
type Notifier struct{}

// NewNotifier creates a new Notifier.
func NewNotifier() *Notifier {
	return &Notifier{}
}

// NotifyPending dispatches a message requesting approval for a deployment.
func (n *Notifier) NotifyPending(ctx context.Context, approval *Approval) error {
	// Scaffold: implement Slack notification logic
	return nil
}
