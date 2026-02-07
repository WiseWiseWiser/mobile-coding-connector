package tools

import (
	"os"
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
	// Also include user-specific install directories
	if home, err := os.UserHomeDir(); err == nil {
		ExtraPaths = append(ExtraPaths,
			home+"/.local/bin",
			home+"/.opencode/bin",
		)
	}
}

var ensureOnce sync.Once

// EnsurePATH ensures ExtraPaths are in the current process's PATH
// so that exec.LookPath finds tools installed by our install scripts.
// It runs at most once.
func EnsurePATH() {
	ensureOnce.Do(func() {
		currentPath := os.Getenv("PATH")
		for _, p := range ExtraPaths {
			if !strings.Contains(currentPath, p) {
				currentPath = currentPath + ":" + p
			}
		}
		os.Setenv("PATH", currentPath)
	})
}

// AppendExtraPaths appends ExtraPaths to the PATH variable in the given
// environment slice and returns the modified slice.
func AppendExtraPaths(env []string) []string {
	for i, e := range env {
		if len(e) > 5 && e[:5] == "PATH=" {
			currentPath := e[5:]
			for _, p := range ExtraPaths {
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
