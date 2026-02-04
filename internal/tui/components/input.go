package components

import (
	"jarvis/internal/styles"

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
	ta.SetHeight(1)
	ta.Focus()

	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.FocusedStyle.Base = lipgloss.NewStyle()
	ta.BlurredStyle.Base = lipgloss.NewStyle()
	ta.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(styles.DimColor)
	ta.BlurredStyle.Placeholder = lipgloss.NewStyle().Foreground(styles.DimColor)

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
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		Bold(true)

	prompt := promptStyle.Render("> ")
	content := i.textarea.View()

	inputLine := lipgloss.JoinHorizontal(lipgloss.Left, prompt, content)

	return lipgloss.NewStyle().
		PaddingLeft(0).
		PaddingTop(1).
		PaddingBottom(0).
		Render(inputLine)
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

func (i *InputArea) CursorPosition() int {
	return len(i.textarea.Value())
}

func (i *InputArea) InsertCompletion(startPos int, newText string) {
	value := i.textarea.Value()
	if startPos < 0 {
		startPos = 0
	}
	if startPos > len(value) {
		startPos = len(value)
	}
	newValue := value[:startPos] + newText
	i.textarea.SetValue(newValue)
}

func (i *InputArea) GetTextAfter(pos int) string {
	value := i.textarea.Value()
	if pos < 0 || pos >= len(value) {
		return ""
	}
	return value[pos:]
}
