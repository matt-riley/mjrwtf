package tui

import (
	"flag"
	"fmt"
	"io"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matt-riley/mjrwtf/internal/tui/tui_config"
)

func Run(args []string) error {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	flagBaseURL := fs.String("base-url", "", "API base URL")
	flagToken := fs.String("token", "", "API bearer token")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	cfg, warnings, err := tui_config.Load(tui_config.LoadOptions{
		FlagBaseURL: *flagBaseURL,
		FlagToken:   *flagToken,
	})
	if err != nil {
		return err
	}

	m := newModel(cfg, warnings)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	if err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}

func stderrf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}
