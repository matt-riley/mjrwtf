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

type deleteURLMsg struct {
	shortCode string
	err       error
}

func deleteURLCmd(cfg tui_config.Config, shortCode string) tea.Cmd {
	return func() tea.Msg {
		base := strings.TrimSpace(cfg.BaseURL)
		if base == "" {
			return deleteURLMsg{shortCode: shortCode, err: fmt.Errorf("base URL not set")}
		}

		c, err := client.New(base, client.WithToken(cfg.Token), client.WithTimeout(5*time.Second))
		if err != nil {
			return deleteURLMsg{shortCode: shortCode, err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()

		if err := c.DeleteURL(ctx, shortCode); err != nil {
			return deleteURLMsg{shortCode: shortCode, err: err}
		}
		return deleteURLMsg{shortCode: shortCode}
	}
}
