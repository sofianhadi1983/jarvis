package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	headerView := m.header.View()
	chatView := m.chat.View()
	inputView := m.input.View()
	statusView := m.status.View()

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerView,
		chatView,
		inputView,
		statusView,
	)
}
