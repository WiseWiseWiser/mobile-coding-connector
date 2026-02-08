package cloudflare

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

const defaultDomainTunnelName = "ai-agent-tunnel"

var (
	domainMu      sync.Mutex
	domainIngress = map[string]IngressRule{} // hostname -> rule
	domainCmd     *exec.Cmd
)

// DomainTunnelStatus describes the runtime status of a domain tunnel.
type DomainTunnelStatus struct {
	Status    string `json:"status"`              // "stopped", "connecting", "active", "error"
	TunnelURL string `json:"tunnel_url,omitempty"` // the public URL (https://<domain>)
	Error     string `json:"error,omitempty"`
}

// CheckStatus returns the cloudflare installation and authentication status.
// This is the exported version of the status check for reuse by other packages.
func CheckStatus() StatusResponse {
	resp := StatusResponse{}

	if !tool_resolve.IsAvailable("cloudflared") {
		resp.Error = "cloudflared is not installed"
		return resp
	}
	resp.Installed = true

	out, err := exec.Command("cloudflared", "tunnel", "list", "--output", "json").CombinedOutput()
	if err != nil {
		errStr := strings.TrimSpace(string(out))
		if strings.Contains(errStr, "login") || strings.Contains(errStr, "auth") || strings.Contains(errStr, "certificate") {
			resp.Error = "Not authenticated. Click Login to authenticate."
		} else {
			resp.Error = fmt.Sprintf("Could not verify authentication: %s", errStr)
		}
		return resp
	}
	resp.Authenticated = true
	resp.CertFiles = ListCertFiles()

	return resp
}

// ParseBaseDomain extracts the base domain from a full hostname.
// e.g. "sub.example.com" -> "example.com", "a.b.example.co.uk" -> "example.co.uk"
// Simple approach: take the last two parts separated by dots.
func ParseBaseDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) <= 2 {
		return domain
	}
	return strings.Join(parts[len(parts)-2:], ".")
}

// GetDomainTunnelStatus returns the current status of a domain tunnel.
func GetDomainTunnelStatus(domain string) DomainTunnelStatus {
	domainMu.Lock()
	defer domainMu.Unlock()

	if _, ok := domainIngress[domain]; !ok {
		return DomainTunnelStatus{Status: "stopped"}
	}

	if domainCmd == nil || domainCmd.Process == nil {
		return DomainTunnelStatus{
			Status:    "error",
			TunnelURL: fmt.Sprintf("https://%s", domain),
			Error:     "tunnel process not running",
		}
	}

	return DomainTunnelStatus{
		Status:    "active",
		TunnelURL: fmt.Sprintf("https://%s", domain),
	}
}

// LogFunc is a callback for streaming log messages during tunnel operations.
type LogFunc func(message string)

// StartDomainTunnel starts a cloudflare named tunnel for the given domain.
// The domain is routed via DNS to the tunnel, and an ingress rule maps it
// to http://localhost:<port>.
// tunnelName is the cloudflare tunnel name to use; if empty, defaults to "ai-agent-tunnel".
// logFn is an optional callback for streaming log messages.
func StartDomainTunnel(domain string, port int, tunnelName string, logFn LogFunc) (*DomainTunnelStatus, error) {
	if tunnelName == "" {
		tunnelName = defaultDomainTunnelName
	}
	if logFn == nil {
		logFn = func(string) {}
	}

	domainMu.Lock()
	defer domainMu.Unlock()

	// Already running for this domain?
	if _, ok := domainIngress[domain]; ok {
		if domainCmd != nil && domainCmd.Process != nil {
			logFn("Tunnel already running for " + domain)
			return &DomainTunnelStatus{
				Status:    "active",
				TunnelURL: fmt.Sprintf("https://%s", domain),
			}, nil
		}
	}

	// Find or create a tunnel
	logFn("Finding or creating tunnel '" + tunnelName + "'...")
	tunnelRef, err := FindOrCreateTunnel(tunnelName)
	if err != nil {
		return nil, fmt.Errorf("failed to find/create tunnel: %v", err)
	}
	logFn("Using tunnel: " + tunnelRef)

	// Create DNS route
	logFn("Creating DNS route for " + domain + "...")
	if err := CreateDNSRoute(tunnelRef, domain); err != nil {
		return nil, err
	}
	logFn("DNS route created")

	// Add ingress rule
	localURL := fmt.Sprintf("http://localhost:%d", port)
	domainIngress[domain] = IngressRule{Hostname: domain, Service: localURL}
	logFn(fmt.Sprintf("Routing %s -> %s", domain, localURL))

	// Write config and start/restart tunnel
	logFn("Starting cloudflared tunnel process...")
	if err := writeDomainConfigAndRestart(tunnelRef); err != nil {
		delete(domainIngress, domain)
		return nil, fmt.Errorf("failed to start tunnel: %v", err)
	}
	logFn("Tunnel started successfully")

	return &DomainTunnelStatus{
		Status:    "active",
		TunnelURL: fmt.Sprintf("https://%s", domain),
	}, nil
}

