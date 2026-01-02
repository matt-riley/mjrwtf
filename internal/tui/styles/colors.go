// Package styles provides centralized styling for the mjr.wtf TUI using the Catppuccin Mocha color palette.
package styles

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha Color Palette
// Reference: https://github.com/catppuccin/catppuccin

// Brand/Accent Colors
var (
	// Mauve - Primary brand color
	Mauve = lipgloss.Color("#cba6f7")
	
	// Sapphire - Links and URLs
	Sapphire = lipgloss.Color("#74c7ec")
	
	// Green - Success states
	Green = lipgloss.Color("#a6e3a1")
	
	// Red - Errors
	Red = lipgloss.Color("#f38ba8")
	
	// Peach - Warnings
	Peach = lipgloss.Color("#fab387")
	
	// Lavender - Highlights
	Lavender = lipgloss.Color("#b4befe")
)

// Text Colors
var (
	// Text - Primary text
	Text = lipgloss.Color("#cdd6f4")
	
	// Subtext1 - Secondary text
	Subtext1 = lipgloss.Color("#bac2de")
	
	// Subtext0 - Muted text
	Subtext0 = lipgloss.Color("#a6adc8")
)

// Surface/Background Colors
var (
	// Base - Main background
	Base = lipgloss.Color("#1e1e2e")
	
	// Surface0 - Elevated surfaces
	Surface0 = lipgloss.Color("#313244")
	
	// Surface1 - More elevated surfaces
	Surface1 = lipgloss.Color("#45475a")
	
	// Overlay0 - Borders and dividers
	Overlay0 = lipgloss.Color("#6c7086")
)
