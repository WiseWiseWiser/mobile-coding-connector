package cloudflareproxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestDialURLPublicIsWSS(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "http://cfproxy.example.com/api/mappings", nil)
	req.Host = "cfproxy.example.com"
	got := dialURL(req, "map-ab", "tok")
	if got != "wss://cfproxy.example.com/dial/map-ab?t=tok" {
		t.Fatalf("got %q", got)
	}
	req.Header.Set("X-Forwarded-Proto", "https")
	got = dialURL(req, "map-ab", "tok")
	if !strings.HasPrefix(got, "wss://") {
		t.Fatalf("forwarded proto: %q", got)
	}
	req, _ = http.NewRequest(http.MethodPost, "http://127.0.0.1:23790/api/mappings", nil)
	req.Host = "127.0.0.1:23790"
	got = dialURL(req, "map-ab", "tok")
	if got != "ws://127.0.0.1:23790/dial/map-ab?t=tok" {
		t.Fatalf("loopback: %q", got)
	}
}

func testServer(t *testing.T, token string) (*Server, *httptest.Server) {
	t.Helper()
	dir := t.TempDir()
	s := newServer(Config{GeneratedToken: token}, dir)
	ts := httptest.NewServer(s)
	t.Cleanup(ts.Close)
	return s, ts
}

func TestVisitorUnknownHost404(t *testing.T) {
	_, ts := testServer(t, "secret")
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/x", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "unknown.example.com"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestVisitorNoWS503(t *testing.T) {
	_, ts := testServer(t, "secret")
	cli := &APIClient{BaseURL: ts.URL, Token: "secret"}
	view, err := cli.AddMapping("foo.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if view.ID == "" || view.DialURL == "" {
		t.Fatalf("add: %+v", view)
	}
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "foo.example.com"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestAPIUnauthorized(t *testing.T) {
	_, ts := testServer(t, "secret")
	cli := &APIClient{BaseURL: ts.URL, Token: "wrong"}
	_, err := cli.ListMappings()
	if err == nil || !strings.Contains(err.Error(), "unauthorized") {
		t.Fatalf("err %v", err)
	}
}

func TestEchoRoundtrip(t *testing.T) {
	_, ts := testServer(t, "secret")
	cli := &APIClient{BaseURL: ts.URL, Token: "secret"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var out syncBuffer
	errCh := make(chan error, 1)
	go func() {
		errCh <- cli.Publish(ctx, PublishOpts{
			Hostname: "foo.example.com",
			Echo:     true,
			Pool:     1,
			Stdout:   &out,
			Stderr:   io.Discard,
		})
	}()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(out.String(), "connected") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(out.String(), "connected") {
		t.Fatalf("never connected:\n%s", out.String())
	}

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "foo.example.com"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d body %s", resp.StatusCode, body)
	}
	got := string(body)
	for _, want := range []string{"cloudflare-proxy echo", "method: GET", "path:   /health", "host:   foo.example.com"} {
		if !strings.Contains(got, want) {
			t.Fatalf("echo body missing %q:\n%s", want, got)
		}
	}
	cancel()
	select {
	case err := <-errCh:
		if err != nil && err != context.Canceled {
			t.Fatalf("publish: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("publish did not return")
	}
}

func TestDeleteThen404(t *testing.T) {
	_, ts := testServer(t, "secret")
	cli := &APIClient{BaseURL: ts.URL, Token: "secret"}
	view, err := cli.AddMapping("gone.example.com")
	if err != nil {
		t.Fatal(err)
	}
	deleted, err := cli.DeleteMapping(view.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if deleted != view.ID {
		t.Fatalf("deleted %q", deleted)
	}
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "gone.example.com"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestForwardRoundtrip(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Origin", "yes")
		_, _ = io.WriteString(w, "from-origin:"+r.URL.Path)
	}))
	t.Cleanup(origin.Close)

	_, ts := testServer(t, "secret")
	cli := &APIClient{BaseURL: ts.URL, Token: "secret"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var out syncBuffer
	go func() {
		_ = cli.Publish(ctx, PublishOpts{
			Hostname: "bar.example.com",
			Forward:  origin.URL,
			Pool:     1,
			Stdout:   &out,
			Stderr:   io.Discard,
		})
	}()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(out.String(), "connected") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(out.String(), "connected") {
		t.Fatalf("never connected:\n%s", out.String())
	}
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "bar.example.com"
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK || string(body) != "from-origin:/ping" {
		t.Fatalf("status %d body %q", resp.StatusCode, body)
	}
	if resp.Header.Get("X-Origin") != "yes" {
		t.Fatalf("missing origin header: %v", resp.Header)
	}
	cancel()
}

func TestWSEchoRoundtrip(t *testing.T) {
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			mt, p, err := c.ReadMessage()
			if err != nil {
				return
			}
			if err := c.WriteMessage(mt, p); err != nil {
				return
			}
		}
	}))
	t.Cleanup(origin.Close)

	_, ts := testServer(t, "secret")
	cli := &APIClient{BaseURL: ts.URL, Token: "secret"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var out syncBuffer
	go func() {
		_ = cli.Publish(ctx, PublishOpts{
			Hostname: "bar.example.com",
			Forward:  origin.URL,
			Pool:     1,
			Stdout:   &out,
			Stderr:   io.Discard,
		})
	}()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(out.String(), "connected") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(out.String(), "connected") {
		t.Fatalf("never connected:\n%s", out.String())
	}

	proxyURL, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	d := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.Dial(network, proxyURL.Host)
		},
	}
	c, resp, err := d.Dial("ws://bar.example.com/echo", nil)
	if err != nil {
		if resp != nil {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("dial: %v status %d body %s", err, resp.StatusCode, b)
		}
		t.Fatal(err)
	}
	defer c.Close()
	if err := c.WriteMessage(websocket.TextMessage, []byte("hi")); err != nil {
		t.Fatal(err)
	}
	_, p, err := c.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(p) != "hi" {
		t.Fatalf("echo %q", p)
	}
	cancel()
}

