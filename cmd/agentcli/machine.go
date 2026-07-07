package agentcli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/ai-critic/cmd/agentcli/streamcmd"
	"github.com/xhd2015/ai-critic/server/machinebackup"
	"github.com/xhd2015/less-gen/flags"
	"golang.org/x/term"
)

const machineHelp = `Usage: remote-agent machine <subcommand> [args...]

Backup and restore the server user's home directory dot-files and dot-directories.

Subcommands:
  analyse-files
      Scan server HOME; stream per-entry blocks and a summary.

  backup [--output PATH] [--dry-run] [--show-config] [--set-config] [--exclude PATH]... [--include PATH]...
      Snapshot server HOME as a streamed tar.xz archive.

  restore [backup.tar.xz] [--dry-run] [--show-config] [--show-meta] [--exclude PATH]... [--include PATH]...
      Restore a machine backup archive to server HOME.
`

const machineAnalyseFilesHelp = `Usage: remote-agent machine analyse-files

Scan the server user's full home directory. Streams one completed entry block at
a time (immediate children, semantic enrichers, aggregates), then a summary.

Options:
  -h, --help        Show this help message
`

const machineBackupHelp = `Usage: remote-agent machine backup [--output PATH] [--dry-run] [--large-dir-threshold SIZE] [--show-config] [--set-config] [--exclude PATH]... [--include PATH]...

Snapshot the server user's home directory dot-files and dot-directories.

Options:
  --output PATH              Destination archive (default: machine-backup-<timestamp>.tar.xz)
  --dry-run                  Print the backup plan without writing an archive
  --large-dir-threshold SIZE Minimum dir size to flag LARGE SIZE in dry-run summary (default 10MB)
  --show-config              Print effective merged exclusion config JSON and exit
  --set-config               Persist --exclude paths to ~/.ai-critic/backup-config.json on server
  --skip-git-dirs-scan       Skip git repo discovery in summary and archive
  --git-dirs-scan-max-depth N Cap git scan depth under included dot-dirs (0 = unlimited)
  --exclude PATH             Additional exclusion (repeatable; merged with built-in rules)
  --include PATH             Re-include a built-in excluded path (repeatable)
  -h, --help                 Show this help message
`

const machineRestoreHelp = `Usage: remote-agent machine restore [backup.tar.xz] [--dry-run] [--show-config] [--show-meta] [--exclude PATH]... [--include PATH]...

Restore a machine backup archive to the server user's home directory.

Options:
  --dry-run         Print the restore plan without writing files
  --show-config     Print exclusion config (effective merged, or from archive .backup/config.json)
  --show-meta       Print .backup meta from archive (except config.json)
  --exclude PATH    Skip restoring PATH (repeatable)
  --include PATH    Re-include a built-in excluded path (repeatable)
  -h, --help        Show this help message
`

func runMachine(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(machineHelp)
		return nil
	}
	switch args[0] {
	case "analyse-files":
		return runMachineAnalyseFiles(resolve, args[1:])
	case "backup":
		return runMachineBackup(resolve, args[1:])
	case "restore":
		return runMachineRestore(resolve, args[1:])
	case "-h", "--help":
		fmt.Print(machineHelp)
		return nil
	default:
		return fmt.Errorf("unknown machine subcommand: %s", args[0])
	}
}

func runMachineAnalyseFiles(resolve func() (*client.Client, error), args []string) error {
	args, err := flags.
		Help("-h,--help", machineAnalyseFilesHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("machine analyse-files takes no positional arguments, got %v", args)
	}
	return streamcmd.Run(resolve, streamcmd.Spec{
		Method: http.MethodPost,
		Path:   "/api/remote-agent/machine/analyse-files/stream",
		Body:   map[string]any{},
		Print:  streamcmd.Logs,
		Printer: streamcmd.Printer{
			Log: printMachineStreamLog,
		},
	})
}

