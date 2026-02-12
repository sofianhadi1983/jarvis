package components

import (
	"strings"

	"jarvis/internal/styles"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const maxInputLines = 10

type InputArea struct {
	textarea    textarea.Model
	width       int
	height      int // current textarea line count
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
		height:      1,
		placeholder: placeholder,
		focused:     true,
	}
}

func (i *InputArea) SetWidth(width int) {
	i.width = width
	i.textarea.SetWidth(width - 4)
	i.recalcHeight()
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
	i.recalcHeight()
	return i, cmd
}

// Height returns the total rendered height of the input area (padding + textarea lines).
func (i *InputArea) Height() int {
	return i.height + 1 // +1 for PaddingTop
}

func (i *InputArea) visualLineCount() int {
	value := i.textarea.Value()
	if value == "" {
		return 1
	}
	usableWidth := i.width - 4
	if usableWidth <= 0 {
		return 1
	}
	lines := strings.Split(value, "\n")
	count := 0
	for _, line := range lines {
		runeLen := len([]rune(line))
		if runeLen == 0 {
			count++
		} else {
			count += (runeLen + usableWidth - 1) / usableWidth
		}
	}
	if count < 1 {
		count = 1
	}
	return count
}

func (i *InputArea) recalcHeight() {
	lines := i.visualLineCount()
	if lines > maxInputLines {
		lines = maxInputLines
	}
	if lines != i.height {
		i.height = lines
		i.textarea.SetHeight(lines)
	}
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
	i.recalcHeight()
}

func (i *InputArea) SetValue(s string) {
	i.textarea.SetValue(s)
	i.recalcHeight()
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
