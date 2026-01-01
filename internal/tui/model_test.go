package tui

import (
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

func TestModel_Update_HealthMsgSetsStatus(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "abcdef"}, nil)
	m.loading = true

	m2, _ := m.Update(healthMsg{detail: "Health OK"})
	mm := m2.(model)
	if mm.loading {
		t.Fatalf("expected loading=false")
	}
	if mm.status != "Health OK" {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestModel_View_MasksToken(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "abcdef"}, nil)
	out := m.View()
	if !strings.Contains(out, "Token:") {
		t.Fatalf("expected token line")
	}
	if strings.Contains(out, "abcdef") {
		t.Fatalf("expected token to be masked")
	}
	if !strings.Contains(out, "ab**ef") {
		t.Fatalf("expected masked token, got: %q", out)
	}
}

func TestHealthCmd(t *testing.T) {
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(okSrv.Close)

	msg := healthCmd(okSrv.URL)().(healthMsg)
	if msg.err != nil {
		t.Fatalf("expected nil err, got %v", msg.err)
	}
	if msg.detail != "Health OK" {
		t.Fatalf("detail=%q", msg.detail)
	}

	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(errSrv.Close)

	msg = healthCmd(errSrv.URL)().(healthMsg)
	if msg.err == nil {
		t.Fatalf("expected err")
	}

	msg = healthCmd("")().(healthMsg)
	if msg.err == nil {
		t.Fatalf("expected err for empty baseURL")
	}
}
