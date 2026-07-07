package machinebackup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuiltinExclusionConfigV11(t *testing.T) {
	cfg := BuiltinExclusionConfig()
	if cfg.Version != "1.1" {
		t.Fatalf("version = %q, want 1.1", cfg.Version)
	}
	want := map[string]string{
		binaryRule:                            "executable binaries (reinstallable)",
		logSuffixRule:                         "log files",
		uploadChunksRule:                      "incomplete upload temp state",
		".local/share/cursor-agent/versions":  "Cursor agent version cache",
		".opencode/bin":                       "OpenCode binary (reinstallable)",
		".codex/.tmp":                         "Codex temporary plugin cache",
		".codex/skills/.system":               "Codex system skills cache",
		".local/share/opencode/repos":         "OpenCode repo clone cache",
		".local/share/opencode/snapshot":      "OpenCode snapshot cache",
		".local/share/opencode/log":           "OpenCode application logs",
		".grok/marketplace-cache":             "Grok plugin marketplace git cache",
		".grok/vendor":                        "Grok vendored dependencies cache",
		".grok/logs":                          "Grok application logs",
	}
	for path, reason := range want {
		found := false
		for _, e := range cfg.ExcludePaths {
			if e.Path == path {
				found = true
				if e.Reason != reason {
					t.Fatalf("%q reason = %q, want %q", path, e.Reason, reason)
				}
			}
		}
		if !found {
			t.Fatalf("missing exclude path %q", path)
		}
	}
}

func TestPathReasonForUploadChunks(t *testing.T) {
	rules := MergeExclusions(nil, nil, nil)
	if got := rules.pathReasonFor(".live-and-love/upload-chunks/chunk-1"); got == "" {
		t.Fatal("upload-chunks segment should be excluded")
	}
	if got := rules.pathReasonFor(".live-and-love/upload-chunks"); got == "" {
		t.Fatal("upload-chunks directory should be excluded")
	}
}

func TestShouldSkipLogSuffix(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, home, ".ai-critic/service.log", "log\n")
	writeTestFile(t, home, ".ai-critic/config.json", "{}\n")

	rules := MergeExclusions(nil, nil, nil)
	skipLog, err := shouldSkipPath(home, ".ai-critic/service.log", rules, 0644)
	if err != nil {
		t.Fatal(err)
	}
	if !skipLog {
		t.Fatal("service.log should be excluded by suffix rule")
	}
	skipCfg, err := shouldSkipPath(home, ".ai-critic/config.json", rules, 0644)
	if err != nil {
		t.Fatal(err)
	}
	if skipCfg {
		t.Fatal("config.json should remain included")
	}
}

func TestIncludeOverrideLogAndBinary(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, home, ".ai-critic/keep.log", "keep\n")
	writeTestFile(t, home, ".ai-critic/service.log", "drop\n")
	writeELFStub(t, home, ".ai-critic/bin/stub")

	rules := MergeExclusions(nil, nil, []string{".ai-critic/keep.log", ".ai-critic/bin/stub"})

	skipKeep, err := shouldSkipPath(home, ".ai-critic/keep.log", rules, 0644)
	if err != nil || skipKeep {
		t.Fatalf("keep.log override: skip=%v err=%v", skipKeep, err)
	}
	skipService, err := shouldSkipPath(home, ".ai-critic/service.log", rules, 0644)
	if err != nil || !skipService {
		t.Fatalf("service.log suffix: skip=%v err=%v", skipService, err)
	}
	skipStub, err := shouldSkipPath(home, ".ai-critic/bin/stub", rules, 0755)
	if err != nil || skipStub {
		t.Fatalf("stub override: skip=%v err=%v", skipStub, err)
	}
}

func TestPathPrefixExclusions(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, home, ".codex/.tmp/junk", "tmp\n")
	writeTestFile(t, home, ".local/share/opencode/repos/foo/clone", "clone\n")
	writeTestFile(t, home, ".grok/marketplace-cache/abc/.git/HEAD", "ref\n")
	writeTestFile(t, home, ".grok/vendor/pkg/main.go", "package main\n")
	writeTestFile(t, home, ".grok/logs/app.log", "log\n")

	rules := MergeExclusions(nil, nil, nil)
	for _, rel := range []string{
		".codex/.tmp/junk",
		".local/share/opencode/repos/foo/clone",
		".grok/marketplace-cache/abc/.git/HEAD",
		".grok/vendor/pkg/main.go",
		".grok/logs/app.log",
	} {
		if rules.pathReasonFor(rel) == "" {
			t.Fatalf("expected path exclusion for %q", rel)
		}
	}

	plan, err := BuildPlan(home, nil, nil, GitScanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, absent := range []string{
		".codex/.tmp/junk",
		".local/share/opencode/repos/foo/clone",
		".grok/marketplace-cache/abc/.git/HEAD",
		".grok/vendor/pkg/main.go",
		".grok/logs/app.log",
	} {
		if contains(plan.Included, absent) {
			t.Fatalf("unexpected included %q", absent)
		}
	}
}

