package agentcli

import (
	"fmt"
	"os"

	"github.com/xhd2015/ai-critic/client"
)

const uploadHelp = `Usage: remote-agent upload [--dry-run] [--no-compress] [--no-override] <LOCAL_PATH> [REMOTE_PATH]

Upload a local file or directory over a single websocket stream.
Interrupted uploads resume from the last server-acked byte offset
(same content → same hash). Directories still pack tar.xz first.

Arguments:
  LOCAL_PATH    Path to a file or directory on this machine.
  REMOTE_PATH   Destination path on the server. Optional; defaults to the
                basename. If REMOTE_PATH ends with '/', it is treated as a
                directory container. For directories, destination follows cp -R
                rules (see below).

Directory destination rules (cp -R style):
  - If REMOTE_PATH does not exist, it is created and local contents are copied
    into it.
  - If REMOTE_PATH exists and is a file, the upload fails.
  - If REMOTE_PATH exists and is a directory, files are placed under
    REMOTE_PATH/<basename(LOCAL_PATH)/> (created if needed). If that nested
    path exists as a file, the upload fails.

Directory transport packs a local tar.xz, uploads one archive, then extracts
and merges on the remote. Existing same-name files are overridden by default
(with warnings). Use --no-override to refuse when any remote file would be
replaced (preflight before packing, plus a remote fail-fast guard).

Options:
  --dry-run       Print the upload plan without making changes.
  --no-compress   Skip gzip of file payloads (directory archives are always xz).
  --no-override   For directory uploads: refuse if any remote file would be
                  overwritten (preflight + fail-fast during apply).

Examples:
  remote-agent upload ./foo.txt /tmp/foo.txt
  remote-agent upload ./foo.txt /tmp/          # basename appended
  remote-agent upload ./foo.txt                # uses saved config + basename
  remote-agent upload ./srcdir /tmp/apps       # if /tmp/apps exists: /tmp/apps/srcdir
  remote-agent upload ./srcdir /tmp/newdir     # creates /tmp/newdir with contents
  remote-agent upload --no-override ./srcdir /tmp/apps
  remote-agent upload --dry-run ./srcdir /tmp/apps
  remote-agent upload --no-compress ./bin /tmp/bin
`

func runUpload(cli *client.Client, args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(uploadHelp)
		return nil
	}

	dryRun, noCompress, noOverride, args := parseUploadFlags(args)
	if len(args) < 1 {
		return fmt.Errorf("upload requires <LOCAL_PATH> [REMOTE_PATH]; see 'remote-agent upload --help'")
	}
	if len(args) > 2 {
		return fmt.Errorf("upload takes at most 2 arguments, got %d", len(args))
	}

	localPath := args[0]
	remotePath := ""
	if len(args) == 2 {
		remotePath = args[1]
	}

	if dryRun {
		fmt.Println("dry-run: upload plan")
	}

	stat, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to stat local path: %w", err)
	}
	if stat.IsDir() {
		return runUploadDir(cli, localPath, remotePath, dryRun, noCompress, noOverride)
	}
	if noOverride {
		return fmt.Errorf("--no-override applies only to directory uploads")
	}

	chmodExec := isExecutableMode(stat.Mode())

	fmt.Printf("Uploading %s (%s) -> %s\n", localPath, formatSize(stat.Size()), describeRemote(remotePath))

	uploadOpts := client.UploadOptions{
		ChmodExec:  chmodExec,
		NoCompress: noCompress,
		DryRun:     dryRun,
	}
	progressFn := printUploadProgress
	if dryRun {
		progressFn = printUploadDryRunProgress
	}

	result, err := cli.UploadFile(localPath, remotePath, uploadOpts, progressFn)
	if err != nil {
		if hint := uploadFailureHint(err); hint != "" {
			return fmt.Errorf("%w\n  %s", err, hint)
		}
		return err
	}

	if dryRun {
		fmt.Printf("dry-run: upload complete: %s (%s)\n", result.Path, formatSize(result.Size))
	} else {
		fmt.Printf("Upload complete: %s (%s)\n", result.Path, formatSize(result.Size))
	}
	return nil
}

func runUploadDir(cli *client.Client, localDir, remotePath string, dryRun, noCompress, noOverride bool) error {
	_ = noCompress // directory archives are always xz-packed

	if dryRun {
		fmt.Fprintln(os.Stderr, "notice: dry-run (probes only; mutations gated)")
	}

	staged := newUploadDirStagePrinter(dryRun)
	uploadOpts := client.UploadOptions{
		DryRun:     dryRun,
		NoOverride: noOverride,
	}

	result, err := cli.UploadDir(localDir, remotePath, uploadOpts, staged.OnProgress)
	if err != nil {
		return err
	}

	staged.FinishApply(result.Overridden)
	staged.PrintProduct(result)
	return nil
}

func isExecutableMode(mode os.FileMode) bool {
	return mode.IsRegular() && mode&0o111 != 0
}

func describeRemote(remotePath string) string {
	if remotePath == "" {
		return "(server home dir)"
	}
	return remotePath
}

func formatSize(n int64) string {
	const (
		kb = 1000
		mb = 1000 * kb
		gb = 1000 * mb
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
