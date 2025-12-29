---
title: Integration testing
description: End-to-end HTTP integration tests (SQLite in-memory).
---


This document describes the comprehensive integration testing suite for the mjr.wtf URL shortener.

## Overview

The integration testing suite provides end-to-end testing of the entire application stack, including:
- HTTP API endpoints
- Database operations (SQLite in-memory)
- Authentication and authorization
- URL creation, redirection, and analytics
- Error handling and edge cases
- Concurrent request handling

## Running Integration Tests

### Quick Start

Run all integration tests:
```bash
make test-integration
```

Run all tests (unit + integration):
```bash
make test
```

Run unit tests only (fast, excludes integration tests):
```bash
make test-unit
```

### Specific Test Suites

Run end-to-end workflow tests:
```bash
go test -v -run TestE2E ./internal/infrastructure/http/server/
```

Run API endpoint tests:
```bash
go test -v -run TestAPI ./internal/infrastructure/http/server/
```

Run authentication tests:
```bash
go test -v -run Auth ./internal/infrastructure/http/server/
```

Run redirect and analytics tests:
```bash
go test -v -run Redirect ./internal/infrastructure/http/server/
```

## Test Structure

### End-to-End Tests (`e2e_integration_test.go`)

Comprehensive workflow tests that verify the complete user journey:

#### TestE2E_FullWorkflow
Tests the complete workflow:
1. **Authenticate** - Create URL with valid auth token
2. **Create URL** - Generate a shortened URL
3. **Redirect** - Follow the short URL to the original
4. **Verify Analytics** - Confirm click tracking works

**What it validates:**
- Authentication middleware
- URL creation with database persistence
- HTTP redirects with proper status codes
- Asynchronous click recording
- Analytics data aggregation

#### TestE2E_AuthenticationFlow
Tests authentication scenarios:
- Valid token acceptance
- Invalid token rejection
- Missing token handling
- Malformed header rejection

**What it validates:**
- Auth middleware behavior
- Token validation logic
- Error response formats

#### TestE2E_ErrorScenarios
Tests error handling:
- Nonexistent short codes (404)
- Invalid URL formats (400)
- Empty URLs (400)
- Malformed JSON (400)
- Analytics for missing URLs (404)

**What it validates:**
- Input validation
- Error responses
- HTTP status codes
- Error message formats

#### TestE2E_APIEndpoints
Tests all API endpoints:
- POST /api/urls - Create URL
- GET /api/urls - List URLs
- GET /api/urls/{shortCode}/analytics - Get analytics
- DELETE /api/urls/{shortCode} - Delete URL

**What it validates:**
- Complete API surface
- Request/response formats
- CRUD operations
- Data persistence

#### TestE2E_MultipleClicks
Tests analytics with multiple redirects:
- Multiple clicks from different referrers
- Async click recording
- Analytics aggregation

**What it validates:**
- Click tracking accuracy
- Referrer tracking
- Async worker queue
- Analytics computation

#### TestE2E_ConcurrentCreation
Tests concurrent URL creation:
- 10 simultaneous URL creation requests
- Database concurrency handling

**What it validates:**
- Thread safety
- Database locking
- Race condition handling

#### TestE2E_HealthCheck
Tests health check endpoint:
- Successful response
- No authentication required

**What it validates:**
- Monitoring endpoints
- Public access routes

### Other Integration Tests

#### API Integration Tests (`api_integration_test.go`)
- Detailed API endpoint testing
- Request validation
- Response format verification

#### Auth Integration Tests (`auth_integration_test.go`)
- Authentication middleware
- Protected vs public routes
- User context propagation

#### Redirect Integration Tests (`redirect_integration_test.go`)
- URL redirection logic
- Click tracking
- Analytics recording

#### Analytics Integration Tests (`analytics_integration_test.go`)
- Analytics computation
- Time range filtering
- Data aggregation

#### Server Integration Tests (`integration_test.go`)
- Middleware execution order
- CORS handling
- Concurrent requests
- Graceful shutdown

## Test Database Setup

All integration tests use **in-memory SQLite databases** for isolation and speed:

```go
db := setupTestDB(t)
defer db.Close()
```

The `setupTestDB` helper:
1. Creates an in-memory SQLite database (`:memory:`)
2. Applies all migrations using goose
3. Returns a ready-to-use database connection

**Benefits:**
- **Fast** - No disk I/O
- **Isolated** - Each test gets a fresh database
- **Portable** - No external dependencies
- **Deterministic** - Clean state for every test

## Test Patterns

### Table-Driven Tests

Most tests use table-driven patterns for comprehensive scenario coverage:

```go
tests := []struct {
    name           string
    authHeader     string
    expectedStatus int
    checkResponse  func(t *testing.T, body map[string]interface{})
}{
    {
        name:           "valid_token",
        authHeader:     "Bearer test-token",
        expectedStatus: http.StatusCreated,
    },
    // More test cases...
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        // Test logic
    })
}
```

### Subtests

Related scenarios are grouped as subtests:

```go
t.Run("create_url", func(t *testing.T) {
    // Create URL test
    
    t.Run("redirect", func(t *testing.T) {
        // Redirect test using created URL
    })
})
```

### Helper Functions

Common test utilities:
- `setupTestDB(t)` - Create test database
- `testLogger()` - Get a disabled logger for tests
- Standard Go testing helpers

## Best Practices

### Test Isolation
- Each test creates its own database
- No shared state between tests
- Independent test execution

