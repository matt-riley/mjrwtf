package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/matt-riley/mjrwtf/internal/client"
	"github.com/matt-riley/mjrwtf/internal/tui/styles"
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

	deleteLoading            bool
	deleteConfirmShortCode   string
	deleteConfirmOriginalURL string

	mode viewMode

	urls        []tuiURL
	filtered    []tuiURL
	cursor      int
	filterQuery string
	total       int

	// Analytics view state
	analyticsLoading   bool
	analytics          *client.GetAnalyticsResponse
	analyticsShortCode string
	analyticsScroll    int

	analyticsStartInput textinput.Model
	analyticsEndInput   textinput.Model
	analyticsRangeFocus int // 0=start, 1=end

	analyticsStartTime *time.Time
	analyticsEndTime   *time.Time

	width  int
	height int

	pageSize int
	offset   int
}

func newModel(cfg tui_config.Config, warnings []string) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	create := textinput.New()
	create.Placeholder = "https://example.com"
	create.CharLimit = 2048
	create.Width = 80

	start := textinput.New()
	start.Placeholder = "2025-11-20T00:00:00Z"
	start.CharLimit = 64
	start.Width = 32

	end := textinput.New()
	end.Placeholder = "2025-11-22T23:59:59Z"
	end.CharLimit = 64
	end.Width = 32

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

		createInput: create,

		analyticsStartInput: start,
		analyticsEndInput:   end,
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}

		switch m.mode {
		case modeCreating:
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

		case modeAnalyticsTimeRange:
			switch msg.String() {
			case "esc":
				m.mode = modeViewingAnalytics
				m.status = "Time range cancelled"
				return m, nil
			case "tab":
				if m.analyticsRangeFocus == 0 {
					m.analyticsRangeFocus = 1
					m.analyticsStartInput.Blur()
					cmd := m.analyticsEndInput.Focus()
					return m, cmd
				}
				m.analyticsRangeFocus = 0
				m.analyticsEndInput.Blur()
				cmd := m.analyticsStartInput.Focus()
				return m, cmd
			case "enter":
				if m.analyticsRangeFocus == 0 {
					m.analyticsRangeFocus = 1
					m.analyticsStartInput.Blur()
					cmd := m.analyticsEndInput.Focus()
					m.status = "Set time range: end_time"
					return m, cmd
				}

				startStr := strings.TrimSpace(m.analyticsStartInput.Value())
				endStr := strings.TrimSpace(m.analyticsEndInput.Value())
				if (startStr != "" && endStr == "") || (startStr == "" && endStr != "") {
					m.status = "start_time and end_time must be provided together"
					return m, nil
				}

				m.analyticsStartTime = nil
				m.analyticsEndTime = nil
				if startStr != "" {
					st, err := time.Parse(time.RFC3339, startStr)
					if err != nil {
						m.status = "invalid start_time (must be RFC3339)"
						return m, nil
					}
					et, err := time.Parse(time.RFC3339, endStr)
					if err != nil {
						m.status = "invalid end_time (must be RFC3339)"
						return m, nil
					}
					if !st.Before(et) {
						m.status = "end_time must be after start_time"
						return m, nil
					}
					m.analyticsStartTime = &st
					m.analyticsEndTime = &et
				}

				if strings.TrimSpace(m.analyticsShortCode) == "" {
					m.mode = modeBrowsing
					m.status = "No selected URL"
					return m, nil
				}

				m.mode = modeViewingAnalytics
				m.analyticsLoading = true
				m.analytics = nil
				m.analyticsScroll = 0
				m.status = "Loading analytics..."
				return m, tea.Batch(m.spinner.Tick, getAnalyticsCmd(m.cfg, m.analyticsShortCode, m.analyticsStartTime, m.analyticsEndTime))
			default:
				var cmd tea.Cmd
				if m.analyticsRangeFocus == 0 {
					m.analyticsStartInput, cmd = m.analyticsStartInput.Update(msg)
				} else {
					m.analyticsEndInput, cmd = m.analyticsEndInput.Update(msg)
				}
				return m, cmd
			}

		case modeViewingAnalytics:
			switch msg.String() {
			case "b", "esc":
				m.mode = modeBrowsing
				m.status = "Back to list"
				return m, nil
			case "t":
				m.mode = modeAnalyticsTimeRange
				m.analyticsRangeFocus = 0

				if m.analyticsStartTime != nil {
					m.analyticsStartInput.SetValue(m.analyticsStartTime.UTC().Format(time.RFC3339))
				} else {
					m.analyticsStartInput.SetValue("")
				}
				if m.analyticsEndTime != nil {
					m.analyticsEndInput.SetValue(m.analyticsEndTime.UTC().Format(time.RFC3339))
				} else {
					m.analyticsEndInput.SetValue("")
				}

				m.analyticsEndInput.Blur()
				cmd := m.analyticsStartInput.Focus()
				m.status = "Set time range: start_time"
				return m, cmd
			case "r":
				if m.analyticsLoading {
					return m, nil
				}
				m.analyticsLoading = true
				m.analytics = nil
				m.analyticsScroll = 0
				m.status = "Refreshing analytics..."
				return m, tea.Batch(m.spinner.Tick, getAnalyticsCmd(m.cfg, m.analyticsShortCode, m.analyticsStartTime, m.analyticsEndTime))
			case "j", "down":
				lines := m.analyticsLines()
				visible := m.analyticsVisibleLines()
				maxScroll := 0
				if len(lines) > visible {
					maxScroll = len(lines) - visible
				}
				if m.analyticsScroll < maxScroll {
					m.analyticsScroll++
				}
				return m, nil
			case "k", "up":
				if m.analyticsScroll > 0 {
					m.analyticsScroll--
				}
				return m, nil
			}

		case modeDeleteConfirm:
			switch msg.String() {
			case "esc", "n":
				m.mode = modeBrowsing
				m.deleteLoading = false
				m.deleteConfirmShortCode = ""
				m.deleteConfirmOriginalURL = ""
				m.status = "Delete cancelled"
				return m, nil
			case "enter", "y":
				if m.deleteLoading {
					return m, nil
				}
				if strings.TrimSpace(m.deleteConfirmShortCode) == "" {
					m.mode = modeBrowsing
					m.status = "No selected URL"
					return m, nil
				}
				m.deleteLoading = true
				m.status = fmt.Sprintf("Deleting: %s...", m.deleteConfirmShortCode)
				return m, tea.Batch(m.spinner.Tick, deleteURLCmd(m.cfg, m.deleteConfirmShortCode))
			}

		default:
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
			case "a":
				if m.loading {
					return m, nil
				}
				if len(m.filtered) == 0 {
					m.status = "No URLs to show analytics for"
					return m, nil
				}
				if m.cursor < 0 || m.cursor >= len(m.filtered) {
					m.status = "No selected URL"
					return m, nil
				}
				m.mode = modeViewingAnalytics
				m.analyticsLoading = true
				m.analytics = nil
				m.analyticsScroll = 0
				m.analyticsShortCode = m.filtered[m.cursor].ShortCode
				m.analyticsStartTime = nil
				m.analyticsEndTime = nil
				m.status = "Loading analytics..."
				return m, tea.Batch(m.spinner.Tick, getAnalyticsCmd(m.cfg, m.analyticsShortCode, nil, nil))
			case "d":
				if m.loading {
					return m, nil
				}
				if len(m.filtered) == 0 {
					m.status = "No URLs to delete"
					return m, nil
				}
				if m.cursor < 0 || m.cursor >= len(m.filtered) {
					m.status = "No selected URL"
					return m, nil
				}
				u := m.filtered[m.cursor]
				m.mode = modeDeleteConfirm
				m.deleteLoading = false
				m.deleteConfirmShortCode = u.ShortCode
				m.deleteConfirmOriginalURL = u.OriginalURL
				m.status = fmt.Sprintf("Confirm delete: %s", u.ShortCode)
				return m, nil
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
		}

	case spinner.TickMsg:
		if !(m.loading || m.createLoading || m.analyticsLoading || m.deleteLoading) {
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

	case getAnalyticsMsg:
		m.analyticsLoading = false
		if msg.err != nil {
			if apiErr, ok := msg.err.(*client.APIError); ok {
				m.status = fmt.Sprintf("Analytics failed (%d): %s", apiErr.StatusCode, apiErr.Message)
			} else {
				m.status = fmt.Sprintf("Analytics failed: %v", msg.err)
			}
			return m, nil
		}
		m.analytics = msg.resp
		m.status = "Analytics loaded"
		return m, nil

	case deleteURLMsg:
		m.deleteLoading = false
		m.mode = modeBrowsing
		m.deleteConfirmShortCode = ""
		m.deleteConfirmOriginalURL = ""
		if msg.err != nil {
			if apiErr, ok := msg.err.(*client.APIError); ok {
				if apiErr.StatusCode == 404 {
					m.status = fmt.Sprintf("Delete: %s not found (already deleted?)", msg.shortCode)
					m.loading = true
					return m, tea.Batch(m.spinner.Tick, listURLsCmd(m.cfg, m.pageSize, m.offset))
				}
				m.status = fmt.Sprintf("Delete failed (%d): %s", apiErr.StatusCode, apiErr.Message)
			} else {
				m.status = fmt.Sprintf("Delete failed: %v", msg.err)
			}
			return m, nil
		}

		if m.total > 0 {
			m.total--
		}

		remaining := make([]tuiURL, 0, len(m.urls))
		for _, u := range m.urls {
			if u.ShortCode == msg.shortCode {
				continue
			}
			remaining = append(remaining, u)
		}
		m.urls = remaining

		remainingFiltered := make([]tuiURL, 0, len(m.filtered))
		for _, u := range m.filtered {
			if u.ShortCode == msg.shortCode {
				continue
			}
			remainingFiltered = append(remainingFiltered, u)
		}
		m.filtered = remainingFiltered
		if m.cursor >= len(m.filtered) {
			m.cursor = len(m.filtered) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}

		m.status = fmt.Sprintf("Deleted: %s", msg.shortCode)
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
	title := styles.TitleStyle.Render("mjr.wtf TUI")

	baseURL := m.cfg.BaseURL
	if baseURL == "" {
		baseURL = "<empty>"
	}

	baseLabel := styles.MutedStyle.Render("Base URL:")
	baseValue := styles.LinkStyle.Render(baseURL)

	body := strings.Join([]string{
		title,
		"",
		fmt.Sprintf("%s %s", baseLabel, baseValue),
		"",
		m.mainLine(),
		"",
		m.footer(),
	}, "\n")

	return body + "\n"
}

