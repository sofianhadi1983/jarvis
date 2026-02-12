package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"jarvis/internal/auth"
	"jarvis/internal/clipboard"
	"jarvis/internal/image"
	"jarvis/internal/references"
	"jarvis/internal/tui/components"
	"jarvis/internal/util"

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

	case TodoUpdateMsg:
		if msg.ActiveForm != "" {
			m.status.SetStatus(msg.ActiveForm)
		}
		if m.loading {
			return m, m.status.SpinnerTick()
		}
		return m, nil

	case ParallelGroupStartMsg:
		m.parallelGroup = &components.ParallelGroupState{
			GroupID:    msg.GroupID,
			TaskNames:  msg.TaskNames,
			AgentTypes: msg.AgentTypes,
			ToolCounts: make([]int, len(msg.TaskNames)),
			Statuses:   make([]string, len(msg.TaskNames)),
		}
		m.chat.AddMessage(components.ChatMessage{
			Role:               components.RoleParallelGroup,
			Timestamp:          time.Now(),
			ParallelTaskNames:  msg.TaskNames,
			ParallelAgentTypes: msg.AgentTypes,
			ParallelToolCounts: make([]int, len(msg.TaskNames)),
			ParallelStatuses:   make([]string, len(msg.TaskNames)),
		})
		m.status.SetStatus(fmt.Sprintf("Running %d agents in parallel...", len(msg.TaskNames)))
		if m.loading {
			return m, m.status.SpinnerTick()
		}
		return m, nil

	case ParallelAgentUpdateMsg:
		if m.parallelGroup != nil && msg.GroupID == m.parallelGroup.GroupID {
			if msg.AgentIndex >= 0 && msg.AgentIndex < len(m.parallelGroup.ToolCounts) {
				m.parallelGroup.ToolCounts[msg.AgentIndex] = msg.ToolCount
				m.parallelGroup.Statuses[msg.AgentIndex] = msg.Status
				m.chat.UpdateParallelGroup(m.parallelGroup)
			}
		}
		if m.loading {
			return m, m.status.SpinnerTick()
		}
		return m, nil

	case ParallelGroupDoneMsg:
		if m.parallelGroup != nil && msg.GroupID == m.parallelGroup.GroupID {
			m.chat.FinalizeParallelGroup(m.parallelGroup)
			m.parallelGroup = nil
		}
		if m.loading {
			return m, m.status.SpinnerTick()
		}
		return m, nil

	case SkillLoadedMsg:
		m.chat.AddMessage(components.ChatMessage{
			Role:      components.RoleSystem,
			Content:   "Skill loaded: " + msg.Name,
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
	if m.showLogin {
		return m.handleLoginKeyMsg(msg)
	}

	if m.autocomplete.IsVisible() {
		switch msg.Type {
		case tea.KeyUp:
			m.autocomplete.SelectPrev()
			return m, nil
		case tea.KeyDown:
			m.autocomplete.SelectNext()
			return m, nil
		case tea.KeyTab, tea.KeyEnter:
			if m.autocomplete.HasItems() {
				path := m.autocomplete.GetSelectedPath()
				atPos := m.autocomplete.GetAtPosition()
				m.input.InsertCompletion(atPos, "@"+path)
			}
			m.autocomplete.Hide()
			return m, nil
		case tea.KeyEsc:
			m.autocomplete.Hide()
			return m, nil
		case tea.KeyBackspace:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.updateAutocompleteQuery()
			return m, cmd
		case tea.KeyRunes:
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			m.updateAutocompleteQuery()
			return m, cmd
		}
	}

	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit

	case tea.KeyCtrlV:
		if !m.loading {
			return m.handlePaste()
		}
		return m, nil

	case tea.KeyEsc:
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
		// If images exist and input empty, select images
		if !m.loading && m.imageIndicator.HasImages() && m.input.Value() == "" {
			if m.imageIndicator.IsSelected() {
				m.imageIndicator.SelectPrev()
			} else {
				m.imageIndicator.SelectLast()
			}
			return m, nil
		}
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
		// Deselect image if selected
		if m.imageIndicator.IsSelected() {
			m.imageIndicator.SelectNext()
			return m, nil
		}
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

	case tea.KeyBackspace, tea.KeyDelete:
		// Remove selected image
		if m.imageIndicator.IsSelected() {
			m.imageIndicator.RemoveSelected()
			return m, nil
		}
		// Fall through to let textarea handle it

	case tea.KeyEnter:
		if m.loading {
			return m, nil
		}
		if msg.Alt {
			return m, nil
		}
		return m.submitMessage()

	case tea.KeyRunes:
		for _, r := range msg.Runes {
			if r == '@' {
				atPos := m.input.CursorPosition()
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				m.autocomplete.Show(atPos)
				return m, cmd
			}
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) updateAutocompleteQuery() {
	if !m.autocomplete.IsVisible() {
		return
	}

	atPos := m.autocomplete.GetAtPosition()
	query := m.input.GetTextAfter(atPos + 1)

	if strings.Contains(query, " ") {
		m.autocomplete.Hide()
		return
	}

	value := m.input.Value()
	if atPos >= len(value) || (atPos < len(value) && value[atPos] != '@') {
		m.autocomplete.Hide()
		return
	}

	m.autocomplete.SetQuery(query)
}

func (m Model) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	m.loginModal.SetSize(m.width, m.height)

	headerHeight := 5
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
	m.autocomplete.SetWidth(m.width)
	m.ready = true

	return m, nil
}

func (m Model) handleLoginKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	var cmd tea.Cmd
	m.loginModal, cmd = m.loginModal.Update(msg)

	if m.loginModal.IsSubmitted() {
		return m.handleLoginComplete()
	}

	return m, cmd
}

