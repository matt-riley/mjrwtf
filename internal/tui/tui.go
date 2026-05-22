package tui

import (
	"flag"
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"github.com/matt-riley/mjrwtf/internal/tui/tui_config"
)

var runProgram = func(m tea.Model) error {
	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

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
	if err := runProgram(m); err != nil {
		return fmt.Errorf("run tui: %w", err)
	}
	return nil
}
