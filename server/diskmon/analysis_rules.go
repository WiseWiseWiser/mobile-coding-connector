package diskmon

import (
	"path/filepath"
	"sort"
	"strings"
)

// categoryDef is one stable bucket. The IDs must never change: trend lines join
// on them, so a rename would split the history.
type categoryDef struct {
	ID    string
	Label string
	Kind  string
}

var analysisCategories = []categoryDef{
	{ID: "xcode", Label: "Xcode / Simulators", Kind: "cache"},
	{ID: "containers", Label: "Containers / VMs", Kind: "cache"},
	{ID: "dev-cache", Label: "Dev caches", Kind: "cache"},
	{ID: "caches", Label: "App caches", Kind: "cache"},
	{ID: "media", Label: "Photos / Media", Kind: "media"},
	{ID: "downloads", Label: "Downloads", Kind: "doc"},
	{ID: "logs", Label: "Logs / tmp", Kind: "cache"},
	{ID: "other", Label: "Everything else", Kind: "other"},
}

// classifyRule maps a path prefix to a category. Longest prefix wins, so
// ~/Library/Developer/Xcode/DerivedData can be reclaimable while its parent
// ~/Library/Developer is not.
type classifyRule struct {
	prefix      string
	category    string
	reclaimable bool
}

type classifier struct {
	rules []classifyRule // sorted longest first
	paths map[string][]string
	// volume is a mount prefix dropped before matching, so rules written
	// against "/" and the home directory also match what a walk finds under
	// /System/Volumes/Data.
	volume string
}

func newClassifier(home, volume string) *classifier {
	home = filepath.Clean(home)
	if volume != "" {
		volume = filepath.Clean(volume)
	}
	join := func(rel string) string { return filepath.Join(home, rel) }

	rules := []classifyRule{
		// Xcode and simulators.
		{join("Library/Developer/Xcode/DerivedData"), "xcode", true},
		{join("Library/Developer/Xcode/iOS DeviceSupport"), "xcode", true},
		{join("Library/Developer/Xcode/watchOS DeviceSupport"), "xcode", true},
		{join("Library/Developer/Xcode/tvOS DeviceSupport"), "xcode", true},
		{join("Library/Developer/CoreSimulator/Caches"), "xcode", true},
		{join("Library/Developer/CoreSimulator/Devices"), "xcode", false},
		{join("Library/Developer/Xcode"), "xcode", false},
		{join("Library/Developer"), "xcode", false},
		{"/Library/Developer", "xcode", false},

		// Containers and virtual machines.
		{join("Library/Containers/com.docker.docker"), "containers", true},
		{join("Library/Group Containers/group.com.docker"), "containers", true},
		{join(".docker"), "containers", true},
		{join(".colima"), "containers", true},
		{join(".orbstack"), "containers", true},
		{join(".local/share/containers"), "containers", true},
		{join("Library/Containers"), "containers", false},
		{"/var/lib/docker", "containers", true},

		// Toolchain caches that can be rebuilt.
		{join(".sandbox"), "dev-cache", true},
		{join(".cache"), "dev-cache", true},
		{join(".cargo"), "dev-cache", true},
		{join(".rustup"), "dev-cache", true},
		{join(".gradle"), "dev-cache", true},
		{join(".m2"), "dev-cache", true},
		{join(".npm"), "dev-cache", true},
		{join(".pnpm-store"), "dev-cache", true},
		{join(".bun"), "dev-cache", true},
		{join(".yarn"), "dev-cache", true},
		{join("go/pkg/mod"), "dev-cache", true},
		{join("Library/Caches/Homebrew"), "dev-cache", true},
		{join("Library/Caches/ms-playwright"), "dev-cache", true},
		{join("Library/Caches/go-build"), "dev-cache", true},
		{join("Library/Caches/pip"), "dev-cache", true},
		{join("Library/Caches/com.apple.dt.Xcode"), "dev-cache", true},
		{join(".vscode/extensions"), "dev-cache", false},
		{join(".nvm"), "dev-cache", false},
		{"/opt/homebrew", "dev-cache", false},
		{"/usr/local", "dev-cache", false},

		// Application caches (Spotify, browsers, Electron shells).
		{join("Library/Caches"), "caches", true},
		{"/Library/Caches", "caches", true},

		// Media the user probably wants to keep.
		{join("Pictures"), "media", false},
		{join("Movies"), "media", false},
		{join("Music"), "media", false},

		{join("Downloads"), "downloads", false},

		// Logs, crash reports, trash.
		{join("Library/Logs"), "logs", true},
		{join(".Trash"), "logs", true},
		{"/private/var/log", "logs", true},
	}

	sort.SliceStable(rules, func(i, j int) bool {
		return len(rules[i].prefix) > len(rules[j].prefix)
	})

	paths := map[string][]string{}
	for _, rule := range rules {
		if len(paths[rule.category]) >= 4 {
			continue
		}
		paths[rule.category] = append(paths[rule.category], shortenHome(rule.prefix, home))
	}
	// The fallback bucket has no paths of its own.
	delete(paths, "other")

	return &classifier{rules: rules, paths: paths, volume: volume}
}

// logical drops the volume mount prefix.
func (c *classifier) logical(path string) string { return logicalPath(path, c.volume) }

// logicalPath strips a mount prefix so that rules and walked paths live in the
// same space. The mount point itself and paths outside it come back unchanged.
func logicalPath(path, volume string) string {
	if volume == "" || path == "" {
		return path
	}
	rest, ok := strings.CutPrefix(path, volume)
	if !ok || !strings.HasPrefix(rest, "/") {
		return path
	}
	return rest
}

func (c *classifier) classify(path string) (string, bool) {
	path = c.logical(path)
	for _, rule := range c.rules {
		if path == rule.prefix || strings.HasPrefix(path, rule.prefix+"/") {
			return rule.category, rule.reclaimable
		}
	}
	return "other", false
}

// isRuleRoot reports whether path is itself a category root; those directories
// are worth listing as entries even when they sit deeper than the depth cut.
func (c *classifier) isRuleRoot(path string) bool {
	path = c.logical(path)
	for _, rule := range c.rules {
		if path == rule.prefix {
			return true
		}
	}
	return false
}

// shortenHome makes paths readable on a phone without leaking the username
// twice: /Users/me/Library/... becomes ~/Library/...
func shortenHome(path, home string) string {
	if home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+"/") {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}
