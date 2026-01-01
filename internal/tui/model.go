package tui

import (
	"fmt"
	"strings"

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

	mode        viewMode
	urls        []tuiURL
	filtered    []tuiURL
	cursor      int
	filterQuery string
	total       int

	pageSize int
	offset   int
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
		mode:     modeBrowsing,
		filtered: []tuiURL{},
		pageSize: 20,
		offset:   0,
	}
	if len(warnings) > 0 {
		m.status = warnings[0]
	}
	return m
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, listURLsCmd(m.cfg, m.pageSize, m.offset))
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
			return m, tea.Batch(m.spinner.Tick, listURLsCmd(m.cfg, m.pageSize, m.offset))
		case "j", "down":
			if m.mode != modeFiltering {
				m.cursorDown()
			}
			return m, nil
		case "k", "up":
			if m.mode != modeFiltering {
				m.cursorUp()
			}
			return m, nil
		case "n":
			return m.nextPage()
		case "p":
			return m.prevPage()
		case "/":
			m.startFilter()
			return m, nil
		case "esc":
			if m.mode == modeFiltering {
				m.cancelFilter()
				return m, nil
			}
		case "enter":
			if m.mode == modeFiltering {
				m.applyFilter()
				m.status = fmt.Sprintf("Filtered to %d/%d", len(m.filtered), len(m.urls))
				return m, nil
			}
		default:
			if m.mode == modeFiltering {
				m.filterInput(msg)
				return m, nil
			}
		}
	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case listURLsMsg:
		m.loading = false
		if msg.err != nil {
			m.status = fmt.Sprintf("List failed: %v", msg.err)
			return m, nil
		}
		m.urls = msg.urls
		m.total = msg.total
		m.applyFilter() // reapply current filter after refresh/page change
		m.status = fmt.Sprintf("Loaded %d/%d", len(m.filtered), m.total)
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

	body := strings.Join([]string{
		title,
		"",
		fmt.Sprintf("Base URL: %s", baseURL),
		"",
		m.mainLine(),
		"",
		m.footer(),
	}, "\n")

	return body + "\n"
}

func (m model) mainLine() string {
	if m.loading {
		return fmt.Sprintf("%s Loading URLs...", m.spinner.View())
	}

	head := lipgloss.NewStyle().Bold(true).Render("short_code  created_at            click_count  original_url")
	lines := []string{head}
	if len(m.filtered) == 0 {
		lines = append(lines, lipgloss.NewStyle().Faint(true).Render("(no URLs)"))
		return strings.Join(lines, "\n")
	}

	for i, u := range m.filtered {
		prefix := "  "
		if i == m.cursor {
			prefix = "> "
		}
		created := ""
		if u.CreatedAt != nil {
			created = u.CreatedAt.Format("2006-01-02 15:04:05")
		}
		line := fmt.Sprintf("%s%-10s  %-19s  %10d  %s", prefix, u.ShortCode, created, u.ClickCount, truncate(u.OriginalURL, 80))
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m model) footer() string {
	hints := lipgloss.NewStyle().Faint(true).Render("[j/k/↑/↓] move  [n/p] page  [/] filter  [r] refresh  [q] quit")
	status := m.status
	if m.mode == modeFiltering {
		status = fmt.Sprintf("Filter: %s", m.filterQuery)
	}
	if status == "" {
		status = " "
	}
	return fmt.Sprintf("%s\n%s", hints, status)
}
