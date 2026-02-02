package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBash_HappyPath(t *testing.T) {
	input, _ := json.Marshal(BashInput{
		Command:     "echo hello",
		Description: "Test echo command",
	})

	result, err := Bash(input)
	if err != nil {
		t.Fatalf("Bash failed: %v", err)
	}

	var bashResult BashResult
	if err := json.Unmarshal([]byte(result), &bashResult); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if bashResult.ReturnCode != 0 {
		t.Errorf("Expected return code 0, got %d", bashResult.ReturnCode)
	}

	if !strings.Contains(bashResult.Stdout, "hello") {
		t.Errorf("Expected stdout to contain 'hello', got '%s'", bashResult.Stdout)
	}
}

func TestBash_EmptyCommand(t *testing.T) {
	input, _ := json.Marshal(BashInput{
		Command: "",
	})

	_, err := Bash(input)
	if err == nil {
		t.Error("Expected error for empty command, got nil")
	}
}

func TestBash_NonZeroExit(t *testing.T) {
	input, _ := json.Marshal(BashInput{
		Command:     "exit 1",
		Description: "Test non-zero exit",
	})

	result, err := Bash(input)
	if err != nil {
		t.Fatalf("Bash returned error: %v", err)
	}

	var bashResult BashResult
	if err := json.Unmarshal([]byte(result), &bashResult); err != nil {
		t.Fatalf("Failed to parse result: %v", err)
	}

	if bashResult.ReturnCode != 1 {
		t.Errorf("Expected return code 1, got %d", bashResult.ReturnCode)
	}
}
