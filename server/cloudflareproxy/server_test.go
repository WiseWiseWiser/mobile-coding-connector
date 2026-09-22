package cloudflareproxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

	var out strings.Builder
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
	var out strings.Builder
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
	var out strings.Builder
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
