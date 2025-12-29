# HTTP Infrastructure

This package provides the HTTP server infrastructure for the mjrwtf URL shortener, including routing, middleware, and graceful shutdown handling.

## Structure

```
internal/infrastructure/http/
├── handlers/            # HTTP request handlers
│   ├── page_handler.go    # HTML page rendering
│   ├── url_handler.go     # API URL management
│   ├── redirect_handler.go # URL redirection
│   └── *_test.go          # Handler tests
├── middleware/          # HTTP middleware components
│   ├── recovery.go        # Panic recovery middleware
│   ├── logger.go          # Request logging middleware
│   ├── auth.go            # Authentication middleware
│   └── *_test.go          # Middleware tests
├── server/              # HTTP server implementation
│   ├── server.go          # Main server with routing
│   ├── server_test.go     # Unit tests
│   └── integration_test.go  # Integration tests
└── templates/           # Templ HTML templates (see templates/README.md)
    ├── layouts/           # Layout templates
    ├── pages/             # Page templates
    └── components/        # Reusable components
```

## Components

### Server (`server/`)

The main HTTP server with:
- **Router**: Uses `chi` router for efficient routing
- **Graceful Shutdown**: Handles SIGTERM/SIGINT with 30s timeout
- **Timeouts**: Configurable read (15s), write (15s), and idle (60s) timeouts
- **Health Check (Liveness)**: `/health` endpoint for monitoring
- **Readiness Check**: `/ready` endpoint for dependency checks (e.g. DB)
- **HTML Rendering**: Serves HTML pages using Templ templates
- **API Routes**: RESTful API endpoints for URL management

### Handlers (`handlers/`)

Request handlers for different concerns:
- **PageHandler**: Renders HTML pages (home, error pages)
- **URLHandler**: API endpoints for URL CRUD operations
- **RedirectHandler**: Handles short URL redirects with analytics

### Middleware (`middleware/`)

Middleware stack executes in this order:
1. **Recovery**: Catches panics and returns 500 errors (HTML for browser, JSON for API)
2. **Logger**: Logs all requests with method, path, status, duration
3. **CORS**: Handles cross-origin requests
4. **Auth**: Authentication middleware (applied to `/api` routes only)
5. **Handlers**: Route handlers

## Usage

### Starting the Server

```go
import (
    "github.com/matt-riley/mjrwtf/internal/infrastructure/config"
    "github.com/matt-riley/mjrwtf/internal/infrastructure/http/server"
)

// Load configuration
cfg, err := config.LoadConfig()
if err != nil {
    log.Fatal(err)
}

// Create and start server
srv := server.New(cfg)
go func() {
    if err := srv.Start(); err != nil {
        log.Fatal(err)
    }
}()

// Graceful shutdown
ctx := context.Background()
srv.Shutdown(ctx)
```

### Adding Routes

```go
srv := server.New(cfg, db)
router := srv.Router()

// Add custom HTML page routes
router.Get("/custom", customPageHandler)

// Add custom API routes
router.Route("/api/custom", func(r chi.Router) {
    r.Use(middleware.Auth(cfg.ActiveAuthTokens()))
    r.Get("/", handleList)
    r.Post("/", handleCreate)
})
```

### Rendering HTML Pages

```go
import "github.com/matt-riley/mjrwtf/internal/adapters/http/templates/pages"

func HomeHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    if err := pages.Home().Render(r.Context(), w); err != nil {
        http.Error(w, "Error rendering page", http.StatusInternalServerError)
    }
}
```

See `internal/adapters/http/templates/README.md` for more on Templ templates.

### Testing

```bash
# Run all HTTP tests
go test ./internal/infrastructure/http/... -v

# Run with coverage
go test ./internal/infrastructure/http/... -cover

# Run benchmarks
go test ./internal/infrastructure/http/... -bench=. -benchmem
```

## Configuration

The server reads configuration from environment variables:

```bash
SERVER_PORT=8080          # Port to bind to (default: 8080)
ALLOWED_ORIGINS=*         # CORS allowed origins (default: "*" - not recommended for production)
DATABASE_URL=...          # Database connection string
AUTH_TOKEN=...            # API authentication token
```

See `.env.example` for full configuration options.

## Middleware Details

### Recovery Middleware

Catches panics in request handlers and returns a 500 Internal Server Error (when headers have not yet been written):

```go
// Automatically applied to all routes
// Logs panic details and (optionally) stack traces
// Stack traces can be disabled via LOG_STACK_TRACES=false
// Re-panics http.ErrAbortHandler to preserve net/http semantics
// Returns a plain text 500 response when possible
```

### Logger Middleware

Logs every request with:
- HTTP method (GET, POST, etc.)
- Request path
- Response status code
- Request duration

Example log output:
```
2025/11/08 15:03:21 GET /health 200 15B 7.023µs
2025/11/08 15:03:22 POST /api/urls 201 1200B 1.234ms
2025/11/08 15:03:23 GET /api/urls 200 512B 456.789µs
```

### CORS Middleware

Configured to allow:
- **Origins**: All (`*`)
- **Methods**: GET, POST, PUT, DELETE, OPTIONS
- **Headers**: Accept, Authorization, Content-Type
- **Max Age**: 300 seconds

## Performance

Benchmark results (on GitHub Actions runner):
```
BenchmarkServer_HealthCheck         396768    2949 ns/op    6602 B/op    26 allocs/op
BenchmarkServer_WithMiddleware      419599    2715 ns/op    6570 B/op    25 allocs/op
```

~3μs per request with full middleware stack.

## Graceful Shutdown

The server supports graceful shutdown with:
- Signal handling (SIGTERM, SIGINT)
- Configurable timeout (30s default)
- Completes in-flight requests
- Closes idle connections

## Future Enhancements

- [x] Authentication middleware
- [x] HTML page rendering with Templ
- [x] Rate limiting middleware
- [ ] Request ID tracking
- [x] Metrics/monitoring middleware
- [ ] Compression middleware
- [ ] ETag support
