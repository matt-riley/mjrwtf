package tui_config

import "strings"

func MaskToken(token string) string {
	t := strings.TrimSpace(token)
	if t == "" {
		return "<empty>"
	}
	if len(t) <= 4 {
		return "****"
	}
	return t[:2] + strings.Repeat("*", len(t)-4) + t[len(t)-2:]
}
