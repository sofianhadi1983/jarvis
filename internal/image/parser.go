package image

import (
	"regexp"
	"strings"
)

// imageRefPattern matches @path references in user input.
// Matches:
// - @https://example.com/image.png (URLs)
// - @./photo.png (relative paths)
// - @/absolute/path/image.jpg (absolute paths)
// - @~/home/image.gif (home-relative paths)
var imageRefPattern = regexp.MustCompile(`@((?:https?://[^\s]+\.(?:jpg|jpeg|png|gif|webp))|(?:[^\s@]+\.(?:jpg|jpeg|png|gif|webp)))`)

// ParsedInput represents the result of parsing user input for image references.
type ParsedInput struct {
	Text   string        // The input text with @references removed
	Images []*ImageInput // Successfully loaded images
	Errors []string      // Error messages for failed image loads
}

// ParseInput extracts @image references from user input, loads the images,
// and returns the cleaned text along with the loaded images.
func ParseInput(input string) *ParsedInput {
	result := &ParsedInput{
		Text:   input,
		Images: []*ImageInput{},
		Errors: []string{},
	}

	// Find all matches
	matches := imageRefPattern.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return result
	}

	// Track processed references to avoid duplicates
	processed := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		fullMatch := match[0] // e.g., "@./photo.png"
		ref := match[1]       // e.g., "./photo.png"

		// Skip duplicates
		if processed[ref] {
			continue
		}
		processed[ref] = true

		// Remove the reference from text
		result.Text = strings.Replace(result.Text, fullMatch, "", 1)

		// Load the image
		if IsURL(ref) {
			result.Images = append(result.Images, NewURLImage(ref))
		} else {
			img, err := LoadImage(ref)
			if err != nil {
				result.Errors = append(result.Errors, err.Error())
				continue
			}
			result.Images = append(result.Images, img)
		}
	}

	// Clean up extra whitespace in the text
	result.Text = strings.TrimSpace(result.Text)
	result.Text = regexp.MustCompile(`\s+`).ReplaceAllString(result.Text, " ")

	return result
}

// HasImages returns true if the parsed input contains any images.
func (p *ParsedInput) HasImages() bool {
	return len(p.Images) > 0
}

// HasErrors returns true if there were any errors loading images.
func (p *ParsedInput) HasErrors() bool {
	return len(p.Errors) > 0
}

// ImageIndicators returns a slice of display strings for the loaded images.
// For example: "[image: ./photo.png]"
func (p *ParsedInput) ImageIndicators() []string {
	indicators := make([]string, len(p.Images))
	for i, img := range p.Images {
		indicators[i] = "[image: " + img.OrigPath + "]"
	}
	return indicators
}
