// Package cursor provides a Go adapter for cursor-agent's stream-json output,
// exposing it through an HTTP API compatible with the frontend's chat interface.
package cursor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/settings"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

// ChatMessage follows ACP message format.
type ChatMessage struct {
	ID    string        `json:"id"`
	Role  string        `json:"role"`            // "user" or "agent"
	Parts []MessagePart `json:"parts"`
	Time  int64         `json:"time,omitempty"`   // Unix timestamp in seconds
	Model string        `json:"model,omitempty"`  // Model ID (agent messages only)
}

// MessagePart follows ACP message part format.
type MessagePart struct {
	ID          string                 `json:"id"`
	ContentType string                 `json:"content_type"`          // "text/plain", "tool/call", "tool/result", "text/thinking"
	Content     string                 `json:"content"`               // Main content text
	Name        string                 `json:"name,omitempty"`        // For tool calls: tool name
	Metadata    map[string]interface{} `json:"metadata,omitempty"`    // Additional metadata
}

// ChatSession represents a chat session with cursor-agent.
type ChatSession struct {
	ID             string `json:"id"`
	CreatedAt      string `json:"created_at"`
	FirstMessage   string `json:"firstMessage,omitempty"`
	CursorSession  string `json:"-"` // cursor-agent's session_id
	ProjectDir     string `json:"-"`
	CommandPath    string `json:"-"`
	ResumeID       string `json:"-"` // For multi-turn: cursor's session_id to resume
	Model          string `json:"-"` // model to use for cursor-agent
	adapter        *Adapter // parent adapter for global broadcast

	mu       sync.Mutex
	messages []ChatMessage
	// SSE subscribers
	subscribers map[chan SSEEvent]struct{}
	// Track if a prompt is currently running
	busy bool
}

// ACPEvent is a standard ACP SSE event sent to subscribers.
type ACPEvent struct {
	Type    string      `json:"type"`    // "acp.message.created", "acp.message.updated", "acp.message.completed"
	Message ChatMessage `json:"message"`
}

// ACP event type constants.
const (
	ACPMessageCreated   = "acp.message.created"
	ACPMessageUpdated   = "acp.message.updated"
	ACPMessageCompleted = "acp.message.completed"
)

// SSEEvent is kept as an alias for ACPEvent for internal use.
type SSEEvent = ACPEvent

// CursorModel represents a model available in cursor-agent.
type CursorModel struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default,omitempty"`
	IsCurrent bool   `json:"is_current,omitempty"`
}

// AdapterSettings holds configurable settings for the cursor adapter.
type AdapterSettings struct {
	PromptAppendMessage   string `json:"prompt_append_message"`
	FollowupAppendMessage string `json:"followup_append_message"`
}

const settingsNamespace = "cursor-agent"

// Adapter manages cursor-agent chat sessions.
type Adapter struct {
	mu               sync.Mutex
	sessions         map[string]*ChatSession
	counter          int
	projectDir       string
	cmdPath          string
	model            string // selected model ID, empty means default
	settings         AdapterSettings
	settingsStore    *settings.Store
	globalSubs       map[chan SSEEvent]struct{}
}

// NewAdapter creates a new cursor adapter for the given project directory.
// The settingsStore is used to persist adapter settings (prompt append, followup append, etc.).
func NewAdapter(projectDir string, settingsStore *settings.Store) (*Adapter, error) {
	cmdPath, err := tool_resolve.LookPath("cursor-agent")
	if err != nil {
		// Fall back to "cursor" command
		cmdPath, err = tool_resolve.LookPath("cursor")
		if err != nil {
			return nil, fmt.Errorf("cursor-agent not found: install Cursor CLI")
		}
	}
	a := &Adapter{
		sessions:      make(map[string]*ChatSession),
		projectDir:    projectDir,
		cmdPath:       cmdPath,
		settingsStore: settingsStore,
		globalSubs:    make(map[chan SSEEvent]struct{}),
	}
	// Load persisted settings
	if settingsStore != nil {
		_ = settingsStore.Load(settingsNamespace, &a.settings)
	}
	return a, nil
}

