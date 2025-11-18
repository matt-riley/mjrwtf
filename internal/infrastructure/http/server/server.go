package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/matt-riley/mjrwtf/internal/adapters/repository"
	"github.com/matt-riley/mjrwtf/internal/application"
	"github.com/matt-riley/mjrwtf/internal/domain/click"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/handlers"
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
	httpServer      *http.Server
	router          *chi.Mux
	config          *config.Config
	db              *sql.DB
	redirectUseCase *application.RedirectURLUseCase
}

// New creates a new HTTP server with configured middleware and dependencies
// Returns an error if the server cannot be initialized properly
func New(cfg *config.Config, db *sql.DB) (*Server, error) {
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
		db:     db,
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.ServerPort),
			Handler:      r,
			ReadTimeout:  readTimeout,
			WriteTimeout: writeTimeout,
			IdleTimeout:  idleTimeout,
		},
	}

	// Setup routes
	if err := server.setupRoutes(); err != nil {
		return nil, err
	}

	return server, nil
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes() error {
	// Health check endpoint
	s.router.Get("/health", s.healthCheckHandler)

	// Initialize repositories based on database driver
	var urlRepo url.Repository
	var clickRepo click.Repository
	if strings.HasPrefix(s.config.DatabaseURL, "postgres://") || strings.HasPrefix(s.config.DatabaseURL, "postgresql://") {
		urlRepo = repository.NewPostgresURLRepository(s.db)
		clickRepo = repository.NewPostgresClickRepository(s.db)
	} else {
		urlRepo = repository.NewSQLiteURLRepository(s.db)
		clickRepo = repository.NewSQLiteClickRepository(s.db)
	}

	// Initialize URL generator
	generator, err := url.NewGenerator(urlRepo, url.DefaultGeneratorConfig())
	if err != nil {
		return fmt.Errorf("failed to create URL generator: %w", err)
	}

	// Initialize use cases
	createUseCase := application.NewCreateURLUseCase(generator, s.config.BaseURL)
	listUseCase := application.NewListURLsUseCase(urlRepo)
	deleteUseCase := application.NewDeleteURLUseCase(urlRepo)
	s.redirectUseCase = application.NewRedirectURLUseCase(urlRepo, clickRepo)

	// Initialize handlers
	urlHandler := handlers.NewURLHandler(createUseCase, listUseCase, deleteUseCase)
	redirectHandler := handlers.NewRedirectHandler(s.redirectUseCase)

	// Public redirect endpoint (no authentication required)
	s.router.Get("/{shortCode}", redirectHandler.Redirect)

	// API routes with authentication
	s.router.Route("/api", func(r chi.Router) {
		r.Route("/urls", func(r chi.Router) {
			// Apply auth middleware to all URL endpoints
			r.Use(middleware.Auth(s.config.AuthToken))

			r.Post("/", urlHandler.Create)
			r.Get("/", urlHandler.List)
			r.Delete("/{shortCode}", urlHandler.Delete)
		})
	})

	return nil
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

	// Shutdown redirect use case workers first
	if s.redirectUseCase != nil {
		s.redirectUseCase.Shutdown()
	}

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
