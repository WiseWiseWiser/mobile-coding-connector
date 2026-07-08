package machinebackup

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	gitReposDryRunHeader      = "  GIT REPOS(.backup/git-repo-worktrees.json):"
	installedDryRunHeader     = "  INSTALLED SOFTWARE(.backup/installed.json):"
	envDryRunHeader             = "  ENV(.backup/ENV):"
	gitReposTableColumnHeader   = "    KIND      PATH                    BRANCH        SHA       STATUS              ORIGIN                                    MESSAGE"
	installedTableColumnHeader  = "    NAME                 VERSION    PATH"
)

// appendMetaSectionTerminator adds a blank line so harness section parsers that
// slice at the next meta header receive extracted text ending with '\n'.
func appendMetaSectionTerminator(lines []string) []string {
	return append(lines, "")
}

func formatInstalledSoftwareSummaryLines() []string {
	data, err := buildInstalledToolsSnapshotFn()
	if err != nil {
		return []string{installedDryRunHeader + " (none)"}
	}
	var snap InstalledToolsSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return []string{installedDryRunHeader + " (none)"}
	}
	if len(snap.Tools) == 0 {
		return []string{installedDryRunHeader + " (none)"}
	}

	tools := append([]InstalledToolRef(nil), snap.Tools...)
	sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })

	lines := []string{
		installedDryRunHeader,
		fmt.Sprintf("    captured_at: %s  (%d tools)", formatMetaCapturedAt(snap.CapturedAt), len(tools)),
		installedTableColumnHeader,
	}
	for _, tool := range tools {
		lines = append(lines, fmt.Sprintf("    %-20s %-10s %s", tool.Name, tool.Version, tool.Path))
	}
	return appendMetaSectionTerminator(lines)
}

func formatEnvSummaryLines() []string {
	data := buildEnvSnapshot()
	lines := []string{envDryRunHeader}
	raw := strings.TrimSuffix(string(data), "\n")
	if raw == "" {
		return appendMetaSectionTerminator(lines)
	}
	for _, line := range strings.Split(raw, "\n") {
		if line == "" {
			continue
		}
		lines = append(lines, "    "+line)
	}
	return appendMetaSectionTerminator(lines)
}

func formatMetaCapturedAt(t time.Time) string {
	return t.UTC().Format(time.RFC3339)
}