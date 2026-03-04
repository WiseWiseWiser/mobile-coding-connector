package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

// Tool categories for UI grouping.
const (
	CategoryFoundation     = "foundation"
	CategoryNetwork        = "network"
	CategoryLanguage       = "language"
	CategoryVirtualization = "virtualization"
	CategoryCoding         = "coding"
	CategoryTesting        = "testing"
	CategoryOther          = "other"
)

// ToolInfo describes a required tool and its status.
type ToolInfo struct {
	Name           string `json:"name"`
	DisplayName    string `json:"display_name,omitempty"`
	Category       string `json:"category"`
	Description    string `json:"description"`
	Purpose        string `json:"purpose"`
	DocURL         string `json:"doc_url,omitempty"`
	Checking       bool   `json:"checking,omitempty"`
	Installed      bool   `json:"installed"`
	Path           string `json:"path,omitempty"`
	Version        string `json:"version,omitempty"`
	InstallMacOS   string `json:"install_macos"`
	InstallLinux   string `json:"install_linux"`
	InstallWindows string `json:"install_windows"`
	AutoInstallCmd string `json:"auto_install_cmd,omitempty"`
	SettingsPath   string `json:"settings_path,omitempty"`
	UpgradeMacOS   string `json:"upgrade_macos,omitempty"`
	UpgradeLinux   string `json:"upgrade_linux,omitempty"`
	UpgradeWindows string `json:"upgrade_windows,omitempty"`
}

