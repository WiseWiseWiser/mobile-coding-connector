package synccmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProfileFileName returns remote-agent-<name>.prf.
func ProfileFileName(name string) string {
	return "remote-agent-" + name + ".prf"
}

// sshRemoteRoot builds ssh://remote-agent//abs/path for an absolute remote path.
func sshRemoteRoot(remote string) string {
	// Unison absolute remote form: host + // + path without re-adding slash issues.
	// remote="/abs/path" → "ssh://remote-agent//abs/path"
	if strings.HasPrefix(remote, "/") {
		return "ssh://remote-agent/" + remote
	}
	return "ssh://remote-agent//" + remote
}

// RenderUnisonProfile renders profile body for pair p using sshConfigDir for -F.
func RenderUnisonProfile(sshConfigDir string, p *Pair) string {
	if p == nil {
		return ""
	}
	sshConfig := filepath.Join(sshConfigDir, "ssh_config")
	var b strings.Builder
	fmt.Fprintf(&b, "root = %s\n", p.Local)
	fmt.Fprintf(&b, "root = %s\n", sshRemoteRoot(p.Remote))
	fmt.Fprintf(&b, "sshargs = -F %s\n", sshConfig)
	fmt.Fprintf(&b, "servercmd = %s\n", p.RemoteUnison)
	fmt.Fprintf(&b, "auto = %t\n", p.Auto)
	fmt.Fprintf(&b, "batch = %t\n", p.Batch)
	fmt.Fprintf(&b, "times = %t\n", p.Times)
	fmt.Fprintf(&b, "prefer = %s\n", p.Prefer)
	for _, ign := range p.Ignore {
		fmt.Fprintf(&b, "ignore = %s\n", ign)
	}
	return b.String()
}

// WriteUnisonProfile writes {unisonDir}/remote-agent-<name>.prf and returns its path.
func WriteUnisonProfile(unisonDir, sshConfigDir string, p *Pair) (string, error) {
	if p == nil {
		return "", fmt.Errorf("pair is nil")
	}
	if err := os.MkdirAll(unisonDir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(unisonDir, ProfileFileName(p.Name))
	content := RenderUnisonProfile(sshConfigDir, p)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
