package mcp

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sofianhadi1983/anthropic-sdk-go"
)

// ConvertMCPSchema converts an MCP tool's InputSchema (any) to the Anthropic
// ToolInputSchemaParam format by marshaling through JSON.
func ConvertMCPSchema(schema any) anthropic.ToolInputSchemaParam {
	param := anthropic.ToolInputSchemaParam{
		Type: "object",
	}

	if schema == nil {
		return param
	}

	// Marshal the schema to JSON, then unmarshal to extract fields
	data, err := json.Marshal(schema)
	if err != nil {
		return param
	}

	var raw struct {
		Properties json.RawMessage `json:"properties"`
		Required   []string        `json:"required"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return param
	}

	if raw.Properties != nil {
		var props any
		if err := json.Unmarshal(raw.Properties, &props); err == nil {
			param.Properties = props
		}
	}
	param.Required = raw.Required

	return param
}

// ExtractToolResult extracts text content from an MCP CallToolResult.
// Returns concatenated text from all TextContent blocks.
// Returns an error if the result indicates a tool error.
func ExtractToolResult(result *mcp.CallToolResult) (string, error) {
	if result.IsError {
		// Collect error text from content blocks
		var errTexts []string
		for _, c := range result.Content {
			if tc, ok := c.(*mcp.TextContent); ok {
				errTexts = append(errTexts, tc.Text)
			}
		}
		if len(errTexts) > 0 {
			return "", fmt.Errorf("MCP tool error: %s", strings.Join(errTexts, "\n"))
		}
		return "", fmt.Errorf("MCP tool returned an error")
	}

	var parts []string
	for _, c := range result.Content {
		switch tc := c.(type) {
		case *mcp.TextContent:
			parts = append(parts, tc.Text)
		case *mcp.ImageContent:
			parts = append(parts, fmt.Sprintf("[image: %s]", tc.MIMEType))
		default:
			parts = append(parts, "[unsupported content type]")
		}
	}

	return strings.Join(parts, "\n"), nil
}