func (m Model) handleLoginComplete() (tea.Model, tea.Cmd) {
	method := m.loginModal.GetMethod()

	if method == components.LoginMethodAPIKey {
		apiKey := m.loginModal.GetAPIKey()
		if wd, err := os.Getwd(); err == nil {
			util.SaveAPIKeyToEnv(filepath.Join(wd, ".env"), apiKey)
		}

		if m.agentFactory != nil {
			agent, err := m.agentFactory(apiKey)
			if err != nil {
				m.chat.AddMessage(components.ChatMessage{
					Role:      components.RoleSystem,
					Content:   "Failed to initialize agent: " + err.Error(),
					Timestamp: time.Now(),
				})
				return m, tea.Quit
			}
			m.agent = agent
		}

		m.showLogin = false
		return m, m.input.Focus()
	}

	if method == components.LoginMethodSubscription {
		code := m.loginModal.GetOAuthCode()
		callbackState := m.loginModal.GetOAuthCallbackState()
		oauthState := m.loginModal.GetOAuthState()

		// Exchange code for tokens
		tokens, err := auth.ExchangeCodeForTokens(code, callbackState, oauthState)
		if err != nil {
			// Reset login modal to allow retry
			m.loginModal.SetError("OAuth failed: " + err.Error())
			m.loginModal.Reset()
			return m, nil
		}

		// Save tokens to .env
		wd, err := os.Getwd()
		if err != nil {
			m.loginModal.SetError("Failed to get working directory: " + err.Error())
			m.loginModal.Reset()
			return m, nil
		}

		envPath := filepath.Join(wd, ".env")
		if err := auth.SaveOAuthTokens(envPath, tokens); err != nil {
			m.loginModal.SetError("Failed to save tokens: " + err.Error())
			m.loginModal.Reset()
			return m, nil
		}

		// Set env var for SDK
		os.Setenv("ANTHROPIC_ACCESS_TOKEN", tokens.AccessToken)

		// Create agent with empty apiKey (signals OAuth mode)
		if m.agentFactory != nil {
			agent, err := m.agentFactory("")
			if err != nil {
				m.loginModal.SetError("Failed to initialize agent: " + err.Error())
				m.loginModal.Reset()
				return m, nil
			}
			m.agent = agent
		}

		m.showLogin = false
		return m, m.input.Focus()
	}

	return m, nil
}

func (m Model) handleStreamChunk(msg StreamChunkMsg) (tea.Model, tea.Cmd) {
	m.chat.AppendToLastMessage(msg.Chunk)
	if m.loading {
		return m, m.status.SpinnerTick()
	}
	return m, nil
}

func (m Model) handlePaste() (tea.Model, tea.Cmd) {
	data, err := clipboard.ReadImage()
	if err != nil {
		m.status.SetStatus("Clipboard error: " + err.Error())
		return m, nil
	}
	if data == nil {
		m.status.SetStatus("No image in clipboard")
		return m, nil
	}

	img, err := image.NewFromBytes(data, fmt.Sprintf("Pasted Image #%d", m.imageIndicator.Count()+1))
	if err != nil {
		m.status.SetStatus("Invalid image: " + err.Error())
		return m, nil
	}

	m.imageIndicator.AddImage(img)
	m.status.SetStatus("")
	return m, nil
}

