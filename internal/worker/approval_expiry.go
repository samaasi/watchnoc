package worker

import (
	"context"
	"log/slog"

	"github.com/hibiken/asynq"
)

const TaskApprovalExpiry = "approval:expire"

type ApprovalExpiryWorker struct{}

func NewApprovalExpiryWorker() *ApprovalExpiryWorker {
	return &ApprovalExpiryWorker{}
}

func (w *ApprovalExpiryWorker) ProcessTask(ctx context.Context, t *asynq.Task) error {
	// Scaffold: query all pending approvals past their expires_at, transition to "expired"
	slog.Info("approval expiry check running")
	return nil
}
