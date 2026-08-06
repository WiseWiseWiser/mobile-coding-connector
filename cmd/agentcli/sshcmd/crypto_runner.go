package sshcmd

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// CryptoSSHRunner implements SSHRunner via golang.org/x/crypto/ssh (commands)
// or the system OpenSSH client (interactive login TTY).
type CryptoSSHRunner struct {
	Signer                ssh.Signer
	Stdout                io.Writer
	Stderr                io.Writer
	Stdin                 io.Reader
	InsecureIgnoreHostKey bool
	// HostKeyCallback overrides host key verification when non-nil.
	HostKeyCallback ssh.HostKeyCallback
	// ForceCrypto disables the OpenSSH interactive path (tests).
	ForceCrypto bool
	// SSHPath overrides the OpenSSH binary (default "ssh" on PATH).
	SSHPath string
}

// Run dials the session local port and either starts an interactive shell
// (empty argv) or runs the joined remote command.
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

	// Interactive login: use system OpenSSH against generated ssh_config.
	// The pure-Go Shell() path does not reliably provide PS1/echo over our
	// ad-hoc server + tunnel; OpenSSH handles TTY correctly.
	if len(remoteArgv) == 0 && !r.ForceCrypto && r.shouldUseOpenSSH(sess) {
		return r.runOpenSSH(sess, nil)
	}

	return r.runCrypto(sess, remoteArgv)
}

func (r *CryptoSSHRunner) shouldUseOpenSSH(sess *Session) bool {
	if sess.ConfigDir == "" {
		return false
	}
	cfg := filepath.Join(sess.ConfigDir, "ssh_config")
	if _, err := os.Stat(cfg); err != nil {
		return false
	}
	id := filepath.Join(sess.ConfigDir, "id_ed25519")
	if _, err := os.Stat(id); err != nil {
		return false
	}
	// Prefer OpenSSH for real TTYs; non-TTY scripted login stays on crypto path.
	fd := stdinFD(r.Stdin)
	if fd < 0 || !term.IsTerminal(fd) {
		return false
	}
	sshBin := r.SSHPath
	if sshBin == "" {
		sshBin = "ssh"
	}
	if _, err := exec.LookPath(sshBin); err != nil {
		return false
	}
	return true
}

// runOpenSSH runs: ssh -F <config> [-tt] remote-agent [cmd...]
func (r *CryptoSSHRunner) runOpenSSH(sess *Session, remoteArgv []string) error {
	sshBin := r.SSHPath
	if sshBin == "" {
		sshBin = "ssh"
	}
	cfg := filepath.Join(sess.ConfigDir, "ssh_config")
	args := []string{
		"-F", cfg,
		"-o", "BatchMode=yes",
		"-o", "LogLevel=ERROR",
	}
	// Force pseudo-TTY for interactive login so remote shell is interactive.
	if len(remoteArgv) == 0 {
		args = append(args, "-tt")
	}
	args = append(args, "remote-agent")
	if len(remoteArgv) > 0 {
		// OpenSSH remote command: ssh host -- cmd args
		args = append(args, remoteArgv...)
	}

	cmd := exec.Command(sshBin, args...)
	cmd.Stdin = r.Stdin
	if cmd.Stdin == nil {
		cmd.Stdin = os.Stdin
	}
	cmd.Stdout = r.Stdout
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = r.Stderr
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}

	if len(remoteArgv) == 0 && r.Stderr != nil {
		_, _ = fmt.Fprintf(r.Stderr, "connected 127.0.0.1:%d (openssh)\n", sess.LocalPort)
	}
	return cmd.Run()
}

func (r *CryptoSSHRunner) runCrypto(sess *Session, remoteArgv []string) error {
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
		// Non-TTY scripted login via crypto: no MakeRaw; empty-ish shell for tests.
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

func stdinFD(r io.Reader) int {
	if f, ok := r.(*os.File); ok {
		return int(f.Fd())
	}
	if r == os.Stdin {
		return int(os.Stdin.Fd())
	}
	return -1
}

// DialTCP returns a DialFunc that opens a TCP connection to addr with a timeout.
func DialTCP(addr string) DialFunc {
	return func() (net.Conn, error) {
		return net.DialTimeout("tcp", addr, 5*time.Second)
	}
}
