package machinebackup

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"

	"github.com/xhd2015/ai-critic/server/streaming/progress"
)

// BackupStreamOptions configures backup SSE streaming.
type BackupStreamOptions struct {
	LargeDirThresholdBytes int64
	GitOpts                GitScanOptions
	WriteArchive           bool
}

// BackupPlanStream walks home and emits SSE progress then a done plan summary.
func BackupPlanStream(w http.ResponseWriter, home string, exclude, include []string, largeDirThresholdBytes int64, gitOpts GitScanOptions) error {
	return BackupStream(w, home, exclude, include, BackupStreamOptions{
		LargeDirThresholdBytes: largeDirThresholdBytes,
		GitOpts:                gitOpts,
	})
}

// BackupStream emits backup plan progress and optionally packs an archive after the summary.
func BackupStream(w http.ResponseWriter, home string, exclude, include []string, opts BackupStreamOptions) error {
	pw := progress.NewWriter(w)
	if pw == nil {
		return fmt.Errorf("streaming not supported")
	}

	prepared, err := prepareBackup(home, exclude, include, opts.GitOpts)
	if err != nil {
		if emitErr := pw.EmitError(err.Error()); emitErr != nil {
			return emitErr
		}
		return nil
	}
	plan := prepared.Plan
	rules := prepared.Rules

	if err := emitBackupPlanProgress(pw, plan); err != nil {
		return err
	}

	if err := emitBackupDryRunSummary(pw, plan, DryRunSummaryOptions{
		LargeDirThresholdBytes: opts.LargeDirThresholdBytes,
		ExclusionRules:         rules,
		SkipGitDirsScan:        opts.GitOpts.SkipGitDirsScan,
	}); err != nil {
		return err
	}

	done := backupStreamDone(plan)
	if opts.WriteArchive {
		token, archiveBytes, err := packBackupArchive(pw, prepared)
		if err != nil {
			if emitErr := pw.EmitError(err.Error()); emitErr != nil {
				return emitErr
			}
			return nil
		}
		done["archive_token"] = token
		done["archive_bytes"] = archiveBytes
	}
	return pw.EmitDone(done)
}

func emitBackupPlanProgress(pw *progress.Writer, plan *MachineBackupPlan) error {
	if err := pw.EmitSection("DOT FILES"); err != nil {
		return err
	}
	files := plan.AllFiles
	if len(files) == 0 {
		files = plan.DotFiles
	}
	for _, f := range files {
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
	if err := pw.EmitProgress(progress.Item{
		Layer:  "excluded_header",
		Detail: formatExcludedColumnHeader(),
	}); err != nil {
		return err
	}
	for _, ex := range plan.Excluded {
		if err := pw.EmitProgress(progress.Item{
			Layer:  "excluded",
			Detail: formatExcludedRuleRow(ex),
		}); err != nil {
			return err
		}
	}
	return nil
}

func packBackupArchive(pw *progress.Writer, prepared *backupPrepared) (token string, archiveBytes int64, err error) {
	if err := pw.EmitSection("PACKING"); err != nil {
		return "", 0, err
	}
	tmp, err := os.CreateTemp("", "machine-backup-*.tar.xz")
	if err != nil {
		return "", 0, fmt.Errorf("create temp archive: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { os.Remove(tmpPath) }
	defer func() {
		if err != nil {
			cleanup()
		}
	}()

	onPack := func(name, detail string) error {
		return pw.EmitProgress(progress.Item{
			Layer:  "pack",
			Name:   name,
			Detail: detail,
		})
	}
	if err := writeArchiveFromWalk(tmp, prepared.Home, prepared.Rules, prepared.Walk, prepared.GitRepos, prepared.GitSkipped, onPack); err != nil {
		tmp.Close()
		return "", 0, err
	}
	if err := tmp.Close(); err != nil {
		return "", 0, fmt.Errorf("close temp archive: %w", err)
	}
	info, err := os.Stat(tmpPath)
	if err != nil {
		return "", 0, fmt.Errorf("stat temp archive: %w", err)
	}
	token, err = registerArchiveSession(tmpPath)
	if err != nil {
		return "", 0, err
	}
	return token, info.Size(), nil
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
		"git_repos":       plan.GitRepos,
	}
}

// RestorePlanStream compares archive entries against home and emits per-entry progress.
func RestorePlanStream(w http.ResponseWriter, home string, archive io.Reader, exclude, include []string, dryRun bool) error {
	pw := progress.NewWriter(w)
	if pw == nil {
		return fmt.Errorf("streaming not supported")
	}

	summary, err := restoreStreaming(home, archive, exclude, include, dryRun, restoreStreamEmit{
		section: pw.EmitSection,
		progress: func(entry RestoreEntry) error {
			return pw.EmitProgress(restoreEntryToItem(entry))
		},
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

type restoreStreamEmit struct {
	section  func(string) error
	progress func(RestoreEntry) error
}

func restoreStreaming(home string, archive io.Reader, exclude, include []string, dryRun bool, emit restoreStreamEmit) (*MachineRestoreSummary, error) {
	home, err := resolveHome(home)
	if err != nil {
		return nil, err
	}
	rules, err := ResolveExclusionRules(home, exclude, include)
	if err != nil {
		return nil, err
	}

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

	if emit.section != nil {
		if err := emit.section("CLASSIFYING"); err != nil {
			return nil, err
		}
	}

	classifyItems := pending
	if dryRun && summary.SkipIdentical == summary.TotalEntries && len(classifyItems) > 0 {
		classifyItems = classifyItems[:1]
	}
	if emit.progress != nil {
		for _, item := range classifyItems {
			if err := emit.progress(item.entry); err != nil {
				return nil, err
			}
		}
	}

	if !dryRun {
		if emit.section != nil {
			if err := emit.section("APPLYING"); err != nil {
				return nil, err
			}
		}
		for _, item := range pending {
			if item.entry.Action == "skip" {
				continue
			}
			if err := applyEntry(home, item.entry.Path, item.ent); err != nil {
				return nil, err
			}
			if emit.progress != nil {
				if err := emit.progress(item.entry); err != nil {
					return nil, err
				}
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