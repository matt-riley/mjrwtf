package tui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matt-riley/mjrwtf/internal/tui/tui_config"
)

func TestModel_Update_Quit(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "abcdef"}, nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatalf("expected quit cmd")
	}
}

func TestModel_Update_ListURLsMsgSetsStatus(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "abcdef"}, nil)
	m.loading = true

	m2, _ := m.Update(listURLsMsg{urls: []tuiURL{}, total: 0})
	mm := m2.(model)
	if mm.loading {
		t.Fatalf("expected loading=false")
	}
	if !strings.Contains(mm.status, "Loaded") {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestModel_View_ShowsBaseURL(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "abcdef"}, nil)
	out := m.View()
	if !strings.Contains(out, "Base URL: http://example") {
		t.Fatalf("expected base URL line")
	}
}

func TestModel_Update_CreateMode_Cancel(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "abcdef"}, nil)
	m.loading = false

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	mm := m2.(model)
	if mm.mode != modeCreating {
		t.Fatalf("mode=%v", mm.mode)
	}

	m3, _ := mm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm2 := m3.(model)
	if mm2.mode != modeBrowsing {
		t.Fatalf("mode=%v", mm2.mode)
	}
}

func TestModel_Update_CreateMode_InvalidURL(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "abcdef"}, nil)
	m.loading = false

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	mm := m2.(model)
	mm.createInput.SetValue("ftp://example.com")

	m3, _ := mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm2 := m3.(model)
	if !strings.Contains(mm2.status, "http") {
		t.Fatalf("status=%q", mm2.status)
	}
	if mm2.createLoading {
		t.Fatalf("expected createLoading=false")
	}
}

func TestCreateURLCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/api/urls" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), `"original_url":"https://example.com"`) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"short_code":"abc123","short_url":"https://mjr.wtf/abc123","original_url":"https://example.com"}`))
	}))
	t.Cleanup(srv.Close)

	msg := createURLCmd(tui_config.Config{BaseURL: srv.URL, Token: "t"}, "https://example.com")().(createURLMsg)
	if msg.err != nil {
		t.Fatalf("expected nil err, got %v", msg.err)
	}
	if msg.resp == nil {
		t.Fatalf("expected resp")
	}
	if msg.resp.ShortCode != "abc123" {
		t.Fatalf("ShortCode=%q", msg.resp.ShortCode)
	}
}

func TestListURLsCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/urls" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("limit") != "20" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("offset") != "40" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"urls":[{"id":1,"short_code":"abc123","original_url":"https://example.com","created_at":"2026-01-01T00:00:00Z","created_by":"me","click_count":2}],"total":1,"limit":20,"offset":40}`))
	}))
	t.Cleanup(srv.Close)

	msg := listURLsCmd(tui_config.Config{BaseURL: srv.URL, Token: "t"}, 20, 40)().(listURLsMsg)
	if msg.err != nil {
		t.Fatalf("expected nil err, got %v", msg.err)
	}
	if msg.total != 1 {
		t.Fatalf("total=%d", msg.total)
	}
	if len(msg.urls) != 1 || msg.urls[0].ShortCode != "abc123" {
		t.Fatalf("urls=%v", msg.urls)
	}
	if msg.urls[0].CreatedAt == nil {
		t.Fatalf("expected created_at")
	}

	msg = listURLsCmd(tui_config.Config{BaseURL: ""}, 20, 0)().(listURLsMsg)
	if msg.err == nil {
		t.Fatalf("expected err for empty baseURL")
	}
}
