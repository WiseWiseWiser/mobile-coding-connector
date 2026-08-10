package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	sharedeb "github.com/xhd2015/dot-pkgs/go-pkgs/eventbus"
)

// PublishServerOpts configures the loopback HTTP publish listener.
type PublishServerOpts struct {
	// Token, when non-empty, requires Authorization: Bearer <Token>.
	// Empty Token leaves publish open (no auth header required).
	Token string
}

// PublishServer is a loopback-only HTTP server that accepts POST /publish.
type PublishServer struct {
	hub    *Hub
	token  string
	ln     net.Listener
	server *http.Server
	addr   string
}

// StartPublishServer binds addr (e.g. "127.0.0.1:0" or "127.0.0.1:23891"),
// serves POST /publish into hub, and returns a started server.
// Binding an address already in use returns an error (hard-fail).
func StartPublishServer(addr string, hub *Hub, opts PublishServerOpts) (*PublishServer, error) {
	if hub == nil {
		return nil, fmt.Errorf("eventbus: nil hub")
	}
	if addr == "" {
		addr = fmt.Sprintf("127.0.0.1:%d", DefaultPublishPort())
	}

	// Prefer loopback: reject non-loopback host when host is specified.
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("eventbus: invalid listen addr %q: %w", addr, err)
	}
	if host != "" {
		ip := net.ParseIP(host)
		if ip == nil || !ip.IsLoopback() {
			return nil, fmt.Errorf("eventbus: publish server must bind loopback, got %q", host)
		}
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	ps := &PublishServer{
		hub:   hub,
		token: opts.Token,
		ln:    ln,
		addr:  ln.Addr().String(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/publish", ps.handlePublish)

	ps.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		_ = ps.server.Serve(ln)
	}()

	return ps, nil
}

// Addr returns the actual bound host:port.
func (ps *PublishServer) Addr() string {
	if ps == nil {
		return ""
	}
	return ps.addr
}

// Close shuts down the publish server.
func (ps *PublishServer) Close() error {
	if ps == nil || ps.server == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := ps.server.Shutdown(ctx)
	if ps.ln != nil {
		_ = ps.ln.Close()
	}
	return err
}

func (ps *PublishServer) handlePublish(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if ps.token != "" {
		auth := r.Header.Get("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(auth, prefix) || strings.TrimPrefix(auth, prefix) != ps.token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}
	var ev sharedeb.Event
	if err := json.Unmarshal(body, &ev); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}

	_ = ps.hub.Publish(ev)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}
