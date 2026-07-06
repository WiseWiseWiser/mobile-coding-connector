package machinebackup

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
)

const (
	nodeModulesRule      = "**/node_modules"
	exclusionConfigVer   = "1.0"
	customExcludeReason  = "user excluded"
	customIncludeReason = "user included"
	backupMetaDir        = ".backup"
)

// ExcludePathEntry is one excluded path with a human-readable reason.
type ExcludePathEntry struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// ExclusionConfig is the JSON written to .backup/config.json.
type ExclusionConfig struct {
	Version      string             `json:"version"`
	ExcludePaths []ExcludePathEntry `json:"exclude_paths"`
}

type builtinExclude struct {
	path   string
	reason string
}

var builtinExclusionEntries = []builtinExclude{
	{".bun", "Bun install cache"},
	{".knowledge-index", "knowledge index cache"},
	{".grok/downloads", "Grok downloads cache"},
	{".config/git-fetch-skill/data", "git-fetch-skill data cache"},
	{".config/chromium", "Chromium profile cache"},
	{".cache", "temporary application cache"},
	{".npm", "npm cache"},
	{".cargo/registry", "Cargo registry cache"},
	{".Trash", "macOS trash"},
	{".local/share/Trash", "Linux trash"},
	{backupMetaDir, "machine backup metadata (injected at pack time)"},
	{nodeModulesRule, "node_modules directories"},
}

// ExclusionRules describes merged built-in and custom backup exclusions.
type ExclusionRules struct {
	ExcludedList []ExcludePathEntry
	fullTrees    map[string]bool
	prefixes     []string
	reasons      map[string]string
}

// BuiltinExclusionConfig returns the default exclusion config JSON object.
func BuiltinExclusionConfig() ExclusionConfig {
	entries := make([]ExcludePathEntry, len(builtinExclusionEntries))
	for i, e := range builtinExclusionEntries {
		entries[i] = ExcludePathEntry{Path: e.path, Reason: e.reason}
	}
	return ExclusionConfig{Version: exclusionConfigVer, ExcludePaths: entries}
}

// BuiltinExclusionConfigJSON returns indented JSON for the built-in exclusion config.
func BuiltinExclusionConfigJSON() ([]byte, error) {
	return json.MarshalIndent(BuiltinExclusionConfig(), "", "  ")
}

// MergeExclusions returns effective exclusions: (defaults − include) ∪ exclude.
// Custom exclude wins over include for the same path.
func MergeExclusions(customExclude, customInclude []string) ExclusionRules {
	entries := make(map[string]string, len(builtinExclusionEntries))
	for _, e := range builtinExclusionEntries {
		entries[normalizeRelPath(e.path)] = e.reason
	}

	for _, p := range customInclude {
		p = normalizeRelPath(p)
		if p != "" {
			delete(entries, p)
		}
	}
	for _, p := range customExclude {
		p = normalizeRelPath(p)
		if p != "" {
			entries[p] = customExcludeReason
		}
	}

	list := make([]ExcludePathEntry, 0, len(entries))
	for path, reason := range entries {
		list = append(list, ExcludePathEntry{Path: path, Reason: reason})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Path < list[j].Path })

	full := make(map[string]bool)
	var prefixes []string
	reasons := make(map[string]string, len(list))
	for _, e := range list {
		reasons[e.Path] = e.Reason
		if e.Path == nodeModulesRule {
			continue
		}
		if strings.Contains(e.Path, "/") {
			prefixes = append(prefixes, e.Path)
			continue
		}
		full[e.Path] = true
	}
	sort.Strings(prefixes)

	return ExclusionRules{
		ExcludedList: list,
		fullTrees:    full,
		prefixes:     prefixes,
		reasons:      reasons,
	}
}

// EffectiveExclusionConfig returns the config that would be stored in an archive.
func (r ExclusionRules) EffectiveExclusionConfig() ExclusionConfig {
	return ExclusionConfig{Version: exclusionConfigVer, ExcludePaths: append([]ExcludePathEntry(nil), r.ExcludedList...)}
}

func (r ExclusionRules) ExcludedPaths() []string {
	out := make([]string, len(r.ExcludedList))
	for i, e := range r.ExcludedList {
		out[i] = e.Path
	}
	return out
}

func (r ExclusionRules) ReasonFor(rel string) string {
	rel = normalizeRelPath(rel)
	if rel == "" {
		return ""
	}
	for tree := range r.fullTrees {
		if rel == tree || strings.HasPrefix(rel, tree+"/") {
			if rel == tree {
				return r.reasons[tree]
			}
			return r.reasons[tree]
		}
	}
	for _, prefix := range r.prefixes {
		if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
			return r.reasons[prefix]
		}
	}
	for _, part := range strings.Split(rel, "/") {
		if part == "node_modules" {
			return r.reasons[nodeModulesRule]
		}
	}
	return ""
}

func (r ExclusionRules) IsExcluded(rel string) bool {
	return r.ReasonFor(rel) != ""
}

func (r ExclusionRules) isTopLevelExcluded(name string) bool {
	return r.fullTrees[normalizeRelPath(name)]
}

func normalizeRelPath(p string) string {
	p = filepath.ToSlash(strings.TrimSpace(p))
	p = strings.TrimPrefix(p, "./")
	return p
}