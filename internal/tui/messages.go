package tui

import "jarvis/internal/types"

type ResponseMsg = types.ResponseMsg
type StreamChunkMsg = types.StreamChunkMsg
type ToolCallMsg = types.ToolCallMsg
type ToolStartMsg = types.ToolStartMsg
type ToolEndMsg = types.ToolEndMsg
type ErrorMsg = types.ErrorMsg
type AgentReadyMsg = types.AgentReadyMsg
type TodoUpdateMsg = types.TodoUpdateMsg
type ParallelGroupStartMsg = types.ParallelGroupStartMsg
type ParallelAgentUpdateMsg = types.ParallelAgentUpdateMsg
type ParallelGroupDoneMsg = types.ParallelGroupDoneMsg
