package observability

import (
	"context"
	"log"
)

func InitTracerProvider(ctx context.Context) (func(), error) {
	log.Println("Initializing tracer provider (placeholder for now)")
	return func() {}, nil
}
