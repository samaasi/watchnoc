package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
)

// Runner manages worker lifecycle: registers handlers, starts the asynq server.
type Runner struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

func NewRunner(redisAddr string) *Runner {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: redisAddr},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			},
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				slog.Error("worker task failed",
					"type", task.Type(),
					"error", err,
				)
			}),
		},
	)
	return &Runner{
		server: srv,
		mux:    asynq.NewServeMux(),
	}
}

func (r *Runner) Register(pattern string, handler asynq.Handler) {
	r.mux.Handle(pattern, handler)
}

func (r *Runner) Start() error {
	slog.Info("starting worker runner")
	return r.server.Start(r.mux)
}

func (r *Runner) Shutdown() {
	r.server.Shutdown()
}

// ScheduleRecurring sets up periodic tasks (cron-style).
func ScheduleRecurring(redisAddr string) (*asynq.Scheduler, error) {
	scheduler := asynq.NewScheduler(
		asynq.RedisClientOpt{Addr: redisAddr},
		&asynq.SchedulerOpts{
			Location: time.UTC,
		},
	)

	// Nightly DORA calculation at 02:00 UTC
	_, err := scheduler.Register("0 2 * * *", asynq.NewTask(TaskDORACalculation, nil))
	if err != nil {
		return nil, err
	}

	// Approval expiry check every 15 minutes
	_, err = scheduler.Register("*/15 * * * *", asynq.NewTask(TaskApprovalExpiry, nil))
	if err != nil {
		return nil, err
	}

	return scheduler, nil
}
