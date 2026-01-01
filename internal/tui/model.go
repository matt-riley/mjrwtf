package tui

import (
	"fmt"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/matt-riley/mjrwtf/internal/client"
	"github.com/matt-riley/mjrwtf/internal/tui/tui_config"
)

var clipboardWriteAll = clipboard.WriteAll

type model struct {
	cfg      tui_config.Config
	warnings []string

	spinner spinner.Model
	loading bool
	status  string

	createInput   textinput.Model
	createLoading bool

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

	ti := textinput.New()
	ti.Placeholder = "https://example.com"
	ti.CharLimit = 2048
	ti.Width = 80

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

		createInput: ti,
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
		}

		if m.mode == modeCreating {
			switch msg.String() {
			case "esc":
				m.mode = modeBrowsing
				m.createLoading = false
				m.status = "Create cancelled"
				return m, nil
			case "enter":
				if m.createLoading {
					return m, nil
				}
				original := strings.TrimSpace(m.createInput.Value())
				if err := validateHTTPURL(original); err != nil {
					m.status = err.Error()
					return m, nil
				}
				m.createLoading = true
				m.status = "Creating..."
				return m, tea.Batch(m.spinner.Tick, createURLCmd(m.cfg, original))
			default:
				var cmd tea.Cmd
				m.createInput, cmd = m.createInput.Update(msg)
				return m, cmd
			}
		}

		switch msg.String() {
		case "r":
			m.loading = true
			m.status = "Refreshing..."
			return m, tea.Batch(m.spinner.Tick, listURLsCmd(m.cfg, m.pageSize, m.offset))
		case "c":
			m.mode = modeCreating
			m.createLoading = false
			m.createInput.SetValue("")
			cmd := m.createInput.Focus()
			m.status = "Create: enter original URL"
			return m, cmd
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
		if !(m.loading || m.createLoading) {
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
	case createURLMsg:
		m.createLoading = false
		if msg.err != nil {
			if apiErr, ok := msg.err.(*client.APIError); ok {
				m.status = fmt.Sprintf("Create failed (%d): %s", apiErr.StatusCode, apiErr.Message)
			} else {
				m.status = fmt.Sprintf("Create failed: %v", msg.err)
			}
			m.mode = modeBrowsing
			m.createInput.SetValue("")
			return m, nil
		}
		if msg.resp == nil {
			m.status = "Create failed: empty response"
			m.mode = modeBrowsing
			m.createInput.SetValue("")
			return m, nil
		}

		m.mode = modeBrowsing
		m.createInput.SetValue("")
		if err := clipboardWriteAll(msg.resp.ShortURL); err != nil {
			m.status = fmt.Sprintf("Created: %s (copy failed: %v)", msg.resp.ShortURL, err)
		} else {
			m.status = fmt.Sprintf("Created: %s (copied to clipboard)", msg.resp.ShortURL)
		}
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, listURLsCmd(m.cfg, m.pageSize, m.offset))
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
	if m.mode == modeCreating {
		return m.createView()
	}
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

func (m model) createView() string {
	lines := []string{
		lipgloss.NewStyle().Bold(true).Render("Create URL"),
		"",
		"Original URL:",
		m.createInput.View(),
	}
	if m.createLoading {
		lines = append(lines, "", fmt.Sprintf("%s Creating...", m.spinner.View()))
	}
	return strings.Join(lines, "\n")
}

func (m model) footer() string {
	hintsLine := "[j/k/↑/↓] move  [n/p] page  [/] filter  [c] create  [r] refresh  [q] quit"
	if m.mode == modeCreating {
		hintsLine = "[enter] submit  [esc] cancel  [q] quit"
	}

	hints := lipgloss.NewStyle().Faint(true).Render(hintsLine)
	status := m.status
	if m.mode == modeFiltering {
		status = fmt.Sprintf("Filter: %s", m.filterQuery)
	}
	if status == "" {
		status = " "
	}
	return fmt.Sprintf("%s\n%s", hints, status)
}
