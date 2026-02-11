package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"jarvis/internal/config"
	"jarvis/internal/image"
	"jarvis/internal/registry"
	"jarvis/internal/subagent"
	"jarvis/internal/todo"
	"jarvis/internal/types"
	"jarvis/pkg/tools"

	"github.com/sofianhadi1983/anthropic-sdk-go"
	"github.com/sofianhadi1983/anthropic-sdk-go/shared/constant"
)

type Agent struct {
	client            anthropic.Client
	registry          *registry.Registry
	config            *config.Config
	conversation      []anthropic.BetaMessageParam
	systemPrompt      string
	todoManager       *todo.Manager
	roundsWithoutTodo int
	callback          func(msg any)
	ctx               context.Context
	maxRounds         int
}

func NewAgent(client anthropic.Client, reg *registry.Registry, cfg *config.Config) (*Agent, error) {
	systemPrompt, err := cfg.LoadSystemPrompt()
	if err != nil {
		return nil, fmt.Errorf("failed to load system prompt: %w", err)
	}

	a := &Agent{
		client:       client,
		registry:     reg,
		config:       cfg,
		conversation: []anthropic.BetaMessageParam{},
		systemPrompt: systemPrompt,
	}

	a.todoManager = todo.NewManager(func(activeForm string) {
		if a.callback != nil {
			a.callback(types.TodoUpdateMsg{ActiveForm: activeForm})
		}
	})

	reg.RegisterOrReplace(tools.ToolDefinition{
		Name:        "TodoWrite",
		Description: "Track tasks for complex multi-step work. Send the COMPLETE todo list each time (not diffs). Max 20 items, only 1 in_progress at a time.",
		InputSchema: tools.GenerateSchema[todo.TodoWriteInput](),
		Function: func(input json.RawMessage) (string, error) {
			var params todo.TodoWriteInput
			if err := json.Unmarshal(input, &params); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			return a.todoManager.Replace(params.Todos)
		},
	})

	reg.RegisterOrReplace(tools.ToolDefinition{
		Name: "Task",
		Description: "Spawn a subagent to handle a focused subtask with isolated context. " +
			"Use for exploration, code changes, or planning that would pollute your context.",
		InputSchema: tools.GenerateSchema[subagent.TaskInput](),
		Function: func(input json.RawMessage) (string, error) {
			var params subagent.TaskInput
			if err := json.Unmarshal(input, &params); err != nil {
				return "", fmt.Errorf("invalid input: %w", err)
			}
			return subagent.RunTask(a.ctx, a.registry, params, a.callback,
				func(reg *registry.Registry, sysPrompt string) subagent.MessageSender {
					return NewChildAgent(a.client, reg, sysPrompt)
				})
		},
	})

	return a, nil
}

func NewChildAgent(client anthropic.Client, reg *registry.Registry, systemPrompt string) *Agent {
	return &Agent{
		client:       client,
		registry:     reg,
		conversation: []anthropic.BetaMessageParam{},
		systemPrompt: systemPrompt,
		todoManager:  todo.NewManager(nil),
		maxRounds:    30,
	}
}

func (a *Agent) SendMessage(ctx context.Context, input string, callback func(msg any)) error {
	userMessage := anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(input))
	a.conversation = append(a.conversation, userMessage)

	return a.processConversation(ctx, callback)
}

// SendMessageWithImages sends a message with images to the agent.
// Images are added before text content for better understanding (per Anthropic docs).
func (a *Agent) SendMessageWithImages(ctx context.Context, text string, images []*image.ImageInput, callback func(msg any)) error {
	contentBlocks := []anthropic.BetaContentBlockParamUnion{}

	// Add images first (better understanding per Anthropic docs)
	for _, img := range images {
		if img.IsURL {
			contentBlocks = append(contentBlocks, anthropic.NewBetaImageBlock(
				anthropic.BetaURLImageSourceParam{
					URL: img.URL,
				},
			))
		} else {
			contentBlocks = append(contentBlocks, anthropic.NewBetaImageBlock(
				anthropic.BetaBase64ImageSourceParam{
					Data:      img.Data,
					MediaType: anthropic.BetaBase64ImageSourceMediaType(img.MediaType),
				},
			))
		}
	}

	// Add text content after images
	if text != "" {
		contentBlocks = append(contentBlocks, anthropic.NewBetaTextBlock(text))
	}

	userMessage := anthropic.BetaMessageParam{
		Role:    anthropic.BetaMessageParamRoleUser,
		Content: contentBlocks,
	}
	a.conversation = append(a.conversation, userMessage)

	return a.processConversation(ctx, callback)
}

