package styles

import "github.com/charmbracelet/lipgloss"

var (
	UserColor      = lipgloss.Color("39")
	AssistantColor = lipgloss.Color("208")
	ToolColor      = lipgloss.Color("46")
	ErrorColor     = lipgloss.Color("196")
	DimColor       = lipgloss.Color("240")
	BorderColor    = lipgloss.Color("62")
	StatusBgColor  = lipgloss.Color("236")
)

var (
	UserStyle = lipgloss.NewStyle().
			Foreground(UserColor).
			Bold(true)

	AssistantStyle = lipgloss.NewStyle().
			Foreground(AssistantColor).
			Bold(true)

	ToolStyle = lipgloss.NewStyle().
			Foreground(ToolColor)

	ToolNameStyle = lipgloss.NewStyle().
			Foreground(ToolColor).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Bold(true)

	DimStyle = lipgloss.NewStyle().
			Foreground(DimColor)

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor)

	InputBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(BorderColor).
				Padding(0, 1)

	StatusBarStyle = lipgloss.NewStyle().
			Background(StatusBgColor).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(AssistantColor)

	HelpStyle = lipgloss.NewStyle().
			Foreground(DimColor)
)
