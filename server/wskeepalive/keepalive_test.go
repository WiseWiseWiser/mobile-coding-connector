package wskeepalive

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestStartPingsIdleConnection proves the helper pings a connection that carries
// no application traffic, which is what keeps an idle proxied WebSocket from
// being reaped.
func TestStartPingsIdleConnection(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	pings := make(chan struct{}, 8)
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		c.SetPingHandler(func(appData string) error {
			select {
			case pings <- struct{}{}:
			default:
			}
			return c.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
		})
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer peer.Close()

	target, err := url.Parse(peer.URL)
	if err != nil {
		t.Fatal(err)
	}
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
		NetDial: func(network, addr string) (net.Conn, error) {
			return net.Dial(network, target.Host)
		},
	}
	conn, _, err := dialer.Dial("ws://"+target.Host+"/idle", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	stop := Start(conn, 25*time.Millisecond)
	select {
	case <-pings:
	case <-time.After(3 * time.Second):
		t.Fatal("no keepalive ping arrived on an idle connection")
	}

	stop()
	for len(pings) > 0 {
		<-pings
	}
	time.Sleep(120 * time.Millisecond)
	if len(pings) != 0 {
		t.Fatal("pings continued after stop")
	}
}

// TestStartWithoutConnection keeps a handler that never upgraded from panicking,
// and stop must be safe to call more than once.
func TestStartWithoutConnection(t *testing.T) {
	stop := Start(nil, 0)
	stop()
	stop()
}
