package sshcmd

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
)

// DialFunc opens the remote side of a tunnel for each accepted local connection.
type DialFunc func() (net.Conn, error)

// LocalRelay listens on a local endpoint and splices each accepted connection to Dial.
// Network defaults to tcp and Address defaults to 127.0.0.1:0 for compatibility.
// Unix listeners are removed when Close completes.
type LocalRelay struct {
	Dial    DialFunc
	Network string
	Address string

	mu       sync.Mutex
	ln       net.Listener
	port     int
	address  string
	conns    map[net.Conn]struct{}
	closed   bool
	acceptWG sync.WaitGroup
}

// Start binds the configured local endpoint and runs the accept loop in the background.
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
	network := r.Network
	if network == "" {
		network = "tcp"
	}
	address := r.Address
	if address == "" {
		address = "127.0.0.1:0"
	}
	if network == "unix" {
		// macOS supports at most 103 bytes plus the NUL terminator in sun_path.
		if len([]byte(address)) >= 104 {
			return fmt.Errorf("unix relay socket path is too long (%d bytes): %s", len([]byte(address)), address)
		}
		if err := os.MkdirAll(filepath.Dir(address), 0o700); err != nil {
			return err
		}
		if _, err := os.Lstat(address); err == nil {
			return fmt.Errorf("unix relay socket already exists: %s", address)
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	ln, err := net.Listen(network, address)
	if err != nil {
		return err
	}
	if network == "unix" {
		if err := os.Chmod(address, 0o600); err != nil {
			_ = ln.Close()
			_ = os.Remove(address)
			return err
		}
	}
	r.ln = ln
	r.address = address
	if addr, ok := ln.Addr().(*net.TCPAddr); ok {
		r.port = addr.Port
	}
	r.conns = make(map[net.Conn]struct{})
	r.closed = false
	r.acceptWG.Add(1)
	go r.acceptLoop(ln)
	return nil
}

// LocalPort returns the bound TCP port after Start, or zero for a Unix relay.
func (r *LocalRelay) LocalPort() int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.port
}

// LocalAddress returns the configured Unix socket path or the actual TCP address.
func (r *LocalRelay) LocalAddress() string {
	if r == nil {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ln != nil {
		return r.ln.Addr().String()
	}
	return r.address
}

// Close stops the listener, closes active connections, and removes a Unix socket.
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
	network := r.Network
	address := r.address
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
	if network == "unix" && address != "" {
		if err := os.Remove(address); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
	}
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

func closeWrite(c net.Conn) error {
	type closeWriter interface{ CloseWrite() error }
	if cw, ok := c.(closeWriter); ok {
		return cw.CloseWrite()
	}
	return nil
}
