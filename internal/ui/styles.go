// Package ui provides terminal styling and rendering using Lipgloss.
//
// This package defines the color palette, styles, and helper functions for
// creating beautiful terminal output with consistent styling throughout the
// claude-sync CLI.
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette
var (
	Primary = lipgloss.Color("#7C3AED") // Purple
	Success = lipgloss.Color("#10B981") // Green
	Warning = lipgloss.Color("#F59E0B") // Amber
	Error   = lipgloss.Color("#EF4444") // Red
	Muted   = lipgloss.Color("#6B7280") // Gray
	Info    = lipgloss.Color("#3B82F6") // Blue
)

// Styles
var (
	TitleStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			Padding(0, 1)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2).
			MarginTop(1).
			MarginBottom(1)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(Error).
			Bold(true)

	WarningStyle = lipgloss.NewStyle().
			Foreground(Warning).
			Bold(true)

	InfoStyle = lipgloss.NewStyle().
			Foreground(Info)

	MutedStyle = lipgloss.NewStyle().
			Foreground(Muted)

	PrimaryStyle = lipgloss.NewStyle().
			Foreground(Primary)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			Underline(true).
			MarginTop(1).
			MarginBottom(1)

	ListItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)

	SectionStyle = lipgloss.NewStyle().
			MarginTop(1).
			MarginBottom(1)
)

// RenderBox renders content inside a styled box with a title
func RenderBox(title, content string) string {
	titleBar := TitleStyle.Render("  " + title + "  ")
	return BoxStyle.Render(titleBar + "\n\n" + content)
}

// RenderSuccess renders a success message with an icon
func RenderSuccess(icon, message string) string {
	return SuccessStyle.Render(icon+" ") + message
}

// RenderError renders an error message with an icon
func RenderError(icon, message string) string {
	return ErrorStyle.Render(icon+" ") + message
}

// RenderWarning renders a warning message with an icon
func RenderWarning(icon, message string) string {
	return WarningStyle.Render(icon+" ") + message
}

// RenderInfo renders an info message with an icon
func RenderInfo(icon, message string) string {
	return InfoStyle.Render(icon+" ") + message
}

// RenderMuted renders text in a muted style
func RenderMuted(text string) string {
	return MutedStyle.Render(text)
}
