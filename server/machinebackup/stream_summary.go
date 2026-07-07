package machinebackup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xhd2015/ai-critic/server/streaming/progress"
)

const largeDirDetailMinBytes = 10 * 1024 * 1024

type dirSizeEntry struct {
	Path  string
	Bytes int64
}

// DryRunSummaryOptions configures backup dry-run summary formatting.
type DryRunSummaryOptions struct {
	LargeDirThresholdBytes int64
	ExclusionRules         ExclusionRules
	SkipGitDirsScan        bool
}

func emitBackupDryRunSummary(pw *progress.Writer, plan *MachineBackupPlan, opts DryRunSummaryOptions) error {
	lines := formatBackupDryRunSummary(plan, opts)
	for _, line := range lines {
		if err := pw.EmitLog(line, true); err != nil {
			return err
		}
	}
	return nil
}

func formatBackupDryRunSummary(plan *MachineBackupPlan, opts DryRunSummaryOptions) []string {
	if plan == nil {
		return nil
	}
	threshold := EffectiveLargeDirThreshold(opts.LargeDirThresholdBytes)

	dotFiles := sortFileStatsBySizeDesc(plan.DotFiles)
	dirStats := sortDirStatsBySizeDesc(plan.DirStats)

	lines := []string{
		"",
		"dry-run: machine backup plan",
		"",
		fmt.Sprintf("  home:         %s", plan.Home),
		fmt.Sprintf("  DOT FILES (%d files, %s)",
			plan.DotFilesTotal.Files+plan.DotFilesTotal.Symlinks,
			formatSize(plan.DotFilesTotal.Bytes)),
	}
	for _, f := range dotFiles {
		lines = append(lines, fmt.Sprintf("    %-24s %s", f.Path, formatSize(f.Bytes)))
	}
	lines = append(lines,
		fmt.Sprintf("  DOT DIRS (%d dirs, %d files, %s)",
			len(plan.DirStats), plan.DotDirsTotal.Files, formatSize(plan.DotDirsTotal.Bytes)),
		fmt.Sprintf("    %-20s %8s %8s", "DIR", "FILES", "SIZE"),
	)
	var largeDirs []DirStat
	for _, st := range dirStats {
		row := fmt.Sprintf("    %-20s %8d %8s", st.Path, st.Files, formatSize(st.Bytes))
		if st.Bytes > threshold {
			row += "  LARGE SIZE"
			largeDirs = append(largeDirs, st)
		}
		lines = append(lines, row)
	}
	detailEntries := collectLargeIncludedDirs(plan.Home, opts.ExclusionRules, largeDirDetailMinBytes)
	if len(detailEntries) > 0 {
		lines = append(lines, "")
		lines = append(lines, "  LARGE DIR DETAIL:")
		lines = append(lines, formatLargeDirDetailFlat(detailEntries)...)
	}
	paths, files, bytes := excludedTotals(plan.Excluded)
	lines = append(lines, formatExcludedSectionHeader(paths, files, bytes))
	lines = append(lines, formatExcludedColumnHeader())
	for _, ex := range plan.Excluded {
		lines = append(lines, formatExcludedRuleRow(ex))
	}
	lines = append(lines, formatGitReposSummaryLines(plan.GitRepos, opts.SkipGitDirsScan)...)
	lines = append(lines,
		fmt.Sprintf("  TOTAL: %d files, %d symlinks, %s",
			plan.GrandTotal.Files, plan.GrandTotal.Symlinks, formatSize(plan.GrandTotal.Bytes)),
	)
	return lines
}

func sortFileStatsBySizeDesc(files []FileStat) []FileStat {
	out := append([]FileStat(nil), files...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Bytes != out[j].Bytes {
			return out[i].Bytes > out[j].Bytes
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func sortDirStatsBySizeDesc(stats []DirStat) []DirStat {
	out := append([]DirStat(nil), stats...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Bytes != out[j].Bytes {
			return out[i].Bytes > out[j].Bytes
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func collectLargeIncludedDirs(home string, rules ExclusionRules, minBytes int64) []dirSizeEntry {
	entries, err := os.ReadDir(home)
	if err != nil {
		return nil
	}

	var result []dirSizeEntry
	for _, ent := range entries {
		name := ent.Name()
		if !strings.HasPrefix(name, ".") || name == "." || name == ".." {
			continue
		}
		rel := normalizeRelPath(name)
		full := filepath.Join(home, name)
		info, err := os.Lstat(full)
		if err != nil {
			continue
		}
		skip, err := shouldSkipPath(home, rel, rules, info.Mode())
		if err != nil || skip {
			continue
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			continue
		case info.IsDir():
			collectLargeIncludedDirTree(home, rel, rules, minBytes, &result)
		default:
			if size := info.Size(); size >= minBytes {
				result = append(result, dirSizeEntry{Path: rel, Bytes: size})
			}
		}
	}
	return sortDirSizeEntries(result)
}

func collectLargeIncludedDirTree(home, rel string, rules ExclusionRules, minBytes int64, result *[]dirSizeEntry) int64 {
	full := filepath.Join(home, filepath.FromSlash(rel))
	dirEntries, err := os.ReadDir(full)
	if err != nil {
		return 0
	}

	var total int64
	for _, ent := range dirEntries {
		childRel := normalizeRelPath(rel + "/" + ent.Name())
		childFull := filepath.Join(home, filepath.FromSlash(childRel))
		info, err := os.Lstat(childFull)
		if err != nil {
			continue
		}
		skip, err := shouldSkipPath(home, childRel, rules, info.Mode())
		if err != nil || skip {
			continue
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			continue
		case info.IsDir():
			total += collectLargeIncludedDirTree(home, childRel, rules, minBytes, result)
		default:
			size := info.Size()
			total += size
			if size >= minBytes {
				*result = append(*result, dirSizeEntry{Path: childRel, Bytes: size})
			}
		}
	}
	if total >= minBytes {
		*result = append(*result, dirSizeEntry{Path: rel, Bytes: total})
	}
	return total
}

func sortDirSizeEntries(entries []dirSizeEntry) []dirSizeEntry {
	out := append([]dirSizeEntry(nil), entries...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Bytes != out[j].Bytes {
			return out[i].Bytes > out[j].Bytes
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func formatLargeDirDetailFlat(entries []dirSizeEntry) []string {
	sorted := sortDirSizeEntries(entries)
	lines := make([]string, 0, len(sorted))
	for _, e := range sorted {
		lines = append(lines, fmt.Sprintf("  > %s  %s", e.Path, formatSize(e.Bytes)))
	}
	return lines
}

func emitRestoreDryRunSummary(pw *progress.Writer, summary *MachineRestoreSummary, dryRun bool) error {
	lines := formatRestoreSummary(summary, dryRun)
	for _, line := range lines {
		if err := pw.EmitLog(line, true); err != nil {
			return err
		}
	}
	return nil
}

func formatRestoreSummary(summary *MachineRestoreSummary, dryRun bool) []string {
	if summary == nil {
		return nil
	}
	title := "machine restore summary"
	if dryRun {
		title = "dry-run: machine restore plan"
	}
	return []string{
		title,
		"",
		fmt.Sprintf("  home:         %s", summary.Home),
		fmt.Sprintf("  skip (identical):  %d", summary.SkipIdentical),
		fmt.Sprintf("  update:            %d", summary.Update),
		fmt.Sprintf("  create:            %d", summary.Create),
		fmt.Sprintf("  TOTAL: %d entries", summary.TotalEntries),
	}
}