package tools

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/server/sse"
	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

// ToolInfo describes a required tool and its status.
type ToolInfo struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	Purpose        string `json:"purpose"`
	Installed      bool   `json:"installed"`
	Path           string `json:"path,omitempty"`
	Version        string `json:"version,omitempty"`
	InstallMacOS   string `json:"install_macos"`
	InstallLinux   string `json:"install_linux"`
	InstallWindows string `json:"install_windows"`
	AutoInstallCmd string `json:"auto_install_cmd,omitempty"`
	SettingsPath   string `json:"settings_path,omitempty"`
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

// toolDef defines a required tool and how to install it.
// installMacOS and installLinux are multi-line bash scripts.
// installWindows is a plain text instruction (no auto-install).
type toolDef struct {
	name           string
	description    string
	purpose        string
	versionCmd     []string
	installMacOS   []string
	installLinux   []string
	installWindows string
	settingsPath   string // relative path for tool-specific settings page
}

// requiredTools defines all tools needed by the backend.
var requiredTools = []toolDef{
	{
		name:        "git",
		description: "Distributed version control system",
		purpose:     "Clone repositories, track changes, create checkpoints",
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
		description: "Command-line tool for transferring data with URLs",
		purpose:     "Download tools and files, diagnose network connectivity",
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
		description: "OpenSSH client for secure remote connections",
		purpose:     "Test SSH keys, connect to git hosts via SSH",
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
		description: "Archive utility for creating and extracting tar files",
		purpose:     "Extract downloaded tool archives (e.g. Go, Node.js)",
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
		description:  "Cloudflare Tunnel client",
		purpose:      "Create secure tunnels for port forwarding (Cloudflare provider)",
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
		name:        "opencode",
		description: "AI-powered coding assistant CLI",
		purpose:     "AI coding agent for code generation and editing",
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
		description: "Go programming language",
		purpose:     "Build and run the backend server, install Go-based tools",
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
		description: "Node.js JavaScript runtime (includes npm/npx)",
		purpose:     "Run frontend dev server, build React app, run localtunnel via npx",
		versionCmd:  []string{"node", "--version"},
		installMacOS: []string{
			"brew install node",
		},
		installLinux: []string{
			"apt-get update",
			"apt-get install -y nodejs",
		},
		installWindows: "Download from https://nodejs.org/",
	},
	{
		name:        "python3",
		description: "Python 3 programming language",
		purpose:     "Run Python scripts and tools",
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
		description: "Command-line JSON processor",
		purpose:     "Parse and transform JSON data in shell scripts",
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
		description: "Container runtime for building and running applications",
		purpose:     "Build and run containerized applications",
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
		description: "Daemonless container engine compatible with Docker",
		purpose:     "Build and run containers without a daemon process",
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
		description: "Command-line fuzzy finder",
		purpose:     "Interactive fuzzy search for files, history, and more",
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

// CheckTools checks all required tools and returns their status.
func CheckTools() *ToolsResponse {
	resp := &ToolsResponse{
		OS:    runtime.GOOS,
		Tools: make([]ToolInfo, 0, len(requiredTools)),
	}

	for _, tool := range requiredTools {
		info := ToolInfo{
			Name:           tool.name,
			Description:    tool.description,
			Purpose:        tool.purpose,
			InstallMacOS:   joinInstallSteps(tool.installMacOS),
			InstallLinux:   joinInstallSteps(tool.installLinux),
			InstallWindows: tool.installWindows,
			SettingsPath:   tool.settingsPath,
		}

		// Check if tool is installed
		path, err := tool_resolve.LookPath(tool.name)
		if err == nil {
			info.Installed = true
			info.Path = path

			// Get version
			if len(tool.versionCmd) > 0 {
				cmd := exec.Command(tool.versionCmd[0], tool.versionCmd[1:]...)
				out, err := cmd.Output()
				if err == nil {
			version := strings.TrimSpace(string(out))
				// Take first line only
				if idx := strings.Index(version, "\n"); idx > 0 {
					version = version[:idx]
				}
				// Limit length to avoid overflow in UI
				const maxVersionLen = 60
				if len(version) > maxVersionLen {
					version = version[:maxVersionLen] + "..."
				}
				info.Version = version
				}
			}
		} else {
			// Not installed — check if auto-install is possible
			info.AutoInstallCmd = getAutoInstallScript(getInstallStepsForOS(tool))
		}

		resp.Tools = append(resp.Tools, info)
	}

	return resp
}

// RegisterAPI registers the tools API endpoint.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/tools", handleTools)
	mux.HandleFunc("/api/tools/install", handleInstallTool)
}

func handleTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := CheckTools()
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

	// Prepend "set -e" so any command failure aborts the script immediately,
	// preventing false "success" messages when intermediate commands fail.
	fullScript := "set -e\n" + script

	sw.SendLog(fmt.Sprintf("$ %s", strings.ReplaceAll(script, "\n", "\n$ ")))

	cmd := exec.Command("bash", "-c", fullScript)
	if err := sw.StreamCmd(cmd); err != nil {
		sw.SendError(fmt.Sprintf("Install failed: %v", err))
	} else {
		sw.SendDone(map[string]string{"message": "Installed successfully"})
	}
}