// listModels runs `cursor-agent models` and parses the output.
func (a *Adapter) listModels() ([]CursorModel, error) {
	cmd := exec.Command(a.cmdPath, "models")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	var models []CursorModel
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Available") || strings.HasPrefix(line, "Tip:") {
			continue
		}
		// Format: "model-id - Model Name  (default)" or "model-id - Model Name  (current)"
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) != 2 {
			continue
		}
		id := strings.TrimSpace(parts[0])
		nameAndFlags := strings.TrimSpace(parts[1])
		isDefault := strings.Contains(nameAndFlags, "(default)")
		isCurrent := strings.Contains(nameAndFlags, "(current)")
		name := nameAndFlags
		name = strings.ReplaceAll(name, "(default)", "")
		name = strings.ReplaceAll(name, "(current)", "")
		name = strings.TrimSpace(name)

		models = append(models, CursorModel{
			ID:        id,
			Name:      name,
			IsDefault: isDefault,
			IsCurrent: isCurrent,
		})
	}
	return models, nil
}

// SetModel sets the model to use for future prompts.
func (a *Adapter) SetModel(model string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.model = model
}

// GetModel returns the current model.
func (a *Adapter) GetModel() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.model
}

// GetSettings returns a copy of the adapter settings.
func (a *Adapter) GetSettings() AdapterSettings {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.settings
}

// SetSettings updates the adapter settings and persists them to disk.
func (a *Adapter) SetSettings(s AdapterSettings) error {
	a.mu.Lock()
	a.settings = s
	store := a.settingsStore
	a.mu.Unlock()

	if store != nil {
		return store.Save(settingsNamespace, s)
	}
	return nil
}

// globalBroadcast sends an event to all global SSE subscribers.
func (a *Adapter) globalBroadcast(event SSEEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for ch := range a.globalSubs {
		select {
		case ch <- event:
		default:
			// Drop if subscriber is slow
		}
	}
}

// GlobalSubscribe creates a new global SSE subscriber channel.
func (a *Adapter) GlobalSubscribe() chan SSEEvent {
	ch := make(chan SSEEvent, 64)
	a.mu.Lock()
	a.globalSubs[ch] = struct{}{}
	a.mu.Unlock()
	return ch
}

// GlobalUnsubscribe removes a global SSE subscriber.
func (a *Adapter) GlobalUnsubscribe(ch chan SSEEvent) {
	a.mu.Lock()
	delete(a.globalSubs, ch)
	a.mu.Unlock()
	close(ch)
}

// CreateSession creates a new chat session.
func (a *Adapter) CreateSession() *ChatSession {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.counter++
	id := fmt.Sprintf("cursor-chat-%d-%d", time.Now().UnixMilli(), a.counter)
	s := &ChatSession{
		ID:          id,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		ProjectDir:  a.projectDir,
		CommandPath: a.cmdPath,
		Model:       a.model,
		adapter:     a,
		messages:    []ChatMessage{},
		subscribers: make(map[chan SSEEvent]struct{}),
	}
	a.sessions[id] = s
	return s
}

// GetSession returns a session by ID.
func (a *Adapter) GetSession(id string) *ChatSession {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sessions[id]
}

// ListSessions returns all sessions.
func (a *Adapter) ListSessions() []map[string]string {
	a.mu.Lock()
	defer a.mu.Unlock()
	result := make([]map[string]string, 0, len(a.sessions))
	for _, s := range a.sessions {
		result = append(result, map[string]string{
			"id": s.ID,
		})
	}
	return result
}

// DeleteSession removes a session.
func (a *Adapter) DeleteSession(id string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, id)
}

