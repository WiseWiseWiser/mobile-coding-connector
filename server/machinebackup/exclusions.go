package machinebackup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xhd2015/bak-files/pathflag"
)

const (
	nodeModulesRule      = "**/node_modules"
	uploadChunksRule     = "**/upload-chunks"
	logSuffixRule        = "**/*.log"
	binaryRule           = "**(binary)"
	binaryRuleReason     = "executable binaries (reinstallable)"
	exclusionConfigVer   = "1.1"
	customExcludeReason  = "user excluded"
	customIncludeReason  = "user included"
	fromUserConfigReason = "from user config"
	backupMetaDir        = ".backup"
	userBackupConfigRel  = ".ai-critic/backup-config.json"
)

// ExcludePathEntry is one excluded path with a human-readable reason.
type ExcludePathEntry struct {
	Path   string `json:"path"`
	Reason string `json:"reason,omitempty"`
	Files  int    `json:"files,omitempty"`
	Bytes  int64  `json:"bytes,omitempty"`
}

// ExclusionConfig is the JSON written to .backup/config.json and user backup-config.json.
type ExclusionConfig struct {
	Version           string             `json:"version"`
	ExcludePaths      []ExcludePathEntry `json:"exclude_paths"`
	LargeDirThreshold string             `json:"large_dir_threshold,omitempty"`
}

type builtinExclude struct {
	path   string
	reason string
}

// builtinExclusionEntries builds the default exclude set from pathflag.Catalog
// plus the synthetic **(binary) row (content-based detect, not a Classify rule).
func builtinExclusionEntries() []builtinExclude {
	cat := pathflag.Catalog()
	entries := make([]builtinExclude, 0, len(cat)+1)
	entries = append(entries, builtinExclude{binaryRule, binaryRuleReason})
	for _, c := range cat {
		entries = append(entries, builtinExclude{c.Rule, c.Reason})
	}
	return entries
}

var specialExclusionRules = map[string]bool{
	nodeModulesRule:  true,
	uploadChunksRule: true,
	logSuffixRule:    true,
	binaryRule:       true,
}

// ExclusionRules describes merged built-in and custom backup exclusions.
type ExclusionRules struct {
	ExcludedList  []ExcludePathEntry
	fullTrees     map[string]bool
	prefixes      []string
	reasons       map[string]string
	includedPaths map[string]bool
}

// BuiltinExclusionConfig returns the default exclusion config JSON object.
func BuiltinExclusionConfig() ExclusionConfig {
	builtin := builtinExclusionEntries()
	entries := make([]ExcludePathEntry, len(builtin))
	for i, e := range builtin {
		entries[i] = ExcludePathEntry{Path: e.path, Reason: e.reason}
	}
	return ExclusionConfig{Version: exclusionConfigVer, ExcludePaths: entries}
}

// BuiltinExclusionConfigJSON returns indented JSON for the built-in exclusion config.
func BuiltinExclusionConfigJSON() ([]byte, error) {
	return json.MarshalIndent(BuiltinExclusionConfig(), "", "  ")
}

// UserBackupConfigPath returns the persisted backup config path under home.
func UserBackupConfigPath(home string) string {
	return filepath.Join(home, filepath.FromSlash(userBackupConfigRel))
}

// LoadUserBackupConfig reads ~/.ai-critic/backup-config.json when present.
func LoadUserBackupConfig(home string) (*ExclusionConfig, error) {
	path := UserBackupConfigPath(home)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read backup config: %w", err)
	}
	var cfg ExclusionConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse backup config: %w", err)
	}
	return &cfg, nil
}

