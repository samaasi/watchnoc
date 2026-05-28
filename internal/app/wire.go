package app

import (
	"context"
	"log"
	"time"

	"github.com/samaasi/watchnoc/internal/config"
	"github.com/samaasi/watchnoc/internal/platform/observability"
	"github.com/samaasi/watchnoc/internal/store"
)

// NewAppFromConfig initializes an App from a Config
func NewAppFromConfig(ctx context.Context, cfg *config.Config) (*App, func(), error) {
	// Initialize logger
	logger := observability.InitLogger(true)
	logger.Info("Initializing app")

	// Initialize tracer provider
	shutdownTracer, err := observability.InitTracerProvider(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Initialize DB
	db, err := store.NewPostgres(cfg.Database)
	if err != nil {
		shutdownTracer()
		return nil, nil, err
	}

	// Initialize Redis
	rdb, err := store.NewRedis(cfg.Redis)
	if err != nil {
		shutdownTracer()
		return nil, nil, err
	}

	// Initialize App
	app, err := NewApp(db)
	if err != nil {
		shutdownTracer()
		return nil, nil, err
	}
	_ = rdb // TODO: use this in future

	// Cleanup function
	cleanup := func() {
		log.Println("Shutting down app")
		_, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		app.Close()
		shutdownTracer()
	}

	return app, cleanup, nil
}
