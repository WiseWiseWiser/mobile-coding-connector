package portforward

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultVisitIdle is the default ad-hoc visit idle timeout (10 minutes).
const DefaultVisitIdle = 10 * time.Minute

// AdhocVisitSession is an in-memory ad-hoc public visit of a local port.
type AdhocVisitSession struct {
	ID           string
	LocalPort    int
	ProxyPort    int
	PublicURL    string
	Provider     string
	Hostname     string
	CreatedAt    time.Time
	LastActivity time.Time
	IdleTimeout  time.Duration
	Status       string
}

// visitSession is the internal mutable session state.
type visitSession struct {
	AdhocVisitSession

	stopTunnel func()
	proxySrv   *http.Server
	proxyLn    net.Listener
	idleTimer  *time.Timer
}

// VisitSessionManager manages in-memory ad-hoc visit sessions with an idle
// reverse-proxy hop in front of the user's local port.
type VisitSessionManager struct {
	mu sync.Mutex

	sessions  map[string]*visitSession // by id
	byPort    map[int]string           // local port -> id
	providers map[string]Provider

	nowFn             func() time.Time
	listeningChecker  func(int) bool
	mappingNamesPath  string
}

// NewVisitSessionManager creates an empty in-memory visit session manager.
func NewVisitSessionManager() *VisitSessionManager {
	return &VisitSessionManager{
		sessions:  make(map[string]*visitSession),
		byPort:    make(map[int]string),
		providers: make(map[string]Provider),
		nowFn:     time.Now,
	}
}

var defaultVisitManager = NewVisitSessionManager()

// GetDefaultVisitSessionManager returns the process-wide visit session manager.
func GetDefaultVisitSessionManager() *VisitSessionManager {
	return defaultVisitManager
}

// RegisterProvider injects a tunnel provider (tests use fakes).
func (m *VisitSessionManager) RegisterProvider(p Provider) {
	if p == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[p.Name()] = p
}

// SetNow injects a clock (for idle Sweep tests).
func (m *VisitSessionManager) SetNow(fn func() time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if fn == nil {
		m.nowFn = time.Now
		return
	}
	m.nowFn = fn
}

// SetListeningChecker injects local-port listen detection.
func (m *VisitSessionManager) SetListeningChecker(fn func(int) bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.listeningChecker = fn
}

// SetMappingNamesPath isolates the port-mapping-names file path for tests.
// Ad-hoc visits never write mapping names (path is reserved for isolation).
func (m *VisitSessionManager) SetMappingNamesPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.mappingNamesPath = path
}

func (m *VisitSessionManager) now() time.Time {
	if m.nowFn == nil {
		return time.Now()
	}
	return m.nowFn()
}

// Start creates an ad-hoc visit for localPort.
// provider: ""/"auto" / "owned" / "quick" / full provider names.
// idle: 0 means DefaultVisitIdle.
func (m *VisitSessionManager) Start(localPort int, provider string, idle time.Duration) (*AdhocVisitSession, error) {
	if localPort <= 0 || localPort > 65535 {
		return nil, fmt.Errorf("invalid port: %d", localPort)
	}
	if idle <= 0 {
		idle = DefaultVisitIdle
	}

	m.mu.Lock()
	if _, exists := m.byPort[localPort]; exists {
		m.mu.Unlock()
		return nil, fmt.Errorf("ad-hoc visit already active on port %d", localPort)
	}
	prov, err := m.selectProviderLocked(provider)
	if err != nil {
		m.mu.Unlock()
		return nil, err
	}
	// Hold lock only for selection; release while starting proxy/tunnel.
	m.mu.Unlock()

	// Ephemeral reverse-proxy hop: tunnel -> proxyPort -> localhost:localPort
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen proxy hop: %w", err)
	}
	proxyPort := ln.Addr().(*net.TCPAddr).Port

	target, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", localPort))
	if err != nil {
		ln.Close()
		return nil, err
	}
	rp := httputil.NewSingleHostReverseProxy(target)

	id, err := randomVisitID()
	if err != nil {
		ln.Close()
		return nil, err
	}

	sess := &visitSession{
		AdhocVisitSession: AdhocVisitSession{
			ID:          id,
			LocalPort:   localPort,
			ProxyPort:   proxyPort,
			Provider:    prov.Name(),
			IdleTimeout: idle,
			Status:      StatusConnecting,
		},
		proxyLn: ln,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.touch(id)
		rp.ServeHTTP(w, r)
	})
	sess.proxySrv = &http.Server{Handler: handler}
	go func() { _ = sess.proxySrv.Serve(ln) }()

	// Owned: empty hostname → provider generates ephemeral subdomain.
	// Never write port-mapping-names for ad-hoc visits.
	hostname := ""
	handle, err := prov.Start(proxyPort, hostname)
	if err != nil {
		m.shutdownProxy(sess)
		return nil, fmt.Errorf("start tunnel: %w", err)
	}
	if handle == nil || handle.Result == nil {
		m.shutdownProxy(sess)
		return nil, fmt.Errorf("start tunnel: nil handle")
	}

	result, ok := <-handle.Result
	if !ok {
		m.shutdownProxy(sess)
		if handle.Stop != nil {
			handle.Stop()
		}
		return nil, fmt.Errorf("start tunnel: result channel closed")
	}
	if result.Err != nil {
		m.shutdownProxy(sess)
		if handle.Stop != nil {
			handle.Stop()
		}
		return nil, result.Err
	}

	readyAt := m.now()
	sess.PublicURL = result.PublicURL
	sess.Hostname = hostnameFromURL(result.PublicURL)
	sess.CreatedAt = readyAt
	sess.LastActivity = readyAt
	sess.Status = StatusActive
	sess.stopTunnel = handle.Stop

	m.mu.Lock()
	// Re-check duplicate after async work.
	if _, exists := m.byPort[localPort]; exists {
		m.mu.Unlock()
		m.shutdownProxy(sess)
		if handle.Stop != nil {
			handle.Stop()
		}
		return nil, fmt.Errorf("ad-hoc visit already active on port %d", localPort)
	}
	m.sessions[id] = sess
	m.byPort[localPort] = id
	m.armIdleTimerLocked(sess)
	out := sess.AdhocVisitSession
	m.mu.Unlock()

	return &out, nil
}