// SendPrompt sends a prompt to cursor-agent and streams the response.
func (s *ChatSession) SendPrompt(prompt string) error {
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		return fmt.Errorf("session is busy processing a prompt")
	}
	s.busy = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.busy = false
		s.mu.Unlock()
	}()

	// Add user message
	now := time.Now()
	userMsg := ChatMessage{
		ID:    fmt.Sprintf("msg-%d", now.UnixMilli()),
		Role:  "user",
		Time:  now.Unix(),
		Parts: []MessagePart{{ID: fmt.Sprintf("part-%d-0", now.UnixMilli()), ContentType: "text/plain", Content: prompt}},
	}
	s.mu.Lock()
	s.messages = append(s.messages, userMsg)
	if s.FirstMessage == "" {
		s.FirstMessage = prompt
	}
	s.mu.Unlock()
	s.broadcast(ACPEvent{Type: ACPMessageCreated, Message: userMsg})

	// Build cursor-agent command
	args := []string{"agent", "--print", "--output-format", "stream-json"}
	if s.Model != "" {
		args = append(args, "--model", s.Model)
	}
	if s.ResumeID != "" {
		args = append(args, "--resume", s.ResumeID)
	}
	args = append(args, prompt)

	cmd := exec.Command(s.CommandPath, args...)
	cmd.Dir = s.ProjectDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start cursor-agent: %w", err)
	}

	// Parse the stream-json output
	s.processStream(stdout)

	// Wait for process to finish
	cmd.Wait()

	return nil
}

// processStream reads cursor-agent's stream-json output and converts events to chat messages.
func (s *ChatSession) processStream(r io.Reader) {
	scanner := bufio.NewScanner(r)
	// Increase buffer size for large outputs
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var currentAssistant *ChatMessage
	var currentToolMsgID string

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var event CursorEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		// Capture cursor session ID for resume
		if event.SessionID != "" && s.ResumeID == "" {
			s.mu.Lock()
			s.ResumeID = event.SessionID
			s.CursorSession = event.SessionID
			s.mu.Unlock()
		}

		switch event.Type {
		case "assistant":
			if event.Message == nil {
				continue
			}
			text := extractText(event.Message.Content)
			if text == "" {
				continue
			}

			now := time.Now()
			if currentAssistant == nil {
				msg := ChatMessage{
					ID:    fmt.Sprintf("msg-%d", now.UnixMilli()),
					Role:  "agent",
					Time:  now.Unix(),
					Parts: []MessagePart{{ID: fmt.Sprintf("part-%d-0", now.UnixMilli()), ContentType: "text/plain", Content: text}},
				}
				currentAssistant = &msg
				s.mu.Lock()
				s.messages = append(s.messages, msg)
				s.mu.Unlock()
				s.broadcast(ACPEvent{Type: ACPMessageCreated, Message: msg})
			} else {
				// Append to existing assistant message
				s.mu.Lock()
				idx := len(s.messages) - 1
				for i := len(s.messages) - 1; i >= 0; i-- {
					if s.messages[i].ID == currentAssistant.ID {
						idx = i
						break
					}
				}
				// Append text to last text/plain part
				appended := false
				for j := len(s.messages[idx].Parts) - 1; j >= 0; j-- {
					if s.messages[idx].Parts[j].ContentType == "text/plain" {
						s.messages[idx].Parts[j].Content += text
						appended = true
						break
					}
				}
				if !appended {
					partID := fmt.Sprintf("part-%s-%d", s.messages[idx].ID, len(s.messages[idx].Parts))
					s.messages[idx].Parts = append(s.messages[idx].Parts, MessagePart{ID: partID, ContentType: "text/plain", Content: text})
				}
				updated := s.messages[idx]
				s.mu.Unlock()
				s.broadcast(ACPEvent{Type: ACPMessageUpdated, Message: updated})
			}

		case "tool_call":
			s.handleToolCall(&event, currentAssistant, &currentToolMsgID)

		case "result":
			if currentAssistant != nil {
				s.broadcast(ACPEvent{Type: ACPMessageCompleted, Message: *currentAssistant})
			}
			currentAssistant = nil
			currentToolMsgID = ""
		}
	}
}