func (m model) mainLine() string {
	switch m.mode {
	case modeCreating:
		return m.createView()
	case modeAnalyticsTimeRange:
		return m.analyticsTimeRangeView()
	case modeViewingAnalytics:
		return m.analyticsView()
	case modeDeleteConfirm:
		return m.deleteConfirmView()
	default:
		if m.loading {
			return fmt.Sprintf("%s Loading URLs...", m.spinner.View())
		}

		if len(m.filtered) == 0 {
			msg := styles.MutedStyle.Render("No URLs yet. Press [c] to create one, or [r] to refresh.")
			return styles.BorderStyle.Padding(1, 2).Render(msg)
		}

		rows := make([][]string, 0, len(m.filtered))
		for _, u := range m.filtered {
			created := ""
			if u.CreatedAt != nil {
				created = u.CreatedAt.Format("2006-01-02 15:04:05")
			}
			rows = append(rows, []string{
				u.ShortCode,
				created,
				fmt.Sprintf("%d", u.ClickCount),
				truncate(u.OriginalURL, 80),
			})
		}

		t := table.New().
			Headers("short_code", "created_at", "click_count", "original_url").
			Rows(rows...).
			Border(lipgloss.RoundedBorder()).
			BorderStyle(styles.BorderStyle).
			Wrap(false).
			StyleFunc(func(row, col int) lipgloss.Style {
				cell := styles.UnselectedRowStyle
				if row == table.HeaderRow {
					cell = styles.TitleStyle
				} else if row == m.cursor {
					cell = styles.SelectedRowStyle
				}
				cell = cell.Padding(0, 1)
				if col == 2 {
					cell = cell.Align(lipgloss.Right)
				}
				return cell
			})
		if m.width > 0 {
			t.Width(m.width)
		}
		return t.Render()
	}
}

