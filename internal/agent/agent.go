package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"chewbacca/internal/config"
	"chewbacca/internal/registry"
	"chewbacca/internal/types"

	"github.com/sofianhadi1983/anthropic-sdk-go"
)

type Agent struct {
	client       *anthropic.Client
	registry     *registry.Registry
	config       *config.Config
	conversation []anthropic.MessageParam
	systemPrompt string
}

func NewAgent(client *anthropic.Client, reg *registry.Registry, cfg *config.Config) (*Agent, error) {
	systemPrompt, err := cfg.LoadSystemPrompt()
	if err != nil {
		return nil, fmt.Errorf("failed to load system prompt: %w", err)
	}

	return &Agent{
		client:       client,
		registry:     reg,
		config:       cfg,
		conversation: []anthropic.MessageParam{},
		systemPrompt: systemPrompt,
	}, nil
}

func (a *Agent) SendMessage(ctx context.Context, input string, callback func(msg any)) error {
	userMessage := anthropic.NewUserMessage(anthropic.NewTextBlock(input))
	a.conversation = append(a.conversation, userMessage)

	for {
		textContent, toolBlocks, err := a.runInferenceWithStreaming(ctx, callback)
		if err != nil {
			return fmt.Errorf("inference error: %w", err)
		}

		if textContent == "" && len(toolBlocks) == 0 {
			return fmt.Errorf("received empty response from API")
		}

		assistantContent := []anthropic.ContentBlockParamUnion{}
		if textContent != "" {
			assistantContent = append(assistantContent, anthropic.NewTextBlock(textContent))
		}

		toolResults := []anthropic.ContentBlockParamUnion{}

		for _, tb := range toolBlocks {
			assistantContent = append(assistantContent, anthropic.ContentBlockParamUnion{
				OfToolUse: &anthropic.ToolUseBlockParam{
					ID:    tb.id,
					Name:  tb.name,
					Input: json.RawMessage(tb.inputJSON),
				},
			})

			callback(types.ToolStartMsg{Name: tb.name})

			result := a.executeTool(tb.id, tb.name, json.RawMessage(tb.inputJSON))
			toolResults = append(toolResults, result)

			resultText := extractToolResult(result)

			// Check if this is an Update tool and extract diff info
			var diffInfo *types.DiffInfo
			if tb.name == "Update" {
				diffInfo = extractDiffInfo(resultText)
			}

			callback(types.ToolCallMsg{
				Name:   tb.name,
				Input:  tb.inputJSON,
				Result: resultText,
				Diff:   diffInfo,
			})
		}

		a.conversation = append(a.conversation, anthropic.MessageParam{
			Role:    anthropic.MessageParamRoleAssistant,
			Content: assistantContent,
		})

		if len(toolResults) == 0 {
			return nil
		}

		a.conversation = append(a.conversation, anthropic.NewUserMessage(toolResults...))
	}
}

type toolBlock struct {
	id        string
	name      string
	inputJSON string
}

func (a *Agent) runInferenceWithStreaming(ctx context.Context, callback func(msg any)) (string, []toolBlock, error) {
	tools := a.registry.ToAnthropicTools()

	stream := a.client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(a.config.Anthropic.Model),
		MaxTokens: a.config.Anthropic.MaxTokens,
		Messages:  a.conversation,
		System: []anthropic.TextBlockParam{
			{Text: a.systemPrompt},
		},
		Tools: tools,
	})

	var messageID string
	var textContent strings.Builder
	var toolBlocks []toolBlock
	var toolInputBuilders []strings.Builder
	var currentBlockIndex int64 = -1

	for stream.Next() {
		event := stream.Current()

		switch event.Type {
		case "message_start":
			messageID = event.Message.ID

		case "content_block_start":
			currentBlockIndex = event.Index
			if event.ContentBlock.Type == "tool_use" {
				toolBlocks = append(toolBlocks, toolBlock{
					id:   event.ContentBlock.ID,
					name: event.ContentBlock.Name,
				})
				toolInputBuilders = append(toolInputBuilders, strings.Builder{})
			}

		case "content_block_delta":
			if event.Delta.Type == "text_delta" {
				textContent.WriteString(event.Delta.Text)
				callback(types.StreamChunkMsg{Chunk: event.Delta.Text})
			} else if event.Delta.Type == "input_json_delta" {
				toolIdx := findToolBlockIndex(toolBlocks, currentBlockIndex)
				if toolIdx >= 0 && toolIdx < len(toolInputBuilders) {
					toolInputBuilders[toolIdx].WriteString(event.Delta.PartialJSON)
				}
			}
		}
	}

	if err := stream.Err(); err != nil {
		return "", nil, fmt.Errorf("streaming error: %w", err)
	}

	stream.Close()

	if messageID == "" {
		return "", nil, fmt.Errorf("no message received from API (did you set ANTHROPIC_API_KEY?)")
	}

	for i := range toolBlocks {
		if i < len(toolInputBuilders) {
			toolBlocks[i].inputJSON = toolInputBuilders[i].String()
		}
	}

	return textContent.String(), toolBlocks, nil
}

func findToolBlockIndex(blocks []toolBlock, contentIndex int64) int {
	textBlockCount := 0
	for i := range blocks {
		expectedIndex := int64(textBlockCount + i)
		if expectedIndex == contentIndex {
			return i
		}
	}
	return len(blocks) - 1
}

func (a *Agent) executeTool(id, name string, input json.RawMessage) anthropic.ContentBlockParamUnion {
	toolDef, found := a.registry.Get(name)
	if !found {
		return anthropic.NewToolResultBlock(id, "tool not found: "+name, true)
	}

	response, err := toolDef.Function(input)
	if err != nil {
		return anthropic.NewToolResultBlock(id, err.Error(), true)
	}
	return anthropic.NewToolResultBlock(id, response, false)
}

func extractToolResult(block anthropic.ContentBlockParamUnion) string {
	if block.OfToolResult != nil {
		if len(block.OfToolResult.Content) > 0 {
			for _, c := range block.OfToolResult.Content {
				if c.OfText != nil {
					return c.OfText.Text
				}
			}
		}
	}
	return ""
}

func (a *Agent) ClearHistory() {
	a.conversation = []anthropic.MessageParam{}
}

// extractDiffInfo parses Update tool result to get diff information
func extractDiffInfo(result string) *types.DiffInfo {
	var updateResult struct {
		Success      bool   `json:"success"`
		FilePath     string `json:"file_path"`
		UnifiedDiff  string `json:"unified_diff"`
		AddedLines   int    `json:"added_lines"`
		RemovedLines int    `json:"removed_lines"`
		StartLine    int    `json:"start_line"`
	}

	if err := json.Unmarshal([]byte(result), &updateResult); err != nil {
		return nil
	}

	if !updateResult.Success {
		return nil
	}

	return &types.DiffInfo{
		FilePath:     updateResult.FilePath,
		UnifiedDiff:  updateResult.UnifiedDiff,
		AddedLines:   updateResult.AddedLines,
		RemovedLines: updateResult.RemovedLines,
		StartLine:    updateResult.StartLine,
	}
}
