package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"chewbacca/internal/config"
	"chewbacca/internal/registry"
	"chewbacca/internal/tui"

	"github.com/anthropics/anthropic-sdk-go"
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
		message, err := a.runInferenceWithStreaming(ctx, callback)
		if err != nil {
			return err
		}

		a.conversation = append(a.conversation, message.ToParam())

		toolResults := []anthropic.ContentBlockParamUnion{}

		for _, content := range message.Content {
			if content.Type == "tool_use" {
				callback(tui.ToolStartMsg{Name: content.Name})

				result := a.executeTool(content.ID, content.Name, content.Input)
				toolResults = append(toolResults, result)

				resultText := extractToolResult(result)
				callback(tui.ToolCallMsg{
					Name:   content.Name,
					Input:  string(content.Input),
					Result: resultText,
				})
			}
		}

		if len(toolResults) == 0 {
			return nil
		}

		a.conversation = append(a.conversation, anthropic.NewUserMessage(toolResults...))
	}
}

func (a *Agent) runInferenceWithStreaming(ctx context.Context, callback func(msg any)) (*anthropic.Message, error) {
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

	message := anthropic.Message{}

	for stream.Next() {
		event := stream.Current()

		switch delta := event.Delta.(type) {
		case anthropic.ContentBlockDeltaEventDelta:
			if delta.Type == "text_delta" {
				callback(tui.StreamChunkMsg{Chunk: delta.Text})
			}
		}

		if event.Type == "message_start" && event.Message.ID != "" {
			message = event.Message
		}

		if event.Type == "message_delta" {
			message.StopReason = event.Delta.(anthropic.MessageDeltaEventDelta).StopReason
		}
	}

	if err := stream.Err(); err != nil {
		return nil, fmt.Errorf("streaming error: %w", err)
	}

	finalMessage := stream.FinalMessage()
	return finalMessage, nil
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