func (m model) createView() string {
	inputBox := styles.InputBoxStyle
	if m.createInput.Focused() {
		inputBox = styles.InputBoxFocusedStyle
	}

	lines := []string{
		styles.TitleStyle.Render("Create URL"),
		"",
		styles.MutedStyle.Render("Original URL:"),
		inputBox.Render(m.createInput.View()),
	}
	if m.createLoading {
		loading := fmt.Sprintf("%s Creating...", m.spinner.View())
		lines = append(lines, "", styles.TitleStyle.Copy().Bold(false).Render(loading))
	}

	return styles.PanelStyle.Render(strings.Join(lines, "\n"))
}

func (m model) deleteConfirmView() string {
	shortCode := styles.TitleStyle.Copy().Foreground(styles.Lavender).Render(m.deleteConfirmShortCode)
	originalURL := styles.LinkStyle.Render(truncate(m.deleteConfirmOriginalURL, 120))

	lines := []string{
		styles.WarningStyle.Render("Confirm Delete"),
		"",
		fmt.Sprintf("%s %s", styles.MutedStyle.Render("Short code:"), shortCode),
		fmt.Sprintf("%s %s", styles.MutedStyle.Render("Original URL:"), originalURL),
	}
	if m.deleteLoading {
		loading := fmt.Sprintf("%s Deleting...", m.spinner.View())
		lines = append(lines, "", styles.WarningStyle.Copy().Bold(false).Render(loading))
	}

	return styles.WarningPanelStyle.Render(strings.Join(lines, "\n"))
}

