package cloudflare

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"time"

	cfutils "github.com/xhd2015/lifelog-private/ai-critic/server/cloudflare"
	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/domains/pick"
	"github.com/xhd2015/lifelog-private/ai-critic/server/proxy/portforward"
)

// --- Quick Tunnel Provider ---

// QuickProvider implements portforward.Provider using cloudflared quick tunnels (trycloudflare.com)
// This provider is kept separate as it uses a different mechanism (trycloudflare.com) than
// the unified tunnel manager.
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

// --- Unified Tunnel Provider ---

// TunnelProvider implements portforward.Provider using the unified Cloudflare tunnel manager.
// All port forwards using this provider share a single cloudflared process with a single config file.
type TunnelProvider struct {
	cfg config.CloudflareTunnelConfig
}

var _ portforward.Provider = (*TunnelProvider)(nil)

func NewTunnelProvider(cfg config.CloudflareTunnelConfig) *TunnelProvider {
	// Configure the extension tunnel group
	tg := cfutils.GetTunnelGroupManager().GetExtensionGroup()
	tg.SetConfig(cfg)

	return &TunnelProvider{cfg: cfg}
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
		return fmt.Sprintf("Uses extension tunnel group with tunnel '%s'. Generates random-words.%s subdomains.", p.cfg.TunnelName, p.cfg.BaseDomain)
	}
	return "Uses extension tunnel group for all port forwards."
}
func (p *TunnelProvider) Available() bool {
	if !portforward.IsCommandAvailable("cloudflared") {
		return false
	}
	return p.cfg.BaseDomain != "" && (p.cfg.TunnelName != "" || p.cfg.TunnelID != "")
}

func (p *TunnelProvider) Start(port int, hostname string) (*portforward.TunnelHandle, error) {
	logs := portforward.NewLogBuffer()

	if p.cfg.BaseDomain == "" {
		return nil, fmt.Errorf("cloudflare tunnel base_domain is not configured")
	}

	// Use provided hostname if valid, otherwise fall back to default logic
	if hostname == "" {
		// Try to use user-configured domain if Cloudflare is authenticated
		userDomains := cfutils.GetOwnedDomains()
		if len(userDomains) > 0 && cfutils.CheckStatus().Authenticated {
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
	tunnelRef := p.cfg.TunnelName
	if tunnelRef == "" {
		tunnelRef = p.cfg.TunnelID
	}

	// Create DNS route
	fmt.Fprintf(logs, "[setup] Creating DNS route: %s -> tunnel %s\n", hostname, tunnelRef)
	if err := cfutils.CreateDNSRoute(tunnelRef, hostname); err != nil {
		fmt.Fprintf(logs, "[setup] Warning: DNS route error: %v\n", err)
	}

	// Add mapping to unified tunnel manager
	mappingID := fmt.Sprintf("port-%d", port)
	mapping := &cfutils.IngressMapping{
		ID:       mappingID,
		Hostname: hostname,
		Service:  localURL,
		Source:   fmt.Sprintf("portforward:%d", port),
	}

	fmt.Fprintf(logs, "[setup] Adding ingress rule: %s -> %s\n", hostname, localURL)
	tg := cfutils.GetTunnelGroupManager().GetExtensionGroup()
	if err := tg.AddMapping(mapping); err != nil {
		return nil, fmt.Errorf("failed to add mapping to extension tunnel: %v", err)
	}

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
			fmt.Fprintf(logs, "[cleanup] Removing ingress rule for port %d\n", port)
			if err := tg.RemoveMapping(mappingID); err != nil {
				fmt.Fprintf(logs, "[cleanup] Warning: failed to remove mapping: %v\n", err)
			}
		},
	}, nil
}

// --- Owned Domain Provider ---

// OwnedProvider implements portforward.Provider using cloudflared with user-owned domains.
// It uses the unified tunnel manager to share a single tunnel process.
type OwnedProvider struct{}

var _ portforward.Provider = (*OwnedProvider)(nil)

