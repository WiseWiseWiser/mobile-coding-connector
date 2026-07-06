package machinebackup

import (
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
		Excluded:     []ExcludePathEntry{{Path: ".cache", Reason: "temporary application cache"}},
	}
	lines := formatBackupDryRunSummary(plan)
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