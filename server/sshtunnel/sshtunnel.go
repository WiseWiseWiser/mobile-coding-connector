// Package sshtunnel provides authenticated SSH session lifecycle and a duplex
// WebSocket tunnel from the agent server to a per-session AdhocServer (or a
// test BackendDial).
package sshtunnel

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"

	"github.com/xhd2015/ai-critic/cmd/agentcli/sshcmd"
)

// Manager holds process-local SSH tunnel sessions.
type Manager struct {
	// RequiredToken, when non-empty, requires Authorization: Bearer <token>
	// on all handlers (L2-friendly without full auth middleware).
	RequiredToken string
	// BackendDial, when set, skips AdhocServer on CreateSession; each tunnel
	// WS upgrade dials this backend instead (binary-echo leaf).
	BackendDial func() (net.Conn, error)

	mu       sync.Mutex
	sessions map[string]*session
}

type session struct {
	id      string
	user    string
	hostKey string
	// adhoc is non-nil when BackendDial was nil at create time.
	adhoc *sshcmd.AdhocServer
	// backendAddr is "127.0.0.1:port" for Adhoc; empty when using BackendDial.
	backendAddr string
}

// NewManager returns an empty Manager.
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*session),
	}
}

var defaultManager = NewManager()

// RegisterAPI mounts SSH tunnel routes on the default process Manager.
func RegisterAPI(mux *http.ServeMux) {
	RegisterAPIWithManager(mux, defaultManager)
}

// RegisterAPIWithManager mounts SSH tunnel routes bound to m (tests inject).
func RegisterAPIWithManager(mux *http.ServeMux, m *Manager) {
	if m == nil {
		m = defaultManager
	}
	mux.HandleFunc("POST /api/remote-agent/ssh/sessions", m.handleCreate)
	mux.HandleFunc("DELETE /api/remote-agent/ssh/sessions/{id}", m.handleDestroy)
	mux.HandleFunc("GET /api/remote-agent/ssh/sessions/{id}/tunnel", m.handleTunnel)
}

type createRequest struct {
	PublicKey string `json:"public_key,omitempty"`
}

type createResponse struct {
	SessionID string `json:"session_id"`
	User      string `json:"user"`
	HostKey   string `json:"host_key,omitempty"`
}

func (m *Manager) handleCreate(w http.ResponseWriter, r *http.Request) {
	if !m.checkAuth(w, r) {
		return
	}

	var req createRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	info, err := m.CreateSession(req.PublicKey)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (m *Manager) handleDestroy(w http.ResponseWriter, r *http.Request) {
	if !m.checkAuth(w, r) {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "session id required")
		return
	}
	if err := m.DestroySession(id); err != nil {
		writeJSONError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

var tunnelUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (m *Manager) handleTunnel(w http.ResponseWriter, r *http.Request) {
	if !m.checkAuth(w, r) {
		return
	}
	id := r.PathValue("id")
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "session id required")
		return
	}

	sess := m.getSession(id)
	if sess == nil {
		writeJSONError(w, http.StatusNotFound, "session not found")
		return
	}

	ws, err := tunnelUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	backend, err := m.dialBackend(sess)
	if err != nil {
		_ = ws.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()))
		return
	}
	defer backend.Close()

	// Bidirectional binary splice: WS ↔ TCP.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		defer backend.Close()
		for {
			msgType, data, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if msgType != websocket.BinaryMessage && msgType != websocket.TextMessage {
				continue
			}
			if len(data) == 0 {
				continue
			}
			if _, err := backend.Write(data); err != nil {
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		defer ws.Close()
		buf := make([]byte, 32*1024)
		for {
			n, err := backend.Read(buf)
			if n > 0 {
				if werr := ws.WriteMessage(websocket.BinaryMessage, buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	wg.Wait()
}

func (m *Manager) dialBackend(sess *session) (net.Conn, error) {
	m.mu.Lock()
	backendDial := m.BackendDial
	// Re-check session still exists (destroy races).
	cur := m.sessions[sess.id]
	m.mu.Unlock()
	if cur == nil {
		return nil, fmt.Errorf("session not found")
	}
	if backendDial != nil {
		return backendDial()
	}
	if sess.backendAddr == "" {
		return nil, fmt.Errorf("session backend not available")
	}
	return net.DialTimeout("tcp", sess.backendAddr, 5*time.Second)
}

// CreateSession mints a session and starts AdhocServer when BackendDial is nil.
func (m *Manager) CreateSession(publicKeyOpenSSH string) (*createResponse, error) {
	if m == nil {
		return nil, fmt.Errorf("manager is nil")
	}
	id, err := mintSessionID()
	if err != nil {
		return nil, err
	}

	user := "agent"
	resp := &createResponse{
		SessionID: id,
		User:      user,
	}

	sess := &session{
		id:   id,
		user: user,
	}

	m.mu.Lock()
	backendDial := m.BackendDial
	m.mu.Unlock()

	if backendDial == nil {
		// ForcePipeShell must stay false so OpenSSH -tt gets a real prompt.
		adhoc := &sshcmd.AdhocServer{User: user, Interactive: true, ForcePipeShell: false}
		if strings.TrimSpace(publicKeyOpenSSH) != "" {
			pub, err := parseAuthorizedKey(publicKeyOpenSSH)
			if err != nil {
				return nil, fmt.Errorf("invalid public_key: %w", err)
			}
			adhoc.SetAuthorizedKeys([]ssh.PublicKey{pub})
		}
		if err := adhoc.Start(); err != nil {
			return nil, fmt.Errorf("start adhoc ssh: %w", err)
		}
		sess.adhoc = adhoc
		sess.backendAddr = adhoc.Addr()
		if hk := adhoc.HostKey(); hk != nil {
			sess.hostKey = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(hk)))
			resp.HostKey = sess.hostKey
		}
	}

	m.mu.Lock()
	if m.sessions == nil {
		m.sessions = make(map[string]*session)
	}
	m.sessions[id] = sess
	m.mu.Unlock()

	return resp, nil
}

// DestroySession tears down a session and its AdhocServer if any.
func (m *Manager) DestroySession(id string) error {
	if m == nil {
		return fmt.Errorf("manager is nil")
	}
	m.mu.Lock()
	sess, ok := m.sessions[id]
	if ok {
		delete(m.sessions, id)
	}
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("session not found")
	}
	if sess.adhoc != nil {
		_ = sess.adhoc.Close()
	}
	return nil
}

func (m *Manager) getSession(id string) *session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.sessions[id]
}

func (m *Manager) checkAuth(w http.ResponseWriter, r *http.Request) bool {
	if m == nil || m.RequiredToken == "" {
		return true
	}
	auth := r.Header.Get("Authorization")
	want := "Bearer " + m.RequiredToken
	if auth != want {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized")
		return false
	}
	return true
}

func parseAuthorizedKey(line string) (ssh.PublicKey, error) {
	pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(strings.TrimSpace(line)))
	return pub, err
}

func mintSessionID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