func (m Model) submitMessage() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.input.Value())

	// Allow submission with just images (no text required)
	if input == "" && !m.imageIndicator.HasImages() {
		return m, nil
	}

	// Hide autocomplete if visible
	m.autocomplete.Hide()

	lowercaseInput := strings.ToLower(input)

	// Handle exit commands
	if lowercaseInput == "exit" || lowercaseInput == "/exit" || lowercaseInput == "quit" || lowercaseInput == "/quit" {
		return m, tea.Quit
	}

	if lowercaseInput == "/logout" || lowercaseInput == "logout" {
		m.chat.ClearMessages()
		if m.history != nil {
			m.history.Clear()
		}
		m.agent.ClearHistory()
		if wd, err := os.Getwd(); err == nil {
			auth.ClearAllTokens(filepath.Join(wd, ".env"))
		}
		return m, tea.Quit
	}

	// Handle clear command - clears history and context
	if lowercaseInput == "/clear" || lowercaseInput == "clear" {
		m.input.Reset()
		m.chat.ClearMessages()
		m.imageIndicator.Clear()
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

	// Check for skill invocation: /skill-name [args]
	if strings.HasPrefix(input, "/") && m.skillLoader != nil {
		parts := strings.SplitN(input[1:], " ", 2)
		skillName := parts[0]
		if m.skillLoader.Has(skillName) {
			skillArgs := ""
			if len(parts) > 1 {
				skillArgs = parts[1]
			}

			content, err := m.skillLoader.GetSkillContent(skillName)
			if err != nil {
				m.chat.AddMessage(components.ChatMessage{
					Role:      components.RoleSystem,
					Content:   "Failed to load skill: " + err.Error(),
					Timestamp: time.Now(),
				})
				return m, nil
			}

			// Display user message
			m.chat.AddMessage(components.ChatMessage{
				Role:      components.RoleUser,
				Content:   input,
				Timestamp: time.Now(),
			})

			// Display skill loaded indicator
			m.chat.AddMessage(components.ChatMessage{
				Role:      components.RoleSystem,
				Content:   "Skill loaded: " + skillName,
				Timestamp: time.Now(),
			})

			// Build combined prompt: skill content + user args
			prompt := fmt.Sprintf("<skill-loaded name=\"%s\">\n%s\n</skill-loaded>\n\nFollow the instructions above.", skillName, content)
			if skillArgs != "" {
				prompt += "\n\nUser request: " + skillArgs
			}

			if m.history != nil && input != "" {
				m.history.Add(input)
				m.history.ResetIndex()
			}
			m.currentInput = ""

			m.input.Reset()
			m.loading = true
			m.status.SetLoading(true)
			m.status.SetStatus("Thinking...")
			m.input.Blur()
			m.chat.StartAssistantMessage()

			ctx, cancel := context.WithCancel(context.Background())
			setCurrentCancel(cancel)

			return m, tea.Batch(m.sendToAgent(ctx, prompt), m.status.SpinnerTick())
		}
	}

	// Parse input for file references (images AND text files)
	parsed := references.ParseInput(input)

	// Show errors for failed file loads
	if parsed.HasErrors() {
		for _, errMsg := range parsed.Errors {
			m.chat.AddMessage(components.ChatMessage{
				Role:      components.RoleSystem,
				Content:   "File error: " + errMsg,
				Timestamp: time.Now(),
			})
		}
	}

	// Combine pasted images with @referenced images
	pastedImages := m.imageIndicator.Images()
	referencedImages := parsed.GetImages()
	allImages := append(pastedImages, referencedImages...)

	// Combine text with file contents
	fullText := parsed.CombineTextContent()

	// Add to history (original input with @references)
	if m.history != nil && input != "" {
		m.history.Add(input)
		m.history.ResetIndex()
	}
	m.currentInput = ""

	// Build display content with file indicators
	displayContent := parsed.Text
	var indicators []string

	// Add pasted image indicators
	for i := range pastedImages {
		indicators = append(indicators, fmt.Sprintf("[Pasted Image #%d]", i+1))
	}
	// Add @referenced file indicators
	indicators = append(indicators, parsed.FileIndicators()...)

	if len(indicators) > 0 {
		if displayContent != "" {
			displayContent += "\n"
		}
		displayContent += strings.Join(indicators, " ")
	}

	m.chat.AddMessage(components.ChatMessage{
		Role:      components.RoleUser,
		Content:   displayContent,
		Timestamp: time.Now(),
	})

	// Clear pasted images after submit
	m.imageIndicator.Clear()

	m.input.Reset()
	m.loading = true
	m.status.SetLoading(true)
	m.status.SetStatus("Thinking...")
	m.input.Blur()

	m.chat.StartAssistantMessage()

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	setCurrentCancel(cancel)

	// Use sendToAgentWithImages if we have images, otherwise use sendToAgent
	if len(allImages) > 0 {
		combinedParsed := &image.ParsedInput{Text: fullText, Images: allImages}
		return m, tea.Batch(m.sendToAgentWithImages(ctx, combinedParsed), m.status.SpinnerTick())
	}
	return m, tea.Batch(m.sendToAgent(ctx, fullText), m.status.SpinnerTick())
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

func (m Model) sendToAgentWithImages(ctx context.Context, parsed *image.ParsedInput) tea.Cmd {
	agent := m.agent
	return func() tea.Msg {
		callback := func(msg any) {
			if p := getProgram(); p != nil {
				p.Send(msg)
			}
		}

		err := agent.SendMessageWithImages(ctx, parsed.Text, parsed.Images, callback)
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
