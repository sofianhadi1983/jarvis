package registry

import (
	"fmt"

	"jarvis/pkg/tools"

	"github.com/sofianhadi1983/anthropic-sdk-go"
)

type Registry struct {
	tools map[string]tools.ToolDefinition
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]tools.ToolDefinition),
	}
}

func (r *Registry) Register(def tools.ToolDefinition) error {
	if _, exists := r.tools[def.Name]; exists {
		return fmt.Errorf("tool %s already registered", def.Name)
	}
	r.tools[def.Name] = def
	return nil
}

func (r *Registry) Get(name string) (tools.ToolDefinition, bool) {
	tool, found := r.tools[name]
	return tool, found
}

func (r *Registry) ToAnthropicTools() []anthropic.ToolUnionParam {
	anthropicTools := []anthropic.ToolUnionParam{}
	for _, tool := range r.tools {
		anthropicTools = append(anthropicTools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: tool.InputSchema,
			},
		})
	}
	return anthropicTools
}

func (r *Registry) List() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}
