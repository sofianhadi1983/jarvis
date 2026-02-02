package tui

import (
	"context"
	"strings"
	"time"

	"chewbacca/internal/tui/components"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg)

	case StreamChunkMsg:
		return m.handleStreamChunk(msg)

	case ResponseMsg:
		m.loading = false
		m.status.SetLoading(false)
		return m, m.input.Focus()

	case ToolCallMsg:
		m.chat.AddMessage(components.ChatMessage{
			Role:      components.RoleTool,
			ToolName:  msg.Name,
			ToolInput: msg.Input,
			Content:   msg.Result,
			Timestamp: time.Now(),
		})
		return m, nil

	case ToolStartMsg:
		m.status.SetStatus("Running " + msg.Name + "...")
		return m, nil

	case ErrorMsg:
		m.loading = false
		m.status.SetLoading(false)
		m.err = msg.Err
		m.chat.AddMessage(components.ChatMessage{
			Role:      components.RoleSystem,
			Content:   "Error: " + msg.Err.Error(),
			Timestamp: time.Now(),
		})
		return m, m.input.Focus()
	}

	var cmd tea.Cmd
	m.chat, cmd = m.chat.Update(msg)
	cmds = append(cmds, cmd)

	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	m.status, cmd = m.status.Update(msg)
	cmds = append(cmds, cmd)

	if m.loading {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit

	case tea.KeyEnter:
		if m.loading {
			return m, nil
		}
		if msg.Alt {
			return m, nil
		}
		return m.submitMessage()
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	inputHeight := 5
	statusHeight := 1
	chatHeight := m.height - inputHeight - statusHeight - 2

	m.chat.SetSize(m.width, chatHeight)
	m.input.SetWidth(m.width)
	m.status.SetWidth(m.width)
	m.ready = true

	return m, nil
}

func (m Model) handleStreamChunk(msg StreamChunkMsg) (tea.Model, tea.Cmd) {
	m.chat.AppendToLastMessage(msg.Chunk)
	return m, nil
}

func (m Model) submitMessage() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.input.Value())
	if input == "" {
		return m, nil
	}

	m.chat.AddMessage(components.ChatMessage{
		Role:      components.RoleUser,
		Content:   input,
		Timestamp: time.Now(),
	})

	m.input.Reset()
	m.loading = true
	m.status.SetLoading(true)
	m.input.Blur()

	m.chat.StartAssistantMessage()

	return m, m.sendToAgent(input)
}

func (m Model) sendToAgent(input string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		callback := func(msg tea.Msg) {
		}

		err := m.agent.SendMessage(ctx, input, callback)
		if err != nil {
			return ErrorMsg{Err: err}
		}

		return ResponseMsg{}
	}
}
