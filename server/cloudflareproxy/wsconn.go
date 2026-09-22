package cloudflareproxy

import (
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// wsNetConn presents a gorilla WebSocket as a stream net.Conn.
// Each Write is one binary message; Read concatenates incoming messages.
type wsNetConn struct {
	ws  *websocket.Conn
	mu  sync.Mutex
	rmu sync.Mutex
	buf []byte
}

func newWSNetConn(ws *websocket.Conn) *wsNetConn {
	return &wsNetConn{ws: ws}
}

func (c *wsNetConn) Read(p []byte) (int, error) {
	c.rmu.Lock()
	defer c.rmu.Unlock()
	for len(c.buf) == 0 {
		_, msg, err := c.ws.ReadMessage()
		if err != nil {
			return 0, err
		}
		c.buf = msg
	}
	n := copy(p, c.buf)
	c.buf = c.buf[n:]
	return n, nil
}

func (c *wsNetConn) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ws.WriteMessage(websocket.BinaryMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *wsNetConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ws.Close()
}

func (c *wsNetConn) LocalAddr() net.Addr  { return c.ws.LocalAddr() }
func (c *wsNetConn) RemoteAddr() net.Addr { return c.ws.RemoteAddr() }

func (c *wsNetConn) SetDeadline(t time.Time) error {
	_ = c.ws.SetReadDeadline(t)
	return c.ws.SetWriteDeadline(t)
}
func (c *wsNetConn) SetReadDeadline(t time.Time) error  { return c.ws.SetReadDeadline(t) }
func (c *wsNetConn) SetWriteDeadline(t time.Time) error { return c.ws.SetWriteDeadline(t) }
