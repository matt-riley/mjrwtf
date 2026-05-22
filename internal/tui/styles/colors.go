// Package styles provides centralized styling for the mjr.wtf TUI using the Catppuccin color palette.
// Supports both dark (Mocha) and light (Latte) terminal themes via adaptive colors.
package styles

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
)

// Catppuccin Color Palette - Adaptive for dark (Mocha) and light (Latte) terminals
// Reference: https://github.com/catppuccin/catppuccin

// Brand/Accent Colors
var (
	// Mauve - Primary brand color
	Mauve = compat.AdaptiveColor{Light: lipgloss.Color("#8839ef"), Dark: lipgloss.Color("#cba6f7")}

	// Sapphire - Links and URLs
	Sapphire = compat.AdaptiveColor{Light: lipgloss.Color("#209fb5"), Dark: lipgloss.Color("#74c7ec")}

	// Green - Success states
	Green = compat.AdaptiveColor{Light: lipgloss.Color("#40a02b"), Dark: lipgloss.Color("#a6e3a1")}

	// Red - Errors
	Red = compat.AdaptiveColor{Light: lipgloss.Color("#d20f39"), Dark: lipgloss.Color("#f38ba8")}

	// Peach - Warnings
	Peach = compat.AdaptiveColor{Light: lipgloss.Color("#fe640b"), Dark: lipgloss.Color("#fab387")}

	// Lavender - Highlights
	Lavender = compat.AdaptiveColor{Light: lipgloss.Color("#7287fd"), Dark: lipgloss.Color("#b4befe")}
)

// Text Colors
var (
	// Text - Primary text
	Text = compat.AdaptiveColor{Light: lipgloss.Color("#4c4f69"), Dark: lipgloss.Color("#cdd6f4")}

	// Subtext1 - Secondary text
	Subtext1 = compat.AdaptiveColor{Light: lipgloss.Color("#5c5f77"), Dark: lipgloss.Color("#bac2de")}

	// Subtext0 - Muted text
	Subtext0 = compat.AdaptiveColor{Light: lipgloss.Color("#6c6f85"), Dark: lipgloss.Color("#a6adc8")}
)

// Surface/Background Colors
var (
	// Base - Main background
	Base = compat.AdaptiveColor{Light: lipgloss.Color("#eff1f5"), Dark: lipgloss.Color("#1e1e2e")}

	// Surface0 - Elevated surfaces
	Surface0 = compat.AdaptiveColor{Light: lipgloss.Color("#e6e9ef"), Dark: lipgloss.Color("#313244")}

	// Surface1 - More elevated surfaces
	Surface1 = compat.AdaptiveColor{Light: lipgloss.Color("#dce0e8"), Dark: lipgloss.Color("#45475a")}

	// Overlay0 - Borders and dividers
	Overlay0 = compat.AdaptiveColor{Light: lipgloss.Color("#9ca0b0"), Dark: lipgloss.Color("#6c7086")}
)
