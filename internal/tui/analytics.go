package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matt-riley/mjrwtf/internal/client"
	"github.com/matt-riley/mjrwtf/internal/tui/tui_config"
)

type getAnalyticsMsg struct {
	resp *client.GetAnalyticsResponse
	err  error
}

func getAnalyticsCmd(cfg tui_config.Config, shortCode string, startTime, endTime *time.Time) tea.Cmd {
	return func() tea.Msg {
		base := strings.TrimSpace(cfg.BaseURL)
		if base == "" {
			return getAnalyticsMsg{err: fmt.Errorf("base URL not set")}
		}

		c, err := client.New(base, client.WithToken(cfg.Token), client.WithTimeout(5*time.Second))
		if err != nil {
			return getAnalyticsMsg{err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()

		resp, err := c.GetAnalytics(ctx, shortCode, startTime, endTime)
		if err != nil {
			return getAnalyticsMsg{err: err}
		}
		return getAnalyticsMsg{resp: resp}
	}
}