// handleToolCall processes a tool_call event.
func (s *ChatSession) handleToolCall(event *CursorEvent, currentAssistant *ChatMessage, currentToolMsgID *string) {
	if event.ToolCall == nil {
		return
	}

	toolName, toolArgs, toolOutput, toolStatus := parseToolCall(event.ToolCall, event.Subtype)

	if event.Subtype == "started" {
		now := time.Now()
		msgID := fmt.Sprintf("msg-%d", now.UnixMilli())
		*currentToolMsgID = msgID

		part := MessagePart{
			ID:          fmt.Sprintf("tool-%s-%s", toolName, msgID),
			ContentType: "tool/call",
			Content:     toolArgs,
			Name:        toolName,
			Metadata:    map[string]interface{}{"status": "running"},
		}

		if currentAssistant == nil {
			msg := ChatMessage{
				ID:    msgID,
				Role:  "agent",
				Time:  now.Unix(),
				Parts: []MessagePart{part},
			}
			s.mu.Lock()
			s.messages = append(s.messages, msg)
			s.mu.Unlock()
			s.broadcast(ACPEvent{Type: ACPMessageUpdated, Message: msg})
		} else {
			s.mu.Lock()
			for i := len(s.messages) - 1; i >= 0; i-- {
				if s.messages[i].ID == currentAssistant.ID {
					s.messages[i].Parts = append(s.messages[i].Parts, part)
					updated := s.messages[i]
					s.mu.Unlock()
					s.broadcast(ACPEvent{Type: ACPMessageUpdated, Message: updated})
					return
				}
			}
			s.mu.Unlock()
		}

	} else if event.Subtype == "completed" {
		// Update the tool call status
		s.mu.Lock()
		var updatedMsg *ChatMessage
		for i := len(s.messages) - 1; i >= 0; i-- {
			for j := len(s.messages[i].Parts) - 1; j >= 0; j-- {
				p := &s.messages[i].Parts[j]
				if p.ContentType == "tool/call" && p.Name == toolName && p.Metadata != nil && p.Metadata["status"] == "running" {
					p.Metadata["status"] = toolStatus
					if toolOutput != "" {
						p.Metadata["output"] = toolOutput
					}
					msg := s.messages[i]
					updatedMsg = &msg
					break
				}
			}
			if updatedMsg != nil {
				break
			}
		}
		s.mu.Unlock()
		if updatedMsg != nil {
			s.broadcast(ACPEvent{Type: ACPMessageUpdated, Message: *updatedMsg})
		}
	}
}

// parseToolCall extracts tool name, args, output and status from a raw tool_call JSON.
func parseToolCall(raw json.RawMessage, subtype string) (name, args, output, status string) {
	// Try to parse as a map to find the tool type
	var toolMap map[string]json.RawMessage
	if err := json.Unmarshal(raw, &toolMap); err != nil {
		return "unknown", "", "", "error"
	}

	for key, val := range toolMap {
		name = toolCallKeyToName(key)

		if subtype == "started" {
			// Extract args
			var data struct {
				Args json.RawMessage `json:"args"`
			}
			json.Unmarshal(val, &data)
			if data.Args != nil {
				args = string(data.Args)
			}
		} else if subtype == "completed" {
			// Extract result
			var data struct {
				Result json.RawMessage `json:"result"`
			}
			json.Unmarshal(val, &data)
			status = "completed"
			if data.Result != nil {
				// Check for success/error/rejected
				var result map[string]json.RawMessage
				json.Unmarshal(data.Result, &result)
				if _, ok := result["success"]; ok {
					status = "completed"
					output = summarizeResult(name, result["success"])
				} else if _, ok := result["rejected"]; ok {
					status = "error"
					output = "Rejected"
				} else {
					status = "error"
					output = "Failed"
				}
			}
		}
		break // Only process the first key
	}
	return
}

// toolCallKeyToName maps cursor-agent tool call keys to human-readable names.
func toolCallKeyToName(key string) string {
	switch key {
	case "shellToolCall", "bashToolCall":
		return "shell"
	case "readToolCall":
		return "read_file"
	case "editToolCall":
		return "edit_file"
	case "writeToolCall":
		return "write_file"
	case "deleteToolCall":
		return "delete_file"
	case "grepToolCall":
		return "grep"
	case "globToolCall":
		return "glob"
	case "lsToolCall":
		return "list_dir"
	case "todoToolCall", "updateTodosToolCall":
		return "todo"
	default:
		return strings.TrimSuffix(key, "ToolCall")
	}
}

