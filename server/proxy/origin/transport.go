// Package origin owns persistent, direct-to-origin HTTP transports. A transport
// belongs to one mapping, never one request, and is not affected by edge dial pools.
package origin

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/xhd2015/ai-critic/server/config"
)

type Transport struct {
	*http.Transport
	mu        sync.Mutex
	closed    bool
	conns     map[*trackedConn]struct{}
	forbidden map[string]bool
}
type trackedConn struct {
	net.Conn
	owner *Transport
	once  sync.Once
}

func (c *trackedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { c.owner.mu.Lock(); delete(c.owner.conns, c); c.owner.mu.Unlock() })
	return err
}

func NewTransport() *Transport {
	t := &Transport{conns: make(map[*trackedConn]struct{}), forbidden: make(map[string]bool)}
	t.Transport = &http.Transport{
		Proxy:       nil, // Never send LAN credentials through an inherited HTTP_PROXY.
		DialContext: t.dial, ForceAttemptHTTP2: true,
		MaxIdleConns: 32, MaxIdleConnsPerHost: 8, MaxConnsPerHost: 32,
		IdleConnTimeout: 90 * time.Second, TLSHandshakeTimeout: 10 * time.Second,
		ExpectContinueTimeout: time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}
	return t
}

// Forbid blocks the mapping's own ephemeral listener from becoming its origin.
func (t *Transport) Forbid(address string) { t.mu.Lock(); t.forbidden[address] = true; t.mu.Unlock() }
func (t *Transport) dial(ctx context.Context, network, address string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	var addrs []netip.Addr
	if a, e := netip.ParseAddr(host); e == nil {
		addrs = []netip.Addr{a}
	} else {
		ips, e := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
		if e != nil {
			return nil, e
		}
		addrs = ips
	}
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no addresses for target %s", host)
	}
	var last error
	// Resolve once and dial the checked IP, preventing a second DNS lookup/rebind.
	for _, a := range addrs {
		if err := config.ValidateOriginIP(a); err != nil {
			return nil, err
		}
	}
	for _, a := range addrs {
		dest := net.JoinHostPort(a.Unmap().String(), port)
		t.mu.Lock()
		denied, closed := t.forbidden[dest], t.closed
		t.mu.Unlock()
		if denied || closed {
			return nil, fmt.Errorf("origin connection blocked (self-reference or closed mapping)")
		}
		d := net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
		c, e := d.DialContext(ctx, network, dest)
		if e != nil {
			last = e
			continue
		}
		tracked := &trackedConn{Conn: c, owner: t}
		t.mu.Lock()
		if t.closed {
			t.mu.Unlock()
			c.Close()
			return nil, net.ErrClosed
		}
		t.conns[tracked] = struct{}{}
		t.mu.Unlock()
		return tracked, nil
	}
	return nil, last
}

// Close includes upgraded WebSocket connections: http.Server.Shutdown alone
// does not close hijacked streams. Close them explicitly on deletion/update.
func (t *Transport) Close() error {
	t.mu.Lock()
	t.closed = true
	conns := make([]*trackedConn, 0, len(t.conns))
	for c := range t.conns {
		conns = append(conns, c)
	}
	t.mu.Unlock()
	t.CloseIdleConnections()
	for _, c := range conns {
		_ = c.Close()
	}
	return nil
}

func Probe(ctx context.Context, target config.OriginTarget) error {
	t := NewTransport()
	defer t.Close()
	c, err := t.dial(ctx, "tcp", target.Address())
	if err != nil {
		return err
	}
	return c.Close()
}
