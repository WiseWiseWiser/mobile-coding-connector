package sshcmd

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/crypto/ssh"
)

// ClientKeyPair holds an ed25519 client key pair for SSH public-key auth.
type ClientKeyPair struct {
	Signer ssh.Signer
	Public ssh.PublicKey
}

// GenerateClientKeyPair creates a fresh ed25519 client key pair.
func GenerateClientKeyPair() (*ClientKeyPair, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		return nil, err
	}
	return &ClientKeyPair{
		Signer: signer,
		Public: signer.PublicKey(),
	}, nil
}

// AdhocServer is an in-process SSH server for tests and local compose.
// Listens on 127.0.0.1:0; accepts public-key auth; session channels support
// remote command exec and login shell (with optional PTY).
type AdhocServer struct {
	// User is the accepted SSH username (default "agent").
	User string
	// Interactive is deprecated in favor of: hasPty && !ForcePipeShell.
	// Kept for callers; when true with hasPty, forces human shell path.
	Interactive bool
	// ForcePipeShell keeps the deterministic pipe shell + empty PS1 used by
	// doctests. Production must leave this false. When a client requests a
	// PTY and ForcePipeShell is false, a human interactive shell is used.
	ForcePipeShell bool

	mu          sync.Mutex
	ln          net.Listener
	port        int
	hostSigner  ssh.Signer
	hostPub     ssh.PublicKey
	authorized  []ssh.PublicKey
	closed      bool
	acceptWG    sync.WaitGroup
	sessionWG   sync.WaitGroup
}

// SetAuthorizedKeys replaces the authorized public keys used for auth.
func (s *AdhocServer) SetAuthorizedKeys(keys []ssh.PublicKey) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authorized = append([]ssh.PublicKey(nil), keys...)
}

// Start listens on 127.0.0.1:0 and accepts SSH connections in the background.
func (s *AdhocServer) Start() error {
	if s == nil {
		return errors.New("AdhocServer is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ln != nil {
		return errors.New("AdhocServer already started")
	}

	hostSigner, err := generateHostSigner()
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		_ = ln.Close()
		return errors.New("AdhocServer: unexpected listen address type")
	}

	s.ln = ln
	s.port = addr.Port
	s.hostSigner = hostSigner
	s.hostPub = hostSigner.PublicKey()
	s.closed = false

	s.acceptWG.Add(1)
	go s.acceptLoop(ln)
	return nil
}

// Addr returns "127.0.0.1:<port>" after Start.
func (s *AdhocServer) Addr() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.port <= 0 {
		return ""
	}
	return fmt.Sprintf("127.0.0.1:%d", s.port)
}

// LocalPort returns the bound ephemeral port after Start.
func (s *AdhocServer) LocalPort() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.port
}

// HostKey returns the server host public key after Start.
func (s *AdhocServer) HostKey() ssh.PublicKey {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hostPub
}

// Close stops the listener and waits for accept/session handlers.
func (s *AdhocServer) Close() error {
	if s == nil {
		return errors.New("AdhocServer is nil")
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	ln := s.ln
	s.ln = nil
	s.mu.Unlock()

	var firstErr error
	if ln != nil {
		if err := ln.Close(); err != nil {
			firstErr = err
		}
	}
	s.acceptWG.Wait()
	s.sessionWG.Wait()
	return firstErr
}

func generateHostSigner() (ssh.Signer, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return ssh.NewSignerFromKey(priv)
}

func (s *AdhocServer) acceptLoop(ln net.Listener) {
	defer s.acceptWG.Done()
	for {
		nConn, err := ln.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		closed := s.closed
		s.mu.Unlock()
		if closed {
			_ = nConn.Close()
			return
		}
		s.sessionWG.Add(1)
		go s.handleConn(nConn)
	}
}

func (s *AdhocServer) handleConn(nConn net.Conn) {
	defer s.sessionWG.Done()
	defer nConn.Close()

	cfg := s.serverConfig()
	if cfg == nil {
		return
	}
	sshConn, chans, reqs, err := ssh.NewServerConn(nConn, cfg)
	if err != nil {
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		s.sessionWG.Add(1)
		go s.handleSession(newChannel)
	}
}

func (s *AdhocServer) serverConfig() *ssh.ServerConfig {
	s.mu.Lock()
	user := s.User
	if user == "" {
		user = "agent"
	}
	hostSigner := s.hostSigner
	authorized := append([]ssh.PublicKey(nil), s.authorized...)
	s.mu.Unlock()

	if hostSigner == nil {
		return nil
	}

	cfg := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			if conn.User() != user {
				return nil, fmt.Errorf("unknown user %q", conn.User())
			}
			for _, ak := range authorized {
				if keysEqual(ak, key) {
					return &ssh.Permissions{}, nil
				}
			}
			return nil, fmt.Errorf("public key rejected for %q", conn.User())
		},
	}
	cfg.AddHostKey(hostSigner)
	return cfg
}

