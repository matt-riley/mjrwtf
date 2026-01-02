package tui

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matt-riley/mjrwtf/internal/client"
	"github.com/matt-riley/mjrwtf/internal/tui/tui_config"
)

type createURLMsg struct {
	resp *client.CreateURLResponse
	err  error
}

func validateHTTPURL(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("Error: original URL is required")
	}

	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("Error: original URL must be a valid http or https URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("Error: original URL must start with http:// or https://")
	}
	return nil
}

func createURLCmd(cfg tui_config.Config, originalURL string) tea.Cmd {
	return func() tea.Msg {
		base := strings.TrimSpace(cfg.BaseURL)
		if base == "" {
			return createURLMsg{err: fmt.Errorf("base URL not set")}
		}

		c, err := client.New(base, client.WithToken(cfg.Token), client.WithTimeout(5*time.Second))
		if err != nil {
			return createURLMsg{err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()

		resp, err := c.CreateURL(ctx, originalURL)
		if err != nil {
			return createURLMsg{err: err}
		}
		return createURLMsg{resp: resp}
	}
}
