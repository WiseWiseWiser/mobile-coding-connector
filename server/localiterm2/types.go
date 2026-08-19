package localiterm2

// Inventory is the GET /api/local/iterm2/inventory response.
type Inventory struct {
	ITermRunning bool           `json:"iterm_running"`
	Desktops     []DesktopGroup `json:"desktops"`
	SavedNotes   []OrphanNote   `json:"saved_notes"`
	// CachedAt is when this snapshot was last built (RFC3339). Empty on a cold miss before store.
	CachedAt string `json:"cached_at,omitempty"`
	// FromCache is true when this response is the in-memory cache (no capture this request).
	FromCache bool `json:"from_cache"`
	// Refreshing is true while more windows may still arrive on the stream.
	Refreshing bool `json:"refreshing"`
}

// DesktopGroup is one Mission Control Desktop and its live iTerm sessions.
type DesktopGroup struct {
	// SpaceIndex is 0-based (space 0 = Desktop 1).
	SpaceIndex int `json:"space_index"`
	// Desktop is the 1-based Mission Control number.
	Desktop int `json:"desktop"`
	// SpaceID is the CGS id64 for this Desktop. 0 when listing failed.
	SpaceID uint64 `json:"space_id,omitempty"`
	// SpaceUUID is the Mission Control UUID when known.
	SpaceUUID string `json:"space_uuid,omitempty"`
	// Label is the user-set Space name. Empty when unset.
	Label string `json:"label,omitempty"`
	// Current is true when this is the user's active Mission Control Space.
	Current  bool          `json:"current,omitempty"`
	Sessions []LiveSession `json:"sessions"`
}

// LiveSession is one iTerm pane on a Desktop.
type LiveSession struct {
	SessionID     string `json:"session_id"`
	SessionName   string `json:"session_name"`
	WindowID      string `json:"window_id"`
	WindowName    string `json:"window_name"`
	TabIndex      int    `json:"tab_index"`
	TabName       string `json:"tab_name"`
	Cwd           string `json:"cwd,omitempty"`
	Idle          *bool  `json:"idle,omitempty"`
	Note          string `json:"note,omitempty"`
	Bookmarked    bool   `json:"bookmarked"`
	SpaceIndex    int    `json:"space_index"`
	Desktop       int    `json:"desktop"`
	AgentRunner   string `json:"agent_runner,omitempty"`
	GrokSessionID string `json:"grok_session_id,omitempty"`
	// PID is the kool chosen foreground process. 0 when unknown (layout-only).
	PID int `json:"pid,omitempty"`
	// Mark is libmark.Record.Content for PID. Empty on miss (old mark binary, no file).
	Mark string `json:"mark,omitempty"`
	// agentKind / agentSessionID are snap-only join keys (any agent kind).
	agentKind      string
	agentSessionID string
}

// OrphanNote is a persisted note whose session is no longer live.
type OrphanNote struct {
	SessionID   string `json:"session_id"`
	Note        string `json:"note"`
	Bookmarked  bool   `json:"bookmarked"`
	SessionName string `json:"session_name,omitempty"`
	WindowName  string `json:"window_name,omitempty"`
	Cwd         string `json:"cwd,omitempty"`
	SpaceIndex  int    `json:"space_index"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

// NotesDocument is the versioned notes file at ~/.ai-critic/iterm-bookmarks.json.
// On-disk and Document() JSON is version 2 with a flat items list.
// Notes is an in-memory v1 index (iterm UUID → record) for Get/Update/Put.
type NotesDocument struct {
	Version int                    `json:"version"`
	Items   []*NoteItem            `json:"items,omitempty"`
	Notes   map[string]*NoteRecord `json:"notes,omitempty"`
}

// NoteItem is one v2 bookmark row (flat snake_case, no nested key).
type NoteItem struct {
	Note           string        `json:"note"`
	Bookmarked     bool          `json:"bookmarked,omitempty"`
	UpdatedAt      string        `json:"updated_at,omitempty"`
	AgentRunner    string        `json:"agent_runner,omitempty"`
	GrokSessionID  string        `json:"grok_session_id,omitempty"`
	ITermSessionID string        `json:"iterm_session_id,omitempty"`
	LastSeen       *NoteLastSeen `json:"last_seen,omitempty"`
}

// NoteRecord is one persisted note keyed by iTerm session UUID (v1 + Get/Update).
type NoteRecord struct {
	Note           string        `json:"note"`
	Bookmarked     bool          `json:"bookmarked,omitempty"`
	UpdatedAt      string        `json:"updated_at"`
	LastSeen       *NoteLastSeen `json:"last_seen,omitempty"`
	AgentRunner    string        `json:"agent_runner,omitempty"`
	GrokSessionID  string        `json:"grok_session_id,omitempty"`
	ITermSessionID string        `json:"iterm_session_id,omitempty"`
}

// NoteLastSeen caches display fields for orphan notes.
type NoteLastSeen struct {
	SessionName string `json:"session_name,omitempty"`
	WindowName  string `json:"window_name,omitempty"`
	Cwd         string `json:"cwd,omitempty"`
	SpaceIndex  int    `json:"space_index"`
}

const (
	NotesDocVersion = 2

	// InventoryPath lists spaces + live sessions + joined notes.
	// Query refresh=1 waits for a full recapture. space_id=<cgs> recaptures
	// that Desktop only and merges it into last-good.
	InventoryPath = "/api/local/iterm2/inventory"
	// InventoryStreamPath is SSE: cold seed then full capture, or warm last-good then layout-diff.
	InventoryStreamPath = "/api/local/iterm2/inventory/stream"
	// FocusPath switches Desktop then focuses an iTerm window/tab/session.
	FocusPath = "/api/local/iterm2/focus"
	// NotesPath upserts or deletes a note for a session UUID.
	NotesPath = "/api/local/iterm2/notes"
	// SpaceLabelsPath upserts or clears a Mission Control Space label.
	SpaceLabelsPath = "/api/local/iterm2/space-labels"
	// SwitchSpacePath activates a Mission Control Space by CGS space_id.
	SwitchSpacePath = "/api/local/iterm2/switch-space"

	// AccessibilityHint is returned on Space switch failure (HTTP 503).
	AccessibilityHint = "Can't switch Desktop — grant Accessibility in System Settings → Privacy & Security → Accessibility"
)
