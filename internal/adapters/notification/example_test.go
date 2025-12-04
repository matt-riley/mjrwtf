package notification_test

import (
	"context"
	"fmt"
	"time"

	"github.com/matt-riley/mjrwtf/internal/adapters/notification"
)

// Example_basicUsage demonstrates basic usage of the Discord notifier
func Example_basicUsage() {
	// Create a Discord notifier with a webhook URL
	webhookURL := "https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_TOKEN"
	notifier := notification.NewDiscordNotifier(webhookURL)

	// Send an error notification
	errCtx := notification.ErrorContext{
		ErrorMessage: "Database connection failed",
		StackTrace:   "at database.go:42\nat main.go:10",
		RequestID:    "req-123",
		Method:       "POST",
		Path:         "/api/users",
		UserID:       "user-456",
		Timestamp:    time.Now(),
	}

	notifier.NotifyError(context.Background(), errCtx)
	fmt.Println("Notification sent to Discord")
	// Output: Notification sent to Discord
}

// Example_withCustomOptions demonstrates using custom options
func Example_withCustomOptions() {
	// Create notifier with custom rate limit
	notifier := notification.NewDiscordNotifier(
		"https://discord.com/api/webhooks/YOUR_WEBHOOK_ID/YOUR_TOKEN",
		notification.WithRateLimit(5*time.Minute), // 5 minute rate limit
		notification.WithAsyncSend(true),          // Send asynchronously
	)

	// Check if notifier is enabled
	if notifier.IsEnabled() {
		fmt.Println("Discord notifications are enabled")
	}
	// Output: Discord notifications are enabled
}

// Example_disabledNotifier demonstrates behavior when disabled
func Example_disabledNotifier() {
	// Empty webhook URL disables the notifier
	notifier := notification.NewDiscordNotifier("")

	// Check if notifier is enabled
	if !notifier.IsEnabled() {
		fmt.Println("Discord notifications are disabled")
	}

	// Calling NotifyError does nothing when disabled
	errCtx := notification.ErrorContext{
		ErrorMessage: "This won't be sent",
		Timestamp:    time.Now(),
	}
	notifier.NotifyError(context.Background(), errCtx)
	fmt.Println("No notification sent")

	// Output:
	// Discord notifications are disabled
	// No notification sent
}
