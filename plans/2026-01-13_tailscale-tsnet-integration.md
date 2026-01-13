# Tailscale tsnet Integration - Implementation Plan

Date: 2026-01-13
Research: Tailscale authentication research (saved in Pieces memory 2026-01-13)

## Goal

Integrate Tailscale authentication using the tsnet library to protect admin routes (dashboard, API endpoints) while keeping public routes (index page, short URL redirects) accessible to everyone. The application will become a Tailscale node, with admin access controlled exclusively by Tailscale network membership.

## Chosen Approach

**tsnet Library Approach**: Embed Tailscale directly into the Go application using `tailscale.com/tsnet`. The application will:
- Run as a full Tailscale node with its own IP on the tailnet
- Listen exclusively on the Tailscale network interface
- Expose public routes (index, redirects) via **Tailscale Funnel** (public internet access)
- Keep admin routes (dashboard, API) private to tailnet users via **Tailscale Serve**
- Use Tailscale **WhoIs API** to identify users and populate request context

**Why this approach:**
- Single binary deployment (no reverse proxy needed)
- Built-in TLS certificates from Tailscale
- Zero-config authentication (network membership = auth)
- Aligns with existing Go-native architecture
- Works seamlessly with existing TUI client

## Dependencies

### Packages to Install
- `tailscale.com/tsnet@latest` - Embed Tailscale node in Go application
- `tailscale.com/client/tailscale@latest` - Tailscale client library for WhoIs API

### Configuration Changes
- File: `internal/infrastructure/config/config.go`
- Add Tailscale configuration fields (hostname, auth key, funnel routes, etc.)

### Environment Variables
- `TAILSCALE_HOSTNAME` - Hostname for this Tailscale node (e.g., "mjrwtf")
- `TAILSCALE_AUTH_KEY` - Optional: Tailscale auth key for automatic registration
- `TAILSCALE_STATE_DIR` - Directory for Tailscale state storage (default: "./tailscale-state")
- `TAILSCALE_ENABLED` - Enable Tailscale mode (default: false for backward compatibility)
- `TAILSCALE_FUNNEL_ENABLED` - Enable Funnel for public routes (default: false)
- `TAILSCALE_CONTROL_URL` - Optional: Custom control server URL (for self-hosted)

### External Setup Required
- Tailscale account and tailnet
- Tailscale CLI installed for initial Funnel configuration
- Auth key generated from Tailscale admin console (optional, for auto-registration)

## Implementation Steps

### Step 1: Add Tailscale Dependencies

**Actions:**
- Run `go get tailscale.com/tsnet@latest`
- Run `go get tailscale.com/client/tailscale@latest`
- Run `go mod tidy`

**Files:**
- Modify: `go.mod`, `go.sum`

**Verification:**
```bash
go list -m tailscale.com/tsnet
go list -m tailscale.com/client/tailscale
```

### Step 2: Add Tailscale Configuration

**Actions:**
- Add Tailscale config fields to `Config` struct
- Add environment variable parsing for Tailscale settings
- Add validation logic for Tailscale configuration
- Update `.env.example` with Tailscale variables

**Files:**
- Modify: `internal/infrastructure/config/config.go`
- Modify: `internal/infrastructure/config/config_test.go` (add tests for new fields)
- Modify: `.env.example`

**Config fields to add:**
```go
// Tailscale configuration
TailscaleEnabled       bool
TailscaleHostname      string
TailscaleAuthKey       string
TailscaleStateDir      string
TailscaleFunnelEnabled bool
TailscaleControlURL    string
```

**Verification:**
```bash
make test # Ensure config tests pass
```

### Step 3: Create Tailscale Network Layer

**Actions:**
- Create new package `internal/infrastructure/tailscale/`
- Implement `TailscaleServer` wrapper around `tsnet.Server`
- Add initialization and listener creation logic
- Add graceful shutdown support
- Add error handling and logging

**Files:**
- Create: `internal/infrastructure/tailscale/server.go`
- Create: `internal/infrastructure/tailscale/doc.go`
- Create: `internal/infrastructure/tailscale/server_test.go`

**Key functionality:**
- `NewTailscaleServer(cfg, logger)` - Initialize tsnet.Server
- `Listen(network, addr)` - Return net.Listener for HTTP server
- `Hostname()` - Get assigned Tailscale hostname
- `Close()` - Graceful shutdown

**Verification:**
- Unit tests for server initialization
- Mock tests for listener creation

### Step 4: Create Tailscale WhoIs Middleware

**Actions:**
- Create new middleware for Tailscale user authentication
- Implement WhoIs API call to get user identity from remote IP
- Extract user info (login, name, email) and store in request context
- Add error handling for WhoIs failures
- Add logging for authentication events

