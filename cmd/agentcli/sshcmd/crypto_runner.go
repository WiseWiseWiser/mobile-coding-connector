package sshcmd

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// CryptoSSHRunner implements SSHRunner via golang.org/x/crypto/ssh.
// It dials 127.0.0.1:sess.LocalPort (the LocalRelay / serve side).
type CryptoSSHRunner struct {
	Signer                ssh.Signer
	Stdout                io.Writer
	Stderr                io.Writer
	Stdin                 io.Reader
	InsecureIgnoreHostKey bool
	// HostKeyCallback overrides host key verification when non-nil.
	HostKeyCallback ssh.HostKeyCallback
}

// Run dials the session local port and either starts a PTY shell (empty argv)
// or runs the joined remote command.
func (r *CryptoSSHRunner) Run(sess *Session, remoteArgv []string, opts RunnerOpts) error {
	_ = opts
	if r == nil {
		return errors.New("CryptoSSHRunner is nil")
	}
	if sess == nil {
		return errors.New("session is nil")
	}
	if r.Signer == nil {
		return errors.New("ssh signer not configured")
	}
	if sess.LocalPort <= 0 {
		return fmt.Errorf("invalid session LocalPort %d", sess.LocalPort)
	}

	user := sess.User
	if user == "" {
		user = "agent"
	}

	hostKeyCb := r.HostKeyCallback
	if hostKeyCb == nil {
		if r.InsecureIgnoreHostKey {
			hostKeyCb = ssh.InsecureIgnoreHostKey()
		} else {
			return errors.New("host key callback not configured")
		}
	}

	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(r.Signer)},
		HostKeyCallback: hostKeyCb,
		Timeout:         10 * time.Second,
	}
	addr := fmt.Sprintf("127.0.0.1:%d", sess.LocalPort)
	client, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	if r.Stdout != nil {
		session.Stdout = r.Stdout
	}
	if r.Stderr != nil {
		session.Stderr = r.Stderr
	}
	if r.Stdin != nil {
		session.Stdin = r.Stdin
	}

	if len(remoteArgv) == 0 {
		modes := ssh.TerminalModes{ssh.ECHO: 0}
		if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
			return err
		}
		if err := session.Shell(); err != nil {
			return err
		}
		return session.Wait()
	}

	cmd := strings.Join(remoteArgv, " ")
	return session.Run(cmd)
}

// DialTCP returns a DialFunc that opens a TCP connection to addr with a timeout.
func DialTCP(addr string) DialFunc {
	return func() (net.Conn, error) {
		return net.DialTimeout("tcp", addr, 5*time.Second)
	}
}
