package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/matt-riley/mjrwtf/internal/adapters/notification"
	"github.com/matt-riley/mjrwtf/internal/adapters/repository"
	"github.com/matt-riley/mjrwtf/internal/application"
	"github.com/matt-riley/mjrwtf/internal/domain/click"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/handlers"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/metrics"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/session"
	"github.com/rs/zerolog"
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
	logger          zerolog.Logger
	metrics         *metrics.Metrics
	sessionStore    *session.Store
	redirectUseCase *application.RedirectURLUseCase
}

// New creates a new HTTP server with configured middleware and dependencies
// Returns an error if the server cannot be initialized properly
func New(cfg *config.Config, db *sql.DB, logger zerolog.Logger) (*Server, error) {
	r := chi.NewRouter()

	// Initialize Prometheus metrics
	m := metrics.New()

	// Initialize Discord notifier for error notifications
	var discordNotifier *notification.DiscordNotifier
	if cfg.DiscordWebhookURL != "" {
		discordNotifier = notification.NewDiscordNotifier(
			cfg.DiscordWebhookURL,
			notification.WithLogger(logger),
		)
		logger.Info().Msg("Discord error notifications enabled")
	} else {
		logger.Info().Msg("Discord error notifications disabled (no webhook URL configured)")
	}

	// Middleware stack (order matters)
	r.Use(middleware.RecoveryWithNotifier(logger, discordNotifier)) // Recover from panics first, with Discord notifications
	r.Use(middleware.RequestID)                                     // Generate/propagate request ID
	r.Use(middleware.InjectLogger(logger))                          // Inject logger with request context
	r.Use(middleware.Logger)                                        // Log all requests
	r.Use(middleware.PrometheusMetrics(m))                          // Record Prometheus metrics

	// Initialize session store (24 hour session TTL)
	sessionStore := session.NewStore(24 * time.Hour)
	
	// Add session middleware globally (checks for session, but doesn't require it)
	r.Use(middleware.SessionMiddleware(sessionStore))

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
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"Link", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	server := &Server{
		router:       r,
		config:       cfg,
		db:           db,
		logger:       logger,
		metrics:      m,
		sessionStore: sessionStore,
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

	// Prometheus metrics endpoint
	// Note: This endpoint is intentionally public for Prometheus scraping.
	// In production, restrict access via network policies or a reverse proxy.
	// The endpoint exposes operational metrics (request rates, error rates, etc.)
	// which may be sensitive. Apply authentication if exposed to the public internet.
	s.router.Handle("/metrics", s.metrics.Handler())

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
	listUseCase := application.NewListURLsUseCase(urlRepo, clickRepo)
	deleteUseCase := application.NewDeleteURLUseCase(urlRepo)
	getAnalyticsUseCase := application.NewGetAnalyticsUseCase(urlRepo, clickRepo)
	s.redirectUseCase = application.NewRedirectURLUseCase(urlRepo, clickRepo)

	// Initialize handlers
	urlHandler := handlers.NewURLHandler(createUseCase, listUseCase, deleteUseCase)
	analyticsHandler := handlers.NewAnalyticsHandler(getAnalyticsUseCase)
	redirectHandler := handlers.NewRedirectHandler(s.redirectUseCase)
	pageHandler := handlers.NewPageHandler(createUseCase, listUseCase, s.config.AuthToken, s.sessionStore)

	// HTML page routes
	s.router.Get("/", pageHandler.Home)
	s.router.HandleFunc("/create", pageHandler.CreatePage)
	s.router.HandleFunc("/login", pageHandler.Login)
	s.router.Get("/logout", pageHandler.Logout)
	
	// Protected dashboard route (requires session)
	s.router.With(middleware.RequireSession(s.sessionStore, "/login")).Get("/dashboard", pageHandler.Dashboard)

	// Public redirect endpoint (no authentication required)
	// Must come after specific routes to avoid capturing them
	s.router.Get("/{shortCode}", redirectHandler.Redirect)

	// API routes with authentication
	s.router.Route("/api", func(r chi.Router) {
		r.Route("/urls", func(r chi.Router) {
			// Apply auth middleware to all URL endpoints
			// Support both Bearer token auth (for API) and session auth (for dashboard)
			r.Use(func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					// Check for session first
					if userID, ok := middleware.GetSessionUserID(req.Context()); ok && userID != "" {
						// Valid session, continue
						next.ServeHTTP(w, req)
						return
					}
					
					// Fall back to Bearer token auth
					middleware.Auth(s.config.AuthToken)(next).ServeHTTP(w, req)
				})
			})

			r.Post("/", urlHandler.Create)
			r.Get("/", urlHandler.List)
			r.Delete("/{shortCode}", urlHandler.Delete)
			r.Get("/{shortCode}/analytics", analyticsHandler.GetAnalytics)
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
	s.logger.Info().Str("addr", s.httpServer.Addr).Msg("starting HTTP server")
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("shutting down HTTP server")

	// Shutdown redirect use case workers first
	if s.redirectUseCase != nil {
		s.redirectUseCase.Shutdown()
	}

	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	s.logger.Info().Msg("HTTP server stopped")
	return nil
}

// Router returns the chi router for testing purposes
func (s *Server) Router() *chi.Mux {
	return s.router
}

// Metrics returns the Prometheus metrics for the server
func (s *Server) Metrics() *metrics.Metrics {
	return s.metrics
}