type statusKind int

const (
	statusKindDefault statusKind = iota
	statusKindSuccess
	statusKindError
	statusKindWarning
)

func statusKindFromText(status string) statusKind {
	lower := strings.ToLower(strings.TrimSpace(status))
	if lower == "" {
		return statusKindDefault
	}

	// Prefer errors/warnings first so non-status text (e.g. URLs) can't accidentally override them.
	if strings.HasPrefix(lower, "create failed") || strings.HasPrefix(lower, "delete failed") || strings.HasPrefix(lower, "list failed") || strings.HasPrefix(lower, "analytics failed") || strings.HasPrefix(lower, "failed:") || strings.HasPrefix(lower, "error:") {
		return statusKindError
	}
	if strings.Contains(lower, "not found") {
		return statusKindError
	}
	if strings.HasPrefix(lower, "warn:") || strings.HasPrefix(lower, "warning:") {
		return statusKindWarning
	}

	if strings.HasPrefix(lower, "created:") || strings.HasPrefix(lower, "deleted:") {
		return statusKindSuccess
	}
	if strings.Contains(lower, "success") || strings.Contains(lower, "copied") {
		return statusKindSuccess
	}

	return statusKindDefault
}

func statusStyleForText(status string) lipgloss.Style {
	switch statusKindFromText(status) {
	case statusKindSuccess:
		return styles.SuccessStyle
	case statusKindError:
		return styles.ErrorStyle
	case statusKindWarning:
		return styles.WarningStyle
	default:
		return styles.MutedStyle
	}
}

