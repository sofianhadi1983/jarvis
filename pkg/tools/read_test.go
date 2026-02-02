package tools

import (
	"encoding/json"
	"os"
	"testing"
)

func TestReadFile_HappyPath(t *testing.T) {
	tmpfile, err := os.CreateTemp("", "test*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	content := "Hello, World!"
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpfile.Close()

	input, _ := json.Marshal(ReadFileInput{Path: tmpfile.Name()})
	result, err := ReadFile(input)

	if err != nil {
		t.Errorf("ReadFile failed: %v", err)
	}
	if result != content {
		t.Errorf("Expected '%s', got '%s'", content, result)
	}
}

func TestReadFile_NonExistent(t *testing.T) {
	input, _ := json.Marshal(ReadFileInput{Path: "/nonexistent/path/file.txt"})
	_, err := ReadFile(input)

	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

func TestReadFile_InvalidJSON(t *testing.T) {
	_, err := ReadFile([]byte("invalid json"))

	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}
