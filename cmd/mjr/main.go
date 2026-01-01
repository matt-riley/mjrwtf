package main

import (
	"fmt"
	"os"

	"github.com/matt-riley/mjrwtf/internal/tui"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage(1)
		return
	}

	switch args[0] {
	case "tui":
		if err := tui.Run(args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		usage(0)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		usage(1)
	}
}

func usage(exitCode int) {
	fmt.Fprint(os.Stderr, `Usage:
  mjr tui [--base-url URL] [--token TOKEN]

Commands:
  tui    Launch the interactive terminal UI
`)
	os.Exit(exitCode)
}
