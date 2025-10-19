package logo

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func Render(width int) string {
	// Simple ASCII art for KILO
	logo := []string{
		"█▄▀ █ █   ▄▀▄",
		"█ █ █ █▄▄ ▀▄▀",
	}

	hotPink := lipgloss.Color("#FF10F0")
	cyan := lipgloss.Color("#00FFFF")
	purple := lipgloss.Color("#B026FF")

	// Apply gradient styling
	var styledLines []string
	for i, line := range logo {
		var style lipgloss.Style
		if i == 0 {
			style = lipgloss.NewStyle().
				Foreground(hotPink).
				Bold(true)
		} else {
			style = lipgloss.NewStyle().
				Foreground(cyan).
				Bold(true)
		}
		styledLines = append(styledLines, style.Render(line))
	}

	// Add decorative lines with purple
	lineStyle := lipgloss.NewStyle().Foreground(purple)
	decorLine := lineStyle.Render(strings.Repeat("▬", lipgloss.Width(logo[0])))

	// Combine everything
	result := []string{
		decorLine,
		styledLines[0],
		styledLines[1],
		decorLine,
	}

	return strings.Join(result, "\n")
}

// RenderWithTagline renders the logo with a tagline underneath
func RenderWithTagline(tagline string) string {
	logo := Render(0)

	// Neon orange for the tagline
	taglineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF6D00")).
		Italic(true).
		Bold(true)

	styledTagline := taglineStyle.Render("  " + tagline)

	return logo + "\n" + styledTagline
}
