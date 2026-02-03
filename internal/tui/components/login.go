package components

import (
	"jarvis/internal/auth"
	"jarvis/internal/styles"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type LoginMethod int

const (
	LoginMethodNone LoginMethod = iota
	LoginMethodAPIKey
	LoginMethodSubscription
)

type LoginState int

const (
	LoginStateSelection LoginState = iota
	LoginStateAPIKeyInput
	LoginStateOAuthWaiting
	LoginStateOAuthCodeInput
)

type LoginModal struct {
	width              int
	height             int
	selectedIndex      int
	state              LoginState
	apiKeyInput        textinput.Model
	oauthCodeInput     textinput.Model
	submitted          bool
	method             LoginMethod
	apiKey             string
	oauthState         *auth.OAuthState
	oauthCode          string
	oauthCallbackState string
	oauthError         string
}

func NewLoginModal() *LoginModal {
	apiInput := textinput.New()
	apiInput.Placeholder = "sk-ant-..."
	apiInput.CharLimit = 200
	apiInput.Width = 50
	apiInput.EchoMode = textinput.EchoPassword
	apiInput.EchoCharacter = '•'

	oauthInput := textinput.New()
	oauthInput.Placeholder = "code#state"
	oauthInput.CharLimit = 500
	oauthInput.Width = 50

	return &LoginModal{
		selectedIndex:  0,
		state:          LoginStateSelection,
		apiKeyInput:    apiInput,
		oauthCodeInput: oauthInput,
	}
}

func (l *LoginModal) SetSize(width, height int) {
	l.width = width
	l.height = height
}

func (l *LoginModal) Update(msg tea.Msg) (*LoginModal, tea.Cmd) {
	switch l.state {
	case LoginStateSelection:
		return l.updateSelection(msg)
	case LoginStateAPIKeyInput:
		return l.updateAPIInput(msg)
	case LoginStateOAuthWaiting:
		return l.updateOAuthWaiting(msg)
	case LoginStateOAuthCodeInput:
		return l.updateOAuthCodeInput(msg)
	}
	return l, nil
}

func (l *LoginModal) updateSelection(msg tea.Msg) (*LoginModal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if l.selectedIndex > 0 {
				l.selectedIndex--
			}
		case "down", "j":
			if l.selectedIndex < 1 {
				l.selectedIndex++
			}
		case "enter":
			if l.selectedIndex == 0 {
				l.state = LoginStateAPIKeyInput
				l.apiKeyInput.Focus()
				return l, textinput.Blink
			} else {
				// Anthropic Subscription selected - start OAuth flow
				oauthState, err := auth.GenerateOAuthState()
				if err != nil {
					l.oauthError = "Failed to generate OAuth state: " + err.Error()
					return l, nil
				}
				l.oauthState = oauthState

				authURL := auth.BuildAuthorizationURL(oauthState)
				if err := auth.OpenBrowser(authURL); err != nil {
					l.oauthError = "Failed to open browser: " + err.Error()
					return l, nil
				}

				l.state = LoginStateOAuthWaiting
				return l, nil
			}
		}
	}
	return l, nil
}

func (l *LoginModal) updateAPIInput(msg tea.Msg) (*LoginModal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			l.state = LoginStateSelection
			l.apiKeyInput.Reset()
			return l, nil
		case "enter":
			if l.apiKeyInput.Value() != "" {
				l.submitted = true
				l.method = LoginMethodAPIKey
				l.apiKey = l.apiKeyInput.Value()
			}
			return l, nil
		}
	}

	var cmd tea.Cmd
	l.apiKeyInput, cmd = l.apiKeyInput.Update(msg)
	return l, cmd
}

func (l *LoginModal) updateOAuthWaiting(msg tea.Msg) (*LoginModal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			l.state = LoginStateSelection
			l.oauthState = nil
			l.oauthError = ""
			return l, nil
		case "enter":
			// Move to code input
			l.state = LoginStateOAuthCodeInput
			l.oauthCodeInput.Focus()
			return l, textinput.Blink
		}
	}
	return l, nil
}

func (l *LoginModal) updateOAuthCodeInput(msg tea.Msg) (*LoginModal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			l.state = LoginStateOAuthWaiting
			l.oauthCodeInput.Reset()
			l.oauthError = ""
			return l, nil
		case "enter":
			input := l.oauthCodeInput.Value()
			if input != "" {
				code, state, err := auth.ParseAuthCode(input)
				if err != nil {
					l.oauthError = err.Error()
					return l, nil
				}
				l.oauthCode = code
				l.oauthCallbackState = state
				l.submitted = true
				l.method = LoginMethodSubscription
			}
			return l, nil
		}
	}

	var cmd tea.Cmd
	l.oauthCodeInput, cmd = l.oauthCodeInput.Update(msg)
	return l, cmd
}

