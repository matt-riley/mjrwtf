package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) cursorDown() {
	if len(m.filtered) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor < len(m.filtered)-1 {
		m.cursor++
	}
}

func (m *model) cursorUp() {
	if len(m.filtered) == 0 {
		m.cursor = 0
		return
	}
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m model) nextPage() (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	if len(m.urls) == 0 {
		return m, nil
	}
	if m.offset+len(m.urls) >= m.total {
		m.status = "Already on last page"
		return m, nil
	}
	m.offset += m.pageSize
	m.loading = true
	m.status = "Loading next page..."
	m.cursor = 0
	return m, tea.Batch(m.spinner.Tick, listURLsCmd(m.cfg, m.pageSize, m.offset))
}

func (m model) prevPage() (tea.Model, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	if m.offset <= 0 {
		m.status = "Already on first page"
		return m, nil
	}
	m.offset -= m.pageSize
	if m.offset < 0 {
		m.offset = 0
	}
	m.loading = true
	m.status = "Loading previous page..."
	m.cursor = 0
	return m, tea.Batch(m.spinner.Tick, listURLsCmd(m.cfg, m.pageSize, m.offset))
}

func (m *model) startFilter() {
	m.mode = modeFiltering
	m.filterQuery = ""
}

func (m *model) cancelFilter() {
	m.mode = modeBrowsing
	m.filterQuery = ""
	m.applyFilter()
	m.status = "Filter cleared"
}

func (m *model) filterInput(k tea.KeyMsg) {
	s := k.String()
	switch s {
	case "backspace":
		if len(m.filterQuery) > 0 {
			m.filterQuery = m.filterQuery[:len(m.filterQuery)-1]
		}
	default:
		if len(s) == 1 {
			m.filterQuery += s
		}
	}
}

func (m *model) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.filterQuery))
	if q == "" {
		m.filtered = append([]tuiURL(nil), m.urls...)
		m.cursor = 0
		m.mode = modeBrowsing
		return
	}

	filtered := make([]tuiURL, 0, len(m.urls))
	for _, u := range m.urls {
		if strings.Contains(strings.ToLower(u.ShortCode), q) || strings.Contains(strings.ToLower(u.OriginalURL), q) {
			filtered = append(filtered, u)
		}
	}
	m.filtered = filtered
	m.cursor = 0
	m.mode = modeBrowsing
}
