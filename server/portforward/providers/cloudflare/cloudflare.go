package cloudflare

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	cfutils "github.com/xhd2015/lifelog-private/ai-critic/server/cloudflare"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/domains/pick"
	"github.com/xhd2015/lifelog-private/ai-critic/server/portforward"
)

// --- Quick Tunnel Provider ---

// QuickProvider implements portforward.Provider using cloudflared quick tunnels (trycloudflare.com)
type QuickProvider struct{}

var _ portforward.Provider = (*QuickProvider)(nil)

func (p *QuickProvider) Name() string        { return portforward.ProviderCloudflareQuick }
func (p *QuickProvider) DisplayName() string { return "Cloudflare Quick Tunnel" }
func (p *QuickProvider) Description() string {
	return "Free tunneling via trycloudflare.com (cloudflared). No account required."
}
func (p *QuickProvider) Available() bool { return portforward.IsCommandAvailable("cloudflared") }

func (p *QuickProvider) Start(port int, _ string) (*portforward.TunnelHandle, error) {
	logs := portforward.NewLogBuffer()

	cmd := exec.Command("cloudflared", "tunnel", "--url", fmt.Sprintf("http://localhost:%d", port))

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipe: %v", err)
	}
	cmd.Stdout = logs

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start cloudflared: %v", err)
	}

	resultCh := make(chan portforward.TunnelResult, 1)
	urlRegex := regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

	go func() {
		scanner := bufio.NewScanner(stderr)
		urlFound := make(chan string, 1)

		go func() {
			for scanner.Scan() {
				line := scanner.Text()
				logs.Write([]byte(line + "\n"))
				if match := urlRegex.FindString(line); match != "" {
					urlFound <- match
					return
				}
			}
		}()

		select {
		case url := <-urlFound:
			resultCh <- portforward.TunnelResult{PublicURL: url}
		case <-time.After(60 * time.Second):
			resultCh <- portforward.TunnelResult{Err: fmt.Errorf("timeout waiting for cloudflared tunnel URL (60s)")}
			cmd.Process.Kill()
			return
		}

		cmd.Wait()
	}()

	return &portforward.TunnelHandle{
		Result: resultCh,
		Logs:   logs,
		Stop: func() {
			if cmd.Process != nil {
				cmd.Process.Kill()
			}
		},
	}, nil
}

// --- Named Tunnel Provider ---

// TunnelProvider implements portforward.Provider using a DEDICATED named
// Cloudflare tunnel (separate from the main app tunnel).
//
// It uses its own config file (config-portforward.yml) and its own tunnel name,
// so it never interferes with the main tunnel process. The dedicated tunnel
// process is started on first port-forward and restarted when ingress rules change.
type TunnelProvider struct {
	cfg config.CloudflareTunnelConfig
	mu  sync.Mutex

	// The dedicated port-forwarding tunnel process
	pfCmd *exec.Cmd
	// Track port-forwarding ingress rules (hostname -> rule)
	pfIngress map[string]cfutils.IngressRule
}

var _ portforward.Provider = (*TunnelProvider)(nil)

func NewTunnelProvider(cfg config.CloudflareTunnelConfig) *TunnelProvider {
	return &TunnelProvider{
		cfg:       cfg,
		pfIngress: make(map[string]cfutils.IngressRule),
	}
}

func (p *TunnelProvider) Name() string { return portforward.ProviderCloudflareTunnel }
func (p *TunnelProvider) DisplayName() string {
	if p.cfg.BaseDomain != "" {
		return fmt.Sprintf("Cloudflare Tunnel (*.%s)", p.cfg.BaseDomain)
	}
	return "Cloudflare Named Tunnel"
}
func (p *TunnelProvider) Description() string {
	if p.cfg.BaseDomain != "" {
		return fmt.Sprintf("Dedicated Cloudflare tunnel (%s). Generates random-words.%s subdomains.", p.cfg.TunnelName, p.cfg.BaseDomain)
	}
	return "Dedicated Cloudflare tunnel for port forwarding."
}
func (p *TunnelProvider) Available() bool {
	if !portforward.IsCommandAvailable("cloudflared") {
		return false
	}
	return p.cfg.BaseDomain != "" && (p.cfg.TunnelName != "" || p.cfg.TunnelID != "")
}

// getUserDomains returns the list of user-configured owned domains from cloudflare config.
func getUserDomains() []string {
	return cfutils.GetOwnedDomains()
}

