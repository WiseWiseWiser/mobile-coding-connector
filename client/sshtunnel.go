package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// CreateSSHSessionRequest is the JSON body for POST /api/remote-agent/ssh/sessions.
type CreateSSHSessionRequest struct {
	PublicKey string `json:"public_key,omitempty"`
}

// SSHSessionInfo is returned by CreateSSHSession.
type SSHSessionInfo struct {
	SessionID string `json:"session_id"`
	User      string `json:"user"`
	HostKey   string `json:"host_key,omitempty"`
}

// CreateSSHSession creates a remote SSH tunnel session on the agent server.
func (c *Client) CreateSSHSession(req CreateSSHSessionRequest) (*SSHSessionInfo, error) {
	if c == nil {
		return nil, fmt.Errorf("client is nil")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := c.NewRequest(http.MethodPost, "/api/remote-agent/ssh/sessions", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, readAPIError(resp)
	}

	var info SSHSessionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode create ssh session: %w", err)
	}
	if info.SessionID == "" {
		return nil, fmt.Errorf("server returned empty session_id")
	}
	return &info, nil
}

// DestroySSHSession tears down a remote SSH tunnel session.
func (c *Client) DestroySSHSession(sessionID string) error {
	if c == nil {
		return fmt.Errorf("client is nil")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return fmt.Errorf("session id required")
	}
	path := "/api/remote-agent/ssh/sessions/" + url.PathEscape(sessionID)
	httpReq, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	resp, err := c.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return readAPIError(resp)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// SSHTunnelDial opens a duplex binary WebSocket tunnel as a net.Conn.
func (c *Client) SSHTunnelDial(sessionID string) (net.Conn, error) {
	if c == nil {
		return nil, fmt.Errorf("client is nil")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session id required")
	}

	wsURL, err := c.sshTunnelWSURL(sessionID)
	if err != nil {
		return nil, err
	}

	hdr := http.Header{}
	if c.Token != "" {
		hdr.Set("Authorization", "Bearer "+c.Token)
	}

	dialer := *websocket.DefaultDialer
	dialer.HandshakeTimeout = 10 * time.Second
	conn, resp, err := dialer.Dial(wsURL, hdr)
	if err != nil {
		if resp != nil {
			defer resp.Body.Close()
			if resp.StatusCode >= 400 {
				return nil, readAPIError(resp)
			}
		}
		return nil, fmt.Errorf("ssh tunnel dial: %w", err)
	}
	return newWSNetConn(conn), nil
}

// SSHTunnelDialFunc returns a DialFunc that opens a new WS tunnel each call.
func (c *Client) SSHTunnelDialFunc(sessionID string) func() (net.Conn, error) {
	return func() (net.Conn, error) {
		return c.SSHTunnelDial(sessionID)
	}
}

func (c *Client) sshTunnelWSURL(sessionID string) (string, error) {
	base := c.Server
	if base == "" {
		return "", fmt.Errorf("server URL is empty")
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse server URL: %w", err)
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
		// already ws
	default:
		return "", fmt.Errorf("unsupported server scheme %q", u.Scheme)
	}
	// Match NewRequest localhost → 127.0.0.1 rewrite for WS too.
	if u.Hostname() == "localhost" {
		host := u.Host
		u.Host = strings.Replace(host, "localhost", "127.0.0.1", 1)
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/remote-agent/ssh/sessions/" + url.PathEscape(sessionID) + "/tunnel"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

// wsNetConn adapts a gorilla WebSocket to net.Conn using binary frames.
type wsNetConn struct {
	conn *websocket.Conn

	rmu  sync.Mutex
	rbuf []byte

	wmu sync.Mutex

	local  net.Addr
	remote net.Addr
}

func newWSNetConn(conn *websocket.Conn) *wsNetConn {
	return &wsNetConn{
		conn:   conn,
		local:  wsAddr("local"),
		remote: wsAddr("remote"),
	}
}

func (c *wsNetConn) Read(p []byte) (int, error) {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	for len(c.rbuf) == 0 {
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			return 0, err
		}
		if msgType != websocket.BinaryMessage && msgType != websocket.TextMessage {
			continue
		}
		c.rbuf = data
	}
	n := copy(p, c.rbuf)
	c.rbuf = c.rbuf[n:]
	return n, nil
}

func (c *wsNetConn) Write(p []byte) (int, error) {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	// Copy so caller may reuse p after Write returns.
	buf := make([]byte, len(p))
	copy(buf, p)
	if err := c.conn.WriteMessage(websocket.BinaryMessage, buf); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (c *wsNetConn) Close() error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	// Best-effort normal close.
	_ = c.conn.WriteControl(
		websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
		time.Now().Add(time.Second),
	)
	return c.conn.Close()
}

func (c *wsNetConn) LocalAddr() net.Addr  { return c.local }
func (c *wsNetConn) RemoteAddr() net.Addr { return c.remote }

func (c *wsNetConn) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	return c.SetWriteDeadline(t)
}

func (c *wsNetConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *wsNetConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

// CloseWrite is a no-op half-close (WS has no TCP FIN equivalent).
func (c *wsNetConn) CloseWrite() error {
	return nil
}

type wsAddr string

func (a wsAddr) Network() string { return "ws" }
func (a wsAddr) String() string  { return string(a) }
