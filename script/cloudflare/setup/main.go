package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/xhd2015/less-gen/flags"
	"github.com/xhd2015/lifelog-private/ai-critic/script/lib"
)

const defaultConfigFile = ".config.local.json"

const help = `
Usage: go run ./script/cloudflare/setup [options]

This script sets up Cloudflare Tunnel for accessing the AI Agent remotely.
It will:
  1. Check if cloudflared is installed
  2. Install cloudflared if needed (with --auto-install)
  3. Check if tunnel is configured for the domain
  4. Set up the tunnel if not configured
  5. Configure DNS route for the domain

Configuration is read from .config.local.json (cloudflare section).
Domain is mandatory in the config file.

Options:
  --auto-install  Automatically install missing binaries
  --dry-run       Show what would be done without making changes
  --force         Force reconfiguration even if already set up
  -h, --help      Show this help message
`

// Config represents the configuration structure
type Config struct {
	Cloudflare CloudflareConfig `json:"cloudflare"`
}

// CloudflareConfig represents the cloudflare-specific configuration
type CloudflareConfig struct {
	Domain     string `json:"domain"`
	TunnelID   string `json:"tunnel_id"`
	LocalPort  string `json:"local_port"`
	ConfigPath string `json:"config_path"`
}

func main() {
	err := Handle(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func Handle(args []string) error {
	var force bool
	var verbose bool
	var env string
	var autoInstall bool
	var dryRun bool

	args, err := flags.String("--env", &env).
		Help("-h,--help", help).
		Bool("--dry-run", &dryRun).
		Bool("--force", &force).
		Bool("--auto-install", &autoInstall).
		Bool("-v,--verbose", &verbose).
		Parse(args)
	if err != nil {
		return err
	}

	// Load configuration from .config.local.json
	config, err := loadConfig()
	if err != nil {
		return err
	}

	defaultPort := strconv.Itoa(lib.DefaultServerPort)

	// Validate mandatory fields
	if config.Cloudflare.Domain == "" {
		return fmt.Errorf(`domain is mandatory but missing from .config.local.json

Example configuration:
{
    "cloudflare": {
        "domain": "agent-baf0d-365bf0f.xhd2015.xyz",
        "tunnel_id": "ai-agent-tunnel",
        "local_port": "%s",
        "config_path": "./cloudflared"
    }
}

Required fields:
  - domain: The subdomain for your AI Agent (mandatory)

Optional fields (with defaults):
  - tunnel_id: Tunnel name (default: "ai-agent-tunnel")
  - local_port: Local server port (default: "%s")
  - config_path: Cloudflared config directory (default: "~/.cloudflared")`, defaultPort, defaultPort)
	}

	// Apply defaults
	tunnelID := config.Cloudflare.TunnelID
	if tunnelID == "" {
		tunnelID = "ai-agent-tunnel"
	}

	localPort := config.Cloudflare.LocalPort
	if localPort == "" {
		localPort = defaultPort
	}

	configPath := config.Cloudflare.ConfigPath
	if configPath == "" {
		homeDir, _ := os.UserHomeDir()
		configPath = filepath.Join(homeDir, ".cloudflared")
	}

	domain := config.Cloudflare.Domain

	fmt.Println("========================================")
	fmt.Println("Cloudflare Tunnel Setup for AI Agent")
	fmt.Println("========================================")
	fmt.Printf("Domain: %s\n", domain)
	fmt.Printf("Tunnel ID: %s\n", tunnelID)
	fmt.Printf("Local Port: %s\n", localPort)
	fmt.Printf("Config Path: %s\n", configPath)
	if dryRun {
		fmt.Println()
		fmt.Println("*** DRY RUN MODE - No changes will be made ***")
	}
	fmt.Println()

	// Step 1: Check if cloudflared is installed
	fmt.Println("Step 1: Checking cloudflared installation...")
	cloudflaredPath, err := exec.LookPath("cloudflared")
	if err != nil {
		fmt.Println("  cloudflared not found.")
		if dryRun {
			if autoInstall {
				fmt.Println("  [DRY RUN] Would auto-install cloudflared")
			}
			// In dry-run mode, show the instructions that would be printed
			fmt.Println("\n  [DRY RUN] Installation instructions that would be shown:")
			fmt.Println()
			switch runtime.GOOS {
			case "darwin":
				fmt.Println("    brew install cloudflared")
				fmt.Println()
				fmt.Println("  Or download directly:")
				fmt.Println("    curl -L -o /usr/local/bin/cloudflared https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-darwin-amd64")
				fmt.Println("    chmod +x /usr/local/bin/cloudflared")
			case "linux":
				fmt.Println("    # For Debian/Ubuntu:")
				fmt.Println("    curl -L --output cloudflared.deb https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb")
				fmt.Println("    sudo dpkg -i cloudflared.deb")
				fmt.Println()
				fmt.Println("    # Or download binary directly:")
				fmt.Println("    curl -L -o /usr/local/bin/cloudflared https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64")
				fmt.Println("    chmod +x /usr/local/bin/cloudflared")
			default:
				fmt.Println("    Visit: https://github.com/cloudflare/cloudflared/releases")
			}
			fmt.Println()
			fmt.Println("  Or run this script with --auto-install flag:")
			fmt.Println("    go run ./script/cloudflare/setup --auto-install")
			return fmt.Errorf("[DRY RUN] cloudflared is required but not installed")
		}
		if autoInstall {
			fmt.Println("  Auto-installing cloudflared...")
			if err := installCloudflared(); err != nil {
				return fmt.Errorf("failed to install cloudflared: %v", err)
			}
			cloudflaredPath = "cloudflared"
			fmt.Println("  ✓ cloudflared installed successfully")
		} else {
			fmt.Println("\n  To install cloudflared, run one of the following commands:")
			fmt.Println()
			switch runtime.GOOS {
			case "darwin":
				fmt.Println("    brew install cloudflared")
				fmt.Println()
				fmt.Println("  Or download directly:")
				fmt.Println("    curl -L -o /usr/local/bin/cloudflared https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-darwin-amd64")
				fmt.Println("    chmod +x /usr/local/bin/cloudflared")
			case "linux":
				fmt.Println("    # For Debian/Ubuntu:")
				fmt.Println("    curl -L --output cloudflared.deb https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb")
				fmt.Println("    sudo dpkg -i cloudflared.deb")
				fmt.Println()
				fmt.Println("    # Or download binary directly:")
				fmt.Println("    curl -L -o /usr/local/bin/cloudflared https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64")
				fmt.Println("    chmod +x /usr/local/bin/cloudflared")
			default:
				fmt.Println("    Visit: https://github.com/cloudflare/cloudflared/releases")
			}
			fmt.Println()
			fmt.Println("  Or run this script with --auto-install flag:")
			fmt.Println("    go run ./script/cloudflare/setup --auto-install")
			return fmt.Errorf("cloudflared is not installed")
		}
	} else {
		fmt.Printf("  ✓ cloudflared found at: %s\n", cloudflaredPath)
	}

	// Step 2: Check if user is authenticated with Cloudflare
	fmt.Println("\nStep 2: Checking Cloudflare authentication...")
	authStatus := isAuthenticated()
	if dryRun {
		if authStatus {
			fmt.Println("  [DRY RUN] User is authenticated with Cloudflare")
		} else {
			fmt.Println("  [DRY RUN] Would run: cloudflared tunnel login")
			fmt.Println("  [DRY RUN] This opens browser to authenticate")
		}
	} else {
		if !authStatus {
			fmt.Println("  Not authenticated. Running 'cloudflared tunnel login'...")
			fmt.Println("  This will open a browser window to authenticate with Cloudflare.")
			fmt.Println("  Please select the zone: xhd2015.xyz")
			if err := runCommand("cloudflared", "tunnel", "login"); err != nil {
				return fmt.Errorf("failed to authenticate: %v", err)
			}
			fmt.Println("  ✓ Authentication successful")
		} else {
			fmt.Println("  ✓ Already authenticated with Cloudflare")
		}
	}

	// Step 3: Check if tunnel exists
	fmt.Println("\nStep 3: Checking tunnel configuration...")
	existingTunnelID, err := getExistingTunnelID(dryRun, tunnelID)
	if err != nil || force {
		if force {
			fmt.Println("  Force flag set. Creating new tunnel...")
		} else {
			fmt.Println("  No existing tunnel found. Creating new tunnel...")
		}

		if dryRun {
			fmt.Printf("  [DRY RUN] Would create tunnel: %s\n", tunnelID)
			fmt.Printf("  [DRY RUN] Would generate tunnel ID: <new-tunnel-id>\n")
		} else {
			existingTunnelID, err = createTunnel(tunnelID)
			if err != nil {
				return fmt.Errorf("failed to create tunnel: %v", err)
			}
			fmt.Printf("  ✓ Tunnel created: %s\n", existingTunnelID)
		}

		// Step 4: Configure DNS route
		fmt.Println("\nStep 4: Configuring DNS route...")
		if dryRun {
			fmt.Printf("  [DRY RUN] Would configure DNS: %s → %s\n", domain, existingTunnelID)
		} else {
			if err := configureDNS(existingTunnelID, domain); err != nil {
				return fmt.Errorf("failed to configure DNS: %v", err)
			}
			fmt.Printf("  ✓ DNS route configured: %s\n", domain)
		}
	} else {
		fmt.Printf("  ✓ Existing tunnel found: %s\n", existingTunnelID)

		// Check if DNS is configured
		fmt.Println("\nStep 4: Checking DNS configuration...")
		dnsConfigured := isDNSConfigured(existingTunnelID, domain, dryRun)
		if dryRun {
			if dnsConfigured {
				fmt.Printf("  [DRY RUN] DNS already configured for %s\n", domain)
			} else {
				fmt.Printf("  [DRY RUN] Would configure DNS: %s → %s\n", domain, existingTunnelID)
			}
		} else {
			if !dnsConfigured {
				fmt.Println("  DNS route not found. Configuring...")
				if err := configureDNS(existingTunnelID, domain); err != nil {
					return fmt.Errorf("failed to configure DNS: %v", err)
				}
				fmt.Printf("  ✓ DNS route configured: %s\n", domain)
			} else {
				fmt.Printf("  ✓ DNS route already configured: %s\n", domain)
			}
		}
	}

	// Step 5: Create/update config file
	fmt.Println("\nStep 5: Creating configuration file...")
	if dryRun {
		fmt.Printf("  [DRY RUN] Would create/update %s/config.yml:\n", configPath)
		fmt.Printf(`  [DRY RUN] Content:
    tunnel: %s
    credentials-file: %s/%s.json
    
    ingress:
      - hostname: %s
        service: http://localhost:%s
      - service: http_status:404
`, existingTunnelID, configPath, existingTunnelID, domain, localPort)
	} else {
		if err := createConfigFile(existingTunnelID, configPath, domain, localPort); err != nil {
			return fmt.Errorf("failed to create config file: %v", err)
		}
		fmt.Println("  ✓ Configuration file created")
	}

	// Step 6: Print summary and instructions
	fmt.Println("\n========================================")
	if dryRun {
		fmt.Println("DRY RUN COMPLETE - No changes were made")
		fmt.Println("========================================")
		fmt.Println()
		fmt.Println("To actually perform the setup, run:")
		fmt.Println("  go run ./script/cloudflare/setup")
		if !autoInstall {
			fmt.Println()
			fmt.Println("To include auto-installation of binaries:")
			fmt.Println("  go run ./script/cloudflare/setup --auto-install")
		}
	} else {
		fmt.Println("Setup Complete!")
		fmt.Println("========================================")
		fmt.Println()
		fmt.Println("To start the tunnel, run:")
		fmt.Printf("  cloudflared tunnel run %s\n", tunnelID)
		fmt.Println()
		fmt.Println("Or install as a system service:")
		fmt.Println("  sudo cloudflared service install")
		fmt.Println("  sudo systemctl start cloudflared")
		fmt.Println()
		fmt.Printf("Your AI Agent will be accessible at:\n")
		fmt.Printf("  https://%s\n", domain)
		fmt.Println()
		fmt.Println("Note: Make sure your AI Agent server is running on port", localPort)
	}

	return nil
}

func loadConfig() (*Config, error) {
	data, err := os.ReadFile(defaultConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, return empty config
			return &Config{}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return &config, nil
}

func installCloudflared() error {
	switch runtime.GOOS {
	case "darwin":
		// macOS - try brew first
		if _, err := exec.LookPath("brew"); err == nil {
			return runCommand("brew", "install", "cloudflared")
		}
		// Fallback to direct download
		return installCloudflaredDirect()
	case "linux":
		return installCloudflaredLinux()
	case "windows":
		return fmt.Errorf("please install cloudflared manually from https://github.com/cloudflare/cloudflared/releases")
	default:
		return fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

func installCloudflaredDirect() error {
	// Download latest release
	url := "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-darwin-amd64"
	if runtime.GOARCH == "arm64" {
		url = "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-darwin-arm64"
	}

	fmt.Printf("  Downloading from %s...\n", url)

	// Download to /usr/local/bin
	targetPath := "/usr/local/bin/cloudflared"
	cmd := exec.Command("curl", "-L", "-o", targetPath, url)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download: %v", err)
	}

	// Make executable
	if err := os.Chmod(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to make executable: %v", err)
	}

	return nil
}

func installCloudflaredLinux() error {
	// Try package managers first
	if _, err := exec.LookPath("apt"); err == nil {
		fmt.Println("  Installing via apt...")
		// Add cloudflare gpg key and repo
		cmds := [][]string{
			{"sh", "-c", "mkdir -p --mode=0755 /usr/share/keyrings"},
			{"sh", "-c", "curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg | tee /usr/share/keyrings/cloudflare-main.gpg >/dev/null"},
			{"sh", "-c", `echo "deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared $(lsb_release -cs) main" | tee /etc/apt/sources.list.d/cloudflared.list`},
			{"apt", "update"},
			{"apt", "install", "-y", "cloudflared"},
		}
		for _, cmd := range cmds {
			if err := runCommand(cmd[0], cmd[1:]...); err != nil {
				// Try direct download as fallback
				return installCloudflaredDirectLinux()
			}
		}
		return nil
	}

	return installCloudflaredDirectLinux()
}

func installCloudflaredDirectLinux() error {
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "amd64"
	} else if arch == "arm64" {
		arch = "arm64"
	}

	url := fmt.Sprintf("https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-%s", arch)
	fmt.Printf("  Downloading from %s...\n", url)

	targetPath := "/usr/local/bin/cloudflared"
	cmd := exec.Command("curl", "-L", "-o", targetPath, url)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to download: %v", err)
	}

	if err := os.Chmod(targetPath, 0755); err != nil {
		return fmt.Errorf("failed to make executable: %v", err)
	}

	return nil
}

func isAuthenticated() bool {
	// Check if cert.pem exists in ~/.cloudflared
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	certPath := filepath.Join(homeDir, ".cloudflared", "cert.pem")
	_, err = os.Stat(certPath)
	return err == nil
}

func getExistingTunnelID(dryRun bool, tunnelName string) (string, error) {
	if dryRun {
		// In dry-run mode, simulate checking for existing tunnel
		// Return empty to simulate no existing tunnel
		return "", fmt.Errorf("[DRY RUN] simulating no existing tunnel")
	}

	// List tunnels and look for one with our tunnel name
	output, err := exec.Command("cloudflared", "tunnel", "list").Output()
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, tunnelName) {
			// Parse tunnel ID from output
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				return fields[0], nil
			}
		}
	}

	return "", fmt.Errorf("no existing tunnel found")
}

