package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"

	"jarvis/internal/registry"
	"jarvis/internal/types"
)

// ParallelTaskInput represents one Task tool call to run in parallel.
type ParallelTaskInput struct {
	ToolUseID string
	Input     TaskInput
	RawInput  json.RawMessage
}

// ParallelTaskResult holds the result of a single parallel task.
type ParallelTaskResult struct {
	ToolUseID string
	Result    string
	IsError   bool
}

// RunTasksParallel executes multiple Task tool calls concurrently.
// Each goroutine writes to its own index in the results slice.
func RunTasksParallel(
	ctx context.Context,
	parentRegistry *registry.Registry,
	tasks []ParallelTaskInput,
	parentCallback func(msg any),
	newChild ChildFactory,
	groupID string,
) []ParallelTaskResult {
	results := make([]ParallelTaskResult, len(tasks))
	var wg sync.WaitGroup

	for idx, task := range tasks {
		wg.Add(1)
		go func(idx int, task ParallelTaskInput) {
			defer wg.Done()

			result, isError := runOneParallelTask(ctx, parentRegistry, task, parentCallback, newChild, groupID, idx)
			results[idx] = ParallelTaskResult{
				ToolUseID: task.ToolUseID,
				Result:    result,
				IsError:   isError,
			}
		}(idx, task)
	}

	wg.Wait()
	return results
}

func runOneParallelTask(
	ctx context.Context,
	parentRegistry *registry.Registry,
	task ParallelTaskInput,
	parentCallback func(msg any),
	newChild ChildFactory,
	groupID string,
	agentIndex int,
) (string, bool) {
	allowedTools, ok := agentAllowedTools[task.Input.AgentType]
	if !ok {
		return fmt.Sprintf("unknown agent_type %q: must be explore, code, or plan", task.Input.AgentType), true
	}

	childReg := registry.NewRegistry()
	for _, toolName := range allowedTools {
		if toolDef, found := parentRegistry.Get(toolName); found {
			childReg.RegisterOrReplace(toolDef)
		}
	}

	systemPrompt := agentSystemPrompts[task.Input.AgentType]
	systemPrompt += "\nWorking directory: use relative paths from the project root."

	child := newChild(childReg, systemPrompt)

	var toolCount int32
	var result strings.Builder

	childCallback := func(msg any) {
		switch m := msg.(type) {
		case types.StreamChunkMsg:
			result.WriteString(m.Chunk)
		case types.ToolStartMsg:
			result.Reset()
			count := int(atomic.AddInt32(&toolCount, 1))
			if parentCallback != nil {
				parentCallback(types.ParallelAgentUpdateMsg{
					GroupID:    groupID,
					AgentIndex: agentIndex,
					ToolCount:  count,
					Status:     m.Name,
				})
			}
		}
	}

	err := child.SendMessage(ctx, task.Input.Prompt, childCallback)
	if err != nil {
		if result.Len() > 0 {
			return result.String(), false
		}
		return fmt.Sprintf("subagent error: %v", err), true
	}

	if result.Len() == 0 {
		return "Subagent completed but produced no output.", false
	}

	return result.String(), false
}
