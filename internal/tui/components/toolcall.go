package components

import (
	"fmt"

	"chewbacca/internal/styles"
	"chewbacca/internal/util"

	"github.com/charmbracelet/lipgloss"
)

type ToolCallView struct {
	name       string
	input      string
	result     string
	expanded   bool
	maxPreview int
}

func NewToolCallView(name, input string) *ToolCallView {
	return &ToolCallView{
		name:       name,
		input:      input,
		maxPreview: 100,
	}
}

func (t *ToolCallView) SetResult(result string) {
	t.result = result
}

func (t *ToolCallView) Toggle() {
	t.expanded = !t.expanded
}

func (t *ToolCallView) View() string {
	headerStyle := lipgloss.NewStyle().
		Foreground(styles.ToolColor).
		Bold(true)

	inputStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor)

	resultStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		PaddingLeft(2)

	header := headerStyle.Render(fmt.Sprintf("[%s]", t.name))
	inputPreview := inputStyle.Render(util.TruncateString(t.input, 60))

	var resultView string
	if t.result != "" {
		if t.expanded {
			resultView = resultStyle.Render(t.result)
		} else {
			resultView = resultStyle.Render(util.TruncateString(t.result, t.maxPreview))
		}
	}

	if resultView != "" {
		return fmt.Sprintf("%s %s\n%s", header, inputPreview, resultView)
	}
	return fmt.Sprintf("%s %s", header, inputPreview)
}
