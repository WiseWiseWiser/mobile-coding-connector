package daemon

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

// versionRegex matches -vN at the end of a binary name (before any extension)
// e.g. "ai-critic-server-linux-amd64-v4" -> version 4
var versionRegex = regexp.MustCompile(`-v(\d+)$`)

// BinaryVersion represents a binary's version information
type BinaryVersion struct {
	Path    string
	Base    string
	Version int
}

// ParseBinVersion extracts the base name and version from a binary path
// Returns (baseName, version). If no -vN suffix, version is 0.
// e.g. "ai-critic-server-linux-amd64"     -> ("ai-critic-server-linux-amd64", 0)
// e.g. "ai-critic-server-linux-amd64-v4"  -> ("ai-critic-server-linux-amd64", 4)
func ParseBinVersion(binPath string) (baseName string, version int) {
	name := filepath.Base(binPath)

	match := versionRegex.FindStringSubmatch(name)
	if match == nil {
		return name, 0
	}

	v, err := strconv.Atoi(match[1])
	if err != nil {
		return name, 0
	}

	baseName = name[:len(name)-len(match[0])]
	return baseName, v
}

// FindNewerBinary looks for a newer versioned binary in the same directory
// Returns the full path to the newer binary, or empty string if none found
func FindNewerBinary(currentBinPath string) string {
	dir := filepath.Dir(currentBinPath)
	currentBase, currentVersion := ParseBinVersion(currentBinPath)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	type candidate struct {
		path    string
		version int
	}

	var candidates []candidate
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()

		// Must start with the same base name
		if !hasPrefix(name, currentBase) {
			continue
		}

		// Parse version
		entryBase, entryVersion := ParseBinVersion(filepath.Join(dir, name))
		if entryBase != currentBase {
			continue
		}

		// Must be strictly newer
		if entryVersion <= currentVersion {
			continue
		}

		// Must be executable (non-zero size)
		info, err := entry.Info()
		if err != nil || info.Size() == 0 {
			continue
		}

		candidates = append(candidates, candidate{
			path:    filepath.Join(dir, name),
			version: entryVersion,
		})
	}

	if len(candidates) == 0 {
		return ""
	}

	// Sort by version descending, pick the highest
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].version > candidates[j].version
	})

	return candidates[0].path
}

// GetNextBinaryPath generates the path for the next versioned binary
func GetNextBinaryPath(currentBinPath string) string {
	dir := filepath.Dir(currentBinPath)
	currentBase, currentVersion := ParseBinVersion(currentBinPath)
	nextVersion := currentVersion + 1
	newName := currentBase + "-v" + strconv.Itoa(nextVersion)
	return filepath.Join(dir, newName)
}

// hasPrefix is a helper to check if a string has a prefix (avoiding strings package for clarity)
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
