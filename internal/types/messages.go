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

// ParallelGroupStartMsg is sent once when a parallel group of subagents starts.
type ParallelGroupStartMsg struct {
	GroupID    string
	TaskNames  []string // description per agent
	AgentTypes []string // "explore"/"code"/"plan" per agent
}

// ParallelAgentUpdateMsg is sent by each subagent as it uses tools.
type ParallelAgentUpdateMsg struct {
	GroupID    string
	AgentIndex int
	ToolCount  int
	Status     string
}

// ParallelGroupDoneMsg is sent when all parallel agents finish.
type ParallelGroupDoneMsg struct {
	GroupID string
}

type SkillLoadedMsg struct {
	Name string
}

type MCPServerStatus struct {
	Name      string
	Command   string
	Connected bool
	Tools     []string
}
