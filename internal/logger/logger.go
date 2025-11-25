// Package logger provides beautiful terminal output using Lipgloss styling.
//
// This package provides a clean TUI experience for CLI applications,
// with styled messages, icons, and boxes.
package logger

import (
	"fmt"

	"github.com/mfenderov/claude-sync/internal/ui"
)

// Logger provides beautiful TUI output
type Logger struct{}

// New creates a new logger
func New() *Logger {
	return &Logger{}
}

// Default creates a logger for CLI output
func Default() *Logger {
	return New()
}

// Title prints a styled title
func (l *Logger) Title(title string) {
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(title))
	fmt.Println()
}

// Success prints a success message with styling
func (l *Logger) Success(icon, message string, _ ...any) {
	fmt.Println(ui.RenderSuccess(icon, message))
}

// Error prints an error message with styling
func (l *Logger) Error(icon, message string, _ error, _ ...any) {
	fmt.Println(ui.RenderError(icon, message))
}

// Warning prints a warning message with styling
func (l *Logger) Warning(icon, message string, _ ...any) {
	fmt.Println(ui.RenderWarning(icon, message))
}

// InfoMsg prints an info message with styling
func (l *Logger) InfoMsg(icon, message string, _ ...any) {
	fmt.Println(ui.RenderInfo(icon, message))
}

// Muted prints a muted message
func (l *Logger) Muted(message string, _ ...any) {
	fmt.Println(ui.RenderMuted(message))
}

// ListItem prints a styled list item
func (l *Logger) ListItem(message string) {
	fmt.Println(ui.ListItemStyle.Render(message))
}

// Box prints content in a styled box
func (l *Logger) Box(title, content string) {
	fmt.Println(ui.RenderBox(title, content))
}

// Newline prints a newline
func (l *Logger) Newline() {
	fmt.Println()
}
