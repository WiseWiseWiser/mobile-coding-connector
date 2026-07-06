package machinebackup

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/xhd2015/ai-critic/server/streaming/progress"
)

// BackupPlanStream walks home and emits SSE progress then a done plan summary.
func BackupPlanStream(w http.ResponseWriter, home string, exclude, include []string) error {
	pw := progress.NewWriter(w)
	if pw == nil {
		return fmt.Errorf("streaming not supported")
	}

	plan, err := BuildPlan(home, exclude, include)
	if err != nil {
		if emitErr := pw.EmitError(err.Error()); emitErr != nil {
			return emitErr
		}
		return nil
	}

	if err := pw.EmitSection("DOT FILES"); err != nil {
		return err
	}
	for _, f := range plan.DotFiles {
		if err := pw.EmitProgress(progress.Item{
			Layer:  "dot_file",
			Name:   f.Path,
			Detail: formatSize(f.Bytes),
		}); err != nil {
			return err
		}
	}

	if err := pw.EmitSection("DOT DIRS"); err != nil {
		return err
	}
	for _, st := range plan.DirStats {
		if err := pw.EmitProgress(progress.Item{
			Layer: "dir",
			Name:  st.Path,
			Detail: fmt.Sprintf("files=%d dirs=%d symlinks=%d    %s",
				st.Files, st.Dirs, st.Symlinks, formatSize(st.Bytes)),
		}); err != nil {
			return err
		}
	}

	if err := pw.EmitSection("EXCLUDED"); err != nil {
		return err
	}
	for _, ex := range plan.Excluded {
		if err := pw.EmitProgress(progress.Item{
			Layer:  "excluded",
			Name:   ex.Path,
			Detail: ex.Reason,
		}); err != nil {
			return err
		}
	}

	if err := emitBackupDryRunSummary(pw, plan); err != nil {
		return err
	}
	return pw.EmitDone(backupStreamDone(plan))
}

// backupStreamDone is the SSE done payload. It omits the full included path list
// (can be hundreds of thousands of entries) so the terminal data: line stays bounded.
func backupStreamDone(plan *MachineBackupPlan) map[string]any {
	return map[string]any{
		"home":            plan.Home,
		"dot_files":       plan.DotFiles,
		"dot_files_total": plan.DotFilesTotal,
		"dir_stats":       plan.DirStats,
		"dot_dirs_total":  plan.DotDirsTotal,
		"grand_total":     plan.GrandTotal,
		"excluded":        plan.Excluded,
		"included_count":  len(plan.Included),
	}
}

// RestorePlanStream compares archive entries against home and emits per-entry progress.
func RestorePlanStream(w http.ResponseWriter, home string, archive io.Reader, exclude, include []string, dryRun bool) error {
	pw := progress.NewWriter(w)
	if pw == nil {
		return fmt.Errorf("streaming not supported")
	}

	summary, err := restoreStreaming(home, archive, exclude, include, dryRun, func(entry RestoreEntry) error {
		return pw.EmitProgress(restoreEntryToItem(entry))
	})
	if err != nil {
		if emitErr := pw.EmitError(err.Error()); emitErr != nil {
			return emitErr
		}
		return nil
	}
	if err := emitRestoreDryRunSummary(pw, summary, dryRun); err != nil {
		return err
	}
	return pw.EmitDone(restoreSummaryToDone(summary))
}

func restoreStreaming(home string, archive io.Reader, exclude, include []string, dryRun bool, emit func(RestoreEntry) error) (*MachineRestoreSummary, error) {
	home, err := resolveHome(home)
	if err != nil {
		return nil, err
	}
	rules := MergeExclusions(exclude, include)

	raw, err := io.ReadAll(archive)
	if err != nil {
		return nil, fmt.Errorf("read archive: %w", err)
	}
	_, entries, err := ReadArchive(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}

	summary := &MachineRestoreSummary{Home: home}
	type pendingRestore struct {
		entry RestoreEntry
		ent   archiveEntry
	}
	var pending []pendingRestore

	for _, ent := range entries {
		rel := normalizeRelPath(ent.Header.Name)
		if rel == "" {
			continue
		}
		target, skip := resolveRestoreTarget(rel)
		if skip {
			continue
		}
		if rules.IsExcluded(target) {
			continue
		}
		action, err := classifyEntry(home, target, ent)
		if err != nil {
			return nil, err
		}
		entry := RestoreEntry{Path: target, Action: action}
		if action == "skip" {
			summary.SkipIdentical++
		} else if action == "update" {
			summary.Update++
		} else if action == "create" {
			summary.Create++
		}
		summary.TotalEntries++
		pending = append(pending, pendingRestore{entry: entry, ent: ent})
	}

	sort.Slice(pending, func(i, j int) bool {
		ai := countSlashes(pending[i].entry.Path)
		aj := countSlashes(pending[j].entry.Path)
		if ai != aj {
			return ai < aj
		}
		return pending[i].entry.Path < pending[j].entry.Path
	})

	streamItems := pending
	if dryRun && summary.SkipIdentical == summary.TotalEntries && len(streamItems) > 0 {
		streamItems = streamItems[:1]
	}

	if emit != nil {
		for _, item := range streamItems {
			if err := emit(item.entry); err != nil {
				return nil, err
			}
		}
	}

	if !dryRun {
		for _, item := range pending {
			if item.entry.Action == "skip" {
				continue
			}
			if err := applyEntry(home, item.entry.Path, item.ent); err != nil {
				return nil, err
			}
		}
	}
	return summary, nil
}

func restoreEntryToItem(entry RestoreEntry) progress.Item {
	return progress.Item{
		Layer:  "restore",
		Name:   entry.Path,
		Status: entry.Action,
	}
}

func restoreSummaryToDone(summary *MachineRestoreSummary) map[string]any {
	return map[string]any{
		"home":           summary.Home,
		"skip_identical": summary.SkipIdentical,
		"update":         summary.Update,
		"create":         summary.Create,
		"total_entries":  summary.TotalEntries,
	}
}

// SortedRestoreEntries returns restore entries in stable display order.
func SortedRestoreEntries(entries []RestoreEntry) []RestoreEntry {
	out := append([]RestoreEntry(nil), entries...)
	sort.Slice(out, func(i, j int) bool {
		ai := countSlashes(out[i].Path)
		aj := countSlashes(out[j].Path)
		if ai != aj {
			return ai < aj
		}
		return out[i].Path < out[j].Path
	})
	return out
}

func countSlashes(path string) int {
	n := 0
	for _, c := range path {
		if c == '/' {
			n++
		}
	}
	return n
}

func formatSize(n int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case n >= gb:
		return fmt.Sprintf("%.2f GB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.2f MB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.2f KB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%d B", n)
	}
}