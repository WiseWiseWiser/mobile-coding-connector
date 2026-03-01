package cursor_acp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/xhd2015/lifelog-private/ai-critic/server/agents/acp"
)

// CursorAgent implements acp.Agent using cursor-agent CLI's native
// --print --output-format stream-json interface.
//
// cursor-agent stream-json output format reference (--print --output-format stream-json --stream-partial-output):
//
//	{"type":"system","subtype":"init","apiKeySource":"login","cwd":"/path/to/workspace","session_id":"<uuid>","model":"Claude 4.6 Opus","permissionMode":"default"}
//	{"type":"user","message":{"role":"user","content":[{"type":"text","text":"<prompt>"}]},"session_id":"<uuid>"}
//	{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Hello"}]},"session_id":"<uuid>","timestamp_ms":1772275089783}
//	{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":" to"}]},"session_id":"<uuid>","timestamp_ms":1772275089815}
//	{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":" you!"}]},"session_id":"<uuid>","timestamp_ms":1772275089865}
//	{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Hello to you!"}]},"session_id":"<uuid>"}
//	{"type":"result","subtype":"success","duration_ms":7372,"duration_api_ms":7372,"is_error":false,"result":"Hello to you!","session_id":"<uuid>","request_id":"<uuid>"}
//
// Notes:
// - Streaming chunks (with timestamp_ms) contain text deltas, not accumulated text
// - Final assistant message (without timestamp_ms) contains the full accumulated text
// - The --resume flag MUST use = syntax: --resume=<chatID> (space-separated doesn't work)
type CursorAgent struct {
	mu           sync.Mutex
	chatID       string
	model        string
	promptCmd    *exec.Cmd
	updates      chan acp.SessionUpdate
	cwd          string
	sessionStore *acp.SessionStore
	messageStore *acp.MessageStore
	debug        bool
	// pendingPrompt stores the last prompt text for retry after trust confirmation
	pendingPrompt string
}

var _ acp.Agent = (*CursorAgent)(nil)

const sessionsFile = ".ai-critic/acp/cursor/sessions.json"

const messagesDir = ".ai-critic/acp/cursor/messages"

func NewCursorAgent() *CursorAgent {
	return &CursorAgent{
		updates:      make(chan acp.SessionUpdate, 256),
		sessionStore: acp.NewSessionStore(sessionsFile),
		messageStore: acp.NewMessageStore(messagesDir),
	}
}

func (a *CursorAgent) Name() string {
	return "Cursor"
}

func (a *CursorAgent) IsConnected() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.chatID != ""
}

func (a *CursorAgent) SessionID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.chatID
}

func (a *CursorAgent) Updates() <-chan acp.SessionUpdate {
	return a.updates
}

func (a *CursorAgent) Sessions() []acp.SessionEntry {
	return a.sessionStore.Load()
}

func (a *CursorAgent) GetMessages(sessionID string) (json.RawMessage, error) {
	return a.messageStore.Load(sessionID)
}

func (a *CursorAgent) SaveMessages(sessionID string, messages json.RawMessage) error {
	return a.messageStore.Save(sessionID, messages)
}

func (a *CursorAgent) UpdateSessionModel(sessionID, model string) {
	a.sessionStore.UpdateModel(sessionID, model)
}

func (a *CursorAgent) Models() ([]acp.ModelInfo, error) {
	agentPath, err := resolveAgentPath()
	if err != nil {
		return nil, fmt.Errorf("cursor-agent not found: %w", err)
	}
	cmd := cursorCommand(agentPath, "models")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}
	return parseModels(string(out)), nil
}

func (a *CursorAgent) Status() acp.StatusInfo {
	cwd, _ := os.Getwd()
	_, err := resolveAgentPath()
	available := err == nil

	a.mu.Lock()
	model := a.model
	a.mu.Unlock()

	info := acp.StatusInfo{
		Available: available,
		Connected: a.IsConnected(),
		SessionID: a.SessionID(),
		CWD:       cwd,
		Model:     model,
	}
	if !available {
		info.Message = "cursor-agent not found in PATH. Install Cursor CLI: curl https://cursor.com/install -fsSL | bash"
	}
	return info
}

