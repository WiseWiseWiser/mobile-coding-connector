package cloudflareproxy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

// Server is the :23790 HTTP/WS reverse-proxy.
type Server struct {
	mu           sync.Mutex
	cfg          Config
	baseDir      string
	byID         map[string]*liveMapping
	byHost       map[string]*liveMapping
	BindHostname func(hostname string, port int) error
}

func newServer(cfg Config, baseDir string) *Server {
	s := &Server{
		cfg:     cfg,
		baseDir: baseDir,
		byID:    map[string]*liveMapping{},
		byHost:  map[string]*liveMapping{},
	}
	s.loadLocked()
	return s
}

func (s *Server) setConfig(cfg Config) {
	s.mu.Lock()
	s.cfg = cfg
	s.mu.Unlock()
}

func (s *Server) mappingCount() (n, connected int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n = len(s.byID)
	for _, m := range s.byID {
		connected += m.wsCount()
	}
	return n, connected
}

func (s *Server) loadLocked() {
	list, err := loadMappingsFile(MappingsPath(s.baseDir))
	if err != nil {
		return
	}
	for _, rec := range list {
		lm := &liveMapping{Mapping: rec}
		s.byID[rec.ID] = lm
		s.byHost[rec.Hostname] = lm
	}
}

func (s *Server) persistLocked() error {
	list := make([]Mapping, 0, len(s.byID))
	for _, m := range s.byID {
		list = append(list, m.Mapping)
	}
	return saveMappingsFile(MappingsPath(s.baseDir), list)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/mappings" || strings.HasPrefix(r.URL.Path, "/api/mappings/"):
		s.handleAPI(w, r)
	case strings.HasPrefix(r.URL.Path, "/dial/"):
		s.handleDial(w, r)
	default:
		s.handleVisitor(w, r)
	}
}

func (s *Server) handleAPI(w http.ResponseWriter, r *http.Request) {
	if !s.authorize(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.apiList(w, r)
	case http.MethodPost:
		s.apiAdd(w, r)
	case http.MethodDelete:
		s.apiDelete(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) authorize(r *http.Request) bool {
	s.mu.Lock()
	want := s.cfg.authToken()
	s.mu.Unlock()
	if want == "" {
		return false
	}
	got := strings.TrimSpace(r.Header.Get("Authorization"))
	got = strings.TrimPrefix(got, "Bearer ")
	got = strings.TrimPrefix(got, "bearer ")
	return got == want
}

func (s *Server) apiList(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	views := make([]MappingView, 0, len(s.byID))
	for _, m := range s.byID {
		views = append(views, MappingView{
			ID:       m.ID,
			Hostname: m.Hostname,
			WS:       m.wsCount(),
			URL:      publicURL(m.Hostname),
		})
	}
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"mappings": views})
}

