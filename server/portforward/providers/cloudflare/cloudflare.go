package cloudflare

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	cfutils "github.com/xhd2015/lifelog-private/ai-critic/server/cloudflare"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/domains"
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

func (p *QuickProvider) Start(port int) (*portforward.TunnelHandle, error) {
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

// getUserDomains returns the list of user-configured domains from domains.json
func getUserDomains() []string {
	cfg, err := domains.LoadDomains()
	if err != nil {
		return nil
	}

	var userDomains []string
	for _, d := range cfg.Domains {
		// Filter out temporary domains
		if !strings.Contains(d.Domain, "trycloudflare.com") && !strings.Contains(d.Domain, "loca.lt") {
			userDomains = append(userDomains, d.Domain)
		}
	}
	return userDomains
}

// isCloudflareAuthenticated checks if cloudflared is authenticated
func isCloudflareAuthenticated() bool {
	status := cfutils.CheckStatus()
	return status.Authenticated
}

func (p *TunnelProvider) Start(port int) (*portforward.TunnelHandle, error) {
	logs := portforward.NewLogBuffer()

	if p.cfg.BaseDomain == "" {
		return nil, fmt.Errorf("cloudflare tunnel base_domain is not configured")
	}

	tunnelRef := p.cfg.TunnelName
	if tunnelRef == "" {
		tunnelRef = p.cfg.TunnelID
	}

	// Try to use user-configured domain if Cloudflare is authenticated
	hostname := ""
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
