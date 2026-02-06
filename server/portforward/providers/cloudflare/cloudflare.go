package cloudflare

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/xhd2015/lifelog-private/ai-critic/server/config"
	"github.com/xhd2015/lifelog-private/ai-critic/server/portforward"
	"gopkg.in/yaml.v3"
)

// Word lists for random subdomain generation
var adjectives = []string{
	"brave", "calm", "dark", "eager", "fair", "glad", "happy", "idle", "keen", "lush",
	"mild", "neat", "open", "pure", "quick", "rich", "safe", "tall", "vast", "warm",
	"able", "bold", "cool", "deep", "even", "fast", "gold", "high", "just", "kind",
	"lean", "main", "nice", "pale", "rare", "slim", "true", "used", "wide", "wise",
}

var nouns = []string{
	"apex", "beam", "cave", "dawn", "edge", "fern", "gate", "haze", "iris", "jade",
	"kite", "lake", "mesa", "node", "onyx", "pine", "quay", "reef", "star", "tide",
	"vale", "wave", "yard", "zinc", "arch", "bark", "cove", "dune", "flux", "glen",
	"hive", "isle", "jazz", "knot", "loom", "moss", "nest", "opal", "peak", "rift",
}

func generateRandomSubdomain() string {
	pickRandom := func(words []string) string {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(words))))
		return words[n.Int64()]
	}
	return fmt.Sprintf("%s-%s-%s", pickRandom(adjectives), pickRandom(nouns), pickRandom(nouns))
}

// --- Quick Tunnel Provider ---

// QuickProvider implements portforward.Provider using cloudflared quick tunnels (trycloudflare.com)
type QuickProvider struct{}

var _ portforward.Provider = (*QuickProvider)(nil)

func (p *QuickProvider) Name() string       { return portforward.ProviderCloudflareQuick }
func (p *QuickProvider) DisplayName() string { return "Cloudflare Quick Tunnel" }
func (p *QuickProvider) Description() string {
	return "Free tunneling via trycloudflare.com (cloudflared). No account required."
}
func (p *QuickProvider) Available() bool { return portforward.IsCommandAvailable("cloudflared") }

func (p *QuickProvider) Start(port int) (*portforward.TunnelHandle, error) {
	logs := portforward.NewLogBuffer()

	cmd := exec.Command("cloudflared", "tunnel", "--url", fmt.Sprintf("http://127.0.0.1:%d", port))

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

// cloudflaredConfig represents a cloudflared config.yml structure
type cloudflaredConfig struct {
	Tunnel          string        `yaml:"tunnel"`
	CredentialsFile string        `yaml:"credentials-file"`
	Ingress         []ingressRule `yaml:"ingress"`
}

type ingressRule struct {
	Hostname string `yaml:"hostname,omitempty"`
	Service  string `yaml:"service"`
}

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
	pfIngress map[string]ingressRule
}

var _ portforward.Provider = (*TunnelProvider)(nil)

func NewTunnelProvider(cfg config.CloudflareTunnelConfig) *TunnelProvider {
	return &TunnelProvider{
		cfg:       cfg,
		pfIngress: make(map[string]ingressRule),
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

func (p *TunnelProvider) Start(port int) (*portforward.TunnelHandle, error) {
	logs := portforward.NewLogBuffer()

	if p.cfg.BaseDomain == "" {
		return nil, fmt.Errorf("cloudflare tunnel base_domain is not configured")
	}

	tunnelRef := p.cfg.TunnelName
	if tunnelRef == "" {
		tunnelRef = p.cfg.TunnelID
	}

	hostname := fmt.Sprintf("%s.%s", generateRandomSubdomain(), p.cfg.BaseDomain)
	localURL := fmt.Sprintf("http://localhost:%d", port)

	configPath := p.cfg.ConfigPath
	if configPath == ""{
		// default to ~/.cloudflared
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %v", err)
		}
		configPath = filepath.Join(homeDir, ".cloudflared")
	}
	pfConfigFile := fmt.Sprintf("%s/config-portforward.yml", configPath)

	// Step 1: Create DNS route pointing to our dedicated tunnel
	fmt.Fprintf(logs, "[setup] Creating DNS route: %s -> tunnel %s\n", hostname, tunnelRef)
	dnsArgs := []string{"tunnel", "route", "dns", "--overwrite-dns", tunnelRef, hostname}
	dnsCmd := exec.Command("cloudflared", dnsArgs...)
	dnsOutput, err := dnsCmd.CombinedOutput()
	fmt.Fprintf(logs, "[setup] DNS: %s\n", strings.TrimSpace(string(dnsOutput)))
	if err != nil {
		outputStr := strings.TrimSpace(string(dnsOutput))
		if !strings.Contains(outputStr, "already exists") && !strings.Contains(outputStr, "Added CNAME") {
			fmt.Fprintf(logs, "[setup] Warning: DNS route error: %v\n", err)
		}
	}

	// Step 2: Add ingress rule and restart the dedicated tunnel
	p.mu.Lock()
	p.pfIngress[hostname] = ingressRule{Hostname: hostname, Service: localURL}
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
				// No more port forwards - stop the tunnel process
				if p.pfCmd != nil && p.pfCmd.Process != nil {
					fmt.Fprintf(logs, "[cleanup] Stopping dedicated tunnel (no more rules)\n")
					p.pfCmd.Process.Kill()
					p.pfCmd.Wait()
					p.pfCmd = nil
				}
			} else {
				// Restart with remaining rules
				p.writeConfigAndRestart(pfConfigFile, tunnelRef, logs)
			}
			p.mu.Unlock()
		},
	}, nil
}