func (l *LoginModal) View() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.AssistantColor).
		MarginBottom(1)

	title := titleStyle.Render("Welcome to Jarvis")

	switch l.state {
	case LoginStateAPIKeyInput:
		return l.renderAPIInputView(title)
	case LoginStateOAuthWaiting:
		return l.renderOAuthWaitingView(title)
	case LoginStateOAuthCodeInput:
		return l.renderOAuthCodeInputView(title)
	default:
		return l.renderSelectionView(title)
	}
}

func (l *LoginModal) renderSelectionView(title string) string {
	subtitleStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor).
		MarginBottom(2)

	subtitle := subtitleStyle.Render("Please select a login method:")

	options := []string{"API Key", "Anthropic Subscription"}
	var optionViews []string

	for i, opt := range options {
		style := lipgloss.NewStyle().PaddingLeft(2)
		if i == l.selectedIndex {
			style = style.Foreground(styles.UserColor).Bold(true)
			optionViews = append(optionViews, style.Render("▸ "+opt))
		} else {
			style = style.Foreground(lipgloss.Color("252"))
			optionViews = append(optionViews, style.Render("  "+opt))
		}
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor).
		MarginTop(2)

	help := helpStyle.Render("↑/↓: navigate • enter: select")

	var content string
	if l.oauthError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			MarginTop(1)
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			subtitle,
			lipgloss.JoinVertical(lipgloss.Left, optionViews...),
			errorStyle.Render(l.oauthError),
			help,
		)
	} else {
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			subtitle,
			lipgloss.JoinVertical(lipgloss.Left, optionViews...),
			help,
		)
	}

	return l.centerContent(content)
}

func (l *LoginModal) renderAPIInputView(title string) string {
	subtitleStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor).
		MarginBottom(2)

	subtitle := subtitleStyle.Render("Enter your Anthropic API Key:")

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.BorderColor).
		Padding(0, 1).
		Width(54)

	input := inputStyle.Render(l.apiKeyInput.View())

	helpStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor).
		MarginTop(2)

	help := helpStyle.Render("enter: submit • esc: back")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		input,
		help,
	)

	return l.centerContent(content)
}

func (l *LoginModal) renderOAuthWaitingView(title string) string {
	subtitleStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor).
		MarginBottom(2)

	subtitle := subtitleStyle.Render("Browser opened for authentication")

	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		MarginBottom(1)

	instructions := []string{
		instructionStyle.Render("1. Complete sign-in in your browser"),
		instructionStyle.Render("2. Copy the authentication code"),
		instructionStyle.Render("3. Press Enter to paste the code"),
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor).
		MarginTop(2)

	help := helpStyle.Render("enter: paste code • esc: cancel")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		subtitle,
		lipgloss.JoinVertical(lipgloss.Left, instructions...),
		help,
	)

	return l.centerContent(content)
}

func (l *LoginModal) renderOAuthCodeInputView(title string) string {
	subtitleStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor).
		MarginBottom(2)

	subtitle := subtitleStyle.Render("Paste the authentication code (code#state):")

	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.BorderColor).
		Padding(0, 1).
		Width(54)

	input := inputStyle.Render(l.oauthCodeInput.View())

	helpStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor).
		MarginTop(2)

	help := helpStyle.Render("enter: submit • esc: back")

	var content string
	if l.oauthError != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			MarginTop(1)
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			subtitle,
			input,
			errorStyle.Render(l.oauthError),
			help,
		)
	} else {
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			subtitle,
			input,
			help,
		)
	}

	return l.centerContent(content)
}

func (l *LoginModal) centerContent(content string) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.BorderColor).
		Padding(2, 4)

	box := boxStyle.Render(content)

	return lipgloss.Place(
		l.width,
		l.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

func (l *LoginModal) IsSubmitted() bool {
	return l.submitted
}

func (l *LoginModal) GetMethod() LoginMethod {
	return l.method
}

func (l *LoginModal) GetAPIKey() string {
	return l.apiKey
}

func (l *LoginModal) GetOAuthCode() string {
	return l.oauthCode
}

func (l *LoginModal) GetOAuthState() *auth.OAuthState {
	return l.oauthState
}

func (l *LoginModal) GetOAuthCallbackState() string {
	return l.oauthCallbackState
}

// SetError sets an error message to display on the login modal.
func (l *LoginModal) SetError(err string) {
	l.oauthError = err
}

// Reset resets the login modal to selection state for retry.
func (l *LoginModal) Reset() {
	l.state = LoginStateSelection
	l.submitted = false
	l.method = LoginMethodNone
	l.oauthCode = ""
	l.oauthCallbackState = ""
	l.oauthState = nil
	l.apiKeyInput.Reset()
	l.oauthCodeInput.Reset()
}
