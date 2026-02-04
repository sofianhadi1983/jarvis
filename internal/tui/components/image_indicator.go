package components

import (
	"fmt"
	"strings"

	"jarvis/internal/image"
	"jarvis/internal/styles"

	"github.com/charmbracelet/lipgloss"
)

// ImageIndicator displays pasted images above the input area.
type ImageIndicator struct {
	images        []*image.ImageInput
	selectedIndex int // -1 = no selection
	width         int
}

// NewImageIndicator creates a new ImageIndicator.
func NewImageIndicator() *ImageIndicator {
	return &ImageIndicator{
		images:        []*image.ImageInput{},
		selectedIndex: -1,
	}
}

// SetWidth sets the width for rendering.
func (i *ImageIndicator) SetWidth(width int) {
	i.width = width
}

// AddImage adds an image and returns its index (1-based for display).
func (i *ImageIndicator) AddImage(img *image.ImageInput) int {
	i.images = append(i.images, img)
	return len(i.images)
}

// RemoveSelected removes the currently selected image.
// Selection moves to the previous image, or deselects if none remain.
func (i *ImageIndicator) RemoveSelected() {
	if i.selectedIndex < 0 || i.selectedIndex >= len(i.images) {
		return
	}

	// Remove the image at selectedIndex
	i.images = append(i.images[:i.selectedIndex], i.images[i.selectedIndex+1:]...)

	// Adjust selection
	if len(i.images) == 0 {
		i.selectedIndex = -1
	} else if i.selectedIndex >= len(i.images) {
		i.selectedIndex = len(i.images) - 1
	}
}

// Clear removes all images and deselects.
func (i *ImageIndicator) Clear() {
	i.images = []*image.ImageInput{}
	i.selectedIndex = -1
}

// Images returns all images.
func (i *ImageIndicator) Images() []*image.ImageInput {
	return i.images
}

// HasImages returns true if there are any images.
func (i *ImageIndicator) HasImages() bool {
	return len(i.images) > 0
}

// Count returns the number of images.
func (i *ImageIndicator) Count() int {
	return len(i.images)
}

// IsSelected returns true if an image is currently selected.
func (i *ImageIndicator) IsSelected() bool {
	return i.selectedIndex >= 0
}

// Select selects an image by index (0-based).
func (i *ImageIndicator) Select(index int) {
	if index >= 0 && index < len(i.images) {
		i.selectedIndex = index
	}
}

// SelectLast selects the last image.
func (i *ImageIndicator) SelectLast() {
	if len(i.images) > 0 {
		i.selectedIndex = len(i.images) - 1
	}
}

// SelectPrev moves selection to the previous image.
func (i *ImageIndicator) SelectPrev() {
	if i.selectedIndex > 0 {
		i.selectedIndex--
	}
}

// SelectNext moves selection to the next image, or deselects if at end.
func (i *ImageIndicator) SelectNext() {
	if i.selectedIndex < len(i.images)-1 {
		i.selectedIndex++
	} else {
		i.selectedIndex = -1
	}
}

// Deselect clears the selection.
func (i *ImageIndicator) Deselect() {
	i.selectedIndex = -1
}

// View renders the image indicator.
func (i *ImageIndicator) View() string {
	if len(i.images) == 0 {
		return ""
	}

	indicatorStyle := lipgloss.NewStyle().
		Foreground(styles.AssistantColor)

	selectedStyle := lipgloss.NewStyle().
		Foreground(styles.AssistantColor).
		Background(lipgloss.Color("236")).
		Bold(true)

	hintStyle := lipgloss.NewStyle().
		Foreground(styles.DimColor)

	var parts []string
	for idx := range i.images {
		label := fmt.Sprintf("[Image #%d]", idx+1)
		if idx == i.selectedIndex {
			parts = append(parts, selectedStyle.Render(label))
		} else {
			parts = append(parts, indicatorStyle.Render(label))
		}
	}

	content := strings.Join(parts, " ")

	// Add hint
	if i.selectedIndex >= 0 {
		content += hintStyle.Render(" (Del to remove)")
	} else {
		content += hintStyle.Render(" (↑ to select)")
	}

	// Add some padding
	return "  " + content
}
