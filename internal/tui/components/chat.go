package components

import (
	"fmt"
	"strings"
	"time"

	"jarvis/internal/styles"
	"jarvis/internal/types"
	"jarvis/internal/util"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
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
	Diff      *types.DiffInfo
}

type ChatView struct {
	viewport   viewport.Model
	messages   []ChatMessage
	agentName  string
	width      int
	height     int
	ready      bool
	mdRenderer *glamour.TermRenderer
}

func NewChatView(agentName string) *ChatView {
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(80),
	)
	return &ChatView{
		messages:   []ChatMessage{},
		agentName:  agentName,
		mdRenderer: renderer,
	}
}

func (c *ChatView) SetSize(width, height int) {
	c.width = width
	c.height = height
	c.viewport = viewport.New(width, height)

	wordWrap := width - 6
	if wordWrap < 40 {
		wordWrap = 40
	}

	c.mdRenderer, _ = glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(wordWrap),
	)

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
	if len(c.messages) == 0 || c.messages[len(c.messages)-1].Role != RoleAssistant {
		c.messages = append(c.messages, ChatMessage{
			Role:      RoleAssistant,
			Content:   "",
			Timestamp: time.Now(),
		})
	}

	c.messages[len(c.messages)-1].Content += content
	if c.ready {
		c.viewport.SetContent(c.renderMessages())
		c.viewport.GotoBottom()
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

	bulletStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor)

	userBulletStyle := lipgloss.NewStyle().
		Foreground(styles.UserColor)

	dimStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor)

	textStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	wrapWidth := c.width - 4
	if wrapWidth < 40 {
		wrapWidth = 40
	}

	for _, msg := range c.messages {
		switch msg.Role {
		case RoleUser:
			sb.WriteString(userBulletStyle.Render("> "))
			wrapped := wrapText(msg.Content, wrapWidth-2)
			lines := strings.Split(wrapped, "\n")
			for i, line := range lines {
				if i == 0 {
					sb.WriteString(textStyle.Render(line))
				} else {
					sb.WriteString("\n  ")
					sb.WriteString(textStyle.Render(line))
				}
			}
			sb.WriteString("\n\n")

		case RoleAssistant:
			content := strings.TrimSpace(msg.Content)
			if content == "" {
				continue
			}

			content = stripEmojis(content)

			var rendered string
			if c.mdRenderer != nil {
				if out, err := c.mdRenderer.Render(content); err == nil {
					rendered = strings.TrimSpace(out)
				} else {
					rendered = content
				}
			} else {
				rendered = content
			}

			lines := strings.Split(rendered, "\n")
			for i, line := range lines {
				if i == 0 {
					sb.WriteString(bulletStyle.Render("* "))
					sb.WriteString(line)
				} else {
					sb.WriteString("  ")
					sb.WriteString(line)
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")

		case RoleTool:
			toolDesc := formatToolInput(msg.ToolName, msg.ToolInput)

			sb.WriteString(bulletStyle.Render("* "))
			sb.WriteString(styles.ToolNameStyle.Render(msg.ToolName))
			if toolDesc != "" {
				sb.WriteString(dimStyle.Render("(" + toolDesc + ")"))
			}
			sb.WriteString("\n")

			if msg.Diff != nil {
				sb.WriteString(renderDiff(msg.Diff))
			} else if msg.Content != "" {
				resultPreview := util.TruncateString(msg.Content, 80)
				sb.WriteString(dimStyle.Render("  L "))
				sb.WriteString(dimStyle.Render(resultPreview))
				sb.WriteString("\n")
			}
			sb.WriteString("\n")

		case RoleSystem:
			sb.WriteString(bulletStyle.Render("* "))
			sb.WriteString(dimStyle.Render(msg.Content))
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}

func renderDiff(diff *types.DiffInfo) string {
	var sb strings.Builder

	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	removedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	dimStyle := lipgloss.NewStyle().Foreground(styles.DimColor)

	summary := fmt.Sprintf("  L Added %d lines", diff.AddedLines)
	if diff.RemovedLines > 0 {
		summary += fmt.Sprintf(", removed %d lines", diff.RemovedLines)
	}
	sb.WriteString(dimStyle.Render(summary))
	sb.WriteString("\n")

	lines := strings.Split(diff.UnifiedDiff, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		if len(line) > 6 {
			marker := ""
			if idx := strings.Index(line, " - "); idx > 0 && idx < 6 {
				marker = "-"
			} else if idx := strings.Index(line, " + "); idx > 0 && idx < 6 {
				marker = "+"
			}

			if marker == "-" {
				sb.WriteString(removedStyle.Render(line))
			} else if marker == "+" {
				sb.WriteString(addedStyle.Render(line))
			} else {
				sb.WriteString(dimStyle.Render(line))
			}
		} else {
			sb.WriteString(dimStyle.Render(line))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func formatToolInput(toolName, input string) string {
	input = strings.TrimSpace(input)
	if input == "" || input == "{}" {
		return ""
	}

	if idx := strings.Index(input, `":`); idx != -1 {
		rest := input[idx+2:]
		if strings.HasPrefix(rest, `"`) || strings.HasPrefix(rest, ` "`) {
			rest = strings.TrimPrefix(rest, " ")
			rest = strings.TrimPrefix(rest, `"`)
			if endIdx := strings.Index(rest, `"`); endIdx != -1 {
				value := rest[:endIdx]
				if len(value) > 60 {
					value = value[:57] + "..."
				}
				return value
			}
		}
	}

	if len(input) > 50 {
		return input[:47] + "..."
	}
	return input
}

func stripEmojis(text string) string {
	emojis := []string{
		"📦", "🎯", "🚀", "💡", "✨", "🔧", "⚙️", "📝", "📁", "📂",
		"✅", "❌", "⚠️", "💻", "🖥️", "📊", "📈", "📉", "🔍", "🔎",
		"💾", "📀", "💿", "🗂️", "🗃️", "🗄️", "📋", "📌", "📍", "🏷️",
		"🔑", "🗝️", "🔒", "🔓", "🛠️", "⛏️", "🔨", "🪓", "⚒️", "🛡️",
		"⚡", "🔥", "💥", "✴️", "❇️", "🌟", "⭐", "🌈", "☀️", "🌙",
		"🎉", "🎊", "🎁", "🎈", "🏆", "🥇", "🥈", "🥉", "🏅", "🎖️",
		"👍", "👎", "👌", "✌️", "🤞", "🤝", "👏", "🙌", "💪", "🤔",
		"😀", "😃", "😄", "😁", "😆", "😅", "🤣", "😂", "🙂", "😊",
		"🧠", "💭", "💬", "🗨️", "🗯️", "💤", "💢", "💫", "🎵", "🎶",
		"📚", "📖", "📕", "📗", "📘", "📙", "📓", "📒", "📃", "📜",
		"🔗", "⛓️", "🧰", "🧲", "⚖️", "🔩", "⚙", "🗜️", "⚗️", "🧪",
		"🐛", "🐞", "🦋", "🐌", "🐜", "🦗", "🕷️", "🦂", "🦟", "🪲",
	}

	result := text
	for _, emoji := range emojis {
		result = strings.ReplaceAll(result, emoji, "")
	}

	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}

	return result
}

func wrapText(text string, width int) string {
	if width <= 0 {
		width = 80
	}

	var result strings.Builder
	words := strings.Fields(text)
	lineLen := 0

	for i, word := range words {
		wordLen := len(word)

		if lineLen+wordLen+1 > width && lineLen > 0 {
			result.WriteString("\n")
			lineLen = 0
		}

		if lineLen > 0 {
			result.WriteString(" ")
			lineLen++
		}

		result.WriteString(word)
		lineLen += wordLen

		_ = i
	}

	return result.String()
}

func (c *ChatView) Messages() []ChatMessage {
	return c.messages
}

func (c *ChatView) GotoBottom() {
	if c.ready {
		c.viewport.GotoBottom()
	}
}

func (c *ChatView) ClearMessages() {
	c.messages = []ChatMessage{}
	if c.ready {
		c.viewport.SetContent(c.renderMessages())
	}
}
