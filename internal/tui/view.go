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
	autocompleteView := m.autocomplete.View()
	inputView := m.input.View()
	statusView := m.status.View()

	// Build list of views to join
	views := []string{headerView, chatView}

	// Add image indicator if present
	if imageIndicatorView != "" {
		views = append(views, imageIndicatorView)
	}

	// Add autocomplete popup if visible
	if autocompleteView != "" {
		views = append(views, autocompleteView)
	}

	views = append(views, inputView, statusView)

	return lipgloss.JoinVertical(lipgloss.Left, views...)
}