// StopDomainTunnel stops the tunnel for the given domain.
// tunnelName is the cloudflare tunnel name; if empty, defaults to "ai-agent-tunnel".
func StopDomainTunnel(domain string, tunnelName string) error {
	if tunnelName == "" {
		tunnelName = defaultDomainTunnelName
	}

	domainMu.Lock()
	defer domainMu.Unlock()

	if _, ok := domainIngress[domain]; !ok {
		return fmt.Errorf("no running tunnel for domain %q", domain)
	}

	delete(domainIngress, domain)

	if len(domainIngress) == 0 {
		// No more domains — stop the tunnel process
		killDomainProcess()
	} else {
		// Restart with remaining ingress rules
		tunnelRef, err := FindOrCreateTunnel(tunnelName)
		if err != nil {
			return err
		}
		if err := writeDomainConfigAndRestart(tunnelRef); err != nil {
			return err
		}
	}

	return nil
}

// killDomainProcess kills the current domain tunnel process if running.
// Must be called with domainMu held.
func killDomainProcess() {
	if domainCmd == nil || domainCmd.Process == nil {
		return
	}
	domainCmd.Process.Kill()
	domainCmd.Wait()
	domainCmd = nil
}

// killExistingCloudflaredForConfig kills any running cloudflared process
// using the domain tunnels config file. This handles the case where the
// server restarted but the detached cloudflared process is still running.
func killExistingCloudflaredForConfig(cfgPath string) {
	// Use pgrep to find cloudflared processes that reference our config
	out, err := exec.Command("pgrep", "-f", "cloudflared.*"+cfgPath).Output()
	if err != nil {
		return // no matching process
	}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var pid int
		if _, err := fmt.Sscanf(line, "%d", &pid); err == nil && pid > 0 {
			if p, err := os.FindProcess(pid); err == nil {
				p.Kill()
			}
		}
	}
	time.Sleep(500 * time.Millisecond)
}

// writeDomainConfigAndRestart writes the domain tunnel config and (re)starts the tunnel process.
// Must be called with domainMu held.
func writeDomainConfigAndRestart(tunnelRef string) error {
	tunnelID, credFile, err := EnsureTunnelExists(tunnelRef)
	if err != nil {
		return err
	}

	cfgDir, err := DefaultConfigDir()
	if err != nil {
		return err
	}
	cfgPath := cfgDir + "/config-domain-tunnels.yml"

	// Kill existing process (from this server session)
	killDomainProcess()
	// Also kill any orphaned cloudflared process from a previous server session
	killExistingCloudflaredForConfig(cfgPath)

	// Build ingress rules
	var rules []IngressRule
	for _, rule := range domainIngress {
		rules = append(rules, rule)
	}
	rules = append(rules, IngressRule{Service: "http_status:404"})

	// Write config
	cfg := &CloudflaredConfig{
		Tunnel:          tunnelID,
		CredentialsFile: credFile,
		Ingress:         rules,
	}
	if err := WriteCloudflaredConfig(cfgPath, cfg); err != nil {
		return err
	}

	// Start tunnel in its own process group so it survives server restart
	cmd := exec.Command("cloudflared", "tunnel", "--config", cfgPath, "run", tunnelRef)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cloudflared: %v", err)
	}

	domainCmd = cmd

	go func() {
		cmd.Wait()
	}()

	return nil
}
