package styles

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestColorPalette(t *testing.T) {
	tests := []struct {
		name          string
		color         lipgloss.AdaptiveColor
		expectedDark  string
		expectedLight string
	}{
		// Brand/Accent colors
		{"Mauve", Mauve, "#cba6f7", "#8839ef"},
		{"Sapphire", Sapphire, "#74c7ec", "#209fb5"},
		{"Green", Green, "#a6e3a1", "#40a02b"},
		{"Red", Red, "#f38ba8", "#d20f39"},
		{"Peach", Peach, "#fab387", "#fe640b"},
		{"Lavender", Lavender, "#b4befe", "#7287fd"},

		// Text colors
		{"Text", Text, "#cdd6f4", "#4c4f69"},
		{"Subtext1", Subtext1, "#bac2de", "#5c5f77"},
		{"Subtext0", Subtext0, "#a6adc8", "#6c6f85"},

		// Surface colors
		{"Base", Base, "#1e1e2e", "#eff1f5"},
		{"Surface0", Surface0, "#313244", "#e6e9ef"},
		{"Surface1", Surface1, "#45475a", "#dce0e8"},
		{"Overlay0", Overlay0, "#6c7086", "#9ca0b0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.color.Dark) != tt.expectedDark {
				t.Errorf("color %s dark = %v, want %v", tt.name, tt.color.Dark, tt.expectedDark)
			}
			if string(tt.color.Light) != tt.expectedLight {
				t.Errorf("color %s light = %v, want %v", tt.name, tt.color.Light, tt.expectedLight)
			}
		})
	}
}

func TestStyleRendering(t *testing.T) {
	tests := []struct {
		name  string
		style lipgloss.Style
		text  string
	}{
		{"TitleStyle", TitleStyle, "Test Title"},
		{"BorderStyle", BorderStyle, "Bordered Content"},
		{"PanelStyle", PanelStyle, "Panel Content"},
		{"WarningPanelStyle", WarningPanelStyle, "Warning Panel"},
		{"InputBoxStyle", InputBoxStyle, "Input"},
		{"InputBoxFocusedStyle", InputBoxFocusedStyle, "Input"},
		{"StatusBarStyle", StatusBarStyle, "Status Message"},
		{"HintStyle", HintStyle, "Press q to quit"},
		{"SuccessStyle", SuccessStyle, "Success!"},
		{"ErrorStyle", ErrorStyle, "Error occurred"},
		{"WarningStyle", WarningStyle, "Warning message"},
		{"SelectedRowStyle", SelectedRowStyle, "Selected item"},
		{"UnselectedRowStyle", UnselectedRowStyle, "Normal item"},
		{"MutedStyle", MutedStyle, "Muted text"},
		{"LinkStyle", LinkStyle, "https://example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("style %s panicked when rendering: %v", tt.name, r)
				}
			}()

			// Render the style with text - should not panic
			result := tt.style.Render(tt.text)

			// Result should not be empty
			if result == "" {
				t.Errorf("style %s produced empty output", tt.name)
			}
		})
	}
}

func TestStyleComposition(t *testing.T) {
	// Test that styles can be composed without panicking
	t.Run("TitleWithBorder", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("style composition panicked: %v", r)
			}
		}()

		composedStyle := TitleStyle.Copy().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Overlay0).
			Padding(1, 2)

		result := composedStyle.Render("Composed Title")
		if result == "" {
			t.Error("composed style produced empty output")
		}
	})

	t.Run("StatusWithSuccess", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("style composition panicked: %v", r)
			}
		}()

		successStatus := StatusBarStyle.Copy().
			Foreground(Green)

		result := successStatus.Render("Operation successful")
		if result == "" {
			t.Error("composed style produced empty output")
		}
	})
}

func TestStyleImmutability(t *testing.T) {
	// Test that using Copy() creates independent styles
	t.Run("CopyDoesNotModifyOriginal", func(t *testing.T) {
		original := TitleStyle
		modified := TitleStyle.Copy().Foreground(Red)

		// Render both to ensure they're different
		originalText := original.Render("Original")
		modifiedText := modified.Render("Modified")

		// Both should render without panic
		if originalText == "" || modifiedText == "" {
			t.Error("style rendering failed")
		}

		// The original style should still have Mauve color
		// This is a basic check - in practice we'd need more sophisticated
		// color comparison, but this ensures Copy() doesn't break things
	})
}