// syncBuffer is a race-free sink for Publish's progress output.
type syncBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func waitConnected(t *testing.T, out *syncBuffer) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(out.String(), "connected") {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("dial pool never connected:\n%s", out.String())
}

func visitorGet(t *testing.T, baseURL, host, path string) (string, int) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, baseURL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = host
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body), resp.StatusCode
}

// An idle pooled dial must not hold an origin connection. The origin HTTP server
// reaps connections that have not sent request headers yet (ReadHeaderTimeout),
// which used to kill the pooled socket behind the edge's back and leave a corpse
// that answered the next visitor with an instant 502.
func TestIdleDialHoldsNoOriginConnection(t *testing.T) {
	var conns atomic.Int64
	origin := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "from-origin:"+r.URL.Path)
	}))
	origin.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			conns.Add(1)
		}
	}
	origin.Start()
	t.Cleanup(origin.Close)

	_, ts := testServer(t, "secret")
	cli := &APIClient{BaseURL: ts.URL, Token: "secret"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var out syncBuffer
	go func() {
		_ = cli.Publish(ctx, PublishOpts{
			Hostname: "idle.example.com",
			Forward:  origin.URL,
			Pool:     1,
			Stdout:   &out,
			Stderr:   io.Discard,
		})
	}()
	waitConnected(t, &out)

	// The dial is pooled and idle, so the origin must not have seen a
	// connection at all.
	time.Sleep(300 * time.Millisecond)
	if got := conns.Load(); got != 0 {
		t.Fatalf("idle dial opened %d origin connection(s); want 0", got)
	}

	// That same dial still serves a request.
	body, status := visitorGet(t, ts.URL, "idle.example.com", "/ping")
	if status != http.StatusOK || body != "from-origin:/ping" {
		t.Fatalf("status %d body %q", status, body)
	}
	if got := conns.Load(); got != 1 {
		t.Fatalf("origin connections after one request = %d; want 1", got)
	}
}

