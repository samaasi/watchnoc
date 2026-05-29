package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/samaasi/watchnoc/internal/app"
	"github.com/samaasi/watchnoc/internal/config"
	"github.com/samaasi/watchnoc/internal/platform/middleware"
	"github.com/samaasi/watchnoc/internal/platform/response"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize responder
	responder := response.NewChiResponder()

	// Initialize context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create app
	application, cleanup, err := app.NewAppFromConfig(ctx, cfg, responder)
	if err != nil {
		log.Fatalf("Failed to create app: %v", err)
	}
	defer cleanup()

	// Set up router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(middleware.Recover(responder))
	r.Use(middleware.Timeout(30 * time.Second))

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Register application routes
	application.RegisterRoutes(r)

	// Start server
	serverAddr := ":8080"
	if cfg.Server.Port != 0 {
		serverAddr = fmt.Sprintf(":%d", cfg.Server.Port)
	}

	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	log.Printf("Starting server on %s", serverAddr)

	// Run server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// Give outstanding requests 15 seconds to complete
	ctxShutdown, cancelShutdown := context.WithTimeout(ctx, 15*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")
}
