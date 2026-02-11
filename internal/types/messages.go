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
	Diff   *DiffInfo
}

type DiffLine struct {
	Type      DiffLineType
	OldLineNo int
	NewLineNo int
	Content   string
}

type DiffLineType int

const (
	DiffLineContext DiffLineType = iota
	DiffLineAdded
	DiffLineRemoved
	DiffLineSkip
)

type DiffInfo struct {
	FilePath     string
	Operation    string
	AddedLines   int
	RemovedLines int
	Lines        []DiffLine
	UnifiedDiff  string
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

type TodoUpdateMsg struct {
	ActiveForm string
}
