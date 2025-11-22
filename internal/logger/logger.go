package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/mfenderov/claude-sync/internal/ui"
)

// Logger wraps slog with beautiful UI output
type Logger struct {
	*slog.Logger
}

// New creates a new logger with beautiful output
func New(w io.Writer, level slog.Level) *Logger {
	if w == nil {
		w = os.Stdout
	}

	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: level,
	})

	return &Logger{
		Logger: slog.New(handler),
	}
}

// Default creates a logger for CLI output
func Default() *Logger {
	return New(os.Stdout, slog.LevelInfo)
}

// Title prints a styled title
func (l *Logger) Title(title string) {
	fmt.Println()
	fmt.Println(ui.TitleStyle.Render(title))
	fmt.Println()
}

// Success logs a success message with styling
func (l *Logger) Success(icon, message string, args ...any) {
	fmt.Println(ui.RenderSuccess(icon, message))
	if len(args) > 0 {
		l.Info(message, args...)
	}
}

// Error logs an error message with styling
func (l *Logger) Error(icon, message string, err error, args ...any) {
	fmt.Println(ui.RenderError(icon, message))
	combinedArgs := append([]any{"error", err}, args...)
	l.Logger.Error(message, combinedArgs...)
}

// Warning logs a warning message with styling
func (l *Logger) Warning(icon, message string, args ...any) {
	fmt.Println(ui.RenderWarning(icon, message))
	if len(args) > 0 {
		l.Warn(message, args...)
	}
}

// InfoMsg logs an info message with styling
func (l *Logger) InfoMsg(icon, message string, args ...any) {
	fmt.Println(ui.RenderInfo(icon, message))
	if len(args) > 0 {
		l.Info(message, args...)
	}
}

// Muted prints a muted message
func (l *Logger) Muted(message string, args ...any) {
	fmt.Println(ui.RenderMuted(message))
	if len(args) > 0 {
		l.Debug(message, args...)
	}
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
