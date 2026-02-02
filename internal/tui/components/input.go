package components

import (
	"chewbacca/internal/tui"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type InputArea struct {
	textarea    textarea.Model
	width       int
	placeholder string
	focused     bool
}

func NewInputArea(placeholder string) *InputArea {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.CharLimit = 4000
	ta.SetHeight(3)
	ta.Focus()

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle()
	ta.BlurredStyle.Base = lipgloss.NewStyle()

	return &InputArea{
		textarea:    ta,
		placeholder: placeholder,
		focused:     true,
	}
}

func (i *InputArea) SetWidth(width int) {
	i.width = width
	i.textarea.SetWidth(width - 4)
}

func (i *InputArea) Focus() tea.Cmd {
	i.focused = true
	return i.textarea.Focus()
}

func (i *InputArea) Blur() {
	i.focused = false
	i.textarea.Blur()
}

func (i *InputArea) Update(msg tea.Msg) (*InputArea, tea.Cmd) {
	var cmd tea.Cmd
	i.textarea, cmd = i.textarea.Update(msg)
	return i, cmd
}

func (i *InputArea) View() string {
	borderStyle := tui.InputBorderStyle.Width(i.width - 2)
	return borderStyle.Render(i.textarea.View())
}

func (i *InputArea) Value() string {
	return i.textarea.Value()
}

func (i *InputArea) Reset() {
	i.textarea.Reset()
}

func (i *InputArea) SetValue(s string) {
	i.textarea.SetValue(s)
}

func (i *InputArea) Focused() bool {
	return i.focused
}
