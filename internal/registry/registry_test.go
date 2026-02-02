package registry

import (
	"encoding/json"
	"testing"

	"jarvis/pkg/tools"
)

func TestRegister(t *testing.T) {
	reg := NewRegistry()

	def := tools.ToolDefinition{
		Name:        "TestTool",
		Description: "A test tool",
		Function: func(input json.RawMessage) (string, error) {
			return "result", nil
		},
	}

	err := reg.Register(def)
	if err != nil {
		t.Errorf("Register failed: %v", err)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	reg := NewRegistry()

	def := tools.ToolDefinition{
		Name:        "TestTool",
		Description: "A test tool",
		Function: func(input json.RawMessage) (string, error) {
			return "result", nil
		},
	}

	reg.Register(def)
	err := reg.Register(def)

	if err == nil {
		t.Error("Expected error for duplicate registration, got nil")
	}
}

func TestGet(t *testing.T) {
	reg := NewRegistry()

	def := tools.ToolDefinition{
		Name:        "TestTool",
		Description: "A test tool",
		Function: func(input json.RawMessage) (string, error) {
			return "result", nil
		},
	}

	reg.Register(def)

	retrieved, found := reg.Get("TestTool")
	if !found {
		t.Error("Expected to find registered tool")
	}
	if retrieved.Name != "TestTool" {
		t.Errorf("Expected name 'TestTool', got '%s'", retrieved.Name)
	}
}

func TestGetNotFound(t *testing.T) {
	reg := NewRegistry()

	_, found := reg.Get("NonExistent")
	if found {
		t.Error("Expected not to find non-existent tool")
	}
}

func TestToAnthropicTools(t *testing.T) {
	reg := NewRegistry()

	def := tools.ToolDefinition{
		Name:        "TestTool",
		Description: "A test tool",
		Function: func(input json.RawMessage) (string, error) {
			return "result", nil
		},
	}

	reg.Register(def)

	anthropicTools := reg.ToAnthropicTools()
	if len(anthropicTools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(anthropicTools))
	}
}