func (m model) footer() string {
	hintsLine := "[j/k/↑/↓] move  [n/p] page  [/] filter  [c] create  [d] delete  [a] analytics  [r] refresh  [q] quit"
	switch m.mode {
	case modeCreating:
		hintsLine = "[enter] submit  [esc] cancel  [q] quit"
	case modeViewingAnalytics:
		hintsLine = "[j/k/↑/↓] scroll  [t] time range  [r] refresh  [b/esc] back  [q] quit"
	case modeAnalyticsTimeRange:
		hintsLine = "[tab] switch field  [enter] next/apply  [esc] cancel  [q] quit"
	case modeDeleteConfirm:
		hintsLine = "[enter/y] confirm  [esc/n] cancel  [q] quit"
	}

	hints := styles.HintStyle.Render(hintsLine)
	status := m.status
	if m.mode == modeFiltering {
		status = fmt.Sprintf("Filter: %s", m.filterQuery)
	}
	if status == "" {
		status = " "
	}

	statusRendered := statusStyleForText(status).Render(status)
	statusBox := styles.StatusBarStyle.Render(statusRendered)
	return fmt.Sprintf("%s\n%s", hints, statusBox)
}

func (m model) analyticsVisibleLines() int {
	if m.height <= 0 {
		return 20
	}
	// title + base URL + blank lines + footer consumes ~7 lines
	v := m.height - 7
	if v < 8 {
		v = 8
	}
	return v
}

func (m model) analyticsTimeRangeView() string {
	startBox := styles.InputBoxStyle
	startLabel := styles.MutedStyle.Render("start_time:")
	if m.analyticsStartInput.Focused() {
		startBox = styles.InputBoxFocusedStyle
		startLabel = styles.TitleStyle.Copy().Bold(false).Render("start_time:")
	}

	endBox := styles.InputBoxStyle
	endLabel := styles.MutedStyle.Render("end_time:")
	if m.analyticsEndInput.Focused() {
		endBox = styles.InputBoxFocusedStyle
		endLabel = styles.TitleStyle.Copy().Bold(false).Render("end_time:")
	}

	lines := []string{
		styles.TitleStyle.Render("Analytics time range (optional)"),
		"",
		styles.MutedStyle.Render("Enter RFC3339 timestamps. Leave both blank for all-time."),
		"",
		startLabel,
		startBox.Render(m.analyticsStartInput.View()),
		"",
		endLabel,
		endBox.Render(m.analyticsEndInput.View()),
	}
	return styles.PanelStyle.Render(strings.Join(lines, "\n"))
}

func (m model) analyticsView() string {
	if m.analyticsLoading {
		return fmt.Sprintf("%s Loading analytics...", m.spinner.View())
	}
	if m.analytics == nil {
		return lipgloss.NewStyle().Faint(true).Render("(no analytics loaded)")
	}

	lines := m.analyticsLines()
	if len(lines) == 0 {
		return lipgloss.NewStyle().Faint(true).Render("(no analytics)")
	}

	visible := m.analyticsVisibleLines()
	maxScroll := 0
	if len(lines) > visible {
		maxScroll = len(lines) - visible
	}
	scroll := m.analyticsScroll
	if scroll < 0 {
		scroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}

	start := scroll
	end := start + visible
	if end > len(lines) {
		end = len(lines)
	}
	view := lines[start:end]
	if len(lines) > visible {
		indicator := fmt.Sprintf("Scroll: %d-%d of %d", start+1, end, len(lines))
		view = append(view, styles.TitleStyle.Copy().Bold(false).Render(indicator))
	}
	return strings.Join(view, "\n")
}