// isCloudflareAuthenticated checks if cloudflared is authenticated
func isCloudflareAuthenticated() bool {
	status := cfutils.CheckStatus()
	return status.Authenticated
}

func (p *TunnelProvider) Start(port int, hostname string) (*portforward.TunnelHandle, error) {
	logs := portforward.NewLogBuffer()

	if p.cfg.BaseDomain == "" {
		return nil, fmt.Errorf("cloudflare tunnel base_domain is not configured")
	}

	tunnelRef := p.cfg.TunnelName
	if tunnelRef == "" {
		tunnelRef = p.cfg.TunnelID
	}

	// Use provided hostname if valid, otherwise fall back to default logic
	if hostname == "" || !strings.Contains(hostname, ".") {
		// Try to use user-configured domain if Cloudflare is authenticated
		userDomains := getUserDomains()
		if len(userDomains) > 0 && isCloudflareAuthenticated() {
			// Use the first available user domain
			hostname = userDomains[0]
			fmt.Fprintf(logs, "[setup] Using user-configured domain: %s\n", hostname)
		} else {
			// Fall back to random subdomain
			hostname = fmt.Sprintf("%s.%s", pick.RandomSubdomain(), p.cfg.BaseDomain)
			fmt.Fprintf(logs, "[setup] Using generated subdomain: %s\n", hostname)
		}
	} else {
		fmt.Fprintf(logs, "[setup] Using provided hostname: %s\n", hostname)
	}

	localURL := fmt.Sprintf("http://localhost:%d", port)

	configDir := p.cfg.ConfigPath
	if configDir == "" {
		var err error
		configDir, err = cfutils.DefaultConfigDir()
		if err != nil {
			return nil, err
		}
	}
	pfConfigFile := configDir + "/config-portforward.yml"

	// Step 1: Create DNS route
	fmt.Fprintf(logs, "[setup] Creating DNS route: %s -> tunnel %s\n", hostname, tunnelRef)
	if err := cfutils.CreateDNSRoute(tunnelRef, hostname); err != nil {
		fmt.Fprintf(logs, "[setup] Warning: DNS route error: %v\n", err)
	}

	// Step 2: Add ingress rule and restart the dedicated tunnel
	p.mu.Lock()
	p.pfIngress[hostname] = cfutils.IngressRule{Hostname: hostname, Service: localURL}
	fmt.Fprintf(logs, "[setup] Adding ingress rule: %s -> %s\n", hostname, localURL)

	if err := p.writeConfigAndRestart(pfConfigFile, tunnelRef, logs); err != nil {
		delete(p.pfIngress, hostname)
		p.mu.Unlock()
		return nil, fmt.Errorf("failed to start tunnel: %v", err)
	}
	p.mu.Unlock()

	publicURL := fmt.Sprintf("https://%s", hostname)

	resultCh := make(chan portforward.TunnelResult, 1)
	go func() {
		// Give the tunnel a few seconds to register
		time.Sleep(5 * time.Second)
		resultCh <- portforward.TunnelResult{PublicURL: publicURL}
	}()

	return &portforward.TunnelHandle{
		Result: resultCh,
		Logs:   logs,
		Stop: func() {
			p.mu.Lock()
			delete(p.pfIngress, hostname)
			fmt.Fprintf(logs, "[cleanup] Removed ingress rule for %s\n", hostname)

			if len(p.pfIngress) == 0 {
				if p.pfCmd != nil && p.pfCmd.Process != nil {
					fmt.Fprintf(logs, "[cleanup] Stopping dedicated tunnel (no more rules)\n")
					p.pfCmd.Process.Kill()
					p.pfCmd.Wait()
					p.pfCmd = nil
				}
			} else {
				p.writeConfigAndRestart(pfConfigFile, tunnelRef, logs)
			}
			p.mu.Unlock()
		},
	}, nil
}

