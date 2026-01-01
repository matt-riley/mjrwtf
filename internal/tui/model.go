package tui

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/matt-riley/mjrwtf/internal/tui/tui_config"
)

type model struct {
	cfg      tui_config.Config
	warnings []string

	spinner spinner.Model
	loading bool
	status  string
}

type healthMsg struct {
	detail string
	err    error
}

func newModel(cfg tui_config.Config, warnings []string) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	m := model{
		cfg:      cfg,
		warnings: warnings,
		spinner:  sp,
		loading:  true,
		status:   "Starting...",
	}
	if len(warnings) > 0 {
		m.status = warnings[0]
	}
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, healthCmd(m.cfg.BaseURL))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.loading = true
			m.status = "Refreshing..."
			return m, tea.Batch(m.spinner.Tick, healthCmd(m.cfg.BaseURL))
		}
	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case healthMsg:
		m.loading = false
		if msg.err != nil {
			m.status = fmt.Sprintf("Health check failed: %v", msg.err)
			return m, nil
		}
		m.status = msg.detail
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	title := lipgloss.NewStyle().Bold(true).Render("mjr.wtf TUI")

	baseURL := m.cfg.BaseURL
	if baseURL == "" {
		baseURL = "<empty>"
	}

	token := tui_config.MaskToken(m.cfg.Token)

	body := strings.Join([]string{
		title,
		"",
		fmt.Sprintf("Base URL: %s", baseURL),
		fmt.Sprintf("Token:    %s", token),
		"",
		m.mainLine(),
		"",
		m.footer(),
	}, "\n")

	return body + "\n"
}

func (m model) mainLine() string {
	if m.loading {
		baseURL := strings.TrimSpace(m.cfg.BaseURL)
		if baseURL == "" {
			return fmt.Sprintf("%s Checking health endpoint (base URL not set)", m.spinner.View())
		}
		return fmt.Sprintf("%s Checking %s/health", m.spinner.View(), baseURL)
	}
	return "Idle"
}

func (m model) footer() string {
	hints := lipgloss.NewStyle().Faint(true).Render("[r] refresh  [q] quit")
	status := m.status
	if status == "" {
		status = " "
	}
	return fmt.Sprintf("%s\n%s", hints, status)
}

func healthCmd(baseURL string) tea.Cmd {
	return func() tea.Msg {
		if baseURL == "" {
			return healthMsg{err: fmt.Errorf("base URL not set")}
		}

		c := http.Client{Timeout: 3 * time.Second}
		req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/health", nil)
		if err != nil {
			return healthMsg{err: err}
		}

		resp, err := c.Do(req)
		if err != nil {
			return healthMsg{err: err}
		}
		defer resp.Body.Close()

		if resp.StatusCode/100 != 2 {
			return healthMsg{err: fmt.Errorf("unexpected status: %s", resp.Status)}
		}

		return healthMsg{detail: "Health OK"}
	}
}