// processConversation handles the inference loop for both text-only and image messages.
func (a *Agent) processConversation(ctx context.Context, callback func(msg any)) error {
	a.callback = callback
	a.ctx = ctx
	defer func() { a.callback = nil; a.ctx = nil }()

	rounds := 0
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
		todoUsedThisRound := false

		for _, tb := range toolBlocks {
			validInput := ensureValidJSON(tb.inputJSON)

			assistantContent = append(assistantContent, anthropic.NewBetaToolUseBlock(tb.id, json.RawMessage(validInput), tb.name))

			callback(types.ToolStartMsg{Name: tb.name})

			result, isError := a.executeTool(tb.id, tb.name, json.RawMessage(validInput))
			toolResults = append(toolResults, a.newBetaToolResult(tb.id, result, isError))

			if tb.name == "TodoWrite" {
				todoUsedThisRound = true
			}

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

		if todoUsedThisRound {
			a.roundsWithoutTodo = 0
		} else if len(toolBlocks) > 0 {
			a.roundsWithoutTodo++
		}

		a.conversation = append(a.conversation, anthropic.BetaMessageParam{
			Role:    anthropic.BetaMessageParamRoleAssistant,
			Content: assistantContent,
		})

		if len(toolResults) == 0 {
			return nil
		}

		// Nag reminder if TodoWrite hasn't been used for a while
		if a.roundsWithoutTodo >= 10 && a.todoManager.HasItems() {
			nag := anthropic.NewBetaTextBlock("[Reminder: Update your todo list with TodoWrite to track progress.]")
			toolResults = append([]anthropic.BetaContentBlockParamUnion{nag}, toolResults...)
			a.roundsWithoutTodo = 0
		}

		a.conversation = append(a.conversation, anthropic.NewBetaUserMessage(toolResults...))

		if a.maxRounds > 0 {
			rounds++
			if rounds >= a.maxRounds {
				return fmt.Errorf("subagent reached max tool rounds (%d)", a.maxRounds)
			}
		}
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
	a.todoManager.Reset()
	a.roundsWithoutTodo = 0
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

	lines := parseUnifiedDiff(updateResult.UnifiedDiff, updateResult.StartLine)

	return &types.DiffInfo{
		FilePath:     updateResult.FilePath,
		UnifiedDiff:  updateResult.UnifiedDiff,
		AddedLines:   updateResult.AddedLines,
		RemovedLines: updateResult.RemovedLines,
		StartLine:    updateResult.StartLine,
		Lines:        lines,
	}
}

func parseUnifiedDiff(unifiedDiff string, startLine int) []types.DiffLine {
	if unifiedDiff == "" {
		return nil
	}

	var result []types.DiffLine
	lines := strings.Split(unifiedDiff, "\n")

	oldLineNo := startLine
	newLineNo := startLine
	if oldLineNo == 0 {
		oldLineNo = 1
		newLineNo = 1
	}

	contextBefore := 3
	contextAfter := 3
	maxLines := 20

	var diffLines []struct {
		line      string
		oldNo     int
		newNo     int
		lineType  types.DiffLineType
	}

	for _, line := range lines {
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "@@") {
			continue
		}

		if strings.HasPrefix(line, "-") {
			diffLines = append(diffLines, struct {
				line     string
				oldNo    int
				newNo    int
				lineType types.DiffLineType
			}{
				line:     strings.TrimPrefix(line, "-"),
				oldNo:    oldLineNo,
				newNo:    0,
				lineType: types.DiffLineRemoved,
			})
			oldLineNo++
		} else if strings.HasPrefix(line, "+") {
			diffLines = append(diffLines, struct {
				line     string
				oldNo    int
				newNo    int
				lineType types.DiffLineType
			}{
				line:     strings.TrimPrefix(line, "+"),
				oldNo:    0,
				newNo:    newLineNo,
				lineType: types.DiffLineAdded,
			})
			newLineNo++
		} else {
			content := strings.TrimPrefix(line, " ")
			diffLines = append(diffLines, struct {
				line     string
				oldNo    int
				newNo    int
				lineType types.DiffLineType
			}{
				line:     content,
				oldNo:    oldLineNo,
				newNo:    newLineNo,
				lineType: types.DiffLineContext,
			})
			oldLineNo++
			newLineNo++
		}
	}

	changeIndices := []int{}
	for i, dl := range diffLines {
		if dl.lineType == types.DiffLineAdded || dl.lineType == types.DiffLineRemoved {
			changeIndices = append(changeIndices, i)
		}
	}

	if len(changeIndices) == 0 {
		return nil
	}

	includeLines := make(map[int]bool)
	for _, idx := range changeIndices {
		for i := idx - contextBefore; i <= idx+contextAfter; i++ {
			if i >= 0 && i < len(diffLines) {
				includeLines[i] = true
			}
		}
	}

	lastIncluded := -2
	lineCount := 0
	for i := 0; i < len(diffLines) && lineCount < maxLines; i++ {
		if !includeLines[i] {
			continue
		}

		if lastIncluded >= 0 && i > lastIncluded+1 {
			result = append(result, types.DiffLine{
				Type:    types.DiffLineSkip,
				Content: "...",
			})
			lineCount++
		}

		dl := diffLines[i]
		result = append(result, types.DiffLine{
			Type:      dl.lineType,
			OldLineNo: dl.oldNo,
			NewLineNo: dl.newNo,
			Content:   dl.line,
		})
		lineCount++
		lastIncluded = i
	}

	return result
}