// writeConfigAndRestart writes config-portforward.yml with current ingress rules
// and (re)starts the dedicated tunnel process. Must be called with p.mu held.
func (p *TunnelProvider) writeConfigAndRestart(pfConfigFile, tunnelRef string, logs *portforward.LogBuffer) error {
	tunnelID, credentialsFile, err := p.resolveTunnelCreds(tunnelRef, logs)
	if err != nil {
		return err
	}

	// Kill existing process if running
	if p.pfCmd != nil && p.pfCmd.Process != nil {
		fmt.Fprintf(logs, "[tunnel] Stopping existing tunnel (PID %d)...\n", p.pfCmd.Process.Pid)
		p.pfCmd.Process.Kill()
		p.pfCmd.Wait()
		p.pfCmd = nil
		time.Sleep(500 * time.Millisecond)
	}

	// Build ingress rules
	var rules []cfutils.IngressRule
	for _, rule := range p.pfIngress {
		rules = append(rules, rule)
	}
	rules = append(rules, cfutils.IngressRule{Service: "http_status:404"})

	// Write config
	cfg := &cfutils.CloudflaredConfig{
		Tunnel:          tunnelID,
		CredentialsFile: credentialsFile,
		Ingress:         rules,
	}
	if err := cfutils.WriteCloudflaredConfig(pfConfigFile, cfg); err != nil {
		return err
	}
	fmt.Fprintf(logs, "[tunnel] Wrote %s (%d ingress rules)\n", pfConfigFile, len(rules)-1)

	// Start dedicated tunnel process
	cmd := exec.Command("cloudflared", "tunnel", "--config", pfConfigFile, "run", tunnelRef)
	cmd.Stdout = logs
	cmd.Stderr = logs

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start cloudflared: %v", err)
	}

	p.pfCmd = cmd
	fmt.Fprintf(logs, "[tunnel] Started dedicated tunnel (PID %d)\n", cmd.Process.Pid)

	go func() {
		cmd.Wait()
		fmt.Fprintf(logs, "[tunnel] Dedicated tunnel process exited\n")
	}()

	return nil
}

// resolveTunnelCreds resolves tunnel ID and credentials file.
// Uses config values if available, otherwise falls back to EnsureTunnelExists.
func (p *TunnelProvider) resolveTunnelCreds(tunnelRef string, logs *portforward.LogBuffer) (string, string, error) {
	tunnelID := p.cfg.TunnelID
	credFile := p.cfg.CredentialsFile

	// If we have both from config and the file exists, use them directly
	if tunnelID != "" && credFile != "" {
		if _, err := os.Stat(credFile); err == nil {
			return tunnelID, credFile, nil
		}
	}

	// Fall back to auto-discovery/creation
	fmt.Fprintf(logs, "[setup] Resolving tunnel credentials for '%s'...\n", tunnelRef)
	id, cred, err := cfutils.EnsureTunnelExists(tunnelRef)
	if err != nil {
		return "", "", err
	}

	fmt.Fprintf(logs, "[setup] Resolved tunnel: id=%s, credentials=%s\n", id, cred)
	return id, cred, nil
}

// --- Owned Domain Provider ---

// OwnedProvider implements portforward.Provider using cloudflared with user-owned domains.
// It creates random subdomains under the user's configured owned domains without requiring
// a named tunnel configuration in .config.local.json.
type OwnedProvider struct{}

var _ portforward.Provider = (*OwnedProvider)(nil)

func (p *OwnedProvider) Name() string        { return portforward.ProviderCloudflareOwned }
func (p *OwnedProvider) DisplayName() string { return "Cloudflare (My Domain)" }
func (p *OwnedProvider) Description() string {
	return "Uses your configured domain to generate random subdomains. Requires cloudflared authentication."
}

func (p *OwnedProvider) Available() bool {
	if !portforward.IsCommandAvailable("cloudflared") {
		return false
	}
	// Check if user has owned domains and is authenticated
	userDomains := getUserDomains()
	if len(userDomains) == 0 {
		return false
	}
	return isCloudflareAuthenticated()
}

