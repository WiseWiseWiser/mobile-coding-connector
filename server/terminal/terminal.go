package terminal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"

	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// ControlMessage is a JSON message sent from client to control the terminal
type ControlMessage struct {
	Type string `json:"type"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

// maxScrollback is the maximum number of bytes kept in the scrollback buffer per session
const maxScrollback = 256 * 1024 // 256 KB

// SessionInfo is the JSON representation of a session returned to the frontend
type SessionInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Cwd       string `json:"cwd"`
	CreatedAt string `json:"created_at"`
	// Connected is true when a WebSocket client is currently attached
	Connected bool `json:"connected"`
}

// TerminalSessionsResponse holds paginated terminal sessions response
type TerminalSessionsResponse struct {
	Sessions   []SessionInfo `json:"sessions"`
	Page       int           `json:"page"`
	PageSize   int           `json:"page_size"`
	Total      int           `json:"total"`
	TotalPages int           `json:"total_pages"`
}

// session holds the state for one persistent terminal session
type session struct {
	id        string
	name      string
	cwd       string
	createdAt time.Time

	cmd  *exec.Cmd
	ptmx *os.File

	mu         sync.Mutex
	scrollback []byte // ring buffer of recent output
	conn       *websocket.Conn
	done       chan struct{} // closed when pty exits
}

// sessionManager keeps track of all active terminal sessions
type sessionManager struct {
	mu       sync.Mutex
	sessions map[string]*session
	counter  int
}

var manager = &sessionManager{
	sessions: make(map[string]*session),
}

// RegisterAPI registers the terminal WebSocket endpoint and session management APIs
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/terminal", handleTerminalWebSocket)
	mux.HandleFunc("/api/terminal/sessions", handleSessions)
	mux.HandleFunc("/api/terminal/config", handleConfig)
}

// ------ Session Manager ------

func (m *sessionManager) create(name, cwd string) (*session, error) {
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get cwd: %w", err)
		}
	}
	if info, err := os.Stat(cwd); err != nil || !info.IsDir() {
		return nil, fmt.Errorf("invalid working directory: %s", cwd)
	}

	m.mu.Lock()
	m.counter++
	id := fmt.Sprintf("session-%d", m.counter)
	m.mu.Unlock()

	// Load terminal config for shell, flags, and extra paths
	termCfg, _ := LoadConfig()

	shellPath := "bash"
	shellFlags := []string{"--login", "-i"}
	if termCfg != nil {
		if termCfg.Shell != "" {
			shellPath = termCfg.Shell
		}
		if len(termCfg.ShellFlags) > 0 {
			shellFlags = termCfg.ShellFlags
		}
	}

	// Build custom RC patch options
	patchOpts := rcPatchOptions{
		ExtraPaths: tool_resolve.AllExtraPaths(),
	}
	if termCfg != nil {
		patchOpts.PS1 = termCfg.PS1
	}

	// Instead of patching user's RC files, create a dedicated RC file
	// and tell the shell to use it, so the user's environment stays clean.
	var extraEnv []string
	shellBase := filepath.Base(shellPath)
	switch {
	case strings.Contains(shellBase, "zsh"):
		if zdotdir, err := writeCustomZshRC(patchOpts); err == nil {
			extraEnv = append(extraEnv, "ZDOTDIR="+zdotdir)
		}
	default:
		// bash or other sh-compatible shells: use --rcfile
		if rcFile, err := writeCustomBashRC(patchOpts); err == nil {
			// Replace --login with --rcfile so bash reads our custom rc
			// instead of the standard login sequence
			shellFlags = replaceLoginWithRCFile(shellFlags, rcFile)
		}
	}

	cmd := exec.Command(shellPath, shellFlags...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	cmd.Env = append(cmd.Env, extraEnv...)
	// Ensure common tool install paths and user-configured paths are in PATH
	cmd.Env = tool_resolve.AppendExtraPaths(cmd.Env)
	// Set custom PS1 prompt if configured
	if termCfg != nil && termCfg.PS1 != "" {
		cmd.Env = append(cmd.Env, "PS1="+termCfg.PS1)
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("start pty: %w", err)
	}

	// Default terminal size
	pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80})

	s := &session{
		id:        id,
		name:      name,
		cwd:       cwd,
		createdAt: time.Now(),
		cmd:       cmd,
		ptmx:      ptmx,
		done:      make(chan struct{}),
	}

	m.mu.Lock()
	m.sessions[id] = s
	m.mu.Unlock()

	// Background goroutine: read PTY output, store in scrollback, forward to attached WS
	go s.readLoop()

	return s, nil
}

func (m *sessionManager) get(id string) *session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[id]
}

func (m *sessionManager) list() []SessionInfo {
	return m.listPaginated(1, 1000).Sessions // default to high limit for backward compatibility
}

func (m *sessionManager) listPaginated(page, pageSize int) *TerminalSessionsResponse {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Convert sessions to slice for sorting
	sessionList := make([]*session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessionList = append(sessionList, s)
	}

	// Sort by creation time (oldest first) to preserve user's creation order
	sort.Slice(sessionList, func(i, j int) bool {
		return sessionList[i].createdAt.Before(sessionList[j].createdAt)
	})

	total := len(sessionList)
	totalPages := (total + pageSize - 1) / pageSize

	// Apply pagination
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	var pagedSessions []*session
	if start < total {
		pagedSessions = sessionList[start:end]
	}

	// Convert to response format
	sessions := make([]SessionInfo, 0, len(pagedSessions))
	for _, s := range pagedSessions {
		s.mu.Lock()
		info := SessionInfo{
			ID:        s.id,
			Name:      s.name,
			Cwd:       s.cwd,
			CreatedAt: s.createdAt.Format(time.RFC3339),
			Connected: s.conn != nil,
		}
		s.mu.Unlock()
		sessions = append(sessions, info)
	}

	return &TerminalSessionsResponse{
		Sessions:   sessions,
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}

func (m *sessionManager) remove(id string) {
	m.mu.Lock()
	s, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
	}
	m.mu.Unlock()

	if !ok {
		return
	}
	s.close()
}

// ------ Session ------

func (s *session) readLoop() {
	defer close(s.done)
	buf := make([]byte, 4096)
	for {
		n, err := s.ptmx.Read(buf)
		if n > 0 {
			data := buf[:n]
			s.mu.Lock()
			// Append to scrollback, trim if over limit
			s.scrollback = append(s.scrollback, data...)
			if len(s.scrollback) > maxScrollback {
				s.scrollback = s.scrollback[len(s.scrollback)-maxScrollback:]
			}
			ws := s.conn
			s.mu.Unlock()

			if ws != nil {
				ws.WriteMessage(websocket.BinaryMessage, data)
			}
		}
		if err != nil {
			if err != io.EOF {
				// Terminal exited unexpectedly; notify attached client
				s.mu.Lock()
				ws := s.conn
				s.mu.Unlock()
				if ws != nil {
					ws.WriteMessage(websocket.TextMessage, []byte("\r\n[Terminal exited]"))
				}
			}
			return
		}
	}
}

func (s *session) attach(conn *websocket.Conn) {
	s.mu.Lock()
	// Detach previous connection if any
	if s.conn != nil {
		s.conn.Close()
	}
	s.conn = conn
	// Replay scrollback to the new client
	if len(s.scrollback) > 0 {
		scrollbackCopy := make([]byte, len(s.scrollback))
		copy(scrollbackCopy, s.scrollback)
		s.mu.Unlock()
		// Send terminal reset sequence first to clear any alternate screen mode
		// and reset terminal state (in case a TUI program like vim was running)
		// ESC[?1049l = exit alternate screen buffer
		// ESC[0m = reset all attributes
		conn.WriteMessage(websocket.BinaryMessage, []byte("\x1b[?1049l\x1b[0m"))
		conn.WriteMessage(websocket.BinaryMessage, scrollbackCopy)
	} else {
		s.mu.Unlock()
	}
}

func (s *session) detach(conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Only detach if this is the current connection
	if s.conn == conn {
		s.conn = nil
	}
}

func (s *session) close() {
	s.mu.Lock()
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	s.mu.Unlock()
	s.ptmx.Close()
	s.cmd.Process.Kill()
	s.cmd.Wait()
}

// replaceLoginWithRCFile replaces --login in shell flags with --rcfile <path>.
// If --login is not found, --rcfile is prepended.
func replaceLoginWithRCFile(flags []string, rcFile string) []string {
	var result []string
	replaced := false
	for _, f := range flags {
		if f == "--login" || f == "-l" {
			result = append(result, "--rcfile", rcFile)
			replaced = true
			continue
		}
		result = append(result, f)
	}
	if !replaced {
		result = append([]string{"--rcfile", rcFile}, result...)
	}
	return result
}

// ------ HTTP Handlers ------

func handleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Failed to upgrade connection", http.StatusInternalServerError)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	cwd := r.URL.Query().Get("cwd")
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "Terminal"
	}

	var s *session

	// Try to reconnect to existing session
	if sessionID != "" {
		s = manager.get(sessionID)
	}

	// Create a new session if not reconnecting
	if s == nil {
		s, err = manager.create(name, cwd)
		if err != nil {
			conn.WriteMessage(websocket.TextMessage, []byte("Error: "+err.Error()))
			conn.Close()
			return
		}
		// Send the assigned session ID to the client
		conn.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf(`{"type":"session_id","session_id":"%s"}`, s.id)))
	}

	// Attach this WebSocket to the session
	s.attach(conn)

	// Read from WebSocket and write to PTY.
	// When the WS closes, this loop exits and we detach.
	type wsCloseResult struct {
		closeCode int
	}
	wsCloseCh := make(chan wsCloseResult, 1)
	deleteOnClose := false // Set to true if client requests deletion via message
	go func() {
		var closeCode int
		defer func() {
			wsCloseCh <- wsCloseResult{closeCode: closeCode}
		}()
		for {
			msgType, message, err := conn.ReadMessage()
			if err != nil {
				// Extract close code from WebSocket close error
				if closeErr, ok := err.(*websocket.CloseError); ok {
					closeCode = closeErr.Code
				}
				return
			}

			// Check if it's a JSON control message
			if msgType == websocket.TextMessage {
				var msg ControlMessage
				if err := json.Unmarshal(message, &msg); err == nil && msg.Type != "" {
					switch msg.Type {
					case "resize":
						if msg.Cols > 0 && msg.Rows > 0 {
							pty.Setsize(s.ptmx, &pty.Winsize{
								Rows: uint16(msg.Rows),
								Cols: uint16(msg.Cols),
							})
						}
						continue
					case "close_delete":
						// Legacy: client requests session deletion on close
						deleteOnClose = true
						continue
					}
				}
			}

			// Regular input
			s.ptmx.Write(message)
		}
	}()

	// Wait for the session to end or the connection to close
	select {
	case <-s.done:
		// PTY exited, remove session
		manager.remove(s.id)
		conn.Close()
	case result := <-wsCloseCh:
		// Close code 4000 = client requests session deletion (e.g., React StrictMode cleanup).
		// Also honor the legacy close_delete message for backward compatibility.
		shouldDelete := result.closeCode == 4000 || deleteOnClose
		if shouldDelete {
			manager.remove(s.id)
		} else {
			// Session stays alive for reconnection
			s.detach(conn)
		}
	}
}

func handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Parse pagination parameters
		page := 1
		pageSize := 20 // default page size

		if p := r.URL.Query().Get("page"); p != "" {
			if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
				page = parsed
			}
		}
		if ps := r.URL.Query().Get("page_size"); ps != "" {
			if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
				pageSize = parsed
			}
		}

		sessions := manager.listPaginated(page, pageSize)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		manager.remove(id)
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
