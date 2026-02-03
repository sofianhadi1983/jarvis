package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"jarvis/internal/config"
	"jarvis/internal/registry"
	"jarvis/internal/types"

	"github.com/sofianhadi1983/anthropic-sdk-go"
	"github.com/sofianhadi1983/anthropic-sdk-go/shared/constant"
)

type Agent struct {
	client       anthropic.Client
	registry     *registry.Registry
	config       *config.Config
	conversation []anthropic.BetaMessageParam
	systemPrompt string
}

func NewAgent(client anthropic.Client, reg *registry.Registry, cfg *config.Config) (*Agent, error) {
	systemPrompt, err := cfg.LoadSystemPrompt()
	if err != nil {
		return nil, fmt.Errorf("failed to load system prompt: %w", err)
	}

	return &Agent{
		client:       client,
		registry:     reg,
		config:       cfg,
		conversation: []anthropic.BetaMessageParam{},
		systemPrompt: systemPrompt,
	}, nil
}

func (a *Agent) SendMessage(ctx context.Context, input string, callback func(msg any)) error {
	userMessage := anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(input))
	a.conversation = append(a.conversation, userMessage)

	for {
		textContent, toolBlocks, err := a.runInferenceWithStreaming(ctx, callback)
		if err != nil {
			return fmt.Errorf("inference error: %w", err)
		}

		if textContent == "" && len(toolBlocks) == 0 {
			return fmt.Errorf("received empty response from API")
		}

		assistantContent := []anthropic.BetaContentBlockParamUnion{}
		if textContent != "" {
			assistantContent = append(assistantContent, anthropic.NewBetaTextBlock(textContent))
		}

		toolResults := []anthropic.BetaContentBlockParamUnion{}

		for _, tb := range toolBlocks {
			validInput := ensureValidJSON(tb.inputJSON)

			assistantContent = append(assistantContent, anthropic.NewBetaToolUseBlock(tb.id, json.RawMessage(validInput), tb.name))

			callback(types.ToolStartMsg{Name: tb.name})

			result, isError := a.executeTool(tb.id, tb.name, json.RawMessage(validInput))
			toolResults = append(toolResults, a.newBetaToolResult(tb.id, result, isError))

			var diffInfo *types.DiffInfo
			if tb.name == "Update" {
				diffInfo = extractDiffInfo(result)
			}

			callback(types.ToolCallMsg{
				Name:   tb.name,
				Input:  validInput,
				Result: result,
				Diff:   diffInfo,
			})
		}

		a.conversation = append(a.conversation, anthropic.BetaMessageParam{
			Role:    anthropic.BetaMessageParamRoleAssistant,
			Content: assistantContent,
		})

		if len(toolResults) == 0 {
			return nil
		}

		a.conversation = append(a.conversation, anthropic.NewBetaUserMessage(toolResults...))
	}
}

type toolBlock struct {
	id        string
	name      string
	inputJSON string
}

func ensureValidJSON(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return "{}"
	}

	var js json.RawMessage
	if err := json.Unmarshal([]byte(input), &js); err != nil {
		return "{}"
	}

	return input
}

func (a *Agent) runInferenceWithStreaming(ctx context.Context, callback func(msg any)) (string, []toolBlock, error) {
	tools := a.registry.ToAnthropicBetaTools()
	stream := a.client.Beta.Messages.NewStreaming(ctx, anthropic.BetaMessageNewParams{
		System: []anthropic.BetaTextBlockParam{
			{
				Type: constant.Text("text"),
				Text: "You are Claude Code, Anthropic's official CLI for Claude.",
				CacheControl: anthropic.BetaCacheControlEphemeralParam{
					Type: constant.Ephemeral("ephemeral"),
					TTL:  anthropic.BetaCacheControlEphemeralTTLTTL1h,
				},
			},
			{
				Type: constant.Text("text"),
				Text: a.systemPrompt,
				CacheControl: anthropic.BetaCacheControlEphemeralParam{
					Type: constant.Ephemeral("ephemeral"),
					TTL:  anthropic.BetaCacheControlEphemeralTTLTTL1h,
				},
			},
		},
		MaxTokens: 32000,
		Messages:  a.conversation,
		Model:     anthropic.ModelClaudeSonnet4_5_20250929,
		Tools:     tools,
	})

	message := anthropic.BetaMessage{}
	var textContent strings.Builder
	var toolBlocks []toolBlock
	var toolInputBuilders []strings.Builder
	var currentToolIndex int = -1

	for stream.Next() {
		event := stream.Current()

		if err := message.Accumulate(event); err != nil {
			continue
		}

		switch eventVariant := event.AsAny().(type) {
		case anthropic.BetaRawMessageStartEvent:

		case anthropic.BetaRawContentBlockStartEvent:
			switch blockVariant := eventVariant.ContentBlock.AsAny().(type) {
			case anthropic.BetaToolUseBlock:
				toolBlocks = append(toolBlocks, toolBlock{
					id:   blockVariant.ID,
					name: blockVariant.Name,
				})
				toolInputBuilders = append(toolInputBuilders, strings.Builder{})
				currentToolIndex = len(toolBlocks) - 1
			}

		case anthropic.BetaRawContentBlockDeltaEvent:
			switch deltaVariant := eventVariant.Delta.AsAny().(type) {
			case anthropic.BetaTextDelta:
				textContent.WriteString(deltaVariant.Text)
				callback(types.StreamChunkMsg{Chunk: deltaVariant.Text})
			case anthropic.BetaInputJSONDelta:
				if currentToolIndex >= 0 && currentToolIndex < len(toolInputBuilders) {
					toolInputBuilders[currentToolIndex].WriteString(deltaVariant.PartialJSON)
				}
			}

		case anthropic.BetaRawContentBlockStopEvent:
			currentToolIndex = -1
		}
	}

	if err := stream.Err(); err != nil {
		return "", nil, fmt.Errorf("streaming error: %v", err)
	}

	if message.ID == "" {
		return "", nil, fmt.Errorf("no message received from API")
	}

	for i := range toolBlocks {
		if i < len(toolInputBuilders) {
			toolBlocks[i].inputJSON = toolInputBuilders[i].String()
		}
	}

	return textContent.String(), toolBlocks, nil
}

func (a *Agent) executeTool(id, name string, input json.RawMessage) (string, bool) {
	toolDef, found := a.registry.Get(name)
	if !found {
		return "tool not found: " + name, true
	}

	response, err := toolDef.Function(input)
	if err != nil {
		return err.Error(), true
	}
	return response, false
}

func (a *Agent) newBetaToolResult(toolUseID, content string, isError bool) anthropic.BetaContentBlockParamUnion {
	return anthropic.BetaContentBlockParamUnion{
		OfToolResult: &anthropic.BetaToolResultBlockParam{
			ToolUseID: toolUseID,
			Content: []anthropic.BetaToolResultBlockParamContentUnion{
				{OfText: &anthropic.BetaTextBlockParam{Text: content}},
			},
			IsError: anthropic.Bool(isError),
		},
	}
}

func (a *Agent) ClearHistory() {
	a.conversation = []anthropic.BetaMessageParam{}
}

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
