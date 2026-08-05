package sshcmd

import (
	"errors"
	"io"
	"net"
	"sync"
)

// DialFunc opens the remote side of a tunnel for each accepted local connection.
type DialFunc func() (net.Conn, error)

// LocalRelay listens on 127.0.0.1:0 and splices each accept to Dial().
type LocalRelay struct {
	Dial DialFunc

	mu       sync.Mutex
	ln       net.Listener
	port     int
	conns    map[net.Conn]struct{}
	closed   bool
	acceptWG sync.WaitGroup
}

// Start binds 127.0.0.1:0 and runs the accept loop in the background.
func (r *LocalRelay) Start() error {
	if r == nil {
		return errors.New("LocalRelay is nil")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ln != nil {
		return errors.New("LocalRelay already started")
	}
	if r.Dial == nil {
		return errors.New("LocalRelay dial not configured")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		_ = ln.Close()
		return errors.New("LocalRelay: unexpected listen address type")
	}
	r.ln = ln
	r.port = addr.Port
	r.conns = make(map[net.Conn]struct{})
	r.closed = false
	r.acceptWG.Add(1)
	go r.acceptLoop(ln)
	return nil
}

// LocalPort returns the bound ephemeral port after Start.
func (r *LocalRelay) LocalPort() int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.port
}

// Close stops the listener and closes active connections.
func (r *LocalRelay) Close() error {
	if r == nil {
		return errors.New("LocalRelay is nil")
	}
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	ln := r.ln
	r.ln = nil
	conns := r.conns
	r.conns = nil
	r.mu.Unlock()

	var firstErr error
	if ln != nil {
		if err := ln.Close(); err != nil {
			firstErr = err
		}
	}
	for c := range conns {
		_ = c.Close()
	}
	r.acceptWG.Wait()
	return firstErr
}

func (r *LocalRelay) acceptLoop(ln net.Listener) {
	defer r.acceptWG.Done()
	for {
		local, err := ln.Accept()
		if err != nil {
			return
		}
		r.mu.Lock()
		if r.closed || r.conns == nil {
			r.mu.Unlock()
			_ = local.Close()
			return
		}
		r.conns[local] = struct{}{}
		dial := r.Dial
		r.mu.Unlock()

		go r.handleConn(local, dial)
	}
}

func (r *LocalRelay) handleConn(local net.Conn, dial DialFunc) {
	defer func() {
		_ = local.Close()
		r.untrackConn(local)
	}()

	if dial == nil {
		return
	}
	remote, err := dial()
	if err != nil {
		return
	}
	if !r.trackConn(remote) {
		_ = remote.Close()
		return
	}
	defer func() {
		_ = remote.Close()
		r.untrackConn(remote)
	}()

	// Bidirectional copy until either side closes.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(remote, local)
		_ = closeWrite(remote)
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(local, remote)
		_ = closeWrite(local)
	}()
	wg.Wait()
}

// trackConn registers c; returns false if relay is closed (caller should close c).
func (r *LocalRelay) trackConn(c net.Conn) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed || r.conns == nil {
		return false
	}
	r.conns[c] = struct{}{}
	return true
}

func (r *LocalRelay) untrackConn(c net.Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.conns != nil {
		delete(r.conns, c)
	}
}

// closeWrite half-closes a TCP conn for write when possible.
func closeWrite(c net.Conn) error {
	type closeWriter interface {
		CloseWrite() error
	}
	if cw, ok := c.(closeWriter); ok {
		return cw.CloseWrite()
	}
	return nil
}
