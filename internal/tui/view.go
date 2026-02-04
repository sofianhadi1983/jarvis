package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.showLogin {
		return m.loginModal.View()
	}

	headerView := m.header.View()
	chatView := m.chat.View()
	imageIndicatorView := m.imageIndicator.View()
	inputView := m.input.View()
	statusView := m.status.View()

	// Build view with optional image indicator
	if imageIndicatorView != "" {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			headerView,
			chatView,
			imageIndicatorView,
			inputView,
			statusView,
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerView,
		chatView,
		inputView,
		statusView,
	)
}
