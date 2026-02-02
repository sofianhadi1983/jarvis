package tui

type ResponseMsg struct {
	Content string
}

type StreamChunkMsg struct {
	Chunk string
}

type ToolCallMsg struct {
	Name   string
	Input  string
	Result string
}

type ToolStartMsg struct {
	Name string
}

type ToolEndMsg struct {
	Name   string
	Result string
}

type ErrorMsg struct {
	Err error
}

type AgentReadyMsg struct{}