func (m *VisitSessionManager) selectProviderLocked(name string) (Provider, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	switch name {
	case "", "auto":
		if p := m.providers[ProviderCloudflareOwned]; p != nil && p.Available() {
			return p, nil
		}
		if p := m.providers[ProviderCloudflareQuick]; p != nil && p.Available() {
			return p, nil
		}
		return nil, fmt.Errorf("no available visit provider (owned and cloudflare_quick unavailable)")
	case "owned", ProviderCloudflareOwned:
		p := m.providers[ProviderCloudflareOwned]
		if p == nil || !p.Available() {
			return nil, fmt.Errorf("provider cloudflare_owned is not available")
		}
		return p, nil
	case "quick", ProviderCloudflareQuick:
		p := m.providers[ProviderCloudflareQuick]
		if p == nil || !p.Available() {
			return nil, fmt.Errorf("provider cloudflare_quick is not available")
		}
		return p, nil
	default:
		p := m.providers[name]
		if p == nil {
			// Try case-preserving lookup
			for k, v := range m.providers {
				if strings.EqualFold(k, name) {
					p = v
					break
				}
			}
		}
		if p == nil {
			return nil, fmt.Errorf("unknown provider %q", name)
		}
		if !p.Available() {
			return nil, fmt.Errorf("provider %s is not available", p.Name())
		}
		return p, nil
	}
}