// InstallResponse is the response from the tool install API.
type InstallResponse struct {
	Success bool   `json:"success"`
	Output  string `json:"output,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ToolsResponse is the response from the tools check API.
type ToolsResponse struct {
	OS    string     `json:"os"`
	Tools []ToolInfo `json:"tools"`
}

type toolDef struct {
	name           string
	displayName    string
	category       string
	description    string
	purpose        string
	docURL         string
	versionCmd     []string
	installMacOS   []string
	installLinux   []string
	installWindows string
	settingsPath   string
	upgradeMacOS   []string
	upgradeLinux   []string
	upgradeWindows string
}

// requiredTools defines all tools needed by the backend.
var requiredTools = []toolDef{
	{
		name:        "git",
		category:    CategoryFoundation,
		description: "Distributed version control system",
		purpose:     "Clone repositories, track changes, create checkpoints",
		docURL:      "https://git-scm.com/doc",
		versionCmd:  []string{"git", "--version"},
		installMacOS: []string{
			"brew install git",
		},
		installLinux: []string{
			"apt-get update",
			"apt-get install -y git",
		},
		installWindows: "Download from https://git-scm.com/download/win",
	},
	{
		name:        "curl",
		category:    CategoryFoundation,
		description: "Command-line tool for transferring data with URLs",
		purpose:     "Download tools and files, diagnose network connectivity",
		docURL:      "https://curl.se/docs/",
		versionCmd:  []string{"curl", "--version"},
		installMacOS: []string{
			"brew install curl",
		},
		installLinux: []string{
			"apt-get update",
			"apt-get install -y curl",
		},
		installWindows: "Download from https://curl.se/windows/",
	},
	{
		name:        "ssh",
		category:    CategoryFoundation,
		description: "OpenSSH client for secure remote connections",
		purpose:     "Test SSH keys, connect to git hosts via SSH",
		docURL:      "https://www.openssh.com/manual.html",
		versionCmd:  []string{"ssh", "-V"},
		installMacOS: []string{
			"brew install openssh",
		},
		installLinux: []string{
			"apt-get update",
			"apt-get install -y openssh-client",
		},
		installWindows: "Install via Windows Optional Features or download from https://github.com/PowerShell/Win32-OpenSSH",
	},
	{
		name:        "tar",
		category:    CategoryFoundation,
		description: "Archive utility for creating and extracting tar files",
		purpose:     "Extract downloaded tool archives (e.g. Go, Node.js)",
		docURL:      "https://man7.org/linux/man-pages/man1/tar.1.html",
		versionCmd:  []string{"tar", "--version"},
		installMacOS: []string{
			"brew install gnu-tar",
		},
		installLinux: []string{
			"apt-get update",
			"apt-get install -y tar",
		},
		installWindows: "Download from https://gnuwin32.sourceforge.net/packages/gtar.htm",
	},
	{
		name:         "cloudflared",
		category:     CategoryNetwork,
		description:  "Cloudflare Tunnel client",
		purpose:      "Create secure tunnels for port forwarding (Cloudflare provider)",
		docURL:       "https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/",
		settingsPath: "../settings/cloudflare",
		versionCmd:   []string{"cloudflared", "--version"},
		installMacOS: []string{
			"brew install cloudflared",
		},
		installLinux: []string{
			// Try apt first; if it fails, fall back to direct binary download.
			// Functions called via || are exempt from set -e, so this is safe.
			`install_via_apt() {`,
			`  echo "Trying apt repository install..."`,
			`  curl -fsSL --retry 3 --retry-delay 2 https://pkg.cloudflare.com/cloudflare-main.gpg -o /usr/share/keyrings/cloudflare-main.gpg`,
			`  echo "deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared $(. /etc/os-release && echo $VERSION_CODENAME) main" > /etc/apt/sources.list.d/cloudflared.list`,
			`  apt-get update`,
			`  apt-get install -y cloudflared`,
			`  command -v cloudflared >/dev/null 2>&1 || { echo "cloudflared binary not found after apt install"; return 1; }`,
			`  echo "cloudflared installed via apt"`,
			`}`,
			`install_via_download() {`,
			`  echo "Apt install failed, falling back to direct binary download..."`,
			`  rm -f /etc/apt/sources.list.d/cloudflared.list /usr/share/keyrings/cloudflare-main.gpg 2>/dev/null || true`,
			`  ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')`,
			`  curl -fsSL --retry 3 --retry-delay 2 -o /usr/local/bin/cloudflared "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-${ARCH}"`,
			`  chmod +x /usr/local/bin/cloudflared`,
			`  echo "cloudflared installed via direct download"`,
			`}`,
			`install_via_apt || install_via_download`,
		},
		installWindows: "Download from https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/",
	},
	{
		name:        "zerotier-cli",
		displayName: "ZeroTier",
		category:    CategoryNetwork,
		description: "Peer-to-peer virtual networking tool",
		purpose:     "Create secure virtual networks for connecting devices anywhere",
		docURL:      "https://docs.zerotier.com/",
		versionCmd:  []string{"zerotier-cli", "-v"},
		installMacOS: []string{
			"brew install --cask zerotier-one",
		},
		installLinux: []string{
			"curl -fsSL https://install.zerotier.com | bash",
		},
		installWindows: "Download from https://www.zerotier.com/download/",
	},
	{
		name:        "ngrok",
		category:    CategoryNetwork,
		description: "Secure tunnels to localhost",
		purpose:     "Expose local servers to the internet via secure tunnels",
		docURL:      "https://ngrok.com/docs",
		versionCmd:  []string{"ngrok", "version"},
		installMacOS: []string{
			"brew install ngrok/ngrok/ngrok",
		},
		installLinux: []string{
			`curl -fsSL https://ngrok-agent.s3.amazonaws.com/ngrok.asc | tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null`,
			`echo "deb https://ngrok-agent.s3.amazonaws.com buster main" | tee /etc/apt/sources.list.d/ngrok.list`,
			`apt-get update`,
			`apt-get install -y ngrok`,
		},
		installWindows: "Download from https://ngrok.com/download",
	},
	{
		name:        "lt",
		displayName: "localtunnel",
		category:    CategoryNetwork,
		description: "Expose localhost to the world via a public URL",
		purpose:     "Quick public URL for local development servers",
		docURL:      "https://theboroer.github.io/localtunnel-www/",
		versionCmd:  []string{"lt", "--version"},
		installMacOS: []string{
			"npm install -g localtunnel",
		},
		installLinux: []string{
			"npm install -g localtunnel",
		},
		installWindows: "npm install -g localtunnel",
	},
	{
		name:        "opencode",
		category:    CategoryCoding,
		description: "AI-powered coding assistant CLI",
		purpose:     "AI coding agent for code generation and editing",
		docURL:      "https://opencode.ai",
		versionCmd:  []string{"opencode", "version"},
		installMacOS: []string{
			"curl -fL --retry 3 --retry-delay 2 https://opencode.ai/install | bash",
		},
		installLinux: []string{
			"curl -fL --retry 3 --retry-delay 2 https://opencode.ai/install | bash",
		},
		installWindows: "Download from https://opencode.ai/download",
	},
	{
		name:        "go",
		category:    CategoryLanguage,
		description: "Go programming language",
		purpose:     "Build and run the backend server, install Go-based tools",
		docURL:      "https://go.dev/doc/",
		versionCmd:  []string{"go", "version"},
		installMacOS: []string{
			"brew install go",
		},
		installLinux: []string{
			`ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')`,
			"mkdir -p /tmp/downloads",
			`TARBALL="/tmp/downloads/go1.24.11.linux-${ARCH}.tar.gz"`,
			`[ -f "$TARBALL" ] && echo "Using cached download" || curl -fL --progress-bar --retry 3 --retry-delay 2 -o "$TARBALL" "https://go.dev/dl/go1.24.11.linux-${ARCH}.tar.gz"`,
			`tar -C /usr/local -xzf "$TARBALL"`,
			"ln -sf /usr/local/go/bin/* /usr/local/bin/",
			`echo "Go installed successfully"`,
		},
		installWindows: "Download from https://go.dev/dl/",
	},
	{
		name:        "node",
		category:    CategoryLanguage,
		description: "Node.js JavaScript runtime v22+ (includes npm/npx)",
		purpose:     "Run frontend dev server, build React app, run localtunnel via npx",
		docURL:      "https://nodejs.org/docs/latest/api/",
		versionCmd:  []string{"node", "--version"},
		installMacOS: []string{
			"brew install nvm",
			"source ~/.nvm/nvm.sh",
			"nvm install 22",
			"nvm alias default 22",
		},
		installLinux: []string{
			"curl -fsSL https://deb.nodesource.com/setup_22.x | bash -",
			"apt-get install -y nodejs",
		},
		installWindows: "Download from https://nodejs.org/",
	},
	{
		name:        "npm",
		category:    CategoryLanguage,
		description: "Node.js package manager (comes with node)",
		purpose:     "Install JavaScript packages, run scripts",
		docURL:      "https://docs.npmjs.com/",
		versionCmd:  []string{"npm", "--version"},
		installMacOS: []string{
			"Already included with node - install node instead",
		},
		installLinux: []string{
			"Already included with node - install node instead",
		},
		installWindows: "Already included with node - install node instead",
	},
	{
		name:        "python3",
		category:    CategoryLanguage,
		description: "Python 3 programming language",
		purpose:     "Run Python scripts and tools",
		docURL:      "https://docs.python.org/3/",
		versionCmd:  []string{"python3", "--version"},
		installMacOS: []string{
			"brew install python3",
		},
		installLinux: []string{
			"apt-get update",
			"apt-get install -y python3",
		},
		installWindows: "Download from https://www.python.org/downloads/",
	},
	{
		name:        "jq",
		category:    CategoryFoundation,
		description: "Command-line JSON processor",
		purpose:     "Parse and transform JSON data in shell scripts",
		docURL:      "https://jqlang.github.io/jq/manual/",
		versionCmd:  []string{"jq", "--version"},
		installMacOS: []string{
			"brew install jq",
		},
		installLinux: []string{
			"apt-get update",
			"apt-get install -y jq",
		},
		installWindows: "Download from https://jqlang.github.io/jq/download/",
	},
	{
		name:        "docker",
		category:    CategoryVirtualization,
		description: "Container runtime for building and running applications",
		purpose:     "Build and run containerized applications",
		docURL:      "https://docs.docker.com/",
		versionCmd:  []string{"docker", "--version"},
		installMacOS: []string{
			`echo "Install Docker Desktop from https://www.docker.com/products/docker-desktop/"`,
		},
		installLinux: []string{
			`curl -fsSL https://get.docker.com | sh`,
		},
		installWindows: "Download from https://www.docker.com/products/docker-desktop/",
	},
	{
		name:        "podman",
		category:    CategoryVirtualization,
		description: "Daemonless container engine compatible with Docker",
		purpose:     "Build and run containers without a daemon process",
		docURL:      "https://podman.io/docs",
		versionCmd:  []string{"podman", "--version"},
		installMacOS: []string{
			"brew install podman",
		},
		installLinux: []string{
			"apt-get update",
			"apt-get install -y podman",
		},
		installWindows: "Download from https://podman.io/getting-started/installation",
	},
	{
		name:        "fzf",
		category:    CategoryFoundation,
		description: "Command-line fuzzy finder",
		purpose:     "Interactive fuzzy search for files, history, and more",
		docURL:      "https://junegunn.github.io/fzf/",
		versionCmd:  []string{"fzf", "--version"},
		installMacOS: []string{
			"git clone --depth 1 https://github.com/junegunn/fzf.git ~/.fzf",
			"~/.fzf/install --all",
		},
		installLinux: []string{
			"git clone --depth 1 https://github.com/junegunn/fzf.git ~/.fzf",
			"~/.fzf/install --all",
		},
		installWindows: "Download from https://github.com/junegunn/fzf/releases",
	},
	{
		name:        "agent",
		displayName: "cursor agent",
		category:    CategoryCoding,
		description: "Cursor Agent CLI for AI-powered coding",
		purpose:     "Run cursor agent for code generation and editing from the terminal",
		docURL:      "https://docs.cursor.com/cli",
		versionCmd:  []string{"agent", "--version"},
		installMacOS: []string{
			"curl https://cursor.com/install -fsSL | bash",
		},
		installLinux: []string{
			"curl https://cursor.com/install -fsSL | bash",
		},
		installWindows: "Download from https://cursor.com/en/cli",
		upgradeMacOS: []string{
			"curl https://cursor.com/install -fsSL | bash",
		},
		upgradeLinux: []string{
			"curl https://cursor.com/install -fsSL | bash",
		},
		upgradeWindows: "Download latest from https://cursor.com/en/cli",
	},
	{
		name:        "codex",
		category:    CategoryCoding,
		description: "OpenAI Codex CLI for AI-powered coding",
		purpose:     "Run OpenAI Codex agent for code generation and editing",
		docURL:      "https://github.com/openai/codex",
		versionCmd:  []string{"codex", "--version"},
		installMacOS: []string{
			"npm install -g @openai/codex",
		},
		installLinux: []string{
			"npm install -g @openai/codex",
		},
		installWindows: "npm install -g @openai/codex",
		upgradeMacOS: []string{
			"npm update -g @openai/codex",
		},
		upgradeLinux: []string{
			"npm update -g @openai/codex",
		},
		upgradeWindows: "npm update -g @openai/codex",
	},
	{
		name:        "openclaw",
		category:    CategoryCoding,
		description: "OpenClaw CLI for AI agent orchestration",
		purpose:     "Run OpenClaw agent and gateway workflows from the terminal",
		docURL:      "https://github.com/openclaw/openclaw",
		versionCmd:  []string{"openclaw", "--version"},
		installMacOS: []string{
			"npm install -g openclaw@latest",
		},
		installLinux: []string{
			"npm install -g openclaw@latest",
		},
		installWindows: "npm install -g openclaw@latest",
		upgradeMacOS: []string{
			"npm update -g openclaw",
		},
		upgradeLinux: []string{
			"npm update -g openclaw",
		},
		upgradeWindows: "npm update -g openclaw",
	},
	{
		name:        "github-copilot",
		category:    CategoryCoding,
		description: "GitHub Copilot CLI - AI-powered code assistant",
		purpose:     "AI coding agent powered by GitHub Copilot for code generation and editing",
		docURL:      "https://docs.github.com/en/copilot",
		versionCmd:  []string{"github-copilot", "--version"},
		installMacOS: []string{
			"npm install -g @github/copilot-cli",
		},
		installLinux: []string{
			"npm install -g @github/copilot-cli",
		},
		installWindows: "npm install -g @github/copilot-cli",
		upgradeMacOS: []string{
			"npm update -g @github/copilot-cli",
		},
		upgradeLinux: []string{
			"npm update -g @github/copilot-cli",
		},
		upgradeWindows: "npm update -g @github/copilot-cli",
	},
	{
		name:        "claude",
		category:    CategoryCoding,
		description: "Claude Code CLI by Anthropic",
		purpose:     "Run Claude Code agent for code generation and editing",
		docURL:      "https://docs.anthropic.com/en/docs/claude-code",
		versionCmd:  []string{"claude", "--version"},
		installMacOS: []string{
			"npm install -g @anthropic-ai/claude-code",
		},
		installLinux: []string{
			"npm install -g @anthropic-ai/claude-code",
		},
		installWindows: "npm install -g @anthropic-ai/claude-code",
		upgradeMacOS: []string{
			"npm update -g @anthropic-ai/claude-code",
		},
		upgradeLinux: []string{
			"npm update -g @anthropic-ai/claude-code",
		},
		upgradeWindows: "npm update -g @anthropic-ai/claude-code",
	},
	{
		name:        "cline",
		category:    CategoryCoding,
		description: "Cline CLI - Autonomous coding agent",
		purpose:     "Run Cline autonomous coding agent for code generation and editing",
		docURL:      "https://github.com/cline/cline",
		versionCmd:  []string{"cline", "version"},
		installMacOS: []string{
			"npm install -g cline",
		},
		installLinux: []string{
			"npm install -g cline",
		},
		installWindows: "npm install -g cline",
	},
	{
		name:        "whats_next",
		category:    CategoryCoding,
		description: "Task tracking and planning CLI tool",
		purpose:     "Track tasks and plan next steps",
		docURL:      "https://github.com/xhd2015/whats_next",
		versionCmd:  []string{"whats_next", "version"},
		installMacOS: []string{
			"go install github.com/xhd2015/whats_next@latest",
		},
		installLinux: []string{
			"go install github.com/xhd2015/whats_next@latest",
		},
		installWindows: "go install github.com/xhd2015/whats_next@latest",
	},
	{
		name:        "kool",
		category:    CategoryCoding,
		description: "Developer toolchain manager",
		purpose:     "Manage development toolchains and environments",
		docURL:      "https://github.com/xhd2015/kool",
		versionCmd:  []string{"kool", "version"},
		installMacOS: []string{
			"go install github.com/xhd2015/kool@latest",
		},
		installLinux: []string{
			"go install github.com/xhd2015/kool@latest",
		},
		installWindows: "go install github.com/xhd2015/kool@latest",
	},
	{
		name:        "kilocode",
		category:    CategoryCoding,
		description: "Kilo Code AI-powered coding assistant CLI",
		purpose:     "AI coding agent for code generation and editing",
		docURL:      "https://kilocode.ai",
		versionCmd:  []string{"kilocode", "--version"},
		installMacOS: []string{
			"npm install -g @kilocode/cli",
		},
		installLinux: []string{
			"npm install -g @kilocode/cli",
		},
		installWindows: "npm install -g @kilocode/cli",
	},
	{
		name:        "lsof",
		category:    CategoryFoundation,
		description: "List open files and network connections",
		purpose:     "Detect local listening ports for port forwarding",
		docURL:      "https://man7.org/linux/man-pages/man8/lsof.8.html",
		versionCmd:  []string{"lsof", "-v"},
		installMacOS: []string{
			"brew install lsof",
		},
		installLinux: []string{
			"apt-get update",
			"apt-get install -y lsof",
		},
		installWindows: "lsof is not available on Windows. Use 'netstat -an' instead.",
	},
	{
		name:        "bun",
		category:    CategoryLanguage,
		description: "Fast JavaScript all-in-one toolkit",
		purpose:     "Run JavaScript/TypeScript with fast package management",
		docURL:      "https://bun.sh/docs",
		versionCmd:  []string{"bun", "--version"},
		installMacOS: []string{
			"curl -fsSL https://bun.sh/install | bash",
		},
		installLinux: []string{
			"curl -fsSL https://bun.sh/install | bash",
		},
		installWindows: "powershell -c \"irm bun.sh/install.ps1|iex\"",
	},
	{
		name:        "chromium",
		category:    CategoryTesting,
		description: "Open-source web browser for web scraping and automation",
		purpose:     "Headless browser for web scraping, PDF generation, and automated testing",
		docURL:      "https://www.chromium.org/developers/",
		versionCmd:  []string{"chromium", "--version"},
		installMacOS: []string{
			"brew install chromium",
		},
		installLinux: []string{
			"apt-get update",
			"apt-get install -y chromium-browser || apt-get install -y chromium",
		},
		installWindows: "Download from https://www.chromium.org/getting-involved/download-chromium",
	},
	{
		name:        "puppeteer",
		category:    CategoryTesting,
		description: "Node.js library for headless Chrome/Chromium control",
		purpose:     "Programmatic browser automation, web scraping, and screenshot generation",
		docURL:      "https://pptr.dev/",
		versionCmd:  []string{"npx", "puppeteer", "--version"},
		installMacOS: []string{
			"npm install -g puppeteer",
		},
		installLinux: []string{
			"npm install -g puppeteer",
		},
		installWindows: "npm install -g puppeteer",
	},
	{
		name:        "playwright",
		category:    CategoryTesting,
		description: "Node.js library for browser automation and testing",
		purpose:     "Cross-browser automation, web testing, and scraping with multiple browser engines",
		docURL:      "https://playwright.dev/docs/intro",
		versionCmd:  []string{"npx", "playwright", "--version"},
		installMacOS: []string{
			"npm install -g playwright",
		},
		installLinux: []string{
			"npm install -g playwright",
		},
		installWindows: "npm install -g playwright",
	},
	{
		name:        "agent-browser",
		category:    CategoryTesting,
		description: "Lightweight headless browser CLI for AI agents",
		purpose:     "Browser automation with deterministic element references for AI coding agents",
		docURL:      "https://github.com/vercel-labs/agent-browser",
		versionCmd:  []string{"agent-browser", "--version"},
		installMacOS: []string{
			"npm install -g agent-browser",
		},
		installLinux: []string{
			"npm install -g agent-browser",
		},
		installWindows: "npm install -g agent-browser",
		upgradeMacOS: []string{
			"npm update -g agent-browser",
		},
		upgradeLinux: []string{
			"npm update -g agent-browser",
		},
		upgradeWindows: "npm update -g agent-browser",
	},
	{
		name:        "playwriter",
		category:    CategoryTesting,
		description: "Chrome extension & CLI/MCP for AI agent browser control",
		purpose:     "Run Playwright snippets in a persistent Chrome session via CLI or MCP for authenticated flows and dashboard testing",
		docURL:      "https://github.com/nichochar/playwriter",
		versionCmd:  []string{"playwriter", "--version"},
		installMacOS: []string{
			"npm install -g playwriter",
		},
		installLinux: []string{
			"npm install -g playwriter",
		},
		installWindows: "npm install -g playwriter",
		upgradeMacOS: []string{
			"npm update -g playwriter",
		},
		upgradeLinux: []string{
			"npm update -g playwriter",
		},
		upgradeWindows: "npm update -g playwriter",
	},
	{
		name:        "dig",
		category:    CategoryFoundation,
		description: "DNS lookup utility",
		purpose:     "Query DNS records for diagnosing domain and network issues",
		docURL:      "https://man7.org/linux/man-pages/man1/dig.1.html",
		versionCmd:  []string{"dig", "+version"},
		installMacOS: []string{
			"brew install bind",
		},
		installLinux: []string{
			"apt-get update",
			"apt-get install -y dnsutils",
		},
		installWindows: "Download from https://www.isc.org/bind/",
	},
}

