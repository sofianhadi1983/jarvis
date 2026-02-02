package components

import (
	"fmt"
	"os"
	"path/filepath"

	"chewbacca/internal/styles"

	"github.com/charmbracelet/lipgloss"
)

type Header struct {
	appName  string
	version  string
	model    string
	workDir  string
	width    int
}

func NewHeader(appName, version, model string) *Header {
	workDir, _ := os.Getwd()
	homeDir, _ := os.UserHomeDir()

	// Convert to relative path with ~ for home directory
	if homeDir != "" {
		if rel, err := filepath.Rel(homeDir, workDir); err == nil && len(rel) < len(workDir) {
			workDir = "~/" + rel
		}
	}

	return &Header{
		appName: appName,
		version: version,
		model:   model,
		workDir: workDir,
	}
}

func (h *Header) SetWidth(width int) {
	h.width = width
}

func (h *Header) View() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(styles.AssistantColor).
		Bold(true)

	versionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	dimStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor)

	title := titleStyle.Render(h.appName)
	version := versionStyle.Render(" " + h.version)
	modelInfo := dimStyle.Render(h.model)
	workDirInfo := dimStyle.Render(h.workDir)

	line1 := fmt.Sprintf("  %s%s", title, version)
	line2 := fmt.Sprintf("  %s", modelInfo)
	line3 := fmt.Sprintf("  %s", workDirInfo)

	return lipgloss.JoinVertical(lipgloss.Left, "", line1, line2, line3, "")
}
