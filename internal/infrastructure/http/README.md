# HTTP Infrastructure

This package provides the HTTP server infrastructure for the mjrwtf URL shortener, including routing, middleware, and graceful shutdown handling.

## Structure

```
internal/infrastructure/http/
├── middleware/          # HTTP middleware components
│   ├── recovery.go      # Panic recovery middleware
│   ├── logger.go        # Request logging middleware
│   └── *_test.go        # Middleware tests
└── server/              # HTTP server implementation
    ├── server.go        # Main server with routing
    ├── server_test.go   # Unit tests
    └── integration_test.go  # Integration tests
```

## Components

### Server (`server/`)

The main HTTP server with:
- **Router**: Uses `chi` router for efficient routing
- **Graceful Shutdown**: Handles SIGTERM/SIGINT with 30s timeout
- **Timeouts**: Configurable read (15s), write (15s), and idle (60s) timeouts
- **Health Check**: `/health` endpoint for monitoring

### Middleware (`middleware/`)

Middleware stack executes in this order:
1. **Recovery**: Catches panics and returns 500 errors
2. **Logger**: Logs all requests with method, path, status, duration
3. **CORS**: Handles cross-origin requests
4. **Auth**: (To be implemented) Authentication middleware
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
srv := server.New(cfg)
router := srv.Router()

// Add custom routes
router.Get("/api/urls", handleListURLs)
router.Post("/api/urls", handleCreateURL)
router.Delete("/api/urls/{id}", handleDeleteURL)
```

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
DATABASE_URL=...          # Database connection string
AUTH_TOKEN=...            # API authentication token
```

See `.env.example` for full configuration options.

## Middleware Details

### Recovery Middleware

Catches panics in request handlers and returns a 500 Internal Server Error:

```go
// Automatically applied to all routes
// Logs panic with stack trace
// Returns JSON error response
```

### Logger Middleware

Logs every request with:
- HTTP method (GET, POST, etc.)
- Request path
- Response status code
- Request duration

Example log output:
```
2025/11/08 15:03:21 GET /health 200 7.023µs
2025/11/08 15:03:22 POST /api/urls 201 1.234ms
2025/11/08 15:03:23 GET /api/urls 200 456.789µs
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

- [ ] Authentication middleware
- [ ] Rate limiting middleware
- [ ] Request ID tracking
- [ ] Metrics/monitoring middleware
- [ ] Compression middleware
- [ ] ETag support