// List returns active ad-hoc sessions (copies).
func (m *VisitSessionManager) List() []AdhocVisitSession {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]AdhocVisitSession, 0, len(m.sessions))
	for _, s := range m.sessions {
		out = append(out, s.AdhocVisitSession)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LocalPort != out[j].LocalPort {
			return out[i].LocalPort < out[j].LocalPort
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// Stop stops a session by id or decimal local port string.
func (m *VisitSessionManager) Stop(idOrPort string) error {
	idOrPort = strings.TrimSpace(idOrPort)
	if idOrPort == "" {
		return fmt.Errorf("stop target required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var sess *visitSession
	if s, ok := m.sessions[idOrPort]; ok {
		sess = s
	} else if port, err := strconv.Atoi(idOrPort); err == nil {
		if id, ok := m.byPort[port]; ok {
			sess = m.sessions[id]
		}
	}
	if sess == nil {
		return fmt.Errorf("visit session not found: %s", idOrPort)
	}
	m.stopSessionLocked(sess)
	return nil
}

// Sweep expires idle sessions using the injected clock.
func (m *VisitSessionManager) Sweep() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := m.now()
	var expired []*visitSession
	for _, s := range m.sessions {
		if s.IdleTimeout > 0 && !s.LastActivity.IsZero() && !now.Before(s.LastActivity.Add(s.IdleTimeout)) {
			expired = append(expired, s)
		}
	}
	for _, s := range expired {
		m.stopSessionLocked(s)
	}
}

func (m *VisitSessionManager) touch(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[id]
	if !ok {
		return
	}
	s.LastActivity = m.now()
	m.armIdleTimerLocked(s)
}

func (m *VisitSessionManager) armIdleTimerLocked(s *visitSession) {
	if s.idleTimer != nil {
		s.idleTimer.Stop()
		s.idleTimer = nil
	}
	if s.IdleTimeout <= 0 {
		return
	}
	id := s.ID
	timeout := s.IdleTimeout
	s.idleTimer = time.AfterFunc(timeout, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		cur, ok := m.sessions[id]
		if !ok {
			return
		}
		// Real-time timer: stop if wall clock idle elapsed.
		// Fake-clock tests rely on Sweep instead.
		if time.Since(cur.LastActivity) >= cur.IdleTimeout {
			m.stopSessionLocked(cur)
		}
	})
}

func (m *VisitSessionManager) stopSessionLocked(s *visitSession) {
	if s == nil {
		return
	}
	if s.idleTimer != nil {
		s.idleTimer.Stop()
		s.idleTimer = nil
	}
	delete(m.sessions, s.ID)
	if m.byPort[s.LocalPort] == s.ID {
		delete(m.byPort, s.LocalPort)
	}
	s.Status = StatusStopped
	stopTunnel := s.stopTunnel
	s.stopTunnel = nil
	// Unlock-sensitive external calls after map cleanup but we hold mu —
	// call stop outside would need restructure. Stop funcs should be quick.
	if stopTunnel != nil {
		stopTunnel()
	}
	// Close proxy without holding issues — Close is safe.
	if s.proxySrv != nil {
		_ = s.proxySrv.Close()
		s.proxySrv = nil
	}
	if s.proxyLn != nil {
		_ = s.proxyLn.Close()
		s.proxyLn = nil
	}
}

func (m *VisitSessionManager) shutdownProxy(s *visitSession) {
	if s == nil {
		return
	}
	if s.proxySrv != nil {
		_ = s.proxySrv.Close()
	}
	if s.proxyLn != nil {
		_ = s.proxyLn.Close()
	}
}

func (m *VisitSessionManager) isListening(port int) bool {
	m.mu.Lock()
	fn := m.listeningChecker
	m.mu.Unlock()
	if fn != nil {
		return fn(port)
	}
	// Default: probe TCP connect to localhost:port
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 200*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// visitAPIResponse is the JSON shape for visit API responses.
type visitAPIResponse struct {
	ID           string  `json:"id"`
	Port         int     `json:"port"`
	ProxyPort    int     `json:"proxy_port,omitempty"`
	PublicURL    string  `json:"public_url"`
	Provider     string  `json:"provider"`
	Hostname     string  `json:"hostname,omitempty"`
	IdleSeconds  float64 `json:"idle_seconds"`
	Status       string  `json:"status"`
	Listening    bool    `json:"listening"`
	CreatedAt    string  `json:"created_at,omitempty"`
	LastActivity string  `json:"last_activity,omitempty"`
}

func sessionToAPI(s AdhocVisitSession, listening bool) visitAPIResponse {
	return visitAPIResponse{
		ID:           s.ID,
		Port:         s.LocalPort,
		ProxyPort:    s.ProxyPort,
		PublicURL:    s.PublicURL,
		Provider:     s.Provider,
		Hostname:     s.Hostname,
		IdleSeconds:  s.IdleTimeout.Seconds(),
		Status:       s.Status,
		Listening:    listening,
		CreatedAt:    s.CreatedAt.UTC().Format(time.RFC3339Nano),
		LastActivity: s.LastActivity.UTC().Format(time.RFC3339Nano),
	}
}

// RegisterAPI mounts POST/GET/DELETE /api/ports/visit on mux.
func (m *VisitSessionManager) RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/ports/visit", m.handleVisitAPI)
}

func (m *VisitSessionManager) handleVisitAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		m.handleVisitList(w, r)
	case http.MethodPost:
		m.handleVisitStart(w, r)
	case http.MethodDelete:
		m.handleVisitStop(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *VisitSessionManager) handleVisitList(w http.ResponseWriter, r *http.Request) {
	list := m.List()
	out := make([]visitAPIResponse, 0, len(list))
	for _, s := range list {
		out = append(out, sessionToAPI(s, true))
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (m *VisitSessionManager) handleVisitStart(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Port        int      `json:"port"`
		Provider    string   `json:"provider"`
		IdleSeconds *float64 `json:"idle_seconds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.Port <= 0 || req.Port > 65535 {
		http.Error(w, fmt.Sprintf("invalid port: %d", req.Port), http.StatusBadRequest)
		return
	}

	var idle time.Duration
	if req.IdleSeconds != nil {
		idle = time.Duration(*req.IdleSeconds * float64(time.Second))
		if idle <= 0 {
			// Explicit zero-ish: still use default for safety
			idle = DefaultVisitIdle
		}
	}

	listening := m.isListening(req.Port)
	sess, err := m.Start(req.Port, req.Provider, idle)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(sessionToAPI(*sess, listening))
}

func (m *VisitSessionManager) handleVisitStop(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		id = r.URL.Query().Get("port")
	}
	if id == "" {
		http.Error(w, "id or port query parameter required", http.StatusBadRequest)
		return
	}
	if err := m.Stop(id); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
}

func randomVisitID() (string, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func hostnameFromURL(publicURL string) string {
	u, err := url.Parse(publicURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}
