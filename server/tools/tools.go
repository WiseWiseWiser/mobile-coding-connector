package tools

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
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
}

// ToolsResponse is the response from the tools check API.
type ToolsResponse struct {
	OS    string     `json:"os"`
	Tools []ToolInfo `json:"tools"`
}

// requiredTools defines all tools needed by the backend.
var requiredTools = []struct {
	name           string
	description    string
	purpose        string
	versionCmd     []string
	installMacOS   string
	installLinux   string
	installWindows string
}{
	{
		name:           "git",
		description:    "Distributed version control system",
		purpose:        "Clone repositories, track changes, create checkpoints",
		versionCmd:     []string{"git", "--version"},
		installMacOS:   "brew install git",
		installLinux:   "sudo apt install git  # or: sudo yum install git",
		installWindows: "Download from https://git-scm.com/download/win",
	},
	{
		name:           "cloudflared",
		description:    "Cloudflare Tunnel client",
		purpose:        "Create secure tunnels for port forwarding (Cloudflare provider)",
		versionCmd:     []string{"cloudflared", "--version"},
		installMacOS:   "brew install cloudflared",
		installLinux:   "curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64 -o /usr/local/bin/cloudflared && chmod +x /usr/local/bin/cloudflared",
		installWindows: "Download from https://developers.cloudflare.com/cloudflare-one/connections/connect-apps/install-and-setup/installation/",
	},
	{
		name:           "npx",
		description:    "Node.js package runner (comes with npm)",
		purpose:        "Run localtunnel for port forwarding (localtunnel provider)",
		versionCmd:     []string{"npx", "--version"},
		installMacOS:   "brew install node  # npx comes with npm",
		installLinux:   "sudo apt install nodejs npm  # or: curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash - && sudo apt install -y nodejs",
		installWindows: "Download Node.js from https://nodejs.org/",
	},
	{
		name:           "opencode",
		description:    "AI-powered coding assistant CLI",
		purpose:        "AI coding agent for code generation and editing",
		versionCmd:     []string{"opencode", "version"},
		installMacOS:   "go install github.com/opencode-ai/opencode@latest",
		installLinux:   "go install github.com/opencode-ai/opencode@latest",
		installWindows: "go install github.com/opencode-ai/opencode@latest",
	},
	{
		name:           "go",
		description:    "Go programming language",
		purpose:        "Build and run the backend server, install Go-based tools",
		versionCmd:     []string{"go", "version"},
		installMacOS:   "brew install go",
		installLinux:   "sudo apt install golang  # or download from https://go.dev/dl/",
		installWindows: "Download from https://go.dev/dl/",
	},
	{
		name:           "node",
		description:    "Node.js JavaScript runtime",
		purpose:        "Run frontend development server, build React app",
		versionCmd:     []string{"node", "--version"},
		installMacOS:   "brew install node",
		installLinux:   "sudo apt install nodejs  # or: curl -fsSL https://deb.nodesource.com/setup_lts.x | sudo -E bash - && sudo apt install -y nodejs",
		installWindows: "Download from https://nodejs.org/",
	},
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
			InstallMacOS:   tool.installMacOS,
			InstallLinux:   tool.installLinux,
			InstallWindows: tool.installWindows,
		}

		// Check if tool is installed
		path, err := exec.LookPath(tool.name)
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
					info.Version = version
				}
			}
		}

		resp.Tools = append(resp.Tools, info)
	}

	return resp
}

// RegisterAPI registers the tools API endpoint.
func RegisterAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/tools", handleTools)
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