func (s *Server) apiAdd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Hostname string `json:"hostname"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	host := normalizeHostname(req.Hostname)
	if host == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "hostname is required"})
		return
	}
	id, err := newMappingID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	dialTok, err := newSecret()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.mu.Lock()
	if _, ok := s.byHost[host]; ok {
		s.mu.Unlock()
		writeJSON(w, http.StatusConflict, map[string]string{"error": "hostname already mapped"})
		return
	}
	for s.byID[id] != nil {
		id, err = newMappingID()
		if err != nil {
			s.mu.Unlock()
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	lm := &liveMapping{Mapping: Mapping{
		ID:        id,
		Hostname:  host,
		DialToken: dialTok,
		CreatedAt: time.Now().UTC(),
	}}
	s.byID[id] = lm
	s.byHost[host] = lm
	_ = s.persistLocked()
	port := s.cfg.port()
	bind := s.BindHostname
	s.mu.Unlock()

	if bind != nil {
		if berr := bind(host, port); berr != nil {
			s.mu.Lock()
			delete(s.byID, id)
			delete(s.byHost, host)
			_ = s.persistLocked()
			s.mu.Unlock()
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "bind hostname: " + berr.Error()})
			return
		}
	}

	view := MappingView{
		ID:       id,
		Hostname: host,
		WS:       0,
		URL:      publicURL(host),
		DialURL:  dialURL(r, id, dialTok),
	}
	writeJSON(w, http.StatusCreated, view)
}

func (s *Server) apiDelete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	host := normalizeHostname(r.URL.Query().Get("hostname"))
	if id == "" && strings.HasPrefix(r.URL.Path, "/api/mappings/") {
		id = strings.TrimPrefix(r.URL.Path, "/api/mappings/")
	}
	s.mu.Lock()
	var lm *liveMapping
	if id != "" {
		lm = s.byID[id]
	} else if host != "" {
		lm = s.byHost[host]
	}
	if lm == nil {
		s.mu.Unlock()
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "mapping not found"})
		return
	}
	delete(s.byID, lm.ID)
	delete(s.byHost, lm.Hostname)
	_ = s.persistLocked()
	s.mu.Unlock()
	lm.closeAll()
	writeJSON(w, http.StatusOK, map[string]string{"deleted": lm.ID})
}

func (s *Server) handleDial(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/dial/")
	id = strings.Trim(id, "/")
	tok := r.URL.Query().Get("t")
	s.mu.Lock()
	lm := s.byID[id]
	s.mu.Unlock()
	if lm == nil || tok == "" || tok != lm.DialToken {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	conn, err := wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	lm.put(newPooledConn(conn))
}

func (s *Server) handleVisitor(w http.ResponseWriter, r *http.Request) {
	host := normalizeHostname(r.Host)
	s.mu.Lock()
	lm := s.byHost[host]
	s.mu.Unlock()
	if lm == nil {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, "not found\n")
		return
	}
	dial := lm.take()
	if dial == nil {
		w.Header().Set("Retry-After", "2")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = io.WriteString(w, "no connected origin\n")
		return
	}
	// One dial serves exactly one request: the origin answers with
	// Connection: close, so the socket is finished once the request ends.
	backend := net.Conn(dial)
	defer backend.Close()

	start := time.Now()
	path := r.URL.RequestURI()
	if isWebSocketUpgrade(r) {
		err := proxyUpgrade(w, r, backend)
		dur := time.Since(start).Round(time.Millisecond)
		if dur < time.Millisecond {
			dur = time.Millisecond
		}
		log.Printf("%-4s %-20s %-7s %s  %s", r.Method, path, "ws", statusOrErr(err), dur)
		return
	}

	rw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
	proxyTCP(rw, r, backend)
	dur := time.Since(start).Round(time.Millisecond)
	if dur < time.Millisecond {
		dur = time.Millisecond
	}
	log.Printf("%-4s %-20s %-7s %d  %s", r.Method, path, "forward", rw.status, dur)
}

func isWebSocketUpgrade(r *http.Request) bool {
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return true
	}
	// Cloudflare/cloudflared may drop Upgrade while leaving the handshake key.
	if strings.TrimSpace(r.Header.Get("Sec-WebSocket-Key")) != "" {
		return true
	}
	conn := strings.ToLower(r.Header.Get("Connection"))
	return strings.Contains(conn, "upgrade") && strings.Contains(strings.ToLower(r.Header.Get("Upgrade")), "websocket")
}

func statusOrErr(err error) string {
	if err != nil {
		return "err"
	}
	return "101"
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("hijack not supported")
	}
	return hj.Hijack()
}

func (w *statusWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func proxyTCP(w http.ResponseWriter, r *http.Request, backend net.Conn) {
	var once sync.Once
	dialed := backend
	rp := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = r.Host
			req.Host = r.Host
		},
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var c net.Conn
				once.Do(func() { c = dialed })
				if c == nil {
					return nil, fmt.Errorf("origin conn already used")
				}
				return c, nil
			},
			DisableKeepAlives: true,
		},
		FlushInterval: -1,
	}
	rp.ServeHTTP(w, r)
}

func proxyUpgrade(w http.ResponseWriter, r *http.Request, backend net.Conn) error {
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack not supported", http.StatusInternalServerError)
		return fmt.Errorf("hijack not supported")
	}
	client, bufrw, err := hj.Hijack()
	if err != nil {
		http.Error(w, "hijack failed", http.StatusInternalServerError)
		return err
	}
	defer client.Close()
	if err := r.Write(backend); err != nil {
		return err
	}
	errc := make(chan struct{}, 2)
	go func() {
		_, _ = io.Copy(backend, bufrw)
		_ = backend.Close()
		errc <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(client, backend)
		_ = client.Close()
		errc <- struct{}{}
	}()
	<-errc
	return nil
}

func dialURL(r *http.Request, id, tok string) string {
	host := r.Host
	if host == "" {
		host = "127.0.0.1"
	}
	scheme := "ws"
	fwd := strings.ToLower(r.Header.Get("X-Forwarded-Proto"))
	loop := strings.HasPrefix(host, "127.0.0.1") || strings.HasPrefix(host, "localhost")
	if r.TLS != nil || fwd == "https" || !loop {
		scheme = "wss"
	}
	if loop && fwd != "https" && r.TLS == nil {
		scheme = "ws"
	}
	return fmt.Sprintf("%s://%s/dial/%s?t=%s", scheme, host, id, tok)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