// getInstallStepsForOS returns the install steps for the current OS.
func getInstallStepsForOS(tool toolDef) []string {
	switch runtime.GOOS {
	case "darwin":
		return tool.installMacOS
	case "linux":
		return tool.installLinux
	default:
		return tool.installLinux
	}
}

// joinInstallSteps joins install steps into a single display string.
func joinInstallSteps(steps []string) string {
	return strings.Join(steps, "\n")
}

func buildToolInfo(tool toolDef) ToolInfo {
	return ToolInfo{
		Name:           tool.name,
		DisplayName:    tool.displayName,
		Category:       tool.category,
		Description:    tool.description,
		Purpose:        tool.purpose,
		DocURL:         tool.docURL,
		InstallMacOS:   joinInstallSteps(tool.installMacOS),
		InstallLinux:   joinInstallSteps(tool.installLinux),
		InstallWindows: tool.installWindows,
		SettingsPath:   tool.settingsPath,
		UpgradeMacOS:   joinInstallSteps(tool.upgradeMacOS),
		UpgradeLinux:   joinInstallSteps(tool.upgradeLinux),
		UpgradeWindows: tool.upgradeWindows,
	}
}

// getAutoInstallScript checks if the install steps can be auto-installed
// and returns the bash script. Returns empty string if not auto-installable.
func getAutoInstallScript(steps []string) string {
	if len(steps) == 0 {
		return ""
	}

	// Windows is never auto-installable
	if runtime.GOOS == "windows" {
		return ""
	}

	// Check the first line to determine if auto-install is possible
	firstLine := steps[0]

	// Strip inline comments for analysis
	analysis := firstLine
	if idx := strings.Index(analysis, "#"); idx > 0 {
		analysis = strings.TrimSpace(analysis[:idx])
	}

	// Not auto-installable if it's a download instruction (text, not a command)
	if strings.HasPrefix(analysis, "Download ") {
		return ""
	}

	// If running as root, sudo is unnecessary; otherwise reject sudo commands
	isRoot := os.Getuid() == 0
	for _, line := range steps {
		if strings.Contains(line, "sudo ") {
			if !isRoot {
				return ""
			}
		}
	}

	// Build the script — strip sudo when running as root.
	// We don't check if the base tool is available; if a dependency is
	// missing, "set -e" in the install runner will abort with a clear error.
	script := joinInstallSteps(steps)
	if isRoot {
		script = strings.ReplaceAll(script, "sudo ", "")
	}

	return script
}