// The edge must keep an idle dial warm across the public hop, and it must notice
// when the peer stops answering.
func TestPooledConnKeepalivePings(t *testing.T) {
	oldPing, oldPong := pooledPingInterval, pooledPongWait
	pooledPingInterval, pooledPongWait = 25*time.Millisecond, 2*time.Second
	t.Cleanup(func() { pooledPingInterval, pooledPongWait = oldPing, oldPong })

	pings := make(chan struct{}, 1)
	var once sync.Once
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		c.SetPingHandler(func(appData string) error {
			once.Do(func() { close(pings) })
			return c.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
		})
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(peer.Close)

	target, err := url.Parse(peer.URL)
	if err != nil {
		t.Fatal(err)
	}
	d := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.Dial(network, target.Host)
		},
	}
	c, _, err := d.Dial("ws://"+target.Host+"/dial/idle", nil)
	if err != nil {
		t.Fatal(err)
	}
	pc := newPooledConn(c)
	t.Cleanup(func() { pc.Close() })

	select {
	case <-pings:
	case <-time.After(3 * time.Second):
		t.Fatal("edge never sent a keepalive ping on an idle dial")
	}

	// Answered pings keep the dial alive.
	time.Sleep(10 * pooledPingInterval)
	if !pc.alive() {
		t.Fatal("dial retired even though the origin kept answering keepalive pings")
	}
}

// A dial that died while pooled is retired instead of being handed to a visitor,
// so the edge answers 503 (no origin) rather than 502 from a corpse.
func TestDeadDialIsNotHandedOut(t *testing.T) {
	s, ts := testServer(t, "secret")
	cli := &APIClient{BaseURL: ts.URL, Token: "secret"}
	view, err := cli.AddMapping("dead.example.com")
	if err != nil {
		t.Fatal(err)
	}
	target, err := url.Parse(view.DialURL)
	if err != nil {
		t.Fatal(err)
	}
	d := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.Dial(network, target.Host)
		},
	}
	c, _, err := d.Dial(view.DialURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Close it without ever serving a request, the way a reaped origin
	// connection used to leave a corpse in the pool.
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, connected := s.mappingCount(); connected == 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, connected := s.mappingCount(); connected != 0 {
		t.Fatalf("dead dial still counted as connected: %d", connected)
	}

	for i := 0; i < 5; i++ {
		if _, status := visitorGet(t, ts.URL, "dead.example.com", "/ping"); status != http.StatusServiceUnavailable {
			t.Fatalf("request %d: status %d; want 503", i, status)
		}
	}
}

// Each dial serves exactly one request, so the pool must refill promptly after
// every served request instead of backing off into 503s.
func TestPoolRefillsAfterServedRequest(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Connection", "close")
		_, _ = io.WriteString(w, "served:"+r.URL.Path)
	}))
	t.Cleanup(origin.Close)

	_, ts := testServer(t, "secret")
	cli := &APIClient{BaseURL: ts.URL, Token: "secret"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var out syncBuffer
	go func() {
		_ = cli.Publish(ctx, PublishOpts{
			Hostname: "refill.example.com",
			Forward:  origin.URL,
			Pool:     1,
			Stdout:   &out,
			Stderr:   io.Discard,
		})
	}()
	waitConnected(t, &out)

	for i := 0; i < 3; i++ {
		deadline := time.Now().Add(5 * time.Second)
		var body string
		var status int
		for time.Now().Before(deadline) {
			body, status = visitorGet(t, ts.URL, "refill.example.com", "/ping")
			if status == http.StatusOK {
				break
			}
			time.Sleep(25 * time.Millisecond)
		}
		if status != http.StatusOK || body != "served:/ping" {
			t.Fatalf("request %d: status %d body %q", i, status, body)
		}
	}
}
