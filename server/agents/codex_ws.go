package agents

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

var codexWSUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type codexWSClientMessage struct {
	Type           string `json:"type"`
	Prompt         string `json:"prompt,omitempty"`
	ProjectDir     string `json:"project_dir,omitempty"`
	Model          string `json:"model,omitempty"`
	Sandbox        string `json:"sandbox,omitempty"`
	ApprovalPolicy string `json:"approval_policy,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
	NewSession     bool   `json:"new_session,omitempty"`
}

type codexWSConnection struct {
	conn *websocket.Conn

	writeMu sync.Mutex
	mu      sync.Mutex
	run     *codexActiveRun
	session string
}

type codexRunRegistry struct {
	mu        sync.Mutex
	bySession map[string]*codexActiveRun
}

type codexActiveRun struct {
	mu         sync.Mutex
	sessionID  string
	projectDir string
	command    []string
	resume     bool
	startedAt  time.Time
	done       bool
	cancel     context.CancelFunc
	cmd        *exec.Cmd
	clients    map[*codexWSConnection]struct{}
	events     []map[string]any
}

var codexRuns = &codexRunRegistry{bySession: make(map[string]*codexActiveRun)}

type codexModelResponse struct {
	Models       []codexModel `json:"models"`
	CurrentModel string       `json:"current_model,omitempty"`
}

type codexModel struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Description           string   `json:"description,omitempty"`
	DefaultReasoningLevel string   `json:"default_reasoning_level,omitempty"`
	ReasoningLevels       []string `json:"reasoning_levels,omitempty"`
}

type codexSessionResponse struct {
	Sessions []codexSession `json:"sessions"`
}

type codexSessionMessagesResponse struct {
	Messages []codexHistoryMessage `json:"messages"`
}

type codexSession struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	ProjectDir string `json:"project_dir,omitempty"`
	Model      string `json:"model,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
	Path       string `json:"path,omitempty"`
}

type codexHistoryMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
	Time string `json:"time,omitempty"`
}

func handleCodexWebSocket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	conn, err := codexWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}

	client := &codexWSConnection{conn: conn}
	defer func() {
		client.detachActive()
		conn.Close()
	}()

	client.sendJSON(map[string]any{"type": "ready"})

	for {
		var msg codexWSClientMessage
		if err := conn.ReadJSON(&msg); err != nil {
			return
		}

		switch msg.Type {
		case "prompt":
			if err := client.startPrompt(r.Context(), msg); err != nil {
				client.sendJSON(map[string]any{"type": "error", "message": err.Error()})
			}
		case "attach":
			if err := client.attachSession(msg); err != nil {
				client.sendJSON(map[string]any{"type": "error", "message": err.Error()})
			}
		case "cancel":
			client.cancelActive()
		default:
			client.sendJSON(map[string]any{"type": "error", "message": "unknown message type"})
		}
	}
}