// ensureTunnelExists checks if the named tunnel exists, and creates it if not.
// Returns the tunnel ID and credentials file path.
func (p *TunnelProvider) ensureTunnelExists(tunnelRef string, logs *portforward.LogBuffer) (tunnelID string, credFile string, err error) {
	tunnelID = p.cfg.TunnelID
	credFile = p.cfg.CredentialsFile

	// If we already have a tunnel ID and credentials file, just verify they exist
	if tunnelID != "" && credFile != "" {
		if _, statErr := os.Stat(credFile); statErr == nil {
			return tunnelID, credFile, nil
		}
		// Credentials file missing, try to find it in default locations
	}

	// Check if tunnel exists by trying `cloudflared tunnel info`
	fmt.Fprintf(logs, "[setup] Checking if tunnel '%s' exists...\n", tunnelRef)
	infoCmd := exec.Command("cloudflared", "tunnel", "info", tunnelRef)
	infoOutput, infoErr := infoCmd.CombinedOutput()
	infoStr := strings.TrimSpace(string(infoOutput))

	if infoErr != nil {
		// Tunnel doesn't exist - create it
		fmt.Fprintf(logs, "[setup] Tunnel not found, creating '%s'...\n", tunnelRef)
		createCmd := exec.Command("cloudflared", "tunnel", "create", tunnelRef)
		createOutput, createErr := createCmd.CombinedOutput()
		createStr := strings.TrimSpace(string(createOutput))
		fmt.Fprintf(logs, "[setup] Create: %s\n", createStr)
		if createErr != nil {
			return "", "", fmt.Errorf("failed to create tunnel: %s", createStr)
		}

		// Parse tunnel ID from output: "Created tunnel <name> with id <uuid>"
		idRegex := regexp.MustCompile(`with id ([a-f0-9-]+)`)
		if match := idRegex.FindStringSubmatch(createStr); len(match) > 1 {
			tunnelID = match[1]
		}

		// Parse credentials file path from output
		credRegex := regexp.MustCompile(`credentials written to (.+\.json)`)
		if match := credRegex.FindStringSubmatch(createStr); len(match) > 1 {
			credFile = match[1]
		}

		// If we still don't have a credentials file, try default location
		if credFile == "" && tunnelID != "" {
			homeDir, _ := os.UserHomeDir()
			defaultCred := filepath.Join(homeDir, ".cloudflared", tunnelID+".json")
			if _, err := os.Stat(defaultCred); err == nil {
				credFile = defaultCred
			}
		}

		fmt.Fprintf(logs, "[setup] Created tunnel: id=%s, credentials=%s\n", tunnelID, credFile)
	} else {
		fmt.Fprintf(logs, "[setup] Tunnel exists: %s\n", infoStr)

		// Parse tunnel ID from info output if we don't have it
		if tunnelID == "" {
			idRegex := regexp.MustCompile(`([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})`)
			if match := idRegex.FindString(infoStr); match != "" {
				tunnelID = match
			}
		}

		// Find credentials file if we don't have it
		if credFile == "" && tunnelID != "" {
			homeDir, _ := os.UserHomeDir()
			defaultCred := filepath.Join(homeDir, ".cloudflared", tunnelID+".json")
			if _, err := os.Stat(defaultCred); err == nil {
				credFile = defaultCred
			}
		}
	}

	if tunnelID == "" {
		return "", "", fmt.Errorf("could not determine tunnel ID for '%s'", tunnelRef)
	}
	if credFile == "" {
		return "", "", fmt.Errorf("could not find credentials file for tunnel '%s' (id: %s)", tunnelRef, tunnelID)
	}

	return tunnelID, credFile, nil
}

// writeConfigAndRestart writes config-portforward.yml with current ingress rules
// and (re)starts the dedicated tunnel process. Must be called with p.mu held.
func (p *TunnelProvider) writeConfigAndRestart(pfConfigFile, tunnelRef string, logs *portforward.LogBuffer) error {
	// Ensure tunnel exists (create if needed)
	tunnelID, credentialsFile, err := p.ensureTunnelExists(tunnelRef, logs)
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
	var rules []ingressRule
	for _, rule := range p.pfIngress {
		rules = append(rules, rule)
	}
	// Catch-all at the end
	rules = append(rules, ingressRule{Service: "http_status:404"})

	// Ensure config directory exists
	cfgDir := filepath.Dir(pfConfigFile)
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %v", cfgDir, err)
	}

	// Write config file
	pfCfg := cloudflaredConfig{
		Tunnel:          tunnelID,
		CredentialsFile: credentialsFile,
		Ingress:         rules,
	}

	cfgData, err := yaml.Marshal(&pfCfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}
	if err := os.WriteFile(pfConfigFile, cfgData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
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

	// Clean up in background
	go func() {
		cmd.Wait()
		fmt.Fprintf(logs, "[tunnel] Dedicated tunnel process exited\n")
	}()

	return nil
}
