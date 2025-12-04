# Discord Error Notification System

This package provides a Discord webhook integration for sending critical error notifications to a Discord channel.

## Features

- **Discord Webhook Integration**: Sends formatted error notifications to Discord
- **Rate Limiting**: Prevents spam with configurable rate limits (default: 1 message per minute per error type)
- **Non-blocking**: Uses background goroutines to send notifications without blocking request handling
- **Graceful Degradation**: Handles Discord API failures without crashing the application
- **Rich Formatting**: Uses Discord embeds with color coding, fields, and code blocks for stack traces
- **Contextual Information**: Includes request ID, method, path, user ID, timestamp, and stack traces

## Configuration

The Discord webhook URL is configured via the `DISCORD_WEBHOOK_URL` environment variable (see `.env.example`).

If the webhook URL is not set or is empty, the notifier is disabled and no notifications will be sent.

## Usage

### Basic Setup

```go
import (
    "github.com/matt-riley/mjrwtf/internal/adapters/notification"
    "github.com/matt-riley/mjrwtf/internal/infrastructure/config"
)

// Load configuration
cfg, err := config.LoadConfig()
if err != nil {
    // handle error
}

// Create Discord notifier
notifier := notification.NewDiscordNotifier(cfg.DiscordWebhookURL)
```

### With Custom Options

```go
import (
    "time"
    "github.com/matt-riley/mjrwtf/internal/adapters/notification"
    "github.com/rs/zerolog"
)

logger := zerolog.New(os.Stdout)

notifier := notification.NewDiscordNotifier(
    webhookURL,
    notification.WithLogger(logger),              // Custom logger
    notification.WithRateLimit(2 * time.Minute),  // Custom rate limit
    notification.WithAsyncSend(true),             // Enable async sending (default)
)
```

### Integration with Recovery Middleware

```go
import (
    "github.com/matt-riley/mjrwtf/internal/adapters/notification"
    "github.com/matt-riley/mjrwtf/internal/infrastructure/http/middleware"
    "github.com/rs/zerolog"
)

logger := zerolog.New(os.Stdout)
notifier := notification.NewDiscordNotifier(cfg.DiscordWebhookURL)

// Create recovery middleware with Discord notifications
recoveryMiddleware := middleware.RecoveryWithNotifier(logger, notifier)

// Use in your HTTP router
router.Use(recoveryMiddleware)
```

### Manual Error Notification

```go
import (
    "context"
    "time"
    "github.com/matt-riley/mjrwtf/internal/adapters/notification"
)

notifier := notification.NewDiscordNotifier(webhookURL)

// Send error notification manually
errCtx := notification.ErrorContext{
    ErrorMessage: "Something went wrong",
    StackTrace:   "stack trace here",
    RequestID:    "req-123",
    Method:       "GET",
    Path:         "/api/endpoint",
    UserID:       "user-456",
    Timestamp:    time.Now(),
}

notifier.NotifyError(context.Background(), errCtx)
```

## Rate Limiting

The notifier implements rate limiting per error type to prevent spam:

- **Default**: 1 message per minute per error type
- **Error Type**: Determined by the first 50 characters of the error message
- **Behavior**: If an error of the same type occurs within the rate limit window, it will be silently dropped

### Configuring Rate Limits

```go
// Allow 1 notification per 5 minutes per error type
notifier := notification.NewDiscordNotifier(
    webhookURL,
    notification.WithRateLimit(5 * time.Minute),
)
```

## Discord Message Format

Error notifications are sent as Discord embeds with the following structure:

- **Title**: 🚨 Critical Error Detected
- **Color**: Red (#FF0000)
- **Fields**:
  - Error Message (truncated to 1024 chars)
  - Request ID (if available)
  - Request Method and Path (if available)
  - User ID (if available)
  - Stack Trace (truncated to 1000 chars, formatted as code block)
- **Timestamp**: ISO 8601 format
- **Footer**: "mjr.wtf Error Notification"

## Testing

The package includes comprehensive tests:

```bash
go test -v ./internal/adapters/notification/...
```

Tests cover:
- Successful notification sending
- Rate limiting functionality
- Handling Discord API failures
- Handling network errors
- Message formatting
- Disabled notifier behavior

## Error Handling

The notifier is designed to fail gracefully:

- **Network Errors**: Logged but do not crash the application
- **Discord API Errors**: Logged but do not crash the application
- **Invalid Configuration**: Silently disabled if webhook URL is empty
- **Non-blocking**: All notifications are sent asynchronously by default

## Production Considerations

1. **Webhook URL Security**: Store the Discord webhook URL securely in environment variables, not in code
2. **Rate Limiting**: Consider the rate limits to avoid notification fatigue
3. **Discord Limits**: Discord has its own rate limits (30 requests per minute per webhook)
4. **Monitoring**: Monitor your logs for failed notification attempts
5. **Testing**: Test with a test Discord channel before deploying to production

## Example Discord Setup

1. Create a Discord server (or use an existing one)
2. Create a channel for error notifications (e.g., `#app-errors`)
3. Go to channel settings → Integrations → Webhooks → New Webhook
4. Copy the webhook URL
5. Add to your `.env` file:

```bash
DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_WEBHOOK_TOKEN
```

## Architecture

The Discord notifier follows the hexagonal architecture pattern:

- **Adapter Layer**: This package (`internal/adapters/notification`)
- **Purpose**: External integration for error notifications
- **Dependencies**: Only depends on standard library and zerolog for logging
- **No Domain Logic**: Pure adapter code, no business logic

## Related Components

- `internal/infrastructure/http/middleware/recovery.go`: Recovery middleware that uses the notifier
- `internal/infrastructure/config/config.go`: Configuration management including Discord webhook URL
- `internal/infrastructure/logging/logger.go`: Logging infrastructure