// summarizeResult creates a brief summary of a tool result.
func summarizeResult(toolName string, successRaw json.RawMessage) string {
	switch toolName {
	case "shell":
		var s ShellSuccess
		json.Unmarshal(successRaw, &s)
		if s.Output != "" {
			if len(s.Output) > 200 {
				return fmt.Sprintf("Exit %d: %s...", s.ExitCode, s.Output[:200])
			}
			return fmt.Sprintf("Exit %d: %s", s.ExitCode, s.Output)
		}
		return fmt.Sprintf("Exit code: %d", s.ExitCode)
	case "read_file":
		var s ReadSuccess
		json.Unmarshal(successRaw, &s)
		return fmt.Sprintf("Read %d lines", s.TotalLines)
	case "write_file":
		var s WriteSuccess
		json.Unmarshal(successRaw, &s)
		return fmt.Sprintf("Wrote %d lines (%d bytes)", s.LinesCreated, s.FileSize)
	case "edit_file":
		return "Edited successfully"
	case "grep":
		return "Search completed"
	case "glob":
		var s GlobSuccess
		json.Unmarshal(successRaw, &s)
		return fmt.Sprintf("Found %d files", s.TotalFiles)
	default:
		return "Completed"
	}
}

// extractText concatenates text content from content blocks.
func extractText(blocks []CursorContentBlock) string {
	var sb strings.Builder
	for _, b := range blocks {
		if b.Type == "text" {
			sb.WriteString(b.Text)
		}
	}
	return sb.String()
}

// broadcast sends an event to all SSE subscribers (per-session and global).
func (s *ChatSession) broadcast(event SSEEvent) {
	s.mu.Lock()
	for ch := range s.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
	s.mu.Unlock()
	// Also broadcast to global adapter subscribers
	if s.adapter != nil {
		s.adapter.globalBroadcast(event)
	}
}

// Subscribe creates a new SSE subscriber channel.
func (s *ChatSession) Subscribe() chan SSEEvent {
	ch := make(chan SSEEvent, 64)
	s.mu.Lock()
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()
	return ch
}

// Unsubscribe removes an SSE subscriber.
func (s *ChatSession) Unsubscribe(ch chan SSEEvent) {
	s.mu.Lock()
	delete(s.subscribers, ch)
	s.mu.Unlock()
	close(ch)
}

// GetMessages returns all messages in the session.
func (s *ChatSession) GetMessages() []ChatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]ChatMessage, len(s.messages))
	copy(result, s.messages)
	return result
}

// RegisterHandlers registers HTTP handlers for the cursor adapter on the given mux.
// prefix should be like "/api/agents/sessions/{sessionID}/proxy"
func (a *Adapter) RegisterHandlers(mux *http.ServeMux) {
	// These are handled via the proxy path in agents.go, not registered directly.
	// Instead, ServeHTTP is called from the proxy handler.
}

// ServeHTTP handles proxied requests from the agent session proxy.
func (a *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case path == "/session" && r.Method == http.MethodGet:
		a.handleListSessions(w, r)
	case path == "/session" && r.Method == http.MethodPost:
		a.handleCreateSession(w, r)
	case strings.HasPrefix(path, "/session/") && strings.HasSuffix(path, "/message") && r.Method == http.MethodGet:
		sessionID := extractSessionID(path, "/message")
		a.handleGetMessages(w, r, sessionID)
	case strings.HasPrefix(path, "/session/") && strings.HasSuffix(path, "/prompt_async") && r.Method == http.MethodPost:
		sessionID := extractSessionID(path, "/prompt_async")
		a.handlePromptAsync(w, r, sessionID)
	case path == "/event" || path == "/global/event":
		a.handleEvents(w, r)
	case path == "/global/health" || path == "/health":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	case path == "/config" && r.Method == http.MethodPatch:
		a.handleConfigUpdate(w, r)
	case path == "/config":
		a.handleConfig(w, r)
	case path == "/config/providers":
		a.handleConfigProviders(w, r)
	case path == "/settings" && r.Method == http.MethodGet:
		a.handleGetSettings(w, r)
	case path == "/settings" && r.Method == http.MethodPut:
		a.handleUpdateSettings(w, r)
	case path == "/templates" && r.Method == http.MethodGet:
		a.handleListTemplates(w, r)
	default:
		http.NotFound(w, r)
	}
}

