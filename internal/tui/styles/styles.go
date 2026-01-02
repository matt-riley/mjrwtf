package styles

import "github.com/charmbracelet/lipgloss"

// Base Styles using Catppuccin adaptive color palette (Mocha for dark, Latte for light terminals)

// TitleStyle - Bold text with Mauve accent for titles and headers
var TitleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(Mauve)

// BorderStyle - Standard border using Overlay0 color
var BorderStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(Overlay0)

// StatusBarStyle - Status bar with Surface0 background
var StatusBarStyle = lipgloss.NewStyle().
	Background(Surface0).
	Padding(0, 1)

// HintStyle - Muted text for keyboard hints and help text
var HintStyle = lipgloss.NewStyle().
	Foreground(Subtext0)

// SuccessStyle - Green foreground for success messages
var SuccessStyle = lipgloss.NewStyle().
	Foreground(Green).
	Bold(true)

// ErrorStyle - Red foreground for error messages
var ErrorStyle = lipgloss.NewStyle().
	Foreground(Red).
	Bold(true)

// WarningStyle - Peach foreground for warnings
var WarningStyle = lipgloss.NewStyle().
	Foreground(Peach).
	Bold(true)

// SelectedRowStyle - Highlighted row with Surface1 background and Lavender accent
var SelectedRowStyle = lipgloss.NewStyle().
	Background(Surface1).
	Foreground(Lavender)

// UnselectedRowStyle - Base background for unselected rows
var UnselectedRowStyle = lipgloss.NewStyle().
	Background(Base).
	Foreground(Text)

// MutedStyle - Subtext0 for muted/secondary information
var MutedStyle = lipgloss.NewStyle().
	Foreground(Subtext0)

// LinkStyle - Sapphire color for URLs and links
var LinkStyle = lipgloss.NewStyle().
	Foreground(Sapphire)