func handleCodexModels(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cmdPath, err := getAgentBinaryPath(AgentIDCodex, "codex")
	if err != nil {
		http.Error(w, fmt.Sprintf("codex not found: %v", err), http.StatusInternalServerError)
		return
	}

	cmd := exec.Command(cmdPath, "debug", "models")
	cmd.Env = tool_resolve.AppendExtraPaths(os.Environ())
	out, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list codex models: %v", err), http.StatusInternalServerError)
		return
	}

	var catalog struct {
		Models []struct {
			Slug                  string `json:"slug"`
			DisplayName           string `json:"display_name"`
			Description           string `json:"description"`
			DefaultReasoningLevel string `json:"default_reasoning_level"`
			Visibility            string `json:"visibility"`
			SupportedReasoning    []struct {
				Effort string `json:"effort"`
			} `json:"supported_reasoning_levels"`
		} `json:"models"`
	}
	if err := json.Unmarshal(out, &catalog); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse codex models: %v", err), http.StatusInternalServerError)
		return
	}

	models := make([]codexModel, 0, len(catalog.Models))
	for _, raw := range catalog.Models {
		if raw.Slug == "" || (raw.Visibility != "" && raw.Visibility != "list") {
			continue
		}
		reasoningLevels := make([]string, 0, len(raw.SupportedReasoning))
		for _, level := range raw.SupportedReasoning {
			if level.Effort != "" {
				reasoningLevels = append(reasoningLevels, level.Effort)
			}
		}
		name := raw.DisplayName
		if name == "" {
			name = raw.Slug
		}
		models = append(models, codexModel{
			ID:                    raw.Slug,
			Name:                  name,
			Description:           raw.Description,
			DefaultReasoningLevel: raw.DefaultReasoningLevel,
			ReasoningLevels:       reasoningLevels,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(codexModelResponse{
		Models:       models,
		CurrentModel: readCodexCurrentModel(),
	})
}

func handleCodexSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectDir := strings.TrimSpace(r.URL.Query().Get("project_dir"))
	sessions, err := listCodexSessions(projectDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(codexSessionResponse{Sessions: sessions})
}

func handleCodexSessionMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	path, err := findCodexSessionPath(sessionID)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	messages, err := readCodexSessionMessages(path)
	if err != nil {
		http.Error(w, fmt.Sprintf("read session messages: %v", err), http.StatusInternalServerError)
		return
	}
	if len(messages) > 10 {
		messages = messages[len(messages)-10:]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(codexSessionMessagesResponse{Messages: messages})
}

func (c *codexWSConnection) startPrompt(_ context.Context, msg codexWSClientMessage) error {
	prompt := strings.TrimSpace(msg.Prompt)
	if prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	projectDir := strings.TrimSpace(msg.ProjectDir)
	if projectDir == "" {
		return fmt.Errorf("project directory is required")
	}
	if info, err := os.Stat(projectDir); err != nil || !info.IsDir() {
		return fmt.Errorf("invalid project directory: %s", projectDir)
	}

	cmdPath, err := getAgentBinaryPath(AgentIDCodex, "codex")
	if err != nil {
		return fmt.Errorf("codex not found: %w", err)
	}

	if run := c.activeRun(); run != nil && run.isRunning() {
		return fmt.Errorf("codex is already processing a prompt")
	}
	sessionID := strings.TrimSpace(msg.SessionID)
	if sessionID == "" && !msg.NewSession {
		sessionID = c.session
	}
	if sessionID != "" && !msg.NewSession {
		if run := codexRuns.get(sessionID); run != nil {
			run.addClient(c, true, true)
			return nil
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	run := &codexActiveRun{
		sessionID:  sessionID,
		projectDir: projectDir,
		resume:     sessionID != "",
		startedAt:  time.Now(),
		cancel:     cancel,
		clients:    make(map[*codexWSConnection]struct{}),
	}
	if sessionID != "" {
		codexRuns.register(sessionID, run)
	}
	run.addClient(c, false, false)
	go run.runCodex(ctx, cmdPath, projectDir, prompt, sessionID, msg)
	return nil
}

func (c *codexWSConnection) attachSession(msg codexWSClientMessage) error {
	sessionID := strings.TrimSpace(msg.SessionID)
	if sessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	c.mu.Lock()
	c.session = sessionID
	c.mu.Unlock()

	run := codexRuns.get(sessionID)
	if run == nil {
		c.detachActive()
		c.mu.Lock()
		c.session = sessionID
		c.mu.Unlock()
		c.sendJSON(map[string]any{
			"type":       "attached",
			"session_id": sessionID,
			"running":    false,
		})
		return nil
	}
	run.addClient(c, true, true)
	return nil
}

func listCodexSessions(projectDir string) ([]codexSession, error) {
	codexHome := codexHomeDir()
	history := readCodexHistory(filepath.Join(codexHome, "history.jsonl"))
	sessionRoot := filepath.Join(codexHome, "sessions")
	entries := make([]codexSession, 0)

	err := filepath.WalkDir(sessionRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			return nil
		}

		session, ok := readCodexSessionFile(path)
		if !ok {
			return nil
		}
		if projectDir != "" && session.ProjectDir != "" && filepath.Clean(session.ProjectDir) != filepath.Clean(projectDir) {
			return nil
		}
		if session.ID == "" {
			session.ID = codexSessionIDFromPath(path)
		}
		if session.ID == "" {
			return nil
		}
		if prompt, ok := history[session.ID]; ok {
			session.Title = prompt.Text
			if prompt.Timestamp > 0 {
				session.UpdatedAt = time.Unix(prompt.Timestamp, 0).Format(time.RFC3339)
			}
		}
		if session.Title == "" {
			session.Title = session.ID
		}
		if info, err := os.Stat(path); err == nil && session.UpdatedAt == "" {
			session.UpdatedAt = info.ModTime().Format(time.RFC3339)
		}
		session.Path = path
		entries = append(entries, session)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("read codex sessions: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].UpdatedAt > entries[j].UpdatedAt
	})
	if len(entries) > 100 {
		entries = entries[:100]
	}
	return entries, nil
}

func findCodexSessionPath(sessionID string) (string, error) {
	sessionRoot := filepath.Join(codexHomeDir(), "sessions")
	var found string
	err := filepath.WalkDir(sessionRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			return nil
		}
		if codexSessionIDFromPath(path) == sessionID {
			found = path
			return nil
		}
		session, ok := readCodexSessionFile(path)
		if ok && session.ID == sessionID {
			found = path
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", os.ErrNotExist
	}
	return found, nil
}

func readCodexSessionMessages(path string) ([]codexHistoryMessage, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	messages := make([]codexHistoryMessage, 0)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		message, ok := parseCodexHistoryMessage(scanner.Bytes())
		if !ok {
			continue
		}
		if len(messages) > 0 {
			last := messages[len(messages)-1]
			if last.Role == message.Role && last.Text == message.Text {
				continue
			}
		}
		messages = append(messages, message)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return messages, nil
}

func parseCodexHistoryMessage(line []byte) (codexHistoryMessage, bool) {
	var item struct {
		Type      string          `json:"type"`
		Timestamp string          `json:"timestamp"`
		Role      string          `json:"role"`
		Content   json.RawMessage `json:"content"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(line, &item); err != nil {
		return codexHistoryMessage{}, false
	}

	if item.Type == "event_msg" {
		var payload struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(item.Payload, &payload); err != nil {
			return codexHistoryMessage{}, false
		}
		role := ""
		switch payload.Type {
		case "user_message":
			role = "user"
		case "agent_message":
			role = "assistant"
		}
		text := strings.TrimSpace(payload.Message)
		if role == "" || text == "" {
			return codexHistoryMessage{}, false
		}
		return codexHistoryMessage{Role: role, Text: text, Time: item.Timestamp}, true
	}

	if item.Type == "response_item" {
		var payload struct {
			Type    string          `json:"type"`
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal(item.Payload, &payload); err != nil || payload.Type != "message" {
			return codexHistoryMessage{}, false
		}
		role := normalizeCodexHistoryRole(payload.Role)
		if role != "assistant" {
			return codexHistoryMessage{}, false
		}
		text := extractCodexHistoryText(payload.Content)
		if text == "" {
			return codexHistoryMessage{}, false
		}
		return codexHistoryMessage{Role: role, Text: text, Time: item.Timestamp}, true
	}

	if item.Type == "message" {
		role := normalizeCodexHistoryRole(item.Role)
		text := extractCodexHistoryText(item.Content)
		if role == "" || text == "" {
			return codexHistoryMessage{}, false
		}
		return codexHistoryMessage{Role: role, Text: text, Time: item.Timestamp}, true
	}

	return codexHistoryMessage{}, false
}

func normalizeCodexHistoryRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "user", "human":
		return "user"
	case "assistant", "agent":
		return "assistant"
	default:
		return ""
	}
}

func extractCodexHistoryText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var direct string
	if err := json.Unmarshal(raw, &direct); err == nil {
		return strings.TrimSpace(direct)
	}

	var parts []struct {
		Text    string `json:"text"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		var builder strings.Builder
		for _, part := range parts {
			switch {
			case part.Text != "":
				builder.WriteString(part.Text)
			case part.Content != "":
				builder.WriteString(part.Content)
			}
		}
		return strings.TrimSpace(builder.String())
	}

	var object struct {
		Text    string `json:"text"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(raw, &object); err == nil {
		if object.Text != "" {
			return strings.TrimSpace(object.Text)
		}
		return strings.TrimSpace(object.Content)
	}

	return ""
}

type codexHistoryPrompt struct {
	Text      string
	Timestamp int64
}

func readCodexHistory(path string) map[string]codexHistoryPrompt {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	prompts := make(map[string]codexHistoryPrompt)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var item struct {
			SessionID string `json:"session_id"`
			Timestamp int64  `json:"ts"`
			Text      string `json:"text"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &item); err != nil || item.SessionID == "" || item.Text == "" {
			continue
		}
		if existing, ok := prompts[item.SessionID]; !ok || item.Timestamp >= existing.Timestamp {
			prompts[item.SessionID] = codexHistoryPrompt{Text: item.Text, Timestamp: item.Timestamp}
		}
	}
	return prompts
}

func readCodexSessionFile(path string) (codexSession, bool) {
	file, err := os.Open(path)
	if err != nil {
		return codexSession{}, false
	}
	defer file.Close()

	var session codexSession
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	lines := 0
	for scanner.Scan() {
		lines++
		var item struct {
			Type    string `json:"type"`
			Payload struct {
				ID        string `json:"id"`
				Timestamp string `json:"timestamp"`
				CWD       string `json:"cwd"`
				Model     string `json:"model"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &item); err == nil && item.Type == "session_meta" {
			session.ID = item.Payload.ID
			session.CreatedAt = item.Payload.Timestamp
			session.ProjectDir = item.Payload.CWD
			session.Model = item.Payload.Model
			break
		}
		if lines >= 25 {
			break
		}
	}
	return session, session.ID != "" || codexSessionIDFromPath(path) != ""
}

var codexSessionIDPattern = regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

func codexSessionIDFromPath(path string) string {
	return codexSessionIDPattern.FindString(filepath.Base(path))
}

func codexHomeDir() string {
	if home := strings.TrimSpace(os.Getenv("CODEX_HOME")); home != "" {
		return home
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".codex")
	}
	return ".codex"
}

func readCodexCurrentModel() string {
	data, err := os.ReadFile(filepath.Join(codexHomeDir(), "config.toml"))
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`(?m)^\s*model\s*=\s*"([^"]+)"`)
	match := re.FindSubmatch(data)
	if len(match) < 2 {
		return ""
	}
	return string(match[1])
}

func (r *codexActiveRun) runCodex(ctx context.Context, cmdPath, projectDir, prompt, sessionID string, msg codexWSClientMessage) {
	defer func() {
		r.markDone()
	}()

	args := buildCodexExecArgs(projectDir, sessionID, msg)
	cmd := exec.CommandContext(ctx, cmdPath, args...)
	cmd.Dir = projectDir
	cmd.Env = tool_resolve.AppendExtraPaths(append(os.Environ(), "TERM=xterm-256color"))

	stdin, err := cmd.StdinPipe()
	if err != nil {
		r.broadcast(map[string]any{"type": "error", "message": fmt.Sprintf("stdin pipe: %v", err)})
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		r.broadcast(map[string]any{"type": "error", "message": fmt.Sprintf("stdout pipe: %v", err)})
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		r.broadcast(map[string]any{"type": "error", "message": fmt.Sprintf("stderr pipe: %v", err)})
		return
	}

	if err := cmd.Start(); err != nil {
		r.broadcast(map[string]any{"type": "error", "message": fmt.Sprintf("start codex: %v", err)})
		return
	}

	command := append([]string{cmdPath}, args...)
	r.mu.Lock()
	r.cmd = cmd
	r.command = command
	r.mu.Unlock()

	r.broadcast(map[string]any{
		"type":    "started",
		"command": command,
		"resume":  sessionID != "",
	})

	go func() {
		_, _ = stdin.Write([]byte(prompt))
		_ = stdin.Close()
	}()

	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		scanner := bufio.NewScanner(stderr)
		scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for scanner.Scan() {
			text := strings.TrimRight(scanner.Text(), "\r\n")
			if text != "" {
				r.broadcast(map[string]any{"type": "stderr", "data": text})
			}
		}
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		r.forwardCodexLine(line)
	}
	if err := scanner.Err(); err != nil {
		r.broadcast(map[string]any{"type": "error", "message": fmt.Sprintf("read codex stdout: %v", err)})
	}

	waitErr := cmd.Wait()
	<-stderrDone

	if ctx.Err() != nil {
		r.broadcast(map[string]any{"type": "cancelled"})
		return
	}
	if waitErr != nil {
		exitCode := -1
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		}
		r.broadcast(map[string]any{"type": "exit", "code": exitCode, "error": waitErr.Error()})
		return
	}
	r.broadcast(map[string]any{"type": "exit", "code": 0})
}

func buildCodexExecArgs(projectDir, sessionID string, msg codexWSClientMessage) []string {
	model := strings.TrimSpace(msg.Model)

	if sessionID != "" {
		args := []string{
			"exec",
			"resume",
			"--json",
			"--skip-git-repo-check",
			"--dangerously-bypass-approvals-and-sandbox",
		}
		if model != "" {
			args = append(args, "--model", model)
		}
		args = append(args, sessionID, "-")
		return args
	}

	sandbox := strings.TrimSpace(msg.Sandbox)
	if sandbox == "" {
		sandbox = "danger-full-access"
	}
	approval := strings.TrimSpace(msg.ApprovalPolicy)
	if approval == "" {
		approval = "never"
	}

	args := []string{
		"exec",
		"--json",
		"--skip-git-repo-check",
		"--cd", projectDir,
		"--ask-for-approval", approval,
		"--sandbox", sandbox,
	}
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, "-")
	return args
}

