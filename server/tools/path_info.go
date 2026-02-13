package tools

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/xhd2015/lifelog-private/ai-critic/server/tool_resolve"
)

// PathInfoResponse contains detailed information about PATH construction
type PathInfoResponse struct {
	// SystemPATH is the original PATH from environment
	SystemPATH string `json:"system_path"`

	// UserPaths are user-configured extra paths
	UserPaths []string `json:"user_paths"`

	// ExtraPaths are well-known extra paths (npm, node, etc.)
	ExtraPaths []string `json:"extra_paths"`

	// FirstPassPATH is system + user + extra paths (before node version sorting)
	FirstPassPATH string `json:"first_pass_path"`

	// SecondPassPATH is first pass + node version prioritization
	SecondPassPATH string `json:"second_pass_path"`

	// FinalPATH is the complete PATH with all directories
	FinalPATH string `json:"final_path"`

	// NodeInstallations shows all discovered node versions
	NodeInstallations []NodeInstallation `json:"node_installations"`
}

// NodeInstallation represents a discovered node installation
type NodeInstallation struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	Dir     string `json:"dir"`
}

// RegisterPathInfoAPI registers the path info API endpoint
func RegisterPathInfoAPI(mux *http.ServeMux) {
	mux.HandleFunc("/api/tools/path-info", handlePathInfo)
}

func handlePathInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	resp := buildPathInfo()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func buildPathInfo() *PathInfoResponse {
	// Get system PATH
	systemPATH := os.Getenv("PATH")

	// Get user paths from tool_resolve
	userPaths := getUserExtraPathsFromResolve()

	// Get extra paths (includes user-configured paths)
	extraPaths := tool_resolve.AllExtraPaths()

	// Build first pass: system + user + extra
	firstPass := buildFirstPassPATH(systemPATH, userPaths, extraPaths)

	// Discover node installations
	nodeInstallations := discoverNodeInstallations()

	// Build second pass: first pass with node version prioritization
	secondPass := prioritizeNodeVersions(firstPass, nodeInstallations)

	resp := &PathInfoResponse{
		SystemPATH:        systemPATH,
		UserPaths:         userPaths,
		ExtraPaths:        extraPaths,
		FirstPassPATH:     firstPass,
		SecondPassPATH:    secondPass,
		FinalPATH:         secondPass,
		NodeInstallations: nodeInstallations,
	}

	return resp
}

// getUserExtraPathsFromResolve retrieves user extra paths
// Since tool_resolve.userExtraPaths is private, we try to get it via reflection
// or return an empty slice for now
func getUserExtraPathsFromResolve() []string {
	// Try to get from tool_resolve using a public method if available
	// For now, return empty - the ExtraPaths already includes user paths
	// via SetUserExtraPaths
	return []string{}
}

func discoverNodeInstallations() []NodeInstallation {
	var installations []NodeInstallation

	// Run 'which -a node' to find all node installations
	out, err := exec.Command("which", "-a", "node").Output()
	if err != nil {
		// Fallback: try 'which node' without -a
		out2, err2 := exec.Command("which", "node").Output()
		if err2 != nil {
			return installations
		}
		out = out2
	}

	paths := strings.Split(strings.TrimSpace(string(out)), "\n")

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		// Get version
		versionOut, err := exec.Command(path, "--version").Output()
		if err != nil {
			continue
		}

		version := strings.TrimSpace(string(versionOut))

		installations = append(installations, NodeInstallation{
			Path:    path,
			Version: version,
			Dir:     filepath.Dir(path),
		})
	}

	return installations
}

func buildFirstPassPATH(systemPATH string, userPaths, extraPaths []string) string {
	var allPaths []string

	// Add system paths
	if systemPATH != "" {
		for _, p := range strings.Split(systemPATH, ":") {
			p = strings.TrimSpace(p)
			if p != "" && !containsString(allPaths, p) {
				allPaths = append(allPaths, p)
			}
		}
	}

	// Add user paths
	for _, p := range userPaths {
		p = strings.TrimSpace(p)
		if p != "" && !containsString(allPaths, p) {
			allPaths = append(allPaths, p)
		}
	}

	// Add extra paths
	for _, p := range extraPaths {
		p = strings.TrimSpace(p)
		if p != "" && !containsString(allPaths, p) {
			allPaths = append(allPaths, p)
		}
	}

	return strings.Join(allPaths, ":")
}

func prioritizeNodeVersions(firstPassPATH string, installations []NodeInstallation) string {
	if len(installations) == 0 {
		return firstPassPATH
	}

	// Group installations by directory and find highest version per directory
	dirVersions := make(map[string]string)
	for _, inst := range installations {
		if existingVersion, ok := dirVersions[inst.Dir]; !ok || inst.Version > existingVersion {
			dirVersions[inst.Dir] = inst.Version
		}
	}

	// Split first pass PATH into ordered list
	paths := strings.Split(firstPassPATH, ":")

	// Separate paths into: those with node (and their versions) and those without
	type pathInfo struct {
		path        string
		nodeVersion string
		hasNode     bool
	}

	var withNode []pathInfo
	var withoutNode []string

	for _, p := range paths {
		info := pathInfo{path: p}
		if version, ok := dirVersions[p]; ok {
			info.hasNode = true
			info.nodeVersion = version
			withNode = append(withNode, info)
		} else {
			withoutNode = append(withoutNode, p)
		}
	}

	// Sort paths with node by version (highest first)
	for i := 0; i < len(withNode)-1; i++ {
		for j := i + 1; j < len(withNode); j++ {
			if withNode[j].nodeVersion > withNode[i].nodeVersion {
				withNode[i], withNode[j] = withNode[j], withNode[i]
			}
		}
	}

	// Build final PATH: high-version node dirs first, then other dirs
	var result []string
	for _, info := range withNode {
		result = append(result, info.path)
	}
	result = append(result, withoutNode...)

	return strings.Join(result, ":")
}

func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
