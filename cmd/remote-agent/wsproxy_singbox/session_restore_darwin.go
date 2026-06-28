//go:build darwin

package wsproxy_singbox

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
)

type tunSessionSnapshot struct {
	NetworkService string             `json:"network_service"`
	DNSServers     []string           `json:"dns_servers,omitempty"`
	DNSTouched     bool               `json:"dns_touched"`
	Proxy          serviceProxyState  `json:"proxy"`
	ProxyTouched   bool               `json:"proxy_touched"`
}

func tunSessionStatePath() (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cacheDir, "remote-agent")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(dir, "tun-session.json"), nil
}

func saveTunSessionSnapshot(service string, previousDNS []string, dnsTouched bool, previousProxy serviceProxyState, proxyTouched bool) error {
	if !dnsTouched && !proxyTouched {
		return nil
	}
	path, err := tunSessionStatePath()
	if err != nil {
		return err
	}
	data, err := json.Marshal(tunSessionSnapshot{
		NetworkService: service,
		DNSServers:     previousDNS,
		DNSTouched:     dnsTouched,
		Proxy:          previousProxy,
		ProxyTouched:   proxyTouched,
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return err
	}
	return chownToSudoUser(path)
}

func chownToSudoUser(path string) error {
	if os.Geteuid() != 0 {
		return nil
	}
	username := os.Getenv("SUDO_USER")
	if username == "" || username == "root" {
		return nil
	}
	u, err := user.Lookup(username)
	if err != nil {
		return nil
	}
	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return nil
	}
	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return nil
	}
	return os.Chown(path, uid, gid)
}

func clearTunSessionSnapshot() {
	path, err := tunSessionStatePath()
	if err != nil {
		return
	}
	_ = os.Remove(path)
}

// RestoreTunSessionSideEffects restores macOS DNS/proxy state left by an
// interrupted ws-proxy vpn session (SIGKILL, smoke-test timeout, etc.).
func RestoreTunSessionSideEffects() error {
	_ = restoreStuckTunDNS()

	path, err := tunSessionStatePath()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var snap tunSessionSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("parse tun session snapshot: %w", err)
	}
	service := snap.NetworkService
	if service == "" {
		_ = os.Remove(path)
		return nil
	}
	if snap.DNSTouched {
		if err := restoreDNSServers(service, snap.DNSServers); err != nil {
			return fmt.Errorf("restore DNS for %q: %w", service, err)
		}
		fmt.Printf("Restored system DNS for %q\n", service)
	}
	if snap.ProxyTouched {
		if err := setServiceProxyState(service, snap.Proxy); err != nil {
			return fmt.Errorf("restore proxy for %q: %w", service, err)
		}
		fmt.Printf("Restored system proxy for %q\n", service)
	}
	_ = os.Remove(path)
	return nil
}