func (r *codexActiveRun) forwardCodexLine(line string) {
	var raw json.RawMessage = json.RawMessage(line)
	var event any
	if err := json.Unmarshal(raw, &event); err != nil {
		r.broadcast(map[string]any{"type": "stdout", "data": line})
		return
	}

	if sessionID := findCodexSessionID(event); sessionID != "" {
		if r.setSessionID(sessionID) {
			r.broadcast(map[string]any{"type": "session", "session_id": sessionID})
		}
	}

	r.broadcast(map[string]any{
		"type":  "codex_event",
		"event": raw,
		"raw":   line,
	})
}

func findCodexSessionID(v any) string {
	switch typed := v.(type) {
	case map[string]any:
		for _, key := range []string{"session_id", "sessionId", "thread_id", "threadId", "conversation_id", "conversationId"} {
			if value, ok := typed[key].(string); ok && strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
		for _, value := range typed {
			if sessionID := findCodexSessionID(value); sessionID != "" {
				return sessionID
			}
		}
	case []any:
		for _, value := range typed {
			if sessionID := findCodexSessionID(value); sessionID != "" {
				return sessionID
			}
		}
	}
	return ""
}

func (r *codexRunRegistry) get(sessionID string) *codexActiveRun {
	r.mu.Lock()
	run := r.bySession[sessionID]
	r.mu.Unlock()

	if run == nil {
		return nil
	}
	if run.isRunning() {
		return run
	}
	r.unregister(run)
	return nil
}

func (r *codexRunRegistry) register(sessionID string, run *codexActiveRun) {
	if sessionID == "" {
		return
	}
	r.mu.Lock()
	r.bySession[sessionID] = run
	r.mu.Unlock()
}

func (r *codexRunRegistry) moveSession(oldSessionID, newSessionID string, run *codexActiveRun) {
	if newSessionID == "" {
		return
	}
	r.mu.Lock()
	if oldSessionID != "" && r.bySession[oldSessionID] == run {
		delete(r.bySession, oldSessionID)
	}
	r.bySession[newSessionID] = run
	r.mu.Unlock()
}

func (r *codexRunRegistry) unregister(run *codexActiveRun) {
	r.mu.Lock()
	for sessionID, activeRun := range r.bySession {
		if activeRun == run {
			delete(r.bySession, sessionID)
		}
	}
	r.mu.Unlock()
}

func (r *codexActiveRun) isRunning() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return !r.done
}

