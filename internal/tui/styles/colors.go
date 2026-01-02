// Package styles provides centralized styling for the mjr.wtf TUI using the Catppuccin color palette.
// Supports both dark (Mocha) and light (Latte) terminal themes via adaptive colors.
package styles

import "github.com/charmbracelet/lipgloss"

// Catppuccin Color Palette - Adaptive for dark (Mocha) and light (Latte) terminals
// Reference: https://github.com/catppuccin/catppuccin

// Brand/Accent Colors
var (
	// Mauve - Primary brand color
	Mauve = lipgloss.AdaptiveColor{Light: "#8839ef", Dark: "#cba6f7"}

	// Sapphire - Links and URLs
	Sapphire = lipgloss.AdaptiveColor{Light: "#209fb5", Dark: "#74c7ec"}

	// Green - Success states
	Green = lipgloss.AdaptiveColor{Light: "#40a02b", Dark: "#a6e3a1"}

	// Red - Errors
	Red = lipgloss.AdaptiveColor{Light: "#d20f39", Dark: "#f38ba8"}

	// Peach - Warnings
	Peach = lipgloss.AdaptiveColor{Light: "#fe640b", Dark: "#fab387"}

	// Lavender - Highlights
	Lavender = lipgloss.AdaptiveColor{Light: "#7287fd", Dark: "#b4befe"}
)

// Text Colors
var (
	// Text - Primary text
	Text = lipgloss.AdaptiveColor{Light: "#4c4f69", Dark: "#cdd6f4"}

	// Subtext1 - Secondary text
	Subtext1 = lipgloss.AdaptiveColor{Light: "#5c5f77", Dark: "#bac2de"}

	// Subtext0 - Muted text
	Subtext0 = lipgloss.AdaptiveColor{Light: "#6c6f85", Dark: "#a6adc8"}
)

// Surface/Background Colors
var (
	// Base - Main background
	Base = lipgloss.AdaptiveColor{Light: "#eff1f5", Dark: "#1e1e2e"}

	// Surface0 - Elevated surfaces
	Surface0 = lipgloss.AdaptiveColor{Light: "#e6e9ef", Dark: "#313244"}

	// Surface1 - More elevated surfaces
	Surface1 = lipgloss.AdaptiveColor{Light: "#dce0e8", Dark: "#45475a"}

	// Overlay0 - Borders and dividers
	Overlay0 = lipgloss.AdaptiveColor{Light: "#9ca0b0", Dark: "#6c7086"}
)
