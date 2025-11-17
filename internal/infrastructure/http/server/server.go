// Package server provides HTTP server infrastructure for the mjrwtf URL shortener,
// including routing, middleware setup, and graceful shutdown.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
)

const (
	// Server timeout configurations
	readTimeout     = 15 * time.Second
	writeTimeout    = 15 * time.Second
	idleTimeout     = 60 * time.Second
	ShutdownTimeout = 30 * time.Second
)

// Server represents the HTTP server
type Server struct {
	httpServer *http.Server
	router     *chi.Mux
	config     *config.Config
}

// New creates a new HTTP server with configured middleware
func New(cfg *config.Config) *Server {
	r := chi.NewRouter()

	// Middleware stack (order matters)
	r.Use(middleware.Recovery) // Recover from panics
	r.Use(middleware.Logger)   // Log all requests

	// Parse CORS allowed origins (supports comma-separated list)
	origins := strings.Split(cfg.AllowedOrigins, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}

	r.Use(cors.Handler(cors.Options{
		// Note: Using "*" for allowed origins is a security risk in production.
		// Configure ALLOWED_ORIGINS environment variable to restrict access to known domains.
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	server := &Server{
		router: r,
		config: cfg,
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
			Handler:      r,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
	}

	// Setup routes
	server.setupRoutes()

	return server
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() {
	// Health check endpoint
	s.router.Get("/health", s.healthCheckHandler)

	// API routes will be added later
	s.router.Route("/api", func(r chi.Router) {
		// Placeholder for API routes
	})
}

// healthCheckHandler returns a simple health check response
func (s *Server) healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting HTTP server on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down HTTP server...")

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	log.Println("HTTP server stopped")
	return nil
}

// Router returns the chi router for testing purposes
func (s *Server) Router() *chi.Mux {
	return s.router
}