func (r *codexActiveRun) addClient(c *codexWSConnection, replay bool, notify bool) {
	r.mu.Lock()
	if r.clients == nil {
		r.clients = make(map[*codexWSConnection]struct{})
	}
	r.clients[c] = struct{}{}
	status := map[string]any{
		"type":       "attached",
		"session_id": r.sessionID,
		"running":    !r.done,
		"command":    append([]string(nil), r.command...),
		"resume":     r.resume,
	}
	events := make([]map[string]any, 0)
	if replay {
		events = append(events, r.events...)
	}
	r.mu.Unlock()

	c.setRun(r)
	if notify {
		c.sendJSON(status)
	}
	for _, event := range events {
		c.sendJSON(event)
	}
}

func (r *codexActiveRun) removeClient(c *codexWSConnection) {
	r.mu.Lock()
	delete(r.clients, c)
	r.mu.Unlock()
}

func (r *codexActiveRun) setSessionID(sessionID string) bool {
	r.mu.Lock()
	oldSessionID := r.sessionID
	if oldSessionID == sessionID {
		r.mu.Unlock()
		return false
	}
	r.sessionID = sessionID
	clients := make([]*codexWSConnection, 0, len(r.clients))
	for client := range r.clients {
		clients = append(clients, client)
	}
	r.mu.Unlock()

	codexRuns.moveSession(oldSessionID, sessionID, r)
	for _, client := range clients {
		client.mu.Lock()
		client.session = sessionID
		client.mu.Unlock()
	}
	return true
}

