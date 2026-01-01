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

type viewMode int

const (
	modeBrowsing viewMode = iota
	modeFiltering
)

type tuiURL struct {
	ShortCode   string
	OriginalURL string
	CreatedAt   *time.Time
	ClickCount  int64
}

type listURLsMsg struct {
	urls  []tuiURL
	total int
	err   error
}

func listURLsCmd(cfg tui_config.Config, limit, offset int) tea.Cmd {
	return func() tea.Msg {
		base := strings.TrimSpace(cfg.BaseURL)
		if base == "" {
			return listURLsMsg{err: fmt.Errorf("base URL not set")}
		}

		c, err := client.New(base, client.WithToken(cfg.Token), client.WithTimeout(5*time.Second))
		if err != nil {
			return listURLsMsg{err: err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		defer cancel()
		resp, err := c.ListURLs(ctx, limit, offset)
		if err != nil {
			return listURLsMsg{err: err}
		}

		out := make([]tuiURL, 0, len(resp.URLs))
		for _, u := range resp.URLs {
			created := u.CreatedAt
			out = append(out, tuiURL{
				ShortCode:   u.ShortCode,
				OriginalURL: u.OriginalURL,
				CreatedAt:   &created,
				ClickCount:  u.ClickCount,
			})
		}

		total := resp.Total
		if total == 0 {
			// If the server doesn’t return total, still behave sensibly.
			total = offset + len(out)
		}
		return listURLsMsg{urls: out, total: total}
	}
}

func truncate(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	if max <= 1 {
		return s[:1]
	}
	return s[:max-1] + "…"
}
