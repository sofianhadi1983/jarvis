package image

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"https URL", "https://example.com/image.png", true},
		{"http URL", "http://example.com/image.jpg", true},
		{"relative path", "./photo.png", false},
		{"absolute path", "/usr/local/image.gif", false},
		{"home path", "~/images/photo.webp", false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsURL(tt.input)
			if result != tt.expected {
				t.Errorf("IsURL(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsSupportedFormat(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"jpg", "image.jpg", true},
		{"jpeg", "photo.jpeg", true},
		{"png", "screenshot.png", true},
		{"gif", "animation.gif", true},
		{"webp", "modern.webp", true},
		{"JPG uppercase", "IMAGE.JPG", true},
		{"PNG uppercase", "SCREENSHOT.PNG", true},
		{"bmp unsupported", "bitmap.bmp", false},
		{"tiff unsupported", "photo.tiff", false},
		{"no extension", "noextension", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSupportedFormat(tt.path)
			if result != tt.expected {
				t.Errorf("IsSupportedFormat(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetMediaType(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"jpg", "image.jpg", "image/jpeg"},
		{"jpeg", "photo.jpeg", "image/jpeg"},
		{"png", "screenshot.png", "image/png"},
		{"gif", "animation.gif", "image/gif"},
		{"webp", "modern.webp", "image/webp"},
		{"unsupported", "file.bmp", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMediaType(tt.path)
			if result != tt.expected {
				t.Errorf("GetMediaType(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestLoadImage(t *testing.T) {
	// Create a temporary test image
	tmpDir := t.TempDir()
	testImagePath := filepath.Join(tmpDir, "test.png")

	// Write minimal valid PNG data
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, // 1x1 image
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE, // color type, compression
		0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, 0x54, // IDAT chunk
		0x08, 0xD7, 0x63, 0xF8, 0xFF, 0xFF, 0x3F, 0x00, // compressed data
		0x05, 0xFE, 0x02, 0xFE, 0xDC, 0xCC, 0x59, 0xE7, // checksum
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, // IEND chunk
		0xAE, 0x42, 0x60, 0x82, // CRC
	}
	if err := os.WriteFile(testImagePath, pngData, 0644); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	t.Run("load valid image", func(t *testing.T) {
		img, err := LoadImage(testImagePath)
		if err != nil {
			t.Fatalf("LoadImage() error = %v", err)
		}
		if img.IsURL {
			t.Error("Expected IsURL to be false")
		}
		if img.MediaType != "image/png" {
			t.Errorf("MediaType = %q, want %q", img.MediaType, "image/png")
		}
		if img.Data == "" {
			t.Error("Expected Data to be non-empty")
		}
		if img.OrigPath != testImagePath {
			t.Errorf("OrigPath = %q, want %q", img.OrigPath, testImagePath)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadImage("/nonexistent/path/image.png")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("unsupported format", func(t *testing.T) {
		unsupportedPath := filepath.Join(tmpDir, "image.bmp")
		if err := os.WriteFile(unsupportedPath, []byte("fake bmp"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		_, err := LoadImage(unsupportedPath)
		if err == nil {
			t.Error("Expected error for unsupported format")
		}
	})
}

func TestNewURLImage(t *testing.T) {
	url := "https://example.com/image.png"
	img := NewURLImage(url)

	if !img.IsURL {
		t.Error("Expected IsURL to be true")
	}
	if img.URL != url {
		t.Errorf("URL = %q, want %q", img.URL, url)
	}
	if img.OrigPath != url {
		t.Errorf("OrigPath = %q, want %q", img.OrigPath, url)
	}
}

func TestParseInput(t *testing.T) {
	// Create a temporary test image for parsing tests
	tmpDir := t.TempDir()
	testImagePath := filepath.Join(tmpDir, "photo.png")
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE,
		0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, 0x54,
		0x08, 0xD7, 0x63, 0xF8, 0xFF, 0xFF, 0x3F, 0x00,
		0x05, 0xFE, 0x02, 0xFE, 0xDC, 0xCC, 0x59, 0xE7,
		0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44,
		0xAE, 0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(testImagePath, pngData, 0644); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	t.Run("no images", func(t *testing.T) {
		result := ParseInput("Hello, how are you?")
		if result.Text != "Hello, how are you?" {
			t.Errorf("Text = %q, want %q", result.Text, "Hello, how are you?")
		}
		if len(result.Images) != 0 {
			t.Errorf("Images count = %d, want 0", len(result.Images))
		}
		if result.HasImages() {
			t.Error("HasImages() should be false")
		}
	})

	t.Run("URL image", func(t *testing.T) {
		result := ParseInput("What is in @https://example.com/photo.png")
		if result.Text != "What is in" {
			t.Errorf("Text = %q, want %q", result.Text, "What is in")
		}
		if len(result.Images) != 1 {
			t.Fatalf("Images count = %d, want 1", len(result.Images))
		}
		if !result.Images[0].IsURL {
			t.Error("Expected IsURL to be true")
		}
		if result.Images[0].URL != "https://example.com/photo.png" {
			t.Errorf("URL = %q, want %q", result.Images[0].URL, "https://example.com/photo.png")
		}
	})

	t.Run("local file", func(t *testing.T) {
		result := ParseInput("Analyze @" + testImagePath)
		if len(result.Images) != 1 {
			t.Fatalf("Images count = %d, want 1", len(result.Images))
		}
		if result.Images[0].IsURL {
			t.Error("Expected IsURL to be false")
		}
		if !result.HasImages() {
			t.Error("HasImages() should be true")
		}
	})

	t.Run("file not found error", func(t *testing.T) {
		result := ParseInput("Look at @/nonexistent/image.png")
		if len(result.Errors) != 1 {
			t.Errorf("Errors count = %d, want 1", len(result.Errors))
		}
		if !result.HasErrors() {
			t.Error("HasErrors() should be true")
		}
	})

	t.Run("multiple images", func(t *testing.T) {
		result := ParseInput("Compare @https://example.com/a.jpg and @https://example.com/b.png")
		if len(result.Images) != 2 {
			t.Errorf("Images count = %d, want 2", len(result.Images))
		}
	})

	t.Run("duplicate images", func(t *testing.T) {
		result := ParseInput("Look at @https://example.com/photo.jpg @https://example.com/photo.jpg")
		if len(result.Images) != 1 {
			t.Errorf("Images count = %d, want 1 (duplicates removed)", len(result.Images))
		}
	})
}

func TestImageIndicators(t *testing.T) {
	parsed := &ParsedInput{
		Images: []*ImageInput{
			{OrigPath: "./photo.png"},
			{OrigPath: "https://example.com/image.jpg"},
		},
	}

	indicators := parsed.ImageIndicators()
	if len(indicators) != 2 {
		t.Fatalf("Indicators count = %d, want 2", len(indicators))
	}
	if indicators[0] != "[image: ./photo.png]" {
		t.Errorf("indicators[0] = %q, want %q", indicators[0], "[image: ./photo.png]")
	}
	if indicators[1] != "[image: https://example.com/image.jpg]" {
		t.Errorf("indicators[1] = %q, want %q", indicators[1], "[image: https://example.com/image.jpg]")
	}
}

func TestDetectMediaType(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name: "PNG",
			data: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A},
			expected: "image/png",
		},
		{
			name: "JPEG",
			data: []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46},
			expected: "image/jpeg",
		},
		{
			name: "GIF",
			data: []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61, 0x00, 0x00},
			expected: "image/gif",
		},
		{
			name: "WebP",
			data: []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50},
			expected: "image/webp",
		},
		{
			name: "unknown format",
			data: []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			expected: "",
		},
		{
			name: "too short",
			data: []byte{0x89, 0x50},
			expected: "",
		},
		{
			name: "empty",
			data: []byte{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectMediaType(tt.data)
			if result != tt.expected {
				t.Errorf("detectMediaType() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNewFromBytes(t *testing.T) {
	// Valid PNG data (minimal header)
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE,
	}

	t.Run("valid PNG", func(t *testing.T) {
		img, err := NewFromBytes(pngData, "Pasted Image #1")
		if err != nil {
			t.Fatalf("NewFromBytes() error = %v", err)
		}
		if img.IsURL {
			t.Error("Expected IsURL to be false")
		}
		if img.MediaType != "image/png" {
			t.Errorf("MediaType = %q, want %q", img.MediaType, "image/png")
		}
		if img.Data == "" {
			t.Error("Expected Data to be non-empty (base64 encoded)")
		}
		if img.OrigPath != "Pasted Image #1" {
			t.Errorf("OrigPath = %q, want %q", img.OrigPath, "Pasted Image #1")
		}
	})

	t.Run("valid JPEG", func(t *testing.T) {
		jpegData := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46}
		img, err := NewFromBytes(jpegData, "Pasted Image #2")
		if err != nil {
			t.Fatalf("NewFromBytes() error = %v", err)
		}
		if img.MediaType != "image/jpeg" {
			t.Errorf("MediaType = %q, want %q", img.MediaType, "image/jpeg")
		}
	})

	t.Run("empty data", func(t *testing.T) {
		_, err := NewFromBytes([]byte{}, "Empty")
		if err == nil {
			t.Error("Expected error for empty data")
		}
	})

	t.Run("unsupported format", func(t *testing.T) {
		badData := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		_, err := NewFromBytes(badData, "Unknown")
		if err == nil {
			t.Error("Expected error for unsupported format")
		}
	})
}
