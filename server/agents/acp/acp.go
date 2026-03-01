package acp

import (
	"encoding/json"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/jsonfile"
)

type SessionEntry struct {
	ID        string `json:"id"`
	CreatedAt int64  `json:"createdAt"`
	Model     string `json:"model,omitempty"`
	Agent     string `json:"agent,omitempty"`
	CWD       string `json:"cwd,omitempty"`
	// TrustWorkspace indicates if the workspace is trusted (for cursor-agent)
	TrustWorkspace bool `json:"trustWorkspace,omitempty"`
	// YoloMode indicates if --yolo flag should be passed to cursor-agent
	YoloMode bool `json:"yoloMode,omitempty"`
}

func NewSessionEntry(id, model, agent, cwd string) SessionEntry {
	return SessionEntry{
		ID:        id,
		CreatedAt: time.Now().UnixMilli(),
		Model:     model,
		Agent:     agent,
		CWD:       cwd,
	}
}

// SessionStore provides persistent session storage backed by a JSON file.
type SessionStore struct {
	file *jsonfile.JSONFile[[]SessionEntry]
}

func NewSessionStore(path string) *SessionStore {
	return &SessionStore{file: jsonfile.New[[]SessionEntry](path)}
}

func (s *SessionStore) Load() []SessionEntry {
	entries, err := s.file.Get()
	if err != nil {
		return nil
	}
	return entries
}

func (s *SessionStore) Add(entry SessionEntry) {
	s.file.Update(func(entries *[]SessionEntry) error {
		*entries = append(*entries, entry)
		return nil
	})
}

func (s *SessionStore) Get(id string) *SessionEntry {
	entries := s.Load()
	for _, e := range entries {
		if e.ID == id {
			return &e
		}
	}
	return nil
}

func (s *SessionStore) UpdateModel(id, model string) {
	s.file.Update(func(entries *[]SessionEntry) error {
		for i := range *entries {
			if (*entries)[i].ID == id {
				(*entries)[i].Model = model
				break
			}
		}
		return nil
	})
}

func (s *SessionStore) UpdateTrustWorkspace(id string, trust bool) {
	s.file.Update(func(entries *[]SessionEntry) error {
		for i := range *entries {
			if (*entries)[i].ID == id {
				(*entries)[i].TrustWorkspace = trust
				break
			}
		}
		return nil
	})
}

func (s *SessionStore) UpdateYoloMode(id string, yolo bool) {
	s.file.Update(func(entries *[]SessionEntry) error {
		for i := range *entries {
			if (*entries)[i].ID == id {
				(*entries)[i].YoloMode = yolo
				break
			}
		}
		return nil
	})
}

// MessageStore handles per-session message persistence using jsonfile.
type MessageStore struct {
	dir string
}

func NewMessageStore(dir string) *MessageStore {
	return &MessageStore{dir: dir}
}

func (m *MessageStore) filePath(sessionID string) string {
	return m.dir + "/" + sessionID + ".json"
}

func (m *MessageStore) Load(sessionID string) (json.RawMessage, error) {
	f := jsonfile.New[json.RawMessage](m.filePath(sessionID))
	data, err := f.Get()
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (m *MessageStore) Save(sessionID string, messages json.RawMessage) error {
	f := jsonfile.New[json.RawMessage](m.filePath(sessionID))
	return f.Set(messages)
}

type ModelInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Provider  string `json:"providerId"`
	ProviderN string `json:"providerName"`
	IsCurrent bool   `json:"is_current,omitempty"`
}

type SessionUpdate struct {
	Type       string          `json:"type"`
	Raw        json.RawMessage `json:"raw,omitempty"`
	Text       string          `json:"text,omitempty"`
	ToolCallID string          `json:"toolCallId,omitempty"`
	Title      string          `json:"title,omitempty"`
	Status     string          `json:"status,omitempty"`
	Content    string          `json:"content,omitempty"`
	Entries    json.RawMessage `json:"entries,omitempty"`
	Message    string          `json:"message,omitempty"`
	Model      string          `json:"model,omitempty"`
}

type PromptResult struct {
	StopReason string `json:"stopReason"`
}

type StatusInfo struct {
	Available bool   `json:"available"`
	Connected bool   `json:"connected"`
	SessionID string `json:"sessionId,omitempty"`
	CWD       string `json:"cwd,omitempty"`
	Message   string `json:"message,omitempty"`
	Model     string `json:"model,omitempty"`
}

// LogFunc is called by Agent implementations to report progress during operations.
type LogFunc func(message string)

// Agent defines the interface for any ACP-compatible agent adapter.
// Implementations include cursor-agent, and can be extended to support
// other ACP-compatible agents in the future.
type Agent interface {
	// Name returns the display name of the agent.
	Name() string

	// IsConnected returns whether the agent process is running and has an active session.
	IsConnected() bool

	// SessionID returns the current session ID, or empty string if not connected.
	SessionID() string

	// Status returns the current status information.
	Status() StatusInfo

	// Sessions returns all known sessions.
	Sessions() []SessionEntry

	// Models returns the available models for this agent.
	Models() ([]ModelInfo, error)

	// UpdateSessionModel persists the model choice for a session.
	UpdateSessionModel(sessionID, model string)

	// Connect creates a new session or resumes an existing one.
	// If resumeSessionID is non-empty, it resumes that session instead of creating new.
	// If debug is true, debug logs are streamed back.
	// The log callback is called with progress messages during connection.
	Connect(cwd string, resumeSessionID string, debug bool, log LogFunc) (sessionID string, err error)

	// Disconnect terminates the agent process and cleans up resources.
	Disconnect()

	// SendPrompt sends a text prompt and blocks until the agent finishes.
	// Updates are delivered via the Updates channel during execution.
	// If model is non-empty, it overrides the default model for this prompt.
	SendPrompt(sessionID string, text string, model string) (*PromptResult, error)

	// Cancel requests cancellation of the current prompt.
	Cancel(sessionID string) error

	// Updates returns a channel that receives real-time session updates
	// (agent messages, tool calls, plans) during prompt execution.
	Updates() <-chan SessionUpdate

	// GetMessages returns stored messages for a session.
	// Agents with built-in history can retrieve from their own backend;
	// others use file-based storage.
	GetMessages(sessionID string) (json.RawMessage, error)

	// SaveMessages persists messages for a session.
	SaveMessages(sessionID string, messages json.RawMessage) error
}