**Files:**
- Create: `internal/infrastructure/http/middleware/tailscale_auth.go`
- Create: `internal/infrastructure/http/middleware/tailscale_auth_test.go`

**Middleware signature:**
```go
func TailscaleAuth(tsServer *tailscale.TailscaleServer) func(http.Handler) http.Handler
```

**Context keys to add:**
```go
const (
    TailscaleUserKey     contextKey = "tailscale_user"
    TailscaleLoginKey    contextKey = "tailscale_login"
    TailscaleNameKey     contextKey = "tailscale_name"
)
```

**Verification:**
- Unit tests with mocked WhoIs responses
- Test cases: valid user, invalid user, network errors

### Step 5: Update Server Initialization

**Actions:**
- Modify `cmd/server/main.go` to support both standard and Tailscale modes
- Add conditional logic: if Tailscale enabled, use tsnet listener; otherwise use standard net listener
- Pass Tailscale server instance to HTTP server
- Update shutdown logic to close Tailscale connection

**Files:**
- Modify: `cmd/server/main.go`

**Logic flow:**
```go
if cfg.TailscaleEnabled {
    tsServer := tailscale.NewTailscaleServer(cfg, logger)
    listener := tsServer.Listen("tcp", ":443")
    // Pass listener to HTTP server
} else {
    // Existing standard HTTP server logic
}
```

**Verification:**
- Test both modes: `TAILSCALE_ENABLED=false` (standard) and `TAILSCALE_ENABLED=true` (tsnet)
- Verify server starts correctly in each mode

### Step 6: Update HTTP Server for Tailscale

**Actions:**
- Modify `server.New()` to accept optional Tailscale server parameter
- Update `setupRoutes()` to conditionally apply Tailscale middleware to admin routes
- Keep Bearer token auth as fallback when Tailscale is disabled
- Ensure public routes remain unauthenticated

**Files:**
- Modify: `internal/infrastructure/http/server/server.go`

**Route protection strategy:**
```go
// Public routes (exposed via Funnel)
s.router.Get("/", pageHandler.Home)
s.router.Get("/{shortCode}", redirectHandler.Redirect)

// Admin routes (tailnet-only)
if s.tailscaleServer != nil {
    // Use Tailscale WhoIs auth
    s.router.With(middleware.TailscaleAuth(s.tailscaleServer)).Get("/dashboard", ...)
    s.router.Route("/api", func(r chi.Router) {
        r.Use(middleware.TailscaleAuth(s.tailscaleServer))
        // API routes
    })
} else {
    // Fallback to existing Bearer/Session auth
    s.router.With(middleware.RequireSession(...)).Get("/dashboard", ...)
    s.router.Route("/api", func(r chi.Router) {
        r.Use(middleware.SessionOrBearerAuth(...))
        // API routes
    })
}
```

**Verification:**
- Test route protection with Tailscale enabled
- Test fallback behavior with Tailscale disabled

### Step 7: Update Page Handlers for Tailscale

**Actions:**
- Modify dashboard and admin page handlers to read Tailscale user from context
- Display Tailscale user identity in UI instead of login form
- Remove login/logout routes when Tailscale is enabled (network auth replaces forms)
- Update templates to show Tailscale user info

**Files:**
- Modify: `internal/infrastructure/http/handlers/page_handler.go`
- Modify: `internal/infrastructure/http/handlers/templates/*.templ` (if needed)

**User context extraction:**
```go
if tsUser := r.Context().Value(middleware.TailscaleUserKey); tsUser != nil {
    user := tsUser.(string) // e.g., "alice@example.com"
    // Use in template rendering
}
```

**Verification:**
- Manual test: access dashboard via Tailscale
- Verify user identity displayed correctly

### Step 8: Add Integration Tests

**Actions:**
- Create integration tests for Tailscale mode
- Test public route accessibility without auth
- Test admin route protection with WhoIs auth
- Test error cases (network issues, invalid users)

**Files:**
- Create: `internal/infrastructure/http/server/tailscale_integration_test.go`

**Test scenarios:**
- Public routes accessible without Tailscale auth
- Admin routes require valid Tailscale user
- WhoIs failure returns 401
- Both Tailscale and standard modes work correctly

**Verification:**
```bash
make test-integration
```

### Step 9: Update Documentation

**Actions:**
- Document Tailscale configuration in README.md
- Add setup guide for Tailscale mode
- Document Funnel configuration commands
- Update deployment documentation
- Add troubleshooting section

**Files:**
- Modify: `README.md`
- Modify: `docs-site/docs/getting-started/configuration.md` (if docs site exists)
- Create: `docs/tailscale-setup.md`