func runMachineBackup(resolve func() (*client.Client, error), args []string) error {
	var outputPath string
	var dryRun bool
	var showConfig bool
	var setConfig bool
	var skipGitDirsScan bool
	var gitDirsScanMaxDepth int
	var largeDirThresholdFlag string
	var exclude []string
	var include []string

	args, err := flags.
		String("--output", &outputPath).
		Bool("--dry-run", &dryRun).
		Bool("--show-config", &showConfig).
		Bool("--set-config", &setConfig).
		Bool("--skip-git-dirs-scan", &skipGitDirsScan).
		Int("--git-dirs-scan-max-depth", &gitDirsScanMaxDepth).
		String("--large-dir-threshold", &largeDirThresholdFlag).
		StringSlice("--exclude", &exclude).
		StringSlice("--include", &include).
		Help("-h,--help", machineBackupHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("machine backup takes no positional arguments, got %v", args)
	}

	var largeDirThresholdBytes int64
	if strings.TrimSpace(largeDirThresholdFlag) != "" {
		largeDirThresholdBytes, err = machinebackup.ParseHumanSize(largeDirThresholdFlag)
		if err != nil {
			return fmt.Errorf("invalid --large-dir-threshold %q: %v", largeDirThresholdFlag, err)
		}
	}

	if showConfig && setConfig {
		return fmt.Errorf("machine backup: --show-config and --set-config are mutually exclusive")
	}
	if setConfig {
		if len(exclude) == 0 && strings.TrimSpace(largeDirThresholdFlag) == "" {
			return fmt.Errorf("machine backup: --set-config requires --exclude and/or --large-dir-threshold")
		}
		if dryRun {
			return fmt.Errorf("machine backup: --set-config is mutually exclusive with --dry-run")
		}
		if strings.TrimSpace(outputPath) != "" {
			return fmt.Errorf("machine backup: --set-config is mutually exclusive with --output")
		}
		if len(include) > 0 {
			return fmt.Errorf("machine backup: --set-config is mutually exclusive with --include")
		}
		cli, err := resolve()
		if err != nil {
			return err
		}
		cfg, err := cli.MachineBackupSetConfig(exclude, largeDirThresholdFlag)
		if err != nil {
			return err
		}
		return printExclusionConfig(cfg)
	}
	if showConfig {
		return printEffectiveExclusionConfig(resolve, exclude, include, largeDirThresholdFlag)
	}

	if dryRun {
		body := map[string]any{"exclude": exclude, "include": include}
		if exclude == nil {
			body["exclude"] = []string{}
		}
		if include == nil {
			body["include"] = []string{}
		}
		if largeDirThresholdBytes > 0 {
			body["large_dir_threshold_bytes"] = largeDirThresholdBytes
		}
		if skipGitDirsScan {
			body["skip_git_dirs_scan"] = true
		}
		if gitDirsScanMaxDepth > 0 {
			body["git_dirs_scan_max_depth"] = gitDirsScanMaxDepth
		}
		return streamcmd.Run(resolve, streamcmd.Spec{
			Method: http.MethodPost,
			Path:   "/api/remote-agent/machine/backup/stream",
			Body:   body,
			Print:  streamcmd.Sections | streamcmd.Logs,
			Printer: streamcmd.Printer{
				Progress: printMachineBackupProgress,
				Log:      printMachineStreamLog,
			},
		})
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	if strings.TrimSpace(outputPath) == "" {
		outputPath = fmt.Sprintf("machine-backup-%s.tar.xz", time.Now().UTC().Format("20060102-150405"))
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil && filepath.Dir(outputPath) != "." {
		return fmt.Errorf("create output directory: %w", err)
	}

	body, err := cli.MachineBackupArchive(exclude, include, client.MachineBackupOptions{
		SkipGitDirsScan:     skipGitDirsScan,
		GitDirsScanMaxDepth: gitDirsScanMaxDepth,
	})
	if err != nil {
		return err
	}
	defer body.Close()

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create archive %s: %w", outputPath, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, body); err != nil {
		return fmt.Errorf("write archive %s: %w", outputPath, err)
	}
	fmt.Println(outputPath)
	return nil
}

func runMachineRestore(resolve func() (*client.Client, error), args []string) error {
	var dryRun bool
	var showConfig bool
	var showMeta bool
	var exclude []string
	var include []string

	args, err := flags.
		Bool("--dry-run", &dryRun).
		Bool("--show-config", &showConfig).
		Bool("--show-meta", &showMeta).
		StringSlice("--exclude", &exclude).
		StringSlice("--include", &include).
		Help("-h,--help", machineRestoreHelp).
		Parse(args)
	if err != nil {
		return err
	}

	var archivePath string
	if len(args) == 1 {
		archivePath = args[0]
	} else if len(args) > 1 {
		return fmt.Errorf("machine restore accepts at most 1 argument <backup.tar.xz>, got %v", args)
	}

	if showMeta {
		if archivePath == "" {
			return fmt.Errorf("machine restore --show-meta requires <backup.tar.xz>")
		}
		return printArchiveMeta(archivePath)
	}
	if showConfig {
		return printRestoreConfig(resolve, archivePath, exclude, include)
	}

	if archivePath == "" {
		return fmt.Errorf("machine restore requires <backup.tar.xz>")
	}

	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive %s: %w", archivePath, err)
	}
	defer f.Close()

	if dryRun {
		query := url.Values{"dry_run": {"true"}}
		for _, ex := range exclude {
			query.Add("exclude", ex)
		}
		for _, inc := range include {
			query.Add("include", inc)
		}
		return streamcmd.Run(resolve, streamcmd.Spec{
			Method: http.MethodPost,
			Path:   "/api/remote-agent/machine/restore/stream",
			Query:  query,
			Body: client.StreamRawBody{
				Reader:      f,
				ContentType: "application/x-xz",
			},
			Print: streamcmd.Logs,
			Printer: streamcmd.Printer{
				Progress: printMachineRestoreProgress,
				Log:      printMachineStreamLog,
			},
		})
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	plan, err := cli.MachineRestoreApply(f, exclude, include)
	if err != nil {
		return err
	}
	printMachineRestorePlan(plan, false)
	return nil
}

func printEffectiveExclusionConfig(resolve func() (*client.Client, error), exclude, include []string, largeDirThreshold string) error {
	cli, err := resolve()
	if err != nil {
		return err
	}
	cfg, err := cli.MachineBackupEffectiveConfig(exclude, include, largeDirThreshold)
	if err != nil {
		return err
	}
	return printExclusionConfig(cfg)
}

func printExclusionConfig(cfg *machinebackup.ExclusionConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func printRestoreConfig(resolve func() (*client.Client, error), archivePath string, exclude, include []string) error {
	if archivePath == "" {
		return printEffectiveExclusionConfig(resolve, exclude, include, "")
	}
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive %s: %w", archivePath, err)
	}
	defer f.Close()
	cfg, err := machinebackup.ReadArchiveConfig(f)
	if err != nil {
		return err
	}
	if cfg == nil {
		return printEffectiveExclusionConfig(resolve, nil, nil, "")
	}
	return printExclusionConfig(cfg)
}

func printArchiveMeta(archivePath string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive %s: %w", archivePath, err)
	}
	defer f.Close()
	meta, err := machinebackup.ReadArchiveMeta(f)
	if err != nil {
		return err
	}
	if len(meta) == 0 {
		return nil
	}
	names := make([]string, 0, len(meta))
	for name := range meta {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("=== .backup/%s ===\n", name)
		fmt.Println(string(meta[name]))
		if len(meta[name]) > 0 && meta[name][len(meta[name])-1] != '\n' {
			fmt.Println()
		}
	}
	return nil
}

