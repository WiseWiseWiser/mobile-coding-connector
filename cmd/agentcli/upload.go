package agentcli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/xhd2015/ai-critic/client"
)

const uploadHelp = `Usage: remote-agent upload <LOCAL_PATH> [REMOTE_PATH]

Upload a local file or directory to the server using chunked upload.

Arguments:
  LOCAL_PATH    Path to a file or directory on this machine.
  REMOTE_PATH   Destination path on the server. Optional; defaults to the
                basename. If REMOTE_PATH ends with '/', the basename is
                appended. For directories, REMOTE_PATH is the mirror root.

Examples:
  remote-agent upload ./foo.txt /tmp/foo.txt
  remote-agent upload ./foo.txt /tmp/          # basename appended
  remote-agent upload ./foo.txt                # uses saved config + basename
  remote-agent upload ./srcdir uploads/mirror  # mirror directory tree
`

func runUpload(cli *client.Client, args []string) error {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		fmt.Print(uploadHelp)
		return nil
	}
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

	stat, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to stat local path: %w", err)
	}
	if stat.IsDir() {
		return runUploadDir(cli, localPath, remotePath)
	}

	chmodExec := isExecutableMode(stat.Mode())

	fmt.Printf("Uploading %s (%s) -> %s\n", localPath, formatSize(stat.Size()), describeRemote(remotePath))

	result, err := cli.UploadFile(localPath, remotePath, client.UploadOptions{
		ChmodExec: chmodExec,
	}, printUploadProgress)
	if err != nil {
		if hint := uploadFailureHint(err); hint != "" {
			return fmt.Errorf("%w\n  %s", err, hint)
		}
		return err
	}

	fmt.Printf("Upload complete: %s (%s)\n", result.Path, formatSize(result.Size))
	return nil
}

func runUploadDir(cli *client.Client, localDir, remotePath string) error {
	itemCount, _, totalSize, err := client.CountUploadDirItems(localDir)
	if err != nil {
		return err
	}

	logicalRemote := describeUploadDirRemote(localDir, remotePath)
	fmt.Printf("Uploading %s/ (%d items, %s) -> %s\n",
		localDir, itemCount, formatSize(totalSize), logicalRemote)

	result, err := cli.UploadDir(localDir, remotePath, client.UploadOptions{}, printUploadDirProgress)
	if err != nil {
		return err
	}

	// Blank line before summary when the plan has no empty subdirs (see streams-progress template).
	if itemCount == result.FileCount {
		fmt.Println()
	}
	fmt.Printf("Upload complete: %s (%d files, %s)\n",
		result.Path, result.FileCount, formatSize(result.TotalSize))
	return nil
}

func describeUploadDirRemote(localDir, remotePath string) string {
	baseName := filepath.Base(localDir)
	if remotePath == "" {
		return baseName
	}
	if strings.HasSuffix(remotePath, "/") {
		return strings.TrimSuffix(remotePath, "/") + "/" + baseName
	}
	return remotePath
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
