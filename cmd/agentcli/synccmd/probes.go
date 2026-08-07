package synccmd

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// defaultLocalVersion probes PATH for unison and parses `unison -version`.
func defaultLocalVersion() (string, error) {
	return defaultLocalEnsure()
}

// defaultServeOK checks Host remote-agent via generated ssh_config.
func defaultServeOK(sshConfigDir string) error {
	cfg := filepath.Join(sshConfigDir, "ssh_config")
	cmd := exec.Command("ssh",
		"-F", cfg,
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "LogLevel=ERROR",
		"remote-agent",
		"true",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := runWithTimeout(cmd, 15*time.Second); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("serve not reachable: %s", firstLine(msg))
		}
		return fmt.Errorf("serve not reachable: %w (start: remote-agent ssh --serve)", err)
	}
	return nil
}

// defaultRemoteVersion runs remote unison -version over ssh Host remote-agent.
func defaultRemoteVersion(sshConfigDir, remoteUnison string) (string, error) {
	if strings.TrimSpace(remoteUnison) == "" {
		remoteUnison = "/usr/local/bin/unison"
	}
	cfg := filepath.Join(sshConfigDir, "ssh_config")
	// Remote shell: run absolute path -version.
	remoteCmd := remoteUnison + " -version"
	cmd := exec.Command("ssh",
		"-F", cfg,
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "LogLevel=ERROR",
		"remote-agent",
		remoteCmd,
	)
	out, err := cmdCombinedWithTimeout(cmd, 20*time.Second)
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg != "" {
			return "", fmt.Errorf("remote unison version: %s", firstLine(msg))
		}
		return "", fmt.Errorf("remote unison version: %w", err)
	}
	ver := parseUnisonVersion(string(out))
	if ver == "" {
		return "", fmt.Errorf("could not parse remote unison version from %q", strings.TrimSpace(string(out)))
	}
	return ver, nil
}

// defaultRemotePathOK checks that remote path exists as a directory over ssh.
func defaultRemotePathOK(sshConfigDir, remote string) error {
	if strings.TrimSpace(remote) == "" {
		return fmt.Errorf("remote path is empty")
	}
	cfg := filepath.Join(sshConfigDir, "ssh_config")
	// Quote path for remote shell safely (single quotes + escape).
	q := shellSingleQuote(remote)
	remoteCmd := "test -d " + q
	cmd := exec.Command("ssh",
		"-F", cfg,
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=5",
		"-o", "LogLevel=ERROR",
		"remote-agent",
		remoteCmd,
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := runWithTimeout(cmd, 15*time.Second); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return fmt.Errorf("remote root %s: %s", remote, firstLine(msg))
		}
		return fmt.Errorf("remote root missing or not a directory: %s", remote)
	}
	return nil
}

func shellSingleQuote(s string) string {
	// 'foo'\''bar' style
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func runWithTimeout(cmd *exec.Cmd, d time.Duration) error {
	if err := cmd.Start(); err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(d):
		_ = cmd.Process.Kill()
		<-done
		return fmt.Errorf("timeout after %s", d)
	}
}

func cmdCombinedWithTimeout(cmd *exec.Cmd, d time.Duration) ([]byte, error) {
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := runWithTimeout(cmd, d)
	return buf.Bytes(), err
}