func (p *OwnedProvider) Name() string        { return portforward.ProviderCloudflareOwned }
func (p *OwnedProvider) DisplayName() string { return "Cloudflare (My Domain)" }
func (p *OwnedProvider) Description() string {
	return "Uses your configured domain to generate subdomains via the unified tunnel manager. Requires cloudflared authentication."
}

func (p *OwnedProvider) Available() bool {
	if !portforward.IsCommandAvailable("cloudflared") {
		return false
	}
	// Check if user has owned domains and is authenticated
	userDomains := cfutils.GetOwnedDomains()
	if len(userDomains) == 0 {
		return false
	}
	return cfutils.CheckStatus().Authenticated
}

func (p *OwnedProvider) Start(port int, hostname string) (*portforward.TunnelHandle, error) {
	logs := portforward.NewLogBuffer()

	fmt.Fprintf(logs, "[OwnedProvider.Start] Received hostname: %q\n", hostname)

	// Get user domains
	userDomains := cfutils.GetOwnedDomains()
	if len(userDomains) == 0 {
		return nil, fmt.Errorf("no owned domains configured")
	}

	// Use provided hostname if valid (contains a dot indicating it's a full domain), otherwise generate one
	if hostname == "" {
		// Use the first owned domain
		baseDomain := userDomains[0]
		fmt.Fprintf(logs, "[setup] No valid hostname provided, using owned domain: %s\n", baseDomain)

		// Generate random subdomain
		subdomain := pick.RandomSubdomain()
		fmt.Fprintf(logs, "[setup] Generated random subdomain: %s\n", subdomain)
		hostname = fmt.Sprintf("%s.%s", subdomain, baseDomain)
		fmt.Fprintf(logs, "[setup] Final generated hostname: %s\n", hostname)
	} else {
		fmt.Fprintf(logs, "[setup] Using provided hostname as-is: %s\n", hostname)
	}

	// Ensure extension tunnel group has a tunnel configured (reuses existing if already set up)
	tg := cfutils.GetTunnelGroupManager().GetExtensionGroup()
	logWrapper := func(msg string) {
		fmt.Fprintf(logs, "[setup] %s\n", msg)
	}

	tunnelRef, _, _, err := cfutils.EnsureGroupTunnelConfigured(cfutils.GroupExtension, "", logWrapper)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure extension tunnel configured: %v", err)
	}
	fmt.Fprintf(logs, "[setup] Using extension tunnel: %s\n", tunnelRef)

	// Create DNS route using the extension tunnel
	fmt.Fprintf(logs, "[setup] Creating DNS route: %s -> tunnel %s\n", hostname, tunnelRef)
	if err := cfutils.CreateDNSRoute(tunnelRef, hostname); err != nil {
		fmt.Fprintf(logs, "[setup] Warning: DNS route error: %v\n", err)
	}

	// Add mapping to extension tunnel group
	localURL := fmt.Sprintf("http://localhost:%d", port)
	mappingID := fmt.Sprintf("owned-port-%d", port)
	mapping := &cfutils.IngressMapping{
		ID:       mappingID,
		Hostname: hostname,
		Service:  localURL,
		Source:   fmt.Sprintf("owned:%d", port),
	}

	fmt.Fprintf(logs, "[setup] Adding ingress rule: %s -> %s\n", hostname, localURL)
	if err := tg.AddMapping(mapping); err != nil {
		return nil, fmt.Errorf("failed to add mapping to extension tunnel: %v", err)
	}

	publicURL := fmt.Sprintf("https://%s", hostname)

	resultCh := make(chan portforward.TunnelResult, 1)
	go func() {
		// Give the tunnel a few seconds to establish
		time.Sleep(5 * time.Second)
		resultCh <- portforward.TunnelResult{PublicURL: publicURL}
	}()

	return &portforward.TunnelHandle{
		Result: resultCh,
		Logs:   logs,
		Stop: func() {
			fmt.Fprintf(logs, "[cleanup] Removing ingress rule for port %d\n", port)
			if err := tg.RemoveMapping(mappingID); err != nil {
				fmt.Fprintf(logs, "[cleanup] Warning: failed to remove mapping: %v\n", err)
			}
		},
	}, nil
}