**Documentation sections:**
- Prerequisites (Tailscale account, auth key)
- Environment variable reference
- Funnel configuration commands
- ACL recommendations
- Migration from Bearer token auth

**Verification:**
- Review documentation for accuracy and completeness

### Step 10: Configure Tailscale Funnel

**Actions:**
- Document Funnel configuration for public routes
- Create helper script or Makefile target for Funnel setup
- Test public route accessibility via Funnel URL

**Funnel commands to document:**
```bash
# Enable Funnel for public routes
tailscale funnel --bg --https=443 --set-path=/ on
tailscale funnel --bg --https=443 --set-path=/{shortCode} on

# Verify Funnel status
tailscale funnel status
```

**Files:**
- Create: `scripts/configure-tailscale-funnel.sh`
- Modify: `Makefile` (add `make tailscale-funnel-setup` target)

**Verification:**
- Access public routes via Funnel URL from non-Tailscale network
- Verify admin routes NOT accessible via Funnel

### Step 11: Update Docker Configuration

**Actions:**
- Update Dockerfile to support Tailscale state directory
- Add volume mount for Tailscale state persistence
- Update docker-compose.yml with Tailscale environment variables
- Document Docker deployment with Tailscale

**Files:**
- Modify: `Dockerfile`
- Modify: `docker-compose.yml`
- Modify: `DOCKER_COMPOSE_TESTING.md`

**Docker considerations:**
- Tailscale state directory needs persistent volume
- Auth key should be passed as environment variable or secret
- Container needs appropriate capabilities for Tailscale

**Verification:**
```bash
make docker-compose-up
# Verify Tailscale initialization in container logs
```

### Step 12: Add Backward Compatibility Flag

**Actions:**
- Ensure application works in both Tailscale and standard modes
- Add deprecation warning for Bearer token auth when Tailscale is available
- Document migration path from Bearer tokens to Tailscale

**Files:**
- Modify: `internal/infrastructure/config/config.go` (validation warnings)
- Modify: `cmd/server/main.go` (startup logging)

**Logging to add:**
```go
if cfg.TailscaleEnabled {
    logger.Info().Msg("Tailscale mode enabled - admin routes protected by network authentication")
} else {
    logger.Warn().Msg("Running in standard mode - consider enabling Tailscale for improved security")
}
```

**Verification:**
- Test both modes work correctly
- Verify appropriate warnings logged

## File Impact

### Created
- `internal/infrastructure/tailscale/server.go` - Tailscale server wrapper
- `internal/infrastructure/tailscale/doc.go` - Package documentation
- `internal/infrastructure/tailscale/server_test.go` - Unit tests
- `internal/infrastructure/http/middleware/tailscale_auth.go` - WhoIs authentication middleware
- `internal/infrastructure/http/middleware/tailscale_auth_test.go` - Middleware tests
- `internal/infrastructure/http/server/tailscale_integration_test.go` - Integration tests
- `scripts/configure-tailscale-funnel.sh` - Funnel setup helper script
- `docs/tailscale-setup.md` - Tailscale setup documentation
- `plans/2026-01-13_tailscale-tsnet-integration.md` - This implementation plan

### Modified
- `go.mod` - Add Tailscale dependencies
- `go.sum` - Dependency checksums
- `.env.example` - Add Tailscale environment variables
- `internal/infrastructure/config/config.go` - Add Tailscale configuration
- `internal/infrastructure/config/config_test.go` - Test new config fields
- `cmd/server/main.go` - Support Tailscale listener mode
- `internal/infrastructure/http/server/server.go` - Conditional route protection
- `internal/infrastructure/http/handlers/page_handler.go` - Support Tailscale user context
- `internal/infrastructure/http/middleware/auth.go` - Document context keys
- `README.md` - Add Tailscale documentation section
- `Dockerfile` - Support Tailscale state directory
- `docker-compose.yml` - Add Tailscale environment variables
- `DOCKER_COMPOSE_TESTING.md` - Document Tailscale Docker setup
- `Makefile` - Add Tailscale-related targets

### Deleted
None - this is additive; existing auth mechanisms remain as fallback

## Verification Checklist

- [ ] Step 1: Tailscale dependencies installed and verified
- [ ] Step 2: Configuration updated and tests passing
- [ ] Step 3: Tailscale server layer created and tested
- [ ] Step 4: WhoIs middleware created and unit tested
- [ ] Step 5: Server initialization updated for both modes
- [ ] Step 6: HTTP server routes conditionally protected
- [ ] Step 7: Page handlers updated for Tailscale user context
- [ ] Step 8: Integration tests passing
- [ ] Step 9: Documentation updated
- [ ] Step 10: Funnel configuration documented and tested
- [ ] Step 11: Docker configuration updated
- [ ] Step 12: Backward compatibility verified
- [ ] All unit tests passing: `make test-unit`
- [ ] All integration tests passing: `make test-integration`
- [ ] Build successful: `make build`
- [ ] Linting passes: `make lint` (if configured)
- [ ] Manual testing: Standard mode works
- [ ] Manual testing: Tailscale mode works
- [ ] Manual testing: Public routes accessible via Funnel
- [ ] Manual testing: Admin routes protected by Tailscale auth
- [ ] Manual testing: TUI client connects successfully

