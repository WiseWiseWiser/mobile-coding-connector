package cloudflareproxy

import (
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Mapping is a persisted hostname publication.
type Mapping struct {
	ID        string    `json:"id"`
	Hostname  string    `json:"hostname"`
	DialToken string    `json:"dial_token"`
	CreatedAt time.Time `json:"created_at"`
}

// MappingView is the public list/add payload (no dial token).
type MappingView struct {
	ID       string `json:"id"`
	Hostname string `json:"hostname"`
	WS       int    `json:"ws"`
	URL      string `json:"url"`
	DialURL  string `json:"dial_url,omitempty"`
}

// Keepalive and liveness knobs for pooled dial sockets. They are variables only
// so tests can compress the intervals.
var (
	// pooledPingInterval is how often the edge pings an idle dial socket. The
	// dial path crosses Cloudflare, which reaps idle proxied WebSockets; the
	// ping keeps traffic moving in both directions, because the origin's socket
	// reader answers it automatically.
	pooledPingInterval = 20 * time.Second
	// pooledPongWait is how long the edge tolerates hearing nothing at all from
	// the origin. Every ping is answered, so sustained silence means the dial is
	// gone and must not be handed to a visitor.
	pooledPongWait = 70 * time.Second
)

const (
	// pooledWriteWait bounds a single control-frame write.
	pooledWriteWait = 10 * time.Second
	// pooledFrameQueue is how many response frames the reader may buffer ahead
	// of the request that is consuming them.
	pooledFrameQueue = 128
)

// pooledConn is one origin dial socket held in the edge's ready pool.
//
// A single reader goroutine owns every read for the socket's whole life: it
// answers keepalive pings, refreshes the liveness deadline, and retires the
// socket the moment the peer goes away. A request reads the frames that reader
// queues, so a socket that died while pooled is never handed to a visitor as if
// it were live.
type pooledConn struct {
	ws     *websocket.Conn
	frames chan []byte
	closed chan struct{}

	closeOnce sync.Once
	mu        sync.Mutex
	dead      bool

	// buf holds the unread tail of the current frame. Only the request
	// goroutine that took this socket touches it.
	buf []byte
}

func newPooledConn(ws *websocket.Conn) *pooledConn {
	p := &pooledConn{
		ws:     ws,
		frames: make(chan []byte, pooledFrameQueue),
		closed: make(chan struct{}),
	}
	go p.readLoop()
	go p.pingLoop()
	return p
}

// readLoop is the socket's only reader.
func (p *pooledConn) readLoop() {
	_ = p.ws.SetReadDeadline(time.Now().Add(pooledPongWait))
	p.ws.SetPongHandler(func(string) error {
		return p.ws.SetReadDeadline(time.Now().Add(pooledPongWait))
	})
	for {
		mt, msg, err := p.ws.ReadMessage()
		if err != nil {
			p.markDead()
			return
		}
		_ = p.ws.SetReadDeadline(time.Now().Add(pooledPongWait))
		if mt != websocket.TextMessage && mt != websocket.BinaryMessage {
			continue
		}
		select {
		case p.frames <- msg:
		case <-p.closed:
			return
		}
	}
}

// pingLoop keeps an idle dial alive across the public hop.
func (p *pooledConn) pingLoop() {
	t := time.NewTicker(pooledPingInterval)
	defer t.Stop()
	for {
		select {
		case <-p.closed:
			return
		case <-t.C:
			// WriteControl is safe alongside a request's data writes.
			if err := p.ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(pooledWriteWait)); err != nil {
				p.markDead()
				return
			}
		}
	}
}

func (p *pooledConn) alive() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return !p.dead
}

// markDead retires the socket exactly once and unblocks its loops.
func (p *pooledConn) markDead() {
	p.closeOnce.Do(func() {
		p.mu.Lock()
		p.dead = true
		p.mu.Unlock()
		close(p.closed)
		_ = p.ws.Close()
	})
}

func (p *pooledConn) Read(b []byte) (int, error) {
	for {
		if len(p.buf) > 0 {
			n := copy(b, p.buf)
			p.buf = p.buf[n:]
			return n, nil
		}
		select {
		case msg := <-p.frames:
			p.buf = msg
		case <-p.closed:
			// Serve frames that arrived before the socket retired, then report
			// the stream as finished.
			select {
			case msg := <-p.frames:
				p.buf = msg
				continue
			default:
			}
			return 0, io.EOF
		}
	}
}

func (p *pooledConn) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}
	if err := p.ws.WriteMessage(websocket.BinaryMessage, b); err != nil {
		p.markDead()
		return 0, err
	}
	return len(b), nil
}

func (p *pooledConn) Close() error {
	p.markDead()
	return nil
}

func (p *pooledConn) LocalAddr() net.Addr  { return p.ws.LocalAddr() }
func (p *pooledConn) RemoteAddr() net.Addr { return p.ws.RemoteAddr() }

// The deadline setters are deliberate no-ops. The reader goroutine owns the
// socket's read deadline, and gorilla treats a read timeout as fatal, so a
// deadline set here by the HTTP transport would kill a healthy dial.
func (p *pooledConn) SetDeadline(time.Time) error      { return nil }
func (p *pooledConn) SetReadDeadline(time.Time) error  { return nil }
func (p *pooledConn) SetWriteDeadline(time.Time) error { return nil }

type liveMapping struct {
	Mapping
	mu   sync.Mutex
	idle []*pooledConn
}

// wsCount reports how many pooled dials are still usable.
func (m *liveMapping) wsCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, c := range m.idle {
		if c.alive() {
			n++
		}
	}
	return n
}

// take returns a live idle dial, retiring any that died while pooled. A dead
// socket must never reach a visitor: writing a request into one is what the edge
// used to answer with an instant 502.
func (m *liveMapping) take() *pooledConn {
	m.mu.Lock()
	defer m.mu.Unlock()
	for len(m.idle) > 0 {
		n := len(m.idle)
		c := m.idle[n-1]
		m.idle = m.idle[:n-1]
		if c.alive() {
			return c
		}
		c.markDead()
	}
	return nil
}

func (m *liveMapping) put(c *pooledConn) {
	if c == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.idle = append(m.idle, c)
}

func (m *liveMapping) closeAll() {
	m.mu.Lock()
	conns := m.idle
	m.idle = nil
	m.mu.Unlock()
	for _, c := range conns {
		c.markDead()
	}
}

func normalizeHostname(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	h = strings.TrimSuffix(h, ".")
	if i := strings.IndexByte(h, ':'); i >= 0 {
		h = h[:i]
	}
	return h
}

func publicURL(hostname string) string {
	if hostname == "" {
		return ""
	}
	return "https://" + hostname
}

func loadMappingsFile(path string) ([]Mapping, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var list []Mapping
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

func saveMappingsFile(path string, list []Mapping) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
