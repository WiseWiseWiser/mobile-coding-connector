package machinebackup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatBackupDryRunSummary(t *testing.T) {
	plan := &MachineBackupPlan{
		Home: "/home/test",
		DotFiles: []FileStat{
			{Path: ".bashrc", Bytes: 3200},
		},
		DotFilesTotal: SectionTotals{Files: 1, Bytes: 3200},
		DirStats: []DirStat{
			{Path: ".ai-critic", Files: 2, Bytes: 8100000},
		},
		DotDirsTotal: SectionTotals{Files: 2, Bytes: 8100000},
		GrandTotal:   SectionTotals{Files: 3, Bytes: 8103200},
		Excluded: []ExcludePathEntry{{
			Path: ".cache", Reason: "temporary application cache", Files: 2, Bytes: 1536,
		}},
	}
	lines := formatBackupDryRunSummary(plan, DryRunSummaryOptions{})
	text := strings.Join(lines, "\n")
	if !strings.Contains(text, "dry-run: machine backup plan") {
		t.Fatalf("missing title: %s", text)
	}
	if !strings.Contains(text, ".bashrc") || !strings.Contains(text, "KB") {
		t.Fatalf("missing dot file size: %s", text)
	}
	if !strings.Contains(text, "TOTAL:") {
		t.Fatalf("missing TOTAL: %s", text)
	}
	if !strings.Contains(text, "EXCLUDED (1 paths, 2 files,") {
		t.Fatalf("missing EXCLUDED aggregate header: %s", text)
	}
	if !strings.Contains(text, "RULE") || !strings.Contains(text, "FILES") {
		t.Fatalf("missing EXCLUDED table headers: %s", text)
	}
	if !strings.Contains(text, "INSTALLED SOFTWARE(.backup/installed.json):") {
		t.Fatalf("missing INSTALLED SOFTWARE section: %s", text)
	}
	if !strings.Contains(text, "ENV(.backup/ENV):") {
		t.Fatalf("missing ENV section: %s", text)
	}
	gitIdx := strings.Index(text, "GIT REPOS")
	installedIdx := strings.Index(text, "INSTALLED SOFTWARE")
	envIdx := strings.Index(text, "ENV(.backup/ENV):")
	totalIdx := strings.Index(text, "TOTAL:")
	if gitIdx < 0 || installedIdx < 0 || envIdx < 0 || totalIdx < 0 {
		t.Fatalf("missing meta section markers: %s", text)
	}
	if !(gitIdx < installedIdx && installedIdx < envIdx && envIdx < totalIdx) {
		t.Fatalf("meta sections out of order: git=%d installed=%d env=%d total=%d", gitIdx, installedIdx, envIdx, totalIdx)
	}
}

func TestBackupStreamDoneOmitsIncludedPaths(t *testing.T) {
	plan := &MachineBackupPlan{
		Home:     "/home/test",
		Included: []string{".bashrc", ".ai-critic/foo"},
		GrandTotal: SectionTotals{Files: 2},
	}
	done := backupStreamDone(plan)
	if _, ok := done["included"]; ok {
		t.Fatalf("stream done must not include full path list")
	}
	if done["included_count"] != 2 {
		t.Fatalf("included_count = %v, want 2", done["included_count"])
	}
}

func TestCollectLargeIncludedDirsFlatSorted(t *testing.T) {
	home := t.TempDir()
	rules, err := ResolveExclusionRules(home, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	writeLargeDirTestFile(t, home, ".big-test/child-a", 30*1024*1024)
	writeLargeDirTestFile(t, home, ".big-test/child-b", 20*1024*1024)
	writeLargeDirTestFile(t, home, ".deep-test/nested-big/file", 12*1024*1024)
	writeLargeDirTestFile(t, home, ".deep-test/small/tiny", 1024)
	writeLargeDirTestFile(t, home, ".cache/blob", 15*1024*1024)

	entries := collectLargeIncludedDirs(home, rules, largeDirDetailMinBytes)
	if len(entries) == 0 {
		t.Fatal("expected large included dirs")
	}

	paths := make(map[string]int64, len(entries))
	for _, e := range entries {
		paths[e.Path] = e.Bytes
	}
	for _, want := range []string{".big-test", ".big-test/child-a", ".big-test/child-b", ".deep-test", ".deep-test/nested-big"} {
		if _, ok := paths[want]; !ok {
			t.Fatalf("missing path %q in %v", want, entries)
		}
	}
	for _, absent := range []string{".deep-test/small", ".cache", ".cache/blob"} {
		if _, ok := paths[absent]; ok {
			t.Fatalf("unexpected path %q in %v", absent, entries)
		}
	}

	lines := formatLargeDirDetailFlat(entries)
	if len(lines) != len(entries) {
		t.Fatalf("line count = %d, want %d", len(lines), len(entries))
	}
	if !strings.HasPrefix(lines[0], "  > .big-test  ") {
		t.Fatalf("first line want .big-test largest, got %q", lines[0])
	}
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Bytes < entries[i].Bytes {
			t.Fatalf("not sorted by size desc: %v", entries)
		}
		if entries[i-1].Bytes == entries[i].Bytes && entries[i-1].Path > entries[i].Path {
			t.Fatalf("tiebreak not path asc: %v", entries)
		}
	}
}

func TestFormatBackupDryRunSummaryLargeDirDetailIndependentThreshold(t *testing.T) {
	home := t.TempDir()
	rules, err := ResolveExclusionRules(home, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	writeLargeDirTestFile(t, home, ".big-test/child-a", 30*1024*1024)
	writeLargeDirTestFile(t, home, ".big-test/child-b", 20*1024*1024)

	plan, err := BuildPlan(home, nil, nil, GitScanOptions{})
	if err != nil {
		t.Fatal(err)
	}

	lines := formatBackupDryRunSummary(plan, DryRunSummaryOptions{
		LargeDirThresholdBytes: 100 * 1024 * 1024,
		ExclusionRules:         rules,
	})
	text := strings.Join(lines, "\n")
	if strings.Contains(text, "LARGE SIZE") {
		t.Fatalf("unexpected LARGE SIZE with raised threshold:\n%s", text)
	}
	for _, want := range []string{"> .big-test  ", "> .big-test/child-a  ", "> .big-test/child-b  "} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing detail row %q in:\n%s", want, text)
		}
	}
}

func writeLargeDirTestFile(t *testing.T, home, rel string, size int) {
	t.Helper()
	full := filepath.Join(home, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, size)
	if err := os.WriteFile(full, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestFormatRestoreSummary(t *testing.T) {
	lines := formatRestoreSummary(&MachineRestoreSummary{
		Home:          "/home/test",
		SkipIdentical: 3,
		Update:        1,
		Create:        0,
		TotalEntries:  4,
	}, true)
	text := strings.Join(lines, "\n")
	if !strings.Contains(text, "dry-run: machine restore plan") {
		t.Fatalf("missing title: %s", text)
	}
	if !strings.Contains(text, "TOTAL: 4 entries") {
		t.Fatalf("missing TOTAL: %s", text)
	}
}