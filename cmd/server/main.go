package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/server"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create server
	srv := server.New(cfg)

	// Channel to listen for errors from the server
	serverErrors := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		serverErrors <- srv.Start()
	}()

	// Channel to listen for interrupt or terminate signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal or an error
	select {
	case err := <-serverErrors:
		log.Fatalf("Server error: %v", err)
	case sig := <-shutdown:
		log.Printf("Received signal: %s. Starting graceful shutdown...", sig)

		// Graceful shutdown with timeout context
		// Match server's ShutdownTimeout to ensure process eventually terminates
		ctx, cancel := context.WithTimeout(context.Background(), server.ShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Error during shutdown: %v", err)
		}

		log.Println("Server shutdown complete")
	}
}
