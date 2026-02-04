package tui

import (
	"context"
	"sync"

	"jarvis/internal/config"
	"jarvis/internal/history"
	"jarvis/internal/image"
	"jarvis/internal/tui/components"

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
	SendMessageWithImages(ctx context.Context, text string, images []*image.ImageInput, callback func(msg any)) error
	ClearHistory()
}

type AgentFactory func(apiKey string) (AgentInterface, error)

type Model struct {
	header         *components.Header
	chat           *components.ChatView
	input          *components.InputArea
	status         *components.StatusBar
	history        *history.Manager
	loginModal     *components.LoginModal
	imageIndicator *components.ImageIndicator
	autocomplete   *components.Autocomplete

	agent        AgentInterface
	agentFactory AgentFactory
	config       *config.Config

	width        int
	height       int
	ready        bool
	loading      bool
	showLogin    bool
	err          error
	currentInput string
}

func New(ag AgentInterface, factory AgentFactory, cfg *config.Config, needsLogin bool) Model {
	model := cfg.Anthropic.Model

	historyMgr, _ := history.NewManager()

	return Model{
		header:         components.NewHeader(cfg.App.Name, Version, model),
		chat:           components.NewChatView(cfg.App.Name),
		input:          components.NewInputArea("Type your message..."),
		status:         components.NewStatusBar(cfg.App.Name, Version, model),
		loginModal:     components.NewLoginModal(),
		imageIndicator: components.NewImageIndicator(),
		autocomplete:   components.NewAutocomplete(),
		history:        historyMgr,
		agent:          ag,
		agentFactory:   factory,
		config:         cfg,
		showLogin:      needsLogin,
	}
}

func (m Model) Init() tea.Cmd {
	return m.status.SpinnerTick()
}