// GetInstallHint returns the install command for a tool on the current OS.
// Returns empty string if the tool is already installed or not found.
func GetInstallHint(name string) string {
	if tool_resolve.IsAvailable(name) {
		return "" // already installed
	}
	for _, tool := range requiredTools {
		if tool.name != name {
			continue
		}
		steps := getInstallStepsForOS(tool)
		if len(steps) == 0 {
			return ""
		}
		return joinInstallSteps(steps)
	}
	return ""
}

// checkSingleTool resolves install status, path, and version for a single tool.
func checkSingleTool(tool toolDef) ToolInfo {
	info := buildToolInfo(tool)

	lookupName := tool.name
	if len(tool.versionCmd) > 0 {
		lookupName = tool.versionCmd[0]
	}
	path, err := tool_resolve.LookPath(lookupName)
	if err != nil {
		info.AutoInstallCmd = getAutoInstallScript(getInstallStepsForOS(tool))
		return info
	}

	info.Installed = true
	info.Path = path

	if len(tool.versionCmd) > 0 {
		cmd := exec.Command(tool.versionCmd[0], tool.versionCmd[1:]...)
		cmd.Env = tool_resolve.AppendExtraPaths(os.Environ())
		out, err := cmd.Output()
		if err == nil {
			version := strings.TrimSpace(string(out))
			if idx := strings.Index(version, "\n"); idx > 0 {
				version = version[:idx]
			}
			const maxVersionLen = 60
			if len(version) > maxVersionLen {
				version = version[:maxVersionLen] + "..."
			}
			info.Version = version
		}
	}
	return info
}