// MergeUserBackupConfig unions new exclude paths into existing persisted config and
// updates large_dir_threshold only when newThreshold is non-empty.
func MergeUserBackupConfig(existing *ExclusionConfig, newExcludes []ExcludePathEntry, newThreshold string) ExclusionConfig {
	byPath := make(map[string]ExcludePathEntry)
	threshold := ""
	if existing != nil {
		for _, e := range existing.ExcludePaths {
			p := normalizeRelPath(e.Path)
			if p == "" {
				continue
			}
			byPath[p] = ExcludePathEntry{Path: p, Reason: e.Reason}
		}
		threshold = existing.LargeDirThreshold
	}
	for _, e := range newExcludes {
		p := normalizeRelPath(e.Path)
		if p == "" {
			continue
		}
		entry := ExcludePathEntry{Path: p}
		if strings.TrimSpace(e.Reason) != "" {
			entry.Reason = e.Reason
		}
		byPath[p] = entry
	}
	excludePaths := make([]ExcludePathEntry, 0, len(byPath))
	for _, e := range byPath {
		excludePaths = append(excludePaths, e)
	}
	sort.Slice(excludePaths, func(i, j int) bool { return excludePaths[i].Path < excludePaths[j].Path })

	newThreshold = strings.TrimSpace(newThreshold)
	if newThreshold != "" {
		threshold = newThreshold
	}
	return ExclusionConfig{
		Version:           exclusionConfigVer,
		ExcludePaths:      excludePaths,
		LargeDirThreshold: threshold,
	}
}

// SaveUserBackupConfig writes user exclude_paths and optional threshold to ~/.ai-critic/backup-config.json.
// New exclude paths are merged into any existing persisted config; threshold is updated only when provided.
func SaveUserBackupConfig(home string, excludePaths []ExcludePathEntry, largeDirThreshold string) error {
	existing, err := LoadUserBackupConfig(home)
	if err != nil {
		return err
	}
	cfg := MergeUserBackupConfig(existing, excludePaths, largeDirThreshold)
	if err := validatePersistedExcludePaths(cfg.ExcludePaths); err != nil {
		return err
	}
	if strings.TrimSpace(cfg.LargeDirThreshold) != "" {
		if _, err := ParseHumanSize(cfg.LargeDirThreshold); err != nil {
			return fmt.Errorf("invalid large_dir_threshold %q: %v", cfg.LargeDirThreshold, err)
		}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal backup config: %w", err)
	}
	path := UserBackupConfigPath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create backup config dir: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("write backup config: %w", err)
	}
	return nil
}

func validatePersistedExcludePaths(entries []ExcludePathEntry) error {
	for _, e := range entries {
		p := normalizeRelPath(e.Path)
		if p == "" {
			return fmt.Errorf("exclude path must not be empty")
		}
		if p == ".ai-critic" || p == userBackupConfigRel {
			return fmt.Errorf("cannot persist exclude for %q: backup config must remain included", p)
		}
	}
	return nil
}

// ResolveExclusionRules loads persisted user config and merges with builtin and CLI flags.
func ResolveExclusionRules(home string, customExclude, customInclude []string) (ExclusionRules, error) {
	user, err := LoadUserBackupConfig(home)
	if err != nil {
		return ExclusionRules{}, err
	}
	return MergeExclusions(user, customExclude, customInclude), nil
}

// EffectiveExclusionConfigForHome returns the merged config with no CLI overrides.
func EffectiveExclusionConfigForHome(home string) (ExclusionConfig, error) {
	return EffectiveExclusionConfigWithOverrides(home, nil, nil, "")
}

// EffectiveExclusionConfigWithOverrides returns merged builtin + persisted + CLI preview config.
func EffectiveExclusionConfigWithOverrides(home string, exclude, include []string, largeDirThreshold string) (ExclusionConfig, error) {
	rules, err := ResolveExclusionRules(home, exclude, include)
	if err != nil {
		return ExclusionConfig{}, err
	}
	cfg := rules.EffectiveExclusionConfig()
	threshold, err := resolveEffectiveLargeDirThresholdDisplay(home, largeDirThreshold)
	if err != nil {
		return ExclusionConfig{}, err
	}
	cfg.LargeDirThreshold = threshold
	return cfg, nil
}

func resolveEffectiveLargeDirThresholdDisplay(home, cliThreshold string) (string, error) {
	cliThreshold = strings.TrimSpace(cliThreshold)
	if cliThreshold != "" {
		if _, err := ParseHumanSize(cliThreshold); err != nil {
			return "", fmt.Errorf("invalid large_dir_threshold %q: %v", cliThreshold, err)
		}
		return cliThreshold, nil
	}
	user, err := LoadUserBackupConfig(home)
	if err != nil {
		return "", err
	}
	if user != nil && strings.TrimSpace(user.LargeDirThreshold) != "" {
		return user.LargeDirThreshold, nil
	}
	return "", nil
}

