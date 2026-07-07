package machinebackup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExcludedStatsAttribution(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, home, ".grok/logs/app.log", strings.Repeat("x", 256))

	plan, err := BuildPlan(home, nil, nil, GitScanOptions{})
	if err != nil {
		t.Fatal(err)
	}

	var grokLogs *ExcludePathEntry
	var logRule *ExcludePathEntry
	for i := range plan.Excluded {
		switch plan.Excluded[i].Path {
		case ".grok/logs":
			grokLogs = &plan.Excluded[i]
		case logSuffixRule:
			logRule = &plan.Excluded[i]
		}
	}
	if grokLogs == nil {
		t.Fatalf("missing .grok/logs entry: %v", plan.Excluded)
	}
	if grokLogs.Files < 1 {
		t.Fatalf(".grok/logs files = %d, want >= 1", grokLogs.Files)
	}
	if grokLogs.Bytes < 256 {
		t.Fatalf(".grok/logs bytes = %d, want >= 256", grokLogs.Bytes)
	}
	if logRule != nil && logRule.Files > 0 {
		t.Fatalf("log suffix rule should not claim .grok/logs/app.log: %+v", logRule)
	}
}

func TestExcludedStatsSort(t *testing.T) {
	entries := []ExcludePathEntry{
		{Path: "**/*.log", Reason: "log files", Files: 1, Bytes: 512},
		{Path: ".cache", Reason: "temporary application cache", Files: 2, Bytes: 1536},
		{Path: ".npm", Reason: "npm cache", Files: 1, Bytes: 1536},
	}
	stats := excludedStats{
		".cache":       {files: 2, bytes: 1536},
		".npm":         {files: 1, bytes: 1536},
		logSuffixRule:  {files: 1, bytes: 512},
	}
	rules := ExclusionRules{
		ExcludedList: entries,
		reasons: map[string]string{
			".cache":        "temporary application cache",
			".npm":          "npm cache",
			logSuffixRule:   "log files",
		},
	}
	got := populateExcludedList(rules, stats)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].Path != ".cache" || got[1].Path != ".npm" || got[2].Path != logSuffixRule {
		t.Fatalf("sort order = %#v, want .cache, .npm, %s", got, logSuffixRule)
	}
}

func TestFormatExcludedSection(t *testing.T) {
	plan := &MachineBackupPlan{
		Excluded: []ExcludePathEntry{
			{Path: ".cache", Reason: "temporary application cache", Files: 2, Bytes: 1536},
			{Path: logSuffixRule, Reason: "log files", Files: 1, Bytes: 512},
		},
	}
	lines := formatBackupDryRunSummary(plan, DryRunSummaryOptions{})
	text := strings.Join(lines, "\n")

	paths, files, bytes := excludedTotals(plan.Excluded)
	header := formatExcludedSectionHeader(paths, files, bytes)
	if !strings.Contains(text, header) {
		t.Fatalf("missing header %q in:\n%s", header, text)
	}
	if !strings.Contains(text, "RULE") || !strings.Contains(text, "FILES") {
		t.Fatalf("missing column headers in:\n%s", text)
	}
	if files != 3 || bytes != 2048 {
		t.Fatalf("totals files=%d bytes=%d, want 3 and 2048", files, bytes)
	}
}

func TestExcludedStatsFromWalk(t *testing.T) {
	home := t.TempDir()
	writeTestFile(t, home, ".cache/junk", strings.Repeat("a", 1024))
	writeTestFile(t, home, ".cache/nested/deep", strings.Repeat("b", 512))
	writeTestFile(t, home, ".ai-critic/service.log", strings.Repeat("c", 512))
	writeTestFile(t, home, ".bashrc", "ok\n")

	plan, err := BuildPlan(home, nil, nil, GitScanOptions{})
	if err != nil {
		t.Fatal(err)
	}

	cache := findExcludedEntry(t, plan.Excluded, ".cache")
	if cache.Files < 2 {
		t.Fatalf(".cache files = %d, want >= 2", cache.Files)
	}
	if cache.Bytes < 1024+512 {
		t.Fatalf(".cache bytes = %d, want >= 1536", cache.Bytes)
	}

	logRule := findExcludedEntry(t, plan.Excluded, logSuffixRule)
	if logRule.Files < 1 {
		t.Fatalf("%s files = %d, want >= 1", logSuffixRule, logRule.Files)
	}

	cacheIdx := excludedEntryIndex(plan.Excluded, ".cache")
	logIdx := excludedEntryIndex(plan.Excluded, logSuffixRule)
	if cacheIdx < 0 || logIdx < 0 || cacheIdx >= logIdx {
		t.Fatalf("expected .cache before %s; indices %d vs %d", logSuffixRule, cacheIdx, logIdx)
	}
}

func findExcludedEntry(t *testing.T, entries []ExcludePathEntry, path string) ExcludePathEntry {
	t.Helper()
	for _, e := range entries {
		if e.Path == path {
			return e
		}
	}
	t.Fatalf("missing excluded entry %q in %#v", path, entries)
	return ExcludePathEntry{}
}

func excludedEntryIndex(entries []ExcludePathEntry, path string) int {
	for i, e := range entries {
		if e.Path == path {
			return i
		}
	}
	return -1
}

func TestExcludedStatsSkipsSymlinks(t *testing.T) {
	home := t.TempDir()
	target := filepath.Join(home, ".cache", "target")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".cache", "link")
	if err := os.Symlink("target", link); err != nil {
		t.Fatal(err)
	}

	plan, err := BuildPlan(home, nil, nil, GitScanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	cache := findExcludedEntry(t, plan.Excluded, ".cache")
	if cache.Files != 1 {
		t.Fatalf(".cache files = %d, want 1 (symlink not counted)", cache.Files)
	}
}