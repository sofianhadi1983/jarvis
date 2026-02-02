package components

import (
	"fmt"

	"chewbacca/internal/tui"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type StatusBar struct {
	agentName string
	spinner   spinner.Model
	loading   bool
	status    string
	width     int
}

func NewStatusBar(agentName string) *StatusBar {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = tui.SpinnerStyle

	return &StatusBar{
		agentName: agentName,
		spinner:   sp,
		status:    "Ready",
	}
}

func (s *StatusBar) SetWidth(width int) {
	s.width = width
}

func (s *StatusBar) SetLoading(loading bool) {
	s.loading = loading
	if loading {
		s.status = "Thinking..."
	} else {
		s.status = "Ready"
	}
}

func (s *StatusBar) SetStatus(status string) {
	s.status = status
}

func (s *StatusBar) Update(msg tea.Msg) (*StatusBar, tea.Cmd) {
	if s.loading {
		var cmd tea.Cmd
		s.spinner, cmd = s.spinner.Update(msg)
		return s, cmd
	}
	return s, nil
}

func (s *StatusBar) View() string {
	leftSection := s.agentName

	var middleSection string
	if s.loading {
		middleSection = fmt.Sprintf("%s %s", s.spinner.View(), s.status)
	} else {
		middleSection = s.status
	}

	rightSection := "Ctrl+C quit"

	leftStyle := lipgloss.NewStyle().
		Foreground(tui.AssistantColor).
		Bold(true)

	middleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	rightStyle := tui.HelpStyle

	left := leftStyle.Render(leftSection)
	middle := middleStyle.Render(middleSection)
	right := rightStyle.Render(rightSection)

	gap := s.width - lipgloss.Width(left) - lipgloss.Width(middle) - lipgloss.Width(right) - 4
	if gap < 0 {
		gap = 0
	}

	content := fmt.Sprintf(" %s  %s%*s%s ", left, middle, gap, "", right)

	return tui.StatusBarStyle.Width(s.width).Render(content)
}

func (s *StatusBar) SpinnerTick() tea.Cmd {
	return s.spinner.Tick
}
