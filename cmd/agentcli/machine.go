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
)

const machineHelp = `Usage: remote-agent machine <subcommand> [args...]

Backup and restore the server user's home directory dot-files and dot-directories.

Subcommands:
  backup [--output PATH] [--dry-run] [--show-config] [--exclude PATH]... [--include PATH]...
      Snapshot server HOME as a streamed tar.xz archive.

  restore [backup.tar.xz] [--dry-run] [--show-config] [--show-meta] [--exclude PATH]... [--include PATH]...
      Restore a machine backup archive to server HOME.
`

const machineBackupHelp = `Usage: remote-agent machine backup [--output PATH] [--dry-run] [--show-config] [--exclude PATH]... [--include PATH]...

Snapshot the server user's home directory dot-files and dot-directories.

Options:
  --output PATH     Destination archive (default: machine-backup-<timestamp>.tar.xz)
  --dry-run         Print the backup plan without writing an archive
  --show-config     Print built-in exclusion config JSON and exit
  --exclude PATH    Additional exclusion (repeatable; merged with built-in rules)
  --include PATH    Re-include a built-in excluded path (repeatable)
  -h, --help        Show this help message
`

const machineRestoreHelp = `Usage: remote-agent machine restore [backup.tar.xz] [--dry-run] [--show-config] [--show-meta] [--exclude PATH]... [--include PATH]...

Restore a machine backup archive to the server user's home directory.

Options:
  --dry-run         Print the restore plan without writing files
  --show-config     Print exclusion config (built-in, or from archive .backup/config.json)
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

func runMachineBackup(resolve func() (*client.Client, error), args []string) error {
	var outputPath string
	var dryRun bool
	var showConfig bool
	var exclude []string
	var include []string

	args, err := flags.
		String("--output", &outputPath).
		Bool("--dry-run", &dryRun).
		Bool("--show-config", &showConfig).
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

	if showConfig {
		return printBuiltinExclusionConfig()
	}

	if dryRun {
		body := map[string]any{"exclude": exclude, "include": include}
		if exclude == nil {
			body["exclude"] = []string{}
		}
		if include == nil {
			body["include"] = []string{}
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

	body, err := cli.MachineBackupArchive(exclude, include)
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
		return printRestoreConfig(archivePath)
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

func printBuiltinExclusionConfig() error {
	data, err := machinebackup.BuiltinExclusionConfigJSON()
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func printRestoreConfig(archivePath string) error {
	if archivePath == "" {
		return printBuiltinExclusionConfig()
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
		return printBuiltinExclusionConfig()
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
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
	case "excluded":
		if ev.Detail != "" {
			fmt.Printf("  %-24s %s\n", ev.Name, ev.Detail)
		} else {
			fmt.Printf("  %s\n", ev.Name)
		}
	}
	streamcmd.FlushStdout()
	return nil
}

func printMachineStreamLog(ev client.StreamEvent) error {
	if ev.Verbatim {
		fmt.Println(ev.Message)
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

func allRestoreEntriesSkip(entries []client.MachineRestoreEntry) bool {
	for _, entry := range entries {
		if entry.Action != "skip" {
			return false
		}
	}
	return len(entries) > 0
}