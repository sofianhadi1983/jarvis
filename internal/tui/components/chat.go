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

	bulletStyle := lipgloss.NewStyle().Foreground(styles.DimColor)
	userBulletStyle := lipgloss.NewStyle().Foreground(styles.UserColor)
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

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
			sb.WriteString(c.renderToolMessage(msg))

		case RoleSystem:
			sb.WriteString(bulletStyle.Render("* "))
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(msg.Content))
			sb.WriteString("\n\n")
		}
	}

	return sb.String()
}

func (c *ChatView) renderToolMessage(msg ChatMessage) string {
	var sb strings.Builder

	bulletStyle := lipgloss.NewStyle().Foreground(styles.ToolColor)
	headerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	pathStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	summaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	toolDesc := formatToolInput(msg.ToolName, msg.ToolInput)

	sb.WriteString(bulletStyle.Render("* "))
	sb.WriteString(headerStyle.Render(msg.ToolName))
	if toolDesc != "" {
		sb.WriteString(pathStyle.Render("(" + toolDesc + ")"))
	}
	sb.WriteString("\n")

	if msg.Diff != nil && len(msg.Diff.Lines) > 0 {
		var summaryParts []string
		if msg.Diff.RemovedLines > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("Removed %d lines", msg.Diff.RemovedLines))
		}
		if msg.Diff.AddedLines > 0 {
			summaryParts = append(summaryParts, fmt.Sprintf("Added %d lines", msg.Diff.AddedLines))
		}
		if len(summaryParts) > 0 {
			sb.WriteString("  L ")
			sb.WriteString(summaryStyle.Render(strings.Join(summaryParts, ", ")))
			sb.WriteString("\n")
		}

		sb.WriteString(c.renderDiffLines(msg.Diff))
	} else if msg.Diff != nil && msg.Diff.UnifiedDiff != "" {
		sb.WriteString(c.renderUnifiedDiff(msg.Diff))
	} else if msg.Content != "" {
		resultStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
		resultPreview := util.TruncateString(msg.Content, 100)
		sb.WriteString("  L ")
		sb.WriteString(resultStyle.Render(resultPreview))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	return sb.String()
}

func (c *ChatView) renderDiffLines(diff *types.DiffInfo) string {
	var sb strings.Builder

	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	removedLineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	addedLineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	removedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	skipStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	maxLineNum := 0
	for _, line := range diff.Lines {
		if line.OldLineNo > maxLineNum {
			maxLineNum = line.OldLineNo
		}
		if line.NewLineNo > maxLineNum {
			maxLineNum = line.NewLineNo
		}
	}
	lineNumWidth := len(fmt.Sprintf("%d", maxLineNum))
	if lineNumWidth < 3 {
		lineNumWidth = 3
	}

	for _, line := range diff.Lines {
		switch line.Type {
		case types.DiffLineRemoved:
			lineNum := fmt.Sprintf("%*d", lineNumWidth, line.OldLineNo)
			sb.WriteString("    ")
			sb.WriteString(removedLineNumStyle.Render(lineNum + " -"))
			sb.WriteString(removedStyle.Render(" " + line.Content))
			sb.WriteString("\n")

		case types.DiffLineAdded:
			lineNum := fmt.Sprintf("%*d", lineNumWidth, line.NewLineNo)
			sb.WriteString("    ")
			sb.WriteString(addedLineNumStyle.Render(lineNum + "  "))
			sb.WriteString(addedStyle.Render(" " + line.Content))
			sb.WriteString("\n")

		case types.DiffLineContext:
			lineNum := fmt.Sprintf("%*d", lineNumWidth, line.NewLineNo)
			sb.WriteString("    ")
			sb.WriteString(lineNumStyle.Render(lineNum + "  "))
			sb.WriteString(contextStyle.Render(" " + line.Content))
			sb.WriteString("\n")

		case types.DiffLineSkip:
			sb.WriteString("    ")
			sb.WriteString(skipStyle.Render("..."))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (c *ChatView) renderUnifiedDiff(diff *types.DiffInfo) string {
	var sb strings.Builder

	summaryStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	removedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	addedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))

	var summaryParts []string
	if diff.AddedLines > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("Added %d lines", diff.AddedLines))
	}
	if diff.RemovedLines > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("removed %d lines", diff.RemovedLines))
	}
	if len(summaryParts) > 0 {
		sb.WriteString("  L ")
		sb.WriteString(summaryStyle.Render(strings.Join(summaryParts, ", ")))
		sb.WriteString("\n")
	}

	lines := strings.Split(diff.UnifiedDiff, "\n")
	lineNum := diff.StartLine
	if lineNum == 0 {
		lineNum = 1
	}

	for _, line := range lines {
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "@@") {
			continue
		}

		lineNumStr := fmt.Sprintf("%4d", lineNum)

		if strings.HasPrefix(line, "-") {
			sb.WriteString("    ")
			sb.WriteString(lineNumStyle.Render(lineNumStr + " -"))
			sb.WriteString(removedStyle.Render(" " + strings.TrimPrefix(line, "-")))
			sb.WriteString("\n")
		} else if strings.HasPrefix(line, "+") {
			sb.WriteString("    ")
			sb.WriteString(lineNumStyle.Render(lineNumStr + "  "))
			sb.WriteString(addedStyle.Render(" " + strings.TrimPrefix(line, "+")))
			sb.WriteString("\n")
			lineNum++
		} else {
			sb.WriteString("    ")
			sb.WriteString(lineNumStyle.Render(lineNumStr + "  "))
			sb.WriteString(contextStyle.Render(" " + strings.TrimPrefix(line, " ")))
			sb.WriteString("\n")
			lineNum++
		}
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
		"~@~X", "~0~_", "~Z~@", "~G~A", "~B~X", "~P~'", "~J~O~>", "~O~]", "~O~A", "~O~B",
		"~E~U", "~N~T", "~J~O~'", "~G~[", "~V~V", "~G~Z", "~O~[", "~O~T", "~O~J", "~O~N",
		"~G~^", "~O~@", "~G~_", "~W~B~>", "~W~C~>", "~W~D~>", "~O~K", "~O~L", "~O~M", "~V~W~>",
		"~P~Q", "~W~]~>", "~P~R", "~P~S", "~W~[~>", "~J~O~T", "~J~O~X", "~J~O~W", "~P~^", "~W~U~>",
		"~J~G", "~P~U", "~G~U", "~E~V~>", "~N~W~>", "~W~O", "~J~K", "~W~H", "~H~\\~>", "~W~I",
		"~V~I", "~V~J", "~V~K", "~V~L", "~V~P", "~W~G", "~W~J", "~W~K", "~V~E", "~V~N~>",
		"~O~H", "~O~I", "~O~L", "~O~A", "~O~F", "~O~E", "~W~Y", "~O~X", "~W~F", "~V~@",
		"~W~P", "~W~Q", "~W~R", "~W~S", "~W~T", "~W~E", "~W~C", "~W~D", "~W~A", "~V~D",
		"~P~P", "~P~Q", "~P~U", "~P~V", "~P~W", "~P~X", "~P~T", "~P~S", "~P~C", "~P~\\",
		"~P~Z", "~P~V", "~P~W", "~P~X", "~P~Y", "~P~[", "~P~T", "~P~R", "~P~\\", "~P~^",
		"~P~J", "~J~O~U", "~W~P~P", "~W~Q~R", "~J~O~V", "~P~I", "~W~\\~>", "~J~O~Y", "~J~O~Z",
		"~P~Y", "~P~Z", "~W~]", "~U~[", "~W~^", "~P~_", "~W~_", "~V~B", "~V~C", "~P~]",
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