// CheckTools checks all required tools and returns their status.
func CheckTools() *ToolsResponse {
	resp := &ToolsResponse{
		OS:    runtime.GOOS,
		Tools: make([]ToolInfo, 0, len(requiredTools)),
	}
	for _, tool := range requiredTools {
		resp.Tools = append(resp.Tools, checkSingleTool(tool))
	}
	return resp
}

// CheckToolsQuick returns tool definitions without expensive status checks.
func CheckToolsQuick() *ToolsResponse {
	resp := &ToolsResponse{
		OS:    runtime.GOOS,
		Tools: make([]ToolInfo, 0, len(requiredTools)),
	}

	for _, tool := range requiredTools {
		info := buildToolInfo(tool)
		info.Checking = true
		resp.Tools = append(resp.Tools, info)
	}
	return resp
}

func handleUpgradeTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	toolName := r.URL.Query().Get("name")
	if toolName == "" {
		http.Error(w, "Missing 'name' parameter", http.StatusBadRequest)
		return
	}

	// Find the tool and its upgrade script
	var script string
	found := false
	for _, tool := range requiredTools {
		if tool.name != toolName {
			continue
		}
		found = true
		script = getUpgradeScriptForOS(tool)
		break
	}
	if !found {
		http.Error(w, "Unknown tool", http.StatusNotFound)
		return
	}
	if script == "" {
		http.Error(w, "Tool cannot be upgraded on this OS", http.StatusBadRequest)
		return
	}

	// Stream upgrade output via SSE
	sw := sse.NewWriter(w)
	if sw == nil {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Prepend "set -e" so any command failure aborts the script immediately
	fullScript := "set -e\n" + script

	sw.SendLog(fmt.Sprintf("$ %s", strings.ReplaceAll(script, "\n", "\n$ ")))

	cmd := exec.Command("bash", "-c", fullScript)
	// Set up environment with extended PATH
	cmd.Env = tool_resolve.AppendExtraPaths(os.Environ())
	if err := sw.StreamCmd(cmd); err != nil {
		sw.SendError(fmt.Sprintf("Upgrade failed: %v", err))
	} else {
		sw.SendDone(map[string]string{"message": "Upgraded successfully"})
	}
}