func keysEqual(a, b ssh.PublicKey) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Type() == b.Type() && bytes.Equal(a.Marshal(), b.Marshal())
}

type ptyRequestMsg struct {
	Term     string
	Columns  uint32
	Rows     uint32
	Width    uint32
	Height   uint32
	Modelist string
}

type execRequestMsg struct {
	Command string
}

type exitStatusMsg struct {
	Status uint32
}

func (s *AdhocServer) handleSession(newChannel ssh.NewChannel) {
	defer s.sessionWG.Done()

	ch, reqs, err := newChannel.Accept()
	if err != nil {
		return
	}
	defer ch.Close()

	hasPty := false
	var rows, cols uint16 = 40, 80

	for req := range reqs {
		switch req.Type {
		case "env":
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
		case "pty-req":
			var msg ptyRequestMsg
			if err := ssh.Unmarshal(req.Payload, &msg); err == nil {
				if msg.Rows > 0 {
					rows = uint16(msg.Rows)
				}
				if msg.Columns > 0 {
					cols = uint16(msg.Columns)
				}
			}
			hasPty = true
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
		case "window-change":
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
		case "shell":
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
			// Process remaining channel requests while shell runs (avoid deadlock).
			go discardSSHRequests(reqs)
			s.runShell(ch, hasPty, rows, cols)
			return
		case "exec":
			var msg execRequestMsg
			_ = ssh.Unmarshal(req.Payload, &msg)
			if req.WantReply {
				_ = req.Reply(true, nil)
			}
			// Do not range reqs after exec: client waits for channel close / exit-status;
			// blocking on reqs deadlocks the session.
			go discardSSHRequests(reqs)
			s.runCommand(ch, msg.Command)
			return
		default:
			if req.WantReply {
				_ = req.Reply(false, nil)
			}
		}
	}
}

func discardSSHRequests(reqs <-chan *ssh.Request) {
	for req := range reqs {
		if req.WantReply {
			_ = req.Reply(false, nil)
		}
	}
}

func (s *AdhocServer) runCommand(ch ssh.Channel, command string) {
	cmd := exec.Command("sh", "-c", command)
	// Stderr goes to SSH stderr stream when available.
	cmd.Stdout = ch
	cmd.Stderr = ch.Stderr()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		sendExitStatus(ch, 1)
		return
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		sendExitStatus(ch, 1)
		return
	}
	go func() {
		_, _ = io.Copy(stdin, ch)
		_ = stdin.Close()
	}()
	waitErr := cmd.Wait()
	sendExitStatus(ch, exitCodeFromErr(waitErr))
	// Half-close so client session.Run observes EOF after exit-status.
	_ = ch.CloseWrite()
}

// runShell starts a login-capable shell.
//
// Human path (client requested PTY and ForcePipeShell is false):
// host PTY + interactive shell with a visible PS1.
// Test path (ForcePipeShell or no PTY): pipes + empty PS1 for doctests.
// Always sends SSH exit-status before returning.
func (s *AdhocServer) runShell(ch ssh.Channel, hasPty bool, rows, cols uint16) {
	forcePipe := true
	interactiveFlag := false
	if s != nil {
		s.mu.Lock()
		forcePipe = s.ForcePipeShell
		interactiveFlag = s.Interactive
		s.mu.Unlock()
	}
	// OpenSSH -tt and real terminals always send pty-req. Treat that as human
	// login unless tests force the pipe shell.
	human := hasPty && !forcePipe
	if !human && interactiveFlag && hasPty {
		human = true
	}
	if human {
		if s.runShellPTY(ch, rows, cols) {
			return
		}
		// PTY failed: human-friendly pipe shell with a basic prompt.
		shell := selectLoginShell()
		cmd := humanShellCmd(shell)
		s.runShellPipes(ch, cmd)
		return
	}
	// Prefer a plain POSIX shell so echo/exit sequences are deterministic.
	shell := "/bin/sh"
	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm", "PS1=", "PROMPT_COMMAND=")
	s.runShellPipes(ch, cmd)
}

