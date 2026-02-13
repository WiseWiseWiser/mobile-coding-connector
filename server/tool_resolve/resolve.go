// Package tool_resolve provides centralized binary resolution that respects
// both the system PATH, the well-known extra install paths (e.g. ~/.local/bin,
// ~/.opencode/bin), and user-configured extra paths from the terminal config.
//
// This package never modifies the process's PATH environment variable.
// Instead, LookPath dynamically searches the system PATH plus all extra paths.
// Callers spawning subprocesses should use AppendExtraPaths to build the env
// for the child process.
package tool_resolve

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// ExtraPaths are common install directories that may not be in the
// server process's PATH but where tools are commonly installed.
var ExtraPaths = []string{
	"/usr/local/bin",
	"/usr/local/go/bin",
}

func init() {
	if home, err := os.UserHomeDir(); err == nil {
		ExtraPaths = append(ExtraPaths,
			home+"/.local/bin",
			home+"/.opencode/bin",
			home+"/go/bin",
			home+"/.bun/bin",
			home+"/.fzf/bin",
		)
	}
	// Dynamically resolve npm's global bin directory (varies by nvm, system install, etc.)
	if out, err := exec.Command("npm", "bin", "-g").Output(); err == nil {
		npmBin := strings.TrimSpace(string(out))
		if npmBin != "" {
			ExtraPaths = append(ExtraPaths, npmBin)
		}
	}
	// Dynamically resolve node's bin directory (needed for npm-installed tools like codex, claude)
	// Use version-aware resolution that prioritizes highest node version
	// The GetFullSearchPATH function will handle the reordering
	if bestNodeDir := findAllNodeVersionDirs(); len(bestNodeDir) > 0 {
		// Add all node directories (higher versions first will be handled in PATH reordering)
		ExtraPaths = append(ExtraPaths, bestNodeDir...)
	}
}

// nodeVersionInfo holds info about a node installation
type nodeVersionInfo struct {
	version string
	dir     string
}

// findAllNodeVersionDirs finds all node installations, groups them by directory,
// and returns directories sorted by their highest node version (highest first)
func findAllNodeVersionDirs() []string {
	// Run 'which -a node' to find all node installations
	out, err := exec.Command("which", "-a", "node").Output()
	if err != nil {
		// Fallback: try 'which node' without -a
		out2, err2 := exec.Command("which", "node").Output()
		if err2 != nil {
			return nil
		}
		out = out2
	}

	paths := strings.Split(strings.TrimSpace(string(out)), "\n")

	// Map to track highest version per directory
	dirVersions := make(map[string]string)

	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}

		// Get the directory
		dir := filepath.Dir(path)

		// Get version from this node
		versionOut, err := exec.Command(path, "--version").Output()
		if err != nil {
			continue
		}

		version := strings.TrimSpace(string(versionOut))
		// Remove 'v' prefix if present
		version = strings.TrimPrefix(version, "v")

		// Keep track of highest version per directory
		if existingVersion, ok := dirVersions[dir]; !ok || version > existingVersion {
			dirVersions[dir] = version
		}
	}

	if len(dirVersions) == 0 {
		return nil
	}

	// Convert map to slice and sort by version (highest first)
	type dirVersion struct {
		dir     string
		version string
	}

	var sorted []dirVersion
	for dir, version := range dirVersions {
		sorted = append(sorted, dirVersion{dir: dir, version: version})
	}

	// Sort by version descending (highest version first)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].version > sorted[i].version {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Extract just the directories
	var result []string
	for _, dv := range sorted {
		result = append(result, dv.dir)
	}

	return result
}

// userExtraPaths holds user-configured extra paths (e.g. from terminal config).
// Set via SetUserExtraPaths at startup or when config changes.
var (
	userPathsMu    sync.RWMutex
	userExtraPaths []string
)

// SetUserExtraPaths sets the user-configured extra paths.
// This is typically called once at startup from the terminal config.
func SetUserExtraPaths(paths []string) {
	userPathsMu.Lock()
	defer userPathsMu.Unlock()
	userExtraPaths = make([]string, len(paths))
	copy(userExtraPaths, paths)
}

// getUserExtraPaths returns a copy of the user extra paths.
func getUserExtraPaths() []string {
	userPathsMu.RLock()
	defer userPathsMu.RUnlock()
	result := make([]string, len(userExtraPaths))
	copy(result, userExtraPaths)
	return result
}

