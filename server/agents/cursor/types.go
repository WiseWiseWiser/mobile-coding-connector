package cursor

import "encoding/json"

// CursorEvent represents a single event from cursor-agent's --output-format stream-json output.
// The stream produces one JSON object per line (NDJSON).
type CursorEvent struct {
	Type      string          `json:"type"`       // "user", "assistant", "tool_call", "result"
	Subtype   string          `json:"subtype"`    // For tool_call: "started", "completed"; For result: subtype
	Message   *CursorMessage  `json:"message"`    // Present for "user" and "assistant" types
	ToolCall  json.RawMessage `json:"tool_call"`  // Present for "tool_call" type (varies by tool)
	SessionID string          `json:"session_id"` // Cursor session ID
	// For "result" type
	DurationMs int `json:"duration_ms"`
}

// CursorMessage represents a user or assistant message.
type CursorMessage struct {
	Role    string               `json:"role"`    // "user" or "assistant"
	Content []CursorContentBlock `json:"content"` // Content blocks
}

// CursorContentBlock represents a content block in a message.
type CursorContentBlock struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

// Tool call types - each tool call event has one of these fields set in the tool_call object.

// ShellToolCall represents a shell command execution.
type ShellToolCall struct {
	ShellToolCall *ShellToolCallData `json:"shellToolCall"`
}

type ShellToolCallData struct {
	Args   *ShellArgs   `json:"args"`
	Result *ShellResult `json:"result"`
}

type ShellArgs struct {
	Command string `json:"command"`
}

type ShellResult struct {
	Success *ShellSuccess `json:"success"`
}

type ShellSuccess struct {
	ExitCode int    `json:"exitCode"`
	Output   string `json:"output"`
}

// ReadToolCall represents a file read operation.
type ReadToolCall struct {
	ReadToolCall *ReadToolCallData `json:"readToolCall"`
}

type ReadToolCallData struct {
	Args   *ReadArgs   `json:"args"`
	Result *ReadResult `json:"result"`
}

type ReadArgs struct {
	Path   string `json:"path"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

type ReadResult struct {
	Success *ReadSuccess `json:"success"`
}

type ReadSuccess struct {
	TotalLines int    `json:"totalLines"`
	Content    string `json:"content"`
}

// EditToolCall represents a file edit operation.
type EditToolCall struct {
	EditToolCall *EditToolCallData `json:"editToolCall"`
}

type EditToolCallData struct {
	Args   *EditArgs   `json:"args"`
	Result *EditResult `json:"result"`
}

type EditArgs struct {
	Path string `json:"path"`
}

type EditResult struct {
	Success *EditSuccess `json:"success"`
}

type EditSuccess struct{}

// WriteToolCall represents a file write operation.
type WriteToolCall struct {
	WriteToolCall *WriteToolCallData `json:"writeToolCall"`
}

type WriteToolCallData struct {
	Args   *WriteArgs   `json:"args"`
	Result *WriteResult `json:"result"`
}

type WriteArgs struct {
	Path     string `json:"path"`
	FileText string `json:"fileText"`
}

type WriteResult struct {
	Success *WriteSuccess `json:"success"`
}

type WriteSuccess struct {
	LinesCreated int `json:"linesCreated"`
	FileSize     int `json:"fileSize"`
}

// GrepToolCall represents a grep/search operation.
type GrepToolCall struct {
	GrepToolCall *GrepToolCallData `json:"grepToolCall"`
}

type GrepToolCallData struct {
	Args   *GrepArgs   `json:"args"`
	Result *GrepResult `json:"result"`
}

type GrepArgs struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
}

type GrepResult struct {
	Success *GrepSuccess `json:"success"`
}

type GrepSuccess struct {
	WorkspaceResults map[string]interface{} `json:"workspaceResults"`
}

// GlobToolCall represents a file glob/find operation.
type GlobToolCall struct {
	GlobToolCall *GlobToolCallData `json:"globToolCall"`
}

type GlobToolCallData struct {
	Args   *GlobArgs   `json:"args"`
	Result *GlobResult `json:"result"`
}

type GlobArgs struct {
	GlobPattern     string `json:"globPattern"`
	TargetDirectory string `json:"targetDirectory"`
}

type GlobResult struct {
	Success *GlobSuccess `json:"success"`
}

type GlobSuccess struct {
	TotalFiles int `json:"totalFiles"`
}

// LsToolCall represents a directory listing operation.
type LsToolCall struct {
	LsToolCall *LsToolCallData `json:"lsToolCall"`
}

type LsToolCallData struct {
	Args   *LsArgs   `json:"args"`
	Result *LsResult `json:"result"`
}

type LsArgs struct {
	Path   string   `json:"path"`
	Ignore []string `json:"ignore"`
}

type LsResult struct {
	Success *LsSuccess `json:"success"`
}

type LsSuccess struct {
	DirectoryTreeRoot interface{} `json:"directoryTreeRoot"`
}

// DeleteToolCall represents a file delete operation.
type DeleteToolCall struct {
	DeleteToolCall *DeleteToolCallData `json:"deleteToolCall"`
}

type DeleteToolCallData struct {
	Args   *DeleteArgs   `json:"args"`
	Result *DeleteResult `json:"result"`
}

type DeleteArgs struct {
	Path string `json:"path"`
}

type DeleteResult struct {
	Success  *DeleteSuccess  `json:"success"`
	Rejected *DeleteRejected `json:"rejected"`
}

type DeleteSuccess struct{}

type DeleteRejected struct {
	Reason string `json:"reason"`
}

// TodoToolCall represents a todo management operation.
type TodoToolCall struct {
	TodoToolCall *TodoToolCallData `json:"todoToolCall"`
}

type TodoToolCallData struct {
	Args   *TodoArgs   `json:"args"`
	Result *TodoResult `json:"result"`
}

type TodoArgs struct {
	Merge bool       `json:"merge"`
	Todos []TodoItem `json:"todos"`
}

type TodoItem struct {
	Content string `json:"content"`
	Status  string `json:"status"`
}

type TodoResult struct {
	Success *TodoSuccess `json:"success"`
}

type TodoSuccess struct {
	Todos []TodoItem `json:"todos"`
}