func createTunnel(tunnelName string) (string, error) {
	output, err := exec.Command("cloudflared", "tunnel", "create", tunnelName).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %v", string(output), err)
	}

	// Parse tunnel ID from output
	outputStr := string(output)
	// Output format: "Tunnel credentials written to /path/.cloudflared/<tunnel-id>.json"
	if idx := strings.Index(outputStr, ".json"); idx != -1 {
		start := strings.LastIndex(outputStr[:idx], "/")
		if start != -1 {
			tunnelID := outputStr[start+1 : idx]
			return tunnelID, nil
		}
	}

	return "", fmt.Errorf("could not parse tunnel ID from output: %s", outputStr)
}

func configureDNS(tunnelID string, domain string) error {
	output, err := exec.Command("cloudflared", "tunnel", "route", "dns", tunnelID, domain).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %v", string(output), err)
	}
	return nil
}

func isDNSConfigured(tunnelID string, domain string, dryRun bool) bool {
	if dryRun {
		// In dry-run mode, simulate checking DNS
		// Return false to show that DNS would be configured
		return false
	}

	// Check if DNS record exists by listing routes
	output, err := exec.Command("cloudflared", "tunnel", "route", "list").Output()
	if err != nil {
		return false
	}

	return strings.Contains(string(output), domain)
}

func createConfigFile(tunnelID string, configPath string, domain string, localPort string) error {
	// Expand ~ to home directory if needed
	if strings.HasPrefix(configPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configPath = filepath.Join(homeDir, configPath[2:])
	}

	if err := os.MkdirAll(configPath, 0755); err != nil {
		return err
	}

	configFilePath := filepath.Join(configPath, "config.yml")

	// Check if config already exists
	if _, err := os.Stat(configFilePath); err == nil {
		fmt.Println("  Config file already exists. Updating...")
	}

	configContent := fmt.Sprintf(`tunnel: %s
credentials-file: %s

ingress:
  - hostname: %s
    service: http://localhost:%s
  - service: http_status:404
`, tunnelID, filepath.Join(configPath, tunnelID+".json"), domain, localPort)

	return os.WriteFile(configFilePath, []byte(configContent), 0644)
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
