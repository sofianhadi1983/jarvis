package subagent

import (
	"context"
	"fmt"
	"strings"

	"jarvis/internal/registry"
	"jarvis/internal/types"
)

type TaskInput struct {
	Description string `json:"description" jsonschema_description:"Short task name (3-5 words)"`
	Prompt      string `json:"prompt" jsonschema_description:"Detailed instructions for the subagent"`
	AgentType   string `json:"agent_type" jsonschema_description:"Agent type: explore, code, or plan"`
}

// MessageSender is implemented by agents that can process a prompt and call back with messages.
type MessageSender interface {
	SendMessage(ctx context.Context, input string, callback func(msg any)) error
}

// ChildFactory creates a child agent with the given registry and system prompt.
type ChildFactory func(reg *registry.Registry, systemPrompt string) MessageSender

var agentAllowedTools = map[string][]string{
	"explore": {"Read", "ListFiles", "Bash"},
	"code":    {"Read", "ListFiles", "Update", "Bash", "Fetch"},
	"plan":    {"Read", "ListFiles", "Bash"},
}

var agentSystemPrompts = map[string]string{
	"explore": "You are an exploration agent. Analyze code, search files, and report findings. " +
		"You cannot modify files. Provide a clear, concise summary of what you find.",
	"code": "You are a coding agent. Implement changes as instructed. " +
		"Read relevant code first, then make precise edits. Summarize what you changed.",
	"plan": "You are a planning agent. Analyze the codebase and design strategies. " +
		"You cannot modify files. Provide a structured plan with clear steps.",
}

func RunTask(ctx context.Context, parentRegistry *registry.Registry,
	input TaskInput, parentCallback func(msg any), newChild ChildFactory) (string, error) {

	allowedTools, ok := agentAllowedTools[input.AgentType]
	if !ok {
		return "", fmt.Errorf("unknown agent_type %q: must be explore, code, or plan", input.AgentType)
	}

	childReg := registry.NewRegistry()
	for _, toolName := range allowedTools {
		if toolDef, found := parentRegistry.Get(toolName); found {
			childReg.RegisterOrReplace(toolDef)
		}
	}

	systemPrompt := agentSystemPrompts[input.AgentType]
	systemPrompt += "\nWorking directory: use relative paths from the project root."

	child := newChild(childReg, systemPrompt)

	var result strings.Builder
	prefix := fmt.Sprintf("[%s] ", input.AgentType)

	childCallback := func(msg any) {
		switch m := msg.(type) {
		case types.StreamChunkMsg:
			result.WriteString(m.Chunk)
		case types.ToolStartMsg:
			result.Reset()
			if parentCallback != nil {
				parentCallback(types.ToolStartMsg{Name: prefix + m.Name})
			}
		}
	}

	err := child.SendMessage(ctx, input.Prompt, childCallback)
	if err != nil {
		if result.Len() > 0 {
			return result.String(), nil
		}
		return "", fmt.Errorf("subagent error: %w", err)
	}

	if result.Len() == 0 {
		return "Subagent completed but produced no output.", nil
	}

	return result.String(), nil
}