func (a *CursorAgent) Connect(cwd string, resumeSessionID string, debug bool, log acp.LogFunc) (string, error) {
	a.mu.Lock()
	if a.chatID != "" {
		a.mu.Unlock()
		return "", fmt.Errorf("already connected")
	}
	a.mu.Unlock()

	a.mu.Lock()
	a.debug = debug
	a.mu.Unlock()

	if log == nil {
		log = func(string) {}
	}

	log("Looking up cursor-agent...")
	agentPath, err := resolveAgentPath()
	if err != nil {
		return "", fmt.Errorf("cursor-agent not found: %w", err)
	}
	log(fmt.Sprintf("Found cursor-agent at %s", agentPath))

	log("Checking authentication...")
	statusCmd := cursorCommand(agentPath, "status")
	statusOut, err := statusCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cursor-agent not authenticated. Run 'cursor-agent login' first: %w", err)
	}
	statusStr := strings.TrimSpace(string(statusOut))
	log(statusStr)

	log("Detecting model...")
	modelsCmd := cursorCommand(agentPath, "models")
	modelsOut, _ := modelsCmd.Output()
	model := parseCurrentModel(string(modelsOut))
	if model != "" {
		log(fmt.Sprintf("Model: %s", model))
	}

	var chatID string
	if resumeSessionID != "" {
		chatID = resumeSessionID
		log(fmt.Sprintf("Resuming session: %s", chatID))
		if entry := a.sessionStore.Get(resumeSessionID); entry != nil {
			if entry.Model != "" {
				model = entry.Model
				log(fmt.Sprintf("Loaded stored model: %s", model))
			}
			if entry.CWD != "" && cwd == "" {
				cwd = entry.CWD
				log(fmt.Sprintf("Loaded stored cwd: %s", cwd))
			}
		}
	} else {
		log("Creating chat session...")
		createCmd := cursorCommand(agentPath, "create-chat")
		if cwd != "" {
			createCmd.Dir = cwd
		}
		chatOut, err := createCmd.Output()
		if err != nil {
			return "", fmt.Errorf("failed to create chat: %w", err)
		}
		chatID = strings.TrimSpace(string(chatOut))
		if chatID == "" {
			return "", fmt.Errorf("empty chat ID returned")
		}
		log(fmt.Sprintf("Chat session created: %s", chatID))

		a.sessionStore.Add(acp.NewSessionEntry(chatID, model, "cursor", cwd))
	}

	a.mu.Lock()
	a.chatID = chatID
	a.model = model
	a.cwd = cwd
	a.mu.Unlock()

	return chatID, nil
}

func (a *CursorAgent) Disconnect() {
	a.mu.Lock()
	a.chatID = ""
	a.cwd = ""
	cmd := a.promptCmd
	a.promptCmd = nil
	a.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		cmd.Process.Kill()
		cmd.Wait()
	}
}

func (a *CursorAgent) SendPrompt(sessionID string, text string, model string) (*acp.PromptResult, error) {
	return a.sendPromptInternal(sessionID, text, model, false)
}

// sendPromptInternal is the internal implementation that supports trust-aware retries
func (a *CursorAgent) sendPromptInternal(sessionID string, text string, model string, trustEnabled bool) (*acp.PromptResult, error) {
	a.mu.Lock()
	if a.chatID == "" || a.chatID != sessionID {
		a.mu.Unlock()
		return nil, fmt.Errorf("invalid session")
	}
	cwd := a.cwd
	debug := a.debug
	a.mu.Unlock()

	sendDebug := func(msg string) {
		if debug {
			select {
			case a.updates <- acp.SessionUpdate{Type: "debug", Message: msg}:
			default:
			}
		}
	}

	sendDebug(fmt.Sprintf("SendPrompt: sessionID=%s cwd=%s debug=%v model=%s trustEnabled=%v", sessionID, cwd, debug, model, trustEnabled))

	if model != "" {
		a.sessionStore.UpdateModel(sessionID, model)
	} else {
		if entry := a.sessionStore.Get(sessionID); entry != nil && entry.Model != "" {
			model = entry.Model
		}
	}

	agentPath, err := resolveAgentPath()
	if err != nil {
		return nil, fmt.Errorf("cursor-agent not found: %w", err)
	}

	args := []string{
		"--print",
		"--output-format", "stream-json",
		"--stream-partial-output",
		"--resume=" + sessionID,
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	// Add trust flag if trust is enabled for this session
	if trustEnabled {
		args = append(args, "--trust")
	}
	args = append(args, text)
	cmd := cursorCommand(agentPath, args...)
	if cwd != "" {
		cmd.Dir = cwd
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start cursor-agent: %w", err)
	}

	a.mu.Lock()
	a.promptCmd = cmd
	a.mu.Unlock()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	trustPromptDetected := false
	var fullOutput strings.Builder

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		// Check for trust prompt in raw output
		lineStr := string(line)
		fullOutput.WriteString(lineStr)
		fullOutput.WriteString("\n")

		// Detect trust prompt patterns
		if !trustEnabled && !trustPromptDetected {
			if strings.Contains(lineStr, "Workspace Trust Required") ||
				strings.Contains(lineStr, "trust workspace") ||
				strings.Contains(lineStr, "trust this workspace") ||
				strings.Contains(lineStr, "--trust") {
				trustPromptDetected = true
				// Send trust prompt update to client
				select {
				case a.updates <- acp.SessionUpdate{
					Type:    "trust_prompt",
					Message: "Workspace trust is required to continue. Enable trust for this session?",
				}:
				default:
				}
				// Kill the command
				cmd.Process.Kill()
				cmd.Wait()
				a.mu.Lock()
				a.promptCmd = nil
				// Store the pending prompt for retry
				a.pendingPrompt = text
				a.mu.Unlock()
				return nil, fmt.Errorf("trust_prompt")
			}
		}

		var msg cursorMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		update := parseMessage(msg)
		if update != nil {
			if debug {
				select {
				case a.updates <- acp.SessionUpdate{Type: "debug", Message: fmt.Sprintf("update: %s", string(line))}:
				default:
				}
			}
			select {
			case a.updates <- *update:
			default:
			}
		}
	}

	err = cmd.Wait()
	a.mu.Lock()
	a.promptCmd = nil
	a.mu.Unlock()

	if err != nil {
		return &acp.PromptResult{StopReason: "error"}, nil
	}
	return &acp.PromptResult{StopReason: "end_turn"}, nil
}

