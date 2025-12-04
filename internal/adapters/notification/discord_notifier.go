package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// ErrorContext contains contextual information about an error
type ErrorContext struct {
	ErrorMessage string
	StackTrace   string
	RequestID    string
	Method       string
	Path         string
	UserID       string
	Timestamp    time.Time
}

// DiscordNotifier sends error notifications to Discord via webhooks
type DiscordNotifier struct {
	webhookURL    string
	client        *http.Client
	logger        zerolog.Logger
	rateLimiter   *rateLimiter
	sendAsync     bool
	errorTypeHash map[string]time.Time
	mu            sync.RWMutex
}

// rateLimiter implements rate limiting per error type
type rateLimiter struct {
	lastSent map[string]time.Time
	mu       sync.RWMutex
	interval time.Duration
}

func newRateLimiter(interval time.Duration) *rateLimiter {
	return &rateLimiter{
		lastSent: make(map[string]time.Time),
		interval: interval,
	}
}

func (rl *rateLimiter) allow(errorType string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	lastSent, exists := rl.lastSent[errorType]
	if !exists || time.Since(lastSent) >= rl.interval {
		rl.lastSent[errorType] = time.Now()
		return true
	}
	return false
}

// DiscordNotifierOption is a functional option for configuring the notifier
type DiscordNotifierOption func(*DiscordNotifier)

// WithLogger sets a custom logger for the notifier
func WithLogger(logger zerolog.Logger) DiscordNotifierOption {
	return func(n *DiscordNotifier) {
		n.logger = logger
	}
}

// WithHTTPClient sets a custom HTTP client for the notifier
func WithHTTPClient(client *http.Client) DiscordNotifierOption {
	return func(n *DiscordNotifier) {
		n.client = client
	}
}

// WithRateLimit sets the rate limit interval (default: 1 minute)
func WithRateLimit(interval time.Duration) DiscordNotifierOption {
	return func(n *DiscordNotifier) {
		n.rateLimiter = newRateLimiter(interval)
	}
}

// WithAsyncSend enables or disables async sending (default: true)
func WithAsyncSend(async bool) DiscordNotifierOption {
	return func(n *DiscordNotifier) {
		n.sendAsync = async
	}
}

// NewDiscordNotifier creates a new Discord notifier
func NewDiscordNotifier(webhookURL string, opts ...DiscordNotifierOption) *DiscordNotifier {
	notifier := &DiscordNotifier{
		webhookURL:    webhookURL,
		client:        &http.Client{Timeout: 10 * time.Second},
		logger:        zerolog.Nop(),
		rateLimiter:   newRateLimiter(1 * time.Minute),
		sendAsync:     true,
		errorTypeHash: make(map[string]time.Time),
	}

	for _, opt := range opts {
		opt(notifier)
	}

	return notifier
}

// NotifyError sends an error notification to Discord
func (n *DiscordNotifier) NotifyError(ctx context.Context, errCtx ErrorContext) {
	// Skip if webhook URL is not configured
	if n.webhookURL == "" {
		return
	}

	// Generate error type hash from error message (simplified)
	errorType := generateErrorType(errCtx.ErrorMessage)

	// Check rate limit
	if !n.rateLimiter.allow(errorType) {
		n.logger.Debug().
			Str("error_type", errorType).
			Msg("error notification rate limited")
		return
	}

	if n.sendAsync {
		go n.sendNotification(errCtx)
	} else {
		n.sendNotification(errCtx)
	}
}

// sendNotification sends the actual notification to Discord
func (n *DiscordNotifier) sendNotification(errCtx ErrorContext) {
	payload := n.formatMessage(errCtx)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		n.logger.Error().Err(err).Msg("failed to marshal Discord payload")
		return
	}

	req, err := http.NewRequest("POST", n.webhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		n.logger.Error().Err(err).Msg("failed to create Discord webhook request")
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		n.logger.Error().Err(err).Msg("failed to send Discord notification")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		n.logger.Error().
			Int("status_code", resp.StatusCode).
			Msg("Discord webhook returned error status")
		return
	}

	n.logger.Debug().
		Str("request_id", errCtx.RequestID).
		Msg("error notification sent to Discord")
}

// formatMessage formats the error context into a Discord webhook payload
func (n *DiscordNotifier) formatMessage(errCtx ErrorContext) map[string]interface{} {
	// Color coding: Red for errors
	color := 0xFF0000 // Red

	fields := []map[string]interface{}{
		{
			"name":   "Error Message",
			"value":  truncate(errCtx.ErrorMessage, 1024),
			"inline": false,
		},
	}

	if errCtx.RequestID != "" {
		fields = append(fields, map[string]interface{}{
			"name":   "Request ID",
			"value":  errCtx.RequestID,
			"inline": true,
		})
	}

	if errCtx.Method != "" && errCtx.Path != "" {
		fields = append(fields, map[string]interface{}{
			"name":   "Request",
			"value":  fmt.Sprintf("%s %s", errCtx.Method, errCtx.Path),
			"inline": true,
		})
	}

	if errCtx.UserID != "" {
		fields = append(fields, map[string]interface{}{
			"name":   "User ID",
			"value":  errCtx.UserID,
			"inline": true,
		})
	}

	if errCtx.StackTrace != "" {
		// Truncate stack trace to fit Discord's limits
		stackTrace := truncate(errCtx.StackTrace, 1000)
		fields = append(fields, map[string]interface{}{
			"name":   "Stack Trace",
			"value":  fmt.Sprintf("```\n%s\n```", stackTrace),
			"inline": false,
		})
	}

	embed := map[string]interface{}{
		"title":       "🚨 Critical Error Detected",
		"description": "A critical error has occurred in the application",
		"color":       color,
		"fields":      fields,
		"timestamp":   errCtx.Timestamp.Format(time.RFC3339),
		"footer": map[string]interface{}{
			"text": "mjr.wtf Error Notification",
		},
	}

	return map[string]interface{}{
		"embeds": []map[string]interface{}{embed},
	}
}

// generateErrorType generates a simple error type identifier from error message
func generateErrorType(errorMsg string) string {
	// Take first 50 characters as error type for rate limiting purposes
	if len(errorMsg) > 50 {
		return errorMsg[:50]
	}
	return errorMsg
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// IsEnabled returns true if the notifier is configured with a webhook URL
func (n *DiscordNotifier) IsEnabled() bool {
	return n.webhookURL != ""
}
