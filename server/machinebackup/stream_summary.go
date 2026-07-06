package machinebackup

import (
	"fmt"

	"github.com/xhd2015/ai-critic/server/streaming/progress"
)

func emitBackupDryRunSummary(pw *progress.Writer, plan *MachineBackupPlan) error {
	lines := formatBackupDryRunSummary(plan)
	for _, line := range lines {
		if err := pw.EmitLog(line, true); err != nil {
			return err
		}
	}
	return nil
}

func formatBackupDryRunSummary(plan *MachineBackupPlan) []string {
	if plan == nil {
		return nil
	}
	lines := []string{
		"",
		"dry-run: machine backup plan",
		"",
		fmt.Sprintf("  home:         %s", plan.Home),
		fmt.Sprintf("  DOT FILES (%d files, %s)",
			plan.DotFilesTotal.Files+plan.DotFilesTotal.Symlinks,
			formatSize(plan.DotFilesTotal.Bytes)),
	}
	for _, f := range plan.DotFiles {
		lines = append(lines, fmt.Sprintf("    %-24s %s", f.Path, formatSize(f.Bytes)))
	}
	lines = append(lines,
		fmt.Sprintf("  DOT DIRS (%d dirs, %d files, %s)",
			len(plan.DirStats), plan.DotDirsTotal.Files, formatSize(plan.DotDirsTotal.Bytes)),
		fmt.Sprintf("    %-20s %8s %8s", "DIR", "FILES", "SIZE"),
	)
	for _, st := range plan.DirStats {
		lines = append(lines,
			fmt.Sprintf("    %-20s %8d %8s", st.Path, st.Files, formatSize(st.Bytes)))
	}
	lines = append(lines, fmt.Sprintf("  EXCLUDED (%d paths)", len(plan.Excluded)))
	for _, ex := range plan.Excluded {
		if ex.Reason != "" {
			lines = append(lines, fmt.Sprintf("    %-24s %s", ex.Path, ex.Reason))
		} else {
			lines = append(lines, fmt.Sprintf("    %s", ex.Path))
		}
	}
	lines = append(lines,
		fmt.Sprintf("  TOTAL: %d files, %d symlinks, %s",
			plan.GrandTotal.Files, plan.GrandTotal.Symlinks, formatSize(plan.GrandTotal.Bytes)),
	)
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