// ExcludePathsFromStrings builds user config entries from CLI --exclude paths.
func ExcludePathsFromStrings(paths []string) []ExcludePathEntry {
	out := make([]ExcludePathEntry, 0, len(paths))
	for _, p := range paths {
		p = normalizeRelPath(p)
		if p == "" {
			continue
		}
		out = append(out, ExcludePathEntry{Path: p})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// MergeExclusions returns effective exclusions:
// builtin → user backup-config.json → (− include) ∪ exclude.
// CLI exclude wins over include; include removes from the exclude set.
func MergeExclusions(user *ExclusionConfig, customExclude, customInclude []string) ExclusionRules {
	builtin := builtinExclusionEntries()
	entries := make(map[string]string, len(builtin))
	for _, e := range builtin {
		entries[normalizeRelPath(e.path)] = e.reason
	}

	if user != nil {
		for _, e := range user.ExcludePaths {
			p := normalizeRelPath(e.Path)
			if p == "" {
				continue
			}
			reason := strings.TrimSpace(e.Reason)
			if reason == "" {
				reason = fromUserConfigReason
			}
			entries[p] = reason
		}
	}

	included := make(map[string]bool)
	for _, p := range customInclude {
		p = normalizeRelPath(p)
		if p != "" {
			included[p] = true
			delete(entries, p)
		}
	}
	for _, p := range customExclude {
		p = normalizeRelPath(p)
		if p != "" {
			entries[p] = customExcludeReason
			delete(included, p)
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
		if specialExclusionRules[e.Path] {
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
		ExcludedList:  list,
		fullTrees:     full,
		prefixes:      prefixes,
		reasons:       reasons,
		includedPaths: included,
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

func (r ExclusionRules) isIncludedOverride(rel string) bool {
	return r.includedPaths[normalizeRelPath(rel)]
}

func (r ExclusionRules) hasLogSuffix(rel string) bool {
	return strings.HasSuffix(filepath.Base(normalizeRelPath(rel)), ".log")
}

func (r ExclusionRules) ReasonFor(rel string) string {
	return r.pathReasonFor(rel)
}

func (r ExclusionRules) pathReasonFor(rel string) string {
	if r.isIncludedOverride(rel) {
		return ""
	}
	key := r.ruleKeyForPath(rel)
	if key == "" {
		return ""
	}
	return r.reasons[key]
}

func (r ExclusionRules) ruleKeyForPath(rel string) string {
	rel = normalizeRelPath(rel)
	if rel == "" {
		return ""
	}
	for tree := range r.fullTrees {
		if rel == tree || strings.HasPrefix(rel, tree+"/") {
			return tree
		}
	}
	for _, prefix := range r.prefixes {
		if strings.HasSuffix(prefix, "/**") {
			base := strings.TrimSuffix(prefix, "/**")
			if rel == base || strings.HasPrefix(rel, base+"/") {
				return prefix
			}
			continue
		}
		if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
			return prefix
		}
	}
	// pathflag catalog: segment rules, **/*.log, and any prefix rule still active
	// in reasons (include may have removed a catalog entry such as .cache).
	res, err := pathflag.Classify(rel)
	if err != nil || res.Flags&pathflag.DefaultSkipMask == 0 {
		return ""
	}
	if res.Rule == "" {
		return ""
	}
	if _, ok := r.reasons[res.Rule]; !ok {
		// Catalog rule removed via --include (e.g. include .cache).
		return ""
	}
	return res.Rule
}

func (r ExclusionRules) IsExcluded(rel string) bool {
	if r.isIncludedOverride(rel) {
		return false
	}
	return r.ruleKeyForPath(rel) != ""
}

func (r ExclusionRules) isTopLevelExcluded(name string) bool {
	return r.fullTrees[normalizeRelPath(name)]
}

func normalizeRelPath(p string) string {
	p = filepath.ToSlash(strings.TrimSpace(p))
	p = strings.TrimPrefix(p, "./")
	return p
}