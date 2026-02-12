package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	"jarvis/internal/config"
	"jarvis/pkg/tools"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Manager manages connections to multiple MCP servers.
type Manager struct {
	configs  map[string]config.MCPServerConfig
	client   *mcp.Client
	sessions map[string]*mcp.ClientSession
}

// NewManager creates a new MCP manager with the given server configurations.
func NewManager(configs map[string]config.MCPServerConfig) *Manager {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "jarvis",
		Version: "1.0.0",
	}, nil)

	return &Manager{
		configs:  configs,
		client:   client,
		sessions: make(map[string]*mcp.ClientSession),
	}
}

// Connect starts all configured MCP servers and establishes sessions.
// Servers that fail to connect are logged and skipped (graceful degradation).
func (m *Manager) Connect(ctx context.Context) error {
	for name, cfg := range m.configs {
		cmd := exec.Command(cfg.Command, cfg.Args...)

		// Set environment variables
		if len(cfg.Env) > 0 {
			cmd.Env = os.Environ()
			for k, v := range cfg.Env {
				cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
			}
		}

		transport := &mcp.CommandTransport{
			Command: cmd,
		}

		session, err := m.client.Connect(ctx, transport, nil)
		if err != nil {
			log.Printf("Warning: failed to connect to MCP server %q: %v", name, err)
			continue
		}

		m.sessions[name] = session
		log.Printf("Connected to MCP server %q", name)
	}

	return nil
}

// GetTools discovers tools from all connected MCP servers and returns them
// as Jarvis ToolDefinitions, namespaced as mcp_{server}_{tool}.
func (m *Manager) GetTools(ctx context.Context) []tools.ToolDefinition {
	var defs []tools.ToolDefinition

	for serverName, session := range m.sessions {
		result, err := session.ListTools(ctx, nil)
		if err != nil {
			log.Printf("Warning: failed to list tools from MCP server %q: %v", serverName, err)
			continue
		}

		for _, tool := range result.Tools {
			def := m.bridgeTool(serverName, session, tool)
			defs = append(defs, def)
		}
	}

	return defs
}

// bridgeTool converts an MCP Tool into a Jarvis ToolDefinition.
func (m *Manager) bridgeTool(serverName string, session *mcp.ClientSession, tool *mcp.Tool) tools.ToolDefinition {
	toolName := fmt.Sprintf("mcp_%s_%s", serverName, tool.Name)

	// Capture session and tool name for the closure
	sess := session
	mcpToolName := tool.Name

	return tools.ToolDefinition{
		Name:        toolName,
		Description: tool.Description,
		InputSchema: ConvertMCPSchema(tool.InputSchema),
		Function: func(input json.RawMessage) (string, error) {
			var args map[string]any
			if len(input) > 0 {
				if err := json.Unmarshal(input, &args); err != nil {
					return "", fmt.Errorf("failed to parse tool arguments: %w", err)
				}
			}

			result, err := sess.CallTool(context.Background(), &mcp.CallToolParams{
				Name:      mcpToolName,
				Arguments: args,
			})
			if err != nil {
				return "", fmt.Errorf("MCP tool call failed: %w", err)
			}

			return ExtractToolResult(result)
		},
	}
}

// Close gracefully shuts down all MCP server sessions.
func (m *Manager) Close() error {
	for name, session := range m.sessions {
		if err := session.Close(); err != nil {
			log.Printf("Warning: failed to close MCP session %q: %v", name, err)
		}
	}
	return nil
}