func printMachineBackupProgress(ev client.StreamEvent) error {
	switch ev.Layer {
	case "dot_file":
		fmt.Printf("  %-40s %s\n", ev.Name, ev.Detail)
	case "dir":
		fmt.Printf("  %-16s %s\n", ev.Name, ev.Detail)
	case "excluded_header", "excluded":
		fmt.Println(ev.Detail)
	}
	streamcmd.FlushStdout()
	return nil
}

func printMachineStreamLog(ev client.StreamEvent) error {
	if ev.Verbatim {
		msg := ev.Message
		if term.IsTerminal(int(os.Stdout.Fd())) && strings.Contains(msg, "LARGE SIZE") {
			msg = colorLargeSizeRed(msg)
		}
		fmt.Println(msg)
	} else {
		if err := streamcmd.DefaultLog(ev); err != nil {
			return err
		}
		return nil
	}
	streamcmd.FlushStdout()
	return nil
}

func printMachineRestoreProgress(ev client.StreamEvent) error {
	switch ev.Status {
	case "skip":
		fmt.Printf("skip (identical): %s\n", ev.Name)
	case "update":
		fmt.Printf("update: %s\n", ev.Name)
	case "create":
		fmt.Printf("create: %s\n", ev.Name)
	default:
		fmt.Printf("%s: %s\n", ev.Status, ev.Name)
	}
	streamcmd.FlushStdout()
	return nil
}

func printMachineRestorePlan(plan *client.MachineRestorePlan, dryRun bool) {
	entries := append([]client.MachineRestoreEntry(nil), plan.Entries...)
	entries = sortRestoreEntries(entries)
	if dryRun && allRestoreEntriesSkip(entries) && len(entries) > 0 {
		entries = entries[:1]
	}
	for _, entry := range entries {
		switch entry.Action {
		case "skip":
			fmt.Printf("skip (identical): %s\n", entry.Path)
		case "update":
			fmt.Printf("update: %s\n", entry.Path)
		case "create":
			fmt.Printf("create: %s\n", entry.Path)
		default:
			fmt.Printf("%s: %s\n", entry.Action, entry.Path)
		}
	}
}

func sortRestoreEntries(entries []client.MachineRestoreEntry) []client.MachineRestoreEntry {
	out := append([]client.MachineRestoreEntry(nil), entries...)
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			ai := strings.Count(out[i].Path, "/")
			aj := strings.Count(out[j].Path, "/")
			if aj < ai || (aj == ai && out[j].Path < out[i].Path) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func colorLargeSizeRed(line string) string {
	const red = "\033[31m"
	const reset = "\033[0m"
	return strings.ReplaceAll(line, "LARGE SIZE", red+"LARGE SIZE"+reset)
}

func allRestoreEntriesSkip(entries []client.MachineRestoreEntry) bool {
	for _, entry := range entries {
		if entry.Action != "skip" {
			return false
		}
	}
	return len(entries) > 0
}