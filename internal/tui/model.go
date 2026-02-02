package tui

import (
	"chewbacca/internal/agent"
	"chewbacca/internal/config"
	"chewbacca/internal/tui/components"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	chat    *components.ChatView
	input   *components.InputArea
	status  *components.StatusBar
	spinner spinner.Model

	agent  *agent.Agent
	config *config.Config

	width   int
	height  int
	ready   bool
	loading bool
	err     error

	streamChan chan string
}

func New(ag *agent.Agent, cfg *config.Config) Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = SpinnerStyle

	return Model{
		chat:    components.NewChatView(cfg.App.Name),
		input:   components.NewInputArea("Type your message..."),
		status:  components.NewStatusBar(cfg.App.Name),
		spinner: sp,
		agent:   ag,
		config:  cfg,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.status.SpinnerTick(),
	)
}
