package references

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseInput_NoReferences(t *testing.T) {
	result := ParseInput("Hello, how are you?")

	if result.Text != "Hello, how are you?" {
		t.Errorf("Text = %q, want %q", result.Text, "Hello, how are you?")
	}
	if len(result.Files) != 0 {
		t.Errorf("Files count = %d, want 0", len(result.Files))
	}
	if result.HasFiles() {
		t.Error("HasFiles() should be false")
	}
}

func TestParseInput_URLImage(t *testing.T) {
	result := ParseInput("Look at @https://example.com/photo.png")

	if result.Text != "Look at" {
		t.Errorf("Text = %q, want %q", result.Text, "Look at")
	}
	if len(result.Files) != 1 {
		t.Fatalf("Files count = %d, want 1", len(result.Files))
	}
	if result.Files[0].Type != FileTypeImage {
		t.Error("Expected FileTypeImage")
	}
	if result.Files[0].ImageData == nil {
		t.Error("Expected ImageData to be set")
	}
}

func TestParseInput_LocalTextFile(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	content := "package main\n\nfunc main() {}\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := ParseInput("Analyze @" + testFile)

	if len(result.Files) != 1 {
		t.Fatalf("Files count = %d, want 1", len(result.Files))
	}
	if result.Files[0].Type != FileTypeText {
		t.Errorf("Type = %v, want FileTypeText", result.Files[0].Type)
	}
	if result.Files[0].Content != content {
		t.Errorf("Content = %q, want %q", result.Files[0].Content, content)
	}
}

func TestParseInput_LocalImage(t *testing.T) {
	// Create temp PNG file
	tmpDir := t.TempDir()
	testImage := filepath.Join(tmpDir, "test.png")
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE,
	}
	if err := os.WriteFile(testImage, pngData, 0644); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	result := ParseInput("What's in @" + testImage)

	if len(result.Files) != 1 {
		t.Fatalf("Files count = %d, want 1", len(result.Files))
	}
	if result.Files[0].Type != FileTypeImage {
		t.Errorf("Type = %v, want FileTypeImage", result.Files[0].Type)
	}
	if result.Files[0].ImageData == nil {
		t.Error("Expected ImageData to be set")
	}
}

func TestParseInput_FileNotFound(t *testing.T) {
	result := ParseInput("Look at @/nonexistent/file.txt")

	if len(result.Files) != 1 {
		t.Fatalf("Files count = %d, want 1", len(result.Files))
	}
	if result.Files[0].Error == "" {
		t.Error("Expected error for nonexistent file")
	}
	if !result.HasErrors() {
		t.Error("HasErrors() should be true")
	}
}

func TestParseInput_MultipleFiles(t *testing.T) {
	result := ParseInput("Compare @https://example.com/a.jpg and @https://example.com/b.png")

	if len(result.Files) != 2 {
		t.Errorf("Files count = %d, want 2", len(result.Files))
	}
}

func TestParseInput_DuplicateFiles(t *testing.T) {
	result := ParseInput("Look at @https://example.com/photo.jpg @https://example.com/photo.jpg")

	if len(result.Files) != 1 {
		t.Errorf("Files count = %d, want 1 (duplicates removed)", len(result.Files))
	}
}

func TestLoadFileReference_URL(t *testing.T) {
	ref := LoadFileReference("https://example.com/image.png")

	if ref.Type != FileTypeImage {
		t.Errorf("Type = %v, want FileTypeImage", ref.Type)
	}
	if ref.ImageData == nil {
		t.Error("Expected ImageData to be set")
	}
	if ref.Error != "" {
		t.Errorf("Unexpected error: %s", ref.Error)
	}
}

func TestLoadFileReference_TextFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("Hello"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ref := LoadFileReference(testFile)

	if ref.Type != FileTypeText {
		t.Errorf("Type = %v, want FileTypeText", ref.Type)
	}
	if ref.Content != "Hello" {
		t.Errorf("Content = %q, want %q", ref.Content, "Hello")
	}
}

func TestLoadFileReference_BinaryFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.exe")
	if err := os.WriteFile(testFile, []byte{0x00}, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ref := LoadFileReference(testFile)

	if ref.Type != FileTypeBinary {
		t.Errorf("Type = %v, want FileTypeBinary", ref.Type)
	}
	if ref.Error == "" {
		t.Error("Expected error for binary file")
	}
}

func TestLoadFileReference_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	ref := LoadFileReference(tmpDir)

	if ref.Error == "" {
		t.Error("Expected error for directory")
	}
}

func TestParsedInput_GetImages(t *testing.T) {
	// Create temp image
	tmpDir := t.TempDir()
	testImage := filepath.Join(tmpDir, "test.png")
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE,
	}
	if err := os.WriteFile(testImage, pngData, 0644); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	// Create temp text file
	testText := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testText, []byte("Hello"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := ParseInput("Check @" + testImage + " and @" + testText)

	images := result.GetImages()
	if len(images) != 1 {
		t.Errorf("GetImages() count = %d, want 1", len(images))
	}
}

func TestParsedInput_GetTextContents(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := ParseInput("Review @" + testFile)

	contents := result.GetTextContents()
	if len(contents) != 1 {
		t.Fatalf("GetTextContents() count = %d, want 1", len(contents))
	}
	if !strings.Contains(contents[0], "package main") {
		t.Error("Expected content to contain file content")
	}
	if !strings.Contains(contents[0], "<file path=") {
		t.Error("Expected content to be wrapped in <file> tags")
	}
}

func TestParsedInput_CombineTextContent(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("File content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := ParseInput("Analyze this @" + testFile)

	combined := result.CombineTextContent()
	if !strings.Contains(combined, "Analyze this") {
		t.Error("Expected combined to contain original text")
	}
	if !strings.Contains(combined, "File content") {
		t.Error("Expected combined to contain file content")
	}
}

func TestParsedInput_FileIndicators(t *testing.T) {
	tmpDir := t.TempDir()

	// Create image
	testImage := filepath.Join(tmpDir, "photo.png")
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xDE,
	}
	if err := os.WriteFile(testImage, pngData, 0644); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	// Create text file
	testText := filepath.Join(tmpDir, "code.go")
	if err := os.WriteFile(testText, []byte("package main"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := ParseInput("Check @" + testImage + " and @" + testText)

	indicators := result.FileIndicators()
	if len(indicators) != 2 {
		t.Fatalf("FileIndicators() count = %d, want 2", len(indicators))
	}

	hasImage := false
	hasFile := false
	for _, ind := range indicators {
		if strings.Contains(ind, "[image:") {
			hasImage = true
		}
		if strings.Contains(ind, "[file:") {
			hasFile = true
		}
	}
	if !hasImage {
		t.Error("Expected [image:] indicator")
	}
	if !hasFile {
		t.Error("Expected [file:] indicator")
	}
}