// RetryPromptWithTrust retries the last prompt with trust enabled
func (a *CursorAgent) RetryPromptWithTrust(sessionID string) {
	a.mu.Lock()
	pendingPrompt := a.pendingPrompt
	a.pendingPrompt = ""
	a.mu.Unlock()

	if pendingPrompt == "" {
		return
	}

	// Retry with trust enabled
	go func() {
		_, _ = a.sendPromptInternal(sessionID, pendingPrompt, "", true)
	}()
}

func (a *CursorAgent) Cancel(sessionID string) error {
	a.mu.Lock()
	cmd := a.promptCmd
	a.mu.Unlock()

	if cmd != nil && cmd.Process != nil {
		return cmd.Process.Kill()
	}
	return nil
}

type cursorMessage struct {
	Type      string `json:"type"`
	Subtype   string `json:"subtype,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Message   *struct {
		Role    string `json:"role"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text,omitempty"`
		} `json:"content"`
	} `json:"message,omitempty"`
	TimestampMs *int64 `json:"timestamp_ms,omitempty"`
	Result      string `json:"result,omitempty"`
	IsError     bool   `json:"is_error,omitempty"`
	DurationMs  int64  `json:"duration_ms,omitempty"`
	Model       string `json:"model,omitempty"`
}

// resolveAgentPath returns the cursor-agent binary path, preferring:
// 1. Configured path from settings
// 2. "cursor-agent" in PATH
// 3. "agent" in PATH (verified by checking help output for "Start the Cursor Agent")
func resolveAgentPath() (string, error) {
	settings, _ := LoadSettings()
	if settings != nil && settings.BinaryPath != "" {
		if _, err := os.Stat(settings.BinaryPath); err == nil {
			return settings.BinaryPath, nil
		}
		return "", fmt.Errorf("configured binary path does not exist: %s", settings.BinaryPath)
	}

	if path, err := exec.LookPath("cursor-agent"); err == nil {
		return path, nil
	}

	if path, err := exec.LookPath("agent"); err == nil {
		out, err := exec.Command(path, "--help").CombinedOutput()
		if err == nil && strings.Contains(string(out), "Start the Cursor Agent") {
			return path, nil
		}
	}

	return "", fmt.Errorf("cursor-agent not found in PATH (tried 'cursor-agent' and 'agent')")
}

// cursorCommand creates an exec.Cmd for cursor-agent with the API key from settings if configured.
func cursorCommand(agentPath string, args ...string) *exec.Cmd {
	settings, _ := LoadSettings()
	if settings != nil && settings.APIKey != "" {
		args = append([]string{"--api-key", settings.APIKey}, args...)
	}
	return exec.Command(agentPath, args...)
}

// parseCurrentModel extracts the model marked as "(current)" from `cursor-agent models` output.
// Each line looks like: "opus-4.6 - Claude 4.6 Opus  (current)"
func parseCurrentModel(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "(current)") {
			continue
		}
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) == 2 {
			displayName := strings.TrimSpace(parts[1])
			displayName = strings.TrimSuffix(displayName, "(current)")
			return strings.TrimSpace(displayName)
		}
	}
	return ""
}

// parseModels parses all models from `cursor-agent models` output.
func parseModels(output string) []acp.ModelInfo {
	var models []acp.ModelInfo
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		isCurrent := strings.Contains(line, "(current)")
		line = strings.ReplaceAll(line, "(current)", "")
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) != 2 {
			continue
		}
		id := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		if id == "" {
			continue
		}
		models = append(models, acp.ModelInfo{
			ID:        id,
			Name:      name,
			Provider:  "cursor",
			ProviderN: "Cursor",
			IsCurrent: isCurrent,
		})
	}
	return models
}

func parseMessage(msg cursorMessage) *acp.SessionUpdate {
	switch msg.Type {
	case "system":
		if msg.Subtype == "init" {
			return &acp.SessionUpdate{
				Type:  "session_info",
				Model: msg.Model,
			}
		}
		return nil

	case "assistant":
		if msg.Message == nil || len(msg.Message.Content) == 0 {
			return nil
		}
		text := msg.Message.Content[0].Text
		if msg.TimestampMs != nil {
			// Streaming chunk
			return &acp.SessionUpdate{
				Type: "agent_message_chunk",
				Text: text,
			}
		}
		// Final full message (no timestamp) - skip since we already streamed chunks
		return nil

	case "result":
		return &acp.SessionUpdate{
			Type:    "done",
			Message: msg.Result,
		}

	default:
		return nil
	}
}