// AllExtraPaths returns ExtraPaths + user extra paths combined.
func AllExtraPaths() []string {
	result := make([]string, len(ExtraPaths))
	copy(result, ExtraPaths)
	result = append(result, getUserExtraPaths()...)
	return result
}

// GetFullSearchPATH returns the system PATH plus all extra paths, deduplicated,
// with node directories reordered so that directories with higher node versions come first.
func GetFullSearchPATH() string {
	systemPath := os.Getenv("PATH")

	// Get all extra paths
	extras := AllExtraPaths()

	// First pass: collect all paths into a map to deduplicate
	pathSet := make(map[string]bool)

	// Add system PATH entries
	for _, p := range strings.Split(systemPath, ":") {
		p = strings.TrimSpace(p)
		if p != "" {
			pathSet[p] = true
		}
	}

	// Add extra paths
	for _, p := range extras {
		p = strings.TrimSpace(p)
		if p != "" {
			pathSet[p] = true
		}
	}

	// Second pass: check which directories have node and their versions
	type dirInfo struct {
		dir         string
		nodeVersion string
		hasNode     bool
	}

	var dirInfos []dirInfo

	for p := range pathSet {
		nodePath := filepath.Join(p, "node")
		info := dirInfo{dir: p}

		if isExecutable(nodePath) {
			info.hasNode = true
			// Get version
			versionOut, err := exec.Command(nodePath, "--version").Output()
			if err == nil {
				version := strings.TrimSpace(string(versionOut))
				version = strings.TrimPrefix(version, "v")
				info.nodeVersion = version
			}
		}

		dirInfos = append(dirInfos, info)
	}

	// Sort: directories with higher node versions come first
	// Directories without node come last (maintaining original order)
	for i := 0; i < len(dirInfos)-1; i++ {
		for j := i + 1; j < len(dirInfos); j++ {
			shouldSwap := false

			// Both have node: higher version comes first
			if dirInfos[i].hasNode && dirInfos[j].hasNode {
				if dirInfos[j].nodeVersion > dirInfos[i].nodeVersion {
					shouldSwap = true
				}
			} else if !dirInfos[i].hasNode && dirInfos[j].hasNode {
				// j has node but i doesn't: swap so node dirs come first
				shouldSwap = true
			}

			if shouldSwap {
				dirInfos[i], dirInfos[j] = dirInfos[j], dirInfos[i]
			}
		}
	}

	// Build final PATH
	var result []string
	for _, info := range dirInfos {
		result = append(result, info.dir)
	}

	return strings.Join(result, ":")
}

// LookPath finds the named binary by searching the system PATH plus all
// extra paths (well-known + user-configured). It does NOT modify the
// process's PATH. Instead, it searches each directory in the combined
// path list for an executable file.
func LookPath(name string) (string, error) {
	// If name contains a slash, it's a path - check directly
	if strings.Contains(name, "/") {
		return lookPathDirect(name)
	}

	dirs := strings.Split(GetFullSearchPATH(), ":")
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, name)
		if isExecutable(candidate) {
			return candidate, nil
		}
	}
	return "", &lookPathError{name: name}
}

// IsAvailable returns true if the named binary can be found.
func IsAvailable(name string) bool {
	_, err := LookPath(name)
	return err == nil
}

// AppendExtraPaths appends all extra paths (well-known + user-configured)
// to the PATH variable in the given environment slice. This is useful when
// spawning child processes that need access to the same tool paths.
func AppendExtraPaths(env []string) []string {
	// Get the full PATH with proper ordering
	fullPath := GetFullSearchPATH()

	for i, e := range env {
		if len(e) > 5 && e[:5] == "PATH=" {
			env[i] = "PATH=" + fullPath
			return env
		}
	}

	// If PATH not found in env, add it
	return append(env, "PATH="+fullPath)
}

func pathContains(pathVal, dir string) bool {
	for _, p := range strings.Split(pathVal, ":") {
		if p == dir {
			return true
		}
	}
	return false
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	// Must be a regular file (not a directory) and executable
	if info.IsDir() {
		return false
	}
	return info.Mode()&0111 != 0
}

func lookPathDirect(name string) (string, error) {
	if isExecutable(name) {
		return name, nil
	}
	return "", &lookPathError{name: name}
}

// lookPathError implements error for LookPath failures,
// matching the interface of exec.ErrNotFound.
type lookPathError struct {
	name string
}

func (e *lookPathError) Error() string {
	return "executable file not found in PATH: " + e.name
}