func extractSessionID(path, suffix string) string {
	// path: /session/{id}/suffix
	path = strings.TrimPrefix(path, "/session/")
	path = strings.TrimSuffix(path, suffix)
	return strings.TrimSuffix(path, "/")
}

func (a *Adapter) handleListSessions(w http.ResponseWriter, _ *http.Request) {
	sessions := a.ListSessions()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessions)
}

func (a *Adapter) handleCreateSession(w http.ResponseWriter, _ *http.Request) {
	s := a.CreateSession()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"id":         s.ID,
		"created_at": s.CreatedAt,
	})
}

func (a *Adapter) handleGetMessages(w http.ResponseWriter, _ *http.Request, sessionID string) {
	s := a.GetSession(sessionID)
	if s == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	messages := s.GetMessages()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func (a *Adapter) handlePromptAsync(w http.ResponseWriter, r *http.Request, sessionID string) {
	s := a.GetSession(sessionID)
	if s == nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	var req struct {
		Content string `json:"content"`
		Parts   []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"parts"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Extract prompt text from either content or parts
	prompt := req.Content
	if prompt == "" && len(req.Parts) > 0 {
		for _, p := range req.Parts {
			if p.Type == "text" && p.Text != "" {
				prompt = p.Text
				break
			}
		}
	}
	if prompt == "" {
		http.Error(w, "empty prompt", http.StatusBadRequest)
		return
	}

	// Append prompt append message from settings
	settings := a.GetSettings()
	if settings.PromptAppendMessage != "" {
		prompt += "\n" + settings.PromptAppendMessage
	}

	// Run prompt asynchronously
	go s.SendPrompt(prompt)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (a *Adapter) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Use global subscription to receive events from all sessions
	ch := a.GlobalSubscribe()
	defer a.GlobalUnsubscribe(ch)

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(event)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (a *Adapter) handleConfigProviders(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	models, err := a.listModels()
	if err != nil {
		// Fallback to a single default model
		models = []CursorModel{{ID: "auto", Name: "Auto", IsDefault: true}}
	}

	// Build models map
	modelsMap := make(map[string]interface{})
	defaultModel := ""
	for _, m := range models {
		modelsMap[m.ID] = map[string]interface{}{
			"id":         m.ID,
			"name":       m.Name,
			"is_default": m.IsDefault,
			"is_current": m.IsCurrent,
			"limit": map[string]int{
				"context": 200000,
				"output":  8192,
			},
		}
		if m.IsDefault && defaultModel == "" {
			defaultModel = m.ID
		}
	}
	if defaultModel == "" && len(models) > 0 {
		defaultModel = models[0].ID
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": []map[string]interface{}{
			{
				"id":     "cursor",
				"name":   "Cursor",
				"models": modelsMap,
			},
		},
		"default": map[string]string{
			"cursor": defaultModel,
		},
	})
}

func (a *Adapter) handleConfig(w http.ResponseWriter, _ *http.Request) {
	currentModel := a.GetModel()
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{
		"name":    "Cursor Agent",
		"version": "1.0.0",
		"capabilities": map[string]bool{
			"chat":       true,
			"streaming":  true,
			"tool_calls": true,
			"file_edit":  true,
			"shell_exec": true,
			"cancel":     false,
		},
	}
	if currentModel != "" {
		resp["model"] = map[string]string{
			"modelID":    currentModel,
			"providerID": "cursor",
		}
	}
	json.NewEncoder(w).Encode(resp)
}

func (a *Adapter) handleConfigUpdate(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Model struct {
			ModelID string `json:"modelID"`
		} `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.SetModel(body.Model.ModelID)

	// Also update existing sessions to use the new model
	a.mu.Lock()
	for _, s := range a.sessions {
		s.Model = body.Model.ModelID
	}
	a.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "model": body.Model.ModelID})
}

func (a *Adapter) handleGetSettings(w http.ResponseWriter, _ *http.Request) {
	settings := a.GetSettings()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func (a *Adapter) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var s AdapterSettings
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := a.SetSettings(s); err != nil {
		http.Error(w, fmt.Sprintf("failed to save settings: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}
