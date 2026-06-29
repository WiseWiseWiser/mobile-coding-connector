package agentcli

import (
	"fmt"
	"math"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/xhd2015/ai-critic/client"
	"github.com/xhd2015/less-gen/flags"
)

const serverHelp = `Usage: remote-agent server <subcommand> [args...]

Execute server-management actions exposed by the remote server UI.

Subcommands:
  build-next [--project <id>]
      Trigger the same "Build Next" action as the Manage Server page.
      Build logs are streamed back live.

  upload-next <local-binary>
      Upload a local server binary to the next remote binary path, deriving
      the remote name from the existing ai-critic-server-vN sequence.

  restart
      Trigger the same "Restart Server" action as the Manage Server page.
      Restart progress is streamed back live.

  status
      Show the same keep-alive and machine status information shown by
      the Manage Server page.
`

const serverBuildNextHelp = `Usage: remote-agent server build-next [--project <id>]

Trigger the remote server's /api/build/build-next action and stream its
logs back to this terminal.

Options:
  --project ID       Build the specified buildable project. If omitted,
                     the server chooses the same default project as the UI.
  -h, --help         Show this help message.
`

const serverUploadNextHelp = `Usage: remote-agent server upload-next <local-binary>

Upload <local-binary> to the next remote server binary path. The remote
filename is derived by scanning the current server binary directory for
the current binary's base name and choosing max(existing -vN)+1.

For example, if the remote directory contains:
  ai-critic-server-v1
  ai-critic-server-v2
  ai-critic-server-v10
  ai-critic-server-v15

the upload target is ai-critic-server-v16 in that same directory.

The uploaded file is marked executable on the remote server.
`

const serverRestartHelp = `Usage: remote-agent server restart

Trigger the remote server's /api/server/exec-restart action and stream
restart progress back to this terminal.
`

const serverStatusHelp = `Usage: remote-agent server status

Show the same keep-alive daemon and machine status details shown on the
Manage Server page, including the server/daemon PID and port, start
time, uptime, restart count, next health-check countdown, plus CPU,
memory, disk and top-process snapshots.
`

func runServer(resolve func() (*client.Client, error), args []string) error {
	if len(args) == 0 {
		fmt.Print(serverHelp)
		return nil
	}

	sub := args[0]
	rest := args[1:]
	switch sub {
	case "build-next":
		return runServerBuildNext(resolve, rest)
	case "upload-next":
		return runServerUploadNext(resolve, rest)
	case "restart":
		return runServerRestart(resolve, rest)
	case "status":
		return runServerStatus(resolve, rest)
	case "-h", "--help":
		fmt.Print(serverHelp)
		return nil
	default:
		return fmt.Errorf("unknown server subcommand: %s", sub)
	}
}

