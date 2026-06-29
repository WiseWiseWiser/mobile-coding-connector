package wsproxy_singbox

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	BrewInstallXrayCmd    = "brew install xray"
	xraySidecarReadyWait  = 15 * time.Second
	xraySidecarVerifyWait = 20 * time.Second
)

// verifyLocalProxyAfterTun is set by run-tun when an xray sidecar is active.
var verifyLocalProxyAfterTun func() error

// XraySidecar runs a local xray SOCKS inbound that dials the ws-proxy VMess endpoint.
type XraySidecar struct {
	Port       int
	ConfigPath string
	cmd        *exec.Cmd
}

func findXrayBinary() (string, error) {
	if p, err := currentHooks.LookPath("xray"); err == nil {
		return p, nil
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("xray not found in PATH")
	}
	cached := filepath.Join(cacheDir, "remote-agent", "xray", "xray")
	if _, err := os.Stat(cached); err == nil {
		return cached, nil
	}
	return "", fmt.Errorf("xray not found in PATH; install with: %s", BrewInstallXrayCmd)
}

func pickFreeLocalTCPPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port, nil
}

func xrayVMessOutboundJSON(vmess *VMessParams) (port, network, security, path string) {
	port = vmess.Port
	if port == "" {
		port = "443"
	}
	network = vmess.Network
	if network == "" {
		network = "ws"
	}
	security = "tls"
	if strings.EqualFold(vmess.TLS, "") || strings.EqualFold(vmess.TLS, "none") {
		security = "none"
	}
	path = vmess.Path
	if path == "" {
		path = "/ws"
	}
	return port, network, security, path
}

// BuildXrayVMessClientConfig renders the xray HTTP-inbound config used by doctor.
func BuildXrayVMessClientConfig(vmess *VMessParams, inboundPort int) string {
	p, network, security, path := xrayVMessOutboundJSON(vmess)
	return fmt.Sprintf(`{
  "inbounds": [{
    "listen": "127.0.0.1",
    "port": %d,
    "protocol": "http",
    "settings": {}
  }],
  "outbounds": [{
    "protocol": "vmess",
    "settings": {
      "vnext": [{
        "address": %q,
        "port": %s,
        "users": [{"id": %q, "alterId": 0, "security": "auto"}]
      }]
    },
    "streamSettings": {
      "network": %q,
      "security": %q,
      "wsSettings": {
        "path": %q,
        "headers": {"Host": %q}
      },
      "tlsSettings": {"serverName": %q}
    }
  }]
}`, inboundPort, vmess.Host, p, vmess.UUID, network, security, path, vmess.Host, vmess.Host)
}

func buildXraySidecarConfig(vmess *VMessParams, socksPort int) string {
	p, network, security, path := xrayVMessOutboundJSON(vmess)
	return fmt.Sprintf(`{
  "dns": {
    "servers": ["8.8.8.8", "1.1.1.1"],
    "queryStrategy": "UseIPv4",
    "disableFallback": true
  },
  "inbounds": [{
    "listen": "127.0.0.1",
    "port": %d,
    "protocol": "socks",
    "settings": {"udp": true, "auth": "noauth"}
  }],
  "outbounds": [{
    "tag": "proxy",
    "protocol": "vmess",
    "settings": {
      "vnext": [{
        "address": %q,
        "port": %s,
        "users": [{"id": %q, "alterId": 0, "security": "auto"}]
      }]
    },
    "streamSettings": {
      "network": %q,
      "security": %q,
      "wsSettings": {
        "path": %q,
        "headers": {"Host": %q}
      },
      "tlsSettings": {"serverName": %q}
    }
  }],
  "routing": {
    "domainStrategy": "AsIs",
    "rules": [{
      "type": "field",
      "network": "udp",
      "port": "53",
      "outboundTag": "proxy"
    }]
  }
}`, socksPort, vmess.Host, p, vmess.UUID, network, security, path, vmess.Host, vmess.Host)
}

func StartXraySidecar(ctx context.Context, vmess *VMessParams) (*XraySidecar, error) {
	if currentHooks.StartXraySidecar != nil {
		return currentHooks.StartXraySidecar(ctx, vmess)
	}
	return startXraySidecarReal(ctx, vmess)
}

func startXraySidecarReal(ctx context.Context, vmess *VMessParams) (*XraySidecar, error) {
	xrayPath, err := findXrayBinary()
	if err != nil {
		return nil, err
	}
	port, err := pickFreeLocalTCPPort()
	if err != nil {
		return nil, fmt.Errorf("pick local port: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "xray-sidecar-*.json")
	if err != nil {
		return nil, fmt.Errorf("create xray config: %w", err)
	}
	configPath := tmpFile.Name()
	if _, err := tmpFile.WriteString(buildXraySidecarConfig(vmess, port)); err != nil {
		tmpFile.Close()
		os.Remove(configPath)
		return nil, fmt.Errorf("write xray config: %w", err)
	}
	tmpFile.Close()

	logPath := xraySidecarLogPath()
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		os.Remove(configPath)
		return nil, fmt.Errorf("open xray log: %w", err)
	}

	cmd := exec.CommandContext(ctx, xrayPath, "run", "-c", configPath)
	cmd.Env = singBoxProcessEnv()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		logFile.Close()
		os.Remove(configPath)
		return nil, fmt.Errorf("start xray: %w", err)
	}
	_ = logFile.Close()

	if err := waitForLocalTCPPort(ctx, port, xraySidecarReadyWait); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		os.Remove(configPath)
		return nil, fmt.Errorf("xray sidecar on 127.0.0.1:%d: %w", port, err)
	}
	if err := verifyXraySOCKSProxy(ctx, port, xraySidecarVerifyWait); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		os.Remove(configPath)
		return nil, fmt.Errorf("xray VMess path: %w (see %s)", err, logPath)
	}

	return &XraySidecar{
		Port:       port,
		ConfigPath: configPath,
		cmd:        cmd,
	}, nil
}

func verifyXraySOCKSProxy(ctx context.Context, port int, timeout time.Duration) error {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	args := []string{
		"-4", "-m", "15", "-sS", "-o", "/dev/null", "-w", "%{http_code}",
		"--socks5-hostname", fmt.Sprintf("127.0.0.1:%d", port),
		"https://www.google.com/generate_204",
	}
	cmd := exec.CommandContext(ctx, "curl", args...)
	cmd.Env = singBoxProcessEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	code := strings.TrimSpace(string(out))
	if code != "204" && code != "200" {
		return fmt.Errorf("generate_204 via xray SOCKS → HTTP %s", code)
	}
	return nil
}

func verifyXrayHTTPProxy(ctx context.Context, port int, timeout time.Duration) error {
	return verifyXraySOCKSProxy(ctx, port, timeout)
}

func waitForLocalTCPPort(ctx context.Context, port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		conn, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return fmt.Errorf("timed out waiting for %s", addr)
}

func (s *XraySidecar) Stop() {
	if s == nil {
		return
	}
	if s.cmd != nil && s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
		_ = s.cmd.Wait()
	}
	if s.ConfigPath != "" {
		_ = os.Remove(s.ConfigPath)
	}
}