### Async Operations
For async operations like click recording, use polling with a deadline instead of a fixed sleep:
```go
// Trigger redirect
srv.router.ServeHTTP(rec, req)

// Wait for async processing using polling with a deadline
deadline := time.Now().Add(2 * time.Second)

for {
    // Check condition (e.g., click recorded in analytics)
    analyticsReq := httptest.NewRequest(http.MethodGet, "/api/urls/"+shortCode+"/analytics", nil)
    analyticsReq.Header.Set("Authorization", "Bearer test-token")
    
    analyticsRec := httptest.NewRecorder()
    srv.router.ServeHTTP(analyticsRec, analyticsReq)
    
    if analyticsRec.Code == http.StatusOK {
        var analytics map[string]interface{}
        if err := json.Unmarshal(analyticsRec.Body.Bytes(), &analytics); err == nil {
            if totalClicks, ok := analytics["total_clicks"].(float64); ok && totalClicks > 0 {
                break
            }
        }
    }

    if time.Now().After(deadline) {
        t.Fatalf("timeout waiting for click to be recorded")
    }

    time.Sleep(10 * time.Millisecond)
}
```

### Concurrent Tests
Use proper synchronization:
```go
results := make(chan error, numRequests)
for i := 0; i < numRequests; i++ {
    go func(index int) {
        // Test logic
        results <- nil
    }(i)
}

// Collect results
for i := 0; i < numRequests; i++ {
    if err := <-results; err != nil {
        t.Error(err)
    }
}
```

### Error Messages
Provide context in error messages:
```go
if rec.Code != tt.expectedStatus {
    t.Errorf("expected status %d, got %d. Body: %s", 
        tt.expectedStatus, rec.Code, rec.Body.String())
}
```

## CI/CD Integration

CI runs on GitHub Actions (see `.github/workflows/ci.yml`) and executes `go test ./...` (plus separate formatting/vet/lint and generated-code checks).

Integration tests live under `internal/infrastructure/http/server/` and can be run locally with:

```bash
make test-integration
```

### Docker Compose Testing

Docker Compose is intended for manual local testing with a persistent SQLite database (bind-mounted to `./data`). See [Docker Compose testing](/operations/docker-compose-testing/).

## Coverage

To get a coverage report for the HTTP integration suite:

```bash
go test -cover ./internal/infrastructure/http/server/
```

Detailed coverage:
```bash
go test -coverprofile=coverage.out ./internal/infrastructure/http/server/
go tool cover -html=coverage.out
```

## Troubleshooting

### Tests Fail with "database is locked"
**Cause:** SQLite doesn't handle high concurrency well in tests.
**Fix:** Reduce concurrent operations or add delays between operations.

### Async tests are flaky
**Cause:** Not waiting long enough for async operations.
**Fix:** Prefer polling with a deadline (the suite already uses this pattern); increase the polling deadline if needed.

### Port already in use
**Cause:** Tests using real HTTP server instead of httptest.
**Fix:** Use `httptest.NewRecorder()` instead of real HTTP server.

## Adding New Integration Tests

### 1. Create Test File
Place in `internal/infrastructure/http/server/`:
```go
package server

import (
    "testing"
    // imports...
)
```

### 2. Follow Naming Convention
- File: `*_integration_test.go` or `*_test.go`
- Function: `TestE2E_*` or `Test*Integration`

### 3. Use Test Helpers
```go
func TestYourFeature(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()
    
    cfg := &config.Config{
        ServerPort:     8080,
        BaseURL:        "http://localhost:8080",
        DatabaseURL:    "test.db",
        AuthToken:      "test-token",
        AllowedOrigins: "*",
    }
    
    srv, err := New(cfg, db, testLogger())
    if err != nil {
        t.Fatalf("failed to create server: %v", err)
    }
    defer srv.Shutdown(context.Background())
    
    // Test logic here
}
```

### 4. Use httptest for HTTP Testing
```go
req := httptest.NewRequest(http.MethodPost, "/api/urls", body)
req.Header.Set("Authorization", "Bearer test-token")
rec := httptest.NewRecorder()

srv.router.ServeHTTP(rec, req)

if rec.Code != http.StatusCreated {
    t.Errorf("expected %d, got %d", http.StatusCreated, rec.Code)
}
```

### 5. Clean Up Resources
Always use `defer` for cleanup:
```go
db := setupTestDB(t)
defer db.Close()

srv, _ := New(cfg, db, logger)
defer srv.Shutdown(context.Background())
```

## Performance Benchmarks

Integration tests include benchmarks:

```bash
# Run benchmarks
go test -bench=. ./internal/infrastructure/http/server/

# With memory profiling
go test -bench=. -benchmem ./internal/infrastructure/http/server/
```

Example benchmark results:
```
BenchmarkServer_HealthCheck-8           500000      2314 ns/op
BenchmarkServer_WithMiddleware-8        200000      8745 ns/op
```

## Related Documentation

- [Docker Compose testing](/operations/docker-compose-testing/) - Docker Compose testing guide
- [README.md](https://github.com/matt-riley/mjrwtf/blob/main/README.md) - Main project documentation
- [Contributing](/contributing/) - Contribution guidelines

## Summary

The integration testing suite provides:

✅ **Comprehensive Coverage** - All API endpoints and workflows tested
✅ **Fast Execution** - In-memory database, ~1 second total
✅ **Reliable** - Designed to minimize flakiness with deterministic assertions (some tests use time-based polling and may be timing-sensitive on heavily loaded environments)
✅ **CI/CD Ready** - No external dependencies required
✅ **Well Organized** - Clear test structure and naming
✅ **Documented** - Examples and patterns for new tests
✅ **Maintainable** - Helper functions and consistent patterns

The tests serve as both quality assurance and living documentation of system behavior.