func runServerBuildNext(resolve func() (*client.Client, error), args []string) error {
	var projectID string

	args, err := flags.
		String("--project", &projectID).
		Help("-h,--help", serverBuildNextHelp).
		Parse(args)
	if err != nil {
		return err
	}
	if len(args) > 0 {
		return fmt.Errorf("server build-next does not accept positional args: %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	var result *client.BuildNextResult
	result, err = cli.BuildNext(projectID, func(ev client.ServerStreamEvent) {
		if ev.Message != "" {
			fmt.Println(ev.Message)
		}
	})
	if err != nil {
		return err
	}

	if result != nil {
		fmt.Printf("Build complete: %s\n", result.BinaryPath)
		if result.ProjectName != "" || result.Version != "" {
			fmt.Printf("Project: %s  Version: %s\n", displayOrDash(result.ProjectName), displayOrDash(result.Version))
		}
	}
	return nil
}

func runServerUploadNext(resolve func() (*client.Client, error), args []string) error {
	if len(args) != 1 {
		if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
			fmt.Print(serverUploadNextHelp)
			return nil
		}
		return fmt.Errorf("server upload-next requires exactly 1 argument <local-binary>")
	}

	localBinary := args[0]
	stat, err := os.Stat(localBinary)
	if err != nil {
		return fmt.Errorf("failed to stat local binary: %w", err)
	}
	if stat.IsDir() {
		return fmt.Errorf("local binary is a directory, not a file: %s", localBinary)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	target, usedCompatTarget, err := getNextBinaryTargetForUpload(cli)
	if err != nil {
		return err
	}
	if strings.TrimSpace(target.BinaryPath) == "" {
		return fmt.Errorf("server returned empty next binary path")
	}

	if usedCompatTarget {
		fmt.Println("Derived target from keep-alive status and remote directory listing.")
	}
	fmt.Printf("Next remote binary: %s\n", target.BinaryPath)
	if target.CurrentPath != "" {
		fmt.Printf("Current remote binary: %s\n", target.CurrentPath)
	}
	if target.PreviousHighestVersion > 0 || target.Version > 0 {
		fmt.Printf("Version: v%d -> v%d\n", target.PreviousHighestVersion, target.Version)
	}
	fmt.Printf("Uploading %s (%s) -> %s\n", localBinary, formatSize(stat.Size()), target.BinaryPath)

	result, err := cli.UploadFile(localBinary, target.BinaryPath, client.UploadOptions{
		ChmodExec: true,
	}, func(p client.UploadProgress) {
		percent := 100
		if p.TotalBytes > 0 {
			percent = int(p.CompletedBytes * 100 / p.TotalBytes)
		}
		fmt.Printf("  chunk %d/%d uploaded (%s / %s, %d%%)\n",
			p.ChunkIndex+1, p.TotalChunks,
			formatSize(p.CompletedBytes), formatSize(p.TotalBytes), percent)
	})
	if err != nil {
		return err
	}

	fmt.Printf("Upload complete: %s (%s)\n", result.Path, formatSize(result.Size))
	return nil
}

func getNextBinaryTargetForUpload(cli *client.Client) (*client.NextBinaryTarget, bool, error) {
	target, err := cli.GetNextBinaryTarget()
	if err == nil {
		return target, false, nil
	}

	compatTarget, compatErr := deriveNextBinaryTargetCompat(cli)
	if compatErr != nil {
		return nil, false, fmt.Errorf("failed to get next binary target: %w; compatibility fallback failed: %v", err, compatErr)
	}
	return compatTarget, true, nil
}

func deriveNextBinaryTargetCompat(cli *client.Client) (*client.NextBinaryTarget, error) {
	ping, err := cli.PingKeepAlive()
	if err != nil {
		return nil, fmt.Errorf("ping keep-alive: %w", err)
	}
	if ping == nil || !ping.Running {
		return nil, fmt.Errorf("keep-alive is not running")
	}

	status, err := cli.GetKeepAliveStatus()
	if err != nil {
		return nil, fmt.Errorf("get keep-alive status: %w", err)
	}
	currentPath := strings.TrimSpace(status.BinaryPath)
	if currentPath == "" {
		return nil, fmt.Errorf("keep-alive status did not include server binary path")
	}

	dir := path.Dir(currentPath)
	browse, err := cli.BrowseDir(dir)
	if err != nil {
		return nil, fmt.Errorf("browse remote binary directory %s: %w", dir, err)
	}
	return nextBinaryTargetFromRemoteEntries(currentPath, browse.Entries)
}

func nextBinaryTargetFromRemoteEntries(currentPath string, entries []client.BrowseEntry) (*client.NextBinaryTarget, error) {
	currentPath = strings.TrimSpace(currentPath)
	if currentPath == "" {
		return nil, fmt.Errorf("current binary path is empty")
	}

	dir := path.Dir(currentPath)
	currentName := path.Base(currentPath)
	baseName, currentVersion := parseRemoteBinaryVersion(currentName)
	if baseName == "" || baseName == "." || baseName == "/" {
		return nil, fmt.Errorf("invalid current binary name: %s", currentName)
	}

	highestVersion := currentVersion
	for _, entry := range entries {
		if entry.IsDir {
			continue
		}
		base, version := parseRemoteBinaryVersion(entry.Name)
		if base == baseName && version > highestVersion {
			highestVersion = version
		}
	}

	nextVersion := highestVersion + 1
	binaryName := fmt.Sprintf("%s-v%d", baseName, nextVersion)
	return &client.NextBinaryTarget{
		BinaryPath:             path.Join(dir, binaryName),
		BinaryName:             binaryName,
		Version:                nextVersion,
		BaseName:               baseName,
		Directory:              dir,
		CurrentPath:            currentPath,
		CurrentName:            currentName,
		CurrentVersion:         currentVersion,
		PreviousHighestVersion: highestVersion,
	}, nil
}

func parseRemoteBinaryVersion(name string) (string, int) {
	if idx := strings.LastIndex(name, "-v"); idx != -1 && idx < len(name)-2 {
		versionStr := name[idx+2:]
		if version, err := strconv.Atoi(versionStr); err == nil {
			return name[:idx], version
		}
	}
	return name, 0
}

func runServerRestart(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 {
		if args[0] == "-h" || args[0] == "--help" {
			fmt.Print(serverRestartHelp)
			return nil
		}
		return fmt.Errorf("server restart takes no arguments, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	result, err := cli.RestartServer(func(ev client.ServerStreamEvent) {
		if ev.Message != "" {
			fmt.Println(ev.Message)
		}
	})
	if err != nil {
		return err
	}

	if result != nil {
		if result.KeepAlive != nil {
			fmt.Printf("Server is back up: PID %d  Uptime %s  Restarts %d\n",
				result.KeepAlive.ServerPID,
				displayOrDash(result.KeepAlive.Uptime),
				result.KeepAlive.RestartCount,
			)
			fmt.Printf("Binary: %s\n", displayOrDash(result.KeepAlive.BinaryPath))
			return nil
		}
		if result.Binary != "" {
			fmt.Printf("Server is back up. Requested binary: %s\n", result.Binary)
			return nil
		}
	}
	fmt.Println("Server is back up and reachable.")
	return nil
}

func runServerStatus(resolve func() (*client.Client, error), args []string) error {
	if len(args) > 0 {
		if args[0] == "-h" || args[0] == "--help" {
			fmt.Print(serverStatusHelp)
			return nil
		}
		return fmt.Errorf("server status takes no arguments, got %v", args)
	}

	cli, err := resolve()
	if err != nil {
		return err
	}

	ping, err := cli.PingKeepAlive()
	if err != nil {
		return err
	}

	var keepAlive *client.KeepAliveStatus
	if ping.Running {
		keepAlive, err = cli.GetKeepAliveStatus()
		if err != nil {
			return err
		}
	}

	serverStatus, err := cli.GetServerStatus()
	if err != nil {
		return err
	}

	printServerProcessStatus(keepAlive)
	fmt.Println()
	printKeepAliveStatus(ping, keepAlive)
	fmt.Println()
	printMachineStatus(serverStatus)
	return nil
}

func printServerProcessStatus(status *client.KeepAliveStatus) {
	printSection("Server Process")

	state := "Reachable (keep-alive not running)"
	pid := "unavailable"
	port := "unavailable"
	startedAt := "unavailable"
	uptime := "unavailable"
	binaryPath := "unavailable"

	if status != nil {
		if status.ServerPID > 0 {
			state = "Running"
			pid = fmt.Sprintf("%d", status.ServerPID)
		} else {
			state = "Not Reported By Keep-Alive"
		}
		if status.ServerPort > 0 {
			port = fmt.Sprintf("%d", status.ServerPort)
		}
		if status.StartedAt != "" {
			startedAt = formatTimestamp(status.StartedAt)
		}
		if status.Uptime != "" {
			uptime = status.Uptime
		}
		if status.BinaryPath != "" {
			binaryPath = status.BinaryPath
		}
	}

	printField("Status", state)
	printField("PID", pid)
	printField("Port", port)
	printField("Started At", startedAt)
	printField("Uptime", uptime)
	printField("Binary", binaryPath)
	if status == nil {
		printField("Note", "PID/port/start time come from keep-alive; only machine stats are available while it is stopped")
	}
}

func printKeepAliveStatus(ping *client.KeepAlivePing, status *client.KeepAliveStatus) {
	printSection("Keep-Alive Daemon")

	if ping == nil || !ping.Running || status == nil {
		printField("Status", "Not Running")
		if ping != nil && ping.StartCommand != "" {
			printField("Start Command", ping.StartCommand)
		}
		return
	}

	printField("Status", "Running")
	printField("PID", formatOptionalInt(status.KeepAlivePID))
	printField("Port", formatOptionalInt(status.KeepAlivePort))
	printField("Daemon Binary", displayOrDash(status.DaemonBinaryPath))
	printField("Restart Count", fmt.Sprintf("%d", status.RestartCount))
	if status.NextBinary != "" {
		printField("Next Binary", status.NextBinary)
	}
	if status.NextHealthCheckTime != "" {
		printField("Next Check", formatNextCheck(status.NextHealthCheckTime))
	}
}

func printMachineStatus(status *client.ServerStatus) {
	printSection("Machine Status")
	if status == nil {
		printField("Status", "Unavailable")
		return
	}

	printField("OS", displayOrDash(status.OSInfo.OS))
	printField("Arch", displayOrDash(status.OSInfo.Arch))
	printField("Kernel", displayOrDash(status.OSInfo.Kernel))
	printField("CPU Cores", fmt.Sprintf("%d", status.CPU.NumCPU))
	printField("CPU Usage", formatPercent(status.CPU.UsedPercent))
	printField("Total Memory", formatBytes(status.Memory.Total))
	printField("Used Memory", fmt.Sprintf("%s (%s)", formatBytes(status.Memory.Used), formatPercent(status.Memory.UsedPercent)))

	if len(status.Disk) == 0 {
		printField("Disks", "none")
	} else {
		fmt.Printf("  %-14s\n", "Disks:")
		for _, disk := range status.Disk {
			fmt.Printf("    %s: %s / %s (%s)\n",
				displayOrDash(disk.MountPoint),
				formatBytes(disk.Used),
				formatBytes(disk.Size),
				formatPercent(disk.UsePercent),
			)
		}
	}

	printProcessList("Top CPU Processes", status.TopCPU)
	printProcessList("Top Memory Processes", status.TopMem)
}

func printProcessList(title string, processes []client.ProcessStatus) {
	fmt.Printf("  %s:\n", title)
	if len(processes) == 0 {
		fmt.Println("    (none)")
		return
	}
	for _, proc := range processes {
		name := strings.TrimSpace(proc.Name)
		if name == "" {
			name = "unknown"
		}
		fmt.Printf("    %s (PID %d): CPU %s | Mem %s\n", name, proc.PID, displayOrDash(proc.CPU), displayOrDash(proc.Mem))
	}
}

func printSection(title string) {
	fmt.Println(title)
	fmt.Println(strings.Repeat("=", len(title)))
}

func printField(name string, value string) {
	fmt.Printf("  %-14s %s\n", name+":", value)
}

func formatOptionalInt(v int) string {
	if v <= 0 {
		return "unavailable"
	}
	return fmt.Sprintf("%d", v)
}

func formatTimestamp(value string) string {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return t.Format("2006-01-02 15:04:05 -0700 MST")
}

func formatNextCheck(value string) string {
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	seconds := int(math.Ceil(time.Until(t).Seconds()))
	if seconds < 0 {
		seconds = 0
	}
	return fmt.Sprintf("%ds", seconds)
}

func formatPercent(v float64) string {
	return fmt.Sprintf("%.1f%%", v)
}

func formatBytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for value := n / unit; value >= unit; value /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
