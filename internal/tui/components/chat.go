package components

import (
	"fmt"
	"strings"
	"time"

	"chewbacca/internal/tui"
	"chewbacca/internal/util"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type MessageRole int

const (
	RoleUser MessageRole = iota
	RoleAssistant
	RoleTool
	RoleSystem
)

type ChatMessage struct {
	Role      MessageRole
	Content   string
	ToolName  string
	ToolInput string
	Timestamp time.Time
}

type ChatView struct {
	viewport  viewport.Model
	messages  []ChatMessage
	agentName string
	width     int
	height    int
	ready     bool
}

func NewChatView(agentName string) *ChatView {
	return &ChatView{
		messages:  []ChatMessage{},
		agentName: agentName,
	}
}

func (c *ChatView) SetSize(width, height int) {
	c.width = width
	c.height = height
	c.viewport = viewport.New(width, height)
	c.viewport.SetContent(c.renderMessages())
	c.ready = true
}

func (c *ChatView) AddMessage(msg ChatMessage) {
	c.messages = append(c.messages, msg)
	if c.ready {
		c.viewport.SetContent(c.renderMessages())
		c.viewport.GotoBottom()
	}
}

func (c *ChatView) AppendToLastMessage(content string) {
	if len(c.messages) > 0 && c.messages[len(c.messages)-1].Role == RoleAssistant {
		c.messages[len(c.messages)-1].Content += content
		if c.ready {
			c.viewport.SetContent(c.renderMessages())
			c.viewport.GotoBottom()
		}
	}
}

func (c *ChatView) StartAssistantMessage() {
	c.messages = append(c.messages, ChatMessage{
		Role:      RoleAssistant,
		Content:   "",
		Timestamp: time.Now(),
	})
}

func (c *ChatView) Update(msg tea.Msg) (*ChatView, tea.Cmd) {
	if !c.ready {
		return c, nil
	}
	var cmd tea.Cmd
	c.viewport, cmd = c.viewport.Update(msg)
	return c, cmd
}

func (c *ChatView) View() string {
	if !c.ready {
		return ""
	}
	return c.viewport.View()
}

func (c *ChatView) renderMessages() string {
	var sb strings.Builder

	userMsgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		PaddingLeft(2)

	assistantMsgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		PaddingLeft(2)

	toolBlockStyle := lipgloss.NewStyle().
		Foreground(tui.ToolColor).
		PaddingLeft(4)

	for _, msg := range c.messages {
		switch msg.Role {
		case RoleUser:
			sb.WriteString(tui.UserStyle.Render("> "))
			sb.WriteString(userMsgStyle.Render(msg.Content))
			sb.WriteString("\n\n")

		case RoleAssistant:
			sb.WriteString(assistantMsgStyle.Render(msg.Content))
			sb.WriteString("\n\n")

		case RoleTool:
			toolHeader := fmt.Sprintf("[%s] %s", msg.ToolName, util.TruncateString(msg.ToolInput, 60))
			sb.WriteString(toolBlockStyle.Render(toolHeader))
			sb.WriteString("\n")
			if msg.Content != "" {
				resultPreview := util.TruncateString(msg.Content, 150)
				sb.WriteString(tui.DimStyle.PaddingLeft(6).Render(resultPreview))
				sb.WriteString("\n")
			}
			sb.WriteString("\n")

		case RoleSystem:
			sb.WriteString(tui.DimStyle.Render(msg.Content))
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}

func (c *ChatView) Messages() []ChatMessage {
	return c.messages
}

func (c *ChatView) GotoBottom() {
	if c.ready {
		c.viewport.GotoBottom()
	}
}
