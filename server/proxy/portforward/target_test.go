package portforward

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/ai-critic/server/config"
)

func originFixture(t *testing.T, h http.Handler) (*Manager, *testProvider, config.OriginTarget) {
	t.Helper()
	up := httptest.NewServer(h)
	t.Cleanup(up.Close)
	u, _ := url.Parse(up.URL)
	port, _ := strconv.Atoi(u.Port())
	target := config.OriginTarget{Scheme: "http", Host: u.Hostname(), Port: port}
	m := NewManager()
	m.authSecretPath = filepath.Join(t.TempDir(), "secret")
	p := &testProvider{name: ProviderCloudflareOwned}
	m.RegisterProvider(p)
	pf, err := m.AddForward(ForwardRequest{Port: port, Target: &target, Provider: p.name, Label: "fixture.example"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = m.RemoveSelector(pf.ID) })
	return m, p, target
}
func TestOriginHopReusesConnectionsAndPreservesHost(t *testing.T) {
	var accepts atomic.Int32
	up := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "public.example" {
			t.Errorf("host rewritten: %s", r.Host)
		}
		io.WriteString(w, "ok")
	}))
	up.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			accepts.Add(1)
		}
	}
	up.Start()
	defer up.Close()
	u, _ := url.Parse(up.URL)
	port, _ := strconv.Atoi(u.Port())
	target := config.OriginTarget{Scheme: "http", Host: u.Hostname(), Port: port}
	m := NewManager()
	p := &testProvider{name: ProviderCloudflareOwned}
	m.RegisterProvider(p)
	pf, err := m.AddForward(ForwardRequest{Target: &target, Provider: p.name, Label: "public.example"})
	if err != nil {
		t.Fatal(err)
	}
	defer m.RemoveSelector(pf.ID)
	// Emulate the edge opening a fresh downstream socket per request.
	edgeTransport := &http.Transport{DisableKeepAlives: true}
	defer edgeTransport.CloseIdleConnections()
	client := &http.Client{Timeout: 3 * time.Second, Transport: edgeTransport}
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest("GET", fmtHop(p.LastPort()), nil)
		req.Host = "public.example"
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, res.Body)
		res.Body.Close()
		if res.StatusCode != 200 {
			t.Fatal(res.Status)
		}
	}
	if accepts.Load() != 1 {
		t.Fatalf("10 requests created %d upstream connections", accepts.Load())
	}
}
func fmtHop(port int) string { return "http://127.0.0.1:" + strconv.Itoa(port) }
func TestOriginHopWebSocketBidirectionalAndCleanup(t *testing.T) {
	closed := make(chan struct{})
	_, p, _ := originFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		c, err := u.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer close(closed)
		defer c.Close()
		for {
			kind, b, e := c.ReadMessage()
			if e != nil {
				return
			}
			if e = c.WriteMessage(kind, b); e != nil {
				return
			}
		}
	}))
	c, resp, err := websocket.DefaultDialer.Dial(strings.Replace(fmtHop(p.LastPort()), "http:", "ws:", 1), nil)
	if err != nil {
		t.Fatalf("upgrade: %v response=%v", err, resp)
	}
	for _, kind := range []int{websocket.TextMessage, websocket.BinaryMessage} {
		if err := c.WriteMessage(kind, []byte("roundtrip")); err != nil {
			t.Fatal(err)
		}
		k, b, err := c.ReadMessage()
		if err != nil || k != kind || string(b) != "roundtrip" {
			t.Fatalf("WS echo: %d %s %v", k, b, err)
		}
	}
	c.Close()
	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("upstream WS not closed")
	}
}
func TestOriginHopSSEStreamsAndCancels(t *testing.T) {
	canceled := make(chan struct{})
	_, p, _ := originFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: first\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
		close(canceled)
	}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", fmtHop(p.LastPort()), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	line, err := bufio.NewReader(resp.Body).ReadString('\n')
	if err != nil || line != "data: first\n" {
		t.Fatalf("SSE buffered: %q %v", line, err)
	}
	cancel()
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("SSE upstream not canceled")
	}
}
func TestDifferentTargetsSamePortHaveIndependentIDs(t *testing.T) {
	m := NewManager()
	p := &testProvider{name: ProviderCloudflareOwned}
	m.RegisterProvider(p)
	a, err := m.AddForward(ForwardRequest{Port: 3080, Label: "pi.example", Provider: p.name})
	if err != nil {
		t.Fatal(err)
	}
	defer m.RemoveSelector(a.ID)
	target := config.OriginTarget{Scheme: "http", Host: "192.168.1.1", Port: 3080}
	b, err := m.AddForward(ForwardRequest{Target: &target, Label: "mac.example", Provider: p.name})
	if err != nil {
		t.Fatal(err)
	}
	defer m.RemoveSelector(b.ID)
	if a.ID == b.ID || len(m.List()) != 2 {
		t.Fatal("mapping collision")
	}
	if err := m.Remove(3080); err == nil {
		t.Fatal("ambiguous port deleted a mapping")
	}
}
func TestMappingAPIStoresTargetAndRestoresAfterRestart(t *testing.T) {
	useTempMappingFiles(t)
	old := defaultManager
	defaultManager = NewManager()
	defer func() { defaultManager = old }()
	p := &testProvider{name: ProviderCloudflareOwned}
	defaultManager.RegisterProvider(p)
	req := httptest.NewRequest("POST", "/api/ports", strings.NewReader(`{"port":3080,"label":"mac.example","provider":"cloudflare_owned","target":{"host":"192.168.1.1","port":3080}}`))
	ctx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(ctx) // no network probe in fixture
	w := httptest.NewRecorder()
	handleAddPort(w, req)
	if w.Code != 201 {
		t.Fatalf("add: %d %s", w.Code, w.Body.String())
	}
	var pf PortForward
	if err := json.Unmarshal(w.Body.Bytes(), &pf); err != nil {
		t.Fatal(err)
	}
	if pf.ID == "" || pf.Target.Host != "192.168.1.1" {
		t.Fatal("target or ID missing")
	}
	_ = defaultManager.RemoveSelector(pf.ID)
	defaultManager.RestoreDesired()
	defer defaultManager.RemoveSelector(pf.ID)
	list := defaultManager.List()
	if len(list) != 1 || list[0].ID != pf.ID || list[0].Target.Host != "192.168.1.1" {
		t.Fatalf("restore lost target: %+v", list)
	}
}

func TestOriginHopDoesNotReplayUnsafePOST(t *testing.T) {
	var posts atomic.Int32
	_, p, _ := originFixture(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts.Add(1)
			_, _ = io.Copy(io.Discard, r.Body)
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				conn.Close()
			}
			return
		}
		io.WriteString(w, "warm")
	}))
	client := &http.Client{Timeout: time.Second}
	res, err := client.Get(fmtHop(p.LastPort()))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, res.Body)
	res.Body.Close()
	res, err = client.Post(fmtHop(p.LastPort()), "application/json", strings.NewReader(`{"mutate":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != 502 || posts.Load() != 1 {
		t.Fatalf("POST status=%d upstream writes=%d", res.StatusCode, posts.Load())
	}
}