## Risk & Rollback

### Potential Risks

1. **Breaking existing deployments** (Medium risk)
   - Mitigation: Use feature flag (`TAILSCALE_ENABLED=false` by default)
   - Existing deployments continue using Bearer token auth unless explicitly enabled

2. **Tailscale network latency for public redirects** (Low-Medium risk)
   - Funnel routes traffic through Tailscale relay, adding latency
   - Mitigation: Monitor redirect response times, consider keeping public server separate if latency unacceptable
   - Alternative: Hybrid deployment (public server + Tailscale admin server)

3. **Tailscale state corruption** (Low risk)
   - State directory corruption could prevent server startup
   - Mitigation: Persistent volume for state, regular backups, documented recovery process

4. **WhoIs API failures** (Low risk)
   - Network issues or Tailscale daemon problems could block admin access
   - Mitigation: Graceful error handling, retry logic, fallback to error page with instructions

5. **Docker container capabilities** (Medium risk)
   - Tailscale may require specific container capabilities
   - Mitigation: Document required capabilities, test in Docker environment

### Rollback Strategy

**If Tailscale integration causes issues:**

1. **Immediate rollback:**
   ```bash
   # Disable Tailscale mode
   export TAILSCALE_ENABLED=false
   # Restart server
   make docker-compose-restart
   ```

2. **Git rollback:**
   ```bash
   # Revert to previous version
   git revert <commit-hash>
   # Or reset to previous release
   git checkout <previous-tag>
   ```

3. **Configuration rollback:**
   - Remove Tailscale environment variables from `.env`
   - Ensure `AUTH_TOKENS` or `AUTH_TOKEN` is set
   - Restart application

4. **Docker rollback:**
   ```bash
   # Use previous Docker image
   docker pull ghcr.io/matt-riley/mjrwtf:<previous-version>
   docker-compose up -d
   ```

**Backup before implementation:**
- Git commit: "Pre-Tailscale implementation snapshot"
- Database backup: `cp database.db database.db.backup`
- Configuration backup: `cp .env .env.backup`

### Post-Implementation Monitoring

**Metrics to monitor:**
- Redirect latency (compare before/after Funnel)
  - Check Prometheus metrics: `http_request_duration_seconds{path="/{shortCode}"}`
- Admin route authentication success/failure rates
  - Monitor 401 responses on `/api/*` and `/dashboard`
- Server startup time with Tailscale initialization
  - Check logs for initialization duration
- WhoIs API call latency and error rates
  - Add custom metrics for WhoIs performance

**Logs to check:**
- Tailscale connection establishment: "Tailscale server started"
- WhoIs authentication events: "User authenticated via Tailscale"
- Funnel route configuration: "Funnel enabled for public routes"
- Error patterns: WhoIs timeouts, network failures

**Health checks:**
- `/health` endpoint still responsive
- `/ready` endpoint validates Tailscale connection status
- Public routes accessible via Funnel URL
- Admin routes accessible only from tailnet

**Alert conditions:**
- High 401 rate on admin routes (auth issues)
- Increased P95 latency on redirects (Funnel overhead)
- Server startup failures (Tailscale init problems)
- WhoIs API errors (network/daemon issues)

## References

- Research: Tailscale authentication research (Pieces memory, 2026-01-13)
- tsnet Documentation: https://tailscale.com/kb/1244/tsnet
- tsnet Package Docs: https://pkg.go.dev/tailscale.com/tsnet
- Funnel Documentation: https://tailscale.com/kb/1223/funnel
- Serve Documentation: https://tailscale.com/kb/1242/tailscale-serve
- WhoIs LocalAPI: https://dorianmonnier.fr/posts/2024-12-26-tailscale-localapi/
- Tailscale Go SDK: https://github.com/tailscale/tailscale-client-go-v2
- Example: Building with tsnet: https://tailscale.com/blog/building-tsidp

## Notes

- This implementation maintains backward compatibility - existing deployments can continue using Bearer token auth
- TUI client (`cmd/mjr`) requires no changes if running on a Tailscale network device
- Consider performance testing public redirects via Funnel before production rollout
- ACL configuration in Tailscale admin console can restrict which users access admin routes
- Future enhancement: Add Tailscale user identity to click analytics (track which admin created each URL)
