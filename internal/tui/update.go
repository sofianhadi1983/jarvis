package tui

import (
	"context"
	"strings"
	"time"

	"jarvis/internal/tui/components"

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
			Diff:      msg.Diff,
			Timestamp: time.Now(),
		})
		if m.loading {
			return m, m.status.SpinnerTick()
		}
		return m, nil

	case ToolStartMsg:
		m.status.SetStatus("Running " + msg.Name + "...")
		if m.loading {
			return m, m.status.SpinnerTick()
		}
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

	return m, tea.Batch(cmds...)
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit

	case tea.KeyEsc:
		// Interrupt running process
		if m.loading {
			if cancel := getCurrentCancel(); cancel != nil {
				cancel()
				clearCurrentCancel()
			}
			m.loading = false
			m.status.SetLoading(false)
			m.status.SetStatus("Interrupted")
			m.chat.AddMessage(components.ChatMessage{
				Role:      components.RoleSystem,
				Content:   "Process interrupted by user",
				Timestamp: time.Now(),
			})
			return m, m.input.Focus()
		}
		return m, nil

	case tea.KeyUp:
		// Navigate to previous history entry
		if !m.loading && m.history != nil {
			// Store current input if we're starting navigation
			if m.currentInput == "" && m.input.Value() != "" {
				m.currentInput = m.input.Value()
			}
			if prev, ok := m.history.Previous(); ok {
				m.input.SetValue(prev)
			}
		}
		return m, nil

	case tea.KeyDown:
		// Navigate to next history entry
		if !m.loading && m.history != nil {
			if next, ok := m.history.Next(); ok {
				if next == "" {
					// Restore current input when reaching the end
					m.input.SetValue(m.currentInput)
					m.currentInput = ""
				} else {
					m.input.SetValue(next)
				}
			}
		}
		return m, nil

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

	headerHeight := 5 // App name, model, workdir + padding
	inputHeight := 3
	statusHeight := 1
	chatHeight := m.height - headerHeight - inputHeight - statusHeight

	if chatHeight < 5 {
		chatHeight = 5
	}

	m.header.SetWidth(m.width)
	m.chat.SetSize(m.width, chatHeight)
	m.input.SetWidth(m.width)
	m.status.SetWidth(m.width)
	m.ready = true

	return m, nil
}

func (m Model) handleStreamChunk(msg StreamChunkMsg) (tea.Model, tea.Cmd) {
	m.chat.AppendToLastMessage(msg.Chunk)
	if m.loading {
		return m, m.status.SpinnerTick()
	}
	return m, nil
}

func (m Model) submitMessage() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.input.Value())
	if input == "" {
		return m, nil
	}

	lowercaseInput := strings.ToLower(input)

	// Handle exit commands
	if lowercaseInput == "exit" || lowercaseInput == "/exit" || lowercaseInput == "quit" || lowercaseInput == "/quit" {
		return m, tea.Quit
	}

	// Handle clear command - clears history and context
	if lowercaseInput == "/clear" || lowercaseInput == "clear" {
		m.input.Reset()
		m.chat.ClearMessages()
		if m.history != nil {
			m.history.Clear()
		}
		m.agent.ClearHistory()
		m.chat.AddMessage(components.ChatMessage{
			Role:      components.RoleSystem,
			Content:   "History and context cleared",
			Timestamp: time.Now(),
		})
		return m, nil
	}

	// Add to history
	if m.history != nil {
		m.history.Add(input)
		m.history.ResetIndex()
	}
	m.currentInput = ""

	m.chat.AddMessage(components.ChatMessage{
		Role:      components.RoleUser,
		Content:   input,
		Timestamp: time.Now(),
	})

	m.input.Reset()
	m.loading = true
	m.status.SetLoading(true)
	m.status.SetStatus("Thinking...")
	m.input.Blur()

	m.chat.StartAssistantMessage()

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	setCurrentCancel(cancel)

	return m, tea.Batch(m.sendToAgent(ctx, input), m.status.SpinnerTick())
}

func (m Model) sendToAgent(ctx context.Context, input string) tea.Cmd {
	agent := m.agent
	return func() tea.Msg {
		callback := func(msg any) {
			if p := getProgram(); p != nil {
				p.Send(msg)
			}
		}

		err := agent.SendMessage(ctx, input, callback)
		if err != nil {
			// Check if it was cancelled
			if ctx.Err() == context.Canceled {
				return nil // Don't send error for cancellation
			}
			return ErrorMsg{Err: err}
		}

		return ResponseMsg{}
	}
}