func writeTestFile(t *testing.T, home, rel, content string) {
	t.Helper()
	full := filepath.Join(home, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestRevertedExclusionsNotBuiltin(t *testing.T) {
	cfg := BuiltinExclusionConfig()
	for _, removed := range []string{
		".config/git-fetch-skill/data",
		".config/confluence-fetch-skill/data",
		".knowledge-index",
	} {
		for _, e := range cfg.ExcludePaths {
			if e.Path == removed {
				t.Fatalf("builtin config still excludes %q", removed)
			}
		}
	}
}

func TestDiscoverExtendedFixtures(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, home, ".bashrc", "x\n")
	writeTestFile(t, home, ".ai-critic/config.json", "{}\n")
	writeTestFile(t, home, ".ai-critic/service.log", "log\n")
	writeTestFile(t, home, ".ai-critic/keep.log", "keep\n")

	plan, err := BuildPlan(home, nil, nil, GitScanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(plan.Included, ".ai-critic/config.json") {
		t.Fatalf("missing config.json: %v", plan.Included)
	}
	if contains(plan.Included, ".ai-critic/service.log") {
		t.Fatalf("service.log should be excluded: %v", plan.Included)
	}

	plan, err = BuildPlan(home, nil, []string{".ai-critic/keep.log"}, GitScanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(plan.Included, ".ai-critic/keep.log") {
		t.Fatalf("keep.log override failed: %v", plan.Included)
	}
}

func TestMergeUserBackupConfig(t *testing.T) {
	user := &ExclusionConfig{
		Version: exclusionConfigVer,
		ExcludePaths: []ExcludePathEntry{
			{Path: ".knowledge-hub", Reason: "team knowledge cache"},
			{Path: ".knowledge-index", Reason: "knowledge index cache"},
		},
	}
	rules := MergeExclusions(user, nil, nil)
	cfg := rules.EffectiveExclusionConfig()
	found := map[string]bool{}
	for _, e := range cfg.ExcludePaths {
		if e.Path == ".knowledge-hub" {
			if e.Reason != "team knowledge cache" {
				t.Fatalf(".knowledge-hub reason = %q", e.Reason)
			}
			found["hub"] = true
		}
		if e.Path == ".knowledge-index" {
			if e.Reason != "knowledge index cache" {
				t.Fatalf(".knowledge-index reason = %q", e.Reason)
			}
			found["index"] = true
		}
	}
	if !found["hub"] || !found["index"] {
		t.Fatalf("missing user excludes in effective config: %+v", cfg.ExcludePaths)
	}
}

func TestCLIExcludeWinsOverUserConfig(t *testing.T) {
	user := &ExclusionConfig{
		Version: exclusionConfigVer,
		ExcludePaths: []ExcludePathEntry{
			{Path: ".docker", Reason: "user persisted"},
		},
	}
	rules := MergeExclusions(user, nil, []string{".docker"})
	if rules.IsExcluded(".docker") {
		t.Fatal("CLI --include should remove user persisted exclude")
	}
}

func TestUserExcludeWinsOverBuiltinInclude(t *testing.T) {
	user := &ExclusionConfig{
		Version: exclusionConfigVer,
		ExcludePaths: []ExcludePathEntry{
			{Path: ".cache", Reason: "user cache"},
		},
	}
	rules := MergeExclusions(user, nil, nil)
	if !rules.IsExcluded(".cache/data") {
		t.Fatal(".cache should stay excluded from user config")
	}
}

func TestSaveAndLoadUserBackupConfig(t *testing.T) {
	home := t.TempDir()
	entries := []ExcludePathEntry{
		{Path: ".knowledge-hub", Reason: "team knowledge cache"},
	}
	if err := SaveUserBackupConfig(home, entries, ""); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadUserBackupConfig(home)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil || len(cfg.ExcludePaths) != 1 || cfg.ExcludePaths[0].Path != ".knowledge-hub" {
		t.Fatalf("loaded config = %+v", cfg)
	}
	rules, err := ResolveExclusionRules(home, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rules.pathReasonFor(".knowledge-hub/x") == "" {
		t.Fatal("persisted exclude not in effective rules")
	}
}

func TestRejectPersistedExcludeAIcritic(t *testing.T) {
	home := t.TempDir()
	for _, path := range []string{".ai-critic", userBackupConfigRel} {
		err := SaveUserBackupConfig(home, []ExcludePathEntry{{Path: path, Reason: "bad"}}, "")
		if err == nil {
			t.Fatalf("expected error persisting exclude %q", path)
		}
	}
}

func TestMergeUserEmptyReasonShowsFromUserConfig(t *testing.T) {
	user := &ExclusionConfig{
		Version: exclusionConfigVer,
		ExcludePaths: []ExcludePathEntry{
			{Path: ".knowledge-hub"},
		},
	}
	rules := MergeExclusions(user, nil, nil)
	cfg := rules.EffectiveExclusionConfig()
	for _, e := range cfg.ExcludePaths {
		if e.Path == ".knowledge-hub" {
			if e.Reason != fromUserConfigReason {
				t.Fatalf(".knowledge-hub reason = %q, want %q", e.Reason, fromUserConfigReason)
			}
			return
		}
	}
	t.Fatal("missing .knowledge-hub in effective config")
}

func TestExcludePathsFromStringsOmitsReason(t *testing.T) {
	entries := ExcludePathsFromStrings([]string{".knowledge-hub"})
	if len(entries) != 1 {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[0].Reason != "" {
		t.Fatalf("reason = %q, want empty", entries[0].Reason)
	}
}

func TestMergeUserBackupConfigUnionsExcludes(t *testing.T) {
	existing := &ExclusionConfig{
		Version: exclusionConfigVer,
		ExcludePaths: []ExcludePathEntry{
			{Path: ".knowledge-hub"},
		},
		LargeDirThreshold: "50MB",
	}
	merged := MergeUserBackupConfig(existing, ExcludePathsFromStrings([]string{".docker"}), "")
	if len(merged.ExcludePaths) != 2 {
		t.Fatalf("exclude_paths = %+v", merged.ExcludePaths)
	}
	found := map[string]bool{}
	for _, e := range merged.ExcludePaths {
		found[e.Path] = true
	}
	if !found[".knowledge-hub"] || !found[".docker"] {
		t.Fatalf("missing merged excludes: %+v", merged.ExcludePaths)
	}
	if merged.LargeDirThreshold != "50MB" {
		t.Fatalf("threshold = %q, want 50MB", merged.LargeDirThreshold)
	}
}

func TestMergeUserBackupConfigPreservesExcludesOnThresholdOnly(t *testing.T) {
	existing := &ExclusionConfig{
		Version: exclusionConfigVer,
		ExcludePaths: []ExcludePathEntry{
			{Path: ".knowledge-hub"},
		},
	}
	merged := MergeUserBackupConfig(existing, nil, "100MB")
	if len(merged.ExcludePaths) != 1 || merged.ExcludePaths[0].Path != ".knowledge-hub" {
		t.Fatalf("exclude_paths = %+v", merged.ExcludePaths)
	}
	if merged.LargeDirThreshold != "100MB" {
		t.Fatalf("threshold = %q, want 100MB", merged.LargeDirThreshold)
	}
}

func TestSaveUserBackupConfigMergesIncrementalExcludes(t *testing.T) {
	home := t.TempDir()
	if err := SaveUserBackupConfig(home, ExcludePathsFromStrings([]string{".knowledge-hub"}), "50MB"); err != nil {
		t.Fatal(err)
	}
	if err := SaveUserBackupConfig(home, ExcludePathsFromStrings([]string{".docker"}), ""); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadUserBackupConfig(home)
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, e := range cfg.ExcludePaths {
		found[e.Path] = true
	}
	if !found[".knowledge-hub"] || !found[".docker"] {
		t.Fatalf("merged persisted excludes = %+v", cfg.ExcludePaths)
	}
	if cfg.LargeDirThreshold != "50MB" {
		t.Fatalf("threshold = %q, want 50MB", cfg.LargeDirThreshold)
	}
}

func TestSaveUserBackupConfigStoresThreshold(t *testing.T) {
	home := t.TempDir()
	if err := SaveUserBackupConfig(home, nil, "100MB"); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadUserBackupConfig(home)
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil || cfg.LargeDirThreshold != "100MB" {
		t.Fatalf("loaded config = %+v", cfg)
	}
	effective, err := EffectiveExclusionConfigForHome(home)
	if err != nil {
		t.Fatal(err)
	}
	if effective.LargeDirThreshold != "100MB" {
		t.Fatalf("effective threshold = %q, want 100MB", effective.LargeDirThreshold)
	}
}

func writeELFStub(t *testing.T, home, rel string) {
	t.Helper()
	data := make([]byte, 104)
	copy(data, []byte{0x7f, 'E', 'L', 'F', 2, 1, 1, 0})
	data[18] = 0x3e
	full := filepath.Join(home, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, data, 0755); err != nil {
		t.Fatal(err)
	}
}