func (p *OwnedProvider) Start(port int, hostname string) (*portforward.TunnelHandle, error) {
	logs := portforward.NewLogBuffer()

	fmt.Fprintf(logs, "[OwnedProvider.Start] Received hostname: %q (contains dot: %v)\n", hostname, strings.Contains(hostname, "."))

	// Get user domains
	userDomains := getUserDomains()
	if len(userDomains) == 0 {
		return nil, fmt.Errorf("no owned domains configured")
	}

	// Use provided hostname if valid (contains a dot indicating it's a full domain), otherwise generate one
	if hostname == "" || !strings.Contains(hostname, ".") {
		// Use the first owned domain
		baseDomain := userDomains[0]
		fmt.Fprintf(logs, "[setup] No valid hostname provided, using owned domain: %s\n", baseDomain)

		// If hostname is empty, generate random subdomain, otherwise use provided hostname as subdomain
		var subdomain string
		if hostname == "" {
			subdomain = pick.RandomSubdomain()
			fmt.Fprintf(logs, "[setup] Generated random subdomain: %s\n", subdomain)
		} else {
			subdomain = hostname
			fmt.Fprintf(logs, "[setup] Using provided value as subdomain: %s\n", subdomain)
		}
		hostname = fmt.Sprintf("%s.%s", subdomain, baseDomain)
		fmt.Fprintf(logs, "[setup] Final generated hostname: %s\n", hostname)
	} else {
		fmt.Fprintf(logs, "[setup] Using provided hostname as-is: %s\n", hostname)
	}

	// Determine tunnel name based on full hostname to ensure uniqueness
	// Each unique subdomain gets its own tunnel to avoid collisions
	tunnelName := cfutils.DefaultTunnelName(hostname)
	fmt.Fprintf(logs, "[setup] Using tunnel: %s\n", tunnelName)

	// Find or create tunnel
	tunnelRef, err := cfutils.FindOrCreateTunnel(tunnelName)
	if err != nil {
		return nil, fmt.Errorf("failed to find/create tunnel: %v", err)
	}
	fmt.Fprintf(logs, "[setup] Tunnel resolved: %s\n", tunnelRef)

	// Get tunnel ID and credentials
	tunnelID, credFile, err := cfutils.EnsureTunnelExists(tunnelRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get tunnel credentials: %v", err)
	}
	fmt.Fprintf(logs, "[setup] Got tunnel credentials: id=%s\n", tunnelID)

	// Create DNS route
	fmt.Fprintf(logs, "[setup] Creating DNS route: %s -> tunnel %s\n", hostname, tunnelRef)
	if err := cfutils.CreateDNSRoute(tunnelRef, hostname); err != nil {
		fmt.Fprintf(logs, "[setup] Warning: DNS route error: %v\n", err)
	}

	// Create a standalone config file for this port mapping
	configDir, err := cfutils.DefaultConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %v", err)
	}

	// Use a unique config file per port to ensure isolation
	// Sanitize hostname for filename (remove dots and special chars)
	safeHostname := strings.ReplaceAll(hostname, ".", "-")
	configFile := filepath.Join(configDir, fmt.Sprintf("config-port-%d-%s.yml", port, safeHostname))

	// Build ingress rules - only this specific hostname
	localURL := fmt.Sprintf("http://localhost:%d", port)
	ingressRules := []cfutils.IngressRule{
		{Hostname: hostname, Service: localURL},
		{Service: "http_status:404"}, // catch-all
	}

	cfg := &cfutils.CloudflaredConfig{
		Tunnel:          tunnelID,
		CredentialsFile: credFile,
		Ingress:         ingressRules,
	}

	if err := cfutils.WriteCloudflaredConfig(configFile, cfg); err != nil {
		return nil, fmt.Errorf("failed to write cloudflared config: %v", err)
	}
	fmt.Fprintf(logs, "[setup] Created standalone config: %s\n", configFile)

	// Start cloudflared with this specific config
	resultCh := make(chan portforward.TunnelResult, 1)
	publicURL := fmt.Sprintf("https://%s", hostname)

	cmd := exec.Command("cloudflared", "tunnel", "--config", configFile, "run", tunnelRef)
	cmd.Stdout = logs
	cmd.Stderr = logs

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start cloudflared: %v", err)
	}
	fmt.Fprintf(logs, "[tunnel] Started cloudflared (PID %d) for port %d\n", cmd.Process.Pid, port)

	go func() {
		// Give the tunnel a few seconds to establish
		time.Sleep(5 * time.Second)
		resultCh <- portforward.TunnelResult{PublicURL: publicURL}

		// Wait for the process to complete
		if err := cmd.Wait(); err != nil {
			fmt.Fprintf(logs, "[tunnel] Cloudflared exited: %v\n", err)
		}
	}()

	return &portforward.TunnelHandle{
		Result: resultCh,
		Logs:   logs,
		Stop: func() {
			fmt.Fprintf(logs, "[cleanup] Stopping cloudflared process...\n")
			if cmd.Process != nil {
				cmd.Process.Kill()
				cmd.Wait()
			}
			// Clean up the config file
			if err := os.Remove(configFile); err == nil {
				fmt.Fprintf(logs, "[cleanup] Removed config file: %s\n", configFile)
			}
		},
	}, nil
}