func (m model) analyticsLines() []string {
	if m.analytics == nil {
		return nil
	}

	shortCode := styles.TitleStyle.Copy().Foreground(styles.Lavender).Bold(false).Render(m.analytics.ShortCode)
	originalURL := styles.LinkStyle.Render(truncate(m.analytics.OriginalURL, 120))
	totalClicks := styles.SuccessStyle.Copy().Bold(true).Render(fmt.Sprintf("%d", m.analytics.TotalClicks))

	rangeLabel := "Time range: all-time"
	if m.analyticsStartTime != nil && m.analyticsEndTime != nil {
		rangeLabel = fmt.Sprintf("Time range: %s → %s", m.analyticsStartTime.UTC().Format(time.RFC3339), m.analyticsEndTime.UTC().Format(time.RFC3339))
	}

	headerLines := []string{
		styles.TitleStyle.Render("Analytics"),
		"",
		fmt.Sprintf("%s %s", styles.MutedStyle.Render("Short code:"), shortCode),
		fmt.Sprintf("%s %s", styles.MutedStyle.Render("Original URL:"), originalURL),
		fmt.Sprintf("%s %s", styles.MutedStyle.Render("Total clicks:"), totalClicks),
		styles.MutedStyle.Render(rangeLabel),
	}
	box := styles.BorderStyle.Copy().BorderForeground(styles.Mauve).Padding(1, 2).Render(strings.Join(headerLines, "\n"))

	lines := splitRenderedLines(box)
	lines = append(lines, "")

	lines = append(lines, formatTopMapSection("By country", m.analytics.ByCountry, 50)...)
	lines = append(lines, "")
	lines = append(lines, formatTopMapSection("By referrer", m.analytics.ByReferrer, 50)...)
	if len(m.analytics.ByDate) > 0 {
		lines = append(lines, "")
		lines = append(lines, formatDateMapSection("By date", m.analytics.ByDate, 60)...)
	}

	return lines
}

type kv struct {
	k string
	v int64
}

func splitRenderedLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func formatTopMapSection(title string, in map[string]int64, maxItems int) []string {
	inner := []string{styles.TitleStyle.Copy().Bold(true).Render(title), ""}
	if len(in) == 0 {
		inner = append(inner, styles.MutedStyle.Render("(none)"))
		return splitRenderedLines(styles.BorderStyle.Copy().Padding(1, 2).Render(strings.Join(inner, "\n")))
	}

	items := make([]kv, 0, len(in))
	for k, v := range in {
		items = append(items, kv{k: k, v: v})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].v == items[j].v {
			return items[i].k < items[j].k
		}
		return items[i].v > items[j].v
	})

	shown := items
	if maxItems > 0 && len(items) > maxItems {
		shown = items[:maxItems]
	}
	for i, it := range shown {
		row := fmt.Sprintf("%-32s %10d", truncate(it.k, 32), it.v)
		rowStyle := lipgloss.NewStyle().Background(styles.Surface0).Foreground(styles.Text).Padding(0, 1)
		if i == 0 {
			rowStyle = lipgloss.NewStyle().Background(styles.Surface1).Foreground(styles.Lavender).Bold(true).Padding(0, 1)
		}
		inner = append(inner, rowStyle.Render(row))
	}
	if len(shown) < len(items) {
		inner = append(inner, styles.MutedStyle.Render(fmt.Sprintf("…and %d more", len(items)-len(shown))))
	}

	return splitRenderedLines(styles.BorderStyle.Copy().Padding(1, 2).Render(strings.Join(inner, "\n")))
}

func formatDateMapSection(title string, in map[string]int64, maxItems int) []string {
	inner := []string{styles.TitleStyle.Copy().Bold(true).Render(title), ""}
	if len(in) == 0 {
		inner = append(inner, styles.MutedStyle.Render("(none)"))
		return splitRenderedLines(styles.BorderStyle.Copy().Padding(1, 2).Render(strings.Join(inner, "\n")))
	}

	keys := make([]string, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	shown := keys
	if maxItems > 0 && len(keys) > maxItems {
		shown = keys[:maxItems]
	}
	for _, k := range shown {
		row := fmt.Sprintf("%s  %10d", k, in[k])
		inner = append(inner, lipgloss.NewStyle().Background(styles.Surface0).Foreground(styles.Text).Padding(0, 1).Render(row))
	}
	if len(shown) < len(keys) {
		inner = append(inner, styles.MutedStyle.Render(fmt.Sprintf("…and %d more", len(keys)-len(shown))))
	}
	return splitRenderedLines(styles.BorderStyle.Copy().Padding(1, 2).Render(strings.Join(inner, "\n")))
}
