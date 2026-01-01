package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matt-riley/mjrwtf/internal/tui/tui_config"
)

func TestRun_UsesConfigAndStartsProgram(t *testing.T) {
	old := runProgram
	t.Cleanup(func() { runProgram = old })

	t.Setenv("MJR_BASE_URL", "http://env")
	t.Setenv("MJR_TOKEN", "envtoken")
	t.Setenv("HOME", t.TempDir())

	var got model
	runProgram = func(m tea.Model) error {
		got = m.(model)
		return nil
	}

	if err := Run([]string{}); err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if got.cfg.BaseURL != "http://env" {
		t.Fatalf("BaseURL=%q", got.cfg.BaseURL)
	}
	if got.cfg.Token != "envtoken" {
		t.Fatalf("Token=%q", got.cfg.Token)
	}
}

func TestRun_FlagsOverrideEnv(t *testing.T) {
	old := runProgram
	t.Cleanup(func() { runProgram = old })

	t.Setenv("MJR_BASE_URL", "http://env")
	t.Setenv("MJR_TOKEN", "envtoken")
	t.Setenv("HOME", t.TempDir())

	var got model
	runProgram = func(m tea.Model) error {
		got = m.(model)
		return nil
	}

	if err := Run([]string{"--base-url", "http://flag", "--token", "flagtoken"}); err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if got.cfg.BaseURL != "http://flag" {
		t.Fatalf("BaseURL=%q", got.cfg.BaseURL)
	}
	if got.cfg.Token != "flagtoken" {
		t.Fatalf("Token=%q", got.cfg.Token)
	}
}

func TestNewModel_WarningsBecomeStatus(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example"}, []string{"warning"})
	if m.status != "warning" {
		t.Fatalf("status=%q", m.status)
	}
}

func TestRun_InvalidFlags(t *testing.T) {
	old := runProgram
	t.Cleanup(func() { runProgram = old })
	runProgram = func(m tea.Model) error { return nil }

	if err := Run([]string{"--nope"}); err == nil {
		t.Fatalf("expected error")
	}
}
