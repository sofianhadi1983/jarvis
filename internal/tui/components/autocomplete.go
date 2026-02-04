package components

import (
	"fmt"
	"strings"

	"jarvis/internal/files"
	"jarvis/internal/styles"

	"github.com/charmbracelet/lipgloss"
)

const (
	maxVisibleItems = 8
	minWidth        = 30
)

type Autocomplete struct {
	visible       bool
	scanner       *files.Scanner
	allFiles      []files.FileEntry
	filtered      []files.FileEntry
	selectedIndex int
	query         string
	atPosition    int
	width         int
	scrollOffset  int
}

func NewAutocomplete() *Autocomplete {
	return &Autocomplete{
		scanner:       files.NewScanner(),
		selectedIndex: 0,
	}
}

func (a *Autocomplete) Show(atPosition int) {
	a.visible = true
	a.atPosition = atPosition
	a.selectedIndex = 0
	a.scrollOffset = 0
	a.query = ""

	entries, err := a.scanner.Scan()
	if err != nil {
		a.allFiles = []files.FileEntry{}
	} else {
		a.allFiles = entries
	}
	a.filtered = a.allFiles
}

func (a *Autocomplete) Hide() {
	a.visible = false
	a.query = ""
	a.selectedIndex = 0
	a.scrollOffset = 0
}

func (a *Autocomplete) IsVisible() bool {
	return a.visible
}

func (a *Autocomplete) SetQuery(query string) {
	if a.query == query {
		return
	}
	a.query = query
	a.filtered = a.scanner.Filter(a.allFiles, query)
	a.selectedIndex = 0
	a.scrollOffset = 0
}

func (a *Autocomplete) GetQuery() string {
	return a.query
}

func (a *Autocomplete) GetAtPosition() int {
	return a.atPosition
}

func (a *Autocomplete) SelectNext() {
	if len(a.filtered) == 0 {
		return
	}
	a.selectedIndex++
	if a.selectedIndex >= len(a.filtered) {
		a.selectedIndex = 0
		a.scrollOffset = 0
	}
	a.adjustScroll()
}

func (a *Autocomplete) SelectPrev() {
	if len(a.filtered) == 0 {
		return
	}
	a.selectedIndex--
	if a.selectedIndex < 0 {
		a.selectedIndex = len(a.filtered) - 1
		a.scrollOffset = max(0, len(a.filtered)-maxVisibleItems)
	}
	a.adjustScroll()
}

func (a *Autocomplete) adjustScroll() {
	if a.selectedIndex < a.scrollOffset {
		a.scrollOffset = a.selectedIndex
	} else if a.selectedIndex >= a.scrollOffset+maxVisibleItems {
		a.scrollOffset = a.selectedIndex - maxVisibleItems + 1
	}
}

func (a *Autocomplete) GetSelected() *files.FileEntry {
	if len(a.filtered) == 0 || a.selectedIndex < 0 || a.selectedIndex >= len(a.filtered) {
		return nil
	}
	entry := a.filtered[a.selectedIndex]
	return &entry
}

func (a *Autocomplete) GetSelectedPath() string {
	entry := a.GetSelected()
	if entry == nil {
		return ""
	}
	path := entry.Path
	if entry.IsDir {
		path += "/"
	}
	return path
}

func (a *Autocomplete) HasItems() bool {
	return len(a.filtered) > 0
}

func (a *Autocomplete) SetWidth(width int) {
	a.width = width
}

func (a *Autocomplete) RefreshFiles() {
	a.scanner.InvalidateCache()
	entries, err := a.scanner.Scan()
	if err != nil {
		a.allFiles = []files.FileEntry{}
	} else {
		a.allFiles = entries
	}
	a.filtered = a.scanner.Filter(a.allFiles, a.query)
}

func (a *Autocomplete) View() string {
	if !a.visible {
		return ""
	}

	if len(a.filtered) == 0 {
		noMatchStyle := lipgloss.NewStyle().
			Foreground(styles.DimColor).
			Italic(true)
		return "  " + noMatchStyle.Render("No matching files")
	}

	visibleCount := min(maxVisibleItems, len(a.filtered))
	endIdx := min(a.scrollOffset+visibleCount, len(a.filtered))

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("255")).
		Background(lipgloss.Color("62")).
		Bold(true)

	dimStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor)

	maxEntryWidth := minWidth
	for i := a.scrollOffset; i < endIdx; i++ {
		entry := a.filtered[i]
		icon := files.GetFileIcon(entry)
		line := fmt.Sprintf(" %s %s ", icon, entry.Path)
		if len(line) > maxEntryWidth {
			maxEntryWidth = len(line)
		}
	}

	if a.width > 0 && maxEntryWidth > a.width-6 {
		maxEntryWidth = a.width - 6
	}

	var lines []string

	if a.scrollOffset > 0 {
		lines = append(lines, dimStyle.Render(fmt.Sprintf("  ^ %d more", a.scrollOffset)))
	}

	for i := a.scrollOffset; i < endIdx; i++ {
		entry := a.filtered[i]
		icon := files.GetFileIcon(entry)
		displayPath := entry.Path
		if entry.IsDir {
			displayPath += "/"
		}

		maxPathLen := maxEntryWidth - 6
		if len(displayPath) > maxPathLen {
			displayPath = "..." + displayPath[len(displayPath)-maxPathLen+3:]
		}

		line := fmt.Sprintf(" %s %s", icon, displayPath)
		padding := maxEntryWidth - len(line)
		if padding > 0 {
			line += strings.Repeat(" ", padding)
		}

		if i == a.selectedIndex {
			lines = append(lines, selectedStyle.Render(line))
		} else {
			lines = append(lines, normalStyle.Render(line))
		}
	}

	remaining := len(a.filtered) - endIdx
	if remaining > 0 {
		lines = append(lines, dimStyle.Render(fmt.Sprintf("  v %d more", remaining)))
	}

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.BorderColor)

	content := strings.Join(lines, "\n")
	boxed := borderStyle.Render(content)

	hintStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor)
	hint := hintStyle.Render("  Up/Down navigate | Tab select | Esc cancel")

	return boxed + "\n" + hint
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