// humanShellCmd builds an interactive shell with a predictable visible prompt.
func humanShellCmd(shell string) *exec.Cmd {
	// -i: interactive. Explicit PS1 so users always see a prompt even when
	// remote profiles are empty or non-interactive heuristics fire.
	base := filepath.Base(shell)
	var cmd *exec.Cmd
	switch base {
	case "bash":
		// --norc/--noprofile: fast, predictable prompt (no slow cloud rc).
		cmd = exec.Command(shell, "--norc", "--noprofile", "-i")
	case "zsh":
		cmd = exec.Command(shell, "-f", "-i") // -f: no rc
	default:
		cmd = exec.Command(shell, "-i")
	}
	cmd.Env = append(os.Environ(),
		"TERM="+envOr("TERM", "xterm-256color"),
		// Visible, simple prompt for remote-agent ssh login.
		"PS1=[remote-agent \\W]\\$ ",
		"PROMPT_COMMAND=",
		"REMOTE_AGENT_SSH=1",
	)
	return cmd
}

func (s *AdhocServer) runShellPipes(ch ssh.Channel, cmd *exec.Cmd) {
	code := 1
	defer func() {
		sendExitStatus(ch, code)
		_ = ch.CloseWrite()
		// Unblock any reader still waiting on the channel.
		_ = ch.Close()
	}()

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return
	}
	cmd.Stdout = ch
	cmd.Stderr = ch
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return
	}
	go func() {
		_, _ = io.Copy(stdin, ch)
		_ = stdin.Close()
	}()
	code = exitCodeFromErr(cmd.Wait())
	// Stop feeding the dead shell so the stdin-copy goroutine can finish.
	_ = stdin.Close()
}

// runShellPTY attaches a human shell to a host PTY bridged to the SSH channel.
// Returns true if the PTY path was used (caller must not also run pipes).
//
// Important: after the shell process exits we must send SSH exit-status and
// close the channel without waiting forever on the channel→PTY copy (client
// often keeps stdin open until it sees session end). Waiting on that copy
// deadlocks interactive `exit`.
func (s *AdhocServer) runShellPTY(ch ssh.Channel, rows, cols uint16) bool {
	if rows == 0 {
		rows = 24
	}
	if cols == 0 {
		cols = 80
	}
	cmd := humanShellCmd(selectLoginShell())
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: rows, Cols: cols})
	if err != nil {
		return false
	}

	inputDone := make(chan struct{})
	go func() {
		defer close(inputDone)
		_, _ = io.Copy(ptmx, ch)
	}()
	go func() {
		_, _ = io.Copy(ch, ptmx)
	}()

	waitErr := cmd.Wait()
	code := exitCodeFromErr(waitErr)

	// Tear down PTY first so the shell side is fully dead.
	_ = ptmx.Close()

	// Send exit-status while the channel is still usable, then close so the
	// client (OpenSSH) completes session.Wait / returns to the local shell.
	sendExitStatus(ch, code)
	_ = ch.CloseWrite()
	_ = ch.Close()

	// Best-effort: don't block forever if client still has stdin open.
	select {
	case <-inputDone:
	case <-time.After(2 * time.Second):
	}
	return true
}

// selectLoginShell picks $SHELL, else bash if present, else sh -l style shell.
func selectLoginShell() string {
	if sh := os.Getenv("SHELL"); sh != "" {
		if _, err := os.Stat(sh); err == nil {
			return sh
		}
	}
	for _, cand := range []string{"/bin/bash", "/usr/bin/bash", "/bin/zsh", "/bin/sh"} {
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	return "/bin/sh"
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func sendExitStatus(ch ssh.Channel, code int) {
	if code < 0 {
		code = 1
	}
	_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(&exitStatusMsg{Status: uint32(code)}))
}

func exitCodeFromErr(err error) int {
	if err == nil {
		return 0
	}
	if ee, ok := err.(*exec.ExitError); ok {
		if status, ok := ee.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
		return ee.ExitCode()
	}
	return 1
}
