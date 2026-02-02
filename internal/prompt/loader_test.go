package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSinglePrompt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prompts")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	content := "Test prompt content"
	if err := os.WriteFile(filepath.Join(tmpDir, "test.md"), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	loader := NewLoader(tmpDir)
	result, err := loader.LoadSinglePrompt("test.md")

	if err != nil {
		t.Errorf("LoadSinglePrompt failed: %v", err)
	}
	if result != content {
		t.Errorf("Expected '%s', got '%s'", content, result)
	}
}

func TestLoadSinglePrompt_NotFound(t *testing.T) {
	loader := NewLoader("./nonexistent")
	_, err := loader.LoadSinglePrompt("missing.md")

	if err == nil {
		t.Error("Expected error for missing file, got nil")
	}
}

func TestProcessIncludes(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prompts")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	includedContent := "Included content"
	mainContent := "Main {{ include \"included.md\" }} end"
	expected := "Main Included content end"

	os.WriteFile(filepath.Join(tmpDir, "included.md"), []byte(includedContent), 0644)
	os.WriteFile(filepath.Join(tmpDir, "main.md"), []byte(mainContent), 0644)

	loader := NewLoader(tmpDir)
	result, err := loader.LoadSinglePrompt("main.md")

	if err != nil {
		t.Errorf("LoadSinglePrompt failed: %v", err)
	}
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestCircularIncludeDetection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "prompts")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "a.md"), []byte("A {{ include \"b.md\" }}"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.md"), []byte("B {{ include \"a.md\" }}"), 0644)

	loader := NewLoader(tmpDir)
	_, err = loader.LoadSinglePrompt("a.md")

	if err == nil {
		t.Error("Expected circular dependency error, got nil")
	}
	if !strings.Contains(err.Error(), "circular") {
		t.Errorf("Expected error to mention 'circular', got: %v", err)
	}
}
