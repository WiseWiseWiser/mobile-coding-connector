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
		)
	}
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

// fullSearchPATH returns the system PATH plus all extra paths, deduplicated.
func fullSearchPATH() string {
	systemPath := os.Getenv("PATH")
	extras := AllExtraPaths()

	for _, p := range extras {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !pathContains(systemPath, p) {
			systemPath = systemPath + ":" + p
		}
	}
	return systemPath
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

	dirs := strings.Split(fullSearchPATH(), ":")
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
	extras := AllExtraPaths()
	for i, e := range env {
		if len(e) > 5 && e[:5] == "PATH=" {
			currentPath := e[5:]
			for _, p := range extras {
				p = strings.TrimSpace(p)
				if p == "" {
					continue
				}
				if !pathContains(currentPath, p) {
					currentPath = currentPath + ":" + p
				}
			}
			env[i] = "PATH=" + currentPath
			return env
		}
	}
	return env
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
