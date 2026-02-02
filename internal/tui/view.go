package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	chatView := m.chat.View()
	inputView := m.input.View()
	statusView := m.status.View()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		chatView,
		inputView,
		statusView,
	)
}
