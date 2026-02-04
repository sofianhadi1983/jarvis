// Package image provides functionality for loading and processing images
// for use with the Anthropic API's vision capabilities.
package image

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Supported image formats for the Anthropic API.
var supportedFormats = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
}

// ImageInput represents an image that can be sent to the API.
type ImageInput struct {
	IsURL     bool   // True if this is a URL reference
	URL       string // The URL if IsURL is true
	Data      string // Base64-encoded image data if not a URL
	MediaType string // MIME type of the image
	OrigPath  string // Original path or URL provided by the user
}

// LoadImage loads an image from a file path and returns an ImageInput.
// The image data is base64-encoded for use with the Anthropic API.
func LoadImage(path string) (*ImageInput, error) {
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[2:])
	}

	// Convert to absolute path if relative
	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		path = filepath.Join(wd, path)
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("image file not found: %s", path)
	}

	// Validate format
	ext := strings.ToLower(filepath.Ext(path))
	mediaType, ok := supportedFormats[ext]
	if !ok {
		return nil, fmt.Errorf("unsupported image format: %s (supported: jpg, jpeg, png, gif, webp)", ext)
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read image file: %w", err)
	}

	// Base64 encode
	encoded := base64.StdEncoding.EncodeToString(data)

	return &ImageInput{
		IsURL:     false,
		Data:      encoded,
		MediaType: mediaType,
		OrigPath:  path,
	}, nil
}

// NewURLImage creates an ImageInput from a URL.
func NewURLImage(url string) *ImageInput {
	return &ImageInput{
		IsURL:    true,
		URL:      url,
		OrigPath: url,
	}
}

// IsURL checks if the given string is a URL (starts with http:// or https://).
func IsURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

// IsSupportedFormat checks if the given path has a supported image extension.
func IsSupportedFormat(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	_, ok := supportedFormats[ext]
	return ok
}

// GetMediaType returns the MIME type for a given file extension.
func GetMediaType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if mediaType, ok := supportedFormats[ext]; ok {
		return mediaType
	}
	return ""
}
