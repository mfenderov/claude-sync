package prompts

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mfenderov/claude-sync/internal/ui"
)

// ConfirmModel is a model for yes/no confirmation prompts
type ConfirmModel struct {
	prompt   string
	response bool
	done     bool
}

// Init implements tea.Model
func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model
func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
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
func (m ConfirmModel) View() string {
	if m.done {
		return ""
	}
	return ui.InfoStyle.Render(m.prompt) + ui.MutedStyle.Render(" [y/n]: ")
}

// Confirm prompts the user for a yes/no confirmation
func Confirm(prompt string) (bool, error) {
	m := ConfirmModel{prompt: prompt}
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return false, err
	}
	return finalModel.(ConfirmModel).response, nil
}

// InputModel is a model for text input prompts
type InputModel struct {
	textInput textinput.Model
	prompt    string
	done      bool
}

// Init implements tea.Model
func (m InputModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model
func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
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
func (m InputModel) View() string {
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

	m := InputModel{
		textInput: ti,
		prompt:    prompt,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	result := finalModel.(InputModel).textInput.Value()
	return strings.TrimSpace(result), nil
}
