package components

import (
	"fmt"

	"jarvis/internal/styles"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type StatusBar struct {
	appName string
	version string
	model   string
	spinner spinner.Model
	loading bool
	status  string
	width   int
}

func NewStatusBar(appName, version, model string) *StatusBar {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = styles.SpinnerStyle

	return &StatusBar{
		appName: appName,
		version: version,
		model:   model,
		spinner: sp,
		status:  "",
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
		s.status = ""
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
	helpStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor)

	rightStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor)

	escHintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	var leftContent string
	if s.loading && s.status != "" {
		spinnerView := s.spinner.View()
		leftContent = fmt.Sprintf("%s %s", spinnerView, s.status)
	} else {
		leftContent = "Press / for commands or @ for files"
	}
	left := helpStyle.Render(leftContent)

	var rightContent string
	if s.loading {
		escHint := escHintStyle.Render("Esc to interrupt")
		rightContent = fmt.Sprintf("%s  %s %s [%s]", escHint, s.appName, s.version, s.model)
	} else {
		rightContent = fmt.Sprintf("%s %s [%s]", s.appName, s.version, s.model)
	}
	right := rightStyle.Render(rightContent)

	gap := s.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	return fmt.Sprintf("%s%*s%s", left, gap, "", right)
}

func (s *StatusBar) SpinnerTick() tea.Cmd {
	return s.spinner.Tick
}
