package portforward

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xhd2015/ai-critic/server/config"
)

func TestAuthenticatedOriginWebSocketAndDeletion(t *testing.T) {
	closed := make(chan struct{})
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("mapping bearer leaked to upstream")
		}
		if r.Header.Get("Origin") != "https://public.example" {
			t.Error("Origin changed")
		}
		u := websocket.Upgrader{Subprotocols: []string{"fixture"}, CheckOrigin: func(*http.Request) bool { return true }}
		c, err := u.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		defer close(closed)
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
	defer up.Close()
	u, _ := url.Parse(up.URL)
	pnum, _ := strconv.Atoi(u.Port())
	target := config.OriginTarget{Scheme: "http", Host: u.Hostname(), Port: pnum}
	m := NewManager()
	m.authSecretPath = filepath.Join(t.TempDir(), "secret")
	p := &testProvider{name: ProviderCloudflareOwned}
	m.RegisterProvider(p)
	f, err := m.AddForward(ForwardRequest{Target: &target, Provider: p.name, Label: "public.example", Auth: config.PortAuth{Required: true, TokenMode: config.PortAuthCustom, Token: "fixture-secret"}})
	if err != nil {
		t.Fatal(err)
	}
	defer m.RemoveSelector(f.ID)
	wsURL := strings.Replace(fmtHop(p.LastPort()), "http:", "ws:", 1)
	c, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		c.Close()
		t.Fatal("unauthorized WebSocket accepted")
	}
	if resp.StatusCode != 401 {
		t.Fatalf("want 401 got %s", resp.Status)
	}
	resp.Body.Close()
	dialer := websocket.Dialer{Subprotocols: []string{"fixture"}, HandshakeTimeout: time.Second}
	c, _, err = dialer.Dial(wsURL, http.Header{"Authorization": {"Bearer fixture-secret"}, "Origin": {"https://public.example"}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if c.Subprotocol() != "fixture" {
		t.Fatal("WS subprotocol lost")
	}
	if err = c.WriteMessage(websocket.BinaryMessage, []byte("binary")); err != nil {
		t.Fatal(err)
	}
	kind, b, err := c.ReadMessage()
	if err != nil || kind != websocket.BinaryMessage || string(b) != "binary" {
		t.Fatalf("echo %d %s %v", kind, b, err)
	}
	if err = m.RemoveSelector(f.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("mapping deletion leaked hijacked upstream WS")
	}
}
