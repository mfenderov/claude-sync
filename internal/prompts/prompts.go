// Package prompts provides interactive terminal prompts for user input.
// It includes confirmation dialogs and text input fields using the
// Bubble Tea framework for a beautiful TUI experience.
package prompts

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mfenderov/claude-sync/internal/ui"
)

// confirmModel is a model for yes/no confirmation prompts
type confirmModel struct {
	prompt   string
	response bool
	done     bool
}

// Init implements tea.Model
func (m confirmModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "y", "Y":
			m.response = true
			m.done = true
			return m, tea.Quit
		case "n", "N":
			m.response = false
			m.done = true
			return m, tea.Quit
		case "ctrl+c", "esc":
			m.response = false
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model
func (m confirmModel) View() string {
	if m.done {
		return ""
	}
	return ui.InfoStyle.Render(m.prompt) + ui.MutedStyle.Render(" [y/n]: ")
}

// Confirm prompts the user for a yes/no confirmation
func Confirm(prompt string) (bool, error) {
	m := confirmModel{prompt: prompt}
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}

	finalConfirm, ok := finalModel.(confirmModel)
	if !ok {
		return false, fmt.Errorf("unexpected model type: %T", finalModel)
	}
	return finalConfirm.response, nil
}

// inputModel is a model for text input prompts
//
//nolint:govet // fieldalignment: struct contains large external library type
type inputModel struct {
	textInput textinput.Model
	prompt    string
	done      bool
}

// Init implements tea.Model
func (m inputModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model
func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyEnter:
			m.done = true
			return m, tea.Quit
		case tea.KeyCtrlC, tea.KeyEsc:
			m.done = true
			m.textInput.SetValue("")
			return m, tea.Quit
		}
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

// View implements tea.Model
func (m inputModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf(
		"%s\n%s\n%s",
		ui.InfoStyle.Render(m.prompt),
		m.textInput.View(),
		ui.MutedStyle.Render("(press Enter to confirm, Esc to cancel)"),
	)
}

// Input prompts the user for text input
func Input(prompt, placeholder string) (string, error) {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	ti.CharLimit = 200
	ti.Width = 60

	m := inputModel{
		textInput: ti,
		prompt:    prompt,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	finalInput, ok := finalModel.(inputModel)
	if !ok {
		return "", fmt.Errorf("unexpected model type: %T", finalModel)
	}
	result := finalInput.textInput.Value()
	return strings.TrimSpace(result), nil
}
