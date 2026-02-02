package types

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
	Diff   *DiffInfo // Optional diff info for file updates
}

// DiffInfo contains diff information for file updates
type DiffInfo struct {
	FilePath     string
	UnifiedDiff  string // Pre-formatted diff with line numbers
	AddedLines   int
	RemovedLines int
	StartLine    int
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