// getUpgradeScriptForOS returns the upgrade script for the current OS.
func getUpgradeScriptForOS(tool toolDef) string {
	var steps []string
	switch runtime.GOOS {
	case "darwin":
		steps = tool.upgradeMacOS
	case "linux":
		steps = tool.upgradeLinux
	case "windows":
		if tool.upgradeWindows != "" {
			return tool.upgradeWindows
		}
		return ""
	default:
		steps = tool.upgradeLinux
	}

	if len(steps) == 0 {
		return ""
	}

	// Use same logic as getAutoInstallScript for consistency
	return getAutoInstallScript(steps)
}

func handleToolsStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	sw := sse.NewWriter(w)
	if sw == nil {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Send initial list with checking=true so the UI can render skeletons
	initTools := make([]ToolInfo, 0, len(requiredTools))
	for _, tool := range requiredTools {
		info := buildToolInfo(tool)
		info.Checking = true
		initTools = append(initTools, info)
	}
	sw.Send(map[string]interface{}{
		"type":  "init",
		"os":    runtime.GOOS,
		"tools": initTools,
	})

	// Check tools concurrently with a semaphore (max 10)
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)
	var mu sync.Mutex

	var wg sync.WaitGroup
	for _, tool := range requiredTools {
		wg.Add(1)
		go func(t toolDef) {
			defer wg.Done()
			sem <- struct{}{}
			info := checkSingleTool(t)
			<-sem

			mu.Lock()
			sw.Send(map[string]interface{}{
				"type": "tool",
				"tool": info,
			})
			mu.Unlock()
		}(tool)
	}
	wg.Wait()

	sw.SendDone(nil)
}

func handleTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	quick := r.URL.Query().Get("quick")
	var resp *ToolsResponse
	if quick == "1" || strings.EqualFold(quick, "true") {
		resp = CheckToolsQuick()
	} else {
		resp = CheckTools()
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleInstallTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	toolName := r.URL.Query().Get("name")
	if toolName == "" {
		http.Error(w, "Missing 'name' parameter", http.StatusBadRequest)
		return
	}

	// Find the tool and its auto-install script
	var script string
	found := false
	for _, tool := range requiredTools {
		if tool.name != toolName {
			continue
		}
		found = true
		script = getAutoInstallScript(getInstallStepsForOS(tool))
		break
	}
	if !found {
		http.Error(w, "Unknown tool", http.StatusNotFound)
		return
	}
	if script == "" {
		http.Error(w, "Tool cannot be auto-installed on this OS", http.StatusBadRequest)
		return
	}

	// Stream install output via SSE
	sw := sse.NewWriter(w)
	if sw == nil {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Prepend "set -eo pipefail" so any command failure (including inside
	// pipelines like "curl ... | bash") aborts the script immediately.
	fullScript := "set -eo pipefail\n" + script

	sw.SendLog(fmt.Sprintf("$ %s", strings.ReplaceAll(script, "\n", "\n$ ")))

	cmd := exec.Command("bash", "-c", fullScript)
	// Set up environment with extended PATH so npm/node can be found
	cmd.Env = tool_resolve.AppendExtraPaths(os.Environ())
	if err := sw.StreamCmd(cmd); err != nil {
		sw.SendError(fmt.Sprintf("Install failed: %v", err))
	} else {
		sw.SendDone(map[string]string{"message": "Installed successfully"})
	}
}

// RegisterAPI registers the tools API endpoint.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/tools", handleTools)
	mux.HandleFunc("/api/tools/stream", handleToolsStream)
	mux.HandleFunc("/api/tools/install", handleInstallTool)
	mux.HandleFunc("/api/tools/upgrade", handleUpgradeTool)
	RegisterPathInfoAPI(mux)
}
