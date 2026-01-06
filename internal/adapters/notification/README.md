# Discord Error Notification System

This package provides a Discord webhook integration for sending critical error notifications to a Discord channel.

## Features

- **Discord Webhook Integration**: Sends formatted error notifications to Discord
- **Rate Limiting**: Prevents spam with configurable rate limits (default: 1 message per minute per error type)
- **Non-blocking**: Uses background goroutines to send notifications without blocking request handling
- **Graceful Degradation**: Handles Discord API failures without crashing the application
- **Rich Formatting**: Uses Discord embeds with color coding, fields, and (optional) code blocks for stack traces
- **Contextual Information**: Can include request ID, method, path, user ID, timestamp, and stack traces

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

cfg, err := config.LoadConfig()
if err != nil {
    // handle error
}

notifier := notification.NewDiscordNotifier(cfg.DiscordWebhookURL)
```

### With Custom Options

```go
import (
    "os"
    "time"

    "github.com/matt-riley/mjrwtf/internal/adapters/notification"
    "github.com/rs/zerolog"
)

logger := zerolog.New(os.Stdout)

notifier := notification.NewDiscordNotifier(
    webhookURL,
    notification.WithLogger(logger),
    notification.WithRateLimit(2*time.Minute),
    notification.WithAsyncSend(true),
)
```

### Integration with Recovery Middleware

The HTTP recovery middleware can pass request context (request id, method/path, user id) into the notifier.

Stack traces are only included when `LOG_STACK_TRACES` is enabled (middleware controls whether a stack trace is captured).

```go
recoveryMiddleware := middleware.RecoveryWithNotifier(logger, notifier)
router.Use(recoveryMiddleware)
```

### Manual Error Notification

```go
import (
    "context"
    "time"

    "github.com/matt-riley/mjrwtf/internal/adapters/notification"
)

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

## Testing

```bash
go test -v ./internal/adapters/notification/...
```

## Related Components

- `internal/infrastructure/http/middleware/recovery.go`: recovery middleware that can notify Discord
- `internal/infrastructure/config/config.go`: configuration (includes `DISCORD_WEBHOOK_URL`)
