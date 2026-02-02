package tui

import (
	"context"
	"sync"

	"chewbacca/internal/config"
	"chewbacca/internal/history"
	"chewbacca/internal/tui/components"

	tea "github.com/charmbracelet/bubbletea"
)

const Version = "v0.1.0"

var (
	programMu sync.Mutex
	program   *tea.Program

	cancelMu      sync.Mutex
	currentCancel context.CancelFunc
)

func SetProgram(p *tea.Program) {
	programMu.Lock()
	defer programMu.Unlock()
	program = p
}

func getProgram() *tea.Program {
	programMu.Lock()
	defer programMu.Unlock()
	return program
}

func setCurrentCancel(cancel context.CancelFunc) {
	cancelMu.Lock()
	defer cancelMu.Unlock()
	currentCancel = cancel
}

func getCurrentCancel() context.CancelFunc {
	cancelMu.Lock()
	defer cancelMu.Unlock()
	return currentCancel
}

func clearCurrentCancel() {
	cancelMu.Lock()
	defer cancelMu.Unlock()
	currentCancel = nil
}

type AgentInterface interface {
	SendMessage(ctx context.Context, input string, callback func(msg any)) error
	ClearHistory()
}

type Model struct {
	header  *components.Header
	chat    *components.ChatView
	input   *components.InputArea
	status  *components.StatusBar
	history *history.Manager

	agent  AgentInterface
	config *config.Config

	width       int
	height      int
	ready       bool
	loading     bool
	err         error
	currentInput string // Store current input when navigating history
}

func New(ag AgentInterface, cfg *config.Config) Model {
	model := cfg.Anthropic.Model

	// Create history manager
	historyMgr, _ := history.NewManager()

	return Model{
		header:  components.NewHeader(cfg.App.Name, Version, model),
		chat:    components.NewChatView(cfg.App.Name),
		input:   components.NewInputArea("Type your message..."),
		status:  components.NewStatusBar(cfg.App.Name, Version, model),
		history: historyMgr,
		agent:   ag,
		config:  cfg,
	}
}

func (m Model) Init() tea.Cmd {
	return m.status.SpinnerTick()
}
