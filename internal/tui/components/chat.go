package components

import (
	"fmt"
	"strings"
	"time"

	"chewbacca/internal/styles"
	"chewbacca/internal/types"
	"chewbacca/internal/util"

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
	Diff      *types.DiffInfo // Optional diff info for Update tool
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

	// Recreate renderer with proper width
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
	// If the last message is not an assistant message, start a new one
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
			// Show user message with > prefix
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

			// Strip emojis from content before rendering
			content = stripEmojis(content)

			// Render with glamour
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

			// Add bullet prefix and proper indentation
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
			// Tool calls with Claude Code style formatting
			toolDesc := formatToolInput(msg.ToolName, msg.ToolInput)

			sb.WriteString(bulletStyle.Render("* "))
			sb.WriteString(styles.ToolNameStyle.Render(msg.ToolName))
			if toolDesc != "" {
				sb.WriteString(dimStyle.Render("(" + toolDesc + ")"))
			}
			sb.WriteString("\n")

			// Check if we have diff info for Update tool
			if msg.Diff != nil {
				sb.WriteString(renderDiff(msg.Diff))
			} else if msg.Content != "" {
				// Show result with tree connector
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

// renderDiff renders a diff view for file updates
func renderDiff(diff *types.DiffInfo) string {
	var sb strings.Builder

	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))    // Green
	removedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196")) // Red
	dimStyle := lipgloss.NewStyle().Foreground(styles.DimColor)

	// Summary line
	summary := fmt.Sprintf("  L Added %d lines", diff.AddedLines)
	if diff.RemovedLines > 0 {
		summary += fmt.Sprintf(", removed %d lines", diff.RemovedLines)
	}
	sb.WriteString(dimStyle.Render(summary))
	sb.WriteString("\n")

	// Render the pre-formatted unified diff with colors
	lines := strings.Split(diff.UnifiedDiff, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		// Check if this is a removal or addition line
		// Format is: "   N - content" or "   N + content"
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

// formatToolInput extracts a readable description from tool input JSON
func formatToolInput(toolName, input string) string {
	input = strings.TrimSpace(input)
	if input == "" || input == "{}" {
		return ""
	}

	// Try to extract meaningful info based on tool name
	// Common patterns: {"path": "..."}, {"command": "..."}, {"url": "..."}

	// Simple extraction - find first string value
	if idx := strings.Index(input, `":`); idx != -1 {
		rest := input[idx+2:]
		// Find the value
		if strings.HasPrefix(rest, `"`) || strings.HasPrefix(rest, ` "`) {
			rest = strings.TrimPrefix(rest, " ")
			rest = strings.TrimPrefix(rest, `"`)
			if endIdx := strings.Index(rest, `"`); endIdx != -1 {
				value := rest[:endIdx]
				// Truncate if too long
				if len(value) > 60 {
					value = value[:57] + "..."
				}
				return value
			}
		}
	}

	// Fallback: just truncate the input
	if len(input) > 50 {
		return input[:47] + "..."
	}
	return input
}

// stripEmojis removes common emojis from text
func stripEmojis(text string) string {
	// Common emoji patterns to remove
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

	// Clean up any double spaces left behind
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}

	return result
}

// wrapText wraps text at the specified width
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

// ClearMessages clears all messages from the chat view
func (c *ChatView) ClearMessages() {
	c.messages = []ChatMessage{}
	if c.ready {
		c.viewport.SetContent(c.renderMessages())
	}
}