func (r *codexActiveRun) broadcast(message map[string]any) {
	r.mu.Lock()
	if messageType, _ := message["type"].(string); messageType != "" && messageType != "attached" {
		r.events = append(r.events, message)
		if len(r.events) > 300 {
			r.events = r.events[len(r.events)-300:]
		}
	}
	clients := make([]*codexWSConnection, 0, len(r.clients))
	for client := range r.clients {
		clients = append(clients, client)
	}
	r.mu.Unlock()

	for _, client := range clients {
		client.sendJSON(message)
	}
}

func (r *codexActiveRun) markDone() {
	r.mu.Lock()
	r.done = true
	r.cancel = nil
	r.cmd = nil
	r.mu.Unlock()
	codexRuns.unregister(r)
}

func (r *codexActiveRun) cancelRun() {
	r.mu.Lock()
	cancel := r.cancel
	cmd := r.cmd
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

func (c *codexWSConnection) activeRun() *codexActiveRun {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.run
}

func (c *codexWSConnection) setRun(run *codexActiveRun) {
	sessionID := ""
	if run != nil {
		run.mu.Lock()
		sessionID = run.sessionID
		run.mu.Unlock()
	}

	c.mu.Lock()
	previous := c.run
	c.run = run
	if sessionID != "" {
		c.session = sessionID
	}
	c.mu.Unlock()

	if previous != nil && previous != run {
		previous.removeClient(c)
	}
}

func (c *codexWSConnection) detachActive() {
	c.mu.Lock()
	run := c.run
	c.run = nil
	c.mu.Unlock()

	if run != nil {
		run.removeClient(c)
	}
}

func (c *codexWSConnection) cancelActive() {
	if run := c.activeRun(); run != nil {
		run.cancelRun()
	}
}

func (c *codexWSConnection) sendJSON(v any) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = c.conn.WriteJSON(v)
}
